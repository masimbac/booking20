package bookings

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/parama/booking/internal/app/ports"
	"github.com/parama/booking/internal/domain"
)

// EventSink receives booking lifecycle notifications (EventBridge/outbox later).
type EventSink interface {
	BookingCreated(ctx context.Context, b domain.Booking) error
	BookingLifecycle(ctx context.Context, b domain.Booking, transition string) error
}

type noopSink struct{}

func (noopSink) BookingCreated(ctx context.Context, b domain.Booking) error { return nil }

func (noopSink) BookingLifecycle(ctx context.Context, b domain.Booking, transition string) error {
	return nil
}

// NoopEvents is a default EventSink for Lambda until asynchronous publish is wired.
var NoopEvents EventSink = noopSink{}

// Application orchestrates booking use cases.
type Application struct {
	Bookings  ports.BookingRepository
	Services  ports.ServiceRepository
	Customers ports.CustomerRepository
	Events    EventSink
	Now       func() time.Time
}

func (a *Application) now() time.Time {
	if a.Now != nil {
		t := a.Now()
		if !t.IsZero() {
			return t
		}
	}
	return time.Now().UTC()
}

func (a *Application) sink() EventSink {
	if a.Events != nil {
		return a.Events
	}
	return NoopEvents
}

// CreateBookingInput mirrors POST /bookings.
type CreateBookingInput struct {
	BusinessID string
	CustomerID string
	ServiceID  string
	StaffID    string
	StartAt    time.Time
	EndAt      *time.Time
}

// CreateBooking persists a booking or replays an idempotent create.
func (a *Application) CreateBooking(ctx context.Context, idempotencyKey string, in CreateBookingInput) (*domain.Booking, bool, error) {
	if strings.TrimSpace(in.BusinessID) == "" {
		return nil, false, fmt.Errorf("%w: business_id is required", domain.ErrInvalid)
	}
	if strings.TrimSpace(in.CustomerID) == "" || strings.TrimSpace(in.ServiceID) == "" {
		return nil, false, fmt.Errorf("%w: customer_id and service_id are required", domain.ErrInvalid)
	}
	staffID := strings.TrimSpace(in.StaffID)
	if staffID == "" {
		return nil, false, fmt.Errorf("%w: staff_id is required for scheduling conflicts", domain.ErrInvalid)
	}
	if _, err := a.Customers.Get(ctx, in.BusinessID, in.CustomerID); err != nil {
		return nil, false, err
	}
	svc, err := a.Services.Get(ctx, in.BusinessID, in.ServiceID)
	if err != nil {
		return nil, false, err
	}
	end := in.EndAt
	if end == nil {
		if svc.DurationMinutes < 1 {
			return nil, false, fmt.Errorf("%w: service has invalid duration", domain.ErrInvalid)
		}
		e := in.StartAt.Add(time.Duration(svc.DurationMinutes) * time.Minute)
		end = &e
	}
	if !end.After(in.StartAt) {
		return nil, false, fmt.Errorf("%w: end_at must be after start_at", domain.ErrInvalid)
	}
	if err := a.assertNoStaffOverlap(ctx, in.BusinessID, staffID, in.StartAt, *end, ""); err != nil {
		return nil, false, err
	}
	t := a.now()
	id := ulid.MustNew(ulid.Timestamp(t), ulid.Monotonic(rand.Reader, 0)).String()
	b := &domain.Booking{
		ID:         id,
		BusinessID: strings.TrimSpace(in.BusinessID),
		CustomerID: strings.TrimSpace(in.CustomerID),
		ServiceID:  strings.TrimSpace(in.ServiceID),
		StaffID:    staffID,
		StartAt:    in.StartAt.UTC(),
		EndAt:      end.UTC(),
		Status:     domain.BookingCreated,
		CreatedAt:  t,
		UpdatedAt:  t,
	}
	out, created, err := a.Bookings.CreateIfNotExistsWithIdempotency(ctx, idempotencyKey, b)
	if err != nil {
		return nil, false, err
	}
	if created {
		_ = a.sink().BookingCreated(ctx, *out)
	}
	return out, created, nil
}

func intervalsOverlap(a0, a1, b0, b1 time.Time) bool {
	return a0.Before(b1) && b0.Before(a1)
}

func (a *Application) assertNoStaffOverlap(ctx context.Context, businessID, staffID string, start, end time.Time, excludeBookingID string) error {
	candidates, err := a.Bookings.FindStaffCandidatesBefore(ctx, businessID, staffID, end, 100)
	if err != nil {
		return err
	}
	for i := range candidates {
		c := &candidates[i]
		if c.ID == excludeBookingID {
			continue
		}
		if !domain.StatusReservesSlot(c.Status) {
			continue
		}
		if intervalsOverlap(c.StartAt, c.EndAt, start, end) {
			return fmt.Errorf("%w: staff has an overlapping booking in that window", domain.ErrConflict)
		}
	}
	return nil
}

// ListBookings returns bookings whose start time falls in [fromUTC, toUTC] on GSI1.
func (a *Application) ListBookings(ctx context.Context, businessID string, fromUTC, toUTC time.Time, opt ports.ListBookingsOptions) ([]domain.Booking, string, error) {
	return a.Bookings.ListByStartRange(ctx, businessID, fromUTC, toUTC, opt)
}

// GetBooking returns one booking.
func (a *Application) GetBooking(ctx context.Context, businessID, bookingID string) (*domain.Booking, error) {
	return a.Bookings.Get(ctx, businessID, bookingID)
}

// ConfirmBooking transitions created → confirmed (re-check overlap against other bookings).
func (a *Application) ConfirmBooking(ctx context.Context, businessID, bookingID string) (*domain.Booking, error) {
	b, err := a.Bookings.Get(ctx, businessID, bookingID)
	if err != nil {
		return nil, err
	}
	if err := a.assertNoStaffOverlap(ctx, businessID, b.StaffID, b.StartAt, b.EndAt, b.ID); err != nil {
		return nil, err
	}
	if err := b.Confirm(a.now()); err != nil {
		return nil, err
	}
	if err := a.Bookings.Save(ctx, b); err != nil {
		return nil, err
	}
	_ = a.sink().BookingLifecycle(ctx, *b, "confirmed")
	return b, nil
}

type CancelBookingInput struct {
	Reason string
}

// CancelBooking ends the appointment.
func (a *Application) CancelBooking(ctx context.Context, businessID, bookingID string, in CancelBookingInput) (*domain.Booking, error) {
	b, err := a.Bookings.Get(ctx, businessID, bookingID)
	if err != nil {
		return nil, err
	}
	if err := b.Cancel(a.now(), strings.TrimSpace(in.Reason)); err != nil {
		return nil, err
	}
	if err := a.Bookings.Save(ctx, b); err != nil {
		return nil, err
	}
	_ = a.sink().BookingLifecycle(ctx, *b, "cancelled")
	return b, nil
}

// CompleteBooking marks a confirmed booking completed.
func (a *Application) CompleteBooking(ctx context.Context, businessID, bookingID string) (*domain.Booking, error) {
	b, err := a.Bookings.Get(ctx, businessID, bookingID)
	if err != nil {
		return nil, err
	}
	if err := b.Complete(a.now()); err != nil {
		return nil, err
	}
	if err := a.Bookings.Save(ctx, b); err != nil {
		return nil, err
	}
	_ = a.sink().BookingLifecycle(ctx, *b, "completed")
	return b, nil
}

// NoShowBooking marks a confirmed booking as no-show.
func (a *Application) NoShowBooking(ctx context.Context, businessID, bookingID string) (*domain.Booking, error) {
	b, err := a.Bookings.Get(ctx, businessID, bookingID)
	if err != nil {
		return nil, err
	}
	if err := b.MarkNoShow(a.now()); err != nil {
		return nil, err
	}
	if err := a.Bookings.Save(ctx, b); err != nil {
		return nil, err
	}
	_ = a.sink().BookingLifecycle(ctx, *b, "no_show")
	return b, nil
}

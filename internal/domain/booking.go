package domain

import (
	"fmt"
	"time"
)

// BookingStatus is the appointment lifecycle (OpenAPI enum).
type BookingStatus string

const (
	BookingCreated    BookingStatus = "created"
	BookingConfirmed  BookingStatus = "confirmed"
	BookingCompleted  BookingStatus = "completed"
	BookingCancelled  BookingStatus = "cancelled"
	BookingNoShow     BookingStatus = "no_show"
)

// Booking is a staff-scoped time reservation for a service.
type Booking struct {
	ID           string         `json:"id"`
	BusinessID   string         `json:"business_id"`
	CustomerID   string         `json:"customer_id"`
	ServiceID    string         `json:"service_id"`
	StaffID      string         `json:"staff_id,omitempty"`
	StartAt      time.Time      `json:"start_at"`
	EndAt        time.Time      `json:"end_at"`
	Status       BookingStatus  `json:"status"`
	CancelReason string         `json:"cancel_reason,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// StatusReservesSlot returns true while the booking blocks the calendar for conflict checks.
func StatusReservesSlot(s BookingStatus) bool {
	switch s {
	case BookingCreated, BookingConfirmed:
		return true
	default:
		return false
	}
}

// Confirm moves created → confirmed.
func (b *Booking) Confirm(now time.Time) error {
	if b.Status != BookingCreated {
		return fmt.Errorf("%w: only created bookings can be confirmed", ErrConflict)
	}
	b.Status = BookingConfirmed
	b.UpdatedAt = now
	return nil
}

// Cancel ends an active booking (created or confirmed).
func (b *Booking) Cancel(now time.Time, reason string) error {
	switch b.Status {
	case BookingCreated, BookingConfirmed:
	default:
		return fmt.Errorf("%w: cannot cancel booking in current state", ErrConflict)
	}
	b.Status = BookingCancelled
	b.CancelReason = reason
	b.UpdatedAt = now
	return nil
}

// Complete marks a confirmed appointment as done.
func (b *Booking) Complete(now time.Time) error {
	if b.Status != BookingConfirmed {
		return fmt.Errorf("%w: only confirmed bookings can be completed", ErrConflict)
	}
	b.Status = BookingCompleted
	b.UpdatedAt = now
	return nil
}

// MarkNoShow records a missed confirmed appointment.
func (b *Booking) MarkNoShow(now time.Time) error {
	if b.Status != BookingConfirmed {
		return fmt.Errorf("%w: only confirmed bookings can be marked no-show", ErrConflict)
	}
	b.Status = BookingNoShow
	b.UpdatedAt = now
	return nil
}

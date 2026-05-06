package ports

import (
	"context"
	"time"

	"github.com/parama/booking/internal/domain"
)

// ListBookingsOptions paginates booking list on GSI1 (start time sort key).
type ListBookingsOptions struct {
	Limit  int32
	Cursor string
}

// BookingRepository persists bookings (SK BOOKING#…, GSI1 date range).
type BookingRepository interface {
	CreateIfNotExistsWithIdempotency(ctx context.Context, idempotencyKey string, b *domain.Booking) (*domain.Booking, bool, error)
	Get(ctx context.Context, businessID, bookingID string) (*domain.Booking, error)
	Save(ctx context.Context, b *domain.Booking) error
	ListByStartRange(ctx context.Context, businessID string, fromUTC, toUTC time.Time, opt ListBookingsOptions) ([]domain.Booking, string, error)
	// FindActiveStaffBookingsStartingBefore returns bookings for staff whose interval may overlap [windowStart, windowEnd).
	// Implementations use GSI1 (start < windowEnd) and rely on callers to filter overlap and status.
	FindStaffCandidatesBefore(ctx context.Context, businessID, staffID string, windowEnd time.Time, limit int32) ([]domain.Booking, error)
}

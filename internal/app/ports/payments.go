package ports

import (
	"context"

	"github.com/parama/booking/internal/domain"
)

// ListPaymentsOptions paginates GSI3 (payments for a booking).
type ListPaymentsOptions struct {
	Limit  int32
	Cursor string
}

// PaymentRepository persists payments (SK PAYMENT#…, GSI3 by booking).
type PaymentRepository interface {
	CreateIfNotExistsWithIdempotency(ctx context.Context, idempotencyKey string, p *domain.Payment) (*domain.Payment, bool, error)
	Get(ctx context.Context, businessID, paymentID string) (*domain.Payment, error)
	Save(ctx context.Context, p *domain.Payment) error
	ListByBooking(ctx context.Context, businessID, bookingID string, opt ListPaymentsOptions) ([]domain.Payment, string, error)
}

// CheckoutSessionInput is passed to a PSP when creating a hosted checkout.
type CheckoutSessionInput struct {
	BusinessID string
	PaymentID  string
	BookingID  string
	Amount     *domain.Money
	Provider   string
	ReturnURL  string
}

// CheckoutSessionResult is returned by a PSP adapter.
type CheckoutSessionResult struct {
	CheckoutURL string
	ExternalRef string
}

// PaymentCheckoutProvider creates provider-hosted payment flows (stub or Stripe, etc.).
type PaymentCheckoutProvider interface {
	CreateCheckoutSession(ctx context.Context, in CheckoutSessionInput) (*CheckoutSessionResult, error)
}

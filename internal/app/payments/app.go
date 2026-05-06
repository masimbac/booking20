package payments

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

// Application orchestrates payment intent creation and provider callbacks.
type Application struct {
	Payments  ports.PaymentRepository
	Bookings  ports.BookingRepository
	Services  ports.ServiceRepository
	Provider  ports.PaymentCheckoutProvider
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

func (a *Application) checkout() ports.PaymentCheckoutProvider {
	return a.Provider
}

// CreatePaymentInput mirrors POST /payments.
type CreatePaymentInput struct {
	BusinessID string
	BookingID  string
	Amount     *domain.Money
	Kind       domain.PaymentKind
	Provider   string
	ReturnURL  string
}

// CreatePayment persists a pending payment and returns a checkout URL from the provider adapter.
func (a *Application) CreatePayment(ctx context.Context, idempotencyKey string, in CreatePaymentInput) (*domain.Payment, bool, error) {
	if strings.TrimSpace(in.BusinessID) == "" || strings.TrimSpace(in.BookingID) == "" {
		return nil, false, fmt.Errorf("%w: business_id and booking_id are required", domain.ErrInvalid)
	}
	if in.Amount == nil || strings.TrimSpace(in.Amount.Amount) == "" {
		return nil, false, fmt.Errorf("%w: amount is required", domain.ErrInvalid)
	}
	switch in.Kind {
	case domain.PaymentDeposit, domain.PaymentFull, domain.PaymentBalance:
	default:
		return nil, false, fmt.Errorf("%w: invalid payment kind", domain.ErrInvalid)
	}
	b, err := a.Bookings.Get(ctx, in.BusinessID, in.BookingID)
	if err != nil {
		return nil, false, err
	}
	if b.Status != domain.BookingCreated && b.Status != domain.BookingConfirmed {
		return nil, false, fmt.Errorf("%w: payments are only allowed for active booking states", domain.ErrInvalid)
	}
	t := a.now()
	pid := ulid.MustNew(ulid.Timestamp(t), ulid.Monotonic(rand.Reader, 0)).String()
	prov := strings.TrimSpace(in.Provider)
	if prov == "" {
		prov = "stub"
	}
	p := &domain.Payment{
		ID:         pid,
		BusinessID: in.BusinessID,
		BookingID:  in.BookingID,
		Amount: &domain.Money{
			Amount:   strings.TrimSpace(in.Amount.Amount),
			Currency: strings.TrimSpace(strings.ToUpper(in.Amount.Currency)),
		},
		Kind:      in.Kind,
		Provider:  prov,
		Status:    domain.PaymentPending,
		CreatedAt: t,
		UpdatedAt: t,
	}
	sess, err := a.checkout().CreateCheckoutSession(ctx, ports.CheckoutSessionInput{
		BusinessID: in.BusinessID,
		PaymentID:  pid,
		BookingID:  in.BookingID,
		Amount:     p.Amount,
		Provider:   prov,
		ReturnURL:  strings.TrimSpace(in.ReturnURL),
	})
	if err != nil {
		return nil, false, err
	}
	if sess != nil {
		p.CheckoutURL = sess.CheckoutURL
		p.ExternalRef = sess.ExternalRef
	}
	out, created, err := a.Payments.CreateIfNotExistsWithIdempotency(ctx, idempotencyKey, p)
	if err != nil {
		return nil, false, err
	}
	return out, created, nil
}

// GetPayment returns one payment scoped to the business.
func (a *Application) GetPayment(ctx context.Context, businessID, paymentID string) (*domain.Payment, error) {
	p, err := a.Payments.Get(ctx, businessID, paymentID)
	if err != nil {
		return nil, err
	}
	if p.BusinessID != businessID {
		return nil, domain.ErrNotFound
	}
	return p, nil
}

// ListPaymentsForBooking lists GSI3 payments for a booking after verifying the booking exists.
func (a *Application) ListPaymentsForBooking(ctx context.Context, businessID, bookingID string, opt ports.ListPaymentsOptions) ([]domain.Payment, string, error) {
	if _, err := a.Bookings.Get(ctx, businessID, bookingID); err != nil {
		return nil, "", err
	}
	return a.Payments.ListByBooking(ctx, businessID, bookingID, opt)
}

// WebhookUpdateInput is the normalized provider callback payload.
type WebhookUpdateInput struct {
	BusinessID  string
	PaymentID   string
	Status      domain.PaymentStatus
	ExternalRef string
}

// RecordWebhook applies a provider webhook outcome to the stored payment row.
func (a *Application) RecordWebhook(ctx context.Context, in WebhookUpdateInput) (*domain.Payment, error) {
	if strings.TrimSpace(in.BusinessID) == "" || strings.TrimSpace(in.PaymentID) == "" {
		return nil, fmt.Errorf("%w: business_id and payment_id are required", domain.ErrInvalid)
	}
	p, err := a.Payments.Get(ctx, in.BusinessID, in.PaymentID)
	if err != nil {
		return nil, err
	}
	switch in.Status {
	case domain.PaymentSucceeded, domain.PaymentFailed, domain.PaymentRefunded, domain.PaymentPending:
		// ok
	default:
		return nil, fmt.Errorf("%w: invalid payment status", domain.ErrInvalid)
	}
	p.Status = in.Status
	if strings.TrimSpace(in.ExternalRef) != "" {
		p.ExternalRef = strings.TrimSpace(in.ExternalRef)
	}
	p.UpdatedAt = a.now()
	if err := a.Payments.Save(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

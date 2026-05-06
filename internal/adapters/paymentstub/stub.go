package paymentstub

import (
	"context"
	"fmt"

	"github.com/parama/booking/internal/app/ports"
)

// Provider is a dev-safe checkout stub (no network).
type Provider struct{}

func (Provider) CreateCheckoutSession(ctx context.Context, in ports.CheckoutSessionInput) (*ports.CheckoutSessionResult, error) {
	_ = ctx
	return &ports.CheckoutSessionResult{
		CheckoutURL: fmt.Sprintf("https://checkout.example.com/session/%s", in.PaymentID),
		ExternalRef: "stub_ref_" + in.PaymentID,
	}, nil
}

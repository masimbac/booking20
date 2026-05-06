package domain

import (
	"strings"
	"time"
)

// PaymentKind classifies the payment (OpenAPI).
type PaymentKind string

const (
	PaymentDeposit PaymentKind = "deposit"
	PaymentFull    PaymentKind = "full"
	PaymentBalance PaymentKind = "balance"
)

// PaymentStatus is the provider/settlement state (OpenAPI).
type PaymentStatus string

const (
	PaymentPending   PaymentStatus = "pending"
	PaymentSucceeded PaymentStatus = "succeeded"
	PaymentFailed    PaymentStatus = "failed"
	PaymentRefunded  PaymentStatus = "refunded"
)

// Payment links money movement to a booking (GSI3: payments by booking).
type Payment struct {
	ID          string        `json:"id"`
	BusinessID  string        `json:"business_id"`
	BookingID   string        `json:"booking_id"`
	Amount      *Money        `json:"amount"`
	Kind        PaymentKind   `json:"kind"`
	Provider    string        `json:"provider,omitempty"`
	ExternalRef string        `json:"external_ref,omitempty"`
	CheckoutURL string        `json:"checkout_url,omitempty"`
	Status      PaymentStatus `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// SucceededPaymentSatisfiesConfirm enforces Phase 7 policy: a booking may be confirmed only
// when there is a succeeded payment that covers the business rule for the catalog price.
//
// Policy: If the service has a non-empty price, require a succeeded payment of kind "full"
// whose amount and currency match that catalog price. If the service has no price configured,
// require at least one succeeded payment of kind "full" or "deposit" (deposit treated as
// sufficient prepay for MVP).
func SucceededPaymentSatisfiesConfirm(servicePrice *Money, payments []Payment) bool {
	for _, p := range payments {
		if p.Status != PaymentSucceeded {
			continue
		}
		if p.Amount == nil || strings.TrimSpace(p.Amount.Amount) == "" {
			continue
		}
		hasCatalogPrice := servicePrice != nil && strings.TrimSpace(servicePrice.Amount) != ""
		if hasCatalogPrice {
			if p.Kind != PaymentFull {
				continue
			}
			if moneyEquals(p.Amount, servicePrice) {
				return true
			}
			continue
		}
		if p.Kind == PaymentFull || p.Kind == PaymentDeposit {
			return true
		}
	}
	return false
}

func moneyEquals(a, b *Money) bool {
	if a == nil || b == nil {
		return false
	}
	return strings.TrimSpace(a.Amount) == strings.TrimSpace(b.Amount) &&
		strings.EqualFold(strings.TrimSpace(a.Currency), strings.TrimSpace(b.Currency))
}

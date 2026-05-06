package domain_test

import (
	"testing"

	"github.com/parama/booking/internal/domain"
)

func TestSucceededPaymentSatisfiesConfirm_withCatalogPrice(t *testing.T) {
	t.Parallel()
	price := &domain.Money{Amount: "50.00", Currency: "USD"}
	payments := []domain.Payment{
		{Status: domain.PaymentSucceeded, Kind: domain.PaymentFull, Amount: &domain.Money{Amount: "50.00", Currency: "usd"}},
	}
	if !domain.SucceededPaymentSatisfiesConfirm(price, payments) {
		t.Fatal("expected satisfied")
	}
	if domain.SucceededPaymentSatisfiesConfirm(price, []domain.Payment{
		{Status: domain.PaymentSucceeded, Kind: domain.PaymentDeposit, Amount: &domain.Money{Amount: "50.00", Currency: "USD"}},
	}) {
		t.Fatal("deposit must not satisfy when catalog price is set")
	}
}

func TestSucceededPaymentSatisfiesConfirm_noCatalogPrice(t *testing.T) {
	t.Parallel()
	payments := []domain.Payment{
		{Status: domain.PaymentSucceeded, Kind: domain.PaymentDeposit, Amount: &domain.Money{Amount: "10", Currency: "USD"}},
	}
	if !domain.SucceededPaymentSatisfiesConfirm(nil, payments) {
		t.Fatal("expected satisfied without catalog price")
	}
}

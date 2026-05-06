package domain_test

import (
	"testing"
	"time"

	"github.com/parama/booking/internal/domain"
)

func TestBooking_lifecycle(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	b := domain.Booking{Status: domain.BookingCreated}

	if err := b.Confirm(now); err != nil {
		t.Fatal(err)
	}
	if b.Status != domain.BookingConfirmed {
		t.Fatalf("status %s", b.Status)
	}
	if err := b.Complete(now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if b.Status != domain.BookingCompleted {
		t.Fatalf("status %s", b.Status)
	}
}

func TestBooking_Cancel_invalidAfterComplete(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	b := domain.Booking{Status: domain.BookingCompleted}
	if err := b.Cancel(now, ""); err == nil {
		t.Fatal("want error")
	}
}

func TestStatusReservesSlot(t *testing.T) {
	t.Parallel()
	if !domain.StatusReservesSlot(domain.BookingCreated) || !domain.StatusReservesSlot(domain.BookingConfirmed) {
		t.Fatal("created/confirmed should reserve")
	}
	if domain.StatusReservesSlot(domain.BookingCancelled) {
		t.Fatal("cancelled should not reserve")
	}
}

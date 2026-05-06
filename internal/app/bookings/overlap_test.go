package bookings

import (
	"testing"
	"time"
)

func Test_intervalsOverlap(t *testing.T) {
	t.Parallel()
	a0 := time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)
	a1 := time.Date(2026, 1, 10, 11, 0, 0, 0, time.UTC)
	b0 := time.Date(2026, 1, 10, 10, 30, 0, 0, time.UTC)
	b1 := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	if !intervalsOverlap(a0, a1, b0, b1) {
		t.Fatal("expected overlap")
	}
	if intervalsOverlap(a0, a1, time.Date(2026, 1, 10, 11, 0, 0, 0, time.UTC), time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)) {
		t.Fatal("adjacent half-open should not overlap")
	}
}

package schedule_test

import (
	"testing"
	"time"

	"github.com/parama/booking/internal/domain"
	"github.com/parama/booking/internal/schedule"
)

func TestBuildSlots_basicDay(t *testing.T) {
	t.Parallel()
	loc := time.UTC
	from := time.Date(2026, 5, 6, 0, 0, 0, 0, time.UTC)  // Wednesday
	to := time.Date(2026, 5, 7, 0, 0, 0, 0, time.UTC)
	rules := []domain.AvailabilityRule{
		{DayOfWeek: 3, StartLocal: "09:00", EndLocal: "10:30"}, // Wednesday
	}
	slots, err := schedule.BuildSlots(loc, rules, 30*time.Minute, from, to, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(slots) != 3 {
		t.Fatalf("got %d slots, want 3: %+v", len(slots), slots)
	}
	if !slots[0].StartAt.Equal(time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("first slot start: %v", slots[0].StartAt)
	}
}

func TestBuildSlots_mergesOverlappingIntervals(t *testing.T) {
	t.Parallel()
	loc := time.UTC
	from := time.Date(2026, 5, 6, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 5, 7, 0, 0, 0, 0, time.UTC)
	staff := "stf_1"
	rules := []domain.AvailabilityRule{
		{StaffID: staff, DayOfWeek: 3, StartLocal: "09:00", EndLocal: "10:00"},
		{StaffID: staff, DayOfWeek: 3, StartLocal: "09:30", EndLocal: "11:00"},
	}
	slots, err := schedule.BuildSlots(loc, rules, 60*time.Minute, from, to, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(slots) != 2 {
		t.Fatalf("merged window 09-11 with 1h slots => 2 slots, got %d %+v", len(slots), slots)
	}
}

func TestBuildSlots_staffFilter(t *testing.T) {
	t.Parallel()
	loc := time.UTC
	from := time.Date(2026, 5, 6, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 5, 7, 0, 0, 0, 0, time.UTC)
	rules := []domain.AvailabilityRule{
		{StaffID: "a", DayOfWeek: 3, StartLocal: "09:00", EndLocal: "10:00"},
		{StaffID: "b", DayOfWeek: 3, StartLocal: "09:00", EndLocal: "10:00"},
	}
	slots, err := schedule.BuildSlots(loc, rules, 60*time.Minute, from, to, "", "a")
	if err != nil {
		t.Fatal(err)
	}
	if len(slots) != 1 || slots[0].StaffID != "a" {
		t.Fatalf("want one slot for staff a, got %+v", slots)
	}
}

func TestBuildSlots_localTZ_crossUTCWindow(t *testing.T) {
	t.Parallel()
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("no tz data")
	}
	// Monday 2026-05-04 in New York: 09:00-10:00 local => 13:00-14:00 UTC (EDT)
	from := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	to := time.Date(2026, 5, 4, 15, 0, 0, 0, time.UTC)
	rules := []domain.AvailabilityRule{
		{DayOfWeek: 1, StartLocal: "09:00", EndLocal: "10:00"},
	}
	slots, err := schedule.BuildSlots(loc, rules, 30*time.Minute, from, to, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(slots) != 2 {
		t.Fatalf("want 2 half-hour slots, got %d %+v", len(slots), slots)
	}
	if !slots[0].StartAt.Equal(time.Date(2026, 5, 4, 13, 0, 0, 0, time.UTC)) {
		t.Fatalf("slot[0].StartAt = %v", slots[0].StartAt)
	}
}

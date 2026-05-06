package scheduling

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/parama/booking/internal/app/ports"
	"github.com/parama/booking/internal/domain"
	"github.com/parama/booking/internal/schedule"
)

const defaultSlotMinutes = 30

// Application orchestrates availability rules and slot computation.
type Application struct {
	Businesses ports.BusinessRepository
	Services   ports.ServiceRepository
	Rules      ports.AvailabilityRepository
	Now        func() time.Time
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

// PutRules replaces weekly availability rules for a business.
func (a *Application) PutRules(ctx context.Context, businessID string, rules []domain.AvailabilityRule) error {
	if strings.TrimSpace(businessID) == "" {
		return fmt.Errorf("%w: business_id is required", domain.ErrInvalid)
	}
	for i := range rules {
		if err := validateRule(&rules[i]); err != nil {
			return err
		}
	}
	return a.Rules.PutRules(ctx, businessID, rules, a.now())
}

func validateRule(r *domain.AvailabilityRule) error {
	if r.DayOfWeek < 0 || r.DayOfWeek > 6 {
		return fmt.Errorf("%w: day_of_week must be 0 (Sunday) through 6 (Saturday)", domain.ErrInvalid)
	}
	st := strings.TrimSpace(r.StartLocal)
	en := strings.TrimSpace(r.EndLocal)
	if st == "" || en == "" {
		return fmt.Errorf("%w: start_local and end_local are required (HH:MM)", domain.ErrInvalid)
	}
	stt, err := time.ParseInLocation("15:04", st, time.UTC)
	if err != nil {
		return fmt.Errorf("%w: start_local must be HH:MM", domain.ErrInvalid)
	}
	et, err := time.ParseInLocation("15:04", en, time.UTC)
	if err != nil {
		return fmt.Errorf("%w: end_local must be HH:MM", domain.ErrInvalid)
	}
	if !et.After(stt) {
		return fmt.Errorf("%w: end_local must be after start_local", domain.ErrInvalid)
	}
	return nil
}

// ListSlotsInput selects a UTC window and optional filters; slot duration comes from the service when serviceID is set.
type ListSlotsInput struct {
	BusinessID  string
	FromUTC     time.Time
	ToUTC       time.Time
	ServiceID   string
	StaffID     string
	SlotMinutes *int
}

// ListSlots merges rules in the business timezone and returns bookable slots in UTC.
func (a *Application) ListSlots(ctx context.Context, in ListSlotsInput) ([]schedule.Slot, error) {
	if strings.TrimSpace(in.BusinessID) == "" {
		return nil, fmt.Errorf("%w: business_id is required", domain.ErrInvalid)
	}
	if !in.ToUTC.After(in.FromUTC) {
		return nil, fmt.Errorf("%w: to must be after from (RFC3339 UTC recommended)", domain.ErrInvalid)
	}
	biz, err := a.Businesses.Get(ctx, in.BusinessID)
	if err != nil {
		return nil, err
	}
	loc := time.UTC
	if tz := strings.TrimSpace(biz.Timezone); tz != "" {
		var lerr error
		loc, lerr = time.LoadLocation(tz)
		if lerr != nil {
			loc = time.UTC
		}
	}
	rules, _, err := a.Rules.GetRules(ctx, in.BusinessID)
	if err != nil {
		return nil, err
	}
	if rules == nil {
		rules = []domain.AvailabilityRule{}
	}
	slotMin := defaultSlotMinutes
	if in.ServiceID != "" {
		svc, err := a.Services.Get(ctx, in.BusinessID, in.ServiceID)
		if err != nil {
			return nil, err
		}
		if svc.DurationMinutes > 0 {
			slotMin = svc.DurationMinutes
		}
	}
	if in.SlotMinutes != nil && *in.SlotMinutes > 0 {
		slotMin = *in.SlotMinutes
	}
	slotDur := time.Duration(slotMin) * time.Minute
	return schedule.BuildSlots(loc, rules, slotDur, in.FromUTC, in.ToUTC, in.ServiceID, in.StaffID)
}

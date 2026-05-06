package schedule

import (
	"sort"
	"strings"
	"time"

	"github.com/parama/booking/internal/domain"
)

// Slot is one bookable window in UTC (RFC3339 in JSON).
type Slot struct {
	StartAt   time.Time `json:"start_at"`
	EndAt     time.Time `json:"end_at"`
	StaffID   string    `json:"staff_id,omitempty"`
	ServiceID string    `json:"service_id,omitempty"`
}

type interval struct {
	start, end time.Time
}

// BuildSlots merges weekly rules in business-local time and emits non-overlapping slot windows per staff in UTC.
func BuildSlots(loc *time.Location, rules []domain.AvailabilityRule, slotDur time.Duration,
	windowFrom, windowTo time.Time, qServiceID, qStaffID string,
) ([]Slot, error) {
	if slotDur <= 0 {
		return nil, nil
	}
	if !windowTo.After(windowFrom) {
		return nil, nil
	}
	type groupKey struct {
		day       string // YYYY-MM-DD in loc
		staffSort string // effective staff for merge + output
	}
	buckets := map[groupKey][]interval{}
	for _, rule := range rules {
		if !ruleMatches(&rule, qServiceID, qStaffID) {
			continue
		}
		staffOut := rule.StaffID
		if staffOut == "" && qStaffID != "" {
			staffOut = qStaffID
		}
		locFrom := windowFrom.In(loc)
		locTo := windowTo.In(loc)
		startDate := time.Date(locFrom.Year(), locFrom.Month(), locFrom.Day(), 0, 0, 0, 0, loc)
		endDate := time.Date(locTo.Year(), locTo.Month(), locTo.Day(), 0, 0, 0, 0, loc)
		for day := startDate; !day.After(endDate); day = day.AddDate(0, 0, 1) {
			if int(day.Weekday()) != rule.DayOfWeek {
				continue
			}
			startStr := strings.TrimSpace(rule.StartLocal)
			endStr := strings.TrimSpace(rule.EndLocal)
			st, err := time.ParseInLocation("2006-01-02 15:04", day.Format("2006-01-02")+" "+startStr, loc)
			if err != nil {
				continue
			}
			et, err := time.ParseInLocation("2006-01-02 15:04", day.Format("2006-01-02")+" "+endStr, loc)
			if err != nil {
				continue
			}
			if !et.After(st) {
				continue
			}
			segStart := st.UTC()
			segEnd := et.UTC()
			if segEnd.Before(windowFrom) || !segStart.Before(windowTo) {
				continue
			}
			if segStart.Before(windowFrom) {
				segStart = windowFrom
			}
			if segEnd.After(windowTo) {
				segEnd = windowTo
			}
			if segEnd.Sub(segStart) < slotDur {
				continue
			}
			dayKey := day.Format("2006-01-02")
			gk := groupKey{day: dayKey, staffSort: staffOut}
			buckets[gk] = append(buckets[gk], interval{start: segStart, end: segEnd})
		}
	}
	var out []Slot
	for gk, ivs := range buckets {
		merged := mergeIntervals(ivs)
		staffOut := gk.staffSort
		svcOut := qServiceID
		for _, m := range merged {
			t := m.start
			for {
				slotEnd := t.Add(slotDur)
				if slotEnd.After(m.end) {
					break
				}
				out = append(out, Slot{
					StartAt:   t,
					EndAt:     slotEnd,
					StaffID:   staffOut,
					ServiceID: svcOut,
				})
				t = slotEnd
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartAt.Before(out[j].StartAt) })
	return out, nil
}

func ruleMatches(r *domain.AvailabilityRule, qService, qStaff string) bool {
	if qService != "" && r.ServiceID != "" && r.ServiceID != qService {
		return false
	}
	if qStaff != "" && r.StaffID != "" && r.StaffID != qStaff {
		return false
	}
	return true
}

func mergeIntervals(ivs []interval) []interval {
	if len(ivs) == 0 {
		return nil
	}
	sort.Slice(ivs, func(i, j int) bool {
		if ivs[i].start.Equal(ivs[j].start) {
			return ivs[i].end.Before(ivs[j].end)
		}
		return ivs[i].start.Before(ivs[j].start)
	})
	var out []interval
	cur := ivs[0]
	for i := 1; i < len(ivs); i++ {
		nx := ivs[i]
		if !nx.start.After(cur.end) {
			if nx.end.After(cur.end) {
				cur.end = nx.end
			}
		} else {
			out = append(out, cur)
			cur = nx
		}
	}
	out = append(out, cur)
	return out
}

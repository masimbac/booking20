package domain

// AvailabilityRule is a weekly block in business-local time (day 0 = Sunday … 6 = Saturday).
type AvailabilityRule struct {
	StaffID    string         `json:"staff_id,omitempty"`
	ServiceID  string         `json:"service_id,omitempty"`
	DayOfWeek  int            `json:"day_of_week"`
	StartLocal string         `json:"start_local"`
	EndLocal   string         `json:"end_local"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

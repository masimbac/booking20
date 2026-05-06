package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/parama/booking/internal/app/customers"
	"github.com/parama/booking/internal/app/ports"
	"github.com/parama/booking/internal/app/scheduling"
	"github.com/parama/booking/internal/domain"
)

func (d *Deps) listCustomers(w http.ResponseWriter, r *http.Request) {
	bid := chi.URLParam(r, "businessId")
	items, next, err := d.Customers.ListCustomers(r.Context(), bid, ports.ListCustomersOptions{
		Limit:  parseLimit32(r, 20, 100),
		Cursor: r.URL.Query().Get("cursor"),
	})
	if mapAppErr(w, r, err) {
		return
	}
	resp := map[string]any{"items": items}
	if next != "" {
		resp["next_cursor"] = next
	}
	writeJSON(w, http.StatusOK, resp)
}

type createCustomerReq struct {
	PhoneE164      string         `json:"phone_e164"`
	DisplayName    string         `json:"display_name,omitempty"`
	Preferences    map[string]any `json:"preferences,omitempty"`
	MarketingOptIn *bool          `json:"marketing_opt_in,omitempty"`
}

func (d *Deps) postCustomer(w http.ResponseWriter, r *http.Request) {
	bid := chi.URLParam(r, "businessId")
	var body createCustomerReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteUnprocessable(w, r, "invalid JSON body")
		return
	}
	out, err := d.Customers.CreateCustomer(r.Context(), customers.CreateCustomerInput{
		BusinessID:     bid,
		PhoneE164:      body.PhoneE164,
		DisplayName:    body.DisplayName,
		Preferences:    body.Preferences,
		MarketingOptIn: body.MarketingOptIn,
	})
	if mapAppErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (d *Deps) getCustomerByPhone(w http.ResponseWriter, r *http.Request) {
	bid := chi.URLParam(r, "businessId")
	phone := r.URL.Query().Get("phone")
	out, err := d.Customers.GetCustomerByPhone(r.Context(), bid, phone)
	if mapAppErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (d *Deps) getCustomer(w http.ResponseWriter, r *http.Request) {
	bid := chi.URLParam(r, "businessId")
	cid := chi.URLParam(r, "customerId")
	out, err := d.Customers.GetCustomer(r.Context(), bid, cid)
	if mapAppErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, out)
}

type patchCustomerBody struct {
	PhoneE164      *string        `json:"phone_e164,omitempty"`
	DisplayName    *string        `json:"display_name,omitempty"`
	Preferences    map[string]any `json:"preferences,omitempty"`
	MarketingOptIn *bool          `json:"marketing_opt_in,omitempty"`
}

func (d *Deps) patchCustomer(w http.ResponseWriter, r *http.Request) {
	bid := chi.URLParam(r, "businessId")
	cid := chi.URLParam(r, "customerId")
	var body patchCustomerBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteUnprocessable(w, r, "invalid JSON body")
		return
	}
	out, err := d.Customers.PatchCustomer(r.Context(), bid, cid, customers.PatchCustomerInput{
		PhoneE164:      body.PhoneE164,
		DisplayName:    body.DisplayName,
		Preferences:    body.Preferences,
		MarketingOptIn: body.MarketingOptIn,
	})
	if mapAppErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, out)
}

type putAvailabilityRulesBody struct {
	Rules []domain.AvailabilityRule `json:"rules"`
}

func (d *Deps) putAvailabilityRules(w http.ResponseWriter, r *http.Request) {
	bid := chi.URLParam(r, "businessId")
	var body putAvailabilityRulesBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteUnprocessable(w, r, "invalid JSON body")
		return
	}
	if err := d.Scheduling.PutRules(r.Context(), bid, body.Rules); mapAppErr(w, r, err) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (d *Deps) getAvailabilitySlots(w http.ResponseWriter, r *http.Request) {
	bid := chi.URLParam(r, "businessId")
	fromS := r.URL.Query().Get("from")
	toS := r.URL.Query().Get("to")
	if fromS == "" || toS == "" {
		WriteUnprocessable(w, r, "from and to query parameters are required (RFC3339)")
		return
	}
	fromT, err := time.Parse(time.RFC3339, fromS)
	if err != nil {
		WriteUnprocessable(w, r, "from must be RFC3339")
		return
	}
	toT, err := time.Parse(time.RFC3339, toS)
	if err != nil {
		WriteUnprocessable(w, r, "to must be RFC3339")
		return
	}
	in := scheduling.ListSlotsInput{
		BusinessID: bid,
		FromUTC:    fromT.UTC(),
		ToUTC:      toT.UTC(),
		ServiceID:  r.URL.Query().Get("service_id"),
		StaffID:    r.URL.Query().Get("staff_id"),
	}
	if s := r.URL.Query().Get("slot_minutes"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 {
			WriteUnprocessable(w, r, "slot_minutes must be a positive integer")
			return
		}
		in.SlotMinutes = &n
	}
	items, err := d.Scheduling.ListSlots(r.Context(), in)
	if mapAppErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/parama/booking/internal/app/catalog"
	"github.com/parama/booking/internal/app/ports"
	"github.com/parama/booking/internal/domain"
)

func parseLimit32(r *http.Request, def, max int32) int32 {
	s := r.URL.Query().Get("limit")
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return def
	}
	v := int32(n)
	if v > max {
		return max
	}
	return v
}

func parseActiveOnly(r *http.Request) bool {
	v := r.URL.Query().Get("active_only")
	if v == "" {
		return true
	}
	return v == "true" || v == "1"
}

type moneyReq struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

type createServiceReq struct {
	Name            string         `json:"name"`
	DurationMinutes int            `json:"duration_minutes"`
	Price           *moneyReq      `json:"price,omitempty"`
	Active          *bool          `json:"active,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

func (d *Deps) listServices(w http.ResponseWriter, r *http.Request) {
	bid := chi.URLParam(r, "businessId")
	cursor := r.URL.Query().Get("cursor")
	items, next, err := d.Catalog.ListServices(r.Context(), bid, ports.ListServicesOptions{
		Limit:      parseLimit32(r, 20, 100),
		Cursor:     cursor,
		ActiveOnly: parseActiveOnly(r),
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

func (d *Deps) postService(w http.ResponseWriter, r *http.Request) {
	bid := chi.URLParam(r, "businessId")
	var body createServiceReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteUnprocessable(w, r, "invalid JSON body")
		return
	}
	in := catalog.CreateServiceInput{
		BusinessID:      bid,
		Name:            body.Name,
		DurationMinutes: body.DurationMinutes,
		Active:          body.Active,
		Metadata:        body.Metadata,
	}
	if body.Price != nil {
		in.Price = &domain.Money{Amount: body.Price.Amount, Currency: body.Price.Currency}
	}
	out, err := d.Catalog.CreateService(r.Context(), in)
	if mapAppErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (d *Deps) getService(w http.ResponseWriter, r *http.Request) {
	bid := chi.URLParam(r, "businessId")
	sid := chi.URLParam(r, "serviceId")
	out, err := d.Catalog.GetService(r.Context(), bid, sid)
	if mapAppErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, out)
}

type patchServiceBody struct {
	Name            *string        `json:"name,omitempty"`
	DurationMinutes *int           `json:"duration_minutes,omitempty"`
	Price           *moneyReq      `json:"price,omitempty"`
	Active          *bool          `json:"active,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

func (d *Deps) patchService(w http.ResponseWriter, r *http.Request) {
	bid := chi.URLParam(r, "businessId")
	sid := chi.URLParam(r, "serviceId")
	var body patchServiceBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteUnprocessable(w, r, "invalid JSON body")
		return
	}
	patch := catalog.PatchServiceInput{
		Name:            body.Name,
		DurationMinutes: body.DurationMinutes,
		Active:          body.Active,
		Metadata:        body.Metadata,
	}
	if body.Price != nil {
		patch.Price = &domain.Money{Amount: body.Price.Amount, Currency: body.Price.Currency}
	}
	out, err := d.Catalog.PatchService(r.Context(), bid, sid, patch)
	if mapAppErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (d *Deps) deleteService(w http.ResponseWriter, r *http.Request) {
	bid := chi.URLParam(r, "businessId")
	sid := chi.URLParam(r, "serviceId")
	if err := d.Catalog.DeleteService(r.Context(), bid, sid); mapAppErr(w, r, err) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type createStaffReq struct {
	DisplayName string         `json:"display_name"`
	Role        string         `json:"role,omitempty"`
	ServiceIDs  []string       `json:"service_ids,omitempty"`
	Active      *bool          `json:"active,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

func (d *Deps) listStaff(w http.ResponseWriter, r *http.Request) {
	bid := chi.URLParam(r, "businessId")
	items, next, err := d.Catalog.ListStaff(r.Context(), bid, ports.ListStaffOptions{
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

func (d *Deps) postStaff(w http.ResponseWriter, r *http.Request) {
	bid := chi.URLParam(r, "businessId")
	var body createStaffReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteUnprocessable(w, r, "invalid JSON body")
		return
	}
	out, err := d.Catalog.CreateStaff(r.Context(), catalog.CreateStaffInput{
		BusinessID:   bid,
		DisplayName:  body.DisplayName,
		Role:         body.Role,
		ServiceIDs:   body.ServiceIDs,
		Active:       body.Active,
		Metadata:     body.Metadata,
	})
	if mapAppErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (d *Deps) getStaff(w http.ResponseWriter, r *http.Request) {
	bid := chi.URLParam(r, "businessId")
	sid := chi.URLParam(r, "staffId")
	out, err := d.Catalog.GetStaff(r.Context(), bid, sid)
	if mapAppErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, out)
}

type patchStaffBody struct {
	DisplayName *string        `json:"display_name,omitempty"`
	Role        *string        `json:"role,omitempty"`
	ServiceIDs  *[]string      `json:"service_ids,omitempty"`
	Active      *bool          `json:"active,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

func (d *Deps) patchStaff(w http.ResponseWriter, r *http.Request) {
	bid := chi.URLParam(r, "businessId")
	sid := chi.URLParam(r, "staffId")
	var body patchStaffBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteUnprocessable(w, r, "invalid JSON body")
		return
	}
	patch := catalog.PatchStaffInput{
		DisplayName: body.DisplayName,
		Role:        body.Role,
		ServiceIDs:  body.ServiceIDs,
		Active:      body.Active,
		Metadata:    body.Metadata,
	}
	out, err := d.Catalog.PatchStaff(r.Context(), bid, sid, patch)
	if mapAppErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (d *Deps) deleteStaff(w http.ResponseWriter, r *http.Request) {
	bid := chi.URLParam(r, "businessId")
	sid := chi.URLParam(r, "staffId")
	if err := d.Catalog.DeleteStaff(r.Context(), bid, sid); mapAppErr(w, r, err) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

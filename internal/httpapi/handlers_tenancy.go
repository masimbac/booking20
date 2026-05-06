package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/parama/booking/internal/app/tenancy"
	"github.com/parama/booking/internal/domain"
)

type registerBusinessReq struct {
	Name      string         `json:"name"`
	LegalName string         `json:"legal_name,omitempty"`
	Timezone  string         `json:"timezone"`
	Contact   map[string]any `json:"contact,omitempty"`
}

func (d *Deps) postPlatformBusinesses(w http.ResponseWriter, r *http.Request) {
	var body registerBusinessReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteUnprocessable(w, r, "invalid JSON body")
		return
	}
	out, err := d.Tenancy.RegisterBusiness(r.Context(), tenancy.RegisterBusinessInput{
		Name:      body.Name,
		LegalName: body.LegalName,
		Timezone:  body.Timezone,
		Contact:   body.Contact,
	})
	if mapAppErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

type patchBusinessBody struct {
	Name      *string                 `json:"name,omitempty"`
	LegalName *string                 `json:"legal_name,omitempty"`
	Timezone  *string                 `json:"timezone,omitempty"`
	Contact   map[string]any          `json:"contact,omitempty"`
	Status    *string                 `json:"status,omitempty"`
}

func (d *Deps) getBusiness(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "businessId")
	b, err := d.Tenancy.GetBusiness(r.Context(), id)
	if mapAppErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (d *Deps) patchBusiness(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "businessId")
	var body patchBusinessBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteUnprocessable(w, r, "invalid JSON body")
		return
	}
	patch := tenancy.PatchBusinessInput{
		Name:      body.Name,
		LegalName: body.LegalName,
		Timezone:  body.Timezone,
		Contact:   body.Contact,
	}
	if body.Status != nil && *body.Status != "" {
		switch domain.BusinessStatus(*body.Status) {
		case domain.BusinessActive, domain.BusinessSuspended:
			st := domain.BusinessStatus(*body.Status)
			patch.Status = &st
		default:
			WriteUnprocessable(w, r, "status must be active or suspended")
			return
		}
	}
	b, err := d.Tenancy.PatchBusiness(r.Context(), id, patch)
	if mapAppErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, b)
}

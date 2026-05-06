package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/parama/booking/internal/app/bookings"
	"github.com/parama/booking/internal/app/ports"
)

func (d *Deps) listBookings(w http.ResponseWriter, r *http.Request) {
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
	items, next, err := d.Bookings.ListBookings(r.Context(), bid, fromT.UTC(), toT.UTC(), ports.ListBookingsOptions{
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

type createBookingReq struct {
	CustomerID string     `json:"customer_id"`
	ServiceID  string     `json:"service_id"`
	StaffID    string     `json:"staff_id,omitempty"`
	StartAt    string     `json:"start_at"`
	EndAt      *string    `json:"end_at,omitempty"`
}

func (d *Deps) postBooking(w http.ResponseWriter, r *http.Request) {
	bid := chi.URLParam(r, "businessId")
	idem := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idem == "" {
		WriteUnprocessable(w, r, "Idempotency-Key header is required")
		return
	}
	var body createBookingReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteUnprocessable(w, r, "invalid JSON body")
		return
	}
	startAt, err := time.Parse(time.RFC3339, body.StartAt)
	if err != nil {
		WriteUnprocessable(w, r, "start_at must be RFC3339")
		return
	}
	var endPtr *time.Time
	if body.EndAt != nil && *body.EndAt != "" {
		et, err := time.Parse(time.RFC3339, *body.EndAt)
		if err != nil {
			WriteUnprocessable(w, r, "end_at must be RFC3339")
			return
		}
		endPtr = &et
	}
	out, created, err := d.Bookings.CreateBooking(r.Context(), idem, bookings.CreateBookingInput{
		BusinessID: bid,
		CustomerID: body.CustomerID,
		ServiceID:  body.ServiceID,
		StaffID:    body.StaffID,
		StartAt:    startAt,
		EndAt:      endPtr,
	})
	if mapAppErr(w, r, err) {
		return
	}
	if created {
		writeJSON(w, http.StatusCreated, out)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (d *Deps) getBooking(w http.ResponseWriter, r *http.Request) {
	bid := chi.URLParam(r, "businessId")
	bookID := chi.URLParam(r, "bookingId")
	out, err := d.Bookings.GetBooking(r.Context(), bid, bookID)
	if mapAppErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (d *Deps) postConfirmBooking(w http.ResponseWriter, r *http.Request) {
	bid := chi.URLParam(r, "businessId")
	bookID := chi.URLParam(r, "bookingId")
	out, err := d.Bookings.ConfirmBooking(r.Context(), bid, bookID)
	if mapAppErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, out)
}

type cancelBookingBody struct {
	Reason string `json:"reason,omitempty"`
}

func (d *Deps) postCancelBooking(w http.ResponseWriter, r *http.Request) {
	bid := chi.URLParam(r, "businessId")
	bookID := chi.URLParam(r, "bookingId")
	var body cancelBookingBody
	_ = json.NewDecoder(r.Body).Decode(&body)
	out, err := d.Bookings.CancelBooking(r.Context(), bid, bookID, bookings.CancelBookingInput{Reason: body.Reason})
	if mapAppErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (d *Deps) postCompleteBooking(w http.ResponseWriter, r *http.Request) {
	bid := chi.URLParam(r, "businessId")
	bookID := chi.URLParam(r, "bookingId")
	out, err := d.Bookings.CompleteBooking(r.Context(), bid, bookID)
	if mapAppErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (d *Deps) postNoShowBooking(w http.ResponseWriter, r *http.Request) {
	bid := chi.URLParam(r, "businessId")
	bookID := chi.URLParam(r, "bookingId")
	out, err := d.Bookings.NoShowBooking(r.Context(), bid, bookID)
	if mapAppErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, out)
}

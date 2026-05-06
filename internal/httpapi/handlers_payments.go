package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/parama/booking/internal/app/payments"
	"github.com/parama/booking/internal/app/ports"
	"github.com/parama/booking/internal/domain"
)

type createPaymentReq struct {
	BookingID string       `json:"booking_id"`
	Amount    domain.Money `json:"amount"`
	Kind      string       `json:"kind"`
	Provider  string       `json:"provider,omitempty"`
	ReturnURL string       `json:"return_url,omitempty"`
}

func (d *Deps) postPayment(w http.ResponseWriter, r *http.Request) {
	bid := chi.URLParam(r, "businessId")
	idem := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idem == "" {
		WriteUnprocessable(w, r, "Idempotency-Key header is required")
		return
	}
	var body createPaymentReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteUnprocessable(w, r, "invalid JSON body")
		return
	}
	kind := domain.PaymentKind(strings.TrimSpace(strings.ToLower(body.Kind)))
	out, created, err := d.Payments.CreatePayment(r.Context(), idem, payments.CreatePaymentInput{
		BusinessID: bid,
		BookingID:  body.BookingID,
		Amount:     &body.Amount,
		Kind:       kind,
		Provider:   body.Provider,
		ReturnURL:  body.ReturnURL,
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

func (d *Deps) getPayment(w http.ResponseWriter, r *http.Request) {
	bid := chi.URLParam(r, "businessId")
	pid := chi.URLParam(r, "paymentId")
	out, err := d.Payments.GetPayment(r.Context(), bid, pid)
	if mapAppErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (d *Deps) listPaymentsForBooking(w http.ResponseWriter, r *http.Request) {
	bid := chi.URLParam(r, "businessId")
	bookID := chi.URLParam(r, "bookingId")
	items, next, err := d.Payments.ListPaymentsForBooking(r.Context(), bid, bookID, ports.ListPaymentsOptions{
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

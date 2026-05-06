package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/parama/booking/internal/app/notifications"
	"github.com/parama/booking/internal/domain"
)

func (d *Deps) listNotifications(w http.ResponseWriter, r *http.Request) {
	bid := chi.URLParam(r, "businessId")
	status := r.URL.Query().Get("status")
	if status != "" {
		switch domain.NotificationStatus(status) {
		case domain.NotificationScheduled, domain.NotificationSent, domain.NotificationFailed:
		default:
			WriteUnprocessable(w, r, "status must be scheduled, sent, or failed")
			return
		}
	}
	items, next, err := d.Notifications.List(r.Context(), bid, status, parseLimit32(r, 20, 100), r.URL.Query().Get("cursor"))
	if mapAppErr(w, r, err) {
		return
	}
	resp := map[string]any{"items": items}
	if next != "" {
		resp["next_cursor"] = next
	}
	writeJSON(w, http.StatusOK, resp)
}

type createNotificationReq struct {
	Kind         string         `json:"kind"`
	Channel      string         `json:"channel"`
	ScheduledAt  string         `json:"scheduled_at"`
	CustomerID   string         `json:"customer_id,omitempty"`
	BookingID    string         `json:"booking_id,omitempty"`
	Payload      map[string]any `json:"payload,omitempty"`
}

func (d *Deps) postNotification(w http.ResponseWriter, r *http.Request) {
	bid := chi.URLParam(r, "businessId")
	var body createNotificationReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteUnprocessable(w, r, "invalid JSON body")
		return
	}
	when, err := time.Parse(time.RFC3339Nano, body.ScheduledAt)
	if err != nil {
		when, err = time.Parse(time.RFC3339, body.ScheduledAt)
		if err != nil {
			WriteUnprocessable(w, r, "scheduled_at must be RFC3339 or RFC3339Nano")
			return
		}
	}
	out, err := d.Notifications.Create(r.Context(), notifications.CreateNotificationInput{
		BusinessID:  bid,
		Kind:        domain.NotificationKind(body.Kind),
		Channel:     body.Channel,
		ScheduledAt: when,
		CustomerID:  body.CustomerID,
		BookingID:   body.BookingID,
		Payload:     body.Payload,
	})
	if mapAppErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (d *Deps) postDispatchDueNotifications(w http.ResponseWriter, r *http.Request) {
	n, err := d.Notifications.DispatchDue(r.Context(), 50)
	if mapAppErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"dispatched": n})
}

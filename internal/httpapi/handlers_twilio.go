package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/parama/booking/internal/domain"
)

func (d *Deps) postTwilioWebhook(w http.ResponseWriter, r *http.Request) {
	if d.Twilio == nil {
		WriteProblem(w, r, ProblemInput{
			Status: http.StatusNotImplemented,
			Title:  "Not Implemented",
			Detail: "twilio webhook not wired",
		})
		return
	}
	if err := r.ParseForm(); err != nil {
		WriteProblem(w, r, ProblemInput{
			Status: http.StatusBadRequest,
			Title:  "Bad Request",
			Detail: "could not parse form body",
		})
		return
	}
	webhookURL := twilioWebhookURL(r)
	if err := d.Twilio.HandleInbound(r.Context(), webhookURL, r.PostForm, r.Header.Get("X-Twilio-Signature")); err != nil {
		st := twilioHookHTTPStatus(err)
		WriteProblem(w, r, ProblemInput{
			Status: st,
			Title:  http.StatusText(st),
			Detail: err.Error(),
		})
		return
	}
	w.WriteHeader(http.StatusOK)
}

func twilioWebhookURL(r *http.Request) string {
	scheme := "https"
	if proto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); proto != "" {
		scheme = proto
	} else if r.TLS == nil && strings.EqualFold(r.Header.Get("X-Forwarded-SSL"), "on") {
		scheme = "https"
	}
	host := r.Host
	if h := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); h != "" {
		host = h
	}
	return scheme + "://" + host + r.URL.RequestURI()
}

func twilioHookHTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	if errors.Is(err, domain.ErrInvalid) {
		return http.StatusUnauthorized
	}
	return http.StatusBadRequest
}

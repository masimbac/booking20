package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/parama/booking/internal/app/payments"
	"github.com/parama/booking/internal/domain"
)

type paymentWebhookBody struct {
	BusinessID  string `json:"business_id"`
	PaymentID   string `json:"payment_id"`
	Status      string `json:"status"`
	ExternalRef string `json:"external_ref"`
}

func (d *Deps) postPaymentProviderWebhook(w http.ResponseWriter, r *http.Request) {
	if d.Payments == nil {
		WriteProblem(w, r, ProblemInput{
			Status: http.StatusNotImplemented,
			Title:  "Not Implemented",
			Detail: "payments not wired",
		})
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		WriteProblem(w, r, ProblemInput{
			Status: http.StatusBadRequest,
			Title:  "Bad Request",
			Detail: "could not read body",
		})
		return
	}
	secret := strings.TrimSpace(os.Getenv("PAYMENT_WEBHOOK_SECRET"))
	if secret != "" {
		sig := strings.TrimSpace(r.Header.Get("X-Payment-Signature"))
		if !strings.HasPrefix(sig, "sha256=") || !verifyHMACSHA256(secret, body, strings.TrimPrefix(sig, "sha256=")) {
			WriteProblem(w, r, ProblemInput{
				Status: http.StatusUnauthorized,
				Title:  "Unauthorized",
				Detail: "invalid webhook signature",
			})
			return
		}
	}
	var raw paymentWebhookBody
	if err := json.Unmarshal(body, &raw); err != nil {
		WriteUnprocessable(w, r, "invalid webhook json")
		return
	}
	st := domain.PaymentStatus(strings.TrimSpace(strings.ToLower(raw.Status)))
	switch st {
	case domain.PaymentSucceeded, domain.PaymentFailed, domain.PaymentRefunded, domain.PaymentPending:
	default:
		WriteUnprocessable(w, r, "status must be pending|succeeded|failed|refunded")
		return
	}
	_, err = d.Payments.RecordWebhook(r.Context(), payments.WebhookUpdateInput{
		BusinessID:  raw.BusinessID,
		PaymentID:   raw.PaymentID,
		Status:      st,
		ExternalRef: raw.ExternalRef,
	})
	if mapAppErr(w, r, err) {
		return
	}
	w.WriteHeader(http.StatusOK)
}

func verifyHMACSHA256(secret string, body []byte, wantHex string) bool {
	want, err := hex.DecodeString(wantHex)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return subtle.ConstantTimeCompare(mac.Sum(nil), want) == 1
}

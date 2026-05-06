package httpapi

import (
	"io"
	"net/http"

	"github.com/parama/booking/internal/app/conversations"
)

func (d *Deps) postWhatsAppWebhook(w http.ResponseWriter, r *http.Request) {
	if d.Conversations == nil {
		WriteProblem(w, r, ProblemInput{
			Status: http.StatusNotImplemented,
			Title:  "Not Implemented",
			Detail: "whatsapp webhook not wired",
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
	if !d.Conversations.VerifyWhatsAppSignature(body, r.Header.Get("X-Hub-Signature-256")) {
		WriteProblem(w, r, ProblemInput{
			Status: http.StatusUnauthorized,
			Title:  "Unauthorized",
			Detail: "invalid webhook signature",
		})
		return
	}
	if err := d.Conversations.HandleWhatsAppWebhook(r.Context(), body); err != nil {
		st := conversations.HookHTTPStatus(err)
		WriteProblem(w, r, ProblemInput{
			Status: st,
			Title:  http.StatusText(st),
			Detail: err.Error(),
		})
		return
	}
	w.WriteHeader(http.StatusOK)
}

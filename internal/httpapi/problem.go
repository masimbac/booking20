package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

const contentTypeProblem = "application/problem+json"

// Problem is RFC 7807 Problem Details with project extensions.
type Problem struct {
	Type       string `json:"type,omitempty"`
	Title      string `json:"title"`
	Status     int    `json:"status"`
	Detail     string `json:"detail,omitempty"`
	Instance   string `json:"instance,omitempty"`
	BusinessID string `json:"business_id,omitempty"`
	RequestID  string `json:"request_id,omitempty"`
}

// ProblemInput carries fields before RequestID enrichment from context.
type ProblemInput struct {
	Type       string
	Title      string
	Status     int
	Detail     string
	Instance   string
	BusinessID string
}

// WriteProblem writes application/problem+json.
func WriteProblem(w http.ResponseWriter, r *http.Request, in ProblemInput) {
	reqID := middleware.GetReqID(r.Context())
	p := Problem{
		Type:       in.Type,
		Title:      in.Title,
		Status:     in.Status,
		Detail:     in.Detail,
		Instance:   in.Instance,
		BusinessID: in.BusinessID,
		RequestID:  reqID,
	}
	w.Header().Set("Content-Type", contentTypeProblem)
	w.WriteHeader(in.Status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	_ = enc.Encode(p)
}

// WriteConflict writes 409 Problem (reserved for domain invariants).
func WriteConflict(w http.ResponseWriter, r *http.Request, detail, businessID string) {
	WriteProblem(w, r, ProblemInput{
		Status:     http.StatusConflict,
		Title:      "Conflict",
		Detail:     detail,
		BusinessID: businessID,
	})
}

// WriteUnprocessable writes 422 Problem (validation / domain rule failure).
func WriteUnprocessable(w http.ResponseWriter, r *http.Request, detail string) {
	WriteProblem(w, r, ProblemInput{
		Status: http.StatusUnprocessableEntity,
		Title:  "Unprocessable Entity",
		Detail: detail,
	})
}

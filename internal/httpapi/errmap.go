package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/parama/booking/internal/domain"
)

func mapAppErr(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, domain.ErrNotFound):
		WriteProblem(w, r, ProblemInput{
			Status: http.StatusNotFound,
			Title:  "Not Found",
			Detail: "resource not found",
		})
	case errors.Is(err, domain.ErrConflict):
		WriteConflict(w, r, err.Error(), "")
	case errors.Is(err, domain.ErrInvalid):
		WriteUnprocessable(w, r, err.Error())
	default:
		WriteProblem(w, r, ProblemInput{
			Status: http.StatusInternalServerError,
			Title:  "Internal Server Error",
			Detail: "unexpected error",
		})
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	_ = enc.Encode(v)
}

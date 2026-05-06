package httpapi

import (
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
)

// StripAPIStagePrefix removes the API Gateway stage segment (e.g. /dev) when present.
func StripAPIStagePrefix(stage string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if stage == "" {
			return next
		}
		prefix := "/" + stage
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, prefix+"/") || r.URL.Path == prefix {
				r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
				if r.URL.Path == "" {
					r.URL.Path = "/"
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ProblemRecoverer turns panics into 500 application/problem+json (detail is generic).
func ProblemRecoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rv := recover(); rv != nil {
				reqID := middleware.GetReqID(r.Context())
				log.Printf("http panic request_id=%s err=%v path=%s", reqID, rv, r.URL.Path)
				WriteProblem(w, r, ProblemInput{
					Status: http.StatusInternalServerError,
					Title:  "Internal Server Error",
					Detail: "an unexpected error occurred",
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

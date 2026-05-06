package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// RouterConfig wires HTTP behavior for the API Lambda.
type RouterConfig struct {
	// Stage is the API Gateway stage name (e.g. dev). When set, /{stage} is stripped from incoming paths.
	Phase string
	Stage string
}

// NewRouter builds the chi mux for REST + Lambda (Phase 2: stubs + health only).
func NewRouter(cfg RouterConfig) *chi.Mux {
	r := chi.NewRouter()

	r.Use(StripAPIStagePrefix(cfg.Stage))
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(ProblemRecoverer)
	r.Use(middleware.Logger)

	r.Route("/v1", func(r chi.Router) {
		r.Get("/health", healthHandler(cfg.Phase))
		registerStubRoutes(r)
	})

	r.NotFound(notFoundHandler)
	r.MethodNotAllowed(methodNotAllowedHandler)

	return r
}

func healthHandler(phase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
			"phase":  phase,
		})
	}
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	WriteProblem(w, r, ProblemInput{
		Status: http.StatusNotFound,
		Title:  "Not Found",
		Detail: "no matching resource for " + r.Method + " on this API",
	})
}

func methodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	WriteProblem(w, r, ProblemInput{
		Status: http.StatusMethodNotAllowed,
		Title:  "Method Not Allowed",
		Detail: r.Method + " is not supported for this resource",
	})
}

func stub501(detail string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		WriteProblem(w, r, ProblemInput{
			Status: http.StatusNotImplemented,
			Title:  "Not Implemented",
			Detail: detail,
		})
	}
}

func registerStubRoutes(r chi.Router) {
	r.Post("/platform/businesses", stub501("POST /platform/businesses not implemented yet"))

	r.Route("/businesses/{businessId}", func(r chi.Router) {
		r.Get("/", stub501("GET /businesses/{businessId} not implemented yet"))
		r.Patch("/", stub501("PATCH /businesses/{businessId} not implemented yet"))

		r.Route("/services", func(r chi.Router) {
			r.Get("/", stub501("list services"))
			r.Post("/", stub501("create service"))
			r.Route("/{serviceId}", func(r chi.Router) {
				r.Get("/", stub501("get service"))
				r.Patch("/", stub501("patch service"))
				r.Delete("/", stub501("delete service"))
			})
		})

		r.Route("/staff", func(r chi.Router) {
			r.Get("/", stub501("list staff"))
			r.Post("/", stub501("create staff"))
			r.Route("/{staffId}", func(r chi.Router) {
				r.Get("/", stub501("get staff"))
				r.Patch("/", stub501("patch staff"))
				r.Delete("/", stub501("delete staff"))
			})
		})

		r.Route("/customers", func(r chi.Router) {
			r.Get("/", stub501("list customers"))
			r.Post("/", stub501("create customer"))
			r.Get("/by-phone", stub501("get customer by phone"))
			r.Route("/{customerId}", func(r chi.Router) {
				r.Get("/", stub501("get customer"))
				r.Patch("/", stub501("patch customer"))
			})
		})

		r.Route("/availability", func(r chi.Router) {
			r.Put("/rules", stub501("put availability rules"))
			r.Get("/slots", stub501("get availability slots"))
		})

		r.Route("/bookings", func(r chi.Router) {
			r.Get("/", stub501("list bookings"))
			r.Post("/", stub501("create booking"))
			r.Route("/{bookingId}", func(r chi.Router) {
				r.Get("/", stub501("get booking"))
				r.Post("/confirm", stub501("confirm booking"))
				r.Post("/cancel", stub501("cancel booking"))
				r.Post("/complete", stub501("complete booking"))
				r.Post("/no-show", stub501("mark no-show"))
				r.Route("/payments", func(r chi.Router) {
					r.Get("/", stub501("list payments for booking"))
				})
			})
		})

		r.Route("/payments", func(r chi.Router) {
			r.Post("/", stub501("create payment"))
			r.Route("/{paymentId}", func(r chi.Router) {
				r.Get("/", stub501("get payment"))
			})
		})

		r.Route("/conversations", func(r chi.Router) {
			r.Post("/", stub501("ensure conversation"))
			r.Route("/{conversationId}", func(r chi.Router) {
				r.Get("/", stub501("get conversation"))
				r.Route("/messages", func(r chi.Router) {
					r.Get("/", stub501("list messages"))
					r.Post("/", stub501("create outbound message"))
				})
			})
		})

		r.Route("/notifications", func(r chi.Router) {
			r.Get("/", stub501("list notifications"))
			r.Post("/", stub501("schedule notification"))
		})
	})

	r.Route("/webhooks", func(r chi.Router) {
		r.Post("/whatsapp", stub501("whatsapp webhook"))
		r.Post("/payments/{provider}", stub501("payment provider webhook"))
	})
}

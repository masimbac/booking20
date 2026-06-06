package httpapi

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// RouterConfig wires HTTP behavior for the API Lambda.
type RouterConfig struct {
	Phase     string
	Stage     string
	Hardening HardeningConfig
}

// NewRouter builds the chi mux. When deps.Tenancy and deps.Catalog are set, Phase 3
// tenancy + catalog routes are registered; otherwise Phase 2 stubs are used. Customer and
// scheduling routes activate when those application services are non-nil.
func NewRouter(cfg RouterConfig, deps *Deps) *chi.Mux {
	if deps == nil {
		deps = &Deps{}
	}
	r := chi.NewRouter()

	var lim *slidingLimiter
	if cfg.Hardening.RateLimitMax > 0 && cfg.Hardening.RateLimitWindow > 0 {
		lim = newSlidingLimiter(cfg.Hardening.RateLimitMax, cfg.Hardening.RateLimitWindow)
	}

	r.Use(StripAPIStagePrefix(cfg.Stage))
	r.Use(middleware.RequestID)
	r.Use(structuredAccessLog)
	r.Use(middleware.RealIP)
	r.Use(rateLimitMiddleware(lim))
	r.Use(corsMiddleware(cfg.Hardening.CORSAllowedOrigins))
	r.Use(securityHeaders)
	r.Use(ProblemRecoverer)

	r.Route("/v1", func(r chi.Router) {
		r.Get("/health", healthHandler(cfg.Phase))
		if deps.Tenancy != nil && deps.Catalog != nil {
			registerPhase3Routes(r, deps)
		} else {
			registerStubRoutes(r, deps)
		}
	})

	r.NotFound(notFoundHandler)
	r.MethodNotAllowed(methodNotAllowedHandler)

	return r
}

func healthHandler(phase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
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

func (d *Deps) platformKeyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if d.RequirePlatformAPIKey && strings.TrimSpace(d.PlatformAPIKey) == "" {
			WriteProblem(w, r, ProblemInput{
				Status: http.StatusServiceUnavailable,
				Title:  "Service Unavailable",
				Detail: "platform API authentication is not configured",
			})
			return
		}
		if strings.TrimSpace(d.PlatformAPIKey) == "" {
			next.ServeHTTP(w, r)
			return
		}
		if r.Header.Get("X-Api-Key") != d.PlatformAPIKey {
			WriteProblem(w, r, ProblemInput{
				Status: http.StatusUnauthorized,
				Title:  "Unauthorized",
				Detail: "invalid or missing platform credentials",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (d *Deps) tenantMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if d.SkipTenantCheck {
			next.ServeHTTP(w, r)
			return
		}
		bid := chi.URLParam(r, "businessId")
		if got := r.Header.Get("X-Tenant-Business-Id"); got != bid {
			WriteProblem(w, r, ProblemInput{
				Status: http.StatusForbidden,
				Title:  "Forbidden",
				Detail: "X-Tenant-Business-Id header must match businessId in the path",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func registerPhase3Routes(r chi.Router, d *Deps) {
	r.With(d.platformKeyMiddleware).Post("/platform/businesses", d.postPlatformBusinesses)
	if d.Notifications != nil {
		r.With(d.platformKeyMiddleware).Post("/platform/notifications/dispatch-due", d.postDispatchDueNotifications)
	}

	r.Route("/businesses/{businessId}", func(r chi.Router) {
		r.Use(d.tenantMiddleware)
		r.Get("/", d.getBusiness)
		r.Patch("/", d.patchBusiness)

		r.Route("/services", func(r chi.Router) {
			r.Get("/", d.listServices)
			r.Post("/", d.postService)
			r.Route("/{serviceId}", func(r chi.Router) {
				r.Get("/", d.getService)
				r.Patch("/", d.patchService)
				r.Delete("/", d.deleteService)
			})
		})

		r.Route("/staff", func(r chi.Router) {
			r.Get("/", d.listStaff)
			r.Post("/", d.postStaff)
			r.Route("/{staffId}", func(r chi.Router) {
				r.Get("/", d.getStaff)
				r.Patch("/", d.patchStaff)
				r.Delete("/", d.deleteStaff)
			})
		})

		registerBusinessScopedStubs(r, d)
	})

	r.Route("/webhooks", func(r chi.Router) {
		if d.Conversations != nil {
			r.Post("/whatsapp", d.postWhatsAppWebhook)
		} else {
			r.Post("/whatsapp", stub501("whatsapp webhook"))
		}
		if d.Twilio != nil {
			r.Post("/twilio", d.postTwilioWebhook)
		} else {
			r.Post("/twilio", stub501("twilio webhook"))
		}
		if d.Payments != nil {
			r.Post("/payments/{provider}", d.postPaymentProviderWebhook)
		} else {
			r.Post("/payments/{provider}", stub501("payment provider webhook"))
		}
	})
}

// registerBusinessScopedStubs are routes under /v1/businesses/{businessId} beyond catalog.
func registerBusinessScopedStubs(r chi.Router, d *Deps) {
	if d != nil && d.Customers != nil && d.Scheduling != nil {
		r.Route("/customers", func(r chi.Router) {
			r.Get("/by-phone", d.getCustomerByPhone)
			r.Get("/", d.listCustomers)
			r.Post("/", d.postCustomer)
			r.Route("/{customerId}", func(r chi.Router) {
				r.Get("/", d.getCustomer)
				r.Patch("/", d.patchCustomer)
			})
		})
		r.Route("/availability", func(r chi.Router) {
			r.Put("/rules", d.putAvailabilityRules)
			r.Get("/slots", d.getAvailabilitySlots)
		})
	} else {
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
	}

	if d != nil && d.Bookings != nil {
		r.Route("/bookings", func(r chi.Router) {
			r.Get("/", d.listBookings)
			r.Post("/", d.postBooking)
			r.Route("/{bookingId}", func(r chi.Router) {
				r.Get("/", d.getBooking)
				r.Post("/confirm", d.postConfirmBooking)
				r.Post("/cancel", d.postCancelBooking)
				r.Post("/complete", d.postCompleteBooking)
				r.Post("/no-show", d.postNoShowBooking)
				r.Route("/payments", func(r chi.Router) {
					if d != nil && d.Payments != nil {
						r.Get("/", d.listPaymentsForBooking)
					} else {
						r.Get("/", stub501("list payments for booking"))
					}
				})
			})
		})
	} else {
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
	}

	r.Route("/payments", func(r chi.Router) {
		if d != nil && d.Payments != nil {
			r.Post("/", d.postPayment)
			r.Route("/{paymentId}", func(r chi.Router) {
				r.Get("/", d.getPayment)
			})
		} else {
			r.Post("/", stub501("create payment"))
			r.Route("/{paymentId}", func(r chi.Router) {
				r.Get("/", stub501("get payment"))
			})
		}
	})

	if d != nil && d.Conversations != nil {
		r.Route("/conversations", func(r chi.Router) {
			r.Post("/", d.postEnsureConversation)
			r.Route("/{conversationId}", func(r chi.Router) {
				r.Get("/", d.getConversationDoc)
				r.Route("/messages", func(r chi.Router) {
					r.Get("/", d.listConversationMessages)
					r.Post("/", d.postConversationMessage)
				})
			})
		})
	} else {
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
	}

	if d != nil && d.Notifications != nil {
		r.Route("/notifications", func(r chi.Router) {
			r.Get("/", d.listNotifications)
			r.Post("/", d.postNotification)
		})
	} else {
		r.Route("/notifications", func(r chi.Router) {
			r.Get("/", stub501("list notifications"))
			r.Post("/", stub501("schedule notification"))
		})
	}
}

func registerStubRoutes(r chi.Router, d *Deps) {
	if d == nil {
		d = &Deps{}
	}
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

		registerBusinessScopedStubs(r, &Deps{})
	})

	r.Route("/webhooks", func(r chi.Router) {
		r.Post("/whatsapp", stub501("whatsapp webhook"))
		if d != nil && d.Twilio != nil {
			r.Post("/twilio", d.postTwilioWebhook)
		} else {
			r.Post("/twilio", stub501("twilio webhook"))
		}
		r.Post("/payments/{provider}", stub501("payment provider webhook"))
	})
}

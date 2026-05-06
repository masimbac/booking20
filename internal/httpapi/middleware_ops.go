package httpapi

import (
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// Router hardening options (Phase 9). Zero values disable optional behavior.
type HardeningConfig struct {
	// CORSAllowedOrigins is an exact-match allowlist for the Origin header (browser clients). Empty disables CORS headers.
	CORSAllowedOrigins []string
	// RateLimitMax is max requests per RateLimitWindow per client IP. 0 disables rate limiting.
	RateLimitMax int
	// RateLimitWindow is the counting window when RateLimitMax > 0.
	RateLimitWindow time.Duration
}

type responseCapture struct {
	http.ResponseWriter
	status int
	n      int
}

func (w *responseCapture) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseCapture) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.n += n
	return n, err
}

// structuredAccessLog emits one JSON log line per request (CloudWatch-friendly in Lambda).
func structuredAccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rc := &responseCapture{ResponseWriter: w, status: 0}
		next.ServeHTTP(rc, r)
		status := rc.status
		if status == 0 {
			status = http.StatusOK
		}
		reqID := middleware.GetReqID(r.Context())
		slog.Info("http_request",
			"request_id", reqID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"bytes", rc.n,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote_ip", r.RemoteAddr,
		)
	})
}

func corsMiddleware(allowed []string) func(http.Handler) http.Handler {
	if len(allowed) == 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	set := make(map[string]struct{}, len(allowed))
	for _, o := range allowed {
		o = strings.TrimSpace(o)
		if o != "" {
			set[o] = struct{}{}
		}
	}
	if len(set) == 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	allowHeaders := "Authorization, Content-Type, X-Api-Key, X-Tenant-Business-Id, Idempotency-Key, X-Hub-Signature-256, X-Payment-Signature"
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if r.Method == http.MethodOptions {
				if _, ok := set[origin]; ok {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Vary", "Origin")
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", allowHeaders)
					w.Header().Set("Access-Control-Max-Age", "86400")
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}
			if _, ok := set[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", allowHeaders)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

type windowVisitor struct {
	mu      sync.Mutex
	count   int
	window0 time.Time // start of current window
}

type slidingLimiter struct {
	mu       sync.Mutex
	visitors map[string]*windowVisitor
	max      int
	window   time.Duration
}

func newSlidingLimiter(max int, window time.Duration) *slidingLimiter {
	if max <= 0 || window <= 0 {
		return nil
	}
	return &slidingLimiter{
		visitors: make(map[string]*windowVisitor),
		max:      max,
		window:   window,
	}
}

func (l *slidingLimiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	v, ok := l.visitors[key]
	if !ok {
		v = &windowVisitor{}
		l.visitors[key] = v
	}
	l.mu.Unlock()

	v.mu.Lock()
	defer v.mu.Unlock()
	if now.Sub(v.window0) >= l.window {
		v.count = 0
		v.window0 = now
	}
	if v.count >= l.max {
		return false
	}
	v.count++
	return true
}

func rateLimitMiddleware(lim *slidingLimiter) func(http.Handler) http.Handler {
	if lim == nil {
		return func(next http.Handler) http.Handler { return next }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions || r.URL.Path == "/v1/health" || strings.HasSuffix(r.URL.Path, "/v1/health") {
				next.ServeHTTP(w, r)
				return
			}
			ip := r.RemoteAddr
			if host, _, err := net.SplitHostPort(ip); err == nil {
				ip = host
			}
			if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
				ip = strings.TrimSpace(strings.Split(xff, ",")[0])
			}
			if !lim.allow(ip, time.Now()) {
				WriteProblem(w, r, ProblemInput{
					Status: http.StatusTooManyRequests,
					Title:  "Too Many Requests",
					Detail: "rate limit exceeded; retry later",
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ParseCORSOrigins splits a comma-separated env-style list into origins (trimmed, empty dropped).
func ParseCORSOrigins(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// ParsePositiveInt returns n > 0 or def if raw is empty, else parses strconv.Atoi.
func ParsePositiveInt(raw string, def int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

// ParseDurationSeconds parses seconds as a duration or returns def.
func ParseDurationSeconds(raw string, def time.Duration) time.Duration {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return def
	}
	sec, err := strconv.Atoi(raw)
	if err != nil || sec <= 0 {
		return def
	}
	return time.Duration(sec) * time.Second
}

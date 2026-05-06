package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func TestWriteConflict_IncludesBusinessID(t *testing.T) {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		WriteConflict(w, req, "slot taken", "biz_1")
	})

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	res, err := http.DefaultClient.Do(mustReq(t, http.MethodGet, srv.URL+"/"))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("status %d", res.StatusCode)
	}
	var p Problem
	if err := json.NewDecoder(res.Body).Decode(&p); err != nil {
		t.Fatal(err)
	}
	if p.BusinessID != "biz_1" || p.Title != "Conflict" {
		t.Fatalf("%+v", p)
	}
	if !strings.Contains(res.Header.Get("Content-Type"), "problem+json") {
		t.Fatal("wrong content type")
	}
}

func TestWriteUnprocessable(t *testing.T) {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		WriteUnprocessable(w, req, "invalid window")
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	res, err := http.DefaultClient.Do(mustReq(t, http.MethodGet, srv.URL+"/"))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 422 {
		t.Fatalf("status %d", res.StatusCode)
	}
}

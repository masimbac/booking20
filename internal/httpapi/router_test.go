package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealth_OK(t *testing.T) {
	r := NewRouter(RouterConfig{Phase: "6"}, nil)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	res, err := http.DefaultClient.Do(mustReq(t, http.MethodGet, srv.URL+"/v1/health"))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type: %q", ct)
	}
	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" || body["phase"] != "6" {
		t.Fatalf("body: %+v", body)
	}
}

func TestHealth_StrippedStagePrefix(t *testing.T) {
	r := NewRouter(RouterConfig{Phase: "6", Stage: "dev"}, nil)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	res, err := http.DefaultClient.Do(mustReq(t, http.MethodGet, srv.URL+"/dev/v1/health"))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", res.StatusCode)
	}
}

func TestNotFound_ProblemJSON(t *testing.T) {
	r := NewRouter(RouterConfig{Phase: "6"}, nil)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	res, err := http.DefaultClient.Do(mustReq(t, http.MethodGet, srv.URL+"/v1/does-not-exist"))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status: %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != contentTypeProblem {
		t.Fatalf("Content-Type: %q want %q", ct, contentTypeProblem)
	}
	raw, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(raw), `"title":"Not Found"`) {
		t.Fatalf("body: %s", raw)
	}
}

func TestStub501_JSON(t *testing.T) {
	r := NewRouter(RouterConfig{Phase: "6"}, nil)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	req := mustReq(t, http.MethodPost, srv.URL+"/v1/platform/businesses", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotImplemented {
		t.Fatalf("status: %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != contentTypeProblem {
		t.Fatalf("Content-Type: %q", ct)
	}
}

func TestProblemHelpers_StatusConstants(t *testing.T) {
	// Sanity: RFC 422 and 409 map to standard helpers when used later.
	if http.StatusUnprocessableEntity != 422 {
		t.Fatal("422 mismatch")
	}
	if http.StatusConflict != 409 {
		t.Fatal("409 mismatch")
	}
}

func mustReq(t *testing.T, method, url string, body ...io.Reader) *http.Request {
	t.Helper()
	var r io.Reader
	if len(body) > 0 {
		r = body[0]
	}
	req, err := http.NewRequestWithContext(context.Background(), method, url, r)
	if err != nil {
		t.Fatal(err)
	}
	return req
}

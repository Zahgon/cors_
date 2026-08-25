package test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cors "github.com/expressjs/cors/lib"
)

func TestHTTPAdapterRequestExposesTheMethodAndHeaders(t *testing.T) {
	raw := httptest.NewRequest(http.MethodOptions, "/", nil)
	raw.Header.Set("Origin", "http://example.com")

	req := cors.NewRequest(raw)

	if got := req.Method(); got != http.MethodOptions {
		t.Errorf("Method = %q, want %q", got, http.MethodOptions)
	}
	if got := req.Header("origin"); got != "http://example.com" {
		t.Errorf("Header(origin) = %q, want %q", got, "http://example.com")
	}
	if got := req.Header("x-absent"); got != "" {
		t.Errorf("Header(x-absent) = %q, want it empty", got)
	}
}

func TestHTTPAdapterResponseDefersTheStatusCodeUntilEnd(t *testing.T) {
	recorder := httptest.NewRecorder()
	res := cors.NewResponse(recorder)

	if got := res.GetHeader("Vary"); got != "" {
		t.Errorf("GetHeader(Vary) = %q, want it empty", got)
	}

	res.SetHeader("Vary", "Origin")
	res.SetStatusCode(http.StatusNoContent)

	if got := res.GetHeader("Vary"); got != "Origin" {
		t.Errorf("GetHeader(Vary) = %q, want %q", got, "Origin")
	}

	res.End()
	res.End()

	if recorder.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if got := recorder.Header().Get("Vary"); got != "Origin" {
		t.Errorf("Vary = %q, want %q", got, "Origin")
	}
}

func TestHTTPAdapterResponseDefaultsToOK(t *testing.T) {
	recorder := httptest.NewRecorder()
	res := cors.NewResponse(recorder)

	res.SetHeader("Access-Control-Allow-Origin", "*")
	res.End()

	if recorder.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "*")
	}
}

func TestHTTPAdapterHandlerAnswersPreflightWithoutCallingNext(t *testing.T) {
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("next ran for a preflight request, want it shortcircuited")
	})
	recorder := httptest.NewRecorder()

	cors.New().Handler(next).ServeHTTP(recorder, httptest.NewRequest(http.MethodOptions, "/", nil))

	if recorder.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "*")
	}
}

func TestHTTPAdapterHandlerPassesASimpleRequestToNext(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	recorder := httptest.NewRecorder()

	cors.New().Handler(next).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if !called {
		t.Fatal("next did not run for a simple request")
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "*")
	}
}

func TestHTTPAdapterHandlerReportsAnOptionsDelegateError(t *testing.T) {
	middleware := cors.NewWithDelegate(func(_ cors.Request, cb func(err error, options *cors.Options)) {
		cb(errors.New("boom"), nil)
	})
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("next ran after a delegate error, want it skipped")
	})
	recorder := httptest.NewRecorder()

	middleware.Handler(next).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(recorder.Body.String(), "boom") {
		t.Errorf("body = %q, want it to contain %q", recorder.Body.String(), "boom")
	}
}

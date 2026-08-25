package support

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Agent issues requests against an App over a real HTTP connection, mirroring
// `supertest(app)`.
type Agent struct {
	app *App
}

// Supertest returns an Agent bound to app.
func Supertest(app *App) *Agent { return &Agent{app: app} }

// Get starts a GET request.
func (a *Agent) Get(path string) *Test { return a.request(http.MethodGet, path) }

// Head starts a HEAD request.
func (a *Agent) Head(path string) *Test { return a.request(http.MethodHead, path) }

// Post starts a POST request.
func (a *Agent) Post(path string) *Test { return a.request(http.MethodPost, path) }

// Options starts an OPTIONS request.
func (a *Agent) Options(path string) *Test { return a.request(http.MethodOptions, path) }

// Delete starts a DELETE request.
func (a *Agent) Delete(path string) *Test { return a.request(http.MethodDelete, path) }

func (a *Agent) request(method, path string) *Test {
	return &Test{app: a.app, method: method, path: path, headers: http.Header{}}
}

// Test is a request under construction.
type Test struct {
	app     *App
	method  string
	path    string
	headers http.Header
}

// Set adds a request header, mirroring `.set(name, value)`.
func (t *Test) Set(name, value string) *Test {
	t.headers.Set(name, value)
	return t
}

// Result is the response to a request issued through an Agent. Expectations are
// deliberately not part of this type: the JavaScript suite chained them onto
// supertest, but a Go test reads better - and reports failures against the
// right line - when it asserts on the result itself.
type Result struct {
	StatusCode int
	Body       string
	headers    http.Header
}

// Header returns the named response header, or an empty string when absent.
func (r *Result) Header(name string) string { return r.headers.Get(name) }

// Do sends the request and returns the response, mirroring `.end(done)`.
func (t *Test) Do(tb testing.TB) *Result {
	tb.Helper()

	server := httptest.NewServer(t.app)
	defer server.Close()

	req, err := http.NewRequest(t.method, server.URL+t.path, nil)
	if err != nil {
		tb.Fatalf("could not build request: %v", err)
	}
	for name, values := range t.headers {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}

	res, err := server.Client().Do(req)
	if err != nil {
		tb.Fatalf("request failed: %v", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		tb.Fatalf("could not read response body: %v", err)
	}

	return &Result{StatusCode: res.StatusCode, Body: string(body), headers: res.Header}
}

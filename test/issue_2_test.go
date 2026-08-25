package test

import (
	"net/http"
	"testing"

	cors "github.com/expressjs/cors/lib"
	"github.com/expressjs/cors/test/support"
)

func newIssue2App() *support.App {
	app := support.NewApp()

	corsOptions := &cors.Options{
		Origin:      true,
		Methods:     []string{"POST"},
		Credentials: true,
		MaxAge:      3600,
	}

	app.Options("/api/login", support.CORS(cors.NewWithOptions(corsOptions)))

	app.Post("/api/login", support.CORS(cors.NewWithOptions(corsOptions)), func(req *support.Request, res *support.Response, next cors.NextFunc) {
		res.Send("LOGIN")
	})

	return app
}

func TestIssue2OptionsWorks(t *testing.T) {
	res := support.Supertest(newIssue2App()).
		Options("/api/login").
		Set("Origin", "http://example.com").
		Do(t)

	if res.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, want %d", res.StatusCode, http.StatusNoContent)
	}
	if got := res.Header("Access-Control-Allow-Origin"); got != "http://example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "http://example.com")
	}
}

func TestIssue2PostWorks(t *testing.T) {
	res := support.Supertest(newIssue2App()).
		Post("/api/login").
		Set("Origin", "http://example.com").
		Do(t)

	if res.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}
	if got := res.Header("Access-Control-Allow-Origin"); got != "http://example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "http://example.com")
	}
	if res.Body != "LOGIN" {
		t.Errorf("body = %q, want %q", res.Body, "LOGIN")
	}
}

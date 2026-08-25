package test

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	cors "github.com/expressjs/cors/lib"
	"github.com/expressjs/cors/test/support"
)

func newErrorResponseApp() *support.App {
	app := support.NewApp()

	app.Use(support.CORS(cors.New()))

	app.Post("/five-hundred", func(req *support.Request, res *support.Response, next cors.NextFunc) {
		next(errors.New("nope"))
	})

	app.Post("/four-oh-one", func(req *support.Request, res *support.Response, next cors.NextFunc) {
		next(errors.New("401"))
	})

	app.Post("/four-oh-four", func(req *support.Request, res *support.Response, next cors.NextFunc) {
		next(nil)
	})

	app.UseError(func(err error, req *support.Request, res *support.Response, next cors.NextFunc) {
		if err.Error() == "401" {
			res.Status(http.StatusUnauthorized).Send("unauthorized")
			return
		}
		next(err)
	})

	return app
}

func TestErrorResponse500(t *testing.T) {
	res := support.Supertest(newErrorResponseApp()).Post("/five-hundred").Do(t)

	if res.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", res.StatusCode, http.StatusInternalServerError)
	}
	if got := res.Header("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "*")
	}
	if !strings.Contains(res.Body, "Error: nope") {
		t.Errorf("body = %q, want it to contain %q", res.Body, "Error: nope")
	}
}

func TestErrorResponse401(t *testing.T) {
	res := support.Supertest(newErrorResponseApp()).Post("/four-oh-one").Do(t)

	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", res.StatusCode, http.StatusUnauthorized)
	}
	if got := res.Header("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "*")
	}
	if res.Body != "unauthorized" {
		t.Errorf("body = %q, want %q", res.Body, "unauthorized")
	}
}

func TestErrorResponse404(t *testing.T) {
	res := support.Supertest(newErrorResponseApp()).Post("/four-oh-four").Do(t)

	if res.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", res.StatusCode, http.StatusNotFound)
	}
	if got := res.Header("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "*")
	}
}

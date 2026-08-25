package test

import (
	"net/http"
	"testing"

	cors "github.com/expressjs/cors/lib"
	"github.com/expressjs/cors/test/support"
)

func newSimpleApp() *support.App {
	app := support.NewApp()

	app.Head("/", support.CORS(cors.New()), func(req *support.Request, res *support.Response, next cors.NextFunc) {
		res.Status(http.StatusNoContent).Send()
	})

	app.Get("/", support.CORS(cors.New()), func(req *support.Request, res *support.Response, next cors.NextFunc) {
		res.Send("Hello World (Get)")
	})

	app.Post("/", support.CORS(cors.New()), func(req *support.Request, res *support.Response, next cors.NextFunc) {
		res.Send("Hello World (Post)")
	})

	return app
}

func newComplexApp() *support.App {
	app := support.NewApp()

	app.Options("/", support.CORS(cors.New()))

	app.Delete("/", support.CORS(cors.New()), func(req *support.Request, res *support.Response, next cors.NextFunc) {
		res.Send("Hello World (Delete)")
	})

	return app
}

func TestExampleAppSimpleMethodsGetWorks(t *testing.T) {
	res := support.Supertest(newSimpleApp()).Get("/").Do(t)

	if res.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}
	if got := res.Header("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "*")
	}
	if res.Body != "Hello World (Get)" {
		t.Errorf("body = %q, want %q", res.Body, "Hello World (Get)")
	}
}

func TestExampleAppSimpleMethodsHeadWorks(t *testing.T) {
	res := support.Supertest(newSimpleApp()).Head("/").Do(t)

	if res.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, want %d", res.StatusCode, http.StatusNoContent)
	}
	if got := res.Header("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "*")
	}
}

func TestExampleAppSimpleMethodsPostWorks(t *testing.T) {
	res := support.Supertest(newSimpleApp()).Post("/").Do(t)

	if res.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}
	if got := res.Header("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "*")
	}
	if res.Body != "Hello World (Post)" {
		t.Errorf("body = %q, want %q", res.Body, "Hello World (Post)")
	}
}

func TestExampleAppComplexMethodsOptionsWorks(t *testing.T) {
	res := support.Supertest(newComplexApp()).Options("/").Do(t)

	if res.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, want %d", res.StatusCode, http.StatusNoContent)
	}
	if got := res.Header("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "*")
	}
}

func TestExampleAppComplexMethodsDeleteWorks(t *testing.T) {
	res := support.Supertest(newComplexApp()).Delete("/").Do(t)

	if res.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}
	if got := res.Header("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "*")
	}
	if res.Body != "Hello World (Delete)" {
		t.Errorf("body = %q, want %q", res.Body, "Hello World (Delete)")
	}
}

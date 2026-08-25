// Package support contains the test scaffolding for the cors middleware. It
// provides the small slice of Express and supertest behaviour that the original
// JavaScript test suite depended on, so the ported tests can express the same
// scenarios with the same assertions.
package support

import (
	"io"
	"net/http"
	"os"
	"strconv"

	cors "github.com/expressjs/cors/lib"
)

// Handler mirrors an Express request handler, `function (req, res, next)`.
type Handler func(req *Request, res *Response, next cors.NextFunc)

// ErrorHandler mirrors Express's four argument error middleware,
// `function (err, req, res, next)`.
type ErrorHandler func(err error, req *Request, res *Response, next cors.NextFunc)

// CORS adapts a cors.Middleware to the Handler signature so it can be mounted
// on an App, mirroring `app.use(cors())`.
func CORS(middleware cors.Middleware) Handler {
	return func(req *Request, res *Response, next cors.NextFunc) {
		middleware(req, res, next)
	}
}

// Request is the request object handed to a Handler. It satisfies cors.Request.
type Request struct {
	raw *http.Request
}

// Method returns the request method.
func (r *Request) Method() string { return r.raw.Method }

// Header returns the named request header, or an empty string when absent.
func (r *Request) Header(name string) string { return r.raw.Header.Get(name) }

// Path returns the request path.
func (r *Request) Path() string { return r.raw.URL.Path }

// Raw returns the underlying request.
func (r *Request) Raw() *http.Request { return r.raw }

// Response is the response object handed to a Handler. It satisfies
// cors.Response and adds the `status`/`send` pair used by the test apps.
type Response struct {
	w          http.ResponseWriter
	statusCode int
	sent       bool
}

// GetHeader returns the named response header, or an empty string when unset.
func (r *Response) GetHeader(name string) string { return r.w.Header().Get(name) }

// SetHeader sets a response header.
func (r *Response) SetHeader(name, value string) { r.w.Header().Set(name, value) }

// SetStatusCode records the status code to send.
func (r *Response) SetStatusCode(code int) { r.statusCode = code }

// StatusCode returns the status code that will be, or has been, sent.
func (r *Response) StatusCode() int { return r.statusCode }

// Status records the status code and returns the response for chaining,
// mirroring `res.status(code)`.
func (r *Response) Status(code int) *Response {
	r.statusCode = code
	return r
}

// Send writes an optional body and completes the response, mirroring
// `res.send([body])`.
func (r *Response) Send(body ...string) {
	payload := ""
	if len(body) > 0 {
		payload = body[0]
	}

	if r.statusCode == http.StatusNoContent || r.statusCode == http.StatusNotModified {
		payload = ""
		r.w.Header().Del("Content-Type")
		r.w.Header().Del("Content-Length")
	} else {
		if r.GetHeader("Content-Type") == "" {
			r.SetHeader("Content-Type", "text/html; charset=utf-8")
		}
		r.SetHeader("Content-Length", strconv.Itoa(len(payload)))
	}

	r.End()
	if payload != "" {
		io.WriteString(r.w, payload)
	}
}

// End completes the response without writing a body, mirroring `res.end()`.
func (r *Response) End() {
	if r.sent {
		return
	}
	r.sent = true
	r.w.WriteHeader(r.statusCode)
}

// App is a minimal Express style application: an ordered stack of layers that
// each either handle a request or handle an error.
type App struct {
	stack []layer
}

type layer struct {
	method   string
	path     string
	handlers []Handler
	onError  ErrorHandler
}

// NewApp returns an empty application, mirroring `express()`.
func NewApp() *App { return &App{} }

// Use mounts handlers that run for every request, mirroring `app.use(...)`.
func (a *App) Use(handlers ...Handler) {
	a.stack = append(a.stack, layer{handlers: handlers})
}

// UseError mounts an error handler, mirroring `app.use(function (err, ...) {})`.
func (a *App) UseError(handler ErrorHandler) {
	a.stack = append(a.stack, layer{onError: handler})
}

// Get mounts handlers for GET requests to path.
func (a *App) Get(path string, handlers ...Handler) { a.route(http.MethodGet, path, handlers) }

// Head mounts handlers for HEAD requests to path.
func (a *App) Head(path string, handlers ...Handler) { a.route(http.MethodHead, path, handlers) }

// Post mounts handlers for POST requests to path.
func (a *App) Post(path string, handlers ...Handler) { a.route(http.MethodPost, path, handlers) }

// Options mounts handlers for OPTIONS requests to path.
func (a *App) Options(path string, handlers ...Handler) { a.route(http.MethodOptions, path, handlers) }

// Delete mounts handlers for DELETE requests to path.
func (a *App) Delete(path string, handlers ...Handler) { a.route(http.MethodDelete, path, handlers) }

func (a *App) route(method, path string, handlers []Handler) {
	a.stack = append(a.stack, layer{method: method, path: path, handlers: handlers})
}

// ServeHTTP runs the request through the layer stack.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.dispatch(&Request{raw: r}, &Response{w: w, statusCode: http.StatusOK}, 0, nil)
}

func (a *App) dispatch(req *Request, res *Response, index int, pending error) {
	for i := index; i < len(a.stack); i++ {
		current := a.stack[i]
		resume := i + 1

		if current.onError != nil {
			if pending == nil {
				continue
			}
			current.onError(pending, req, res, func(err error) {
				a.dispatch(req, res, resume, err)
			})
			return
		}

		if pending != nil || !current.matches(req) {
			continue
		}
		a.runHandlers(current.handlers, req, res, resume)
		return
	}

	a.finalize(req, res, pending)
}

func (a *App) runHandlers(handlers []Handler, req *Request, res *Response, resume int) {
	var run func(index int)
	run = func(index int) {
		if index >= len(handlers) {
			a.dispatch(req, res, resume, nil)
			return
		}
		handlers[index](req, res, func(err error) {
			if err != nil {
				a.dispatch(req, res, resume, err)
				return
			}
			run(index + 1)
		})
	}
	run(0)
}

func (a *App) finalize(req *Request, res *Response, pending error) {
	if pending != nil {
		res.Status(http.StatusInternalServerError).Send(errorBody(pending))
		return
	}
	res.Status(http.StatusNotFound).Send("Cannot " + req.Method() + " " + req.Path() + "\n")
}

func (l layer) matches(req *Request) bool {
	if l.path != "" && l.path != req.Path() {
		return false
	}
	if l.method == "" || l.method == req.Method() {
		return true
	}
	// Express answers HEAD requests with the routes registered for GET.
	return l.method == http.MethodGet && req.Method() == http.MethodHead
}

// errorBody mirrors Express's default error handler, which exposes the error
// outside of production and hides it otherwise.
func errorBody(err error) string {
	if os.Getenv("GO_ENV") == "production" {
		return http.StatusText(http.StatusInternalServerError)
	}
	return "Error: " + err.Error()
}

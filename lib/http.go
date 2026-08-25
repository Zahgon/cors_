package cors

import "net/http"

// NewRequest adapts a *http.Request so it can be handed to a Middleware.
func NewRequest(r *http.Request) Request {
	return httpRequest{r: r}
}

type httpRequest struct {
	r *http.Request
}

func (h httpRequest) Method() string {
	return h.r.Method
}

func (h httpRequest) Header(name string) string {
	return h.r.Header.Get(name)
}

// NewResponse adapts an http.ResponseWriter so it can be handed to a
// Middleware. Nothing is written until End is called.
func NewResponse(w http.ResponseWriter) Response {
	return &httpResponse{w: w, statusCode: http.StatusOK}
}

type httpResponse struct {
	w          http.ResponseWriter
	statusCode int
	written    bool
}

func (h *httpResponse) GetHeader(name string) string {
	return h.w.Header().Get(name)
}

func (h *httpResponse) SetHeader(name, value string) {
	h.w.Header().Set(name, value)
}

func (h *httpResponse) SetStatusCode(code int) {
	h.statusCode = code
}

func (h *httpResponse) End() {
	if h.written {
		return
	}
	h.written = true
	h.w.WriteHeader(h.statusCode)
}

// Handler wraps next with the middleware, which is the net/http equivalent of
// `app.use(cors())`. A preflight request is answered by the middleware itself
// and never reaches next.
func (m Middleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m(NewRequest(r), NewResponse(w), func(err error) {
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			next.ServeHTTP(w, r)
		})
	})
}

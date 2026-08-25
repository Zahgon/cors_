package test

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"testing"

	cors "github.com/expressjs/cors/lib"
)

// corsResponseHeaders are the headers the middleware must leave untouched when
// CORS is disabled for a request.
var corsResponseHeaders = []string{
	"Access-Control-Allow-Origin",
	"Access-Control-Allow-Methods",
	"Access-Control-Allow-Headers",
	"Access-Control-Allow-Credentials",
	"Access-Control-Max-Age",
}

// enabled is how a true boolean option renders in a response header.
var enabled = strconv.FormatBool(true)

// defaultMethods is the value of the `methods` option when it is not set.
const defaultMethods = "GET,HEAD,PUT,PATCH,POST,DELETE"

func TestCorsDoesNotAlterOptionsConfigurationObject(t *testing.T) {
	options := &cors.Options{Origin: "custom-origin"}
	snapshot := *options

	res := runRequest(t, cors.NewWithOptions(options), newFakeRequest("GET", nil), nil)

	if *options != snapshot {
		t.Errorf("options = %+v, want them unchanged at %+v", *options, snapshot)
	}
	if got := res.GetHeader("Access-Control-Allow-Origin"); got != "custom-origin" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "custom-origin")
	}
}

func TestCorsPassesControlToNextMiddleware(t *testing.T) {
	called := false
	var nextErr error

	cors.New()(newFakeRequest("GET", nil), newFakeResponse(), func(err error) {
		called = true
		nextErr = err
	})

	if !called {
		t.Fatal("next was not called")
	}
	if nextErr != nil {
		t.Errorf("next err = %v, want nil", nextErr)
	}
}

func TestCorsShortcircuitsPreflightRequests(t *testing.T) {
	res := runPreflight(t, cors.New(), newFakeRequest("OPTIONS", nil))

	if got := res.StatusCode(); got != 204 {
		t.Errorf("status code = %d, want %d", got, 204)
	}
}

func TestCorsCanConfigurePreflightSuccessResponseStatusCode(t *testing.T) {
	middleware := cors.NewWithOptions(&cors.Options{OptionsSuccessStatus: 200})

	res := runPreflight(t, middleware, newFakeRequest("OPTIONS", nil))

	if got := res.StatusCode(); got != 200 {
		t.Errorf("status code = %d, want %d", got, 200)
	}
}

func TestCorsDoesNotShortcircuitPreflightRequestsWithPreflightContinueOption(t *testing.T) {
	res := newFakeResponse()
	res.On("finish", func() {
		t.Error("the response should not be finished")
	})

	middleware := cors.NewWithOptions(&cors.Options{PreflightContinue: true})
	runRequest(t, middleware, newFakeRequest("OPTIONS", nil), res)

	if got := res.StatusCode(); got != 200 {
		t.Errorf("status code = %d, want it left at %d", got, 200)
	}
}

func TestCorsNormalizesMethodNames(t *testing.T) {
	res := runPreflight(t, cors.New(), newFakeRequest("options", nil))

	if got := res.StatusCode(); got != 204 {
		t.Errorf("status code = %d, want %d", got, 204)
	}
}

func TestCorsIncludesContentLengthResponseHeader(t *testing.T) {
	res := runPreflight(t, cors.New(), newFakeRequest("OPTIONS", nil))

	if got := res.GetHeader("Content-Length"); got != "0" {
		t.Errorf("Content-Length = %q, want %q", got, "0")
	}
}

func TestCorsNoOptionsEnablesDefaultCorsToAllOrigins(t *testing.T) {
	res := runRequest(t, cors.New(), newFakeRequest("GET", nil), nil)

	if got := res.GetHeader("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "*")
	}
	if got, ok := res.LookupHeader("Access-Control-Allow-Methods"); ok {
		t.Errorf("Access-Control-Allow-Methods = %q, want it unset", got)
	}
}

func TestCorsOptionCallWithNoOptionsEnablesDefaultCorsToAllOriginsAndMethods(t *testing.T) {
	res := runPreflight(t, cors.New(), newFakeRequest("OPTIONS", nil))

	if got := res.StatusCode(); got != 204 {
		t.Errorf("status code = %d, want %d", got, 204)
	}
	if got := res.GetHeader("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "*")
	}
	if got := res.GetHeader("Access-Control-Allow-Methods"); got != defaultMethods {
		t.Errorf("Access-Control-Allow-Methods = %q, want %q", got, defaultMethods)
	}
}

func TestPassingStaticOptionsOverridesDefaults(t *testing.T) {
	options := &cors.Options{
		Origin:      "http://example.com",
		Methods:     []string{"FOO", "bar"},
		Headers:     []string{"FIZZ", "buzz"},
		Credentials: true,
		MaxAge:      123,
	}

	res := runPreflight(t, cors.NewWithOptions(options), newFakeRequest("OPTIONS", nil))

	if got := res.StatusCode(); got != 204 {
		t.Errorf("status code = %d, want %d", got, 204)
	}
	for _, want := range []struct{ name, value string }{
		{"Access-Control-Allow-Origin", "http://example.com"},
		{"Access-Control-Allow-Methods", "FOO,bar"},
		{"Access-Control-Allow-Headers", "FIZZ,buzz"},
		{"Access-Control-Allow-Credentials", enabled},
		{"Access-Control-Max-Age", "123"},
	} {
		if got := res.GetHeader(want.name); got != want.value {
			t.Errorf("%s = %q, want %q", want.name, got, want.value)
		}
	}
}

func TestPassingStaticOptionsMatchesRequestOriginAgainstRegexp(t *testing.T) {
	req := newFakeRequest("GET", nil)
	options := &cors.Options{Origin: regexp.MustCompile(`://(.+\.)?example.com$`)}

	res := runRequest(t, cors.NewWithOptions(options), req, nil)

	if got, want := res.GetHeader("Access-Control-Allow-Origin"), req.Header("origin"); got != want {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, want)
	}
	if got := res.GetHeader("Vary"); got != "Origin" {
		t.Errorf("Vary = %q, want %q", got, "Origin")
	}
}

func TestPassingStaticOptionsMatchesRequestOriginAgainstArrayOfOriginChecks(t *testing.T) {
	req := newFakeRequest("GET", nil)
	options := &cors.Options{Origin: []any{regexp.MustCompile(`foo\.com$`), "http://example.com"}}

	res := runRequest(t, cors.NewWithOptions(options), req, nil)

	if got, want := res.GetHeader("Access-Control-Allow-Origin"), req.Header("origin"); got != want {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, want)
	}
	if got := res.GetHeader("Vary"); got != "Origin" {
		t.Errorf("Vary = %q, want %q", got, "Origin")
	}
}

func TestPassingStaticOptionsDoesNotMatchRequestOriginAgainstArrayOfInvalidOriginChecks(t *testing.T) {
	req := newFakeRequest("GET", nil)
	options := &cors.Options{Origin: []any{regexp.MustCompile(`foo\.com$`), "bar.com"}}

	res := runRequest(t, cors.NewWithOptions(options), req, nil)

	if got, ok := res.LookupHeader("Access-Control-Allow-Origin"); ok {
		t.Errorf("Access-Control-Allow-Origin = %q, want it unset", got)
	}
	if got := res.GetHeader("Vary"); got != "Origin" {
		t.Errorf("Vary = %q, want %q", got, "Origin")
	}
}

func TestPassingStaticOptionsOriginOfFalseDisablesCors(t *testing.T) {
	options := &cors.Options{
		Origin:      false,
		Methods:     []string{"FOO", "bar"},
		Headers:     []string{"FIZZ", "buzz"},
		Credentials: true,
		MaxAge:      123,
	}

	res := runRequest(t, cors.NewWithOptions(options), newFakeRequest("GET", nil), nil)

	for _, name := range corsResponseHeaders {
		if got, ok := res.LookupHeader(name); ok {
			t.Errorf("%s = %q, want it unset", name, got)
		}
	}
}

func TestPassingStaticOptionsCanOverrideOrigin(t *testing.T) {
	options := &cors.Options{Origin: "http://example.com"}

	res := runRequest(t, cors.NewWithOptions(options), newFakeRequest("GET", nil), nil)

	if got := res.GetHeader("Access-Control-Allow-Origin"); got != "http://example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "http://example.com")
	}
}

func TestPassingStaticOptionsIncludesVaryHeaderForSpecificOrigins(t *testing.T) {
	options := &cors.Options{Origin: "http://example.com"}

	res := runRequest(t, cors.NewWithOptions(options), newFakeRequest("GET", nil), nil)

	if got := res.GetHeader("Vary"); got != "Origin" {
		t.Errorf("Vary = %q, want %q", got, "Origin")
	}
}

func TestPassingStaticOptionsAppendsToAnExistingVaryHeader(t *testing.T) {
	res := newFakeResponse()
	res.SetHeader("Vary", "Foo")

	options := &cors.Options{Origin: "http://example.com"}
	runRequest(t, cors.NewWithOptions(options), newFakeRequest("GET", nil), res)

	if got := res.GetHeader("Vary"); got != "Foo, Origin" {
		t.Errorf("Vary = %q, want %q", got, "Foo, Origin")
	}
}

func TestPassingStaticOptionsOriginDefaultsToStar(t *testing.T) {
	res := runRequest(t, cors.New(), newFakeRequest("GET", nil), nil)

	if got := res.GetHeader("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "*")
	}
}

func TestPassingStaticOptionsSpecifyingTrueForOriginReflectsRequestingOrigin(t *testing.T) {
	options := &cors.Options{Origin: true}

	res := runRequest(t, cors.NewWithOptions(options), newFakeRequest("GET", nil), nil)

	if got := res.GetHeader("Access-Control-Allow-Origin"); got != "http://example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "http://example.com")
	}
}

func TestPassingStaticOptionsShouldAllowOriginWhenCallbackReturnsTrue(t *testing.T) {
	options := &cors.Options{Origin: cors.OriginDelegate(func(sentOrigin string, cb func(err error, allow any)) {
		cb(nil, true)
	})}

	res := runRequest(t, cors.NewWithOptions(options), newFakeRequest("GET", nil), nil)

	if got := res.GetHeader("Access-Control-Allow-Origin"); got != "http://example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "http://example.com")
	}
}

func TestPassingStaticOptionsShouldNotAllowOriginWhenCallbackReturnsFalse(t *testing.T) {
	options := &cors.Options{Origin: cors.OriginDelegate(func(sentOrigin string, cb func(err error, allow any)) {
		cb(nil, false)
	})}

	res := runRequest(t, cors.NewWithOptions(options), newFakeRequest("GET", nil), nil)

	for _, name := range corsResponseHeaders {
		if got, ok := res.LookupHeader(name); ok {
			t.Errorf("%s = %q, want it unset", name, got)
		}
	}
}

func TestPassingStaticOptionsShouldNotOverrideOptionsOriginCallback(t *testing.T) {
	options := &cors.Options{Origin: cors.OriginDelegate(func(sentOrigin string, cb func(err error, allow any)) {
		cb(nil, sentOrigin == "http://example.com")
	})}
	middleware := cors.NewWithOptions(options)

	allowed := runRequest(t, middleware, newFakeRequest("GET", nil), nil)
	if got := allowed.GetHeader("Access-Control-Allow-Origin"); got != "http://example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "http://example.com")
	}

	denied := runRequest(t, middleware, newFakeRequest("GET", map[string]string{
		"origin": "http://localhost",
	}), nil)
	for _, name := range corsResponseHeaders {
		if got, ok := denied.LookupHeader(name); ok {
			t.Errorf("%s = %q, want it unset for a denied origin", name, got)
		}
	}
}

func TestPassingStaticOptionsCanOverrideMethods(t *testing.T) {
	options := &cors.Options{Methods: []string{"method1", "method2"}}

	res := runPreflight(t, cors.NewWithOptions(options), newFakeRequest("OPTIONS", nil))

	if got := res.StatusCode(); got != 204 {
		t.Errorf("status code = %d, want %d", got, 204)
	}
	if got := res.GetHeader("Access-Control-Allow-Methods"); got != "method1,method2" {
		t.Errorf("Access-Control-Allow-Methods = %q, want %q", got, "method1,method2")
	}
}

func TestPassingStaticOptionsMethodsDefaultsToGetHeadPutPatchPostDelete(t *testing.T) {
	res := runPreflight(t, cors.New(), newFakeRequest("OPTIONS", nil))

	if got := res.StatusCode(); got != 204 {
		t.Errorf("status code = %d, want %d", got, 204)
	}
	if got := res.GetHeader("Access-Control-Allow-Methods"); got != defaultMethods {
		t.Errorf("Access-Control-Allow-Methods = %q, want %q", got, defaultMethods)
	}
}

func TestPassingStaticOptionsCanSpecifyAllowedHeadersAsArray(t *testing.T) {
	options := &cors.Options{AllowedHeaders: []string{"header1", "header2"}}

	res := runPreflight(t, cors.NewWithOptions(options), newFakeRequest("OPTIONS", nil))

	if got := res.GetHeader("Access-Control-Allow-Headers"); got != "header1,header2" {
		t.Errorf("Access-Control-Allow-Headers = %q, want %q", got, "header1,header2")
	}
	if got, ok := res.LookupHeader("Vary"); ok {
		t.Errorf("Vary = %q, want it unset", got)
	}
}

func TestPassingStaticOptionsCanSpecifyAllowedHeadersAsString(t *testing.T) {
	options := &cors.Options{AllowedHeaders: "header1,header2"}

	res := runPreflight(t, cors.NewWithOptions(options), newFakeRequest("OPTIONS", nil))

	if got := res.GetHeader("Access-Control-Allow-Headers"); got != "header1,header2" {
		t.Errorf("Access-Control-Allow-Headers = %q, want %q", got, "header1,header2")
	}
	if got, ok := res.LookupHeader("Vary"); ok {
		t.Errorf("Vary = %q, want it unset", got)
	}
}

func TestPassingStaticOptionsSpecifyingAnEmptyListOfAllowedHeadersWillResultInNoResponseHeaderForAllowedHeaders(t *testing.T) {
	options := &cors.Options{AllowedHeaders: []string{}}

	res := runRequest(t, cors.NewWithOptions(options), newFakeRequest("GET", nil), nil)

	if got, ok := res.LookupHeader("Access-Control-Allow-Headers"); ok {
		t.Errorf("Access-Control-Allow-Headers = %q, want it unset", got)
	}
	if got, ok := res.LookupHeader("Vary"); ok {
		t.Errorf("Vary = %q, want it unset", got)
	}
}

func TestPassingStaticOptionsIfNoAllowedHeadersAreSpecifiedDefaultsToRequestedAllowedHeaders(t *testing.T) {
	res := runPreflight(t, cors.New(), newFakeRequest("OPTIONS", nil))

	if got := res.GetHeader("Access-Control-Allow-Headers"); got != "x-header-1, x-header-2" {
		t.Errorf("Access-Control-Allow-Headers = %q, want %q", got, "x-header-1, x-header-2")
	}
	if got := res.GetHeader("Vary"); got != "Access-Control-Request-Headers" {
		t.Errorf("Vary = %q, want %q", got, "Access-Control-Request-Headers")
	}
}

func TestPassingStaticOptionsCanSpecifyExposedHeadersAsArray(t *testing.T) {
	options := &cors.Options{ExposedHeaders: []string{"custom-header1", "custom-header2"}}

	res := runRequest(t, cors.NewWithOptions(options), newFakeRequest("GET", nil), nil)

	if got := res.GetHeader("Access-Control-Expose-Headers"); got != "custom-header1,custom-header2" {
		t.Errorf("Access-Control-Expose-Headers = %q, want %q", got, "custom-header1,custom-header2")
	}
}

func TestPassingStaticOptionsCanSpecifyExposedHeadersAsString(t *testing.T) {
	options := &cors.Options{ExposedHeaders: "custom-header1,custom-header2"}

	res := runRequest(t, cors.NewWithOptions(options), newFakeRequest("GET", nil), nil)

	if got := res.GetHeader("Access-Control-Expose-Headers"); got != "custom-header1,custom-header2" {
		t.Errorf("Access-Control-Expose-Headers = %q, want %q", got, "custom-header1,custom-header2")
	}
}

func TestPassingStaticOptionsSpecifyingAnEmptyListOfExposedHeadersWillResultInNoResponseHeaderForExposedHeaders(t *testing.T) {
	options := &cors.Options{ExposedHeaders: []string{}}

	res := runRequest(t, cors.NewWithOptions(options), newFakeRequest("GET", nil), nil)

	if got, ok := res.LookupHeader("Access-Control-Expose-Headers"); ok {
		t.Errorf("Access-Control-Expose-Headers = %q, want it unset", got)
	}
}

func TestPassingStaticOptionsIncludesCredentialsIfExplicitlyEnabled(t *testing.T) {
	options := &cors.Options{Credentials: true}

	res := runPreflight(t, cors.NewWithOptions(options), newFakeRequest("OPTIONS", nil))

	if got := res.GetHeader("Access-Control-Allow-Credentials"); got != enabled {
		t.Errorf("Access-Control-Allow-Credentials = %q, want %q", got, enabled)
	}
}

func TestPassingStaticOptionsDoesNotIncludeCredentialsUnlessExplicitlyEnabled(t *testing.T) {
	res := runRequest(t, cors.New(), newFakeRequest("GET", nil), nil)

	if got, ok := res.LookupHeader("Access-Control-Allow-Credentials"); ok {
		t.Errorf("Access-Control-Allow-Credentials = %q, want it unset", got)
	}
}

func TestPassingStaticOptionsIncludesMaxAgeWhenSpecified(t *testing.T) {
	options := &cors.Options{MaxAge: 456}

	res := runPreflight(t, cors.NewWithOptions(options), newFakeRequest("OPTIONS", nil))

	if got := res.GetHeader("Access-Control-Max-Age"); got != "456" {
		t.Errorf("Access-Control-Max-Age = %q, want %q", got, "456")
	}
}

func TestPassingStaticOptionsIncludesMaxAgeWhenSpecifiedAndEqualsToZero(t *testing.T) {
	options := &cors.Options{MaxAge: 0}

	res := runPreflight(t, cors.NewWithOptions(options), newFakeRequest("OPTIONS", nil))

	if got := res.GetHeader("Access-Control-Max-Age"); got != "0" {
		t.Errorf("Access-Control-Max-Age = %q, want %q", got, "0")
	}
}

func TestPassingStaticOptionsDoesNotIncludeMaxAgeUnlessSpecified(t *testing.T) {
	res := runRequest(t, cors.New(), newFakeRequest("GET", nil), nil)

	if got, ok := res.LookupHeader("Access-Control-Max-Age"); ok {
		t.Errorf("Access-Control-Max-Age = %q, want it unset", got)
	}
}

func TestPassingAFunctionToBuildOptionsHandlesOptionsSpecifiedViaCallback(t *testing.T) {
	delegate := cors.OptionsDelegate(func(req cors.Request, cb func(err error, options *cors.Options)) {
		cb(nil, &cors.Options{Origin: "delegate.com"})
	})

	res := runRequest(t, cors.NewWithDelegate(delegate), newFakeRequest("GET", nil), nil)

	if got := res.GetHeader("Access-Control-Allow-Origin"); got != "delegate.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "delegate.com")
	}
}

func TestPassingAFunctionToBuildOptionsHandlesOptionsSpecifiedViaCallbackForPreflight(t *testing.T) {
	delegate := cors.OptionsDelegate(func(req cors.Request, cb func(err error, options *cors.Options)) {
		cb(nil, &cors.Options{Origin: "delegate.com", MaxAge: 1000})
	})

	res := runPreflight(t, cors.NewWithDelegate(delegate), newFakeRequest("OPTIONS", nil))

	if got := res.GetHeader("Access-Control-Allow-Origin"); got != "delegate.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "delegate.com")
	}
	if got := res.GetHeader("Access-Control-Max-Age"); got != "1000" {
		t.Errorf("Access-Control-Max-Age = %q, want %q", got, "1000")
	}
}

func TestPassingAFunctionToBuildOptionsHandlesErrorSpecifiedViaCallback(t *testing.T) {
	delegate := cors.OptionsDelegate(func(req cors.Request, cb func(err error, options *cors.Options)) {
		cb(errors.New("some error"), nil)
	})

	var received error
	called := false
	cors.NewWithDelegate(delegate)(newFakeRequest("GET", nil), newFakeResponse(), func(err error) {
		received = err
		called = true
	})

	if !called {
		t.Fatal("next was not called")
	}
	if received == nil || received.Error() != "some error" {
		t.Errorf("next err = %v, want %q", received, "some error")
	}
}

// runRequest runs a non preflight request and asserts that control was passed
// to the next middleware without an error. When res is nil a fresh response is
// created. The response is returned so headers can be asserted.
func runRequest(t *testing.T, middleware cors.Middleware, req *FakeRequest, res *FakeResponse) *FakeResponse {
	t.Helper()

	if res == nil {
		res = newFakeResponse()
	}

	called := false
	middleware(req, res, func(err error) {
		if err != nil {
			t.Errorf("next was called with %v", err)
		}
		called = true
	})

	if !called {
		t.Fatal("next was not called")
	}
	return res
}

// runPreflight runs a preflight request and asserts that it was shortcircuited:
// the response must be finished and the next middleware must never run.
func runPreflight(t *testing.T, middleware cors.Middleware, req *FakeRequest) *FakeResponse {
	t.Helper()

	res := newFakeResponse()
	finished := false
	res.On("finish", func() { finished = true })

	middleware(req, res, func(err error) {
		if err != nil {
			t.Errorf("next was called with %v", err)
			return
		}
		t.Error("next should not be called")
	})

	if !finished {
		t.Fatal("the response was never finished")
	}
	return res
}

// FakeRequest is the stand in request used throughout this suite.
type FakeRequest struct {
	headers map[string]string
	method  string
}

func newFakeRequest(method string, headers map[string]string) *FakeRequest {
	if headers == nil {
		headers = map[string]string{
			"origin":                         "http://example.com",
			"access-control-request-headers": "x-header-1, x-header-2",
		}
	}
	if method == "" {
		method = "GET"
	}
	return &FakeRequest{headers: headers, method: method}
}

// Method returns the request method.
func (r *FakeRequest) Method() string { return r.method }

// Header returns the named request header, or an empty string when absent.
func (r *FakeRequest) Header(name string) string { return r.headers[strings.ToLower(name)] }

// FakeResponse is the stand in response used throughout this suite. It records
// headers and the status code, and emits a "finish" event when ended.
type FakeResponse struct {
	headers    map[string]string
	statusCode int
	listeners  map[string][]func()
}

func newFakeResponse() *FakeResponse {
	return &FakeResponse{
		headers:    map[string]string{},
		statusCode: 200,
		listeners:  map[string][]func(){},
	}
}

// On registers a listener for the named event.
func (r *FakeResponse) On(event string, listener func()) {
	r.listeners[event] = append(r.listeners[event], listener)
}

// End completes the response and emits "finish".
func (r *FakeResponse) End() {
	for _, listener := range r.listeners["finish"] {
		listener()
	}
}

// GetHeader returns the named header, or an empty string when unset.
func (r *FakeResponse) GetHeader(name string) string { return r.headers[strings.ToLower(name)] }

// LookupHeader returns the named header along with whether it was ever set,
// which distinguishes an absent header from an empty one.
func (r *FakeResponse) LookupHeader(name string) (string, bool) {
	value, ok := r.headers[strings.ToLower(name)]
	return value, ok
}

// SetHeader sets a header.
func (r *FakeResponse) SetHeader(name, value string) { r.headers[strings.ToLower(name)] = value }

// SetStatusCode records the status code.
func (r *FakeResponse) SetStatusCode(code int) { r.statusCode = code }

// StatusCode returns the recorded status code.
func (r *FakeResponse) StatusCode() int { return r.statusCode }

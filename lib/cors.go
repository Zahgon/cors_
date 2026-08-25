// Package cors provides Connect/Express style middleware that can be used to
// enable CORS with various options.
//
// It is a port of the `cors` middleware for Node.js
// (https://github.com/expressjs/cors) and reproduces its behaviour exactly,
// including its defaults, its header ordering and its edge cases.
package cors

import (
	"reflect"
	"regexp"
	"strings"

	"github.com/expressjs/cors/internal/vary"
)

// Request is the subset of an incoming HTTP request the middleware needs.
//
// Header performs a case insensitive lookup and returns an empty string when
// the header is absent, which stands in for JavaScript's `undefined`.
type Request interface {
	Method() string
	Header(name string) string
}

// Response is the subset of an outgoing HTTP response the middleware needs. It
// mirrors the `getHeader`/`setHeader`/`statusCode`/`end` surface the JavaScript
// version relied on.
type Response interface {
	GetHeader(name string) string
	SetHeader(name, value string)
	SetStatusCode(code int)
	End()
}

// NextFunc mirrors Connect's `next(err)` continuation. A nil error means the
// request should simply continue down the middleware stack.
type NextFunc func(err error)

// Middleware is a Connect style middleware function.
type Middleware func(req Request, res Response, next NextFunc)

// OriginDelegate mirrors supplying a function for `options.origin`. It receives
// the request's `Origin` header and calls cb with either an error or the origin
// configuration to use, which may be a bool, a string, a *regexp.Regexp or a
// slice of any of those.
type OriginDelegate func(origin string, cb func(err error, allow any))

// OptionsDelegate mirrors passing a function to cors() so that options can be
// computed per request.
type OptionsDelegate func(req Request, cb func(err error, options *Options))

// Options configures the middleware. Every field is optional: a nil field is
// treated as an absent key and falls back to the package default.
//
// The fields are dynamically typed because the original accepted several shapes
// for most of them:
//
//	Origin               bool, string, *regexp.Regexp, a slice of those, or an OriginDelegate
//	Methods              string or []string
//	AllowedHeaders       string or []string
//	Headers              string or []string (legacy alias for AllowedHeaders)
//	ExposedHeaders       string or []string
//	Credentials          bool
//	MaxAge               a number
//	PreflightContinue    bool
//	OptionsSuccessStatus a number
type Options struct {
	Origin               any
	Methods              any
	AllowedHeaders       any
	Headers              any
	ExposedHeaders       any
	Credentials          any
	MaxAge               any
	PreflightContinue    any
	OptionsSuccessStatus any
}

// config is the fully resolved option set for a single request. It is a value
// type, so merging never mutates the caller's Options.
type config struct {
	origin               any
	methods              any
	allowedHeaders       any
	headers              any
	exposedHeaders       any
	credentials          any
	maxAge               any
	preflightContinue    any
	optionsSuccessStatus any
}

// defaults mirrors the `defaults` object of the JavaScript implementation.
func defaults() config {
	return config{
		origin:               "*",
		methods:              "GET,HEAD,PUT,PATCH,POST,DELETE",
		preflightContinue:    false,
		optionsSuccessStatus: 204,
	}
}

// merge mirrors `assign({}, defaults, options)`: values supplied by the caller
// override the defaults, and the caller's object is left untouched.
func merge(base config, options *Options) config {
	if options == nil {
		return base
	}

	merged := base
	for _, override := range []struct {
		value any
		into  *any
	}{
		{options.Origin, &merged.origin},
		{options.Methods, &merged.methods},
		{options.AllowedHeaders, &merged.allowedHeaders},
		{options.Headers, &merged.headers},
		{options.ExposedHeaders, &merged.exposedHeaders},
		{options.Credentials, &merged.credentials},
		{options.MaxAge, &merged.maxAge},
		{options.PreflightContinue, &merged.preflightContinue},
		{options.OptionsSuccessStatus, &merged.optionsSuccessStatus},
	} {
		if override.value != nil {
			*override.into = override.value
		}
	}
	return merged
}

// New returns middleware configured with the package defaults, mirroring
// `cors()`.
func New() Middleware {
	return NewWithOptions(nil)
}

// NewWithOptions returns middleware configured with the given static options,
// mirroring `cors(options)`. The options are never modified.
func NewWithOptions(options *Options) Middleware {
	return newMiddleware(func(_ Request, cb func(err error, options *Options)) {
		cb(nil, options)
	})
}

// NewWithDelegate returns middleware whose options are computed per request,
// mirroring `cors(delegate)`.
func NewWithDelegate(delegate OptionsDelegate) Middleware {
	return newMiddleware(delegate)
}

func newMiddleware(optionsCallback OptionsDelegate) Middleware {
	return func(req Request, res Response, next NextFunc) {
		optionsCallback(req, func(err error, options *Options) {
			if err != nil {
				next(err)
				return
			}

			corsOptions := merge(defaults(), options)

			var originCallback OriginDelegate
			if isTruthy(corsOptions.origin) {
				switch delegate := corsOptions.origin.(type) {
				case OriginDelegate:
					originCallback = delegate
				case func(origin string, cb func(err error, allow any)):
					originCallback = delegate
				default:
					staticOrigin := corsOptions.origin
					originCallback = func(_ string, cb func(err error, allow any)) {
						cb(nil, staticOrigin)
					}
				}
			}

			if originCallback == nil {
				next(nil)
				return
			}

			originCallback(req.Header("origin"), func(originErr error, origin any) {
				if originErr != nil || !isTruthy(origin) {
					next(originErr)
					return
				}
				corsOptions.origin = origin
				apply(corsOptions, req, res, next)
			})
		})
	}
}

// apply is the core of the middleware, mirroring the JavaScript `cors`
// function. Header ordering is significant and is preserved verbatim.
func apply(options config, req Request, res Response, next NextFunc) {
	var headers []headerNode
	method := strings.ToUpper(req.Method())

	if method == "OPTIONS" {
		headers = append(headers,
			configureOrigin(options, req),
			configureCredentials(options),
			configureMethods(options),
			configureAllowedHeaders(options, req),
			configureMaxAge(options),
			configureExposedHeaders(options),
		)
		if err := applyHeaders(headers, res); err != nil {
			next(err)
			return
		}

		if isTruthy(options.preflightContinue) {
			next(nil)
			return
		}

		// Safari (and potentially other browsers) need content-length 0 for
		// 204 or they just hang waiting for a body.
		res.SetStatusCode(toStatusCode(options.optionsSuccessStatus))
		res.SetHeader("Content-Length", "0")
		res.End()
		return
	}

	headers = append(headers,
		configureOrigin(options, req),
		configureCredentials(options),
		configureExposedHeaders(options),
	)
	if err := applyHeaders(headers, res); err != nil {
		next(err)
		return
	}
	next(nil)
}

func isOriginAllowed(origin string, allowedOrigin any) bool {
	if allowed, ok := toList(allowedOrigin); ok {
		for _, candidate := range allowed {
			if isOriginAllowed(origin, candidate) {
				return true
			}
		}
		return false
	}
	if allowed, ok := allowedOrigin.(string); ok {
		return origin == allowed
	}
	if allowed, ok := allowedOrigin.(*regexp.Regexp); ok {
		return allowed.MatchString(origin)
	}
	return isTruthy(allowedOrigin)
}

func configureOrigin(options config, req Request) headerNode {
	requestOrigin := req.Header("origin")
	var headers headerGroup

	origin, isStringOrigin := options.origin.(string)
	if !isTruthy(options.origin) || (isStringOrigin && origin == "*") {
		// Allow any origin.
		headers = append(headers, headerGroup{headerField{"Access-Control-Allow-Origin", "*"}})
		return headers
	}

	if isStringOrigin {
		// Fixed origin.
		headers = append(headers, headerGroup{headerField{"Access-Control-Allow-Origin", origin}})
		headers = append(headers, headerGroup{headerField{"Vary", "Origin"}})
		return headers
	}

	// Reflect origin.
	value := ""
	if isOriginAllowed(requestOrigin, options.origin) {
		value = requestOrigin
	}
	headers = append(headers, headerGroup{headerField{"Access-Control-Allow-Origin", value}})
	headers = append(headers, headerGroup{headerField{"Vary", "Origin"}})
	return headers
}

func configureMethods(options config) headerNode {
	return headerField{"Access-Control-Allow-Methods", joinValue(options.methods)}
}

func configureCredentials(options config) headerNode {
	if credentials, ok := options.credentials.(bool); ok && credentials {
		return headerField{"Access-Control-Allow-Credentials", "true"}
	}
	return nil
}

func configureAllowedHeaders(options config, req Request) headerNode {
	allowedHeaders := options.allowedHeaders
	if !isTruthy(allowedHeaders) {
		allowedHeaders = options.headers
	}

	var headers headerGroup
	var value string

	if !isTruthy(allowedHeaders) {
		// Reflect the request's headers.
		value = req.Header("access-control-request-headers")
		headers = append(headers, headerGroup{headerField{"Vary", "Access-Control-Request-Headers"}})
	} else {
		value = joinValue(allowedHeaders)
	}

	if value != "" {
		headers = append(headers, headerGroup{headerField{"Access-Control-Allow-Headers", value}})
	}
	return headers
}

func configureExposedHeaders(options config) headerNode {
	if !isTruthy(options.exposedHeaders) {
		return nil
	}
	value := joinValue(options.exposedHeaders)
	if value == "" {
		return nil
	}
	return headerField{"Access-Control-Expose-Headers", value}
}

func configureMaxAge(options config) headerNode {
	if !isNumber(options.maxAge) && !isTruthy(options.maxAge) {
		return nil
	}
	maxAge := jsToString(options.maxAge)
	if maxAge == "" {
		return nil
	}
	return headerField{"Access-Control-Max-Age", maxAge}
}

// toStatusCode coerces a configured status code to an int.
func toStatusCode(value any) int {
	if value == nil {
		return 0
	}

	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int(rv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int(rv.Uint())
	case reflect.Float32, reflect.Float64:
		return int(rv.Float())
	default:
		return 0
	}
}

// headerNode is either a single header field or a nested group of them. The
// JavaScript version built an array that mixed objects and arrays of objects;
// this mirrors that structure so the traversal order stays identical.
type headerNode any

type headerField struct {
	Key   string
	Value string
}

type headerGroup []headerNode

// applyHeaders walks the header tree and writes every non-empty field to the
// response. Fields with an empty value are skipped, mirroring the falsy checks
// of the original.
func applyHeaders(headers []headerNode, res Response) error {
	for _, header := range headers {
		if header == nil {
			continue
		}

		switch node := header.(type) {
		case headerGroup:
			if err := applyHeaders(node, res); err != nil {
				return err
			}
		case headerField:
			if node.Key == "Vary" {
				if node.Value != "" {
					if err := vary.Vary(res, node.Value); err != nil {
						return err
					}
				}
			} else if node.Value != "" {
				res.SetHeader(node.Key, node.Value)
			}
		}
	}
	return nil
}

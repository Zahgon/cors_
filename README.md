# cors

[![NPM Version][npm-image]][npm-url]
[![Build Status][ci-image]][ci-url]

CORS is a package for providing Connect/Express style middleware that can be
used to enable [CORS](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS)
with various options.

This is the Go port of the original Node.js middleware. It has no dependencies
outside of the standard library.

**[Installation](#installation)**
**[Usage](#usage)**
  - [Simple Usage](#simple-usage-enable-all-cors-requests)
  - [Enable CORS for a Single Route](#enable-cors-for-a-single-route)
  - [Configuring CORS](#configuring-cors)
  - [Configuring CORS Asynchronously](#configuring-cors-asynchronously)
**[Configuration Options](#configuration-options)**
**[Demo](#demo)**
**[License](#license)**
**[Author](#author)**

## Installation

```sh
go get github.com/expressjs/cors
```

## Usage

### Simple Usage (Enable *All* CORS Requests)

```go
package main

import (
	"log"
	"net/http"

	cors "github.com/expressjs/cors/lib"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/products/1", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"msg":"This is CORS-enabled for all origins!"}`))
	})

	log.Fatal(http.ListenAndServe(":8080", cors.New().Handler(mux)))
}
```

### Enable CORS for a Single Route

```go
mux := http.NewServeMux()
mux.Handle("/products/1", cors.New().Handler(http.HandlerFunc(getProduct)))
```

### Configuring CORS

```go
middleware := cors.NewWithOptions(&cors.Options{
	Origin:               "http://example.com",
	OptionsSuccessStatus: 200, // some legacy browsers choke on 204
})

log.Fatal(http.ListenAndServe(":8080", middleware.Handler(mux)))
```

`Origin` may also be a `bool`, a `*regexp.Regexp`, a slice mixing strings and
regular expressions, or a `cors.OriginDelegate` for a dynamic decision:

```go
allowlist := map[string]bool{
	"http://example1.com": true,
	"http://example2.com": true,
}

middleware := cors.NewWithOptions(&cors.Options{
	Origin: cors.OriginDelegate(func(origin string, cb func(err error, allow any)) {
		if allowlist[origin] {
			cb(nil, true)
			return
		}
		cb(errors.New("Not allowed by CORS"), nil)
	}),
})
```

### Configuring CORS Asynchronously

```go
allowlist := map[string]bool{"http://example1.com": true, "http://example2.com": true}

delegate := cors.OptionsDelegate(func(req cors.Request, cb func(err error, options *cors.Options)) {
	// Reflect (enable) the requested origin in the CORS response, or disable
	// CORS for this request.
	cb(nil, &cors.Options{Origin: allowlist[req.Header("Origin")]})
})

middleware := cors.NewWithDelegate(delegate)
```

## Configuration Options

Every field of `cors.Options` is optional. A `nil` field falls back to the
default below.

* `Origin`: Configures the **Access-Control-Allow-Origin** CORS header.
  Possible values:
  - `bool` - set to `true` to reflect the [request origin](http://tools.ietf.org/html/draft-abarth-origin-09), or `false` to disable CORS.
  - `string` - set to a specific origin. Setting it to `"*"` will allow any origin.
  - `*regexp.Regexp` - set to a pattern that will be used to test the request origin. If it matches, the request origin will be reflected. For example the pattern `\.example2\.com$` will reflect any request that is coming from an origin ending with `.example2.com`.
  - a slice - set to a slice of valid origins. Each value can be a `string` or a `*regexp.Regexp`, for example `[]any{"http://example1.com", regexp.MustCompile(`\.example2\.com$`)}`.
  - `cors.OriginDelegate` - set to a function implementing custom logic. It takes the request origin as the first parameter and a callback as the second, which expects the signature `cb(err error, allow any)`.
* `Methods`: Configures the **Access-Control-Allow-Methods** CORS header. Expects a comma delimited `string` (ex: `"GET,PUT,POST"`) or a `[]string` (ex: `[]string{"GET", "PUT", "POST"}`).
* `AllowedHeaders`: Configures the **Access-Control-Allow-Headers** CORS header. Expects a comma delimited `string` (ex: `"Content-Type,Authorization"`) or a `[]string` (ex: `[]string{"Content-Type", "Authorization"}`). If not specified, defaults to reflecting the headers specified in the request's **Access-Control-Request-Headers** header.
* `ExposedHeaders`: Configures the **Access-Control-Expose-Headers** CORS header. Expects a comma delimited `string` (ex: `"Content-Range,X-Content-Range"`) or a `[]string` (ex: `[]string{"Content-Range", "X-Content-Range"}`). If not specified, no custom headers are exposed.
* `Credentials`: Configures the **Access-Control-Allow-Credentials** CORS header. Set to `true` to pass the header, otherwise it is omitted.
* `MaxAge`: Configures the **Access-Control-Max-Age** CORS header. Set to an integer to pass the header, otherwise it is omitted.
* `PreflightContinue`: Pass the CORS preflight response to the next handler.
* `OptionsSuccessStatus`: Provides a status code to use for successful `OPTIONS` requests, since some legacy browsers (IE11, various SmartTVs) choke on `204`.

The default configuration is the equivalent of:

```go
&cors.Options{
	Origin:               "*",
	Methods:              "GET,HEAD,PUT,PATCH,POST,DELETE",
	PreflightContinue:    false,
	OptionsSuccessStatus: 204,
}
```

For details on the effect of each CORS header, read [this](https://web.dev/cross-origin-resource-sharing/) article on web.dev.

## Demo

A demo that illustrates CORS working (and not working) using jQuery is available here: [http://node-cors-client.herokuapp.com/](http://node-cors-client.herokuapp.com/)

Code for that demo can be found here:

* Client: [https://github.com/TroyGoode/node-cors-client](https://github.com/TroyGoode/node-cors-client)
* Server: [https://github.com/TroyGoode/node-cors-server](https://github.com/TroyGoode/node-cors-server)

## License

[MIT License](http://www.opensource.org/licenses/mit-license.php)

## Author

[Troy Goode](https://github.com/TroyGoode) ([troygoode@gmail.com](mailto:troygoode@gmail.com))

[ci-image]: https://badgen.net/github/checks/expressjs/cors/master?label=ci
[ci-url]: https://github.com/expressjs/cors/actions/workflows/ci.yml
[npm-image]: https://img.shields.io/npm/v/cors.svg
[npm-url]: https://npmjs.org/package/cors

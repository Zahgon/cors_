// Package vary manipulates the HTTP `Vary` response header.
//
// It is a port of the `vary` npm package that the original JavaScript
// implementation of this middleware depended on, and it reproduces that
// package's behaviour exactly. The only deliberate difference is that the
// JavaScript version signalled invalid input by throwing a `TypeError`,
// whereas this package returns an `error`.
package vary

import (
	"errors"
	"regexp"
	"strings"
)

// fieldNameRegexp matches a valid HTTP header field name (RFC 7230 token).
var fieldNameRegexp = regexp.MustCompile("^[!#$%&'*+\\-.^_`|~0-9A-Za-z]+$")

var (
	// ErrResponseRequired mirrors `TypeError: res argument is required`.
	ErrResponseRequired = errors.New("res argument is required")
	// ErrFieldRequired mirrors `TypeError: field argument is required`.
	ErrFieldRequired = errors.New("field argument is required")
	// ErrInvalidFieldName mirrors
	// `TypeError: field argument contains an invalid header name`.
	ErrInvalidFieldName = errors.New("field argument contains an invalid header name")
)

// ResponseHeaders is the subset of a response object that this package needs.
// It matches the `getHeader`/`setHeader` pair the JavaScript version relied on.
type ResponseHeaders interface {
	GetHeader(name string) string
	SetHeader(name, value string)
}

// Append appends the given field, which may itself be a comma separated list,
// to the supplied `Vary` header value and returns the new header value.
//
// An empty field is rejected, matching the JavaScript version where a falsy
// `field` argument threw.
func Append(header, field string) (string, error) {
	if field == "" {
		return "", ErrFieldRequired
	}
	return appendFields(header, parse(field))
}

// AppendFields appends every field in fields to the supplied `Vary` header
// value and returns the new header value.
//
// A nil slice stands in for JavaScript's `null`/`undefined` and is rejected,
// while an empty (but non-nil) slice mirrors `[]`, which the JavaScript version
// accepted and treated as a no-op.
func AppendFields(header string, fields []string) (string, error) {
	if fields == nil {
		return "", ErrFieldRequired
	}
	return appendFields(header, fields)
}

// Vary marks the given field as varying in the response.
func Vary(res ResponseHeaders, field string) error {
	if res == nil {
		return ErrResponseRequired
	}

	val, err := Append(res.GetHeader("Vary"), field)
	if err != nil {
		return err
	}
	if val != "" {
		res.SetHeader("Vary", val)
	}
	return nil
}

// VaryFields marks every given field as varying in the response.
func VaryFields(res ResponseHeaders, fields []string) error {
	if res == nil {
		return ErrResponseRequired
	}

	val, err := AppendFields(res.GetHeader("Vary"), fields)
	if err != nil {
		return err
	}
	if val != "" {
		res.SetHeader("Vary", val)
	}
	return nil
}

func appendFields(header string, fields []string) (string, error) {
	// Assert on invalid field names.
	for _, field := range fields {
		if !fieldNameRegexp.MatchString(field) {
			return "", ErrInvalidFieldName
		}
	}

	// Existing, unspecified vary.
	if header == "*" {
		return header, nil
	}

	// Enumerate current values.
	val := header
	vals := parse(strings.ToLower(header))

	// Unspecified vary.
	if contains(fields, "*") || contains(vals, "*") {
		return "*", nil
	}

	for _, field := range fields {
		fld := strings.ToLower(field)

		// Append value (case preserving).
		if !contains(vals, fld) {
			vals = append(vals, fld)
			if val != "" {
				val = val + ", " + field
			} else {
				val = field
			}
		}
	}

	return val, nil
}

// parse splits a header value into an array of field names, reproducing the
// byte-wise tokenizer used by the JavaScript implementation. Note that, just
// like the original, an empty header yields a single empty entry.
func parse(header string) []string {
	end := 0
	start := 0
	list := make([]string, 0, 1)

	// Gather tokens.
	for i := 0; i < len(header); i++ {
		switch header[i] {
		case 0x20: // ' '
			if start == end {
				start = i + 1
				end = i + 1
			}
		case 0x2c: // ','
			list = append(list, header[start:end])
			start = i + 1
			end = i + 1
		default:
			end = i + 1
		}
	}

	// Final token.
	return append(list, header[start:end])
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

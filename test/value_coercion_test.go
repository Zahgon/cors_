package test

import (
	"math"
	"testing"

	cors "github.com/expressjs/cors/lib"
)

// preflightContinued reports whether a preflight request was handed to the next
// middleware instead of being answered, which is how the truthiness of the
// preflightContinue option becomes observable.
func preflightContinued(t *testing.T, preflightContinue any) bool {
	t.Helper()

	options := &cors.Options{PreflightContinue: preflightContinue}
	res := newFakeResponse()
	finished := false
	res.On("finish", func() { finished = true })

	called := false
	cors.NewWithOptions(options)(newFakeRequest("OPTIONS", nil), res, func(err error) {
		if err != nil {
			t.Errorf("next was called with %v", err)
		}
		called = true
	})

	if called == finished {
		t.Fatalf("next called = %t and response finished = %t, want exactly one", called, finished)
	}
	return called
}

func maxAgeHeaderFor(t *testing.T, maxAge any) string {
	t.Helper()

	options := &cors.Options{MaxAge: maxAge}
	res := runPreflight(t, cors.NewWithOptions(options), newFakeRequest("OPTIONS", nil))
	return res.GetHeader("Access-Control-Max-Age")
}

func methodsHeaderFor(t *testing.T, methods any) string {
	t.Helper()

	options := &cors.Options{Methods: methods}
	res := runPreflight(t, cors.NewWithOptions(options), newFakeRequest("OPTIONS", nil))
	return res.GetHeader("Access-Control-Allow-Methods")
}

func TestCoercionCredentialsRequiresExactlyTrue(t *testing.T) {
	res := runPreflight(t, cors.NewWithOptions(&cors.Options{Credentials: true}), newFakeRequest("OPTIONS", nil))
	if got := res.GetHeader("Access-Control-Allow-Credentials"); got != enabled {
		t.Errorf("Access-Control-Allow-Credentials = %q, want %q", got, enabled)
	}

	for _, value := range []any{nil, false, "true", 1} {
		res := runPreflight(t, cors.NewWithOptions(&cors.Options{Credentials: value}), newFakeRequest("OPTIONS", nil))
		if got, ok := res.LookupHeader("Access-Control-Allow-Credentials"); ok {
			t.Errorf("credentials %v: header = %q, want it unset", value, got)
		}
	}
}

func TestCoercionPreflightContinueTreatsEveryNonEmptyValueAsEnabled(t *testing.T) {
	pointee := 1

	for _, tc := range []struct {
		name  string
		value any
	}{
		{"bool", true},
		{"string", "yes"},
		{"int", 1},
		{"uint", uint(2)},
		{"float", 1.5},
		{"pointer", &pointee},
		{"empty slice", []string{}},
		{"map", map[string]string{"a": "b"}},
		{"struct", struct{}{}},
	} {
		if !preflightContinued(t, tc.value) {
			t.Errorf("preflightContinue %s: request was answered, want it passed on", tc.name)
		}
	}
}

func TestCoercionPreflightContinueTreatsEveryEmptyValueAsDisabled(t *testing.T) {
	var nilSlice []string
	var nilMap map[string]string
	var nilPointer *int
	var nilFunc func()
	var nilChannel chan int

	for _, tc := range []struct {
		name  string
		value any
	}{
		{"nil", nil},
		{"bool", false},
		{"empty string", ""},
		{"zero int", 0},
		{"zero uint", uint(0)},
		{"zero float", 0.0},
		{"NaN", math.NaN()},
		{"nil slice", nilSlice},
		{"nil map", nilMap},
		{"nil pointer", nilPointer},
		{"nil func", nilFunc},
		{"nil channel", nilChannel},
	} {
		if preflightContinued(t, tc.value) {
			t.Errorf("preflightContinue %s: request was passed on, want it answered", tc.name)
		}
	}
}

func TestCoercionMaxAgeRendersEveryNumericKind(t *testing.T) {
	for _, tc := range []struct {
		value any
		want  string
	}{
		{0, "0"},
		{123, "123"},
		{int64(-7), "-7"},
		{uint(8), "8"},
		{int8(9), "9"},
		{456.0, "456"},
		{-0.5, "-0.5"},
		{1e21, "1e+21"},
		{math.NaN(), "NaN"},
		{math.Inf(1), "Infinity"},
		{math.Inf(-1), "-Infinity"},
	} {
		if got := maxAgeHeaderFor(t, tc.value); got != tc.want {
			t.Errorf("maxAge %v: header = %q, want %q", tc.value, got, tc.want)
		}
	}
}

func TestCoercionMaxAgeRendersNonNumericValuesThatAreNotEmpty(t *testing.T) {
	for _, tc := range []struct {
		value any
		want  string
	}{
		{"600", "600"},
		{true, "true"},
		{[]int{1}, "[1]"},
	} {
		if got := maxAgeHeaderFor(t, tc.value); got != tc.want {
			t.Errorf("maxAge %v: header = %q, want %q", tc.value, got, tc.want)
		}
	}

	for _, value := range []any{nil, false, ""} {
		if got := maxAgeHeaderFor(t, value); got != "" {
			t.Errorf("maxAge %v: header = %q, want it unset", value, got)
		}
	}
}

func TestCoercionMethodsAcceptsArraysAndStrings(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value any
		want  string
	}{
		{"slice of strings", []string{"FOO", "bar"}, "FOO,bar"},
		{"array of strings", [2]string{"FOO", "bar"}, "FOO,bar"},
		{"slice of any", []any{"FOO", 2, true}, "FOO,2,true"},
		{"slice of ints", []int{1, 2}, "1,2"},
		{"string", "FOO,bar", "FOO,bar"},
	} {
		if got := methodsHeaderFor(t, tc.value); got != tc.want {
			t.Errorf("methods %s: header = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestCoercionExposedHeadersRendersUnknownValuesWithTheirGoRepresentation(t *testing.T) {
	options := &cors.Options{ExposedHeaders: []any{struct{ A int }{1}}}

	res := runRequest(t, cors.NewWithOptions(options), newFakeRequest("GET", nil), nil)

	if got := res.GetHeader("Access-Control-Expose-Headers"); got != "{1}" {
		t.Errorf("Access-Control-Expose-Headers = %q, want %q", got, "{1}")
	}
}

package cors

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
)

// The option values accepted by this package are dynamically typed, exactly as
// they were in the JavaScript original. The helpers below reproduce the handful
// of JavaScript coercion rules that the middleware relies on, so that a Go
// caller passing `false`, `""`, `0`, `nil` or an empty slice observes the same
// behaviour the JavaScript caller did.

// isTruthy reports whether value would be truthy in JavaScript.
//
// A nil interface stands in for `null`/`undefined`. Booleans, strings and
// numbers follow the usual rules. Everything else is an object, and objects are
// always truthy, which notably makes an empty (non-nil) slice truthy just like
// `[]` is.
func isTruthy(value any) bool {
	if value == nil {
		return false
	}

	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Bool:
		return rv.Bool()
	case reflect.String:
		return rv.String() != ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rv.Uint() != 0
	case reflect.Float32, reflect.Float64:
		f := rv.Float()
		return f != 0 && !math.IsNaN(f)
	case reflect.Slice, reflect.Map, reflect.Pointer, reflect.Func, reflect.Interface, reflect.Chan:
		return !rv.IsNil()
	default:
		return true
	}
}

// isString reports whether value is a string, mirroring the original's
// `typeof s === 'string' || s instanceof String` check.
func isString(value any) bool {
	_, ok := value.(string)
	return ok
}

// isNumber reports whether value would satisfy `typeof value === 'number'`.
func isNumber(value any) bool {
	if value == nil {
		return false
	}

	switch reflect.ValueOf(value).Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

// toList reports whether value is an array in the JavaScript sense and, if so,
// returns its elements. Strings are not arrays, and a nil slice stands in for
// `null` rather than for `[]`.
func toList(value any) ([]any, bool) {
	if value == nil || isString(value) {
		return nil, false
	}

	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Slice:
		if rv.IsNil() {
			return nil, false
		}
	case reflect.Array:
	default:
		return nil, false
	}

	items := make([]any, rv.Len())
	for i := range items {
		items[i] = rv.Index(i).Interface()
	}
	return items, true
}

// joinValue reproduces the `if (value.join) value = value.join(',')` idiom: an
// array is joined with commas, anything else is stringified as JavaScript would.
func joinValue(value any) string {
	if items, ok := toList(value); ok {
		parts := make([]string, len(items))
		for i, item := range items {
			parts[i] = jsToString(item)
		}
		return strings.Join(parts, ",")
	}
	return jsToString(value)
}

// jsToString reproduces JavaScript's `String(value)` for the value kinds this
// package accepts.
func jsToString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case bool:
		return strconv.FormatBool(typed)
	}

	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(rv.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(rv.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return formatNumber(rv.Float())
	default:
		return fmt.Sprint(value)
	}
}

// formatNumber renders a float the way JavaScript's Number#toString does for
// the values that make sense here, so that 456 prints as "456" and not "456.0".
func formatNumber(f float64) string {
	if math.IsNaN(f) {
		return "NaN"
	}
	if math.IsInf(f, 1) {
		return "Infinity"
	}
	if math.IsInf(f, -1) {
		return "-Infinity"
	}
	if f == math.Trunc(f) && math.Abs(f) < 1e21 {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'g', -1, 64)
}

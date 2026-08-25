package support

import "os"

// The JavaScript suite was run with `mocha --require test/support/env`, which
// set NODE_ENV to "test" before anything else loaded. Importing this package
// has the same effect for the Go suite.
func init() {
	os.Setenv("GO_ENV", "test")
}

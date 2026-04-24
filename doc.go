// Package yaegibrowser embeds the Yaegi Go interpreter so that Go source can
// be executed inside a browser or Node.js Web Worker.
//
// The package exposes two entry points:
//
//   - Run: evaluates a package main program and captures stdout, stderr and
//     an exit code.
//   - RunTests: discovers TestXxx(*testing.T) functions in the source,
//     executes each one under a lightweight testing.T stand-in, and reports
//     pass/fail lists.
//
// A thin cmd/wasm binary wires these into syscall/js so a Worker script can
// call civnodeYaegi.run and civnodeYaegi.runTests directly. The host-side
// tests in runner_test.go exercise the same Run / RunTests functions without
// WebAssembly, which keeps go test ./... meaningful on any platform.
//
// See https://github.com/CivNode/yaegi-browser for build and integration
// details.
package yaegibrowser

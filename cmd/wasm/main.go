//go:build js && wasm

// Command wasm compiles to a single yaegi.wasm binary that attaches a
// civnodeYaegi object to the Worker's global scope. It exposes three
// functions: run, runTests, and version. All three take and return plain
// JavaScript objects — no promises, no callbacks — which keeps the Worker
// bridge on the CivNode side straightforward.
package main

import (
	"runtime"
	"syscall/js"
	"time"

	yaegibrowser "github.com/CivNode/yaegi-browser"
)

// builtAt is set at build time via -ldflags "-X main.builtAt=...". It falls
// back to "unknown" so the binary still works without the linker flag.
var builtAt = "unknown"

func main() {
	global := js.Global()
	api := js.ValueOf(map[string]any{})
	api.Set("run", js.FuncOf(runFunc))
	api.Set("runTests", js.FuncOf(runTestsFunc))
	api.Set("version", js.FuncOf(versionFunc))
	global.Set("civnodeYaegi", api)

	// Signal readiness via a well-known global flag so the Worker's glue
	// code can resolve its loading promise without polling.
	global.Set("civnodeYaegiReady", js.ValueOf(true))

	// Block forever: the Go runtime must stay alive while JS holds
	// references to js.FuncOf values.
	select {}
}

// runFunc expects (source string, timeoutMs number) and returns a plain
// object with stdout, stderr, exitCode, and durationMs.
func runFunc(this js.Value, args []js.Value) any {
	if len(args) < 1 || args[0].Type() != js.TypeString {
		return errorObject("run: expected (source string, timeoutMs number)")
	}
	source := args[0].String()
	timeout := parseTimeout(args, 1)
	res := yaegibrowser.Run(source, timeout)
	return map[string]any{
		"stdout":     res.Stdout,
		"stderr":     res.Stderr,
		"exitCode":   res.ExitCode,
		"durationMs": res.DurationMs,
	}
}

// runTestsFunc expects (source string, timeoutMs number) and returns a plain
// object with stdout, stderr, passed (array of names), failed (array of
// {name, message}), and durationMs.
func runTestsFunc(this js.Value, args []js.Value) any {
	if len(args) < 1 || args[0].Type() != js.TypeString {
		return errorObject("runTests: expected (source string, timeoutMs number)")
	}
	source := args[0].String()
	timeout := parseTimeout(args, 1)
	res := yaegibrowser.RunTests(source, timeout)

	passed := make([]any, len(res.Passed))
	for i, name := range res.Passed {
		passed[i] = name
	}
	failed := make([]any, len(res.Failed))
	for i, f := range res.Failed {
		failed[i] = map[string]any{"name": f.Name, "message": f.Message}
	}
	return map[string]any{
		"stdout":     res.Stdout,
		"stderr":     res.Stderr,
		"passed":     passed,
		"failed":     failed,
		"durationMs": res.DurationMs,
	}
}

// versionFunc reports linked yaegi and Go versions, plus the build timestamp
// supplied by the Makefile's release target.
func versionFunc(_ js.Value, _ []js.Value) any {
	return map[string]any{
		"yaegiVersion": yaegibrowser.YaegiVersion(),
		"goVersion":    runtime.Version(),
		"builtAt":      builtAt,
	}
}

// parseTimeout reads args[i] as a JS number of milliseconds and returns a
// time.Duration. If the slot is missing or not a number, 0 is returned and
// the runner falls back to its default timeout.
func parseTimeout(args []js.Value, i int) time.Duration {
	if len(args) <= i {
		return 0
	}
	v := args[i]
	if v.Type() != js.TypeNumber {
		return 0
	}
	ms := v.Int()
	if ms <= 0 {
		return 0
	}
	return time.Duration(ms) * time.Millisecond
}

// errorObject wraps a message in the same shape that run/runTests use so the
// caller can branch on exitCode without special casing the error path.
func errorObject(msg string) any {
	return map[string]any{
		"stdout":     "",
		"stderr":     msg,
		"exitCode":   2,
		"durationMs": 0,
	}
}

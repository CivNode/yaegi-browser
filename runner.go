package yaegibrowser

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"runtime/debug"
	"sync"
	"time"

	"github.com/traefik/yaegi/interp"

	"github.com/CivNode/yaegi-browser/internal/symbols"
)

// YaegiVersion reports the version of github.com/traefik/yaegi that was
// linked into this binary. It is populated lazily the first time it is
// requested because runtime/debug.ReadBuildInfo is comparatively expensive.
var (
	yaegiVersionOnce  sync.Once
	yaegiVersionCache string
)

// YaegiVersion returns the semantic version of the embedded Yaegi module,
// or "unknown" if the build info is unavailable.
func YaegiVersion() string {
	yaegiVersionOnce.Do(func() {
		info, ok := debug.ReadBuildInfo()
		if !ok {
			yaegiVersionCache = "unknown"
			return
		}
		for _, dep := range info.Deps {
			if dep.Path == "github.com/traefik/yaegi" {
				yaegiVersionCache = dep.Version
				return
			}
		}
		yaegiVersionCache = "unknown"
	})
	return yaegiVersionCache
}

// RunResult captures the outcome of a single Run invocation.
type RunResult struct {
	Stdout     string
	Stderr     string
	ExitCode   int
	DurationMs int64
}

// TestFailure describes a single failed sub-test under RunTests.
type TestFailure struct {
	Name    string
	Message string
}

// RunTestsResult captures the outcome of a RunTests invocation. Passed holds
// names of tests that returned without marking themselves failed; Failed
// holds a name + captured message for each test that called Errorf/Fatalf or
// panicked.
type RunTestsResult struct {
	Stdout     string
	Stderr     string
	Passed     []string
	Failed     []TestFailure
	DurationMs int64
}

// civT is the stand-in used to replace testing.T inside interpreted code. It
// records the first Errorf/Fatalf message and reports Failed() accordingly,
// which is enough for the small didactic snippets Training plays with.
type civT struct {
	name    string
	failed  bool
	message string
}

// Errorf records a failure message and marks the test as failed. It matches
// the (format string, args ...any) shape of testing.T.Errorf so the user can
// write the real signature verbatim.
func (t *civT) Errorf(format string, args ...any) {
	t.failed = true
	if t.message == "" {
		t.message = fmt.Sprintf(format, args...)
	}
}

// Fatalf records a failure, marks the test failed, and then panics with a
// sentinel value so the runner's deferred recover can stop the test early
// without aborting the whole runTests loop.
func (t *civT) Fatalf(format string, args ...any) {
	t.failed = true
	if t.message == "" {
		t.message = fmt.Sprintf(format, args...)
	}
	panic(errFatalSentinel)
}

// Fail marks the test as failed without recording a message.
func (t *civT) Fail() { t.failed = true }

// FailNow marks the test as failed and unwinds via Fatalf's sentinel.
func (t *civT) FailNow() {
	t.failed = true
	panic(errFatalSentinel)
}

// Failed reports whether the test has been marked failed.
func (t *civT) Failed() bool { return t.failed }

// Name returns the test function name.
func (t *civT) Name() string { return t.name }

// Helper is a no-op for compatibility with user code that calls t.Helper().
func (t *civT) Helper() {}

// Log writes args to the captured message buffer joined by spaces.
func (t *civT) Log(args ...any) {
	if t.message != "" {
		t.message += "\n"
	}
	t.message += fmt.Sprint(args...)
}

// Logf writes a formatted message to the captured buffer.
func (t *civT) Logf(format string, args ...any) {
	if t.message != "" {
		t.message += "\n"
	}
	t.message += fmt.Sprintf(format, args...)
}

// errFatalSentinel lets Fatalf unwind a single test without aborting the
// whole RunTests run.
var errFatalSentinel = errors.New("civT.Fatalf")

// newInterpreter builds a yaegi interpreter wired up with stdlib symbols and
// the testing.T override that RunTests relies on. Stdout and Stderr go to
// the supplied buffers.
func newInterpreter(stdout, stderr *bytes.Buffer) (*interp.Interpreter, error) {
	i := interp.New(interp.Options{
		Stdout: stdout,
		Stderr: stderr,
	})
	if err := i.Use(symbols.Symbols); err != nil {
		return nil, fmt.Errorf("load stdlib symbols: %w", err)
	}
	override := map[string]map[string]reflect.Value{
		"testing/testing": {
			"T": reflect.ValueOf((*civT)(nil)),
		},
	}
	if err := i.Use(override); err != nil {
		return nil, fmt.Errorf("override testing.T: %w", err)
	}
	return i, nil
}

// Run evaluates a complete "package main" Go program under Yaegi, capturing
// stdout, stderr and an approximate exit code. ExitCode is 0 on success, 124
// on timeout (matching the coreutils convention), and 1 for any other
// interpreter error. Panics surface in Stderr and also produce a non-zero
// ExitCode.
func Run(source string, timeout time.Duration) (result RunResult) {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	var stdout, stderr bytes.Buffer
	start := time.Now()
	defer func() {
		result.Stdout = stdout.String()
		result.Stderr = stderr.String()
		result.DurationMs = time.Since(start).Milliseconds()
	}()

	i, err := newInterpreter(&stdout, &stderr)
	if err != nil {
		stderr.WriteString(err.Error())
		result.ExitCode = 1
		return result
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err = i.EvalWithContext(ctx, source)
	switch {
	case err == nil:
		result.ExitCode = 0
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		if stderr.Len() > 0 {
			stderr.WriteString("\n")
		}
		stderr.WriteString("timeout: execution exceeded ")
		stderr.WriteString(timeout.String())
		result.ExitCode = 124
	default:
		if stderr.Len() > 0 {
			stderr.WriteString("\n")
		}
		stderr.WriteString(err.Error())
		result.ExitCode = 1
	}
	return result
}

// RunTests parses source to discover func TestXxx(*testing.T) declarations,
// then invokes each one under a civT stand-in. Tests that call Errorf,
// Fatalf, or panic spontaneously land in Failed; the rest land in Passed.
// stdout / stderr capture any fmt output that user code produced while the
// tests ran. DurationMs measures the whole run, not individual tests.
func RunTests(source string, timeout time.Duration) (result RunTestsResult) {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	start := time.Now()
	var stdout, stderr bytes.Buffer
	result = RunTestsResult{
		Passed: []string{},
		Failed: []TestFailure{},
	}
	defer func() {
		result.Stdout = stdout.String()
		result.Stderr = stderr.String()
		result.DurationMs = time.Since(start).Milliseconds()
	}()

	tests, err := findTestFunctions(source)
	if err != nil {
		stderr.WriteString(err.Error())
		return result
	}
	if len(tests) == 0 {
		stderr.WriteString("no test functions found: expected func TestXxx(t *testing.T)")
		return result
	}

	i, err := newInterpreter(&stdout, &stderr)
	if err != nil {
		stderr.WriteString(err.Error())
		return result
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// First load the user program so the test functions become addressable.
	// A user-provided main() is tolerated but unnecessary; either way we
	// invoke tests explicitly afterwards via i.Eval.
	if _, err := i.EvalWithContext(ctx, source); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			stderr.WriteString("\ntimeout while loading source: ")
			stderr.WriteString(timeout.String())
			return result
		}
		if stderr.Len() > 0 {
			stderr.WriteString("\n")
		}
		stderr.WriteString(err.Error())
		return result
	}

	// Invoke each discovered test inside its own recover wrapper by asking
	// Yaegi to evaluate a tiny call site. The per-call timeout shares the
	// parent deadline.
	for _, name := range tests {
		if ctx.Err() != nil {
			break
		}
		passed, message := invokeTest(i, ctx, name)
		if passed {
			result.Passed = append(result.Passed, name)
		} else {
			result.Failed = append(result.Failed, TestFailure{Name: name, Message: message})
		}
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		if stderr.Len() > 0 {
			stderr.WriteString("\n")
		}
		stderr.WriteString("timeout: test run exceeded ")
		stderr.WriteString(timeout.String())
	}
	return result
}

// invokeTest runs a single TestXxx function and returns (passed, message).
// The call site runs on a fresh goroutine so that a misbehaving test which
// spins forever cannot block the whole runTests invocation. When ctx fires
// first the goroutine is abandoned — yaegi's evaluator cannot be preempted
// from outside, so the leaked goroutine runs until the host process exits.
// For one-shot browser/Node.js worker usage that is acceptable; the worker
// is typically disposed after a single run.
func invokeTest(i *interp.Interpreter, ctx context.Context, name string) (bool, string) {
	globals := i.Symbols("main")
	mainSyms, ok := globals["main"]
	if !ok {
		return false, "interpreter has no main package symbols"
	}
	fnVal, ok := mainSyms[name]
	if !ok {
		return false, "test function " + name + " not found in compiled symbols"
	}
	if !fnVal.IsValid() || fnVal.Kind() != reflect.Func {
		return false, "symbol " + name + " is not a function"
	}
	t := &civT{name: name}
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				if r == errFatalSentinel {
					return
				}
				t.failed = true
				if t.message == "" {
					t.message = fmt.Sprintf("panic: %v", r)
				} else {
					t.message = fmt.Sprintf("panic: %v\n%s", r, t.message)
				}
			}
		}()
		fnVal.Call([]reflect.Value{reflect.ValueOf(t)})
	}()
	select {
	case <-done:
	case <-ctx.Done():
		return false, fmt.Sprintf("timeout: %s", ctx.Err())
	}
	if t.failed {
		return false, t.message
	}
	return true, ""
}

// findTestFunctions parses source and returns the names of every top-level
// function whose signature looks like func TestXxx(*testing.T). Any parse
// error is returned verbatim so the caller can surface it on stderr.
func findTestFunctions(source string) ([]string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", source, parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("parse source: %w", err)
	}
	var names []string
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil {
			continue
		}
		if !isTestName(fn.Name.Name) {
			continue
		}
		if !hasTestingTParam(fn.Type) {
			continue
		}
		names = append(names, fn.Name.Name)
	}
	return names, nil
}

// isTestName reports whether name matches the TestXxx convention: starts with
// "Test" and is followed by a non-lowercase rune (or nothing, for bare
// "Test").
func isTestName(name string) bool {
	const prefix = "Test"
	if len(name) < len(prefix) || name[:len(prefix)] != prefix {
		return false
	}
	if len(name) == len(prefix) {
		return true
	}
	c := name[len(prefix)]
	// next rune must not be a lowercase ASCII letter — the stdlib rule.
	return c < 'a' || c > 'z'
}

// hasTestingTParam checks the function type has a single parameter of type
// *testing.T.
func hasTestingTParam(ft *ast.FuncType) bool {
	if ft.Params == nil || len(ft.Params.List) != 1 {
		return false
	}
	p := ft.Params.List[0]
	if len(p.Names) > 1 {
		return false
	}
	star, ok := p.Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return pkg.Name == "testing" && sel.Sel.Name == "T"
}

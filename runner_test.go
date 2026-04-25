package yaegibrowser

import (
	"strings"
	"testing"
	"time"
)

func TestRun_HelloWorld(t *testing.T) {
	src := `package main
import "fmt"
func main() { fmt.Println("hello world") }
`
	got := Run(src, "", 2*time.Second)
	if got.ExitCode != 0 {
		t.Fatalf("exit code: want 0, got %d, stderr=%q", got.ExitCode, got.Stderr)
	}
	if !strings.Contains(got.Stdout, "hello world") {
		t.Fatalf("stdout: want contains %q, got %q", "hello world", got.Stdout)
	}
	if got.DurationMs < 0 {
		t.Fatalf("duration: want >= 0, got %d", got.DurationMs)
	}
}

func TestRun_Panic(t *testing.T) {
	src := `package main
func main() { panic("boom") }
`
	got := Run(src, "", 2*time.Second)
	if got.ExitCode == 0 {
		t.Fatalf("exit code: want non-zero, got 0")
	}
	if !strings.Contains(got.Stderr, "boom") {
		t.Fatalf("stderr: want contains boom, got %q", got.Stderr)
	}
}

func TestRun_Timeout(t *testing.T) {
	src := `package main
func main() { for {} }
`
	got := Run(src, "", 150*time.Millisecond)
	if got.ExitCode != 124 {
		t.Fatalf("exit code: want 124, got %d", got.ExitCode)
	}
	if !strings.Contains(got.Stderr, "timeout") {
		t.Fatalf("stderr: want contains timeout, got %q", got.Stderr)
	}
	if got.DurationMs < 100 {
		t.Fatalf("duration: want >= 100ms, got %d", got.DurationMs)
	}
}

func TestRun_DefaultTimeout(t *testing.T) {
	src := `package main
import "fmt"
func main() { fmt.Print("ok") }
`
	got := Run(src, "", 0)
	if got.ExitCode != 0 {
		t.Fatalf("default timeout should still run quick programs: exit=%d stderr=%q", got.ExitCode, got.Stderr)
	}
	if got.Stdout != "ok" {
		t.Fatalf("stdout: want ok, got %q", got.Stdout)
	}
}

func TestRun_CompileError(t *testing.T) {
	src := `package main
func main() { undeclared() }
`
	got := Run(src, "", time.Second)
	if got.ExitCode == 0 {
		t.Fatalf("compile error should yield non-zero exit, got 0")
	}
	if got.Stderr == "" {
		t.Fatalf("stderr: want non-empty, got empty")
	}
}

func TestRun_BufioScannerStdin(t *testing.T) {
	src := `package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	var count int
	var sum float64
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		n, err := strconv.ParseFloat(line, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skipping %q: %v\n", line, err)
			continue
		}
		count++
		sum += n
	}
	fmt.Printf("count=%d\nsum=%.2f\n", count, sum)
}
`
	stdin := "1.5\n2.5\nbad\n3\n"
	got := Run(src, stdin, 2*time.Second)
	if got.ExitCode != 0 {
		t.Fatalf("exit code: want 0, got %d, stderr=%q", got.ExitCode, got.Stderr)
	}
	if !strings.Contains(got.Stdout, "count=3") {
		t.Fatalf("stdout: want contains count=3, got %q", got.Stdout)
	}
	if !strings.Contains(got.Stdout, "sum=7.00") {
		t.Fatalf("stdout: want contains sum=7.00, got %q", got.Stdout)
	}
	if !strings.Contains(got.Stderr, `skipping "bad"`) {
		t.Fatalf("stderr: want skip notice for bad input, got %q", got.Stderr)
	}
}

func TestRun_FmtScanStdin(t *testing.T) {
	src := `package main

import "fmt"

func main() {
	var name string
	var age int
	fmt.Scan(&name, &age)
	fmt.Printf("hello %s, age %d\n", name, age)
}
`
	got := Run(src, "ada 36\n", time.Second)
	if got.ExitCode != 0 {
		t.Fatalf("exit code: want 0, got %d, stderr=%q", got.ExitCode, got.Stderr)
	}
	if !strings.Contains(got.Stdout, "hello ada, age 36") {
		t.Fatalf("stdout: want hello line, got %q", got.Stdout)
	}
}

func TestRun_OsExitDeniedAtModule(t *testing.T) {
	src := `package main

import "os"

func main() {
	os.Exit(7)
}
`
	got := Run(src, "", time.Second)
	if got.ExitCode == 0 {
		t.Fatalf("os.Exit should produce non-zero exit, got 0")
	}
	if !strings.Contains(got.Stderr, "os.Exit(7)") {
		t.Fatalf("stderr: want os.Exit(7) trace, got %q", got.Stderr)
	}
}

func TestRun_OsOpenIsTrimmed(t *testing.T) {
	src := `package main

import "os"

func main() {
	_, _ = os.Open("anything")
}
`
	got := Run(src, "", time.Second)
	if got.ExitCode == 0 {
		t.Fatalf("os.Open should not be available; want non-zero exit, got 0")
	}
	if !strings.Contains(got.Stderr, "Open") {
		t.Fatalf("stderr: want mention of Open, got %q", got.Stderr)
	}
}

func TestRunTests_MixedPassFail(t *testing.T) {
	src := `package main
import "testing"

func TestPass(t *testing.T) {
	if 1+1 != 2 { t.Errorf("math broke") }
}

func TestFail(t *testing.T) {
	if 1+1 != 3 { t.Errorf("want 3, got %d", 1+1) }
}
`
	got := RunTests(src, "", 2*time.Second)
	if len(got.Passed) != 1 || got.Passed[0] != "TestPass" {
		t.Fatalf("passed: want [TestPass], got %v", got.Passed)
	}
	if len(got.Failed) != 1 || got.Failed[0].Name != "TestFail" {
		t.Fatalf("failed: want [TestFail], got %v", got.Failed)
	}
	if !strings.Contains(got.Failed[0].Message, "want 3") {
		t.Fatalf("failed message: want contains 'want 3', got %q", got.Failed[0].Message)
	}
}

func TestRunTests_Fatalf(t *testing.T) {
	src := `package main
import "testing"
func TestFatalStops(t *testing.T) {
	t.Fatalf("bail at %d", 42)
	t.Errorf("should not run")
}
`
	got := RunTests(src, "", time.Second)
	if len(got.Failed) != 1 {
		t.Fatalf("failed: want 1, got %d", len(got.Failed))
	}
	if !strings.Contains(got.Failed[0].Message, "bail at 42") {
		t.Fatalf("failed message: want contains 'bail at 42', got %q", got.Failed[0].Message)
	}
}

func TestRunTests_PanicCaught(t *testing.T) {
	src := `package main
import "testing"
func TestPanics(t *testing.T) {
	panic("deliberate")
}
`
	got := RunTests(src, "", time.Second)
	if len(got.Failed) != 1 {
		t.Fatalf("failed: want 1, got %d (passed=%v)", len(got.Failed), got.Passed)
	}
	if !strings.Contains(got.Failed[0].Message, "panic") {
		t.Fatalf("failed message: want contains 'panic', got %q", got.Failed[0].Message)
	}
	if !strings.Contains(got.Failed[0].Message, "deliberate") {
		t.Fatalf("failed message: want contains 'deliberate', got %q", got.Failed[0].Message)
	}
}

func TestRunTests_NoneFound(t *testing.T) {
	src := `package main
func main() {}
`
	got := RunTests(src, "", time.Second)
	if len(got.Passed)+len(got.Failed) != 0 {
		t.Fatalf("want no tests, got passed=%v failed=%v", got.Passed, got.Failed)
	}
	if !strings.Contains(got.Stderr, "no test functions") {
		t.Fatalf("stderr: want hint, got %q", got.Stderr)
	}
}

func TestRunTests_ParseError(t *testing.T) {
	src := `package main
func TestBroken(t *testing.T { }
`
	got := RunTests(src, "", time.Second)
	if got.Stderr == "" {
		t.Fatalf("stderr: want parse error, got empty")
	}
}

func TestRunTests_Timeout(t *testing.T) {
	src := `package main
import "testing"
func TestSpin(t *testing.T) { for {} }
`
	got := RunTests(src, "", 120*time.Millisecond)
	if len(got.Failed) == 0 && !strings.Contains(got.Stderr, "timeout") {
		t.Fatalf("want timeout signal, got failed=%v stderr=%q", got.Failed, got.Stderr)
	}
}

func TestFindTestFunctions(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "two tests + helpers",
			src: `package main
import "testing"
func helper() {}
func TestOne(t *testing.T) {}
func TestTwo(t *testing.T) {}
func testskipLowercase(t *testing.T) {}
func TestxSkipLowerNext(t *testing.T) {}
`,
			want: []string{"TestOne", "TestTwo"},
		},
		{
			name: "bare Test accepted",
			src: `package main
import "testing"
func Test(t *testing.T) {}
`,
			want: []string{"Test"},
		},
		{
			name: "method not counted",
			src: `package main
import "testing"
type S struct{}
func (s S) TestMethod(t *testing.T) {}
`,
			want: nil,
		},
		{
			name: "wrong param type rejected",
			src: `package main
func TestNotReal(i int) {}
`,
			want: nil,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := findTestFunctions(tc.src)
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}
			if !equalStrings(got, tc.want) {
				t.Fatalf("want %v, got %v", tc.want, got)
			}
		})
	}
}

func TestIsTestName(t *testing.T) {
	cases := map[string]bool{
		"Test":       true,
		"TestA":      true,
		"TestPlain":  true,
		"Test1":      true,
		"Test_Thing": true,
		"Testa":      false,
		"Tests":      false,
		"Example":    false,
		"":           false,
		"Tes":        false,
		"TestxLower": false,
	}
	for name, want := range cases {
		if got := isTestName(name); got != want {
			t.Errorf("isTestName(%q): want %v, got %v", name, want, got)
		}
	}
}

func TestYaegiVersion(t *testing.T) {
	v := YaegiVersion()
	if v == "" {
		t.Fatalf("version: want non-empty string")
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Stub helpers referenced by the generated os.go.
//
// `yaegi extract os` rewrites `os.Exit` and `os.FindProcess` to local
// helper names — upstream yaegi/stdlib provides the same wrappers in
// restricted.go to keep interpreted code from terminating the host
// process or finding the host's own pid. We mirror that contract so the
// extracted file compiles against this package.
//
// `osFindProcess` is referenced even though os_curated.go strips
// FindProcess from the symbol map after init(); it has to exist for
// os.go to type-check.
package symbols

import (
	"errors"
	"io"
	"log"
	"os"
	"strconv"
)

var errRestricted = errors.New("restricted")

// osExit panics with a recognisable message instead of terminating the
// interpreter's host process. The Run wrapper recovers from this panic
// and reports a non-zero exit code in RunResult.
func osExit(code int) { panic("os.Exit(" + strconv.Itoa(code) + ")") }

// osFindProcess refuses to look up the host's own pid, otherwise
// delegates to the real os.FindProcess. The function exists only to
// satisfy the generated reference; FindProcess is removed from the
// public symbol table by os_curated.go.
func osFindProcess(pid int) (*os.Process, error) {
	if pid == os.Getpid() {
		return nil, errRestricted
	}
	return os.FindProcess(pid)
}

// log.Fatal* terminates the host process via os.Exit. Yaegi's extract
// rewrites the references to logFatal/Fatalf/Fatalln; we mirror the
// upstream restricted.go wrappers and route them through Panic so the
// runner can recover.
func logFatal(v ...interface{})            { log.Panic(v...) }
func logFatalf(f string, v ...interface{}) { log.Panicf(f, v...) }
func logFatalln(v ...interface{})          { log.Panicln(v...) }

// logLogger is a thin wrapper around *log.Logger that swaps Fatal*
// methods for Panic*. The interpreted code sees this type instead of
// log.Logger directly, so any *log.Logger value flowing through the
// sandbox is wrapped here.
type logLogger struct {
	l *log.Logger
}

func logNew(out io.Writer, prefix string, flag int) *logLogger {
	return &logLogger{log.New(out, prefix, flag)}
}

func (l *logLogger) Fatal(v ...interface{})            { l.l.Panic(v...) }
func (l *logLogger) Fatalf(f string, v ...interface{}) { l.l.Panicf(f, v...) }
func (l *logLogger) Fatalln(v ...interface{})          { l.l.Panicln(v...) }
func (l *logLogger) Flags() int                        { return l.l.Flags() }
func (l *logLogger) Output(d int, s string) error      { return l.l.Output(d, s) }
func (l *logLogger) Panic(v ...interface{})            { l.l.Panic(v...) }
func (l *logLogger) Panicf(f string, v ...interface{}) { l.l.Panicf(f, v...) }
func (l *logLogger) Panicln(v ...interface{})          { l.l.Panicln(v...) }
func (l *logLogger) Prefix() string                    { return l.l.Prefix() }
func (l *logLogger) Print(v ...interface{})            { l.l.Print(v...) }
func (l *logLogger) Printf(f string, v ...interface{}) { l.l.Printf(f, v...) }
func (l *logLogger) Println(v ...interface{})          { l.l.Println(v...) }
func (l *logLogger) SetFlags(flag int)                 { l.l.SetFlags(flag) }
func (l *logLogger) SetOutput(w io.Writer)             { l.l.SetOutput(w) }
func (l *logLogger) Writer() io.Writer                 { return l.l.Writer() }

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

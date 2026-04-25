// Curated trim of the os package symbols.
//
// `yaegi extract os` registers every exported os identifier into
// Symbols["os/os"]. Most of them dial into the host kernel through
// syscalls that simply do not exist on js/wasm, and the rest are
// process-control entry points that have no meaning inside a single
// Web Worker. Either way, exposing them only invites a reader to write
// code that compiles, runs, and silently does nothing useful.
//
// This file deletes those keys after the generated init() in os.go has
// populated the map. Init order in Go is by filename within a package,
// so os_curated.go reliably runs after os.go.
//
// What survives the trim:
//   - Stdin / Stdout / Stderr — yaegi's fixStdlib rebinds these to the
//     interp.Options streams when YAEGI_SPECIAL_STDIO=true.
//   - Args                    — rebound by yaegi to the interpreter args.
//   - Exit                    — terminates the program with a status.
//   - Env access              — Getenv/LookupEnv/Setenv/Environ/...,
//     all routed through the interpreter's sandboxed env map.
//   - Pid / uid / gid getters — return host values, harmless to expose.
//   - Errors and mode constants — pure values used in error inspection.
//   - The struct/interface types (File, FileInfo, FileMode, DirEntry,
//     PathError, LinkError, SyscallError, Signal) — reachable through
//     other packages' return types (e.g. fs.FS), so dropping them would
//     break legitimate code that never calls os filesystem entry points.
//
// Adding a new os symbol back later: delete its entry from
// disallowedOsKeys below, run `make wasm`, and verify the size budget.
package symbols

func init() {
	pkg := Symbols["os/os"]
	if pkg == nil {
		return
	}
	for _, k := range disallowedOsKeys {
		delete(pkg, k)
	}
}

var disallowedOsKeys = []string{
	"Chdir",
	"Chmod",
	"Chown",
	"Chtimes",
	"CopyFS",
	"Create",
	"CreateTemp",
	"DirFS",
	"Executable",
	"FindProcess",
	"Getwd",
	"Lchown",
	"Link",
	"Lstat",
	"Mkdir",
	"MkdirAll",
	"MkdirTemp",
	"NewFile",
	"NewSyscallError",
	"Open",
	"OpenFile",
	"OpenInRoot",
	"OpenRoot",
	"Pipe",
	"ProcAttr",
	"Process",
	"ProcessState",
	"ReadDir",
	"ReadFile",
	"Readlink",
	"Remove",
	"RemoveAll",
	"Rename",
	"Root",
	"SameFile",
	"StartProcess",
	"Stat",
	"Symlink",
	"TempDir",
	"Truncate",
	"UserCacheDir",
	"UserConfigDir",
	"UserHomeDir",
	"WriteFile",

	// Process-control vars (signals). The Signal interface type stays
	// because it is referenced by the os.Signal struct field on
	// Process — keep the type, drop the values.
	"Interrupt",
	"Kill",
}

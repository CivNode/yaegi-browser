// Package symbols is a minimal Yaegi stdlib wrapper covering only the
// packages that Training snippets actually need: fmt, strings, strconv,
// errors, bytes, sort, math, unicode, unicode/utf8, io, time, bufio,
// and a curated subset of os.
//
// It exists because github.com/traefik/yaegi/stdlib registers every
// standard library package via init() side effects — about 40 MB of
// symbol data once compiled to WebAssembly. This curated subset keeps
// the wasm binary comfortably under the size budget while still
// supporting the didactic examples CivNode runs in the browser.
//
// Files in this directory are produced with `yaegi extract` and should
// be regenerated with `go generate ./internal/symbols` if the target
// Go version changes. Do not edit them by hand. The os symbol table is
// trimmed in os_curated.go before init() runs — file-system entry points
// like Open/Create/Remove are deleted because they cannot work inside a
// browser sandbox and would only mislead a reader.
package symbols

import "reflect"

//go:generate yaegi extract -name symbols fmt strings strconv errors bytes sort math unicode unicode/utf8 io time bufio os

// Symbols is the Yaegi-style symbol map that each generated file
// registers into from its init() function.
var Symbols = map[string]map[string]reflect.Value{}

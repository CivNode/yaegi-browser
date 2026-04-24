// Package symbols is a minimal Yaegi stdlib wrapper covering only the
// packages that Training snippets actually need: fmt, strings, strconv,
// errors, bytes, sort, math, unicode, unicode/utf8, io, and time.
//
// It exists because github.com/traefik/yaegi/stdlib registers every
// standard library package via init() side effects — about 40 MB of
// symbol data once compiled to WebAssembly. This curated subset keeps
// the wasm binary comfortably under 8 MB while still supporting the
// didactic examples CivNode runs in the browser.
//
// Files in this directory are produced with `yaegi extract` and should
// be regenerated with `go generate ./internal/symbols` if the target Go
// version changes. Do not edit them by hand.
package symbols

import "reflect"

//go:generate yaegi extract -name symbols fmt strings strconv errors bytes sort math unicode unicode/utf8 io time

// Symbols is the Yaegi-style symbol map that each generated file
// registers into from its init() function.
var Symbols = map[string]map[string]reflect.Value{}

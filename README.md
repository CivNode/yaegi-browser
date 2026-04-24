# yaegi-browser

Yaegi Go interpreter compiled to WebAssembly for in-browser Go execution. Part of the CivNode Training platform.

## Status

v0.1.0. Ships three JavaScript entry points bound to the Worker global
`civnodeYaegi`:

```js
await civnodeYaegi.run(source, timeoutMs);
await civnodeYaegi.runTests(source, timeoutMs);
civnodeYaegi.version();
```

`run` evaluates a `package main` program and resolves to
`{ stdout, stderr, exitCode, durationMs }`. `runTests` discovers every
`func TestXxx(t *testing.T)` in the source, invokes each under a
lightweight `testing.T` stand-in, and resolves to
`{ stdout, stderr, passed: [name], failed: [{name, message}], durationMs }`.
`version` reports the linked Yaegi and Go versions plus the build
timestamp.

## Install

The Go package is importable on any host (`go get
github.com/CivNode/yaegi-browser`). For browser use, build the wasm
bundle with:

```
make wasm
```

This produces three files in `dist/`:

- `yaegi.wasm`       — the interpreter bundle
- `wasm_exec.js`     — Go's runtime glue copied from `GOROOT/lib/wasm`
- `yaegi.manifest.json` — version + sha256 + size metadata

Serve `yaegi.wasm` with `Content-Type: application/wasm` and
`Content-Encoding: gzip` for best wire size.

## Integration sketch

```js
// inside a Web Worker
importScripts("/vendor/yaegi-browser/wasm_exec.js");
const go = new Go();
const { instance } = await WebAssembly.instantiateStreaming(
  fetch("/vendor/yaegi-browser/yaegi.wasm"),
  go.importObject,
);
go.run(instance);

const out = await self.civnodeYaegi.run(
  `package main\nimport "fmt"\nfunc main(){ fmt.Println("hello") }`,
  5000,
);
self.postMessage(out);
```

## Size budget

`dist/yaegi.wasm` is approximately **11.9 MB** raw, **2.9 MB** gzipped
in a release build. The Yaegi interpreter contributes about 10.5 MB
(reflect, go/ast, go/parser, go/token, the CFG and run loop); the
curated standard library wrappers in `internal/symbols` add roughly
1 MB and cover `fmt`, `strings`, `strconv`, `errors`, `bytes`, `sort`,
`math`, `unicode`, `unicode/utf8`, `io`, and `time`.

The CI ceiling is 12 MB raw, enforced by `make sizecheck`. Raise it
only after adding a symbol package that grows the curated subset.

## Timeout enforcement

`context.WithTimeout` works reliably for any user code that contains a
blocking call (channel receive, `time.Sleep`, `fmt.Println`, syscall,
etc.). On `js/wasm` the Go scheduler cannot preempt a tight
`for { x++ }` loop with no yield points; that is a Go runtime
limitation, not a Yaegi one.

For full safety, the recommended pattern is to run the interpreter
inside a `Worker` and `terminate()` the Worker after the deadline
expires from the host page. The `timeoutMs` argument is then a fast
path for well-behaved programs; the `Worker.terminate` is the
cut-off for everything else.

## Development

```
make test      # go test ./... -race -count=1
make lint      # gofumpt + golangci-lint v2
make wasm      # build dist/yaegi.wasm and dist/wasm_exec.js
make sizecheck # fail if dist/yaegi.wasm > 12 MB
make release   # clean + wasm + sizecheck
node testharness/node.mjs   # end-to-end smoke test
```

CI runs `test`, `lint`, and the `wasm` job (which calls `make wasm`,
`make sizecheck`, and `node testharness/node.mjs`).

## Upstream

`yaegi-browser` wraps [github.com/traefik/yaegi](https://github.com/traefik/yaegi)
(MIT licence). The curated `internal/symbols` package is generated with
`yaegi extract` and regenerated via `go generate ./internal/symbols` when
the target Go version changes.

## Licence

Apache-2.0 for this package. See [LICENSE](./LICENSE). Bundled Yaegi
symbols are redistributed under their original MIT licence.

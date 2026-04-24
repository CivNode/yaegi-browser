.PHONY: test lint fuzz bench fmt tidy wasm sizecheck release clean

# Host-side tests and lint — the same pipeline CI runs.
test:
	go test ./... -race -count=1
lint:
	gofumpt -l -d .
	golangci-lint run ./...
fuzz:
	@echo "run individual fuzz targets, e.g. go test -fuzz=FuzzX ./..."
bench:
	go test -bench=. -benchmem ./...
fmt:
	gofumpt -w .
tidy:
	go mod tidy

# WASM build — produces dist/yaegi.wasm, copies wasm_exec.js out of GOROOT
# so consumers never have to reach into it, and writes a release manifest.
#
# Size budget: 12 MB raw / 4 MB gzipped. The Yaegi interpreter itself is
# about 10.5 MB compiled for js/wasm before any symbols are registered;
# internal/symbols adds roughly 1 MB of curated fmt, strings, strconv,
# errors, bytes, sort, math, unicode, io, and time wrappers. Serving the
# asset with Content-Encoding: gzip or brotli drops the wire size to
# about 2.9 MB.
#
# MAX_WASM_BYTES below is the CI ceiling. Raise it here (and in CI) if
# you extend the curated symbol set.
VERSION        ?= dev
BUILD_AT       := $(shell date -u +%FT%TZ)
MAX_WASM_BYTES := 12582912

# Track every Go source file that influences the wasm build so `make wasm`
# actually rebuilds when they change.
WASM_SOURCES := $(shell find . -type f -name '*.go' -not -path './dist/*' -not -path './testharness/*')

wasm: dist/yaegi.wasm dist/wasm_exec.js dist/yaegi.manifest.json

dist:
	mkdir -p dist

dist/yaegi.wasm: $(WASM_SOURCES) go.mod go.sum | dist
	GOOS=js GOARCH=wasm go build \
		-trimpath \
		-ldflags "-s -w -X main.builtAt=$(BUILD_AT)" \
		-o dist/yaegi.wasm \
		./cmd/wasm

dist/wasm_exec.js: | dist
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" dist/wasm_exec.js

dist/yaegi.manifest.json: dist/yaegi.wasm | dist
	@sha=$$(sha256sum dist/yaegi.wasm | awk '{print $$1}'); \
	size=$$(wc -c < dist/yaegi.wasm); \
	goV=$$(go version | awk '{print $$3}'); \
	yaegiV=$$(go list -m -f '{{.Version}}' github.com/traefik/yaegi); \
	printf '{\n  "version": "%s",\n  "builtAt": "%s",\n  "goVersion": "%s",\n  "yaegiVersion": "%s",\n  "sha256": "%s",\n  "sizeBytes": %s\n}\n' \
		"$(VERSION)" "$(BUILD_AT)" "$$goV" "$$yaegiV" "$$sha" "$$size" > dist/yaegi.manifest.json

# sizecheck fails if dist/yaegi.wasm exceeds MAX_WASM_BYTES. CI invokes
# this after `make wasm`.
sizecheck: dist/yaegi.wasm
	@size=$$(wc -c < dist/yaegi.wasm); \
	if [ "$$size" -gt $(MAX_WASM_BYTES) ]; then \
		echo "FAIL: dist/yaegi.wasm is $$size bytes, limit is $(MAX_WASM_BYTES)"; \
		exit 1; \
	fi; \
	echo "OK: dist/yaegi.wasm is $$size bytes (limit $(MAX_WASM_BYTES))"

# release: build + write release artifacts to dist/ ready for gh release upload.
release: clean wasm sizecheck
	@echo "release artifacts:"
	@ls -la dist/

clean:
	rm -rf dist/

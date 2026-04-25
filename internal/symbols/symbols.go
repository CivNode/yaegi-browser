// Package symbols is a Yaegi stdlib wrapper covering the standard
// library surface that makes sense in a browser sandbox: every pure
// computation package (fmt, strings, strconv, bytes, sort, slices,
// maps, cmp, errors, regexp, math/*, encoding/*, hash/*, crypto/*
// hashing+ciphers, container/*, archive/*, compress/*, image/*,
// text/*, html/*, mime/*, net/url for URL parsing, log + log/slog,
// runtime, reflect, sync, context, time, io, io/fs, bufio, path,
// path/filepath, unicode/*) plus a curated subset of os whose
// filesystem-mutating entry points are stripped because they cannot
// work in a Web Worker.
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

//go:generate yaegi extract -name symbols fmt strings strconv errors bytes sort math unicode unicode/utf8 unicode/utf16 io io/fs time bufio path path/filepath context sync sync/atomic reflect runtime runtime/debug log log/slog flag regexp regexp/syntax slices maps cmp container/list container/heap container/ring math/big math/rand math/cmplx math/bits hash hash/adler32 hash/crc32 hash/crc64 hash/fnv hash/maphash crypto crypto/md5 crypto/sha1 crypto/sha256 crypto/sha512 crypto/rand crypto/hmac crypto/aes crypto/cipher crypto/subtle crypto/des crypto/rc4 encoding encoding/json encoding/base64 encoding/base32 encoding/hex encoding/csv encoding/binary encoding/ascii85 encoding/pem encoding/asn1 encoding/xml encoding/gob text/template text/template/parse text/tabwriter text/scanner html html/template archive/tar archive/zip compress/gzip compress/zlib compress/flate compress/bzip2 compress/lzw mime mime/multipart mime/quotedprintable net/url os

// Symbols is the Yaegi-style symbol map that each generated file
// registers into from its init() function.
var Symbols = map[string]map[string]reflect.Value{}

#!/usr/bin/env node
// Node 22 harness that loads dist/wasm_exec.js + dist/yaegi.wasm, calls
// civnodeYaegi.run / runTests / version, and asserts the outputs match
// expectations. Exit 0 on success, 1 on any assertion failure. This is the
// gate CI runs after `make wasm`.

import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import vm from "node:vm";

const here = dirname(fileURLToPath(import.meta.url));
const root = resolve(here, "..");
const wasmExecPath = resolve(root, "dist/wasm_exec.js");
const wasmPath = resolve(root, "dist/yaegi.wasm");
const fixturesDir = resolve(here, "fixtures");

let failures = 0;

function fail(label, detail) {
  failures++;
  console.error(`FAIL ${label}\n    ${detail}`);
}

function pass(label) {
  console.log(`PASS ${label}`);
}

function assertContains(label, haystack, needle) {
  if (typeof haystack !== "string" || !haystack.includes(needle)) {
    fail(label, `expected to contain ${JSON.stringify(needle)}, got ${JSON.stringify(haystack)}`);
    return false;
  }
  pass(label);
  return true;
}

function assertEqual(label, got, want) {
  if (got !== want) {
    fail(label, `expected ${JSON.stringify(want)}, got ${JSON.stringify(got)}`);
    return false;
  }
  pass(label);
  return true;
}

// wasm_exec.js installs globalThis.Go; we load it in the current context so
// that Go's js.Global() and the civnodeYaegi object land on globalThis.
const wasmExecSource = readFileSync(wasmExecPath, "utf8");
vm.runInThisContext(wasmExecSource, { filename: "wasm_exec.js" });

if (typeof globalThis.Go !== "function") {
  console.error("FAIL wasm_exec.js did not install globalThis.Go");
  process.exit(1);
}

const wasmBytes = readFileSync(wasmPath);

async function startInterpreter() {
  // A fresh Go + Instance per invocation so one test cannot leave state
  // lying around for the next.
  const go = new globalThis.Go();
  const { instance } = await WebAssembly.instantiate(wasmBytes, go.importObject);
  // Go's main() blocks on select{} so the promise below never resolves; we
  // just need the exports wired up. Kick it off without awaiting.
  go.run(instance);
  // The Go runtime sets civnodeYaegiReady once the FuncOfs are installed.
  // In practice this happens synchronously on the first microtask after
  // go.run; await a microtask before calling in.
  await new Promise((r) => setImmediate(r));
  if (typeof globalThis.civnodeYaegi !== "object" || globalThis.civnodeYaegi === null) {
    throw new Error("civnodeYaegi global not set after go.run");
  }
  return { go, api: globalThis.civnodeYaegi };
}

async function runFixture(name) {
  const path = resolve(fixturesDir, name);
  return readFileSync(path, "utf8");
}

async function main() {
  const { api } = await startInterpreter();

  // Test 1: hello world
  {
    const src = await runFixture("hello.txt");
    const got = await api.run(src, 5000);
    if (assertEqual("hello.exitCode", got.exitCode, 0)) {
      assertContains("hello.stdout", got.stdout, "hello from yaegi-browser");
    }
    if (typeof got.durationMs !== "number") {
      fail("hello.durationMs", `expected number, got ${typeof got.durationMs}`);
    } else {
      pass("hello.durationMs");
    }
  }

  // Test 2: timeout path
  {
    const src = await runFixture("spin.txt");
    const got = await api.run(src, 250);
    assertEqual("spin.exitCode", got.exitCode, 124);
    assertContains("spin.stderr", got.stderr, "timeout");
  }

  // Test 3: panic path
  {
    const src = await runFixture("panic.txt");
    const got = await api.run(src, 2000);
    if (got.exitCode === 0) {
      fail("panic.exitCode", "expected non-zero, got 0");
    } else {
      pass("panic.exitCode");
    }
    assertContains("panic.stderr", got.stderr, "boom");
  }

  // Test 4: runTests mixed pass / fail
  {
    const src = await runFixture("tests_mixed.txt");
    const got = await api.runTests(src, 5000);
    const passed = Array.isArray(got.passed) ? got.passed : [];
    const failed = Array.isArray(got.failed) ? got.failed : [];
    if (!passed.includes("TestAddPositive")) {
      fail("tests_mixed.passed", `expected TestAddPositive in passed, got ${JSON.stringify(passed)}`);
    } else {
      pass("tests_mixed.passed");
    }
    const failNames = failed.map((f) => f && f.name);
    if (!failNames.includes("TestAddBroken")) {
      fail("tests_mixed.failed", `expected TestAddBroken in failed, got ${JSON.stringify(failNames)}`);
    } else {
      pass("tests_mixed.failed");
    }
    const brokenEntry = failed.find((f) => f && f.name === "TestAddBroken");
    if (brokenEntry) {
      assertContains("tests_mixed.message", brokenEntry.message || "", "expected 3");
    }
  }

  // Test 5: version metadata
  {
    const got = api.version();
    assertContains("version.yaegi", got.yaegiVersion || "", "v");
    assertContains("version.go", got.goVersion || "", "go1");
    if (typeof got.builtAt !== "string" || got.builtAt.length === 0) {
      fail("version.builtAt", `expected non-empty string, got ${JSON.stringify(got.builtAt)}`);
    } else {
      pass("version.builtAt");
    }
  }

  // Test 6: bufio.NewScanner reading os.Stdin from the JS-supplied stdin.
  {
    const src = await runFixture("scanner.txt");
    const got = await api.run(src, "1.5\n2.5\nbad\n3\n", 5000);
    if (assertEqual("scanner.exitCode", got.exitCode, 0)) {
      assertContains("scanner.count", got.stdout, "count=3");
      assertContains("scanner.sum", got.stdout, "sum=7.00");
      assertContains("scanner.skip", got.stderr, "skipping \"bad\"");
    }
  }

  // Test 7: legacy (source, timeoutMs) still works (no stdin slot).
  {
    const src = await runFixture("hello.txt");
    const got = await api.run(src, 5000);
    assertEqual("legacy.exitCode", got.exitCode, 0);
  }

  if (failures > 0) {
    console.error(`\n${failures} check(s) failed`);
    process.exit(1);
  }
  console.log("\nall checks passed");
  process.exit(0);
}

main().catch((err) => {
  console.error("harness error:", err);
  process.exit(1);
});

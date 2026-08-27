#!/usr/bin/env bash
# Drives the real startup gate, not a helper, with a forged Metal shader library.
#
# `MetalShaderLibraryCheck` runs in `Main.main()` before the listener binds. A
# reviewer showed that a *directory* named `default.metallib` satisfied the old
# existence-only check: the gate admitted, the port bound, and the managed
# launcher started polling a runtime that could never reach the GPU. That is the
# forged-evidence shape, so it is probed here at the composed entry point.
#
# Requires a `swift build` product: it is the build that genuinely lacks the
# shader library, which is what makes the gate observable at all.

set -uo pipefail

BINARY="${BINARY:?set BINARY to a swift-build (non-Xcode) prototype executable}"
PORT="${PORT:-28117}"
HOST=127.0.0.1
OUT="${OUT:-./metallib-gate-out}"

BUNDLE_REL="mlx-swift_Cmlx.bundle/Contents/Resources"
LIB_NAME="default.metallib"

rm -rf "$OUT"
mkdir -p "$OUT"
OUT="$(cd "$OUT" && pwd)"

FIXTURE="$OUT/fixture-model"
mkdir -p "$FIXTURE"
printf '{"model_type": "not_a_real_architecture", "hidden_size": 8}\n' > "$FIXTURE/config.json"
printf '{"tokenizer_class": "PreTrainedTokenizerFast"}\n' > "$FIXTURE/tokenizer_config.json"
: > "$FIXTURE/model.safetensors"

FAILURES=0
pass() { printf 'PASS  %s\n' "$*"; }
fail() { printf 'FAIL  %s\n' "$*"; FAILURES=$((FAILURES + 1)); }
info() { printf 'INFO  %s\n' "$*"; }

if lsof -nP -iTCP:"$PORT" -sTCP:LISTEN >/dev/null 2>&1; then
    fail "port $PORT is already in use"
    exit 1
fi

# stage <name>; copies the binary into a fresh directory and echoes that directory
stage() {
    local dir="$OUT/$1"
    mkdir -p "$dir"
    cp "$BINARY" "$dir/prototype"
    printf '%s' "$dir"
}

# run_serve <dir> <log>; runs the staged binary in the background, waits up to
# 20s for it to exit, and reports its status plus whether it ever listened.
run_serve() {
    local dir="$1" log="$2"
    "$dir/prototype" serve --model "$FIXTURE" --host "$HOST" --port "$PORT" > "$log" 2>&1 &
    local pid=$!
    local listened=0
    for _ in $(seq 1 200); do
        if lsof -nP -iTCP:"$PORT" -sTCP:LISTEN >/dev/null 2>&1; then listened=1; break; fi
        if ! kill -0 "$pid" 2>/dev/null; then break; fi
        sleep 0.1
    done
    if kill -0 "$pid" 2>/dev/null; then
        kill -TERM "$pid" 2>/dev/null
        for _ in $(seq 1 100); do
            kill -0 "$pid" 2>/dev/null || break
            sleep 0.1
        done
        kill -KILL "$pid" 2>/dev/null
    fi
    wait "$pid" 2>/dev/null
    SERVE_EXIT=$?
    SERVE_LISTENED=$listened
}

# ------------------------------------------------- 1. forged: a directory --
info "forged evidence: a directory named $LIB_NAME"
DIR_CASE="$(stage forged-directory)"
mkdir -p "$DIR_CASE/$BUNDLE_REL/$LIB_NAME"
run_serve "$DIR_CASE" "$OUT/forged-directory.log"
[ "$SERVE_EXIT" -eq 2 ] \
    && pass "a directory at the library path exits 2 (got $SERVE_EXIT)" \
    || fail "a directory at the library path exited $SERVE_EXIT, expected 2"
[ "$SERVE_LISTENED" -eq 0 ] \
    && pass "no listener was ever bound with the forged directory" \
    || fail "the forged directory bound a listener on $PORT"
grep -q "is a directory" "$OUT/forged-directory.log" \
    && pass "the refusal names the forged object" \
    || fail "the refusal does not name the forged object"
grep -q '"event":"listening"' "$OUT/forged-directory.log" \
    && fail "a listening event was emitted with the forged directory" \
    || pass "no listening event with the forged directory"

# --------------------------------------------- 2. forged: dangling symlink --
info "forged evidence: a dangling symlink named $LIB_NAME"
LINK_CASE="$(stage forged-symlink)"
mkdir -p "$LINK_CASE/$BUNDLE_REL"
ln -s "$LINK_CASE/nowhere" "$LINK_CASE/$BUNDLE_REL/$LIB_NAME"
run_serve "$LINK_CASE" "$OUT/forged-symlink.log"
[ "$SERVE_EXIT" -eq 2 ] \
    && pass "a dangling symlink at the library path exits 2 (got $SERVE_EXIT)" \
    || fail "a dangling symlink at the library path exited $SERVE_EXIT, expected 2"
[ "$SERVE_LISTENED" -eq 0 ] \
    && pass "no listener was ever bound with the dangling symlink" \
    || fail "the dangling symlink bound a listener on $PORT"

# ------------------------------------------------- 3. control: nothing there --
info "control: no shader bundle at all"
BARE_CASE="$(stage no-bundle)"
run_serve "$BARE_CASE" "$OUT/no-bundle.log"
[ "$SERVE_EXIT" -eq 2 ] \
    && pass "a swift-build product with no bundle exits 2 (got $SERVE_EXIT)" \
    || fail "a swift-build product with no bundle exited $SERVE_EXIT, expected 2"
[ "$SERVE_LISTENED" -eq 0 ] \
    && pass "no listener was ever bound without the bundle" \
    || fail "the missing bundle bound a listener on $PORT"

# ------------------------------------ 4. control: the gate is not blanket-off --
# A regular file at the library path must be admitted. Without this the refusals
# above would also be satisfied by a gate that refuses unconditionally, which
# proves nothing about the object type it is supposed to be checking.
info "control: a regular file at the library path is admitted"
FILE_CASE="$(stage regular-file)"
mkdir -p "$FILE_CASE/$BUNDLE_REL"
printf 'not a real metallib' > "$FILE_CASE/$BUNDLE_REL/$LIB_NAME"
run_serve "$FILE_CASE" "$OUT/regular-file.log"
if [ "$SERVE_LISTENED" -eq 1 ] || grep -q '"event":"listening"' "$OUT/regular-file.log"; then
    pass "a regular file at the library path passes the gate and binds"
else
    fail "a regular file at the library path did not pass the gate (exit $SERVE_EXIT)"
    printf '    log: %s\n' "$(head -c 500 "$OUT/regular-file.log")"
fi

sleep 1
if lsof -nP -iTCP:"$PORT" -sTCP:LISTEN >/dev/null 2>&1; then
    fail "port $PORT still has a listener at the end of the probe"
else
    pass "port $PORT is free at the end of the probe"
fi

printf '\n%s\n' "----------------------------------------"
if [ "$FAILURES" -eq 0 ]; then
    printf 'METALLIB GATE PROBE OK (0 failures)\n'
    exit 0
fi
printf 'METALLIB GATE PROBE FAILED (%d failures)\n' "$FAILURES"
exit 1

#!/bin/bash
set -euo pipefail

if [[ $(uname -s) != Darwin ]]; then
    printf '%s\n' 'macos-test: run this script on macOS' >&2
    exit 2
fi

cd "$(dirname "$0")/.."
export GOTOOLCHAIN=${GOTOOLCHAIN:-go1.27.1}
command -v go >/dev/null || { printf '%s\n' 'macos-test: install Go 1.21 or newer (it fetches the pinned 1.27.1)' >&2; exit 2; }
compiler=${VELES_CLANG:-clang}
command -v "$compiler" >/dev/null || { printf '%s\n' 'macos-test: install Xcode Command Line Tools with xcode-select --install' >&2; exit 2; }

go version
"$compiler" --version
if [[ $# -gt 0 ]]; then
    "$@"
else
    go build ./...
    go vet ./...
    log_base=${VELES_TEST_LOG_DIR:-${TMPDIR:-/tmp}}
    mkdir -p "$log_base"
    log_dir=$(mktemp -d "$log_base/veles-macos-test.XXXXXX")
    test_events="$log_dir/go-test.jsonl"
    test_stderr="$log_dir/go-test.stderr"
    trap 'cat "$test_stderr" >&2; printf "\nmacos-test: test events: %s\nmacos-test: test stderr: %s\n" "$test_events" "$test_stderr" >&2' EXIT
    go test -json ./... 2>"$test_stderr" | tee "$test_events"
fi
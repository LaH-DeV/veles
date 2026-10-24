#!/bin/bash
set -euo pipefail

if [[ $(uname -s) != Darwin ]]; then
    printf '%s\n' 'macos-test: run this script on macOS' >&2
    exit 2
fi

cd "$(dirname "$0")/.."
export GOTOOLCHAIN=${GOTOOLCHAIN:-go1.23.2}
command -v go >/dev/null || { printf '%s\n' 'macos-test: install Go 1.23 or newer' >&2; exit 2; }
compiler=${VELES_CLANG:-clang}
command -v "$compiler" >/dev/null || { printf '%s\n' 'macos-test: install Xcode Command Line Tools with xcode-select --install' >&2; exit 2; }

go version
"$compiler" --version
if [[ $# -gt 0 ]]; then
    "$@"
else
    go build ./...
    go vet ./...
    go test ./...
fi
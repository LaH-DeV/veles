#!/bin/bash
# Runs a command against this working tree on Linux, from Windows:
#
#   wsl -d Ubuntu -- bash /mnt/c/<path to the repo>/internal/wsl-test.sh [command...]
#
# (from Git Bash, set MSYS_NO_PATHCONV=1 first, or it rewrites the path).
#
# It keeps a clone at ~/veles inside WSL — the Linux file system, so the
# build is fast and the files have LF endings — brings it to the Windows
# tree's HEAD, copies over every changed or untracked file with CRLF
# stripped, and runs the command there (default: build, vet and every test).
# Go is looked for in ~/.local/go (a user-space install needs no sudo:
# `curl -sSL https://go.dev/dl/go1.23.2.linux-amd64.tar.gz | tar -xz -C ~/.local`);
# clang comes from `sudo apt install clang`.
#
# Stress, as in the veles-test skill:
#   ... wsl-test.sh 'for t in 1 2 4 8; do VELES_THREADS=\$t go test ./examples -count=5; done'
# wsl.exe hands its command line to a shell first, so a `$` meant for the
# command is written `\$`.
# Sanitizers (plan A2): ... wsl-test.sh 'go test ./examples -sanitize'
set -u
here=$(cd "$(dirname "$0")/.." && pwd)
export PATH=$HOME/.local/go/bin:$HOME/go/bin:/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin
command -v go >/dev/null || { echo "wsl-test: no Go in WSL (see the header of this script)"; exit 2; }
command -v clang >/dev/null || { echo "wsl-test: no clang in WSL: sudo apt install clang"; exit 2; }
clone=$HOME/veles
[ -d "$clone/.git" ] || git clone -q "$here" "$clone" || exit 1
cd "$clone" || exit 1
# the clone follows the Windows tree's HEAD; its own edits are only the copies below
git checkout -q -- . && git clean -qfd && git fetch -q "$here" HEAD && git checkout -q --detach FETCH_HEAD
# WSL's git has no core.autocrlf, so it also lists files that differ only in
# CRLF; copying those is harmless. Files marked -text are copied as bytes.
(cd "$here" && git -c core.quotepath=off status --porcelain --untracked-files=all | cut -c4-) | while read -r f; do
	case "$f" in *" -> "*) f=${f#* -> } ;; esac
	if [ -f "$here/$f" ]; then
		mkdir -p "$(dirname "$f")"
		if [ "$(cd "$here" && git check-attr text -- "$f" | sed 's/.*: //')" = unset ]; then
			cp "$here/$f" "$f"
		else
			sed 's/\r$//' "$here/$f" >"$f"
		fi
	else
		rm -f "$f"
	fi
done
if [ $# -eq 0 ]; then
	set -- 'go build ./... && go vet ./... && go test ./...'
fi
eval "$@"

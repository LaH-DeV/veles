package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// std/build (D160, D204): `build.target` is the target, `build.program` the
// package the build started from, both constants — a constant branch on
// them compiles only the branch taken, and `static assert` can test them.
func TestBuildModule(t *testing.T) {
	t.Parallel()
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	const program = `use io
use build

const SEPARATOR: string = if (build.target.os == build.Os.Windows) "\\" else "/"

static assert(build.program.name == "tool", "the manifest names the program")

fun main() {
  io.println("${build.target.os} ${build.target.arch} ${build.target.release} ${build.program.name} ${build.program.version} $SEPARATOR")
}
`
	dir := t.TempDir()
	files := map[string]string{
		"veles.toml": "[package]\nname = \"tool\"\nversion = \"1.4.2\"\n",
		"main.vs":    program,
	}
	for name, text := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	osName, sep := "Linux", "/"
	switch runtime.GOOS {
	case "windows":
		osName, sep = "Windows", `\`
	case "darwin":
		osName = "MacOS"
	}
	arch := "X64"
	if runtime.GOARCH == "arm64" {
		arch = "Arm64"
	}
	for _, release := range profiles() {
		exe := filepath.Join(dir, "tool.exe")
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release}); code != 0 {
			t.Fatalf("release=%v: build failed with exit %d", release, code)
		}
		out, err := exec.Command(exe).CombinedOutput()
		want := osName + " " + arch + " " + map[bool]string{false: "false", true: "true"}[release] + " tool 1.4.2 " + sep
		if got := strings.TrimSpace(string(out)); err != nil || got != want {
			t.Errorf("release=%v: got %q (%v), want %q", release, got, err, want)
		}
	}
}

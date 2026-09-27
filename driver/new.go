package driver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// New is `veles new <dir>`: a package that runs and tests on the first
// try — a manifest, a main module with a function, `main` and a test that
// passes, and a .gitignore for what `veles build` writes. The package is
// named after the directory. An existing directory is used only when it
// is empty, so nothing is ever overwritten.
func New(dir string) int {
	name := filepath.Base(filepath.Clean(dir))
	if !packageName(name) {
		fmt.Fprintf(os.Stderr, "veles new: %q is not a package name: start with a letter, then letters, digits or '_' (other packages write it in `use`)\n", name)
		return 2
	}
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		fmt.Fprintf(os.Stderr, "veles new: %s is not empty; choose a new directory\n", dir)
		return 1
	}
	files := map[string]string{
		"veles.toml": "[package]\nname = \"" + name + "\"\nversion = \"0.1.0\"\n",
		"main.vs": `use io

/// What the program says to ` + "`name`" + `.
fun greeting(name: string): string = "Hello, $name!"

fun main() {
  io.println(greeting("world"))
}

test "greets by name" {
  expect(greeting("Veles") == "Hello, Veles!")
}
`,
		".gitignore": "/" + name + "\n/" + name + ".exe\n*.ll\n",
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "veles new:", err)
		return 1
	}
	for _, f := range []string{"veles.toml", "main.vs", ".gitignore"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte(files[f]), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "veles new:", err)
			return 1
		}
	}
	fmt.Printf("created package %s in %s\n\n  cd %s\n  veles run       # Hello, world!\n  veles test      # runs greetsByName\n", name, dir, dir)
	return 0
}

// packageName reports whether s can name a package: an identifier, since
// a package that depends on it writes the name in `use name.module`.
func packageName(s string) bool {
	if s == "" || !(s[0] >= 'a' && s[0] <= 'z' || s[0] >= 'A' && s[0] <= 'Z') {
		return false
	}
	return strings.IndexFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_')
	}) < 0
}

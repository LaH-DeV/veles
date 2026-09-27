package driver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/LaH-DeV/veles/sema"
	"github.com/LaH-DeV/veles/source"
)

// Doc is `veles doc [path] [-o dir]`: the package's public API as
// Markdown — every module on standard output, or one `<module>.md` per
// module in dir.
func Doc(path, outDir string) int {
	diags := &source.Diagnostics{}
	pkg, err := sema.LoadPackage(path, diags)
	if err != nil {
		fmt.Fprintln(os.Stderr, "veles:", err)
		return 1
	}
	var docs []sema.ModuleDoc
	if !diags.HasErrors() {
		pkg.LoadAll() // a module nothing imports is part of the API too
		docs = sema.DocPackage(pkg, diags)
	}
	if diags.HasErrors() {
		fmt.Fprint(os.Stderr, diags.Render())
		fmt.Fprintln(os.Stderr, "veles doc: the package does not check; fix the errors above first")
		return 1
	}
	if outDir == "" {
		for i, d := range docs {
			if i > 0 {
				fmt.Println()
			}
			fmt.Print(d.Text)
		}
		return 0
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "veles doc:", err)
		return 1
	}
	for _, d := range docs {
		file := filepath.Join(outDir, strings.ReplaceAll(d.Name, ".", "-")+".md")
		if err := os.WriteFile(file, []byte(d.Text), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "veles doc:", err)
			return 1
		}
		fmt.Println("wrote", file)
	}
	return 0
}

package driver

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/LaH-DeV/veles/format"
	"github.com/LaH-DeV/veles/sema"
	"github.com/LaH-DeV/veles/source"
)

// FormatOptions drive `veles fmt`.
type FormatOptions struct {
	Paths  []string // files or directories (directories are walked)
	Check  bool     // list files that would change and exit 1 instead of writing
	Stdout bool     // print the formatted text instead of writing
}

// Format runs the formatter over the given paths and returns an exit code:
// 0 when everything is formatted (or was rewritten), 1 when a file failed
// to parse or, with Check, would change.
func Format(opts FormatOptions) int {
	var files []string
	for _, p := range opts.Paths {
		info, err := os.Stat(p)
		if err != nil {
			fmt.Fprintln(os.Stderr, "veles fmt:", err)
			return 1
		}
		if !info.IsDir() {
			files = append(files, p)
			continue
		}
		filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() && path != p && (strings.HasPrefix(d.Name(), ".") || d.Name() == "node_modules") {
				return fs.SkipDir
			}
			if !d.IsDir() && strings.HasSuffix(d.Name(), ".vs") {
				files = append(files, path)
			}
			return nil
		})
	}
	code := 0
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, "veles fmt:", err)
			code = 1
			continue
		}
		style, err := StyleFor(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, "veles fmt:", err)
			code = 1
			continue
		}
		text := string(data)
		out, diags := format.Source(source.NewFile(path, text), style)
		if diags.HasErrors() {
			fmt.Fprint(os.Stderr, diags.Render())
			code = 1
			continue
		}
		switch {
		case opts.Stdout:
			fmt.Print(out)
		case out == strings.ReplaceAll(text, "\r\n", "\n"):
			// already formatted
		case opts.Check:
			fmt.Println(path)
			code = 1
		default:
			if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
				fmt.Fprintln(os.Stderr, "veles fmt:", err)
				code = 1
				continue
			}
			fmt.Println(path)
		}
	}
	return code
}

// StyleFor returns the formatting options for a file: the `[format]` table
// of the package it belongs to, or the defaults.
func StyleFor(path string) (format.Options, error) {
	man, err := sema.ReadManifest(path)
	if err != nil || man == nil {
		return format.Default, err
	}
	return format.Options{Indent: man.Format.Indent, MaxBlankLines: man.Format.MaxBlankLines}, nil
}

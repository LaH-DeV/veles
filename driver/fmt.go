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
	walkFailed := false
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
		// An entry that cannot be read is reported and makes the run fail,
		// but the walk goes on: one unreadable directory should not hide
		// the state of every other file, and `--check` must never pass on
		// a tree it did not fully read.
		walkErr := filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				fmt.Fprintln(os.Stderr, "veles fmt:", err)
				walkFailed = true
				if d != nil && d.IsDir() && path != p {
					return fs.SkipDir
				}
				return nil
			}
			if d.IsDir() && path != p && (strings.HasPrefix(d.Name(), ".") || d.Name() == "node_modules") {
				return fs.SkipDir
			}
			if !d.IsDir() && (strings.HasSuffix(d.Name(), ".vs") || sema.IsScript(d.Name())) {
				files = append(files, path)
			}
			return nil
		})
		if walkErr != nil {
			fmt.Fprintln(os.Stderr, "veles fmt:", walkErr)
			walkFailed = true
		}
	}
	code := 0
	if walkFailed {
		code = 1
	}
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
			if err := writeAtomic(path, []byte(out)); err != nil {
				fmt.Fprintln(os.Stderr, "veles fmt:", err)
				code = 1
				continue
			}
			fmt.Println(path)
		}
	}
	return code
}

// writeAtomic replaces the file at path with data so that a crash, a full
// disk or a kill leaves either the old contents or the new ones, never a
// truncated file: the text goes to a temporary file in the same directory
// (one filesystem, so the rename is atomic), is flushed, takes the
// original's permission bits, and is renamed over the original.
func writeAtomic(path string, data []byte) (err error) {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".veles-fmt-*")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
		}
	}()
	if _, err = tmp.Write(data); err != nil {
		return err
	}
	if err = tmp.Sync(); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Chmod(tmp.Name(), info.Mode().Perm()); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
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

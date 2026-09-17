package sema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/source"
)

// FuzzCheckMutated keeps the checker busy: the fuzzer's bytes drive a
// small set of token-level edits on a seed program (swap two tokens,
// delete one, duplicate one, replace an identifier with another from the
// same file), so most inputs still parse and the type checker, not the
// parser, sees the variation.
func FuzzCheckMutated(f *testing.F) {
	var seeds []string
	for _, root := range []string{"../examples"} {
		filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && strings.HasSuffix(path, "main.vs") {
				if data, err := os.ReadFile(path); err == nil {
					seeds = append(seeds, string(data))
				}
			}
			return nil
		})
	}
	if len(seeds) == 0 {
		f.Skip("no seeds")
	}
	for i := range seeds {
		f.Add(i, []byte{1, 2, 3})
	}
	dir := f.TempDir()
	path := filepath.Join(dir, "main.vs")
	os.WriteFile(path, []byte("fun main() { }\n"), 0o644)
	f.Fuzz(func(t *testing.T, seed int, edits []byte) {
		if seed < 0 {
			seed = -seed
		}
		src := mutate(seeds[seed%len(seeds)], edits)
		done := make(chan struct{})
		go func() {
			defer close(done)
			diags := &source.Diagnostics{}
			pkg, err := LoadPackageOverlay(dir, diags, map[string]string{OverlayKey(path): src})
			if err != nil {
				return
			}
			if !diags.HasErrors() {
				Check(pkg, diags, false)
			}
		}()
		select {
		case <-done:
		case <-time.After(20 * time.Second):
			t.Fatalf("checking did not finish within 20s:\n%s", src)
		}
	})
}

// mutate applies up to len(edits)/3 edits described by the bytes to the
// atoms of the program — identifiers and literals — leaving the punctuation
// and keywords alone, so most results still parse: mostly replacing an atom
// by another of the same kind, sometimes swapping two, rarely deleting or
// duplicating one.
func mutate(src string, edits []byte) string {
	file := source.NewFile("seed.vs", src)
	diags := &source.Diagnostics{}
	toks := lexer.Tokenize(file, diags)
	if diags.HasErrors() || len(toks) < 4 {
		return src
	}
	type piece struct {
		ws, text string
		kind     lexer.TokenKind
	}
	var pieces []piece
	prev := 0
	for _, tk := range toks {
		if tk.Kind == lexer.EOF {
			break
		}
		if tk.Kind == lexer.Semi && tk.AutoSemi {
			continue // regenerated from the line breaks in ws
		}
		pieces = append(pieces, piece{src[prev:tk.Span.Start], src[tk.Span.Start:tk.Span.End], tk.Kind})
		prev = tk.Span.End
	}
	tail := src[prev:]
	isAtom := func(k lexer.TokenKind) bool {
		return k == lexer.Ident || k == lexer.Int || k == lexer.Float || k == lexer.String
	}
	var atoms []int // indexes of atom pieces
	for i, p := range pieces {
		if isAtom(p.kind) {
			atoms = append(atoms, i)
		}
	}
	if len(atoms) < 2 {
		return src
	}
	byKind := map[lexer.TokenKind][]string{}
	for _, i := range atoms {
		byKind[pieces[i].kind] = append(byKind[pieces[i].kind], pieces[i].text)
	}
	for i := 0; i+2 < len(edits) && len(atoms) >= 2; i += 3 {
		op := int(edits[i]) % 8
		n := int(edits[i+1])<<8 | int(edits[i+2])
		a := atoms[n%len(atoms)]
		if a >= len(pieces) || !isAtom(pieces[a].kind) {
			continue
		}
		switch {
		case op < 4: // replace by another atom of the same kind
			pool := byKind[pieces[a].kind]
			pieces[a].text = pool[(n/len(atoms)+i)%len(pool)]
		case op < 6: // swap two atoms
			b := atoms[(n/7)%len(atoms)]
			if b < len(pieces) && isAtom(pieces[b].kind) {
				pieces[a].text, pieces[b].text = pieces[b].text, pieces[a].text
			}
		case op == 6: // delete
			pieces = append(pieces[:a], pieces[a+1:]...)
			atoms = atoms[:0]
			for j, p := range pieces {
				if isAtom(p.kind) {
					atoms = append(atoms, j)
				}
			}
		default: // duplicate
			pieces = append(pieces[:a+1], append([]piece{{" ", pieces[a].text, pieces[a].kind}}, pieces[a+1:]...)...)
			atoms = atoms[:0]
			for j, p := range pieces {
				if isAtom(p.kind) {
					atoms = append(atoms, j)
				}
			}
		}
	}
	var sb strings.Builder
	for _, p := range pieces {
		sb.WriteString(p.ws)
		sb.WriteString(p.text)
	}
	sb.WriteString(tail)
	return sb.String()
}

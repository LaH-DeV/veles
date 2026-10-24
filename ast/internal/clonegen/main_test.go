package main

import (
	"bytes"
	"os"
	"testing"
)

// TestGeneratedIsCurrent: ast/clone_gen.go is what clonegen makes of the
// node types in ast/ast.go, so a new node or field is copied too.
func TestGeneratedIsCurrent(t *testing.T) {
	src, err := os.ReadFile("../../ast.go")
	if err != nil {
		t.Fatal(err)
	}
	want, err := generate(src)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile("../../clone_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bytes.ReplaceAll(got, []byte("\r\n"), []byte("\n")), want) {
		t.Fatal("ast/clone_gen.go is out of date: run `go generate ./ast`")
	}
}

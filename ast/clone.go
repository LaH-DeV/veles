package ast

//go:generate go run ./internal/clonegen

// Clone returns a deep copy of a parsed file: every node, and every slice of
// the tree, is new, so a checker may rewrite the copy in place (D158 writes
// `this.name` for a bare member, D176 wraps a decorated body) while another
// checker holds the original or another copy. What the original shares (a
// declaration reached twice) the copy shares too. What is not part of the
// tree is shared: the source file, spans, and any value of another package
// that a field of type `any` holds (the checker's own, in synthesized
// nodes). The copy is generated from ast.go (clone_gen.go).
func Clone(f *File) *File {
	var c cloner
	return c.cloneFile(f)
}

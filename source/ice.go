package source

import "sync/atomic"

// Where the compiler was when something went wrong: the phase and the
// function being worked on. It is recorded once per function (checking a
// body, generating its code), which is cheap and names the unit a bug
// report needs. Only the command-line driver reads it, to turn a panic into
// an internal-compiler-error report; concurrent writers (the language
// server) are safe but the value is then only a hint.
type Where struct {
	Phase string // "checking", "generating code for"
	Name  string // the function, as diagnostics spell it
	Span  Span
}

var where atomic.Pointer[Where]

// SetWhere records what the compiler is working on now.
func SetWhere(phase, name string, span Span) {
	where.Store(&Where{Phase: phase, Name: name, Span: span})
}

// CurrentWhere returns the last SetWhere, or nil.
func CurrentWhere() *Where { return where.Load() }

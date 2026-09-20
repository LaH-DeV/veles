// Package types is the semantic type representation shared by the checker
// and the code generator.
package types

import (
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
)

type Type interface {
	String() string
}

// ---------------------------------------------------------------------------
// Basic types

type BasicKind int

const (
	Invalid BasicKind = iota
	Unit
	Bool
	I8
	I16
	I32
	I64
	ISize
	U8
	U16
	U32
	U64
	USize
	F32
	F64
	String
	Never // type of `return`, `break`, and panics; assignable to anything
)

type Basic struct {
	Kind  BasicKind
	Name  string
	Alias string // display name from a `type` alias (Aliased)
}

func (b *Basic) String() string {
	if b.Alias != "" {
		return b.Alias
	}
	return b.Name
}

var (
	TInvalid = &Basic{Kind: Invalid, Name: "<invalid>"}
	TUnit    = &Basic{Kind: Unit, Name: "()"}
	TBool    = &Basic{Kind: Bool, Name: "bool"}
	TI8      = &Basic{Kind: I8, Name: "i8"}
	TI16     = &Basic{Kind: I16, Name: "i16"}
	TI32     = &Basic{Kind: I32, Name: "i32"}
	TI64     = &Basic{Kind: I64, Name: "i64"}
	TISize   = &Basic{Kind: ISize, Name: "isize"}
	TU8      = &Basic{Kind: U8, Name: "u8"}
	TU16     = &Basic{Kind: U16, Name: "u16"}
	TU32     = &Basic{Kind: U32, Name: "u32"}
	TU64     = &Basic{Kind: U64, Name: "u64"}
	TUSize   = &Basic{Kind: USize, Name: "usize"}
	TF32     = &Basic{Kind: F32, Name: "f32"}
	TF64     = &Basic{Kind: F64, Name: "f64"}
	TString  = &Basic{Kind: String, Name: "string"}
	TNever   = &Basic{Kind: Never, Name: "Never"}
)

var Primitives = map[string]*Basic{
	"bool": TBool, "i8": TI8, "i16": TI16, "i32": TI32, "i64": TI64, "isize": TISize,
	"u8": TU8, "u16": TU16, "u32": TU32, "u64": TU64, "usize": TUSize,
	"f32": TF32, "f64": TF64, "string": TString,
	"Never": TNever, // the bottom type: a function that never returns (D20)
}

func IsInteger(t Type) bool {
	b, ok := t.(*Basic)
	return ok && b.Kind >= I8 && b.Kind <= USize
}

func IsSigned(t Type) bool {
	b, ok := t.(*Basic)
	return ok && b.Kind >= I8 && b.Kind <= ISize
}

func IsUnsigned(t Type) bool {
	b, ok := t.(*Basic)
	return ok && b.Kind >= U8 && b.Kind <= USize
}

func IsFloat(t Type) bool {
	b, ok := t.(*Basic)
	return ok && (b.Kind == F32 || b.Kind == F64)
}

func IsNumeric(t Type) bool { return IsInteger(t) || IsFloat(t) }

func IsBool(t Type) bool   { b, ok := t.(*Basic); return ok && b.Kind == Bool }
func IsString(t Type) bool { b, ok := t.(*Basic); return ok && b.Kind == String }
func IsUnit(t Type) bool   { b, ok := t.(*Basic); return ok && b.Kind == Unit }
func IsNever(t Type) bool  { b, ok := t.(*Basic); return ok && b.Kind == Never }
func IsInvalid(t Type) bool {
	b, ok := t.(*Basic)
	return ok && b.Kind == Invalid
}

// BitSize returns the width of an integer or float type.
func BitSize(t Type) int {
	b, ok := t.(*Basic)
	if !ok {
		return 0
	}
	switch b.Kind {
	case I8, U8:
		return 8
	case I16, U16:
		return 16
	case I32, U32, F32:
		return 32
	case I64, U64, ISize, USize, F64:
		return 64
	case Bool:
		return 1
	}
	return 0
}

// ---------------------------------------------------------------------------
// Composite types

// Pointer is `*T` (GC-managed) or `*raw T`.
type Pointer struct {
	Elem  Type
	Raw   bool
	Alias string
}

func (p *Pointer) String() string {
	if p.Alias != "" {
		return p.Alias
	}
	if p.Raw {
		return "*raw " + p.Elem.String()
	}
	return "*" + p.Elem.String()
}

// Nullable is `T?`, i.e. Option<T> (D5). Nests: `T??` is distinct from `T?`.
type Nullable struct {
	Elem  Type
	Alias string
}

func (n *Nullable) String() string {
	if n.Alias != "" {
		return n.Alias
	}
	if _, ok := n.Elem.(*Pointer); ok {
		return "(" + n.Elem.String() + ")?"
	}
	return n.Elem.String() + "?"
}

type Tuple struct {
	Elems []Type
	Alias string
}

func (t *Tuple) String() string {
	if t.Alias != "" {
		return t.Alias
	}
	parts := make([]string, len(t.Elems))
	for i, e := range t.Elems {
		parts[i] = e.String()
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

// Range is the type of `lo..hi` / `lo..<hi` (D29).
type Range struct {
	Elem  Type
	Alias string
}

func (r *Range) String() string {
	if r.Alias != "" {
		return r.Alias
	}
	return "Range<" + r.Elem.String() + ">"
}

// List is the immutable `List<T>`; MutableList is `MutableList<T>` (D25/D41).
type List struct {
	Elem    Type
	Mutable bool
	Alias   string
}

func (l *List) String() string {
	if l.Alias != "" {
		return l.Alias
	}
	if l.Mutable {
		return "MutableList<" + l.Elem.String() + ">"
	}
	return "List<" + l.Elem.String() + ">"
}

// Effects declared or inferred on a function (D2/D4/D40).
type Effects struct {
	Suspends bool
	Throws   bool
	Error    Type // nil while being inferred; an ErrorUnion or a single type
}

type Param struct {
	Name       string
	Type       Type
	HasDefault bool
	Variadic   bool // Type is the List<T> the trailing arguments are collected into
}

// Func is a function signature. Sendable marks a value that may cross a
// task boundary (D35): a named function, or a closure whose captures are
// all `val`s of Sendable types; the flag is part of the type, and a
// sendable function is assignable to a plain function type, not the reverse.
type Func struct {
	Params   []Param
	Ret      Type
	Effects  Effects
	Sendable bool
	Alias    string
}

func (f *Func) String() string {
	if f.Alias != "" {
		return f.Alias
	}
	parts := make([]string, len(f.Params))
	for i, p := range f.Params {
		parts[i] = p.Type.String()
	}
	s := "fun(" + strings.Join(parts, ", ") + ")"
	if f.Sendable {
		s = "sendable " + s
	}
	if f.Ret != nil && !IsUnit(f.Ret) {
		s += ": " + f.Ret.String()
	}
	if f.Effects.Suspends {
		s += " suspends"
	}
	if f.Effects.Throws {
		s += " throws"
		if f.Effects.Error != nil {
			s += " " + f.Effects.Error.String()
		}
	}
	return s
}

// ErrorUnion is `A | B`, only in error position (D45). Members are kept
// sorted by name and flattened so that equal unions compare identical.
type ErrorUnion struct {
	Members []Type
	Alias   string // the `error Set = A | B` name this union was written as
}

func (u *ErrorUnion) String() string {
	if u.Alias != "" {
		return u.Alias
	}
	parts := make([]string, len(u.Members))
	for i, m := range u.Members {
		parts[i] = m.String()
	}
	return strings.Join(parts, " | ")
}

// MakeErrorUnion flattens and sorts members; a single member is returned as
// itself and zero members as nil.
func MakeErrorUnion(members ...Type) Type {
	var flat []Type
	sawNever := false
	var add func(t Type)
	add = func(t Type) {
		if t == nil {
			return
		}
		if u, ok := t.(*ErrorUnion); ok {
			for _, m := range u.Members {
				add(m)
			}
			return
		}
		if IsNever(t) {
			// `E | Timeout` with E bound to Never (a lambda that does not
			// throw): nothing to add — the union is the other members
			sawNever = true
			return
		}
		for _, m := range flat {
			if Identical(m, t) {
				return
			}
		}
		flat = append(flat, t)
	}
	for _, m := range members {
		add(m)
	}
	switch len(flat) {
	case 0:
		if sawNever {
			return TNever // every member was Never
		}
		return nil
	case 1:
		return flat[0]
	}
	sort.Slice(flat, func(i, j int) bool { return flat[i].String() < flat[j].String() })
	return &ErrorUnion{Members: flat}
}

// UnionMembers returns the members of an error type (one for a non-union).
func UnionMembers(t Type) []Type {
	if t == nil {
		return nil
	}
	if u, ok := t.(*ErrorUnion); ok {
		return u.Members
	}
	return []Type{t}
}

// UnionIndex returns the position of member m in error type u, or -1.
func UnionIndex(u Type, m Type) int {
	for i, x := range UnionMembers(u) {
		if Identical(x, m) {
			return i
		}
	}
	return -1
}

// ---------------------------------------------------------------------------
// Named types

// TypeParam is a generic parameter inside a template declaration.
type TypeParam struct {
	Name   string
	Bounds []*Trait
	Index  int
	Owner  string // for diagnostics
	id     int
}

func (t *TypeParam) String() string { return t.Name }

type Field struct {
	Name       string
	Type       Type
	Pub        bool
	Private    bool // visible only inside the type's own declarations
	Var        bool // assignable after construction (D22 v0.30); a bare field never changes
	Protected  bool // `protected var`: assignable only by the type's own declarations; read like any field
	Init       bool // assigned by the struct's `init { }` block: not a constructor parameter (D28 v0.30)
	HasDefault bool
	Index      int
}

// Struct is a user (or builtin) struct. A generic struct declaration is a
// template with TypeParams set and TypeArgs nil; each instantiation is a
// separate *Struct with TypeArgs set and Template pointing back (D8 — one
// body per instantiation is the bootstrap policy; GC-shape sharing is a
// later optimisation invisible to semantics per D15).
type Struct struct {
	Name       string
	Module     string
	Pub        bool
	Extern     bool
	TypeParams []*TypeParam
	TypeArgs   []Type
	Template   *Struct
	Fields     []*Field
	Sealed     *Sealed // the sealed trait this struct is a variant of
	Tag        int     // variant index within Sealed
	Instances  map[string]*Struct
	Methods    map[string]any // *sema.Func; opaque here to avoid a cycle
	// Decl is the declaring AST node (opaque).
	Decl any
}

func (s *Struct) String() string {
	return qualified(s.Name, s.TypeArgs)
}

// Sealed is a sealed trait: a closed sum type laid out inline (D12).
type Sealed struct {
	Name       string
	Module     string
	Pub        bool
	TypeParams []*TypeParam
	TypeArgs   []Type
	Template   *Sealed
	Variants   []*Struct
	Instances  map[string]*Sealed
	Trait      *Trait // the trait half: methods declared on the sealed trait
	Decl       any
}

func (s *Sealed) String() string {
	return qualified(s.Name, s.TypeArgs)
}

// VariantByName finds a variant of a sealed type.
func (s *Sealed) VariantByName(name string) *Struct {
	for _, v := range s.Variants {
		if v.Name == name {
			return v
		}
	}
	return nil
}

// Trait is an open (non-sealed) trait (D6). A trait used as a type denotes
// a boxed trait object (D9).
type Trait struct {
	Name        string
	Module      string
	Pub         bool
	TypeParams  []*TypeParam
	AssocTypes  []string
	AssocBounds map[string][]*Trait
	Methods     map[string]*Func
	MethodList  []string
	Decl        any
	// SelfParam is the synthetic type parameter standing for Self in the
	// trait's default bodies (created by the checker on first use).
	SelfParam *TypeParam
	// ImplicitError is set when a method is declared with a bare `throws`:
	// the trait then carries an associated type `Error` that each impl
	// defines through its methods' error types (D40, v0.24).
	ImplicitError bool
}

func (t *Trait) String() string { return t.Name }

func qualified(name string, args []Type) string {
	if len(args) == 0 {
		return name
	}
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = a.String()
	}
	return name + "<" + strings.Join(parts, ", ") + ">"
}

// ---------------------------------------------------------------------------
// Identity and substitution

// Identical reports structural identity. Named types are identical only
// when they are the same declaration instance.
func Identical(a, b Type) bool {
	if a == b {
		return true
	}
	switch a := a.(type) {
	case *Basic:
		b, ok := b.(*Basic)
		return ok && a.Kind == b.Kind
	case *Pointer:
		b, ok := b.(*Pointer)
		return ok && a.Raw == b.Raw && Identical(a.Elem, b.Elem)
	case *Nullable:
		b, ok := b.(*Nullable)
		return ok && Identical(a.Elem, b.Elem)
	case *Tuple:
		b, ok := b.(*Tuple)
		if !ok || len(a.Elems) != len(b.Elems) {
			return false
		}
		for i := range a.Elems {
			if !Identical(a.Elems[i], b.Elems[i]) {
				return false
			}
		}
		return true
	case *Range:
		b, ok := b.(*Range)
		return ok && Identical(a.Elem, b.Elem)
	case *List:
		b, ok := b.(*List)
		return ok && a.Mutable == b.Mutable && Identical(a.Elem, b.Elem)
	case *Map:
		b, ok := b.(*Map)
		return ok && a.Mutable == b.Mutable && Identical(a.Key, b.Key) && Identical(a.Value, b.Value)
	case *Channel:
		b, ok := b.(*Channel)
		return ok && Identical(a.Elem, b.Elem)
	case *Task:
		b, ok := b.(*Task)
		return ok && Identical(a.Result, b.Result)
	case *Set:
		b, ok := b.(*Set)
		return ok && a.Mutable == b.Mutable && Identical(a.Elem, b.Elem)
	case *Func:
		b, ok := b.(*Func)
		if !ok || len(a.Params) != len(b.Params) || !Identical(a.Ret, b.Ret) {
			return false
		}
		for i := range a.Params {
			if !Identical(a.Params[i].Type, b.Params[i].Type) {
				return false
			}
		}
		if a.Effects.Suspends != b.Effects.Suspends || a.Effects.Throws != b.Effects.Throws || a.Sendable != b.Sendable {
			return false
		}
		if a.Effects.Throws && !Identical(a.Effects.Error, b.Effects.Error) {
			return false
		}
		return true
	case *ErrorUnion:
		b, ok := b.(*ErrorUnion)
		if !ok || len(a.Members) != len(b.Members) {
			return false
		}
		for i := range a.Members {
			if !Identical(a.Members[i], b.Members[i]) {
				return false
			}
		}
		return true
	case *Assoc:
		b, ok := b.(*Assoc)
		return ok && a.Trait == b.Trait && a.Name == b.Name && Identical(a.Base, b.Base)
	case *Struct, *Sealed, *Trait, *TypeParam:
		return a == b
	}
	return false
}

// Hooks are the checker's callbacks Subst needs to re-instantiate generic
// structs and sealed types and to resolve associated-type projections once
// their base is concrete. Each checker owns one, so checkers may run
// concurrently (a test suite, a language server).
type Hooks struct {
	StructInstantiator func(tmpl *Struct, args []Type) Type
	SealedInstantiator func(tmpl *Sealed, args []Type) Type
	// AssocResolver looks up the binding of an associated type for a
	// concrete implementing type; nil when there is no impl.
	AssocResolver func(base Type, trait *Trait, name string) Type
}

// Subst replaces type parameters in t according to the mapping.
func (h *Hooks) Subst(t Type, m map[*TypeParam]Type) Type {
	if len(m) == 0 || t == nil {
		return t
	}
	if h == nil {
		h = &Hooks{}
	}
	switch t := t.(type) {
	case *TypeParam:
		if r, ok := m[t]; ok {
			return r
		}
		return t
	case *Assoc:
		base := h.Subst(t.Base, m)
		if base == t.Base {
			return h.ResolveAssoc(t)
		}
		return h.ResolveAssoc(&Assoc{Base: base, Trait: t.Trait, Name: t.Name})
	case *Pointer:
		return &Pointer{Elem: h.Subst(t.Elem, m), Raw: t.Raw}
	case *Nullable:
		return &Nullable{Elem: h.Subst(t.Elem, m)}
	case *Tuple:
		out := make([]Type, len(t.Elems))
		for i, e := range t.Elems {
			out[i] = h.Subst(e, m)
		}
		return &Tuple{Elems: out}
	case *Range:
		return &Range{Elem: h.Subst(t.Elem, m)}
	case *List:
		return &List{Elem: h.Subst(t.Elem, m), Mutable: t.Mutable}
	case *Map:
		return &Map{Key: h.Subst(t.Key, m), Value: h.Subst(t.Value, m), Mutable: t.Mutable}
	case *Set:
		return &Set{Elem: h.Subst(t.Elem, m), Mutable: t.Mutable}
	case *Channel:
		return &Channel{Elem: h.Subst(t.Elem, m)}
	case *Task:
		return &Task{Result: h.Subst(t.Result, m)}
	case *Func:
		out := &Func{Ret: h.Subst(t.Ret, m), Effects: t.Effects, Sendable: t.Sendable}
		out.Effects.Error = h.Subst(t.Effects.Error, m)
		if out.Effects.Throws && out.Effects.Error != nil && IsNever(out.Effects.Error) {
			// `throws E` with E bound to nothing: the function cannot fail,
			// so the instance is an ordinary non-throwing function
			out.Effects.Throws, out.Effects.Error = false, nil
		}
		for _, p := range t.Params {
			out.Params = append(out.Params, Param{Name: p.Name, Type: h.Subst(p.Type, m), HasDefault: p.HasDefault, Variadic: p.Variadic})
		}
		return out
	case *ErrorUnion:
		members := make([]Type, len(t.Members))
		for i, e := range t.Members {
			members[i] = h.Subst(e, m)
		}
		return MakeErrorUnion(members...)
	case *Struct:
		if len(t.TypeArgs) > 0 && h.StructInstantiator != nil {
			if args, changed := h.substArgs(t.TypeArgs, m); changed {
				tmpl := t
				if t.Template != nil {
					tmpl = t.Template
				}
				return h.StructInstantiator(tmpl, args)
			}
		}
	case *Sealed:
		if len(t.TypeArgs) > 0 && h.SealedInstantiator != nil {
			if args, changed := h.substArgs(t.TypeArgs, m); changed {
				tmpl := t
				if t.Template != nil {
					tmpl = t.Template
				}
				return h.SealedInstantiator(tmpl, args)
			}
		}
	}
	return t
}

// ContainsTypeParam reports whether t mentions any generic parameter.
func ContainsTypeParam(t Type) bool {
	switch t := t.(type) {
	case *TypeParam:
		return true
	case *Assoc:
		return true
	case *Pointer:
		return ContainsTypeParam(t.Elem)
	case *Nullable:
		return ContainsTypeParam(t.Elem)
	case *Tuple:
		for _, e := range t.Elems {
			if ContainsTypeParam(e) {
				return true
			}
		}
	case *Range:
		return ContainsTypeParam(t.Elem)
	case *List:
		return ContainsTypeParam(t.Elem)
	case *Map:
		return ContainsTypeParam(t.Key) || ContainsTypeParam(t.Value)
	case *Set:
		return ContainsTypeParam(t.Elem)
	case *Channel:
		return ContainsTypeParam(t.Elem)
	case *Task:
		return ContainsTypeParam(t.Result)
	case *Func:
		for _, p := range t.Params {
			if ContainsTypeParam(p.Type) {
				return true
			}
		}
		return ContainsTypeParam(t.Ret) || ContainsTypeParam(t.Effects.Error)
	case *ErrorUnion:
		for _, m := range t.Members {
			if ContainsTypeParam(m) {
				return true
			}
		}
	case *Struct:
		for _, a := range t.TypeArgs {
			if ContainsTypeParam(a) {
				return true
			}
		}
		return len(t.TypeParams) > 0 && t.TypeArgs == nil
	case *Sealed:
		for _, a := range t.TypeArgs {
			if ContainsTypeParam(a) {
				return true
			}
		}
		return len(t.TypeParams) > 0 && t.TypeArgs == nil
	}
	return false
}

// nextParamID numbers type parameters for Key; atomic, since several checkers
// (a test suite, a language server) may run at once.
var nextParamID atomic.Int64

// Key returns a canonical string for use as a map key. Unlike String, it
// distinguishes type parameters that merely share a name.
func Key(t Type) string {
	switch t := t.(type) {
	case nil:
		return "<nil>"
	case *TypeParam:
		if t.id == 0 {
			t.id = int(nextParamID.Add(1))
		}
		return fmt.Sprintf("%s#%d", t.Name, t.id)
	case *Assoc:
		return Key(t.Base) + "::" + t.Name
	case *Pointer:
		if t.Raw {
			return "*raw " + Key(t.Elem)
		}
		return "*" + Key(t.Elem)
	case *Nullable:
		return "(" + Key(t.Elem) + ")?"
	case *Tuple:
		return "(" + keys(t.Elems) + ")"
	case *Range:
		return "Range<" + Key(t.Elem) + ">"
	case *List:
		if t.Mutable {
			return "MutableList<" + Key(t.Elem) + ">"
		}
		return "List<" + Key(t.Elem) + ">"
	case *Map:
		return "Map" + fmt.Sprint(t.Mutable) + "<" + Key(t.Key) + "," + Key(t.Value) + ">"
	case *Set:
		return "Set" + fmt.Sprint(t.Mutable) + "<" + Key(t.Elem) + ">"
	case *Channel:
		return "Channel<" + Key(t.Elem) + ">"
	case *Task:
		return "Task<" + Key(t.Result) + ">"
	case *Func:
		var ps []Type
		for _, p := range t.Params {
			ps = append(ps, p.Type)
		}
		s := "fun(" + keys(ps) + "):" + Key(t.Ret)
		if t.Sendable {
			s = "sendable " + s
		}
		if t.Effects.Throws {
			s += " throws " + Key(t.Effects.Error)
		}
		return s
	case *ErrorUnion:
		return keys(t.Members)
	case *Struct:
		if len(t.TypeArgs) == 0 {
			return t.Module + "." + t.Name
		}
		return t.Module + "." + t.Name + "<" + keys(t.TypeArgs) + ">"
	case *Sealed:
		if len(t.TypeArgs) == 0 {
			return t.Module + "." + t.Name
		}
		return t.Module + "." + t.Name + "<" + keys(t.TypeArgs) + ">"
	}
	return t.String()
}

func keys(ts []Type) string {
	parts := make([]string, len(ts))
	for i, t := range ts {
		parts[i] = Key(t)
	}
	return strings.Join(parts, ",")
}

func (h *Hooks) substArgs(args []Type, m map[*TypeParam]Type) ([]Type, bool) {
	changed := false
	out := make([]Type, len(args))
	for i, a := range args {
		out[i] = h.Subst(a, m)
		if out[i] != a {
			changed = true
		}
	}
	return out, changed
}

// Assoc is an associated-type projection `Base::Name` (D27) whose base is
// still generic (a type parameter or a trait's Self). Once the base becomes
// concrete, Subst resolves the projection through AssocResolver.
type Assoc struct {
	Base  Type
	Trait *Trait
	Name  string
}

func (a *Assoc) String() string { return a.Base.String() + "." + a.Name }

// ResolveAssoc resolves a projection whose base is concrete.
func (h *Hooks) ResolveAssoc(a *Assoc) Type {
	if _, bare := a.Base.(*TypeParam); bare || h == nil || h.AssocResolver == nil {
		return a
	}
	if r := h.AssocResolver(a.Base, a.Trait, a.Name); r != nil {
		return r
	}
	return a
}

// Map is `Map<K, V>` / `MutableMap<K, V>`; Set is `Set<T>` / `MutableSet<T>`
// (D25: insertion-ordered, reference types with an immutable/mutable split).
type Map struct {
	Key, Value Type
	Mutable    bool
	Alias      string
}

func (m *Map) String() string {
	if m.Alias != "" {
		return m.Alias
	}
	name := "Map"
	if m.Mutable {
		name = "MutableMap"
	}
	return name + "<" + m.Key.String() + ", " + m.Value.String() + ">"
}

type Set struct {
	Elem    Type
	Mutable bool
	Alias   string
}

func (s *Set) String() string {
	if s.Alias != "" {
		return s.Alias
	}
	name := "Set"
	if s.Mutable {
		name = "MutableSet"
	}
	return name + "<" + s.Elem.String() + ">"
}

// Channel is `Channel<T>` (D16); Task is the handle of a launched task
// (D3) whose completion yields Result.
type Channel struct {
	Elem  Type
	Alias string
}

func (c *Channel) String() string {
	if c.Alias != "" {
		return c.Alias
	}
	return "Channel<" + c.Elem.String() + ">"
}

type Task struct {
	Result Type
	Alias  string
}

func (t *Task) String() string {
	if t.Alias != "" {
		return t.Alias
	}
	return "Task<" + t.Result.String() + ">"
}

// ---------------------------------------------------------------------------
// type aliases (display only)

// Aliased returns t as written through a `type` alias: a shallow copy of a
// structural type carrying name as its display name. Identity, keys and
// codegen see the structure; only String shows the alias. Named types —
// structs, sealed types, traits, type parameters — keep their own name and
// are returned unchanged.
func Aliased(t Type, name string) Type {
	switch t := t.(type) {
	case *Basic:
		c := *t
		c.Alias = name
		return &c
	case *Pointer:
		c := *t
		c.Alias = name
		return &c
	case *Nullable:
		c := *t
		c.Alias = name
		return &c
	case *Tuple:
		c := *t
		c.Alias = name
		return &c
	case *Range:
		c := *t
		c.Alias = name
		return &c
	case *List:
		c := *t
		c.Alias = name
		return &c
	case *Map:
		c := *t
		c.Alias = name
		return &c
	case *Set:
		c := *t
		c.Alias = name
		return &c
	case *Channel:
		c := *t
		c.Alias = name
		return &c
	case *Task:
		c := *t
		c.Alias = name
		return &c
	case *Func:
		c := *t
		c.Alias = name
		return &c
	case *ErrorUnion:
		c := *t
		c.Alias = name
		return &c
	}
	return t
}

// AliasOf returns the display alias t was written through, or "".
func AliasOf(t Type) string {
	switch t := t.(type) {
	case *Basic:
		return t.Alias
	case *Pointer:
		return t.Alias
	case *Nullable:
		return t.Alias
	case *Tuple:
		return t.Alias
	case *Range:
		return t.Alias
	case *List:
		return t.Alias
	case *Map:
		return t.Alias
	case *Set:
		return t.Alias
	case *Channel:
		return t.Alias
	case *Task:
		return t.Alias
	case *Func:
		return t.Alias
	case *ErrorUnion:
		return t.Alias
	}
	return ""
}

// Unaliased strips display aliases: one level when deep is false (the
// alias's own definition, with its parts as written), every level when deep
// is true (the fully expanded type).
func Unaliased(t Type, deep bool) Type {
	if t == nil {
		return nil
	}
	if !deep {
		return Aliased(t, "")
	}
	switch t := t.(type) {
	case *Basic:
		return Aliased(t, "")
	case *Pointer:
		return &Pointer{Elem: Unaliased(t.Elem, true), Raw: t.Raw}
	case *Nullable:
		return &Nullable{Elem: Unaliased(t.Elem, true)}
	case *Tuple:
		out := make([]Type, len(t.Elems))
		for i, e := range t.Elems {
			out[i] = Unaliased(e, true)
		}
		return &Tuple{Elems: out}
	case *Range:
		return &Range{Elem: Unaliased(t.Elem, true)}
	case *List:
		return &List{Elem: Unaliased(t.Elem, true), Mutable: t.Mutable}
	case *Map:
		return &Map{Key: Unaliased(t.Key, true), Value: Unaliased(t.Value, true), Mutable: t.Mutable}
	case *Set:
		return &Set{Elem: Unaliased(t.Elem, true), Mutable: t.Mutable}
	case *Channel:
		return &Channel{Elem: Unaliased(t.Elem, true)}
	case *Task:
		return &Task{Result: Unaliased(t.Result, true)}
	case *Func:
		c := *t
		c.Alias = ""
		c.Ret = Unaliased(t.Ret, true)
		c.Effects.Error = Unaliased(t.Effects.Error, true)
		c.Params = make([]Param, len(t.Params))
		for i, p := range t.Params {
			c.Params[i] = Param{Name: p.Name, Type: Unaliased(p.Type, true), HasDefault: p.HasDefault, Variadic: p.Variadic}
		}
		return &c
	case *ErrorUnion:
		out := make([]Type, len(t.Members))
		for i, m := range t.Members {
			out[i] = Unaliased(m, true)
		}
		return &ErrorUnion{Members: out}
	}
	return t
}

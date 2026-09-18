package sema

import (
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

type SymbolKind int

const (
	SymType SymbolKind = iota // struct, sealed, trait, primitive
	SymFunc
	SymGlobal
	SymLocal
	SymModule
	SymVariantCtor // prelude Some / None / Ok / Err
)

type Symbol struct {
	Name   string
	Kind   SymbolKind
	Pub    bool
	Module *Module
	Span   source.Span

	Type   types.Type    // SymType
	Alias  *errorAlias   // SymType naming an error set (`error X = A | B`); Type is filled on first use
	Func   *FuncTemplate // SymFunc
	Global *Global       // SymGlobal
	Var    *Var          // SymLocal
	Mod    *Module       // SymModule
}

// Scope is a lexical symbol table.
type Scope struct {
	parent  *Scope
	symbols map[string]*Symbol
	// module is set on module-level scopes so lookups can enforce `pub`.
	module *Module
}

func NewScope(parent *Scope) *Scope {
	return &Scope{parent: parent, symbols: map[string]*Symbol{}}
}

func (s *Scope) Insert(sym *Symbol) *Symbol {
	if old, ok := s.symbols[sym.Name]; ok {
		return old
	}
	s.symbols[sym.Name] = sym
	return nil
}

// LookupLocal finds a name in this scope only.
func (s *Scope) LookupLocal(name string) *Symbol {
	return s.symbols[name]
}

// Lookup walks outward through enclosing scopes.
func (s *Scope) Lookup(name string) *Symbol {
	for sc := s; sc != nil; sc = sc.parent {
		if sym, ok := sc.symbols[name]; ok {
			return sym
		}
	}
	return nil
}

// Module is a directory of source files sharing one namespace (M2).
type Module struct {
	Path    string // slash-separated path relative to the package root; "" for root
	Dir     string // filesystem directory
	Std     bool
	Files   []*ast.File
	Scope   *Scope               // module-level declarations
	Imports map[*ast.File]*Scope // per-file import scopes, parent = module scope
	Deps    []*Module
	Pkg     *Package
	Uses    map[*ast.UseDecl]*Module // resolved imports, filled by the loader
	state   int                      // 0 unloaded, 1 loading, 2 loaded
}

func (m *Module) Name() string {
	if m.Path == "" {
		return "main"
	}
	for i := len(m.Path) - 1; i >= 0; i-- {
		if m.Path[i] == '/' {
			return m.Path[i+1:]
		}
	}
	return m.Path
}

// Mangle prefix for symbols in this module.
func (m *Module) prefix() string {
	if m.Path == "" {
		return "main"
	}
	out := make([]byte, 0, len(m.Path))
	for i := 0; i < len(m.Path); i++ {
		if m.Path[i] == '/' {
			out = append(out, '.')
		} else {
			out = append(out, m.Path[i])
		}
	}
	return string(out)
}

// FuncTemplate is a function declaration before instantiation. Non-generic
// functions have exactly one instance.
type FuncTemplate struct {
	Name       string
	Mangled    string
	Module     *Module
	File       *ast.File
	Decl       *ast.FunDecl
	Pub        bool
	Extern     bool
	TypeParams []*types.TypeParam
	Sig        *types.Func
	// Owner is the struct (template) an inherent method belongs to.
	Owner *types.Struct
	// Impl is set for trait-impl methods.
	Impl *Impl
	// Trait is set for a default-bodied trait method.
	Trait *types.Trait

	Instances map[string]*Func
	Attrs     map[string]*ast.Attribute

	// Inferred error type for `throws` without a declared type, refined
	// across rounds until it reaches a fixpoint (D45).
	InferError types.Type
	InferDone  bool
	// InferRet marks `fun f(...) = expr` with no declared return type; the
	// type is inferred from the body when the function is instantiated.
	InferRet bool
}

// Impl records `impl Trait for Type` (D17: one per pair program-wide).
type Impl struct {
	Trait      *types.Trait
	Target     types.Type // may contain the impl's own type params
	TypeParams []*types.TypeParam
	Methods    map[string]*FuncTemplate
	AssocTypes map[string]types.Type
	// ImplicitError: the trait's `Error` is not bound by a `type Error =`
	// line but is the union of what the impl's methods throw (D40, v0.24).
	ImplicitError bool
	Module     *Module
	Decl       *ast.ImplDecl
}

// Doc is the module's documentation: the top-of-file doc comments of its
// files, in file order, joined by blank lines.
func (m *Module) Doc() string {
	var parts []string
	for _, f := range m.Files {
		if f.Doc != "" {
			parts = append(parts, f.Doc)
		}
	}
	return strings.Join(parts, "\n\n")
}

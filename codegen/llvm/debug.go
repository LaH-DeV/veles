package llvm

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/LaH-DeV/veles/sema"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// Debug information (plan S8): in a debug build every function the program
// declares gets a DISubprogram, every instruction the line and column of
// the statement (or call) it comes from, and every local variable and
// parameter a DILocalVariable with its type, so gdb, lldb and an editor's
// debugger stop at a Veles `file:line`, step, and print locals. DWARF on
// every target this compiler emits for (Windows is x86_64-w64-windows-gnu,
// whose debuggers read DWARF too).

type debugInfo struct {
	nodes []string       // the metadata, `!i = nodes[i]`
	files map[string]int // a path's DIFile
	types map[string]int // a type's DIType, by types.Key
	locs  map[string]int // "scope:line:col" to its DILocation
	cu    int
}

func newDebugInfo() *debugInfo {
	return &debugInfo{files: map[string]int{}, types: map[string]int{}, locs: map[string]int{}, cu: -1}
}

// node adds a metadata node and returns its number.
func (d *debugInfo) node(text string) int {
	d.nodes = append(d.nodes, text)
	return len(d.nodes) - 1
}

func (d *debugInfo) file(f *source.File) int {
	if id, ok := d.files[f.Path]; ok {
		return id
	}
	path := f.Path
	if abs, err := filepath.Abs(path); err == nil && !f.Embedded {
		path = abs
	}
	dir := filepath.Dir(path)
	if f.Embedded {
		// std is compiled into the tool, not read from a directory: a
		// debugger names the file, and cannot show it (checklist §11)
		dir = "<veles std>"
	}
	id := d.node(fmt.Sprintf("!DIFile(filename: %s, directory: %s)", mdString(filepath.Base(path)), mdString(dir)))
	d.files[f.Path] = id
	if d.cu < 0 {
		d.cu = d.node(fmt.Sprintf("distinct !DICompileUnit(language: DW_LANG_C, file: !%d, producer: \"veles\", isOptimized: false, runtimeVersion: 0, emissionKind: FullDebug)", id))
	}
	return id
}

// write appends the module's debug metadata.
func (d *debugInfo) write(sb *strings.Builder) {
	if d.cu < 0 {
		return // nothing with a position was compiled
	}
	version := d.node("!{i32 7, !\"Dwarf Version\", i32 4}")
	info := d.node("!{i32 2, !\"Debug Info Version\", i32 3}")
	fmt.Fprintf(sb, "\n!llvm.dbg.cu = !{!%d}\n!llvm.module.flags = !{!%d, !%d}\n", d.cu, version, info)
	for i, n := range d.nodes {
		fmt.Fprintf(sb, "!%d = %s\n", i, n)
	}
}

// mdString quotes text as a metadata string.
func mdString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < 0x20 || c >= 0x7f || c == '"' || c == '\\' {
			fmt.Fprintf(&b, "\\%02X", c)
		} else {
			b.WriteByte(c)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// debugFunction opens fn's subprogram: from here on what is emitted is at
// fn's line, until a statement moves it.
func (g *gen) debugFunction(fn *sema.Func) {
	g.dbgSP, g.dbgLoc = -1, ""
	if g.dbg == nil || !fn.Span.IsValid() {
		return
	}
	d := g.dbg
	file := d.file(fn.Span.File)
	line, _ := fn.Span.File.Position(fn.Span.Start)
	ty := d.node("!DISubroutineType(types: !{})")
	g.dbgSP = d.node(fmt.Sprintf("distinct !DISubprogram(name: %s, scope: !%d, file: !%d, line: %d, type: !%d, scopeLine: %d, spFlags: DISPFlagDefinition, unit: !%d)",
		mdString(fn.Display), file, file, line, ty, line, d.cu))
	g.debugAt(fn.Span)
}

// debugAt makes sp the position of what is emitted next in the function.
func (g *gen) debugAt(sp source.Span) {
	if g.dbgSP < 0 || !sp.IsValid() {
		return
	}
	line, col := sp.File.Position(sp.Start)
	key := strconv.Itoa(g.dbgSP) + ":" + strconv.Itoa(line) + ":" + strconv.Itoa(col)
	id, ok := g.dbg.locs[key]
	if !ok {
		id = g.dbg.node(fmt.Sprintf("!DILocation(line: %d, column: %d, scope: !%d)", line, col, g.dbgSP))
		g.dbg.locs[key] = id
	}
	g.dbgLoc = fmt.Sprintf(", !dbg !%d", id)
}

// debugDefine is what follows a function's attributes in its `define`.
func (g *gen) debugDefine() string {
	if g.dbgSP < 0 {
		return ""
	}
	return fmt.Sprintf(" !dbg !%d", g.dbgSP)
}

// stmtPos is where a statement was written, or an invalid span.
func stmtPos(s sema.Stmt) source.Span {
	switch s := s.(type) {
	case *sema.VarDecl:
		return s.Pos
	case *sema.Assign:
		return s.Pos
	case *sema.ExprStmt:
		return s.Pos
	case *sema.Return:
		return s.Pos
	case *sema.Loop:
		return s.Pos
	case *sema.Break:
		return s.Pos
	case *sema.Continue:
		return s.Pos
	case *sema.With:
		return s.Pos
	}
	return source.Span{}
}

// debugVar declares a local variable or a parameter (arg > 0) living at
// storage; byRef: the storage holds a pointer to the value (a heap cell).
func (g *gen) debugVar(v *sema.Var, storage string, arg int, byRef bool) {
	if g.dbgSP < 0 || v.Name == "" || strings.HasPrefix(v.Name, "$") || !v.Span.IsValid() || types.IsUnit(v.Type) {
		return
	}
	d := g.dbg
	file := d.file(v.Span.File)
	line, _ := v.Span.File.Position(v.Span.Start)
	argField := ""
	if arg > 0 {
		argField = fmt.Sprintf(", arg: %d", arg)
	}
	variable := d.node(fmt.Sprintf("!DILocalVariable(name: %s%s, scope: !%d, file: !%d, line: %d, type: !%d)", mdString(v.Name), argField, g.dbgSP, file, line, g.debugType(v.Type)))
	expr := "!DIExpression()"
	if byRef {
		expr = "!DIExpression(DW_OP_deref)"
	}
	g.emit("call void @llvm.dbg.declare(metadata ptr %s, metadata !%d, metadata %s)", storage, variable, expr)
}

// debugParam declares parameter number arg (1 is the receiver when there is
// one) where the prologue put it; the receiver and a parameter in a heap
// cell are reached through the pointer their storage holds.
func (g *gen) debugParam(v *sema.Var, arg int) {
	st, ok := g.storage[v]
	if !ok || v.Captured {
		return
	}
	g.debugVar(v, st, arg, v.AddrTaken || v.IsSelf)
}

// debugType describes t to the debugger: numbers and bools as base types,
// a string as its bytes and length, a struct with its fields at their
// offsets, a pointer to what it points at; the rest by name and size.
func (g *gen) debugType(t types.Type) int {
	d := g.dbg
	key := types.Key(t)
	if id, ok := d.types[key]; ok {
		return id
	}
	size, align := g.layout(t)
	base := func(name string, bits int, enc string) int {
		return d.node(fmt.Sprintf("!DIBasicType(name: %s, size: %d, encoding: %s)", mdString(name), bits, enc))
	}
	var id int
	switch tt := t.(type) {
	case *types.Basic:
		switch {
		case tt.Kind == types.Bool:
			id = base("bool", 8, "DW_ATE_boolean")
		case types.IsFloat(t):
			id = base(t.String(), size*8, "DW_ATE_float")
		case tt.Kind == types.U8:
			id = base("u8", 8, "DW_ATE_unsigned_char")
		case types.IsSigned(t):
			id = base(t.String(), size*8, "DW_ATE_signed")
		case types.IsInteger(t):
			id = base(t.String(), size*8, "DW_ATE_unsigned")
		case tt.Kind == types.String:
			// {data, len}: gdb prints the bytes through the pointer
			id = d.node("") // placed first: the members refer to it
			d.types[key] = id
			bytes := d.node(fmt.Sprintf("!DIDerivedType(tag: DW_TAG_pointer_type, baseType: !%d, size: 64)", g.debugType(types.TU8)))
			data := d.node(fmt.Sprintf("!DIDerivedType(tag: DW_TAG_member, name: \"data\", scope: !%d, baseType: !%d, size: 64, offset: 0)", id, bytes))
			length := d.node(fmt.Sprintf("!DIDerivedType(tag: DW_TAG_member, name: \"len\", scope: !%d, baseType: !%d, size: 64, offset: 64)", id, g.debugType(types.TI64)))
			d.nodes[id] = fmt.Sprintf("!DICompositeType(tag: DW_TAG_structure_type, name: \"string\", size: 128, elements: !{!%d, !%d})", data, length)
			return id
		default:
			id = d.node(fmt.Sprintf("!DICompositeType(tag: DW_TAG_structure_type, name: %s, size: %d)", mdString(t.String()), size*8))
		}
	case *types.Enum:
		id = g.debugType(tt.Base)
	case *types.Pointer:
		id = d.node("")
		d.types[key] = id
		d.nodes[id] = fmt.Sprintf("!DIDerivedType(tag: DW_TAG_pointer_type, baseType: !%d, size: 64)", g.debugType(tt.Elem))
		return id
	case *types.Struct:
		if tt.Union || len(tt.TypeParams) > 0 && len(tt.TypeArgs) == 0 {
			id = d.node(fmt.Sprintf("!DICompositeType(tag: DW_TAG_structure_type, name: %s, size: %d)", mdString(t.String()), size*8))
			break
		}
		id = d.node("")
		d.types[key] = id
		plan := g.layouter().StructPlan(tt)
		var members []string
		for i, f := range tt.Fields {
			if types.IsUnit(f.Type) {
				continue
			}
			fs, _ := g.layout(f.Type)
			members = append(members, fmt.Sprintf("!%d", d.node(fmt.Sprintf("!DIDerivedType(tag: DW_TAG_member, name: %s, scope: !%d, baseType: !%d, size: %d, offset: %d)",
				mdString(f.Name), id, g.debugType(f.Type), fs*8, plan.Offsets[i]*8))))
		}
		d.nodes[id] = fmt.Sprintf("!DICompositeType(tag: DW_TAG_structure_type, name: %s, size: %d, align: %d, elements: !{%s})", mdString(t.String()), size*8, align*8, strings.Join(members, ", "))
		return id
	default:
		// a list, a map, a closure, a sealed or nullable value: named, sized,
		// its parts not described yet (checklist §11)
		id = d.node(fmt.Sprintf("!DICompositeType(tag: DW_TAG_structure_type, name: %s, size: %d)", mdString(t.String()), size*8))
	}
	d.types[key] = id
	return id
}

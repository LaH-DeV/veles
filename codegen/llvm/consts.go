package llvm

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/LaH-DeV/veles/sema"
	"github.com/LaH-DeV/veles/types"
)

// Constant tables (D113 part 2): a `const` List, Map or Set is laid out in
// the binary as the runtime's own structures — a header, the element (or
// entry) arrays and, for a map, its metadata and hash index — all LLVM
// `constant`, so they cost nothing at start-up, exist once however often
// they are used, and a write to one faults instead of changing a constant.
// The collector never sees them: span_of finds no heap span for their
// addresses, and they point only at other constants.

// constTable returns the global holding a constant table, emitting it on
// first use: named after its constant (`@const.main.PRIMES`), a table inside
// it after its place there (`@const.main.NESTED.1`).
func (g *gen) constTable(v sema.ConstVal, name string) string {
	if existing, ok := g.consts[v]; ok {
		return existing
	}
	if name == "" {
		name = fmt.Sprintf("anon.%d", len(g.consts))
	}
	base := constNameRe.ReplaceAllString(name, "_") // an LLVM name without quotes
	name = "@const." + base
	g.consts[v] = name
	switch v := v.(type) {
	case *sema.CList:
		data := g.constArray(name+".data", base+".", v.T.Elem, v.Elems)
		size, _ := g.layout(v.T.Elem)
		fmt.Fprintf(&g.constOut, "%s = private unnamed_addr constant %s { ptr %s, i64 %d, i64 %d, i64 %d, ptr %s, i64 0, i64 0 }\n",
			name, listHeader, data, len(v.Elems), len(v.Elems), size, g.arrayDescOf(v.T.Elem))
	case *sema.CArray:
		// the array itself, inline: a use reads it in place or loads it
		g.constArrayAs(name, base+".", v.T.Elem, v.Elems)
	case *sema.CMap:
		g.constMap(name, base, v.T.Key, v.T.Value, v.Keys, v.Vals)
	case *sema.CSet:
		g.constMap(name, base, v.T.Elem, nil, v.Elems, nil)
	default:
		panic(fmt.Sprintf("constTable: %T", v))
	}
	return name
}

// emptyList, emptyMap: the one empty immutable List / Map / Set of an element
// (key, value) type, a constant like the tables above. An immutable
// collection never changes, so every `[]` of that type can be the same
// object: it costs no allocation at all (D187). Its storage pointers are null
// and its counts zero, which is how the runtime already sees an empty one.
func (g *gen) emptyList(elem types.Type) string {
	desc := g.arrayDescOf(elem)
	key := "list " + desc
	if name, ok := g.emptyShared[key]; ok {
		return name
	}
	name := fmt.Sprintf("@const.empty.list.%d", len(g.emptyShared))
	g.emptyShared[key] = name
	size, _ := g.layout(elem)
	fmt.Fprintf(&g.constOut, "%s = private unnamed_addr constant %s { ptr null, i64 0, i64 0, i64 %d, ptr %s, i64 0, i64 0 }\n",
		name, listHeader, size, desc)
	return name
}

// emptyMap is the shared empty Map (vt non-nil) or Set (vt nil).
func (g *gen) emptyMap(kt, vt types.Type) string {
	kd, vd, valSize := g.arrayDescOf(kt), "null", 0
	if vt != nil {
		vd = g.arrayDescOf(vt)
		valSize, _ = g.layout(vt)
	}
	key := "map " + kd + " " + vd
	if name, ok := g.emptyShared[key]; ok {
		return name
	}
	name := fmt.Sprintf("@const.empty.map.%d", len(g.emptyShared))
	g.emptyShared[key] = name
	keySize, _ := g.layout(kt)
	fmt.Fprintf(&g.constOut, "%s = private unnamed_addr constant %s { ptr null, ptr null, ptr null, ptr %s, ptr %s, ptr null, i64 0, i64 0, i64 0, i64 0, i64 %d, i64 %d, i64 0, i64 0 }\n",
		name, mapHeader, kd, vd, keySize, valSize)
	return name
}

// constArray emits an array of constants and returns its name; a table
// among them is named at + its index.
func (g *gen) constArray(name, at string, elem types.Type, xs []sema.ConstVal) string {
	et := g.llType(elem)
	parts := make([]string, len(xs))
	for i, x := range xs {
		parts[i] = et + " " + g.constInit(x, at+strconv.Itoa(i))
	}
	body := "zeroinitializer"
	if len(parts) > 0 {
		body = "[" + strings.Join(parts, ", ") + "]"
	}
	fmt.Fprintf(&g.constOut, "%s = private unnamed_addr constant [%d x %s] %s\n", name, len(xs), et, body)
	return name
}

// constArrayAs is constArray under a name that is already decided.
func (g *gen) constArrayAs(name, at string, elem types.Type, xs []sema.ConstVal) {
	g.constArray(name, at, elem, xs)
}

// mapHeader is veles_map as veles_rt.c lays it out: keys, vals, meta,
// keyDesc, valDesc, index, icap, len, used, cap, keySize, valSize, mods.
// constNameRe matches what an unquoted LLVM global name cannot hold.
var constNameRe = regexp.MustCompile(`[^A-Za-z0-9._]`)

const mapHeader = "{ ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 }" // ..., mods, flags (D188)

// constMap emits a map (vt nil: a set) with its entries in order and an
// open-addressing index with at least half its slots free, as
// veles_map_insert keeps it. The hashes are keyed by a secret drawn at
// start-up (veles_rt.c, against hash flooding), so the compiler cannot know
// them: the metadata and the index are writable globals that
// veles_init_const_maps fills before the program's first instruction. They
// are internal — no program can name them — so only the header and the
// entries stay `constant` and fault when written.
func (g *gen) constMap(name, base string, kt, vt types.Type, keys, vals []sema.ConstVal) {
	n := len(keys)
	icap := 16
	for n*2 >= icap {
		icap *= 2
	}
	meta := make([]string, n)
	for e := range keys {
		meta[e] = "{ i64, i8 } { i64 0, i8 1 }"
	}
	if len(g.constMaps) == 0 {
		g.constOut.WriteString("declare void @veles_map_const_hash(ptr, i64, i64)\ndeclare void @veles_map_const_index(ptr)\n")
	}
	g.constMaps = append(g.constMaps, constMapRef{name, kt, n})
	keyArr := g.constArray(name+".keys", base+".k", kt, keys)
	valArr, valDesc, valSize := name+".vals", "null", 0
	if vt != nil {
		g.constArray(valArr, base+".v", vt, vals)
		valDesc = g.arrayDescOf(vt)
		valSize, _ = g.layout(vt)
	} else {
		fmt.Fprintf(&g.constOut, "%s = private unnamed_addr constant [1 x i8] zeroinitializer\n", valArr)
	}
	metaBody := "zeroinitializer"
	if n > 0 {
		metaBody = "[" + strings.Join(meta, ", ") + "]"
	}
	fmt.Fprintf(&g.constOut, "%s.meta = internal global [%d x { i64, i8 }] %s\n", name, n, metaBody)
	fmt.Fprintf(&g.constOut, "%s.index = internal global [%d x i64] zeroinitializer\n", name, icap)
	keySize, _ := g.layout(kt)
	fmt.Fprintf(&g.constOut, "%s = private unnamed_addr constant %s { ptr %s, ptr %s, ptr %s.meta, ptr %s, ptr %s, ptr %s.index, i64 %d, i64 %d, i64 %d, i64 %d, i64 %d, i64 %d, i64 0, i64 0 }\n",
		name, mapHeader, keyArr, valArr, name, g.arrayDescOf(kt), valDesc, name, icap, n, n, n, keySize, valSize)
}

// constMapRef is a constant Map or Set whose hashes veles_init_const_maps fills.
type constMapRef struct {
	name string
	key  types.Type
	n    int
}

// constMapsInit defines veles_init_const_maps, which hashes every constant
// map's keys with this run's key and builds its index. The entry point calls
// it first, so it must already know every constant map: one first emitted
// after it would never be hashed.
func (g *gen) constMapsInit() {
	if len(g.constMaps) != g.constAtEntry {
		panic("codegen: a constant map was emitted after the entry point")
	}
	if len(g.constMaps) == 0 {
		return
	}
	g.defineHelper("veles_init_const_maps", "void", nil, func() {
		for _, cm := range g.constMaps {
			i := g.alloca("i64")
			g.emit("store i64 0, ptr %s", i)
			headL, bodyL, doneL := g.newLabel("cmap.head"), g.newLabel("cmap.body"), g.newLabel("cmap.done")
			g.emitTerm("br label %%%s", headL)
			g.placeLabel(headL)
			iv := g.newTmp()
			g.emit("%s = load i64, ptr %s", iv, i)
			more := g.newTmp()
			g.emit("%s = icmp slt i64 %s, %d", more, iv, cm.n)
			g.emitTerm("br i1 %s, label %%%s, label %%%s", more, bodyL, doneL)
			g.placeLabel(bodyL)
			kp := g.newTmp()
			g.emit("%s = call ptr @veles_map_key_at(ptr %s, i64 %s)", kp, cm.name, iv)
			h := g.hash(cm.key, g.loadVal(cm.key, kp))
			g.emit("call void @veles_map_const_hash(ptr %s, i64 %s, i64 %s)", cm.name, iv, h)
			next := g.newTmp()
			g.emit("%s = add i64 %s, 1", next, iv)
			g.emit("store i64 %s, ptr %s", next, i)
			g.emitTerm("br label %%%s", headL)
			g.placeLabel(doneL)
			g.emit("call void @veles_map_const_index(ptr %s)", cm.name)
		}
		g.emitTerm("ret void")
	})
}

// constInit is a constant as an LLVM constant of its type's representation;
// a table in it is named at (and at.i for its parts' tables).
func (g *gen) constInit(v sema.ConstVal, at string) string {
	switch v := v.(type) {
	case *sema.CInt:
		return v.V.String()
	case *sema.CFloat:
		return floatConst(v.V)
	case *sema.CBool:
		if v.V {
			return "true"
		}
		return "false"
	case *sema.CString:
		return g.stringConst(v.V)
	case *sema.CUnit:
		return "zeroinitializer"
	case *sema.CNull:
		if isPtrLike(v.T.Elem) {
			return "null"
		}
		return "zeroinitializer"
	case *sema.CSome:
		if isPtrLike(v.T.Elem) {
			return g.constInit(v.V, at)
		}
		return fmt.Sprintf("{ i1 true, %s %s }", g.llType(v.T.Elem), g.constInit(v.V, at))
	case *sema.CTuple:
		return g.constFields(v.T.Elems, v.Elems, at)
	case *sema.CStruct:
		ts := make([]types.Type, len(v.T.Fields))
		for i, f := range v.T.Fields {
			ts[i] = f.Type
		}
		return g.constFields(ts, v.Fields, at)
	case *sema.CArray:
		// inline in what holds it: `[N x T] [...]` (D121)
		parts := make([]string, len(v.Elems))
		et := g.llType(v.T.Elem)
		for i, x := range v.Elems {
			parts[i] = et + " " + g.constInit(x, at+"."+strconv.Itoa(i))
		}
		if len(parts) == 0 {
			return "zeroinitializer"
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case *sema.CList, *sema.CMap, *sema.CSet:
		return g.constTable(v, at)
	}
	panic(fmt.Sprintf("constInit: %T", v))
}

func (g *gen) constFields(ts []types.Type, xs []sema.ConstVal, at string) string {
	if len(xs) == 0 {
		return "zeroinitializer"
	}
	parts := make([]string, len(xs))
	for i, x := range xs {
		parts[i] = g.llType(ts[i]) + " " + g.constInit(x, at+"."+strconv.Itoa(i))
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

// stackListHeader builds the header of a mutable list in a slot of the
// current function's frame, the way veles_list_new would leave it for a
// list with no storage yet (D188). The slot is reached once per execution of
// the declaration, so the fields are written there each time.
func (g *gen) stackListHeader(elem types.Type) string {
	hdr := g.alloca(listHeader)
	size, _ := g.layout(elem)
	g.emit("store %s zeroinitializer, ptr %s", listHeader, hdr)
	es, ds := g.newTmp(), g.newTmp()
	g.emit("%s = getelementptr inbounds %s, ptr %s, i32 0, i32 3", es, listHeader, hdr)
	g.emit("store i64 %d, ptr %s", size, es)
	g.emit("%s = getelementptr inbounds %s, ptr %s, i32 0, i32 4", ds, listHeader, hdr)
	g.emit("store ptr %s, ptr %s", g.arrayDescOf(elem), ds)
	return hdr
}

// stackMapHeader is stackListHeader for a map (vt nil: a set).
func (g *gen) stackMapHeader(kt, vt types.Type) string {
	hdr := g.alloca(mapHeader)
	g.emit("store %s zeroinitializer, ptr %s", mapHeader, hdr)
	keySize, _ := g.layout(kt)
	valSize, valDesc := 0, "null"
	if vt != nil {
		valSize, _ = g.layout(vt)
		valDesc = g.arrayDescOf(vt)
	}
	for _, f := range []struct {
		idx int
		ty  string
		val string
	}{
		{3, "ptr", g.arrayDescOf(kt)},
		{4, "ptr", valDesc},
		{10, "i64", fmt.Sprint(keySize)},
		{11, "i64", fmt.Sprint(valSize)},
	} {
		p := g.newTmp()
		g.emit("%s = getelementptr inbounds %s, ptr %s, i32 0, i32 %d", p, mapHeader, hdr, f.idx)
		g.emit("store %s %s, ptr %s", f.ty, f.val, p)
	}
	return hdr
}

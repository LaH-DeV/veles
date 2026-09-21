package llvm

import (
	"fmt"

	"github.com/LaH-DeV/veles/sema"
	"github.com/LaH-DeV/veles/types"
)

const mapDecls = `declare ptr @veles_map_new(i64, i64)
declare i64 @veles_map_len(ptr)
declare i64 @veles_map_find(ptr, i64, ptr, ptr)
declare i64 @veles_map_insert(ptr, i64, ptr, ptr, ptr)
declare i1 @veles_map_remove(ptr, i64, ptr, ptr)
declare ptr @veles_map_key_at(ptr, i64)
declare ptr @veles_map_val_at(ptr, i64)
declare i64 @veles_map_used(ptr)
declare i1 @veles_map_live(ptr, i64)
declare void @veles_map_clear(ptr)
declare ptr @veles_map_copy(ptr)
declare ptr @veles_map_keys(ptr)
declare ptr @veles_map_values(ptr)
declare ptr @veles_map_entries(ptr, ptr, i64, i64)
declare i64 @veles_hash_bytes(ptr, i64)
declare i64 @veles_hash_mix(i64, i64)
`

// keyValTypes returns the key and value types of a map or set.
func keyValTypes(t types.Type) (types.Type, types.Type) {
	switch t := t.(type) {
	case *types.Map:
		return t.Key, t.Value
	case *types.Set:
		return t.Elem, types.TUnit
	}
	panic("not a map")
}

func (g *gen) sizeOf(t types.Type) int {
	s, _ := g.layout(t)
	return s
}

// keyArgs evaluates a key and returns (hash, pointer-to-key, eq function).
func (g *gen) keyArgs(kt types.Type, key string) (string, string, string) {
	h := g.hash(kt, key)
	kp := g.alloca(g.llType(kt))
	g.emit("store %s %s, ptr %s", g.llType(kt), key, kp)
	return h, kp, "@" + g.eqPtrHelper(kt)
}

func (g *gen) mapLit(e *sema.MapLit) string {
	mt := e.Type().(*types.Map)
	m := g.newTmp()
	g.emit("%s = call ptr @veles_map_new(ptr %s, ptr %s)", m, g.arrayDescOf(mt.Key), g.arrayDescOf(mt.Value))
	for _, en := range e.Entries {
		k := g.expr(en[0])
		v := g.expr(en[1])
		h, kp, eq := g.keyArgs(mt.Key, k)
		vp := g.alloca(g.llType(mt.Value))
		g.emit("store %s %s, ptr %s", g.llType(mt.Value), v, vp)
		g.emit("call i64 @veles_map_insert(ptr %s, i64 %s, ptr %s, ptr %s, ptr %s)", m, h, kp, vp, eq)
	}
	return m
}

func (g *gen) mapBuiltin(e *sema.Builtin) (string, bool) {
	switch e.Op {
	case "set.new":
		st := e.Type().(*types.Set)
		m := g.newTmp()
		g.emit("%s = call ptr @veles_map_new(ptr %s, ptr null)", m, g.arrayDescOf(st.Elem))
		return m, true
	case "map.len":
		m := g.expr(e.Args[0])
		v := g.newTmp()
		g.emit("%s = call i64 @veles_map_len(ptr %s)", v, m)
		return v, true
	case "map.get":
		kt, vt := keyValTypes(e.Args[0].Type())
		m := g.expr(e.Args[0])
		k := g.expr(e.Args[1])
		h, kp, eq := g.keyArgs(kt, k)
		idx := g.newTmp()
		g.emit("%s = call i64 @veles_map_find(ptr %s, i64 %s, ptr %s, ptr %s)", idx, m, h, kp, eq)
		found := g.newTmp()
		g.emit("%s = icmp sge i64 %s, 0", found, idx)
		nt := e.Type().(*types.Nullable)
		res := g.alloca(g.llType(nt))
		g.emit("store %s zeroinitializer, ptr %s", g.llType(nt), res)
		hit, end := g.newLabel("map.hit"), g.newLabel("map.end")
		g.emitTerm("br i1 %s, label %%%s, label %%%s", found, hit, end)
		g.placeLabel(hit)
		vp := g.newTmp()
		g.emit("%s = call ptr @veles_map_val_at(ptr %s, i64 %s)", vp, m, idx)
		val := g.newTmp()
		g.emit("%s = load %s, ptr %s", val, g.llType(vt), vp)
		g.emit("store %s %s, ptr %s", g.llType(nt), g.makeNullable(nt, "true", val), res)
		g.emitTerm("br label %%%s", end)
		g.placeLabel(end)
		out := g.newTmp()
		g.emit("%s = load %s, ptr %s", out, g.llType(nt), res)
		return out, true
	case "map.ref", "map.refOrPanic":
		// the value slot as a pointer: null when the key is absent (`map.ref`,
		// typed `(*V)?`) or a panic (`map.refOrPanic`, typed `*V`). The slot
		// stays valid until the map grows; the collector keeps the old
		// storage alive while a pointer into it exists (D10).
		kt, _ := keyValTypes(e.Args[0].Type())
		m := g.expr(e.Args[0])
		k := g.expr(e.Args[1])
		h, kp, eq := g.keyArgs(kt, k)
		idx := g.newTmp()
		g.emit("%s = call i64 @veles_map_find(ptr %s, i64 %s, ptr %s, ptr %s)", idx, m, h, kp, eq)
		found := g.newTmp()
		g.emit("%s = icmp sge i64 %s, 0", found, idx)
		if e.Op == "map.refOrPanic" {
			hit, miss := g.newLabel("map.hit"), g.newLabel("map.miss")
			g.emitTerm("br i1 %s, label %%%s, label %%%s", found, hit, miss)
			g.placeLabel(miss)
			msg := g.concat(g.concat(g.stringConst("key "), g.show(kt, k)), g.stringConst(" not found in map"))
			sp, sl := g.strPtrLen(msg)
			g.emit("call void @veles_panic(ptr %s, i64 %s)", sp, sl)
			g.emitTerm("unreachable")
			g.placeLabel(hit)
			vp := g.newTmp()
			g.emit("%s = call ptr @veles_map_val_at(ptr %s, i64 %s)", vp, m, idx)
			return vp, true
		}
		res := g.alloca("ptr")
		g.emit("store ptr null, ptr %s", res)
		hit, end := g.newLabel("map.hit"), g.newLabel("map.end")
		g.emitTerm("br i1 %s, label %%%s, label %%%s", found, hit, end)
		g.placeLabel(hit)
		vp := g.newTmp()
		g.emit("%s = call ptr @veles_map_val_at(ptr %s, i64 %s)", vp, m, idx)
		g.emit("store ptr %s, ptr %s", vp, res)
		g.emitTerm("br label %%%s", end)
		g.placeLabel(end)
		out := g.newTmp()
		g.emit("%s = load ptr, ptr %s", out, res)
		return out, true
	case "map.contains":
		kt, _ := keyValTypes(e.Args[0].Type())
		m := g.expr(e.Args[0])
		k := g.expr(e.Args[1])
		h, kp, eq := g.keyArgs(kt, k)
		idx := g.newTmp()
		g.emit("%s = call i64 @veles_map_find(ptr %s, i64 %s, ptr %s, ptr %s)", idx, m, h, kp, eq)
		v := g.newTmp()
		g.emit("%s = icmp sge i64 %s, 0", v, idx)
		return v, true
	case "map.set":
		kt, vt := keyValTypes(e.Args[0].Type())
		m := g.expr(e.Args[0])
		k := g.expr(e.Args[1])
		v := g.expr(e.Args[2])
		h, kp, eq := g.keyArgs(kt, k)
		vp := g.alloca(g.llType(vt))
		g.emit("store %s %s, ptr %s", g.llType(vt), v, vp)
		g.emit("call i64 @veles_map_insert(ptr %s, i64 %s, ptr %s, ptr %s, ptr %s)", m, h, kp, vp, eq)
		return "zeroinitializer", true
	case "set.add":
		kt, _ := keyValTypes(e.Args[0].Type())
		m := g.expr(e.Args[0])
		k := g.expr(e.Args[1])
		h, kp, eq := g.keyArgs(kt, k)
		before := g.newTmp()
		g.emit("%s = call i64 @veles_map_len(ptr %s)", before, m)
		g.emit("call i64 @veles_map_insert(ptr %s, i64 %s, ptr %s, ptr null, ptr %s)", m, h, kp, eq)
		after := g.newTmp()
		g.emit("%s = call i64 @veles_map_len(ptr %s)", after, m)
		v := g.newTmp()
		g.emit("%s = icmp ne i64 %s, %s", v, before, after)
		return v, true
	case "map.remove":
		kt, _ := keyValTypes(e.Args[0].Type())
		m := g.expr(e.Args[0])
		k := g.expr(e.Args[1])
		h, kp, eq := g.keyArgs(kt, k)
		v := g.newTmp()
		g.emit("%s = call i1 @veles_map_remove(ptr %s, i64 %s, ptr %s, ptr %s)", v, m, h, kp, eq)
		return v, true
	case "map.clear":
		m := g.expr(e.Args[0])
		g.emit("call void @veles_map_clear(ptr %s)", m)
		return "zeroinitializer", true
	case "map.copy":
		m := g.expr(e.Args[0])
		v := g.newTmp()
		g.emit("%s = call ptr @veles_map_copy(ptr %s)", v, m)
		return v, true
	case "map.keys", "map.values":
		m := g.expr(e.Args[0])
		v := g.newTmp()
		fn := "veles_map_keys"
		if e.Op == "map.values" {
			fn = "veles_map_values"
		}
		g.emit("%s = call ptr @%s(ptr %s)", v, fn, m)
		return v, true
	case "map.entries":
		kt, vt := keyValTypes(e.Args[0].Type())
		tt := &types.Tuple{Elems: []types.Type{kt, vt}}
		_, va := g.layout(vt)
		ks := g.sizeOf(kt)
		valOffset := (ks + va - 1) / va * va
		m := g.expr(e.Args[0])
		v := g.newTmp()
		g.emit("%s = call ptr @veles_map_entries(ptr %s, ptr %s, i64 %d, i64 %d)", v, m, g.arrayDescOf(tt), valOffset, g.sizeOf(tt))
		return v, true
	}
	return "", false
}

// ---------------------------------------------------------------------------
// hashing

// hash computes a 64-bit hash of a value consistent with structural
// equality.
func (g *gen) hash(t types.Type, v string) string {
	t = types.Underlying(t) // an enum hashes as its integer (D57)
	switch tt := t.(type) {
	case *types.Basic:
		out := g.newTmp()
		switch {
		case tt.Kind == types.String:
			p, l := g.strPtrLen(v)
			g.emit("%s = call i64 @veles_hash_bytes(ptr %s, i64 %s)", out, p, l)
		case tt.Kind == types.Bool:
			g.emit("%s = zext i1 %s to i64", out, v)
		case tt.Kind == types.F64:
			g.emit("%s = bitcast double %s to i64", out, v)
		case tt.Kind == types.F32:
			b := g.newTmp()
			g.emit("%s = bitcast float %s to i32", b, v)
			g.emit("%s = zext i32 %s to i64", out, b)
		case tt.Kind == types.Unit || tt.Kind == types.Never:
			return "0"
		default:
			llt := g.llType(t)
			if llt == "i64" {
				return v
			}
			if types.IsSigned(t) {
				g.emit("%s = sext %s %s to i64", out, llt, v)
			} else {
				g.emit("%s = zext %s %s to i64", out, llt, v)
			}
		}
		return out
	case *types.Pointer:
		out := g.newTmp()
		g.emit("%s = ptrtoint ptr %s to i64", out, v)
		return out
	}
	// collections hash over their elements, matching their `==` (v0.26)
	name := g.hashHelper(t)
	out := g.newTmp()
	g.emit("%s = call i64 @%s(%s %s)", out, name, g.llType(t), v)
	return out
}

func (g *gen) hashHelper(t types.Type) string {
	key := types.Key(t)
	if name, ok := g.hashFns[key]; ok {
		return name
	}
	name := "hash." + mangleType(t)
	g.hashFns[key] = name
	llt := g.llType(t)
	g.pending = append(g.pending, func() {
		g.defineHelper(name, "i64", []string{llt + " %v"}, func() {
			r := g.hashBody(t, "%v")
			g.emitTerm("ret i64 %s", r)
		})
	})
	return name
}

func (g *gen) mix(a, b string) string {
	out := g.newTmp()
	g.emit("%s = call i64 @veles_hash_mix(i64 %s, i64 %s)", out, a, b)
	return out
}

func (g *gen) hashBody(t types.Type, v string) string {
	if ops := g.custom(t); ops != nil && ops.Hash != nil {
		r := g.newTmp()
		g.emit("%s = call i64 @%s(%s)", r, ops.Hash.Name, g.recvOperand(ops.Hash, t, v))
		return r
	}
	llt := g.llType(t)
	switch tt := t.(type) {
	case *types.Nullable:
		if isPtrLike(tt.Elem) {
			if _, isPtr := tt.Elem.(*types.Pointer); isPtr {
				out := g.newTmp()
				g.emit("%s = ptrtoint ptr %s to i64", out, v)
				return out
			}
			// a nullable collection: 0 for null, else the collection's hash
			res := g.alloca("i64")
			g.emit("store i64 0, ptr %s", res)
			isNull := g.newTmp()
			g.emit("%s = icmp eq ptr %s, null", isNull, v)
			some, end := g.newLabel("hash.some"), g.newLabel("hash.end")
			g.emitTerm("br i1 %s, label %%%s, label %%%s", isNull, end, some)
			g.placeLabel(some)
			g.emit("store i64 %s, ptr %s", g.mix("1", g.hash(tt.Elem, v)), res)
			g.emitTerm("br label %%%s", end)
			g.placeLabel(end)
			out := g.newTmp()
			g.emit("%s = load i64, ptr %s", out, res)
			return out
		}
		tag := g.newTmp()
		g.emit("%s = extractvalue %s %s, 0", tag, llt, v)
		res := g.alloca("i64")
		g.emit("store i64 0, ptr %s", res)
		some, end := g.newLabel("hash.some"), g.newLabel("hash.end")
		g.emitTerm("br i1 %s, label %%%s, label %%%s", tag, some, end)
		g.placeLabel(some)
		inner := g.newTmp()
		g.emit("%s = extractvalue %s %s, 1", inner, llt, v)
		g.emit("store i64 %s, ptr %s", g.mix("1", g.hash(tt.Elem, inner)), res)
		g.emitTerm("br label %%%s", end)
		g.placeLabel(end)
		out := g.newTmp()
		g.emit("%s = load i64, ptr %s", out, res)
		return out
	case *types.Tuple:
		return g.hashFields(llt, v, tt.Elems)
	case *types.Struct:
		var fs []types.Type
		for _, f := range tt.Fields {
			fs = append(fs, f.Type)
		}
		return g.hashFields(llt, v, fs)
	case *types.Sealed:
		return g.hashTagged(llt, v, func(i int) types.Type { return tt.Variants[i] }, len(tt.Variants))
	case *types.ErrorUnion:
		return g.hashTagged(llt, v, func(i int) types.Type { return tt.Members[i] }, len(tt.Members))
	case *types.List:
		return g.hashList(tt.Elem, v)
	case *types.Map, *types.Set:
		return g.hashMap(t, v)
	}
	return "0"
}

// hashList folds the element hashes in order, seeded with the length.
func (g *gen) hashList(elem types.Type, v string) string {
	n := g.newTmp()
	g.emit("%s = call i64 @veles_list_len(ptr %s)", n, v)
	acc := g.alloca("i64")
	g.emit("store i64 %s, ptr %s", g.mix("17", n), acc)
	i := g.alloca("i64")
	g.emit("store i64 0, ptr %s", i)
	condL, bodyL, endL := g.newLabel("hash.cond"), g.newLabel("hash.body"), g.newLabel("hash.end")
	g.emitTerm("br label %%%s", condL)
	g.placeLabel(condL)
	iv := g.newTmp()
	g.emit("%s = load i64, ptr %s", iv, i)
	more := g.newTmp()
	g.emit("%s = icmp slt i64 %s, %s", more, iv, n)
	g.emitTerm("br i1 %s, label %%%s, label %%%s", more, bodyL, endL)
	g.placeLabel(bodyL)
	p := g.newTmp()
	g.emit("%s = call ptr @veles_list_ref(ptr %s, i64 %s)", p, v, iv)
	x := g.newTmp()
	g.emit("%s = load %s, ptr %s", x, g.llType(elem), p)
	cur := g.newTmp()
	g.emit("%s = load i64, ptr %s", cur, acc)
	g.emit("store i64 %s, ptr %s", g.mix(cur, g.hash(elem, x)), acc)
	inc := g.newTmp()
	g.emit("%s = add i64 %s, 1", inc, iv)
	g.emit("store i64 %s, ptr %s", inc, i)
	g.emitTerm("br label %%%s", condL)
	g.placeLabel(endL)
	out := g.newTmp()
	g.emit("%s = load i64, ptr %s", out, acc)
	return out
}

// hashMap sums the entry hashes so that the result does not depend on
// insertion order, which `==` ignores too.
func (g *gen) hashMap(t types.Type, v string) string {
	kt, vt := keyValTypes(t)
	_, isSet := t.(*types.Set)
	n := g.newTmp()
	g.emit("%s = call i64 @veles_map_len(ptr %s)", n, v)
	used := g.newTmp()
	g.emit("%s = call i64 @veles_map_used(ptr %s)", used, v)
	acc := g.alloca("i64")
	g.emit("store i64 %s, ptr %s", g.mix("23", n), acc)
	i := g.alloca("i64")
	g.emit("store i64 0, ptr %s", i)
	condL, bodyL, liveL, nextL, endL := g.newLabel("hash.cond"), g.newLabel("hash.body"), g.newLabel("hash.live"), g.newLabel("hash.next"), g.newLabel("hash.end")
	g.emitTerm("br label %%%s", condL)
	g.placeLabel(condL)
	iv := g.newTmp()
	g.emit("%s = load i64, ptr %s", iv, i)
	more := g.newTmp()
	g.emit("%s = icmp slt i64 %s, %s", more, iv, used)
	g.emitTerm("br i1 %s, label %%%s, label %%%s", more, bodyL, endL)
	g.placeLabel(bodyL)
	live := g.newTmp()
	g.emit("%s = call i1 @veles_map_live(ptr %s, i64 %s)", live, v, iv)
	g.emitTerm("br i1 %s, label %%%s, label %%%s", live, liveL, nextL)
	g.placeLabel(liveL)
	kp := g.newTmp()
	g.emit("%s = call ptr @veles_map_key_at(ptr %s, i64 %s)", kp, v, iv)
	k := g.newTmp()
	g.emit("%s = load %s, ptr %s", k, g.llType(kt), kp)
	h := g.hash(kt, k)
	if !isSet {
		vp := g.newTmp()
		g.emit("%s = call ptr @veles_map_val_at(ptr %s, i64 %s)", vp, v, iv)
		val := g.newTmp()
		g.emit("%s = load %s, ptr %s", val, g.llType(vt), vp)
		h = g.mix(h, g.hash(vt, val))
	}
	cur := g.newTmp()
	g.emit("%s = load i64, ptr %s", cur, acc)
	sum := g.newTmp()
	g.emit("%s = add i64 %s, %s", sum, cur, h)
	g.emit("store i64 %s, ptr %s", sum, acc)
	g.emitTerm("br label %%%s", nextL)
	g.placeLabel(nextL)
	inc := g.newTmp()
	g.emit("%s = add i64 %s, 1", inc, iv)
	g.emit("store i64 %s, ptr %s", inc, i)
	g.emitTerm("br label %%%s", condL)
	g.placeLabel(endL)
	out := g.newTmp()
	g.emit("%s = load i64, ptr %s", out, acc)
	return out
}

func (g *gen) hashFields(llt, v string, fields []types.Type) string {
	acc := "17"
	for i, ft := range fields {
		fv := g.newTmp()
		g.emit("%s = extractvalue %s %s, %d", fv, llt, v, i)
		acc = g.mix(acc, g.hash(ft, fv))
	}
	return acc
}

func (g *gen) hashTagged(llt, v string, member func(i int) types.Type, n int) string {
	tag := g.newTmp()
	g.emit("%s = extractvalue %s %s, 0", tag, llt, v)
	tag64 := g.newTmp()
	g.emit("%s = zext i32 %s to i64", tag64, tag)
	res := g.alloca("i64")
	g.emit("store i64 %s, ptr %s", tag64, res)
	end := g.newLabel("hash.end")
	labels := make([]string, n)
	cases := ""
	for i := 0; i < n; i++ {
		labels[i] = g.newLabel("hash.case")
		cases += fmt.Sprintf(" i32 %d, label %%%s", i, labels[i])
	}
	g.emitTerm("switch i32 %s, label %%%s [%s ]", tag, end, cases)
	for i := 0; i < n; i++ {
		g.placeLabel(labels[i])
		mt := member(i)
		payload := g.extractTagged(llt, v, g.llType(mt))
		g.emit("store i64 %s, ptr %s", g.mix(tag64, g.hash(mt, payload)), res)
		g.emitTerm("br label %%%s", end)
	}
	g.placeLabel(end)
	out := g.newTmp()
	g.emit("%s = load i64, ptr %s", out, res)
	return out
}

// eqPtrHelper wraps structural equality in the runtime's callback shape:
// `i1 (ptr, ptr)`.
func (g *gen) eqPtrHelper(t types.Type) string {
	key := types.Key(t)
	if name, ok := g.eqPtrFns[key]; ok {
		return name
	}
	name := "eqp." + mangleType(t)
	g.eqPtrFns[key] = name
	llt := g.llType(t)
	g.pending = append(g.pending, func() {
		g.defineHelper(name, "i1", []string{"ptr %a", "ptr %b"}, func() {
			a := g.newTmp()
			g.emit("%s = load %s, ptr %%a", a, llt)
			b := g.newTmp()
			g.emit("%s = load %s, ptr %%b", b, llt)
			r := g.equal(t, a, b)
			g.emitTerm("ret i1 %s", r)
		})
	})
	return name
}

// showMap renders `{k: v, ...}` or `{a, b}`.
func (g *gen) showMap(t types.Type, v string) string {
	kt, vt := keyValTypes(t)
	isSet := false
	if _, ok := t.(*types.Set); ok {
		isSet = true
	}
	res := g.alloca(strType)
	g.emit("store %s %s, ptr %s", strType, g.stringConst("{"), res)
	first := g.alloca("i1")
	g.emit("store i1 true, ptr %s", first)
	n := g.newTmp()
	g.emit("%s = call i64 @veles_map_used(ptr %s)", n, v)
	i := g.alloca("i64")
	g.emit("store i64 0, ptr %s", i)
	condL, bodyL, liveL, nextL, endL := g.newLabel("show.cond"), g.newLabel("show.body"), g.newLabel("show.live"), g.newLabel("show.next"), g.newLabel("show.end")
	g.emitTerm("br label %%%s", condL)
	g.placeLabel(condL)
	iv := g.newTmp()
	g.emit("%s = load i64, ptr %s", iv, i)
	c := g.newTmp()
	g.emit("%s = icmp slt i64 %s, %s", c, iv, n)
	g.emitTerm("br i1 %s, label %%%s, label %%%s", c, bodyL, endL)
	g.placeLabel(bodyL)
	live := g.newTmp()
	g.emit("%s = call i1 @veles_map_live(ptr %s, i64 %s)", live, v, iv)
	g.emitTerm("br i1 %s, label %%%s, label %%%s", live, liveL, nextL)
	g.placeLabel(liveL)
	cur := g.newTmp()
	g.emit("%s = load %s, ptr %s", cur, strType, res)
	isFirst := g.newTmp()
	g.emit("%s = load i1, ptr %s", isFirst, first)
	sep := g.newTmp()
	g.emit("%s = select i1 %s, %s %s, %s %s", sep, isFirst, strType, g.stringConst(""), strType, g.stringConst(", "))
	g.emit("store i1 false, ptr %s", first)
	cur = g.concat(cur, sep)
	kp := g.newTmp()
	g.emit("%s = call ptr @veles_map_key_at(ptr %s, i64 %s)", kp, v, iv)
	k := g.newTmp()
	g.emit("%s = load %s, ptr %s", k, g.llType(kt), kp)
	cur = g.concat(cur, g.show(kt, k))
	if !isSet {
		vp := g.newTmp()
		g.emit("%s = call ptr @veles_map_val_at(ptr %s, i64 %s)", vp, v, iv)
		val := g.newTmp()
		g.emit("%s = load %s, ptr %s", val, g.llType(vt), vp)
		cur = g.concat(g.concat(cur, g.stringConst(": ")), g.show(vt, val))
	}
	g.emit("store %s %s, ptr %s", strType, cur, res)
	g.emitTerm("br label %%%s", nextL)
	g.placeLabel(nextL)
	next := g.newTmp()
	g.emit("%s = add i64 %s, 1", next, iv)
	g.emit("store i64 %s, ptr %s", next, i)
	g.emitTerm("br label %%%s", condL)
	g.placeLabel(endL)
	last := g.newTmp()
	g.emit("%s = load %s, ptr %s", last, strType, res)
	return g.concat(last, g.stringConst("}"))
}

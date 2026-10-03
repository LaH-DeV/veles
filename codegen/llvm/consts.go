package llvm

import (
	"fmt"
	"math"
	"math/big"
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
		fmt.Fprintf(&g.constOut, "%s = private unnamed_addr constant %s { ptr %s, i64 %d, i64 %d, i64 %d, ptr %s, i64 0 }\n",
			name, listHeader, data, len(v.Elems), len(v.Elems), size, g.arrayDescOf(v.T.Elem))
	case *sema.CMap:
		g.constMap(name, base, v.T.Key, v.T.Value, v.Keys, v.Vals)
	case *sema.CSet:
		g.constMap(name, base, v.T.Elem, nil, v.Elems, nil)
	default:
		panic(fmt.Sprintf("constTable: %T", v))
	}
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

// mapHeader is veles_map as veles_rt.c lays it out: keys, vals, meta,
// keyDesc, valDesc, index, icap, len, used, cap, keySize, valSize, mods.
// constNameRe matches what an unquoted LLVM global name cannot hold.
var constNameRe = regexp.MustCompile(`[^A-Za-z0-9._]`)

const mapHeader = "{ ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64 }"

// constMap emits a map (vt nil: a set) with its entries in order, their
// hashes as the runtime computes them, and an open-addressing index with
// at least half its slots free, as veles_map_insert keeps it.
func (g *gen) constMap(name, base string, kt, vt types.Type, keys, vals []sema.ConstVal) {
	n := len(keys)
	icap := 16
	for n*2 >= icap {
		icap *= 2
	}
	index := make([]int64, icap)
	meta := make([]string, n)
	for e, k := range keys {
		h := constHash(kt, k)
		meta[e] = fmt.Sprintf("{ i64, i8 } { i64 %d, i8 1 }", h)
		i := uint64(h) & uint64(icap-1)
		for index[i] > 0 {
			i = (i + 1) & uint64(icap-1)
		}
		index[i] = int64(e) + 1
	}
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
	fmt.Fprintf(&g.constOut, "%s.meta = private unnamed_addr constant [%d x { i64, i8 }] %s\n", name, n, metaBody)
	slots := make([]string, icap)
	for i, s := range index {
		slots[i] = fmt.Sprintf("i64 %d", s)
	}
	fmt.Fprintf(&g.constOut, "%s.index = private unnamed_addr constant [%d x i64] [%s]\n", name, icap, strings.Join(slots, ", "))
	keySize, _ := g.layout(kt)
	fmt.Fprintf(&g.constOut, "%s = private unnamed_addr constant %s { ptr %s, ptr %s, ptr %s.meta, ptr %s, ptr %s, ptr %s.index, i64 %d, i64 %d, i64 %d, i64 %d, i64 %d, i64 %d, i64 0 }\n",
		name, mapHeader, keyArr, valArr, name, g.arrayDescOf(kt), valDesc, name, icap, n, n, n, keySize, valSize)
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

// constHash is the hash the generated code computes for a key (maps.go
// hash/hashBody), here over a constant. The checker refuses key types with
// a `hash` or `equals` of their own (D113), so the structural rules are all.
func constHash(t types.Type, v sema.ConstVal) int64 {
	t = types.Underlying(t)
	switch v := v.(type) {
	case *sema.CInt:
		n := new(big.Int).Set(v.V)
		if n.Sign() < 0 {
			n.Add(n, new(big.Int).Lsh(big.NewInt(1), 64))
		}
		return int64(n.Uint64()) // sign- or zero-extended to 64 bits: the value itself
	case *sema.CFloat:
		if types.BitSize(t) == 32 {
			return int64(math.Float32bits(float32(v.V)))
		}
		return int64(math.Float64bits(v.V))
	case *sema.CBool:
		if v.V {
			return 1
		}
		return 0
	case *sema.CString:
		return hashBytes(v.V)
	case *sema.CUnit:
		return 0
	case *sema.CNull:
		return 0
	case *sema.CSome:
		return hashMix(1, constHash(t.(*types.Nullable).Elem, v.V))
	case *sema.CTuple:
		acc := int64(17)
		for i, x := range v.Elems {
			acc = hashMix(acc, constHash(v.T.Elems[i], x))
		}
		return acc
	case *sema.CStruct:
		acc := int64(17)
		for i, x := range v.Fields {
			acc = hashMix(acc, constHash(v.T.Fields[i].Type, x))
		}
		return acc
	case *sema.CList:
		acc := hashMix(17, int64(len(v.Elems)))
		for _, x := range v.Elems {
			acc = hashMix(acc, constHash(v.T.Elem, x))
		}
		return acc
	case *sema.CSet:
		acc := hashMix(23, int64(len(v.Elems)))
		for _, x := range v.Elems {
			acc += constHash(v.T.Elem, x)
		}
		return acc
	case *sema.CMap:
		acc := hashMix(23, int64(len(v.Keys)))
		for i, k := range v.Keys {
			acc += hashMix(constHash(v.T.Key, k), constHash(v.T.Value, v.Vals[i]))
		}
		return acc
	}
	panic(fmt.Sprintf("constHash: %T", v))
}

// hashBytes is veles_hash_bytes: FNV-1a over the bytes.
func hashBytes(s string) int64 {
	h := uint64(1469598103934665603)
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= 1099511628211
	}
	return int64(h)
}

// hashMix is veles_hash_mix.
func hashMix(a, b int64) int64 {
	h := uint64(a) * 0x9E3779B97F4A7C15
	h ^= uint64(b) + 0x7F4A7C15 + (h << 6) + (h >> 2)
	return int64(h)
}

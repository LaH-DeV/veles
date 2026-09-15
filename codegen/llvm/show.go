package llvm

import (
	"fmt"
	"strings"

	"github.com/LaH-DeV/veles/sema"
	"github.com/LaH-DeV/veles/types"
)

// Helper functions (string conversion for interpolation and structural
// equality) are generated on demand, one per type, after the function that
// first needs them.

// defineHelper generates a standalone function using the normal builder.
func (g *gen) defineHelper(name, retLL string, params []string, body func()) {
	saved := struct {
		fn      *sema.Func
		body    string
		allocas string
		tmp     int
		label   int
		term    bool
		storage map[*sema.Var]string
		loops   map[*sema.Loop]loopLabels
		result  types.Type
	}{g.fn, g.body.String(), g.allocas.String(), g.tmp, g.label, g.term, g.storage, g.loops, g.fnResult}
	g.resetFn(&sema.Func{Name: name, Sig: &types.Func{Ret: types.TUnit}})
	body()
	fmt.Fprintf(&g.helpers, "define internal %s @%s(%s) {\nentry:\n", retLL, name, strings.Join(params, ", "))
	g.helpers.WriteString(g.allocas.String())
	g.helpers.WriteString(g.body.String())
	g.helpers.WriteString("}\n\n")
	g.fn = saved.fn
	g.body.Reset()
	g.body.WriteString(saved.body)
	g.allocas.Reset()
	g.allocas.WriteString(saved.allocas)
	g.tmp, g.label, g.term, g.storage, g.loops, g.fnResult = saved.tmp, saved.label, saved.term, saved.storage, saved.loops, saved.result
}

// concat appends b to a (both %str values).
func (g *gen) concat(a, b string) string {
	ap, al := g.strPtrLen(a)
	bp, bl := g.strPtrLen(b)
	out := g.alloca(strType)
	g.emit("call void @veles_string_concat(ptr %s, ptr %s, i64 %s, ptr %s, i64 %s)", out, ap, al, bp, bl)
	v := g.newTmp()
	g.emit("%s = load %s, ptr %s", v, strType, out)
	return v
}

// show converts a value to its string form.
func (g *gen) show(t types.Type, v string) string {
	switch tt := t.(type) {
	case *types.Basic:
		switch tt.Kind {
		case types.String:
			return v
		case types.Unit, types.Never:
			return g.stringConst("()")
		case types.Bool:
			out := g.alloca(strType)
			g.emit("call void @veles_bool_to_string(ptr %s, i1 %s)", out, v)
			r := g.newTmp()
			g.emit("%s = load %s, ptr %s", r, strType, out)
			return r
		}
		out := g.alloca(strType)
		if types.IsFloat(t) {
			x := v
			if tt.Kind == types.F32 {
				x = g.newTmp()
				g.emit("%s = fpext float %s to double", x, v)
			}
			g.emit("call void @veles_f64_to_string(ptr %s, double %s)", out, x)
		} else {
			x := v
			llt := g.llType(t)
			if llt != "i64" {
				x = g.newTmp()
				if types.IsSigned(t) {
					g.emit("%s = sext %s %s to i64", x, llt, v)
				} else {
					g.emit("%s = zext %s %s to i64", x, llt, v)
				}
			}
			if types.IsSigned(t) {
				g.emit("call void @veles_i64_to_string(ptr %s, i64 %s)", out, x)
			} else {
				g.emit("call void @veles_u64_to_string(ptr %s, i64 %s)", out, x)
			}
		}
		r := g.newTmp()
		g.emit("%s = load %s, ptr %s", r, strType, out)
		return r
	}
	name := g.showHelper(t)
	r := g.newTmp()
	g.emit("%s = call %s @%s(%s %s)", r, strType, name, g.llType(t), v)
	return r
}

func (g *gen) showHelper(t types.Type) string {
	key := types.Key(t)
	if name, ok := g.showFns[key]; ok {
		return name
	}
	name := "show." + mangleType(t)
	g.showFns[key] = name
	llt := g.llType(t)
	g.pending = append(g.pending, func() {
		g.defineHelper(name, strType, []string{llt + " %v"}, func() {
			r := g.showBody(t, "%v")
			g.emitTerm("ret %s %s", strType, r)
		})
	})
	return name
}

func (g *gen) showBody(t types.Type, v string) string {
	switch tt := t.(type) {
	case *types.Pointer:
		if tt.Raw {
			return g.stringConst("<raw pointer>")
		}
		inner := g.newTmp()
		g.emit("%s = load %s, ptr %s", inner, g.llType(tt.Elem), v)
		return g.concat(g.stringConst("&"), g.show(tt.Elem, inner))
	case *types.Nullable:
		res := g.alloca(strType)
		isNull := g.newTmp()
		if isPtrLike(tt.Elem) {
			g.emit("%s = icmp eq ptr %s, null", isNull, v)
		} else {
			tag := g.newTmp()
			g.emit("%s = extractvalue %s %s, 0", tag, g.llType(tt), v)
			g.emit("%s = xor i1 %s, true", isNull, tag)
		}
		nullL, someL, endL := g.newLabel("show.null"), g.newLabel("show.some"), g.newLabel("show.end")
		g.emitTerm("br i1 %s, label %%%s, label %%%s", isNull, nullL, someL)
		g.placeLabel(nullL)
		g.emit("store %s %s, ptr %s", strType, g.stringConst("null"), res)
		g.emitTerm("br label %%%s", endL)
		g.placeLabel(someL)
		inner := v
		if !isPtrLike(tt.Elem) {
			inner = g.newTmp()
			g.emit("%s = extractvalue %s %s, 1", inner, g.llType(tt), v)
		}
		s := g.show(tt.Elem, inner)
		g.emit("store %s %s, ptr %s", strType, s, res)
		g.emitTerm("br label %%%s", endL)
		g.placeLabel(endL)
		r := g.newTmp()
		g.emit("%s = load %s, ptr %s", r, strType, res)
		return r
	case *types.Tuple:
		acc := g.stringConst("(")
		for i, e := range tt.Elems {
			if i > 0 {
				acc = g.concat(acc, g.stringConst(", "))
			}
			el := g.newTmp()
			g.emit("%s = extractvalue %s %s, %d", el, g.llType(tt), v, i)
			acc = g.concat(acc, g.show(e, el))
		}
		return g.concat(acc, g.stringConst(")"))
	case *types.Range:
		lo := g.newTmp()
		g.emit("%s = extractvalue %s %s, 0", lo, g.llType(tt), v)
		hi := g.newTmp()
		g.emit("%s = extractvalue %s %s, 1", hi, g.llType(tt), v)
		incl := g.newTmp()
		g.emit("%s = extractvalue %s %s, 2", incl, g.llType(tt), v)
		sep := g.newTmp()
		g.emit("%s = select i1 %s, %s %s, %s %s", sep, incl, strType, g.stringConst(".."), strType, g.stringConst("..<"))
		return g.concat(g.concat(g.show(tt.Elem, lo), sep), g.show(tt.Elem, hi))
	case *types.List:
		// "[" + elements joined by ", " + "]"
		res := g.alloca(strType)
		g.emit("store %s %s, ptr %s", strType, g.stringConst("["), res)
		n := g.newTmp()
		g.emit("%s = call i64 @veles_list_len(ptr %s)", n, v)
		i := g.alloca("i64")
		g.emit("store i64 0, ptr %s", i)
		condL, bodyL, endL := g.newLabel("show.cond"), g.newLabel("show.body"), g.newLabel("show.end")
		g.emitTerm("br label %%%s", condL)
		g.placeLabel(condL)
		iv := g.newTmp()
		g.emit("%s = load i64, ptr %s", iv, i)
		c := g.newTmp()
		g.emit("%s = icmp slt i64 %s, %s", c, iv, n)
		g.emitTerm("br i1 %s, label %%%s, label %%%s", c, bodyL, endL)
		g.placeLabel(bodyL)
		cur := g.newTmp()
		g.emit("%s = load %s, ptr %s", cur, strType, res)
		first := g.newTmp()
		g.emit("%s = icmp eq i64 %s, 0", first, iv)
		sep := g.newTmp()
		g.emit("%s = select i1 %s, %s %s, %s %s", sep, first, strType, g.stringConst(""), strType, g.stringConst(", "))
		cur = g.concat(cur, sep)
		p := g.newTmp()
		g.emit("%s = call ptr @veles_list_ref(ptr %s, i64 %s)", p, v, iv)
		el := g.newTmp()
		g.emit("%s = load %s, ptr %s", el, g.llType(tt.Elem), p)
		cur = g.concat(cur, g.show(tt.Elem, el))
		g.emit("store %s %s, ptr %s", strType, cur, res)
		next := g.newTmp()
		g.emit("%s = add i64 %s, 1", next, iv)
		g.emit("store i64 %s, ptr %s", next, i)
		g.emitTerm("br label %%%s", condL)
		g.placeLabel(endL)
		last := g.newTmp()
		g.emit("%s = load %s, ptr %s", last, strType, res)
		return g.concat(last, g.stringConst("]"))
	case *types.Struct:
		if len(tt.Fields) == 0 {
			return g.stringConst(tt.Name)
		}
		acc := g.stringConst(tt.Name + "(")
		for i, f := range tt.Fields {
			if i > 0 {
				acc = g.concat(acc, g.stringConst(", "))
			}
			acc = g.concat(acc, g.stringConst(f.Name+": "))
			fv := g.newTmp()
			g.emit("%s = extractvalue %s %s, %d", fv, g.llType(tt), v, i)
			acc = g.concat(acc, g.show(f.Type, fv))
		}
		return g.concat(acc, g.stringConst(")"))
	case *types.Sealed:
		return g.showTagged(g.llType(tt), v, func(i int) (types.Type, string) {
			return tt.Variants[i], g.llType(tt.Variants[i])
		}, len(tt.Variants))
	case *types.ErrorUnion:
		return g.showTagged(g.llType(tt), v, func(i int) (types.Type, string) {
			return tt.Members[i], g.llType(tt.Members[i])
		}, len(tt.Members))
	}
	return g.stringConst("<" + t.String() + ">")
}

// showTagged switches on a tag and shows the selected payload.
func (g *gen) showTagged(llt, v string, member func(i int) (types.Type, string), n int) string {
	res := g.alloca(strType)
	g.emit("store %s %s, ptr %s", strType, g.stringConst("?"), res)
	tag := g.newTmp()
	g.emit("%s = extractvalue %s %s, 0", tag, llt, v)
	endL := g.newLabel("show.end")
	var cases []string
	labels := make([]string, n)
	for i := 0; i < n; i++ {
		labels[i] = g.newLabel("show.case")
		cases = append(cases, fmt.Sprintf("i32 %d, label %%%s", i, labels[i]))
	}
	g.emitTerm("switch i32 %s, label %%%s [ %s ]", tag, endL, strings.Join(cases, " "))
	for i := 0; i < n; i++ {
		g.placeLabel(labels[i])
		mt, mll := member(i)
		payload := g.extractTagged(llt, v, mll)
		s := g.show(mt, payload)
		g.emit("store %s %s, ptr %s", strType, s, res)
		g.emitTerm("br label %%%s", endL)
	}
	g.placeLabel(endL)
	r := g.newTmp()
	g.emit("%s = load %s, ptr %s", r, strType, res)
	return r
}

// ---------------------------------------------------------------------------
// structural equality

func (g *gen) equal(t types.Type, l, r string) string {
	switch tt := t.(type) {
	case *types.Basic:
		v := g.newTmp()
		switch {
		case tt.Kind == types.String:
			lp, ll := g.strPtrLen(l)
			rp, rl := g.strPtrLen(r)
			g.emit("%s = call i1 @veles_string_eq(ptr %s, i64 %s, ptr %s, i64 %s)", v, lp, ll, rp, rl)
		case types.IsFloat(t):
			g.emit("%s = fcmp oeq %s %s, %s", v, g.llType(t), l, r)
		case tt.Kind == types.Unit || tt.Kind == types.Never:
			return "true"
		default:
			g.emit("%s = icmp eq %s %s, %s", v, g.llType(t), l, r)
		}
		return v
	case *types.Pointer, *types.List:
		v := g.newTmp()
		g.emit("%s = icmp eq ptr %s, %s", v, l, r)
		return v
	}
	name := g.eqHelper(t)
	v := g.newTmp()
	g.emit("%s = call i1 @%s(%s %s, %s %s)", v, name, g.llType(t), l, g.llType(t), r)
	return v
}

func (g *gen) eqHelper(t types.Type) string {
	key := types.Key(t)
	if name, ok := g.eqFns[key]; ok {
		return name
	}
	name := "eq." + mangleType(t)
	g.eqFns[key] = name
	llt := g.llType(t)
	g.pending = append(g.pending, func() {
		g.defineHelper(name, "i1", []string{llt + " %a", llt + " %b"}, func() {
			r := g.eqBody(t, "%a", "%b")
			g.emitTerm("ret i1 %s", r)
		})
	})
	return name
}

func (g *gen) eqBody(t types.Type, a, b string) string {
	llt := g.llType(t)
	switch tt := t.(type) {
	case *types.Nullable:
		if isPtrLike(tt.Elem) {
			v := g.newTmp()
			g.emit("%s = icmp eq ptr %s, %s", v, a, b)
			return v
		}
		ta := g.newTmp()
		g.emit("%s = extractvalue %s %s, 0", ta, llt, a)
		tb := g.newTmp()
		g.emit("%s = extractvalue %s %s, 0", tb, llt, b)
		sameTag := g.newTmp()
		g.emit("%s = icmp eq i1 %s, %s", sameTag, ta, tb)
		res := g.alloca("i1")
		g.emit("store i1 %s, ptr %s", sameTag, res)
		bothL, endL := g.newLabel("eq.both"), g.newLabel("eq.end")
		both := g.newTmp()
		g.emit("%s = and i1 %s, %s", both, sameTag, ta)
		g.emitTerm("br i1 %s, label %%%s, label %%%s", both, bothL, endL)
		g.placeLabel(bothL)
		va := g.newTmp()
		g.emit("%s = extractvalue %s %s, 1", va, llt, a)
		vb := g.newTmp()
		g.emit("%s = extractvalue %s %s, 1", vb, llt, b)
		e := g.equal(tt.Elem, va, vb)
		g.emit("store i1 %s, ptr %s", e, res)
		g.emitTerm("br label %%%s", endL)
		g.placeLabel(endL)
		v := g.newTmp()
		g.emit("%s = load i1, ptr %s", v, res)
		return v
	case *types.Tuple:
		return g.eqFields(llt, a, b, tt.Elems)
	case *types.Range:
		return g.eqFields(llt, a, b, []types.Type{tt.Elem, tt.Elem, types.TBool})
	case *types.Struct:
		var fs []types.Type
		for _, f := range tt.Fields {
			fs = append(fs, f.Type)
		}
		return g.eqFields(llt, a, b, fs)
	case *types.Sealed:
		return g.eqTagged(llt, a, b, func(i int) types.Type { return tt.Variants[i] }, len(tt.Variants))
	case *types.ErrorUnion:
		return g.eqTagged(llt, a, b, func(i int) types.Type { return tt.Members[i] }, len(tt.Members))
	}
	return "false"
}

func (g *gen) eqFields(llt, a, b string, fields []types.Type) string {
	acc := "true"
	for i, ft := range fields {
		fa := g.newTmp()
		g.emit("%s = extractvalue %s %s, %d", fa, llt, a, i)
		fb := g.newTmp()
		g.emit("%s = extractvalue %s %s, %d", fb, llt, b, i)
		e := g.equal(ft, fa, fb)
		n := g.newTmp()
		g.emit("%s = and i1 %s, %s", n, acc, e)
		acc = n
	}
	return acc
}

func (g *gen) eqTagged(llt, a, b string, member func(i int) types.Type, n int) string {
	ta := g.newTmp()
	g.emit("%s = extractvalue %s %s, 0", ta, llt, a)
	tb := g.newTmp()
	g.emit("%s = extractvalue %s %s, 0", tb, llt, b)
	res := g.alloca("i1")
	g.emit("store i1 false, ptr %s", res)
	sameTag := g.newTmp()
	g.emit("%s = icmp eq i32 %s, %s", sameTag, ta, tb)
	swL, endL := g.newLabel("eq.switch"), g.newLabel("eq.end")
	g.emitTerm("br i1 %s, label %%%s, label %%%s", sameTag, swL, endL)
	g.placeLabel(swL)
	labels := make([]string, n)
	var cases []string
	for i := 0; i < n; i++ {
		labels[i] = g.newLabel("eq.case")
		cases = append(cases, fmt.Sprintf("i32 %d, label %%%s", i, labels[i]))
	}
	g.emitTerm("switch i32 %s, label %%%s [ %s ]", ta, endL, strings.Join(cases, " "))
	for i := 0; i < n; i++ {
		g.placeLabel(labels[i])
		mt := member(i)
		pa := g.extractTagged(llt, a, g.llType(mt))
		pb := g.extractTagged(llt, b, g.llType(mt))
		e := g.equal(mt, pa, pb)
		g.emit("store i1 %s, ptr %s", e, res)
		g.emitTerm("br label %%%s", endL)
	}
	g.placeLabel(endL)
	v := g.newTmp()
	g.emit("%s = load i1, ptr %s", v, res)
	return v
}

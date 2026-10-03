package sema

import "strings"

// Module-level values are computed before main, in the order they need each
// other: a global whose initializer reads another — directly, or through any
// function it calls, a lambda it builds included — comes after it. Source
// order breaks ties, so programs without such reads keep their order. A
// cycle has no first value to compute and is an error; before this order
// existed, `val a = f()` with `f` reading a later `val` saw that value's
// zero bits (a null list, an empty string), not its value.

// globalDeps is what one function body or initializer reads directly.
type globalDeps struct {
	globals []*Global
	callees []*Func
}

func directDeps(visitRoot func(visit func(node any))) globalDeps {
	var d globalDeps
	seenG := map[*Global]bool{}
	seenF := map[*Func]bool{}
	addF := func(fn *Func) {
		if fn != nil && !seenF[fn] {
			seenF[fn] = true
			d.callees = append(d.callees, fn)
		}
	}
	visitRoot(func(node any) {
		switch n := node.(type) {
		case *VarRef:
			if n.Var.IsGlobal && n.Var.Global != nil && !seenG[n.Var.Global] {
				seenG[n.Var.Global] = true
				d.globals = append(d.globals, n.Var.Global)
			}
		case *Call:
			addF(n.Fn)
		case *FuncRef:
			addF(n.Fn)
		case *Closure:
			addF(n.Fn)
		}
	})
	return d
}

// orderGlobals sorts prog.Globals so every initializer runs after the
// globals it reads, and reports each cycle once.
func (c *Checker) orderGlobals(prog *Program) {
	fnDeps := map[*Func]globalDeps{}
	depsOf := func(fn *Func) globalDeps {
		if d, ok := fnDeps[fn]; ok {
			return d
		}
		fnDeps[fn] = globalDeps{} // a recursive function reaches itself
		d := directDeps(func(visit func(node any)) { walkBlock(fn.Body, visit) })
		fnDeps[fn] = d
		return d
	}
	type edge struct {
		to  *Global
		via *Func // the function the read is in, nil when the initializer reads it itself
	}
	inProg := map[*Global]bool{}
	for _, g := range prog.Globals {
		inProg[g] = true
	}
	edges := map[*Global][]edge{}
	for _, g := range prog.Globals {
		if g.Init == nil {
			continue
		}
		root := directDeps(func(visit func(node any)) { walkExpr(g.Init, visit) })
		seen := map[*Global]bool{}
		for _, to := range root.globals {
			if inProg[to] && !seen[to] {
				seen[to] = true
				edges[g] = append(edges[g], edge{to, nil})
			}
		}
		// breadth first through the calls, remembering the first call on
		// the way for the message
		type item struct{ fn, via *Func }
		visited := map[*Func]bool{}
		var queue []item
		for _, fn := range root.callees {
			if !visited[fn] {
				visited[fn] = true
				queue = append(queue, item{fn, fn})
			}
		}
		for len(queue) > 0 {
			it := queue[0]
			queue = queue[1:]
			d := depsOf(it.fn)
			for _, to := range d.globals {
				if inProg[to] && !seen[to] {
					seen[to] = true
					edges[g] = append(edges[g], edge{to, it.via})
				}
			}
			for _, fn := range d.callees {
				if !visited[fn] {
					visited[fn] = true
					queue = append(queue, item{fn, it.via})
				}
			}
		}
	}

	const (
		visiting = 1
		done     = 2
	)
	state := map[*Global]int{}
	var stack []*Global
	var stackVia []*Func
	ordered := make([]*Global, 0, len(prog.Globals))
	var visit func(g *Global)
	visit = func(g *Global) {
		switch state[g] {
		case done:
			return
		case visiting:
			// the cycle is the stack from g's entry
			start := 0
			for i, s := range stack {
				if s == g {
					start = i
				}
			}
			c.reportInitCycle(stack[start:], stackVia[start:])
			return
		}
		state[g] = visiting
		for _, e := range edges[g] {
			stack = append(stack, g)
			stackVia = append(stackVia, e.via)
			visit(e.to)
			stack = stack[:len(stack)-1]
			stackVia = stackVia[:len(stackVia)-1]
		}
		state[g] = done
		ordered = append(ordered, g)
	}
	for _, g := range prog.Globals {
		visit(g)
	}
	prog.Globals = ordered
}

// reportInitCycle reports a cycle of globals at its first member: cycle[i]
// reads cycle[i+1] (the last reads the first), through via[i] when not nil.
func (c *Checker) reportInitCycle(cycle []*Global, via []*Func) {
	if len(cycle) == 1 {
		how := ""
		if via[0] != nil {
			how = " through its call of '" + via[0].Display + "'"
		}
		c.errorf(cycle[0].Span, "initialization cycle: '%s' reads itself%s, before it has a value; compute the value in a function instead, or pass it as a parameter", cycle[0].Display, how)
		return
	}
	var b strings.Builder
	for i, g := range cycle {
		next := cycle[(i+1)%len(cycle)]
		if i > 0 {
			b.WriteString(", which ")
		} else {
			b.WriteString("'" + g.Display + "' ")
		}
		b.WriteString("reads '" + next.Display + "'")
		if via[i] != nil {
			b.WriteString(" (through its call of '" + via[i].Display + "')")
		}
	}
	c.errorf(cycle[0].Span, "initialization cycle: %s; no value can be computed first — compute one of them in a function, or pass it as a parameter", b.String())
}

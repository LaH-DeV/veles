package sema

// D217: a function value whose type does not say `suspends` — a named
// function or a variable — is accepted for a `suspends` parameter of a
// conditionally suspending function (`xs.map(double)`): the call is then
// plain, as with a plain lambda, and the plain copy of the callee D116
// makes calls it as it is. The checker accepts it as a Cast and records
// where; after suspension inference this keeps only the ones that are
// such an argument. Anywhere else the plain value would be called as a
// suspending function, so it is refused with the lambda that adapts it:
// a parameter the callee keeps (it is then not conditional), a call made
// suspending by another of its arguments, a call through a function value
// or a trait object, or any place that is not an argument at all.

// checkPlainFuncArgs refuses the plain function values that are not
// arguments of a plain call of a conditionally suspending function.
func (c *Checker) checkPlainFuncArgs(prog *Program) {
	if len(c.plainFuncAt) == 0 {
		return
	}
	fine := map[*Cast]bool{}
	why := map[*Cast]string{}
	w := susp{}
	for _, fn := range prog.Funcs {
		walkBlock(fn.Body, func(n any) {
			call, ok := n.(*Call)
			if !ok {
				return
			}
			for i, a := range call.Args {
				cast, ok := a.(*Cast)
				if !ok {
					continue
				}
				if _, plain := c.plainFuncAt[cast]; !plain {
					continue
				}
				switch {
				case call.Fn == nil || !call.Fn.Conditional:
					why[cast] = "the function this is passed to keeps it and may call it later as one that suspends, which a function that does not suspend cannot be; pass a lambda that calls it, '(x) => f(x)', which adapts (D217)"
				case w.argSuspends(cast):
					why[cast] = "this function suspends, but its type as a value does not say so; pass a lambda that calls it, '(x) => f(x)', which adapts (D217)"
				case otherSuspends(w, call.Args, i):
					why[cast] = "another argument makes this call suspend, so every function passed to it is called as one that may, and this one does not suspend; pass a lambda that calls it, '(x) => f(x)', which adapts (D217)"
				default:
					fine[cast] = true
				}
			}
		})
	}
	for cast, span := range c.plainFuncAt {
		if fine[cast] {
			continue
		}
		msg := why[cast]
		if msg == "" {
			msg = "a function that does not suspend stands where one that suspends is expected, which it may only do as an argument a function calls; pass a lambda that calls it, '(x) => f(x)', which adapts (D217)"
		}
		c.errorf(span, "%s", msg)
	}
}

// otherSuspends reports whether an argument of a call other than the i-th
// makes it suspend.
func otherSuspends(w susp, args []Expr, i int) bool {
	for j, a := range args {
		if j != i && funcTypeOf(a.Type()) != nil && w.argSuspends(a) {
			return true
		}
	}
	return false
}

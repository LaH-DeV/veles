package llvm

import "strings"

// pruneModule drops the function definitions and private constants of a
// finished module that nothing reachable names, so clang compiles only what
// the program can run: hello world uses a few dozen of the hundreds of std
// functions every module defines.
//
// It works on the text, not the HIR, because the text is where every way of
// reaching a function ends up as an `@name`: a call, a function value, a
// closure, a coroutine's resume function, a trait's method table, a thunk the
// runtime is handed. The roots are what is named from outside the module —
// the definitions C can link to (no `.` in the name: `veles_main`,
// `veles_init_globals`, `extern "C"` exports) and the globals that are not
// private — plus whatever the declarations and type lines name. The rest is
// kept in its order, unchanged.
func pruneModule(ir string) string {
	type item struct {
		text string
		name string // the defined @name; "" for lines that define nothing
		drop bool   // may be dropped when unreachable
	}
	var items []item
	for rest := ir; rest != ""; {
		line := rest
		if i := strings.IndexByte(rest, '\n'); i >= 0 {
			line = rest[:i+1]
		}
		switch {
		case strings.HasPrefix(line, "define "):
			// the body ends at the first line that is just "}", and the
			// blank line after it goes with it
			end := strings.Index(rest, "\n}\n")
			if end < 0 {
				return ir // not a module this pass understands: leave it alone
			}
			end += len("\n}\n")
			if strings.HasPrefix(rest[end:], "\n") {
				end++
			}
			name := symbolAt(line, strings.IndexByte(line, '@'))
			items = append(items, item{rest[:end], name, strings.IndexByte(name, '.') >= 0})
			rest = rest[end:]
			continue
		case strings.HasPrefix(line, "@"):
			name := symbolAt(line, 0)
			def := line[len(name)+1:]
			local := strings.HasPrefix(def, " = private ") || strings.HasPrefix(def, " = internal ")
			items = append(items, item{line, name, local})
		default:
			items = append(items, item{text: line})
		}
		rest = rest[len(line):]
	}

	index := map[string]int{}
	for i, it := range items {
		if it.name != "" {
			index[it.name] = i
		}
	}
	live := make([]bool, len(items))
	var work []int
	for i, it := range items {
		if !it.drop {
			live[i] = true
			work = append(work, i)
		}
	}
	for len(work) > 0 {
		text := items[work[len(work)-1]].text
		work = work[:len(work)-1]
		for i := strings.IndexByte(text, '@'); i >= 0; {
			name := symbolAt(text, i)
			if j, ok := index[name]; ok && !live[j] {
				live[j] = true
				work = append(work, j)
			}
			next := strings.IndexByte(text[i+1:], '@')
			if next < 0 {
				break
			}
			i += 1 + next
		}
	}

	var sb strings.Builder
	sb.Grow(len(ir))
	for i, it := range items {
		if live[i] {
			sb.WriteString(it.text)
		}
	}
	return sb.String()
}

// symbolAt returns the global name written at text[at] ("@name" or
// `@"name"`), without the @; "" when there is none.
func symbolAt(text string, at int) string {
	if at < 0 || at+1 >= len(text) {
		return ""
	}
	s := text[at+1:]
	if s[0] == '"' {
		if end := strings.IndexByte(s[1:], '"'); end >= 0 {
			return s[:end+2]
		}
		return ""
	}
	n := 0
	for n < len(s) && isNameByte(s[n]) {
		n++
	}
	return s[:n]
}

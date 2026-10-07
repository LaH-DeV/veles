package sema

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

// A small TOML reader for `veles.toml` (D138). It reads what a manifest
// needs and refuses the rest with a message: tables (`[a]`, `[a.b]`), keys,
// basic and literal strings, integers, booleans, arrays (several lines, a
// trailing comma) and inline tables. Dotted keys, array-of-tables, dates
// and floats are errors, so a manifest cannot mean something here and
// something else elsewhere. Kept small on purpose: it is the one piece of
// the package system that Veles itself will have to read once the front
// end is self-hosted.

type tomlKind int

const (
	tomlStr tomlKind = iota
	tomlInt
	tomlBool
	tomlList
	tomlTab
)

func (k tomlKind) String() string {
	switch k {
	case tomlStr:
		return "a string"
	case tomlInt:
		return "a number"
	case tomlBool:
		return "true or false"
	case tomlList:
		return "a list"
	}
	return "a table"
}

type tomlVal struct {
	kind tomlKind
	str  string
	num  int64
	b    bool
	list []*tomlVal
	tab  *tomlTable
	line int
}

type tomlTable struct {
	vals    map[string]*tomlVal
	line    int
	defined bool // named by a `[header]` or written inline, not just a parent of one
}

func newTomlTable(line int) *tomlTable {
	return &tomlTable{vals: map[string]*tomlVal{}, line: line}
}

// keys returns the table's keys in a fixed order.
func (t *tomlTable) keys() []string {
	out := make([]string, 0, len(t.vals))
	for k := range t.vals {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

type tomlError struct {
	line int
	msg  string
}

func (e *tomlError) Error() string { return fmt.Sprintf("line %d: %s", e.line, e.msg) }

type tomlParser struct {
	src  string
	pos  int
	line int
}

// parseTOML parses text into its root table.
func parseTOML(text string) (*tomlTable, error) {
	p := &tomlParser{src: strings.ReplaceAll(text, "\r\n", "\n"), line: 1}
	p.src = strings.TrimPrefix(p.src, string(rune(0xFEFF))) // a byte-order mark
	root := newTomlTable(1)
	cur := root
	for {
		p.skipSpace(true)
		if p.pos >= len(p.src) {
			return root, nil
		}
		if p.src[p.pos] == '[' {
			if strings.HasPrefix(p.src[p.pos:], "[[") {
				return nil, p.fail("arrays of tables ([[...]]) are not supported in veles.toml")
			}
			p.pos++
			var path []string
			for {
				p.skipSpace(false)
				key, err := p.key()
				if err != nil {
					return nil, err
				}
				path = append(path, key)
				p.skipSpace(false)
				if p.eat('.') {
					continue
				}
				if !p.eat(']') {
					return nil, p.fail("expected ']' to close the table name")
				}
				break
			}
			t, err := p.table(root, path)
			if err != nil {
				return nil, err
			}
			cur = t
			if err := p.endOfLine(); err != nil {
				return nil, err
			}
			continue
		}
		key, err := p.key()
		if err != nil {
			return nil, err
		}
		line := p.line
		p.skipSpace(false)
		if p.pos < len(p.src) && p.src[p.pos] == '.' {
			return nil, p.fail("dotted keys are not supported: write '[" + key + "]' as a table or '{ ... }' inline")
		}
		if !p.eat('=') {
			return nil, p.fail("expected 'key = value'")
		}
		p.skipSpace(false)
		v, err := p.value()
		if err != nil {
			return nil, err
		}
		v.line = line
		if _, dup := cur.vals[key]; dup {
			return nil, &tomlError{line, fmt.Sprintf("key %q is set twice", key)}
		}
		cur.vals[key] = v
		if err := p.endOfLine(); err != nil {
			return nil, err
		}
	}
}

// table finds or creates the table at path below root. A table named twice
// is an error; a table reached through a parent that is defined later is
// fine (`[a.b]` before `[a]`).
func (p *tomlParser) table(root *tomlTable, path []string) (*tomlTable, error) {
	t := root
	for i, k := range path {
		v, ok := t.vals[k]
		if !ok {
			nt := newTomlTable(p.line)
			t.vals[k] = &tomlVal{kind: tomlTab, tab: nt, line: p.line}
			t = nt
			continue
		}
		if v.kind != tomlTab {
			return nil, p.fail(fmt.Sprintf("%q is already %s, not a table", strings.Join(path[:i+1], "."), v.kind))
		}
		if i == len(path)-1 && v.tab.defined {
			return nil, p.fail(fmt.Sprintf("table [%s] is defined twice", strings.Join(path, ".")))
		}
		t = v.tab
	}
	t.defined = true
	return t, nil
}

func (p *tomlParser) fail(msg string) error { return &tomlError{p.line, msg} }

func (p *tomlParser) eat(c byte) bool {
	if p.pos < len(p.src) && p.src[p.pos] == c {
		p.pos++
		return true
	}
	return false
}

// skipSpace skips blanks and comments; newlines too when nl is set.
func (p *tomlParser) skipSpace(nl bool) {
	for p.pos < len(p.src) {
		switch c := p.src[p.pos]; {
		case c == ' ' || c == '\t':
			p.pos++
		case c == '#':
			for p.pos < len(p.src) && p.src[p.pos] != '\n' {
				p.pos++
			}
		case c == '\n' && nl:
			p.pos++
			p.line++
		default:
			return
		}
	}
}

// endOfLine requires nothing but a comment before the next newline.
func (p *tomlParser) endOfLine() error {
	p.skipSpace(false)
	if p.pos < len(p.src) && p.src[p.pos] != '\n' {
		return p.fail("unexpected text after the value")
	}
	return nil
}

func (p *tomlParser) key() (string, error) {
	if p.pos < len(p.src) && (p.src[p.pos] == '"' || p.src[p.pos] == '\'') {
		return p.str()
	}
	start := p.pos
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-' {
			p.pos++
			continue
		}
		break
	}
	if start == p.pos {
		return "", p.fail("expected a key")
	}
	return p.src[start:p.pos], nil
}

func (p *tomlParser) str() (string, error) {
	q := p.src[p.pos]
	p.pos++
	var sb strings.Builder
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		switch {
		case c == '\n':
			return "", p.fail("a string must close on its own line")
		case c == q:
			p.pos++
			return sb.String(), nil
		case c == '\\' && q == '"':
			p.pos++
			if p.pos >= len(p.src) {
				return "", p.fail("unfinished escape")
			}
			switch e := p.src[p.pos]; e {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case '"', '\\':
				sb.WriteByte(e)
			case 'u':
				if p.pos+4 >= len(p.src) {
					return "", p.fail("unfinished \\u escape")
				}
				r, err := strconv.ParseUint(p.src[p.pos+1:p.pos+5], 16, 32)
				if err != nil || !utf8.ValidRune(rune(r)) {
					return "", p.fail("bad \\u escape")
				}
				sb.WriteRune(rune(r))
				p.pos += 4
			default:
				return "", p.fail(fmt.Sprintf("unknown escape \\%c", e))
			}
			p.pos++
		default:
			sb.WriteByte(c)
			p.pos++
		}
	}
	return "", p.fail("unterminated string")
}

func (p *tomlParser) value() (*tomlVal, error) {
	if p.pos >= len(p.src) {
		return nil, p.fail("expected a value")
	}
	line := p.line
	switch c := p.src[p.pos]; {
	case c == '"' || c == '\'':
		s, err := p.str()
		if err != nil {
			return nil, err
		}
		return &tomlVal{kind: tomlStr, str: s, line: line}, nil
	case c == '[':
		p.pos++
		v := &tomlVal{kind: tomlList, line: line}
		for {
			p.skipSpace(true)
			if p.eat(']') {
				return v, nil
			}
			item, err := p.value()
			if err != nil {
				return nil, err
			}
			v.list = append(v.list, item)
			p.skipSpace(true)
			if p.eat(',') {
				continue
			}
			p.skipSpace(true)
			if !p.eat(']') {
				return nil, p.fail("expected ',' or ']' in the list")
			}
			return v, nil
		}
	case c == '{':
		p.pos++
		t := newTomlTable(line)
		t.defined = true
		v := &tomlVal{kind: tomlTab, tab: t, line: line}
		p.skipSpace(false)
		if p.eat('}') {
			return v, nil
		}
		for {
			p.skipSpace(false)
			if p.pos < len(p.src) && p.src[p.pos] == '\n' {
				return nil, p.fail("an inline table is one line: close it with '}' or use a [table] header")
			}
			k, err := p.key()
			if err != nil {
				return nil, err
			}
			p.skipSpace(false)
			if !p.eat('=') {
				return nil, p.fail("expected '=' in the inline table")
			}
			p.skipSpace(false)
			item, err := p.value()
			if err != nil {
				return nil, err
			}
			item.line = line
			if _, dup := t.vals[k]; dup {
				return nil, p.fail(fmt.Sprintf("key %q is set twice", k))
			}
			t.vals[k] = item
			p.skipSpace(false)
			if p.eat(',') {
				continue
			}
			if !p.eat('}') {
				return nil, p.fail("expected ',' or '}' in the inline table (an inline table is one line)")
			}
			return v, nil
		}
	default:
		start := p.pos
		for p.pos < len(p.src) && !strings.ContainsRune(" \t\n,]}#", rune(p.src[p.pos])) {
			p.pos++
		}
		word := p.src[start:p.pos]
		switch word {
		case "true", "false":
			return &tomlVal{kind: tomlBool, b: word == "true", line: line}, nil
		case "":
			return nil, p.fail("expected a value")
		}
		if n, err := strconv.ParseInt(strings.ReplaceAll(word, "_", ""), 10, 64); err == nil {
			return &tomlVal{kind: tomlInt, num: n, line: line}, nil
		}
		return nil, p.fail(fmt.Sprintf("cannot read %q: strings are written in quotes (\"%s\")", word, word))
	}
}

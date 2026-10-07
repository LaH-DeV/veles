package fetch

import (
	"fmt"
	"regexp"
	"strings"
)

// Editing veles.toml as text (D138): `veles add`, `update` and `remove` change
// one line of a dependency table and leave everything else — comments,
// order, blank lines, the file's line endings — as the author wrote it. A
// manifest is read back through the real reader after every edit, so an edit
// that would produce an invalid file never reaches the disk.

var (
	headerRE = regexp.MustCompile(`^\s*\[\s*([^\[\]]+?)\s*\]\s*(#.*)?$`)
	keyRE    = regexp.MustCompile(`^\s*("([^"]*)"|'([^']*)'|[A-Za-z0-9_-]+)\s*=`)
)

// splitLines returns the lines of text without their endings, and the
// ending the file uses.
func splitLines(text string) ([]string, string) {
	eol := "\n"
	if strings.Contains(text, "\r\n") {
		eol = "\r\n"
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.TrimSuffix(text, "\n")
	if text == "" {
		return nil, eol
	}
	return strings.Split(text, "\n"), eol
}

func keyOf(line string) (string, bool) {
	m := keyRE.FindStringSubmatch(line)
	if m == nil {
		return "", false
	}
	switch {
	case m[2] != "":
		return m[2], true
	case m[3] != "":
		return m[3], true
	}
	return m[1], true
}

// section finds a table: the index of its header line and the index just
// past its last line (the next header, or the end).
func section(lines []string, name string) (start, end int, found bool) {
	start = -1
	for i, line := range lines {
		m := headerRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if start >= 0 {
			return start, i, true
		}
		if m[1] == name {
			start = i
		}
	}
	if start >= 0 {
		return start, len(lines), true
	}
	return 0, 0, false
}

// subTable reports whether the manifest declares `[table.name]`, which the
// editor does not rewrite.
func subTable(lines []string, table, name string) bool {
	for _, line := range lines {
		if m := headerRE.FindStringSubmatch(line); m != nil && (m[1] == table+"."+name || m[1] == table+`."`+name+`"`) {
			return true
		}
	}
	return false
}

// SetDependency adds `name = value` to the table (`dependencies` or
// `dev-dependencies`), or replaces the value of the entry already there. value
// is TOML: `"../lib"` or `{ git = "…", version = "1.2.3" }`. It returns the
// new text and whether an entry was replaced.
func SetDependency(text, table, name, value string) (string, bool, error) {
	lines, eol := splitLines(text)
	if subTable(lines, table, name) {
		return "", false, fmt.Errorf("dependency '%s' is written as a [%s.%s] table; edit it by hand", name, table, name)
	}
	line := name + " = " + value
	start, end, found := section(lines, table)
	if !found {
		out := append([]string{}, lines...)
		if len(out) > 0 {
			out = append(out, "")
		}
		out = append(out, "["+table+"]", line)
		return strings.Join(out, eol) + eol, false, nil
	}
	last := start // the last line of the table that holds something
	for i := start + 1; i < end; i++ {
		if k, ok := keyOf(lines[i]); ok {
			if k == name {
				lines[i] = line
				return strings.Join(lines, eol) + eol, true, nil
			}
			last = i
		} else if t := strings.TrimSpace(lines[i]); t != "" && !strings.HasPrefix(t, "#") {
			last = i
		}
	}
	if last == start { // an empty table: below the comments that open it
		for last+1 < end && strings.HasPrefix(strings.TrimSpace(lines[last+1]), "#") {
			last++
		}
	}
	out := append([]string{}, lines[:last+1]...)
	out = append(out, line)
	out = append(out, lines[last+1:]...)
	return strings.Join(out, eol) + eol, false, nil
}

// RemoveDependency deletes the entry; it reports whether one was there.
func RemoveDependency(text, table, name string) (string, bool, error) {
	lines, eol := splitLines(text)
	if subTable(lines, table, name) {
		return "", false, fmt.Errorf("dependency '%s' is written as a [%s.%s] table; remove it by hand", name, table, name)
	}
	start, end, found := section(lines, table)
	if !found {
		return text, false, nil
	}
	for i := start + 1; i < end; i++ {
		if k, ok := keyOf(lines[i]); ok && k == name {
			out := append(append([]string{}, lines[:i]...), lines[i+1:]...)
			return strings.Join(out, eol) + eol, true, nil
		}
	}
	return text, false, nil
}

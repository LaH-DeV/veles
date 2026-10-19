package sema

import (
	"fmt"
	"strconv"
	"strings"
)

// TOMLDump is the text the self-hosted TOML reader (selfhost/sema/toml.vs) is
// compared against, plan R1: the tree parseTOML makes, keys in order, or the
// one error. selfhost/sema/toml_dump.vs prints the same.
func TOMLDump(text string) string {
	root, err := parseTOML(text)
	if err != nil {
		return "error " + err.Error() + "\n"
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "root @%d %s\n", root.line, tomlDefined(root))
	dumpTomlTable(&sb, root, "  ")
	return sb.String()
}

func tomlDefined(t *tomlTable) string {
	if t.defined {
		return "defined"
	}
	return "implied"
}

func dumpTomlTable(sb *strings.Builder, t *tomlTable, indent string) {
	for _, k := range t.keys() {
		dumpTomlVal(sb, indent+strconv.Quote(k), t.vals[k], indent)
	}
}

func dumpTomlVal(sb *strings.Builder, label string, v *tomlVal, indent string) {
	switch v.kind {
	case tomlStr:
		fmt.Fprintf(sb, "%s: str @%d %s\n", label, v.line, strconv.Quote(v.str))
	case tomlInt:
		fmt.Fprintf(sb, "%s: int @%d %d\n", label, v.line, v.num)
	case tomlBool:
		fmt.Fprintf(sb, "%s: bool @%d %t\n", label, v.line, v.b)
	case tomlList:
		fmt.Fprintf(sb, "%s: list @%d %d\n", label, v.line, len(v.list))
		for _, item := range v.list {
			dumpTomlVal(sb, indent+"  -", item, indent+"  ")
		}
	case tomlTab:
		fmt.Fprintf(sb, "%s: table @%d %s @%d\n", label, v.line, tomlDefined(v.tab), v.tab.line)
		dumpTomlTable(sb, v.tab, indent+"  ")
	}
}

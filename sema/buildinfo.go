package sema

import (
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/types"
)

// D160, D204: std/build's constants are answered by the compiler. Its
// `target` and `program` are written in Veles over two std-only calls,
// `compilerTarget(key)` and `compilerProgram(key)`, each a string constant
// known while compiling: the target is the machine the compiler builds for
// (there is no cross-compile yet), the program is the package the build
// started from.
func (f *fnCtx) compilerFact(fn string, e *ast.CallExpr) Expr {
	lit, ok := e.Args[0].Value.(*ast.StringLit)
	key := ""
	if ok && len(lit.Parts) == 1 && lit.Parts[0].Expr == nil {
		key = lit.Parts[0].Text
	}
	value, known := "", false
	if fn == "compilerTarget" {
		value, known = targetFact(key, f.c.release)
	} else {
		value, known = f.c.programFact(key)
	}
	if !known {
		f.errorf(e.Pos, "%s takes one of its keys as a literal; %s is not one", fn, strconv.Quote(key))
		return bad()
	}
	return &StringConst{exprBase{types.TString}, value}
}

func targetFact(key string, release bool) (string, bool) {
	switch key {
	case "os":
		switch runtime.GOOS {
		case "windows":
			return "windows", true
		case "darwin":
			return "macos", true
		}
		return "linux", true
	case "arch":
		if runtime.GOARCH == "arm64" {
			return "arm64", true
		}
		return "x64", true
	case "release":
		return strconv.FormatBool(release), true
	}
	return "", false
}

// programFact is the name or version of the package being built: its
// manifest's, or, for a program without one, the name of the directory (or
// of the file, when the build was pointed at one).
func (c *Checker) programFact(key string) (string, bool) {
	switch key {
	case "name":
		if c.pkg != nil && c.pkg.Manifest != nil && c.pkg.Manifest.Name != "" {
			return c.pkg.Manifest.Name, true
		}
		if c.pkg != nil && c.pkg.Root != "" {
			return strings.TrimSuffix(filepath.Base(c.pkg.Root), ".vs"), true
		}
		return "", true
	case "version":
		if c.pkg != nil && c.pkg.Manifest != nil {
			return c.pkg.Manifest.Version, true
		}
		return "", true
	}
	return "", false
}

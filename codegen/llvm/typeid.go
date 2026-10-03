package llvm

import (
	"fmt"
	"strings"

	"github.com/LaH-DeV/veles/sema"
	"github.com/LaH-DeV/veles/types"
)

// Type identity of trait objects (D117, D135). Slot 0 of every method table
// points to the info global of the type the table was built for, an
// `internal constant i64` holding a dense id. `x is Circle` compares that
// pointer with Circle's; `x is Flusher` reads the id and loads Flusher's
// method table for it from `@it.<Flusher>`, an array over every id, null
// where the type does not implement the trait. Ids are given to the types
// that are boxed or tested for, so the arrays are as long as that list.

// typeInfo names the info global of a type, giving the type its id.
func (g *gen) typeInfo(t types.Type) string {
	name := "ti." + mangleType(t)
	if _, ok := g.typeIDs[name]; !ok {
		g.typeIDs[name] = len(g.typeInfos)
		g.typeInfos = append(g.typeInfos, name)
	}
	return name
}

// traitTableName is the global of a trait's table.
func traitTableName(trait *types.Trait) string {
	return "it." + mangleModule(trait.Module) + "." + trait.Name
}

func mangleModule(m string) string {
	return strings.NewReplacer("/", "_", " ", "_").Replace(m)
}

// traitTables emits the method tables the `is` tables point to; the tables
// themselves are written by typeInfoGlobals, once every id is given.
func (g *gen) traitTables() {
	for _, tab := range g.prog.TraitTables {
		for _, e := range tab.Entries {
			g.vtableFor(tab.Trait, e.Type, e.Methods)
		}
	}
}

// typeInfoGlobals writes each type's info global and each trait's table.
func (g *gen) typeInfoGlobals() {
	for id, name := range g.typeInfos {
		fmt.Fprintf(&g.helpers, "@%s = internal constant i64 %d\n", name, id)
	}
	for _, tab := range g.prog.TraitTables {
		slots := make([]string, len(g.typeInfos))
		for i := range slots {
			slots[i] = "ptr null"
		}
		for _, e := range tab.Entries {
			slots[g.typeIDs[g.typeInfo(e.Type)]] = "ptr @" + vtableName(tab.Trait, e.Type)
		}
		fmt.Fprintf(&g.helpers, "@%s = internal constant [%d x ptr] [%s]\n", traitTableName(tab.Trait), len(slots), strings.Join(slots, ", "))
	}
	if len(g.typeInfos) > 0 {
		g.helpers.WriteString("\n")
	}
}

// objectType loads the info pointer of a trait object's type.
func (g *gen) objectType(obj string) string {
	vt := g.newTmp()
	g.emit("%s = extractvalue { ptr, ptr } %s, 1", vt, obj)
	ti := g.newTmp()
	g.emit("%s = load ptr, ptr %s", ti, vt)
	return ti
}

// objectTable loads Trait's method table for a trait object's type, or null.
func (g *gen) objectTable(obj string, trait *types.Trait) string {
	id := g.newTmp()
	g.emit("%s = load i64, ptr %s", id, g.objectType(obj))
	p := g.newTmp()
	g.emit("%s = getelementptr ptr, ptr @%s, i64 %s", p, traitTableName(trait), id)
	vt := g.newTmp()
	g.emit("%s = load ptr, ptr %s", vt, p)
	return vt
}

func (g *gen) typeTest(e *sema.TypeTest) string {
	ti := g.objectType(g.expr(e.X))
	v := g.newTmp()
	g.emit("%s = icmp eq ptr %s, @%s", v, ti, g.typeInfo(e.Target))
	return v
}

func (g *gen) downcast(e *sema.Downcast) string {
	data := g.newTmp()
	g.emit("%s = extractvalue { ptr, ptr } %s, 0", data, g.expr(e.X))
	return g.loadVal(e.Type(), data)
}

func (g *gen) traitTest(e *sema.TraitTest) string {
	vt := g.objectTable(g.expr(e.X), e.Trait)
	v := g.newTmp()
	g.emit("%s = icmp ne ptr %s, null", v, vt)
	return v
}

func (g *gen) traitCast(e *sema.TraitCast) string {
	obj := g.expr(e.X)
	vt := g.objectTable(obj, e.Type().(*types.Trait))
	v := g.newTmp()
	g.emit("%s = insertvalue { ptr, ptr } %s, ptr %s, 1", v, obj, vt)
	return v
}

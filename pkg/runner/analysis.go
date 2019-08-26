package runner

import (
	"fmt"
	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

type callGraphConstructor struct {
	entrypoints []*ssa.Function
	Graph       *callgraph.Graph
}

func newCallGraphBuilder() *callGraphConstructor {
	return &callGraphConstructor{}
}

func (c *callGraphConstructor) update() {
	rtares := rta.Analyze(c.entrypoints, true)
	c.Graph = rtares.CallGraph
}

func (c *callGraphConstructor) AddEntrypoints(entrypoints []*ssa.Function) {
	c.entrypoints = append(c.entrypoints, entrypoints...)
	c.update()
}

func (c *callGraphConstructor) AddPackageMains(pkgs []*ssa.Package) {
	for _, mainPkg := range ssautil.MainPackages(pkgs) {
		c.entrypoints = append(c.entrypoints, mainPkg.Func("main"))

		// main package init should recursively call init of imported packages
		c.entrypoints = append(c.entrypoints, mainPkg.Func("init"))
	}
	c.update()
}

func constructSSA(pkgName string) (*ssa.Program, []*ssa.Package, error) {
	// Load, parse, and type-check the whole program.
	cfg := packages.Config{
		Mode: packages.LoadAllSyntax, // TODO fix - AllPackages does say that this is the expected value
	}
	initial, err := packages.Load(&cfg, pkgName)
	if err != nil {
		return nil, nil, err
	}

	for _, pkg := range initial {
		if pkg.Types == nil || pkg.IllTyped {
			return nil, nil, fmt.Errorf("package %s is ill typed", pkg.Name)
		}
	}

	// Create SSA packages for well-typed packages and their dependencies.
	prog, pkgs := ssautil.AllPackages(initial, ssa.BuilderMode(0))
	_ = pkgs

	// Build SSA code for the whole program.
	prog.Build()

	return prog, pkgs, nil
}

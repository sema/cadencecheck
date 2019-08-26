package runner

import (
	"github.com/sema/cadencecheck/pkg/entities"
	"github.com/sema/cadencecheck/pkg/reporter"
	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/ssa"
)

var (
	_cadenceRegisterPatterns = []entities.FunctionPattern{
		{
			Package: "go.uber.org/cadence/workflow",
			Type:    "",
			Method:  "Register",
		},
		{
			Package: "go.uber.org/cadence/workflow",
			Type:    "",
			Method:  "RegisterWithOptions",
		},
	}

	_fxProviderPatterns = []entities.FunctionPattern{
		{
			Package: "go.uber.org/fx",
			Type:    "",
			Method:  "Provide",
		},
	}
)

type Check interface {
	Check(f *ssa.Function, callGraph *callgraph.Graph, reporter *reporter.TerminalReporter) error
}

type Runner struct {
	reporter *reporter.TerminalReporter
	checks   []Check
}

func New(reporter *reporter.TerminalReporter, checks []Check) *Runner {
	return &Runner{
		reporter: reporter,
		checks:   checks,
	}
}

func (r *Runner) Run(pkgName string) error {
	prog, pkgs, err := constructSSA(pkgName)
	if err != nil {
		return err
	}

	callGraph := newCallGraphBuilder()
	callGraph.AddPackageMains(pkgs)

	var fxProviderFunctions []*ssa.Function
	for _, fxProviderPattern := range _fxProviderPatterns {
		fns, err := findRegisteredFunctions(r.reporter, prog, callGraph.Graph, fxProviderPattern)
		if err != nil {
			return err
		}

		fxProviderFunctions = append(fxProviderFunctions, fns...)
	}

	callGraph.AddEntrypoints(fxProviderFunctions)

	/* DEBUG CALL GRAPH
	for f, _ := range callGraph.Nodes {
		r.reporter.Debug("call-graph-func %s (%s)", r.reporter.FormatFunction(f), f.Name())
	}
	*/

	var cadenceWorkflowFunctions []*ssa.Function
	for _, cadenceRegisterPattern := range _cadenceRegisterPatterns {
		fns, err := findRegisteredFunctions(r.reporter, prog, callGraph.Graph, cadenceRegisterPattern)
		if err != nil {
			return err
		}

		cadenceWorkflowFunctions = append(cadenceWorkflowFunctions, fns...)
	}

	// TODO it should not be necessary to add the Cadence functions as entrypoints to the cc analysis -
	// however, the call graph has been shown to be missing edges in large programs.
	callGraph.AddEntrypoints(cadenceWorkflowFunctions)

	for _, f := range cadenceWorkflowFunctions {
		r.reporter.EnterWorkflow(f.RelString(nil))

		for _, check := range r.checks {
			if err := check.Check(f, callGraph.Graph, r.reporter); err != nil {
				return err
			}
		}

		r.reporter.ExitWorkflow()
	}

	return nil
}

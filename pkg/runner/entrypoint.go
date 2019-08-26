package runner

import (
	"fmt"
	"github.com/sema/cadencecheck/pkg/entities"
	"github.com/sema/cadencecheck/pkg/reporter"
	"go/token"
	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/ssa"
	"reflect"
)

// findRegisteredFunctions finds functions F in a program passed to known "registration functions"
//
// A registration function is a pattern used by e.g. Cadence and Fx when registering workflows and providers,
// respectively. For examples, a program can be searched for any function F passed in as an argument to Cadence's
// workflow.Register(F).
//
// This method:
// 1) Searches for the definition of a registration function, R
// 2) Uses the call graph to find all calls to R, C
// 3) Analyses operands for calls C to R, extracting any passed in functions F
func findRegisteredFunctions(
	r *reporter.TerminalReporter,
	prog *ssa.Program,
	callGraph *callgraph.Graph,
	registrationFuncPattern entities.FunctionPattern,
) ([]*ssa.Function, error) {

	registerFunction, err := findRegisterFunctions(prog, registrationFuncPattern)
	if err != nil {
		return nil, err
	}
	if registerFunction == nil {
		r.Debug("registration function %s not in program", registrationFuncPattern.String())
		return nil, nil
	}
	r.Debug("found registration function %s", registrationFuncPattern.String())

	callSites := getCallSitesToFunction(registerFunction, callGraph)
	r.Debug("found %d callers to %s", len(callSites), registrationFuncPattern.String())

	var result []*ssa.Function
	for _, callSite := range callSites {
		var operands []*ssa.Value
		operands = callSite.Operands(operands)

		seen := map[ssa.Value]bool{}
		cadenceWorkflowFunctions, err := resolveFunctionFromSSAValue(*operands[1], callGraph, seen)
		if err != nil {
			// Fail soft - we do not support inferring the value of workflow.Register calls in all cases
			r.Warning(fmt.Sprintf(
				"Unable to infer Cadence workflow function registered at callsite %s: %s",
				r.FormatCallSite(callSite),
				err))

			// DEBUG
			/*
				buf := bytes.Buffer{}
				ssa.WriteFunction(&buf, callSite.Parent())
				r.Debug(buf.String())
			*/
			continue
		}

		if len(cadenceWorkflowFunctions) == 0 {
			r.Warning(fmt.Sprintf(
				"Unable to infer Cadence workflow function registered at callsite %s: inferred 0 functions",
				r.FormatCallSite(callSite)))
		}

		result = append(result, cadenceWorkflowFunctions...)
	}

	r.Debug("found %d functions registered using %s", len(result), registrationFuncPattern.String())

	return result, nil
}

// findRegisterFunctions searches for a function given a FunctionPattern
//
// May return nil if no function matching FunctionPattern is present in the program.
func findRegisterFunctions(prog *ssa.Program, pattern entities.FunctionPattern) (*ssa.Function, error) {
	if pattern.Type != "" {
		return nil, fmt.Errorf("unable to find registration function %s: pattern matching on type unsupported", pattern)
	}

	for _, pkg := range prog.AllPackages() {
		pkgPath := entities.StripVendor(pkg.Pkg.Path())
		if pattern.Package == pkgPath && pkg.Func(pattern.Method) != nil {
			return pkg.Func(pattern.Method), nil
		}
	}

	return nil, nil
}

func getCallSitesToFunction(callee *ssa.Function, callGraph *callgraph.Graph) []ssa.CallInstruction {
	var callSites []ssa.CallInstruction

	node := callGraph.Nodes[callee]
	if node != nil {
		// node may be nil if there are no callers to the method
		for _, in := range node.In {
			callSites = append(callSites, in.Site)
		}
	}

	return callSites
}

// TODO document this as it is non-trivial
// TODO current implementation is filled with hacks and does just enough (and makes a lot of assumptions) to
// get the current examples to work. This should either be a proper DFA or we should find another approach for
// identifying entrypoints
func resolveFunctionFromSSAValue(value ssa.Value, callGraph *callgraph.Graph, seen map[ssa.Value]bool) ([]*ssa.Function, error) {

	if seen[value] {
		return nil, fmt.Errorf("breaking circle")
	}

	seen[value] = true

	switch v := value.(type) {
	case *ssa.MakeInterface:
		var operands []*ssa.Value
		operands = v.Operands(operands)
		return resolveFunctionFromSSAValue(*operands[0], callGraph, seen)

	case *ssa.Function:
		return []*ssa.Function{v}, nil

	case *ssa.Phi:
		var result []*ssa.Function
		for _, e := range v.Edges {
			fs, err := resolveFunctionFromSSAValue(e, callGraph, seen)
			if err != nil {
				return nil, err
			}
			result = append(result, fs...)
		}
		return result, nil

	case *ssa.UnOp:
		// NOT SUB ARROW MUL XOR
		switch v.Op {
		case token.MUL:
			return resolveFunctionFromSSAValue(v.X, callGraph, seen)
		default:
			return nil, fmt.Errorf("unsupported SSA value type %s[%s] encountered when unpacking SSA value", reflect.TypeOf(v), v.Op)
		}

	case *ssa.Parameter:
		idx, err := getParamIndex(v, v.Parent())
		if err != nil {
			return nil, err
		}

		var result []*ssa.Function
		for _, edge := range callGraph.Nodes[v.Parent()].In {
			fs, err := resolveFunctionFromSSAValue(edge.Site.Common().Args[idx+1], callGraph, seen)
			if err != nil {
				return nil, err
			}

			result = append(result, fs...)
		}

		return result, nil

	case *ssa.MakeClosure:
		return resolveFunctionFromSSAValue(v.Fn, callGraph, seen)

	case *ssa.Slice:
		// Encountered for fx.Provide which takes a slice, follow to underlying allocation
		return resolveFunctionFromSSAValue(v.X, callGraph, seen)

	case *ssa.Alloc:
		var result []*ssa.Function
		for _, r := range *v.Referrers() {
			if _, ok := r.(*ssa.Slice); ok {
				continue // HACK bypass circular dependency - this is not a solid analysis FYI
			}

			if _, ok := r.(ssa.Value); !ok {
				continue // must be a value, this is not good either
			}

			fns, err := resolveFunctionFromSSAValue(r.(ssa.Value), callGraph, seen)
			if err != nil {
				return nil, err
			}
			result = append(result, fns...)
		}
		return result, nil

	case *ssa.IndexAddr:
		var result []*ssa.Function
		for _, r := range *v.Referrers() {
			if vv, ok := r.(*ssa.Store); ok {
				fns, err := resolveFunctionFromSSAValue(vv.Val, callGraph, seen)
				if err != nil {
					return nil, err
				}
				result = append(result, fns...)
				continue
			}

			if _, ok := r.(ssa.Value); !ok {
				continue // must be a value, this is not good either
			}

			fns, err := resolveFunctionFromSSAValue(r.(ssa.Value), callGraph, seen)
			if err != nil {
				return nil, err
			}
			result = append(result, fns...)
		}
		return result, nil

	default:
		return nil, fmt.Errorf("unsupported SSA value type %s encountered when unpacking SSA value", reflect.TypeOf(v))
	}
}

func getParamIndex(param *ssa.Parameter, f *ssa.Function) (int, error) {
	for i := 0; i < f.Signature.Params().Len(); i++ {
		if f.Signature.Params().At(i).Name() == param.Name() {
			return i, nil
		}
	}

	return 0, fmt.Errorf("unable to find param in function")
}

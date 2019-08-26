package denypackages

import (
	"errors"
	"fmt"
	"github.com/sema/cadencecheck/pkg/entities"
	"github.com/sema/cadencecheck/pkg/reporter"
	"go/types"
	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/ssa"
	"reflect"
	"strings"
)

const (
	_kindNonDeterministicCall = "ERROR-NON-DETERMINISTIC-CALL"
)

var (
	errIgnoreReceiver = errors.New("ignored receiver type")
)

// TODO move to config file
var inclusion = []entities.FunctionPattern{
	{
		Package: "go.uber.org/cadence/internal",
		Type:    "decodeFutureImpl",
		Method:  "Get",
	},
	{
		Package: "go.uber.org/cadence/internal",
		Type:    "futureImpl",
		Method:  "Get",
	},
	{
		Package: "go.uber.org/cadence/workflow",
		Type:    "",
		Method:  "GetLastCompletionResult",
	},
	{
		Package: "go.uber.org/cadence/workflow",
		Type:    "",
		Method:  "ExecuteActivity",
	},
	{
		Package: "sync",
		Type:    "Pool",
		Method:  "Get",
	},
	{
		Package: "fmt",
		Type:    "",
		Method:  "Sprintf",
	},
	{
		Package: "fmt",
		Type:    "",
		Method:  "Sprint",
	},
	{
		// Undefined if this is actually safe
		Package: "sort",
		Type:    "",
		Method:  "Stable",
	},
	{
		// This is technically bad - however, we currently get a lot of false positives due to the correct interface
		// for emitting metrics in Cadence uses the same interface
		Package: "github.com/uber-go/tally",
		Type:    "scope",
		Method:  "Tagged",
	},
	{
		// This is technically bad - however, we currently get a lot of false positives due to the correct interface
		// for emitting metrics in Cadence uses the same interface
		Package: "github.com/uber-go/tally",
		Type:    "scope",
		Method:  "Counter",
	},
}

var exclusion = []entities.FunctionPattern{
	{
		Package: "time",
		Type:    "",
		Method:  "Now",
	},
	{
		Package: "go.uber.org/zap",
		Type:    "Logger",
		Method:  "Info",
	},
}

type Check struct {
	inclusionMap map[entities.FunctionPattern]bool
	exclusionMap map[entities.FunctionPattern]bool
}

func New() *Check {
	inclusionMap := map[entities.FunctionPattern]bool{}
	for _, i := range inclusion {
		inclusionMap[i] = true
	}

	exclusionMap := map[entities.FunctionPattern]bool{}
	for _, e := range exclusion {
		exclusionMap[e] = true
	}

	return &Check{
		inclusionMap: inclusionMap,
		exclusionMap: exclusionMap,
	}
}

func (c *Check) Check(f *ssa.Function, callGraph *callgraph.Graph, reporter *reporter.TerminalReporter) error {
	root, ok := callGraph.Nodes[f]
	if !ok {
		return fmt.Errorf("could not find callgraph for function %s", reporter.FormatFunction(f))
	}

	seen := map[string]bool{}

	GraphVisitEdges(root, func(edge *callgraph.Edge, previous []*callgraph.Edge) (follow bool) {
		// TODO graph is being traversed multiple times, or we have identical edges. Remove hash dedupe and tests fail
		hash := stackTraceHash(append(previous, edge))
		if seen[hash] {
			return false // already visited
		}
		seen[hash] = true

		signature, err := functionSignature(*edge.Callee.Func)
		if err != nil {
			reporter.Warning(fmt.Sprintf(
				"Unable to determine function signature of callee at %s: %s",
				reporter.FormatCallSite(edge.Site), err))
			return true
		}

		reporter.Debug("workflow calls %s:%s:%s", signature.Package, signature.Type, signature.Method)

		if c.exclusionMap[signature] {
			stackTrace := append(previous, edge)
			calleeName := edge.Callee.Func.RelString(nil)
			reporter.WorkflowIssue(_kindNonDeterministicCall, fmt.Sprintf("detected call to %s", calleeName), stackTrace)

			return false
		}
		if c.inclusionMap[signature] {
			return false
		}

		return true
	})

	return nil
}

func receiverTypeSignature(typ types.Type) (pkgName string, typeName string, err error) {
	switch t := typ.(type) {
	case *types.Named:
		return t.Obj().Pkg().Path(), t.Obj().Name(), nil
	case *types.Pointer:
		return receiverTypeSignature(t.Elem())
	case *types.Struct:
		// ignore structs - these usually represent bound methods and the call graph will point
		// to the method itself
		return "", "", errIgnoreReceiver
	default:
		return "", "", fmt.Errorf("unsupported receiver encountered with type %s: %s",
			reflect.TypeOf(typ).Name(),
			types.TypeString(typ, types.RelativeTo(nil)))
	}
}

func functionSignature(f ssa.Function) (signature entities.FunctionPattern, err error) {
	// Anonymous?
	if f.Parent() != nil {
		// No support for matching anonymous functions
		return entities.FunctionPattern{}, nil
	}

	var recvType types.Type

	if recv := f.Signature.Recv(); recv != nil {
		// Method (declared or wrapper)?
		recvType = recv.Type()
	} else if f.Synthetic == "thunk" {
		// Thunk?
		// NOTE: other synthetic cases have f.Signature.Recv, and are thus handled by the previous case
		recvType = f.Signature.Params().At(0).Type()
	} else if len(f.FreeVars) == 1 && strings.HasSuffix(f.Name(), "$bound") {
		// Bound?
		recvType = f.FreeVars[0].Type()
	}

	if recvType != nil {
		pkgName, typeName, err := receiverTypeSignature(recvType)
		if err != nil {
			if err == errIgnoreReceiver {
				return entities.FunctionPattern{}, nil
			}
			return entities.FunctionPattern{}, err
		}

		return entities.FunctionPattern{
			Package: entities.StripVendor(pkgName),
			Type:    typeName,
			Method:  f.Name(),
		}, nil
	}

	// Package-level function?
	// Prefix with package name for cross-package references only.
	if f.Pkg != nil {
		return entities.FunctionPattern{
			Package: entities.StripVendor(f.Pkg.Pkg.Path()),
			Type:    "",
			Method:  f.Name(),
		}, nil
	}

	return entities.FunctionPattern{}, fmt.Errorf("unable to create signature for function")
}

func stackTraceHash(stackTrace []*callgraph.Edge) string {
	hash := strings.Builder{}
	for _, edge := range stackTrace {
		hash.WriteString(edge.Caller.Func.String())
	}
	last := stackTrace[len(stackTrace)-1]
	hash.WriteString(last.Callee.Func.String())

	return hash.String()
}

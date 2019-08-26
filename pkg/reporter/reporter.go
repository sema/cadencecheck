package reporter

import (
	"fmt"
	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/ssa"
	"io"
	"log"
)

type TerminalReporter struct {
	stdout      io.Writer
	stderr      io.Writer
	verbose     bool
	countIssues int
}

func NewTerminalReporter(stdout io.Writer, stderr io.Writer, verbose bool) *TerminalReporter {
	return &TerminalReporter{
		stdout:  stdout,
		stderr:  stderr,
		verbose: verbose,
	}
}

func (t *TerminalReporter) Debug(format string, a ...interface{}) {
	if t.verbose {
		t.fprintln("DEBUG %s", fmt.Sprintf(format, a...))
	}
}

func (t *TerminalReporter) Warning(message string) {
	t.fprintln("WARNING %s", message)
}

func (t *TerminalReporter) Error(format string, a ...interface{}) {
	t.fprintln(format, a...)
}

func (t *TerminalReporter) EnterWorkflow(relPath string) {
	t.fprintln("CHECK %s", relPath)
}

func (t *TerminalReporter) WorkflowIssue(kind string, message string, stackTrace []*callgraph.Edge) {
	t.countIssues += 1

	t.fprintln("[%s] %s", kind, message)

	nextIdx := 1
	for _, edge := range stackTrace {
		t.fprintln("\t#%3d %s (%s) -->", nextIdx, t.FormatCallSite(edge.Site), edge.Caller.Func.String())
		nextIdx += 1
	}
	last := stackTrace[len(stackTrace)-1]
	t.fprintln("\t#%3d %s (%s)", nextIdx, t.FormatFunction(last.Callee.Func), last.Callee.Func.String())
}

func (t *TerminalReporter) ExitWorkflow() {

}

func (t *TerminalReporter) Footer() {
	if t.countIssues > 0 {
		t.fprintln("Found %d issues", t.countIssues)
	} else {
		t.fprintln("OK - No issues found")
	}
}

func (t *TerminalReporter) handleError(e error) {
	// if we can't write output, then fail hard
	if e != nil {
		log.Fatal(e)
	}
}

func (t *TerminalReporter) FormatCallSite(callSite ssa.CallInstruction) string {
	fset := callSite.Parent().Prog.Fset
	return fset.Position(callSite.Pos()).String()
}

func (t *TerminalReporter) FormatFunction(f *ssa.Function) string {
	if f == nil {
		return ""
	}
	if f.Prog == nil || f.Prog.Fset == nil {
		return fmt.Sprintf("<unknown>.%s", f.Name())
	}

	fset := f.Prog.Fset
	return fset.Position(f.Pos()).String()
}

func (t *TerminalReporter) fprintln(format string, a ...interface{}) {
	_, err := fmt.Fprintln(t.stdout, fmt.Sprintf(format, a...))
	t.handleError(err)
}

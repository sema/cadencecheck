package runner

import (
	"github.com/sema/cadencecheck/pkg/checks/denypackages"
	"github.com/sema/cadencecheck/pkg/reporter"
	"io"
)

// Run wires together services to create a cadence checker, and runs the checker
func Run(pkgName string, stdout io.Writer, stderr io.Writer, verbose bool) error {
	terminalReporter := reporter.NewTerminalReporter(stdout, stderr, verbose)

	checks := []Check{
		denypackages.New(),
	}

	checker := New(terminalReporter, checks)
	err := checker.Run(pkgName)
	if err != nil {
		terminalReporter.Error(err.Error())
		return nil
	}

	terminalReporter.Footer()
	return nil
}

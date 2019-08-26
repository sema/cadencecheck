package main

import (
	"go.uber.org/cadence/workflow"
	"go.uber.org/fx"
)

func main() {
	fx.New(module).Run()
}

var module = fx.Options(
	fx.Provide(
		NewExecutor,
	),
)

type params struct {
	fx.In
}

type Executor struct{}

func (Executor) runWorkflow() {}

// NewExecutor initializes and registers workflow's
func NewExecutor(p params) *Executor {
	executor := &Executor{}

	workflow.RegisterWithOptions(executor.runWorkflow, workflow.RegisterOptions{})
	return executor
}

package main

import (
	"go.uber.org/cadence/workflow"
)

func workflowImpl1() {}
func workflowImpl2() {}

func main() {
	var workflows []func()
	workflows = append(workflows, workflowImpl1)
	workflows = append(workflows, workflowImpl2)

	for _, f := range workflows {
		workflow.Register(f)
	}

	return
}

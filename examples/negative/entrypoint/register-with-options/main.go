package main

import (
	"go.uber.org/cadence/workflow"
)

func workflowImpl() {}

func main() {
	workflow.RegisterWithOptions(workflowImpl, workflow.RegisterOptions{})
	return
}

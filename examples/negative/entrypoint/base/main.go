package main

import (
	"go.uber.org/cadence/workflow"
)

func workflowImpl() {}

func main() {
	workflow.Register(workflowImpl)
	return
}

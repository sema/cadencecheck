package main

import (
	"go.uber.org/cadence/workflow"
)

func workflowImpl() {}

func main() {
	f := workflowImpl

	workflow.Register(f)
	return
}

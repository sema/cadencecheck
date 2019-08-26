package main

import (
	"go.uber.org/cadence/workflow"
)

func workflowImpl1() {}
func workflowImpl2() {}

func main() {
	f1 := workflowImpl1
	f2 := workflowImpl2

	f := f1
	if true {
		f = f2
	}

	workflow.Register(f)
	return
}

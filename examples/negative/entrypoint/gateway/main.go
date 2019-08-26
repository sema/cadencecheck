package main

import (
	"go.uber.org/cadence/workflow"
)

type gateway struct{}

func (gateway) Register(wf interface{}) {
	workflow.Register(wf)
}

func workflowImpl() {}

func main() {
	g := &gateway{}
	g.Register(workflowImpl)
	return
}

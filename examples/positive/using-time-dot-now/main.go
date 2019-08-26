package main

import (
	"fmt"
	"go.uber.org/cadence/workflow"
	"time"
)

func workflowImpl() {
	now := time.Now()
	println(fmt.Sprintf("Hello World @ %s", now))
}

func main() {
	workflow.Register(workflowImpl)
	return
}

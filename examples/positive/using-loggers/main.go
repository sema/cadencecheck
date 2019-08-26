package main

import (
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

func workflowImpl() {
	logger := zap.NewNop()
	logger.Info("message")
}

func main() {
	workflow.Register(workflowImpl)
	return
}

package main

import (
	"go.uber.org/cadence/workflow"
)

func main() {
	workflow.Register(func() {})
	return
}

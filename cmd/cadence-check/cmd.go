package main

import (
	"github.com/sema/cadencecheck/pkg/runner"
	"gopkg.in/alecthomas/kingpin.v2"
	"log"
	"os"
)

var (
	pkgName = kingpin.Arg("package", "Go package to check").Required().String()
	verbose = kingpin.Flag("verbose", "print debug information").Bool()
)

func main() {
	kingpin.Parse()

	err := runner.Run(*pkgName, os.Stdout, os.Stderr, *verbose)
	if err != nil {
		log.Fatalf("Error %s", err)
	}
}

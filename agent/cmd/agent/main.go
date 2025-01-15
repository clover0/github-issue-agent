package main

import (
	"os"

	"github.com/clover0/issue-agent/cli"
	"github.com/clover0/issue-agent/logger"
)

func main() {
	// TODO:
	//lo := logger.NewDefaultLogger()
	lo := logger.NewPrinter("info")
	lo.Info("start agent on container...\n")

	if err := cli.Execute(); err != nil {
		lo.Error("failed to execute command: %s\n", err)
		os.Exit(1)
	}
}

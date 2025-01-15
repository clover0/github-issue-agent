package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/clover0/issue-agent/logger"
)

func Parse() (command string, flags []string, err error) {
	if len(os.Args) < 2 {
		return "", nil, fmt.Errorf("command is required")
	}

	return os.Args[1], os.Args[2:], nil
}

func Execute() error {
	command, flags, err := Parse()
	if err != nil {
		return fmt.Errorf("failed to parse input: %w", err)
	}

	// TODO: bind common flags to common struct here

	lo := logger.NewPrinter("info")
	switch command {
	case "version":
		return VersionCommand()
	case "issue":
		return IssueCommand(flags)
	case "help":
		help(lo)
		return nil
	default:
		help(lo)
		return fmt.Errorf("unknown command: %s", command)
	}

}

func help(lo logger.Logger) {
	msg := "Usage\n"
	msg += "  issue-agent <command> [flags]\n"
	msg += "Commands"
	msg += "  help: Show usage of commands and flags\n"
	msg += "  version: Show version of issue-agent CLI\n"
	issueFlags, _ := IssueFlags()
	msg += "  issue:\n"
	issueFlags.VisitAll(func(flg *flag.Flag) {
		msg += fmt.Sprintf("    --%s\n", flg.Name)
		msg += fmt.Sprintf("        %s\n", flg.Usage)
	})
	lo.Info(msg)
}

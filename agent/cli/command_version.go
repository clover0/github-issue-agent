package cli

import "fmt"

// This value is set at release build time
// ldflags "-X github.com/clover0/issue-agent/cli.version=1.0.0)"
var version = "development"

func VersionCommand() error {
	fmt.Printf("Version: %s\n", version)
	return nil
}

package cli

// ParseArgFlags parses the argument and flags from command line inputs.
// Separate [inputs] to`arg` and `[flags]`.
func ParseArgFlags(argAndFlags []string) (string, []string) {
	if len(argAndFlags) == 0 {
		return "", []string{}
	}
	if len(argAndFlags) == 1 {
		return argAndFlags[0], []string{}
	}

	return argAndFlags[0], argAndFlags[1:]
}

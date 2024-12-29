package cli

import (
	"flag"
)

type CommonInput struct {
	Config string
}

func addCommonFlags(fs *flag.FlagSet, cfg *CommonInput) {
	fs.StringVar(&cfg.Config, "config", "", "Path to the configuration file. Default is `config/default_config.yaml`")
}

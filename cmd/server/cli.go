package main

import (
	"flag"
	"fmt"
	"os"
	"slices"

	"github.com/alexpls/untils_go/internal/must"
)

var validSubcommands = []string{"serve", "seed"}

func subcommand() string {
	if len(os.Args) < 2 || !slices.Contains(validSubcommands, os.Args[1]) {
		var allowedStr string
		for i, a := range validSubcommands {
			allowedStr += "'" + a + "'"
			if i < len(validSubcommands)-1 {
				allowedStr += " or "
			}
		}
		panic(fmt.Sprintf("unknown subcommand. specify %s.", allowedStr))
	}
	return os.Args[1]
}

func parseServe() (*config, *serveConfig) {
	must.True(os.Args[1] == "serve")

	f := flag.NewFlagSet("serve", flag.ExitOnError)

	gc := config{}
	globalFlags(&gc, f)

	sc := serveConfig{}
	serveFlags(&sc, f)

	f.Parse(os.Args[2:])

	validateGlobalConfig(&gc)
	validateServeConfig(&sc)

	return &gc, &sc
}

func parseSeed() *config {
	must.True(os.Args[1] == "seed")

	f := flag.NewFlagSet("seed", flag.ExitOnError)

	gc := config{}
	globalFlags(&gc, f)

	f.Parse(os.Args[2:])

	validateGlobalConfig(&gc)

	return &gc
}

func globalFlags(c *config, f *flag.FlagSet) {
	f.StringVar(&c.env, "env", "prod", "environment (dev, prod)")
	f.StringVar(&c.dbUrl, "db", "", "postgresql connection url")
	f.StringVar(&c.xAIKey, "xai-key", "", "x.ai API key")
	f.StringVar(&c.openAIKey, "openai-key", "", "OpenAI API key")
	f.StringVar(&c.pushoverKey, "pushover-key", "", "Pushover API key")
}

func serveFlags(c *serveConfig, f *flag.FlagSet) {
	f.IntVar(&c.port, "port", 4200, "http server port")
}

func validateGlobalConfig(c *config) {
	if c.env != "prod" && c.env != "dev" {
		panic("env must be either prod or dev")
	}
}

func validateServeConfig(c *serveConfig) {
	// no validations yet
}

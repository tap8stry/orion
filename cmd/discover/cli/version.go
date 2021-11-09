package cli

import (
	"context"
	"flag"
	"fmt"

	"github.com/peterbourgon/ff/v3/ffcli"
	goVersion "go.hein.dev/go-version"
)

var (
	shortened = false
	version   = "dev"
	commit    = "none"
	date      = "unknown"
	output    = "json"
)

//Version :
func Version() *ffcli.Command {
	var (
		flagset = flag.NewFlagSet("tapestry-discover version", flag.ExitOnError)
	)
	return &ffcli.Command{
		Name:       "version",
		ShortUsage: "tapestry-discover version",
		ShortHelp:  "Prints the tapestry-discover version",
		FlagSet:    flagset,
		Exec: func(ctx context.Context, args []string) error {
			resp := goVersion.FuncWithOutput(shortened, version, commit, date, output)
			fmt.Print(resp)
			return nil
		},
	}
}

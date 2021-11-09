//
// Copyright 2020 IBM Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

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
		flagset = flag.NewFlagSet("orion version", flag.ExitOnError)
	)
	return &ffcli.Command{
		Name:       "version",
		ShortUsage: "orion version",
		ShortHelp:  "Prints the orion version",
		FlagSet:    flagset,
		Exec: func(ctx context.Context, args []string) error {
			resp := goVersion.FuncWithOutput(shortened, version, commit, date, output)
			fmt.Print(resp)
			return nil
		},
	}
}

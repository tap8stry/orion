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
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/pkg/errors"
	"github.com/tap8stry/orion/pkg/common"
	"github.com/tap8stry/orion/pkg/engine"
)

//Discover :
func Discover() *ffcli.Command {
	var (
		flagset    = flag.NewFlagSet("discover", flag.ExitOnError)
		dockerfile = flagset.String("d", "", "dockerfile pathname")
		image      = flagset.String("i", "", "image name:tag")
		namespace  = flagset.String("n", "", "SBOM namespace")
		outputfp   = flagset.String("f", "", "output file path, default: ./result.spdx")
		format     = flagset.String("o", "", "output format (json, spdx) default: spdx")
	)
	return &ffcli.Command{
		Name:       "discover",
		ShortUsage: "orion discover -d <dockerfile pathname> -i <iamge name:tag> -n <sbom namespace> -f <output filepath> -o <format>",
		ShortHelp:  `Discover software dependencies`,
		LongHelp: `Discover software dependencies not managed by package managers
EXAMPLES
  # discover all dependencies not managed by package managers
  orion discover -d ./Dockerfile -i binderancient:latest -n https://github.com/myorg/myproject -f result.spdx -o spdx
`,
		FlagSet: flagset,
		Exec: func(ctx context.Context, args []string) error {

			dopts := common.DiscoverOpts{
				DockerfilePath: *dockerfile,
				Image:          *image,
				Namespace:      *namespace,
				OutFilepath:    strings.TrimSpace(*outputfp),
				Format:         *format,
			}

			if err := DiscoveryDeps(ctx, dopts); err != nil {
				return errors.Wrapf(err, "discovery task for %s failed", dopts.DockerfilePath)
			}

			return nil
		},
	}
}

//DiscoveryDeps :
func DiscoveryDeps(ctx context.Context, dopts common.DiscoverOpts) error {
	b, _ := json.Marshal(dopts)
	fmt.Printf("\nStart discovery with inputs: %q", string(b))
	engine.StartDiscovery(context.Background(), dopts)
	return nil
}

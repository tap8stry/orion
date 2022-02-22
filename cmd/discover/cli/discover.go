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
	goVersion "go.hein.dev/go-version"
)

//Discover :
func Discover() *ffcli.Command {
	var (
		flagset    = flag.NewFlagSet("discover", flag.ExitOnError)
		dockerfile = flagset.String("f", "", "dockerfile pathname")
		image      = flagset.String("i", "", "image name:tag")
		namespace  = flagset.String("n", "", "SBOM namespace")
		outputfp   = flagset.String("r", "", "output file path, default: result")
		format     = flagset.String("o", "cdx", "output format (json, spdx, cdx), default: cdx")
		savetrace  = flagset.Bool("s", false, "save trace report, default: false")
	)
	return &ffcli.Command{
		Name:       "discover",
		ShortUsage: "orion discover -f <dockerfile pathname> -i <iamge name:tag> -k <ibmcloud apikey if using ibmcloud cr> -n <sbom namespace> -r <output filepath> -o <format> -s <save traces>",
		ShortHelp:  `Discover software dependencies`,
		LongHelp: `Discover software dependencies not managed by package managers
EXAMPLES
  # discover all dependencies not managed by package managers
  orion discover -f ./Dockerfile -i binderancient:latest -n https://github.com/myorg/myproject -r result.spdx -o spdx
  (for images in ibmcloud container registry)
  orion discover -f ./Dockerfile -i us.icr.io/binderancient:latest -k OrwMix_u6MOuU1-tENTewGtp2v9 -n https://github.com/myorg/myproject -r result.spdx -o spdx
`,
		FlagSet: flagset,
		Exec: func(ctx context.Context, args []string) error {

			dopts := common.DiscoverOpts{
				DockerfilePath: *dockerfile,
				Image:          *image,
				Namespace:      *namespace,
				OutFilepath:    strings.TrimSpace(*outputfp),
				Format:         *format,
				SaveTrace:      *savetrace,
			}

			v := goVersion.Func(shortened, version, commit, date)
			var vjson = goVersion.Info{}
			json.Unmarshal([]byte(v), &vjson)
			if err := DiscoveryDeps(ctx, dopts, vjson.Version); err != nil {
				return errors.Wrapf(err, "discovery task for %s failed", dopts.DockerfilePath)
			}

			return nil
		},
	}
}

//DiscoveryDeps :
func DiscoveryDeps(ctx context.Context, dopts common.DiscoverOpts, version string) error {
	b, _ := json.Marshal(dopts)
	fmt.Printf("\nStart discovery tool verion %s with inputs: %q", version, string(b))
	engine.StartDiscovery(context.Background(), dopts, version)
	return nil
}

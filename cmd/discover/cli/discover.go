package cli

import (
	"context"
	"encoding/json"
	"flag"

	"github.com/google/go-containerregistry/pkg/logs"
	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/pkg/errors"
	"github.com/tapestry-discover/pkg/common"
	"github.com/tapestry-discover/pkg/engine"
)

//Discover :
func Discover() *ffcli.Command {
	var (
		flagset    = flag.NewFlagSet("run", flag.ExitOnError)
		dockerfile = flagset.String("d", "", "dockerfile pathname")
		image      = flagset.String("i", "", "image name:tag")
		namespace  = flagset.String("n", "", "SBOM namespace")
		giturl     = flagset.String("g", "", "git url for the source code")
		revision   = flagset.String("r", "", "revision/branch name")
		commitid   = flagset.String("c", "", "commit id for the code")
		outputfp   = flagset.String("f", "", "output file path, default: ./result.json")
		format     = flagset.String("o", "", "output format (json, spdx) default: json")
	)
	return &ffcli.Command{
		Name:       "run",
		ShortUsage: "tapestry-discover run -d <dockerfile pathname> -i <iamge name:tag> -n <sbom namespace> -g <git-url> -r <git-revision> -c <commit-id> -f <output filepath> -o <format>",
		ShortHelp:  `Discover software dependencies`,
		LongHelp: `Discover software dependencies not managed by package managers
EXAMPLES
  # discover all dependencies not managed by package managers
  tapestry-discover run -d ./Dockerfile -i binderancient:latest -n https://github.com/myorg/myproject -f result.json -o json
`,
		FlagSet: flagset,
		Exec: func(ctx context.Context, args []string) error {

			dopts := common.DiscoverOpts{
				GitURL:         *giturl,
				GitRevision:    *revision,
				GitCommitID:    *commitid,
				DockerfilePath: *dockerfile,
				Image:          *image,
				Namespace:      *namespace,
				OutFilepath:    *outputfp,
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
	logs.Debug.Printf("Start discovery with inputs: %q", string(b))
	engine.StartDiscovery(context.Background(), dopts)
	return nil
}

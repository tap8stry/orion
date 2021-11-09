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

package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/tap8stry/orion/pkg/common"
	"github.com/tap8stry/orion/pkg/parser/addon"
	"github.com/tap8stry/orion/pkg/parser/dockerfile"
)

//StartDiscovery : entrypoint for discovery core function
func StartDiscovery(ctx context.Context, dopts common.DiscoverOpts) error {
	//get Dockerfile
	dfile, err := dockerfile.GetDockerfile(dopts.DockerfilePath)
	if err != nil {
		return err
	}
	fmt.Printf("\n got dockerfile = %s", dfile.Filepath)

	var spdxReport string

	//get add-on traces per build stage
	for j, stage := range dfile.BuildStages {
		installTraces, envs, image := addon.DiscoverAddonArtifacts(&stage, dopts, dfile.BuildArgs)
		dfile.BuildStages[j].AddOnInstalls = append(dfile.BuildStages[j].AddOnInstalls, installTraces...)
		dfile.BuildStages[j].EnvVariables = envs
		dfile.BuildStages[j].Image = image
		fmt.Printf("\ngenerating addon traces for dockerfile = %q, stage = %q, #addons = %d", dfile.Filepath, stage.StageID, len(installTraces))
	}
	//save traces
	filename := fmt.Sprintf("%s-trace.%s", common.DefaultFilename, common.FormatJSON)
	if len(dopts.OutFilepath) > 0 {
		filename = fmt.Sprintf("%s-trace.%s", dopts.OutFilepath[:strings.LastIndex(dopts.OutFilepath, ".")], common.FormatJSON)
	}
	data, _ := json.MarshalIndent(dfile, "", "  ")
	common.SaveFile(filename, data)

	//verify and produce SPDX if image provided
	if len(dopts.Image) > 0 {
		fmt.Printf("\nget image %q for dockerfile %q", dopts.Image, dfile.Filepath)
		buildContextDir, err := ioutil.TempDir(os.TempDir(), "build-ctx")
		if err != nil {
			fmt.Printf("\nerror creating build context dir: %s", err.Error())
			return err
		}
		defer os.RemoveAll(buildContextDir)
		artifacts, err := addon.VerifyAddOnInstalls(buildContextDir, dopts.Image, &dfile.BuildStages[len(dfile.BuildStages)-1])
		if err != nil {
			fmt.Printf("\nerror verifying addon installs: %s", err.Error())
			return err
		}
		spdxReport, err = addon.GenerateSpdxReport(dfile.Filepath, dopts.Image, dopts.Namespace, artifacts)
	}
	//save spdx report
	filename = fmt.Sprintf("%s.%s", common.DefaultFilename, common.FormatSpdx)
	if len(dopts.OutFilepath) > 0 {
		filename = fmt.Sprintf("%s.%s", dopts.OutFilepath[:strings.LastIndex(dopts.OutFilepath, ".")], common.FormatSpdx)
	}
	common.SaveFile(filename, []byte(spdxReport))
	return nil
}

//  ###############################################################################
//  # Licensed Materials - Property of IBM
//  # (c) Copyright IBM Corporation 2021. All Rights Reserved.
//  #
//  # Note to U.S. Government Users Restricted Rights:
//  # Use, duplication or disclosure restricted by GSA ADP Schedule
//  # Contract with IBM Corp.
//  ###############################################################################

package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/google/go-containerregistry/pkg/logs"
	"github.com/tapestry-discover/pkg/common"
	"github.com/tapestry-discover/pkg/parser/addon"
	"github.com/tapestry-discover/pkg/parser/dockerfile"
)

//StartDiscovery : entrypoint for discovery core function
func StartDiscovery(ctx context.Context, dopts common.DiscoverOpts) error {
	//get Dockerfile
	dfile, err := dockerfile.GetDockerfile(dopts.DockerfilePath)
	if err != nil {
		return err
	}
	logs.Progress.Printf(" got dockerfile = %s", dfile.Filepath)

	var spdxReport string

	//get add-on traces per build stage
	for j, stage := range dfile.BuildStages {
		installTraces, envs, image := addon.DiscoverAddonArtifacts(&stage, dopts, dfile.BuildArgs)
		dfile.BuildStages[j].AddOnInstalls = append(dfile.BuildStages[j].AddOnInstalls, installTraces...)
		dfile.BuildStages[j].EnvVariables = envs
		dfile.BuildStages[j].Image = image
		logs.Progress.Printf("generating addon traces for dockerfile = %q, stage = %q, #addons = %d", dfile.Filepath, stage.StageID, len(installTraces))
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
		logs.Debug.Printf("get image %q for dockerfile %q", dopts.Image, dfile.Filepath)
		buildContextDir, err := ioutil.TempDir(os.TempDir(), "build-ctx")
		if err != nil {
			logs.Warn.Printf("error creating build context dir: %s", err.Error())
			return err
		}
		defer os.RemoveAll(buildContextDir)
		artifacts, err := addon.VerifyAddOnInstalls(buildContextDir, dopts.Image, &dfile.BuildStages[len(dfile.BuildStages)-1])
		if err != nil {
			logs.Warn.Printf("error verifying addon installs: %s", err.Error())
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

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

package dockerfile

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/tap8stry/orion/pkg/common"
	"golang.org/x/crypto/sha3"
)

const (
	FROM    = "from"
	RUN     = "run"
	COPY    = "copy"
	ADD     = "add"
	WORKDIR = "workdir"
	ARG     = "arg"
	ENV     = "env"
)

//GetDockerfileReader reads a file into Dockerfile parser
func GetDockerfileReader(filepath string) (*parser.Result, error) {
	file, err := os.Open(filepath)
	if err != nil {
		fmt.Printf("\nerror opening dockerfile: %s", err.Error())
		return nil, err
	}

	res, err := parser.Parse(file)
	if err != nil {
		fmt.Printf("\nerror parsing dockerfile: %s", err.Error())
		return nil, err
	}

	return res, nil
}

//GetDockerfile: gets dcokerfile
func GetDockerfile(f string) (common.Dockerfile, error) {
	cm := common.Dockerfile{}
	cm.Filepath = f
	data, err := ioutil.ReadFile(f)
	if err != nil {
		fmt.Printf("\nerror reading dockerfile %q: %s", f, err.Error())
		return cm, err
	}
	cm.Filehash = fmt.Sprintf("%x", sha3.Sum256(data))
	cm.FileType = common.DockerfileEcosystem
	cm.BuildArgs, _ = DiscoverBuildArgs(f)
	cm.BuildStages, _ = DiscoverBuildStages(f, cm.Filehash)
	return cm, nil

}

//DiscoverDockerfile scans a directory and returns array of dockerfiles
func DiscoverDockerfile(repoDir, filePattern string) []common.Dockerfile {
	d := []common.Dockerfile{}
	files := common.SearchFiles(repoDir, filePattern)
	for _, f := range files {
		cm := common.Dockerfile{}
		cm.Filepath = strings.Split(f, repoDir)[1]
		cm.Filepath = strings.TrimLeft(cm.Filepath, "/")
		data, _ := ioutil.ReadFile(f)
		cm.Filehash = fmt.Sprintf("%x", sha3.Sum256(data))
		cm.FileType = common.DockerfileEcosystem
		cm.BuildArgs, _ = DiscoverBuildArgs(f)
		cm.BuildStages, _ = DiscoverBuildStages(f, cm.Filehash)
		d = append(d, cm)
	}
	return d
}

//DiscoverBuildArgs returns a map of docker build arguments defined before stages (FROM)
func DiscoverBuildArgs(dockerfp string) (map[string]string, error) {
	docker, err := GetDockerfileReader(dockerfp)
	if err != nil {
		return nil, errors.New("unable to read the dockerfile")
	}
	buildargs := make(map[string]string)
	for _, cmd := range docker.AST.Children {
		if strings.EqualFold(cmd.Value, ARG) {
			args := ParseArgEnv(cmd.Original)
			for key, value := range args {
				buildargs[key] = value
			}
		}
		if strings.EqualFold(cmd.Value, FROM) {
			break
		}
	}
	return buildargs, nil
}

//ParseArgEnv parses an ARG/ENV instruction and returns variable name and value
func ParseArgEnv(cmd string) map[string]string {
	args := make(map[string]string)
	cmdTokens := strings.Fields(cmd)
	if hasEqualMark(cmdTokens[1:]) { //use ARG name=value or ENV name=value
		for i := range cmdTokens[1:] {
			splits := strings.Split(cmdTokens[i+1], "=")
			args[splits[0]] = common.TrimQuoteMarks(splits[1])
		}
	} else { //use ARG name value or ENV name value
		if len(cmdTokens) == 3 {
			args[cmdTokens[1]] = cmdTokens[2]
		} else {
			args[cmdTokens[1]] = ""
		}
	}
	return args
}

func hasEqualMark(strs []string) bool {
	for _, str := range strs {
		if !strings.Contains(str, "=") {
			return false
		}
	}
	return true
}

//DiscoverBuildStages :
func DiscoverBuildStages(dockerfp, fileKey string) ([]common.BuildStage, error) {
	result, err := GetDockerfileReader(dockerfp)
	if err != nil {
		return nil, errors.New("unable to read the dockerfile")
	}
	stages := []common.BuildStage{}
	curStage := common.BuildStage{}
	lineIdx := 0
	buildStageIdx := 0

	for _, cmd := range result.AST.Children {
		lineIdx++
		if strings.EqualFold(cmd.Value, FROM) {
			if curStage.StageID != "" {
				curStage.EndLineNo = lineIdx
				stages = append(stages, curStage)
				buildStageIdx++
			}
			curStage = common.BuildStage{}

			if strings.Contains(strings.ToLower(cmd.Original), " as ") {
				cmdTokens := strings.Fields(cmd.Original)
				curStage.StageID = cmdTokens[len(cmdTokens)-1]
				curStage.Context = fmt.Sprintf("%s:%s", fileKey, curStage.StageID)
			} else {
				curStage.StageID = fmt.Sprintf("%d", buildStageIdx)
				curStage.Context = fmt.Sprintf("%s:%s", fileKey, curStage.StageID)
			}

			if strings.Contains(cmd.Original, "scratch") {
				curStage.ScratchBuild = true
				curStage.Image.Name = "scratch"
				curStage.Image.Tag = ""
				curStage.Image.SHA256 = "scratch_image_key"
			} else {
				curStage.ScratchBuild = false
				cmdTokens := strings.Fields(cmd.Original)
				imageName := cmdTokens[1]
				if strings.IndexAny(imageName, ":") > 0 {
					curStage.Image.Name = strings.Split(imageName, ":")[0]
					curStage.Image.Tag = strings.Split(imageName, ":")[1]
					if strings.Contains(curStage.Image.Tag, "@sha256") {
						curStage.Image.SHA256 = strings.Split(curStage.Image.Tag, "@")[1]
					}
				} else {
					curStage.Image.Name = imageName
					curStage.Image.Tag = "latest"
				}
			}
			curStage.StartLineNo = lineIdx
		} else if strings.EqualFold(cmd.Value, COPY) {
			copyFlags := cmd.Flags
			for _, flag := range copyFlags {
				if !strings.Contains(flag, "--from") {
					continue
				}
				parentStageID := strings.Split(flag, "=")[1]
				parentStage, err := getParentStage(stages, parentStageID)
				if err != nil {
					fmt.Printf("\nerror parsing dockerfile: %v", err)
				}
				curStage.DependsOn = parentStage.StageID
			}
		}
		curStage.DockerFileCmds = append(curStage.DockerFileCmds, cmd)
	}

	curStage.EndLineNo = lineIdx
	stages = append(stages, curStage)
	return stages, nil
}

func getParentStage(stages []common.BuildStage, stageID string) (common.BuildStage, error) {
	for _, stage := range stages {
		if reflect.DeepEqual(stage.StageID, stageID) {
			return stage, nil
		}
	}
	return common.BuildStage{}, fmt.Errorf("stage %v not found", stageID)
}

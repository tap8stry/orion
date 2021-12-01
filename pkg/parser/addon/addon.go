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

package addon

import (
	"strings"

	"github.com/tap8stry/orion/pkg/common"
	"github.com/tap8stry/orion/pkg/parser/dockerfile"
)

const (
	CURL        = "curl"
	WGET        = "wget"
	TAR         = "tar"
	UNZIP       = "unzip"
	CP          = "cp"
	MV          = "mv"
	CD          = "cd"
	MKDIR       = "mkdir"
	GIT         = "git"
	GITCLONE    = "git clone"
	GITCHECKOUT = "git checkout"
	COPY        = "COPY"
	ADD         = "ADD"
)

// DiscoverAddonArtifacts returns a list of artifaces installed by RUN curl/wget commands
func DiscoverAddonArtifacts(buildStage *common.BuildStage, dopts common.DiscoverOpts, buildArgs map[string]string) ([]common.InstallTrace, map[string]string, common.Image) {
	installTraces := []common.InstallTrace{}
	workdir := "/"
	stageArgsEnvs := make(map[string]string)
	for k, v := range buildArgs { //copy the build Args
		stageArgsEnvs[k] = v
	}
	stageEnvs := make(map[string]string)

	for _, cmd := range buildStage.DockerFileCmds {
		if strings.EqualFold(cmd.Value, dockerfile.ARG) { //get ARGs
			args := dockerfile.ParseArgEnv(cmd.Original)
			for key, value := range args {
				if len(value) > 0 { // new ARG or new value to build ARGs
					stageArgsEnvs[key] = replaceArgEnvVariable(value, stageArgsEnvs)
				}
			}
		}
		if strings.EqualFold(cmd.Value, dockerfile.ENV) { //get ENVs
			envs := dockerfile.ParseArgEnv(cmd.Original)
			for key, value := range envs {
				value = replaceArgEnvVariable(value, stageArgsEnvs) //for ENV value referencing other args or envs
				stageArgsEnvs[key] = value
				stageEnvs[key] = value
			}
		}

		if strings.EqualFold(cmd.Value, dockerfile.WORKDIR) { //get WORKDIR
			workdir = replaceArgEnvVariable(cmd.Next.Value, stageArgsEnvs)
			continue
		}

		if strings.EqualFold(cmd.Value, dockerfile.RUN) &&
			(strings.Contains(cmd.Next.Value, CURL) ||
				strings.Contains(cmd.Next.Value, WGET) ||
				strings.Contains(cmd.Next.Value, GIT)) { // process RUN curl/wget/git

			installs := generateCurlWgetGitTraces(workdir, cmd.Next.Value, stageArgsEnvs)
			if len(installs) > 0 {
				installTraces = append(installTraces, installs...)
			}
		}

		if strings.EqualFold(cmd.Value, dockerfile.COPY) || strings.EqualFold(cmd.Value, dockerfile.ADD) { // process COPY/ADD
			installs := generateCopyAddTraces(workdir, cmd.Original, dopts.Namespace, stageArgsEnvs)
			if installs != nil {
				installTraces = append(installTraces, installs...)
			}
		}
	}
	//update base image with the Args values
	buildStage.Image.Name = replaceArgEnvVariable(buildStage.Image.Name, stageArgsEnvs)
	buildStage.Image.Tag = replaceArgEnvVariable(buildStage.Image.Tag, stageArgsEnvs)
	return installTraces, stageEnvs, buildStage.Image
}

// generateCurlWgetGitTraces produces the traces of one RUN of "curl" or/and "wget" install commands
func generateCurlWgetGitTraces(workdir, cmd string, stageargs map[string]string) []common.InstallTrace {
	installTraces := []common.InstallTrace{}
	installsets := parseSubcommands(cmd)
	currentdir := workdir

	for index := range installsets {
		installTrace := common.InstallTrace{}
		m := make(map[int]common.Trace)
		j := 0
		gitcloneUrl := ""

		for k := 0; k < len(installsets[index].Commands); k++ {
			subCmd := installsets[index].Commands[k]
			args := parseLine(subCmd, " ")
			switch args[0] {
			case CURL:
				installTrace.Origin, m[j] = processCurl(args, currentdir, stageargs)
				j++
			case WGET:
				installTrace.Origin, m[j] = processWget(args, currentdir, stageargs)
				j++
			case GIT:
				if len(args) > 2 && strings.EqualFold(args[1], strings.Fields(GITCLONE)[1]) {
					installTrace.Origin, m[j] = processGitClone(args, currentdir, stageargs)
					j++
					gitcloneUrl = installTrace.Origin
				}
				if len(args) > 2 && strings.EqualFold(args[1], strings.Fields(GITCHECKOUT)[1]) {
					m[j] = processGitCheckout(args, currentdir, gitcloneUrl, stageargs)
					j++
				}
			case TAR:
				trace := processTar(args, currentdir, stageargs)
				if len(trace.Source) > 0 {
					if existInInstallTrace(m, trace.Source) { //belongs to the current install
						m[j] = trace
						j++
					} else {
						checkEarlierInstalls(&installTraces, trace)
					}
				}
			case UNZIP:
				trace := processUnzip(args, currentdir, stageargs)
				if len(trace.Source) > 0 {
					if existInInstallTrace(m, trace.Source) {
						m[j] = trace
						j++
					} else {
						checkEarlierInstalls(&installTraces, trace)
					}
				}
			case CP:
				trace := processCp(args, currentdir, stageargs)
				if len(trace.Source) > 0 {
					if existInInstallTrace(m, trace.Source) {
						m[j] = trace
						j++
					} else {
						checkEarlierInstalls(&installTraces, trace)
					}
				}
			case MV:
				trace := processMv(args, currentdir, stageargs)
				if len(trace.Source) > 0 {
					if existInInstallTrace(m, trace.Source) {
						m[j] = trace
						j++
					} else {
						checkEarlierInstalls(&installTraces, trace)
					}
				}
			case CD: //update the current dir
				currentdir = processCd(args, currentdir, stageargs)
				/*case MKDIR: //add a new possible destination
				trace := processMkdir(args, currentdir, stageargs)
				m[j] = trace
				j++ */
			}
		}
		if len(installTrace.Origin) > 0 && len(m) > 0 {
			installTrace.Traces = m
			installTraces = append(installTraces, installTrace)
		}
	}
	return installTraces
}

// generateCopyTraces produces the traces of one RUN of "curl" or/and "wget" install commands
func generateCopyAddTraces(workdir, cmd, namespace string, stageargs map[string]string) []common.InstallTrace {
	installTraces := []common.InstallTrace{}
	args := parseLine(cmd, " ")
	installTrace, err := processCopyAdd(args, workdir, namespace, stageargs)
	if err == nil {
		installTraces = append(installTraces, installTrace)
	}
	return installTraces
}

// parseLine parses a line by the separator into an array and trims spaces and quotation marks
func parseLine(line, separator string) []string {
	cmds := strings.Split(line, separator)
	newCmds := []string{}
	for i := range cmds {
		for true { //remove all tabs
			cmds[i] = strings.ReplaceAll(cmds[i], "\t", "")
			if !strings.Contains(cmds[i], "\t") {
				break
			}
		}
		cmds[i] = strings.Trim(cmds[i], " ")
		if len(cmds[i]) > 0 {
			newCmds = append(newCmds, cmds[i])
		}
	}
	return newCmds
}

// parseSubcommands parse shell commands in a docker RUN operation
func parseSubcommands(line string) []common.CommandSet {
	sets := []common.CommandSet{}
	cmdSetMap := common.CommandSet{}
	m := make(map[int]string)
	first := true
	exclude := true //ignore subcmds before the first CURL/WGET/GIT
	j := 0

	separator := "&&"                                                    //default
	if !strings.Contains(line, "&&") && strings.Contains(line, "; \t") { //some shell scripts use '; \' as end of a command
		separator = "; \t"
	}

	cmds := parseLine(line, separator)
	for i := range cmds {
		if strings.HasPrefix(cmds[i], CURL) || strings.HasPrefix(cmds[i], WGET) || strings.HasPrefix(cmds[i], GITCLONE) {
			exclude = false
			if first {
				first = false
			} else {
				cmdSetMap.Commands = m
				sets = append(sets, cmdSetMap)
				m = make(map[int]string)
				j = 0
			}
			m[j] = cmds[i]
			j++
		} else {
			if !exclude {
				m[j] = cmds[i]
				j++
			}
		}
	}
	cmdSetMap.Commands = m
	sets = append(sets, cmdSetMap)
	return sets
}

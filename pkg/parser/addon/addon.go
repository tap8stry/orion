//  ###############################################################################
//  # Licensed Materials - Property of IBM
//  # (c) Copyright IBM Corporation 2021. All Rights Reserved.
//  #
//  # Note to U.S. Government Users Restricted Rights:
//  # Use, duplication or disclosure restricted by GSA ADP Schedule
//  # Contract with IBM Corp.
//  ###############################################################################

package addon

import (
	"strings"

	"github.com/tapestry-discover/pkg/common"
	"github.com/tapestry-discover/pkg/parser/dockerfile"
)

const (
	CURL        = "curl"
	WGET        = "wget"
	TAR         = "tar"
	CP          = "cp"
	MV          = "mv"
	CD          = "cd"
	GIT         = "git"
	GITCLONE    = "git clone"
	GITCHECKOUT = "git checkout"
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
			installs := generateCopyAddTraces(workdir, cmd.Original, dopts.Namespace, dopts.GitCommitID, stageArgsEnvs)
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

	installsets := parseSubcommands(cmd, "&&")
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
				if len(trace.Source) > 0 && existInInstallTrace(m, trace.Source) {
					m[j] = trace
					j++
				}
			case CP:
				trace := processCp(args, currentdir, stageargs)
				if len(trace.Source) > 0 && existInInstallTrace(m, trace.Source) {
					m[j] = trace
					j++
				}
			case MV:
				trace := processMv(args, currentdir, stageargs)
				if len(trace.Source) > 0 && existInInstallTrace(m, trace.Source) {
					m[j] = trace
					j++
				}
			case CD: //update the current dir
				currentdir = processCd(args, currentdir, stageargs)
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
func generateCopyAddTraces(workdir, cmd, giturl, commitid string, stageargs map[string]string) []common.InstallTrace {
	installTraces := []common.InstallTrace{}
	args := parseLine(cmd, " ")
	installTrace, err := processCopyAdd(args, workdir, giturl, commitid, stageargs)
	if err == nil {
		installTraces = append(installTraces, installTrace)
	}
	return installTraces
}

// parseLine parses a line by the separator into an array and trims spaces and quotation marks
func parseLine(line, separator string) []string {
	cmds := strings.Split(line, separator)
	newCmds := []string{}
	for i, str := range cmds {
		cmds[i] = strings.Trim(str, " ")         //trim space
		cmds[i] = common.TrimQuoteMarks(cmds[i]) //trim quotation marks
		if !strings.EqualFold(cmds[i], "") {
			newCmds = append(newCmds, cmds[i])
		}
	}
	return newCmds
}

// parseSubcommands parses a command (a subcommand in docker run)
func parseSubcommands(line, sep string) []common.CommandSet {
	sets := []common.CommandSet{}
	cmdSetMap := common.CommandSet{}
	m := make(map[int]string)
	first := true
	j := 0

	cmds := parseLine(line, sep)
	for i := range cmds {
		if strings.HasPrefix(cmds[i], CURL) || strings.HasPrefix(cmds[i], WGET) || strings.HasPrefix(cmds[i], GITCLONE) {
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
			m[j] = cmds[i]
			j++
		}
	}
	cmdSetMap.Commands = m
	sets = append(sets, cmdSetMap)
	return sets
}

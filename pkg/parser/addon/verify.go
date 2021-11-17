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
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/tap8stry/orion/pkg/common"
	"github.com/tap8stry/orion/pkg/imagefs"
	"golang.org/x/mod/sumdb/dirhash"
)

const (
	NOASSERTION = "NOASSERTION"
	NOSHA       = "checksum calculation failed. a manual review of install traces is recommended."
)

func displayDockerfile(pDfp string) {
	dat, err := os.ReadFile(pDfp)
	check(err)
	fmt.Print("\ntemp dockerfile = \n" + string(dat))
}
func check(e error) {
	if e != nil {
		panic(e)
	}
}

//VerifyAddOnInstalls :
func VerifyAddOnInstalls(buildContextDir, image string, buildStage *common.BuildStage) ([]common.VerifiedArtifact, error) {
	if len(buildStage.AddOnInstalls) == 0 {
		fmt.Printf("\nno AddOnInstalls found for build stage = %s", buildStage.StageID)
		return nil, nil
	}

	fsdir, err := imagefs.Get(image, buildContextDir)
	if err != nil {
		fmt.Printf("\nerror in creating image filesystem: %s\n", err.Error())
		return nil, err
	}
	fmt.Printf("\nverify against image's filesystem at %q", fsdir)
	verified := verifyArtifacts(buildStage.AddOnInstalls, fsdir)
	return verified, nil
}

func verifyArtifacts(installs []common.InstallTrace, fsdir string) []common.VerifiedArtifact {
	artifacts := []common.VerifiedArtifact{}
	for _, install := range installs {
		isDownload := false
		for _, trace := range install.Traces {
			if strings.EqualFold(trace.Command, CURL) ||
				strings.EqualFold(trace.Command, WGET) ||
				strings.EqualFold(trace.Command, GIT) ||
				strings.EqualFold(trace.Command, GITCHECKOUT) ||
				strings.EqualFold(trace.Command, GITCLONE) {
				isDownload = true
			}
			path, isdir, err := getPath(trace, fsdir)
			if err != nil { //skip
				continue
			}
			art := common.VerifiedArtifact{
				Name:             path,
				Path:             path,
				Version:          "",
				IsDirectory:      isdir,
				IsDownload:       isDownload,
				DownloadLocation: install.Origin,
			}
			artifacts = append(artifacts, art)
		}
	}
	fmt.Printf("\n# of verified artifacts = %d", len(artifacts))
	return artifacts
}

func getPath(trace common.Trace, dir string) (string, bool, error) {
	var err error
	des := trace.Destination
	switch strings.Fields(trace.Command)[0] {
	case "COPY":
		des = checkCOPYADDDestination(trace)
	case "ADD":
		des = checkCOPYADDDestination(trace)
	case "cp":
		des, err = checkCpDestination(trace, dir)
		if err != nil {
			return des, false, err
		}
	case "mv":
		des, err = checkMvDestination(trace, dir)
		if err != nil {
			return des, false, err
		}
	case "tar": //TODO: investigate how to determin the destination from 'tar -x'
		fmt.Printf("\ndestination unknown, skip the trace for %s, ", trace.Command)
		return des, false, errors.New("destination unknown, need manual review")
	default: //do nothing
	}

	info, err := os.Stat(dir + des)
	if os.IsNotExist(err) {
		fmt.Printf("\nfolder/file %s does not exist for verifying", dir+trace.Destination)
		return "", false, err
	}
	if strings.EqualFold(des, "/") || len(des) == 0 { // root directory
		fmt.Printf("\ndestination unknown")
		return des, info.IsDir(), errors.New("destination unknown")
	}
	if info.IsDir() {
		_, err = dirhash.HashDir(dir+des, "", dirhash.DefaultHash) //
		if err != nil {
			fmt.Printf("\nsha for folder %s cannot be calculated", dir+trace.Destination)
			return dir + des, info.IsDir(), err
		}
	}
	return dir + des, info.IsDir(), nil
}

func checkCOPYADDDestination(trace common.Trace) string {
	des := trace.Destination
	so := trace.Source

	if strings.HasSuffix(so, "/") {
		so = so[:len(so)-1]
	}
	so = so[strings.LastIndex(so, "/")+1:]

	if strings.HasSuffix(des, "/") { //des is a directory
		des += so
	}
	if strings.HasSuffix(des, "/.") {
		des = des[:len(des)-1] + so
	}
	return des
}

func checkCpDestination(trace common.Trace, dir string) (string, error) {
	des := trace.Destination
	des = strings.TrimSuffix(des, ".") //e.g. /usr/bin/. --> /usr/bin/
	des = strings.TrimSuffix(des, "/") //e.g. /usr/bin/ --> /usr/bin
	info, err := os.Stat(dir + des)    // e.g. /tmp/build-ctx00032/rootfs/usr/bin
	if os.IsNotExist(err) {
		return "", fmt.Errorf("\nfolder/file %s does not exist", dir+des)
	}
	if info.IsDir() { // destination is a directory
		sostrs := strings.Split(trace.Source, "/")
		so := sostrs[len(sostrs)-1]
		des += "/" + so
	}
	return des, nil
}

func checkMvDestination(trace common.Trace, dir string) (string, error) {
	des := trace.Destination
	sostrs := strings.Split(trace.Source, "/")
	so := sostrs[len(sostrs)-1]
	if strings.HasSuffix(des, "/.") { //e.g. /usr/bin/.
		des += "/" + so
	}
	return des, nil
}

func GeneratePartialDockerData(buildArgs map[string]string, cmds []*parser.Node) string {
	data := ""
	for k, v := range buildArgs {
		data += "ARG " + k + "=" + v + "\n"
	}
	for _, cmd := range cmds {
		data += cmd.Original + "\n"
	}
	return data
}

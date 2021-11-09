/*###############################################################################
  # Licensed Materials - Property of IBM
  # (c) Copyright IBM Corporation 2020. All Rights Reserved.
  #
  # Note to U.S. Government Users Restricted Rights:
  # Use, duplication or disclosure restricted by GSA ADP Schedule
  # Contract with IBM Corp.
  ###############################################################################  */
package addon

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/google/go-containerregistry/pkg/logs"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/tapestry-discover/pkg/common"
	"github.com/tapestry-discover/pkg/dind"
	"golang.org/x/mod/sumdb/dirhash"
)

const (
	NOASSERTION = "NOASSERTION"
	NOSHA       = "checksum calculation failed. a manual review of install traces is recommended."
)

func displayDockerfile(pDfp string) {
	dat, err := os.ReadFile(pDfp)
	check(err)
	fmt.Print("temp dockerfile = \n" + string(dat))
}
func check(e error) {
	if e != nil {
		panic(e)
	}
}

//VerifyAddOnInstalls :
func VerifyAddOnInstalls(buildContextDir, image string, buildStage *common.BuildStage) ([]common.VerifiedArtifact, error) {
	if len(buildStage.AddOnInstalls) == 0 {
		logs.Progress.Printf("no AddOnInstalls found for build stage = %s", buildStage.StageID)
		return nil, nil
	}

	fsdir, err := getImageFs(image, buildContextDir)
	if err != nil {
		logs.Warn.Printf("error in creating image filesystem: %s\n", err.Error())
		return nil, err
	}
	logs.Progress.Printf("verify, image-fsdir=%s", fsdir)
	verified := verifyArtifacts(buildStage.AddOnInstalls, fsdir)
	return verified, nil
}

//getImageFs returns the file system from image build for the input dockerfile data
func getImageFs(image, buildContextDir string) (string, error) {

	/*	imageName := strings.ToLower(randomdata.SillyName())
		//build docker image
		if err := dind.BuildImage(repodir, pDfp, imageName); err != nil {
			logs.Debug.Printf("error running dind build: %v", err)
			return "", errors.New("unable to build Docker image")
		}
	*/
	//create a docker container of the image
	containerID, err := dind.CreateContainer(image)
	if err != nil {
		logs.Debug.Printf("error creatting container: %v", err)
		return "", errors.New("unable to run Dokcer container")
	}

	//export the container and untar it to the filesystem
	unpackDirRootfs := path.Join(buildContextDir, "rootfs")
	os.MkdirAll(unpackDirRootfs, 0744)
	tarfilePath := path.Join(buildContextDir, fmt.Sprintf("%s.tar.gz", image[:strings.LastIndex(image, ":")]))
	logs.Progress.Printf("export image to file system: %s", tarfilePath)
	if err := dind.ExportImageToLocalDir(tarfilePath, containerID); err != nil {
		logs.Debug.Printf("error running dind export: %v", err)
		return "", errors.New("unable to export Docker image")
	}
	logs.Progress.Printf("untar image file to: %s", unpackDirRootfs)
	if err := dind.UntarImageToLocalDir(tarfilePath, unpackDirRootfs); err != nil {
		logs.Debug.Printf("error running untar: %v", err)
		return "", errors.New("unable to extract Docker image locally")
	}
	return unpackDirRootfs, nil
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
	logs.Debug.Printf("# of verified artifacts = %d", len(artifacts))
	return artifacts
}

func getPath(trace common.Trace, dir string) (string, bool, error) {
	var err error
	des := trace.Destination
	switch trace.Command {
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
	default: //do nothing
	}

	//des = dir + des
	info, err := os.Stat(dir + des)
	if os.IsNotExist(err) {
		logs.Debug.Printf("folder/file %s does not exist for verifying", dir+trace.Destination)
		return "", false, err
	}
	if strings.EqualFold(des, "/") || len(des) == 0 { // root directory
		return des, info.IsDir(), errors.New("destination unknown")
	}
	if info.IsDir() {
		_, err = dirhash.HashDir(dir+des, "", dirhash.DefaultHash) //
		if err != nil {
			logs.Debug.Printf("sha for folder %s cannot be calculated", dir+trace.Destination)
			return dir + des, info.IsDir(), err
		}
	}
	return dir + des, info.IsDir(), nil
}

func checkCOPYADDDestination(trace common.Trace) string {
	des := trace.Destination
	if strings.HasSuffix(des, "/") { //des is a directory
		if !strings.HasSuffix(trace.Source, "/") { //source is a file
			sostrs := strings.Split(trace.Source, "/")
			so := sostrs[len(sostrs)-1] //get source filename
			des += so
		}
	}
	return des
}

func checkCpDestination(trace common.Trace, dir string) (string, error) {
	des := trace.Destination
	des = strings.TrimSuffix(des, ".") //e.g. /usr/bin/. --> /usr/bin/
	des = strings.TrimSuffix(des, "/") //e.g. /usr/bin/ --> /usr/bin
	info, err := os.Stat(dir + des)    // e.g. /tmp/build-ctx00032/rootfs/usr/bin
	if os.IsNotExist(err) {
		logs.Debug.Printf("folder/file %s does not exist", dir+des)
		return "", err
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

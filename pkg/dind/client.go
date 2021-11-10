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

package dind

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"strings"

	"github.com/Pallinder/go-randomdata"
	"github.com/docker/docker/client"
)

const (
	imagePullCmdTmpl            = `docker pull {{.ImageName}}`
	buildkitBuildCmdTmpl        = `docker build -t {{.ImageName}} -f {{.Dockerfile}} .`
	createContainerCmdTmpl      = `docker create --name {{.ContainerName}} {{.ImageName}} `
	exportImageCmdTmpl          = `docker export {{.ContainerID}} -o {{.TarPath}}`
	untarImageToLocalDirCmdTmpl = `tar -C {{.LocalDir}} -xf {{.TarPath}}`
)

//PullDockerImage : Currently we are using docker command option.
// go-sdk function for pull doesn't work for registries like
// `registry.access.redhat.com`
func PullDockerImage(name string, tag string) error {
	fmt.Printf("\nabout to pull image %s:%s", name, tag)
	var cmd bytes.Buffer
	imageName := fmt.Sprintf("%s:%s", name, tag)
	execCmd, _ := template.New("imagePull").Parse(imagePullCmdTmpl)
	if err := execCmd.Execute(&cmd, map[string]string{
		"ImageName": imageName,
	}); err != nil {
		fmt.Printf("\nerror formating docker pull exec cmd: %v", err)
		return errors.New("unable to create Docker command [pull]")
	}

	cmdTokens := strings.Fields(cmd.String())
	_, err := exec.Command(cmdTokens[0], cmdTokens[1:]...).CombinedOutput()
	if err != nil {
		fmt.Printf("\nerror executing docker: cmd %v", err)
		return errors.New("unable to pull image from Docker repository")

	}

	fmt.Printf("\nsuccessfully pulled an image %s:%s", name, tag)
	return nil
}

//GetImageDigest :
func GetImageDigest(name string, tag string) (string, error) {
	fmt.Printf("\nabout to fetch image digest for %s:%s", name, tag)
	var digest string
	if err := PullDockerImage(name, tag); err != nil {
		fmt.Printf("\nerror pulling an image: %v", err)
		return digest, errors.New("unable to retrieve Docker image digest")
	}

	cli, err := client.NewEnvClient()
	if err != nil {
		fmt.Printf("\nerror initializing client: %v", err)
		return digest, errors.New("unable to initialize Docker client interface")
	}
	imageID := fmt.Sprintf("%s:%s", name, tag)
	imgInspect, _, err := cli.ImageInspectWithRaw(context.Background(), imageID)
	if err != nil {
		fmt.Printf("\nerror retrieving image digest: %v", err)
		return digest, errors.New("unable to inspect Docker image")
	}
	digest = imgInspect.ID
	fmt.Printf("\nsuccessfully fetched image digest for %s:%s: %s", name, tag, digest)
	return digest, nil
}

//BuildImage :
func BuildImage(buildContext, dockerfile, imageName string) error {
	fmt.Printf("\nsimulating partial image build for: %s", imageName)
	var cmd bytes.Buffer
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(buildContext)
	fmt.Printf("\nbuildContext = %s", buildContext)

	execCmd, _ := template.New("buildkitBuilder").Parse(buildkitBuildCmdTmpl)
	if err := execCmd.Execute(&cmd, map[string]string{
		"ImageName":  imageName,
		"Dockerfile": dockerfile,
	}); err != nil {
		fmt.Printf("\nerror formating docker build exec cmd: %s", err.Error())
		return errors.New("unable to create Docker command [build]")
	}

	cmdTokens := strings.Fields(cmd.String())
	fmt.Printf("\ncommand to execute %s", cmdTokens)
	_, err := exec.Command(cmdTokens[0], cmdTokens[1:]...).CombinedOutput()
	if err != nil {
		fmt.Printf("\nerror executing docker cmd %v", err.Error())
		return errors.New("unable to build Docker image")

	}
	fmt.Printf("\npartial image build completed for: %s", imageName)
	return nil
}

//CreateContainer :
func CreateContainer(imageName string) (string, error) {
	fmt.Printf("\ncreating container for image: %s", imageName)
	var cmd bytes.Buffer
	containerName := strings.ToLower(randomdata.SillyName())

	execCmd, _ := template.New("createContainer").Parse(createContainerCmdTmpl)
	if err := execCmd.Execute(&cmd, map[string]string{
		"ImageName":     imageName,
		"ContainerName": containerName,
	}); err != nil {
		fmt.Printf("\nerror formating docker create exec cmd: %s", err)
		return "", errors.New("unable to create Docker command [create]")
	}

	cmdTokens := strings.Fields(cmd.String())
	outBytes, err := exec.Command(cmdTokens[0], cmdTokens[1:]...).CombinedOutput()
	if err != nil {
		fmt.Printf("\nerror executing docker cmd %v", err)
		return "", errors.New("unable to create a container for the Docker image")

	}

	type containerInspect []struct {
		ID string `json:"Id"`
	}
	var inspectOut containerInspect

	inpsectContainerCmd := fmt.Sprintf("docker inspect %s ", containerName)
	cmdTokens = strings.Fields(inpsectContainerCmd)
	outBytes, err = exec.Command(cmdTokens[0], cmdTokens[1:]...).CombinedOutput()

	if err != nil {
		fmt.Printf("\nerror executing docker inspect %v", err)
		return "", errors.New("unable to create Docker command [inspect]")
	}
	if err := json.Unmarshal(outBytes, &inspectOut); err != nil {
		fmt.Printf("\nerror parsing container inspect: %v", err)
		return "", errors.New("unable to inspect a Docker container")
	}
	containerID := inspectOut[0].ID
	fmt.Printf("\ncontainer creation completed for image: %s", imageName)
	return containerID, nil
}

//ExportImageToLocalDir :
func ExportImageToLocalDir(tarfilePath, containerID string) error {
	var cmd bytes.Buffer
	execCmd, _ := template.New("exportImage").Parse(exportImageCmdTmpl)
	if err := execCmd.Execute(&cmd, map[string]string{
		"ContainerID": containerID,
		"TarPath":     tarfilePath,
	}); err != nil {
		fmt.Printf("\nerror formating docker export exec cmd: %s", err)
		return errors.New("unable to create Docker command [export]")
	}

	cmdTokens := strings.Fields(cmd.String())
	fmt.Printf("\ncommand to execute %s", cmdTokens)
	_, err := exec.Command(cmdTokens[0], cmdTokens[1:]...).CombinedOutput()
	if err != nil {
		fmt.Printf("\nerror executing docker cmd %v", err)
		// fmt.Printf("\noutput from docker export: " + string(outBytes))
		return errors.New("unable to export a Docker container")

	}
	fmt.Printf("\nimage export to local directory completed")
	return nil
}

//UntarImageToLocalDir :
func UntarImageToLocalDir(tarfilePath, localdir string) error {
	var cmd bytes.Buffer
	execCmd, _ := template.New("untarImage").Parse(untarImageToLocalDirCmdTmpl)

	if err := execCmd.Execute(&cmd, map[string]string{
		"LocalDir": localdir,
		"TarPath":  tarfilePath,
	}); err != nil {
		fmt.Printf("\nerror formating untar exec cmd: %s", err)
		return errors.New("unable to create Docker command [tar]")
	}

	cmdTokens := strings.Fields(cmd.String())
	fmt.Printf("\ncommand to execute %s", cmdTokens)
	fmt.Printf("\nlocalDir: %s, tarFile: %s", localdir, tarfilePath)
	_, err := exec.Command(cmdTokens[0], cmdTokens[1:]...).CombinedOutput()
	if err != nil {
		fmt.Printf("\nerror executing untar cmd: %v", err.Error())
		return errors.New("unable to compress a Docker image")
	}
	return nil
}

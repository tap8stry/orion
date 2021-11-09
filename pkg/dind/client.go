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
	"github.com/google/go-containerregistry/pkg/logs"
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
	logs.Debug.Printf("about to pull image %s:%s", name, tag)
	var cmd bytes.Buffer
	imageName := fmt.Sprintf("%s:%s", name, tag)
	execCmd, _ := template.New("imagePull").Parse(imagePullCmdTmpl)
	if err := execCmd.Execute(&cmd, map[string]string{
		"ImageName": imageName,
	}); err != nil {
		logs.Debug.Printf("error formating docker pull exec cmd: %v", err)
		return errors.New("unable to create Docker command [pull]")
	}

	cmdTokens := strings.Fields(cmd.String())
	// logs.Debug.Printf("command to execute %s", cmdTokens)
	_, err := exec.Command(cmdTokens[0], cmdTokens[1:]...).CombinedOutput()
	if err != nil {
		logs.Debug.Printf("error executing docker: cmd %v", err)
		// logs.Debug.Printf("output from docker pull: " + string(outBytes))
		return errors.New("unable to pull image from Docker repository")

	}
	// ctx := context.Background()
	// cli, err := client.NewEnvClient()
	// if err != nil {
	// 	logs.Debug.Printf("error initializing client: %v", err)
	// 	return errors.New("`image pull` failed")
	// }
	// if strings.Contains(name, "/") {
	// 	imageID = fmt.Sprintf("docker.io/%s:%s", name, tag)
	// } else {
	// 	imageID = fmt.Sprintf("docker.io/library/%s:%s", name, tag)
	// }
	// out, err := cli.ImagePull(ctx, imageID, types.ImagePullOptions{})
	// if err != nil {
	// 	logs.Debug.Printf("error pulling image ", err)
	// 	return errors.New("`image pull` failed")
	// }
	// defer out.Close()
	// io.Copy(ioutil.Discard, out)
	// cli.Close()

	logs.Debug.Printf("successfully pulled an image %s:%s", name, tag)
	return nil
}

//GetImageDigest :
func GetImageDigest(name string, tag string) (string, error) {
	logs.Debug.Printf("about to fetch image digest for %s:%s", name, tag)
	var digest string
	if err := PullDockerImage(name, tag); err != nil {
		logs.Debug.Printf("error pulling an image: %v", err)
		return digest, errors.New("unable to retrieve Docker image digest")
	}

	cli, err := client.NewEnvClient()
	if err != nil {
		logs.Debug.Printf("error initializing client: %v", err)
		return digest, errors.New("unable to initialize Docker client interface")
	}
	imageID := fmt.Sprintf("%s:%s", name, tag)
	imgInspect, _, err := cli.ImageInspectWithRaw(context.Background(), imageID)
	if err != nil {
		logs.Debug.Printf("error retrieving image digest: %v", err)
		return digest, errors.New("unable to inspect Docker image")
	}
	digest = imgInspect.ID
	logs.Debug.Printf("successfully fetched image digest for %s:%s: %s", name, tag, digest)
	return digest, nil
}

//BuildImage :
func BuildImage(buildContext, dockerfile, imageName string) error {
	logs.Debug.Printf("simulating partial image build for: %s", imageName)
	var cmd bytes.Buffer
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(buildContext)
	logs.Debug.Printf("buildContext = %s", buildContext)

	execCmd, _ := template.New("buildkitBuilder").Parse(buildkitBuildCmdTmpl)
	if err := execCmd.Execute(&cmd, map[string]string{
		"ImageName":  imageName,
		"Dockerfile": dockerfile,
	}); err != nil {
		logs.Debug.Printf("error formating docker build exec cmd: %s", err.Error())
		return errors.New("unable to create Docker command [build]")
	}

	cmdTokens := strings.Fields(cmd.String())
	logs.Debug.Printf("command to execute %s", cmdTokens)
	_, err := exec.Command(cmdTokens[0], cmdTokens[1:]...).CombinedOutput()
	if err != nil {
		logs.Debug.Printf("error executing docker cmd %v", err.Error())
		return errors.New("unable to build Docker image")

	}
	logs.Debug.Printf("partial image build completed for: %s", imageName)
	return nil
}

//CreateContainer :
func CreateContainer(imageName string) (string, error) {
	logs.Debug.Printf("creating container for image: %s", imageName)
	var cmd bytes.Buffer
	containerName := strings.ToLower(randomdata.SillyName())

	execCmd, _ := template.New("createContainer").Parse(createContainerCmdTmpl)
	if err := execCmd.Execute(&cmd, map[string]string{
		"ImageName":     imageName,
		"ContainerName": containerName,
	}); err != nil {
		logs.Debug.Printf("error formating docker create exec cmd: %s", err)
		return "", errors.New("unable to create Docker command [create]")
	}

	cmdTokens := strings.Fields(cmd.String())
	// logs.Debug.Printf("command to execute %s", cmdTokens)
	outBytes, err := exec.Command(cmdTokens[0], cmdTokens[1:]...).CombinedOutput()
	if err != nil {
		logs.Debug.Printf("error executing docker cmd %v", err)
		// logs.Debug.Printf("output from docker create: " + string(outBytes))
		return "", errors.New("unable to create a container for the Docker image")

	}

	type containerInspect []struct {
		ID string `json:"Id"`
	}
	var inspectOut containerInspect

	inpsectContainerCmd := fmt.Sprintf("docker inspect %s ", containerName)
	cmdTokens = strings.Fields(inpsectContainerCmd)
	// logs.Debug.Printf("command to execute %s", cmdTokens)
	outBytes, err = exec.Command(cmdTokens[0], cmdTokens[1:]...).CombinedOutput()
	if err != nil {
		logs.Debug.Printf("error executing docker inspect %v", err)
		//logs.Debug.Printf("output from docker inspect: " + string(outBytes))
		return "", errors.New("unable to create Docker command [inspect]")

	}
	if err := json.Unmarshal(outBytes, &inspectOut); err != nil {
		logs.Debug.Printf("error parsing container inspect: %v", err)
		return "", errors.New("unable to inspect a Docker container")
	}
	containerID := inspectOut[0].ID
	logs.Debug.Printf("container creation completed for image: %s", imageName)
	return containerID, nil
}

//ExportImageToLocalDir :
func ExportImageToLocalDir(tarfilePath, containerID string) error {
	logs.Debug.Printf("exporting image to local directory: ")
	var cmd bytes.Buffer
	execCmd, _ := template.New("exportImage").Parse(exportImageCmdTmpl)
	if err := execCmd.Execute(&cmd, map[string]string{
		"ContainerID": containerID,
		"TarPath":     tarfilePath,
	}); err != nil {
		logs.Debug.Printf("error formating docker export exec cmd: %s", err)
		return errors.New("unable to create Docker command [export]")
	}

	cmdTokens := strings.Fields(cmd.String())
	logs.Debug.Printf("command to execute %s", cmdTokens)
	_, err := exec.Command(cmdTokens[0], cmdTokens[1:]...).CombinedOutput()
	if err != nil {
		logs.Debug.Printf("error executing docker cmd %v", err)
		// logs.Debug.Printf("output from docker export: " + string(outBytes))
		return errors.New("unable to export a Docker container")

	}
	logs.Debug.Printf("image export to local directory completed")
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
		logs.Debug.Printf("error formating untar exec cmd: %s", err)
		return errors.New("unable to create Docker command [tar]")
	}

	cmdTokens := strings.Fields(cmd.String())
	logs.Debug.Printf("command to execute %s", cmdTokens)
	logs.Debug.Printf("localDir: %s, tarFile: %s", localdir, tarfilePath)
	_, err := exec.Command(cmdTokens[0], cmdTokens[1:]...).CombinedOutput()
	if err != nil {
		logs.Debug.Printf("error executing untar cmd: %v", err.Error())
		//logs.Debug.Printf("output from untar: " + string(outBytes))
		return errors.New("unable to compress a Docker image")
	}
	return nil
}

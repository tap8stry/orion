package imagefs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/release/pkg/spdx"
)

const (
	untarToLocalDirCmdTmpl = `tar -C {{.LocalDir}} -xf {{.TarPath}}`
	removeTarFileCmdTmpl   = `rm {{.File}}`
)

//Get generates a file system of the image and returns file system path
func Get(imageName, buildContextDir string) (string, error) {
	rootDir := ""
	imageRefs, err := getImageReferences(imageName)
	if err != nil {
		fmt.Printf("\nerror in getImageReferences: %s", err.Error())
		return "", err
	}
	if len(imageRefs) == 0 || len(imageRefs) > 1 {
		fmt.Printf("\n%d image references found for %q", len(imageRefs), imageName)
		return "", err
	}

	for _, refData := range imageRefs {
		ref, err := name.ParseReference(refData.Digest)
		if err != nil {
			fmt.Printf("\nparsing reference %s", imageName)
			return "", err
		}

		img, err := remote.Image(ref)
		if err != nil {
			fmt.Printf("\nerror getting image %q", ref.Name())
			return "", err
		}

		rootDir, err = generateImageFileSystem(buildContextDir, img, ref)
		if err != nil {
			fmt.Printf("\nerror from getImageFileSystem(): %s", err.Error())
			return "", err
		}
	}
	fmt.Printf("\ndirectory for image %q is: %s", imageName, rootDir)
	return rootDir, nil
}

//generateImageFileSystem downloads the image and untar it to the directory
func generateImageFileSystem(unpackdir string, img v1.Image, ref name.Reference) (string, error) {
	tarfile := path.Join(unpackdir, "image.tar")
	if err := tarball.WriteToFile(tarfile, ref, img); err != nil {
		fmt.Printf("\nerror writing image to %q", tarfile)
		return "", err
	}
	fmt.Printf("\nwrote image to %q", tarfile)

	unpackDirRootfs := path.Join(unpackdir, "rootfs")
	os.MkdirAll(unpackDirRootfs, 0744)
	if err := unpackImageToFileSystem(unpackDirRootfs, tarfile); err != nil {
		fmt.Printf("\nerror unpack %q to file system %q", tarfile, unpackDirRootfs)
		return unpackDirRootfs, err
	}

	manifest, err := getManifest(path.Join(unpackDirRootfs, "manifest.json"))
	if err != nil {
		fmt.Printf("\nerror untar image %q: %s", tarfile, err.Error())
		return "", err
	}
	for _, file := range manifest.LayerFiles { //each layer is in a tar.gz file
		filepath := path.Join(unpackDirRootfs, file)
		err = unpackImageToFileSystem(unpackDirRootfs, filepath) //untar each layer
		if err != nil {
			fmt.Printf("\nerror untar image layer %q: %s", file, err.Error())
			return "", err
		}
	}
	return unpackDirRootfs, nil
}

// getImageReferences gets a reference string and returns all image
func getImageReferences(imageName string) ([]struct {
	Digest string
	Arch   string
	OS     string
}, error) {
	ref, err := name.ParseReference(imageName)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing image reference %s", imageName)
	}
	//descr, err := remote.Get(ref)
	descr, err := remote.Get(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return nil, errors.Wrap(err, "fetching remote descriptor")
	}

	images := []struct {
		Digest string
		Arch   string
		OS     string
	}{}

	// If we got a digest, we reuse it as is
	if _, ok := ref.(name.Digest); ok {
		images = append(images, struct {
			Digest string
			Arch   string
			OS     string
		}{Digest: ref.(name.Digest).String()})
		return images, nil
	}

	// If the reference is not an image, it has to work as a tag
	tag, ok := ref.(name.Tag)
	if !ok {
		return nil, errors.Errorf("could not cast tag from reference %s", imageName)
	}
	// If the reference points to an image, return it
	if descr.MediaType.IsImage() {
		logrus.Infof("Reference %s points to a single image", imageName)
		// Check if we can get an image
		im, err := descr.Image()
		if err != nil {
			return nil, errors.Wrap(err, "getting image from descriptor")
		}

		imageDigest, err := im.Digest()
		if err != nil {
			return nil, errors.Wrap(err, "while calculating image digest")
		}

		dig, err := name.NewDigest(
			fmt.Sprintf(
				"%s/%s@%s:%s",
				tag.RegistryStr(), tag.RepositoryStr(),
				imageDigest.Algorithm, imageDigest.Hex,
			),
		)
		if err != nil {
			return nil, errors.Wrap(err, "building single image digest")
		}

		logrus.Infof("Adding image digest %s from reference", dig.String())
		return append(images, struct {
			Digest string
			Arch   string
			OS     string
		}{Digest: dig.String()}), nil
	}

	// Get the image index
	index, err := descr.ImageIndex()
	if err != nil {
		return nil, errors.Wrapf(err, "getting image index for %s", imageName)
	}
	indexManifest, err := index.IndexManifest()
	if err != nil {
		return nil, errors.Wrapf(err, "getting index manifest from %s", imageName)
	}
	logrus.Infof("Reference image index points to %d manifests", len(indexManifest.Manifests))

	for _, manifest := range indexManifest.Manifests {
		dig, err := name.NewDigest(
			fmt.Sprintf(
				"%s/%s@%s:%s",
				tag.RegistryStr(), tag.RepositoryStr(),
				manifest.Digest.Algorithm, manifest.Digest.Hex,
			))
		if err != nil {
			return nil, errors.Wrap(err, "generating digest for image")
		}

		logrus.Infof(
			"Adding image %s/%s@%s:%s (%s/%s)",
			tag.RegistryStr(), tag.RepositoryStr(), manifest.Digest.Algorithm, manifest.Digest.Hex,
			manifest.Platform.Architecture, manifest.Platform.OS,
		)
		arch, osid := "", ""
		if manifest.Platform != nil {
			arch = manifest.Platform.Architecture
			osid = manifest.Platform.OS
		}
		images = append(images,
			struct {
				Digest string
				Arch   string
				OS     string
			}{
				Digest: dig.String(),
				Arch:   arch,
				OS:     osid,
			})
	}
	return images, nil
}

//unpackImageToFileSystem untars a tarball to the directory
func unpackImageToFileSystem(dir, tarfile string) error {

	var cmd bytes.Buffer
	execCmd, _ := template.New("untarImage").Parse(untarToLocalDirCmdTmpl)

	if err := execCmd.Execute(&cmd, map[string]string{
		"LocalDir": dir,
		"TarPath":  tarfile,
	}); err != nil {
		fmt.Printf("\nerror formating untar exec cmd: %s", err)
		return errors.New("unable to create command [tar]")
	}

	cmdTokens := strings.Fields(cmd.String())
	fmt.Printf("\ncommand to execute %s", cmdTokens)
	_, err := exec.Command(cmdTokens[0], cmdTokens[1:]...).CombinedOutput()
	if err != nil {
		fmt.Printf("\nerror executing untar cmd: %s", err.Error())
		return errors.New("unable to untar an image file")
	}
	removeFile(tarfile)
	return nil
}

//getManifest reads the file and parses it for image manifest
func getManifest(filename string) (spdx.ArchiveManifest, error) {
	manifestData := []spdx.ArchiveManifest{}

	manifestJSON, err := os.ReadFile(filename)
	if err != nil {
		return spdx.ArchiveManifest{}, errors.New(fmt.Sprintf("unable to read manifest file %q: %s", filename, err.Error()))
	}
	if err := json.Unmarshal(manifestJSON, &manifestData); err != nil {
		fmt.Println(string(manifestJSON))
		return spdx.ArchiveManifest{}, errors.New(fmt.Sprintf("error unmarshalling manifest file %q: %s", filename, err.Error()))
	}
	return manifestData[0], nil
}

//removeFile deletes file
func removeFile(file string) error {

	var cmd bytes.Buffer
	execCmd, _ := template.New("removeTarFile").Parse(removeTarFileCmdTmpl)

	if err := execCmd.Execute(&cmd, map[string]string{
		"File": file,
	}); err != nil {
		fmt.Printf("\nerror formating rm exec cmd: %s", err)
		return errors.New("unable to create rm command")
	}

	cmdTokens := strings.Fields(cmd.String())
	fmt.Printf("\ncommand to execute %s", cmdTokens)
	_, err := exec.Command(cmdTokens[0], cmdTokens[1:]...).CombinedOutput()
	if err != nil {
		fmt.Printf("\nerror executing rm cmd: %s", err.Error())
		return errors.New("unable to rm file")
	}
	return nil
}

package imagefs

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/pkg/errors"
	"k8s.io/release/pkg/spdx"
)

//Get generates a file system of the image and returns file system path
func Get(imageName, buildContextDir string) (string, error) {
	rootDir := ""
	imageRefs, err := getImageReferences(imageName)
	if err != nil {
		fmt.Printf("\nerror in getImageReferences: %s", err.Error())
		return "", errors.Wrap(err, "getting image references from container registry")
	}
	if len(imageRefs) == 0 {
		return "", fmt.Errorf("\n%d image references found for %q", len(imageRefs), imageName)
	}
	refData := imageRefs[0]
	if len(imageRefs) > 1 {
		//e.g. images for different hardware architectures such as amd64, s390x, ppc64le
		//will pick amd64 if present, otherwise pick the first
		fmt.Printf("\n%d image references found for %q", len(imageRefs), imageName)
		for i, refData := range imageRefs {
			if strings.Contains(strings.ToLower(refData.Arch), "amd64") {
				refData = imageRefs[i]
			}
		}
	}

	fmt.Printf("\ndownload tarball for %q, arch=%q", imageName, refData.Arch)
	ref, err := name.ParseReference(refData.Digest)
	if err != nil {
		fmt.Printf("\nparsing reference %s", imageName)
		return "", errors.Wrap(err, "parsing image reference")
	}

	img, err := remote.Image(ref)
	if err != nil {
		fmt.Printf("\nerror getting image %q", ref.Name())
		return "", errors.Wrap(err, "getting remote image")
	}

	rootDir, err = generateImageFileSystem(buildContextDir, img, ref)
	if err != nil {
		fmt.Printf("\nerror from getImageFileSystem(): %s", err.Error())
		return "", errors.Wrap(err, "creating image fifle system")
	}
	return rootDir, nil
}

//generateImageFileSystem downloads the image and untar it to the directory
func generateImageFileSystem(unpackdir string, img v1.Image, ref name.Reference) (string, error) {
	tarfile := path.Join(unpackdir, "image.tar")
	if err := tarball.WriteToFile(tarfile, ref, img); err != nil {
		fmt.Printf("\nerror writing image to %q", tarfile)
		return "", errors.Wrap(err, "writing image to file system")
	}
	fmt.Printf("\nwrote image to %q", tarfile)

	unpackDirRootfs := path.Join(unpackdir, "rootfs")
	os.MkdirAll(unpackDirRootfs, 0744)
	if err := untar(tarfile, unpackDirRootfs); err != nil {
		fmt.Printf("\nerror unpack %s to file system %s", tarfile, unpackDirRootfs)
		return unpackDirRootfs, errors.Wrap(err, "unpacking image tarball")
	}

	manifest, err := getManifest(path.Join(unpackDirRootfs, "manifest.json"))
	if err != nil {
		fmt.Printf("\nerror retrieving image manifest.json: %s", err.Error())
		return "", errors.Wrap(err, "reading image manifest.json")
	}

	for _, file := range manifest.LayerFiles { //untar the tar.gz file for each layer
		filepath := path.Join(unpackDirRootfs, file)
		err = untar(filepath, unpackDirRootfs)
		if err != nil {
			fmt.Printf("\nerror untar image layer %q: %s", file, err.Error())
			return "", errors.Wrapf(err, "untaring image layer %s", file)
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
		fmt.Printf("Reference %s points to a single image", imageName)
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

		fmt.Printf("Adding image digest %s from reference", dig.String())
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
	fmt.Printf("Reference image index points to %d manifests", len(indexManifest.Manifests))

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

		fmt.Printf(
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

//untar .tar or .tar.gz file into target directory
func untar(tarball, target string) error {
	fmt.Printf("\nuntar %q to %s", tarball, target)
	file, err := os.Open(tarball)
	if err != nil {
		return errors.Wrap(err, "opening tarball")
	}
	defer file.Close()
	defer os.Remove(tarball)

	var fileReader io.ReadCloser = file
	if strings.HasSuffix(tarball, ".gz") {
		if fileReader, err = gzip.NewReader(file); err != nil {
			return errors.Wrap(err, "creating gzip reader")
		}
		defer fileReader.Close()
	}

	tarReader := tar.NewReader(fileReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return errors.Wrap(err, "creating tar reader")
		}

		path := path.Join(target, header.Name)
		info := header.FileInfo()
		if info.IsDir() {
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return errors.Wrap(err, "creating directory")
			}
			continue
		}
		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return errors.Wrap(err, "creating a file")
		}

		_, err = io.Copy(file, tarReader)
		if err != nil {
			return errors.Wrap(err, "copying from tarball to file")
		}
	}
	return nil
}

//getManifest reads the file and parses it for image manifest
func getManifest(filename string) (spdx.ArchiveManifest, error) {
	manifestData := []spdx.ArchiveManifest{}

	manifestJSON, err := os.ReadFile(filename)
	if err != nil {
		return spdx.ArchiveManifest{}, errors.Wrap(err, "reading image manifest file")
	}
	if err := json.Unmarshal(manifestJSON, &manifestData); err != nil {
		fmt.Println(string(manifestJSON))
		return spdx.ArchiveManifest{}, errors.Wrap(err, "unmarshalling image manifest%q: %s")
	}
	return manifestData[0], nil
}

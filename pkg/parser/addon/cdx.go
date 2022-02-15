package addon

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/pkg/errors"
	"github.com/tap8stry/orion/pkg/common"
)

//StoreCdxJSON :
func StoreCdxJSON(outfp string,
	image common.Image, namespace string, installs []common.VerifiedArtifact) error {

	metadata := cdx.Metadata{
		// Define metadata about the main component
		// (the component which the BOM will describe)
		Component: &cdx.Component{
			BOMRef:  fmt.Sprintf("image:%s@%s", image.Name, image.SHA256),
			Type:    cdx.ComponentTypeContainer,
			Name:    image.Name,
			Version: image.SHA256,
		},
		// Use properties to include an internal identifier for this BOM
		// https://cyclonedx.org/use-cases/#properties--name-value-store
		Properties: &[]cdx.Property{
			{
				Name:  "internal:scan-timestamp",
				Value: time.Now().String(),
			},
		},
	}
	components := []cdx.Component{}
	imgC := createImageComponent(image)  //for the image
	imgC.Components = &[]cdx.Component{} // for addon components in the image

	for _, ins := range installs {
		if ins.IsDownload {
			//create a componemt for each dowload
			downloadName := ins.DownloadLocation[strings.LastIndex(ins.DownloadLocation, "://")+3:]
			compD := createAssembleComponent(downloadName, ins.DownloadLocation)
			compD.Components = &[]cdx.Component{}
			for _, art := range ins.Artifacts { //create a component for each verified artifact
				compC := cdx.Component{}
				if art.IsDirectory {
					compC = createApplicationComponent(art.Path, downloadName, ins.DownloadLocation)
				} else {
					compC = createFileComponent(art.Path, downloadName, art.SHA256, ins.DownloadLocation)
				}
				//add artifact component to the download components
				*compD.Components = append(*compD.Components, compC)
			}
			//add download component to image component
			//*imgC.Components = append(*imgC.Components, compD)
			*imgC.Components = append(*imgC.Components, compD)
		} else { //artifacts from COPY/ADD operations
			sourse := ins.DownloadLocation
			for _, art := range ins.Artifacts { //create a component for each verified artifact
				compC := cdx.Component{}
				if art.IsDirectory {
					compC = createApplicationComponent(art.Path, sourse, ins.DownloadLocation)
				} else {
					fpath := art.Path[strings.Index(art.Path, "/rootfs")+7:]
					compC = createFileComponent(fpath, sourse, art.SHA256, ins.DownloadLocation)
				}
				//add artifact component to the image components
				*imgC.Components = append(*imgC.Components, compC)
			}
		}
	}
	components = append(components, imgC)

	// Assemble the BOM
	bom := cdx.NewBOM()
	bom.Metadata = &metadata
	bom.Components = &components

	// Encode the BOM
	fmt.Printf("\nresults saved to %s\n", outfp)
	if _, err := os.Stat(outfp); err == nil { //delete if exists to avoid any leftover of old contents
		fmt.Printf("an old report %s exists and will override\n", outfp)
		os.Remove(outfp)
	}
	bomWriter, err := os.OpenFile(outfp, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		fmt.Printf("error opening output file for writing: %v\n", err)
	}
	defer bomWriter.Close()

	encoder := cdx.NewBOMEncoder(bomWriter, cdx.BOMFileFormatJSON)
	encoder.SetPretty(true)
	if err := encoder.Encode(bom); err != nil {
		fmt.Printf("error encoding cdx format: %s\n", err.Error())
		return errors.Wrap(err, "encoding CycloneDX BOM file")
	}
	return nil
}

func createImageComponent(img common.Image) cdx.Component {
	c := cdx.Component{}
	c.BOMRef = fmt.Sprintf("image:%s", img.SHA256)
	c.Type = cdx.ComponentTypeContainer
	c.Name = img.Name
	c.Version = img.Tag
	c.PackageURL = img.SHA256
	c.Components = &[]cdx.Component{
		{
			BOMRef:  fmt.Sprintf("os:%s@%s", img.OSName, img.OSVersion),
			Type:    cdx.ComponentTypeOS,
			Name:    img.OSName,
			Version: img.OSVersion,
		},
	}
	return c
}

func createAssembleComponent(name, downloadURL string) cdx.Component {
	c := cdx.Component{
		Type: cdx.ComponentTypeApplication,
		Supplier: &cdx.OrganizationalEntity{
			URL: &[]string{downloadURL},
		},
		Name:       name,
		PackageURL: downloadURL,
	}
	return c
}

func createApplicationComponent(filepath, group, downloadURL string) cdx.Component {
	c := cdx.Component{
		Type: cdx.ComponentTypeApplication,
		Supplier: &cdx.OrganizationalEntity{
			URL: &[]string{downloadURL},
		},
		Group:      group,
		Name:       path.Base(filepath),
		Hashes:     &[]cdx.Hash{},
		PackageURL: downloadURL,
	}
	return c
}

func createFileComponent(filepath, group, filehash, downloadURL string) cdx.Component {
	c := cdx.Component{
		BOMRef: fmt.Sprintf("file:%s", filepath),
		Type:   cdx.ComponentTypeFile,
		Supplier: &cdx.OrganizationalEntity{
			URL: &[]string{downloadURL},
		},
		Group:   group,
		Name:    path.Base(filepath),
		Version: filehash,
	}
	c.Hashes = &[]cdx.Hash{
		{
			Algorithm: "SHA-256",
			Value:     filehash,
		},
	}
	return c
}

func createPackageComponent(pkgName, pkgVersion, pkgURL string) cdx.Component {
	pkgRef := fmt.Sprintf("pkg:%s@%s", pkgName, pkgVersion)
	c := cdx.Component{}
	c.BOMRef = pkgRef
	c.Type = cdx.ComponentTypeLibrary
	c.Name = pkgName
	c.Version = pkgVersion
	c.PackageURL = pkgURL
	return c
}

func createOSComponent(osName, osVersion string) cdx.Component {
	osRef := fmt.Sprintf("os:%s@%s", osName, osVersion)
	c := cdx.Component{}
	c.BOMRef = osRef
	c.Type = cdx.ComponentTypeOS
	c.Name = osName
	c.Version = osVersion
	return c
}

func createPackageDependencies(srcBOMRef string, deps []string) cdx.Dependency {
	dRefs := []cdx.Dependency{}
	for _, d := range deps {
		pkgMeta := strings.Split(d, ":")
		if len(pkgMeta) == 2 {
			depPkgRef := fmt.Sprintf("pkg:%s@%s", pkgMeta[0], pkgMeta[1])
			dRefs = append(dRefs, cdx.Dependency{Ref: depPkgRef})
		} else {
			dRefs = append(dRefs, cdx.Dependency{Ref: d})
		}
	}
	c := cdx.Dependency{
		Ref:          srcBOMRef,
		Dependencies: &dRefs,
	}
	return c
}

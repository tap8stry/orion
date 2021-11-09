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

	"github.com/google/go-containerregistry/pkg/logs"
	"github.com/tapestry-discover/pkg/common"
	"k8s.io/release/pkg/spdx"
)

func GenerateSpdxReport(dockerfilename, image, namespace string, artifacts []common.VerifiedArtifact) (string, error) {
	doc := spdx.NewDocument()
	doc.Name = "SPDX-Docker-Image-Addons-" + image
	doc.Namespace = namespace
	doc.Creator.Person = "Tester Tester"
	doc.Creator.Tool = []string{"https://github.ibm.com/tapestry/tapestry-discover", "k8s.io/release/pkg/spdx"}
	logs.Progress.Printf("create a new SPDX doc %q, namespace=%q", doc.Name, doc.Namespace)

	for _, art := range artifacts {
		if art.IsDirectory {
			myspdx := spdx.NewSPDX()
			pkg, err := myspdx.PackageFromDirectory(art.Path)
			if err != nil {
				logs.Warn.Printf("\nerror creating package for directory %s, error = %s", art.Path, err.Error())
			}
			pkg.DownloadLocation = art.DownloadLocation
			pkg.FileName = art.Path[strings.Index(art.Path, "rootfs/")+7:]
			if err := doc.AddPackage(pkg); err != nil {
				logs.Warn.Printf("\nerror in adding package to document: %s", err.Error())
			}
		} else {
			f := spdx.NewFile()
			name := art.Name[strings.LastIndex(art.Name, "rootfs/")+7:]
			f.FileName = name
			f.SourceFile = art.Path
			f.Name = name

			if art.IsDownload { //create a Package, add the download file to package, add package to document
				pkg := spdx.NewPackage()
				pkg.Name = name[strings.LastIndex(name, "/")+1:]
				pkg.ID = "SPDXRef-Package-" + name
				pkg.DownloadLocation = art.DownloadLocation
				pkg.FileName = name
				if err := pkg.AddFile(f); err != nil {
					logs.Warn.Printf("error in adding file to package: %s", err.Error())
				}
				if err := doc.AddPackage(pkg); err != nil {
					logs.Warn.Printf("error in adding package to document: %s", err.Error())
				}
			} else { //add file to document
				if err := doc.AddFile(f); err != nil {
					logs.Warn.Printf("error in adding file to document: %s", err.Error())
				}
			}
		}
	}
	markup, err := doc.Render()
	if err != nil {
		logs.Debug.Printf("error in rendering SPDX document: %s", err.Error())
		return "", err
	}
	return markup, nil
}

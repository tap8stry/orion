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
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/tap8stry/orion/pkg/common"
	"k8s.io/release/pkg/spdx"
)

func GenerateSpdxReport(dockerfilename, image, namespace string, installs []common.VerifiedArtifact, toolversion string) (string, error) {
	oriontool := "https://github.com/tap8stry/orion"
	if len(toolversion) > 0 {
		oriontool += fmt.Sprintf("@%s", toolversion)
	}
	doc := spdx.NewDocument()
	doc.Name = "SPDX-Docker-Image-Addons-" + image
	doc.ID = "SPDXRef-DOCUMENT-FOR-ADDONS"
	doc.Namespace = namespace
	doc.Creator.Person = "Tester Tester"
	doc.Creator.Tool = []string{oriontool, "k8s.io/release/pkg/spdx"}
	fmt.Printf("\ncreate a new SPDX doc %q, namespace=%q", doc.Name, doc.Namespace)

	for _, ins := range installs {
		if ins.IsDownload { //create a Package, add the download file to package, add package to document
			dpkg := spdx.NewPackage()
			name := ins.DownloadLocation[strings.LastIndex(ins.DownloadLocation, "://")+3:]
			dpkg.Name = name
			dpkg.ID = "SPDXRef-Package-" + name
			dpkg.FileName = name
			dpkg.DownloadLocation = ins.DownloadLocation

			for _, art := range ins.Artifacts {
				if art.IsDirectory {
					myspdx := spdx.NewSPDX()
					pkg, err := myspdx.PackageFromDirectory(art.Path)
					if err != nil {
						fmt.Printf("\n\nerror creating package for directory %s, error = %s", art.Path, err.Error())
					}
					pkg.FileName = art.Path[strings.Index(art.Path, "rootfs/")+7:]
					pkg.DownloadLocation = dpkg.DownloadLocation
					if err := dpkg.AddPackage(pkg); err != nil {
						fmt.Printf("\n\nerror in adding a directory to download package: %s", err.Error())
					}
				} else {
					f := spdx.NewFile()
					name := art.Name[strings.LastIndex(art.Name, "rootfs/")+7:]
					f.FileName = name
					f.SourceFile = art.Path
					f.Name = name
					if err := dpkg.AddFile(f); err != nil {
						fmt.Printf("\n\nerror in adding a file to download package: %s", err.Error())
					}
				}
			}
			if err := doc.AddPackage(dpkg); err != nil {
				fmt.Printf("\n\nerror in adding package to document: %s", err.Error())
			}
		} else {
			for _, art := range ins.Artifacts {
				if art.IsDirectory {
					myspdx := spdx.NewSPDX()
					pkg, err := myspdx.PackageFromDirectory(art.Path)
					if err != nil {
						fmt.Printf("\n\nerror creating package for directory %s, error = %s", art.Path, err.Error())
					}
					pkg.FileName = art.Path[strings.Index(art.Path, "rootfs/")+7:]
					if err := doc.AddPackage(pkg); err != nil {
						fmt.Printf("\n\nerror in adding a directory to download package: %s", err.Error())
					}
				} else {
					f := spdx.NewFile()
					name := art.Name[strings.LastIndex(art.Name, "rootfs/")+7:]
					f.FileName = name
					f.SourceFile = art.Path
					f.Name = name
					if err := doc.AddFile(f); err != nil {
						fmt.Printf("\n\nerror in adding a file to download package: %s", err.Error())
					}
				}
			}
		}
	}

	markup, err := doc.Render()
	if err != nil {
		fmt.Printf("\nerror in rendering SPDX document: %s", err.Error())
		return "", errors.Wrap(err, "rendering SPDX document")
	}
	return markup, nil
}

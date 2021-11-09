//  ###############################################################################
//  # Licensed Materials - Property of IBM
//  # (c) Copyright IBM Corporation 2021. All Rights Reserved.
//  #
//  # Note to U.S. Government Users Restricted Rights:
//  # Use, duplication or disclosure restricted by GSA ADP Schedule
//  # Contract with IBM Corp.
//  ###############################################################################

package common

import (
	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

//DockerfileEcosystem :
const (
	DockerfileEcosystem = "dockerfile"
	FormatJSON          = "json"
	FormatSpdx          = "spdx"
	DefaultFilename     = "results"
)

//DiscoverOpts :
type DiscoverOpts struct {
	GitURL         string
	GitRevision    string
	GitCommitID    string
	DockerfilePath string
	OutFilepath    string
	Format         string
	Image          string
	Namespace      string
}

//Dockerfile :
type Dockerfile struct {
	Filepath    string            `json:"filepath"`
	Filehash    string            `json:"filehash"`
	FileType    string            `json:"filetype"`
	BuildStages []BuildStage      `json:"build_stages"`
	BuildArgs   map[string]string `json:"build_args,omitempty"`
}

//BuildStage :
type BuildStage struct {
	StageID         string            `json:"stage_id"`
	Context         string            `json:"key"`
	DependsOn       string            `json:"parent_stage"`
	ScratchBuild    bool              `json:"is_scratch_build"`
	StartLineNo     int               `json:"start_line"`
	EndLineNo       int               `json:"end_line"`
	Image           Image             `json:"base_image"`
	Packages        []Package         `json:"os_packages"`
	AppPackages     []Package         `json:"app_packages"`
	PackageOverride []PackageOverride `json:"package_override"`
	DockerFileCmds  []*parser.Node    `json:"-"`
	AddOnInstalls   []InstallTrace    `json:"addon_installs"`
	EnvVariables    map[string]string `json:"env_variables,omitempty"`
	AddOnSpdxReport string            `json:"addon_spdx_report"`
}

//Image :
type Image struct {
	Name      string    `json:"name"`
	Tag       string    `json:"tag"`
	OSName    string    `json:"os_name"`
	OSVersion string    `json:"os_version"`
	SHA256    string    `json:"sha256"`
	Metadata  string    `json:"metadata"`
	Packages  []Package `json:"packages"`
	Scanned   bool      `json:"scanned"`
}

//ManifestFile :
type ManifestFile struct {
	CommitID  string    `json:"commitid"`
	GitURL    string    `json:"giturl"`
	GitBranch string    `json:"gitbranch"`
	Filepath  string    `json:"filepath"`
	Filehash  string    `json:"filehash"`
	FileType  string    `json:"filetype"`
	Packages  []Package `json:"packages"`
	Scanned   bool      `json:"scanned"`
}

//Package :
type Package struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Ecosystem    string   `json:"ecosystem"`
	Source       string   `json:"source,omitempty"`
	Key          string   `json:"key,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
}

//PackageOverride :
type PackageOverride struct {
	BasePackage     string `json:"base_package"`
	OverridePackage string `json:"override_package"`
}

//Trace : step trace of dockerfile add-on installations via RUN curl/wget/ or COPY/ADD
type Trace struct {
	Command     string `json:"command"`
	Source      string `json:"source,omitempty"`
	Destination string `json:"destination,omitempty"`
	Workdir     string `json:"workdir,omitempty"`
}

//InstallTrace presents an add-on installation and its traces
type InstallTrace struct {
	Origin     string        `json:"origin"`
	OriginHash string        `json:"originhash,omitempty"`
	Traces     map[int]Trace `json:"traces,omitempty"`
}

//Artifact presents a resource
type Artifact struct {
	Source   string `json:"source,omitempty"`
	Filepath string `json:"filepath,omitempty"`
	Filehash string `json:"filehash,omitempty"`
	IsDir    bool   `json:"isDir"`
}

//CommandSet is a set of commands in their execution order
type CommandSet struct {
	Commands map[int]string
}

type SpdxRelationship string

type SpdxFile struct {
	FileName             string `json:"filename"`
	SPDXID               string `json:"spdxid"`
	FileChecksum         string `json:"fileCheckSum,omitempty"`
	FileDownloadLocation string `json:"fileDownloadLocation,omitempty"`
	LicenseConcluded     string `json:"licenseConcluded,omitempty"`
	LicenseInfoInFile    string `json:"licenseInfoInfile,omitempty"`
	FileCopyrightText    string `json:"fileCopyrightText,omitempty"`
	FileComment          string `json:"fileComment,omitempty"`
}

type VerifiedArtifact struct {
	Name             string `json:"name"`
	Path             string `json:"path"`
	Version          string `json:"version,omitempty"`
	IsDirectory      bool   `json:"isDirectory"`
	IsDownload       bool   `json:"isDownload"`
	DownloadLocation string `json:"downloadLocation"`
	Comment          string `json:"packageComment,omitempty"`
}

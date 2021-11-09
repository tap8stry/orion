# tapestry-discover 

The repository is a tool that generates software inventory of a container image, specifically the software installations that are not managed through package managers. Developers ofter install additional software artifacts through RUN shell commands and COPY/ADD during docker build besides OS packages and open-source software packages. Such installations need to be counted in order to produce a complete and accurate SBOM for security compliance and auditing purposes. 

There are a number of open-source tools on SBOM: (1) [tern](https://github.com/tern-tools/tern) for container image, (2) [spdx sbom generator](https://github.com/spdx/spdx-sbom-generator) for open-source software packages of various languages, and (3) [Kubernetes Release Tooling](https://github.com/kubernetes/release) for golang applications.

Our project compliments these tools with the capabilities to track software artifacts installed outside of the package management tools, independent of the platform and language specific package management tools. The project also leavages the spdx module of [Kubernetes Release Tooling](https://github.com/kubernetes/release) in generating SPDX document.


### Features

There are many ways that developers can compose their Dockerfile to install additional artifacts in docker build. We have selected some [sample Dockerfiles](https://github.ibm.com/tapestry/tapestry-discover/tree/main/sample-dockerfiles) from the popular github projects (those with most stars). From examining them we extract scenarios/patterns that are used in these Dockerfiles. 

**The project has implemented the tracing capaibilities for artifacts installed by the following docker operations and shell commands, which covers the majority of the scenarios.**

- WORKDIR
- ARG
- ENV
- RUN
    - curl
    - wget
    - tar -x...
    - git clone, git checkout
    - cp
    - mv 
    - cd 
- COPY
- ADD

**The scenarios/paterns to be considered are listed [here](https://github.ibm.com/tapestry/tapestry-discover/blob/main/doc/new-scenarios.md) for further development.**


### How to run it

1. Clone the project and make a build

```
% git clone https://github.ibm.com/tapestry/tapestry-discover.git
% cd tapestry-discover
% make
```

2. Command to scan Dockefile and produce addon installation traces

```
% ./tapestry-discover -d=true run -d <dockerfile-path> -n <sbom-namespace> -f <output-file-path>
```

where 
- dockerfile-path: Dockerfile pathname
- sbom-namespace: namespace, e.g. your project's github repository URL
- output-file-path: file name for saving discovery results. The traces is saved to `<output-filepath>-trace.json`

3. Command to produce/verify addon installation traces and produce SBOM report

```
% ./tapestry-discover -d=true run -d <dockerfile-path> -n <sbom-namespace> -i <image-name:tag> -f <output-file-path>
```
The SBOM report is saved to `<output-filepath>.spdx`.


# orion

The repository is a tool that generates software inventory of a container image, specifically the software installations that are not managed through package managers. Developers ofter install additional software artifacts through RUN shell commands and COPY/ADD during docker build besides OS packages and open-source software packages. Such installations need to be counted in order to produce a complete and accurate SBOM for security compliance and auditing purposes. 

There are a number of open-source tools on SBOM: (1) [tern](https://github.com/tern-tools/tern) for container image, (2) [spdx sbom generator](https://github.com/spdx/spdx-sbom-generator) for open-source software packages of various languages, and (3) [Kubernetes Release Tooling](https://github.com/kubernetes/release) for golang applications.

Our project compliments these tools with the capabilities to track software artifacts installed outside of the package management tools, independent of the platform and language specific package management tools. The project also leavages the spdx module of [Kubernetes Release Tooling](https://github.com/kubernetes/release) in generating SPDX document.


### Features

There are many ways that developers can compose their Dockerfile to install additional artifacts in docker build. We have selected some from the popular github projects (those with most stars). From examining them we extract scenarios/patterns that are used in these Dockerfiles. 

**The project has implemented the tracing capaibilities for artifacts installed by the following docker operations and shell commands, which covers the majority of the scenarios.**

- WORKDIR
- ARG
- ENV
- RUN
    - curl
    - wget
    - tar -x...
    - unzip
    - git clone, git checkout
    - cp
    - mv 
    - cd 
- COPY
- ADD

**The scenarios/paterns yet to be addressed are listed [here](https://github.com/tap8stry/orion/blob/main/doc/new-scenarios.md) for further development.**


### How to run it

1. Clone the project and make a build

```
% git clone https://github.com/tap8stry/orion.git
% cd orion
% make
```

2. Command to scan Dockefile and produce addon installation traces

```
% ./orion discover -f <dockerfile-path> -n <sbom-namespace> -r <output-file-path>
```

where 
- dockerfile-path: Dockerfile pathname
- sbom-namespace: namespace, e.g. your project's github repository URL
- output-file-path: file name for saving discovery results. The traces is saved to `<output-filepath>-trace.json`.

3. Command to produce/verify addon installation traces and produce SBOM report

```
% ./orion discover -f <dockerfile-path> -n <sbom-namespace> -i <image-name:tag> -k <ibmcloud apikey> -r <output-file-path> -o <bom format, cdx or spdx>
```

4. Work around if encounter access permission issue when decompressing image tarball

You may encounter error messages like the following when running the command 3. 

```
error executing untar cmd: exit status 1
error untar image layer "356f18f3a935b2f226093720b65383048249413ed99da72c87d5be58cc46661c.tar.gz": unable to untar an image file
```

This is caused by the access permission when decompressing the image tar file to a temperary file system. You can use `sudo` command to work around this problem, see the command below.

```
% sudo ./orion discover -d <dockerfile-path> -n <sbom-namespace> -i <image-name:tag> -f <output-file-path>
```

### Access to Private Images on Cloud Providers' Container Registries

A credential helper is required to handle access credentials when pulling images from private repositories on cloud providers' container registry. The helper implements the interface as defined in [the authn package of go-containerregistry project](https://github.com/google/go-containerregistry/tree/main/pkg/authn). 

The helper for IBM Cloud Container Registry is included under [pkg/credhelpers/icr](https://github.com/lluan444/orion/tree/main/pkg/credhelpers/icr). Example helpers for other cloud providers are suggested [here](https://github.com/google/go-containerregistry/tree/main/pkg/authn).


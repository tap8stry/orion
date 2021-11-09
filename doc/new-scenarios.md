## Outstanding Installation Scenarios Observed in Sample Dockerfiles ##

The following is a list of scenarios observed in the sample dockerfiles. They are currently under investigation and yet to be supported. 


### Case 1: Cross stage reference to git origin ###
`git clone` in one stage, a following stage mounts to the previous stage and `git fetch` or `git checkout`

```
FROM git AS containerd-src
ARG CONTAINERD_VERSION
ARG CONTAINERD_ALT_VERSION
WORKDIR /usr/src
RUN git clone https://github.com/containerd/containerd.git containerd

FROM gobuild-base AS containerd-base
WORKDIR /go/src/github.com/containerd/containerd
ARG TARGETPLATFORM
ENV CGO_ENABLED=1 BUILDTAGS=no_btrfs
RUN xx-apk add musl-dev gcc && xx-go --wrap

FROM containerd-base AS containerd
ARG CONTAINERD_VERSION
RUN --mount=from=containerd-src,src=/usr/src/containerd,readwrite --mount=target=/root/.cache,type=cache \
  git fetch origin \
  && git checkout -q "$CONTAINERD_VERSION" \
  && make bin/containerd \
  && make bin/containerd-shim-runc-v2 \
  && make bin/ctr \
  && mv bin /out
```

### Case 2: Run `make` command for application build ### 

```
FROM --platform=$BUILDPLATFORM alpine:${ALPINE_VERSION} AS idmap
RUN apk add --no-cache git autoconf automake clang lld gettext-dev libtool make byacc binutils
COPY --from=xx / /
ARG SHADOW_VERSION
RUN git clone https://github.com/shadow-maint/shadow.git /shadow && cd /shadow && git checkout $SHADOW_VERSION
WORKDIR /shadow
ARG TARGETPLATFORM
RUN xx-apk add --no-cache musl-dev gcc libcap-dev
RUN CC=$(xx-clang --print-target-triple)-clang ./autogen.sh --disable-nls --disable-man --without-audit --without-selinux --without-acl --without-attr --without-tcb --without-nscd --host $(xx-clang --print-target-triple) \
  && make -j $(nproc) \
  && xx-verify src/newuidmap src/newuidmap \
  && cp src/newuidmap src/newgidmap /usr/bin
```

### Case 3:  `git clone` and `git checkout` are in separate RUN operations ### 

This requires corelation between `git checkout` and `git clone` in order to trace to the same git url. 

```
FROM gobuild-base AS rootlesskit
ARG ROOTLESSKIT_VERSION
RUN git clone https://github.com/rootless-containers/rootlesskit.git /go/src/github.com/rootless-containers/rootlesskit
WORKDIR /go/src/github.com/rootless-containers/rootlesskit
ARG TARGETPLATFORM
RUN  --mount=target=/root/.cache,type=cache \
  git checkout -q "$ROOTLESSKIT_VERSION"  && \
  CGO_ENABLED=0 xx-go build -o /rootlesskit ./cmd/rootlesskit && \
  xx-verify --static /rootlesskit
```

This scenario can be avoided if some best practice is followed in Dockerfile writing, i.e. organize all the commands relating to a git repo under one `RUN` operation as shown below.

```
FROM gobuild-base AS rootlesskit
ARG ROOTLESSKIT_VERSION
ARG TARGETPLATFORM
RUN  --mount=target=/root/.cache,type=cache \
  git clone https://github.com/rootless-containers/rootlesskit.git /go/src/github.com/rootless-containers/rootlesskit && \
  cd /go/src/github.com/rootless-containers/rootlesskit && \
  git checkout -q "$ROOTLESSKIT_VERSION"  && \
  CGO_ENABLED=0 xx-go build -o /rootlesskit ./cmd/rootlesskit && \
  xx-verify --static /rootlesskit
```

### Case 4: ENV value not available when parsing Dockerfile ###

```
WORKDIR $GOPATH/src/github.com/grafana/grafana
COPY go.mod go.sum embed.go ./
```

### Case 5: Use of other commands to install python alternatives (see Dockerfile-compose) ###

```
RUN curl -L https://www.python.org/ftp/python/${PYTHON_VERSION}/Python-${PYTHON_VERSION}.tgz | tar xzf - \
    && cd Python-${PYTHON_VERSION} \
    && ./configure --enable-optimizations --enable-shared --prefix=/usr LDFLAGS="-Wl,-rpath /usr/lib" \
    && make altinstall
RUN alternatives --install /usr/bin/python python /usr/bin/python2.7 50
RUN alternatives --install /usr/bin/python python /usr/bin/python$(echo "${PYTHON_VERSION%.*}") 60
RUN curl https://bootstrap.pypa.io/get-pip.py | python -
```

### Case 6: `git remote add upstream` and `git pull` ###

```
RUN mkdir "$pandas_home" \
    && git clone "https://github.com/$gh_username/pandas.git" "$pandas_home" \
    && cd "$pandas_home" \
    && git remote add upstream "https://github.com/pandas-dev/pandas.git" \
    && git pull upstream master
```

### Case 7:  `--mount` and `install.sh` (Dockerfile-moby) ###

```
FROM base AS criu
ARG DEBIAN_FRONTEND
ADD --chmod=0644 https://download.opensuse.org/repositories/devel:/tools:/criu/Debian_10/Release.key /etc/apt/trusted.gpg.d/criu.gpg.asc
RUN --mount=type=cache,sharing=locked,id=moby-criu-aptlib,target=/var/lib/apt \
    --mount=type=cache,sharing=locked,id=moby-criu-aptcache,target=/var/cache/apt \
        echo 'deb https://download.opensuse.org/repositories/devel:/tools:/criu/Debian_10/ /' > /etc/apt/sources.list.d/criu.list \
        && apt-get update \
        && apt-get install -y --no-install-recommends criu \
        && install -D /usr/sbin/criu /build/criu

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=bind,src=hack/dockerfile/install,target=/tmp/install \
        PREFIX=/build /tmp/install/install.sh containerd
```


### Case 8: Use of inline shell scripts (Dockerfile-moby) ### 

```
FROM base AS registry
WORKDIR /go/src/github.com/docker/distribution
# Install two versions of the registry. The first one is a recent version that
# supports both schema 1 and 2 manifests. The second one is an older version that
# only supports schema1 manifests. This allows integration-cli tests to cover
# push/pull with both schema1 and schema2 manifests.
# The old version of the registry is not working on arm64, so installation is
# skipped on that architecture.
ENV REGISTRY_COMMIT_SCHEMA1 ec87e9b6971d831f0eff752ddb54fb64693e51cd
ENV REGISTRY_COMMIT 47a064d4195a9b56133891bbb13620c3ac83a827
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=tmpfs,target=/go/src/ \
        set -x \
        && git clone https://github.com/docker/distribution.git . \
        && git checkout -q "$REGISTRY_COMMIT" \
        && GOPATH="/go/src/github.com/docker/distribution/Godeps/_workspace:$GOPATH" \
           go build -buildmode=pie -o /build/registry-v2 github.com/docker/distribution/cmd/registry \
        && case $(dpkg --print-architecture) in \
               amd64|armhf|ppc64*|s390x) \
               git checkout -q "$REGISTRY_COMMIT_SCHEMA1"; \
               GOPATH="/go/src/github.com/docker/distribution/Godeps/_workspace:$GOPATH"; \
                   go build -buildmode=pie -o /build/registry-v2-schema1 github.com/docker/distribution/cmd/registry; \
                ;; \
           esac
``` 

### Case 9: Use of shell script file (Dockerfile-moby) ###

```
RUN /download-frozen-image-v2.sh /build \
        busybox:latest@sha256:95cf004f559831017cdf4628aaf1bb30133677be8702a8c5f2994629f637a209 \
        busybox:glibc@sha256:1f81263701cddf6402afe9f33fca0266d9fff379e59b1748f33d3072da71ee85 \
        debian:bullseye-slim@sha256:dacf278785a4daa9de07596ec739dbc07131e189942772210709c5c0777e8437 \
        hello-world:latest@sha256:d58e752213a51785838f9eed2b7a498ffa1cb3aa7f946dda11af39286c3db9a9 \
        arm32v7/hello-world:latest@sha256:50b8560ad574c779908da71f7ce370c0a2471c098d44d1c8f6b513c5a55eeeb1


```

### Case 10: Changes to resource configurations using sed command ###

```
#(Dockerfile-chaosblade) 
FROM alpine:3.10.4
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories
```

```
#(Dockerfile-moby) 
FROM ${GOLANG_IMAGE} AS base
RUN echo 'Binary::apt::APT::Keep-Downloaded-Packages "true";' > /etc/apt/apt.conf.d/keep-cache
ARG APT_MIRROR
RUN sed -ri "s/(httpredir|deb).debian.org/${APT_MIRROR:-deb.debian.org}/g" /etc/apt/sources.list \
 && sed -ri "s/(security).debian.org/${APT_MIRROR:-security.debian.org}/g" /etc/apt/sources.list
 ```

This could be a security exposure if changing to a malicious site. We may need to flag it.

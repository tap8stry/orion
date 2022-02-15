###############################################################################
# Licensed Materials - Property of IBM
# (c) Copyright IBM Corporation 2020. All Rights Reserved.
#
# Note to U.S. Government Users Restricted Rights:
# Use, duplication or disclosure restricted by GSA ADP Schedule
# Contract with IBM Corp.
###############################################################################
FROM golang:1.17-alpine

RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssh docker

WORKDIR /go/src/github.com/tap8stry/orion

COPY pkg/    pkg/
COPY cmd/    cmd/

# Copy Go Modules manifests and download the modules
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

#RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build --tags static_all -v -o ./bin/orion cmd/api-mgr/main.go

RUN wget https://download.clis.cloud.ibm.com/ibm-cloud-cli/2.4.0/binaries/IBM_Cloud_CLI_2.4.0_linux_amd64.tgz &&\
    tar xvf IBM_Cloud_CLI_2.4.0_linux_amd64.tgz &&\
    rm IBM_Cloud_CLI_2.4.0_linux_amd64.tgz &&\
    cp IBM_Cloud_CLI/ibmcloud /usr/local/bin/. &&\
    rm -rf IBM_Cloud_CLI &&\
    ibmcloud --version &&\
    ibmcloud plugin install container-registry &&\
    go build -ldflags "-X github.com/tap8stry/orion/cmd/discover/cli.version=1 -X github.com/tap8stry/orion/cmd/discover/cli.commit=laura-test -X github.com/tap8stry/orion/cmd/discover/cli.date=2021-11-22T14:37:37-0500" -o orion cmd/discover/main.go

ENTRYPOINT ["tail", "-f", "/dev/null"]

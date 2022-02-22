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

RUN go build -ldflags "-X github.com/tap8stry/orion/cmd/discover/cli.version=1 -X github.com/tap8stry/orion/cmd/discover/cli.commit=laura-test -X github.com/tap8stry/orion/cmd/discover/cli.date=2021-11-22T14:37:37-0500" -o orion cmd/discover/main.go

ENTRYPOINT ["tail", "-f", "/dev/null"]

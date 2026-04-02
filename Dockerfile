#
# Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
# Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
#

# Build the manager binary
FROM golang:1.25 AS builder

WORKDIR /workspace
COPY go.mod go.sum ./
COPY vendor/ vendor/
COPY . ./

ARG CONTROLLER_MAIN=.
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG CGO_ENABLED=0
ARG GOEXPERIMENT=
RUN GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" CGO_ENABLED="${CGO_ENABLED}" GOEXPERIMENT="${GOEXPERIMENT}" \
    go build -mod vendor -a -o manager ${CONTROLLER_MAIN}

FROM oraclelinux:9-slim
WORKDIR /

COPY --from=builder /workspace/manager .

USER 65532:65532

ENTRYPOINT ["/manager"]

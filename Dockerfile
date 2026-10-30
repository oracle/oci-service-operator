#
# Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
# Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
#

# Build the manager binary
FROM golang:1.26.10 AS builder

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

# ELSA-2026-88459: remove trust in the compromised eMudhra certificate chains.
RUN microdnf --refresh install -y 'ca-certificates >= 2026.2.92_v9.0.319-90.3.el9_8' \
    && update-ca-trust extract \
    && microdnf clean all

WORKDIR /

COPY --from=builder /workspace/manager .

USER 65532:65532

ENTRYPOINT ["/manager"]

#
# Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
# Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
#

# Build the manager binary
FROM golang:1.16 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
#RUN go mod download

# Copy the go source
#COPY main.go main.go
#COPY api/ api/
#COPY apis/ apis/
#COPY controllers/ controllers/

COPY . ./

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -mod vendor -a -o manager main.go

FROM oraclelinux:7-slim
WORKDIR /

# Copy the manager binary
COPY --from=builder /workspace/manager .

# Copy CRDs
COPY --from=builder /workspace/bundle/manifests/oci.oracle.com_*.yaml /
COPY --from=builder /workspace/bundle/manifests/servicemesh.oci.oracle.com_*.yaml /

# Copy vendor directory
COPY --from=builder /workspace/vendor vendor

USER 65532:65532

ENTRYPOINT ["/manager"]

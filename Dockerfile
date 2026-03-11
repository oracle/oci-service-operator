#
# Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
# Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
#

# Build the manager binary
FROM golang:1.25 as builder

WORKDIR /workspace
COPY go.mod go.sum ./
COPY vendor/ vendor/
COPY . ./

RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 GOEXPERIMENT=boringcrypto go build -mod vendor -a -o manager main.go

FROM oraclelinux:9-slim
WORKDIR /

COPY --from=builder /workspace/manager .

USER 65532:65532

ENTRYPOINT ["/manager"]

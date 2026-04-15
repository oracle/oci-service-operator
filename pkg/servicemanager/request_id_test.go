/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package servicemanager

import (
	"testing"

	"github.com/oracle/oci-service-operator/pkg/errorutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

type requestIDResponse struct {
	OpcRequestId *string `presentIn:"header" name:"opc-request-id"`
}

func TestResponseOpcRequestIDReadsOCIHeader(t *testing.T) {
	t.Parallel()

	requestID := "opc-create-1"
	if got := ResponseOpcRequestID(requestIDResponse{OpcRequestId: &requestID}); got != requestID {
		t.Fatalf("ResponseOpcRequestID() = %q, want %q", got, requestID)
	}
}

func TestRecordResponseOpcRequestIDPreservesLastNonEmptyValue(t *testing.T) {
	t.Parallel()

	status := &shared.OSOKStatus{OpcRequestID: "opc-seeded"}
	RecordResponseOpcRequestID(status, requestIDResponse{})

	if got := status.OpcRequestID; got != "opc-seeded" {
		t.Fatalf("status.opcRequestId = %q, want preserved seeded value", got)
	}
}

func TestRecordErrorOpcRequestIDReadsNormalizedOCIError(t *testing.T) {
	t.Parallel()

	_, err := errorutil.NewServiceFailureFromResponse("IncorrectState", 409, "opc-conflict-1", "conflict")
	if err == nil {
		t.Fatal("NewServiceFailureFromResponse() error = nil, want normalized conflict error")
	}

	status := &shared.OSOKStatus{}
	RecordErrorOpcRequestID(status, err)

	if got := status.OpcRequestID; got != "opc-conflict-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", got, "opc-conflict-1")
	}
}

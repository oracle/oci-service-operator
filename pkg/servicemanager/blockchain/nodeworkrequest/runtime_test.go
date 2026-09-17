/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */

package nodeworkrequest

import (
	"testing"

	"github.com/oracle/oci-go-sdk/v65/blockchain"
	"github.com/oracle/oci-go-sdk/v65/common"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

func TestRecoverResourceIDSelectsPhaseAndEntity(t *testing.T) {
	workRequest := blockchain.WorkRequest{
		Id: common.String("work-request"),
		Resources: []blockchain.WorkRequestResource{
			{EntityType: common.String("BlockchainPlatform"), ActionType: blockchain.WorkRequestResourceActionTypeUpdated, Identifier: common.String("platform-id")},
			{EntityType: common.String("Ordering-Service_Node"), ActionType: blockchain.WorkRequestResourceActionTypeCreated, Identifier: common.String("osn-key")},
		},
	}

	got, err := RecoverResourceID(workRequest, shared.OSOKAsyncPhaseCreate, "osn", "ordering service node")
	if err != nil {
		t.Fatal(err)
	}
	if got != "osn-key" {
		t.Fatalf("recovered resource ID = %q, want osn-key", got)
	}
}

func TestRecoverResourceIDRejectsAmbiguousMatches(t *testing.T) {
	workRequest := blockchain.WorkRequest{
		Id: common.String("work-request"),
		Resources: []blockchain.WorkRequestResource{
			{EntityType: common.String("peer"), ActionType: blockchain.WorkRequestResourceActionTypeUpdated, Identifier: common.String("peer-a")},
			{EntityType: common.String("Peer"), ActionType: blockchain.WorkRequestResourceActionTypeUpdated, Identifier: common.String("peer-b")},
		},
	}

	if _, err := RecoverResourceID(&workRequest, shared.OSOKAsyncPhaseUpdate, "peer"); err == nil {
		t.Fatal("expected ambiguous resource identifiers to fail")
	}
}

func TestRecoverResourceIDRejectsUnexpectedWorkRequest(t *testing.T) {
	if _, err := RecoverResourceID("not-a-work-request", shared.OSOKAsyncPhaseDelete, "peer"); err == nil {
		t.Fatal("expected unexpected work-request type to fail")
	}
}

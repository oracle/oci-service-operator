/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package adhocquery

import (
	"testing"

	cloudguardsdk "github.com/oracle/oci-go-sdk/v65/cloudguard"
	"github.com/oracle/oci-go-sdk/v65/common"
	cloudguardv1beta1 "github.com/oracle/oci-service-operator/api/cloudguard/v1beta1"
)

func TestAdhocQueryRuntimeSemanticsAndCollisionSafeProjection(t *testing.T) {
	semantics := newAdhocQueryRuntimeSemantics()
	if semantics == nil || len(semantics.Lifecycle.ActiveStates) != 1 || semantics.Lifecycle.ActiveStates[0] != "ACTIVE" {
		t.Fatalf("AdhocQuery semantics = %#v", semantics)
	}
	resource := &cloudguardv1beta1.AdhocQuery{}
	response := cloudguardsdk.GetAdhocQueryResponse{AdhocQuery: cloudguardsdk.AdhocQuery{
		Id:             common.String("ocid1.adhocquery.oc1..test"),
		CompartmentId:  common.String("ocid1.compartment.oc1..test"),
		Status:         cloudguardsdk.AdhocQueryStatusCompleted,
		LifecycleState: cloudguardsdk.LifecycleStateActive,
	}}
	if err := projectAdhocQueryStatus(resource, response); err != nil {
		t.Fatal(err)
	}
	if resource.Status.Status != "COMPLETED" || resource.Status.LifecycleState != "ACTIVE" {
		t.Fatalf("projected AdhocQuery status = %+v", resource.Status)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != "ocid1.adhocquery.oc1..test" {
		t.Fatalf("projected OCI identity = %q", got)
	}
}

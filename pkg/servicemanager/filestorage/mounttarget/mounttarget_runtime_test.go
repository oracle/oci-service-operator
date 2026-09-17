/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package mounttarget

import (
	"context"
	"reflect"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	filestoragesdk "github.com/oracle/oci-go-sdk/v65/filestorage"
	filestoragev1beta1 "github.com/oracle/oci-service-operator/api/filestorage/v1beta1"
)

func TestMountTargetRuntimeHooksModelLifecycleAndConfirmedDelete(t *testing.T) {
	t.Parallel()

	hooks := newMountTargetRuntimeHooks(&MountTargetServiceManager{}, filestoragesdk.FileStorageClient{})
	if hooks.Semantics == nil {
		t.Fatal("MountTarget runtime semantics = nil")
	}
	if len(hooks.Semantics.Lifecycle.ProvisioningStates) != 1 ||
		hooks.Semantics.Lifecycle.ProvisioningStates[0] != "CREATING" {
		t.Fatalf("provisioning states = %#v", hooks.Semantics.Lifecycle.ProvisioningStates)
	}
	if hooks.Semantics.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("delete follow-up = %q, want confirm-delete", hooks.Semantics.DeleteFollowUp.Strategy)
	}
	if hooks.Semantics.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("finalizer policy = %q", hooks.Semantics.FinalizerPolicy)
	}
}

func TestBuildMountTargetUpdateBodyEmitsOnlyReviewedDrift(t *testing.T) {
	t.Parallel()

	resource := &filestoragev1beta1.MountTarget{Spec: filestoragev1beta1.MountTargetSpec{
		DisplayName:  "updated-name",
		FreeformTags: map[string]string{"stage": "updated"},
	}}
	current := filestoragesdk.GetMountTargetResponse{MountTarget: filestoragesdk.MountTarget{
		DisplayName:  common.String("old-name"),
		FreeformTags: map[string]string{"stage": "old"},
	}}

	body, needed, err := buildMountTargetUpdateBody(context.Background(), resource, "", current)
	if err != nil {
		t.Fatal(err)
	}
	if !needed {
		t.Fatal("buildMountTargetUpdateBody() needed = false")
	}
	details := body.(filestoragesdk.UpdateMountTargetDetails)
	if details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName {
		t.Fatalf("displayName = %v", details.DisplayName)
	}
	if !reflect.DeepEqual(details.FreeformTags, resource.Spec.FreeformTags) {
		t.Fatalf("freeformTags = %#v", details.FreeformTags)
	}
	if details.LdapIdmap != nil || details.Kerberos != nil || details.IdmapType != "" {
		t.Fatalf("update contains unrequested optional blocks: %+v", details)
	}
}

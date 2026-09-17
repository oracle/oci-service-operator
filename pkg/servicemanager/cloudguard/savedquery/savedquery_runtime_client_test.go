/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package savedquery

import (
	"context"
	"reflect"
	"testing"

	cloudguardsdk "github.com/oracle/oci-go-sdk/v65/cloudguard"
	"github.com/oracle/oci-go-sdk/v65/common"
	cloudguardv1beta1 "github.com/oracle/oci-service-operator/api/cloudguard/v1beta1"
)

func TestBuildSavedQueryUpdateBodyPreservesFullProviderRequest(t *testing.T) {
	t.Parallel()
	resource := &cloudguardv1beta1.SavedQuery{Spec: cloudguardv1beta1.SavedQuerySpec{
		DisplayName:  "query",
		Query:        "select name from processes",
		Description:  "updated",
		FreeformTags: map[string]string{"phase": "updated"},
	}}
	current := cloudguardsdk.GetSavedQueryResponse{SavedQuery: cloudguardsdk.SavedQuery{
		DisplayName:  common.String("query"),
		Query:        common.String("select name from processes"),
		Description:  common.String("created"),
		FreeformTags: map[string]string{"phase": "created"},
	}}
	body, needed, err := buildSavedQueryUpdateBody(context.Background(), resource, "", current)
	if err != nil {
		t.Fatal(err)
	}
	if !needed {
		t.Fatal("buildSavedQueryUpdateBody() needed = false")
	}
	want := cloudguardsdk.UpdateSavedQueryDetails{
		DisplayName:  common.String("query"),
		Query:        common.String("select name from processes"),
		Description:  common.String("updated"),
		FreeformTags: map[string]string{"phase": "updated"},
	}
	if !reflect.DeepEqual(body, want) {
		t.Fatalf("buildSavedQueryUpdateBody() = %+v, want %+v", body, want)
	}
}

func TestBuildSavedQueryUpdateBodySkipsConvergedResource(t *testing.T) {
	t.Parallel()
	resource := &cloudguardv1beta1.SavedQuery{Spec: cloudguardv1beta1.SavedQuerySpec{
		DisplayName:  "query",
		Query:        "select name from processes",
		Description:  "same",
		FreeformTags: map[string]string{"phase": "same"},
	}}
	current := cloudguardsdk.GetSavedQueryResponse{SavedQuery: cloudguardsdk.SavedQuery{
		DisplayName:  common.String("query"),
		Query:        common.String("select name from processes"),
		Description:  common.String("same"),
		FreeformTags: map[string]string{"phase": "same"},
	}}
	_, needed, err := buildSavedQueryUpdateBody(context.Background(), resource, "", current)
	if err != nil {
		t.Fatal(err)
	}
	if needed {
		t.Fatal("buildSavedQueryUpdateBody() needed = true for converged resource")
	}
}

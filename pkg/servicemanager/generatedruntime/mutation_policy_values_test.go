/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	containerenginesdk "github.com/oracle/oci-go-sdk/v65/containerengine"
	mysqlsdk "github.com/oracle/oci-go-sdk/v65/mysql"
	containerenginev1beta1 "github.com/oracle/oci-service-operator/api/containerengine/v1beta1"
	mysqlv1beta1 "github.com/oracle/oci-service-operator/api/mysql/v1beta1"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

func TestMutationValuesMergesStatusOnlyFieldsWithLiveResponse(t *testing.T) {
	t.Parallel()
	resource := &mysqlv1beta1.DbSystem{Status: mysqlv1beta1.DbSystemStatus{DisplayName: "status-name", Source: mysqlv1beta1.DbSystemSourceObservedState{}, AdminUsername: shared.UsernameSource{Secret: shared.SecretSource{SecretName: "admin-secret"}}, AdminPassword: shared.PasswordSource{Secret: shared.SecretSource{SecretName: "admin-secret"}}}}
	_, currentValues, err := mutationValues(resource, mysqlsdk.GetDbSystemResponse{DbSystem: mysqlsdk.DbSystem{Id: common.String("ocid1.mysqldbsystem.oc1..created"), DisplayName: common.String("live-name")}})
	if err != nil {
		t.Fatalf("mutationValues() error = %v", err)
	}
	if got, ok := lookupValueByPath(currentValues, "displayName"); !ok || got != "live-name" {
		t.Fatalf("currentValues.displayName = %#v, want live response value", got)
	}
	if got, ok := lookupValueByPath(currentValues, "adminUsername.secret.secretName"); !ok || got != "admin-secret" {
		t.Fatalf("currentValues.adminUsername.secret.secretName = %#v, want status-only secret source preserved", got)
	}
	if got, ok := lookupValueByPath(currentValues, "adminPassword.secret.secretName"); !ok || got != "admin-secret" {
		t.Fatalf("currentValues.adminPassword.secret.secretName = %#v, want status-only secret source preserved", got)
	}
}

func TestValidateForceNewFieldsIgnoresEmptyNestedSpecObjects(t *testing.T) {
	t.Parallel()
	client := ServiceClient[*mysqlv1beta1.DbSystem]{config: Config[*mysqlv1beta1.DbSystem]{Kind: "DbSystem", Semantics: &Semantics{Mutation: MutationSemantics{ForceNew: []string{"source"}}}}}
	err := client.validateForceNewFields(&mysqlv1beta1.DbSystem{}, map[string]any{"source": map[string]any{}}, map[string]any{"source": map[string]any{"sourceType": "BACKUP"}})
	if err != nil {
		t.Fatalf("validateForceNewFields() error = %v, want empty spec object ignored", err)
	}
}

func TestValidateForceNewFieldsRejectsNestedFalseClusterBool(t *testing.T) {
	t.Parallel()
	client := ServiceClient[*containerenginev1beta1.Cluster]{config: Config[*containerenginev1beta1.Cluster]{Kind: "Cluster", Semantics: &Semantics{Mutation: MutationSemantics{ForceNew: []string{"endpointConfig.isPublicIpEnabled"}}}}}
	resource := &containerenginev1beta1.Cluster{Spec: containerenginev1beta1.ClusterSpec{EndpointConfig: containerenginev1beta1.ClusterEndpointConfig{IsPublicIpEnabled: false}}}
	specValues, currentValues, err := mutationValues(resource, containerenginesdk.GetClusterResponse{Cluster: containerenginesdk.Cluster{EndpointConfig: &containerenginesdk.ClusterEndpointConfig{IsPublicIpEnabled: common.Bool(true)}}})
	if err != nil {
		t.Fatalf("mutationValues() error = %v", err)
	}
	err = client.validateForceNewFields(resource, specValues, currentValues)
	if err == nil {
		t.Fatal("validateForceNewFields() error = nil, want explicit false bool drift rejected")
	}
	if want := "Cluster formal semantics require replacement when endpointConfig.isPublicIpEnabled changes"; err.Error() != want {
		t.Fatalf("validateForceNewFields() error = %v, want %q", err, want)
	}
}

func TestHasMutableDriftDetectsNestedFalseClusterBool(t *testing.T) {
	t.Parallel()
	client := ServiceClient[*containerenginev1beta1.Cluster]{config: Config[*containerenginev1beta1.Cluster]{Kind: "Cluster", Semantics: &Semantics{Mutation: MutationSemantics{Mutable: []string{"imagePolicyConfig.isPolicyEnabled"}}}}}
	resource := &containerenginev1beta1.Cluster{Spec: containerenginev1beta1.ClusterSpec{ImagePolicyConfig: containerenginev1beta1.ClusterImagePolicyConfig{IsPolicyEnabled: false}}}
	drifted, err := client.hasMutableDrift(resource, containerenginesdk.GetClusterResponse{Cluster: containerenginesdk.Cluster{ImagePolicyConfig: &containerenginesdk.ImagePolicyConfig{IsPolicyEnabled: common.Bool(true)}}})
	if err != nil {
		t.Fatalf("hasMutableDrift() error = %v", err)
	}
	if !drifted {
		t.Fatal("hasMutableDrift() = false, want explicit false bool drift detected")
	}
}

func TestFilteredUpdateBodyPreservesNestedFalseClusterBool(t *testing.T) {
	t.Parallel()
	client := ServiceClient[*containerenginev1beta1.Cluster]{config: Config[*containerenginev1beta1.Cluster]{Kind: "Cluster", Semantics: &Semantics{Mutation: MutationSemantics{Mutable: []string{"imagePolicyConfig.isPolicyEnabled"}}}, Update: &Operation{}}}
	resource := &containerenginev1beta1.Cluster{Spec: containerenginev1beta1.ClusterSpec{ImagePolicyConfig: containerenginev1beta1.ClusterImagePolicyConfig{IsPolicyEnabled: false}}}
	body, ok, err := client.filteredUpdateBody(resource, requestBuildOptions{CurrentResponse: containerenginesdk.GetClusterResponse{Cluster: containerenginesdk.Cluster{ImagePolicyConfig: &containerenginesdk.ImagePolicyConfig{IsPolicyEnabled: common.Bool(true)}}}})
	if err != nil {
		t.Fatalf("filteredUpdateBody() error = %v", err)
	}
	if !ok {
		t.Fatal("filteredUpdateBody() ok = false, want explicit false bool projected into update body")
	}
	bodyValues, ok := body.(map[string]any)
	if !ok {
		t.Fatalf("filteredUpdateBody() type = %T, want map[string]any", body)
	}
	if got, ok := lookupValueByPath(bodyValues, "imagePolicyConfig.isPolicyEnabled"); !ok {
		t.Fatal("filteredUpdateBody() omitted imagePolicyConfig.isPolicyEnabled")
	} else if boolValue, ok := got.(bool); !ok || boolValue {
		t.Fatalf("filteredUpdateBody() imagePolicyConfig.isPolicyEnabled = %#v, want false", got)
	}
}

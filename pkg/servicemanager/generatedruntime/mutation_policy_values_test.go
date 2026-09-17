/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	cloudguardsdk "github.com/oracle/oci-go-sdk/v65/cloudguard"
	"github.com/oracle/oci-go-sdk/v65/common"
	containerenginesdk "github.com/oracle/oci-go-sdk/v65/containerengine"
	dataintegrationsdk "github.com/oracle/oci-go-sdk/v65/dataintegration"
	mysqlsdk "github.com/oracle/oci-go-sdk/v65/mysql"
	cloudguardv1beta1 "github.com/oracle/oci-service-operator/api/cloudguard/v1beta1"
	containerenginev1beta1 "github.com/oracle/oci-service-operator/api/containerengine/v1beta1"
	dataintegrationv1beta1 "github.com/oracle/oci-service-operator/api/dataintegration/v1beta1"
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

func TestFilteredUpdateBodyOmitsDeclaredZeroObjectWhenObservedValueIsAbsent(t *testing.T) {
	t.Parallel()
	client := ServiceClient[*containerenginev1beta1.Cluster]{config: Config[*containerenginev1beta1.Cluster]{
		Kind: "Cluster",
		Semantics: &Semantics{Mutation: MutationSemantics{
			Mutable:                 []string{"imagePolicyConfig"},
			ZeroValueNullEquivalent: []string{"imagePolicyConfig"},
		}},
		Update: &Operation{},
	}}
	resource := &containerenginev1beta1.Cluster{Spec: containerenginev1beta1.ClusterSpec{
		ImagePolicyConfig: containerenginev1beta1.ClusterImagePolicyConfig{IsPolicyEnabled: false},
	}}
	body, ok, err := client.filteredUpdateBody(resource, requestBuildOptions{
		CurrentResponse: containerenginesdk.GetClusterResponse{Cluster: containerenginesdk.Cluster{}},
	})
	if err != nil {
		t.Fatalf("filteredUpdateBody() error = %v", err)
	}
	if ok || body != nil {
		t.Fatalf("filteredUpdateBody() = (%#v, %t), want no update for declared zero/null equivalence", body, ok)
	}
}

func TestFilteredUpdateBodyIncludesUnchangedMandatorySDKFieldsAfterDrift(t *testing.T) {
	t.Parallel()
	client := ServiceClient[*cloudguardv1beta1.WlpAgent]{config: Config[*cloudguardv1beta1.WlpAgent]{
		Kind: "WlpAgent",
		Semantics: &Semantics{Mutation: MutationSemantics{
			Mutable: []string{"certificateSignedRequest", "freeformTags"},
		}},
		Update: &Operation{
			NewRequest: func() any { return &cloudguardsdk.UpdateWlpAgentRequest{} },
			Fields: []RequestField{{
				FieldName:    "UpdateWlpAgentDetails",
				Contribution: "body",
			}},
		},
	}}
	resource := &cloudguardv1beta1.WlpAgent{Spec: cloudguardv1beta1.WlpAgentSpec{
		CertificateSignedRequest: "unchanged-csr",
		FreeformTags:             map[string]string{"phase": "updated"},
	}}
	current := cloudguardsdk.GetWlpAgentResponse{WlpAgent: cloudguardsdk.WlpAgent{
		CertificateSignedRequest: common.String("unchanged-csr"),
		FreeformTags:             map[string]string{"phase": "created"},
	}}

	body, ok, err := client.filteredUpdateBody(resource, requestBuildOptions{CurrentResponse: current})
	if err != nil {
		t.Fatalf("filteredUpdateBody() error = %v", err)
	}
	if !ok {
		t.Fatal("filteredUpdateBody() ok = false, want tag drift update")
	}
	values := body.(map[string]any)
	if got, exists := lookupValueByPath(values, "certificateSignedRequest"); !exists || got != "unchanged-csr" {
		t.Fatalf("mandatory certificateSignedRequest = %#v, exists=%t", got, exists)
	}
	if got, exists := lookupValueByPath(values, "freeformTags.phase"); !exists || got != "updated" {
		t.Fatalf("freeformTags.phase = %#v, exists=%t", got, exists)
	}
}

func TestFilteredUpdateBodySourcesMissingMandatorySDKFieldsFromObservedState(t *testing.T) {
	t.Parallel()
	client := ServiceClient[*dataintegrationv1beta1.Application]{config: Config[*dataintegrationv1beta1.Application]{
		Kind: "Application",
		Semantics: &Semantics{Mutation: MutationSemantics{
			Mutable: []string{"description", "key", "modelType", "objectVersion"},
		}},
		Update: &Operation{
			NewRequest: func() any { return &dataintegrationsdk.UpdateApplicationRequest{} },
			Fields:     []RequestField{{FieldName: "UpdateApplicationDetails", Contribution: "body"}},
		},
	}}
	resource := &dataintegrationv1beta1.Application{Spec: dataintegrationv1beta1.ApplicationSpec{
		Description: "updated", ObjectVersion: 1,
	}}
	current := dataintegrationsdk.GetApplicationResponse{Application: dataintegrationsdk.Application{
		Key: common.String("application-key"), ModelType: common.String("INTEGRATION_APPLICATION"),
		ObjectVersion: common.Int(1), Description: common.String("created"),
	}}
	body, ok, err := client.filteredUpdateBody(resource, requestBuildOptions{CurrentResponse: current})
	if err != nil {
		t.Fatalf("filteredUpdateBody() error = %v", err)
	}
	if !ok {
		t.Fatal("filteredUpdateBody() ok = false, want description update")
	}
	values := body.(map[string]any)
	for path, want := range map[string]any{
		"description":   "updated",
		"key":           "application-key",
		"modelType":     "INTEGRATION_APPLICATION",
		"objectVersion": float64(1),
	} {
		if got, exists := lookupValueByPath(values, path); !exists || !valuesEqual(got, want) {
			t.Fatalf("filteredUpdateBody() %s = %#v, exists=%t, want %#v", path, got, exists, want)
		}
	}
}

func TestFilteredUpdateBodyIncludesMandatoryFieldsForPolymorphicSDKBody(t *testing.T) {
	t.Parallel()
	client := ServiceClient[*dataintegrationv1beta1.Connection]{config: Config[*dataintegrationv1beta1.Connection]{
		Kind: "Connection",
		Semantics: &Semantics{Mutation: MutationSemantics{
			Mutable: []string{"description", "key", "objectVersion"},
		}},
		Update: &Operation{
			NewRequest: func() any { return &dataintegrationsdk.UpdateConnectionRequest{} },
			Fields:     []RequestField{{FieldName: "UpdateConnectionDetails", Contribution: "body"}},
		},
	}}
	resource := &dataintegrationv1beta1.Connection{Spec: dataintegrationv1beta1.ConnectionSpec{
		Description: "updated", ModelType: "REST_NO_AUTH_CONNECTION", ObjectVersion: 1,
	}}
	current := dataintegrationsdk.GetConnectionResponse{Connection: dataintegrationsdk.ConnectionFromRestNoAuth{
		Key: common.String("connection-key"), ObjectVersion: common.Int(1), Description: common.String("created"),
	}}
	body, ok, err := client.filteredUpdateBody(resource, requestBuildOptions{CurrentResponse: current})
	if err != nil {
		t.Fatalf("filteredUpdateBody() error = %v", err)
	}
	if !ok {
		t.Fatal("filteredUpdateBody() ok = false, want description update")
	}
	values := body.(map[string]any)
	for path, want := range map[string]any{
		"description":   "updated",
		"key":           "connection-key",
		"modelType":     "REST_NO_AUTH_CONNECTION",
		"objectVersion": float64(1),
	} {
		if got, exists := lookupValueByPath(values, path); !exists || !valuesEqual(got, want) {
			t.Fatalf("filteredUpdateBody() %s = %#v, exists=%t, want %#v", path, got, exists, want)
		}
	}
}

func TestMandatoryUpdateBodyFieldPathsUsesConcretePolymorphicSDKType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		targetType reflect.Type
		modelType  string
		want       []string
	}{
		{
			name: "HDFS connection", targetType: dataIntegrationConnectionUpdateType, modelType: "HDFS_CONNECTION",
			want: []string{"dataNodePrincipal", "hdfsPrincipal", "key", "nameNodePrincipal", "objectVersion"},
		},
		{
			name: "REST data asset", targetType: dataIntegrationDataAssetUpdateType, modelType: "REST_DATA_ASSET",
			want: []string{"baseUrl", "defaultConnection", "key", "manifestFileContent", "objectVersion"},
		},
		{
			name: "REST task", targetType: dataIntegrationTaskUpdateType, modelType: "REST_TASK",
			want: []string{"key", "objectVersion"},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := mandatoryUpdateBodyFieldPaths(test.targetType, map[string]any{"modelType": test.modelType}, nil)
			if err != nil {
				t.Fatal(err)
			}
			sort.Strings(got)
			sort.Strings(test.want)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("mandatoryUpdateBodyFieldPaths() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestIncludeMandatoryUpdateBodyFieldsCarriesNestedRequiredSibling(t *testing.T) {
	t.Parallel()
	type workflow struct {
		Scope    *string  `mandatory:"true" json:"scope"`
		Workflow []string `mandatory:"true" json:"workflow"`
	}
	type details struct {
		RollbackWorkflowDetails *workflow `mandatory:"false" json:"rollbackWorkflowDetails"`
	}
	type request struct {
		Details details `contributesTo:"body"`
	}

	body := map[string]any{"rollbackWorkflowDetails": map[string]any{"scope": "TARGET"}}
	spec := map[string]any{"rollbackWorkflowDetails": map[string]any{"scope": "TARGET", "workflow": []any{}}}
	operation := &Operation{
		NewRequest: func() any { return &request{} },
		Fields:     []RequestField{{FieldName: "Details", Contribution: "body"}},
	}
	if err := includeMandatoryUpdateBodyFields(body, spec, nil, operation); err != nil {
		t.Fatal(err)
	}
	workflowValue, ok := lookupValueByPath(body, "rollbackWorkflowDetails.workflow")
	if !ok {
		t.Fatalf("body = %#v, want nested required workflow", body)
	}
	if items, ok := workflowValue.([]any); !ok || len(items) != 0 {
		t.Fatalf("rollbackWorkflowDetails.workflow = %#v, want explicit empty list", workflowValue)
	}
}

func TestFilteredUpdateBodyDoesNotEmitMandatoryFieldsWithoutDrift(t *testing.T) {
	t.Parallel()
	client := ServiceClient[*cloudguardv1beta1.WlpAgent]{config: Config[*cloudguardv1beta1.WlpAgent]{
		Kind: "WlpAgent",
		Semantics: &Semantics{Mutation: MutationSemantics{
			Mutable: []string{"certificateSignedRequest", "freeformTags"},
		}},
		Update: &Operation{
			NewRequest: func() any { return &cloudguardsdk.UpdateWlpAgentRequest{} },
			Fields: []RequestField{{
				FieldName:    "UpdateWlpAgentDetails",
				Contribution: "body",
			}},
		},
	}}
	resource := &cloudguardv1beta1.WlpAgent{Spec: cloudguardv1beta1.WlpAgentSpec{
		CertificateSignedRequest: "same-csr",
		FreeformTags:             map[string]string{"phase": "same"},
	}}
	current := cloudguardsdk.GetWlpAgentResponse{WlpAgent: cloudguardsdk.WlpAgent{
		CertificateSignedRequest: common.String("same-csr"),
		FreeformTags:             map[string]string{"phase": "same"},
	}}

	body, ok, err := client.filteredUpdateBody(resource, requestBuildOptions{CurrentResponse: current})
	if err != nil {
		t.Fatalf("filteredUpdateBody() error = %v", err)
	}
	if ok || body != nil {
		t.Fatalf("filteredUpdateBody() = (%#v, %t), want no update", body, ok)
	}
}

func TestValidateForceNewTreatsDeclaredZeroOptionalObjectAsNull(t *testing.T) {
	t.Parallel()
	client := ServiceClient[*containerenginev1beta1.Cluster]{config: Config[*containerenginev1beta1.Cluster]{
		Kind: "Cluster",
		Semantics: &Semantics{Mutation: MutationSemantics{
			ForceNew:                []string{"imagePolicyConfig"},
			ZeroValueNullEquivalent: []string{"imagePolicyConfig"},
		}},
	}}
	resource := &containerenginev1beta1.Cluster{
		Spec: containerenginev1beta1.ClusterSpec{
			ImagePolicyConfig: containerenginev1beta1.ClusterImagePolicyConfig{IsPolicyEnabled: false},
		},
	}
	current := containerenginesdk.GetClusterResponse{Cluster: containerenginesdk.Cluster{ImagePolicyConfig: nil}}
	if err := client.validateMutationPolicy(resource, true, current); err != nil {
		t.Fatalf("validateMutationPolicy() error = %v, want declared zero object and null to compare equal", err)
	}
}

func TestValidateForceNewPreservesDeclaredNonzeroOptionalObjectIntent(t *testing.T) {
	t.Parallel()
	client := ServiceClient[*containerenginev1beta1.Cluster]{config: Config[*containerenginev1beta1.Cluster]{
		Kind: "Cluster",
		Semantics: &Semantics{Mutation: MutationSemantics{
			ForceNew:                []string{"imagePolicyConfig"},
			ZeroValueNullEquivalent: []string{"imagePolicyConfig"},
		}},
	}}
	resource := &containerenginev1beta1.Cluster{
		Spec: containerenginev1beta1.ClusterSpec{
			ImagePolicyConfig: containerenginev1beta1.ClusterImagePolicyConfig{IsPolicyEnabled: true},
		},
	}
	current := containerenginesdk.GetClusterResponse{Cluster: containerenginesdk.Cluster{ImagePolicyConfig: nil}}
	err := client.validateMutationPolicy(resource, true, current)
	if err == nil || !strings.Contains(err.Error(), "require replacement when imagePolicyConfig changes") {
		t.Fatalf("validateMutationPolicy() error = %v, want nonzero force-new drift", err)
	}
}

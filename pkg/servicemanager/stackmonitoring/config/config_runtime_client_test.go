/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package config

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	stackmonitoringsdk "github.com/oracle/oci-go-sdk/v65/stackmonitoring"
	stackmonitoringv1beta1 "github.com/oracle/oci-service-operator/api/stackmonitoring/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testConfigID            = "ocid1.stackmonitoringconfig.oc1..config"
	testConfigCompartmentID = "ocid1.compartment.oc1..config"
)

func TestConfigRuntimeHooksConfigured(t *testing.T) {
	hooks := newConfigDefaultRuntimeHooks(stackmonitoringsdk.StackMonitoringClient{})
	applyConfigRuntimeHooks(&hooks)

	assertConfigRuntimeHooksRegistered(t, hooks)
	assertConfigAutoPromoteCreateBody(t, hooks)
}

func assertConfigRuntimeHooksRegistered(t *testing.T, hooks ConfigRuntimeHooks) {
	t.Helper()

	checks := []struct {
		name string
		ok   bool
	}{
		{name: "Semantics", ok: hooks.Semantics != nil},
		{name: "BuildCreateBody", ok: hooks.BuildCreateBody != nil},
		{name: "BuildUpdateBody", ok: hooks.BuildUpdateBody != nil},
		{name: "Identity.Resolve", ok: hooks.Identity.Resolve != nil},
		{name: "DeleteHooks.HandleError", ok: hooks.DeleteHooks.HandleError != nil},
		{name: "WrapGeneratedClient", ok: len(hooks.WrapGeneratedClient) > 0},
	}
	for _, check := range checks {
		if !check.ok {
			t.Fatalf("hooks.%s not configured", check.name)
		}
	}
}

func assertConfigAutoPromoteCreateBody(t *testing.T, hooks ConfigRuntimeHooks) {
	t.Helper()

	body, err := hooks.BuildCreateBody(context.Background(), testAutoPromoteConfig(), "default")
	if err != nil {
		t.Fatalf("BuildCreateBody() error = %v", err)
	}
	details, ok := body.(stackmonitoringsdk.CreateAutoPromoteConfigDetails)
	if !ok {
		t.Fatalf("BuildCreateBody() type = %T, want CreateAutoPromoteConfigDetails", body)
	}
	if details.IsEnabled == nil || *details.IsEnabled {
		t.Fatalf("BuildCreateBody() IsEnabled = %#v, want explicit false pointer", details.IsEnabled)
	}
	if got := details.ResourceType; got != stackmonitoringsdk.CreateAutoPromoteConfigDetailsResourceTypeHost {
		t.Fatalf("BuildCreateBody() ResourceType = %q, want HOST", got)
	}
}

func TestConfigCreateBodySupportsPolymorphicConfigTypes(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(*stackmonitoringv1beta1.Config)
		wantType any
	}{
		{
			name:     "auto promote",
			wantType: stackmonitoringsdk.CreateAutoPromoteConfigDetails{},
		},
		{
			name: "compute auto activate plugin",
			mutate: func(resource *stackmonitoringv1beta1.Config) {
				resource.Spec.ConfigType = string(stackmonitoringsdk.ConfigConfigTypeComputeAutoActivatePlugin)
				resource.Spec.ResourceType = ""
				resource.Spec.IsEnabled = true
			},
			wantType: stackmonitoringsdk.CreateComputeAutoActivatePluginConfigDetails{},
		},
		{
			name: "license auto assign",
			mutate: func(resource *stackmonitoringv1beta1.Config) {
				resource.Spec.ConfigType = string(stackmonitoringsdk.ConfigConfigTypeLicenseAutoAssign)
				resource.Spec.ResourceType = ""
				resource.Spec.License = string(stackmonitoringsdk.LicenseTypeEnterpriseEdition)
			},
			wantType: stackmonitoringsdk.CreateLicenseAutoAssignConfigDetails{},
		},
		{
			name: "license enterprise extensibility",
			mutate: func(resource *stackmonitoringv1beta1.Config) {
				resource.Spec.ConfigType = string(stackmonitoringsdk.ConfigConfigTypeLicenseEnterpriseExtensibility)
				resource.Spec.ResourceType = ""
				resource.Spec.IsEnabled = true
			},
			wantType: stackmonitoringsdk.CreateLicenseEnterpriseExtensibilityConfigDetails{},
		},
		{
			name: "onboard",
			mutate: func(resource *stackmonitoringv1beta1.Config) {
				resource.Spec.ConfigType = string(stackmonitoringsdk.ConfigConfigTypeOnboard)
				resource.Spec.ResourceType = ""
				resource.Spec.IsManuallyOnboarded = true
				resource.Spec.PolicyNames = []string{"policy-a"}
				resource.Spec.DynamicGroups = []stackmonitoringv1beta1.ConfigDynamicGroup{{
					Name:                      "dg",
					StackMonitoringAssignment: string(stackmonitoringsdk.DynamicGroupDetailsStackMonitoringAssignmentManagementAgents),
				}}
				resource.Spec.UserGroups = []stackmonitoringv1beta1.ConfigUserGroup{{
					Name:                "ug",
					StackMonitoringRole: "ADMINISTRATOR",
				}}
				resource.Spec.AdditionalConfigurations.PropertiesMap = map[string]string{"k": "v"}
			},
			wantType: stackmonitoringsdk.CreateOnboardConfigDetails{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := testAutoPromoteConfig()
			if tt.mutate != nil {
				tt.mutate(resource)
			}
			body, err := buildConfigCreateBody(context.Background(), resource, "default")
			if err != nil {
				t.Fatalf("buildConfigCreateBody() error = %v", err)
			}
			if got, want := typeName(body), typeName(tt.wantType); got != want {
				t.Fatalf("buildConfigCreateBody() type = %s, want %s", got, want)
			}
		})
	}
}

func TestConfigCreateRecordsIdentityAndRequestID(t *testing.T) {
	resource := testAutoPromoteConfig()
	config := sdkAutoPromoteConfig(resource, testConfigID, stackmonitoringsdk.ConfigLifecycleStateActive)
	fake := &fakeConfigOCIClient{
		createConfig: func(_ context.Context, request stackmonitoringsdk.CreateConfigRequest) (stackmonitoringsdk.CreateConfigResponse, error) {
			assertConfigCreateRequest(t, request)
			return stackmonitoringsdk.CreateConfigResponse{
				Config:       config,
				OpcRequestId: common.String("opc-create"),
			}, nil
		},
		getConfig: func(context.Context, stackmonitoringsdk.GetConfigRequest) (stackmonitoringsdk.GetConfigResponse, error) {
			return stackmonitoringsdk.GetConfigResponse{Config: config}, nil
		},
	}

	response, err := newTestConfigClient(fake).CreateOrUpdate(context.Background(), resource, testConfigRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() IsSuccessful = false, want true")
	}
	assertConfigCallCount(t, "CreateConfig()", fake.createCalls, 1)
	assertConfigRecordedID(t, resource, testConfigID)
	assertConfigOpcRequestID(t, resource, "opc-create")
}

func TestConfigCreateOrUpdateBindsExistingFromLaterListPage(t *testing.T) {
	resource := testAutoPromoteConfig()
	config := sdkAutoPromoteConfig(resource, testConfigID, stackmonitoringsdk.ConfigLifecycleStateActive)
	var pages []string
	fake := &fakeConfigOCIClient{
		listConfigs: func(_ context.Context, request stackmonitoringsdk.ListConfigsRequest) (stackmonitoringsdk.ListConfigsResponse, error) {
			pages = append(pages, stringValue(request.Page))
			if request.Page == nil {
				return stackmonitoringsdk.ListConfigsResponse{
					ConfigCollection: stackmonitoringsdk.ConfigCollection{
						Items: []stackmonitoringsdk.ConfigSummary{
							sdkAutoPromoteSummary(resource, "ocid1.stackmonitoringconfig.oc1..other", "terminal-config", stackmonitoringsdk.ConfigLifecycleStateDeleted),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			return stackmonitoringsdk.ListConfigsResponse{
				ConfigCollection: stackmonitoringsdk.ConfigCollection{
					Items: []stackmonitoringsdk.ConfigSummary{
						sdkAutoPromoteSummary(resource, testConfigID, "renamed-in-oci", stackmonitoringsdk.ConfigLifecycleStateActive),
					},
				},
			}, nil
		},
		getConfig: func(_ context.Context, request stackmonitoringsdk.GetConfigRequest) (stackmonitoringsdk.GetConfigResponse, error) {
			if got := stringValue(request.ConfigId); got != testConfigID {
				t.Fatalf("GetConfig() ConfigId = %q, want %q", got, testConfigID)
			}
			return stackmonitoringsdk.GetConfigResponse{Config: config}, nil
		},
	}

	response, err := newTestConfigClient(fake).CreateOrUpdate(context.Background(), resource, testConfigRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() IsSuccessful = false, want true")
	}
	assertConfigCallCount(t, "CreateConfig()", fake.createCalls, 0)
	if got := strings.Join(pages, ","); got != ",page-2" {
		t.Fatalf("ListConfigs() pages = %q, want \",page-2\"", got)
	}
	assertConfigRecordedID(t, resource, testConfigID)
}

func TestConfigCreateOrUpdateNoopsWhenObservedStateMatches(t *testing.T) {
	resource := testAutoPromoteConfig()
	resource.Status.OsokStatus.Ocid = shared.OCID(testConfigID)
	config := sdkAutoPromoteConfig(resource, testConfigID, stackmonitoringsdk.ConfigLifecycleStateActive)
	fake := &fakeConfigOCIClient{
		getConfig: func(context.Context, stackmonitoringsdk.GetConfigRequest) (stackmonitoringsdk.GetConfigResponse, error) {
			return stackmonitoringsdk.GetConfigResponse{Config: config}, nil
		},
		updateConfig: func(context.Context, stackmonitoringsdk.UpdateConfigRequest) (stackmonitoringsdk.UpdateConfigResponse, error) {
			t.Fatal("UpdateConfig() called during no-op reconcile")
			return stackmonitoringsdk.UpdateConfigResponse{}, nil
		},
	}

	response, err := newTestConfigClient(fake).CreateOrUpdate(context.Background(), resource, testConfigRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() IsSuccessful = false, want true")
	}
}

func TestConfigMutableUpdateUsesPolymorphicUpdatePath(t *testing.T) {
	resource := testAutoPromoteConfig()
	resource.Status.OsokStatus.Ocid = shared.OCID(testConfigID)
	current := sdkAutoPromoteConfig(resource, testConfigID, stackmonitoringsdk.ConfigLifecycleStateActive)
	current.IsEnabled = common.Bool(true)
	updated := sdkAutoPromoteConfig(resource, testConfigID, stackmonitoringsdk.ConfigLifecycleStateActive)
	getResponses := []stackmonitoringsdk.GetConfigResponse{
		{Config: current},
		{Config: updated},
	}
	fake := &fakeConfigOCIClient{
		getConfig: getConfigResponses(t, &getResponses),
		updateConfig: func(_ context.Context, request stackmonitoringsdk.UpdateConfigRequest) (stackmonitoringsdk.UpdateConfigResponse, error) {
			assertConfigUpdateRequest(t, request)
			return stackmonitoringsdk.UpdateConfigResponse{
				Config:       updated,
				OpcRequestId: common.String("opc-update"),
			}, nil
		},
	}

	response, err := newTestConfigClient(fake).CreateOrUpdate(context.Background(), resource, testConfigRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() IsSuccessful = false, want true")
	}
	assertConfigCallCount(t, "UpdateConfig()", fake.updateCalls, 1)
	assertConfigOpcRequestID(t, resource, "opc-update")
}

func TestConfigImmutableDriftRejectedBeforeUpdate(t *testing.T) {
	resource := testAutoPromoteConfig()
	resource.Status.OsokStatus.Ocid = shared.OCID(testConfigID)
	current := sdkAutoPromoteConfig(resource, testConfigID, stackmonitoringsdk.ConfigLifecycleStateActive)
	current.CompartmentId = common.String("ocid1.compartment.oc1..different")
	fake := &fakeConfigOCIClient{
		getConfig: func(context.Context, stackmonitoringsdk.GetConfigRequest) (stackmonitoringsdk.GetConfigResponse, error) {
			return stackmonitoringsdk.GetConfigResponse{Config: current}, nil
		},
		updateConfig: func(context.Context, stackmonitoringsdk.UpdateConfigRequest) (stackmonitoringsdk.UpdateConfigResponse, error) {
			t.Fatal("UpdateConfig() called despite compartmentId drift")
			return stackmonitoringsdk.UpdateConfigResponse{}, nil
		},
	}

	_, err := newTestConfigClient(fake).CreateOrUpdate(context.Background(), resource, testConfigRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want immutable drift rejection")
	}
	if !strings.Contains(err.Error(), "compartmentId") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartmentId drift detail", err)
	}
}

func TestConfigDeleteRetainsFinalizerWhileDeleteIsPending(t *testing.T) {
	resource := testAutoPromoteConfig()
	resource.Status.OsokStatus.Ocid = shared.OCID(testConfigID)
	active := sdkAutoPromoteConfig(resource, testConfigID, stackmonitoringsdk.ConfigLifecycleStateActive)
	deleting := sdkAutoPromoteConfig(resource, testConfigID, stackmonitoringsdk.ConfigLifecycleStateDeleting)
	getResponses := []stackmonitoringsdk.GetConfigResponse{
		{Config: active},
		{Config: active},
		{Config: deleting},
	}
	fake := &fakeConfigOCIClient{
		getConfig: getConfigResponses(t, &getResponses),
		deleteConfig: func(_ context.Context, request stackmonitoringsdk.DeleteConfigRequest) (stackmonitoringsdk.DeleteConfigResponse, error) {
			if got := stringValue(request.ConfigId); got != testConfigID {
				t.Fatalf("DeleteConfig() ConfigId = %q, want %q", got, testConfigID)
			}
			return stackmonitoringsdk.DeleteConfigResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}

	deleted, err := newTestConfigClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while OCI lifecycle is DELETING")
	}
	assertConfigCallCount(t, "DeleteConfig()", fake.deleteCalls, 1)
	assertConfigOpcRequestID(t, resource, "opc-delete")
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.status.async.current = nil, want pending delete lifecycle tracker")
	}
}

func TestConfigDeleteRejectsAuthShapedPreDeleteRead(t *testing.T) {
	resource := testAutoPromoteConfig()
	resource.Status.OsokStatus.Ocid = shared.OCID(testConfigID)
	authNotFound := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	authNotFound.OpcRequestID = "opc-delete-auth"
	fake := &fakeConfigOCIClient{
		getConfig: func(context.Context, stackmonitoringsdk.GetConfigRequest) (stackmonitoringsdk.GetConfigResponse, error) {
			return stackmonitoringsdk.GetConfigResponse{}, authNotFound
		},
		deleteConfig: func(context.Context, stackmonitoringsdk.DeleteConfigRequest) (stackmonitoringsdk.DeleteConfigResponse, error) {
			t.Fatal("DeleteConfig() called after auth-shaped pre-delete read")
			return stackmonitoringsdk.DeleteConfigResponse{}, nil
		},
	}

	deleted, err := newTestConfigClient(fake).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous auth-shaped not-found rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous 404 NotAuthorizedOrNotFound", err)
	}
	assertConfigOpcRequestID(t, resource, "opc-delete-auth")
	assertConfigCallCount(t, "DeleteConfig()", fake.deleteCalls, 0)
}

func TestConfigCreateOrUpdateRecordsOpcRequestIDFromOCIError(t *testing.T) {
	resource := testAutoPromoteConfig()
	createErr := errortest.NewServiceError(500, errorutil.InternalServerError, "create failed")
	createErr.OpcRequestID = "opc-create-error"
	fake := &fakeConfigOCIClient{
		createConfig: func(context.Context, stackmonitoringsdk.CreateConfigRequest) (stackmonitoringsdk.CreateConfigResponse, error) {
			return stackmonitoringsdk.CreateConfigResponse{}, createErr
		},
	}

	_, err := newTestConfigClient(fake).CreateOrUpdate(context.Background(), resource, testConfigRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI create error")
	}
	assertConfigOpcRequestID(t, resource, "opc-create-error")
}

func newTestConfigClient(fake *fakeConfigOCIClient) ConfigServiceClient {
	hooks := newConfigDefaultRuntimeHooks(stackmonitoringsdk.StackMonitoringClient{})
	hooks.Create.Call = fake.CreateConfig
	hooks.Get.Call = fake.GetConfig
	hooks.List.Call = fake.ListConfigs
	hooks.Update.Call = fake.UpdateConfig
	hooks.Delete.Call = fake.DeleteConfig
	applyConfigRuntimeHooks(&hooks)
	manager := &ConfigServiceManager{Log: loggerutil.OSOKLogger{Logger: logr.Discard()}}
	delegate := defaultConfigServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*stackmonitoringv1beta1.Config](
			buildConfigGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapConfigGeneratedClient(hooks, delegate)
}

type fakeConfigOCIClient struct {
	createConfig func(context.Context, stackmonitoringsdk.CreateConfigRequest) (stackmonitoringsdk.CreateConfigResponse, error)
	getConfig    func(context.Context, stackmonitoringsdk.GetConfigRequest) (stackmonitoringsdk.GetConfigResponse, error)
	listConfigs  func(context.Context, stackmonitoringsdk.ListConfigsRequest) (stackmonitoringsdk.ListConfigsResponse, error)
	updateConfig func(context.Context, stackmonitoringsdk.UpdateConfigRequest) (stackmonitoringsdk.UpdateConfigResponse, error)
	deleteConfig func(context.Context, stackmonitoringsdk.DeleteConfigRequest) (stackmonitoringsdk.DeleteConfigResponse, error)

	createCalls int
	getCalls    int
	listCalls   int
	updateCalls int
	deleteCalls int
}

func (f *fakeConfigOCIClient) CreateConfig(
	ctx context.Context,
	request stackmonitoringsdk.CreateConfigRequest,
) (stackmonitoringsdk.CreateConfigResponse, error) {
	f.createCalls++
	if f.createConfig == nil {
		return stackmonitoringsdk.CreateConfigResponse{}, nil
	}
	return f.createConfig(ctx, request)
}

func (f *fakeConfigOCIClient) GetConfig(
	ctx context.Context,
	request stackmonitoringsdk.GetConfigRequest,
) (stackmonitoringsdk.GetConfigResponse, error) {
	f.getCalls++
	if f.getConfig == nil {
		return stackmonitoringsdk.GetConfigResponse{}, nil
	}
	return f.getConfig(ctx, request)
}

func (f *fakeConfigOCIClient) ListConfigs(
	ctx context.Context,
	request stackmonitoringsdk.ListConfigsRequest,
) (stackmonitoringsdk.ListConfigsResponse, error) {
	f.listCalls++
	if f.listConfigs == nil {
		return stackmonitoringsdk.ListConfigsResponse{}, nil
	}
	return f.listConfigs(ctx, request)
}

func (f *fakeConfigOCIClient) UpdateConfig(
	ctx context.Context,
	request stackmonitoringsdk.UpdateConfigRequest,
) (stackmonitoringsdk.UpdateConfigResponse, error) {
	f.updateCalls++
	if f.updateConfig == nil {
		return stackmonitoringsdk.UpdateConfigResponse{}, nil
	}
	return f.updateConfig(ctx, request)
}

func (f *fakeConfigOCIClient) DeleteConfig(
	ctx context.Context,
	request stackmonitoringsdk.DeleteConfigRequest,
) (stackmonitoringsdk.DeleteConfigResponse, error) {
	f.deleteCalls++
	if f.deleteConfig == nil {
		return stackmonitoringsdk.DeleteConfigResponse{}, nil
	}
	return f.deleteConfig(ctx, request)
}

func getConfigResponses(
	t *testing.T,
	responses *[]stackmonitoringsdk.GetConfigResponse,
) func(context.Context, stackmonitoringsdk.GetConfigRequest) (stackmonitoringsdk.GetConfigResponse, error) {
	t.Helper()

	return func(context.Context, stackmonitoringsdk.GetConfigRequest) (stackmonitoringsdk.GetConfigResponse, error) {
		if len(*responses) == 0 {
			t.Fatal("GetConfig() called more times than expected")
		}
		response := (*responses)[0]
		*responses = (*responses)[1:]
		return response, nil
	}
}

func testAutoPromoteConfig() *stackmonitoringv1beta1.Config {
	return &stackmonitoringv1beta1.Config{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "config",
			Namespace: "default",
			UID:       types.UID("config-uid"),
		},
		Spec: stackmonitoringv1beta1.ConfigSpec{
			CompartmentId: testConfigCompartmentID,
			DisplayName:   "config",
			ConfigType:    string(stackmonitoringsdk.ConfigConfigTypeAutoPromote),
			ResourceType:  string(stackmonitoringsdk.CreateAutoPromoteConfigDetailsResourceTypeHost),
			IsEnabled:     false,
			FreeformTags:  map[string]string{"owner": "osok"},
		},
	}
}

func testConfigRequest(resource *stackmonitoringv1beta1.Config) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		},
	}
}

func sdkAutoPromoteConfig(
	resource *stackmonitoringv1beta1.Config,
	id string,
	lifecycle stackmonitoringsdk.ConfigLifecycleStateEnum,
) stackmonitoringsdk.AutoPromoteConfigDetails {
	return stackmonitoringsdk.AutoPromoteConfigDetails{
		Id:             common.String(id),
		CompartmentId:  common.String(resource.Spec.CompartmentId),
		DisplayName:    common.String(resource.Spec.DisplayName),
		FreeformTags:   cloneConfigStringMap(resource.Spec.FreeformTags),
		ResourceType:   stackmonitoringsdk.AutoPromoteConfigDetailsResourceTypeHost,
		IsEnabled:      common.Bool(resource.Spec.IsEnabled),
		LifecycleState: lifecycle,
	}
}

func sdkAutoPromoteSummary(
	resource *stackmonitoringv1beta1.Config,
	id string,
	displayName string,
	lifecycle stackmonitoringsdk.ConfigLifecycleStateEnum,
) stackmonitoringsdk.AutoPromoteConfigSummary {
	return stackmonitoringsdk.AutoPromoteConfigSummary{
		Id:             common.String(id),
		CompartmentId:  common.String(resource.Spec.CompartmentId),
		DisplayName:    common.String(displayName),
		FreeformTags:   cloneConfigStringMap(resource.Spec.FreeformTags),
		ResourceType:   stackmonitoringsdk.AutoPromoteConfigSummaryResourceTypeHost,
		IsEnabled:      common.Bool(resource.Spec.IsEnabled),
		LifecycleState: lifecycle,
	}
}

func assertConfigCreateRequest(t *testing.T, request stackmonitoringsdk.CreateConfigRequest) {
	t.Helper()

	details, ok := request.CreateConfigDetails.(stackmonitoringsdk.CreateAutoPromoteConfigDetails)
	if !ok {
		t.Fatalf("CreateConfig() body type = %T, want CreateAutoPromoteConfigDetails", request.CreateConfigDetails)
	}
	if got := stringValue(details.CompartmentId); got != testConfigCompartmentID {
		t.Fatalf("CreateConfig() compartmentId = %q, want %q", got, testConfigCompartmentID)
	}
	if details.IsEnabled == nil || *details.IsEnabled {
		t.Fatalf("CreateConfig() IsEnabled = %#v, want explicit false pointer", details.IsEnabled)
	}
	if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
		t.Fatal("CreateConfig() OpcRetryToken is empty, want deterministic retry token")
	}
}

func assertConfigUpdateRequest(t *testing.T, request stackmonitoringsdk.UpdateConfigRequest) {
	t.Helper()

	if got := stringValue(request.ConfigId); got != testConfigID {
		t.Fatalf("UpdateConfig() ConfigId = %q, want %q", got, testConfigID)
	}
	details, ok := request.UpdateConfigDetails.(stackmonitoringsdk.UpdateAutoPromoteConfigDetails)
	if !ok {
		t.Fatalf("UpdateConfig() body type = %T, want UpdateAutoPromoteConfigDetails", request.UpdateConfigDetails)
	}
	if details.IsEnabled == nil || *details.IsEnabled {
		t.Fatalf("UpdateConfig() IsEnabled = %#v, want explicit false pointer", details.IsEnabled)
	}
}

func assertConfigRecordedID(t *testing.T, resource *stackmonitoringv1beta1.Config, want string) {
	t.Helper()

	if got := string(resource.Status.OsokStatus.Ocid); got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
	if got := resource.Status.Id; got != want {
		t.Fatalf("status.id = %q, want %q", got, want)
	}
}

func assertConfigOpcRequestID(t *testing.T, resource *stackmonitoringv1beta1.Config, want string) {
	t.Helper()

	if got := resource.Status.OsokStatus.OpcRequestID; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func assertConfigCallCount(t *testing.T, operation string, got int, want int) {
	t.Helper()

	if got != want {
		t.Fatalf("%s calls = %d, want %d", operation, got, want)
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func typeName(value any) string {
	if value == nil {
		return "<nil>"
	}
	return reflect.TypeOf(value).String()
}

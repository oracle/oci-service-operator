/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package wlsdomain

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	wlmssdk "github.com/oracle/oci-go-sdk/v65/wlms"
	wlmsv1beta1 "github.com/oracle/oci-service-operator/api/wlms/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testWlsDomainID          = "ocid1.wlsdomain.oc1..example"
	testWlsDomainCompartment = "ocid1.compartment.oc1..example"
	testWlsDomainName        = "wlsdomain-sample"
)

type fakeWlsDomainOCIClient struct {
	getFn    func(context.Context, wlmssdk.GetWlsDomainRequest) (wlmssdk.GetWlsDomainResponse, error)
	listFn   func(context.Context, wlmssdk.ListWlsDomainsRequest) (wlmssdk.ListWlsDomainsResponse, error)
	updateFn func(context.Context, wlmssdk.UpdateWlsDomainRequest) (wlmssdk.UpdateWlsDomainResponse, error)
	deleteFn func(context.Context, wlmssdk.DeleteWlsDomainRequest) (wlmssdk.DeleteWlsDomainResponse, error)

	getRequests    []wlmssdk.GetWlsDomainRequest
	listRequests   []wlmssdk.ListWlsDomainsRequest
	updateRequests []wlmssdk.UpdateWlsDomainRequest
	deleteRequests []wlmssdk.DeleteWlsDomainRequest
}

func (f *fakeWlsDomainOCIClient) GetWlsDomain(
	ctx context.Context,
	req wlmssdk.GetWlsDomainRequest,
) (wlmssdk.GetWlsDomainResponse, error) {
	f.getRequests = append(f.getRequests, req)
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return wlmssdk.GetWlsDomainResponse{}, nil
}

func (f *fakeWlsDomainOCIClient) ListWlsDomains(
	ctx context.Context,
	req wlmssdk.ListWlsDomainsRequest,
) (wlmssdk.ListWlsDomainsResponse, error) {
	f.listRequests = append(f.listRequests, req)
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return wlmssdk.ListWlsDomainsResponse{}, nil
}

func (f *fakeWlsDomainOCIClient) UpdateWlsDomain(
	ctx context.Context,
	req wlmssdk.UpdateWlsDomainRequest,
) (wlmssdk.UpdateWlsDomainResponse, error) {
	f.updateRequests = append(f.updateRequests, req)
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return wlmssdk.UpdateWlsDomainResponse{}, nil
}

func (f *fakeWlsDomainOCIClient) DeleteWlsDomain(
	ctx context.Context,
	req wlmssdk.DeleteWlsDomainRequest,
) (wlmssdk.DeleteWlsDomainResponse, error) {
	f.deleteRequests = append(f.deleteRequests, req)
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return wlmssdk.DeleteWlsDomainResponse{}, nil
}

func TestWlsDomainRuntimeConfigHasNoCreateOperation(t *testing.T) {
	t.Parallel()

	config := newWlsDomainRuntimeConfig(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}, &fakeWlsDomainOCIClient{})
	if config.Create != nil {
		t.Fatal("config.Create != nil, want manage-existing WlsDomain runtime without Create")
	}
	if len(config.Semantics.AuxiliaryOperations) != 0 {
		t.Fatalf("Semantics.AuxiliaryOperations = %#v, want none", config.Semantics.AuxiliaryOperations)
	}
	if len(config.List.Fields) != 6 {
		t.Fatalf("List fields = %#v, want focused bind fields plus pagination", config.List.Fields)
	}
}

func TestWlsDomainCreateOrUpdateRejectsMissingBindingIdentity(t *testing.T) {
	t.Parallel()

	resource := newWlsDomainResource()
	resource.Spec = wlmsv1beta1.WlsDomainSpec{}
	client := newTestWlsDomainClient(&fakeWlsDomainOCIClient{})

	response, err := client.CreateOrUpdate(context.Background(), resource, requestForResource(resource))
	if err == nil || !strings.Contains(err.Error(), "manage-existing flow requires spec.id or spec.compartmentId plus spec.displayName") {
		t.Fatalf("CreateOrUpdate() error = %v, want manage-existing bind validation", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful validation failure", response)
	}
}

func TestWlsDomainCreateOrUpdateBindsExistingDomainByPagedList(t *testing.T) {
	t.Parallel()

	resource := newWlsDomainResource()
	fake := &fakeWlsDomainOCIClient{}
	getCalls := 0
	fake.listFn = func(_ context.Context, req wlmssdk.ListWlsDomainsRequest) (wlmssdk.ListWlsDomainsResponse, error) {
		requireStringPtr(t, "ListWlsDomainsRequest.CompartmentId", req.CompartmentId, testWlsDomainCompartment)
		requireStringPtr(t, "ListWlsDomainsRequest.DisplayName", req.DisplayName, testWlsDomainName)
		if got := string(req.MiddlewareType); got != resource.Spec.MiddlewareType {
			t.Fatalf("ListWlsDomainsRequest.MiddlewareType = %q, want %q", got, resource.Spec.MiddlewareType)
		}
		if got := string(req.WeblogicVersion); got != resource.Spec.WeblogicVersion {
			t.Fatalf("ListWlsDomainsRequest.WeblogicVersion = %q, want %q", got, resource.Spec.WeblogicVersion)
		}
		if req.LifecycleState != "" {
			t.Fatalf("ListWlsDomainsRequest.LifecycleState = %q, want empty", req.LifecycleState)
		}
		if req.PatchReadinessStatus != "" {
			t.Fatalf("ListWlsDomainsRequest.PatchReadinessStatus = %q, want empty", req.PatchReadinessStatus)
		}
		switch page := stringPtrValue(req.Page); page {
		case "":
			return wlmssdk.ListWlsDomainsResponse{
				WlsDomainCollection: wlmssdk.WlsDomainCollection{
					Items: []wlmssdk.WlsDomainSummary{
						wlsDomainSummary("ocid1.wlsdomain.oc1..other", "other"),
					},
				},
				OpcNextPage: common.String("page-2"),
			}, nil
		case "page-2":
			return wlmssdk.ListWlsDomainsResponse{
				WlsDomainCollection: wlmssdk.WlsDomainCollection{
					Items: []wlmssdk.WlsDomainSummary{
						wlsDomainSummary(testWlsDomainID, testWlsDomainName),
					},
				},
			}, nil
		default:
			t.Fatalf("unexpected list page %q", page)
			return wlmssdk.ListWlsDomainsResponse{}, nil
		}
	}
	fake.getFn = func(_ context.Context, req wlmssdk.GetWlsDomainRequest) (wlmssdk.GetWlsDomainResponse, error) {
		getCalls++
		requireStringPtr(t, "GetWlsDomainRequest.WlsDomainId", req.WlsDomainId, testWlsDomainID)
		return wlmssdk.GetWlsDomainResponse{
			WlsDomain: activeWlsDomainSDK(testWlsDomainID),
		}, nil
	}

	response, err := newTestWlsDomainClient(fake).CreateOrUpdate(context.Background(), resource, requestForResource(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want active bind without requeue", response)
	}
	if len(fake.listRequests) != 2 {
		t.Fatalf("List calls = %d, want 2 paged lookups", len(fake.listRequests))
	}
	if getCalls != 1 {
		t.Fatalf("Get calls = %d, want 1 live read after list bind", getCalls)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("Update calls = %d, want no update on matching bind", len(fake.updateRequests))
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testWlsDomainID {
		t.Fatalf("status.ocid = %q, want %q", got, testWlsDomainID)
	}
}

func TestWlsDomainCreateOrUpdateUpdatesConfigurationAndTags(t *testing.T) {
	t.Parallel()

	resource := trackedWlsDomainResource()
	resource.Spec.Configuration = map[string]shared.JSONValue{
		"isPatchEnabled":             jsonValue("false"),
		"serversShutdownTimeout":     jsonValue("0"),
		"adminServerStartScriptPath": jsonValue(`""`),
	}
	resource.Spec.FreeformTags = map[string]string{}
	resource.Spec.DefinedTags = map[string]shared.MapValue{}

	fake := &fakeWlsDomainOCIClient{}
	updateApplied := false
	fake.getFn = func(_ context.Context, req wlmssdk.GetWlsDomainRequest) (wlmssdk.GetWlsDomainResponse, error) {
		requireStringPtr(t, "GetWlsDomainRequest.WlsDomainId", req.WlsDomainId, testWlsDomainID)
		if updateApplied {
			return wlmssdk.GetWlsDomainResponse{
				WlsDomain: activeWlsDomainWithConfigSDK(
					testWlsDomainID,
					false,
					0,
					"",
					map[string]string{},
					map[string]map[string]interface{}{},
				),
			}, nil
		}
		return wlmssdk.GetWlsDomainResponse{
			WlsDomain: activeWlsDomainWithConfigSDK(
				testWlsDomainID,
				true,
				30,
				"/old/start.sh",
				map[string]string{"env": "prod"},
				map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
			),
		}, nil
	}
	fake.updateFn = func(_ context.Context, req wlmssdk.UpdateWlsDomainRequest) (wlmssdk.UpdateWlsDomainResponse, error) {
		requireStringPtr(t, "UpdateWlsDomainRequest.WlsDomainId", req.WlsDomainId, testWlsDomainID)
		if req.Configuration == nil {
			t.Fatal("UpdateWlsDomainRequest.Configuration = nil, want explicit configuration update")
		}
		if req.Configuration.IsPatchEnabled == nil || *req.Configuration.IsPatchEnabled {
			t.Fatalf("Configuration.IsPatchEnabled = %#v, want false", req.Configuration.IsPatchEnabled)
		}
		if req.Configuration.ServersShutdownTimeout == nil || *req.Configuration.ServersShutdownTimeout != 0 {
			t.Fatalf("Configuration.ServersShutdownTimeout = %#v, want 0", req.Configuration.ServersShutdownTimeout)
		}
		if req.Configuration.AdminServerStartScriptPath == nil || *req.Configuration.AdminServerStartScriptPath != "" {
			t.Fatalf("Configuration.AdminServerStartScriptPath = %#v, want empty string", req.Configuration.AdminServerStartScriptPath)
		}
		if req.FreeformTags == nil || len(req.FreeformTags) != 0 {
			t.Fatalf("FreeformTags = %#v, want explicit empty map", req.FreeformTags)
		}
		if req.DefinedTags == nil || len(req.DefinedTags) != 0 {
			t.Fatalf("DefinedTags = %#v, want explicit empty map", req.DefinedTags)
		}
		updateApplied = true
		return wlmssdk.UpdateWlsDomainResponse{
			WlsDomain:    activeWlsDomainSDK(testWlsDomainID),
			OpcRequestId: common.String("opc-update"),
		}, nil
	}

	response, err := newTestWlsDomainClient(fake).CreateOrUpdate(context.Background(), resource, requestForResource(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful in-place update", response)
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("Update calls = %d, want 1", len(fake.updateRequests))
	}
	if resource.Status.Configuration.IsPatchEnabled {
		t.Fatal("status.configuration.isPatchEnabled = true, want false after update")
	}
	if resource.Status.Configuration.ServersShutdownTimeout != 0 {
		t.Fatalf("status.configuration.serversShutdownTimeout = %d, want 0", resource.Status.Configuration.ServersShutdownTimeout)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.opcRequestId = %q, want opc-update", got)
	}
}

func TestWlsDomainCreateOrUpdateRejectsForceNewDisplayNameDrift(t *testing.T) {
	t.Parallel()

	resource := trackedWlsDomainResource()
	resource.Spec.DisplayName = "renamed-domain"
	fake := &fakeWlsDomainOCIClient{}
	fake.getFn = func(_ context.Context, _ wlmssdk.GetWlsDomainRequest) (wlmssdk.GetWlsDomainResponse, error) {
		return wlmssdk.GetWlsDomainResponse{
			WlsDomain: activeWlsDomainSDK(testWlsDomainID),
		}, nil
	}
	fake.updateFn = func(context.Context, wlmssdk.UpdateWlsDomainRequest) (wlmssdk.UpdateWlsDomainResponse, error) {
		t.Fatal("UpdateWlsDomain should not be called after force-new drift")
		return wlmssdk.UpdateWlsDomainResponse{}, nil
	}

	response, err := newTestWlsDomainClient(fake).CreateOrUpdate(context.Background(), resource, requestForResource(resource))
	if err == nil || !strings.Contains(err.Error(), "require replacement when displayName changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want force-new displayName drift failure", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful drift rejection", response)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("Update calls = %d, want 0", len(fake.updateRequests))
	}
}

func TestWlsDomainDeleteWaitsForConfirmedDelete(t *testing.T) {
	t.Parallel()

	resource := trackedWlsDomainResource()
	getCalls := 0
	fake := &fakeWlsDomainOCIClient{}
	fake.getFn = func(_ context.Context, req wlmssdk.GetWlsDomainRequest) (wlmssdk.GetWlsDomainResponse, error) {
		getCalls++
		requireStringPtr(t, "GetWlsDomainRequest.WlsDomainId", req.WlsDomainId, testWlsDomainID)
		if getCalls == 1 {
			return wlmssdk.GetWlsDomainResponse{
				WlsDomain: activeWlsDomainSDK(testWlsDomainID),
			}, nil
		}
		domain := activeWlsDomainSDK(testWlsDomainID)
		domain.LifecycleState = wlmssdk.WlsDomainLifecycleStateDeleted
		return wlmssdk.GetWlsDomainResponse{
			WlsDomain: domain,
		}, nil
	}
	fake.deleteFn = func(_ context.Context, req wlmssdk.DeleteWlsDomainRequest) (wlmssdk.DeleteWlsDomainResponse, error) {
		requireStringPtr(t, "DeleteWlsDomainRequest.WlsDomainId", req.WlsDomainId, testWlsDomainID)
		return wlmssdk.DeleteWlsDomainResponse{
			OpcRequestId: common.String("opc-delete"),
		}, nil
	}

	deleted, err := newTestWlsDomainClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after confirmed DELETED readback")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("Delete calls = %d, want 1", len(fake.deleteRequests))
	}
	if getCalls != 2 {
		t.Fatalf("Get calls = %d, want 2 (pre-delete observe + confirm-delete)", getCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete confirmation timestamp")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.opcRequestId = %q, want opc-delete", got)
	}
}

func newTestWlsDomainClient(client wlsDomainOCIClient) WlsDomainServiceClient {
	return newWlsDomainServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
		client,
	)
}

func newWlsDomainResource() *wlmsv1beta1.WlsDomain {
	return &wlmsv1beta1.WlsDomain{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testWlsDomainName,
			Namespace: "default",
		},
		Spec: wlmsv1beta1.WlsDomainSpec{
			CompartmentId:   testWlsDomainCompartment,
			DisplayName:     testWlsDomainName,
			MiddlewareType:  "WLS",
			WeblogicVersion: "v14.1.2.0",
			FreeformTags:    map[string]string{"env": "prod"},
			DefinedTags:     map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func trackedWlsDomainResource() *wlmsv1beta1.WlsDomain {
	resource := newWlsDomainResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testWlsDomainID)
	resource.Status.Id = testWlsDomainID
	resource.Status.CompartmentId = testWlsDomainCompartment
	resource.Status.DisplayName = testWlsDomainName
	resource.Status.MiddlewareType = resource.Spec.MiddlewareType
	resource.Status.WeblogicVersion = resource.Spec.WeblogicVersion
	resource.Status.LifecycleState = string(wlmssdk.WlsDomainLifecycleStateActive)
	return resource
}

func requestForResource(resource *wlmsv1beta1.WlsDomain) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		},
	}
}

func activeWlsDomainSDK(id string) wlmssdk.WlsDomain {
	return activeWlsDomainWithConfigSDK(
		id,
		true,
		30,
		"/old/start.sh",
		map[string]string{"env": "prod"},
		map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	)
}

func activeWlsDomainWithConfigSDK(
	id string,
	isPatchEnabled bool,
	serversShutdownTimeout int,
	adminServerStartScriptPath string,
	freeformTags map[string]string,
	definedTags map[string]map[string]interface{},
) wlmssdk.WlsDomain {
	return wlmssdk.WlsDomain{
		Id:              common.String(id),
		DisplayName:     common.String(testWlsDomainName),
		CompartmentId:   common.String(testWlsDomainCompartment),
		LifecycleState:  wlmssdk.WlsDomainLifecycleStateActive,
		MiddlewareType:  common.String("WLS"),
		WeblogicVersion: common.String("v14.1.2.0"),
		Configuration: &wlmssdk.WlsDomainConfiguration{
			IsPatchEnabled:             common.Bool(isPatchEnabled),
			ServersShutdownTimeout:     common.Int(serversShutdownTimeout),
			AdminServerStartScriptPath: common.String(adminServerStartScriptPath),
		},
		FreeformTags: freeformTags,
		DefinedTags:  definedTags,
	}
}

func wlsDomainSummary(id string, displayName string) wlmssdk.WlsDomainSummary {
	return wlmssdk.WlsDomainSummary{
		Id:              common.String(id),
		DisplayName:     common.String(displayName),
		CompartmentId:   common.String(testWlsDomainCompartment),
		LifecycleState:  wlmssdk.WlsDomainLifecycleStateActive,
		MiddlewareType:  common.String("WLS"),
		WeblogicVersion: common.String("v14.1.2.0"),
	}
}

func jsonValue(raw string) shared.JSONValue {
	return shared.JSONValue{Raw: []byte(raw)}
}

func requireStringPtr(t *testing.T, label string, actual *string, want string) {
	t.Helper()
	if actual == nil || strings.TrimSpace(*actual) != want {
		t.Fatalf("%s = %#v, want %q", label, actual, want)
	}
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

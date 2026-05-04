/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package monitoredresourcetype

import (
	"context"
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
	testMonitoredResourceTypeID            = "ocid1.stackmonitoringresourcetype.oc1..type"
	testMonitoredResourceTypeCompartmentID = "ocid1.compartment.oc1..mrt"
)

func TestMonitoredResourceTypeRuntimeHooksConfigured(t *testing.T) {
	hooks := newMonitoredResourceTypeDefaultRuntimeHooks(stackmonitoringsdk.StackMonitoringClient{})
	applyMonitoredResourceTypeRuntimeHooks(&hooks)

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

	resource := testMonitoredResourceType()
	resource.Spec.Metadata = stackmonitoringv1beta1.MonitoredResourceTypeMetadata{
		Format:             string(stackmonitoringsdk.ResourceTypeMetadataDetailsFormatSystemFormat),
		RequiredProperties: []string{"hostName"},
		UniquePropertySets: []stackmonitoringv1beta1.MonitoredResourceTypeMetadataUniquePropertySet{
			{Properties: []string{"hostName"}},
		},
	}
	body, err := hooks.BuildCreateBody(context.Background(), resource, "default")
	if err != nil {
		t.Fatalf("BuildCreateBody() error = %v", err)
	}
	details, ok := body.(stackmonitoringsdk.CreateMonitoredResourceTypeDetails)
	if !ok {
		t.Fatalf("BuildCreateBody() type = %T, want CreateMonitoredResourceTypeDetails", body)
	}
	metadata, ok := details.Metadata.(stackmonitoringsdk.SystemFormatResourceTypeMetadataDetails)
	if !ok {
		t.Fatalf("BuildCreateBody() metadata type = %T, want SystemFormatResourceTypeMetadataDetails", details.Metadata)
	}
	if len(metadata.UniquePropertySets) != 1 || len(metadata.UniquePropertySets[0].Properties) != 1 || metadata.UniquePropertySets[0].Properties[0] != "hostName" {
		t.Fatalf("BuildCreateBody() metadata uniquePropertySets = %#v, want hostName set", metadata.UniquePropertySets)
	}
}

func TestMonitoredResourceTypeCreateRecordsIdentityAndRequestID(t *testing.T) {
	resource := testMonitoredResourceType()
	created := sdkMonitoredResourceType(resource, testMonitoredResourceTypeID, stackmonitoringsdk.ResourceTypeLifecycleStateActive)
	fake := &fakeMonitoredResourceTypeOCIClient{
		createMonitoredResourceType: func(_ context.Context, request stackmonitoringsdk.CreateMonitoredResourceTypeRequest) (stackmonitoringsdk.CreateMonitoredResourceTypeResponse, error) {
			assertMonitoredResourceTypeCreateRequest(t, request, resource)
			return stackmonitoringsdk.CreateMonitoredResourceTypeResponse{
				MonitoredResourceType: created,
				OpcRequestId:          common.String("opc-create"),
			}, nil
		},
		getMonitoredResourceType: func(context.Context, stackmonitoringsdk.GetMonitoredResourceTypeRequest) (stackmonitoringsdk.GetMonitoredResourceTypeResponse, error) {
			return stackmonitoringsdk.GetMonitoredResourceTypeResponse{MonitoredResourceType: created}, nil
		},
	}

	response, err := newTestMonitoredResourceTypeClient(fake).CreateOrUpdate(context.Background(), resource, testMonitoredResourceTypeRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() IsSuccessful = false, want true")
	}
	assertMonitoredResourceTypeCallCount(t, "CreateMonitoredResourceType()", fake.createCalls, 1)
	assertMonitoredResourceTypeRecordedID(t, resource, testMonitoredResourceTypeID)
	assertMonitoredResourceTypeOpcRequestID(t, resource, "opc-create")
	if got := resource.Status.LifecycleState; got != string(stackmonitoringsdk.ResourceTypeLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
}

func TestMonitoredResourceTypeCreateOrUpdateBindsExistingFromLaterListPage(t *testing.T) {
	resource := testMonitoredResourceType()
	existing := sdkMonitoredResourceType(resource, testMonitoredResourceTypeID, stackmonitoringsdk.ResourceTypeLifecycleStateActive)
	var pages []string
	fake := &fakeMonitoredResourceTypeOCIClient{
		listMonitoredResourceTypes: func(_ context.Context, request stackmonitoringsdk.ListMonitoredResourceTypesRequest) (stackmonitoringsdk.ListMonitoredResourceTypesResponse, error) {
			pages = append(pages, stringValue(request.Page))
			if got := stringValue(request.CompartmentId); got != resource.Spec.CompartmentId {
				t.Fatalf("ListMonitoredResourceTypes() CompartmentId = %q, want %q", got, resource.Spec.CompartmentId)
			}
			if got := stringValue(request.Name); got != resource.Spec.Name {
				t.Fatalf("ListMonitoredResourceTypes() Name = %q, want %q", got, resource.Spec.Name)
			}
			if request.Page == nil {
				other := testMonitoredResourceType()
				other.Spec.Name = "other-type"
				return stackmonitoringsdk.ListMonitoredResourceTypesResponse{
					MonitoredResourceTypesCollection: stackmonitoringsdk.MonitoredResourceTypesCollection{
						Items: []stackmonitoringsdk.MonitoredResourceTypeSummary{
							sdkMonitoredResourceTypeSummary(other, "ocid1.stackmonitoringresourcetype.oc1..other", stackmonitoringsdk.ResourceTypeLifecycleStateActive),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			return stackmonitoringsdk.ListMonitoredResourceTypesResponse{
				MonitoredResourceTypesCollection: stackmonitoringsdk.MonitoredResourceTypesCollection{
					Items: []stackmonitoringsdk.MonitoredResourceTypeSummary{
						sdkMonitoredResourceTypeSummary(resource, testMonitoredResourceTypeID, stackmonitoringsdk.ResourceTypeLifecycleStateActive),
					},
				},
			}, nil
		},
		getMonitoredResourceType: func(_ context.Context, request stackmonitoringsdk.GetMonitoredResourceTypeRequest) (stackmonitoringsdk.GetMonitoredResourceTypeResponse, error) {
			if got := stringValue(request.MonitoredResourceTypeId); got != testMonitoredResourceTypeID {
				t.Fatalf("GetMonitoredResourceType() MonitoredResourceTypeId = %q, want %q", got, testMonitoredResourceTypeID)
			}
			return stackmonitoringsdk.GetMonitoredResourceTypeResponse{MonitoredResourceType: existing}, nil
		},
		createMonitoredResourceType: func(context.Context, stackmonitoringsdk.CreateMonitoredResourceTypeRequest) (stackmonitoringsdk.CreateMonitoredResourceTypeResponse, error) {
			t.Fatal("CreateMonitoredResourceType() called despite existing list match")
			return stackmonitoringsdk.CreateMonitoredResourceTypeResponse{}, nil
		},
	}

	response, err := newTestMonitoredResourceTypeClient(fake).CreateOrUpdate(context.Background(), resource, testMonitoredResourceTypeRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if got := strings.Join(pages, ","); got != ",page-2" {
		t.Fatalf("ListMonitoredResourceTypes() pages = %q, want \",page-2\"", got)
	}
	assertMonitoredResourceTypeRecordedID(t, resource, testMonitoredResourceTypeID)
	assertMonitoredResourceTypeCallCount(t, "CreateMonitoredResourceType()", fake.createCalls, 0)
}

func TestMonitoredResourceTypeCreateOrUpdateNoopsWhenObservedStateMatches(t *testing.T) {
	resource := testMonitoredResourceType()
	resource.Status.OsokStatus.Ocid = shared.OCID(testMonitoredResourceTypeID)
	current := sdkMonitoredResourceType(resource, testMonitoredResourceTypeID, stackmonitoringsdk.ResourceTypeLifecycleStateActive)
	fake := &fakeMonitoredResourceTypeOCIClient{
		getMonitoredResourceType: func(context.Context, stackmonitoringsdk.GetMonitoredResourceTypeRequest) (stackmonitoringsdk.GetMonitoredResourceTypeResponse, error) {
			return stackmonitoringsdk.GetMonitoredResourceTypeResponse{MonitoredResourceType: current}, nil
		},
		updateMonitoredResourceType: func(context.Context, stackmonitoringsdk.UpdateMonitoredResourceTypeRequest) (stackmonitoringsdk.UpdateMonitoredResourceTypeResponse, error) {
			t.Fatal("UpdateMonitoredResourceType() called during no-op reconcile")
			return stackmonitoringsdk.UpdateMonitoredResourceTypeResponse{}, nil
		},
	}

	response, err := newTestMonitoredResourceTypeClient(fake).CreateOrUpdate(context.Background(), resource, testMonitoredResourceTypeRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() IsSuccessful = false, want true")
	}
}

func TestMonitoredResourceTypeMutableUpdateUsesUpdatePath(t *testing.T) {
	resource := testMonitoredResourceType()
	resource.Status.OsokStatus.Ocid = shared.OCID(testMonitoredResourceTypeID)
	currentResource := testMonitoredResourceType()
	currentResource.Spec.DisplayName = "old-display"
	current := sdkMonitoredResourceType(currentResource, testMonitoredResourceTypeID, stackmonitoringsdk.ResourceTypeLifecycleStateActive)
	updated := sdkMonitoredResourceType(resource, testMonitoredResourceTypeID, stackmonitoringsdk.ResourceTypeLifecycleStateActive)
	getResponses := []stackmonitoringsdk.GetMonitoredResourceTypeResponse{
		{MonitoredResourceType: current},
		{MonitoredResourceType: updated},
	}
	fake := &fakeMonitoredResourceTypeOCIClient{
		getMonitoredResourceType: getMonitoredResourceTypeResponses(t, &getResponses),
		updateMonitoredResourceType: func(_ context.Context, request stackmonitoringsdk.UpdateMonitoredResourceTypeRequest) (stackmonitoringsdk.UpdateMonitoredResourceTypeResponse, error) {
			assertMonitoredResourceTypeUpdateRequest(t, request, resource)
			return stackmonitoringsdk.UpdateMonitoredResourceTypeResponse{
				MonitoredResourceType: updated,
				OpcRequestId:          common.String("opc-update"),
			}, nil
		},
	}

	response, err := newTestMonitoredResourceTypeClient(fake).CreateOrUpdate(context.Background(), resource, testMonitoredResourceTypeRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() IsSuccessful = false, want true")
	}
	assertMonitoredResourceTypeCallCount(t, "UpdateMonitoredResourceType()", fake.updateCalls, 1)
	assertMonitoredResourceTypeOpcRequestID(t, resource, "opc-update")
}

func TestMonitoredResourceTypeImmutableDriftRejectedBeforeUpdate(t *testing.T) {
	resource := testMonitoredResourceType()
	resource.Status.OsokStatus.Ocid = shared.OCID(testMonitoredResourceTypeID)
	currentResource := testMonitoredResourceType()
	currentResource.Spec.Name = "renamed-in-oci"
	current := sdkMonitoredResourceType(currentResource, testMonitoredResourceTypeID, stackmonitoringsdk.ResourceTypeLifecycleStateActive)
	fake := &fakeMonitoredResourceTypeOCIClient{
		getMonitoredResourceType: func(context.Context, stackmonitoringsdk.GetMonitoredResourceTypeRequest) (stackmonitoringsdk.GetMonitoredResourceTypeResponse, error) {
			return stackmonitoringsdk.GetMonitoredResourceTypeResponse{MonitoredResourceType: current}, nil
		},
		updateMonitoredResourceType: func(context.Context, stackmonitoringsdk.UpdateMonitoredResourceTypeRequest) (stackmonitoringsdk.UpdateMonitoredResourceTypeResponse, error) {
			t.Fatal("UpdateMonitoredResourceType() called despite name drift")
			return stackmonitoringsdk.UpdateMonitoredResourceTypeResponse{}, nil
		},
	}

	_, err := newTestMonitoredResourceTypeClient(fake).CreateOrUpdate(context.Background(), resource, testMonitoredResourceTypeRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want immutable drift rejection")
	}
	if !strings.Contains(err.Error(), "name") {
		t.Fatalf("CreateOrUpdate() error = %v, want name drift detail", err)
	}
}

func TestMonitoredResourceTypeDeleteRetainsFinalizerWhileDeleteIsPending(t *testing.T) {
	resource := testMonitoredResourceType()
	resource.Status.OsokStatus.Ocid = shared.OCID(testMonitoredResourceTypeID)
	active := sdkMonitoredResourceType(resource, testMonitoredResourceTypeID, stackmonitoringsdk.ResourceTypeLifecycleStateActive)
	deleting := sdkMonitoredResourceType(resource, testMonitoredResourceTypeID, stackmonitoringsdk.ResourceTypeLifecycleStateDeleting)
	getResponses := []stackmonitoringsdk.GetMonitoredResourceTypeResponse{
		{MonitoredResourceType: active},
		{MonitoredResourceType: active},
		{MonitoredResourceType: deleting},
	}
	fake := &fakeMonitoredResourceTypeOCIClient{
		getMonitoredResourceType: getMonitoredResourceTypeResponses(t, &getResponses),
		deleteMonitoredResourceType: func(_ context.Context, request stackmonitoringsdk.DeleteMonitoredResourceTypeRequest) (stackmonitoringsdk.DeleteMonitoredResourceTypeResponse, error) {
			if got := stringValue(request.MonitoredResourceTypeId); got != testMonitoredResourceTypeID {
				t.Fatalf("DeleteMonitoredResourceType() MonitoredResourceTypeId = %q, want %q", got, testMonitoredResourceTypeID)
			}
			return stackmonitoringsdk.DeleteMonitoredResourceTypeResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}

	deleted, err := newTestMonitoredResourceTypeClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while OCI lifecycle is DELETING")
	}
	assertMonitoredResourceTypeCallCount(t, "DeleteMonitoredResourceType()", fake.deleteCalls, 1)
	assertMonitoredResourceTypeOpcRequestID(t, resource, "opc-delete")
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.status.async.current = nil, want pending delete lifecycle tracker")
	}
}

func TestMonitoredResourceTypeDeleteConfirmsTerminalLifecycle(t *testing.T) {
	resource := testMonitoredResourceType()
	resource.Status.OsokStatus.Ocid = shared.OCID(testMonitoredResourceTypeID)
	active := sdkMonitoredResourceType(resource, testMonitoredResourceTypeID, stackmonitoringsdk.ResourceTypeLifecycleStateActive)
	deletedResource := sdkMonitoredResourceType(resource, testMonitoredResourceTypeID, stackmonitoringsdk.ResourceTypeLifecycleStateDeleted)
	getResponses := []stackmonitoringsdk.GetMonitoredResourceTypeResponse{
		{MonitoredResourceType: active},
		{MonitoredResourceType: active},
		{MonitoredResourceType: deletedResource},
	}
	fake := &fakeMonitoredResourceTypeOCIClient{
		getMonitoredResourceType: getMonitoredResourceTypeResponses(t, &getResponses),
		deleteMonitoredResourceType: func(context.Context, stackmonitoringsdk.DeleteMonitoredResourceTypeRequest) (stackmonitoringsdk.DeleteMonitoredResourceTypeResponse, error) {
			return stackmonitoringsdk.DeleteMonitoredResourceTypeResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}

	deleted, err := newTestMonitoredResourceTypeClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after terminal DELETED lifecycle")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want terminal delete timestamp")
	}
	assertMonitoredResourceTypeOpcRequestID(t, resource, "opc-delete")
}

func TestMonitoredResourceTypeDeleteRejectsAuthShapedPreDeleteRead(t *testing.T) {
	resource := testMonitoredResourceType()
	resource.Status.OsokStatus.Ocid = shared.OCID(testMonitoredResourceTypeID)
	authNotFound := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	authNotFound.OpcRequestID = "opc-delete-auth"
	fake := &fakeMonitoredResourceTypeOCIClient{
		getMonitoredResourceType: func(context.Context, stackmonitoringsdk.GetMonitoredResourceTypeRequest) (stackmonitoringsdk.GetMonitoredResourceTypeResponse, error) {
			return stackmonitoringsdk.GetMonitoredResourceTypeResponse{}, authNotFound
		},
		deleteMonitoredResourceType: func(context.Context, stackmonitoringsdk.DeleteMonitoredResourceTypeRequest) (stackmonitoringsdk.DeleteMonitoredResourceTypeResponse, error) {
			t.Fatal("DeleteMonitoredResourceType() called after auth-shaped pre-delete read")
			return stackmonitoringsdk.DeleteMonitoredResourceTypeResponse{}, nil
		},
	}

	deleted, err := newTestMonitoredResourceTypeClient(fake).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous auth-shaped not-found rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous 404 NotAuthorizedOrNotFound", err)
	}
	assertMonitoredResourceTypeOpcRequestID(t, resource, "opc-delete-auth")
	assertMonitoredResourceTypeCallCount(t, "DeleteMonitoredResourceType()", fake.deleteCalls, 0)
}

func TestMonitoredResourceTypeCreateOrUpdateRecordsOpcRequestIDFromOCIError(t *testing.T) {
	resource := testMonitoredResourceType()
	createErr := errortest.NewServiceError(500, errorutil.InternalServerError, "create failed")
	createErr.OpcRequestID = "opc-create-error"
	fake := &fakeMonitoredResourceTypeOCIClient{
		createMonitoredResourceType: func(context.Context, stackmonitoringsdk.CreateMonitoredResourceTypeRequest) (stackmonitoringsdk.CreateMonitoredResourceTypeResponse, error) {
			return stackmonitoringsdk.CreateMonitoredResourceTypeResponse{}, createErr
		},
	}

	_, err := newTestMonitoredResourceTypeClient(fake).CreateOrUpdate(context.Background(), resource, testMonitoredResourceTypeRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI create error")
	}
	assertMonitoredResourceTypeOpcRequestID(t, resource, "opc-create-error")
}

func newTestMonitoredResourceTypeClient(fake *fakeMonitoredResourceTypeOCIClient) MonitoredResourceTypeServiceClient {
	hooks := newMonitoredResourceTypeDefaultRuntimeHooks(stackmonitoringsdk.StackMonitoringClient{})
	hooks.Create.Call = fake.CreateMonitoredResourceType
	hooks.Get.Call = fake.GetMonitoredResourceType
	hooks.List.Call = fake.ListMonitoredResourceTypes
	hooks.Update.Call = fake.UpdateMonitoredResourceType
	hooks.Delete.Call = fake.DeleteMonitoredResourceType
	applyMonitoredResourceTypeRuntimeHooks(&hooks)
	manager := &MonitoredResourceTypeServiceManager{Log: loggerutil.OSOKLogger{Logger: logr.Discard()}}
	delegate := defaultMonitoredResourceTypeServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*stackmonitoringv1beta1.MonitoredResourceType](
			buildMonitoredResourceTypeGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapMonitoredResourceTypeGeneratedClient(hooks, delegate)
}

type fakeMonitoredResourceTypeOCIClient struct {
	createMonitoredResourceType func(context.Context, stackmonitoringsdk.CreateMonitoredResourceTypeRequest) (stackmonitoringsdk.CreateMonitoredResourceTypeResponse, error)
	getMonitoredResourceType    func(context.Context, stackmonitoringsdk.GetMonitoredResourceTypeRequest) (stackmonitoringsdk.GetMonitoredResourceTypeResponse, error)
	listMonitoredResourceTypes  func(context.Context, stackmonitoringsdk.ListMonitoredResourceTypesRequest) (stackmonitoringsdk.ListMonitoredResourceTypesResponse, error)
	updateMonitoredResourceType func(context.Context, stackmonitoringsdk.UpdateMonitoredResourceTypeRequest) (stackmonitoringsdk.UpdateMonitoredResourceTypeResponse, error)
	deleteMonitoredResourceType func(context.Context, stackmonitoringsdk.DeleteMonitoredResourceTypeRequest) (stackmonitoringsdk.DeleteMonitoredResourceTypeResponse, error)

	createCalls int
	getCalls    int
	listCalls   int
	updateCalls int
	deleteCalls int
}

func (f *fakeMonitoredResourceTypeOCIClient) CreateMonitoredResourceType(
	ctx context.Context,
	request stackmonitoringsdk.CreateMonitoredResourceTypeRequest,
) (stackmonitoringsdk.CreateMonitoredResourceTypeResponse, error) {
	f.createCalls++
	if f.createMonitoredResourceType == nil {
		return stackmonitoringsdk.CreateMonitoredResourceTypeResponse{}, nil
	}
	return f.createMonitoredResourceType(ctx, request)
}

func (f *fakeMonitoredResourceTypeOCIClient) GetMonitoredResourceType(
	ctx context.Context,
	request stackmonitoringsdk.GetMonitoredResourceTypeRequest,
) (stackmonitoringsdk.GetMonitoredResourceTypeResponse, error) {
	f.getCalls++
	if f.getMonitoredResourceType == nil {
		return stackmonitoringsdk.GetMonitoredResourceTypeResponse{}, nil
	}
	return f.getMonitoredResourceType(ctx, request)
}

func (f *fakeMonitoredResourceTypeOCIClient) ListMonitoredResourceTypes(
	ctx context.Context,
	request stackmonitoringsdk.ListMonitoredResourceTypesRequest,
) (stackmonitoringsdk.ListMonitoredResourceTypesResponse, error) {
	f.listCalls++
	if f.listMonitoredResourceTypes == nil {
		return stackmonitoringsdk.ListMonitoredResourceTypesResponse{}, nil
	}
	return f.listMonitoredResourceTypes(ctx, request)
}

func (f *fakeMonitoredResourceTypeOCIClient) UpdateMonitoredResourceType(
	ctx context.Context,
	request stackmonitoringsdk.UpdateMonitoredResourceTypeRequest,
) (stackmonitoringsdk.UpdateMonitoredResourceTypeResponse, error) {
	f.updateCalls++
	if f.updateMonitoredResourceType == nil {
		return stackmonitoringsdk.UpdateMonitoredResourceTypeResponse{}, nil
	}
	return f.updateMonitoredResourceType(ctx, request)
}

func (f *fakeMonitoredResourceTypeOCIClient) DeleteMonitoredResourceType(
	ctx context.Context,
	request stackmonitoringsdk.DeleteMonitoredResourceTypeRequest,
) (stackmonitoringsdk.DeleteMonitoredResourceTypeResponse, error) {
	f.deleteCalls++
	if f.deleteMonitoredResourceType == nil {
		return stackmonitoringsdk.DeleteMonitoredResourceTypeResponse{}, nil
	}
	return f.deleteMonitoredResourceType(ctx, request)
}

func getMonitoredResourceTypeResponses(
	t *testing.T,
	responses *[]stackmonitoringsdk.GetMonitoredResourceTypeResponse,
) func(context.Context, stackmonitoringsdk.GetMonitoredResourceTypeRequest) (stackmonitoringsdk.GetMonitoredResourceTypeResponse, error) {
	t.Helper()

	return func(context.Context, stackmonitoringsdk.GetMonitoredResourceTypeRequest) (stackmonitoringsdk.GetMonitoredResourceTypeResponse, error) {
		if len(*responses) == 0 {
			t.Fatal("GetMonitoredResourceType() called more times than expected")
		}
		response := (*responses)[0]
		*responses = (*responses)[1:]
		return response, nil
	}
}

func testMonitoredResourceType() *stackmonitoringv1beta1.MonitoredResourceType {
	return &stackmonitoringv1beta1.MonitoredResourceType{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sample-resource-type",
			Namespace: "default",
			UID:       types.UID("sample-resource-type-uid"),
		},
		Spec: stackmonitoringv1beta1.MonitoredResourceTypeSpec{
			Name:             "osok_sample_type",
			CompartmentId:    testMonitoredResourceTypeCompartmentID,
			DisplayName:      "OSOK sample type",
			Description:      "OSOK monitored resource type",
			MetricNamespace:  "osok_custom",
			SourceType:       string(stackmonitoringsdk.SourceTypeSmRepoOnly),
			ResourceCategory: string(stackmonitoringsdk.ResourceCategoryApplication),
			FreeformTags:     map[string]string{"owner": "osok"},
		},
	}
}

func testMonitoredResourceTypeRequest(resource *stackmonitoringv1beta1.MonitoredResourceType) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		},
	}
}

func sdkMonitoredResourceType(
	resource *stackmonitoringv1beta1.MonitoredResourceType,
	id string,
	lifecycle stackmonitoringsdk.ResourceTypeLifecycleStateEnum,
) stackmonitoringsdk.MonitoredResourceType {
	sourceType, _ := monitoredResourceTypeSourceType(resource.Spec.SourceType)
	resourceCategory, _ := monitoredResourceTypeResourceCategory(resource.Spec.ResourceCategory)
	metadata, _ := monitoredResourceTypeMetadata(resource.Spec.Metadata)
	return stackmonitoringsdk.MonitoredResourceType{
		Id:               common.String(id),
		Name:             common.String(resource.Spec.Name),
		CompartmentId:    common.String(resource.Spec.CompartmentId),
		DisplayName:      monitoredResourceTypeOptionalString(resource.Spec.DisplayName),
		Description:      monitoredResourceTypeOptionalString(resource.Spec.Description),
		MetricNamespace:  monitoredResourceTypeOptionalString(resource.Spec.MetricNamespace),
		LifecycleState:   lifecycle,
		SourceType:       sourceType,
		ResourceCategory: resourceCategory,
		Metadata:         metadata,
		FreeformTags:     cloneMonitoredResourceTypeStringMap(resource.Spec.FreeformTags),
		DefinedTags:      monitoredResourceTypeDefinedTags(resource.Spec.DefinedTags),
	}
}

func sdkMonitoredResourceTypeSummary(
	resource *stackmonitoringv1beta1.MonitoredResourceType,
	id string,
	lifecycle stackmonitoringsdk.ResourceTypeLifecycleStateEnum,
) stackmonitoringsdk.MonitoredResourceTypeSummary {
	sourceType, _ := monitoredResourceTypeSourceType(resource.Spec.SourceType)
	resourceCategory, _ := monitoredResourceTypeResourceCategory(resource.Spec.ResourceCategory)
	metadata, _ := monitoredResourceTypeMetadata(resource.Spec.Metadata)
	return stackmonitoringsdk.MonitoredResourceTypeSummary{
		Id:               common.String(id),
		Name:             common.String(resource.Spec.Name),
		CompartmentId:    common.String(resource.Spec.CompartmentId),
		DisplayName:      monitoredResourceTypeOptionalString(resource.Spec.DisplayName),
		Description:      monitoredResourceTypeOptionalString(resource.Spec.Description),
		MetricNamespace:  monitoredResourceTypeOptionalString(resource.Spec.MetricNamespace),
		LifecycleState:   lifecycle,
		SourceType:       sourceType,
		ResourceCategory: resourceCategory,
		Metadata:         metadata,
		FreeformTags:     cloneMonitoredResourceTypeStringMap(resource.Spec.FreeformTags),
		DefinedTags:      monitoredResourceTypeDefinedTags(resource.Spec.DefinedTags),
	}
}

func assertMonitoredResourceTypeCreateRequest(
	t *testing.T,
	request stackmonitoringsdk.CreateMonitoredResourceTypeRequest,
	resource *stackmonitoringv1beta1.MonitoredResourceType,
) {
	t.Helper()

	details := request.CreateMonitoredResourceTypeDetails
	if got := stringValue(details.Name); got != resource.Spec.Name {
		t.Fatalf("CreateMonitoredResourceType() name = %q, want %q", got, resource.Spec.Name)
	}
	if got := stringValue(details.CompartmentId); got != resource.Spec.CompartmentId {
		t.Fatalf("CreateMonitoredResourceType() compartmentId = %q, want %q", got, resource.Spec.CompartmentId)
	}
	if got := stringValue(details.DisplayName); got != resource.Spec.DisplayName {
		t.Fatalf("CreateMonitoredResourceType() displayName = %q, want %q", got, resource.Spec.DisplayName)
	}
	if got := string(details.SourceType); got != resource.Spec.SourceType {
		t.Fatalf("CreateMonitoredResourceType() sourceType = %q, want %q", got, resource.Spec.SourceType)
	}
	if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
		t.Fatal("CreateMonitoredResourceType() OpcRetryToken is empty, want deterministic retry token")
	}
}

func assertMonitoredResourceTypeUpdateRequest(
	t *testing.T,
	request stackmonitoringsdk.UpdateMonitoredResourceTypeRequest,
	resource *stackmonitoringv1beta1.MonitoredResourceType,
) {
	t.Helper()

	if got := stringValue(request.MonitoredResourceTypeId); got != testMonitoredResourceTypeID {
		t.Fatalf("UpdateMonitoredResourceType() MonitoredResourceTypeId = %q, want %q", got, testMonitoredResourceTypeID)
	}
	details := request.UpdateMonitoredResourceTypeDetails
	if got := stringValue(details.DisplayName); got != resource.Spec.DisplayName {
		t.Fatalf("UpdateMonitoredResourceType() displayName = %q, want %q", got, resource.Spec.DisplayName)
	}
	if got := stringValue(details.Description); got != resource.Spec.Description {
		t.Fatalf("UpdateMonitoredResourceType() description = %q, want %q", got, resource.Spec.Description)
	}
}

func assertMonitoredResourceTypeRecordedID(t *testing.T, resource *stackmonitoringv1beta1.MonitoredResourceType, want string) {
	t.Helper()

	if got := string(resource.Status.OsokStatus.Ocid); got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
	if got := resource.Status.Id; got != want {
		t.Fatalf("status.id = %q, want %q", got, want)
	}
}

func assertMonitoredResourceTypeOpcRequestID(t *testing.T, resource *stackmonitoringv1beta1.MonitoredResourceType, want string) {
	t.Helper()

	if got := resource.Status.OsokStatus.OpcRequestID; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func assertMonitoredResourceTypeCallCount(t *testing.T, operation string, got int, want int) {
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

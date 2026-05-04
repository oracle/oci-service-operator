/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package digitaltwinrelationship

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	iotsdk "github.com/oracle/oci-go-sdk/v65/iot"
	iotv1beta1 "github.com/oracle/oci-service-operator/api/iot/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testDigitalTwinRelationshipID       = "ocid1.digitaltwinrelationship.oc1..relationship"
	testDigitalTwinRelationshipDomainID = "ocid1.iotdomain.oc1..domain"
	testSourceDigitalTwinInstanceID     = "ocid1.digitaltwininstance.oc1..source"
	testTargetDigitalTwinInstanceID     = "ocid1.digitaltwininstance.oc1..target"
	testDigitalTwinRelationshipPath     = "contains"
	testDigitalTwinRelationshipName     = "source-contains-target"
)

type fakeDigitalTwinRelationshipOCIClient struct {
	createFn func(context.Context, iotsdk.CreateDigitalTwinRelationshipRequest) (iotsdk.CreateDigitalTwinRelationshipResponse, error)
	getFn    func(context.Context, iotsdk.GetDigitalTwinRelationshipRequest) (iotsdk.GetDigitalTwinRelationshipResponse, error)
	listFn   func(context.Context, iotsdk.ListDigitalTwinRelationshipsRequest) (iotsdk.ListDigitalTwinRelationshipsResponse, error)
	updateFn func(context.Context, iotsdk.UpdateDigitalTwinRelationshipRequest) (iotsdk.UpdateDigitalTwinRelationshipResponse, error)
	deleteFn func(context.Context, iotsdk.DeleteDigitalTwinRelationshipRequest) (iotsdk.DeleteDigitalTwinRelationshipResponse, error)
}

type pagedDigitalTwinRelationshipList struct {
	t        *testing.T
	resource *iotv1beta1.DigitalTwinRelationship
	calls    int
}

func (f *fakeDigitalTwinRelationshipOCIClient) CreateDigitalTwinRelationship(ctx context.Context, req iotsdk.CreateDigitalTwinRelationshipRequest) (iotsdk.CreateDigitalTwinRelationshipResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return iotsdk.CreateDigitalTwinRelationshipResponse{}, nil
}

func (f *fakeDigitalTwinRelationshipOCIClient) GetDigitalTwinRelationship(ctx context.Context, req iotsdk.GetDigitalTwinRelationshipRequest) (iotsdk.GetDigitalTwinRelationshipResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return iotsdk.GetDigitalTwinRelationshipResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "digital twin relationship is missing")
}

func (f *fakeDigitalTwinRelationshipOCIClient) ListDigitalTwinRelationships(ctx context.Context, req iotsdk.ListDigitalTwinRelationshipsRequest) (iotsdk.ListDigitalTwinRelationshipsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return iotsdk.ListDigitalTwinRelationshipsResponse{}, nil
}

func (f *fakeDigitalTwinRelationshipOCIClient) UpdateDigitalTwinRelationship(ctx context.Context, req iotsdk.UpdateDigitalTwinRelationshipRequest) (iotsdk.UpdateDigitalTwinRelationshipResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return iotsdk.UpdateDigitalTwinRelationshipResponse{}, nil
}

func (f *fakeDigitalTwinRelationshipOCIClient) DeleteDigitalTwinRelationship(ctx context.Context, req iotsdk.DeleteDigitalTwinRelationshipRequest) (iotsdk.DeleteDigitalTwinRelationshipResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return iotsdk.DeleteDigitalTwinRelationshipResponse{}, nil
}

func (f *pagedDigitalTwinRelationshipList) list(
	_ context.Context,
	req iotsdk.ListDigitalTwinRelationshipsRequest,
) (iotsdk.ListDigitalTwinRelationshipsResponse, error) {
	f.calls++
	requireStringPtr(f.t, "ListDigitalTwinRelationshipsRequest.IotDomainId", req.IotDomainId, f.resource.Spec.IotDomainId)
	requireStringPtr(f.t, "ListDigitalTwinRelationshipsRequest.ContentPath", req.ContentPath, f.resource.Spec.ContentPath)
	requireStringPtr(f.t, "ListDigitalTwinRelationshipsRequest.SourceDigitalTwinInstanceId", req.SourceDigitalTwinInstanceId, f.resource.Spec.SourceDigitalTwinInstanceId)
	requireStringPtr(f.t, "ListDigitalTwinRelationshipsRequest.TargetDigitalTwinInstanceId", req.TargetDigitalTwinInstanceId, f.resource.Spec.TargetDigitalTwinInstanceId)
	if f.calls == 1 {
		return f.firstPage(req), nil
	}
	return f.secondPage(req), nil
}

func (f *pagedDigitalTwinRelationshipList) firstPage(
	req iotsdk.ListDigitalTwinRelationshipsRequest,
) iotsdk.ListDigitalTwinRelationshipsResponse {
	if req.Page != nil {
		f.t.Fatalf("first ListDigitalTwinRelationshipsRequest.Page = %q, want nil", *req.Page)
	}
	other := makeSDKDigitalTwinRelationshipSummary(f.t, "ocid1.digitaltwinrelationship.oc1..other", f.resource.Spec, iotsdk.LifecycleStateActive)
	other.TargetDigitalTwinInstanceId = common.String("ocid1.digitaltwininstance.oc1..other")
	return iotsdk.ListDigitalTwinRelationshipsResponse{
		DigitalTwinRelationshipCollection: iotsdk.DigitalTwinRelationshipCollection{
			Items: []iotsdk.DigitalTwinRelationshipSummary{other},
		},
		OpcNextPage: common.String("page-2"),
	}
}

func (f *pagedDigitalTwinRelationshipList) secondPage(
	req iotsdk.ListDigitalTwinRelationshipsRequest,
) iotsdk.ListDigitalTwinRelationshipsResponse {
	requireStringPtr(f.t, "second ListDigitalTwinRelationshipsRequest.Page", req.Page, "page-2")
	return iotsdk.ListDigitalTwinRelationshipsResponse{
		DigitalTwinRelationshipCollection: iotsdk.DigitalTwinRelationshipCollection{
			Items: []iotsdk.DigitalTwinRelationshipSummary{
				makeSDKDigitalTwinRelationshipSummary(f.t, testDigitalTwinRelationshipID, f.resource.Spec, iotsdk.LifecycleStateActive),
			},
		},
	}
}

func newTestDigitalTwinRelationshipClient(client digitalTwinRelationshipOCIClient) DigitalTwinRelationshipServiceClient {
	return newDigitalTwinRelationshipServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
		client,
	)
}

func makeDigitalTwinRelationshipResource() *iotv1beta1.DigitalTwinRelationship {
	return &iotv1beta1.DigitalTwinRelationship{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDigitalTwinRelationshipName,
			Namespace: "default",
		},
		Spec: iotv1beta1.DigitalTwinRelationshipSpec{
			IotDomainId:                 testDigitalTwinRelationshipDomainID,
			ContentPath:                 testDigitalTwinRelationshipPath,
			SourceDigitalTwinInstanceId: testSourceDigitalTwinInstanceID,
			TargetDigitalTwinInstanceId: testTargetDigitalTwinInstanceID,
			DisplayName:                 testDigitalTwinRelationshipName,
			Description:                 "initial relationship",
			Content:                     testDigitalTwinRelationshipContent(),
			FreeformTags: map[string]string{
				"env": "test",
			},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func makeTrackedDigitalTwinRelationshipResource() *iotv1beta1.DigitalTwinRelationship {
	resource := makeDigitalTwinRelationshipResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testDigitalTwinRelationshipID)
	resource.Status.Id = testDigitalTwinRelationshipID
	resource.Status.IotDomainId = testDigitalTwinRelationshipDomainID
	resource.Status.ContentPath = testDigitalTwinRelationshipPath
	resource.Status.SourceDigitalTwinInstanceId = testSourceDigitalTwinInstanceID
	resource.Status.TargetDigitalTwinInstanceId = testTargetDigitalTwinInstanceID
	resource.Status.DisplayName = testDigitalTwinRelationshipName
	resource.Status.LifecycleState = string(iotsdk.LifecycleStateActive)
	resource.Status.Content = testDigitalTwinRelationshipContent()
	return resource
}

func makeDigitalTwinRelationshipRequest(resource *iotv1beta1.DigitalTwinRelationship) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		},
	}
}

func testDigitalTwinRelationshipContent() map[string]shared.JSONValue {
	return map[string]shared.JSONValue{
		"enabled":     jsonValue("true"),
		"temperature": jsonValue("72.5"),
		"units":       jsonValue(`"fahrenheit"`),
	}
}

func testDigitalTwinRelationshipContentObject(t *testing.T) map[string]interface{} {
	t.Helper()
	content, err := digitalTwinRelationshipContent(testDigitalTwinRelationshipContent())
	if err != nil {
		t.Fatalf("digitalTwinRelationshipContent() error = %v", err)
	}
	return content
}

func updatedDigitalTwinRelationshipContent() map[string]shared.JSONValue {
	return map[string]shared.JSONValue{
		"enabled":     jsonValue("false"),
		"temperature": jsonValue("68"),
		"units":       jsonValue(`"fahrenheit"`),
	}
}

func updatedDigitalTwinRelationshipContentObject(t *testing.T) map[string]interface{} {
	t.Helper()
	content, err := digitalTwinRelationshipContent(updatedDigitalTwinRelationshipContent())
	if err != nil {
		t.Fatalf("digitalTwinRelationshipContent() error = %v", err)
	}
	return content
}

func makeSDKDigitalTwinRelationship(
	t *testing.T,
	id string,
	spec iotv1beta1.DigitalTwinRelationshipSpec,
	state iotsdk.LifecycleStateEnum,
) iotsdk.DigitalTwinRelationship {
	t.Helper()
	content, err := digitalTwinRelationshipContent(spec.Content)
	if err != nil {
		t.Fatalf("digitalTwinRelationshipContent() error = %v", err)
	}
	return iotsdk.DigitalTwinRelationship{
		Id:                          common.String(id),
		IotDomainId:                 common.String(spec.IotDomainId),
		ContentPath:                 common.String(spec.ContentPath),
		SourceDigitalTwinInstanceId: common.String(spec.SourceDigitalTwinInstanceId),
		TargetDigitalTwinInstanceId: common.String(spec.TargetDigitalTwinInstanceId),
		DisplayName:                 common.String(spec.DisplayName),
		Description:                 common.String(spec.Description),
		Content:                     content,
		LifecycleState:              state,
		FreeformTags:                cloneDigitalTwinRelationshipStringMap(spec.FreeformTags),
		DefinedTags:                 digitalTwinRelationshipDefinedTags(spec.DefinedTags),
	}
}

func makeSDKDigitalTwinRelationshipSummary(
	t *testing.T,
	id string,
	spec iotv1beta1.DigitalTwinRelationshipSpec,
	state iotsdk.LifecycleStateEnum,
) iotsdk.DigitalTwinRelationshipSummary {
	t.Helper()
	return iotsdk.DigitalTwinRelationshipSummary{
		Id:                          common.String(id),
		IotDomainId:                 common.String(spec.IotDomainId),
		ContentPath:                 common.String(spec.ContentPath),
		SourceDigitalTwinInstanceId: common.String(spec.SourceDigitalTwinInstanceId),
		TargetDigitalTwinInstanceId: common.String(spec.TargetDigitalTwinInstanceId),
		DisplayName:                 common.String(spec.DisplayName),
		Description:                 common.String(spec.Description),
		LifecycleState:              state,
		FreeformTags:                cloneDigitalTwinRelationshipStringMap(spec.FreeformTags),
		DefinedTags:                 digitalTwinRelationshipDefinedTags(spec.DefinedTags),
	}
}

func TestDigitalTwinRelationshipCreateOrUpdateBindsExistingRelationshipByPagedIdentityList(t *testing.T) {
	t.Parallel()

	resource := makeDigitalTwinRelationshipResource()
	createCalled := false
	updateCalled := false
	listPages := &pagedDigitalTwinRelationshipList{t: t, resource: resource}
	getCalls := 0

	client := newTestDigitalTwinRelationshipClient(&fakeDigitalTwinRelationshipOCIClient{
		listFn: listPages.list,
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinRelationshipRequest) (iotsdk.GetDigitalTwinRelationshipResponse, error) {
			getCalls++
			requireStringPtr(t, "GetDigitalTwinRelationshipRequest.DigitalTwinRelationshipId", req.DigitalTwinRelationshipId, testDigitalTwinRelationshipID)
			return iotsdk.GetDigitalTwinRelationshipResponse{
				DigitalTwinRelationship: makeSDKDigitalTwinRelationship(t, testDigitalTwinRelationshipID, resource.Spec, iotsdk.LifecycleStateActive),
			}, nil
		},
		createFn: func(context.Context, iotsdk.CreateDigitalTwinRelationshipRequest) (iotsdk.CreateDigitalTwinRelationshipResponse, error) {
			createCalled = true
			return iotsdk.CreateDigitalTwinRelationshipResponse{}, nil
		},
		updateFn: func(context.Context, iotsdk.UpdateDigitalTwinRelationshipRequest) (iotsdk.UpdateDigitalTwinRelationshipResponse, error) {
			updateCalled = true
			return iotsdk.UpdateDigitalTwinRelationshipResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeDigitalTwinRelationshipRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if createCalled {
		t.Fatal("CreateDigitalTwinRelationship() called, want bind without create")
	}
	if updateCalled {
		t.Fatal("UpdateDigitalTwinRelationship() called, want no update for matching readback")
	}
	if listPages.calls != 2 {
		t.Fatalf("ListDigitalTwinRelationships() calls = %d, want 2 paged calls", listPages.calls)
	}
	if getCalls != 1 {
		t.Fatalf("GetDigitalTwinRelationship() calls = %d, want 1 follow-up get", getCalls)
	}
	requireDigitalTwinRelationshipBoundStatus(t, resource)
	requireLastCondition(t, resource, shared.Active)
}

func TestDigitalTwinRelationshipCreateRecordsRetryTokenRequestIDAndStatus(t *testing.T) {
	t.Parallel()

	resource := makeDigitalTwinRelationshipResource()
	listCalls := 0
	createCalls := 0
	getCalls := 0

	client := newTestDigitalTwinRelationshipClient(&fakeDigitalTwinRelationshipOCIClient{
		listFn: func(context.Context, iotsdk.ListDigitalTwinRelationshipsRequest) (iotsdk.ListDigitalTwinRelationshipsResponse, error) {
			listCalls++
			return iotsdk.ListDigitalTwinRelationshipsResponse{}, nil
		},
		createFn: func(_ context.Context, req iotsdk.CreateDigitalTwinRelationshipRequest) (iotsdk.CreateDigitalTwinRelationshipResponse, error) {
			createCalls++
			requireDigitalTwinRelationshipCreateRequest(t, req, resource)
			return iotsdk.CreateDigitalTwinRelationshipResponse{
				DigitalTwinRelationship: makeSDKDigitalTwinRelationship(t, testDigitalTwinRelationshipID, resource.Spec, iotsdk.LifecycleStateActive),
				OpcRequestId:            common.String("opc-create"),
			}, nil
		},
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinRelationshipRequest) (iotsdk.GetDigitalTwinRelationshipResponse, error) {
			getCalls++
			requireStringPtr(t, "GetDigitalTwinRelationshipRequest.DigitalTwinRelationshipId", req.DigitalTwinRelationshipId, testDigitalTwinRelationshipID)
			return iotsdk.GetDigitalTwinRelationshipResponse{
				DigitalTwinRelationship: makeSDKDigitalTwinRelationship(t, testDigitalTwinRelationshipID, resource.Spec, iotsdk.LifecycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeDigitalTwinRelationshipRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if listCalls != 1 {
		t.Fatalf("ListDigitalTwinRelationships() calls = %d, want 1 pre-create lookup", listCalls)
	}
	if createCalls != 1 {
		t.Fatalf("CreateDigitalTwinRelationship() calls = %d, want 1", createCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetDigitalTwinRelationship() calls = %d, want 1 create follow-up", getCalls)
	}
	requireDigitalTwinRelationshipBoundStatus(t, resource)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", got)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestDigitalTwinRelationshipCreateOrUpdateSkipsNoopUpdate(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDigitalTwinRelationshipResource()
	getCalls := 0

	client := newTestDigitalTwinRelationshipClient(&fakeDigitalTwinRelationshipOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinRelationshipRequest) (iotsdk.GetDigitalTwinRelationshipResponse, error) {
			getCalls++
			requireStringPtr(t, "GetDigitalTwinRelationshipRequest.DigitalTwinRelationshipId", req.DigitalTwinRelationshipId, testDigitalTwinRelationshipID)
			return iotsdk.GetDigitalTwinRelationshipResponse{
				DigitalTwinRelationship: makeSDKDigitalTwinRelationship(t, testDigitalTwinRelationshipID, resource.Spec, iotsdk.LifecycleStateActive),
			}, nil
		},
		createFn: func(context.Context, iotsdk.CreateDigitalTwinRelationshipRequest) (iotsdk.CreateDigitalTwinRelationshipResponse, error) {
			t.Fatal("CreateDigitalTwinRelationship() called, want no-op observe")
			return iotsdk.CreateDigitalTwinRelationshipResponse{}, nil
		},
		listFn: func(context.Context, iotsdk.ListDigitalTwinRelationshipsRequest) (iotsdk.ListDigitalTwinRelationshipsResponse, error) {
			t.Fatal("ListDigitalTwinRelationships() called, want tracked GET observe")
			return iotsdk.ListDigitalTwinRelationshipsResponse{}, nil
		},
		updateFn: func(context.Context, iotsdk.UpdateDigitalTwinRelationshipRequest) (iotsdk.UpdateDigitalTwinRelationshipResponse, error) {
			t.Fatal("UpdateDigitalTwinRelationship() called, want no update for matching readback")
			return iotsdk.UpdateDigitalTwinRelationshipResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeDigitalTwinRelationshipRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if getCalls != 1 {
		t.Fatalf("GetDigitalTwinRelationship() calls = %d, want 1", getCalls)
	}
	requireDigitalTwinRelationshipBoundStatus(t, resource)
	requireLastCondition(t, resource, shared.Active)
}

func TestDigitalTwinRelationshipMutableUpdateShapesSupportedFields(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDigitalTwinRelationshipResource()
	resource.Spec.DisplayName = "updated relationship"
	resource.Spec.Description = "updated description"
	resource.Spec.Content = updatedDigitalTwinRelationshipContent()
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "99"}}
	getCalls := 0
	updateCalls := 0

	oldSpec := resource.Spec
	oldSpec.DisplayName = testDigitalTwinRelationshipName
	oldSpec.Description = "initial relationship"
	oldSpec.Content = testDigitalTwinRelationshipContent()
	oldSpec.FreeformTags = map[string]string{"env": "test"}
	oldSpec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "42"}}

	client := newTestDigitalTwinRelationshipClient(&fakeDigitalTwinRelationshipOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinRelationshipRequest) (iotsdk.GetDigitalTwinRelationshipResponse, error) {
			getCalls++
			requireStringPtr(t, "GetDigitalTwinRelationshipRequest.DigitalTwinRelationshipId", req.DigitalTwinRelationshipId, testDigitalTwinRelationshipID)
			spec := oldSpec
			if getCalls > 1 {
				spec = resource.Spec
			}
			return iotsdk.GetDigitalTwinRelationshipResponse{
				DigitalTwinRelationship: makeSDKDigitalTwinRelationship(t, testDigitalTwinRelationshipID, spec, iotsdk.LifecycleStateActive),
			}, nil
		},
		updateFn: func(_ context.Context, req iotsdk.UpdateDigitalTwinRelationshipRequest) (iotsdk.UpdateDigitalTwinRelationshipResponse, error) {
			updateCalls++
			requireDigitalTwinRelationshipUpdateRequest(t, req, resource)
			return iotsdk.UpdateDigitalTwinRelationshipResponse{
				DigitalTwinRelationship: makeSDKDigitalTwinRelationship(t, testDigitalTwinRelationshipID, resource.Spec, iotsdk.LifecycleStateActive),
				OpcRequestId:            common.String("opc-update"),
			}, nil
		},
		createFn: func(context.Context, iotsdk.CreateDigitalTwinRelationshipRequest) (iotsdk.CreateDigitalTwinRelationshipResponse, error) {
			t.Fatal("CreateDigitalTwinRelationship() called for tracked relationship update")
			return iotsdk.CreateDigitalTwinRelationshipResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeDigitalTwinRelationshipRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if getCalls != 2 {
		t.Fatalf("GetDigitalTwinRelationship() calls = %d, want 2", getCalls)
	}
	if updateCalls != 1 {
		t.Fatalf("UpdateDigitalTwinRelationship() calls = %d, want 1", updateCalls)
	}
	if got := resource.Status.DisplayName; got != resource.Spec.DisplayName {
		t.Fatalf("status.displayName = %q, want %q", got, resource.Spec.DisplayName)
	}
	requireStatusContent(t, resource, updatedDigitalTwinRelationshipContentObject(t))
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", got)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestDigitalTwinRelationshipCreateOnlyDriftRejectedBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDigitalTwinRelationshipResource()
	resource.Spec.ContentPath = "changed-relationship-path"
	updateCalled := false

	currentSpec := resource.Spec
	currentSpec.ContentPath = testDigitalTwinRelationshipPath
	client := newTestDigitalTwinRelationshipClient(&fakeDigitalTwinRelationshipOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinRelationshipRequest) (iotsdk.GetDigitalTwinRelationshipResponse, error) {
			requireStringPtr(t, "GetDigitalTwinRelationshipRequest.DigitalTwinRelationshipId", req.DigitalTwinRelationshipId, testDigitalTwinRelationshipID)
			return iotsdk.GetDigitalTwinRelationshipResponse{
				DigitalTwinRelationship: makeSDKDigitalTwinRelationship(t, testDigitalTwinRelationshipID, currentSpec, iotsdk.LifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, iotsdk.UpdateDigitalTwinRelationshipRequest) (iotsdk.UpdateDigitalTwinRelationshipResponse, error) {
			updateCalled = true
			return iotsdk.UpdateDigitalTwinRelationshipResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeDigitalTwinRelationshipRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if !strings.Contains(err.Error(), "contentPath") {
		t.Fatalf("CreateOrUpdate() error = %v, want contentPath drift", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false")
	}
	if updateCalled {
		t.Fatal("UpdateDigitalTwinRelationship() called despite create-only drift")
	}
	requireLastCondition(t, resource, shared.Failed)
}

func TestDigitalTwinRelationshipDeleteRetainsFinalizerWhileDeletePending(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDigitalTwinRelationshipResource()
	now := metav1.Now()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		RawStatus:       string(iotsdk.LifecycleStateActive),
		NormalizedClass: shared.OSOKAsyncClassPending,
		UpdatedAt:       &now,
	}
	getCalls := 0

	client := newTestDigitalTwinRelationshipClient(&fakeDigitalTwinRelationshipOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinRelationshipRequest) (iotsdk.GetDigitalTwinRelationshipResponse, error) {
			getCalls++
			requireStringPtr(t, "GetDigitalTwinRelationshipRequest.DigitalTwinRelationshipId", req.DigitalTwinRelationshipId, testDigitalTwinRelationshipID)
			return iotsdk.GetDigitalTwinRelationshipResponse{
				DigitalTwinRelationship: makeSDKDigitalTwinRelationship(t, testDigitalTwinRelationshipID, resource.Spec, iotsdk.LifecycleStateActive),
			}, nil
		},
		deleteFn: func(context.Context, iotsdk.DeleteDigitalTwinRelationshipRequest) (iotsdk.DeleteDigitalTwinRelationshipResponse, error) {
			t.Fatal("DeleteDigitalTwinRelationship() called while local delete async state is pending")
			return iotsdk.DeleteDigitalTwinRelationshipResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if getCalls != 2 {
		t.Fatalf("GetDigitalTwinRelationship() calls = %d, want 2 confirmation reads", getCalls)
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil || current.Phase != shared.OSOKAsyncPhaseDelete || current.NormalizedClass != shared.OSOKAsyncClassPending ||
		current.RawStatus != string(iotsdk.LifecycleStateActive) {
		t.Fatalf("status.async.current = %#v, want pending lifecycle delete", current)
	}
	requireLastCondition(t, resource, shared.Terminating)
}

func TestDigitalTwinRelationshipDeleteConfirmsDeletedAfterRequest(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDigitalTwinRelationshipResource()
	getCalls := 0
	deleteCalls := 0

	client := newTestDigitalTwinRelationshipClient(&fakeDigitalTwinRelationshipOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinRelationshipRequest) (iotsdk.GetDigitalTwinRelationshipResponse, error) {
			getCalls++
			requireStringPtr(t, "GetDigitalTwinRelationshipRequest.DigitalTwinRelationshipId", req.DigitalTwinRelationshipId, testDigitalTwinRelationshipID)
			state := iotsdk.LifecycleStateActive
			if getCalls == 3 {
				state = iotsdk.LifecycleStateDeleted
			}
			return iotsdk.GetDigitalTwinRelationshipResponse{
				DigitalTwinRelationship: makeSDKDigitalTwinRelationship(t, testDigitalTwinRelationshipID, resource.Spec, state),
			}, nil
		},
		deleteFn: func(_ context.Context, req iotsdk.DeleteDigitalTwinRelationshipRequest) (iotsdk.DeleteDigitalTwinRelationshipResponse, error) {
			deleteCalls++
			requireStringPtr(t, "DeleteDigitalTwinRelationshipRequest.DigitalTwinRelationshipId", req.DigitalTwinRelationshipId, testDigitalTwinRelationshipID)
			return iotsdk.DeleteDigitalTwinRelationshipResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want confirmed deletion")
	}
	if getCalls != 3 {
		t.Fatalf("GetDigitalTwinRelationship() calls = %d, want 3 confirmation reads", getCalls)
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteDigitalTwinRelationship() calls = %d, want 1", deleteCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", got)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
	requireLastCondition(t, resource, shared.Terminating)
}

func TestDigitalTwinRelationshipDeleteRejectsAmbiguousAuthShapedNotFound(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDigitalTwinRelationshipResource()
	deleteCalled := false
	client := newTestDigitalTwinRelationshipClient(&fakeDigitalTwinRelationshipOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinRelationshipRequest) (iotsdk.GetDigitalTwinRelationshipResponse, error) {
			requireStringPtr(t, "GetDigitalTwinRelationshipRequest.DigitalTwinRelationshipId", req.DigitalTwinRelationshipId, testDigitalTwinRelationshipID)
			return iotsdk.GetDigitalTwinRelationshipResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
		deleteFn: func(context.Context, iotsdk.DeleteDigitalTwinRelationshipRequest) (iotsdk.DeleteDigitalTwinRelationshipResponse, error) {
			deleteCalled = true
			return iotsdk.DeleteDigitalTwinRelationshipResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous auth-shaped not-found rejection")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous auth-shaped not-found rejection", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if deleteCalled {
		t.Fatal("DeleteDigitalTwinRelationship() called after ambiguous confirm read")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func requireDigitalTwinRelationshipCreateRequest(
	t *testing.T,
	req iotsdk.CreateDigitalTwinRelationshipRequest,
	resource *iotv1beta1.DigitalTwinRelationship,
) {
	t.Helper()
	requireStringPtr(t, "CreateDigitalTwinRelationshipDetails.IotDomainId", req.IotDomainId, resource.Spec.IotDomainId)
	requireStringPtr(t, "CreateDigitalTwinRelationshipDetails.ContentPath", req.ContentPath, resource.Spec.ContentPath)
	requireStringPtr(t, "CreateDigitalTwinRelationshipDetails.SourceDigitalTwinInstanceId", req.SourceDigitalTwinInstanceId, resource.Spec.SourceDigitalTwinInstanceId)
	requireStringPtr(t, "CreateDigitalTwinRelationshipDetails.TargetDigitalTwinInstanceId", req.TargetDigitalTwinInstanceId, resource.Spec.TargetDigitalTwinInstanceId)
	requireStringPtr(t, "CreateDigitalTwinRelationshipDetails.DisplayName", req.DisplayName, resource.Spec.DisplayName)
	requireStringPtr(t, "CreateDigitalTwinRelationshipDetails.Description", req.Description, resource.Spec.Description)
	requireMap(t, "CreateDigitalTwinRelationshipDetails.Content", req.Content, testDigitalTwinRelationshipContentObject(t))
	if !reflect.DeepEqual(req.FreeformTags, resource.Spec.FreeformTags) {
		t.Fatalf("CreateDigitalTwinRelationshipDetails.FreeformTags = %#v, want %#v", req.FreeformTags, resource.Spec.FreeformTags)
	}
	requireMap(t, "CreateDigitalTwinRelationshipDetails.DefinedTags", req.DefinedTags, digitalTwinRelationshipDefinedTags(resource.Spec.DefinedTags))
	if req.OpcRetryToken == nil || strings.TrimSpace(*req.OpcRetryToken) == "" {
		t.Fatal("CreateDigitalTwinRelationshipRequest.OpcRetryToken is empty, want deterministic retry token")
	}
}

func requireDigitalTwinRelationshipUpdateRequest(
	t *testing.T,
	req iotsdk.UpdateDigitalTwinRelationshipRequest,
	resource *iotv1beta1.DigitalTwinRelationship,
) {
	t.Helper()
	requireStringPtr(t, "UpdateDigitalTwinRelationshipRequest.DigitalTwinRelationshipId", req.DigitalTwinRelationshipId, testDigitalTwinRelationshipID)
	requireStringPtr(t, "UpdateDigitalTwinRelationshipDetails.DisplayName", req.DisplayName, resource.Spec.DisplayName)
	requireStringPtr(t, "UpdateDigitalTwinRelationshipDetails.Description", req.Description, resource.Spec.Description)
	requireMap(t, "UpdateDigitalTwinRelationshipDetails.Content", req.Content, updatedDigitalTwinRelationshipContentObject(t))
	if !reflect.DeepEqual(req.FreeformTags, resource.Spec.FreeformTags) {
		t.Fatalf("UpdateDigitalTwinRelationshipDetails.FreeformTags = %#v, want %#v", req.FreeformTags, resource.Spec.FreeformTags)
	}
	requireMap(t, "UpdateDigitalTwinRelationshipDetails.DefinedTags", req.DefinedTags, digitalTwinRelationshipDefinedTags(resource.Spec.DefinedTags))
}

func requireDigitalTwinRelationshipBoundStatus(t *testing.T, resource *iotv1beta1.DigitalTwinRelationship) {
	t.Helper()
	if got := string(resource.Status.OsokStatus.Ocid); got != testDigitalTwinRelationshipID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testDigitalTwinRelationshipID)
	}
	if got := resource.Status.Id; got != testDigitalTwinRelationshipID {
		t.Fatalf("status.id = %q, want %q", got, testDigitalTwinRelationshipID)
	}
	if got := resource.Status.IotDomainId; got != resource.Spec.IotDomainId {
		t.Fatalf("status.iotDomainId = %q, want %q", got, resource.Spec.IotDomainId)
	}
	if got := resource.Status.ContentPath; got != resource.Spec.ContentPath {
		t.Fatalf("status.contentPath = %q, want %q", got, resource.Spec.ContentPath)
	}
	if got := resource.Status.SourceDigitalTwinInstanceId; got != resource.Spec.SourceDigitalTwinInstanceId {
		t.Fatalf("status.sourceDigitalTwinInstanceId = %q, want %q", got, resource.Spec.SourceDigitalTwinInstanceId)
	}
	if got := resource.Status.TargetDigitalTwinInstanceId; got != resource.Spec.TargetDigitalTwinInstanceId {
		t.Fatalf("status.targetDigitalTwinInstanceId = %q, want %q", got, resource.Spec.TargetDigitalTwinInstanceId)
	}
	requireStatusContent(t, resource, testDigitalTwinRelationshipContentObject(t))
}

func requireStatusContent(
	t *testing.T,
	resource *iotv1beta1.DigitalTwinRelationship,
	want map[string]interface{},
) {
	t.Helper()
	got, err := digitalTwinRelationshipContent(resource.Status.Content)
	if err != nil {
		t.Fatalf("status.content decode error = %v", err)
	}
	requireMap(t, "status.content", got, want)
}

func requireMap(t *testing.T, name string, got any, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
}

func requireStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", name, *got, want)
	}
}

func requireLastCondition(t *testing.T, resource *iotv1beta1.DigitalTwinRelationship, want shared.OSOKConditionType) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.conditions is empty, want last condition %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last status condition = %s, want %s", got, want)
	}
}

func jsonValue(raw string) shared.JSONValue {
	return shared.JSONValue{Raw: []byte(raw)}
}

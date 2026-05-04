/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package digitaltwinmodel

import (
	"context"
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
	testDigitalTwinModelID       = "ocid1.digitaltwinmodel.oc1..model"
	testDigitalTwinModelDomainID = "ocid1.iotdomain.oc1..domain"
	testDigitalTwinModelSpecURI  = "dtmi:com:example:Thermostat;1"
	testDigitalTwinModelName     = "thermostat-model"
)

type fakeDigitalTwinModelOCIClient struct {
	createFn  func(context.Context, iotsdk.CreateDigitalTwinModelRequest) (iotsdk.CreateDigitalTwinModelResponse, error)
	getFn     func(context.Context, iotsdk.GetDigitalTwinModelRequest) (iotsdk.GetDigitalTwinModelResponse, error)
	getSpecFn func(context.Context, iotsdk.GetDigitalTwinModelSpecRequest) (iotsdk.GetDigitalTwinModelSpecResponse, error)
	listFn    func(context.Context, iotsdk.ListDigitalTwinModelsRequest) (iotsdk.ListDigitalTwinModelsResponse, error)
	updateFn  func(context.Context, iotsdk.UpdateDigitalTwinModelRequest) (iotsdk.UpdateDigitalTwinModelResponse, error)
	deleteFn  func(context.Context, iotsdk.DeleteDigitalTwinModelRequest) (iotsdk.DeleteDigitalTwinModelResponse, error)
}

type pagedDigitalTwinModelSpecURIList struct {
	t        *testing.T
	resource *iotv1beta1.DigitalTwinModel
	calls    int
}

func (f *fakeDigitalTwinModelOCIClient) CreateDigitalTwinModel(ctx context.Context, req iotsdk.CreateDigitalTwinModelRequest) (iotsdk.CreateDigitalTwinModelResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return iotsdk.CreateDigitalTwinModelResponse{}, nil
}

func (f *fakeDigitalTwinModelOCIClient) GetDigitalTwinModel(ctx context.Context, req iotsdk.GetDigitalTwinModelRequest) (iotsdk.GetDigitalTwinModelResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return iotsdk.GetDigitalTwinModelResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "digital twin model is missing")
}

func (f *fakeDigitalTwinModelOCIClient) GetDigitalTwinModelSpec(ctx context.Context, req iotsdk.GetDigitalTwinModelSpecRequest) (iotsdk.GetDigitalTwinModelSpecResponse, error) {
	if f.getSpecFn != nil {
		return f.getSpecFn(ctx, req)
	}
	return iotsdk.GetDigitalTwinModelSpecResponse{}, nil
}

func (f *fakeDigitalTwinModelOCIClient) ListDigitalTwinModels(ctx context.Context, req iotsdk.ListDigitalTwinModelsRequest) (iotsdk.ListDigitalTwinModelsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return iotsdk.ListDigitalTwinModelsResponse{}, nil
}

func (f *fakeDigitalTwinModelOCIClient) UpdateDigitalTwinModel(ctx context.Context, req iotsdk.UpdateDigitalTwinModelRequest) (iotsdk.UpdateDigitalTwinModelResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return iotsdk.UpdateDigitalTwinModelResponse{}, nil
}

func (f *fakeDigitalTwinModelOCIClient) DeleteDigitalTwinModel(ctx context.Context, req iotsdk.DeleteDigitalTwinModelRequest) (iotsdk.DeleteDigitalTwinModelResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return iotsdk.DeleteDigitalTwinModelResponse{}, nil
}

func (f *pagedDigitalTwinModelSpecURIList) list(
	_ context.Context,
	req iotsdk.ListDigitalTwinModelsRequest,
) (iotsdk.ListDigitalTwinModelsResponse, error) {
	f.calls++
	requireStringPtr(f.t, "ListDigitalTwinModelsRequest.IotDomainId", req.IotDomainId, f.resource.Spec.IotDomainId)
	requireStringPtr(f.t, "ListDigitalTwinModelsRequest.SpecUriStartsWith", req.SpecUriStartsWith, testDigitalTwinModelSpecURI)
	if f.calls == 1 {
		return f.firstPage(req), nil
	}
	return f.secondPage(req), nil
}

func (f *pagedDigitalTwinModelSpecURIList) firstPage(
	req iotsdk.ListDigitalTwinModelsRequest,
) iotsdk.ListDigitalTwinModelsResponse {
	if req.Page != nil {
		f.t.Fatalf("first ListDigitalTwinModelsRequest.Page = %q, want nil", *req.Page)
	}
	other := makeSDKDigitalTwinModelSummary(f.t, "ocid1.digitaltwinmodel.oc1..other", f.resource.Spec, iotsdk.LifecycleStateActive)
	other.SpecUri = common.String(testDigitalTwinModelSpecURI + "0")
	return iotsdk.ListDigitalTwinModelsResponse{
		DigitalTwinModelCollection: iotsdk.DigitalTwinModelCollection{
			Items: []iotsdk.DigitalTwinModelSummary{other},
		},
		OpcNextPage: common.String("page-2"),
	}
}

func (f *pagedDigitalTwinModelSpecURIList) secondPage(
	req iotsdk.ListDigitalTwinModelsRequest,
) iotsdk.ListDigitalTwinModelsResponse {
	requireStringPtr(f.t, "second ListDigitalTwinModelsRequest.Page", req.Page, "page-2")
	return iotsdk.ListDigitalTwinModelsResponse{
		DigitalTwinModelCollection: iotsdk.DigitalTwinModelCollection{
			Items: []iotsdk.DigitalTwinModelSummary{
				makeSDKDigitalTwinModelSummary(f.t, testDigitalTwinModelID, f.resource.Spec, iotsdk.LifecycleStateActive),
			},
		},
	}
}

func newTestDigitalTwinModelClient(client digitalTwinModelOCIClient) DigitalTwinModelServiceClient {
	return newDigitalTwinModelServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
		client,
	)
}

func makeDigitalTwinModelResource() *iotv1beta1.DigitalTwinModel {
	return &iotv1beta1.DigitalTwinModel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDigitalTwinModelName,
			Namespace: "default",
		},
		Spec: iotv1beta1.DigitalTwinModelSpec{
			IotDomainId: testDigitalTwinModelDomainID,
			Spec:        testDigitalTwinModelSpec(),
			DisplayName: testDigitalTwinModelName,
			Description: "initial description",
			FreeformTags: map[string]string{
				"env": "test",
			},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func makeTrackedDigitalTwinModelResource() *iotv1beta1.DigitalTwinModel {
	resource := makeDigitalTwinModelResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testDigitalTwinModelID)
	resource.Status.Id = testDigitalTwinModelID
	resource.Status.IotDomainId = testDigitalTwinModelDomainID
	resource.Status.SpecUri = testDigitalTwinModelSpecURI
	resource.Status.DisplayName = testDigitalTwinModelName
	resource.Status.LifecycleState = string(iotsdk.LifecycleStateActive)
	return resource
}

func makeDigitalTwinModelRequest(resource *iotv1beta1.DigitalTwinModel) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		},
	}
}

func testDigitalTwinModelSpec() map[string]shared.JSONValue {
	return map[string]shared.JSONValue{
		"@context":    jsonValue(`"dtmi:dtdl:context;3"`),
		"@id":         jsonValue(`"` + testDigitalTwinModelSpecURI + `"`),
		"@type":       jsonValue(`"Interface"`),
		"displayName": jsonValue(`"Thermostat"`),
		"contents":    jsonValue(`[{"@type":"Property","name":"temperature","schema":"double"}]`),
	}
}

func testDigitalTwinModelSpecObject(t *testing.T) map[string]interface{} {
	t.Helper()
	spec, err := digitalTwinModelSpecMap(testDigitalTwinModelSpec())
	if err != nil {
		t.Fatalf("digitalTwinModelSpecMap() error = %v", err)
	}
	return spec
}

func makeSDKDigitalTwinModel(
	t *testing.T,
	id string,
	spec iotv1beta1.DigitalTwinModelSpec,
	state iotsdk.LifecycleStateEnum,
) iotsdk.DigitalTwinModel {
	t.Helper()
	specURI, err := desiredDigitalTwinModelSpecURI(&iotv1beta1.DigitalTwinModel{Spec: spec})
	if err != nil {
		t.Fatalf("desiredDigitalTwinModelSpecURI() error = %v", err)
	}
	return iotsdk.DigitalTwinModel{
		Id:             common.String(id),
		IotDomainId:    common.String(spec.IotDomainId),
		SpecUri:        common.String(specURI),
		DisplayName:    common.String(spec.DisplayName),
		Description:    common.String(spec.Description),
		LifecycleState: state,
		FreeformTags:   cloneDigitalTwinModelStringMap(spec.FreeformTags),
		DefinedTags:    digitalTwinModelDefinedTags(spec.DefinedTags),
	}
}

func makeSDKDigitalTwinModelSummary(
	t *testing.T,
	id string,
	spec iotv1beta1.DigitalTwinModelSpec,
	state iotsdk.LifecycleStateEnum,
) iotsdk.DigitalTwinModelSummary {
	t.Helper()
	specURI, err := desiredDigitalTwinModelSpecURI(&iotv1beta1.DigitalTwinModel{Spec: spec})
	if err != nil {
		t.Fatalf("desiredDigitalTwinModelSpecURI() error = %v", err)
	}
	return iotsdk.DigitalTwinModelSummary{
		Id:             common.String(id),
		IotDomainId:    common.String(spec.IotDomainId),
		SpecUri:        common.String(specURI),
		DisplayName:    common.String(spec.DisplayName),
		Description:    common.String(spec.Description),
		LifecycleState: state,
		FreeformTags:   cloneDigitalTwinModelStringMap(spec.FreeformTags),
		DefinedTags:    digitalTwinModelDefinedTags(spec.DefinedTags),
	}
}

func TestDigitalTwinModelCreateOrUpdateBindsExistingModelByPagedSpecURIList(t *testing.T) {
	t.Parallel()

	resource := makeDigitalTwinModelResource()
	createCalled := false
	updateCalled := false
	listPages := &pagedDigitalTwinModelSpecURIList{t: t, resource: resource}
	getCalls := 0
	getSpecCalls := 0

	client := newTestDigitalTwinModelClient(&fakeDigitalTwinModelOCIClient{
		listFn: listPages.list,
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinModelRequest) (iotsdk.GetDigitalTwinModelResponse, error) {
			getCalls++
			requireStringPtr(t, "GetDigitalTwinModelRequest.DigitalTwinModelId", req.DigitalTwinModelId, testDigitalTwinModelID)
			return iotsdk.GetDigitalTwinModelResponse{
				DigitalTwinModel: makeSDKDigitalTwinModel(t, testDigitalTwinModelID, resource.Spec, iotsdk.LifecycleStateActive),
			}, nil
		},
		getSpecFn: func(_ context.Context, req iotsdk.GetDigitalTwinModelSpecRequest) (iotsdk.GetDigitalTwinModelSpecResponse, error) {
			getSpecCalls++
			requireStringPtr(t, "GetDigitalTwinModelSpecRequest.DigitalTwinModelId", req.DigitalTwinModelId, testDigitalTwinModelID)
			return iotsdk.GetDigitalTwinModelSpecResponse{Object: testDigitalTwinModelSpecObject(t)}, nil
		},
		createFn: func(context.Context, iotsdk.CreateDigitalTwinModelRequest) (iotsdk.CreateDigitalTwinModelResponse, error) {
			createCalled = true
			return iotsdk.CreateDigitalTwinModelResponse{}, nil
		},
		updateFn: func(context.Context, iotsdk.UpdateDigitalTwinModelRequest) (iotsdk.UpdateDigitalTwinModelResponse, error) {
			updateCalled = true
			return iotsdk.UpdateDigitalTwinModelResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeDigitalTwinModelRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	requireDigitalTwinModelBindCalls(t, createCalled, updateCalled, listPages.calls, getCalls, getSpecCalls)
	requireDigitalTwinModelBoundStatus(t, resource)
	requireLastCondition(t, resource, shared.Active)
}

func TestDigitalTwinModelCreateRecordsSpecRetryTokenRequestIDAndStatus(t *testing.T) {
	t.Parallel()

	resource := makeDigitalTwinModelResource()
	listCalls := 0
	createCalls := 0

	client := newTestDigitalTwinModelClient(&fakeDigitalTwinModelOCIClient{
		listFn: func(context.Context, iotsdk.ListDigitalTwinModelsRequest) (iotsdk.ListDigitalTwinModelsResponse, error) {
			listCalls++
			return iotsdk.ListDigitalTwinModelsResponse{}, nil
		},
		createFn: func(_ context.Context, req iotsdk.CreateDigitalTwinModelRequest) (iotsdk.CreateDigitalTwinModelResponse, error) {
			createCalls++
			requireDigitalTwinModelCreateRequest(t, req, resource)
			return iotsdk.CreateDigitalTwinModelResponse{
				DigitalTwinModel: makeSDKDigitalTwinModel(t, testDigitalTwinModelID, resource.Spec, iotsdk.LifecycleStateActive),
				OpcRequestId:     common.String("opc-create"),
			}, nil
		},
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinModelRequest) (iotsdk.GetDigitalTwinModelResponse, error) {
			requireStringPtr(t, "GetDigitalTwinModelRequest.DigitalTwinModelId", req.DigitalTwinModelId, testDigitalTwinModelID)
			return iotsdk.GetDigitalTwinModelResponse{
				DigitalTwinModel: makeSDKDigitalTwinModel(t, testDigitalTwinModelID, resource.Spec, iotsdk.LifecycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeDigitalTwinModelRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if listCalls != 1 {
		t.Fatalf("ListDigitalTwinModels() calls = %d, want 1 pre-create lookup", listCalls)
	}
	if createCalls != 1 {
		t.Fatalf("CreateDigitalTwinModel() calls = %d, want 1", createCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testDigitalTwinModelID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testDigitalTwinModelID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", got)
	}
	if got := resource.Status.SpecUri; got != testDigitalTwinModelSpecURI {
		t.Fatalf("status.specUri = %q, want %q", got, testDigitalTwinModelSpecURI)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestDigitalTwinModelCreateOrUpdateNoopsWhenReadbackAndSpecMatch(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDigitalTwinModelResource()
	updateCalled := false

	client := newTestDigitalTwinModelClient(&fakeDigitalTwinModelOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinModelRequest) (iotsdk.GetDigitalTwinModelResponse, error) {
			requireStringPtr(t, "GetDigitalTwinModelRequest.DigitalTwinModelId", req.DigitalTwinModelId, testDigitalTwinModelID)
			return iotsdk.GetDigitalTwinModelResponse{
				DigitalTwinModel: makeSDKDigitalTwinModel(t, testDigitalTwinModelID, resource.Spec, iotsdk.LifecycleStateActive),
			}, nil
		},
		getSpecFn: func(_ context.Context, req iotsdk.GetDigitalTwinModelSpecRequest) (iotsdk.GetDigitalTwinModelSpecResponse, error) {
			requireStringPtr(t, "GetDigitalTwinModelSpecRequest.DigitalTwinModelId", req.DigitalTwinModelId, testDigitalTwinModelID)
			return iotsdk.GetDigitalTwinModelSpecResponse{Object: testDigitalTwinModelSpecObject(t)}, nil
		},
		updateFn: func(context.Context, iotsdk.UpdateDigitalTwinModelRequest) (iotsdk.UpdateDigitalTwinModelResponse, error) {
			updateCalled = true
			return iotsdk.UpdateDigitalTwinModelResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeDigitalTwinModelRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if updateCalled {
		t.Fatal("UpdateDigitalTwinModel() called for matching readback")
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestDigitalTwinModelCreateOrUpdateUpdatesMutableFields(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDigitalTwinModelResource()
	resource.Spec.DisplayName = "updated-model"
	resource.Spec.Description = "updated description"
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "99"}}
	currentSpec := resource.Spec
	currentSpec.DisplayName = testDigitalTwinModelName
	currentSpec.Description = "initial description"
	currentSpec.FreeformTags = map[string]string{"env": "test"}
	currentSpec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "42"}}
	getCalls := 0
	updateCalls := 0

	client := newTestDigitalTwinModelClient(&fakeDigitalTwinModelOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinModelRequest) (iotsdk.GetDigitalTwinModelResponse, error) {
			getCalls++
			requireStringPtr(t, "GetDigitalTwinModelRequest.DigitalTwinModelId", req.DigitalTwinModelId, testDigitalTwinModelID)
			if getCalls == 1 {
				return iotsdk.GetDigitalTwinModelResponse{
					DigitalTwinModel: makeSDKDigitalTwinModel(t, testDigitalTwinModelID, currentSpec, iotsdk.LifecycleStateActive),
				}, nil
			}
			return iotsdk.GetDigitalTwinModelResponse{
				DigitalTwinModel: makeSDKDigitalTwinModel(t, testDigitalTwinModelID, resource.Spec, iotsdk.LifecycleStateActive),
			}, nil
		},
		getSpecFn: func(_ context.Context, req iotsdk.GetDigitalTwinModelSpecRequest) (iotsdk.GetDigitalTwinModelSpecResponse, error) {
			requireStringPtr(t, "GetDigitalTwinModelSpecRequest.DigitalTwinModelId", req.DigitalTwinModelId, testDigitalTwinModelID)
			return iotsdk.GetDigitalTwinModelSpecResponse{Object: testDigitalTwinModelSpecObject(t)}, nil
		},
		updateFn: func(_ context.Context, req iotsdk.UpdateDigitalTwinModelRequest) (iotsdk.UpdateDigitalTwinModelResponse, error) {
			updateCalls++
			requireStringPtr(t, "UpdateDigitalTwinModelRequest.DigitalTwinModelId", req.DigitalTwinModelId, testDigitalTwinModelID)
			requireStringPtr(t, "UpdateDigitalTwinModelDetails.DisplayName", req.DisplayName, resource.Spec.DisplayName)
			requireStringPtr(t, "UpdateDigitalTwinModelDetails.Description", req.Description, resource.Spec.Description)
			if got := req.FreeformTags["env"]; got != "prod" {
				t.Fatalf("UpdateDigitalTwinModelDetails.FreeformTags[env] = %q, want prod", got)
			}
			if got := req.DefinedTags["Operations"]["CostCenter"]; got != "99" {
				t.Fatalf("UpdateDigitalTwinModelDetails.DefinedTags[Operations][CostCenter] = %v, want 99", got)
			}
			return iotsdk.UpdateDigitalTwinModelResponse{
				DigitalTwinModel: makeSDKDigitalTwinModel(t, testDigitalTwinModelID, resource.Spec, iotsdk.LifecycleStateActive),
				OpcRequestId:     common.String("opc-update"),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeDigitalTwinModelRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if updateCalls != 1 {
		t.Fatalf("UpdateDigitalTwinModel() calls = %d, want 1", updateCalls)
	}
	if getCalls != 2 {
		t.Fatalf("GetDigitalTwinModel() calls = %d, want current read and update follow-up", getCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", got)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestDigitalTwinModelCreateOrUpdateRejectsIotDomainDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDigitalTwinModelResource()
	resource.Spec.IotDomainId = "ocid1.iotdomain.oc1..different"
	currentSpec := resource.Spec
	currentSpec.IotDomainId = testDigitalTwinModelDomainID
	updateCalled := false

	client := newTestDigitalTwinModelClient(&fakeDigitalTwinModelOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinModelRequest) (iotsdk.GetDigitalTwinModelResponse, error) {
			requireStringPtr(t, "GetDigitalTwinModelRequest.DigitalTwinModelId", req.DigitalTwinModelId, testDigitalTwinModelID)
			return iotsdk.GetDigitalTwinModelResponse{
				DigitalTwinModel: makeSDKDigitalTwinModel(t, testDigitalTwinModelID, currentSpec, iotsdk.LifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, iotsdk.UpdateDigitalTwinModelRequest) (iotsdk.UpdateDigitalTwinModelResponse, error) {
			updateCalled = true
			return iotsdk.UpdateDigitalTwinModelResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeDigitalTwinModelRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false")
	}
	if updateCalled {
		t.Fatal("UpdateDigitalTwinModel() called after create-only iotDomainId drift")
	}
	if !strings.Contains(err.Error(), "iotDomainId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want iotDomainId force-new rejection", err)
	}
	requireLastCondition(t, resource, shared.Failed)
}

func TestDigitalTwinModelCreateOrUpdateRejectsSpecDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDigitalTwinModelResource()
	resource.Spec.Spec["contents"] = jsonValue(`[{"@type":"Property","name":"humidity","schema":"double"}]`)
	updateCalled := false
	getSpecCalls := 0

	client := newTestDigitalTwinModelClient(&fakeDigitalTwinModelOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinModelRequest) (iotsdk.GetDigitalTwinModelResponse, error) {
			requireStringPtr(t, "GetDigitalTwinModelRequest.DigitalTwinModelId", req.DigitalTwinModelId, testDigitalTwinModelID)
			return iotsdk.GetDigitalTwinModelResponse{
				DigitalTwinModel: makeSDKDigitalTwinModel(t, testDigitalTwinModelID, resource.Spec, iotsdk.LifecycleStateActive),
			}, nil
		},
		getSpecFn: func(_ context.Context, req iotsdk.GetDigitalTwinModelSpecRequest) (iotsdk.GetDigitalTwinModelSpecResponse, error) {
			getSpecCalls++
			requireStringPtr(t, "GetDigitalTwinModelSpecRequest.DigitalTwinModelId", req.DigitalTwinModelId, testDigitalTwinModelID)
			return iotsdk.GetDigitalTwinModelSpecResponse{Object: testDigitalTwinModelSpecObject(t)}, nil
		},
		updateFn: func(context.Context, iotsdk.UpdateDigitalTwinModelRequest) (iotsdk.UpdateDigitalTwinModelResponse, error) {
			updateCalled = true
			return iotsdk.UpdateDigitalTwinModelResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeDigitalTwinModelRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want spec drift rejection")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false")
	}
	if updateCalled {
		t.Fatal("UpdateDigitalTwinModel() called after create-only spec drift")
	}
	if getSpecCalls == 0 {
		t.Fatal("GetDigitalTwinModelSpec() was not called to check create-only spec drift")
	}
	if !strings.Contains(err.Error(), "spec changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want spec force-new rejection", err)
	}
	requireLastCondition(t, resource, shared.Failed)
}

func TestDigitalTwinModelDeleteKeepsFinalizerUntilReadConfirmsNotFound(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDigitalTwinModelResource()
	getCalls := 0
	deleteCalls := 0

	client := newTestDigitalTwinModelClient(&fakeDigitalTwinModelOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinModelRequest) (iotsdk.GetDigitalTwinModelResponse, error) {
			getCalls++
			requireStringPtr(t, "GetDigitalTwinModelRequest.DigitalTwinModelId", req.DigitalTwinModelId, testDigitalTwinModelID)
			if getCalls <= 3 {
				return iotsdk.GetDigitalTwinModelResponse{
					DigitalTwinModel: makeSDKDigitalTwinModel(t, testDigitalTwinModelID, resource.Spec, iotsdk.LifecycleStateActive),
				}, nil
			}
			return iotsdk.GetDigitalTwinModelResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "digital twin model is gone")
		},
		deleteFn: func(_ context.Context, req iotsdk.DeleteDigitalTwinModelRequest) (iotsdk.DeleteDigitalTwinModelResponse, error) {
			deleteCalls++
			requireStringPtr(t, "DeleteDigitalTwinModelRequest.DigitalTwinModelId", req.DigitalTwinModelId, testDigitalTwinModelID)
			return iotsdk.DeleteDigitalTwinModelResponse{
				OpcRequestId: common.String("opc-delete"),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() first error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() first deleted = true, want false while readback remains ACTIVE")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteDigitalTwinModel() calls after first delete = %d, want 1", deleteCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", got)
	}
	requireDeletePendingStatus(t, resource)
	requireLastCondition(t, resource, shared.Terminating)

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() second error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() second deleted = false, want true after unambiguous NotFound")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteDigitalTwinModel() calls after confirmed delete = %d, want still 1", deleteCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
	requireLastCondition(t, resource, shared.Terminating)
}

func TestDigitalTwinModelDeleteTreatsAuthShapedNotFoundConservatively(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDigitalTwinModelResource()

	client := newTestDigitalTwinModelClient(&fakeDigitalTwinModelOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinModelRequest) (iotsdk.GetDigitalTwinModelResponse, error) {
			requireStringPtr(t, "GetDigitalTwinModelRequest.DigitalTwinModelId", req.DigitalTwinModelId, testDigitalTwinModelID)
			return iotsdk.GetDigitalTwinModelResponse{
				DigitalTwinModel: makeSDKDigitalTwinModel(t, testDigitalTwinModelID, resource.Spec, iotsdk.LifecycleStateActive),
			}, nil
		},
		deleteFn: func(_ context.Context, req iotsdk.DeleteDigitalTwinModelRequest) (iotsdk.DeleteDigitalTwinModelResponse, error) {
			requireStringPtr(t, "DeleteDigitalTwinModelRequest.DigitalTwinModelId", req.DigitalTwinModelId, testDigitalTwinModelID)
			return iotsdk.DeleteDigitalTwinModelResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization or existence is ambiguous")
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous not-found error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped 404")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous not-found classification", err)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestDigitalTwinModelDeleteRejectsAuthShapedPreDeleteConfirmRead(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDigitalTwinModelResource()
	deleteCalled := false

	client := newTestDigitalTwinModelClient(&fakeDigitalTwinModelOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinModelRequest) (iotsdk.GetDigitalTwinModelResponse, error) {
			requireStringPtr(t, "GetDigitalTwinModelRequest.DigitalTwinModelId", req.DigitalTwinModelId, testDigitalTwinModelID)
			return iotsdk.GetDigitalTwinModelResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization or existence is ambiguous")
		},
		deleteFn: func(context.Context, iotsdk.DeleteDigitalTwinModelRequest) (iotsdk.DeleteDigitalTwinModelResponse, error) {
			deleteCalled = true
			return iotsdk.DeleteDigitalTwinModelResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous pre-delete confirm-read error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped pre-delete confirm read")
	}
	if deleteCalled {
		t.Fatal("DeleteDigitalTwinModel() called after auth-shaped pre-delete confirm read")
	}
	if !strings.Contains(err.Error(), "delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous confirm-read classification", err)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestDigitalTwinModelDeleteRejectsAuthShapedPostDeleteConfirmRead(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDigitalTwinModelResource()
	getCalls := 0
	deleteCalls := 0

	client := newTestDigitalTwinModelClient(&fakeDigitalTwinModelOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinModelRequest) (iotsdk.GetDigitalTwinModelResponse, error) {
			getCalls++
			requireStringPtr(t, "GetDigitalTwinModelRequest.DigitalTwinModelId", req.DigitalTwinModelId, testDigitalTwinModelID)
			if getCalls < 3 {
				return iotsdk.GetDigitalTwinModelResponse{
					DigitalTwinModel: makeSDKDigitalTwinModel(t, testDigitalTwinModelID, resource.Spec, iotsdk.LifecycleStateActive),
				}, nil
			}
			return iotsdk.GetDigitalTwinModelResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization or existence is ambiguous")
		},
		deleteFn: func(_ context.Context, req iotsdk.DeleteDigitalTwinModelRequest) (iotsdk.DeleteDigitalTwinModelResponse, error) {
			deleteCalls++
			requireStringPtr(t, "DeleteDigitalTwinModelRequest.DigitalTwinModelId", req.DigitalTwinModelId, testDigitalTwinModelID)
			return iotsdk.DeleteDigitalTwinModelResponse{
				OpcRequestId: common.String("opc-delete"),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous post-delete confirm-read error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped post-delete confirm read")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteDigitalTwinModel() calls = %d, want 1", deleteCalls)
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous confirm-read classification", err)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestDigitalTwinModelCreateOrUpdateRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	resource := makeDigitalTwinModelResource()

	client := newTestDigitalTwinModelClient(&fakeDigitalTwinModelOCIClient{
		listFn: func(context.Context, iotsdk.ListDigitalTwinModelsRequest) (iotsdk.ListDigitalTwinModelsResponse, error) {
			return iotsdk.ListDigitalTwinModelsResponse{}, nil
		},
		createFn: func(context.Context, iotsdk.CreateDigitalTwinModelRequest) (iotsdk.CreateDigitalTwinModelResponse, error) {
			return iotsdk.CreateDigitalTwinModelResponse{}, errortest.NewServiceError(500, errorutil.InternalServerError, "create failed")
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeDigitalTwinModelRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI service error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
	requireLastCondition(t, resource, shared.Failed)
}

func requireDigitalTwinModelCreateRequest(
	t *testing.T,
	req iotsdk.CreateDigitalTwinModelRequest,
	resource *iotv1beta1.DigitalTwinModel,
) {
	t.Helper()
	requireStringPtr(t, "CreateDigitalTwinModelDetails.IotDomainId", req.IotDomainId, resource.Spec.IotDomainId)
	requireStringPtr(t, "CreateDigitalTwinModelDetails.DisplayName", req.DisplayName, resource.Spec.DisplayName)
	requireStringPtr(t, "CreateDigitalTwinModelDetails.Description", req.Description, resource.Spec.Description)
	if got := req.Spec["@id"]; got != testDigitalTwinModelSpecURI {
		t.Fatalf("CreateDigitalTwinModelDetails.Spec[@id] = %v, want %q", got, testDigitalTwinModelSpecURI)
	}
	if got := req.Spec["@type"]; got != "Interface" {
		t.Fatalf("CreateDigitalTwinModelDetails.Spec[@type] = %v, want Interface", got)
	}
	if got := req.FreeformTags["env"]; got != "test" {
		t.Fatalf("CreateDigitalTwinModelDetails.FreeformTags[env] = %q, want test", got)
	}
	if got := req.DefinedTags["Operations"]["CostCenter"]; got != "42" {
		t.Fatalf("CreateDigitalTwinModelDetails.DefinedTags[Operations][CostCenter] = %v, want 42", got)
	}
	if req.OpcRetryToken == nil || strings.TrimSpace(*req.OpcRetryToken) == "" {
		t.Fatal("CreateDigitalTwinModelRequest.OpcRetryToken is empty, want deterministic retry token")
	}
}

func requireDigitalTwinModelBindCalls(
	t *testing.T,
	createCalled bool,
	updateCalled bool,
	listCalls int,
	getCalls int,
	getSpecCalls int,
) {
	t.Helper()
	if createCalled {
		t.Fatal("CreateDigitalTwinModel() called for existing model")
	}
	if updateCalled {
		t.Fatal("UpdateDigitalTwinModel() called for matching model")
	}
	if listCalls != 2 {
		t.Fatalf("ListDigitalTwinModels() calls = %d, want 2 paginated calls", listCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetDigitalTwinModel() calls = %d, want 1", getCalls)
	}
	if getSpecCalls != 1 {
		t.Fatalf("GetDigitalTwinModelSpec() calls = %d, want 1", getSpecCalls)
	}
}

func requireDigitalTwinModelBoundStatus(t *testing.T, resource *iotv1beta1.DigitalTwinModel) {
	t.Helper()
	if got := string(resource.Status.OsokStatus.Ocid); got != testDigitalTwinModelID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testDigitalTwinModelID)
	}
	if got := resource.Status.SpecUri; got != testDigitalTwinModelSpecURI {
		t.Fatalf("status.specUri = %q, want %q", got, testDigitalTwinModelSpecURI)
	}
}

func requireDeletePendingStatus(t *testing.T, resource *iotv1beta1.DigitalTwinModel) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.status.async.current = nil, want delete pending tracker")
	}
	if current.Source != shared.OSOKAsyncSourceLifecycle ||
		current.Phase != shared.OSOKAsyncPhaseDelete ||
		current.NormalizedClass != shared.OSOKAsyncClassPending ||
		current.RawStatus != string(iotsdk.LifecycleStateActive) {
		t.Fatalf("status.status.async.current = %#v, want lifecycle delete pending ACTIVE", current)
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

func requireLastCondition(t *testing.T, resource *iotv1beta1.DigitalTwinModel, want shared.OSOKConditionType) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.status.conditions is empty, want trailing %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last condition = %s, want %s", got, want)
	}
}

func jsonValue(raw string) shared.JSONValue {
	return shared.JSONValue{Raw: []byte(raw)}
}

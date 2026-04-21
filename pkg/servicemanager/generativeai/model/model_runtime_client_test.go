/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package model

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	generativeaisdk "github.com/oracle/oci-go-sdk/v65/generativeai"
	generativeaiv1beta1 "github.com/oracle/oci-service-operator/api/generativeai/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeModelOCIClient struct {
	createFn func(context.Context, generativeaisdk.CreateModelRequest) (generativeaisdk.CreateModelResponse, error)
	getFn    func(context.Context, generativeaisdk.GetModelRequest) (generativeaisdk.GetModelResponse, error)
	listFn   func(context.Context, generativeaisdk.ListModelsRequest) (generativeaisdk.ListModelsResponse, error)
	updateFn func(context.Context, generativeaisdk.UpdateModelRequest) (generativeaisdk.UpdateModelResponse, error)
	deleteFn func(context.Context, generativeaisdk.DeleteModelRequest) (generativeaisdk.DeleteModelResponse, error)
}

func (f *fakeModelOCIClient) CreateModel(
	ctx context.Context,
	req generativeaisdk.CreateModelRequest,
) (generativeaisdk.CreateModelResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return generativeaisdk.CreateModelResponse{}, nil
}

func (f *fakeModelOCIClient) GetModel(
	ctx context.Context,
	req generativeaisdk.GetModelRequest,
) (generativeaisdk.GetModelResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return generativeaisdk.GetModelResponse{}, fakeModelServiceError{
		statusCode: 404,
		code:       "NotFound",
		message:    "missing",
	}
}

func (f *fakeModelOCIClient) ListModels(
	ctx context.Context,
	req generativeaisdk.ListModelsRequest,
) (generativeaisdk.ListModelsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return generativeaisdk.ListModelsResponse{}, nil
}

func (f *fakeModelOCIClient) UpdateModel(
	ctx context.Context,
	req generativeaisdk.UpdateModelRequest,
) (generativeaisdk.UpdateModelResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return generativeaisdk.UpdateModelResponse{}, nil
}

func (f *fakeModelOCIClient) DeleteModel(
	ctx context.Context,
	req generativeaisdk.DeleteModelRequest,
) (generativeaisdk.DeleteModelResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return generativeaisdk.DeleteModelResponse{}, nil
}

type fakeModelServiceError struct {
	statusCode int
	code       string
	message    string
}

func (f fakeModelServiceError) Error() string          { return f.message }
func (f fakeModelServiceError) GetHTTPStatusCode() int { return f.statusCode }
func (f fakeModelServiceError) GetMessage() string     { return f.message }
func (f fakeModelServiceError) GetCode() string        { return f.code }
func (f fakeModelServiceError) GetOpcRequestID() string {
	return ""
}

func newTestModelDelegate(client *fakeModelOCIClient) ModelServiceClient {
	if client == nil {
		client = &fakeModelOCIClient{}
	}

	hooks := newModelDefaultRuntimeHooks(generativeaisdk.GenerativeAiClient{})
	hooks.Create.Call = func(ctx context.Context, request generativeaisdk.CreateModelRequest) (generativeaisdk.CreateModelResponse, error) {
		return client.CreateModel(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request generativeaisdk.GetModelRequest) (generativeaisdk.GetModelResponse, error) {
		return client.GetModel(ctx, request)
	}
	hooks.List.Call = func(ctx context.Context, request generativeaisdk.ListModelsRequest) (generativeaisdk.ListModelsResponse, error) {
		return client.ListModels(ctx, request)
	}
	hooks.Update.Call = func(ctx context.Context, request generativeaisdk.UpdateModelRequest) (generativeaisdk.UpdateModelResponse, error) {
		return client.UpdateModel(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request generativeaisdk.DeleteModelRequest) (generativeaisdk.DeleteModelResponse, error) {
		return client.DeleteModel(ctx, request)
	}
	applyModelRuntimeHooks(&hooks)
	config := buildModelGeneratedRuntimeConfig(&ModelServiceManager{
		Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
	}, hooks)

	return defaultModelServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*generativeaiv1beta1.Model](config),
	}
}

func newTestModelClient(client *fakeModelOCIClient) ModelServiceClient {
	return newTestModelDelegate(client)
}

func makeSpecModel() *generativeaiv1beta1.Model {
	return &generativeaiv1beta1.Model{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "model-sample",
			Namespace: "default",
		},
		Spec: generativeaiv1beta1.ModelSpec{
			CompartmentId: "ocid1.compartment.oc1..exampleuniqueID",
			BaseModelId:   "ocid1.generativeaimodel.oc1..base",
			DisplayName:   "osok-model-sample",
			Description:   "OSOK Model sample",
			FineTuneDetails: generativeaiv1beta1.ModelFineTuneDetails{
				DedicatedAiClusterId: "ocid1.generativeaidedicatedaicluster.oc1..cluster",
				TrainingDataset: generativeaiv1beta1.ModelFineTuneDetailsTrainingDataset{
					DatasetType:   "OBJECT_STORAGE",
					NamespaceName: "object-storage-namespace",
					BucketName:    "generativeai-training-data",
					ObjectName:    "datasets/model-training.jsonl",
				},
			},
			FreeformTags: map[string]string{
				"managed-by": "osok",
			},
		},
	}
}

func makeSDKModel(
	id string,
	spec generativeaiv1beta1.ModelSpec,
	state generativeaisdk.ModelLifecycleStateEnum,
) generativeaisdk.Model {
	model := generativeaisdk.Model{
		Id:            common.String(id),
		CompartmentId: common.String(spec.CompartmentId),
		BaseModelId:   common.String(spec.BaseModelId),
		FineTuneDetails: &generativeaisdk.FineTuneDetails{
			DedicatedAiClusterId: common.String(spec.FineTuneDetails.DedicatedAiClusterId),
			TrainingDataset: generativeaisdk.ObjectStorageDataset{
				NamespaceName: common.String(spec.FineTuneDetails.TrainingDataset.NamespaceName),
				BucketName:    common.String(spec.FineTuneDetails.TrainingDataset.BucketName),
				ObjectName:    common.String(spec.FineTuneDetails.TrainingDataset.ObjectName),
			},
		},
		LifecycleState: state,
		Type:           generativeaisdk.ModelTypeCustom,
		Capabilities:   []generativeaisdk.ModelCapabilityEnum{},
		DefinedTags:    map[string]map[string]interface{}{},
		FreeformTags:   map[string]string{},
		SystemTags:     map[string]map[string]interface{}{},
	}
	if spec.DisplayName != "" {
		model.DisplayName = common.String(spec.DisplayName)
	}
	if spec.Vendor != "" {
		model.Vendor = common.String(spec.Vendor)
	}
	if spec.Version != "" {
		model.Version = common.String(spec.Version)
	}
	if spec.FreeformTags != nil {
		model.FreeformTags = spec.FreeformTags
	}
	return model
}

func makeSDKModelSummary(
	id string,
	spec generativeaiv1beta1.ModelSpec,
	state generativeaisdk.ModelLifecycleStateEnum,
) generativeaisdk.ModelSummary {
	model := generativeaisdk.ModelSummary{
		Id:            common.String(id),
		CompartmentId: common.String(spec.CompartmentId),
		BaseModelId:   common.String(spec.BaseModelId),
		FineTuneDetails: &generativeaisdk.FineTuneDetails{
			DedicatedAiClusterId: common.String(spec.FineTuneDetails.DedicatedAiClusterId),
			TrainingDataset: generativeaisdk.ObjectStorageDataset{
				NamespaceName: common.String(spec.FineTuneDetails.TrainingDataset.NamespaceName),
				BucketName:    common.String(spec.FineTuneDetails.TrainingDataset.BucketName),
				ObjectName:    common.String(spec.FineTuneDetails.TrainingDataset.ObjectName),
			},
		},
		LifecycleState: state,
		Type:           generativeaisdk.ModelTypeCustom,
		Capabilities:   []generativeaisdk.ModelCapabilityEnum{},
	}
	if spec.DisplayName != "" {
		model.DisplayName = common.String(spec.DisplayName)
	}
	if spec.Vendor != "" {
		model.Vendor = common.String(spec.Vendor)
	}
	if spec.Version != "" {
		model.Version = common.String(spec.Version)
	}
	if spec.FreeformTags != nil {
		model.FreeformTags = spec.FreeformTags
	}
	return model
}

func TestModelCreateOrUpdateSkipsReuseWhenDisplayNameMissing(t *testing.T) {
	t.Parallel()

	const createdID = "ocid1.generativeaimodel.oc1..created"

	resource := makeSpecModel()
	resource.Spec.DisplayName = ""

	createCalls := 0
	listCalls := 0
	getCalls := 0

	client := newTestModelClient(&fakeModelOCIClient{
		createFn: func(_ context.Context, req generativeaisdk.CreateModelRequest) (generativeaisdk.CreateModelResponse, error) {
			createCalls++
			if req.CompartmentId == nil || *req.CompartmentId != resource.Spec.CompartmentId {
				t.Fatalf("CreateModelRequest.CompartmentId = %v, want %q", req.CompartmentId, resource.Spec.CompartmentId)
			}
			if req.DisplayName != nil {
				t.Fatalf("CreateModelRequest.DisplayName = %v, want nil when spec.displayName is empty", req.DisplayName)
			}
			if req.BaseModelId == nil || *req.BaseModelId != resource.Spec.BaseModelId {
				t.Fatalf("CreateModelRequest.BaseModelId = %v, want %q", req.BaseModelId, resource.Spec.BaseModelId)
			}
			return generativeaisdk.CreateModelResponse{
				Model:        makeSDKModel(createdID, resource.Spec, generativeaisdk.ModelLifecycleStateCreating),
				OpcRequestId: common.String("opc-create-1"),
			}, nil
		},
		getFn: func(_ context.Context, req generativeaisdk.GetModelRequest) (generativeaisdk.GetModelResponse, error) {
			getCalls++
			if req.ModelId == nil || *req.ModelId != createdID {
				t.Fatalf("GetModelRequest.ModelId = %v, want %q", req.ModelId, createdID)
			}
			return generativeaisdk.GetModelResponse{
				Model: makeSDKModel(createdID, resource.Spec, generativeaisdk.ModelLifecycleStateActive),
			}, nil
		},
		listFn: func(context.Context, generativeaisdk.ListModelsRequest) (generativeaisdk.ListModelsResponse, error) {
			listCalls++
			return generativeaisdk.ListModelsResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want success", response)
	}
	if response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want no requeue after ACTIVE follow-up", response)
	}
	if createCalls != 1 {
		t.Fatalf("CreateModel() calls = %d, want 1", createCalls)
	}
	if listCalls != 0 {
		t.Fatalf("ListModels() calls = %d, want 0 when displayName is empty", listCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetModel() calls = %d, want 1", getCalls)
	}
	if got := resource.Status.Id; got != createdID {
		t.Fatalf("status.id = %q, want %q", got, createdID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != createdID {
		t.Fatalf("status.status.ocid = %q, want %q", got, createdID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", got, "opc-create-1")
	}
	if got := resource.Status.LifecycleState; got != "ACTIVE" {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
}

func TestModelCreateOrUpdateClearsStaleTrackedIDWhenDisplayNameMissing(t *testing.T) {
	t.Parallel()

	const (
		staleID = "ocid1.generativeaimodel.oc1..stale"
		newID   = "ocid1.generativeaimodel.oc1..new"
	)

	resource := makeSpecModel()
	resource.Spec.DisplayName = ""
	resource.Status.Id = staleID
	resource.Status.OsokStatus.Ocid = shared.OCID(staleID)

	createCalls := 0
	listCalls := 0
	getIDs := make([]string, 0, 2)

	client := newTestModelClient(&fakeModelOCIClient{
		createFn: func(_ context.Context, req generativeaisdk.CreateModelRequest) (generativeaisdk.CreateModelResponse, error) {
			createCalls++
			if req.DisplayName != nil {
				t.Fatalf("CreateModelRequest.DisplayName = %v, want nil when spec.displayName is empty", req.DisplayName)
			}
			return generativeaisdk.CreateModelResponse{
				Model: makeSDKModel(newID, resource.Spec, generativeaisdk.ModelLifecycleStateCreating),
			}, nil
		},
		getFn: func(_ context.Context, req generativeaisdk.GetModelRequest) (generativeaisdk.GetModelResponse, error) {
			gotID := ""
			if req.ModelId != nil {
				gotID = *req.ModelId
			}
			getIDs = append(getIDs, gotID)
			switch gotID {
			case staleID:
				return generativeaisdk.GetModelResponse{}, fakeModelServiceError{
					statusCode: 404,
					code:       "NotFound",
					message:    "missing",
				}
			case newID:
				return generativeaisdk.GetModelResponse{
					Model: makeSDKModel(newID, resource.Spec, generativeaisdk.ModelLifecycleStateActive),
				}, nil
			default:
				t.Fatalf("unexpected GetModelRequest.ModelId %q", gotID)
				return generativeaisdk.GetModelResponse{}, nil
			}
		},
		listFn: func(context.Context, generativeaisdk.ListModelsRequest) (generativeaisdk.ListModelsResponse, error) {
			listCalls++
			return generativeaisdk.ListModelsResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want success", response)
	}
	if createCalls != 1 {
		t.Fatalf("CreateModel() calls = %d, want 1 after clearing stale ID", createCalls)
	}
	if listCalls != 0 {
		t.Fatalf("ListModels() calls = %d, want 0 when displayName is empty", listCalls)
	}
	if len(getIDs) != 2 || getIDs[0] != staleID || getIDs[1] != newID {
		t.Fatalf("GetModel() ids = %v, want [%q %q]", getIDs, staleID, newID)
	}
	if got := resource.Status.Id; got != newID {
		t.Fatalf("status.id = %q, want %q", got, newID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != newID {
		t.Fatalf("status.status.ocid = %q, want %q", got, newID)
	}
}

func TestModelCreateOrUpdateBindsExistingResourceByDisplayName(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.generativeaimodel.oc1..existing"

	resource := makeSpecModel()
	resource.Spec.Description = ""
	createCalled := false
	updateCalled := false
	listCalls := 0
	getCalls := 0

	mismatchSpec := resource.Spec
	mismatchSpec.BaseModelId = "ocid1.generativeaimodel.oc1..other-base"

	client := newTestModelClient(&fakeModelOCIClient{
		createFn: func(context.Context, generativeaisdk.CreateModelRequest) (generativeaisdk.CreateModelResponse, error) {
			createCalled = true
			return generativeaisdk.CreateModelResponse{}, nil
		},
		getFn: func(_ context.Context, req generativeaisdk.GetModelRequest) (generativeaisdk.GetModelResponse, error) {
			getCalls++
			if req.ModelId == nil || *req.ModelId != existingID {
				t.Fatalf("GetModelRequest.ModelId = %v, want %q", req.ModelId, existingID)
			}
			return generativeaisdk.GetModelResponse{
				Model: makeSDKModel(existingID, resource.Spec, generativeaisdk.ModelLifecycleStateActive),
			}, nil
		},
		listFn: func(_ context.Context, req generativeaisdk.ListModelsRequest) (generativeaisdk.ListModelsResponse, error) {
			listCalls++
			if req.CompartmentId == nil || *req.CompartmentId != resource.Spec.CompartmentId {
				t.Fatalf("ListModelsRequest.CompartmentId = %v, want %q", req.CompartmentId, resource.Spec.CompartmentId)
			}
			if req.DisplayName == nil || *req.DisplayName != resource.Spec.DisplayName {
				t.Fatalf("ListModelsRequest.DisplayName = %v, want %q", req.DisplayName, resource.Spec.DisplayName)
			}
			return generativeaisdk.ListModelsResponse{
				ModelCollection: generativeaisdk.ModelCollection{
					Items: []generativeaisdk.ModelSummary{
						makeSDKModelSummary("ocid1.generativeaimodel.oc1..other", mismatchSpec, generativeaisdk.ModelLifecycleStateActive),
						makeSDKModelSummary(existingID, resource.Spec, generativeaisdk.ModelLifecycleStateActive),
					},
				},
			}, nil
		},
		updateFn: func(context.Context, generativeaisdk.UpdateModelRequest) (generativeaisdk.UpdateModelResponse, error) {
			updateCalled = true
			return generativeaisdk.UpdateModelResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want success", response)
	}
	if response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want no requeue for ACTIVE bind", response)
	}
	if createCalled {
		t.Fatal("CreateModel() called, want displayName bind path")
	}
	if updateCalled {
		t.Fatal("UpdateModel() called, want observe-only bind path")
	}
	if listCalls != 1 {
		t.Fatalf("ListModels() calls = %d, want 1", listCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetModel() calls = %d, want 1", getCalls)
	}
	if got := resource.Status.Id; got != existingID {
		t.Fatalf("status.id = %q, want %q", got, existingID)
	}
}

func TestModelCreateOrUpdateUpdatesMutableDescription(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.generativeaimodel.oc1..existing"

	var updateRequest generativeaisdk.UpdateModelRequest
	getCalls := 0
	updateCalls := 0

	client := newTestModelClient(&fakeModelOCIClient{
		getFn: func(_ context.Context, req generativeaisdk.GetModelRequest) (generativeaisdk.GetModelResponse, error) {
			getCalls++
			if req.ModelId == nil || *req.ModelId != existingID {
				t.Fatalf("GetModelRequest.ModelId = %v, want %q", req.ModelId, existingID)
			}

			spec := makeSpecModel().Spec
			if getCalls == 1 {
				spec.Description = "old description"
			}
			return generativeaisdk.GetModelResponse{
				Model: makeSDKModel(existingID, spec, generativeaisdk.ModelLifecycleStateActive),
			}, nil
		},
		updateFn: func(_ context.Context, req generativeaisdk.UpdateModelRequest) (generativeaisdk.UpdateModelResponse, error) {
			updateCalls++
			updateRequest = req
			return generativeaisdk.UpdateModelResponse{
				Model:        makeSDKModel(existingID, makeSpecModel().Spec, generativeaisdk.ModelLifecycleStateActive),
				OpcRequestId: common.String("opc-update-1"),
			}, nil
		},
	})

	resource := makeSpecModel()
	resource.Status.Id = existingID
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success after updating mutable description")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue once update follow-up GetModel reports ACTIVE")
	}
	if updateCalls != 1 {
		t.Fatalf("UpdateModel() calls = %d, want 1", updateCalls)
	}
	if getCalls != 2 {
		t.Fatalf("GetModel() calls = %d, want 2 (observe + follow-up)", getCalls)
	}
	if updateRequest.ModelId == nil || *updateRequest.ModelId != existingID {
		t.Fatalf("update modelId = %v, want %q", updateRequest.ModelId, existingID)
	}
	if updateRequest.UpdateModelDetails.Description == nil || *updateRequest.UpdateModelDetails.Description != resource.Spec.Description {
		t.Fatalf("update description = %v, want %q", updateRequest.UpdateModelDetails.Description, resource.Spec.Description)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", got, "opc-update-1")
	}
}

func TestModelCreateOrUpdateRejectsReplacementOnlyBaseModelDrift(t *testing.T) {
	t.Parallel()

	updateCalled := false
	client := newTestModelClient(&fakeModelOCIClient{
		getFn: func(_ context.Context, _ generativeaisdk.GetModelRequest) (generativeaisdk.GetModelResponse, error) {
			return generativeaisdk.GetModelResponse{
				Model: makeSDKModel("ocid1.generativeaimodel.oc1..existing", makeSpecModel().Spec, generativeaisdk.ModelLifecycleStateActive),
			}, nil
		},
		updateFn: func(_ context.Context, _ generativeaisdk.UpdateModelRequest) (generativeaisdk.UpdateModelResponse, error) {
			updateCalled = true
			return generativeaisdk.UpdateModelResponse{}, nil
		},
	})

	resource := makeSpecModel()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.generativeaimodel.oc1..existing")
	resource.Spec.BaseModelId = "ocid1.generativeaimodel.oc1..replacement"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "require replacement when baseModelId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want replacement-required drift failure", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful response", response)
	}
	if updateCalled {
		t.Fatal("UpdateModel() should not be called when replacement-only drift is detected")
	}
}

func TestModelDeleteConfirmsDeletion(t *testing.T) {
	t.Parallel()

	var deleteRequest generativeaisdk.DeleteModelRequest
	getCalls := 0

	client := newTestModelClient(&fakeModelOCIClient{
		getFn: func(_ context.Context, req generativeaisdk.GetModelRequest) (generativeaisdk.GetModelResponse, error) {
			getCalls++
			if req.ModelId == nil || *req.ModelId != "ocid1.generativeaimodel.oc1..existing" {
				t.Fatalf("GetModelRequest.ModelId = %v, want existing model OCID", req.ModelId)
			}

			state := generativeaisdk.ModelLifecycleStateActive
			if getCalls > 1 {
				state = generativeaisdk.ModelLifecycleStateDeleted
			}
			return generativeaisdk.GetModelResponse{
				Model: makeSDKModel("ocid1.generativeaimodel.oc1..existing", makeSpecModel().Spec, state),
			}, nil
		},
		deleteFn: func(_ context.Context, req generativeaisdk.DeleteModelRequest) (generativeaisdk.DeleteModelResponse, error) {
			deleteRequest = req
			return generativeaisdk.DeleteModelResponse{
				OpcRequestId: common.String("opc-delete-1"),
			}, nil
		},
	})

	resource := makeSpecModel()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.generativeaimodel.oc1..existing")

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() should report success once follow-up GetModel confirms DELETED")
	}
	if getCalls != 2 {
		t.Fatalf("GetModel() calls = %d, want 2 (preflight + confirmation)", getCalls)
	}
	if deleteRequest.ModelId == nil || *deleteRequest.ModelId != "ocid1.generativeaimodel.oc1..existing" {
		t.Fatalf("delete modelId = %v, want existing model OCID", deleteRequest.ModelId)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete confirmation timestamp")
	}
	if got := resource.Status.LifecycleState; got != string(generativeaisdk.ModelLifecycleStateDeleted) {
		t.Fatalf("status.lifecycleState = %q, want DELETED", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", got, "opc-delete-1")
	}
}

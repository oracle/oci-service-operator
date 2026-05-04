/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package namedcredential

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	managementagentsdk "github.com/oracle/oci-go-sdk/v65/managementagent"
	managementagentv1beta1 "github.com/oracle/oci-service-operator/api/managementagent/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testNamedCredentialID                = "ocid1.namedcredential.oc1..example"
	testNamedCredentialManagementAgentID = "ocid1.managementagent.oc1..agent"
	testNamedCredentialName              = "db-credential"
	testNamedCredentialType              = "DB"
)

type fakeNamedCredentialOCIClient struct {
	createFn      func(context.Context, managementagentsdk.CreateNamedCredentialRequest) (managementagentsdk.CreateNamedCredentialResponse, error)
	getFn         func(context.Context, managementagentsdk.GetNamedCredentialRequest) (managementagentsdk.GetNamedCredentialResponse, error)
	listFn        func(context.Context, managementagentsdk.ListNamedCredentialsRequest) (managementagentsdk.ListNamedCredentialsResponse, error)
	updateFn      func(context.Context, managementagentsdk.UpdateNamedCredentialRequest) (managementagentsdk.UpdateNamedCredentialResponse, error)
	deleteFn      func(context.Context, managementagentsdk.DeleteNamedCredentialRequest) (managementagentsdk.DeleteNamedCredentialResponse, error)
	workRequestFn func(context.Context, managementagentsdk.GetWorkRequestRequest) (managementagentsdk.GetWorkRequestResponse, error)
}

func (f *fakeNamedCredentialOCIClient) CreateNamedCredential(ctx context.Context, req managementagentsdk.CreateNamedCredentialRequest) (managementagentsdk.CreateNamedCredentialResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return managementagentsdk.CreateNamedCredentialResponse{}, nil
}

func (f *fakeNamedCredentialOCIClient) GetNamedCredential(ctx context.Context, req managementagentsdk.GetNamedCredentialRequest) (managementagentsdk.GetNamedCredentialResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return managementagentsdk.GetNamedCredentialResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "named credential is missing")
}

func (f *fakeNamedCredentialOCIClient) ListNamedCredentials(ctx context.Context, req managementagentsdk.ListNamedCredentialsRequest) (managementagentsdk.ListNamedCredentialsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return managementagentsdk.ListNamedCredentialsResponse{}, nil
}

func (f *fakeNamedCredentialOCIClient) UpdateNamedCredential(ctx context.Context, req managementagentsdk.UpdateNamedCredentialRequest) (managementagentsdk.UpdateNamedCredentialResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return managementagentsdk.UpdateNamedCredentialResponse{}, nil
}

func (f *fakeNamedCredentialOCIClient) DeleteNamedCredential(ctx context.Context, req managementagentsdk.DeleteNamedCredentialRequest) (managementagentsdk.DeleteNamedCredentialResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return managementagentsdk.DeleteNamedCredentialResponse{}, nil
}

func (f *fakeNamedCredentialOCIClient) GetWorkRequest(ctx context.Context, req managementagentsdk.GetWorkRequestRequest) (managementagentsdk.GetWorkRequestResponse, error) {
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, req)
	}
	return managementagentsdk.GetWorkRequestResponse{}, nil
}

func newTestNamedCredentialClient(client namedCredentialOCIClient) NamedCredentialServiceClient {
	return newNamedCredentialServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
		client,
	)
}

func makeNamedCredentialResource() *managementagentv1beta1.NamedCredential {
	return &managementagentv1beta1.NamedCredential{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testNamedCredentialName,
			Namespace: "default",
		},
		Spec: managementagentv1beta1.NamedCredentialSpec{
			Name:              testNamedCredentialName,
			Type:              testNamedCredentialType,
			ManagementAgentId: testNamedCredentialManagementAgentID,
			Properties: []managementagentv1beta1.NamedCredentialProperty{
				{
					Name:          "username",
					Value:         "app-user",
					ValueCategory: string(managementagentsdk.ValueCategoryTypeClearText),
				},
				{
					Name:          "password",
					Value:         "ocid1.vaultsecret.oc1..password",
					ValueCategory: string(managementagentsdk.ValueCategoryTypeSecretIdentifier),
				},
			},
			Description:  "initial description",
			FreeformTags: map[string]string{"env": "test"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {
					"CostCenter": "42",
				},
			},
		},
	}
}

func makeNamedCredentialRequest(resource *managementagentv1beta1.NamedCredential) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		},
	}
}

func makeSDKNamedCredential(
	id string,
	spec managementagentv1beta1.NamedCredentialSpec,
	state managementagentsdk.NamedCredentialLifecycleStateEnum,
) managementagentsdk.NamedCredential {
	return managementagentsdk.NamedCredential{
		Id:                common.String(id),
		Name:              common.String(spec.Name),
		Type:              common.String(spec.Type),
		ManagementAgentId: common.String(spec.ManagementAgentId),
		Properties:        namedCredentialSDKProperties(spec.Properties),
		Description:       common.String(spec.Description),
		FreeformTags:      cloneNamedCredentialStringMap(spec.FreeformTags),
		DefinedTags:       namedCredentialDefinedTags(spec.DefinedTags),
		LifecycleState:    state,
	}
}

func makeSDKNamedCredentialSummary(
	id string,
	spec managementagentv1beta1.NamedCredentialSpec,
	state managementagentsdk.NamedCredentialLifecycleStateEnum,
) managementagentsdk.NamedCredentialSummary {
	return managementagentsdk.NamedCredentialSummary{
		Id:                common.String(id),
		Name:              common.String(spec.Name),
		Type:              common.String(spec.Type),
		ManagementAgentId: common.String(spec.ManagementAgentId),
		Properties:        namedCredentialSDKProperties(spec.Properties),
		Description:       common.String(spec.Description),
		FreeformTags:      cloneNamedCredentialStringMap(spec.FreeformTags),
		DefinedTags:       namedCredentialDefinedTags(spec.DefinedTags),
		LifecycleState:    state,
	}
}

func makeNamedCredentialWorkRequest(
	id string,
	operation managementagentsdk.OperationTypesEnum,
	status managementagentsdk.OperationStatusEnum,
	action managementagentsdk.ActionTypesEnum,
	resourceID string,
) managementagentsdk.WorkRequest {
	return managementagentsdk.WorkRequest{
		Id:            common.String(id),
		OperationType: operation,
		Status:        status,
		Resources: []managementagentsdk.WorkRequestResource{
			{
				EntityType: common.String("NamedCredential"),
				ActionType: action,
				Identifier: common.String(resourceID),
				EntityUri:  common.String("/namedCredentials/" + resourceID),
			},
		},
	}
}

func TestNamedCredentialRuntimeSemanticsEncodesWorkRequestContract(t *testing.T) {
	t.Parallel()

	got := newNamedCredentialRuntimeSemantics()
	if got == nil {
		t.Fatal("newNamedCredentialRuntimeSemantics() = nil")
	}
	requireNamedCredentialAsyncSemantics(t, got)
	requireNamedCredentialLifecycleSemantics(t, got)
	requireNamedCredentialMutationSemantics(t, got)
}

func requireNamedCredentialAsyncSemantics(t *testing.T, got *generatedruntime.Semantics) {
	t.Helper()
	if got.Async == nil || got.Async.Strategy != "workrequest" || got.Async.Runtime != "generatedruntime" {
		t.Fatalf("Async = %#v, want generatedruntime workrequest", got.Async)
	}
	if got.Async.WorkRequest == nil || got.Async.WorkRequest.Source != "service-sdk" {
		t.Fatalf("Async.WorkRequest = %#v, want service-sdk work request contract", got.Async.WorkRequest)
	}
	assertNamedCredentialStringSliceEqual(t, "Async.WorkRequest.Phases", got.Async.WorkRequest.Phases, []string{"create", "update", "delete"})
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" || got.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("delete semantics = %#v followUp=%#v, want required confirm-delete", got.Delete, got.DeleteFollowUp)
	}
}

func requireNamedCredentialLifecycleSemantics(t *testing.T, got *generatedruntime.Semantics) {
	t.Helper()
	assertNamedCredentialStringSliceEqual(t, "Lifecycle.ProvisioningStates", got.Lifecycle.ProvisioningStates, []string{"CREATING"})
	assertNamedCredentialStringSliceEqual(t, "Lifecycle.UpdatingStates", got.Lifecycle.UpdatingStates, []string{"UPDATING"})
	assertNamedCredentialStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE"})
	assertNamedCredentialStringSliceEqual(t, "Delete.PendingStates", got.Delete.PendingStates, []string{"DELETING"})
	assertNamedCredentialStringSliceEqual(t, "Delete.TerminalStates", got.Delete.TerminalStates, []string{"DELETED"})
	assertNamedCredentialStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"managementAgentId", "name"})
}

func requireNamedCredentialMutationSemantics(t *testing.T, got *generatedruntime.Semantics) {
	t.Helper()
	assertNamedCredentialStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"properties", "description", "freeformTags", "definedTags"})
	assertNamedCredentialStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"managementAgentId", "name", "type"})
}

func TestNamedCredentialCreateThroughWorkRequestProjectsStatus(t *testing.T) {
	t.Parallel()

	resource := makeNamedCredentialResource()
	var createRequest managementagentsdk.CreateNamedCredentialRequest
	listCalls := 0
	createCalls := 0
	workRequestCalls := 0
	getCalls := 0

	client := newTestNamedCredentialClient(&fakeNamedCredentialOCIClient{
		listFn: func(_ context.Context, req managementagentsdk.ListNamedCredentialsRequest) (managementagentsdk.ListNamedCredentialsResponse, error) {
			listCalls++
			requireNamedCredentialStringPtr(t, "ListNamedCredentialsRequest.ManagementAgentId", req.ManagementAgentId, resource.Spec.ManagementAgentId)
			if len(req.Name) != 0 || len(req.Type) != 0 {
				t.Fatalf("ListNamedCredentialsRequest name/type filters = %v/%v, want empty slices so generatedruntime can match locally", req.Name, req.Type)
			}
			return managementagentsdk.ListNamedCredentialsResponse{}, nil
		},
		createFn: func(_ context.Context, req managementagentsdk.CreateNamedCredentialRequest) (managementagentsdk.CreateNamedCredentialResponse, error) {
			createCalls++
			createRequest = req
			return managementagentsdk.CreateNamedCredentialResponse{
				NamedCredential:  makeSDKNamedCredential(testNamedCredentialID, resource.Spec, managementagentsdk.NamedCredentialLifecycleStateCreating),
				OpcWorkRequestId: common.String("wr-create"),
				OpcRequestId:     common.String("opc-create"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req managementagentsdk.GetWorkRequestRequest) (managementagentsdk.GetWorkRequestResponse, error) {
			workRequestCalls++
			requireNamedCredentialStringPtr(t, "GetWorkRequestRequest.WorkRequestId", req.WorkRequestId, "wr-create")
			return managementagentsdk.GetWorkRequestResponse{
				WorkRequest: makeNamedCredentialWorkRequest(
					"wr-create",
					managementagentsdk.OperationTypesCreateNamedcredentials,
					managementagentsdk.OperationStatusSucceeded,
					managementagentsdk.ActionTypesCreated,
					testNamedCredentialID,
				),
			}, nil
		},
		getFn: func(_ context.Context, req managementagentsdk.GetNamedCredentialRequest) (managementagentsdk.GetNamedCredentialResponse, error) {
			getCalls++
			requireNamedCredentialStringPtr(t, "GetNamedCredentialRequest.NamedCredentialId", req.NamedCredentialId, testNamedCredentialID)
			return managementagentsdk.GetNamedCredentialResponse{
				NamedCredential: makeSDKNamedCredential(testNamedCredentialID, resource.Spec, managementagentsdk.NamedCredentialLifecycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeNamedCredentialRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	requireNamedCredentialSuccessfulResponse(t, "CreateOrUpdate()", response)
	if listCalls != 1 || createCalls != 1 || workRequestCalls != 1 || getCalls != 1 {
		t.Fatalf("call counts list/create/workRequest/get = %d/%d/%d/%d, want 1/1/1/1", listCalls, createCalls, workRequestCalls, getCalls)
	}
	requireNamedCredentialCreateRequest(t, createRequest, resource)
	requireNamedCredentialActiveStatus(t, resource, "opc-create")
}

func TestNamedCredentialCreateOrUpdateBindsExistingFromPagedList(t *testing.T) {
	t.Parallel()

	resource := makeNamedCredentialResource()
	listCalls := 0
	getCalls := 0
	createCalled := false
	updateCalled := false

	client := newTestNamedCredentialClient(&fakeNamedCredentialOCIClient{
		listFn: func(_ context.Context, req managementagentsdk.ListNamedCredentialsRequest) (managementagentsdk.ListNamedCredentialsResponse, error) {
			listCalls++
			requireNamedCredentialStringPtr(t, "ListNamedCredentialsRequest.ManagementAgentId", req.ManagementAgentId, resource.Spec.ManagementAgentId)
			switch listCalls {
			case 1:
				if req.Page != nil {
					t.Fatalf("first ListNamedCredentialsRequest.Page = %q, want nil", *req.Page)
				}
				return managementagentsdk.ListNamedCredentialsResponse{
					NamedCredentialCollection: managementagentsdk.NamedCredentialCollection{
						Items: []managementagentsdk.NamedCredentialSummary{
							makeSDKNamedCredentialSummary("ocid1.namedcredential.oc1..other", managementagentv1beta1.NamedCredentialSpec{
								Name:              "other",
								Type:              resource.Spec.Type,
								ManagementAgentId: resource.Spec.ManagementAgentId,
								Properties:        resource.Spec.Properties,
							}, managementagentsdk.NamedCredentialLifecycleStateActive),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			default:
				requireNamedCredentialStringPtr(t, "second ListNamedCredentialsRequest.Page", req.Page, "page-2")
				return managementagentsdk.ListNamedCredentialsResponse{
					NamedCredentialCollection: managementagentsdk.NamedCredentialCollection{
						Items: []managementagentsdk.NamedCredentialSummary{
							makeSDKNamedCredentialSummary(testNamedCredentialID, resource.Spec, managementagentsdk.NamedCredentialLifecycleStateActive),
						},
					},
				}, nil
			}
		},
		getFn: func(_ context.Context, req managementagentsdk.GetNamedCredentialRequest) (managementagentsdk.GetNamedCredentialResponse, error) {
			getCalls++
			requireNamedCredentialStringPtr(t, "GetNamedCredentialRequest.NamedCredentialId", req.NamedCredentialId, testNamedCredentialID)
			return managementagentsdk.GetNamedCredentialResponse{
				NamedCredential: makeSDKNamedCredential(testNamedCredentialID, resource.Spec, managementagentsdk.NamedCredentialLifecycleStateActive),
			}, nil
		},
		createFn: func(context.Context, managementagentsdk.CreateNamedCredentialRequest) (managementagentsdk.CreateNamedCredentialResponse, error) {
			createCalled = true
			return managementagentsdk.CreateNamedCredentialResponse{}, nil
		},
		updateFn: func(context.Context, managementagentsdk.UpdateNamedCredentialRequest) (managementagentsdk.UpdateNamedCredentialResponse, error) {
			updateCalled = true
			return managementagentsdk.UpdateNamedCredentialResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeNamedCredentialRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	requireNamedCredentialSuccessfulResponse(t, "CreateOrUpdate()", response)
	if createCalled {
		t.Fatal("CreateNamedCredential() called for existing named credential")
	}
	if updateCalled {
		t.Fatal("UpdateNamedCredential() called for matching named credential")
	}
	if listCalls != 2 || getCalls != 1 {
		t.Fatalf("call counts list/get = %d/%d, want 2/1", listCalls, getCalls)
	}
	requireNamedCredentialActiveStatus(t, resource, "")
}

func TestNamedCredentialCreateOrUpdateRejectsTypeDriftAfterNameBind(t *testing.T) {
	t.Parallel()

	resource := makeNamedCredentialResource()
	currentSpec := resource.Spec
	currentSpec.Type = "SSH"
	listCalls := 0
	getCalls := 0
	createCalled := false
	updateCalled := false

	client := newTestNamedCredentialClient(&fakeNamedCredentialOCIClient{
		listFn: func(_ context.Context, req managementagentsdk.ListNamedCredentialsRequest) (managementagentsdk.ListNamedCredentialsResponse, error) {
			listCalls++
			requireNamedCredentialStringPtr(t, "ListNamedCredentialsRequest.ManagementAgentId", req.ManagementAgentId, resource.Spec.ManagementAgentId)
			if len(req.Name) != 0 || len(req.Type) != 0 {
				t.Fatalf("ListNamedCredentialsRequest name/type filters = %v/%v, want empty slices so type drift cannot hide an existing name", req.Name, req.Type)
			}
			return managementagentsdk.ListNamedCredentialsResponse{
				NamedCredentialCollection: managementagentsdk.NamedCredentialCollection{
					Items: []managementagentsdk.NamedCredentialSummary{
						makeSDKNamedCredentialSummary(testNamedCredentialID, currentSpec, managementagentsdk.NamedCredentialLifecycleStateActive),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, req managementagentsdk.GetNamedCredentialRequest) (managementagentsdk.GetNamedCredentialResponse, error) {
			getCalls++
			requireNamedCredentialStringPtr(t, "GetNamedCredentialRequest.NamedCredentialId", req.NamedCredentialId, testNamedCredentialID)
			return managementagentsdk.GetNamedCredentialResponse{
				NamedCredential: makeSDKNamedCredential(testNamedCredentialID, currentSpec, managementagentsdk.NamedCredentialLifecycleStateActive),
			}, nil
		},
		createFn: func(context.Context, managementagentsdk.CreateNamedCredentialRequest) (managementagentsdk.CreateNamedCredentialResponse, error) {
			createCalled = true
			return managementagentsdk.CreateNamedCredentialResponse{}, nil
		},
		updateFn: func(context.Context, managementagentsdk.UpdateNamedCredentialRequest) (managementagentsdk.UpdateNamedCredentialResponse, error) {
			updateCalled = true
			return managementagentsdk.UpdateNamedCredentialResponse{}, nil
		},
	})

	_, err := client.CreateOrUpdate(context.Background(), resource, makeNamedCredentialRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only type drift rejection")
	}
	if createCalled {
		t.Fatal("CreateNamedCredential() called after same-name existing credential was listed")
	}
	if updateCalled {
		t.Fatal("UpdateNamedCredential() called after create-only type drift")
	}
	if listCalls != 1 || getCalls != 1 {
		t.Fatalf("call counts list/get = %d/%d, want 1/1", listCalls, getCalls)
	}
	if !strings.Contains(err.Error(), "replacement when type changes") {
		t.Fatalf("CreateOrUpdate() error = %q, want type replacement message", err.Error())
	}
}

func TestNamedCredentialCreateOrUpdateUpdatesMutableFields(t *testing.T) {
	t.Parallel()

	resource := makeNamedCredentialResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testNamedCredentialID)
	currentSpec := resource.Spec
	resource.Spec.Properties = []managementagentv1beta1.NamedCredentialProperty{
		{
			Name:          "username",
			Value:         "app-user",
			ValueCategory: string(managementagentsdk.ValueCategoryTypeClearText),
		},
		{
			Name:          "password",
			Value:         "ocid1.vaultsecret.oc1..rotated",
			ValueCategory: string(managementagentsdk.ValueCategoryTypeSecretIdentifier),
		},
	}
	resource.Spec.Description = "updated description"
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "84"}}

	getCalls := 0
	updateCalls := 0
	workRequestCalls := 0
	var updateRequest managementagentsdk.UpdateNamedCredentialRequest

	client := newTestNamedCredentialClient(&fakeNamedCredentialOCIClient{
		getFn: func(_ context.Context, req managementagentsdk.GetNamedCredentialRequest) (managementagentsdk.GetNamedCredentialResponse, error) {
			getCalls++
			requireNamedCredentialStringPtr(t, "GetNamedCredentialRequest.NamedCredentialId", req.NamedCredentialId, testNamedCredentialID)
			if getCalls == 1 {
				return managementagentsdk.GetNamedCredentialResponse{
					NamedCredential: makeSDKNamedCredential(testNamedCredentialID, currentSpec, managementagentsdk.NamedCredentialLifecycleStateActive),
				}, nil
			}
			return managementagentsdk.GetNamedCredentialResponse{
				NamedCredential: makeSDKNamedCredential(testNamedCredentialID, resource.Spec, managementagentsdk.NamedCredentialLifecycleStateActive),
			}, nil
		},
		updateFn: func(_ context.Context, req managementagentsdk.UpdateNamedCredentialRequest) (managementagentsdk.UpdateNamedCredentialResponse, error) {
			updateCalls++
			updateRequest = req
			return managementagentsdk.UpdateNamedCredentialResponse{
				NamedCredential:  makeSDKNamedCredential(testNamedCredentialID, resource.Spec, managementagentsdk.NamedCredentialLifecycleStateUpdating),
				OpcWorkRequestId: common.String("wr-update"),
				OpcRequestId:     common.String("opc-update"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req managementagentsdk.GetWorkRequestRequest) (managementagentsdk.GetWorkRequestResponse, error) {
			workRequestCalls++
			requireNamedCredentialStringPtr(t, "GetWorkRequestRequest.WorkRequestId", req.WorkRequestId, "wr-update")
			return managementagentsdk.GetWorkRequestResponse{
				WorkRequest: makeNamedCredentialWorkRequest(
					"wr-update",
					managementagentsdk.OperationTypesUpdateNamedcredentials,
					managementagentsdk.OperationStatusSucceeded,
					managementagentsdk.ActionTypesUpdated,
					testNamedCredentialID,
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeNamedCredentialRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	requireNamedCredentialSuccessfulResponse(t, "CreateOrUpdate()", response)
	if getCalls != 2 || updateCalls != 1 || workRequestCalls != 1 {
		t.Fatalf("call counts get/update/workRequest = %d/%d/%d, want 2/1/1", getCalls, updateCalls, workRequestCalls)
	}
	requireNamedCredentialStringPtr(t, "UpdateNamedCredentialRequest.NamedCredentialId", updateRequest.NamedCredentialId, testNamedCredentialID)
	requireNamedCredentialProperties(t, "UpdateNamedCredentialRequest.Properties", updateRequest.Properties, resource.Spec.Properties)
	requireNamedCredentialStringPtr(t, "UpdateNamedCredentialRequest.Description", updateRequest.Description, resource.Spec.Description)
	if got := updateRequest.FreeformTags["env"]; got != "prod" {
		t.Fatalf("UpdateNamedCredentialRequest.FreeformTags[env] = %q, want prod", got)
	}
	if got := updateRequest.DefinedTags["Operations"]["CostCenter"]; got != "84" {
		t.Fatalf("UpdateNamedCredentialRequest.DefinedTags[Operations][CostCenter] = %#v, want 84", got)
	}
	requireNamedCredentialActiveStatus(t, resource, "opc-update")
}

func TestNamedCredentialCreateOrUpdateRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := makeNamedCredentialResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testNamedCredentialID)
	currentSpec := resource.Spec
	resource.Spec.Type = "SSH"
	updateCalled := false

	client := newTestNamedCredentialClient(&fakeNamedCredentialOCIClient{
		getFn: func(_ context.Context, req managementagentsdk.GetNamedCredentialRequest) (managementagentsdk.GetNamedCredentialResponse, error) {
			requireNamedCredentialStringPtr(t, "GetNamedCredentialRequest.NamedCredentialId", req.NamedCredentialId, testNamedCredentialID)
			return managementagentsdk.GetNamedCredentialResponse{
				NamedCredential: makeSDKNamedCredential(testNamedCredentialID, currentSpec, managementagentsdk.NamedCredentialLifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, managementagentsdk.UpdateNamedCredentialRequest) (managementagentsdk.UpdateNamedCredentialResponse, error) {
			updateCalled = true
			return managementagentsdk.UpdateNamedCredentialResponse{}, nil
		},
	})

	_, err := client.CreateOrUpdate(context.Background(), resource, makeNamedCredentialRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if updateCalled {
		t.Fatal("UpdateNamedCredential() called after create-only type drift")
	}
	if !strings.Contains(err.Error(), "replacement when type changes") {
		t.Fatalf("CreateOrUpdate() error = %q, want type replacement message", err.Error())
	}
}

func TestNamedCredentialDeleteWaitsForPendingWriteWorkRequest(t *testing.T) {
	t.Parallel()

	resource := makeNamedCredentialResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testNamedCredentialID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:        shared.OSOKAsyncSourceWorkRequest,
		Phase:         shared.OSOKAsyncPhaseCreate,
		WorkRequestID: "wr-create",
	}
	deleteCalled := false

	client := newTestNamedCredentialClient(&fakeNamedCredentialOCIClient{
		workRequestFn: func(_ context.Context, req managementagentsdk.GetWorkRequestRequest) (managementagentsdk.GetWorkRequestResponse, error) {
			requireNamedCredentialStringPtr(t, "GetWorkRequestRequest.WorkRequestId", req.WorkRequestId, "wr-create")
			return managementagentsdk.GetWorkRequestResponse{
				WorkRequest: makeNamedCredentialWorkRequest(
					"wr-create",
					managementagentsdk.OperationTypesCreateNamedcredentials,
					managementagentsdk.OperationStatusInProgress,
					managementagentsdk.ActionTypesCreated,
					testNamedCredentialID,
				),
			}, nil
		},
		deleteFn: func(context.Context, managementagentsdk.DeleteNamedCredentialRequest) (managementagentsdk.DeleteNamedCredentialResponse, error) {
			deleteCalled = true
			return managementagentsdk.DeleteNamedCredentialResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while create work request is pending")
	}
	if deleteCalled {
		t.Fatal("DeleteNamedCredential() called while create work request is pending")
	}
}

func TestNamedCredentialDeleteTracksWorkRequestUntilConfirmedNotFound(t *testing.T) {
	t.Parallel()

	resource := makeNamedCredentialResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testNamedCredentialID)
	fixture := newNamedCredentialDeleteWorkRequestFixture(t, resource)
	client := newTestNamedCredentialClient(fixture.client())

	deleted, err := client.Delete(context.Background(), resource)
	fixture.requirePendingDeleteResult(t, deleted, err)

	fixture.workRequestStatus = managementagentsdk.OperationStatusSucceeded
	deleted, err = client.Delete(context.Background(), resource)
	fixture.requireConfirmedDeleteResult(t, deleted, err)
}

type namedCredentialDeleteWorkRequestFixture struct {
	t                 *testing.T
	resource          *managementagentv1beta1.NamedCredential
	deleteCalls       int
	workRequestCalls  int
	getCalls          int
	workRequestStatus managementagentsdk.OperationStatusEnum
}

func newNamedCredentialDeleteWorkRequestFixture(
	t *testing.T,
	resource *managementagentv1beta1.NamedCredential,
) *namedCredentialDeleteWorkRequestFixture {
	t.Helper()
	return &namedCredentialDeleteWorkRequestFixture{
		t:                 t,
		resource:          resource,
		workRequestStatus: managementagentsdk.OperationStatusInProgress,
	}
}

func (f *namedCredentialDeleteWorkRequestFixture) client() *fakeNamedCredentialOCIClient {
	return &fakeNamedCredentialOCIClient{
		getFn:         f.get,
		deleteFn:      f.delete,
		workRequestFn: f.workRequest,
	}
}

func (f *namedCredentialDeleteWorkRequestFixture) get(_ context.Context, req managementagentsdk.GetNamedCredentialRequest) (managementagentsdk.GetNamedCredentialResponse, error) {
	f.getCalls++
	requireNamedCredentialStringPtr(f.t, "GetNamedCredentialRequest.NamedCredentialId", req.NamedCredentialId, testNamedCredentialID)
	if f.workRequestStatus == managementagentsdk.OperationStatusSucceeded {
		return managementagentsdk.GetNamedCredentialResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "named credential is gone")
	}
	return managementagentsdk.GetNamedCredentialResponse{
		NamedCredential: makeSDKNamedCredential(testNamedCredentialID, f.resource.Spec, managementagentsdk.NamedCredentialLifecycleStateActive),
	}, nil
}

func (f *namedCredentialDeleteWorkRequestFixture) delete(_ context.Context, req managementagentsdk.DeleteNamedCredentialRequest) (managementagentsdk.DeleteNamedCredentialResponse, error) {
	f.deleteCalls++
	requireNamedCredentialStringPtr(f.t, "DeleteNamedCredentialRequest.NamedCredentialId", req.NamedCredentialId, testNamedCredentialID)
	return managementagentsdk.DeleteNamedCredentialResponse{
		OpcWorkRequestId: common.String("wr-delete"),
		OpcRequestId:     common.String("opc-delete"),
	}, nil
}

func (f *namedCredentialDeleteWorkRequestFixture) workRequest(_ context.Context, req managementagentsdk.GetWorkRequestRequest) (managementagentsdk.GetWorkRequestResponse, error) {
	f.workRequestCalls++
	requireNamedCredentialStringPtr(f.t, "GetWorkRequestRequest.WorkRequestId", req.WorkRequestId, "wr-delete")
	return managementagentsdk.GetWorkRequestResponse{
		WorkRequest: makeNamedCredentialWorkRequest(
			"wr-delete",
			managementagentsdk.OperationTypesDeleteNamedcredentials,
			f.workRequestStatus,
			managementagentsdk.ActionTypesDeleted,
			testNamedCredentialID,
		),
	}, nil
}

func (f *namedCredentialDeleteWorkRequestFixture) requirePendingDeleteResult(t *testing.T, deleted bool, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("first Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("first Delete() deleted = true, want false while delete work request is pending")
	}
	if f.deleteCalls != 1 || f.workRequestCalls != 1 || f.getCalls != 1 {
		t.Fatalf("first call counts delete/workRequest/get = %d/%d/%d, want 1/1/1", f.deleteCalls, f.workRequestCalls, f.getCalls)
	}
	if got := f.resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", got)
	}
	if current := f.resource.Status.OsokStatus.Async.Current; current == nil || current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.async.current = %#v, want delete work request", current)
	}
}

func (f *namedCredentialDeleteWorkRequestFixture) requireConfirmedDeleteResult(t *testing.T, deleted bool, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("second Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("second Delete() deleted = false, want true after unambiguous not found confirmation")
	}
	if f.resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want delete confirmation timestamp")
	}
}

func TestNamedCredentialDeleteKeepsFinalizerOnAuthShapedDeleteError(t *testing.T) {
	t.Parallel()

	resource := makeNamedCredentialResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testNamedCredentialID)
	deleteCalls := 0

	client := newTestNamedCredentialClient(&fakeNamedCredentialOCIClient{
		getFn: func(_ context.Context, req managementagentsdk.GetNamedCredentialRequest) (managementagentsdk.GetNamedCredentialResponse, error) {
			requireNamedCredentialStringPtr(t, "GetNamedCredentialRequest.NamedCredentialId", req.NamedCredentialId, testNamedCredentialID)
			return managementagentsdk.GetNamedCredentialResponse{
				NamedCredential: makeSDKNamedCredential(testNamedCredentialID, resource.Spec, managementagentsdk.NamedCredentialLifecycleStateActive),
			}, nil
		},
		deleteFn: func(context.Context, managementagentsdk.DeleteNamedCredentialRequest) (managementagentsdk.DeleteNamedCredentialResponse, error) {
			deleteCalls++
			return managementagentsdk.DeleteNamedCredentialResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization or existence is ambiguous")
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped not-found rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped not found")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteNamedCredential() calls = %d, want 1", deleteCalls)
	}
	if !strings.Contains(err.Error(), "NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %q, want NotAuthorizedOrNotFound context", err.Error())
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt set after ambiguous delete")
	}
}

func TestNamedCredentialCompletedDeleteWorkRequestRejectsAuthShapedConfirmRead(t *testing.T) {
	t.Parallel()

	resource := makeNamedCredentialResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testNamedCredentialID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:        shared.OSOKAsyncSourceWorkRequest,
		Phase:         shared.OSOKAsyncPhaseDelete,
		WorkRequestID: "wr-delete",
	}
	deleteCalled := false

	client := newTestNamedCredentialClient(&fakeNamedCredentialOCIClient{
		workRequestFn: func(_ context.Context, req managementagentsdk.GetWorkRequestRequest) (managementagentsdk.GetWorkRequestResponse, error) {
			requireNamedCredentialStringPtr(t, "GetWorkRequestRequest.WorkRequestId", req.WorkRequestId, "wr-delete")
			return managementagentsdk.GetWorkRequestResponse{
				WorkRequest: makeNamedCredentialWorkRequest(
					"wr-delete",
					managementagentsdk.OperationTypesDeleteNamedcredentials,
					managementagentsdk.OperationStatusSucceeded,
					managementagentsdk.ActionTypesDeleted,
					testNamedCredentialID,
				),
			}, nil
		},
		getFn: func(_ context.Context, req managementagentsdk.GetNamedCredentialRequest) (managementagentsdk.GetNamedCredentialResponse, error) {
			requireNamedCredentialStringPtr(t, "GetNamedCredentialRequest.NamedCredentialId", req.NamedCredentialId, testNamedCredentialID)
			return managementagentsdk.GetNamedCredentialResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization or existence is ambiguous")
		},
		deleteFn: func(context.Context, managementagentsdk.DeleteNamedCredentialRequest) (managementagentsdk.DeleteNamedCredentialResponse, error) {
			deleteCalled = true
			return managementagentsdk.DeleteNamedCredentialResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped confirm-read rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped confirm read")
	}
	if deleteCalled {
		t.Fatal("DeleteNamedCredential() called after completed delete work request")
	}
	if !strings.Contains(err.Error(), "authorization-shaped not found") {
		t.Fatalf("Delete() error = %q, want authorization-shaped not found context", err.Error())
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt set after ambiguous confirm read")
	}
}

func requireNamedCredentialCreateRequest(t *testing.T, req managementagentsdk.CreateNamedCredentialRequest, resource *managementagentv1beta1.NamedCredential) {
	t.Helper()
	requireNamedCredentialStringPtr(t, "CreateNamedCredentialRequest.Name", req.Name, resource.Spec.Name)
	requireNamedCredentialStringPtr(t, "CreateNamedCredentialRequest.Type", req.Type, resource.Spec.Type)
	requireNamedCredentialStringPtr(t, "CreateNamedCredentialRequest.ManagementAgentId", req.ManagementAgentId, resource.Spec.ManagementAgentId)
	requireNamedCredentialProperties(t, "CreateNamedCredentialRequest.Properties", req.Properties, resource.Spec.Properties)
	requireNamedCredentialStringPtr(t, "CreateNamedCredentialRequest.Description", req.Description, resource.Spec.Description)
	if req.OpcRetryToken == nil || strings.TrimSpace(*req.OpcRetryToken) == "" {
		t.Fatal("CreateNamedCredentialRequest.OpcRetryToken is empty, want deterministic retry token")
	}
	if got := req.FreeformTags["env"]; got != "test" {
		t.Fatalf("CreateNamedCredentialRequest.FreeformTags[env] = %q, want test", got)
	}
	if got := req.DefinedTags["Operations"]["CostCenter"]; got != "42" {
		t.Fatalf("CreateNamedCredentialRequest.DefinedTags[Operations][CostCenter] = %#v, want 42", got)
	}
}

func requireNamedCredentialProperties(
	t *testing.T,
	label string,
	got []managementagentsdk.NamedCredentialProperty,
	want []managementagentv1beta1.NamedCredentialProperty,
) {
	t.Helper()
	if !namedCredentialPropertiesEqual(got, namedCredentialSDKProperties(want)) {
		t.Fatalf("%s = %#v, want properties equivalent to %#v", label, got, want)
	}
}

func requireNamedCredentialActiveStatus(t *testing.T, resource *managementagentv1beta1.NamedCredential, requestID string) {
	t.Helper()
	if got := string(resource.Status.OsokStatus.Ocid); got != testNamedCredentialID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testNamedCredentialID)
	}
	if got := resource.Status.Id; got != testNamedCredentialID {
		t.Fatalf("status.id = %q, want %q", got, testNamedCredentialID)
	}
	if requestID != "" {
		if got := resource.Status.OsokStatus.OpcRequestID; got != requestID {
			t.Fatalf("status.status.opcRequestId = %q, want %s", got, requestID)
		}
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after completed work request", resource.Status.OsokStatus.Async.Current)
	}
	if got := lastNamedCredentialConditionType(resource); got != shared.Active {
		t.Fatalf("last condition = %q, want Active", got)
	}
}

func requireNamedCredentialSuccessfulResponse(t *testing.T, operation string, response servicemanager.OSOKResponse) {
	t.Helper()
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("%s response = %#v, want successful non-requeue", operation, response)
	}
}

func requireNamedCredentialStringPtr(t *testing.T, label string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", label, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", label, *got, want)
	}
}

func assertNamedCredentialStringSliceEqual(t *testing.T, label string, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s length = %d, want %d: got %#v want %#v", label, len(got), len(want), got, want)
	}
	for index := range got {
		if got[index] != want[index] {
			t.Fatalf("%s[%d] = %q, want %q: got %#v want %#v", label, index, got[index], want[index], got, want)
		}
	}
}

func lastNamedCredentialConditionType(resource *managementagentv1beta1.NamedCredential) shared.OSOKConditionType {
	if resource == nil || len(resource.Status.OsokStatus.Conditions) == 0 {
		return ""
	}
	return resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type
}

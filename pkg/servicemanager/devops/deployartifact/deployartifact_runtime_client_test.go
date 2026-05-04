/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package deployartifact

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	devopssdk "github.com/oracle/oci-go-sdk/v65/devops"
	devopsv1beta1 "github.com/oracle/oci-service-operator/api/devops/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testDeployArtifactID     = "ocid1.devopsdeployartifact.oc1..artifact"
	testDeployArtifactOther  = "ocid1.devopsdeployartifact.oc1..other"
	testDeployArtifactProjID = "ocid1.devopsproject.oc1..project"
	testDeployArtifactRepoID = "ocid1.artifactrepository.oc1..repository"
	testDeployArtifactCompID = "ocid1.compartment.oc1..compartment"
)

type fakeDeployArtifactOCIClient struct {
	createRequests      []devopssdk.CreateDeployArtifactRequest
	getRequests         []devopssdk.GetDeployArtifactRequest
	listRequests        []devopssdk.ListDeployArtifactsRequest
	updateRequests      []devopssdk.UpdateDeployArtifactRequest
	deleteRequests      []devopssdk.DeleteDeployArtifactRequest
	workRequestRequests []devopssdk.GetWorkRequestRequest

	createFn      func(context.Context, devopssdk.CreateDeployArtifactRequest) (devopssdk.CreateDeployArtifactResponse, error)
	getFn         func(context.Context, devopssdk.GetDeployArtifactRequest) (devopssdk.GetDeployArtifactResponse, error)
	listFn        func(context.Context, devopssdk.ListDeployArtifactsRequest) (devopssdk.ListDeployArtifactsResponse, error)
	updateFn      func(context.Context, devopssdk.UpdateDeployArtifactRequest) (devopssdk.UpdateDeployArtifactResponse, error)
	deleteFn      func(context.Context, devopssdk.DeleteDeployArtifactRequest) (devopssdk.DeleteDeployArtifactResponse, error)
	workRequestFn func(context.Context, devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error)
}

func (f *fakeDeployArtifactOCIClient) CreateDeployArtifact(
	ctx context.Context,
	request devopssdk.CreateDeployArtifactRequest,
) (devopssdk.CreateDeployArtifactResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return devopssdk.CreateDeployArtifactResponse{}, nil
}

func (f *fakeDeployArtifactOCIClient) GetDeployArtifact(
	ctx context.Context,
	request devopssdk.GetDeployArtifactRequest,
) (devopssdk.GetDeployArtifactResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return devopssdk.GetDeployArtifactResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "deploy artifact not found")
}

func (f *fakeDeployArtifactOCIClient) ListDeployArtifacts(
	ctx context.Context,
	request devopssdk.ListDeployArtifactsRequest,
) (devopssdk.ListDeployArtifactsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return devopssdk.ListDeployArtifactsResponse{}, nil
}

func (f *fakeDeployArtifactOCIClient) UpdateDeployArtifact(
	ctx context.Context,
	request devopssdk.UpdateDeployArtifactRequest,
) (devopssdk.UpdateDeployArtifactResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return devopssdk.UpdateDeployArtifactResponse{}, nil
}

func (f *fakeDeployArtifactOCIClient) DeleteDeployArtifact(
	ctx context.Context,
	request devopssdk.DeleteDeployArtifactRequest,
) (devopssdk.DeleteDeployArtifactResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return devopssdk.DeleteDeployArtifactResponse{}, nil
}

func (f *fakeDeployArtifactOCIClient) GetWorkRequest(
	ctx context.Context,
	request devopssdk.GetWorkRequestRequest,
) (devopssdk.GetWorkRequestResponse, error) {
	f.workRequestRequests = append(f.workRequestRequests, request)
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, request)
	}
	return devopssdk.GetWorkRequestResponse{}, nil
}

func newDeployArtifactTestClient(fake *fakeDeployArtifactOCIClient) DeployArtifactServiceClient {
	return newDeployArtifactServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
}

func newDeployArtifactResource() *devopsv1beta1.DeployArtifact {
	return &devopsv1beta1.DeployArtifact{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deploy-artifact",
			Namespace: "default",
		},
		Spec: devopsv1beta1.DeployArtifactSpec{
			DeployArtifactType:       string(devopssdk.DeployArtifactDeployArtifactTypeKubernetesManifest),
			ArgumentSubstitutionMode: string(devopssdk.DeployArtifactArgumentSubstitutionModeNone),
			ProjectId:                testDeployArtifactProjID,
			DisplayName:              "manifest-artifact",
			Description:              "desired artifact",
			DeployArtifactSource: devopsv1beta1.DeployArtifactSource{
				DeployArtifactSourceType: string(devopssdk.DeployArtifactSourceDeployArtifactSourceTypeGenericArtifact),
				RepositoryId:             testDeployArtifactRepoID,
				DeployArtifactPath:       "manifests/app.yaml",
				DeployArtifactVersion:    "1.0.0",
			},
			FreeformTags: map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func newExistingDeployArtifactResource(id string) *devopsv1beta1.DeployArtifact {
	resource := newDeployArtifactResource()
	resource.Status.Id = id
	resource.Status.OsokStatus.Ocid = shared.OCID(id)
	return resource
}

func sdkDeployArtifactFromSpec(
	t *testing.T,
	id string,
	spec devopsv1beta1.DeployArtifactSpec,
	state devopssdk.DeployArtifactLifecycleStateEnum,
) devopssdk.DeployArtifact {
	t.Helper()

	source, err := deployArtifactSourceFromSpec(spec.DeployArtifactSource)
	if err != nil {
		t.Fatalf("deployArtifactSourceFromSpec() error = %v", err)
	}
	return devopssdk.DeployArtifact{
		Id:                       common.String(id),
		ProjectId:                common.String(spec.ProjectId),
		CompartmentId:            common.String(testDeployArtifactCompID),
		DeployArtifactType:       devopssdk.DeployArtifactDeployArtifactTypeEnum(spec.DeployArtifactType),
		DeployArtifactSource:     source,
		ArgumentSubstitutionMode: devopssdk.DeployArtifactArgumentSubstitutionModeEnum(spec.ArgumentSubstitutionMode),
		Description:              optionalDeployArtifactTestString(spec.Description),
		DisplayName:              optionalDeployArtifactTestString(spec.DisplayName),
		LifecycleState:           state,
		FreeformTags:             deployArtifactCloneStringMap(spec.FreeformTags),
		DefinedTags:              deployArtifactDefinedTagsFromSpec(spec.DefinedTags),
		SystemTags:               map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "true"}},
	}
}

func sdkDeployArtifactSummaryFromSpec(
	t *testing.T,
	id string,
	spec devopsv1beta1.DeployArtifactSpec,
	state devopssdk.DeployArtifactLifecycleStateEnum,
) devopssdk.DeployArtifactSummary {
	t.Helper()

	artifact := sdkDeployArtifactFromSpec(t, id, spec, state)
	return devopssdk.DeployArtifactSummary{
		Id:                       artifact.Id,
		ProjectId:                artifact.ProjectId,
		CompartmentId:            artifact.CompartmentId,
		DeployArtifactType:       artifact.DeployArtifactType,
		DeployArtifactSource:     artifact.DeployArtifactSource,
		ArgumentSubstitutionMode: artifact.ArgumentSubstitutionMode,
		Description:              artifact.Description,
		DisplayName:              artifact.DisplayName,
		LifecycleState:           artifact.LifecycleState,
		FreeformTags:             artifact.FreeformTags,
		DefinedTags:              artifact.DefinedTags,
		SystemTags:               artifact.SystemTags,
	}
}

func deployArtifactWorkRequest(
	id string,
	operation devopssdk.OperationTypeEnum,
	status devopssdk.OperationStatusEnum,
	action devopssdk.ActionTypeEnum,
	artifactID string,
) devopssdk.WorkRequest {
	percentComplete := float32(42)
	workRequest := devopssdk.WorkRequest{
		OperationType:   operation,
		Status:          status,
		Id:              common.String(id),
		PercentComplete: &percentComplete,
	}
	if artifactID != "" {
		workRequest.Resources = []devopssdk.WorkRequestResource{
			{
				EntityType: common.String("deployArtifact"),
				ActionType: action,
				Identifier: common.String(artifactID),
				EntityUri:  common.String("/20210630/deployArtifacts/" + artifactID),
			},
		}
	}
	return workRequest
}

func TestDeployArtifactRuntimeSemanticsEncodesWorkRequestAndDeleteContracts(t *testing.T) {
	t.Parallel()

	got := newDeployArtifactRuntimeSemantics()
	if got.FormalService != "devops" || got.FormalSlug != "deployartifact" {
		t.Fatalf("formal identity = %s/%s, want devops/deployartifact", got.FormalService, got.FormalSlug)
	}
	if got.Async == nil || got.Async.Strategy != "workrequest" || got.Async.Runtime != "generatedruntime" {
		t.Fatalf("async semantics = %#v, want generatedruntime workrequest", got.Async)
	}
	assertDeployArtifactStrings(t, "work request phases", got.Async.WorkRequest.Phases, []string{"create", "update", "delete"})
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("finalizer policy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" || got.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("delete semantics = %#v follow-up %#v, want required confirm-delete", got.Delete, got.DeleteFollowUp)
	}
	assertDeployArtifactStrings(t, "list match fields", got.List.MatchFields, []string{
		"projectId",
		"displayName",
		"deployArtifactType",
		"argumentSubstitutionMode",
		"deployArtifactSource.deployArtifactSourceType",
		"deployArtifactSource.repositoryId",
		"deployArtifactSource.deployArtifactPath",
		"deployArtifactSource.deployArtifactVersion",
		"deployArtifactSource.chartUrl",
		"deployArtifactSource.imageUri",
		"deployArtifactSource.imageDigest",
		"deployArtifactSource.base64EncodedContent",
		"deployArtifactSource.helmArtifactSourceType",
		"deployArtifactSource.helmVerificationKeySource.verificationKeySourceType",
		"deployArtifactSource.helmVerificationKeySource.currentPublicKey",
		"deployArtifactSource.helmVerificationKeySource.previousPublicKey",
		"deployArtifactSource.helmVerificationKeySource.vaultSecretId",
	})
	assertDeployArtifactStrings(t, "mutable fields", got.Mutation.Mutable, []string{
		"deployArtifactType",
		"deployArtifactSource",
		"argumentSubstitutionMode",
		"description",
		"displayName",
		"freeformTags",
		"definedTags",
	})
	assertDeployArtifactStrings(t, "force-new fields", got.Mutation.ForceNew, []string{"projectId"})
}

func TestDeployArtifactCreateUsesPolymorphicSourceAndWorkRequest(t *testing.T) {
	t.Parallel()

	resource := newDeployArtifactResource()
	workRequestStatus := devopssdk.OperationStatusInProgress
	createCalls := 0
	getCalls := 0
	var createRequest devopssdk.CreateDeployArtifactRequest

	client := newDeployArtifactTestClient(&fakeDeployArtifactOCIClient{
		listFn: func(_ context.Context, request devopssdk.ListDeployArtifactsRequest) (devopssdk.ListDeployArtifactsResponse, error) {
			requireDeployArtifactStringPtr(t, "list projectId", request.ProjectId, testDeployArtifactProjID)
			requireDeployArtifactStringPtr(t, "list displayName", request.DisplayName, resource.Spec.DisplayName)
			return devopssdk.ListDeployArtifactsResponse{}, nil
		},
		createFn: func(_ context.Context, request devopssdk.CreateDeployArtifactRequest) (devopssdk.CreateDeployArtifactResponse, error) {
			createCalls++
			createRequest = request
			return devopssdk.CreateDeployArtifactResponse{
				DeployArtifact:   sdkDeployArtifactFromSpec(t, testDeployArtifactID, resource.Spec, devopssdk.DeployArtifactLifecycleStateCreating),
				OpcWorkRequestId: common.String("wr-create"),
				OpcRequestId:     common.String("opc-create"),
			}, nil
		},
		workRequestFn: func(_ context.Context, request devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error) {
			requireDeployArtifactStringPtr(t, "workRequestId", request.WorkRequestId, "wr-create")
			action := devopssdk.ActionTypeInProgress
			if workRequestStatus == devopssdk.OperationStatusSucceeded {
				action = devopssdk.ActionTypeCreated
			}
			return devopssdk.GetWorkRequestResponse{
				WorkRequest: deployArtifactWorkRequest("wr-create", devopssdk.OperationTypeCreateDeployArtifact, workRequestStatus, action, testDeployArtifactID),
			}, nil
		},
		getFn: func(_ context.Context, request devopssdk.GetDeployArtifactRequest) (devopssdk.GetDeployArtifactResponse, error) {
			getCalls++
			requireDeployArtifactStringPtr(t, "get deployArtifactId", request.DeployArtifactId, testDeployArtifactID)
			return devopssdk.GetDeployArtifactResponse{
				DeployArtifact: sdkDeployArtifactFromSpec(t, testDeployArtifactID, resource.Spec, devopssdk.DeployArtifactLifecycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireDeployArtifactNoError(t, "CreateOrUpdate()", err)
	requireDeployArtifactRequeueResponse(t, response, true, "while work request is pending")
	requireDeployArtifactCreateRequest(t, createRequest, resource)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", got)
	}
	requireDeployArtifactAsync(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create", shared.OSOKAsyncClassPending)

	workRequestStatus = devopssdk.OperationStatusSucceeded
	response, err = client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireDeployArtifactNoError(t, "CreateOrUpdate() after work request success", err)
	requireDeployArtifactRequeueResponse(t, response, false, "after completed create work request")
	requireDeployArtifactCallCount(t, "CreateDeployArtifact()", createCalls, 1)
	requireDeployArtifactCallCount(t, "GetDeployArtifact()", getCalls, 1)
	requireDeployArtifactIdentity(t, resource, testDeployArtifactID)
	requireDeployArtifactCondition(t, resource, shared.Active)
	requireDeployArtifactNoCurrentAsync(t, resource)
}

func TestDeployArtifactCreateOrUpdateBindsExistingAcrossListPages(t *testing.T) {
	t.Parallel()

	resource := newDeployArtifactResource()
	otherSpec := resource.Spec
	otherSpec.DeployArtifactSource.DeployArtifactPath = "manifests/other.yaml"
	fake := &fakeDeployArtifactOCIClient{
		listFn: func(_ context.Context, request devopssdk.ListDeployArtifactsRequest) (devopssdk.ListDeployArtifactsResponse, error) {
			switch deployArtifactString(request.Page) {
			case "":
				return devopssdk.ListDeployArtifactsResponse{
					DeployArtifactCollection: devopssdk.DeployArtifactCollection{Items: []devopssdk.DeployArtifactSummary{
						sdkDeployArtifactSummaryFromSpec(t, testDeployArtifactOther, otherSpec, devopssdk.DeployArtifactLifecycleStateActive),
					}},
					OpcNextPage: common.String("page-2"),
				}, nil
			case "page-2":
				return devopssdk.ListDeployArtifactsResponse{
					DeployArtifactCollection: devopssdk.DeployArtifactCollection{Items: []devopssdk.DeployArtifactSummary{
						sdkDeployArtifactSummaryFromSpec(t, testDeployArtifactID, resource.Spec, devopssdk.DeployArtifactLifecycleStateActive),
					}},
				}, nil
			default:
				t.Fatalf("unexpected list page %q", deployArtifactString(request.Page))
				return devopssdk.ListDeployArtifactsResponse{}, nil
			}
		},
		getFn: func(_ context.Context, request devopssdk.GetDeployArtifactRequest) (devopssdk.GetDeployArtifactResponse, error) {
			requireDeployArtifactStringPtr(t, "get deployArtifactId", request.DeployArtifactId, testDeployArtifactID)
			return devopssdk.GetDeployArtifactResponse{
				DeployArtifact: sdkDeployArtifactFromSpec(t, testDeployArtifactID, resource.Spec, devopssdk.DeployArtifactLifecycleStateActive),
			}, nil
		},
		createFn: func(context.Context, devopssdk.CreateDeployArtifactRequest) (devopssdk.CreateDeployArtifactResponse, error) {
			t.Fatal("CreateDeployArtifact() should not be called when list resolves an existing artifact")
			return devopssdk.CreateDeployArtifactResponse{}, nil
		},
	}
	client := newDeployArtifactTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful no requeue", response)
	}
	if len(fake.listRequests) != 2 {
		t.Fatalf("ListDeployArtifacts() calls = %d, want 2 paginated calls", len(fake.listRequests))
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateDeployArtifact() calls = %d, want 0", len(fake.createRequests))
	}
	requireDeployArtifactIdentity(t, resource, testDeployArtifactID)
}

func TestDeployArtifactCreateOrUpdateSkipsUpdateWhenCurrentMatches(t *testing.T) {
	t.Parallel()

	resource := newExistingDeployArtifactResource(testDeployArtifactID)
	client := newDeployArtifactTestClient(&fakeDeployArtifactOCIClient{
		getFn: func(_ context.Context, request devopssdk.GetDeployArtifactRequest) (devopssdk.GetDeployArtifactResponse, error) {
			requireDeployArtifactStringPtr(t, "get deployArtifactId", request.DeployArtifactId, testDeployArtifactID)
			return devopssdk.GetDeployArtifactResponse{
				DeployArtifact: sdkDeployArtifactFromSpec(t, testDeployArtifactID, resource.Spec, devopssdk.DeployArtifactLifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, devopssdk.UpdateDeployArtifactRequest) (devopssdk.UpdateDeployArtifactResponse, error) {
			t.Fatal("UpdateDeployArtifact() should not be called when desired and observed state match")
			return devopssdk.UpdateDeployArtifactResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful no requeue", response)
	}
}

func TestDeployArtifactCreateOrUpdateUpdatesMutableFieldsAndCompletesWorkRequest(t *testing.T) {
	t.Parallel()

	original := newDeployArtifactResource()
	resource := newExistingDeployArtifactResource(testDeployArtifactID)
	resource.Spec.Description = "updated artifact"
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}

	getCalls := 0
	var updateRequest devopssdk.UpdateDeployArtifactRequest
	client := newDeployArtifactTestClient(&fakeDeployArtifactOCIClient{
		getFn: func(_ context.Context, request devopssdk.GetDeployArtifactRequest) (devopssdk.GetDeployArtifactResponse, error) {
			getCalls++
			spec := original.Spec
			if getCalls > 1 {
				spec = resource.Spec
			}
			requireDeployArtifactStringPtr(t, "get deployArtifactId", request.DeployArtifactId, testDeployArtifactID)
			return devopssdk.GetDeployArtifactResponse{
				DeployArtifact: sdkDeployArtifactFromSpec(t, testDeployArtifactID, spec, devopssdk.DeployArtifactLifecycleStateActive),
			}, nil
		},
		updateFn: func(_ context.Context, request devopssdk.UpdateDeployArtifactRequest) (devopssdk.UpdateDeployArtifactResponse, error) {
			updateRequest = request
			return devopssdk.UpdateDeployArtifactResponse{
				DeployArtifact:   sdkDeployArtifactFromSpec(t, testDeployArtifactID, resource.Spec, devopssdk.DeployArtifactLifecycleStateUpdating),
				OpcWorkRequestId: common.String("wr-update"),
				OpcRequestId:     common.String("opc-update"),
			}, nil
		},
		workRequestFn: func(_ context.Context, request devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error) {
			requireDeployArtifactStringPtr(t, "workRequestId", request.WorkRequestId, "wr-update")
			return devopssdk.GetWorkRequestResponse{
				WorkRequest: deployArtifactWorkRequest("wr-update", devopssdk.OperationTypeUpdateDeployArtifact, devopssdk.OperationStatusSucceeded, devopssdk.ActionTypeUpdated, testDeployArtifactID),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful no requeue after completed update work request", response)
	}
	requireDeployArtifactStringPtr(t, "update deployArtifactId", updateRequest.DeployArtifactId, testDeployArtifactID)
	requireDeployArtifactStringPtr(t, "update description", updateRequest.Description, resource.Spec.Description)
	if got := updateRequest.FreeformTags["env"]; got != "prod" {
		t.Fatalf("update freeformTags[env] = %q, want prod", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", got)
	}
	requireDeployArtifactCondition(t, resource, shared.Active)
}

func TestDeployArtifactCreateOrUpdateRejectsProjectDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := newExistingDeployArtifactResource(testDeployArtifactID)
	current := sdkDeployArtifactFromSpec(t, testDeployArtifactID, resource.Spec, devopssdk.DeployArtifactLifecycleStateActive)
	current.ProjectId = common.String("ocid1.devopsproject.oc1..different")
	fake := &fakeDeployArtifactOCIClient{
		getFn: func(context.Context, devopssdk.GetDeployArtifactRequest) (devopssdk.GetDeployArtifactResponse, error) {
			return devopssdk.GetDeployArtifactResponse{DeployArtifact: current}, nil
		},
		updateFn: func(context.Context, devopssdk.UpdateDeployArtifactRequest) (devopssdk.UpdateDeployArtifactResponse, error) {
			t.Fatal("UpdateDeployArtifact() should not be called when projectId drifts")
			return devopssdk.UpdateDeployArtifactResponse{}, nil
		},
	}
	client := newDeployArtifactTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want projectId drift rejection")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if !strings.Contains(err.Error(), "projectId") {
		t.Fatalf("CreateOrUpdate() error = %v, want projectId detail", err)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateDeployArtifact() calls = %d, want 0", len(fake.updateRequests))
	}
}

func TestDeployArtifactDeletePollsWorkRequestAndConfirmsNotFound(t *testing.T) {
	t.Parallel()

	resource := newExistingDeployArtifactResource(testDeployArtifactID)
	getCalls := 0
	client := newDeployArtifactTestClient(&fakeDeployArtifactOCIClient{
		getFn: func(_ context.Context, request devopssdk.GetDeployArtifactRequest) (devopssdk.GetDeployArtifactResponse, error) {
			getCalls++
			requireDeployArtifactStringPtr(t, "get deployArtifactId", request.DeployArtifactId, testDeployArtifactID)
			if getCalls == 1 {
				return devopssdk.GetDeployArtifactResponse{
					DeployArtifact: sdkDeployArtifactFromSpec(t, testDeployArtifactID, resource.Spec, devopssdk.DeployArtifactLifecycleStateActive),
				}, nil
			}
			return devopssdk.GetDeployArtifactResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "deleted")
		},
		deleteFn: func(_ context.Context, request devopssdk.DeleteDeployArtifactRequest) (devopssdk.DeleteDeployArtifactResponse, error) {
			requireDeployArtifactStringPtr(t, "delete deployArtifactId", request.DeployArtifactId, testDeployArtifactID)
			return devopssdk.DeleteDeployArtifactResponse{
				OpcWorkRequestId: common.String("wr-delete"),
				OpcRequestId:     common.String("opc-delete"),
			}, nil
		},
		workRequestFn: func(_ context.Context, request devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error) {
			requireDeployArtifactStringPtr(t, "workRequestId", request.WorkRequestId, "wr-delete")
			return devopssdk.GetWorkRequestResponse{
				WorkRequest: deployArtifactWorkRequest("wr-delete", devopssdk.OperationTypeDeleteDeployArtifact, devopssdk.OperationStatusSucceeded, devopssdk.ActionTypeDeleted, testDeployArtifactID),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after work request and not-found confirmation")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", got)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
}

func TestDeployArtifactDeleteTreatsAuthShapedNotFoundConservatively(t *testing.T) {
	t.Parallel()

	resource := newExistingDeployArtifactResource(testDeployArtifactID)
	fake := &fakeDeployArtifactOCIClient{
		getFn: func(context.Context, devopssdk.GetDeployArtifactRequest) (devopssdk.GetDeployArtifactResponse, error) {
			return devopssdk.GetDeployArtifactResponse{
				DeployArtifact: sdkDeployArtifactFromSpec(t, testDeployArtifactID, resource.Spec, devopssdk.DeployArtifactLifecycleStateActive),
			}, nil
		},
		deleteFn: func(context.Context, devopssdk.DeleteDeployArtifactRequest) (devopssdk.DeleteDeployArtifactResponse, error) {
			return devopssdk.DeleteDeployArtifactResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous")
		},
	}
	client := newDeployArtifactTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped not-found to stay fatal")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous not-found")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("DeleteDeployArtifact() calls = %d, want 1", len(fake.deleteRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestDeployArtifactDeleteTreatsPreReadAuthShapedNotFoundAsAmbiguous(t *testing.T) {
	t.Parallel()

	resource := newExistingDeployArtifactResource(testDeployArtifactID)
	fake := &fakeDeployArtifactOCIClient{
		getFn: func(context.Context, devopssdk.GetDeployArtifactRequest) (devopssdk.GetDeployArtifactResponse, error) {
			return devopssdk.GetDeployArtifactResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous pre-read")
		},
		deleteFn: func(context.Context, devopssdk.DeleteDeployArtifactRequest) (devopssdk.DeleteDeployArtifactResponse, error) {
			t.Fatal("DeleteDeployArtifact() should not be called after ambiguous pre-delete read")
			return devopssdk.DeleteDeployArtifactResponse{}, nil
		},
	}
	client := newDeployArtifactTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous pre-read to stay fatal")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeleteDeployArtifact() calls = %d, want 0", len(fake.deleteRequests))
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("Delete() error = %v, want ambiguous detail", err)
	}
}

func TestDeployArtifactCreateRecordsOpcRequestIDFromOCIError(t *testing.T) {
	t.Parallel()

	resource := newDeployArtifactResource()
	client := newDeployArtifactTestClient(&fakeDeployArtifactOCIClient{
		createFn: func(context.Context, devopssdk.CreateDeployArtifactRequest) (devopssdk.CreateDeployArtifactResponse, error) {
			return devopssdk.CreateDeployArtifactResponse{}, errortest.NewServiceError(409, errorutil.IncorrectState, "conflict")
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
	requireDeployArtifactCondition(t, resource, shared.Failed)
}

func requireDeployArtifactNoError(t *testing.T, label string, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("%s error = %v", label, err)
	}
}

func requireDeployArtifactRequeueResponse(
	t *testing.T,
	response servicemanager.OSOKResponse,
	shouldRequeue bool,
	context string,
) {
	t.Helper()

	if !response.IsSuccessful || response.ShouldRequeue != shouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue=%t %s", response, shouldRequeue, context)
	}
}

func requireDeployArtifactCallCount(t *testing.T, label string, got int, want int) {
	t.Helper()

	if got != want {
		t.Fatalf("%s calls = %d, want %d", label, got, want)
	}
}

func requireDeployArtifactNoCurrentAsync(t *testing.T, resource *devopsv1beta1.DeployArtifact) {
	t.Helper()

	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after completed create", resource.Status.OsokStatus.Async.Current)
	}
}

func requireDeployArtifactCreateRequest(
	t *testing.T,
	request devopssdk.CreateDeployArtifactRequest,
	resource *devopsv1beta1.DeployArtifact,
) {
	t.Helper()

	requireDeployArtifactStringPtr(t, "create projectId", request.ProjectId, testDeployArtifactProjID)
	requireDeployArtifactStringPtr(t, "create displayName", request.DisplayName, resource.Spec.DisplayName)
	if request.DeployArtifactType != devopssdk.DeployArtifactDeployArtifactTypeKubernetesManifest {
		t.Fatalf("create deployArtifactType = %q, want KUBERNETES_MANIFEST", request.DeployArtifactType)
	}
	source, ok := request.DeployArtifactSource.(devopssdk.GenericDeployArtifactSource)
	if !ok {
		t.Fatalf("create deployArtifactSource type = %T, want GenericDeployArtifactSource", request.DeployArtifactSource)
	}
	requireDeployArtifactStringPtr(t, "create source repositoryId", source.RepositoryId, testDeployArtifactRepoID)
	requireDeployArtifactStringPtr(t, "create source path", source.DeployArtifactPath, resource.Spec.DeployArtifactSource.DeployArtifactPath)
	requireDeployArtifactStringPtr(t, "create source version", source.DeployArtifactVersion, resource.Spec.DeployArtifactSource.DeployArtifactVersion)
	if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
		t.Fatal("create opc retry token is empty")
	}
}

func requireDeployArtifactIdentity(t *testing.T, resource *devopsv1beta1.DeployArtifact, want string) {
	t.Helper()

	if got := resource.Status.Id; got != want {
		t.Fatalf("status.id = %q, want %q", got, want)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
}

func requireDeployArtifactAsync(
	t *testing.T,
	resource *devopsv1beta1.DeployArtifact,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	class shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()

	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want tracked work request")
	}
	if current.Phase != phase || current.WorkRequestID != workRequestID || current.NormalizedClass != class {
		t.Fatalf("status.async.current = %#v, want phase=%q workRequestID=%q class=%q", current, phase, workRequestID, class)
	}
}

func requireDeployArtifactCondition(
	t *testing.T,
	resource *devopsv1beta1.DeployArtifact,
	want shared.OSOKConditionType,
) {
	t.Helper()

	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.conditions is empty, want %q", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last condition = %q, want %q", got, want)
	}
}

func requireDeployArtifactStringPtr(t *testing.T, label string, got *string, want string) {
	t.Helper()

	if got == nil {
		t.Fatalf("%s = nil, want %q", label, want)
	}
	if strings.TrimSpace(*got) != want {
		t.Fatalf("%s = %q, want %q", label, *got, want)
	}
}

func assertDeployArtifactStrings(t *testing.T, label string, got []string, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("%s length = %d, want %d (%v)", label, len(got), len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s[%d] = %q, want %q (got %v)", label, i, got[i], want[i], got)
		}
	}
}

func optionalDeployArtifactTestString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return common.String(value)
}

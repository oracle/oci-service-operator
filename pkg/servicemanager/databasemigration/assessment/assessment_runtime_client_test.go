/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package assessment

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	databasemigrationsdk "github.com/oracle/oci-go-sdk/v65/databasemigration"
	databasemigrationv1beta1 "github.com/oracle/oci-service-operator/api/databasemigration/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeAssessmentOCIClient struct {
	createFn      func(context.Context, databasemigrationsdk.CreateAssessmentRequest) (databasemigrationsdk.CreateAssessmentResponse, error)
	getFn         func(context.Context, databasemigrationsdk.GetAssessmentRequest) (databasemigrationsdk.GetAssessmentResponse, error)
	listFn        func(context.Context, databasemigrationsdk.ListAssessmentsRequest) (databasemigrationsdk.ListAssessmentsResponse, error)
	updateFn      func(context.Context, databasemigrationsdk.UpdateAssessmentRequest) (databasemigrationsdk.UpdateAssessmentResponse, error)
	deleteFn      func(context.Context, databasemigrationsdk.DeleteAssessmentRequest) (databasemigrationsdk.DeleteAssessmentResponse, error)
	workRequestFn func(context.Context, databasemigrationsdk.GetWorkRequestRequest) (databasemigrationsdk.GetWorkRequestResponse, error)
}

func (f *fakeAssessmentOCIClient) CreateAssessment(
	ctx context.Context,
	req databasemigrationsdk.CreateAssessmentRequest,
) (databasemigrationsdk.CreateAssessmentResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return databasemigrationsdk.CreateAssessmentResponse{}, nil
}

func (f *fakeAssessmentOCIClient) GetAssessment(
	ctx context.Context,
	req databasemigrationsdk.GetAssessmentRequest,
) (databasemigrationsdk.GetAssessmentResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return databasemigrationsdk.GetAssessmentResponse{}, nil
}

func (f *fakeAssessmentOCIClient) ListAssessments(
	ctx context.Context,
	req databasemigrationsdk.ListAssessmentsRequest,
) (databasemigrationsdk.ListAssessmentsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return databasemigrationsdk.ListAssessmentsResponse{}, nil
}

func (f *fakeAssessmentOCIClient) UpdateAssessment(
	ctx context.Context,
	req databasemigrationsdk.UpdateAssessmentRequest,
) (databasemigrationsdk.UpdateAssessmentResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return databasemigrationsdk.UpdateAssessmentResponse{}, nil
}

func (f *fakeAssessmentOCIClient) DeleteAssessment(
	ctx context.Context,
	req databasemigrationsdk.DeleteAssessmentRequest,
) (databasemigrationsdk.DeleteAssessmentResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return databasemigrationsdk.DeleteAssessmentResponse{}, nil
}

func (f *fakeAssessmentOCIClient) GetWorkRequest(
	ctx context.Context,
	req databasemigrationsdk.GetWorkRequestRequest,
) (databasemigrationsdk.GetWorkRequestResponse, error) {
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, req)
	}
	return databasemigrationsdk.GetWorkRequestResponse{}, nil
}

type assessmentRequestBodyBuilder interface {
	HTTPRequest(
		method string,
		path string,
		binaryRequestBody *common.OCIReadSeekCloser,
		extraHeaders map[string]string,
	) (http.Request, error)
}

func newTestAssessmentClient(fake *fakeAssessmentOCIClient) AssessmentServiceClient {
	return newAssessmentServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: logr.Discard()},
		fake,
	)
}

func TestReviewedAssessmentRuntimeSemanticsEncodesWorkRequestContract(t *testing.T) {
	t.Parallel()

	got := reviewedAssessmentRuntimeSemantics()
	if got == nil {
		t.Fatal("reviewedAssessmentRuntimeSemantics() = nil")
	}

	if got.FormalService != "databasemigration" {
		t.Fatalf("FormalService = %q, want databasemigration", got.FormalService)
	}
	if got.FormalSlug != "assessment" {
		t.Fatalf("FormalSlug = %q, want assessment", got.FormalSlug)
	}
	if got.Async == nil {
		t.Fatal("Async = nil, want workrequest semantics")
	}
	if got.Async.Strategy != "workrequest" {
		t.Fatalf("Async.Strategy = %q, want workrequest", got.Async.Strategy)
	}
	if got.Async.Runtime != "generatedruntime" {
		t.Fatalf("Async.Runtime = %q, want generatedruntime", got.Async.Runtime)
	}
	if got.Async.WorkRequest == nil {
		t.Fatal("Async.WorkRequest = nil")
	}
	assertAssessmentStringSliceEqual(t, "Async.WorkRequest.Phases", got.Async.WorkRequest.Phases, []string{"create", "update", "delete"})
	assertAssessmentStringSliceEqual(t, "Lifecycle.ProvisioningStates", got.Lifecycle.ProvisioningStates, []string{"CREATING", "IN_PROGRESS"})
	assertAssessmentStringSliceEqual(t, "Lifecycle.UpdatingStates", got.Lifecycle.UpdatingStates, []string{"UPDATING"})
	assertAssessmentStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE", "SUCCEEDED"})
	assertAssessmentStringSliceEqual(t, "Delete.PendingStates", got.Delete.PendingStates, []string{"DELETING"})
	assertAssessmentStringSliceEqual(t, "Delete.TerminalStates", got.Delete.TerminalStates, []string{"DELETED"})
	assertAssessmentStringSliceContainsAll(t, "Mutation.ForceNew", got.Mutation.ForceNew, "bulkIncludeExcludeData", "compartmentId", "databaseCombination", "excludeObjects", "includeObjects")
	assertAssessmentStringSliceContainsAll(t, "Mutation.Mutable", got.Mutation.Mutable, "displayName", "description", "sourceDatabaseConnection", "targetDatabaseConnection")
	if got.List != nil {
		t.Fatalf("List = %#v, want nil generic list semantics because exact bind-before-create is custom", got.List)
	}
	if got.CreateFollowUp.Strategy != "GetWorkRequest -> GetAssessment" {
		t.Fatalf("CreateFollowUp.Strategy = %q, want workrequest-backed Assessment follow-up", got.CreateFollowUp.Strategy)
	}
	if got.UpdateFollowUp.Strategy != "GetWorkRequest -> GetAssessment" {
		t.Fatalf("UpdateFollowUp.Strategy = %q, want workrequest-backed Assessment follow-up", got.UpdateFollowUp.Strategy)
	}
	if got.DeleteFollowUp.Strategy != "GetWorkRequest -> GetAssessment/ListAssessments confirm-delete" {
		t.Fatalf("DeleteFollowUp.Strategy = %q, want workrequest-backed delete confirmation", got.DeleteFollowUp.Strategy)
	}
	if len(got.Unsupported) != 0 {
		t.Fatalf("Unsupported = %#v, want runtime-enforced rejection instead of an always-open formal block", got.Unsupported)
	}
}

func TestGuardAssessmentExistingBeforeCreate(t *testing.T) {
	t.Parallel()

	resource := makeMySQLAssessmentResource()
	resource.Spec.DisplayName = ""

	decision, err := guardAssessmentExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardAssessmentExistingBeforeCreate(empty displayName) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionSkip {
		t.Fatalf("guardAssessmentExistingBeforeCreate(empty displayName) = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionSkip)
	}

	resource.Spec.DisplayName = "mysql-assessment"
	resource.Spec.DatabaseCombination = ""
	decision, err = guardAssessmentExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardAssessmentExistingBeforeCreate(empty databaseCombination) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionSkip {
		t.Fatalf("guardAssessmentExistingBeforeCreate(empty databaseCombination) = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionSkip)
	}

	resource.Spec.DatabaseCombination = "MYSQL"
	decision, err = guardAssessmentExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardAssessmentExistingBeforeCreate(valid identity) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionAllow {
		t.Fatalf("guardAssessmentExistingBeforeCreate(valid identity) = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionAllow)
	}
}

func TestBuildAssessmentCreateDetailsUsesConcretePolymorphicBodies(t *testing.T) {
	t.Parallel()

	t.Run("MYSQL", func(t *testing.T) {
		t.Parallel()

		resource := makeMySQLAssessmentResource()
		details, err := buildAssessmentCreateDetails(context.Background(), resource, resource.Namespace)
		if err != nil {
			t.Fatalf("buildAssessmentCreateDetails(MYSQL) error = %v", err)
		}

		createDetails, ok := details.(databasemigrationsdk.CreateMySqlAssessmentDetails)
		if !ok {
			t.Fatalf("create body type = %T, want databasemigration.CreateMySqlAssessmentDetails", details)
		}
		requireAssessmentStringPtr(t, "sourceDatabaseConnection.id", createDetails.SourceDatabaseConnection.Id, resource.Spec.SourceDatabaseConnection.Id)
		requireAssessmentStringPtr(t, "targetDatabaseConnection.id", createDetails.TargetDatabaseConnection.Id, resource.Spec.TargetDatabaseConnection.Id)
		if createDetails.NetworkSpeedMegabitPerSecond != databasemigrationsdk.NetworkSpeedMegabitPerSecondMbps100 {
			t.Fatalf("networkSpeedMegabitPerSecond = %q, want %q", createDetails.NetworkSpeedMegabitPerSecond, databasemigrationsdk.NetworkSpeedMegabitPerSecondMbps100)
		}

		body := assessmentSerializedRequestBody(t, databasemigrationsdk.CreateAssessmentRequest{CreateAssessmentDetails: details}, http.MethodPost, "/assessments")
		for _, want := range []string{
			`"databaseCombination":"MYSQL"`,
			`"networkSpeedMegabitPerSecond":"MBPS_100"`,
			`"acceptableDowntime":"LESS_THAN_1_HOUR"`,
			`"databaseDataSize":"GB_10_50"`,
			`"sourceDatabaseConnection":{"id":"ocid1.connection.oc1..mysql-source"}`,
		} {
			if !strings.Contains(body, want) {
				t.Fatalf("request body %s does not contain %s", body, want)
			}
		}
	})

	t.Run("ORACLE", func(t *testing.T) {
		t.Parallel()

		resource := makeOracleAssessmentResource()
		details, err := buildAssessmentCreateDetails(context.Background(), resource, resource.Namespace)
		if err != nil {
			t.Fatalf("buildAssessmentCreateDetails(ORACLE) error = %v", err)
		}

		createDetails, ok := details.(databasemigrationsdk.CreateOracleAssessmentDetails)
		if !ok {
			t.Fatalf("create body type = %T, want databasemigration.CreateOracleAssessmentDetails", details)
		}
		requireAssessmentStringPtr(t, "sourceDatabaseConnection.id", createDetails.SourceDatabaseConnection.Id, resource.Spec.SourceDatabaseConnection.Id)
		requireAssessmentStringPtr(t, "targetDatabaseConnection.id", createDetails.TargetDatabaseConnection.Id, resource.Spec.TargetDatabaseConnection.Id)

		body := assessmentSerializedRequestBody(t, databasemigrationsdk.CreateAssessmentRequest{CreateAssessmentDetails: details}, http.MethodPost, "/assessments")
		for _, want := range []string{
			`"databaseCombination":"ORACLE"`,
			`"ddlExpectation":"DDL_EXPECTED"`,
			`"creationType":"CREATE_AND_RUN_ASSESSORS"`,
			`"targetDatabaseConnection":{"id":"ocid1.connection.oc1..oracle-target"}`,
		} {
			if !strings.Contains(body, want) {
				t.Fatalf("request body %s does not contain %s", body, want)
			}
		}
	})
}

func TestBuildAssessmentCreateDetailsRejectsOutOfScopeObjectHelpers(t *testing.T) {
	t.Parallel()

	resource := makeOracleAssessmentResource()
	resource.Spec.IncludeObjects = []databasemigrationv1beta1.AssessmentIncludeObject{
		{Owner: "HR", ObjectName: "EMPLOYEES"},
	}
	resource.Spec.BulkIncludeExcludeData = "HR,EMPLOYEES,TABLE"

	if _, err := buildAssessmentCreateDetails(context.Background(), resource, resource.Namespace); err == nil {
		t.Fatal("buildAssessmentCreateDetails() error = nil, want out-of-scope helper rejection")
	} else {
		for _, want := range []string{"spec.includeObjects", "spec.bulkIncludeExcludeData"} {
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("buildAssessmentCreateDetails() error = %v, want %s in message", err, want)
			}
		}
	}
}

func TestBuildAssessmentUpdateDetailsUsesConcretePolymorphicBody(t *testing.T) {
	t.Parallel()

	resource := makeOracleAssessmentResource()
	resource.Spec.DisplayName = "oracle-assessment-updated"
	resource.Spec.Description = ""
	resource.Spec.FreeformTags = map[string]string{}

	details, updateNeeded, err := buildAssessmentUpdateDetails(
		context.Background(),
		resource,
		resource.Namespace,
		databasemigrationsdk.GetAssessmentResponse{
			Assessment: makeOracleSDKAssessment("ocid1.assessment.oc1..oracle", makeOracleAssessmentResource(), databasemigrationsdk.AssessmentLifecycleStatesActive),
		},
	)
	if err != nil {
		t.Fatalf("buildAssessmentUpdateDetails() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildAssessmentUpdateDetails() updateNeeded = false, want true after mutable drift")
	}

	updateDetails, ok := details.(databasemigrationsdk.UpdateOracleAssessmentDetails)
	if !ok {
		t.Fatalf("update body type = %T, want databasemigration.UpdateOracleAssessmentDetails", details)
	}
	requireAssessmentStringPtr(t, "displayName", updateDetails.DisplayName, resource.Spec.DisplayName)
	if len(updateDetails.FreeformTags) != 0 {
		t.Fatalf("freeformTags = %#v, want explicit empty-map clear", updateDetails.FreeformTags)
	}

	body := assessmentSerializedRequestBody(
		t,
		databasemigrationsdk.UpdateAssessmentRequest{
			AssessmentId:            common.String("ocid1.assessment.oc1..oracle"),
			UpdateAssessmentDetails: details,
		},
		http.MethodPut,
		"/assessments/ocid1.assessment.oc1..oracle",
	)
	for _, want := range []string{
		`"databaseCombination":"ORACLE"`,
		`"displayName":"oracle-assessment-updated"`,
		`"freeformTags":{}`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("request body %s does not contain %s", body, want)
		}
	}
}

func TestLookupExistingAssessmentReturnsExactMatchOnly(t *testing.T) {
	t.Parallel()

	t.Run("returns exact match", func(t *testing.T) {
		t.Parallel()

		resource := makeMySQLAssessmentResource()
		identity, err := resolveAssessmentIdentity(resource)
		if err != nil {
			t.Fatalf("resolveAssessmentIdentity() error = %v", err)
		}

		var getRequests []string
		response, err := lookupExistingAssessment(
			context.Background(),
			&fakeAssessmentOCIClient{
				listFn: func(_ context.Context, req databasemigrationsdk.ListAssessmentsRequest) (databasemigrationsdk.ListAssessmentsResponse, error) {
					requireAssessmentStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
					requireAssessmentStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
					return databasemigrationsdk.ListAssessmentsResponse{
						AssessmentCollection: databasemigrationsdk.AssessmentCollection{
							Items: []databasemigrationsdk.AssessmentSummary{
								makeMySQLAssessmentSummary("ocid1.assessment.oc1..mysql", resource, databasemigrationsdk.AssessmentLifecycleStatesActive),
							},
						},
					}, nil
				},
				getFn: func(_ context.Context, req databasemigrationsdk.GetAssessmentRequest) (databasemigrationsdk.GetAssessmentResponse, error) {
					getRequests = append(getRequests, assessmentStringPtr(req.AssessmentId))
					return databasemigrationsdk.GetAssessmentResponse{
						Assessment: makeMySQLSDKAssessment("ocid1.assessment.oc1..mysql", resource, databasemigrationsdk.AssessmentLifecycleStatesActive),
					}, nil
				},
			},
			nil,
			resource,
			identity,
		)
		if err != nil {
			t.Fatalf("lookupExistingAssessment() error = %v", err)
		}
		if len(getRequests) != 1 || getRequests[0] != "ocid1.assessment.oc1..mysql" {
			t.Fatalf("GetAssessment requests = %#v, want exact candidate fetch", getRequests)
		}
		got, ok := response.(databasemigrationsdk.GetAssessmentResponse)
		if !ok {
			t.Fatalf("lookupExistingAssessment() response type = %T, want databasemigration.GetAssessmentResponse", response)
		}
		if assessmentIDFromAny(got.Assessment) != "ocid1.assessment.oc1..mysql" {
			t.Fatalf("lookupExistingAssessment() returned id %q, want ocid1.assessment.oc1..mysql", assessmentIDFromAny(got.Assessment))
		}
	})

	t.Run("does not bind non-identical candidate", func(t *testing.T) {
		t.Parallel()

		resource := makeMySQLAssessmentResource()
		identity, err := resolveAssessmentIdentity(resource)
		if err != nil {
			t.Fatalf("resolveAssessmentIdentity() error = %v", err)
		}

		response, err := lookupExistingAssessment(
			context.Background(),
			&fakeAssessmentOCIClient{
				listFn: func(_ context.Context, _ databasemigrationsdk.ListAssessmentsRequest) (databasemigrationsdk.ListAssessmentsResponse, error) {
					return databasemigrationsdk.ListAssessmentsResponse{
						AssessmentCollection: databasemigrationsdk.AssessmentCollection{
							Items: []databasemigrationsdk.AssessmentSummary{
								makeMySQLAssessmentSummary("ocid1.assessment.oc1..other", resource, databasemigrationsdk.AssessmentLifecycleStatesActive),
							},
						},
					}, nil
				},
				getFn: func(_ context.Context, _ databasemigrationsdk.GetAssessmentRequest) (databasemigrationsdk.GetAssessmentResponse, error) {
					mismatched := makeMySQLAssessmentResource()
					mismatched.Spec.TargetDatabaseConnection.Id = "ocid1.connection.oc1..different"
					return databasemigrationsdk.GetAssessmentResponse{
						Assessment: makeMySQLSDKAssessment("ocid1.assessment.oc1..other", mismatched, databasemigrationsdk.AssessmentLifecycleStatesActive),
					}, nil
				},
			},
			nil,
			resource,
			identity,
		)
		if err != nil {
			t.Fatalf("lookupExistingAssessment() error = %v", err)
		}
		if response != nil {
			t.Fatalf("lookupExistingAssessment() = %#v, want nil for non-identical candidate", response)
		}
	})
}

func TestRecoverAssessmentIDFromGeneratedWorkRequest(t *testing.T) {
	t.Parallel()

	workRequest := databasemigrationsdk.WorkRequest{
		Id:            common.String("wr-assessment-create"),
		OperationType: databasemigrationsdk.OperationTypesCreateAssessment,
		Status:        databasemigrationsdk.OperationStatusSucceeded,
		Resources: []databasemigrationsdk.WorkRequestResource{
			{
				ActionType: databasemigrationsdk.WorkRequestResourceActionTypeRelated,
				EntityType: common.String("migration"),
				Identifier: common.String("ocid1.migration.oc1..ignore"),
			},
			{
				ActionType: databasemigrationsdk.WorkRequestResourceActionTypeCreated,
				EntityType: common.String("Assessment"),
				Identifier: common.String("ocid1.assessment.oc1..created"),
			},
		},
	}

	id, err := recoverAssessmentIDFromGeneratedWorkRequest(nil, workRequest, shared.OSOKAsyncPhaseCreate)
	if err != nil {
		t.Fatalf("recoverAssessmentIDFromGeneratedWorkRequest() error = %v", err)
	}
	if id != "ocid1.assessment.oc1..created" {
		t.Fatalf("recoverAssessmentIDFromGeneratedWorkRequest() = %q, want %q", id, "ocid1.assessment.oc1..created")
	}
}

func TestAssessmentServiceClientCreatesAndResumesWorkRequest(t *testing.T) {
	t.Parallel()

	const (
		createdID     = "ocid1.assessment.oc1..created"
		workRequestID = "wr-assessment-create"
	)

	resource := makeMySQLAssessmentResource()
	workRequests := map[string]databasemigrationsdk.WorkRequest{
		workRequestID: makeAssessmentWorkRequest(
			workRequestID,
			databasemigrationsdk.OperationTypesCreateAssessment,
			databasemigrationsdk.OperationStatusInProgress,
			databasemigrationsdk.WorkRequestResourceActionTypeInProgress,
			"",
		),
	}

	var createRequest databasemigrationsdk.CreateAssessmentRequest
	var listRequest databasemigrationsdk.ListAssessmentsRequest
	getCalls := 0

	client := newTestAssessmentClient(&fakeAssessmentOCIClient{
		listFn: func(_ context.Context, req databasemigrationsdk.ListAssessmentsRequest) (databasemigrationsdk.ListAssessmentsResponse, error) {
			listRequest = req
			return databasemigrationsdk.ListAssessmentsResponse{}, nil
		},
		createFn: func(_ context.Context, req databasemigrationsdk.CreateAssessmentRequest) (databasemigrationsdk.CreateAssessmentResponse, error) {
			createRequest = req
			return databasemigrationsdk.CreateAssessmentResponse{
				OpcWorkRequestId: common.String(workRequestID),
				OpcRequestId:     common.String("opc-create-assessment"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req databasemigrationsdk.GetWorkRequestRequest) (databasemigrationsdk.GetWorkRequestResponse, error) {
			requireAssessmentStringPtr(t, "workRequestId", req.WorkRequestId, workRequestID)
			return databasemigrationsdk.GetWorkRequestResponse{WorkRequest: workRequests[workRequestID]}, nil
		},
		getFn: func(_ context.Context, req databasemigrationsdk.GetAssessmentRequest) (databasemigrationsdk.GetAssessmentResponse, error) {
			getCalls++
			requireAssessmentStringPtr(t, "get assessmentId", req.AssessmentId, createdID)
			return databasemigrationsdk.GetAssessmentResponse{
				Assessment: makeMySQLSDKAssessment(createdID, resource, databasemigrationsdk.AssessmentLifecycleStatesSucceeded),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful pending create", response)
	}
	requireAssessmentStringPtr(t, "list compartmentId", listRequest.CompartmentId, resource.Spec.CompartmentId)
	requireAssessmentStringPtr(t, "list displayName", listRequest.DisplayName, resource.Spec.DisplayName)
	if listRequest.LifecycleState != "" {
		t.Fatalf("list lifecycleState = %q, want empty reviewed lookup filter", listRequest.LifecycleState)
	}
	if getCalls != 0 {
		t.Fatalf("GetAssessment() calls = %d, want 0 while work request is pending", getCalls)
	}
	requireAssessmentAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, workRequestID, shared.OSOKAsyncClassPending)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-assessment" {
		t.Fatalf("status.opcRequestId = %q, want opc-create-assessment", got)
	}

	body := assessmentSerializedRequestBody(t, createRequest, http.MethodPost, "/assessments")
	for _, want := range []string{
		`"databaseCombination":"MYSQL"`,
		`"networkSpeedMegabitPerSecond":"MBPS_100"`,
		`"targetDatabaseConnection":{"id":"ocid1.connection.oc1..mysql-target"}`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("create request body %s does not contain %s", body, want)
		}
	}

	workRequests[workRequestID] = makeAssessmentWorkRequest(
		workRequestID,
		databasemigrationsdk.OperationTypesCreateAssessment,
		databasemigrationsdk.OperationStatusSucceeded,
		databasemigrationsdk.WorkRequestResourceActionTypeCreated,
		createdID,
	)

	response, err = client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() after work request success error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() after work request success response = %#v, want converged success", response)
	}
	if getCalls != 1 {
		t.Fatalf("GetAssessment() calls = %d, want 1 follow-up read", getCalls)
	}
	if got := resource.Status.Id; got != createdID {
		t.Fatalf("status.id = %q, want %q", got, createdID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != createdID {
		t.Fatalf("status.ocid = %q, want %q", got, createdID)
	}
	if got := resource.Status.DatabaseCombination; got != "MYSQL" {
		t.Fatalf("status.databaseCombination = %q, want MYSQL", got)
	}
	if got := resource.Status.LifecycleState; got != string(databasemigrationsdk.AssessmentLifecycleStatesSucceeded) {
		t.Fatalf("status.lifecycleState = %q, want %q", got, databasemigrationsdk.AssessmentLifecycleStatesSucceeded)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after successful reconciliation", resource.Status.OsokStatus.Async.Current)
	}
}

func TestAssessmentServiceClientRejectsOutOfScopeObjectHelpersBeforeOCI(t *testing.T) {
	t.Parallel()

	resource := makeOracleAssessmentResource()
	resource.Spec.ExcludeObjects = []databasemigrationv1beta1.AssessmentExcludeObject{
		{Owner: "HR", ObjectName: "EMPLOYEES"},
	}

	createCalled := false
	client := newTestAssessmentClient(&fakeAssessmentOCIClient{
		listFn: func(_ context.Context, _ databasemigrationsdk.ListAssessmentsRequest) (databasemigrationsdk.ListAssessmentsResponse, error) {
			return databasemigrationsdk.ListAssessmentsResponse{}, nil
		},
		createFn: func(_ context.Context, _ databasemigrationsdk.CreateAssessmentRequest) (databasemigrationsdk.CreateAssessmentResponse, error) {
			createCalled = true
			return databasemigrationsdk.CreateAssessmentResponse{}, nil
		},
	})

	if _, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{}); err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want out-of-scope helper rejection")
	} else if !strings.Contains(err.Error(), "spec.excludeObjects") {
		t.Fatalf("CreateOrUpdate() error = %v, want spec.excludeObjects rejection", err)
	}
	if createCalled {
		t.Fatal("CreateAssessment() called after out-of-scope helper rejection")
	}
}

func makeMySQLAssessmentResource() *databasemigrationv1beta1.Assessment {
	return &databasemigrationv1beta1.Assessment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mysql-assessment",
			Namespace: "default",
		},
		Spec: databasemigrationv1beta1.AssessmentSpec{
			DisplayName:                  "mysql-assessment",
			CompartmentId:                "ocid1.compartment.oc1..mysql",
			DatabaseCombination:          "MYSQL",
			NetworkSpeedMegabitPerSecond: "MBPS_100",
			AcceptableDowntime:           "LESS_THAN_1_HOUR",
			DatabaseDataSize:             "GB_10_50",
			DdlExpectation:               "DDL_NOT_EXPECTED",
			CreationType:                 "CREATE_ONLY",
			Description:                  "mysql assessment",
			SourceDatabaseConnection: databasemigrationv1beta1.AssessmentSourceDatabaseConnection{
				Id: "ocid1.connection.oc1..mysql-source",
			},
			TargetDatabaseConnection: databasemigrationv1beta1.AssessmentTargetDatabaseConnection{
				Id: "ocid1.connection.oc1..mysql-target",
			},
			FreeformTags: map[string]string{"env": "test"},
			DefinedTags:  map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func makeOracleAssessmentResource() *databasemigrationv1beta1.Assessment {
	return &databasemigrationv1beta1.Assessment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "oracle-assessment",
			Namespace: "default",
		},
		Spec: databasemigrationv1beta1.AssessmentSpec{
			DisplayName:                  "oracle-assessment",
			CompartmentId:                "ocid1.compartment.oc1..oracle",
			DatabaseCombination:          "ORACLE",
			NetworkSpeedMegabitPerSecond: "MBPS_1000",
			AcceptableDowntime:           "LESS_THAN_4_HOURS",
			DatabaseDataSize:             "TB_1_3",
			DdlExpectation:               "DDL_EXPECTED",
			CreationType:                 "CREATE_AND_RUN_ASSESSORS",
			Description:                  "oracle assessment",
			SourceDatabaseConnection: databasemigrationv1beta1.AssessmentSourceDatabaseConnection{
				Id: "ocid1.connection.oc1..oracle-source",
			},
			TargetDatabaseConnection: databasemigrationv1beta1.AssessmentTargetDatabaseConnection{
				Id: "ocid1.connection.oc1..oracle-target",
			},
			FreeformTags: map[string]string{"env": "test"},
			DefinedTags:  map[string]shared.MapValue{"Operations": {"CostCenter": "84"}},
		},
	}
}

func makeMySQLSDKAssessment(
	id string,
	resource *databasemigrationv1beta1.Assessment,
	state databasemigrationsdk.AssessmentLifecycleStatesEnum,
) databasemigrationsdk.MySqlAssessment {
	now := &common.SDKTime{Time: time.Unix(1713240000, 0).UTC()}
	return databasemigrationsdk.MySqlAssessment{
		Id:                           common.String(id),
		DisplayName:                  common.String(resource.Spec.DisplayName),
		CompartmentId:                common.String(resource.Spec.CompartmentId),
		TimeCreated:                  now,
		TimeUpdated:                  now,
		Description:                  common.String(resource.Spec.Description),
		FreeformTags:                 map[string]string{"env": "test"},
		DefinedTags:                  map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		SourceDatabaseConnection:     &databasemigrationsdk.SourceAssessmentConnection{Id: common.String(resource.Spec.SourceDatabaseConnection.Id)},
		TargetDatabaseConnection:     &databasemigrationsdk.TargetAssessmentConnection{Id: common.String(resource.Spec.TargetDatabaseConnection.Id)},
		NetworkSpeedMegabitPerSecond: databasemigrationsdk.NetworkSpeedMegabitPerSecondMbps100,
		AcceptableDowntime:           databasemigrationsdk.AcceptableDowntimeLessThan1Hour,
		DatabaseDataSize:             databasemigrationsdk.DatabaseDataSizeGb1050,
		DdlExpectation:               databasemigrationsdk.DdlExpectationDdlNotExpected,
		CreationType:                 databasemigrationsdk.CreationTypeCreateOnly,
		LifecycleState:               state,
	}
}

func makeOracleSDKAssessment(
	id string,
	resource *databasemigrationv1beta1.Assessment,
	state databasemigrationsdk.AssessmentLifecycleStatesEnum,
) databasemigrationsdk.OracleAssessment {
	now := &common.SDKTime{Time: time.Unix(1713240000, 0).UTC()}
	isCdbSupported := true
	return databasemigrationsdk.OracleAssessment{
		Id:                           common.String(id),
		DisplayName:                  common.String(resource.Spec.DisplayName),
		CompartmentId:                common.String(resource.Spec.CompartmentId),
		TimeCreated:                  now,
		TimeUpdated:                  now,
		Description:                  common.String(resource.Spec.Description),
		FreeformTags:                 map[string]string{"env": "test"},
		DefinedTags:                  map[string]map[string]interface{}{"Operations": {"CostCenter": "84"}},
		SourceDatabaseConnection:     &databasemigrationsdk.SourceAssessmentConnection{Id: common.String(resource.Spec.SourceDatabaseConnection.Id)},
		TargetDatabaseConnection:     &databasemigrationsdk.TargetAssessmentConnection{Id: common.String(resource.Spec.TargetDatabaseConnection.Id)},
		NetworkSpeedMegabitPerSecond: databasemigrationsdk.NetworkSpeedMegabitPerSecondMbps1000,
		AcceptableDowntime:           databasemigrationsdk.AcceptableDowntimeLessThan4Hours,
		DatabaseDataSize:             databasemigrationsdk.DatabaseDataSizeTb13,
		DdlExpectation:               databasemigrationsdk.DdlExpectationDdlExpected,
		CreationType:                 databasemigrationsdk.CreationTypeCreateAndRunAssessors,
		LifecycleState:               state,
		IsCdbSupported:               &isCdbSupported,
	}
}

func makeMySQLAssessmentSummary(
	id string,
	resource *databasemigrationv1beta1.Assessment,
	state databasemigrationsdk.AssessmentLifecycleStatesEnum,
) databasemigrationsdk.MySqlAssessmentSummary {
	now := &common.SDKTime{Time: time.Unix(1713240000, 0).UTC()}
	return databasemigrationsdk.MySqlAssessmentSummary{
		Id:             common.String(id),
		DisplayName:    common.String(resource.Spec.DisplayName),
		CompartmentId:  common.String(resource.Spec.CompartmentId),
		TimeCreated:    now,
		TimeUpdated:    now,
		FreeformTags:   map[string]string{"env": "test"},
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		LifecycleState: state,
	}
}

func makeAssessmentWorkRequest(
	id string,
	operation databasemigrationsdk.OperationTypesEnum,
	status databasemigrationsdk.OperationStatusEnum,
	action databasemigrationsdk.WorkRequestResourceActionTypeEnum,
	resourceID string,
) databasemigrationsdk.WorkRequest {
	now := &common.SDKTime{Time: time.Unix(1713240000, 0).UTC()}
	resources := []databasemigrationsdk.WorkRequestResource{
		{
			ActionType: action,
			EntityType: common.String("Assessment"),
			Identifier: common.String(resourceID),
		},
	}
	return databasemigrationsdk.WorkRequest{
		Id:              common.String(id),
		CompartmentId:   common.String("ocid1.compartment.oc1..workrequest"),
		OperationType:   operation,
		Status:          status,
		Resources:       resources,
		PercentComplete: common.Float32(25),
		TimeAccepted:    now,
	}
}

func assessmentSerializedRequestBody(
	t *testing.T,
	request assessmentRequestBodyBuilder,
	method string,
	path string,
) string {
	t.Helper()

	httpRequest, err := request.HTTPRequest(method, path, nil, nil)
	if err != nil {
		t.Fatalf("HTTPRequest() error = %v", err)
	}

	body, err := io.ReadAll(httpRequest.Body)
	if err != nil {
		t.Fatalf("ReadAll(request body) error = %v", err)
	}
	return string(body)
}

func requireAssessmentStringPtr(t *testing.T, field string, actual *string, want string) {
	t.Helper()
	if actual == nil {
		t.Fatalf("%s = nil, want %q", field, want)
	}
	if *actual != want {
		t.Fatalf("%s = %q, want %q", field, *actual, want)
	}
}

func requireAssessmentAsyncCurrent(
	t *testing.T,
	resource *databasemigrationv1beta1.Assessment,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	class shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()

	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil")
	}
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		t.Fatalf("status.async.current.source = %q, want %q", current.Source, shared.OSOKAsyncSourceWorkRequest)
	}
	if current.Phase != phase {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, phase)
	}
	if current.WorkRequestID != workRequestID {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", current.WorkRequestID, workRequestID)
	}
	if current.NormalizedClass != class {
		t.Fatalf("status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, class)
	}
}

func assertAssessmentStringSliceEqual(t *testing.T, field string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", field, got, want)
	}
}

func assertAssessmentStringSliceContainsAll(t *testing.T, field string, got []string, want ...string) {
	t.Helper()
	for _, candidate := range want {
		if !containsAssessmentString(got, candidate) {
			t.Fatalf("%s = %#v, want %q included", field, got, candidate)
		}
	}
}

func containsAssessmentString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

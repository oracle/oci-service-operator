/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package reportdefinition

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/shared"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testReportDefinitionID            = "ocid1.datasafereportdefinition.oc1..report"
	testOtherReportDefinitionID       = "ocid1.datasafereportdefinition.oc1..other"
	testReportDefinitionCompartmentID = "ocid1.compartment.oc1..report"
	testReportDefinitionParentID      = "ocid1.datasafereportdefinition.oc1..parent"
	testOtherReportDefinitionParentID = "ocid1.datasafereportdefinition.oc1..otherparent"
	testReportDefinitionDisplayName   = "monthly-audit"
)

type fakeReportDefinitionOCIClient struct {
	resources map[string]datasafesdk.ReportDefinition

	createRequests []datasafesdk.CreateReportDefinitionRequest
	getRequests    []datasafesdk.GetReportDefinitionRequest
	listRequests   []datasafesdk.ListReportDefinitionsRequest
	updateRequests []datasafesdk.UpdateReportDefinitionRequest
	deleteRequests []datasafesdk.DeleteReportDefinitionRequest

	getResults    []reportDefinitionGetResult
	listResponses []datasafesdk.ListReportDefinitionsResponse

	createErr error
	listErr   error
	updateErr error
	deleteErr error
}

type reportDefinitionGetResult struct {
	response datasafesdk.GetReportDefinitionResponse
	err      error
}

func (f *fakeReportDefinitionOCIClient) CreateReportDefinition(
	_ context.Context,
	request datasafesdk.CreateReportDefinitionRequest,
) (datasafesdk.CreateReportDefinitionResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createErr != nil {
		return datasafesdk.CreateReportDefinitionResponse{}, f.createErr
	}
	resource := reportDefinitionFromCreateDetails(
		testReportDefinitionID,
		request.CreateReportDefinitionDetails,
		datasafesdk.ReportDefinitionLifecycleStateCreating,
	)
	f.ensureResources()[testReportDefinitionID] = resource
	return datasafesdk.CreateReportDefinitionResponse{
		ReportDefinition: resource,
		OpcWorkRequestId: common.String("wr-create-1"),
		OpcRequestId:     common.String("opc-create-1"),
	}, nil
}

func (f *fakeReportDefinitionOCIClient) GetReportDefinition(
	_ context.Context,
	request datasafesdk.GetReportDefinitionRequest,
) (datasafesdk.GetReportDefinitionResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if len(f.getResults) > 0 {
		result := f.getResults[0]
		f.getResults = f.getResults[1:]
		return result.response, result.err
	}
	id := reportDefinitionStringValue(request.ReportDefinitionId)
	resource, ok := f.resources[id]
	if !ok {
		return datasafesdk.GetReportDefinitionResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "not found")
	}
	return datasafesdk.GetReportDefinitionResponse{
		ReportDefinition: resource,
		OpcRequestId:     common.String("opc-get-1"),
	}, nil
}

func (f *fakeReportDefinitionOCIClient) ListReportDefinitions(
	_ context.Context,
	request datasafesdk.ListReportDefinitionsRequest,
) (datasafesdk.ListReportDefinitionsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listErr != nil {
		return datasafesdk.ListReportDefinitionsResponse{}, f.listErr
	}
	if len(f.listResponses) > 0 {
		response := f.listResponses[0]
		f.listResponses = f.listResponses[1:]
		return response, nil
	}
	var items []datasafesdk.ReportDefinitionSummary
	for _, resource := range f.resources {
		if reportDefinitionMatchesListRequest(resource, request) {
			items = append(items, reportDefinitionSummaryFromSDK(resource))
		}
	}
	return datasafesdk.ListReportDefinitionsResponse{
		ReportDefinitionCollection: datasafesdk.ReportDefinitionCollection{Items: items},
		OpcRequestId:               common.String("opc-list-1"),
	}, nil
}

func (f *fakeReportDefinitionOCIClient) UpdateReportDefinition(
	_ context.Context,
	request datasafesdk.UpdateReportDefinitionRequest,
) (datasafesdk.UpdateReportDefinitionResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateErr != nil {
		return datasafesdk.UpdateReportDefinitionResponse{}, f.updateErr
	}
	return datasafesdk.UpdateReportDefinitionResponse{
		OpcWorkRequestId: common.String("wr-update-1"),
		OpcRequestId:     common.String("opc-update-1"),
	}, nil
}

func (f *fakeReportDefinitionOCIClient) DeleteReportDefinition(
	_ context.Context,
	request datasafesdk.DeleteReportDefinitionRequest,
) (datasafesdk.DeleteReportDefinitionResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteErr != nil {
		return datasafesdk.DeleteReportDefinitionResponse{}, f.deleteErr
	}
	return datasafesdk.DeleteReportDefinitionResponse{
		OpcWorkRequestId: common.String("wr-delete-1"),
		OpcRequestId:     common.String("opc-delete-1"),
	}, nil
}

func (f *fakeReportDefinitionOCIClient) ensureResources() map[string]datasafesdk.ReportDefinition {
	if f.resources == nil {
		f.resources = map[string]datasafesdk.ReportDefinition{}
	}
	return f.resources
}

func TestReportDefinitionRuntimeHooksConfigured(t *testing.T) {
	hooks := newReportDefinitionDefaultRuntimeHooks(datasafesdk.DataSafeClient{})
	applyReportDefinitionRuntimeHooks(&hooks)

	checks := []struct {
		name string
		ok   bool
	}{
		{name: "Semantics", ok: hooks.Semantics != nil},
		{name: "BuildCreateBody", ok: hooks.BuildCreateBody != nil},
		{name: "BuildUpdateBody", ok: hooks.BuildUpdateBody != nil},
		{name: "Identity.Resolve", ok: hooks.Identity.Resolve != nil},
		{name: "Identity.LookupExisting", ok: hooks.Identity.LookupExisting != nil},
		{name: "DeleteHooks.ConfirmRead", ok: hooks.DeleteHooks.ConfirmRead != nil},
		{name: "DeleteHooks.HandleError", ok: hooks.DeleteHooks.HandleError != nil},
		{name: "DeleteHooks.ApplyOutcome", ok: hooks.DeleteHooks.ApplyOutcome != nil},
		{name: "StatusHooks.ProjectStatus", ok: hooks.StatusHooks.ProjectStatus != nil},
		{name: "WrapGeneratedClient", ok: len(hooks.WrapGeneratedClient) > 0},
	}
	for _, check := range checks {
		if !check.ok {
			t.Fatalf("hooks.%s not configured", check.name)
		}
	}
	if got := hooks.Semantics.FinalizerPolicy; got != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got)
	}
	assertContainsAll(t, "Mutation.Mutable", hooks.Semantics.Mutation.Mutable, "displayName", "columnInfo", "columnFilters", "columnSortings", "summary", "description", "freeformTags", "definedTags")
	assertContainsAll(t, "Mutation.ForceNew", hooks.Semantics.Mutation.ForceNew, "compartmentId", "parentId")
	assertContainsAll(t, "List.MatchFields", hooks.Semantics.List.MatchFields, "compartmentId", "displayName", "parentId")

	body, err := hooks.BuildCreateBody(context.Background(), makeReportDefinitionResource(), "default")
	if err != nil {
		t.Fatalf("BuildCreateBody() error = %v", err)
	}
	details, ok := body.(datasafesdk.CreateReportDefinitionDetails)
	if !ok {
		t.Fatalf("BuildCreateBody() type = %T, want CreateReportDefinitionDetails", body)
	}
	requireStringPtr(t, "CreateReportDefinitionDetails.ParentId", details.ParentId, testReportDefinitionParentID)
	requireBoolPtr(t, "CreateReportDefinitionDetails.ColumnInfo[0].IsHidden", details.ColumnInfo[0].IsHidden, false)
	requireBoolPtr(t, "CreateReportDefinitionDetails.ColumnFilters[0].IsEnabled", details.ColumnFilters[0].IsEnabled, true)
	requireBoolPtr(t, "CreateReportDefinitionDetails.Summary[0].IsHidden", details.Summary[0].IsHidden, false)
}

func TestReportDefinitionCreateRecordsIdentityRequestIDAndLifecycle(t *testing.T) {
	client := &fakeReportDefinitionOCIClient{}
	serviceClient := newTestReportDefinitionClient(client)
	resource := makeReportDefinitionResource()

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, reportDefinitionRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = false, want requeue while CREATING")
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("create requests = %d, want 1", len(client.createRequests))
	}
	if len(client.getRequests) != 1 {
		t.Fatalf("get requests = %d, want 1 follow-up readback", len(client.getRequests))
	}
	assertReportDefinitionRecordedID(t, resource, testReportDefinitionID)
	if got := resource.Status.ParentId; got != testReportDefinitionParentID {
		t.Fatalf("status.parentId = %q, want %q", got, testReportDefinitionParentID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-1", got)
	}
	if resource.Status.OsokStatus.Async.Current == nil ||
		resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseCreate {
		t.Fatalf("status.status.async.current = %#v, want create lifecycle tracker", resource.Status.OsokStatus.Async.Current)
	}
}

func TestReportDefinitionCreateOrUpdateBindsExistingFromLaterListPageWithParentMatch(t *testing.T) {
	resource := makeReportDefinitionResource()
	existing := sdkReportDefinition(resource, testReportDefinitionID, datasafesdk.ReportDefinitionLifecycleStateActive)
	otherResource := makeReportDefinitionResource()
	otherResource.Spec.ParentId = testOtherReportDefinitionParentID
	other := sdkReportDefinition(otherResource, testOtherReportDefinitionID, datasafesdk.ReportDefinitionLifecycleStateActive)
	client := &fakeReportDefinitionOCIClient{
		resources: map[string]datasafesdk.ReportDefinition{
			testOtherReportDefinitionID: other,
			testReportDefinitionID:      existing,
		},
		listResponses: []datasafesdk.ListReportDefinitionsResponse{
			{
				ReportDefinitionCollection: datasafesdk.ReportDefinitionCollection{
					Items: []datasafesdk.ReportDefinitionSummary{reportDefinitionSummaryFromSDK(other)},
				},
				OpcNextPage: common.String("page-2"),
			},
			{
				ReportDefinitionCollection: datasafesdk.ReportDefinitionCollection{
					Items: []datasafesdk.ReportDefinitionSummary{reportDefinitionSummaryFromSDK(existing)},
				},
			},
		},
	}
	serviceClient := newTestReportDefinitionClient(client)

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, reportDefinitionRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("create requests = %d, want 0 for bind", len(client.createRequests))
	}
	if len(client.listRequests) != 2 {
		t.Fatalf("list requests = %d, want 2 paginated lookup calls", len(client.listRequests))
	}
	if got := reportDefinitionStringValue(client.listRequests[1].Page); got != "page-2" {
		t.Fatalf("second list page = %q, want page-2", got)
	}
	assertReportDefinitionRecordedID(t, resource, testReportDefinitionID)
	if got := resource.Status.ParentId; got != testReportDefinitionParentID {
		t.Fatalf("status.parentId = %q, want %q", got, testReportDefinitionParentID)
	}
}

func TestReportDefinitionCreateOrUpdateNoopsWhenObservedStateMatches(t *testing.T) {
	resource := makeReportDefinitionResource()
	resource.Status.Id = testReportDefinitionID
	resource.Status.CompartmentId = testReportDefinitionCompartmentID
	resource.Status.ParentId = testReportDefinitionParentID
	resource.Status.OsokStatus.Ocid = shared.OCID(testReportDefinitionID)
	client := &fakeReportDefinitionOCIClient{
		resources: map[string]datasafesdk.ReportDefinition{
			testReportDefinitionID: sdkReportDefinition(resource, testReportDefinitionID, datasafesdk.ReportDefinitionLifecycleStateActive),
		},
	}
	serviceClient := newTestReportDefinitionClient(client)

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, reportDefinitionRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("create requests = %d, want 0 for tracked no-op", len(client.createRequests))
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0 when state matches", len(client.updateRequests))
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestReportDefinitionCreateOrUpdateMutableUpdate(t *testing.T) {
	resource := makeReportDefinitionResource()
	resource.Status.Id = testReportDefinitionID
	resource.Status.CompartmentId = testReportDefinitionCompartmentID
	resource.Status.ParentId = testReportDefinitionParentID
	resource.Status.OsokStatus.Ocid = shared.OCID(testReportDefinitionID)

	current := sdkReportDefinition(resource, testReportDefinitionID, datasafesdk.ReportDefinitionLifecycleStateActive)
	current.DisplayName = common.String("old-report")
	current.Description = common.String("old description")
	updated := sdkReportDefinition(resource, testReportDefinitionID, datasafesdk.ReportDefinitionLifecycleStateActive)
	updated.DisplayName = common.String("updated-report")
	updated.Description = common.String("")
	updated.FreeformTags = map[string]string{"env": "prod"}

	client := &fakeReportDefinitionOCIClient{
		getResults: []reportDefinitionGetResult{
			{response: datasafesdk.GetReportDefinitionResponse{ReportDefinition: current, OpcRequestId: common.String("opc-get-current")}},
			{response: datasafesdk.GetReportDefinitionResponse{ReportDefinition: updated, OpcRequestId: common.String("opc-get-updated")}},
		},
	}
	serviceClient := newTestReportDefinitionClient(client)
	resource.Spec.DisplayName = "updated-report"
	resource.Spec.Description = ""
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, reportDefinitionRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if len(client.updateRequests) != 1 {
		t.Fatalf("update requests = %d, want 1", len(client.updateRequests))
	}
	request := client.updateRequests[0]
	requireStringPtr(t, "UpdateReportDefinitionRequest.ReportDefinitionId", request.ReportDefinitionId, testReportDefinitionID)
	requireStringPtr(t, "UpdateReportDefinitionDetails.DisplayName", request.DisplayName, "updated-report")
	requireStringPtr(t, "UpdateReportDefinitionDetails.Description", request.Description, "")
	if got := request.FreeformTags["env"]; got != "prod" {
		t.Fatalf("UpdateReportDefinitionDetails.FreeformTags[env] = %q, want prod", got)
	}
	if got := resource.Status.DisplayName; got != "updated-report" {
		t.Fatalf("status.displayName = %q, want updated-report", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update-1", got)
	}
}

func TestReportDefinitionTrackedParentDriftRejectedBeforeOCI(t *testing.T) {
	resource := makeReportDefinitionResource()
	resource.Status.Id = testReportDefinitionID
	resource.Status.CompartmentId = testReportDefinitionCompartmentID
	resource.Status.ParentId = testReportDefinitionParentID
	resource.Status.OsokStatus.Ocid = shared.OCID(testReportDefinitionID)
	resource.Spec.ParentId = testOtherReportDefinitionParentID
	client := &fakeReportDefinitionOCIClient{}
	serviceClient := newTestReportDefinitionClient(client)

	_, err := serviceClient.CreateOrUpdate(context.Background(), resource, reportDefinitionRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want parentId drift rejection")
	}
	if !strings.Contains(err.Error(), "parentId") {
		t.Fatalf("CreateOrUpdate() error = %v, want parentId drift detail", err)
	}
	if len(client.getRequests) != 0 || len(client.listRequests) != 0 || len(client.createRequests) != 0 ||
		len(client.updateRequests) != 0 || len(client.deleteRequests) != 0 {
		t.Fatalf("OCI calls after drift rejection: get=%d list=%d create=%d update=%d delete=%d",
			len(client.getRequests), len(client.listRequests), len(client.createRequests), len(client.updateRequests), len(client.deleteRequests))
	}
	requireLastCondition(t, resource, shared.Failed)
}

func TestReportDefinitionDeleteRetainsFinalizerWhileReadbackStillActive(t *testing.T) {
	resource := makeReportDefinitionResource()
	resource.Status.Id = testReportDefinitionID
	resource.Status.CompartmentId = testReportDefinitionCompartmentID
	resource.Status.ParentId = testReportDefinitionParentID
	resource.Status.OsokStatus.Ocid = shared.OCID(testReportDefinitionID)
	client := &fakeReportDefinitionOCIClient{
		resources: map[string]datasafesdk.ReportDefinition{
			testReportDefinitionID: sdkReportDefinition(resource, testReportDefinitionID, datasafesdk.ReportDefinitionLifecycleStateActive),
		},
	}
	serviceClient := newTestReportDefinitionClient(client)

	deleted, err := serviceClient.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while readback is still ACTIVE")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("delete requests = %d, want 1", len(client.deleteRequests))
	}
	if resource.Status.OsokStatus.Async.Current == nil ||
		resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.status.async.current = %#v, want delete lifecycle tracker", resource.Status.OsokStatus.Async.Current)
	}
	requireLastCondition(t, resource, shared.Terminating)
}

func TestReportDefinitionDeleteReleasesFinalizerWhenUntrackedLookupHasNoMatches(t *testing.T) {
	resource := makeReportDefinitionResource()
	client := &fakeReportDefinitionOCIClient{
		listResponses: []datasafesdk.ListReportDefinitionsResponse{
			{
				ReportDefinitionCollection: datasafesdk.ReportDefinitionCollection{},
				OpcNextPage:                common.String("page-2"),
			},
			{
				ReportDefinitionCollection: datasafesdk.ReportDefinitionCollection{},
			},
		},
	}
	serviceClient := newTestReportDefinitionClient(client)

	deleted, err := serviceClient.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after paginated zero-match confirmation")
	}
	if len(client.listRequests) != 2 {
		t.Fatalf("list requests = %d, want 2 paginated lookup calls", len(client.listRequests))
	}
	if got := reportDefinitionStringValue(client.listRequests[1].Page); got != "page-2" {
		t.Fatalf("second list page = %q, want page-2", got)
	}
	if len(client.getRequests) != 0 {
		t.Fatalf("get requests = %d, want 0 when list returns no candidates", len(client.getRequests))
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("delete requests = %d, want 0 when lookup confirms absence", len(client.deleteRequests))
	}
	requireDeleted(t, resource)
}

func TestReportDefinitionDeleteResolvesExistingWithoutTrackedIDFromLaterListPageWithParentMatch(t *testing.T) {
	resource := makeReportDefinitionResource()
	existing := sdkReportDefinition(resource, testReportDefinitionID, datasafesdk.ReportDefinitionLifecycleStateActive)
	otherResource := makeReportDefinitionResource()
	otherResource.Spec.ParentId = testOtherReportDefinitionParentID
	other := sdkReportDefinition(otherResource, testOtherReportDefinitionID, datasafesdk.ReportDefinitionLifecycleStateActive)
	client := &fakeReportDefinitionOCIClient{
		resources: map[string]datasafesdk.ReportDefinition{
			testOtherReportDefinitionID: other,
			testReportDefinitionID:      existing,
		},
		listResponses: []datasafesdk.ListReportDefinitionsResponse{
			{
				ReportDefinitionCollection: datasafesdk.ReportDefinitionCollection{
					Items: []datasafesdk.ReportDefinitionSummary{reportDefinitionSummaryFromSDK(other)},
				},
				OpcNextPage: common.String("page-2"),
			},
			{
				ReportDefinitionCollection: datasafesdk.ReportDefinitionCollection{
					Items: []datasafesdk.ReportDefinitionSummary{reportDefinitionSummaryFromSDK(existing)},
				},
			},
		},
	}
	serviceClient := newTestReportDefinitionClient(client)

	deleted, err := serviceClient.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while confirmed readback is still ACTIVE")
	}
	if len(client.listRequests) != 2 {
		t.Fatalf("list requests = %d, want 2 paginated lookup calls", len(client.listRequests))
	}
	if got := reportDefinitionStringValue(client.listRequests[1].Page); got != "page-2" {
		t.Fatalf("second list page = %q, want page-2", got)
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("delete requests = %d, want 1 after resolving existing resource", len(client.deleteRequests))
	}
	requireStringPtr(t, "DeleteReportDefinitionRequest.ReportDefinitionId", client.deleteRequests[0].ReportDefinitionId, testReportDefinitionID)
	assertReportDefinitionRecordedID(t, resource, testReportDefinitionID)
	if got := resource.Status.ParentId; got != testReportDefinitionParentID {
		t.Fatalf("status.parentId = %q, want %q", got, testReportDefinitionParentID)
	}
	requireLastCondition(t, resource, shared.Terminating)
}

func TestReportDefinitionDeleteRetainsFinalizerOnUntrackedAuthShapedNotFound(t *testing.T) {
	resource := makeReportDefinitionResource()
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	client := &fakeReportDefinitionOCIClient{listErr: authErr}
	serviceClient := newTestReportDefinitionClient(client)

	deleted, err := serviceClient.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous not-found rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("delete requests = %d, want 0 after pre-delete auth-shaped 404", len(client.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %v, want nil", resource.Status.OsokStatus.DeletedAt)
	}
	if !strings.Contains(err.Error(), "NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want NotAuthorizedOrNotFound detail", err)
	}
}

func TestReportDefinitionDeleteRetainsFinalizerOnUntrackedDuplicateMatches(t *testing.T) {
	resource := makeReportDefinitionResource()
	existing := sdkReportDefinition(resource, testReportDefinitionID, datasafesdk.ReportDefinitionLifecycleStateActive)
	duplicate := sdkReportDefinition(resource, testOtherReportDefinitionID, datasafesdk.ReportDefinitionLifecycleStateActive)
	client := &fakeReportDefinitionOCIClient{
		resources: map[string]datasafesdk.ReportDefinition{
			testReportDefinitionID:      existing,
			testOtherReportDefinitionID: duplicate,
		},
		listResponses: []datasafesdk.ListReportDefinitionsResponse{
			{
				ReportDefinitionCollection: datasafesdk.ReportDefinitionCollection{
					Items: []datasafesdk.ReportDefinitionSummary{
						reportDefinitionSummaryFromSDK(existing),
						reportDefinitionSummaryFromSDK(duplicate),
					},
				},
			},
		},
	}
	serviceClient := newTestReportDefinitionClient(client)

	deleted, err := serviceClient.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want duplicate match rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("delete requests = %d, want 0 when lookup is ambiguous", len(client.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %v, want nil", resource.Status.OsokStatus.DeletedAt)
	}
	if !strings.Contains(err.Error(), "multiple OCI report definitions") {
		t.Fatalf("Delete() error = %v, want duplicate match detail", err)
	}
}

func TestReportDefinitionDeleteRejectsAuthShapedNotFound(t *testing.T) {
	resource := makeReportDefinitionResource()
	resource.Status.Id = testReportDefinitionID
	resource.Status.CompartmentId = testReportDefinitionCompartmentID
	resource.Status.ParentId = testReportDefinitionParentID
	resource.Status.OsokStatus.Ocid = shared.OCID(testReportDefinitionID)
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	client := &fakeReportDefinitionOCIClient{
		getResults: []reportDefinitionGetResult{{err: authErr}},
	}
	serviceClient := newTestReportDefinitionClient(client)

	deleted, err := serviceClient.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous not-found rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("delete requests = %d, want 0 after pre-delete auth-shaped 404", len(client.deleteRequests))
	}
	if !strings.Contains(err.Error(), "NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want NotAuthorizedOrNotFound detail", err)
	}
}

func TestReportDefinitionCreateErrorRecordsOpcRequestID(t *testing.T) {
	resource := makeReportDefinitionResource()
	createErr := errortest.NewServiceError(500, "InternalError", "create failed")
	client := &fakeReportDefinitionOCIClient{createErr: createErr}
	serviceClient := newTestReportDefinitionClient(client)

	_, err := serviceClient.CreateOrUpdate(context.Background(), resource, reportDefinitionRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create error")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
	requireLastCondition(t, resource, shared.Failed)
}

func newTestReportDefinitionClient(client reportDefinitionOCIClient) ReportDefinitionServiceClient {
	return newReportDefinitionServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
}

func makeReportDefinitionResource() *datasafev1beta1.ReportDefinition {
	return &datasafev1beta1.ReportDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "report-definition",
			Namespace: "default",
		},
		Spec: datasafev1beta1.ReportDefinitionSpec{
			CompartmentId: testReportDefinitionCompartmentID,
			DisplayName:   testReportDefinitionDisplayName,
			ParentId:      testReportDefinitionParentID,
			ColumnInfo: []datasafev1beta1.ReportDefinitionColumnInfo{{
				DisplayName:         "User",
				FieldName:           "userName",
				IsHidden:            false,
				DisplayOrder:        1,
				DataType:            "STRING",
				IsVirtual:           false,
				ApplicableOperators: []string{"EQ", "IN"},
			}},
			ColumnFilters: []datasafev1beta1.ReportDefinitionColumnFilter{{
				FieldName:   "userName",
				Operator:    "EQ",
				Expressions: []string{"app"},
				IsEnabled:   true,
				IsHidden:    false,
			}},
			ColumnSortings: []datasafev1beta1.ReportDefinitionColumnSorting{{
				FieldName:    "userName",
				IsAscending:  true,
				SortingOrder: 1,
			}},
			Summary: []datasafev1beta1.ReportDefinitionSummary{{
				Name:             "Users",
				DisplayOrder:     1,
				IsHidden:         false,
				GroupByFieldName: "userName",
				CountOf:          "id",
			}},
			Description:  "initial report",
			FreeformTags: map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func reportDefinitionRequest(resource *datasafev1beta1.ReportDefinition) ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Name: resource.Name, Namespace: resource.Namespace}}
}

func sdkReportDefinition(
	resource *datasafev1beta1.ReportDefinition,
	id string,
	state datasafesdk.ReportDefinitionLifecycleStateEnum,
) datasafesdk.ReportDefinition {
	body, err := buildReportDefinitionCreateBody(context.Background(), resource, resource.Namespace)
	if err != nil {
		panic(err)
	}
	return reportDefinitionFromCreateDetails(id, body.(datasafesdk.CreateReportDefinitionDetails), state)
}

func reportDefinitionFromCreateDetails(
	id string,
	details datasafesdk.CreateReportDefinitionDetails,
	state datasafesdk.ReportDefinitionLifecycleStateEnum,
) datasafesdk.ReportDefinition {
	return datasafesdk.ReportDefinition{
		DisplayName:      details.DisplayName,
		Id:               common.String(id),
		CompartmentId:    details.CompartmentId,
		LifecycleState:   state,
		ParentId:         details.ParentId,
		Category:         datasafesdk.ReportDefinitionCategoryCustomReports,
		Description:      details.Description,
		DataSource:       datasafesdk.ReportDefinitionDataSourceAlerts,
		IsSeeded:         common.Bool(false),
		DisplayOrder:     common.Int(1),
		ColumnInfo:       details.ColumnInfo,
		ColumnFilters:    details.ColumnFilters,
		ColumnSortings:   details.ColumnSortings,
		Summary:          details.Summary,
		FreeformTags:     reportDefinitionStringMap(details.FreeformTags),
		DefinedTags:      reportDefinitionDefinedTagsFromSDK(details.DefinedTags),
		LifecycleDetails: common.String("active"),
	}
}

func reportDefinitionSummaryFromSDK(resource datasafesdk.ReportDefinition) datasafesdk.ReportDefinitionSummary {
	return datasafesdk.ReportDefinitionSummary{
		DisplayName:    resource.DisplayName,
		Id:             resource.Id,
		TimeCreated:    &common.SDKTime{Time: metav1.Now().Time},
		CompartmentId:  resource.CompartmentId,
		LifecycleState: resource.LifecycleState,
		Category:       datasafesdk.ReportDefinitionSummaryCategoryEnum(resource.Category),
		Description:    resource.Description,
		IsSeeded:       resource.IsSeeded,
		DisplayOrder:   resource.DisplayOrder,
		DataSource:     resource.DataSource,
		FreeformTags:   reportDefinitionStringMap(resource.FreeformTags),
		DefinedTags:    reportDefinitionDefinedTagsFromSDK(resource.DefinedTags),
	}
}

func reportDefinitionMatchesListRequest(resource datasafesdk.ReportDefinition, request datasafesdk.ListReportDefinitionsRequest) bool {
	if request.CompartmentId != nil && !reportDefinitionStringPtrEqual(resource.CompartmentId, *request.CompartmentId) {
		return false
	}
	if request.DisplayName != nil && !reportDefinitionStringPtrEqual(resource.DisplayName, *request.DisplayName) {
		return false
	}
	if request.IsSeeded != nil && resource.IsSeeded != nil && *request.IsSeeded != *resource.IsSeeded {
		return false
	}
	return true
}

func assertReportDefinitionRecordedID(t *testing.T, resource *datasafev1beta1.ReportDefinition, want string) {
	t.Helper()
	if got := resource.Status.Id; got != want {
		t.Fatalf("status.id = %q, want %q", got, want)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
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

func requireBoolPtr(t *testing.T, name string, got *bool, want bool) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %v", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %v, want %v", name, *got, want)
	}
}

func requireLastCondition(t *testing.T, resource *datasafev1beta1.ReportDefinition, want shared.OSOKConditionType) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.status.conditions is empty, want %s", want)
	}
	got := conditions[len(conditions)-1]
	if got.Type != want {
		t.Fatalf("last condition type = %q, want %q", got.Type, want)
	}
	if want == shared.Failed && got.Status != corev1.ConditionFalse {
		t.Fatalf("failed condition status = %q, want False", got.Status)
	}
}

func requireDeleted(t *testing.T, resource *datasafev1beta1.ReportDefinition) {
	t.Helper()
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
	requireLastCondition(t, resource, shared.Terminating)
}

func assertContainsAll(t *testing.T, name string, values []string, expected ...string) {
	t.Helper()
	seen := map[string]bool{}
	for _, value := range values {
		seen[value] = true
	}
	for _, value := range expected {
		if !seen[value] {
			t.Fatalf("%s missing %q in %#v", name, value, values)
		}
	}
}

/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package ingesttimerule

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	loganalyticssdk "github.com/oracle/oci-go-sdk/v65/loganalytics"
	loganalyticsv1beta1 "github.com/oracle/oci-service-operator/api/loganalytics/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestIngestTimeRuleRuntimeCreateResolvesNamespaceAndBuildsPolymorphicBody(t *testing.T) {
	resource := newTestIngestTimeRule()
	resource.Spec.Id = "missing-id"
	fake := &fakeIngestTimeRuleOCIClient{
		namespaces: []loganalyticssdk.NamespaceSummary{testNamespace("tenantnamespace")},
		getResults: []fakeIngestTimeRuleGetResult{
			{err: errortest.NewServiceError(404, errorutil.NotFound, "not found")},
			{response: getIngestTimeRuleResponse(testSDKIngestTimeRule("created-id", resource.Spec), "get-opc")},
			{response: getIngestTimeRuleResponse(testSDKIngestTimeRule("created-id", resource.Spec), "second-get-opc")},
		},
		listResults: []fakeIngestTimeRuleListResult{{response: loganalyticssdk.ListIngestTimeRulesResponse{}}},
		createResponse: createIngestTimeRuleResponse(
			testSDKIngestTimeRule("created-id", resource.Spec),
			"create-opc",
		),
	}
	client := newIngestTimeRuleServiceClientWithOCIClient(testLogger(), fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, testRequest())
	assertSuccessfulIngestTimeRuleOperation(t, "CreateOrUpdate()", response.IsSuccessful, err)
	assertIngestTimeRuleFirstCreateReconcile(t, fake, resource)

	response, err = client.CreateOrUpdate(context.Background(), resource, testRequest())
	assertSuccessfulIngestTimeRuleOperation(t, "second CreateOrUpdate()", response.IsSuccessful, err)
	assertIngestTimeRuleSecondCreateReconcile(t, fake, resource)
}

func TestIngestTimeRuleRuntimeBindsExistingFromPaginatedList(t *testing.T) {
	resource := newTestIngestTimeRule()
	resource.Spec.Id = ""
	fake := &fakeIngestTimeRuleOCIClient{
		namespaces: []loganalyticssdk.NamespaceSummary{testNamespace("tenantnamespace")},
		listResults: []fakeIngestTimeRuleListResult{
			{
				response: loganalyticssdk.ListIngestTimeRulesResponse{
					IngestTimeRuleSummaryCollection: loganalyticssdk.IngestTimeRuleSummaryCollection{
						Items: []loganalyticssdk.IngestTimeRuleSummary{
							testSDKIngestTimeRuleSummary("other-id", "other", "fieldA", "valueA"),
						},
					},
					OpcNextPage: common.String("page-2"),
				},
			},
			{
				response: loganalyticssdk.ListIngestTimeRulesResponse{
					IngestTimeRuleSummaryCollection: loganalyticssdk.IngestTimeRuleSummaryCollection{
						Items: []loganalyticssdk.IngestTimeRuleSummary{
							testSDKIngestTimeRuleSummary("existing-id", resource.Spec.DisplayName, resource.Spec.Conditions.FieldName, resource.Spec.Conditions.FieldValue),
						},
					},
				},
			},
		},
		getResults: []fakeIngestTimeRuleGetResult{
			{response: getIngestTimeRuleResponse(testSDKIngestTimeRule("existing-id", resource.Spec), "get-opc")},
		},
	}

	response, err := newIngestTimeRuleServiceClientWithOCIClient(testLogger(), fake).
		CreateOrUpdate(context.Background(), resource, testRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if got := len(fake.listRequests); got != 2 {
		t.Fatalf("list requests = %d, want 2", got)
	}
	if got := stringPtrValue(fake.listRequests[1].Page); got != "page-2" {
		t.Fatalf("second list page = %q, want page-2", got)
	}
	if got := len(fake.createRequests); got != 0 {
		t.Fatalf("create requests = %d, want 0 for bind", got)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != "existing-id" {
		t.Fatalf("status ocid = %q, want existing-id", got)
	}
}

func TestIngestTimeRuleRuntimeNoOpSkipsUpdate(t *testing.T) {
	resource := newTestIngestTimeRule()
	resource.Status.OsokStatus.Ocid = shared.OCID("existing-id")
	fake := &fakeIngestTimeRuleOCIClient{
		namespaces: []loganalyticssdk.NamespaceSummary{testNamespace("tenantnamespace")},
		getResults: []fakeIngestTimeRuleGetResult{{response: getIngestTimeRuleResponse(testSDKIngestTimeRule("existing-id", resource.Spec), "get-opc")}},
	}

	response, err := newIngestTimeRuleServiceClientWithOCIClient(testLogger(), fake).
		CreateOrUpdate(context.Background(), resource, testRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if got := len(fake.updateRequests); got != 0 {
		t.Fatalf("update requests = %d, want 0", got)
	}
}

func TestIngestTimeRuleRuntimeMutableUpdate(t *testing.T) {
	resource := newTestIngestTimeRule()
	resource.Status.OsokStatus.Ocid = shared.OCID("existing-id")
	resource.Spec.IsEnabled = false
	currentSpec := resource.Spec
	currentSpec.DisplayName = "old-name"
	currentSpec.Description = "old description"
	currentSpec.IsEnabled = true
	updatedSpec := resource.Spec
	fake := &fakeIngestTimeRuleOCIClient{
		namespaces: []loganalyticssdk.NamespaceSummary{testNamespace("tenantnamespace")},
		getResults: []fakeIngestTimeRuleGetResult{
			{response: getIngestTimeRuleResponse(testSDKIngestTimeRule("existing-id", currentSpec), "get-before-update")},
			{response: getIngestTimeRuleResponse(testSDKIngestTimeRule("existing-id", updatedSpec), "get-after-update")},
		},
		updateResponse: updateIngestTimeRuleResponse(testSDKIngestTimeRule("existing-id", updatedSpec), "update-opc"),
	}

	response, err := newIngestTimeRuleServiceClientWithOCIClient(testLogger(), fake).
		CreateOrUpdate(context.Background(), resource, testRequest())
	assertSuccessfulIngestTimeRuleOperation(t, "CreateOrUpdate()", response.IsSuccessful, err)
	assertIngestTimeRuleMutableUpdateRequest(t, fake, resource)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "update-opc" {
		t.Fatalf("status opcRequestId = %q, want update-opc", got)
	}
}

func TestIngestTimeRuleRuntimeRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	resource := newTestIngestTimeRule()
	resource.Status.OsokStatus.Ocid = shared.OCID("existing-id")
	currentSpec := resource.Spec
	currentSpec.CompartmentId = "other-compartment"
	fake := &fakeIngestTimeRuleOCIClient{
		namespaces: []loganalyticssdk.NamespaceSummary{testNamespace("tenantnamespace")},
		getResults: []fakeIngestTimeRuleGetResult{{response: getIngestTimeRuleResponse(testSDKIngestTimeRule("existing-id", currentSpec), "get-opc")}},
	}

	_, err := newIngestTimeRuleServiceClientWithOCIClient(testLogger(), fake).
		CreateOrUpdate(context.Background(), resource, testRequest())
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift error")
	}
	if !strings.Contains(err.Error(), "compartmentId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartmentId replacement message", err)
	}
	if got := len(fake.updateRequests); got != 0 {
		t.Fatalf("update requests = %d, want 0", got)
	}
}

func TestIngestTimeRuleRuntimeUsesStatusIDWhenSpecIDDiffers(t *testing.T) {
	resource := newTestIngestTimeRule()
	resource.Status.OsokStatus.Ocid = shared.OCID("existing-id")
	resource.Spec.Id = "other-id"
	fake := &fakeIngestTimeRuleOCIClient{
		namespaces: []loganalyticssdk.NamespaceSummary{testNamespace("tenantnamespace")},
		getResults: []fakeIngestTimeRuleGetResult{{response: getIngestTimeRuleResponse(testSDKIngestTimeRule("existing-id", resource.Spec), "get-opc")}},
	}

	response, err := newIngestTimeRuleServiceClientWithOCIClient(testLogger(), fake).
		CreateOrUpdate(context.Background(), resource, testRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if got := len(fake.getRequests); got != 1 {
		t.Fatalf("get requests = %d, want 1", got)
	}
	if got := stringPtrValue(fake.getRequests[0].IngestTimeRuleId); got != "existing-id" {
		t.Fatalf("get id = %q, want existing-id", got)
	}
	if got := len(fake.updateRequests); got != 0 {
		t.Fatalf("update requests = %d, want 0", got)
	}
}

func TestIngestTimeRuleRuntimeUpdatesIsEnabledOnlyDrift(t *testing.T) {
	resource := newTestIngestTimeRule()
	resource.Status.OsokStatus.Ocid = shared.OCID("existing-id")
	resource.Spec.IsEnabled = false
	currentSpec := resource.Spec
	currentSpec.IsEnabled = true
	updatedSpec := resource.Spec
	fake := &fakeIngestTimeRuleOCIClient{
		namespaces: []loganalyticssdk.NamespaceSummary{testNamespace("tenantnamespace")},
		getResults: []fakeIngestTimeRuleGetResult{
			{response: getIngestTimeRuleResponse(testSDKIngestTimeRule("existing-id", currentSpec), "get-before-update")},
			{response: getIngestTimeRuleResponse(testSDKIngestTimeRule("existing-id", updatedSpec), "get-after-update")},
		},
		updateResponse: updateIngestTimeRuleResponse(testSDKIngestTimeRule("existing-id", updatedSpec), "update-opc"),
	}

	response, err := newIngestTimeRuleServiceClientWithOCIClient(testLogger(), fake).
		CreateOrUpdate(context.Background(), resource, testRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if got := len(fake.updateRequests); got != 1 {
		t.Fatalf("update requests = %d, want 1", got)
	}
	if got := fake.updateRequests[0].UpdateIngestTimeRuleDetails.IsEnabled; got == nil || *got {
		t.Fatalf("update isEnabled = %v, want desired false", got)
	}
	if resource.Status.IsEnabled {
		t.Fatal("status isEnabled = true, want OCI readback false")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "update-opc" {
		t.Fatalf("status opcRequestId = %q, want update-opc", got)
	}
}

func TestIngestTimeRuleRuntimeRejectsInvalidAdditionalConditionBeforeCreate(t *testing.T) {
	resource := newTestIngestTimeRule()
	resource.Spec.Id = ""
	resource.Spec.Conditions.AdditionalConditions = []loganalyticsv1beta1.IngestTimeRuleConditionsAdditionalCondition{
		{
			ConditionField:    "severity",
			ConditionOperator: "UNSUPPORTED",
			ConditionValue:    "ERROR",
		},
	}
	fake := &fakeIngestTimeRuleOCIClient{
		namespaces:  []loganalyticssdk.NamespaceSummary{testNamespace("tenantnamespace")},
		listResults: []fakeIngestTimeRuleListResult{{response: loganalyticssdk.ListIngestTimeRulesResponse{}}},
	}

	_, err := newIngestTimeRuleServiceClientWithOCIClient(testLogger(), fake).
		CreateOrUpdate(context.Background(), resource, testRequest())
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want additional condition validation error")
	}
	if !strings.Contains(err.Error(), "additionalConditions[0].conditionOperator") {
		t.Fatalf("CreateOrUpdate() error = %v, want additional condition operator message", err)
	}
	if got := len(fake.createRequests); got != 0 {
		t.Fatalf("create requests = %d, want 0", got)
	}
}

func TestIngestTimeRuleRuntimeDeleteRetainsFinalizerUntilConfirmed(t *testing.T) {
	resource := newTestIngestTimeRule()
	resource.Status.OsokStatus.Ocid = shared.OCID("existing-id")
	active := testSDKIngestTimeRule("existing-id", resource.Spec)
	fake := &fakeIngestTimeRuleOCIClient{
		namespaces: []loganalyticssdk.NamespaceSummary{testNamespace("tenantnamespace")},
		getResults: []fakeIngestTimeRuleGetResult{
			{response: getIngestTimeRuleResponse(active, "pre-delete-guard")},
			{response: getIngestTimeRuleResponse(active, "pre-delete-get")},
			{response: getIngestTimeRuleResponse(active, "post-delete-get")},
		},
		deleteResponse: loganalyticssdk.DeleteIngestTimeRuleResponse{OpcRequestId: common.String("delete-opc")},
	}

	deleted, err := newIngestTimeRuleServiceClientWithOCIClient(testLogger(), fake).
		Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while OCI still returns ACTIVE")
	}
	if got := len(fake.deleteRequests); got != 1 {
		t.Fatalf("delete requests = %d, want 1", got)
	}
	if got := resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type; got != shared.Terminating {
		t.Fatalf("last condition = %q, want Terminating", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "delete-opc" {
		t.Fatalf("status opcRequestId = %q, want delete-opc", got)
	}
}

func TestIngestTimeRuleRuntimeDeleteConfirmsNotFound(t *testing.T) {
	resource := newTestIngestTimeRule()
	resource.Status.OsokStatus.Ocid = shared.OCID("existing-id")
	active := testSDKIngestTimeRule("existing-id", resource.Spec)
	fake := &fakeIngestTimeRuleOCIClient{
		namespaces: []loganalyticssdk.NamespaceSummary{testNamespace("tenantnamespace")},
		getResults: []fakeIngestTimeRuleGetResult{
			{response: getIngestTimeRuleResponse(active, "pre-delete-guard")},
			{response: getIngestTimeRuleResponse(active, "pre-delete-get")},
			{err: errortest.NewServiceError(404, errorutil.NotFound, "deleted")},
		},
		deleteResponse: loganalyticssdk.DeleteIngestTimeRuleResponse{OpcRequestId: common.String("delete-opc")},
	}

	deleted, err := newIngestTimeRuleServiceClientWithOCIClient(testLogger(), fake).
		Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after unambiguous not found")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status deletedAt = nil, want timestamp")
	}
}

func TestIngestTimeRuleRuntimeDeleteKeepsAuthShapedPostDeleteConfirmationAmbiguous(t *testing.T) {
	resource := newTestIngestTimeRule()
	resource.Status.OsokStatus.Ocid = shared.OCID("existing-id")
	active := testSDKIngestTimeRule("existing-id", resource.Spec)
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	fake := &fakeIngestTimeRuleOCIClient{
		namespaces: []loganalyticssdk.NamespaceSummary{testNamespace("tenantnamespace")},
		getResults: []fakeIngestTimeRuleGetResult{
			{response: getIngestTimeRuleResponse(active, "pre-delete-guard")},
			{response: getIngestTimeRuleResponse(active, "pre-delete-get")},
			{err: authErr},
		},
		deleteResponse: loganalyticssdk.DeleteIngestTimeRuleResponse{OpcRequestId: common.String("delete-opc")},
	}

	deleted, err := newIngestTimeRuleServiceClientWithOCIClient(testLogger(), fake).
		Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous post-delete confirmation error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous post-delete confirmation")
	}
	if got := len(fake.deleteRequests); got != 1 {
		t.Fatalf("delete requests = %d, want 1", got)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status deletedAt is set, want nil while post-delete confirmation is ambiguous")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status opcRequestId = %q, want opc-request-id", got)
	}
	if !strings.Contains(err.Error(), "ambiguous not-found") {
		t.Fatalf("Delete() error = %v, want ambiguous not-found message", err)
	}
}

func TestIngestTimeRuleRuntimeDeleteRejectsAuthShapedPreDeleteRead(t *testing.T) {
	resource := newTestIngestTimeRule()
	resource.Status.OsokStatus.Ocid = shared.OCID("existing-id")
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	fake := &fakeIngestTimeRuleOCIClient{
		namespaces: []loganalyticssdk.NamespaceSummary{testNamespace("tenantnamespace")},
		getResults: []fakeIngestTimeRuleGetResult{{err: authErr}},
	}

	deleted, err := newIngestTimeRuleServiceClientWithOCIClient(testLogger(), fake).
		Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous pre-delete not-found error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous pre-delete read")
	}
	if got := len(fake.deleteRequests); got != 0 {
		t.Fatalf("delete requests = %d, want 0", got)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status deletedAt is set, want nil while delete is ambiguous")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status opcRequestId = %q, want opc-request-id", got)
	}
	if !strings.Contains(err.Error(), "refusing to call delete") {
		t.Fatalf("Delete() error = %v, want pre-delete guard message", err)
	}
}

func TestIngestTimeRuleRuntimeDeleteKeepsAuthShaped404Ambiguous(t *testing.T) {
	resource := newTestIngestTimeRule()
	resource.Status.OsokStatus.Ocid = shared.OCID("existing-id")
	active := testSDKIngestTimeRule("existing-id", resource.Spec)
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	fake := &fakeIngestTimeRuleOCIClient{
		namespaces: []loganalyticssdk.NamespaceSummary{testNamespace("tenantnamespace")},
		getResults: []fakeIngestTimeRuleGetResult{
			{response: getIngestTimeRuleResponse(active, "pre-delete-guard")},
			{response: getIngestTimeRuleResponse(active, "pre-delete-get")},
		},
		deleteErr: authErr,
	}

	deleted, err := newIngestTimeRuleServiceClientWithOCIClient(testLogger(), fake).
		Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous not-found error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous not found")
	}
	if !strings.Contains(err.Error(), "ambiguous not-found") {
		t.Fatalf("Delete() error = %v, want ambiguous not-found message", err)
	}
}

func assertSuccessfulIngestTimeRuleOperation(t *testing.T, operation string, successful bool, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s error = %v", operation, err)
	}
	if !successful {
		t.Fatalf("%s IsSuccessful = false, want true", operation)
	}
}

func assertIngestTimeRuleFirstCreateReconcile(
	t *testing.T,
	fake *fakeIngestTimeRuleOCIClient,
	resource *loganalyticsv1beta1.IngestTimeRule,
) {
	t.Helper()
	if got := len(fake.createRequests); got != 1 {
		t.Fatalf("create requests = %d, want 1", got)
	}
	assertCreateIngestTimeRuleRequest(t, fake.createRequests[0], resource)
	assertIngestTimeRuleCreatedStatus(t, resource)
	if got := resource.Namespace; got != "k8s-ns" {
		t.Fatalf("resource namespace after reconcile = %q, want restored k8s-ns", got)
	}
}

func assertIngestTimeRuleSecondCreateReconcile(
	t *testing.T,
	fake *fakeIngestTimeRuleOCIClient,
	resource *loganalyticsv1beta1.IngestTimeRule,
) {
	t.Helper()
	if got := len(fake.createRequests); got != 1 {
		t.Fatalf("create requests after second reconcile = %d, want 1", got)
	}
	if got := len(fake.updateRequests); got != 0 {
		t.Fatalf("update requests after second reconcile = %d, want 0", got)
	}
	if got := stringPtrValue(fake.getRequests[len(fake.getRequests)-1].IngestTimeRuleId); got != "created-id" {
		t.Fatalf("second reconcile get id = %q, want created-id", got)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != "created-id" {
		t.Fatalf("status ocid after second reconcile = %q, want created-id", got)
	}
}

func assertIngestTimeRuleMutableUpdateRequest(
	t *testing.T,
	fake *fakeIngestTimeRuleOCIClient,
	resource *loganalyticsv1beta1.IngestTimeRule,
) {
	t.Helper()
	if got := len(fake.updateRequests); got != 1 {
		t.Fatalf("update requests = %d, want 1", got)
	}
	updateRequest := fake.updateRequests[0]
	if got := stringPtrValue(updateRequest.NamespaceName); got != "tenantnamespace" {
		t.Fatalf("update namespace = %q, want tenantnamespace", got)
	}
	if got := stringPtrValue(updateRequest.IngestTimeRuleId); got != "existing-id" {
		t.Fatalf("update id = %q, want existing-id", got)
	}
	if got := stringPtrValue(updateRequest.UpdateIngestTimeRuleDetails.DisplayName); got != resource.Spec.DisplayName {
		t.Fatalf("update displayName = %q, want %q", got, resource.Spec.DisplayName)
	}
	if got := stringPtrValue(updateRequest.UpdateIngestTimeRuleDetails.Description); got != resource.Spec.Description {
		t.Fatalf("update description = %q, want %q", got, resource.Spec.Description)
	}
	if updateRequest.UpdateIngestTimeRuleDetails.IsEnabled == nil || *updateRequest.UpdateIngestTimeRuleDetails.IsEnabled {
		t.Fatalf("update isEnabled = %v, want desired false", updateRequest.UpdateIngestTimeRuleDetails.IsEnabled)
	}
}

func assertCreateIngestTimeRuleRequest(
	t *testing.T,
	request loganalyticssdk.CreateIngestTimeRuleRequest,
	resource *loganalyticsv1beta1.IngestTimeRule,
) {
	t.Helper()
	if got := stringPtrValue(request.NamespaceName); got != "tenantnamespace" {
		t.Fatalf("create namespace = %q, want tenantnamespace", got)
	}
	if got := stringPtrValue(request.DisplayName); got != resource.Spec.DisplayName {
		t.Fatalf("create displayName = %q, want %q", got, resource.Spec.DisplayName)
	}
	condition, ok := request.Conditions.(loganalyticssdk.IngestTimeRuleFieldCondition)
	if !ok {
		t.Fatalf("create condition type = %T, want IngestTimeRuleFieldCondition", request.Conditions)
	}
	if got := stringPtrValue(condition.FieldName); got != resource.Spec.Conditions.FieldName {
		t.Fatalf("create condition fieldName = %q, want %q", got, resource.Spec.Conditions.FieldName)
	}
	action, ok := request.Actions[0].(loganalyticssdk.IngestTimeRuleMetricExtractionAction)
	if !ok {
		t.Fatalf("create action type = %T, want IngestTimeRuleMetricExtractionAction", request.Actions[0])
	}
	if got := stringPtrValue(action.MetricName); got != resource.Spec.Actions[0].MetricName {
		t.Fatalf("create action metricName = %q, want %q", got, resource.Spec.Actions[0].MetricName)
	}
}

func assertIngestTimeRuleCreatedStatus(t *testing.T, resource *loganalyticsv1beta1.IngestTimeRule) {
	t.Helper()
	if got := string(resource.Status.OsokStatus.Ocid); got != "created-id" {
		t.Fatalf("status ocid = %q, want created-id", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "create-opc" {
		t.Fatalf("status opcRequestId = %q, want create-opc", got)
	}
}

type fakeIngestTimeRuleOCIClient struct {
	namespaces []loganalyticssdk.NamespaceSummary

	createRequests []loganalyticssdk.CreateIngestTimeRuleRequest
	getRequests    []loganalyticssdk.GetIngestTimeRuleRequest
	listRequests   []loganalyticssdk.ListIngestTimeRulesRequest
	updateRequests []loganalyticssdk.UpdateIngestTimeRuleRequest
	deleteRequests []loganalyticssdk.DeleteIngestTimeRuleRequest

	createResponse loganalyticssdk.CreateIngestTimeRuleResponse
	createErr      error
	getResults     []fakeIngestTimeRuleGetResult
	listResults    []fakeIngestTimeRuleListResult
	updateResponse loganalyticssdk.UpdateIngestTimeRuleResponse
	updateErr      error
	deleteResponse loganalyticssdk.DeleteIngestTimeRuleResponse
	deleteErr      error
}

type fakeIngestTimeRuleGetResult struct {
	response loganalyticssdk.GetIngestTimeRuleResponse
	err      error
}

type fakeIngestTimeRuleListResult struct {
	response loganalyticssdk.ListIngestTimeRulesResponse
	err      error
}

func (f *fakeIngestTimeRuleOCIClient) CreateIngestTimeRule(
	_ context.Context,
	request loganalyticssdk.CreateIngestTimeRuleRequest,
) (loganalyticssdk.CreateIngestTimeRuleResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createErr != nil {
		return loganalyticssdk.CreateIngestTimeRuleResponse{}, f.createErr
	}
	return f.createResponse, nil
}

func (f *fakeIngestTimeRuleOCIClient) GetIngestTimeRule(
	_ context.Context,
	request loganalyticssdk.GetIngestTimeRuleRequest,
) (loganalyticssdk.GetIngestTimeRuleResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if len(f.getResults) == 0 {
		return loganalyticssdk.GetIngestTimeRuleResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "not found")
	}
	result := f.getResults[0]
	f.getResults = f.getResults[1:]
	if result.err != nil {
		return loganalyticssdk.GetIngestTimeRuleResponse{}, result.err
	}
	return result.response, nil
}

func (f *fakeIngestTimeRuleOCIClient) ListIngestTimeRules(
	_ context.Context,
	request loganalyticssdk.ListIngestTimeRulesRequest,
) (loganalyticssdk.ListIngestTimeRulesResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if len(f.listResults) == 0 {
		return loganalyticssdk.ListIngestTimeRulesResponse{}, nil
	}
	result := f.listResults[0]
	f.listResults = f.listResults[1:]
	if result.err != nil {
		return loganalyticssdk.ListIngestTimeRulesResponse{}, result.err
	}
	return result.response, nil
}

func (f *fakeIngestTimeRuleOCIClient) UpdateIngestTimeRule(
	_ context.Context,
	request loganalyticssdk.UpdateIngestTimeRuleRequest,
) (loganalyticssdk.UpdateIngestTimeRuleResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateErr != nil {
		return loganalyticssdk.UpdateIngestTimeRuleResponse{}, f.updateErr
	}
	return f.updateResponse, nil
}

func (f *fakeIngestTimeRuleOCIClient) DeleteIngestTimeRule(
	_ context.Context,
	request loganalyticssdk.DeleteIngestTimeRuleRequest,
) (loganalyticssdk.DeleteIngestTimeRuleResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteErr != nil {
		return loganalyticssdk.DeleteIngestTimeRuleResponse{}, f.deleteErr
	}
	return f.deleteResponse, nil
}

func (f *fakeIngestTimeRuleOCIClient) ListNamespaces(
	_ context.Context,
	request loganalyticssdk.ListNamespacesRequest,
) (loganalyticssdk.ListNamespacesResponse, error) {
	return loganalyticssdk.ListNamespacesResponse{
		NamespaceCollection: loganalyticssdk.NamespaceCollection{Items: f.namespaces},
	}, nil
}

func newTestIngestTimeRule() *loganalyticsv1beta1.IngestTimeRule {
	return &loganalyticsv1beta1.IngestTimeRule{
		ObjectMeta: metav1.ObjectMeta{Name: "rule", Namespace: "k8s-ns"},
		Spec: loganalyticsv1beta1.IngestTimeRuleSpec{
			CompartmentId: "ocid1.compartment.oc1..example",
			DisplayName:   "ingest-rule",
			Id:            "existing-id",
			Description:   "desired description",
			IsEnabled:     true,
			Conditions: loganalyticsv1beta1.IngestTimeRuleConditions{
				Kind:          "FIELD",
				FieldName:     "source",
				FieldValue:    "app",
				FieldOperator: "EQUAL",
			},
			Actions: []loganalyticsv1beta1.IngestTimeRuleAction{
				{
					Type:          "METRIC_EXTRACTION",
					CompartmentId: "ocid1.compartment.oc1..example",
					Namespace:     "custom_metrics",
					MetricName:    "requests",
					ResourceGroup: "api",
					Dimensions:    []string{"SOURCE_NAME"},
				},
			},
			FreeformTags: map[string]string{"env": "test"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func testRequest() ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "k8s-ns", Name: "rule"}}
}

func testNamespace(name string) loganalyticssdk.NamespaceSummary {
	return loganalyticssdk.NamespaceSummary{
		NamespaceName:  common.String(name),
		CompartmentId:  common.String("ocid1.tenancy.oc1..example"),
		IsOnboarded:    common.Bool(true),
		LifecycleState: loganalyticssdk.NamespaceSummaryLifecycleStateActive,
	}
}

func testSDKIngestTimeRule(
	id string,
	spec loganalyticsv1beta1.IngestTimeRuleSpec,
) loganalyticssdk.IngestTimeRule {
	condition, err := ingestTimeRuleConditionFromSpec(spec.Conditions)
	if err != nil {
		panic(err)
	}
	actions, err := ingestTimeRuleActionsFromSpec(spec.Actions)
	if err != nil {
		panic(err)
	}
	return loganalyticssdk.IngestTimeRule{
		Id:             common.String(id),
		CompartmentId:  common.String(spec.CompartmentId),
		DisplayName:    common.String(spec.DisplayName),
		Description:    common.String(spec.Description),
		FreeformTags:   spec.FreeformTags,
		DefinedTags:    *utilDefinedTags(spec.DefinedTags),
		LifecycleState: loganalyticssdk.ConfigLifecycleStateActive,
		IsEnabled:      common.Bool(spec.IsEnabled),
		Conditions:     condition,
		Actions:        actions,
	}
}

func testSDKIngestTimeRuleSummary(id string, displayName string, fieldName string, fieldValue string) loganalyticssdk.IngestTimeRuleSummary {
	return loganalyticssdk.IngestTimeRuleSummary{
		Id:             common.String(id),
		CompartmentId:  common.String("ocid1.compartment.oc1..example"),
		DisplayName:    common.String(displayName),
		LifecycleState: loganalyticssdk.ConfigLifecycleStateActive,
		IsEnabled:      common.Bool(true),
		ConditionKind:  loganalyticssdk.IngestTimeRuleSummaryConditionKindField,
		FieldName:      common.String(fieldName),
		FieldValue:     common.String(fieldValue),
	}
}

func createIngestTimeRuleResponse(
	rule loganalyticssdk.IngestTimeRule,
	opcRequestID string,
) loganalyticssdk.CreateIngestTimeRuleResponse {
	return loganalyticssdk.CreateIngestTimeRuleResponse{
		IngestTimeRule: rule,
		OpcRequestId:   common.String(opcRequestID),
	}
}

func getIngestTimeRuleResponse(
	rule loganalyticssdk.IngestTimeRule,
	opcRequestID string,
) loganalyticssdk.GetIngestTimeRuleResponse {
	return loganalyticssdk.GetIngestTimeRuleResponse{
		IngestTimeRule: rule,
		OpcRequestId:   common.String(opcRequestID),
	}
}

func updateIngestTimeRuleResponse(
	rule loganalyticssdk.IngestTimeRule,
	opcRequestID string,
) loganalyticssdk.UpdateIngestTimeRuleResponse {
	return loganalyticssdk.UpdateIngestTimeRuleResponse{
		IngestTimeRule: rule,
		OpcRequestId:   common.String(opcRequestID),
	}
}

func utilDefinedTags(tags map[string]shared.MapValue) *map[string]map[string]interface{} {
	converted := make(map[string]map[string]interface{}, len(tags))
	for namespace, values := range tags {
		children := make(map[string]interface{}, len(values))
		for key, value := range values {
			children[key] = value
		}
		converted[namespace] = children
	}
	return &converted
}

func testLogger() loggerutil.OSOKLogger {
	return loggerutil.OSOKLogger{}
}

/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package loganalyticsentity

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	loganalyticssdk "github.com/oracle/oci-go-sdk/v65/loganalytics"
	loganalyticsv1beta1 "github.com/oracle/oci-service-operator/api/loganalytics/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testLogAnalyticsNamespaceName = "logan"
	testLogAnalyticsKubeNamespace = "kube-ns"
	testLogAnalyticsEntityID      = "ocid1.loganalyticsentity.oc1..entity"
	testLogAnalyticsExistingID    = "ocid1.loganalyticsentity.oc1..existing"
	testLogAnalyticsCompartmentID = "ocid1.compartment.oc1..logan"
	testLogAnalyticsEntityName    = "entity-alpha"
	testLogAnalyticsEntityType    = "host"
	testLogAnalyticsCloudID       = "ocid1.instance.oc1..cloud"
	testLogAnalyticsAgentID       = "ocid1.managementagent.oc1..agent"
)

type logAnalyticsEntityNamespaceResult struct {
	response loganalyticssdk.ListNamespacesResponse
	err      error
}

type logAnalyticsEntityCreateResult struct {
	response loganalyticssdk.CreateLogAnalyticsEntityResponse
	err      error
}

type logAnalyticsEntityGetResult struct {
	response loganalyticssdk.GetLogAnalyticsEntityResponse
	err      error
}

type logAnalyticsEntityListResult struct {
	response loganalyticssdk.ListLogAnalyticsEntitiesResponse
	err      error
}

type logAnalyticsEntityUpdateResult struct {
	response loganalyticssdk.UpdateLogAnalyticsEntityResponse
	err      error
}

type logAnalyticsEntityDeleteResult struct {
	response loganalyticssdk.DeleteLogAnalyticsEntityResponse
	err      error
}

type fakeLogAnalyticsEntityOCIClient struct {
	namespaceRequests []loganalyticssdk.ListNamespacesRequest
	createRequests    []loganalyticssdk.CreateLogAnalyticsEntityRequest
	getRequests       []loganalyticssdk.GetLogAnalyticsEntityRequest
	listRequests      []loganalyticssdk.ListLogAnalyticsEntitiesRequest
	updateRequests    []loganalyticssdk.UpdateLogAnalyticsEntityRequest
	deleteRequests    []loganalyticssdk.DeleteLogAnalyticsEntityRequest

	namespaceResults []logAnalyticsEntityNamespaceResult
	createResults    []logAnalyticsEntityCreateResult
	getResults       []logAnalyticsEntityGetResult
	listResults      []logAnalyticsEntityListResult
	updateResults    []logAnalyticsEntityUpdateResult
	deleteResults    []logAnalyticsEntityDeleteResult
}

func (f *fakeLogAnalyticsEntityOCIClient) ListNamespaces(
	_ context.Context,
	request loganalyticssdk.ListNamespacesRequest,
) (loganalyticssdk.ListNamespacesResponse, error) {
	f.namespaceRequests = append(f.namespaceRequests, request)
	if len(f.namespaceResults) == 0 {
		return loganalyticssdk.ListNamespacesResponse{}, fmt.Errorf("unexpected ListNamespaces call")
	}
	next := f.namespaceResults[0]
	f.namespaceResults = f.namespaceResults[1:]
	return next.response, next.err
}

func (f *fakeLogAnalyticsEntityOCIClient) CreateLogAnalyticsEntity(
	_ context.Context,
	request loganalyticssdk.CreateLogAnalyticsEntityRequest,
) (loganalyticssdk.CreateLogAnalyticsEntityResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if len(f.createResults) == 0 {
		return loganalyticssdk.CreateLogAnalyticsEntityResponse{}, fmt.Errorf("unexpected CreateLogAnalyticsEntity call")
	}
	next := f.createResults[0]
	f.createResults = f.createResults[1:]
	return next.response, next.err
}

func (f *fakeLogAnalyticsEntityOCIClient) GetLogAnalyticsEntity(
	_ context.Context,
	request loganalyticssdk.GetLogAnalyticsEntityRequest,
) (loganalyticssdk.GetLogAnalyticsEntityResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if len(f.getResults) == 0 {
		return loganalyticssdk.GetLogAnalyticsEntityResponse{}, fmt.Errorf("unexpected GetLogAnalyticsEntity call")
	}
	next := f.getResults[0]
	f.getResults = f.getResults[1:]
	return next.response, next.err
}

func (f *fakeLogAnalyticsEntityOCIClient) ListLogAnalyticsEntities(
	_ context.Context,
	request loganalyticssdk.ListLogAnalyticsEntitiesRequest,
) (loganalyticssdk.ListLogAnalyticsEntitiesResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if len(f.listResults) == 0 {
		return loganalyticssdk.ListLogAnalyticsEntitiesResponse{}, fmt.Errorf("unexpected ListLogAnalyticsEntities call")
	}
	next := f.listResults[0]
	f.listResults = f.listResults[1:]
	return next.response, next.err
}

func (f *fakeLogAnalyticsEntityOCIClient) UpdateLogAnalyticsEntity(
	_ context.Context,
	request loganalyticssdk.UpdateLogAnalyticsEntityRequest,
) (loganalyticssdk.UpdateLogAnalyticsEntityResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if len(f.updateResults) == 0 {
		return loganalyticssdk.UpdateLogAnalyticsEntityResponse{}, fmt.Errorf("unexpected UpdateLogAnalyticsEntity call")
	}
	next := f.updateResults[0]
	f.updateResults = f.updateResults[1:]
	return next.response, next.err
}

func (f *fakeLogAnalyticsEntityOCIClient) DeleteLogAnalyticsEntity(
	_ context.Context,
	request loganalyticssdk.DeleteLogAnalyticsEntityRequest,
) (loganalyticssdk.DeleteLogAnalyticsEntityResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if len(f.deleteResults) == 0 {
		return loganalyticssdk.DeleteLogAnalyticsEntityResponse{}, fmt.Errorf("unexpected DeleteLogAnalyticsEntity call")
	}
	next := f.deleteResults[0]
	f.deleteResults = f.deleteResults[1:]
	return next.response, next.err
}

func TestLogAnalyticsEntityServiceClientCreatesWithResolvedNamespaceAndTracksRequestID(t *testing.T) {
	resource := testLogAnalyticsEntityResource()
	fake := &fakeLogAnalyticsEntityOCIClient{
		namespaceResults: []logAnalyticsEntityNamespaceResult{{response: testLogAnalyticsEntityNamespaceResponse()}},
		listResults:      []logAnalyticsEntityListResult{{response: loganalyticssdk.ListLogAnalyticsEntitiesResponse{}}},
		createResults: []logAnalyticsEntityCreateResult{{
			response: loganalyticssdk.CreateLogAnalyticsEntityResponse{
				LogAnalyticsEntity: testLogAnalyticsEntitySDK(testLogAnalyticsEntityID, resource, loganalyticssdk.EntityLifecycleStatesActive),
				OpcRequestId:       common.String("opc-create"),
			},
		}},
		getResults: []logAnalyticsEntityGetResult{{
			response: loganalyticssdk.GetLogAnalyticsEntityResponse{
				LogAnalyticsEntity: testLogAnalyticsEntitySDK(testLogAnalyticsEntityID, resource, loganalyticssdk.EntityLifecycleStatesActive),
			},
		}},
	}

	response, err := newTestLogAnalyticsEntityServiceClient(fake).CreateOrUpdate(context.Background(), resource, testLogAnalyticsEntityRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() successful = false, want true")
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("CreateLogAnalyticsEntity calls = %d, want 1", len(fake.createRequests))
	}
	requireLogAnalyticsEntityStringPtr(t, "namespace compartmentId", fake.namespaceRequests[0].CompartmentId, testLogAnalyticsCompartmentID)
	requireLogAnalyticsEntityStringPtr(t, "list namespaceName", fake.listRequests[0].NamespaceName, testLogAnalyticsNamespaceName)
	requireLogAnalyticsEntityStringPtr(t, "create namespaceName", fake.createRequests[0].NamespaceName, testLogAnalyticsNamespaceName)
	requireLogAnalyticsEntityStringPtr(t, "get namespaceName", fake.getRequests[0].NamespaceName, testLogAnalyticsNamespaceName)
	if got := stringValue(fake.createRequests[0].NamespaceName); got == testLogAnalyticsKubeNamespace {
		t.Fatalf("create namespaceName = Kubernetes namespace %q, want Log Analytics namespace", got)
	}
	if fake.createRequests[0].OpcRetryToken == nil {
		t.Fatal("create request OpcRetryToken = nil, want deterministic retry token")
	}
	assertLogAnalyticsEntityCreateBody(t, fake.createRequests[0].CreateLogAnalyticsEntityDetails, resource)
	if resource.Status.OsokStatus.Ocid != shared.OCID(testLogAnalyticsEntityID) {
		t.Fatalf("status ocid = %q, want %q", resource.Status.OsokStatus.Ocid, testLogAnalyticsEntityID)
	}
	if resource.Status.Id != testLogAnalyticsEntityID {
		t.Fatalf("status id = %q, want %q", resource.Status.Id, testLogAnalyticsEntityID)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-create" {
		t.Fatalf("status opcRequestId = %q, want opc-create", resource.Status.OsokStatus.OpcRequestID)
	}
	assertLogAnalyticsEntityLatestCondition(t, resource, shared.Active)
}

func TestLogAnalyticsEntityServiceClientBindsExistingEntityAcrossListPages(t *testing.T) {
	resource := testLogAnalyticsEntityResource()
	fake := &fakeLogAnalyticsEntityOCIClient{
		namespaceResults: []logAnalyticsEntityNamespaceResult{{response: testLogAnalyticsEntityNamespaceResponse()}},
		listResults: []logAnalyticsEntityListResult{
			{
				response: loganalyticssdk.ListLogAnalyticsEntitiesResponse{
					OpcNextPage: common.String("page-2"),
				},
			},
			{
				response: loganalyticssdk.ListLogAnalyticsEntitiesResponse{
					LogAnalyticsEntityCollection: loganalyticssdk.LogAnalyticsEntityCollection{
						Items: []loganalyticssdk.LogAnalyticsEntitySummary{
							testLogAnalyticsEntitySummary(testLogAnalyticsExistingID, resource, loganalyticssdk.EntityLifecycleStatesActive),
						},
					},
				},
			},
		},
		getResults: []logAnalyticsEntityGetResult{{
			response: loganalyticssdk.GetLogAnalyticsEntityResponse{
				LogAnalyticsEntity: testLogAnalyticsEntitySDK(testLogAnalyticsExistingID, resource, loganalyticssdk.EntityLifecycleStatesActive),
			},
		}},
	}

	response, err := newTestLogAnalyticsEntityServiceClient(fake).CreateOrUpdate(context.Background(), resource, testLogAnalyticsEntityRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() successful = false, want true")
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateLogAnalyticsEntity calls = %d, want 0", len(fake.createRequests))
	}
	if len(fake.listRequests) != 2 {
		t.Fatalf("ListLogAnalyticsEntities calls = %d, want 2", len(fake.listRequests))
	}
	if fake.listRequests[0].Page != nil {
		t.Fatalf("first list page = %q, want nil", stringValue(fake.listRequests[0].Page))
	}
	requireLogAnalyticsEntityStringPtr(t, "second list page", fake.listRequests[1].Page, "page-2")
	requireLogAnalyticsEntityStringPtr(t, "get logAnalyticsEntityId", fake.getRequests[0].LogAnalyticsEntityId, testLogAnalyticsExistingID)
	if resource.Status.OsokStatus.Ocid != shared.OCID(testLogAnalyticsExistingID) {
		t.Fatalf("status ocid = %q, want %q", resource.Status.OsokStatus.Ocid, testLogAnalyticsExistingID)
	}
	assertLogAnalyticsEntityLatestCondition(t, resource, shared.Active)
}

func TestLogAnalyticsEntityServiceClientSkipsNoopUpdateWhenObservedStateMatches(t *testing.T) {
	resource := testLogAnalyticsEntityResource()
	recordLogAnalyticsEntityID(resource, testLogAnalyticsEntityID)
	fake := &fakeLogAnalyticsEntityOCIClient{
		namespaceResults: []logAnalyticsEntityNamespaceResult{{response: testLogAnalyticsEntityNamespaceResponse()}},
		getResults: []logAnalyticsEntityGetResult{{
			response: loganalyticssdk.GetLogAnalyticsEntityResponse{
				LogAnalyticsEntity: testLogAnalyticsEntitySDK(testLogAnalyticsEntityID, resource, loganalyticssdk.EntityLifecycleStatesActive),
			},
		}},
	}

	response, err := newTestLogAnalyticsEntityServiceClient(fake).CreateOrUpdate(context.Background(), resource, testLogAnalyticsEntityRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() successful = false, want true")
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateLogAnalyticsEntity calls = %d, want 0", len(fake.updateRequests))
	}
	requireLogAnalyticsEntityStringPtr(t, "get namespaceName", fake.getRequests[0].NamespaceName, testLogAnalyticsNamespaceName)
}

func TestLogAnalyticsEntityServiceClientUpdatesMutableFields(t *testing.T) {
	resource := testLogAnalyticsEntityResource()
	resource.Spec.Properties = map[string]string{"role": "api", "tier": "prod"}
	recordLogAnalyticsEntityID(resource, testLogAnalyticsEntityID)
	current := testLogAnalyticsEntitySDK(testLogAnalyticsEntityID, resource, loganalyticssdk.EntityLifecycleStatesActive)
	current.Properties = map[string]string{"role": "api", "tier": "dev"}
	updated := testLogAnalyticsEntitySDK(testLogAnalyticsEntityID, resource, loganalyticssdk.EntityLifecycleStatesActive)
	fake := &fakeLogAnalyticsEntityOCIClient{
		namespaceResults: []logAnalyticsEntityNamespaceResult{{response: testLogAnalyticsEntityNamespaceResponse()}},
		getResults: []logAnalyticsEntityGetResult{
			{response: loganalyticssdk.GetLogAnalyticsEntityResponse{LogAnalyticsEntity: current}},
			{response: loganalyticssdk.GetLogAnalyticsEntityResponse{LogAnalyticsEntity: updated}},
		},
		updateResults: []logAnalyticsEntityUpdateResult{{
			response: loganalyticssdk.UpdateLogAnalyticsEntityResponse{
				LogAnalyticsEntity: updated,
				OpcRequestId:       common.String("opc-update"),
			},
		}},
	}

	response, err := newTestLogAnalyticsEntityServiceClient(fake).CreateOrUpdate(context.Background(), resource, testLogAnalyticsEntityRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() successful = false, want true")
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("UpdateLogAnalyticsEntity calls = %d, want 1", len(fake.updateRequests))
	}
	requireLogAnalyticsEntityStringPtr(t, "update namespaceName", fake.updateRequests[0].NamespaceName, testLogAnalyticsNamespaceName)
	requireLogAnalyticsEntityStringPtr(t, "update logAnalyticsEntityId", fake.updateRequests[0].LogAnalyticsEntityId, testLogAnalyticsEntityID)
	if got := fake.updateRequests[0].Properties; !reflect.DeepEqual(got, resource.Spec.Properties) {
		t.Fatalf("update properties = %#v, want %#v", got, resource.Spec.Properties)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-update" {
		t.Fatalf("status opcRequestId = %q, want opc-update", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestLogAnalyticsEntityServiceClientRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	resource := testLogAnalyticsEntityResource()
	resource.Spec.EntityTypeName = "linux-host"
	recordLogAnalyticsEntityID(resource, testLogAnalyticsEntityID)
	current := testLogAnalyticsEntitySDK(testLogAnalyticsEntityID, resource, loganalyticssdk.EntityLifecycleStatesActive)
	current.EntityTypeName = common.String("database")
	fake := &fakeLogAnalyticsEntityOCIClient{
		namespaceResults: []logAnalyticsEntityNamespaceResult{{response: testLogAnalyticsEntityNamespaceResponse()}},
		getResults: []logAnalyticsEntityGetResult{{
			response: loganalyticssdk.GetLogAnalyticsEntityResponse{LogAnalyticsEntity: current},
		}},
	}

	response, err := newTestLogAnalyticsEntityServiceClient(fake).CreateOrUpdate(context.Background(), resource, testLogAnalyticsEntityRequest(resource))
	if err == nil || !strings.Contains(err.Error(), "entityTypeName") {
		t.Fatalf("CreateOrUpdate() error = %v, want entityTypeName force-new drift failure", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() successful = true, want false")
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateLogAnalyticsEntity calls = %d, want 0", len(fake.updateRequests))
	}
	assertLogAnalyticsEntityLatestCondition(t, resource, shared.Failed)
}

func TestLogAnalyticsEntityDeleteConfirmsNotFoundBeforeFinalizerRelease(t *testing.T) {
	resource := testLogAnalyticsEntityResource()
	recordLogAnalyticsEntityID(resource, testLogAnalyticsEntityID)
	fake := &fakeLogAnalyticsEntityOCIClient{
		namespaceResults: []logAnalyticsEntityNamespaceResult{{response: testLogAnalyticsEntityNamespaceResponse()}},
		getResults: []logAnalyticsEntityGetResult{
			{
				response: loganalyticssdk.GetLogAnalyticsEntityResponse{
					LogAnalyticsEntity: testLogAnalyticsEntitySDK(testLogAnalyticsEntityID, resource, loganalyticssdk.EntityLifecycleStatesActive),
				},
			},
			{
				response: loganalyticssdk.GetLogAnalyticsEntityResponse{
					LogAnalyticsEntity: testLogAnalyticsEntitySDK(testLogAnalyticsEntityID, resource, loganalyticssdk.EntityLifecycleStatesActive),
				},
			},
			{
				err: errortest.NewServiceError(404, errorutil.NotFound, "entity deleted"),
			},
		},
		listResults: []logAnalyticsEntityListResult{{response: loganalyticssdk.ListLogAnalyticsEntitiesResponse{}}},
		deleteResults: []logAnalyticsEntityDeleteResult{{
			response: loganalyticssdk.DeleteLogAnalyticsEntityResponse{OpcRequestId: common.String("opc-delete")},
		}},
	}

	deleted, err := newTestLogAnalyticsEntityServiceClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("DeleteLogAnalyticsEntity calls = %d, want 1", len(fake.deleteRequests))
	}
	requireLogAnalyticsEntityStringPtr(t, "delete namespaceName", fake.deleteRequests[0].NamespaceName, testLogAnalyticsNamespaceName)
	requireLogAnalyticsEntityStringPtr(t, "delete logAnalyticsEntityId", fake.deleteRequests[0].LogAnalyticsEntityId, testLogAnalyticsEntityID)
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status deletedAt = nil, want timestamp")
	}
	assertLogAnalyticsEntityLatestCondition(t, resource, shared.Terminating)
}

func TestLogAnalyticsEntityDeleteRetainsFinalizerWhileReadbackIsActive(t *testing.T) {
	resource := testLogAnalyticsEntityResource()
	recordLogAnalyticsEntityID(resource, testLogAnalyticsEntityID)
	fake := &fakeLogAnalyticsEntityOCIClient{
		namespaceResults: []logAnalyticsEntityNamespaceResult{{response: testLogAnalyticsEntityNamespaceResponse()}},
		getResults: []logAnalyticsEntityGetResult{
			{
				response: loganalyticssdk.GetLogAnalyticsEntityResponse{
					LogAnalyticsEntity: testLogAnalyticsEntitySDK(testLogAnalyticsEntityID, resource, loganalyticssdk.EntityLifecycleStatesActive),
				},
			},
			{
				response: loganalyticssdk.GetLogAnalyticsEntityResponse{
					LogAnalyticsEntity: testLogAnalyticsEntitySDK(testLogAnalyticsEntityID, resource, loganalyticssdk.EntityLifecycleStatesActive),
				},
			},
			{
				response: loganalyticssdk.GetLogAnalyticsEntityResponse{
					LogAnalyticsEntity: testLogAnalyticsEntitySDK(testLogAnalyticsEntityID, resource, loganalyticssdk.EntityLifecycleStatesActive),
				},
			},
		},
		deleteResults: []logAnalyticsEntityDeleteResult{{
			response: loganalyticssdk.DeleteLogAnalyticsEntityResponse{OpcRequestId: common.String("opc-delete")},
		}},
	}

	deleted, err := newTestLogAnalyticsEntityServiceClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while OCI readback is ACTIVE")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status deletedAt = %v, want nil", resource.Status.OsokStatus.DeletedAt)
	}
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status async current = nil, want delete pending operation")
	}
	if resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete ||
		resource.Status.OsokStatus.Async.Current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("async current = %#v, want pending delete", resource.Status.OsokStatus.Async.Current)
	}
	assertLogAnalyticsEntityLatestCondition(t, resource, shared.Terminating)
}

func TestLogAnalyticsEntityDeleteStopsBeforeDeleteWhenPreDeleteReadIsAuthShapedNotFound(t *testing.T) {
	resource := testLogAnalyticsEntityResource()
	recordLogAnalyticsEntityID(resource, testLogAnalyticsEntityID)
	authNotFound := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous missing entity")
	authNotFound.OpcRequestID = "opc-pre-delete-auth-404"
	fake := &fakeLogAnalyticsEntityOCIClient{
		namespaceResults: []logAnalyticsEntityNamespaceResult{{response: testLogAnalyticsEntityNamespaceResponse()}},
		getResults: []logAnalyticsEntityGetResult{{
			err: authNotFound,
		}},
	}

	deleted, err := newTestLogAnalyticsEntityServiceClient(fake).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want conservative auth-shaped 404 failure", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous pre-delete auth-shaped 404")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeleteLogAnalyticsEntity calls = %d, want 0", len(fake.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status deletedAt = %v, want nil", resource.Status.OsokStatus.DeletedAt)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-pre-delete-auth-404" {
		t.Fatalf("status opcRequestId = %q, want opc-pre-delete-auth-404", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestLogAnalyticsEntityDeleteTreatsAuthShapedNotFoundConservatively(t *testing.T) {
	resource := testLogAnalyticsEntityResource()
	recordLogAnalyticsEntityID(resource, testLogAnalyticsEntityID)
	authNotFound := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous missing entity")
	authNotFound.OpcRequestID = "opc-auth-404"
	fake := &fakeLogAnalyticsEntityOCIClient{
		namespaceResults: []logAnalyticsEntityNamespaceResult{{response: testLogAnalyticsEntityNamespaceResponse()}},
		getResults: []logAnalyticsEntityGetResult{
			{
				response: loganalyticssdk.GetLogAnalyticsEntityResponse{
					LogAnalyticsEntity: testLogAnalyticsEntitySDK(testLogAnalyticsEntityID, resource, loganalyticssdk.EntityLifecycleStatesActive),
				},
			},
			{
				response: loganalyticssdk.GetLogAnalyticsEntityResponse{
					LogAnalyticsEntity: testLogAnalyticsEntitySDK(testLogAnalyticsEntityID, resource, loganalyticssdk.EntityLifecycleStatesActive),
				},
			},
			{
				err: authNotFound,
			},
		},
		deleteResults: []logAnalyticsEntityDeleteResult{{
			response: loganalyticssdk.DeleteLogAnalyticsEntityResponse{OpcRequestId: common.String("opc-delete")},
		}},
	}

	deleted, err := newTestLogAnalyticsEntityServiceClient(fake).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want conservative auth-shaped 404 failure", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous auth-shaped 404")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status deletedAt = %v, want nil", resource.Status.OsokStatus.DeletedAt)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-auth-404" {
		t.Fatalf("status opcRequestId = %q, want opc-auth-404", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestLogAnalyticsEntityCreateRejectsInvalidDiscoveryTimeBeforeOCI(t *testing.T) {
	resource := testLogAnalyticsEntityResource()
	resource.Spec.TimeLastDiscovered = "not-a-timestamp"
	fake := &fakeLogAnalyticsEntityOCIClient{
		namespaceResults: []logAnalyticsEntityNamespaceResult{{response: testLogAnalyticsEntityNamespaceResponse()}},
		listResults:      []logAnalyticsEntityListResult{{response: loganalyticssdk.ListLogAnalyticsEntitiesResponse{}}},
	}

	response, err := newTestLogAnalyticsEntityServiceClient(fake).CreateOrUpdate(context.Background(), resource, testLogAnalyticsEntityRequest(resource))
	if err == nil || !strings.Contains(err.Error(), "timeLastDiscovered") {
		t.Fatalf("CreateOrUpdate() error = %v, want timeLastDiscovered parse failure", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() successful = true, want false")
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateLogAnalyticsEntity calls = %d, want 0", len(fake.createRequests))
	}
	assertLogAnalyticsEntityLatestCondition(t, resource, shared.Failed)
}

func newTestLogAnalyticsEntityServiceClient(fake *fakeLogAnalyticsEntityOCIClient) LogAnalyticsEntityServiceClient {
	hooks := newLogAnalyticsEntityRuntimeHooksWithOCIClient(fake)
	applyLogAnalyticsEntityRuntimeHooks(&hooks, fake, nil)
	manager := &LogAnalyticsEntityServiceManager{}
	config := buildLogAnalyticsEntityGeneratedRuntimeConfig(manager, hooks)
	delegate := defaultLogAnalyticsEntityServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*loganalyticsv1beta1.LogAnalyticsEntity](config),
	}
	return wrapLogAnalyticsEntityGeneratedClient(hooks, delegate)
}

func testLogAnalyticsEntityResource() *loganalyticsv1beta1.LogAnalyticsEntity {
	return &loganalyticsv1beta1.LogAnalyticsEntity{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testLogAnalyticsEntityName,
			Namespace: testLogAnalyticsKubeNamespace,
		},
		Spec: loganalyticsv1beta1.LogAnalyticsEntitySpec{
			Name:               testLogAnalyticsEntityName,
			CompartmentId:      testLogAnalyticsCompartmentID,
			EntityTypeName:     testLogAnalyticsEntityType,
			ManagementAgentId:  testLogAnalyticsAgentID,
			CloudResourceId:    testLogAnalyticsCloudID,
			TimezoneRegion:     "UTC",
			Hostname:           "entity-alpha.example.com",
			SourceId:           "enterprise-manager",
			Properties:         map[string]string{"role": "api"},
			FreeformTags:       map[string]string{"owner": "logging"},
			DefinedTags:        map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
			TimeLastDiscovered: "2026-04-29T12:00:00Z",
			Metadata: loganalyticsv1beta1.LogAnalyticsEntityMetadata{
				Items: []loganalyticsv1beta1.LogAnalyticsEntityMetadataItem{
					{Name: "environment", Value: "dev", Type: "string"},
				},
			},
		},
	}
}

func testLogAnalyticsEntityRequest(resource *loganalyticsv1beta1.LogAnalyticsEntity) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      resource.Name,
			Namespace: resource.Namespace,
		},
	}
}

func recordLogAnalyticsEntityID(resource *loganalyticsv1beta1.LogAnalyticsEntity, id string) {
	resource.Status.Id = id
	resource.Status.CompartmentId = resource.Spec.CompartmentId
	resource.Status.OsokStatus.Ocid = shared.OCID(id)
}

func testLogAnalyticsEntityNamespaceResponse() loganalyticssdk.ListNamespacesResponse {
	return loganalyticssdk.ListNamespacesResponse{
		NamespaceCollection: loganalyticssdk.NamespaceCollection{
			Items: []loganalyticssdk.NamespaceSummary{
				{
					NamespaceName:  common.String(testLogAnalyticsNamespaceName),
					CompartmentId:  common.String(testLogAnalyticsCompartmentID),
					IsOnboarded:    logAnalyticsEntityBoolPtr(true),
					LifecycleState: loganalyticssdk.NamespaceSummaryLifecycleStateActive,
				},
			},
		},
	}
}

func testLogAnalyticsEntitySDK(
	id string,
	resource *loganalyticsv1beta1.LogAnalyticsEntity,
	state loganalyticssdk.EntityLifecycleStatesEnum,
) loganalyticssdk.LogAnalyticsEntity {
	created := logAnalyticsEntitySDKTime("2026-04-29T11:00:00Z")
	discovered := logAnalyticsEntitySDKTime(resource.Spec.TimeLastDiscovered)
	return loganalyticssdk.LogAnalyticsEntity{
		Id:                           common.String(id),
		Name:                         common.String(resource.Spec.Name),
		CompartmentId:                common.String(resource.Spec.CompartmentId),
		EntityTypeName:               common.String(resource.Spec.EntityTypeName),
		EntityTypeInternalName:       common.String("internal-" + resource.Spec.EntityTypeName),
		LifecycleState:               state,
		LifecycleDetails:             common.String("ready"),
		TimeCreated:                  created,
		TimeUpdated:                  created,
		ManagementAgentId:            optionalLogAnalyticsEntityString(resource.Spec.ManagementAgentId),
		ManagementAgentDisplayName:   common.String("agent"),
		ManagementAgentCompartmentId: common.String(resource.Spec.CompartmentId),
		TimezoneRegion:               optionalLogAnalyticsEntityString(resource.Spec.TimezoneRegion),
		Properties:                   logAnalyticsEntityCopyStringMap(resource.Spec.Properties),
		TimeLastDiscovered:           discovered,
		Metadata:                     logAnalyticsEntityMetadataSummary(resource.Spec.Metadata),
		AreLogsCollected:             logAnalyticsEntityBoolPtr(true),
		CloudResourceId:              optionalLogAnalyticsEntityString(resource.Spec.CloudResourceId),
		Hostname:                     optionalLogAnalyticsEntityString(resource.Spec.Hostname),
		SourceId:                     optionalLogAnalyticsEntityString(resource.Spec.SourceId),
		FreeformTags:                 logAnalyticsEntityCopyStringMap(resource.Spec.FreeformTags),
		DefinedTags:                  logAnalyticsEntityDefinedTags(resource.Spec.DefinedTags),
		AssociatedSourcesCount:       logAnalyticsEntityIntPtr(3),
	}
}

func testLogAnalyticsEntitySummary(
	id string,
	resource *loganalyticsv1beta1.LogAnalyticsEntity,
	state loganalyticssdk.EntityLifecycleStatesEnum,
) loganalyticssdk.LogAnalyticsEntitySummary {
	created := logAnalyticsEntitySDKTime("2026-04-29T11:00:00Z")
	return loganalyticssdk.LogAnalyticsEntitySummary{
		Id:                     common.String(id),
		Name:                   common.String(resource.Spec.Name),
		CompartmentId:          common.String(resource.Spec.CompartmentId),
		EntityTypeName:         common.String(resource.Spec.EntityTypeName),
		EntityTypeInternalName: common.String("internal-" + resource.Spec.EntityTypeName),
		LifecycleState:         state,
		LifecycleDetails:       common.String("ready"),
		TimeCreated:            created,
		TimeUpdated:            created,
		ManagementAgentId:      optionalLogAnalyticsEntityString(resource.Spec.ManagementAgentId),
		CloudResourceId:        optionalLogAnalyticsEntityString(resource.Spec.CloudResourceId),
		TimezoneRegion:         optionalLogAnalyticsEntityString(resource.Spec.TimezoneRegion),
		TimeLastDiscovered:     logAnalyticsEntitySDKTime(resource.Spec.TimeLastDiscovered),
		Metadata: &loganalyticssdk.LogAnalyticsMetadataCollection{
			Items: logAnalyticsEntityMetadataItems(resource.Spec.Metadata),
		},
		AreLogsCollected:       logAnalyticsEntityBoolPtr(true),
		SourceId:               optionalLogAnalyticsEntityString(resource.Spec.SourceId),
		FreeformTags:           logAnalyticsEntityCopyStringMap(resource.Spec.FreeformTags),
		DefinedTags:            logAnalyticsEntityDefinedTags(resource.Spec.DefinedTags),
		AssociatedSourcesCount: logAnalyticsEntityIntPtr(3),
	}
}

func assertLogAnalyticsEntityCreateBody(
	t *testing.T,
	body loganalyticssdk.CreateLogAnalyticsEntityDetails,
	resource *loganalyticsv1beta1.LogAnalyticsEntity,
) {
	t.Helper()
	requireLogAnalyticsEntityStringPtr(t, "create name", body.Name, resource.Spec.Name)
	requireLogAnalyticsEntityStringPtr(t, "create compartmentId", body.CompartmentId, resource.Spec.CompartmentId)
	requireLogAnalyticsEntityStringPtr(t, "create entityTypeName", body.EntityTypeName, resource.Spec.EntityTypeName)
	requireLogAnalyticsEntityStringPtr(t, "create managementAgentId", body.ManagementAgentId, resource.Spec.ManagementAgentId)
	requireLogAnalyticsEntityStringPtr(t, "create sourceId", body.SourceId, resource.Spec.SourceId)
	if !reflect.DeepEqual(body.Properties, resource.Spec.Properties) {
		t.Fatalf("create properties = %#v, want %#v", body.Properties, resource.Spec.Properties)
	}
	if !reflect.DeepEqual(body.FreeformTags, resource.Spec.FreeformTags) {
		t.Fatalf("create freeformTags = %#v, want %#v", body.FreeformTags, resource.Spec.FreeformTags)
	}
	if got := body.DefinedTags; !logAnalyticsEntityJSONEqual(got, logAnalyticsEntityDefinedTags(resource.Spec.DefinedTags)) {
		t.Fatalf("create definedTags = %#v, want %#v", got, resource.Spec.DefinedTags)
	}
	if body.TimeLastDiscovered == nil || !body.TimeLastDiscovered.Equal(logAnalyticsEntitySDKTime(resource.Spec.TimeLastDiscovered).Time) {
		t.Fatalf("create timeLastDiscovered = %#v, want %s", body.TimeLastDiscovered, resource.Spec.TimeLastDiscovered)
	}
	if body.Metadata == nil || len(body.Metadata.Items) != 1 {
		t.Fatalf("create metadata = %#v, want one item", body.Metadata)
	}
}

func assertLogAnalyticsEntityLatestCondition(
	t *testing.T,
	resource *loganalyticsv1beta1.LogAnalyticsEntity,
	want shared.OSOKConditionType,
) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("conditions = nil, want latest %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("latest condition = %q, want %q", got, want)
	}
}

func requireLogAnalyticsEntityStringPtr(t *testing.T, label string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", label, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", label, *got, want)
	}
}

func logAnalyticsEntitySDKTime(value string) *common.SDKTime {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		panic(err)
	}
	return &common.SDKTime{Time: parsed}
}

func optionalLogAnalyticsEntityString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return common.String(value)
}

func logAnalyticsEntityBoolPtr(value bool) *bool {
	return &value
}

func logAnalyticsEntityIntPtr(value int) *int {
	return &value
}

func logAnalyticsEntityCopyStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	copied := make(map[string]string, len(source))
	for key, value := range source {
		copied[key] = value
	}
	return copied
}

func logAnalyticsEntityMetadataSummary(
	metadata loganalyticsv1beta1.LogAnalyticsEntityMetadata,
) *loganalyticssdk.LogAnalyticsMetadataSummary {
	if metadata.Items == nil {
		return nil
	}
	return &loganalyticssdk.LogAnalyticsMetadataSummary{
		Items: logAnalyticsEntityMetadataItems(metadata),
	}
}

func logAnalyticsEntityMetadataItems(
	metadata loganalyticsv1beta1.LogAnalyticsEntityMetadata,
) []loganalyticssdk.LogAnalyticsMetadata {
	items := make([]loganalyticssdk.LogAnalyticsMetadata, 0, len(metadata.Items))
	for _, item := range metadata.Items {
		items = append(items, loganalyticssdk.LogAnalyticsMetadata{
			Name:  optionalLogAnalyticsEntityString(item.Name),
			Value: optionalLogAnalyticsEntityString(item.Value),
			Type:  optionalLogAnalyticsEntityString(item.Type),
		})
	}
	return items
}

/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package loganalyticsentitytype

import (
	"context"
	"errors"
	"strings"
	"testing"

	loganalyticssdk "github.com/oracle/oci-go-sdk/v65/loganalytics"
	objectstoragesdk "github.com/oracle/oci-go-sdk/v65/objectstorage"
	loganalyticsv1beta1 "github.com/oracle/oci-service-operator/api/loganalytics/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const resolvedLogAnalyticsEntityTypeNamespace = "oci-loganalytics-namespace"

func TestLogAnalyticsEntityTypeCreateProjectsStatusAndSyntheticIdentity(t *testing.T) {
	resource := newLogAnalyticsEntityTypeResource()
	fake := &fakeLogAnalyticsEntityTypeOCIClient{
		createFunc: func(_ context.Context, request loganalyticssdk.CreateLogAnalyticsEntityTypeRequest) (loganalyticssdk.CreateLogAnalyticsEntityTypeResponse, error) {
			assertCreateLogAnalyticsEntityTypeRequest(t, request, resolvedLogAnalyticsEntityTypeNamespace, resource.Spec.Name, resource.Spec.Category, 1)
			return loganalyticssdk.CreateLogAnalyticsEntityTypeResponse{OpcRequestId: stringPtr("opc-create")}, nil
		},
		getFunc: func(_ context.Context, request loganalyticssdk.GetLogAnalyticsEntityTypeRequest) (loganalyticssdk.GetLogAnalyticsEntityTypeResponse, error) {
			assertGetLogAnalyticsEntityTypeRequest(t, request, resolvedLogAnalyticsEntityTypeNamespace, resource.Spec.Name)
			return loganalyticssdk.GetLogAnalyticsEntityTypeResponse{
				LogAnalyticsEntityType: logAnalyticsEntityTypeBody(resource.Spec.Name, "custom_internal", resource.Spec.Category, string(loganalyticssdk.EntityLifecycleStatesActive), logAnalyticsEntityTypePropertiesFromSpec(resource.Spec.Properties)),
				OpcRequestId:           stringPtr("opc-get"),
			}, nil
		},
	}

	response, err := newTestLogAnalyticsEntityTypeClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireCreateOrUpdateSuccess(t, response, err)
	assertInt(t, "CreateLogAnalyticsEntityType calls", len(fake.createRequests), 1)
	assertInt(t, "GetLogAnalyticsEntityType calls", len(fake.getRequests), 1)
	assertString(t, "status.ocid", string(resource.Status.OsokStatus.Ocid), "custom_internal")
	assertString(t, "status.internalName", resource.Status.InternalName, "custom_internal")
	assertString(t, "status.opcRequestId", resource.Status.OsokStatus.OpcRequestID, "opc-create")
}

func TestLogAnalyticsEntityTypeCreateFailsWhenNamespaceLookupFails(t *testing.T) {
	resource := newLogAnalyticsEntityTypeResource()
	fake := &fakeLogAnalyticsEntityTypeOCIClient{
		createFunc: func(context.Context, loganalyticssdk.CreateLogAnalyticsEntityTypeRequest) (loganalyticssdk.CreateLogAnalyticsEntityTypeResponse, error) {
			t.Fatal("CreateLogAnalyticsEntityType called after namespace lookup failure")
			return loganalyticssdk.CreateLogAnalyticsEntityTypeResponse{}, nil
		},
	}
	namespaceGetter := &fakeLogAnalyticsEntityTypeNamespaceGetter{err: errors.New("namespace lookup failed")}

	response, err := newLogAnalyticsEntityTypeServiceClientWithOCIClientAndNamespaceGetter(
		loggerutil.OSOKLogger{},
		fake,
		namespaceGetter,
	).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want namespace lookup failure")
	}
	if !strings.Contains(err.Error(), "lookup LogAnalyticsEntityType namespace") {
		t.Fatalf("CreateOrUpdate() error = %q, want namespace lookup context", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() success = true, want failure")
	}
	if got := len(fake.createRequests); got != 0 {
		t.Fatalf("CreateLogAnalyticsEntityType calls = %d, want 0", got)
	}
}

func TestLogAnalyticsEntityTypeBindsExistingThroughPaginatedList(t *testing.T) {
	resource := newLogAnalyticsEntityTypeResource()
	fake := &fakeLogAnalyticsEntityTypeOCIClient{
		getFunc: func(_ context.Context, request loganalyticssdk.GetLogAnalyticsEntityTypeRequest) (loganalyticssdk.GetLogAnalyticsEntityTypeResponse, error) {
			assertGetLogAnalyticsEntityTypeRequest(t, request, resolvedLogAnalyticsEntityTypeNamespace, "custom_internal")
			return loganalyticssdk.GetLogAnalyticsEntityTypeResponse{
				LogAnalyticsEntityType: logAnalyticsEntityTypeBody(resource.Spec.Name, "custom_internal", resource.Spec.Category, string(loganalyticssdk.EntityLifecycleStatesActive), logAnalyticsEntityTypePropertiesFromSpec(resource.Spec.Properties)),
			}, nil
		},
		listFunc: func(_ context.Context, request loganalyticssdk.ListLogAnalyticsEntityTypesRequest) (loganalyticssdk.ListLogAnalyticsEntityTypesResponse, error) {
			assertListLogAnalyticsEntityTypesRequest(t, request, resolvedLogAnalyticsEntityTypeNamespace, resource.Spec.Name)
			if request.Page == nil {
				return loganalyticssdk.ListLogAnalyticsEntityTypesResponse{
					LogAnalyticsEntityTypeCollection: loganalyticssdk.LogAnalyticsEntityTypeCollection{
						Items: []loganalyticssdk.LogAnalyticsEntityTypeSummary{
							logAnalyticsEntityTypeSummary("OtherType", "other_internal", resource.Spec.Category, string(loganalyticssdk.EntityLifecycleStatesActive)),
						},
					},
					OpcNextPage: stringPtr("page-2"),
				}, nil
			}
			if got, want := derefString(request.Page), "page-2"; got != want {
				t.Fatalf("ListLogAnalyticsEntityTypes page = %q, want %q", got, want)
			}
			return loganalyticssdk.ListLogAnalyticsEntityTypesResponse{
				LogAnalyticsEntityTypeCollection: loganalyticssdk.LogAnalyticsEntityTypeCollection{
					Items: []loganalyticssdk.LogAnalyticsEntityTypeSummary{
						logAnalyticsEntityTypeSummary(resource.Spec.Name, "custom_internal", resource.Spec.Category, string(loganalyticssdk.EntityLifecycleStatesActive)),
					},
				},
			}, nil
		},
	}

	response, err := newTestLogAnalyticsEntityTypeClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireCreateOrUpdateSuccess(t, response, err)
	assertInt(t, "CreateLogAnalyticsEntityType calls", len(fake.createRequests), 0)
	assertInt(t, "ListLogAnalyticsEntityTypes calls", len(fake.listRequests), 2)
	assertInt(t, "GetLogAnalyticsEntityType calls", len(fake.getRequests), 1)
	assertString(t, "status.ocid", string(resource.Status.OsokStatus.Ocid), "custom_internal")
	assertString(t, "status.internalName", resource.Status.InternalName, "custom_internal")
}

func TestLogAnalyticsEntityTypeBindListUsesInternalNameForSameReconcileUpdate(t *testing.T) {
	resource := newLogAnalyticsEntityTypeResource()
	resource.Spec.Category = "updated"
	resource.Spec.Properties = []loganalyticsv1beta1.LogAnalyticsEntityTypeProperty{{Name: "host", Description: "updated host"}}

	getCalls := 0
	fake := &fakeLogAnalyticsEntityTypeOCIClient{
		listFunc: func(_ context.Context, request loganalyticssdk.ListLogAnalyticsEntityTypesRequest) (loganalyticssdk.ListLogAnalyticsEntityTypesResponse, error) {
			assertListLogAnalyticsEntityTypesRequest(t, request, resolvedLogAnalyticsEntityTypeNamespace, resource.Spec.Name)
			return loganalyticssdk.ListLogAnalyticsEntityTypesResponse{
				LogAnalyticsEntityTypeCollection: loganalyticssdk.LogAnalyticsEntityTypeCollection{
					Items: []loganalyticssdk.LogAnalyticsEntityTypeSummary{
						logAnalyticsEntityTypeSummary(resource.Spec.Name, "custom_internal", "old", string(loganalyticssdk.EntityLifecycleStatesActive)),
					},
				},
			}, nil
		},
		getFunc: func(_ context.Context, request loganalyticssdk.GetLogAnalyticsEntityTypeRequest) (loganalyticssdk.GetLogAnalyticsEntityTypeResponse, error) {
			assertGetLogAnalyticsEntityTypeRequest(t, request, resolvedLogAnalyticsEntityTypeNamespace, "custom_internal")
			getCalls++
			if getCalls == 1 {
				return loganalyticssdk.GetLogAnalyticsEntityTypeResponse{
					LogAnalyticsEntityType: logAnalyticsEntityTypeBody(resource.Spec.Name, "custom_internal", "old", string(loganalyticssdk.EntityLifecycleStatesActive), []loganalyticssdk.EntityTypeProperty{{Name: stringPtr("host"), Description: stringPtr("old host")}}),
				}, nil
			}
			return loganalyticssdk.GetLogAnalyticsEntityTypeResponse{
				LogAnalyticsEntityType: logAnalyticsEntityTypeBody(resource.Spec.Name, "custom_internal", resource.Spec.Category, string(loganalyticssdk.EntityLifecycleStatesActive), logAnalyticsEntityTypePropertiesFromSpec(resource.Spec.Properties)),
			}, nil
		},
		updateFunc: func(_ context.Context, request loganalyticssdk.UpdateLogAnalyticsEntityTypeRequest) (loganalyticssdk.UpdateLogAnalyticsEntityTypeResponse, error) {
			assertUpdateLogAnalyticsEntityTypeRequest(t, request, resolvedLogAnalyticsEntityTypeNamespace, "custom_internal", resource.Spec.Category, 1)
			return loganalyticssdk.UpdateLogAnalyticsEntityTypeResponse{OpcRequestId: stringPtr("opc-update")}, nil
		},
	}

	response, err := newTestLogAnalyticsEntityTypeClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireCreateOrUpdateSuccess(t, response, err)
	assertInt(t, "CreateLogAnalyticsEntityType calls", len(fake.createRequests), 0)
	assertInt(t, "UpdateLogAnalyticsEntityType calls", len(fake.updateRequests), 1)
	assertString(t, "status.internalName", resource.Status.InternalName, "custom_internal")
}

func TestLogAnalyticsEntityTypeMutableUpdateUsesInternalNamePath(t *testing.T) {
	resource := newLogAnalyticsEntityTypeResource()
	resource.Status.InternalName = "custom_internal"
	resource.Status.OsokStatus.Ocid = shared.OCID("custom_internal")
	resource.Spec.Category = "updated"
	resource.Spec.Properties = []loganalyticsv1beta1.LogAnalyticsEntityTypeProperty{{Name: "host", Description: "updated host"}}

	getCalls := 0
	fake := &fakeLogAnalyticsEntityTypeOCIClient{
		getFunc: func(_ context.Context, request loganalyticssdk.GetLogAnalyticsEntityTypeRequest) (loganalyticssdk.GetLogAnalyticsEntityTypeResponse, error) {
			assertGetLogAnalyticsEntityTypeRequest(t, request, resolvedLogAnalyticsEntityTypeNamespace, "custom_internal")
			getCalls++
			if getCalls == 1 {
				return loganalyticssdk.GetLogAnalyticsEntityTypeResponse{
					LogAnalyticsEntityType: logAnalyticsEntityTypeBody(resource.Spec.Name, "custom_internal", "old", string(loganalyticssdk.EntityLifecycleStatesActive), []loganalyticssdk.EntityTypeProperty{{Name: stringPtr("host"), Description: stringPtr("old host")}}),
				}, nil
			}
			return loganalyticssdk.GetLogAnalyticsEntityTypeResponse{
				LogAnalyticsEntityType: logAnalyticsEntityTypeBody(resource.Spec.Name, "custom_internal", resource.Spec.Category, string(loganalyticssdk.EntityLifecycleStatesActive), logAnalyticsEntityTypePropertiesFromSpec(resource.Spec.Properties)),
			}, nil
		},
		updateFunc: func(_ context.Context, request loganalyticssdk.UpdateLogAnalyticsEntityTypeRequest) (loganalyticssdk.UpdateLogAnalyticsEntityTypeResponse, error) {
			assertUpdateLogAnalyticsEntityTypeRequest(t, request, resolvedLogAnalyticsEntityTypeNamespace, "custom_internal", resource.Spec.Category, 1)
			return loganalyticssdk.UpdateLogAnalyticsEntityTypeResponse{OpcRequestId: stringPtr("opc-update")}, nil
		},
	}

	response, err := newTestLogAnalyticsEntityTypeClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireCreateOrUpdateSuccess(t, response, err)
	assertInt(t, "UpdateLogAnalyticsEntityType calls", len(fake.updateRequests), 1)
	assertString(t, "status.category", resource.Status.Category, "updated")
	assertString(t, "status.opcRequestId", resource.Status.OsokStatus.OpcRequestID, "opc-update")
}

func TestLogAnalyticsEntityTypeRejectsNameDriftBeforeUpdate(t *testing.T) {
	resource := newLogAnalyticsEntityTypeResource()
	resource.Status.InternalName = "custom_internal"
	resource.Status.OsokStatus.Ocid = shared.OCID("custom_internal")
	resource.Spec.Name = "ChangedType"
	fake := &fakeLogAnalyticsEntityTypeOCIClient{
		getFunc: func(context.Context, loganalyticssdk.GetLogAnalyticsEntityTypeRequest) (loganalyticssdk.GetLogAnalyticsEntityTypeResponse, error) {
			return loganalyticssdk.GetLogAnalyticsEntityTypeResponse{
				LogAnalyticsEntityType: logAnalyticsEntityTypeBody("OriginalType", "custom_internal", resource.Spec.Category, string(loganalyticssdk.EntityLifecycleStatesActive), nil),
			}, nil
		},
	}

	response, err := newTestLogAnalyticsEntityTypeClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want immutable name drift rejection")
	}
	if !strings.Contains(err.Error(), "name changes") {
		t.Fatalf("CreateOrUpdate() error = %q, want name drift rejection", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() success = true, want failure")
	}
	if got := len(fake.updateRequests); got != 0 {
		t.Fatalf("UpdateLogAnalyticsEntityType calls = %d, want 0", got)
	}
}

func TestLogAnalyticsEntityTypeDeleteWaitsAndThenConfirmsDeleted(t *testing.T) {
	resource := newLogAnalyticsEntityTypeResource()
	resource.Status.InternalName = "custom_internal"
	resource.Status.OsokStatus.Ocid = shared.OCID("custom_internal")

	getCalls := 0
	fake := &fakeLogAnalyticsEntityTypeOCIClient{
		getFunc: func(context.Context, loganalyticssdk.GetLogAnalyticsEntityTypeRequest) (loganalyticssdk.GetLogAnalyticsEntityTypeResponse, error) {
			getCalls++
			state := string(loganalyticssdk.EntityLifecycleStatesActive)
			if getCalls == 2 {
				state = string(loganalyticssdk.EntityLifecycleStatesDeleted)
			}
			return loganalyticssdk.GetLogAnalyticsEntityTypeResponse{
				LogAnalyticsEntityType: logAnalyticsEntityTypeBody(resource.Spec.Name, "custom_internal", resource.Spec.Category, state, nil),
			}, nil
		},
		deleteFunc: func(_ context.Context, request loganalyticssdk.DeleteLogAnalyticsEntityTypeRequest) (loganalyticssdk.DeleteLogAnalyticsEntityTypeResponse, error) {
			assertDeleteLogAnalyticsEntityTypeRequest(t, request, resolvedLogAnalyticsEntityTypeNamespace, "custom_internal")
			return loganalyticssdk.DeleteLogAnalyticsEntityTypeResponse{OpcRequestId: stringPtr("opc-delete")}, nil
		},
	}

	deleted, err := newTestLogAnalyticsEntityTypeClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() = false, want deleted after terminal readback")
	}
	if got, want := len(fake.deleteRequests), 1; got != want {
		t.Fatalf("DeleteLogAnalyticsEntityType calls = %d, want %d", got, want)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete confirmation timestamp")
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-delete"; got != want {
		t.Fatalf("status.opcRequestId = %q, want %q", got, want)
	}
}

func TestLogAnalyticsEntityTypeDeleteRetainsFinalizerWhileReadbackRemainsActive(t *testing.T) {
	resource := newLogAnalyticsEntityTypeResource()
	resource.Status.InternalName = "custom_internal"
	resource.Status.OsokStatus.Ocid = shared.OCID("custom_internal")
	fake := &fakeLogAnalyticsEntityTypeOCIClient{
		getFunc: func(context.Context, loganalyticssdk.GetLogAnalyticsEntityTypeRequest) (loganalyticssdk.GetLogAnalyticsEntityTypeResponse, error) {
			return loganalyticssdk.GetLogAnalyticsEntityTypeResponse{
				LogAnalyticsEntityType: logAnalyticsEntityTypeBody(resource.Spec.Name, "custom_internal", resource.Spec.Category, string(loganalyticssdk.EntityLifecycleStatesActive), nil),
			}, nil
		},
		deleteFunc: func(context.Context, loganalyticssdk.DeleteLogAnalyticsEntityTypeRequest) (loganalyticssdk.DeleteLogAnalyticsEntityTypeResponse, error) {
			return loganalyticssdk.DeleteLogAnalyticsEntityTypeResponse{OpcRequestId: stringPtr("opc-delete")}, nil
		},
	}

	deleted, err := newTestLogAnalyticsEntityTypeClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() = true, want finalizer retained while readback remains ACTIVE")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt set, want no delete timestamp while readback remains ACTIVE")
	}
	if resource.Status.OsokStatus.Async.Current == nil || resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.async.current = %#v, want pending delete async operation", resource.Status.OsokStatus.Async.Current)
	}
}

func TestLogAnalyticsEntityTypeDeleteAlreadyPendingDoesNotReissueDeleteWhileReadbackActive(t *testing.T) {
	resource := newLogAnalyticsEntityTypeResource()
	resource.Status.InternalName = "custom_internal"
	resource.Status.OsokStatus.Ocid = shared.OCID("custom_internal")
	now := metav1.Now()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         "delete pending",
		UpdatedAt:       &now,
	}
	fake := &fakeLogAnalyticsEntityTypeOCIClient{
		getFunc: func(context.Context, loganalyticssdk.GetLogAnalyticsEntityTypeRequest) (loganalyticssdk.GetLogAnalyticsEntityTypeResponse, error) {
			return loganalyticssdk.GetLogAnalyticsEntityTypeResponse{
				LogAnalyticsEntityType: logAnalyticsEntityTypeBody(resource.Spec.Name, "custom_internal", resource.Spec.Category, string(loganalyticssdk.EntityLifecycleStatesActive), nil),
			}, nil
		},
		deleteFunc: func(context.Context, loganalyticssdk.DeleteLogAnalyticsEntityTypeRequest) (loganalyticssdk.DeleteLogAnalyticsEntityTypeResponse, error) {
			t.Fatal("DeleteLogAnalyticsEntityType called for already-pending delete")
			return loganalyticssdk.DeleteLogAnalyticsEntityTypeResponse{}, nil
		},
	}

	deleted, err := newTestLogAnalyticsEntityTypeClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() = true, want finalizer retained while pending delete readback remains ACTIVE")
	}
	if got, want := len(fake.getRequests), 1; got != want {
		t.Fatalf("GetLogAnalyticsEntityType calls = %d, want %d", got, want)
	}
	if got := len(fake.deleteRequests); got != 0 {
		t.Fatalf("DeleteLogAnalyticsEntityType calls = %d, want 0", got)
	}
	if resource.Status.OsokStatus.Async.Current == nil || resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.async.current = %#v, want pending delete async operation", resource.Status.OsokStatus.Async.Current)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt set, want no delete timestamp while readback remains ACTIVE")
	}
}

func TestLogAnalyticsEntityTypeDeleteWithoutTrackedInternalNameUsesPaginatedList(t *testing.T) {
	resource := newLogAnalyticsEntityTypeResource()

	getCalls := 0
	fake := &fakeLogAnalyticsEntityTypeOCIClient{
		listFunc: func(_ context.Context, request loganalyticssdk.ListLogAnalyticsEntityTypesRequest) (loganalyticssdk.ListLogAnalyticsEntityTypesResponse, error) {
			assertListLogAnalyticsEntityTypesRequest(t, request, resolvedLogAnalyticsEntityTypeNamespace, resource.Spec.Name)
			if request.Page == nil {
				return loganalyticssdk.ListLogAnalyticsEntityTypesResponse{
					LogAnalyticsEntityTypeCollection: loganalyticssdk.LogAnalyticsEntityTypeCollection{
						Items: []loganalyticssdk.LogAnalyticsEntityTypeSummary{
							logAnalyticsEntityTypeSummary("OtherType", "other_internal", resource.Spec.Category, string(loganalyticssdk.EntityLifecycleStatesActive)),
						},
					},
					OpcNextPage: stringPtr("page-2"),
				}, nil
			}
			if got, want := derefString(request.Page), "page-2"; got != want {
				t.Fatalf("ListLogAnalyticsEntityTypes page = %q, want %q", got, want)
			}
			return loganalyticssdk.ListLogAnalyticsEntityTypesResponse{
				LogAnalyticsEntityTypeCollection: loganalyticssdk.LogAnalyticsEntityTypeCollection{
					Items: []loganalyticssdk.LogAnalyticsEntityTypeSummary{
						logAnalyticsEntityTypeSummary(resource.Spec.Name, "custom_internal", resource.Spec.Category, string(loganalyticssdk.EntityLifecycleStatesActive)),
					},
				},
			}, nil
		},
		getFunc: func(_ context.Context, request loganalyticssdk.GetLogAnalyticsEntityTypeRequest) (loganalyticssdk.GetLogAnalyticsEntityTypeResponse, error) {
			assertGetLogAnalyticsEntityTypeRequest(t, request, resolvedLogAnalyticsEntityTypeNamespace, "custom_internal")
			getCalls++
			state := string(loganalyticssdk.EntityLifecycleStatesActive)
			if getCalls == 2 {
				state = string(loganalyticssdk.EntityLifecycleStatesDeleted)
			}
			return loganalyticssdk.GetLogAnalyticsEntityTypeResponse{
				LogAnalyticsEntityType: logAnalyticsEntityTypeBody(resource.Spec.Name, "custom_internal", resource.Spec.Category, state, nil),
			}, nil
		},
		deleteFunc: func(_ context.Context, request loganalyticssdk.DeleteLogAnalyticsEntityTypeRequest) (loganalyticssdk.DeleteLogAnalyticsEntityTypeResponse, error) {
			assertDeleteLogAnalyticsEntityTypeRequest(t, request, resolvedLogAnalyticsEntityTypeNamespace, "custom_internal")
			return loganalyticssdk.DeleteLogAnalyticsEntityTypeResponse{OpcRequestId: stringPtr("opc-delete")}, nil
		},
	}

	deleted, err := newTestLogAnalyticsEntityTypeClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() = false, want deleted after list-discovered internal name and terminal readback")
	}
	assertInt(t, "ListLogAnalyticsEntityTypes calls", len(fake.listRequests), 2)
	assertInt(t, "DeleteLogAnalyticsEntityType calls", len(fake.deleteRequests), 1)
	assertString(t, "status.internalName", resource.Status.InternalName, "custom_internal")
}

func TestLogAnalyticsEntityTypeDeleteWithoutTrackedInternalNameTreatsListAbsenceAsDeleted(t *testing.T) {
	resource := newLogAnalyticsEntityTypeResource()
	fake := &fakeLogAnalyticsEntityTypeOCIClient{
		listFunc: func(_ context.Context, request loganalyticssdk.ListLogAnalyticsEntityTypesRequest) (loganalyticssdk.ListLogAnalyticsEntityTypesResponse, error) {
			assertListLogAnalyticsEntityTypesRequest(t, request, resolvedLogAnalyticsEntityTypeNamespace, resource.Spec.Name)
			if request.Page == nil {
				return loganalyticssdk.ListLogAnalyticsEntityTypesResponse{
					LogAnalyticsEntityTypeCollection: loganalyticssdk.LogAnalyticsEntityTypeCollection{
						Items: []loganalyticssdk.LogAnalyticsEntityTypeSummary{
							logAnalyticsEntityTypeSummary("OtherType", "other_internal", resource.Spec.Category, string(loganalyticssdk.EntityLifecycleStatesActive)),
						},
					},
					OpcNextPage: stringPtr("page-2"),
				}, nil
			}
			if got, want := derefString(request.Page), "page-2"; got != want {
				t.Fatalf("ListLogAnalyticsEntityTypes page = %q, want %q", got, want)
			}
			return loganalyticssdk.ListLogAnalyticsEntityTypesResponse{}, nil
		},
		getFunc: func(context.Context, loganalyticssdk.GetLogAnalyticsEntityTypeRequest) (loganalyticssdk.GetLogAnalyticsEntityTypeResponse, error) {
			t.Fatal("GetLogAnalyticsEntityType called after list confirmed absence")
			return loganalyticssdk.GetLogAnalyticsEntityTypeResponse{}, nil
		},
		deleteFunc: func(context.Context, loganalyticssdk.DeleteLogAnalyticsEntityTypeRequest) (loganalyticssdk.DeleteLogAnalyticsEntityTypeResponse, error) {
			t.Fatal("DeleteLogAnalyticsEntityType called after list confirmed absence")
			return loganalyticssdk.DeleteLogAnalyticsEntityTypeResponse{}, nil
		},
	}

	deleted, err := newTestLogAnalyticsEntityTypeClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() = false, want deleted after authoritative list absence")
	}
	assertInt(t, "ListLogAnalyticsEntityTypes calls", len(fake.listRequests), 2)
	assertInt(t, "GetLogAnalyticsEntityType calls", len(fake.getRequests), 0)
	assertInt(t, "DeleteLogAnalyticsEntityType calls", len(fake.deleteRequests), 0)
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete confirmation timestamp")
	}
	if got := resource.Status.InternalName; got != "" {
		t.Fatalf("status.internalName = %q, want empty after list absence", got)
	}
}

func TestLogAnalyticsEntityTypeDeleteRejectsAuthShapedConfirmRead(t *testing.T) {
	resource := newLogAnalyticsEntityTypeResource()
	resource.Status.InternalName = "custom_internal"
	resource.Status.OsokStatus.Ocid = shared.OCID("custom_internal")

	getCalls := 0
	authNotFound := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous not found")
	authNotFound.OpcRequestID = "opc-auth-confirm"
	fake := &fakeLogAnalyticsEntityTypeOCIClient{
		getFunc: func(context.Context, loganalyticssdk.GetLogAnalyticsEntityTypeRequest) (loganalyticssdk.GetLogAnalyticsEntityTypeResponse, error) {
			getCalls++
			if getCalls == 1 {
				return loganalyticssdk.GetLogAnalyticsEntityTypeResponse{
					LogAnalyticsEntityType: logAnalyticsEntityTypeBody(resource.Spec.Name, "custom_internal", resource.Spec.Category, string(loganalyticssdk.EntityLifecycleStatesActive), nil),
				}, nil
			}
			return loganalyticssdk.GetLogAnalyticsEntityTypeResponse{}, authNotFound
		},
		deleteFunc: func(context.Context, loganalyticssdk.DeleteLogAnalyticsEntityTypeRequest) (loganalyticssdk.DeleteLogAnalyticsEntityTypeResponse, error) {
			return loganalyticssdk.DeleteLogAnalyticsEntityTypeResponse{OpcRequestId: stringPtr("opc-delete")}, nil
		},
	}

	deleted, err := newTestLogAnalyticsEntityTypeClient(fake).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped confirm read rejection")
	}
	if deleted {
		t.Fatal("Delete() = true, want finalizer retained after ambiguous 404")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %q, want ambiguous 404 message", err)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-auth-confirm"; got != want {
		t.Fatalf("status.opcRequestId = %q, want %q", got, want)
	}
}

func requireCreateOrUpdateSuccess(t *testing.T, response servicemanager.OSOKResponse, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() unsuccessful, want success")
	}
}

func assertString(t *testing.T, label string, got string, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %q, want %q", label, got, want)
	}
}

func assertInt(t *testing.T, label string, got int, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %d, want %d", label, got, want)
	}
}

func assertCreateLogAnalyticsEntityTypeRequest(
	t *testing.T,
	request loganalyticssdk.CreateLogAnalyticsEntityTypeRequest,
	namespace string,
	name string,
	category string,
	propertyCount int,
) {
	t.Helper()
	assertString(t, "CreateLogAnalyticsEntityType namespace", derefString(request.NamespaceName), namespace)
	assertString(t, "CreateLogAnalyticsEntityType name", derefString(request.Name), name)
	assertString(t, "CreateLogAnalyticsEntityType category", derefString(request.Category), category)
	assertInt(t, "CreateLogAnalyticsEntityType properties", len(request.Properties), propertyCount)
}

func assertGetLogAnalyticsEntityTypeRequest(
	t *testing.T,
	request loganalyticssdk.GetLogAnalyticsEntityTypeRequest,
	namespace string,
	entityTypeName string,
) {
	t.Helper()
	assertString(t, "GetLogAnalyticsEntityType namespace", derefString(request.NamespaceName), namespace)
	assertString(t, "GetLogAnalyticsEntityType entityTypeName", derefString(request.EntityTypeName), entityTypeName)
}

func assertListLogAnalyticsEntityTypesRequest(
	t *testing.T,
	request loganalyticssdk.ListLogAnalyticsEntityTypesRequest,
	namespace string,
	name string,
) {
	t.Helper()
	assertString(t, "ListLogAnalyticsEntityTypes namespace", derefString(request.NamespaceName), namespace)
	assertString(t, "ListLogAnalyticsEntityTypes name", derefString(request.Name), name)
}

func assertUpdateLogAnalyticsEntityTypeRequest(
	t *testing.T,
	request loganalyticssdk.UpdateLogAnalyticsEntityTypeRequest,
	namespace string,
	entityTypeName string,
	category string,
	propertyCount int,
) {
	t.Helper()
	assertString(t, "UpdateLogAnalyticsEntityType namespace", derefString(request.NamespaceName), namespace)
	assertString(t, "UpdateLogAnalyticsEntityType entityTypeName", derefString(request.EntityTypeName), entityTypeName)
	assertString(t, "UpdateLogAnalyticsEntityType category", derefString(request.Category), category)
	assertInt(t, "UpdateLogAnalyticsEntityType properties", len(request.Properties), propertyCount)
}

func assertDeleteLogAnalyticsEntityTypeRequest(
	t *testing.T,
	request loganalyticssdk.DeleteLogAnalyticsEntityTypeRequest,
	namespace string,
	entityTypeName string,
) {
	t.Helper()
	assertString(t, "DeleteLogAnalyticsEntityType namespace", derefString(request.NamespaceName), namespace)
	assertString(t, "DeleteLogAnalyticsEntityType entityTypeName", derefString(request.EntityTypeName), entityTypeName)
}

type fakeLogAnalyticsEntityTypeOCIClient struct {
	createRequests []loganalyticssdk.CreateLogAnalyticsEntityTypeRequest
	getRequests    []loganalyticssdk.GetLogAnalyticsEntityTypeRequest
	listRequests   []loganalyticssdk.ListLogAnalyticsEntityTypesRequest
	updateRequests []loganalyticssdk.UpdateLogAnalyticsEntityTypeRequest
	deleteRequests []loganalyticssdk.DeleteLogAnalyticsEntityTypeRequest

	createFunc func(context.Context, loganalyticssdk.CreateLogAnalyticsEntityTypeRequest) (loganalyticssdk.CreateLogAnalyticsEntityTypeResponse, error)
	getFunc    func(context.Context, loganalyticssdk.GetLogAnalyticsEntityTypeRequest) (loganalyticssdk.GetLogAnalyticsEntityTypeResponse, error)
	listFunc   func(context.Context, loganalyticssdk.ListLogAnalyticsEntityTypesRequest) (loganalyticssdk.ListLogAnalyticsEntityTypesResponse, error)
	updateFunc func(context.Context, loganalyticssdk.UpdateLogAnalyticsEntityTypeRequest) (loganalyticssdk.UpdateLogAnalyticsEntityTypeResponse, error)
	deleteFunc func(context.Context, loganalyticssdk.DeleteLogAnalyticsEntityTypeRequest) (loganalyticssdk.DeleteLogAnalyticsEntityTypeResponse, error)
}

func (f *fakeLogAnalyticsEntityTypeOCIClient) CreateLogAnalyticsEntityType(
	ctx context.Context,
	request loganalyticssdk.CreateLogAnalyticsEntityTypeRequest,
) (loganalyticssdk.CreateLogAnalyticsEntityTypeResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createFunc != nil {
		return f.createFunc(ctx, request)
	}
	return loganalyticssdk.CreateLogAnalyticsEntityTypeResponse{}, nil
}

func (f *fakeLogAnalyticsEntityTypeOCIClient) GetLogAnalyticsEntityType(
	ctx context.Context,
	request loganalyticssdk.GetLogAnalyticsEntityTypeRequest,
) (loganalyticssdk.GetLogAnalyticsEntityTypeResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getFunc != nil {
		return f.getFunc(ctx, request)
	}
	return loganalyticssdk.GetLogAnalyticsEntityTypeResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "missing")
}

func (f *fakeLogAnalyticsEntityTypeOCIClient) ListLogAnalyticsEntityTypes(
	ctx context.Context,
	request loganalyticssdk.ListLogAnalyticsEntityTypesRequest,
) (loganalyticssdk.ListLogAnalyticsEntityTypesResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFunc != nil {
		return f.listFunc(ctx, request)
	}
	return loganalyticssdk.ListLogAnalyticsEntityTypesResponse{}, nil
}

func (f *fakeLogAnalyticsEntityTypeOCIClient) UpdateLogAnalyticsEntityType(
	ctx context.Context,
	request loganalyticssdk.UpdateLogAnalyticsEntityTypeRequest,
) (loganalyticssdk.UpdateLogAnalyticsEntityTypeResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateFunc != nil {
		return f.updateFunc(ctx, request)
	}
	return loganalyticssdk.UpdateLogAnalyticsEntityTypeResponse{}, nil
}

func (f *fakeLogAnalyticsEntityTypeOCIClient) DeleteLogAnalyticsEntityType(
	ctx context.Context,
	request loganalyticssdk.DeleteLogAnalyticsEntityTypeRequest,
) (loganalyticssdk.DeleteLogAnalyticsEntityTypeResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteFunc != nil {
		return f.deleteFunc(ctx, request)
	}
	return loganalyticssdk.DeleteLogAnalyticsEntityTypeResponse{}, nil
}

type fakeLogAnalyticsEntityTypeNamespaceGetter struct {
	requests []objectstoragesdk.GetNamespaceRequest
	value    string
	err      error
}

func (f *fakeLogAnalyticsEntityTypeNamespaceGetter) GetNamespace(
	_ context.Context,
	request objectstoragesdk.GetNamespaceRequest,
) (objectstoragesdk.GetNamespaceResponse, error) {
	f.requests = append(f.requests, request)
	if f.err != nil {
		return objectstoragesdk.GetNamespaceResponse{}, f.err
	}
	value := f.value
	if value == "" {
		value = resolvedLogAnalyticsEntityTypeNamespace
	}
	return objectstoragesdk.GetNamespaceResponse{Value: stringPtr(value)}, nil
}

func newTestLogAnalyticsEntityTypeClient(fake *fakeLogAnalyticsEntityTypeOCIClient) LogAnalyticsEntityTypeServiceClient {
	return newLogAnalyticsEntityTypeServiceClientWithOCIClientAndNamespaceGetter(
		loggerutil.OSOKLogger{},
		fake,
		&fakeLogAnalyticsEntityTypeNamespaceGetter{},
	)
}

func newLogAnalyticsEntityTypeResource() *loganalyticsv1beta1.LogAnalyticsEntityType {
	return &loganalyticsv1beta1.LogAnalyticsEntityType{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sample",
			Namespace: "tenantns",
		},
		Spec: loganalyticsv1beta1.LogAnalyticsEntityTypeSpec{
			Name:       "CustomType",
			Category:   "custom",
			Properties: []loganalyticsv1beta1.LogAnalyticsEntityTypeProperty{{Name: "host", Description: "hostname"}},
		},
	}
}

func logAnalyticsEntityTypeBody(
	name string,
	internalName string,
	category string,
	lifecycleState string,
	properties []loganalyticssdk.EntityTypeProperty,
) loganalyticssdk.LogAnalyticsEntityType {
	return loganalyticssdk.LogAnalyticsEntityType{
		Name:                             stringPtr(name),
		InternalName:                     stringPtr(internalName),
		Category:                         stringPtr(category),
		CloudType:                        loganalyticssdk.EntityCloudTypeNonCloud,
		LifecycleState:                   loganalyticssdk.EntityLifecycleStatesEnum(lifecycleState),
		Properties:                       properties,
		ManagementAgentEligibilityStatus: loganalyticssdk.LogAnalyticsEntityTypeManagementAgentEligibilityStatusEligible,
	}
}

func logAnalyticsEntityTypeSummary(name string, internalName string, category string, lifecycleState string) loganalyticssdk.LogAnalyticsEntityTypeSummary {
	return loganalyticssdk.LogAnalyticsEntityTypeSummary{
		Name:                             stringPtr(name),
		InternalName:                     stringPtr(internalName),
		Category:                         stringPtr(category),
		CloudType:                        loganalyticssdk.EntityCloudTypeNonCloud,
		LifecycleState:                   loganalyticssdk.EntityLifecycleStatesEnum(lifecycleState),
		ManagementAgentEligibilityStatus: loganalyticssdk.LogAnalyticsEntityTypeSummaryManagementAgentEligibilityStatusEligible,
	}
}

func stringPtr(value string) *string {
	return &value
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

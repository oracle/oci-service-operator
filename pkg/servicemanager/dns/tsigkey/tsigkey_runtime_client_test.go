/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package tsigkey

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	dnssdk "github.com/oracle/oci-go-sdk/v65/dns"
	dnsv1beta1 "github.com/oracle/oci-service-operator/api/dns/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testTsigKeyID     = "ocid1.dnstsigkey.oc1..example"
	testCompartmentID = "ocid1.compartment.oc1..example"
	testTsigSecret    = "c2VjcmV0"
)

type fakeTsigKeyOCIClient struct {
	createTsigKeyFn func(context.Context, dnssdk.CreateTsigKeyRequest) (dnssdk.CreateTsigKeyResponse, error)
	getTsigKeyFn    func(context.Context, dnssdk.GetTsigKeyRequest) (dnssdk.GetTsigKeyResponse, error)
	listTsigKeysFn  func(context.Context, dnssdk.ListTsigKeysRequest) (dnssdk.ListTsigKeysResponse, error)
	updateTsigKeyFn func(context.Context, dnssdk.UpdateTsigKeyRequest) (dnssdk.UpdateTsigKeyResponse, error)
	deleteTsigKeyFn func(context.Context, dnssdk.DeleteTsigKeyRequest) (dnssdk.DeleteTsigKeyResponse, error)
}

func (f *fakeTsigKeyOCIClient) CreateTsigKey(ctx context.Context, req dnssdk.CreateTsigKeyRequest) (dnssdk.CreateTsigKeyResponse, error) {
	if f.createTsigKeyFn != nil {
		return f.createTsigKeyFn(ctx, req)
	}
	return dnssdk.CreateTsigKeyResponse{}, nil
}

func (f *fakeTsigKeyOCIClient) GetTsigKey(ctx context.Context, req dnssdk.GetTsigKeyRequest) (dnssdk.GetTsigKeyResponse, error) {
	if f.getTsigKeyFn != nil {
		return f.getTsigKeyFn(ctx, req)
	}
	return dnssdk.GetTsigKeyResponse{}, nil
}

func (f *fakeTsigKeyOCIClient) ListTsigKeys(ctx context.Context, req dnssdk.ListTsigKeysRequest) (dnssdk.ListTsigKeysResponse, error) {
	if f.listTsigKeysFn != nil {
		return f.listTsigKeysFn(ctx, req)
	}
	return dnssdk.ListTsigKeysResponse{}, nil
}

func (f *fakeTsigKeyOCIClient) UpdateTsigKey(ctx context.Context, req dnssdk.UpdateTsigKeyRequest) (dnssdk.UpdateTsigKeyResponse, error) {
	if f.updateTsigKeyFn != nil {
		return f.updateTsigKeyFn(ctx, req)
	}
	return dnssdk.UpdateTsigKeyResponse{}, nil
}

func (f *fakeTsigKeyOCIClient) DeleteTsigKey(ctx context.Context, req dnssdk.DeleteTsigKeyRequest) (dnssdk.DeleteTsigKeyResponse, error) {
	if f.deleteTsigKeyFn != nil {
		return f.deleteTsigKeyFn(ctx, req)
	}
	return dnssdk.DeleteTsigKeyResponse{}, nil
}

func testTsigKeyClient(fake *fakeTsigKeyOCIClient) TsigKeyServiceClient {
	return newTsigKeyServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)
}

func makeTsigKeyResource() *dnsv1beta1.TsigKey {
	return &dnsv1beta1.TsigKey{
		Spec: dnsv1beta1.TsigKeySpec{
			Algorithm:     "hmac-sha256",
			Name:          "example-key",
			CompartmentId: testCompartmentID,
			Secret:        testTsigSecret,
			FreeformTags:  map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func makeSDKTsigKey(
	id string,
	compartmentID string,
	name string,
	algorithm string,
	secret string,
	state dnssdk.TsigKeyLifecycleStateEnum,
) dnssdk.TsigKey {
	return dnssdk.TsigKey{
		Id:             common.String(id),
		CompartmentId:  common.String(compartmentID),
		Name:           common.String(name),
		Algorithm:      common.String(algorithm),
		Secret:         common.String(secret),
		Self:           common.String("https://dns.example.test/tsigKeys/" + id),
		LifecycleState: state,
		FreeformTags:   map[string]string{"env": "dev"},
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	}
}

func makeSDKTsigKeySummary(
	id string,
	compartmentID string,
	name string,
	algorithm string,
	state dnssdk.TsigKeySummaryLifecycleStateEnum,
) dnssdk.TsigKeySummary {
	return dnssdk.TsigKeySummary{
		Id:             common.String(id),
		CompartmentId:  common.String(compartmentID),
		Name:           common.String(name),
		Algorithm:      common.String(algorithm),
		Self:           common.String("https://dns.example.test/tsigKeys/" + id),
		LifecycleState: state,
		FreeformTags:   map[string]string{"env": "dev"},
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	}
}

func TestTsigKeyRuntimeSemanticsEncodesLifecycleContract(t *testing.T) {
	got := newTsigKeyRuntimeSemantics()
	if got == nil {
		t.Fatal("newTsigKeyRuntimeSemantics() = nil")
	}
	if got.FormalService != "dns" || got.FormalSlug != "tsigkey" {
		t.Fatalf("formal binding = %s/%s, want dns/tsigkey", got.FormalService, got.FormalSlug)
	}
	if got.Async == nil || got.Async.Strategy != "lifecycle" || got.Async.Runtime != "generatedruntime" {
		t.Fatalf("async semantics = %#v, want generatedruntime lifecycle", got.Async)
	}
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" || got.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("delete semantics = %#v followUp=%#v, want required confirm-delete", got.Delete, got.DeleteFollowUp)
	}
	assertTsigKeyStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"compartmentId", "name", "id"})
	assertTsigKeyStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"freeformTags", "definedTags"})
	assertTsigKeyStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"algorithm", "compartmentId", "name", "secret"})
}

func TestTsigKeyServiceClientCreatesAndProjectsStatus(t *testing.T) {
	resource := makeTsigKeyResource()
	listCalls := 0
	createCalls := 0
	getCalls := 0
	var createRequest dnssdk.CreateTsigKeyRequest

	client := testTsigKeyClient(&fakeTsigKeyOCIClient{
		listTsigKeysFn: func(_ context.Context, req dnssdk.ListTsigKeysRequest) (dnssdk.ListTsigKeysResponse, error) {
			listCalls++
			requireStringPtr(t, "list compartmentId", req.CompartmentId, testCompartmentID)
			requireStringPtr(t, "list name", req.Name, resource.Spec.Name)
			return dnssdk.ListTsigKeysResponse{}, nil
		},
		createTsigKeyFn: func(_ context.Context, req dnssdk.CreateTsigKeyRequest) (dnssdk.CreateTsigKeyResponse, error) {
			createCalls++
			createRequest = req
			return dnssdk.CreateTsigKeyResponse{
				TsigKey:      makeSDKTsigKey(testTsigKeyID, testCompartmentID, resource.Spec.Name, resource.Spec.Algorithm, resource.Spec.Secret, dnssdk.TsigKeyLifecycleStateCreating),
				OpcRequestId: common.String("opc-create-1"),
			}, nil
		},
		getTsigKeyFn: func(_ context.Context, req dnssdk.GetTsigKeyRequest) (dnssdk.GetTsigKeyResponse, error) {
			getCalls++
			requireStringPtr(t, "get tsigKeyId", req.TsigKeyId, testTsigKeyID)
			return dnssdk.GetTsigKeyResponse{
				TsigKey: makeSDKTsigKey(testTsigKeyID, testCompartmentID, resource.Spec.Name, resource.Spec.Algorithm, resource.Spec.Secret, dnssdk.TsigKeyLifecycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate().IsSuccessful = false, want true")
	}
	if listCalls != 1 || createCalls != 1 || getCalls != 1 {
		t.Fatalf("calls list/create/get = %d/%d/%d, want 1/1/1", listCalls, createCalls, getCalls)
	}
	requireStringPtr(t, "create algorithm", createRequest.Algorithm, resource.Spec.Algorithm)
	requireStringPtr(t, "create compartmentId", createRequest.CompartmentId, testCompartmentID)
	requireStringPtr(t, "create name", createRequest.Name, resource.Spec.Name)
	requireStringPtr(t, "create secret", createRequest.Secret, testTsigSecret)
	if !reflect.DeepEqual(createRequest.FreeformTags, resource.Spec.FreeformTags) {
		t.Fatalf("create freeformTags = %#v, want %#v", createRequest.FreeformTags, resource.Spec.FreeformTags)
	}
	if got := resource.Status.Id; got != testTsigKeyID {
		t.Fatalf("status.id = %q, want %q", got, testTsigKeyID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testTsigKeyID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testTsigKeyID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-1", got)
	}
}

func TestTsigKeyServiceClientBindsExistingAcrossListPages(t *testing.T) {
	resource := makeTsigKeyResource()
	listCalls := 0
	createCalls := 0
	getCalls := 0

	client := testTsigKeyClient(&fakeTsigKeyOCIClient{
		listTsigKeysFn: func(_ context.Context, req dnssdk.ListTsigKeysRequest) (dnssdk.ListTsigKeysResponse, error) {
			listCalls++
			switch listCalls {
			case 1:
				if req.Page != nil {
					t.Fatalf("first list page = %q, want nil", *req.Page)
				}
				return dnssdk.ListTsigKeysResponse{
					Items: []dnssdk.TsigKeySummary{
						makeSDKTsigKeySummary("ocid1.dnstsigkey.oc1..other", testCompartmentID, "other-key", resource.Spec.Algorithm, dnssdk.TsigKeySummaryLifecycleStateActive),
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			case 2:
				requireStringPtr(t, "second list page", req.Page, "page-2")
				return dnssdk.ListTsigKeysResponse{
					Items: []dnssdk.TsigKeySummary{
						makeSDKTsigKeySummary(testTsigKeyID, testCompartmentID, resource.Spec.Name, resource.Spec.Algorithm, dnssdk.TsigKeySummaryLifecycleStateActive),
					},
				}, nil
			default:
				t.Fatalf("unexpected list call %d", listCalls)
				return dnssdk.ListTsigKeysResponse{}, nil
			}
		},
		getTsigKeyFn: func(_ context.Context, req dnssdk.GetTsigKeyRequest) (dnssdk.GetTsigKeyResponse, error) {
			getCalls++
			requireStringPtr(t, "get tsigKeyId", req.TsigKeyId, testTsigKeyID)
			return dnssdk.GetTsigKeyResponse{
				TsigKey: makeSDKTsigKey(testTsigKeyID, testCompartmentID, resource.Spec.Name, resource.Spec.Algorithm, resource.Spec.Secret, dnssdk.TsigKeyLifecycleStateActive),
			}, nil
		},
		createTsigKeyFn: func(context.Context, dnssdk.CreateTsigKeyRequest) (dnssdk.CreateTsigKeyResponse, error) {
			createCalls++
			return dnssdk.CreateTsigKeyResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate().IsSuccessful = false, want true")
	}
	if listCalls != 2 || getCalls != 1 || createCalls != 0 {
		t.Fatalf("calls list/get/create = %d/%d/%d, want 2/1/0", listCalls, getCalls, createCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testTsigKeyID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testTsigKeyID)
	}
}

func TestTsigKeyServiceClientRejectsDuplicateListMatchesAcrossPages(t *testing.T) {
	resource := makeTsigKeyResource()
	listCalls := 0
	createCalls := 0

	client := testTsigKeyClient(&fakeTsigKeyOCIClient{
		listTsigKeysFn: func(_ context.Context, req dnssdk.ListTsigKeysRequest) (dnssdk.ListTsigKeysResponse, error) {
			listCalls++
			if listCalls == 1 {
				return dnssdk.ListTsigKeysResponse{
					Items:       []dnssdk.TsigKeySummary{makeSDKTsigKeySummary(testTsigKeyID, testCompartmentID, resource.Spec.Name, resource.Spec.Algorithm, dnssdk.TsigKeySummaryLifecycleStateActive)},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			requireStringPtr(t, "second list page", req.Page, "page-2")
			return dnssdk.ListTsigKeysResponse{
				Items: []dnssdk.TsigKeySummary{makeSDKTsigKeySummary("ocid1.dnstsigkey.oc1..duplicate", testCompartmentID, resource.Spec.Name, resource.Spec.Algorithm, dnssdk.TsigKeySummaryLifecycleStateActive)},
			}, nil
		},
		createTsigKeyFn: func(context.Context, dnssdk.CreateTsigKeyRequest) (dnssdk.CreateTsigKeyResponse, error) {
			createCalls++
			return dnssdk.CreateTsigKeyResponse{}, nil
		},
	})

	_, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want duplicate list match error")
	}
	if !strings.Contains(err.Error(), "multiple matching resources") {
		t.Fatalf("CreateOrUpdate() error = %q, want multiple matching resources", err.Error())
	}
	if createCalls != 0 {
		t.Fatalf("create calls = %d, want 0", createCalls)
	}
}

func TestTsigKeyServiceClientNoopsWhenObservedStateMatches(t *testing.T) {
	resource := makeTsigKeyResource()
	resource.Status.Id = testTsigKeyID
	resource.Status.OsokStatus.Ocid = shared.OCID(testTsigKeyID)
	updateCalls := 0

	client := testTsigKeyClient(&fakeTsigKeyOCIClient{
		getTsigKeyFn: func(_ context.Context, req dnssdk.GetTsigKeyRequest) (dnssdk.GetTsigKeyResponse, error) {
			requireStringPtr(t, "get tsigKeyId", req.TsigKeyId, testTsigKeyID)
			return dnssdk.GetTsigKeyResponse{
				TsigKey: makeSDKTsigKey(testTsigKeyID, testCompartmentID, resource.Spec.Name, resource.Spec.Algorithm, resource.Spec.Secret, dnssdk.TsigKeyLifecycleStateActive),
			}, nil
		},
		updateTsigKeyFn: func(context.Context, dnssdk.UpdateTsigKeyRequest) (dnssdk.UpdateTsigKeyResponse, error) {
			updateCalls++
			return dnssdk.UpdateTsigKeyResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate().IsSuccessful = false, want true")
	}
	if updateCalls != 0 {
		t.Fatalf("update calls = %d, want 0", updateCalls)
	}
}

func TestTsigKeyServiceClientUpdatesMutableTagsAndCanClearTags(t *testing.T) {
	resource := makeTsigKeyResource()
	resource.Spec.FreeformTags = nil
	resource.Spec.DefinedTags = map[string]shared.MapValue{}
	resource.Status.Id = testTsigKeyID
	resource.Status.OsokStatus.Ocid = shared.OCID(testTsigKeyID)
	getCalls := 0
	updateCalls := 0
	var updateRequest dnssdk.UpdateTsigKeyRequest

	client := testTsigKeyClient(&fakeTsigKeyOCIClient{
		getTsigKeyFn: func(context.Context, dnssdk.GetTsigKeyRequest) (dnssdk.GetTsigKeyResponse, error) {
			getCalls++
			current := makeSDKTsigKey(testTsigKeyID, testCompartmentID, resource.Spec.Name, resource.Spec.Algorithm, resource.Spec.Secret, dnssdk.TsigKeyLifecycleStateActive)
			if getCalls > 1 {
				current.FreeformTags = map[string]string{}
				current.DefinedTags = map[string]map[string]interface{}{}
			}
			return dnssdk.GetTsigKeyResponse{TsigKey: current}, nil
		},
		updateTsigKeyFn: func(_ context.Context, req dnssdk.UpdateTsigKeyRequest) (dnssdk.UpdateTsigKeyResponse, error) {
			updateCalls++
			updateRequest = req
			updated := makeSDKTsigKey(testTsigKeyID, testCompartmentID, resource.Spec.Name, resource.Spec.Algorithm, resource.Spec.Secret, dnssdk.TsigKeyLifecycleStateUpdating)
			updated.FreeformTags = map[string]string{}
			updated.DefinedTags = map[string]map[string]interface{}{}
			return dnssdk.UpdateTsigKeyResponse{
				TsigKey:      updated,
				OpcRequestId: common.String("opc-update-1"),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate().IsSuccessful = false, want true")
	}
	if updateCalls != 1 {
		t.Fatalf("update calls = %d, want 1", updateCalls)
	}
	requireStringPtr(t, "update tsigKeyId", updateRequest.TsigKeyId, testTsigKeyID)
	if len(updateRequest.FreeformTags) != 0 || updateRequest.FreeformTags == nil {
		t.Fatalf("update freeformTags = %#v, want explicit empty map", updateRequest.FreeformTags)
	}
	if len(updateRequest.DefinedTags) != 0 || updateRequest.DefinedTags == nil {
		t.Fatalf("update definedTags = %#v, want explicit empty map", updateRequest.DefinedTags)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update-1", got)
	}
}

func TestTsigKeyServiceClientRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	resource := makeTsigKeyResource()
	resource.Status.Id = testTsigKeyID
	resource.Status.OsokStatus.Ocid = shared.OCID(testTsigKeyID)
	updateCalls := 0

	client := testTsigKeyClient(&fakeTsigKeyOCIClient{
		getTsigKeyFn: func(context.Context, dnssdk.GetTsigKeyRequest) (dnssdk.GetTsigKeyResponse, error) {
			current := makeSDKTsigKey(testTsigKeyID, testCompartmentID, resource.Spec.Name, resource.Spec.Algorithm, "different-secret", dnssdk.TsigKeyLifecycleStateActive)
			return dnssdk.GetTsigKeyResponse{TsigKey: current}, nil
		},
		updateTsigKeyFn: func(context.Context, dnssdk.UpdateTsigKeyRequest) (dnssdk.UpdateTsigKeyResponse, error) {
			updateCalls++
			return dnssdk.UpdateTsigKeyResponse{}, nil
		},
	})

	_, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift error")
	}
	if !strings.Contains(err.Error(), "secret") {
		t.Fatalf("CreateOrUpdate() error = %q, want secret drift", err.Error())
	}
	if strings.Contains(err.Error(), testTsigSecret) || strings.Contains(err.Error(), "different-secret") {
		t.Fatalf("CreateOrUpdate() error leaked secret value: %q", err.Error())
	}
	if updateCalls != 0 {
		t.Fatalf("update calls = %d, want 0", updateCalls)
	}
}

func TestTsigKeyServiceClientDeleteRetainsFinalizerUntilConfirmed(t *testing.T) {
	resource := makeTsigKeyResource()
	resource.Status.Id = testTsigKeyID
	resource.Status.OsokStatus.Ocid = shared.OCID(testTsigKeyID)
	client, lifecycle := testTsigKeyDeleteLifecycleClient(t, resource)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() first error = %v", err)
	}
	assertTsigKeyDeleteInProgress(t, resource, deleted)

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() second error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() second deleted = false, want true after OCI is DELETED")
	}
	if lifecycle.deleteCalls != 1 {
		t.Fatalf("delete calls = %d, want 1", lifecycle.deleteCalls)
	}
}

type tsigKeyDeleteLifecycle struct {
	t           *testing.T
	resource    *dnsv1beta1.TsigKey
	getCalls    int
	deleteCalls int
}

func testTsigKeyDeleteLifecycleClient(
	t *testing.T,
	resource *dnsv1beta1.TsigKey,
) (TsigKeyServiceClient, *tsigKeyDeleteLifecycle) {
	t.Helper()
	lifecycle := &tsigKeyDeleteLifecycle{t: t, resource: resource}
	client := testTsigKeyClient(&fakeTsigKeyOCIClient{
		getTsigKeyFn:    lifecycle.getTsigKey,
		deleteTsigKeyFn: lifecycle.deleteTsigKey,
	})
	return client, lifecycle
}

func (l *tsigKeyDeleteLifecycle) getTsigKey(context.Context, dnssdk.GetTsigKeyRequest) (dnssdk.GetTsigKeyResponse, error) {
	l.getCalls++
	return dnssdk.GetTsigKeyResponse{
		TsigKey: makeSDKTsigKey(
			testTsigKeyID,
			testCompartmentID,
			l.resource.Spec.Name,
			l.resource.Spec.Algorithm,
			l.resource.Spec.Secret,
			tsigKeyDeleteLifecycleState(l.getCalls),
		),
	}, nil
}

func (l *tsigKeyDeleteLifecycle) deleteTsigKey(
	_ context.Context,
	req dnssdk.DeleteTsigKeyRequest,
) (dnssdk.DeleteTsigKeyResponse, error) {
	l.deleteCalls++
	requireStringPtr(l.t, "delete tsigKeyId", req.TsigKeyId, testTsigKeyID)
	return dnssdk.DeleteTsigKeyResponse{OpcRequestId: common.String("opc-delete-1")}, nil
}

func tsigKeyDeleteLifecycleState(call int) dnssdk.TsigKeyLifecycleStateEnum {
	states := []dnssdk.TsigKeyLifecycleStateEnum{
		dnssdk.TsigKeyLifecycleStateActive,
		dnssdk.TsigKeyLifecycleStateActive,
		dnssdk.TsigKeyLifecycleStateDeleting,
		dnssdk.TsigKeyLifecycleStateDeleted,
	}
	if call <= 0 || call > len(states) {
		return dnssdk.TsigKeyLifecycleStateDeleted
	}
	return states[call-1]
}

func assertTsigKeyDeleteInProgress(t *testing.T, resource *dnsv1beta1.TsigKey, deleted bool) {
	t.Helper()
	if deleted {
		t.Fatal("Delete() first deleted = true, want false while OCI is DELETING")
	}
	if resource.Status.OsokStatus.Async.Current == nil || resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status async current = %#v, want delete phase", resource.Status.OsokStatus.Async.Current)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete-1", got)
	}
}

func TestTsigKeyServiceClientDeleteTreatsAuthShapedNotFoundAsError(t *testing.T) {
	resource := makeTsigKeyResource()
	resource.Status.Id = testTsigKeyID
	resource.Status.OsokStatus.Ocid = shared.OCID(testTsigKeyID)
	deleteCalls := 0

	client := testTsigKeyClient(&fakeTsigKeyOCIClient{
		getTsigKeyFn: func(context.Context, dnssdk.GetTsigKeyRequest) (dnssdk.GetTsigKeyResponse, error) {
			return dnssdk.GetTsigKeyResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
		deleteTsigKeyFn: func(context.Context, dnssdk.DeleteTsigKeyRequest) (dnssdk.DeleteTsigKeyResponse, error) {
			deleteCalls++
			return dnssdk.DeleteTsigKeyResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped not-found error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %q, want ambiguous auth-shaped 404", err.Error())
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
	if deleteCalls != 0 {
		t.Fatalf("delete calls = %d, want 0", deleteCalls)
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

func assertTsigKeyStringSliceEqual(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
}

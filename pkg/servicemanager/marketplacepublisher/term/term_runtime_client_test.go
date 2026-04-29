/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package term

import (
	"context"
	"crypto/rsa"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	marketplacepublishersdk "github.com/oracle/oci-go-sdk/v65/marketplacepublisher"
	marketplacepublisherv1beta1 "github.com/oracle/oci-service-operator/api/marketplacepublisher/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testTermID            = "ocid1.term.oc1..example"
	testTermCompartmentID = "ocid1.compartment.oc1..term"
	testTermName          = "term-alpha"
)

type fakeTermOCIClient struct {
	createFn func(context.Context, marketplacepublishersdk.CreateTermRequest) (marketplacepublishersdk.CreateTermResponse, error)
	getFn    func(context.Context, marketplacepublishersdk.GetTermRequest) (marketplacepublishersdk.GetTermResponse, error)
	listFn   func(context.Context, marketplacepublishersdk.ListTermsRequest) (marketplacepublishersdk.ListTermsResponse, error)
	updateFn func(context.Context, marketplacepublishersdk.UpdateTermRequest) (marketplacepublishersdk.UpdateTermResponse, error)
	deleteFn func(context.Context, marketplacepublishersdk.DeleteTermRequest) (marketplacepublishersdk.DeleteTermResponse, error)
}

type erroringTermConfigProvider struct {
	calls int
}

func (p *erroringTermConfigProvider) PrivateRSAKey() (*rsa.PrivateKey, error) {
	p.calls++
	return nil, errors.New("term provider invalid")
}

func (p *erroringTermConfigProvider) KeyID() (string, error) {
	p.calls++
	return "", errors.New("term provider invalid")
}

func (p *erroringTermConfigProvider) TenancyOCID() (string, error) {
	p.calls++
	return "", errors.New("term provider invalid")
}

func (p *erroringTermConfigProvider) UserOCID() (string, error) {
	p.calls++
	return "", errors.New("term provider invalid")
}

func (p *erroringTermConfigProvider) KeyFingerprint() (string, error) {
	p.calls++
	return "", errors.New("term provider invalid")
}

func (p *erroringTermConfigProvider) Region() (string, error) {
	p.calls++
	return "", errors.New("term provider invalid")
}

func (p *erroringTermConfigProvider) AuthType() (common.AuthConfig, error) {
	return common.AuthConfig{}, nil
}

func (f *fakeTermOCIClient) CreateTerm(ctx context.Context, req marketplacepublishersdk.CreateTermRequest) (marketplacepublishersdk.CreateTermResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return marketplacepublishersdk.CreateTermResponse{}, nil
}

func (f *fakeTermOCIClient) GetTerm(ctx context.Context, req marketplacepublishersdk.GetTermRequest) (marketplacepublishersdk.GetTermResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return marketplacepublishersdk.GetTermResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "term is missing")
}

func (f *fakeTermOCIClient) ListTerms(ctx context.Context, req marketplacepublishersdk.ListTermsRequest) (marketplacepublishersdk.ListTermsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return marketplacepublishersdk.ListTermsResponse{}, nil
}

func (f *fakeTermOCIClient) UpdateTerm(ctx context.Context, req marketplacepublishersdk.UpdateTermRequest) (marketplacepublishersdk.UpdateTermResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return marketplacepublishersdk.UpdateTermResponse{}, nil
}

func (f *fakeTermOCIClient) DeleteTerm(ctx context.Context, req marketplacepublishersdk.DeleteTermRequest) (marketplacepublishersdk.DeleteTermResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return marketplacepublishersdk.DeleteTermResponse{}, nil
}

func TestTermRuntimeSemantics(t *testing.T) {
	t.Parallel()

	got := newTermRuntimeSemantics()
	if got == nil {
		t.Fatal("newTermRuntimeSemantics() = nil")
	}
	if got.Async == nil || got.Async.Strategy != "lifecycle" || got.Async.Runtime != "generatedruntime" {
		t.Fatalf("Async = %#v, want generatedruntime lifecycle", got.Async)
	}
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "best-effort" || got.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("delete semantics = %#v followUp=%#v, want best-effort confirm-delete", got.Delete, got.DeleteFollowUp)
	}
	assertTermStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE", "INACTIVE"})
	assertTermStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"compartmentId", "name", "id"})
	assertTermStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"freeformTags", "definedTags"})
	assertTermStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"compartmentId", "name"})
}

func TestTermCreateOrUpdateBindsExistingTermByPagedList(t *testing.T) {
	t.Parallel()

	resource := makeTermResource()
	createCalled := false
	updateCalled := false
	getCalls := 0
	listCalls := 0

	client := newTestTermClient(&fakeTermOCIClient{
		listFn: func(_ context.Context, req marketplacepublishersdk.ListTermsRequest) (marketplacepublishersdk.ListTermsResponse, error) {
			listCalls++
			requireTermStringPtr(t, "ListTermsRequest.CompartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireTermStringPtr(t, "ListTermsRequest.Name", req.Name, resource.Spec.Name)
			if listCalls == 1 {
				if req.Page != nil {
					t.Fatalf("first ListTermsRequest.Page = %q, want nil", *req.Page)
				}
				return marketplacepublishersdk.ListTermsResponse{
					TermCollection: marketplacepublishersdk.TermCollection{
						Items: []marketplacepublishersdk.TermSummary{
							makeSDKTermSummary("ocid1.term.oc1..other", marketplacepublisherv1beta1.TermSpec{
								CompartmentId: resource.Spec.CompartmentId,
								Name:          "other-term",
							}, marketplacepublishersdk.TermLifecycleStateActive),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			requireTermStringPtr(t, "second ListTermsRequest.Page", req.Page, "page-2")
			return marketplacepublishersdk.ListTermsResponse{
				TermCollection: marketplacepublishersdk.TermCollection{
					Items: []marketplacepublishersdk.TermSummary{
						makeSDKTermSummary(testTermID, resource.Spec, marketplacepublishersdk.TermLifecycleStateActive),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, req marketplacepublishersdk.GetTermRequest) (marketplacepublishersdk.GetTermResponse, error) {
			getCalls++
			requireTermStringPtr(t, "GetTermRequest.TermId", req.TermId, testTermID)
			return marketplacepublishersdk.GetTermResponse{
				Term: makeSDKTerm(testTermID, resource.Spec, marketplacepublishersdk.TermLifecycleStateActive),
			}, nil
		},
		createFn: func(context.Context, marketplacepublishersdk.CreateTermRequest) (marketplacepublishersdk.CreateTermResponse, error) {
			createCalled = true
			return marketplacepublishersdk.CreateTermResponse{}, nil
		},
		updateFn: func(context.Context, marketplacepublishersdk.UpdateTermRequest) (marketplacepublishersdk.UpdateTermResponse, error) {
			updateCalled = true
			return marketplacepublishersdk.UpdateTermResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeTermRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	requireTermSuccessfulResponse(t, "CreateOrUpdate", response)
	if createCalled {
		t.Fatal("CreateTerm() called for existing term")
	}
	if updateCalled {
		t.Fatal("UpdateTerm() called for matching term")
	}
	if listCalls != 2 {
		t.Fatalf("ListTerms() calls = %d, want 2 paginated calls", listCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetTerm() calls = %d, want 1", getCalls)
	}
	requireTermActiveStatus(t, resource, testTermID)
}

func TestTermCreateRecordsRetryTokenRequestIDAndLifecycle(t *testing.T) {
	t.Parallel()

	resource := makeTermResource()
	var createRequest marketplacepublishersdk.CreateTermRequest
	listCalls := 0
	getCalls := 0

	client := newTestTermClient(&fakeTermOCIClient{
		listFn: func(_ context.Context, req marketplacepublishersdk.ListTermsRequest) (marketplacepublishersdk.ListTermsResponse, error) {
			listCalls++
			requireTermStringPtr(t, "ListTermsRequest.CompartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireTermStringPtr(t, "ListTermsRequest.Name", req.Name, resource.Spec.Name)
			return marketplacepublishersdk.ListTermsResponse{}, nil
		},
		createFn: func(_ context.Context, req marketplacepublishersdk.CreateTermRequest) (marketplacepublishersdk.CreateTermResponse, error) {
			createRequest = req
			return marketplacepublishersdk.CreateTermResponse{
				Term:         makeSDKTerm(testTermID, resource.Spec, marketplacepublishersdk.TermLifecycleStateActive),
				OpcRequestId: common.String("opc-create-1"),
			}, nil
		},
		getFn: func(_ context.Context, req marketplacepublishersdk.GetTermRequest) (marketplacepublishersdk.GetTermResponse, error) {
			getCalls++
			requireTermStringPtr(t, "GetTermRequest.TermId", req.TermId, testTermID)
			return marketplacepublishersdk.GetTermResponse{
				Term: makeSDKTerm(testTermID, resource.Spec, marketplacepublishersdk.TermLifecycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeTermRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	requireTermSuccessfulResponse(t, "CreateOrUpdate", response)
	if listCalls != 1 {
		t.Fatalf("ListTerms() calls = %d, want 1", listCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetTerm() calls = %d, want 1 read-after-create", getCalls)
	}
	requireTermStringPtr(t, "create compartmentId", createRequest.CompartmentId, resource.Spec.CompartmentId)
	requireTermStringPtr(t, "create name", createRequest.Name, resource.Spec.Name)
	requireTermStringPtr(t, "create retry token", createRequest.OpcRetryToken, string(resource.UID))
	if !reflect.DeepEqual(createRequest.FreeformTags, resource.Spec.FreeformTags) {
		t.Fatalf("create freeformTags = %#v, want %#v", createRequest.FreeformTags, resource.Spec.FreeformTags)
	}
	if !reflect.DeepEqual(createRequest.DefinedTags, termDefinedTags(resource.Spec.DefinedTags)) {
		t.Fatalf("create definedTags = %#v, want %#v", createRequest.DefinedTags, termDefinedTags(resource.Spec.DefinedTags))
	}
	requireTermActiveStatus(t, resource, testTermID)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-1", got)
	}
}

func TestTermNoOpDoesNotUpdate(t *testing.T) {
	t.Parallel()

	resource := makeTermResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testTermID)
	updateCalled := false

	client := newTestTermClient(&fakeTermOCIClient{
		getFn: func(_ context.Context, req marketplacepublishersdk.GetTermRequest) (marketplacepublishersdk.GetTermResponse, error) {
			requireTermStringPtr(t, "GetTermRequest.TermId", req.TermId, testTermID)
			return marketplacepublishersdk.GetTermResponse{
				Term: makeSDKTerm(testTermID, resource.Spec, marketplacepublishersdk.TermLifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, marketplacepublishersdk.UpdateTermRequest) (marketplacepublishersdk.UpdateTermResponse, error) {
			updateCalled = true
			return marketplacepublishersdk.UpdateTermResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeTermRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	requireTermSuccessfulResponse(t, "CreateOrUpdate", response)
	if updateCalled {
		t.Fatal("UpdateTerm() called for matching term")
	}
	requireTermActiveStatus(t, resource, testTermID)
}

func TestTermMutableTagsUpdate(t *testing.T) {
	t.Parallel()

	resource := makeTermResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testTermID)
	currentSpec := resource.Spec
	currentSpec.FreeformTags = map[string]string{"env": "old"}
	currentSpec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "old"}}
	getCalls := 0
	var updateRequest marketplacepublishersdk.UpdateTermRequest

	client := newTestTermClient(&fakeTermOCIClient{
		getFn: func(_ context.Context, req marketplacepublishersdk.GetTermRequest) (marketplacepublishersdk.GetTermResponse, error) {
			getCalls++
			requireTermStringPtr(t, "GetTermRequest.TermId", req.TermId, testTermID)
			if getCalls == 1 {
				return marketplacepublishersdk.GetTermResponse{
					Term: makeSDKTerm(testTermID, currentSpec, marketplacepublishersdk.TermLifecycleStateActive),
				}, nil
			}
			return marketplacepublishersdk.GetTermResponse{
				Term: makeSDKTerm(testTermID, resource.Spec, marketplacepublishersdk.TermLifecycleStateActive),
			}, nil
		},
		updateFn: func(_ context.Context, req marketplacepublishersdk.UpdateTermRequest) (marketplacepublishersdk.UpdateTermResponse, error) {
			updateRequest = req
			return marketplacepublishersdk.UpdateTermResponse{
				Term:         makeSDKTerm(testTermID, resource.Spec, marketplacepublishersdk.TermLifecycleStateActive),
				OpcRequestId: common.String("opc-update-1"),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeTermRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	requireTermSuccessfulResponse(t, "CreateOrUpdate", response)
	if getCalls != 2 {
		t.Fatalf("GetTerm() calls = %d, want initial and read-after-update", getCalls)
	}
	requireTermStringPtr(t, "UpdateTermRequest.TermId", updateRequest.TermId, testTermID)
	if !reflect.DeepEqual(updateRequest.FreeformTags, resource.Spec.FreeformTags) {
		t.Fatalf("update freeformTags = %#v, want %#v", updateRequest.FreeformTags, resource.Spec.FreeformTags)
	}
	if !reflect.DeepEqual(updateRequest.DefinedTags, termDefinedTags(resource.Spec.DefinedTags)) {
		t.Fatalf("update definedTags = %#v, want %#v", updateRequest.DefinedTags, termDefinedTags(resource.Spec.DefinedTags))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update-1", got)
	}
	requireTermActiveStatus(t, resource, testTermID)
}

func TestTermRejectsNameDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := makeTermResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testTermID)
	currentSpec := resource.Spec
	currentSpec.Name = "old-term"
	updateCalled := false

	client := newTestTermClient(&fakeTermOCIClient{
		getFn: func(_ context.Context, req marketplacepublishersdk.GetTermRequest) (marketplacepublishersdk.GetTermResponse, error) {
			requireTermStringPtr(t, "GetTermRequest.TermId", req.TermId, testTermID)
			return marketplacepublishersdk.GetTermResponse{
				Term: makeSDKTerm(testTermID, currentSpec, marketplacepublishersdk.TermLifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, marketplacepublishersdk.UpdateTermRequest) (marketplacepublishersdk.UpdateTermResponse, error) {
			updateCalled = true
			return marketplacepublishersdk.UpdateTermResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeTermRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want name drift rejection")
	}
	if !strings.Contains(err.Error(), "require replacement when name changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want replacement drift", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response successful = true, want false")
	}
	if updateCalled {
		t.Fatal("UpdateTerm() called after immutable name drift")
	}
	requireLastTermCondition(t, resource, shared.Failed)
}

func TestTermRejectsMissingRequiredIdentityBeforeUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mutate    func(*marketplacepublisherv1beta1.Term)
		wantField string
	}{
		{
			name: "compartmentId cleared",
			mutate: func(resource *marketplacepublisherv1beta1.Term) {
				resource.Spec.CompartmentId = ""
			},
			wantField: "compartmentId",
		},
		{
			name: "name blanked",
			mutate: func(resource *marketplacepublisherv1beta1.Term) {
				resource.Spec.Name = "  "
			},
			wantField: "name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resource := makeTermResource()
			resource.Status.OsokStatus.Ocid = shared.OCID(testTermID)
			currentSpec := resource.Spec
			tt.mutate(resource)
			updateCalled := false

			client := newTestTermClient(&fakeTermOCIClient{
				getFn: func(_ context.Context, req marketplacepublishersdk.GetTermRequest) (marketplacepublishersdk.GetTermResponse, error) {
					requireTermStringPtr(t, "GetTermRequest.TermId", req.TermId, testTermID)
					return marketplacepublishersdk.GetTermResponse{
						Term: makeSDKTerm(testTermID, currentSpec, marketplacepublishersdk.TermLifecycleStateActive),
					}, nil
				},
				updateFn: func(context.Context, marketplacepublishersdk.UpdateTermRequest) (marketplacepublishersdk.UpdateTermResponse, error) {
					updateCalled = true
					return marketplacepublishersdk.UpdateTermResponse{}, nil
				},
			})

			response, err := client.CreateOrUpdate(context.Background(), resource, makeTermRequest(resource))
			if err == nil {
				t.Fatalf("CreateOrUpdate() error = nil, want missing %s rejection", tt.wantField)
			}
			if !strings.Contains(err.Error(), "term spec is missing required field(s): "+tt.wantField) {
				t.Fatalf("CreateOrUpdate() error = %v, want missing %s rejection", err, tt.wantField)
			}
			if response.IsSuccessful {
				t.Fatalf("CreateOrUpdate() response successful = true, want false")
			}
			if updateCalled {
				t.Fatal("UpdateTerm() called after required identity field was removed")
			}
			requireLastTermCondition(t, resource, shared.Failed)
		})
	}
}

func TestTermDeleteConfirmsNotFoundAfterDelete(t *testing.T) {
	t.Parallel()

	resource := makeTermResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testTermID)
	getCalls := 0
	deleteCalls := 0

	client := newTestTermClient(&fakeTermOCIClient{
		getFn: func(_ context.Context, req marketplacepublishersdk.GetTermRequest) (marketplacepublishersdk.GetTermResponse, error) {
			getCalls++
			requireTermStringPtr(t, "GetTermRequest.TermId", req.TermId, testTermID)
			if getCalls < 3 {
				return marketplacepublishersdk.GetTermResponse{
					Term: makeSDKTerm(testTermID, resource.Spec, marketplacepublishersdk.TermLifecycleStateActive),
				}, nil
			}
			return marketplacepublishersdk.GetTermResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "term is deleted")
		},
		deleteFn: func(_ context.Context, req marketplacepublishersdk.DeleteTermRequest) (marketplacepublishersdk.DeleteTermResponse, error) {
			deleteCalls++
			requireTermStringPtr(t, "DeleteTermRequest.TermId", req.TermId, testTermID)
			return marketplacepublishersdk.DeleteTermResponse{
				OpcRequestId: common.String("opc-delete-1"),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after not-found confirmation")
	}
	if getCalls != 3 {
		t.Fatalf("GetTerm() calls = %d, want wrapper, pre-delete, and post-delete confirmation", getCalls)
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteTerm() calls = %d, want 1", deleteCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete-1", got)
	}
}

func TestTermDeleteRetainsFinalizerWhenReadbackStillExists(t *testing.T) {
	t.Parallel()

	resource := makeTermResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testTermID)
	getCalls := 0
	deleteCalls := 0

	client := newTestTermClient(&fakeTermOCIClient{
		getFn: func(_ context.Context, req marketplacepublishersdk.GetTermRequest) (marketplacepublishersdk.GetTermResponse, error) {
			getCalls++
			requireTermStringPtr(t, "GetTermRequest.TermId", req.TermId, testTermID)
			return marketplacepublishersdk.GetTermResponse{
				Term: makeSDKTerm(testTermID, resource.Spec, marketplacepublishersdk.TermLifecycleStateActive),
			}, nil
		},
		deleteFn: func(_ context.Context, req marketplacepublishersdk.DeleteTermRequest) (marketplacepublishersdk.DeleteTermResponse, error) {
			deleteCalls++
			requireTermStringPtr(t, "DeleteTermRequest.TermId", req.TermId, testTermID)
			return marketplacepublishersdk.DeleteTermResponse{
				OpcRequestId: common.String("opc-delete-1"),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while readback still exists")
	}
	if getCalls != 3 {
		t.Fatalf("GetTerm() calls = %d, want wrapper, pre-delete, and post-delete confirmation", getCalls)
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteTerm() calls = %d, want 1", deleteCalls)
	}
	requireLastTermCondition(t, resource, shared.Terminating)
	if resource.Status.OsokStatus.Async.Current == nil || resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.async.current = %#v, want lifecycle delete pending", resource.Status.OsokStatus.Async.Current)
	}
}

func TestTermDeleteConfirmsPreDeleteNotFound(t *testing.T) {
	t.Parallel()

	resource := makeTermResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testTermID)
	getCalls := 0
	deleteCalled := false

	client := newTestTermClient(&fakeTermOCIClient{
		getFn: func(_ context.Context, req marketplacepublishersdk.GetTermRequest) (marketplacepublishersdk.GetTermResponse, error) {
			getCalls++
			requireTermStringPtr(t, "GetTermRequest.TermId", req.TermId, testTermID)
			return marketplacepublishersdk.GetTermResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "term is deleted")
		},
		deleteFn: func(context.Context, marketplacepublishersdk.DeleteTermRequest) (marketplacepublishersdk.DeleteTermResponse, error) {
			deleteCalled = true
			return marketplacepublishersdk.DeleteTermResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after unambiguous pre-delete not found")
	}
	if deleteCalled {
		t.Fatal("DeleteTerm() called after unambiguous pre-delete not found")
	}
	if getCalls != 2 {
		t.Fatalf("GetTerm() calls = %d, want wrapper and generatedruntime confirmation", getCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
	requireLastTermCondition(t, resource, shared.Terminating)
}

func TestTermDeleteRejectsAuthShapedPreDeleteRead(t *testing.T) {
	t.Parallel()

	resource := makeTermResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testTermID)
	deleteCalled := false

	client := newTestTermClient(&fakeTermOCIClient{
		getFn: func(_ context.Context, req marketplacepublishersdk.GetTermRequest) (marketplacepublishersdk.GetTermResponse, error) {
			requireTermStringPtr(t, "GetTermRequest.TermId", req.TermId, testTermID)
			return marketplacepublishersdk.GetTermResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
		deleteFn: func(context.Context, marketplacepublishersdk.DeleteTermRequest) (marketplacepublishersdk.DeleteTermResponse, error) {
			deleteCalled = true
			return marketplacepublishersdk.DeleteTermResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous NotAuthorizedOrNotFound")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous NotAuthorizedOrNotFound", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if deleteCalled {
		t.Fatal("DeleteTerm() called after auth-shaped pre-delete read")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want error opc request id", got)
	}
}

func TestTermDeleteRejectsAuthShapedPostDeleteRead(t *testing.T) {
	t.Parallel()

	resource := makeTermResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testTermID)
	getCalls := 0
	deleteCalls := 0

	client := newTestTermClient(&fakeTermOCIClient{
		getFn: func(_ context.Context, req marketplacepublishersdk.GetTermRequest) (marketplacepublishersdk.GetTermResponse, error) {
			getCalls++
			requireTermStringPtr(t, "GetTermRequest.TermId", req.TermId, testTermID)
			if getCalls < 3 {
				return marketplacepublishersdk.GetTermResponse{
					Term: makeSDKTerm(testTermID, resource.Spec, marketplacepublishersdk.TermLifecycleStateActive),
				}, nil
			}
			return marketplacepublishersdk.GetTermResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
		deleteFn: func(_ context.Context, req marketplacepublishersdk.DeleteTermRequest) (marketplacepublishersdk.DeleteTermResponse, error) {
			deleteCalls++
			requireTermStringPtr(t, "DeleteTermRequest.TermId", req.TermId, testTermID)
			return marketplacepublishersdk.DeleteTermResponse{
				OpcRequestId: common.String("opc-delete-1"),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous NotAuthorizedOrNotFound")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous NotAuthorizedOrNotFound", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteTerm() calls = %d, want 1", deleteCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want post-delete error opc request id", got)
	}
}

func TestTermDeletePreservesGeneratedOCIInitErrorBeforePreflight(t *testing.T) {
	resource := makeTermResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testTermID)
	provider := &erroringTermConfigProvider{}
	manager := &TermServiceManager{
		Provider: provider,
		Log:      loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
	}
	client := newTermServiceClient(manager)
	callsAfterInit := provider.calls

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want OCI client initialization error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if !strings.Contains(err.Error(), "initialize Term OCI client") {
		t.Fatalf("Delete() error = %v, want Term OCI client initialization failure", err)
	}
	if !strings.Contains(err.Error(), "term provider invalid") {
		t.Fatalf("Delete() error = %v, want provider failure detail", err)
	}
	if provider.calls != callsAfterInit {
		t.Fatalf("provider calls after Delete() = %d, want %d; delete preflight should not run before InitError", provider.calls, callsAfterInit)
	}
}

func newTestTermClient(client termOCIClient) TermServiceClient {
	return newTermServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}, client)
}

func makeTermResource() *marketplacepublisherv1beta1.Term {
	return &marketplacepublisherv1beta1.Term{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testTermName,
			Namespace: "default",
			UID:       types.UID("term-uid"),
		},
		Spec: marketplacepublisherv1beta1.TermSpec{
			CompartmentId: testTermCompartmentID,
			Name:          testTermName,
			FreeformTags: map[string]string{
				"env": "test",
			},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {
					"CostCenter": "42",
				},
			},
		},
	}
}

func makeTermRequest(resource *marketplacepublisherv1beta1.Term) ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Namespace: resource.Namespace, Name: resource.Name}}
}

func makeSDKTerm(
	id string,
	spec marketplacepublisherv1beta1.TermSpec,
	state marketplacepublishersdk.TermLifecycleStateEnum,
) marketplacepublishersdk.Term {
	return marketplacepublishersdk.Term{
		Id:             common.String(id),
		CompartmentId:  common.String(spec.CompartmentId),
		Name:           common.String(spec.Name),
		Author:         marketplacepublishersdk.TermAuthorPartner,
		PublisherId:    common.String("publisher-1"),
		LifecycleState: state,
		FreeformTags:   cloneTermStringMap(spec.FreeformTags),
		DefinedTags:    termDefinedTags(spec.DefinedTags),
	}
}

func makeSDKTermSummary(
	id string,
	spec marketplacepublisherv1beta1.TermSpec,
	state marketplacepublishersdk.TermLifecycleStateEnum,
) marketplacepublishersdk.TermSummary {
	return marketplacepublishersdk.TermSummary{
		Id:             common.String(id),
		CompartmentId:  common.String(spec.CompartmentId),
		Name:           common.String(spec.Name),
		Author:         marketplacepublishersdk.TermAuthorPartner,
		LifecycleState: state,
		FreeformTags:   cloneTermStringMap(spec.FreeformTags),
		DefinedTags:    termDefinedTags(spec.DefinedTags),
	}
}

func requireTermSuccessfulResponse(t *testing.T, operation string, response servicemanager.OSOKResponse) {
	t.Helper()
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("%s response = %#v, want successful non-requeue", operation, response)
	}
}

func requireTermActiveStatus(t *testing.T, resource *marketplacepublisherv1beta1.Term, id string) {
	t.Helper()
	if got := string(resource.Status.OsokStatus.Ocid); got != id {
		t.Fatalf("status.status.ocid = %q, want %q", got, id)
	}
	if got := resource.Status.Id; got != id {
		t.Fatalf("status.id = %q, want %q", got, id)
	}
	requireLastTermCondition(t, resource, shared.Active)
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil", resource.Status.OsokStatus.Async.Current)
	}
}

func requireLastTermCondition(t *testing.T, resource *marketplacepublisherv1beta1.Term, want shared.OSOKConditionType) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status conditions are empty, want trailing %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last condition = %q, want %q", got, want)
	}
}

func requireTermStringPtr(t *testing.T, label string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", label, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", label, *got, want)
	}
}

func assertTermStringSliceEqual(t *testing.T, label string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", label, got, want)
	}
}

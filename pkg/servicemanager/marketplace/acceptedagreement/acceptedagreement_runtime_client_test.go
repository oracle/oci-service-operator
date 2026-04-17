/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package acceptedagreement

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	marketplacesdk "github.com/oracle/oci-go-sdk/v65/marketplace"
	marketplacev1beta1 "github.com/oracle/oci-service-operator/api/marketplace/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testAcceptedAgreementID      = "ocid1.acceptedagreement.oc1..exampleuniqueID"
	testAcceptedAgreementDisplay = "acceptedagreement-sample"
	testCompartmentID            = "ocid1.compartment.oc1..exampleuniqueID"
	testListingID                = "ocid1.listing.oc1..exampleuniqueID"
	testPackageVersion           = "1.0.0"
	testAgreementID              = "ocid1.agreement.oc1..exampleuniqueID"
	testSignature                = "signed-agreement-v1"
)

type fakeAcceptedAgreementOCIClient struct {
	create func(context.Context, marketplacesdk.CreateAcceptedAgreementRequest) (marketplacesdk.CreateAcceptedAgreementResponse, error)
	get    func(context.Context, marketplacesdk.GetAcceptedAgreementRequest) (marketplacesdk.GetAcceptedAgreementResponse, error)
	list   func(context.Context, marketplacesdk.ListAcceptedAgreementsRequest) (marketplacesdk.ListAcceptedAgreementsResponse, error)
	update func(context.Context, marketplacesdk.UpdateAcceptedAgreementRequest) (marketplacesdk.UpdateAcceptedAgreementResponse, error)
	delete func(context.Context, marketplacesdk.DeleteAcceptedAgreementRequest) (marketplacesdk.DeleteAcceptedAgreementResponse, error)

	createRequests []marketplacesdk.CreateAcceptedAgreementRequest
	getRequests    []marketplacesdk.GetAcceptedAgreementRequest
	listRequests   []marketplacesdk.ListAcceptedAgreementsRequest
	updateRequests []marketplacesdk.UpdateAcceptedAgreementRequest
	deleteRequests []marketplacesdk.DeleteAcceptedAgreementRequest
}

func (f *fakeAcceptedAgreementOCIClient) CreateAcceptedAgreement(ctx context.Context, request marketplacesdk.CreateAcceptedAgreementRequest) (marketplacesdk.CreateAcceptedAgreementResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.create == nil {
		return marketplacesdk.CreateAcceptedAgreementResponse{}, nil
	}
	return f.create(ctx, request)
}

func (f *fakeAcceptedAgreementOCIClient) GetAcceptedAgreement(ctx context.Context, request marketplacesdk.GetAcceptedAgreementRequest) (marketplacesdk.GetAcceptedAgreementResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.get == nil {
		return marketplacesdk.GetAcceptedAgreementResponse{}, nil
	}
	return f.get(ctx, request)
}

func (f *fakeAcceptedAgreementOCIClient) ListAcceptedAgreements(ctx context.Context, request marketplacesdk.ListAcceptedAgreementsRequest) (marketplacesdk.ListAcceptedAgreementsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.list == nil {
		return marketplacesdk.ListAcceptedAgreementsResponse{}, nil
	}
	return f.list(ctx, request)
}

func (f *fakeAcceptedAgreementOCIClient) UpdateAcceptedAgreement(ctx context.Context, request marketplacesdk.UpdateAcceptedAgreementRequest) (marketplacesdk.UpdateAcceptedAgreementResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.update == nil {
		return marketplacesdk.UpdateAcceptedAgreementResponse{}, nil
	}
	return f.update(ctx, request)
}

func (f *fakeAcceptedAgreementOCIClient) DeleteAcceptedAgreement(ctx context.Context, request marketplacesdk.DeleteAcceptedAgreementRequest) (marketplacesdk.DeleteAcceptedAgreementResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.delete == nil {
		return marketplacesdk.DeleteAcceptedAgreementResponse{}, nil
	}
	return f.delete(ctx, request)
}

func TestAcceptedAgreementRuntimeClientCreateMarksActiveAndMirrorsSignature(t *testing.T) {
	resource := testAcceptedAgreementResource()
	client := &fakeAcceptedAgreementOCIClient{
		list: func(context.Context, marketplacesdk.ListAcceptedAgreementsRequest) (marketplacesdk.ListAcceptedAgreementsResponse, error) {
			return marketplacesdk.ListAcceptedAgreementsResponse{}, nil
		},
		get: func(_ context.Context, request marketplacesdk.GetAcceptedAgreementRequest) (marketplacesdk.GetAcceptedAgreementResponse, error) {
			if request.AcceptedAgreementId == nil || *request.AcceptedAgreementId != testAcceptedAgreementID {
				t.Fatalf("GetAcceptedAgreementRequest.AcceptedAgreementId = %v, want %q", request.AcceptedAgreementId, testAcceptedAgreementID)
			}
			return marketplacesdk.GetAcceptedAgreementResponse{
				AcceptedAgreement: sdkAcceptedAgreement(testAcceptedAgreementID, testAcceptedAgreementDisplay),
			}, nil
		},
		create: func(_ context.Context, request marketplacesdk.CreateAcceptedAgreementRequest) (marketplacesdk.CreateAcceptedAgreementResponse, error) {
			if request.Signature == nil || *request.Signature != testSignature {
				t.Fatalf("CreateAcceptedAgreementRequest.Signature = %v, want %q", request.Signature, testSignature)
			}
			return marketplacesdk.CreateAcceptedAgreementResponse{
				AcceptedAgreement: sdkAcceptedAgreement(testAcceptedAgreementID, testAcceptedAgreementDisplay),
			}, nil
		},
	}

	runtimeClient := newAcceptedAgreementRuntimeClient(newAcceptedAgreementRuntimeTestManager(), client, nil)
	response, err := runtimeClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = true, want false")
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("CreateAcceptedAgreement() calls = %d, want 1", len(client.createRequests))
	}
	if got := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); got != testAcceptedAgreementID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testAcceptedAgreementID)
	}
	if got := resource.Status.AppliedSignature; got != testSignature {
		t.Fatalf("status.appliedSignature = %q, want %q", got, testSignature)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Active) {
		t.Fatalf("status.reason = %q, want %q", got, shared.Active)
	}
}

func TestAcceptedAgreementRuntimeClientBindsExistingAgreementFromList(t *testing.T) {
	resource := testAcceptedAgreementResource()
	client := &fakeAcceptedAgreementOCIClient{
		list: func(context.Context, marketplacesdk.ListAcceptedAgreementsRequest) (marketplacesdk.ListAcceptedAgreementsResponse, error) {
			return marketplacesdk.ListAcceptedAgreementsResponse{
				Items: []marketplacesdk.AcceptedAgreementSummary{
					sdkAcceptedAgreementSummary(testAcceptedAgreementID, testAcceptedAgreementDisplay),
				},
			}, nil
		},
		get: func(_ context.Context, request marketplacesdk.GetAcceptedAgreementRequest) (marketplacesdk.GetAcceptedAgreementResponse, error) {
			if request.AcceptedAgreementId == nil || *request.AcceptedAgreementId != testAcceptedAgreementID {
				t.Fatalf("GetAcceptedAgreementRequest.AcceptedAgreementId = %v, want %q", request.AcceptedAgreementId, testAcceptedAgreementID)
			}
			return marketplacesdk.GetAcceptedAgreementResponse{
				AcceptedAgreement: sdkAcceptedAgreement(testAcceptedAgreementID, testAcceptedAgreementDisplay),
			}, nil
		},
	}

	runtimeClient := newAcceptedAgreementRuntimeClient(newAcceptedAgreementRuntimeTestManager(), client, nil)
	response, err := runtimeClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = true, want false")
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("CreateAcceptedAgreement() calls = %d, want 0 on bind path", len(client.createRequests))
	}
	if got := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); got != testAcceptedAgreementID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testAcceptedAgreementID)
	}
}

func TestAcceptedAgreementRuntimeClientUpdateUsesMutableFieldsOnly(t *testing.T) {
	resource := testAcceptedAgreementResource()
	resource.Spec.DisplayName = "updated-display-name"
	resource.Status.OsokStatus.Ocid = shared.OCID(testAcceptedAgreementID)
	resource.Status.AppliedSignature = testSignature

	client := &fakeAcceptedAgreementOCIClient{
		get: func(_ context.Context, request marketplacesdk.GetAcceptedAgreementRequest) (marketplacesdk.GetAcceptedAgreementResponse, error) {
			if request.AcceptedAgreementId == nil || *request.AcceptedAgreementId != testAcceptedAgreementID {
				t.Fatalf("GetAcceptedAgreementRequest.AcceptedAgreementId = %v, want %q", request.AcceptedAgreementId, testAcceptedAgreementID)
			}
			return marketplacesdk.GetAcceptedAgreementResponse{
				AcceptedAgreement: sdkAcceptedAgreement(testAcceptedAgreementID, "old-display-name"),
			}, nil
		},
		update: func(_ context.Context, request marketplacesdk.UpdateAcceptedAgreementRequest) (marketplacesdk.UpdateAcceptedAgreementResponse, error) {
			if request.AcceptedAgreementId == nil || *request.AcceptedAgreementId != testAcceptedAgreementID {
				t.Fatalf("UpdateAcceptedAgreementRequest.AcceptedAgreementId = %v, want %q", request.AcceptedAgreementId, testAcceptedAgreementID)
			}
			if request.DisplayName == nil || *request.DisplayName != resource.Spec.DisplayName {
				t.Fatalf("UpdateAcceptedAgreementRequest.DisplayName = %v, want %q", request.DisplayName, resource.Spec.DisplayName)
			}
			return marketplacesdk.UpdateAcceptedAgreementResponse{
				AcceptedAgreement: sdkAcceptedAgreement(testAcceptedAgreementID, resource.Spec.DisplayName),
			}, nil
		},
	}

	runtimeClient := newAcceptedAgreementRuntimeClient(newAcceptedAgreementRuntimeTestManager(), client, nil)
	response, err := runtimeClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = true, want false")
	}
	if len(client.updateRequests) != 1 {
		t.Fatalf("UpdateAcceptedAgreement() calls = %d, want 1", len(client.updateRequests))
	}
	if got := resource.Status.AppliedSignature; got != testSignature {
		t.Fatalf("status.appliedSignature = %q, want %q", got, testSignature)
	}
}

func TestAcceptedAgreementRuntimeClientRejectsSignatureDrift(t *testing.T) {
	resource := testAcceptedAgreementResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testAcceptedAgreementID)
	resource.Status.AppliedSignature = "signed-agreement-v0"

	client := &fakeAcceptedAgreementOCIClient{}
	runtimeClient := newAcceptedAgreementRuntimeClient(newAcceptedAgreementRuntimeTestManager(), client, nil)

	response, err := runtimeClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want signature drift failure")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = true, want false for signature drift")
	}
	if !strings.Contains(err.Error(), "replacement when signature changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want replacement drift error", err)
	}
	if len(client.getRequests) != 0 || len(client.listRequests) != 0 || len(client.updateRequests) != 0 || len(client.createRequests) != 0 {
		t.Fatalf("OCI calls after signature drift = create:%d list:%d get:%d update:%d, want all 0", len(client.createRequests), len(client.listRequests), len(client.getRequests), len(client.updateRequests))
	}
}

func TestAcceptedAgreementRuntimeClientDeleteConfirmsOnNotFound(t *testing.T) {
	resource := testAcceptedAgreementResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testAcceptedAgreementID)
	resource.Status.AppliedSignature = testSignature

	client := &fakeAcceptedAgreementOCIClient{
		delete: func(_ context.Context, request marketplacesdk.DeleteAcceptedAgreementRequest) (marketplacesdk.DeleteAcceptedAgreementResponse, error) {
			if request.AcceptedAgreementId == nil || *request.AcceptedAgreementId != testAcceptedAgreementID {
				t.Fatalf("DeleteAcceptedAgreementRequest.AcceptedAgreementId = %v, want %q", request.AcceptedAgreementId, testAcceptedAgreementID)
			}
			if request.Signature == nil || *request.Signature != testSignature {
				t.Fatalf("DeleteAcceptedAgreementRequest.Signature = %v, want %q", request.Signature, testSignature)
			}
			return marketplacesdk.DeleteAcceptedAgreementResponse{}, nil
		},
		get: func(context.Context, marketplacesdk.GetAcceptedAgreementRequest) (marketplacesdk.GetAcceptedAgreementResponse, error) {
			return marketplacesdk.GetAcceptedAgreementResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "accepted agreement gone")
		},
	}

	runtimeClient := newAcceptedAgreementRuntimeClient(newAcceptedAgreementRuntimeTestManager(), client, nil)
	deleted, err := runtimeClient.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true")
	}
	if resource.Status.AppliedSignature != "" {
		t.Fatalf("status.appliedSignature = %q, want cleared on delete", resource.Status.AppliedSignature)
	}
}

func newAcceptedAgreementRuntimeTestManager() *AcceptedAgreementServiceManager {
	return &AcceptedAgreementServiceManager{Log: loggerutil.OSOKLogger{}}
}

func testAcceptedAgreementResource() *marketplacev1beta1.AcceptedAgreement {
	return &marketplacev1beta1.AcceptedAgreement{
		Spec: marketplacev1beta1.AcceptedAgreementSpec{
			CompartmentId:  testCompartmentID,
			ListingId:      testListingID,
			PackageVersion: testPackageVersion,
			AgreementId:    testAgreementID,
			Signature:      testSignature,
			DisplayName:    testAcceptedAgreementDisplay,
		},
	}
}

func sdkAcceptedAgreement(id string, displayName string) marketplacesdk.AcceptedAgreement {
	return marketplacesdk.AcceptedAgreement{
		Id:             common.String(id),
		DisplayName:    common.String(displayName),
		CompartmentId:  common.String(testCompartmentID),
		ListingId:      common.String(testListingID),
		PackageVersion: common.String(testPackageVersion),
		AgreementId:    common.String(testAgreementID),
	}
}

func sdkAcceptedAgreementSummary(id string, displayName string) marketplacesdk.AcceptedAgreementSummary {
	return marketplacesdk.AcceptedAgreementSummary{
		Id:             common.String(id),
		DisplayName:    common.String(displayName),
		CompartmentId:  common.String(testCompartmentID),
		ListingId:      common.String(testListingID),
		PackageVersion: common.String(testPackageVersion),
		AgreementId:    common.String(testAgreementID),
	}
}

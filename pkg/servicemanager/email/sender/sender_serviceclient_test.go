/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package sender

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	emailsdk "github.com/oracle/oci-go-sdk/v65/email"
	emailv1beta1 "github.com/oracle/oci-service-operator/api/email/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeSenderOCIClient struct {
	createSenderFn func(context.Context, emailsdk.CreateSenderRequest) (emailsdk.CreateSenderResponse, error)
	getSenderFn    func(context.Context, emailsdk.GetSenderRequest) (emailsdk.GetSenderResponse, error)
	listSendersFn  func(context.Context, emailsdk.ListSendersRequest) (emailsdk.ListSendersResponse, error)
	updateSenderFn func(context.Context, emailsdk.UpdateSenderRequest) (emailsdk.UpdateSenderResponse, error)
	deleteSenderFn func(context.Context, emailsdk.DeleteSenderRequest) (emailsdk.DeleteSenderResponse, error)
}

func (f *fakeSenderOCIClient) CreateSender(ctx context.Context, req emailsdk.CreateSenderRequest) (emailsdk.CreateSenderResponse, error) {
	if f.createSenderFn != nil {
		return f.createSenderFn(ctx, req)
	}
	return emailsdk.CreateSenderResponse{}, nil
}

func (f *fakeSenderOCIClient) GetSender(ctx context.Context, req emailsdk.GetSenderRequest) (emailsdk.GetSenderResponse, error) {
	if f.getSenderFn != nil {
		return f.getSenderFn(ctx, req)
	}
	return emailsdk.GetSenderResponse{}, nil
}

func (f *fakeSenderOCIClient) ListSenders(ctx context.Context, req emailsdk.ListSendersRequest) (emailsdk.ListSendersResponse, error) {
	if f.listSendersFn != nil {
		return f.listSendersFn(ctx, req)
	}
	return emailsdk.ListSendersResponse{}, nil
}

func (f *fakeSenderOCIClient) UpdateSender(ctx context.Context, req emailsdk.UpdateSenderRequest) (emailsdk.UpdateSenderResponse, error) {
	if f.updateSenderFn != nil {
		return f.updateSenderFn(ctx, req)
	}
	return emailsdk.UpdateSenderResponse{}, nil
}

func (f *fakeSenderOCIClient) DeleteSender(ctx context.Context, req emailsdk.DeleteSenderRequest) (emailsdk.DeleteSenderResponse, error) {
	if f.deleteSenderFn != nil {
		return f.deleteSenderFn(ctx, req)
	}
	return emailsdk.DeleteSenderResponse{}, nil
}

func testSenderClient(fake *fakeSenderOCIClient) defaultSenderServiceClient {
	return defaultSenderServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*emailv1beta1.Sender](generatedruntime.Config[*emailv1beta1.Sender]{
			Kind:      "Sender",
			SDKName:   "Sender",
			Log:       loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
			Semantics: newSenderRuntimeSemantics(),
			Create: &generatedruntime.Operation{
				NewRequest: func() any { return &emailsdk.CreateSenderRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return fake.CreateSender(ctx, *request.(*emailsdk.CreateSenderRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "CreateSenderDetails", RequestName: "CreateSenderDetails", Contribution: "body", PreferResourceID: false},
				},
			},
			Get: &generatedruntime.Operation{
				NewRequest: func() any { return &emailsdk.GetSenderRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return fake.GetSender(ctx, *request.(*emailsdk.GetSenderRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "SenderId", RequestName: "senderId", Contribution: "path", PreferResourceID: true},
				},
			},
			List: &generatedruntime.Operation{
				NewRequest: func() any { return &emailsdk.ListSendersRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return fake.ListSenders(ctx, *request.(*emailsdk.ListSendersRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", PreferResourceID: false},
					{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query", PreferResourceID: false},
					{FieldName: "Domain", RequestName: "domain", Contribution: "query", PreferResourceID: false},
					{FieldName: "EmailAddress", RequestName: "emailAddress", Contribution: "query", PreferResourceID: false},
					{FieldName: "Page", RequestName: "page", Contribution: "query", PreferResourceID: false},
					{FieldName: "Limit", RequestName: "limit", Contribution: "query", PreferResourceID: false},
					{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query", PreferResourceID: false},
					{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query", PreferResourceID: false},
				},
			},
			Update: &generatedruntime.Operation{
				NewRequest: func() any { return &emailsdk.UpdateSenderRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return fake.UpdateSender(ctx, *request.(*emailsdk.UpdateSenderRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "SenderId", RequestName: "senderId", Contribution: "path", PreferResourceID: true},
					{FieldName: "UpdateSenderDetails", RequestName: "UpdateSenderDetails", Contribution: "body", PreferResourceID: false},
				},
			},
			Delete: &generatedruntime.Operation{
				NewRequest: func() any { return &emailsdk.DeleteSenderRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return fake.DeleteSender(ctx, *request.(*emailsdk.DeleteSenderRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "SenderId", RequestName: "senderId", Contribution: "path", PreferResourceID: true},
				},
			},
		}),
	}
}

func makeSenderResource() *emailv1beta1.Sender {
	return &emailv1beta1.Sender{
		Spec: emailv1beta1.SenderSpec{
			CompartmentId: "ocid1.compartment.oc1..senderexample",
			EmailAddress:  "sender@example.com",
		},
	}
}

func makeSDKSender(id string, state emailsdk.SenderLifecycleStateEnum, freeformTags map[string]string) emailsdk.Sender {
	return emailsdk.Sender{
		Id:             common.String(id),
		CompartmentId:  common.String("ocid1.compartment.oc1..senderexample"),
		EmailAddress:   common.String("sender@example.com"),
		LifecycleState: state,
		FreeformTags:   freeformTags,
	}
}

func makeSDKSenderSummary(id string, state emailsdk.SenderLifecycleStateEnum) emailsdk.SenderSummary {
	return emailsdk.SenderSummary{
		Id:             common.String(id),
		CompartmentId:  common.String("ocid1.compartment.oc1..senderexample"),
		EmailAddress:   common.String("sender@example.com"),
		LifecycleState: state,
	}
}

func TestSenderServiceClientCreatesAndProjectsStatus(t *testing.T) {
	t.Parallel()

	var createRequest emailsdk.CreateSenderRequest
	getCalls := 0
	client := testSenderClient(&fakeSenderOCIClient{
		listSendersFn: func(_ context.Context, req emailsdk.ListSendersRequest) (emailsdk.ListSendersResponse, error) {
			if req.CompartmentId == nil || *req.CompartmentId != "ocid1.compartment.oc1..senderexample" {
				t.Fatalf("list compartmentId = %v, want sender compartment", req.CompartmentId)
			}
			if req.EmailAddress == nil || *req.EmailAddress != "sender@example.com" {
				t.Fatalf("list emailAddress = %v, want sender@example.com", req.EmailAddress)
			}
			if req.Domain != nil {
				t.Fatalf("list domain = %v, want nil", req.Domain)
			}
			return emailsdk.ListSendersResponse{}, nil
		},
		createSenderFn: func(_ context.Context, req emailsdk.CreateSenderRequest) (emailsdk.CreateSenderResponse, error) {
			createRequest = req
			return emailsdk.CreateSenderResponse{
				Sender:       makeSDKSender("ocid1.sender.oc1..created", emailsdk.SenderLifecycleStateCreating, nil),
				OpcRequestId: common.String("opc-create-1"),
			}, nil
		},
		getSenderFn: func(_ context.Context, req emailsdk.GetSenderRequest) (emailsdk.GetSenderResponse, error) {
			getCalls++
			if req.SenderId == nil || *req.SenderId != "ocid1.sender.oc1..created" {
				t.Fatalf("get senderId = %v, want created sender OCID", req.SenderId)
			}
			return emailsdk.GetSenderResponse{
				Sender: makeSDKSender("ocid1.sender.oc1..created", emailsdk.SenderLifecycleStateActive, map[string]string{"managed-by": "osok"}),
			}, nil
		},
	})

	resource := makeSenderResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue once follow-up GetSender reports ACTIVE")
	}
	if createRequest.CreateSenderDetails.CompartmentId == nil || *createRequest.CreateSenderDetails.CompartmentId != resource.Spec.CompartmentId {
		t.Fatalf("create compartmentId = %v, want %q", createRequest.CreateSenderDetails.CompartmentId, resource.Spec.CompartmentId)
	}
	if createRequest.CreateSenderDetails.EmailAddress == nil || *createRequest.CreateSenderDetails.EmailAddress != resource.Spec.EmailAddress {
		t.Fatalf("create emailAddress = %v, want %q", createRequest.CreateSenderDetails.EmailAddress, resource.Spec.EmailAddress)
	}
	if getCalls != 1 {
		t.Fatalf("GetSender() calls = %d, want 1 follow-up read", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != "ocid1.sender.oc1..created" {
		t.Fatalf("status.ocid = %q, want created sender OCID", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", got, "opc-create-1")
	}
	if got := resource.Status.Id; got != "ocid1.sender.oc1..created" {
		t.Fatalf("status.id = %q, want created sender OCID", got)
	}
	if got := resource.Status.LifecycleState; got != string(emailsdk.SenderLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
}

func TestSenderServiceClientBindsExistingSenderWithoutCreate(t *testing.T) {
	t.Parallel()

	createCalled := false
	client := testSenderClient(&fakeSenderOCIClient{
		listSendersFn: func(_ context.Context, req emailsdk.ListSendersRequest) (emailsdk.ListSendersResponse, error) {
			if req.CompartmentId == nil || *req.CompartmentId != "ocid1.compartment.oc1..senderexample" {
				t.Fatalf("list compartmentId = %v, want sender compartment", req.CompartmentId)
			}
			if req.EmailAddress == nil || *req.EmailAddress != "sender@example.com" {
				t.Fatalf("list emailAddress = %v, want sender@example.com", req.EmailAddress)
			}
			return emailsdk.ListSendersResponse{
				Items: []emailsdk.SenderSummary{
					makeSDKSenderSummary("ocid1.sender.oc1..existing", emailsdk.SenderLifecycleStateActive),
				},
			}, nil
		},
		getSenderFn: func(_ context.Context, req emailsdk.GetSenderRequest) (emailsdk.GetSenderResponse, error) {
			if req.SenderId == nil || *req.SenderId != "ocid1.sender.oc1..existing" {
				t.Fatalf("get senderId = %v, want existing sender OCID", req.SenderId)
			}
			return emailsdk.GetSenderResponse{
				Sender: makeSDKSender("ocid1.sender.oc1..existing", emailsdk.SenderLifecycleStateActive, nil),
			}, nil
		},
		createSenderFn: func(_ context.Context, _ emailsdk.CreateSenderRequest) (emailsdk.CreateSenderResponse, error) {
			createCalled = true
			return emailsdk.CreateSenderResponse{}, nil
		},
	})

	resource := makeSenderResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success when list lookup reuses an existing sender")
	}
	if createCalled {
		t.Fatal("CreateSender() should not be called when ListSenders finds a matching sender")
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != "ocid1.sender.oc1..existing" {
		t.Fatalf("status.ocid = %q, want existing sender OCID", got)
	}
}

func TestSenderServiceClientUpdatesFreeformTags(t *testing.T) {
	t.Parallel()

	var updateRequest emailsdk.UpdateSenderRequest
	getCalls := 0
	updateCalls := 0
	client := testSenderClient(&fakeSenderOCIClient{
		getSenderFn: func(_ context.Context, req emailsdk.GetSenderRequest) (emailsdk.GetSenderResponse, error) {
			getCalls++
			if req.SenderId == nil || *req.SenderId != "ocid1.sender.oc1..existing" {
				t.Fatalf("get senderId = %v, want existing sender OCID", req.SenderId)
			}
			tags := map[string]string{"managed-by": "legacy"}
			if getCalls > 1 {
				tags = map[string]string{"managed-by": "osok"}
			}
			return emailsdk.GetSenderResponse{
				Sender: makeSDKSender("ocid1.sender.oc1..existing", emailsdk.SenderLifecycleStateActive, tags),
			}, nil
		},
		updateSenderFn: func(_ context.Context, req emailsdk.UpdateSenderRequest) (emailsdk.UpdateSenderResponse, error) {
			updateCalls++
			updateRequest = req
			return emailsdk.UpdateSenderResponse{
				Sender:       makeSDKSender("ocid1.sender.oc1..existing", emailsdk.SenderLifecycleStateActive, map[string]string{"managed-by": "osok"}),
				OpcRequestId: common.String("opc-update-1"),
			}, nil
		},
	})

	resource := makeSenderResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.sender.oc1..existing")
	resource.Spec.FreeformTags = map[string]string{"managed-by": "osok"}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success after updating mutable tags")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue once update follow-up GetSender reports ACTIVE")
	}
	if updateCalls != 1 {
		t.Fatalf("UpdateSender() calls = %d, want 1", updateCalls)
	}
	if getCalls != 2 {
		t.Fatalf("GetSender() calls = %d, want 2 (observe + follow-up)", getCalls)
	}
	if updateRequest.SenderId == nil || *updateRequest.SenderId != "ocid1.sender.oc1..existing" {
		t.Fatalf("update senderId = %v, want existing sender OCID", updateRequest.SenderId)
	}
	if got := updateRequest.UpdateSenderDetails.FreeformTags["managed-by"]; got != "osok" {
		t.Fatalf("update freeformTags[managed-by] = %q, want %q", got, "osok")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", got, "opc-update-1")
	}
	if got := resource.Status.FreeformTags["managed-by"]; got != "osok" {
		t.Fatalf("status.freeformTags[managed-by] = %q, want %q", got, "osok")
	}
}

func TestSenderServiceClientRejectsReplacementOnlyEmailAddressDrift(t *testing.T) {
	t.Parallel()

	updateCalled := false
	client := testSenderClient(&fakeSenderOCIClient{
		getSenderFn: func(_ context.Context, _ emailsdk.GetSenderRequest) (emailsdk.GetSenderResponse, error) {
			return emailsdk.GetSenderResponse{
				Sender: makeSDKSender("ocid1.sender.oc1..existing", emailsdk.SenderLifecycleStateActive, nil),
			}, nil
		},
		updateSenderFn: func(_ context.Context, _ emailsdk.UpdateSenderRequest) (emailsdk.UpdateSenderResponse, error) {
			updateCalled = true
			return emailsdk.UpdateSenderResponse{}, nil
		},
	})

	resource := makeSenderResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.sender.oc1..existing")
	resource.Spec.EmailAddress = "replacement@example.com"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "require replacement when emailAddress changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want replacement-required drift failure", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful response", response)
	}
	if updateCalled {
		t.Fatal("UpdateSender() should not be called when replacement-only drift is detected")
	}
}

func TestSenderServiceClientDeleteConfirmsDeletion(t *testing.T) {
	t.Parallel()

	var deleteRequest emailsdk.DeleteSenderRequest
	getCalls := 0
	client := testSenderClient(&fakeSenderOCIClient{
		getSenderFn: func(_ context.Context, req emailsdk.GetSenderRequest) (emailsdk.GetSenderResponse, error) {
			getCalls++
			if req.SenderId == nil || *req.SenderId != "ocid1.sender.oc1..existing" {
				t.Fatalf("get senderId = %v, want existing sender OCID", req.SenderId)
			}
			state := emailsdk.SenderLifecycleStateActive
			if getCalls > 1 {
				state = emailsdk.SenderLifecycleStateDeleted
			}
			return emailsdk.GetSenderResponse{
				Sender: makeSDKSender("ocid1.sender.oc1..existing", state, nil),
			}, nil
		},
		deleteSenderFn: func(_ context.Context, req emailsdk.DeleteSenderRequest) (emailsdk.DeleteSenderResponse, error) {
			deleteRequest = req
			return emailsdk.DeleteSenderResponse{
				OpcRequestId: common.String("opc-delete-1"),
			}, nil
		},
	})

	resource := makeSenderResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.sender.oc1..existing")

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() should report success once follow-up GetSender confirms DELETED")
	}
	if getCalls != 2 {
		t.Fatalf("GetSender() calls = %d, want 2 (preflight + confirmation)", getCalls)
	}
	if deleteRequest.SenderId == nil || *deleteRequest.SenderId != "ocid1.sender.oc1..existing" {
		t.Fatalf("delete senderId = %v, want existing sender OCID", deleteRequest.SenderId)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete confirmation timestamp")
	}
	if got := resource.Status.LifecycleState; got != string(emailsdk.SenderLifecycleStateDeleted) {
		t.Fatalf("status.lifecycleState = %q, want DELETED", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", got, "opc-delete-1")
	}
}

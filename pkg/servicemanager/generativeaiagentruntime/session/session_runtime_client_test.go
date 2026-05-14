/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package session

import (
	"context"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	generativeaiagentruntimesdk "github.com/oracle/oci-go-sdk/v65/generativeaiagentruntime"
	generativeaiagentruntimev1beta1 "github.com/oracle/oci-service-operator/api/generativeaiagentruntime/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type stubSessionServiceClient struct {
	createOrUpdate func(context.Context, *generativeaiagentruntimev1beta1.Session, ctrl.Request) (servicemanager.OSOKResponse, error)
	delete         func(context.Context, *generativeaiagentruntimev1beta1.Session) (bool, error)
}

func (s stubSessionServiceClient) CreateOrUpdate(
	ctx context.Context,
	resource *generativeaiagentruntimev1beta1.Session,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return s.createOrUpdate(ctx, resource, req)
}

func (s stubSessionServiceClient) Delete(
	ctx context.Context,
	resource *generativeaiagentruntimev1beta1.Session,
) (bool, error) {
	return s.delete(ctx, resource)
}

func TestReviewedSessionRuntimeSemanticsClearsLifecycle(t *testing.T) {
	semantics := reviewedSessionRuntimeSemantics()
	if len(semantics.Lifecycle.ActiveStates) != 0 {
		t.Fatalf("Lifecycle.ActiveStates = %v, want empty", semantics.Lifecycle.ActiveStates)
	}
	if len(semantics.Lifecycle.ProvisioningStates) != 0 {
		t.Fatalf("Lifecycle.ProvisioningStates = %v, want empty", semantics.Lifecycle.ProvisioningStates)
	}
}

func TestBuildSessionCreateBodyIncludesOptionalFields(t *testing.T) {
	body, err := buildSessionCreateBody(context.Background(), &generativeaiagentruntimev1beta1.Session{
		Spec: generativeaiagentruntimev1beta1.SessionSpec{
			AgentEndpointId: "ocid1.generativeaiagentendpoint.oc1..example",
			DisplayName:     "  chat session  ",
			Description:     "  some description  ",
		},
	}, "")
	if err != nil {
		t.Fatalf("buildSessionCreateBody() error = %v", err)
	}

	details, ok := body.(generativeaiagentruntimesdk.CreateSessionDetails)
	if !ok {
		t.Fatalf("buildSessionCreateBody() returned %T, want CreateSessionDetails", body)
	}
	if got := stringPointerValue(details.DisplayName); got != "chat session" {
		t.Fatalf("DisplayName = %q, want %q", got, "chat session")
	}
	if got := stringPointerValue(details.Description); got != "some description" {
		t.Fatalf("Description = %q, want %q", got, "some description")
	}
}

func TestBuildSessionUpdateBodySupportsClearingFields(t *testing.T) {
	body, updateNeeded, err := buildSessionUpdateBody(
		context.Background(),
		&generativeaiagentruntimev1beta1.Session{
			Spec: generativeaiagentruntimev1beta1.SessionSpec{
				AgentEndpointId: "ocid1.generativeaiagentendpoint.oc1..example",
				DisplayName:     "",
				Description:     "",
			},
		},
		"",
		generativeaiagentruntimesdk.GetSessionResponse{
			Session: generativeaiagentruntimesdk.Session{
				DisplayName: common.String("existing"),
				Description: common.String("existing description"),
			},
		},
	)
	if err != nil {
		t.Fatalf("buildSessionUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildSessionUpdateBody() updateNeeded = false, want true")
	}

	details, ok := body.(generativeaiagentruntimesdk.UpdateSessionDetails)
	if !ok {
		t.Fatalf("buildSessionUpdateBody() returned %T, want UpdateSessionDetails", body)
	}
	if got := stringPointerValue(details.DisplayName); got != "" {
		t.Fatalf("DisplayName = %q, want empty string clear", got)
	}
	if got := stringPointerValue(details.Description); got != "" {
		t.Fatalf("Description = %q, want empty string clear", got)
	}
}

func TestSessionStatusMirrorClientProjectsPathIdentity(t *testing.T) {
	resource := &generativeaiagentruntimev1beta1.Session{
		Spec: generativeaiagentruntimev1beta1.SessionSpec{
			AgentEndpointId: "ocid1.generativeaiagentendpoint.oc1..example",
		},
		Status: generativeaiagentruntimev1beta1.SessionStatus{
			Id: "session-id",
		},
	}

	client := wrapSessionStatusMirrorClient(stubSessionServiceClient{
		createOrUpdate: func(context.Context, *generativeaiagentruntimev1beta1.Session, ctrl.Request) (servicemanager.OSOKResponse, error) {
			return servicemanager.OSOKResponse{IsSuccessful: true}, nil
		},
		delete: func(context.Context, *generativeaiagentruntimev1beta1.Session) (bool, error) {
			return true, nil
		},
	})

	if _, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{}); err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if got := resource.Status.AgentEndpointId; got != resource.Spec.AgentEndpointId {
		t.Fatalf("status.agentEndpointId = %q, want %q", got, resource.Spec.AgentEndpointId)
	}
	if got := resource.Status.SessionId; got != resource.Status.Id {
		t.Fatalf("status.sessionId = %q, want %q", got, resource.Status.Id)
	}
}

func TestSynchronousSessionServiceClientSettlesActive(t *testing.T) {
	resource := &generativeaiagentruntimev1beta1.Session{
		Spec: generativeaiagentruntimev1beta1.SessionSpec{
			DisplayName: "session-name",
		},
		Status: generativeaiagentruntimev1beta1.SessionStatus{
			OsokStatus: shared.OSOKStatus{
				Reason: string(shared.Provisioning),
			},
		},
	}

	client := &synchronousSessionServiceClient{
		delegate: stubSessionServiceClient{
			createOrUpdate: func(context.Context, *generativeaiagentruntimev1beta1.Session, ctrl.Request) (servicemanager.OSOKResponse, error) {
				return servicemanager.OSOKResponse{
					IsSuccessful:  true,
					ShouldRequeue: true,
				}, nil
			},
			delete: func(context.Context, *generativeaiagentruntimev1beta1.Session) (bool, error) {
				return true, nil
			},
		},
		log: loggerutil.OSOKLogger{},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if response.ShouldRequeue {
		t.Fatal("response.ShouldRequeue = true, want false")
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Active) {
		t.Fatalf("status.reason = %q, want %q", got, shared.Active)
	}
}

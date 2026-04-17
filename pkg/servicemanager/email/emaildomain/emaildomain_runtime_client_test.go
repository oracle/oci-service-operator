/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package emaildomain

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type emailDomainOCIClient interface {
	CreateEmailDomain(context.Context, emailsdk.CreateEmailDomainRequest) (emailsdk.CreateEmailDomainResponse, error)
	GetEmailDomain(context.Context, emailsdk.GetEmailDomainRequest) (emailsdk.GetEmailDomainResponse, error)
	ListEmailDomains(context.Context, emailsdk.ListEmailDomainsRequest) (emailsdk.ListEmailDomainsResponse, error)
	UpdateEmailDomain(context.Context, emailsdk.UpdateEmailDomainRequest) (emailsdk.UpdateEmailDomainResponse, error)
	DeleteEmailDomain(context.Context, emailsdk.DeleteEmailDomainRequest) (emailsdk.DeleteEmailDomainResponse, error)
}

type fakeEmailDomainOCIClient struct {
	createFn func(context.Context, emailsdk.CreateEmailDomainRequest) (emailsdk.CreateEmailDomainResponse, error)
	getFn    func(context.Context, emailsdk.GetEmailDomainRequest) (emailsdk.GetEmailDomainResponse, error)
	listFn   func(context.Context, emailsdk.ListEmailDomainsRequest) (emailsdk.ListEmailDomainsResponse, error)
	updateFn func(context.Context, emailsdk.UpdateEmailDomainRequest) (emailsdk.UpdateEmailDomainResponse, error)
	deleteFn func(context.Context, emailsdk.DeleteEmailDomainRequest) (emailsdk.DeleteEmailDomainResponse, error)
}

func (f *fakeEmailDomainOCIClient) CreateEmailDomain(ctx context.Context, req emailsdk.CreateEmailDomainRequest) (emailsdk.CreateEmailDomainResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return emailsdk.CreateEmailDomainResponse{}, nil
}

func (f *fakeEmailDomainOCIClient) GetEmailDomain(ctx context.Context, req emailsdk.GetEmailDomainRequest) (emailsdk.GetEmailDomainResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return emailsdk.GetEmailDomainResponse{}, fakeEmailDomainServiceError{
		statusCode: 404,
		code:       "NotFound",
		message:    "missing",
	}
}

func (f *fakeEmailDomainOCIClient) ListEmailDomains(ctx context.Context, req emailsdk.ListEmailDomainsRequest) (emailsdk.ListEmailDomainsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return emailsdk.ListEmailDomainsResponse{}, nil
}

func (f *fakeEmailDomainOCIClient) UpdateEmailDomain(ctx context.Context, req emailsdk.UpdateEmailDomainRequest) (emailsdk.UpdateEmailDomainResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return emailsdk.UpdateEmailDomainResponse{}, nil
}

func (f *fakeEmailDomainOCIClient) DeleteEmailDomain(ctx context.Context, req emailsdk.DeleteEmailDomainRequest) (emailsdk.DeleteEmailDomainResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return emailsdk.DeleteEmailDomainResponse{}, nil
}

type fakeEmailDomainServiceError struct {
	statusCode int
	code       string
	message    string
}

func (f fakeEmailDomainServiceError) Error() string          { return f.message }
func (f fakeEmailDomainServiceError) GetHTTPStatusCode() int { return f.statusCode }
func (f fakeEmailDomainServiceError) GetMessage() string     { return f.message }
func (f fakeEmailDomainServiceError) GetCode() string        { return f.code }
func (f fakeEmailDomainServiceError) GetOpcRequestID() string {
	return ""
}

func newTestEmailDomainDelegate(client emailDomainOCIClient) EmailDomainServiceClient {
	if client == nil {
		client = &fakeEmailDomainOCIClient{}
	}

	config := generatedruntime.Config[*emailv1beta1.EmailDomain]{
		Kind:      "EmailDomain",
		SDKName:   "EmailDomain",
		Log:       loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
		Semantics: newEmailDomainRuntimeSemantics(),
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &emailsdk.CreateEmailDomainRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.CreateEmailDomain(ctx, *request.(*emailsdk.CreateEmailDomainRequest))
			},
			Fields: []generatedruntime.RequestField{{FieldName: "CreateEmailDomainDetails", RequestName: "CreateEmailDomainDetails", Contribution: "body", PreferResourceID: false}},
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &emailsdk.GetEmailDomainRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.GetEmailDomain(ctx, *request.(*emailsdk.GetEmailDomainRequest))
			},
			Fields: []generatedruntime.RequestField{{FieldName: "EmailDomainId", RequestName: "emailDomainId", Contribution: "path", PreferResourceID: true}},
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &emailsdk.ListEmailDomainsRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.ListEmailDomains(ctx, *request.(*emailsdk.ListEmailDomainsRequest))
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", PreferResourceID: false},
				{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: false},
				{FieldName: "Name", RequestName: "name", Contribution: "query", PreferResourceID: false},
				{FieldName: "Limit", RequestName: "limit", Contribution: "query", PreferResourceID: false},
				{FieldName: "Page", RequestName: "page", Contribution: "query", PreferResourceID: false},
				{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query", PreferResourceID: false},
				{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query", PreferResourceID: false},
				{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query", PreferResourceID: false},
			},
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &emailsdk.UpdateEmailDomainRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.UpdateEmailDomain(ctx, *request.(*emailsdk.UpdateEmailDomainRequest))
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "EmailDomainId", RequestName: "emailDomainId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateEmailDomainDetails", RequestName: "UpdateEmailDomainDetails", Contribution: "body", PreferResourceID: false},
			},
		},
		Delete: &generatedruntime.Operation{
			NewRequest: func() any { return &emailsdk.DeleteEmailDomainRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.DeleteEmailDomain(ctx, *request.(*emailsdk.DeleteEmailDomainRequest))
			},
			Fields: []generatedruntime.RequestField{{FieldName: "EmailDomainId", RequestName: "emailDomainId", Contribution: "path", PreferResourceID: true}},
		},
	}

	return defaultEmailDomainServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*emailv1beta1.EmailDomain](config),
	}
}

func newTestEmailDomainManager(client emailDomainOCIClient) *EmailDomainServiceManager {
	manager := &EmailDomainServiceManager{
		Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
	}
	return manager.WithClient(newTestEmailDomainDelegate(client))
}

func makeSpecEmailDomain() *emailv1beta1.EmailDomain {
	return &emailv1beta1.EmailDomain{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emaildomain-sample",
			Namespace: "default",
		},
		Spec: emailv1beta1.EmailDomainSpec{
			Name:          "mail.example.com",
			CompartmentId: "ocid1.compartment.oc1..exampleuniqueID",
			Description:   "EmailDomain sample",
		},
	}
}

func makeSDKEmailDomain(id, name, compartmentID, description string, lifecycleState emailsdk.EmailDomainLifecycleStateEnum) emailsdk.EmailDomain {
	return emailsdk.EmailDomain{
		Id:             common.String(id),
		Name:           common.String(name),
		CompartmentId:  common.String(compartmentID),
		Description:    common.String(description),
		LifecycleState: lifecycleState,
	}
}

func makeSDKEmailDomainSummary(id, name, compartmentID, description string, lifecycleState emailsdk.EmailDomainLifecycleStateEnum) emailsdk.EmailDomainSummary {
	return emailsdk.EmailDomainSummary{
		Id:             common.String(id),
		Name:           common.String(name),
		CompartmentId:  common.String(compartmentID),
		Description:    common.String(description),
		LifecycleState: lifecycleState,
	}
}

func TestEmailDomainCreateOrUpdateBindsExistingResourceByList(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.emaildomain.oc1..existing"

	resource := makeSpecEmailDomain()
	createCalled := false
	updateCalled := false
	listCalls := 0
	getCalls := 0

	manager := newTestEmailDomainManager(&fakeEmailDomainOCIClient{
		createFn: func(context.Context, emailsdk.CreateEmailDomainRequest) (emailsdk.CreateEmailDomainResponse, error) {
			createCalled = true
			return emailsdk.CreateEmailDomainResponse{}, nil
		},
		getFn: func(_ context.Context, req emailsdk.GetEmailDomainRequest) (emailsdk.GetEmailDomainResponse, error) {
			getCalls++
			if req.EmailDomainId == nil || *req.EmailDomainId != existingID {
				t.Fatalf("GetEmailDomainRequest.EmailDomainId = %v, want %q", req.EmailDomainId, existingID)
			}
			return emailsdk.GetEmailDomainResponse{
				EmailDomain: makeSDKEmailDomain(existingID, resource.Spec.Name, resource.Spec.CompartmentId, resource.Spec.Description, emailsdk.EmailDomainLifecycleStateActive),
			}, nil
		},
		listFn: func(_ context.Context, req emailsdk.ListEmailDomainsRequest) (emailsdk.ListEmailDomainsResponse, error) {
			listCalls++
			if req.CompartmentId == nil || *req.CompartmentId != resource.Spec.CompartmentId {
				t.Fatalf("ListEmailDomainsRequest.CompartmentId = %v, want %q", req.CompartmentId, resource.Spec.CompartmentId)
			}
			if req.Name == nil || *req.Name != resource.Spec.Name {
				t.Fatalf("ListEmailDomainsRequest.Name = %v, want %q", req.Name, resource.Spec.Name)
			}
			return emailsdk.ListEmailDomainsResponse{
				EmailDomainCollection: emailsdk.EmailDomainCollection{
					Items: []emailsdk.EmailDomainSummary{
						makeSDKEmailDomainSummary("ocid1.emaildomain.oc1..other", "other.example.com", resource.Spec.CompartmentId, "other", emailsdk.EmailDomainLifecycleStateActive),
						makeSDKEmailDomainSummary(existingID, resource.Spec.Name, resource.Spec.CompartmentId, resource.Spec.Description, emailsdk.EmailDomainLifecycleStateActive),
					},
				},
			}, nil
		},
		updateFn: func(context.Context, emailsdk.UpdateEmailDomainRequest) (emailsdk.UpdateEmailDomainResponse, error) {
			updateCalled = true
			return emailsdk.UpdateEmailDomainResponse{}, nil
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want success", response)
	}
	if response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want no requeue for ACTIVE bind", response)
	}
	if createCalled {
		t.Fatal("CreateEmailDomain() called, want list-bind path")
	}
	if updateCalled {
		t.Fatal("UpdateEmailDomain() called, want observe-only bind path")
	}
	if listCalls != 1 {
		t.Fatalf("ListEmailDomains() calls = %d, want 1", listCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetEmailDomain() calls = %d, want 1", getCalls)
	}
	if got := resource.Status.Id; got != existingID {
		t.Fatalf("status.id = %q, want %q", got, existingID)
	}
	if got := resource.Status.LifecycleState; got != "ACTIVE" {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != existingID {
		t.Fatalf("status.status.ocid = %q, want %q", got, existingID)
	}
}

func TestEmailDomainCreateOrUpdateUpdatesMutableDescription(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.emaildomain.oc1..existing"

	resource := makeSpecEmailDomain()
	resource.Spec.Description = "Updated EmailDomain description"
	resource.Status.Id = existingID
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)

	getCalls := 0
	updateCalls := 0

	manager := newTestEmailDomainManager(&fakeEmailDomainOCIClient{
		getFn: func(_ context.Context, req emailsdk.GetEmailDomainRequest) (emailsdk.GetEmailDomainResponse, error) {
			getCalls++
			if req.EmailDomainId == nil || *req.EmailDomainId != existingID {
				t.Fatalf("GetEmailDomainRequest.EmailDomainId = %v, want %q", req.EmailDomainId, existingID)
			}
			description := "EmailDomain sample"
			if getCalls > 1 {
				description = resource.Spec.Description
			}
			return emailsdk.GetEmailDomainResponse{
				EmailDomain: makeSDKEmailDomain(existingID, resource.Spec.Name, resource.Spec.CompartmentId, description, emailsdk.EmailDomainLifecycleStateActive),
			}, nil
		},
		updateFn: func(_ context.Context, req emailsdk.UpdateEmailDomainRequest) (emailsdk.UpdateEmailDomainResponse, error) {
			updateCalls++
			if req.EmailDomainId == nil || *req.EmailDomainId != existingID {
				t.Fatalf("UpdateEmailDomainRequest.EmailDomainId = %v, want %q", req.EmailDomainId, existingID)
			}
			if req.Description == nil || *req.Description != resource.Spec.Description {
				t.Fatalf("UpdateEmailDomainRequest.Description = %v, want %q", req.Description, resource.Spec.Description)
			}
			if req.FreeformTags != nil {
				t.Fatalf("UpdateEmailDomainRequest.FreeformTags = %#v, want nil when unchanged", req.FreeformTags)
			}
			if req.DefinedTags != nil {
				t.Fatalf("UpdateEmailDomainRequest.DefinedTags = %#v, want nil when unchanged", req.DefinedTags)
			}
			return emailsdk.UpdateEmailDomainResponse{}, nil
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want success", response)
	}
	if response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want no requeue after ACTIVE update follow-up", response)
	}
	if updateCalls != 1 {
		t.Fatalf("UpdateEmailDomain() calls = %d, want 1", updateCalls)
	}
	if getCalls != 2 {
		t.Fatalf("GetEmailDomain() calls = %d, want 2", getCalls)
	}
	if got := resource.Status.Description; got != resource.Spec.Description {
		t.Fatalf("status.description = %q, want %q", got, resource.Spec.Description)
	}
	if got := resource.Status.LifecycleState; got != "ACTIVE" {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
}

func TestEmailDomainCreateOrUpdateRejectsForceNewNameDrift(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.emaildomain.oc1..existing"

	resource := makeSpecEmailDomain()
	resource.Spec.Name = "replacement.example.com"
	resource.Status.Id = existingID
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)

	updateCalled := false

	manager := newTestEmailDomainManager(&fakeEmailDomainOCIClient{
		getFn: func(_ context.Context, req emailsdk.GetEmailDomainRequest) (emailsdk.GetEmailDomainResponse, error) {
			if req.EmailDomainId == nil || *req.EmailDomainId != existingID {
				t.Fatalf("GetEmailDomainRequest.EmailDomainId = %v, want %q", req.EmailDomainId, existingID)
			}
			return emailsdk.GetEmailDomainResponse{
				EmailDomain: makeSDKEmailDomain(existingID, "mail.example.com", resource.Spec.CompartmentId, resource.Spec.Description, emailsdk.EmailDomainLifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, emailsdk.UpdateEmailDomainRequest) (emailsdk.UpdateEmailDomainResponse, error) {
			updateCalled = true
			return emailsdk.UpdateEmailDomainResponse{}, nil
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "require replacement when name changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want replacement-required name drift failure", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful response", response)
	}
	if updateCalled {
		t.Fatal("UpdateEmailDomain() called, want force-new drift rejection before update")
	}
}

func TestEmailDomainDeleteConfirmsTerminalLifecycle(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.emaildomain.oc1..existing"

	resource := makeSpecEmailDomain()
	resource.Status.Id = existingID
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)

	getCalls := 0
	deleteCalls := 0

	manager := newTestEmailDomainManager(&fakeEmailDomainOCIClient{
		getFn: func(_ context.Context, req emailsdk.GetEmailDomainRequest) (emailsdk.GetEmailDomainResponse, error) {
			getCalls++
			if req.EmailDomainId == nil || *req.EmailDomainId != existingID {
				t.Fatalf("GetEmailDomainRequest.EmailDomainId = %v, want %q", req.EmailDomainId, existingID)
			}
			switch getCalls {
			case 1:
				return emailsdk.GetEmailDomainResponse{
					EmailDomain: makeSDKEmailDomain(existingID, resource.Spec.Name, resource.Spec.CompartmentId, resource.Spec.Description, emailsdk.EmailDomainLifecycleStateActive),
				}, nil
			case 2:
				return emailsdk.GetEmailDomainResponse{
					EmailDomain: makeSDKEmailDomain(existingID, resource.Spec.Name, resource.Spec.CompartmentId, resource.Spec.Description, emailsdk.EmailDomainLifecycleStateDeleting),
				}, nil
			default:
				return emailsdk.GetEmailDomainResponse{
					EmailDomain: makeSDKEmailDomain(existingID, resource.Spec.Name, resource.Spec.CompartmentId, resource.Spec.Description, emailsdk.EmailDomainLifecycleStateDeleted),
				}, nil
			}
		},
		deleteFn: func(_ context.Context, req emailsdk.DeleteEmailDomainRequest) (emailsdk.DeleteEmailDomainResponse, error) {
			deleteCalls++
			if req.EmailDomainId == nil || *req.EmailDomainId != existingID {
				t.Fatalf("DeleteEmailDomainRequest.EmailDomainId = %v, want %q", req.EmailDomainId, existingID)
			}
			return emailsdk.DeleteEmailDomainResponse{}, nil
		},
	})

	deleted, err := manager.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() first call error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() first call = true, want delete confirmation requeue while DELETING")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteEmailDomain() calls after first delete = %d, want 1", deleteCalls)
	}
	if got := resource.Status.LifecycleState; got != "DELETING" {
		t.Fatalf("status.lifecycleState after first delete = %q, want DELETING", got)
	}

	deleted, err = manager.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() second call error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() second call = false, want terminal delete confirmation")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteEmailDomain() calls after second delete = %d, want 1", deleteCalls)
	}
	if getCalls != 3 {
		t.Fatalf("GetEmailDomain() calls = %d, want 3", getCalls)
	}
}

/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package steeringpolicyattachment

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	dnssdk "github.com/oracle/oci-go-sdk/v65/dns"
	dnsv1beta1 "github.com/oracle/oci-service-operator/api/dns/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const mockSteeringPolicyAttachmentID = "ocid1.steeringpolicyattachment.oc1..mock"

// Contract evidence: recorded attachment CRUD, parent-policy resolution, resource-local delete confirmation, OCI SDK, and pinned provider.
func TestMockIntegrationSteeringPolicyAttachmentLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &dnsv1beta1.SteeringPolicyAttachment{ObjectMeta: metav1.ObjectMeta{Name: "mock-steering-policy-attachment", Namespace: "default", UID: types.UID("mock-steering-policy-attachment-uid")}, Spec: dnsv1beta1.SteeringPolicyAttachmentSpec{
		SteeringPolicyId: "ocid1.steeringpolicy.oc1..mock", ZoneId: "ocid1.dns-zone.oc1..mock", DomainName: "app.mock.invalid", DisplayName: "mock-steering-policy-attachment",
	}}
	responder, err := newSteeringPolicyAttachmentMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://dns.mock.invalid", BasePath: "20180115", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close SteeringPolicyAttachment OCI mock: %v", err)
		}
	})
	manager := &SteeringPolicyAttachmentServiceManager{}
	sdkClient := dnssdk.DnsClient{BaseClient: session.BaseClient()}
	hooks := newSteeringPolicyAttachmentDefaultRuntimeHooks(sdkClient)
	applySteeringPolicyAttachmentRuntimeHooks(&hooks, sdkClient, nil)
	client := wrapSteeringPolicyAttachmentGeneratedClient(hooks, defaultSteeringPolicyAttachmentServiceClient{ServiceClient: generatedruntime.NewServiceClient[*dnsv1beta1.SteeringPolicyAttachment](buildSteeringPolicyAttachmentGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*dnsv1beta1.SteeringPolicyAttachment]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *dnsv1beta1.SteeringPolicyAttachment) error {
			if current.Status.Id != mockSteeringPolicyAttachmentID || current.Status.DisplayName != resource.Spec.DisplayName || current.Status.LifecycleState != string(dnssdk.SteeringPolicyAttachmentLifecycleStateActive) {
				return fmt.Errorf("created SteeringPolicyAttachment status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *dnsv1beta1.SteeringPolicyAttachment) {
			current.Spec.DisplayName = "mock-steering-policy-attachment-updated"
		},
		ValidateUpdated: func(current *dnsv1beta1.SteeringPolicyAttachment) error {
			if current.Status.DisplayName != current.Spec.DisplayName {
				return fmt.Errorf("updated SteeringPolicyAttachment status = %+v", current.Status)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
}

type steeringPolicyAttachmentMockResponder struct {
	crud       *ocimock.CRUDResponder[dnssdk.SteeringPolicyAttachment]
	resource   *dnsv1beta1.SteeringPolicyAttachment
	parentRead bool
}

func newSteeringPolicyAttachmentMockResponder(resource *dnsv1beta1.SteeringPolicyAttachment) (*steeringPolicyAttachmentMockResponder, error) {
	now := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	crud, err := ocimock.NewCRUDResponder(ocimock.CRUDOptions[dnssdk.SteeringPolicyAttachment]{
		CollectionPath: "/20180115/steeringPolicyAttachments", ItemPath: "/20180115/steeringPolicyAttachments/" + mockSteeringPolicyAttachmentID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true, RequireUpdateRead: true, RequireDeleteRead: true,
		List: func(request ocimock.Request, present bool, state dnssdk.SteeringPolicyAttachment) (ocimock.Response, error) {
			query := request.URL.Query()
			if query.Get("steeringPolicyId") != resource.Spec.SteeringPolicyId || query.Get("zoneId") != resource.Spec.ZoneId || query.Get("domain") != resource.Spec.DomainName {
				return ocimock.Response{}, fmt.Errorf("unexpected ListSteeringPolicyAttachments query: %s", request.URL.RawQuery)
			}
			if !present {
				return ocimock.JSONResponse(http.StatusOK, []dnssdk.SteeringPolicyAttachment{})
			}
			return ocimock.JSONResponse(http.StatusOK, []dnssdk.SteeringPolicyAttachment{state})
		},
		Create: func(request ocimock.Request) (dnssdk.SteeringPolicyAttachment, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero dnssdk.SteeringPolicyAttachment
				return zero, ocimock.Response{}, err
			}
			var details dnssdk.CreateSteeringPolicyAttachmentDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return dnssdk.SteeringPolicyAttachment{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return dnssdk.SteeringPolicyAttachment{}, ocimock.Response{}, err
			}
			if details.SteeringPolicyId == nil || *details.SteeringPolicyId != resource.Spec.SteeringPolicyId || details.ZoneId == nil || *details.ZoneId != resource.Spec.ZoneId || details.DomainName == nil || *details.DomainName != resource.Spec.DomainName {
				return dnssdk.SteeringPolicyAttachment{}, ocimock.Response{}, fmt.Errorf("unexpected CreateSteeringPolicyAttachment details: %+v", details)
			}
			state := dnssdk.SteeringPolicyAttachment{SteeringPolicyId: details.SteeringPolicyId, ZoneId: details.ZoneId, DomainName: details.DomainName,
				DisplayName: details.DisplayName, Rtypes: []string{"A"}, CompartmentId: common.String("ocid1.compartment.oc1..mock"),
				Self: common.String("https://dns.mock.invalid/steeringPolicyAttachments/" + mockSteeringPolicyAttachmentID), Id: common.String(mockSteeringPolicyAttachmentID),
				TimeCreated: &now, LifecycleState: dnssdk.SteeringPolicyAttachmentLifecycleStateActive}
			response, err := ocimock.JSONResponse(http.StatusCreated, state)
			return state, response, err
		},
		Read: func(_ ocimock.Request, state dnssdk.SteeringPolicyAttachment) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		Update: func(request ocimock.Request, state dnssdk.SteeringPolicyAttachment) (dnssdk.SteeringPolicyAttachment, ocimock.Response, error) {
			var details dnssdk.UpdateSteeringPolicyAttachmentDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return dnssdk.SteeringPolicyAttachment{}, ocimock.Response{}, err
			}
			if details.DisplayName == nil || *details.DisplayName != "mock-steering-policy-attachment-updated" {
				return dnssdk.SteeringPolicyAttachment{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateSteeringPolicyAttachment details: %+v", details)
			}
			state.DisplayName = details.DisplayName
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Delete: func(_ ocimock.Request, _ dnssdk.SteeringPolicyAttachment) (ocimock.Response, error) {
			return ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
	if err != nil {
		return nil, err
	}
	return &steeringPolicyAttachmentMockResponder{crud: crud, resource: resource}, nil
}

func (r *steeringPolicyAttachmentMockResponder) Respond(request ocimock.Request) (ocimock.Response, error) {
	if request.Method == http.MethodGet && request.URL.Path == "/20180115/steeringPolicies/"+r.resource.Spec.SteeringPolicyId {
		r.parentRead = true
		return ocimock.JSONResponse(http.StatusOK, dnssdk.SteeringPolicy{Id: common.String(r.resource.Spec.SteeringPolicyId), CompartmentId: common.String("ocid1.compartment.oc1..mock"), DisplayName: common.String("mock-parent-policy"), Ttl: common.Int(30), Template: dnssdk.SteeringPolicyTemplateCustom, FreeformTags: map[string]string{}, DefinedTags: map[string]map[string]interface{}{}, Answers: []dnssdk.SteeringPolicyAnswer{}, Rules: []dnssdk.SteeringPolicyRule{}, Self: common.String("https://dns.mock.invalid/steeringPolicies/" + r.resource.Spec.SteeringPolicyId), TimeCreated: &common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}, LifecycleState: dnssdk.SteeringPolicyLifecycleStateActive})
	}
	return r.crud.Respond(request)
}

func (r *steeringPolicyAttachmentMockResponder) Verify() error {
	if err := r.crud.Verify(); err != nil {
		return err
	}
	if !r.parentRead {
		return fmt.Errorf("SteeringPolicyAttachment mock did not resolve its parent steering policy")
	}
	return nil
}

/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package steeringpolicy

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

const mockSteeringPolicyID = "ocid1.steeringpolicy.oc1..mock"

// Contract evidence: recorded steering-policy CRUD, resource-local polymorphic mapping, OCI SDK, and pinned provider.
func TestMockIntegrationSteeringPolicyLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &dnsv1beta1.SteeringPolicy{ObjectMeta: metav1.ObjectMeta{Name: "mock-steering-policy", Namespace: "default", UID: types.UID("mock-steering-policy-uid")}, Spec: dnsv1beta1.SteeringPolicySpec{
		CompartmentId: "ocid1.compartment.oc1..mock", DisplayName: "mock-steering-policy", Template: "CUSTOM", Ttl: 30,
		Answers: []dnsv1beta1.SteeringPolicyAnswer{{Name: "primary", Rtype: "A", Rdata: "192.0.2.10", Pool: "blue"}},
		Rules:   []dnsv1beta1.SteeringPolicyRule{{RuleType: "FILTER", DefaultAnswerData: []dnsv1beta1.SteeringPolicyRuleDefaultAnswerData{{AnswerCondition: "answer.isDisabled != true", ShouldKeep: true}}}, {RuleType: "LIMIT", DefaultCount: 1}},
	}}
	responder, err := newSteeringPolicyMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://dns.mock.invalid", BasePath: "20180115", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close SteeringPolicy OCI mock: %v", err)
		}
	})
	manager := &SteeringPolicyServiceManager{}
	hooks := newSteeringPolicyRuntimeHooks(manager, dnssdk.DnsClient{BaseClient: session.BaseClient()})
	client := wrapSteeringPolicyGeneratedClient(hooks, defaultSteeringPolicyServiceClient{ServiceClient: generatedruntime.NewServiceClient[*dnsv1beta1.SteeringPolicy](buildSteeringPolicyGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*dnsv1beta1.SteeringPolicy]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *dnsv1beta1.SteeringPolicy) error {
			if current.Status.Id != mockSteeringPolicyID || current.Status.DisplayName != resource.Spec.DisplayName || current.Status.Ttl != 30 || current.Status.LifecycleState != string(dnssdk.SteeringPolicyLifecycleStateActive) {
				return fmt.Errorf("created SteeringPolicy status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *dnsv1beta1.SteeringPolicy) {
			current.Spec.DisplayName = "mock-steering-policy-updated"
			current.Spec.Ttl = 60
		},
		ValidateUpdated: func(current *dnsv1beta1.SteeringPolicy) error {
			if current.Status.DisplayName != current.Spec.DisplayName || current.Status.Ttl != 60 {
				return fmt.Errorf("updated SteeringPolicy status = %+v", current.Status)
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

func newSteeringPolicyMockResponder(resource *dnsv1beta1.SteeringPolicy) (*ocimock.CRUDResponder[dnssdk.SteeringPolicy], error) {
	now := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	updateRead, deleteRead := false, false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[dnssdk.SteeringPolicy]{
		CollectionPath: "/20180115/steeringPolicies", ItemPath: "/20180115/steeringPolicies/" + mockSteeringPolicyID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true, RequireUpdateRead: true, RequireDeleteRead: true, RetainStateAfterDelete: true,
		List: func(_ ocimock.Request, present bool, state dnssdk.SteeringPolicy) (ocimock.Response, error) {
			if !present || (state.LifecycleState == dnssdk.SteeringPolicyLifecycleStateDeleting && deleteRead) {
				return ocimock.JSONResponse(http.StatusOK, []dnssdk.SteeringPolicy{})
			}
			return ocimock.JSONResponse(http.StatusOK, []dnssdk.SteeringPolicy{state})
		},
		Create: func(request ocimock.Request) (dnssdk.SteeringPolicy, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero dnssdk.SteeringPolicy
				return zero, ocimock.Response{}, err
			}
			var details dnssdk.CreateSteeringPolicyDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return dnssdk.SteeringPolicy{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return dnssdk.SteeringPolicy{}, ocimock.Response{}, err
			}
			if details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId || details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName || details.Template != dnssdk.CreateSteeringPolicyDetailsTemplateCustom || details.Ttl == nil || *details.Ttl != 30 || len(details.Answers) != 1 || len(details.Rules) != 2 {
				return dnssdk.SteeringPolicy{}, ocimock.Response{}, fmt.Errorf("unexpected CreateSteeringPolicy details: %+v", details)
			}
			state := dnssdk.SteeringPolicy{CompartmentId: details.CompartmentId, DisplayName: details.DisplayName, Ttl: details.Ttl,
				Template: dnssdk.SteeringPolicyTemplateCustom, FreeformTags: details.FreeformTags, DefinedTags: map[string]map[string]interface{}{},
				Answers: details.Answers, Rules: details.Rules, Self: common.String("https://dns.mock.invalid/steeringPolicies/" + mockSteeringPolicyID),
				Id: common.String(mockSteeringPolicyID), TimeCreated: &now, LifecycleState: dnssdk.SteeringPolicyLifecycleStateActive}
			response, err := ocimock.JSONResponse(http.StatusCreated, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state dnssdk.SteeringPolicy) (dnssdk.SteeringPolicy, ocimock.Response, error) {
			switch state.LifecycleState {
			case dnssdk.SteeringPolicyLifecycleStateEnum("UPDATING"):
				if updateRead {
					state.LifecycleState = dnssdk.SteeringPolicyLifecycleStateActive
				} else {
					updateRead = true
				}
			case dnssdk.SteeringPolicyLifecycleStateDeleting:
				if deleteRead {
					response, err := ocimock.JSONResponse(http.StatusNotFound, map[string]string{"code": "NotAuthorizedOrNotFound", "message": "resource not found"})
					return state, response, err
				}
				deleteRead = true
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state dnssdk.SteeringPolicy) (dnssdk.SteeringPolicy, ocimock.Response, error) {
			var details dnssdk.UpdateSteeringPolicyDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return dnssdk.SteeringPolicy{}, ocimock.Response{}, err
			}
			if details.DisplayName == nil || *details.DisplayName != "mock-steering-policy-updated" || details.Ttl == nil || *details.Ttl != 60 || len(details.Answers) != 1 || len(details.Rules) != 2 {
				return dnssdk.SteeringPolicy{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateSteeringPolicy details: %+v", details)
			}
			state.DisplayName, state.Ttl, state.Answers, state.Rules, state.LifecycleState = details.DisplayName, details.Ttl, details.Answers, details.Rules, dnssdk.SteeringPolicyLifecycleStateEnum("UPDATING")
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state dnssdk.SteeringPolicy) (dnssdk.SteeringPolicy, ocimock.Response, error) {
			state.LifecycleState = dnssdk.SteeringPolicyLifecycleStateDeleting
			return state, ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}

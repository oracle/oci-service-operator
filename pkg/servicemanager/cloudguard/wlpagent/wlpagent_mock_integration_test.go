/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package wlpagent

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	cloudguardsdk "github.com/oracle/oci-go-sdk/v65/cloudguard"
	"github.com/oracle/oci-go-sdk/v65/common"
	cloudguardv1beta1 "github.com/oracle/oci-service-operator/api/cloudguard/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockWlpAgentID = "ocid1.wlpagent.oc1..mock"

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - formal provider facts: formal/imports/cloudguard/wlpagent.json
//   - repo-authored runtime: formal/controllers/cloudguard/wlpagent/diagrams/runtime-lifecycle.yaml
//   - custom state-free runtime: wlpagent_runtime_client.go
//   - Terraform provider: terraform-provider-oci@eb653febb1ba internal/service/cloud_guard/cloud_guard_wlp_agent_resource.go
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/cloudguard
func TestMockIntegrationWlpAgentSynchronousCRUD(t *testing.T) {
	t.Parallel()

	resource := &cloudguardv1beta1.WlpAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-wlp-agent", Namespace: "default", UID: types.UID("mock-wlp-agent-uid")},
		Spec: cloudguardv1beta1.WlpAgentSpec{
			CompartmentId:            "ocid1.compartment.oc1..mock",
			AgentVersion:             "1.0.0",
			CertificateSignedRequest: "mock-csr",
			OsInfo:                   "linux_amd64",
			FreeformTags:             map[string]string{"osok-mock": "create"},
		},
	}
	responder, err := newWlpAgentMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://cloudguard.mock.invalid", BasePath: "20200131", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close WlpAgent OCI mock: %v", err)
		}
	})

	sdkClient := cloudguardsdk.CloudGuardClient{BaseClient: session.BaseClient()}
	manager := &WlpAgentServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newWlpAgentRuntimeHooks(manager, sdkClient)
	client := wrapWlpAgentGeneratedClient(hooks, defaultWlpAgentServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*cloudguardv1beta1.WlpAgent](buildWlpAgentGeneratedRuntimeConfig(manager, hooks)),
	})

	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*cloudguardv1beta1.WlpAgent]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *cloudguardv1beta1.WlpAgent) error {
			if current.Status.Id != mockWlpAgentID ||
				current.Status.AgentVersion != resource.Spec.AgentVersion ||
				current.Status.CertificateId != "ocid1.certificate.oc1..mock" ||
				current.Status.FreeformTags["osok-mock"] != "create" {
				return fmt.Errorf("created WlpAgent status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *cloudguardv1beta1.WlpAgent) {
			// certificateSignedRequest is mandatory in the SDK update body even
			// when only tags drift.
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *cloudguardv1beta1.WlpAgent) error {
			if current.Status.CertificateSignedRequest != current.Spec.CertificateSignedRequest ||
				current.Status.FreeformTags["osok-mock"] != "update" {
				return fmt.Errorf("updated WlpAgent status = %+v", current.Status)
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

func newWlpAgentMockResponder(resource *cloudguardv1beta1.WlpAgent) (*ocimock.CRUDResponder[cloudguardsdk.WlpAgent], error) {
	createdAt := common.SDKTime{Time: time.Date(2026, time.September, 4, 13, 0, 0, 0, time.UTC)}
	updatedAt := common.SDKTime{Time: createdAt.Time.Add(time.Minute)}
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[cloudguardsdk.WlpAgent]{
		CollectionPath:     "/20200131/wlpAgents",
		ItemPath:           "/20200131/wlpAgents/" + mockWlpAgentID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		List: func(request ocimock.Request, present bool, state cloudguardsdk.WlpAgent) (ocimock.Response, error) {
			if got := request.URL.Query().Get("compartmentId"); got != resource.Spec.CompartmentId {
				return ocimock.Response{}, fmt.Errorf("list compartmentId = %q", got)
			}
			items := []cloudguardsdk.WlpAgentSummary{}
			if present {
				items = append(items, wlpAgentSummaryFromMockState(state))
			}
			return ocimock.JSONResponse(http.StatusOK, cloudguardsdk.WlpAgentCollection{Items: items})
		},
		Create: func(request ocimock.Request) (cloudguardsdk.WlpAgent, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero cloudguardsdk.WlpAgent
				return zero, ocimock.Response{}, err
			}
			var details cloudguardsdk.CreateWlpAgentDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return cloudguardsdk.WlpAgent{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return cloudguardsdk.WlpAgent{}, ocimock.Response{}, err
			}
			expected := cloudguardsdk.CreateWlpAgentDetails{
				CompartmentId:            common.String(resource.Spec.CompartmentId),
				AgentVersion:             common.String(resource.Spec.AgentVersion),
				CertificateSignedRequest: common.String(resource.Spec.CertificateSignedRequest),
				OsInfo:                   common.String(resource.Spec.OsInfo),
				FreeformTags:             map[string]string{"osok-mock": "create"},
			}
			if !reflect.DeepEqual(details, expected) {
				return cloudguardsdk.WlpAgent{}, ocimock.Response{}, fmt.Errorf("create WlpAgent details = %+v, want %+v", details, expected)
			}
			state := cloudguardsdk.WlpAgent{
				Id:                       common.String(mockWlpAgentID),
				CompartmentId:            details.CompartmentId,
				AgentVersion:             details.AgentVersion,
				CertificateId:            common.String("ocid1.certificate.oc1..mock"),
				CertificateSignedRequest: details.CertificateSignedRequest,
				HostId:                   common.String("ocid1.instance.oc1..mock"),
				TenantId:                 common.String("ocid1.tenancy.oc1..mock"),
				TimeCreated:              &createdAt,
				TimeUpdated:              &createdAt,
				FreeformTags:             details.FreeformTags,
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Read: func(_ ocimock.Request, state cloudguardsdk.WlpAgent) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		Update: func(request ocimock.Request, state cloudguardsdk.WlpAgent) (cloudguardsdk.WlpAgent, ocimock.Response, error) {
			var details cloudguardsdk.UpdateWlpAgentDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return cloudguardsdk.WlpAgent{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return cloudguardsdk.WlpAgent{}, ocimock.Response{}, err
			}
			expected := cloudguardsdk.UpdateWlpAgentDetails{
				CertificateSignedRequest: common.String(resource.Spec.CertificateSignedRequest),
				FreeformTags:             map[string]string{"osok-mock": "update"},
			}
			if !reflect.DeepEqual(details, expected) {
				return cloudguardsdk.WlpAgent{}, ocimock.Response{}, fmt.Errorf("update WlpAgent details = %+v, want %+v", details, expected)
			}
			state.CertificateSignedRequest = details.CertificateSignedRequest
			state.FreeformTags = details.FreeformTags
			state.TimeUpdated = &updatedAt
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Delete: func(request ocimock.Request, _ cloudguardsdk.WlpAgent) (ocimock.Response, error) {
			if len(request.Body) != 0 {
				return ocimock.Response{}, fmt.Errorf("delete WlpAgent body = %s", request.Body)
			}
			return ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}

func wlpAgentSummaryFromMockState(state cloudguardsdk.WlpAgent) cloudguardsdk.WlpAgentSummary {
	return cloudguardsdk.WlpAgentSummary{
		Id:            state.Id,
		CompartmentId: state.CompartmentId,
		AgentVersion:  state.AgentVersion,
		CertificateId: state.CertificateId,
		HostId:        state.HostId,
		TenantId:      state.TenantId,
		TimeCreated:   state.TimeCreated,
		TimeUpdated:   state.TimeUpdated,
		FreeformTags:  state.FreeformTags,
		DefinedTags:   state.DefinedTags,
		SystemTags:    state.SystemTags,
	}
}

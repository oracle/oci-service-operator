/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package softwaresource

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	osmanagementhubsdk "github.com/oracle/oci-go-sdk/v65/osmanagementhub"
	osmanagementhubv1beta1 "github.com/oracle/oci-service-operator/api/osmanagementhub/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

const mockSoftwareSourceID = "ocid1.osmhsoftwaresource.oc1..mock"

// Contract evidence: recorded private software-source CRUD, resource-local semantics, OCI SDK, and pinned provider.
func TestMockIntegrationSoftwareSourceLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &osmanagementhubv1beta1.SoftwareSource{Spec: osmanagementhubv1beta1.SoftwareSourceSpec{
		CompartmentId: "ocid1.compartment.oc1..mock", DisplayName: "mock-private-source", Description: "mock create", SoftwareSourceType: string(osmanagementhubsdk.SoftwareSourceTypePrivate),
		Url: "https://yum.oracle.com/repo/OracleLinux/OL8/baseos/latest/x86_64/", OsFamily: string(osmanagementhubsdk.OsFamilyOracleLinux8), ArchType: string(osmanagementhubsdk.ArchTypeX8664),
		IsGpgCheckEnabled: false, IsSslVerifyEnabled: true, FreeformTags: map[string]string{"mock": "create"},
	}}
	ocimock.InitializeResource(resource, "mock-software-source")
	responder, err := newSoftwareSourceMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://osmh.mock.invalid", BasePath: "20220901", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close SoftwareSource OCI mock: %v", err)
		}
	})
	client := newSoftwareSourceServiceClientWithOCIClient(osmanagementhubsdk.SoftwareSourceClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*osmanagementhubv1beta1.SoftwareSource]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *osmanagementhubv1beta1.SoftwareSource) error {
			if current.Status.Id != mockSoftwareSourceID || current.Status.DisplayName != resource.Spec.DisplayName {
				return fmt.Errorf("created SoftwareSource status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *osmanagementhubv1beta1.SoftwareSource) { current.Spec.Description = "mock update" },
		ValidateUpdated: func(current *osmanagementhubv1beta1.SoftwareSource) error {
			if current.Status.Description != current.Spec.Description {
				return fmt.Errorf("updated SoftwareSource status = %+v", current.Status)
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

func newSoftwareSourceMockResponder(resource *osmanagementhubv1beta1.SoftwareSource) (*ocimock.CRUDResponder[osmanagementhubsdk.PrivateSoftwareSource], error) {
	now := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[osmanagementhubsdk.PrivateSoftwareSource]{
		CollectionPath: "/20220901/softwareSources", ItemPath: "/20220901/softwareSources/" + mockSoftwareSourceID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete}, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true,
		List: func(_ ocimock.Request, present bool, state osmanagementhubsdk.PrivateSoftwareSource) (ocimock.Response, error) {
			if !present {
				return ocimock.JSONResponse(http.StatusOK, map[string]any{"items": []osmanagementhubsdk.PrivateSoftwareSource{}})
			}
			return ocimock.JSONResponse(http.StatusOK, map[string]any{"items": []osmanagementhubsdk.PrivateSoftwareSource{state}})
		},
		Create: func(request ocimock.Request) (osmanagementhubsdk.PrivateSoftwareSource, ocimock.Response, error) {
			if got := request.Header.Get("opc-retry-token"); got != string(resource.UID) {
				return osmanagementhubsdk.PrivateSoftwareSource{}, ocimock.Response{}, fmt.Errorf("CreateSoftwareSource retry token = %q, want resource UID %q", got, resource.UID)
			}
			type createDetails osmanagementhubsdk.CreatePrivateSoftwareSourceDetails
			var payload struct {
				createDetails
				SoftwareSourceType string `json:"softwareSourceType"`
			}
			if err := ocimock.DecodeJSONRequest(request, &payload); err != nil {
				return osmanagementhubsdk.PrivateSoftwareSource{}, ocimock.Response{}, err
			}
			details := osmanagementhubsdk.CreatePrivateSoftwareSourceDetails(payload.createDetails)
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return osmanagementhubsdk.PrivateSoftwareSource{}, ocimock.Response{}, err
			}
			if details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName || details.Url == nil || *details.Url != resource.Spec.Url {
				return osmanagementhubsdk.PrivateSoftwareSource{}, ocimock.Response{}, fmt.Errorf("unexpected CreatePrivateSoftwareSource details: %+v", details)
			}
			state := osmanagementhubsdk.PrivateSoftwareSource{Id: common.String(mockSoftwareSourceID), CompartmentId: details.CompartmentId, DisplayName: details.DisplayName, Description: details.Description, TimeCreated: &now,
				RepoId: common.String("private.mock-private-source"), Url: details.Url, FreeformTags: details.FreeformTags, DefinedTags: details.DefinedTags, IsGpgCheckEnabled: details.IsGpgCheckEnabled, IsSslVerifyEnabled: details.IsSslVerifyEnabled,
				IsMirrorSyncAllowed: details.IsMirrorSyncAllowed, Availability: osmanagementhubsdk.AvailabilitySelected, AvailabilityAtOci: osmanagementhubsdk.AvailabilitySelected, OsFamily: details.OsFamily, ArchType: details.ArchType, LifecycleState: osmanagementhubsdk.SoftwareSourceLifecycleStateActive}
			response, err := ocimock.JSONResponse(http.StatusCreated, state)
			return state, response, err
		},
		Read: func(_ ocimock.Request, state osmanagementhubsdk.PrivateSoftwareSource) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		Update: func(request ocimock.Request, state osmanagementhubsdk.PrivateSoftwareSource) (osmanagementhubsdk.PrivateSoftwareSource, ocimock.Response, error) {
			type updateDetails osmanagementhubsdk.UpdatePrivateSoftwareSourceDetails
			var payload struct {
				updateDetails
				SoftwareSourceType string `json:"softwareSourceType"`
			}
			if err := ocimock.DecodeJSONRequest(request, &payload); err != nil {
				return state, ocimock.Response{}, err
			}
			details := osmanagementhubsdk.UpdatePrivateSoftwareSourceDetails(payload.updateDetails)
			if details.Description == nil || *details.Description != "mock update" {
				return state, ocimock.Response{}, fmt.Errorf("unexpected UpdatePrivateSoftwareSource details: %+v", details)
			}
			state.DisplayName, state.Description, state.Url, state.FreeformTags = details.DisplayName, details.Description, details.Url, details.FreeformTags
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Delete: func(_ ocimock.Request, _ osmanagementhubsdk.PrivateSoftwareSource) (ocimock.Response, error) {
			return ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}

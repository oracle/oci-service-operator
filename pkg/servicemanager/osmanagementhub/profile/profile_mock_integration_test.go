/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package profile

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	mockProfileID       = "ocid1.osmhprofile.oc1..mock"
	mockProfileSourceID = "ocid1.osmhsoftwaresource.oc1..mock"
)

// Contract evidence: recorded profile CRUD, resource-local semantics, OCI SDK, and pinned provider.
func TestMockIntegrationProfileLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &osmanagementhubv1beta1.Profile{ObjectMeta: metav1.ObjectMeta{Name: "mock-profile", Namespace: "default", UID: types.UID("mock-profile-uid")}, Spec: osmanagementhubv1beta1.ProfileSpec{
		DisplayName: "mock-profile", CompartmentId: "ocid1.compartment.oc1..mock", Description: "mock create",
		ProfileType: string(osmanagementhubsdk.ProfileTypeSoftwaresource), RegistrationType: string(osmanagementhubsdk.ProfileRegistrationTypeOciLinux),
		VendorName: string(osmanagementhubsdk.VendorNameOracle), OsFamily: string(osmanagementhubsdk.OsFamilyOracleLinux8), ArchType: string(osmanagementhubsdk.ArchTypeX8664),
		SoftwareSourceIds: []string{mockProfileSourceID}, FreeformTags: map[string]string{"mock": "create"},
	}}
	responder, err := newProfileMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://osmh.mock.invalid", BasePath: "20220901", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Profile OCI mock: %v", err)
		}
	})
	client := newProfileServiceClientWithOCIClient(osmanagementhubsdk.OnboardingClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*osmanagementhubv1beta1.Profile]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *osmanagementhubv1beta1.Profile) error {
			if current.Status.Id != mockProfileID || current.Status.DisplayName != resource.Spec.DisplayName || current.Status.LifecycleState != string(osmanagementhubsdk.ProfileLifecycleStateActive) {
				return fmt.Errorf("created Profile status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *osmanagementhubv1beta1.Profile) {
			current.Spec.DisplayName = "mock-profile-updated"
			current.Spec.Description = "mock update"
		},
		ValidateUpdated: func(current *osmanagementhubv1beta1.Profile) error {
			if current.Status.DisplayName != current.Spec.DisplayName || current.Status.Description != current.Spec.Description {
				return fmt.Errorf("updated Profile status = %+v", current.Status)
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

func newProfileMockResponder(resource *osmanagementhubv1beta1.Profile) (*ocimock.CRUDResponder[osmanagementhubsdk.SoftwareSourceProfile], error) {
	now := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[osmanagementhubsdk.SoftwareSourceProfile]{
		CollectionPath: "/20220901/profiles", ItemPath: "/20220901/profiles/" + mockProfileID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete}, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true,
		List: func(_ ocimock.Request, present bool, state osmanagementhubsdk.SoftwareSourceProfile) (ocimock.Response, error) {
			if !present {
				return ocimock.JSONResponse(http.StatusOK, map[string]any{"items": []osmanagementhubsdk.SoftwareSourceProfile{}})
			}
			return ocimock.JSONResponse(http.StatusOK, map[string]any{"items": []osmanagementhubsdk.SoftwareSourceProfile{state}})
		},
		Create: func(request ocimock.Request) (osmanagementhubsdk.SoftwareSourceProfile, ocimock.Response, error) {
			if err := ocimock.ValidateRetryTokenValue(request, profileRetryToken(resource, resource.Namespace)); err != nil {
				var zero osmanagementhubsdk.SoftwareSourceProfile
				return zero, ocimock.Response{}, err
			}
			type createDetails osmanagementhubsdk.CreateSoftwareSourceProfileDetails
			var payload struct {
				createDetails
				ProfileType string `json:"profileType"`
			}
			if err := ocimock.DecodeJSONRequest(request, &payload); err != nil {
				return osmanagementhubsdk.SoftwareSourceProfile{}, ocimock.Response{}, err
			}
			details := osmanagementhubsdk.CreateSoftwareSourceProfileDetails(payload.createDetails)
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return osmanagementhubsdk.SoftwareSourceProfile{}, ocimock.Response{}, err
			}
			if details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName || len(details.SoftwareSourceIds) != 1 || details.SoftwareSourceIds[0] != mockProfileSourceID {
				return osmanagementhubsdk.SoftwareSourceProfile{}, ocimock.Response{}, fmt.Errorf("unexpected CreateSoftwareSourceProfile details: %+v", details)
			}
			state := osmanagementhubsdk.SoftwareSourceProfile{Id: common.String(mockProfileID), CompartmentId: details.CompartmentId, DisplayName: details.DisplayName, Description: details.Description,
				SoftwareSources: []osmanagementhubsdk.SoftwareSourceDetails{{Id: common.String(mockProfileSourceID), DisplayName: common.String("mock-source"), SoftwareSourceType: osmanagementhubsdk.SoftwareSourceTypeVendor}},
				VendorName:      details.VendorName, OsFamily: details.OsFamily, ArchType: details.ArchType, RegistrationType: details.RegistrationType, LifecycleState: osmanagementhubsdk.ProfileLifecycleStateActive,
				IsDefaultProfile: details.IsDefaultProfile, IsServiceProvidedProfile: common.Bool(false), TimeCreated: &now, ProfileVersion: common.String("1"), FreeformTags: details.FreeformTags, DefinedTags: details.DefinedTags}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Read: func(_ ocimock.Request, state osmanagementhubsdk.SoftwareSourceProfile) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		Update: func(request ocimock.Request, state osmanagementhubsdk.SoftwareSourceProfile) (osmanagementhubsdk.SoftwareSourceProfile, ocimock.Response, error) {
			var details osmanagementhubsdk.UpdateProfileDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return state, ocimock.Response{}, err
			}
			if details.DisplayName == nil || *details.DisplayName != "mock-profile-updated" || details.Description == nil || *details.Description != "mock update" {
				return state, ocimock.Response{}, fmt.Errorf("unexpected UpdateProfile details: %+v", details)
			}
			state.DisplayName, state.Description, state.ProfileVersion, state.TimeModified = details.DisplayName, details.Description, common.String("2"), &now
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Delete: func(_ ocimock.Request, _ osmanagementhubsdk.SoftwareSourceProfile) (ocimock.Response, error) {
			return ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}

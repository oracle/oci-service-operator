/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package sensitivetype

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockSensitiveTypeID = "ocid1.sensitivetype.oc1..mock"

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - formal provider facts: formal/imports/datasafe/sensitivetype.json
//   - repo-authored runtime: formal/controllers/datasafe/sensitivetype/diagrams/runtime-lifecycle.yaml
//   - polymorphic request runtime: sensitivetype_runtime_client.go
//   - Terraform provider: terraform-provider-oci@eb653febb1ba internal/service/data_safe/data_safe_sensitive_type_resource.go
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/datasafe
func TestMockIntegrationSensitiveTypeLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &datasafev1beta1.SensitiveType{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-sensitive-type", Namespace: "default", UID: types.UID("mock-sensitive-type-uid")},
		Spec: datasafev1beta1.SensitiveTypeSpec{
			CompartmentId: "ocid1.compartment.oc1..mock",
			EntityType:    "SENSITIVE_TYPE",
			DisplayName:   "mock-sensitive-type",
			ShortName:     "MOCK_SENSITIVE_TYPE",
			Description:   "mock create",
			NamePattern:   "(?i).*mock.*",
			SearchType:    "OR",
			FreeformTags:  map[string]string{"osok-mock": "create"},
		},
	}
	responder, err := newSensitiveTypeMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://datasafe.mock.invalid", BasePath: "20181201", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close SensitiveType OCI mock: %v", err)
		}
	})

	sdkClient := datasafesdk.DataSafeClient{BaseClient: session.BaseClient()}
	manager := &SensitiveTypeServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newSensitiveTypeRuntimeHooks(manager, sdkClient)
	client := wrapSensitiveTypeGeneratedClient(hooks, defaultSensitiveTypeServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*datasafev1beta1.SensitiveType](buildSensitiveTypeGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*datasafev1beta1.SensitiveType]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *datasafev1beta1.SensitiveType) error {
			if current.Status.Id != mockSensitiveTypeID ||
				current.Status.DisplayName != "mock-sensitive-type" ||
				current.Status.EntityType != "SENSITIVE_TYPE" ||
				current.Status.LifecycleState != string(datasafesdk.DiscoveryLifecycleStateActive) ||
				current.Status.NamePattern != "(?i).*mock.*" {
				return fmt.Errorf("created SensitiveType status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *datasafev1beta1.SensitiveType) {
			current.Spec.Description = "mock update"
			current.Spec.CommentPattern = "(?i).*mock comment.*"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *datasafev1beta1.SensitiveType) error {
			if current.Status.Description != current.Spec.Description ||
				current.Status.CommentPattern != current.Spec.CommentPattern ||
				current.Status.FreeformTags["osok-mock"] != "update" ||
				current.Status.LifecycleState != string(datasafesdk.DiscoveryLifecycleStateActive) {
				return fmt.Errorf("updated SensitiveType status = %+v", current.Status)
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

func newSensitiveTypeMockResponder(resource *datasafev1beta1.SensitiveType) (*ocimock.CRUDResponder[datasafesdk.SensitiveTypePattern], error) {
	createdAt := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	updatedAt := common.SDKTime{Time: createdAt.Time.Add(time.Minute)}
	deleteRead := false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[datasafesdk.SensitiveTypePattern]{
		CollectionPath:         "/20181201/sensitiveTypes",
		ItemPath:               "/20181201/sensitiveTypes/" + mockSensitiveTypeID,
		ExpectedOperations:     []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:      true,
		RequireUpdateRead:      true,
		RequireDeleteRead:      true,
		RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (datasafesdk.SensitiveTypePattern, ocimock.Response, error) {
			var details datasafesdk.CreateSensitiveTypePatternDetails
			if err := ocimock.DecodeDiscriminatedJSONRequest(request, &details, "entityType", "SENSITIVE_TYPE"); err != nil {
				return datasafesdk.SensitiveTypePattern{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return datasafesdk.SensitiveTypePattern{}, ocimock.Response{}, err
			}
			expected := datasafesdk.CreateSensitiveTypePatternDetails{
				CompartmentId: common.String(resource.Spec.CompartmentId),
				DisplayName:   common.String(resource.Spec.DisplayName),
				ShortName:     common.String(resource.Spec.ShortName),
				Description:   common.String(resource.Spec.Description),
				NamePattern:   common.String(resource.Spec.NamePattern),
				SearchType:    datasafesdk.SensitiveTypePatternSearchTypeOr,
				FreeformTags:  map[string]string{"osok-mock": "create"},
			}
			if !reflect.DeepEqual(details, expected) {
				return datasafesdk.SensitiveTypePattern{}, ocimock.Response{}, fmt.Errorf("create SensitiveType details = %+v, want %+v", details, expected)
			}
			if request.Header.Get("opc-retry-token") == "" {
				return datasafesdk.SensitiveTypePattern{}, ocimock.Response{}, fmt.Errorf("create SensitiveType opc-retry-token is empty")
			}
			state := datasafesdk.SensitiveTypePattern{
				Id:                     common.String(mockSensitiveTypeID),
				DisplayName:            details.DisplayName,
				CompartmentId:          details.CompartmentId,
				TimeCreated:            &createdAt,
				TimeUpdated:            &createdAt,
				ShortName:              details.ShortName,
				Description:            details.Description,
				ParentCategoryId:       details.ParentCategoryId,
				IsCommon:               common.Bool(false),
				FreeformTags:           details.FreeformTags,
				DefinedTags:            details.DefinedTags,
				SystemTags:             map[string]map[string]interface{}{},
				NamePattern:            details.NamePattern,
				CommentPattern:         details.CommentPattern,
				DataPattern:            details.DataPattern,
				DefaultMaskingFormatId: details.DefaultMaskingFormatId,
				SearchType:             details.SearchType,
				LifecycleState:         datasafesdk.DiscoveryLifecycleStateCreating,
				Source:                 datasafesdk.SensitiveTypeSourceUser,
			}
			response, err := ocimock.JSONResponse(http.StatusCreated, state)
			response.Header = http.Header{"Opc-Work-Request-Id": []string{"ocid1.datasafeworkrequest.oc1..mock-create"}}
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state datasafesdk.SensitiveTypePattern) (datasafesdk.SensitiveTypePattern, ocimock.Response, error) {
			if state.LifecycleState == datasafesdk.DiscoveryLifecycleStateCreating || state.LifecycleState == datasafesdk.DiscoveryLifecycleStateUpdating {
				state.LifecycleState = datasafesdk.DiscoveryLifecycleStateActive
			} else if state.LifecycleState == datasafesdk.DiscoveryLifecycleStateDeleting {
				if deleteRead {
					state.LifecycleState = datasafesdk.DiscoveryLifecycleStateDeleted
				} else {
					deleteRead = true
				}
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state datasafesdk.SensitiveTypePattern) (datasafesdk.SensitiveTypePattern, ocimock.Response, error) {
			var details datasafesdk.UpdateSensitiveTypePatternDetails
			if err := ocimock.DecodeDiscriminatedJSONRequest(request, &details, "entityType", "SENSITIVE_TYPE"); err != nil {
				return datasafesdk.SensitiveTypePattern{}, ocimock.Response{}, err
			}
			expected := datasafesdk.UpdateSensitiveTypePatternDetails{
				DisplayName:    common.String("mock-sensitive-type"),
				ShortName:      common.String("MOCK_SENSITIVE_TYPE"),
				Description:    common.String("mock update"),
				NamePattern:    common.String("(?i).*mock.*"),
				CommentPattern: common.String("(?i).*mock comment.*"),
				SearchType:     datasafesdk.SensitiveTypePatternSearchTypeOr,
				FreeformTags:   map[string]string{"osok-mock": "update"},
			}
			if !reflect.DeepEqual(details, expected) {
				return datasafesdk.SensitiveTypePattern{}, ocimock.Response{}, fmt.Errorf("update SensitiveType details = %+v, want %+v", details, expected)
			}
			state.DisplayName = details.DisplayName
			state.ShortName = details.ShortName
			state.Description = details.Description
			state.ParentCategoryId = details.ParentCategoryId
			state.FreeformTags = details.FreeformTags
			state.DefinedTags = details.DefinedTags
			state.NamePattern = details.NamePattern
			state.CommentPattern = details.CommentPattern
			state.DataPattern = details.DataPattern
			state.DefaultMaskingFormatId = details.DefaultMaskingFormatId
			state.SearchType = details.SearchType
			state.LifecycleState = datasafesdk.DiscoveryLifecycleStateUpdating
			state.TimeUpdated = &updatedAt
			response := ocimock.EmptyResponse(http.StatusAccepted)
			response.Header = http.Header{"Opc-Work-Request-Id": []string{"ocid1.datasafeworkrequest.oc1..mock-update"}}
			return state, response, nil
		},
		DeleteTransition: func(request ocimock.Request, state datasafesdk.SensitiveTypePattern) (datasafesdk.SensitiveTypePattern, ocimock.Response, error) {
			if len(request.Body) != 0 {
				return datasafesdk.SensitiveTypePattern{}, ocimock.Response{}, fmt.Errorf("delete SensitiveType body = %s", request.Body)
			}
			state.LifecycleState = datasafesdk.DiscoveryLifecycleStateDeleting
			return state, ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}

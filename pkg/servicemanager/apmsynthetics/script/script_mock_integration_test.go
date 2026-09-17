/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package script

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	apmsyntheticssdk "github.com/oracle/oci-go-sdk/v65/apmsynthetics"
	"github.com/oracle/oci-go-sdk/v65/common"
	apmsyntheticsv1beta1 "github.com/oracle/oci-service-operator/api/apmsynthetics/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	mockScriptID          = "ocid1.apmsyntheticsscript.oc1..mock"
	mockScriptAPMDomainID = "ocid1.apmdomain.oc1..mock-script"
)

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - formal provider facts: formal/imports/apmsynthetics/script.json
//   - repo-authored runtime: formal/controllers/apmsynthetics/script/diagrams/runtime-lifecycle.yaml
//   - Terraform provider: terraform-provider-oci@eb653febb1ba internal/service/apm_synthetics/apm_synthetics_script_resource.go
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/apmsynthetics
func TestMockIntegrationScriptSynchronousCRUD(t *testing.T) {
	t.Parallel()

	resource := &apmsyntheticsv1beta1.Script{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-apm-script", Namespace: "default", UID: types.UID("mock-apm-script-uid")},
		Spec: apmsyntheticsv1beta1.ScriptSpec{
			DisplayName:     "mock-apm-script",
			ContentType:     "SIDE",
			Content:         mockSeleniumScript("OSOK mock"),
			ContentFileName: "mock.side",
			ApmDomainId:     mockScriptAPMDomainID,
			FreeformTags:    map[string]string{"osok-mock": "create"},
		},
	}
	responder, err := newScriptMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://apm-synthetic.mock.invalid", BasePath: "20200630", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Script OCI mock: %v", err)
		}
	})

	client := newScriptServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")},
		apmsyntheticssdk.ApmSyntheticClient{BaseClient: session.BaseClient()},
	)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apmsyntheticsv1beta1.Script]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apmsyntheticsv1beta1.Script) error {
			if current.Status.Id != mockScriptID ||
				current.Status.ApmDomainId != mockScriptAPMDomainID ||
				current.Status.DisplayName != "mock-apm-script" ||
				current.Status.ContentType != "SIDE" ||
				current.Status.ContentFileName != "mock.side" ||
				current.Status.FreeformTags["osok-mock"] != "create" {
				return fmt.Errorf("created Script status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apmsyntheticsv1beta1.Script) {
			current.Spec.DisplayName = "mock-apm-script-updated"
			current.Spec.Content = mockSeleniumScript("OSOK mock updated")
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *apmsyntheticsv1beta1.Script) error {
			if current.Status.DisplayName != current.Spec.DisplayName ||
				current.Status.FreeformTags["osok-mock"] != "update" ||
				current.Status.ContentSizeInBytes != len(current.Spec.Content) {
				return fmt.Errorf("updated Script status = %+v", current.Status)
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

func newScriptMockResponder(resource *apmsyntheticsv1beta1.Script) (*ocimock.CRUDResponder[apmsyntheticssdk.Script], error) {
	createdAt := common.SDKTime{Time: time.Date(2026, time.September, 4, 16, 0, 0, 0, time.UTC)}
	updatedAt := common.SDKTime{Time: createdAt.Time.Add(time.Minute)}
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[apmsyntheticssdk.Script]{
		CollectionPath:     "/20200630/scripts",
		ItemPath:           "/20200630/scripts/" + mockScriptID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		Create: func(request ocimock.Request) (apmsyntheticssdk.Script, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero apmsyntheticssdk.Script
				return zero, ocimock.Response{}, err
			}
			if err := validateScriptDomain(request); err != nil {
				return apmsyntheticssdk.Script{}, ocimock.Response{}, err
			}
			var details apmsyntheticssdk.CreateScriptDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return apmsyntheticssdk.Script{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return apmsyntheticssdk.Script{}, ocimock.Response{}, err
			}
			expected := apmsyntheticssdk.CreateScriptDetails{
				DisplayName:     common.String("mock-apm-script"),
				ContentType:     apmsyntheticssdk.ContentTypesSide,
				Content:         common.String(resource.Spec.Content),
				ContentFileName: common.String("mock.side"),
				FreeformTags:    map[string]string{"osok-mock": "create"},
			}
			if !reflect.DeepEqual(details, expected) {
				return apmsyntheticssdk.Script{}, ocimock.Response{}, fmt.Errorf("create Script details = %+v, want %+v", details, expected)
			}
			state := mockScriptState(details, createdAt)
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Read: func(request ocimock.Request, state apmsyntheticssdk.Script) (ocimock.Response, error) {
			if err := validateScriptDomain(request); err != nil {
				return ocimock.Response{}, err
			}
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		Update: func(request ocimock.Request, state apmsyntheticssdk.Script) (apmsyntheticssdk.Script, ocimock.Response, error) {
			if err := validateScriptDomain(request); err != nil {
				return apmsyntheticssdk.Script{}, ocimock.Response{}, err
			}
			var details apmsyntheticssdk.UpdateScriptDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return apmsyntheticssdk.Script{}, ocimock.Response{}, err
			}
			expected := apmsyntheticssdk.UpdateScriptDetails{
				DisplayName:  common.String("mock-apm-script-updated"),
				Content:      common.String(mockSeleniumScript("OSOK mock updated")),
				FreeformTags: map[string]string{"osok-mock": "update"},
			}
			if !reflect.DeepEqual(details, expected) {
				return apmsyntheticssdk.Script{}, ocimock.Response{}, fmt.Errorf("update Script details = %+v, want %+v", details, expected)
			}
			state.DisplayName = details.DisplayName
			state.Content = details.Content
			state.FreeformTags = details.FreeformTags
			state.TimeUpdated = &updatedAt
			state.TimeUploaded = &updatedAt
			state.ContentSizeInBytes = common.Int(len(*details.Content))
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Delete: func(request ocimock.Request, _ apmsyntheticssdk.Script) (ocimock.Response, error) {
			if err := validateScriptDomain(request); err != nil {
				return ocimock.Response{}, err
			}
			if len(request.Body) != 0 {
				return ocimock.Response{}, fmt.Errorf("delete Script body = %s", request.Body)
			}
			return ocimock.EmptyResponse(http.StatusNoContent), nil
		},
		NotFound: func(request ocimock.Request) (ocimock.Response, error) {
			if err := validateScriptDomain(request); err != nil {
				return ocimock.Response{}, err
			}
			return ocimock.JSONResponse(http.StatusNotFound, map[string]any{"code": "NotAuthorizedOrNotFound", "message": "resource not found"})
		},
	})
}

func mockScriptState(details apmsyntheticssdk.CreateScriptDetails, timestamp common.SDKTime) apmsyntheticssdk.Script {
	contentSize := 0
	if details.Content != nil {
		contentSize = len(*details.Content)
	}
	return apmsyntheticssdk.Script{
		Id:                    common.String(mockScriptID),
		DisplayName:           details.DisplayName,
		ContentType:           details.ContentType,
		MonitorStatusCountMap: &apmsyntheticssdk.MonitorStatusCountMap{Total: common.Int(0), Enabled: common.Int(0), Disabled: common.Int(0), Invalid: common.Int(0)},
		Content:               details.Content,
		TimeUploaded:          &timestamp,
		ContentSizeInBytes:    common.Int(contentSize),
		ContentFileName:       details.ContentFileName,
		Parameters:            []apmsyntheticssdk.ScriptParameterInfo{},
		TimeCreated:           &timestamp,
		TimeUpdated:           &timestamp,
		FreeformTags:          details.FreeformTags,
		DefinedTags:           details.DefinedTags,
	}
}

func validateScriptDomain(request ocimock.Request) error {
	if got := request.URL.Query().Get("apmDomainId"); got != mockScriptAPMDomainID {
		return fmt.Errorf("apmDomainId = %q, want %q", got, mockScriptAPMDomainID)
	}
	return nil
}

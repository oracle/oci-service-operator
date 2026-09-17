/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package config

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	apmconfigsdk "github.com/oracle/oci-go-sdk/v65/apmconfig"
	"github.com/oracle/oci-go-sdk/v65/common"
	apmconfigv1beta1 "github.com/oracle/oci-service-operator/api/apmconfig/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	mockConfigID    = "ocid1.apmconfig.oc1..mock"
	mockAPMDomainID = "ocid1.apmdomain.oc1..mock"
)

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - formal provider facts: formal/imports/apmconfig/config.json
//   - repo-authored runtime: formal/controllers/apmconfig/config/diagrams/runtime-lifecycle.yaml
//   - Terraform provider: terraform-provider-oci@eb653febb1ba internal/service/apm_config/apm_config_config_resource.go
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/apmconfig
func TestMockIntegrationConfigSpanFilterSynchronousCRUD(t *testing.T) {
	t.Parallel()

	resource := &apmconfigv1beta1.Config{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-span-filter", Namespace: "default", UID: types.UID("mock-span-filter-uid")},
		Spec: apmconfigv1beta1.ConfigSpec{
			ApmDomainId:  mockAPMDomainID,
			ConfigType:   "SPAN_FILTER",
			DisplayName:  "mock-span-filter",
			FilterText:   `service.name = "mock"`,
			Description:  "mock span filter",
			FreeformTags: map[string]string{"osok-mock": "create"},
		},
	}
	responder, err := newConfigMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://apm-config.mock.invalid", BasePath: "20210201", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Config OCI mock: %v", err)
		}
	})

	client := newConfigServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")},
		apmconfigsdk.ConfigClient{BaseClient: session.BaseClient()},
	)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apmconfigv1beta1.Config]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apmconfigv1beta1.Config) error {
			if current.Status.Id != mockConfigID ||
				current.Status.ApmDomainId != mockAPMDomainID ||
				current.Status.ConfigType != "SPAN_FILTER" ||
				current.Status.DisplayName != "mock-span-filter" ||
				current.Status.FilterText != `service.name = "mock"` {
				return fmt.Errorf("created Config status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apmconfigv1beta1.Config) {
			current.Spec.FilterText = `service.name = "mock-updated"`
			current.Spec.Description = "mock span filter updated"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *apmconfigv1beta1.Config) error {
			if current.Status.FilterText != current.Spec.FilterText ||
				current.Status.Description != current.Spec.Description ||
				current.Status.FreeformTags["osok-mock"] != "update" {
				return fmt.Errorf("updated Config status = %+v", current.Status)
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

func newConfigMockResponder(resource *apmconfigv1beta1.Config) (*ocimock.CRUDResponder[apmconfigsdk.SpanFilter], error) {
	createdAt := common.SDKTime{Time: time.Date(2026, time.September, 4, 15, 0, 0, 0, time.UTC)}
	updatedAt := common.SDKTime{Time: createdAt.Time.Add(time.Minute)}
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[apmconfigsdk.SpanFilter]{
		CollectionPath:     "/20210201/configs",
		ItemPath:           "/20210201/configs/" + mockConfigID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		Create: func(request ocimock.Request) (apmconfigsdk.SpanFilter, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero apmconfigsdk.SpanFilter
				return zero, ocimock.Response{}, err
			}
			if err := validateConfigDomain(request); err != nil {
				return apmconfigsdk.SpanFilter{}, ocimock.Response{}, err
			}
			var details apmconfigsdk.CreateSpanFilterDetails
			if err := ocimock.DecodeDiscriminatedJSONRequest(request, &details, "configType", "SPAN_FILTER"); err != nil {
				return apmconfigsdk.SpanFilter{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return apmconfigsdk.SpanFilter{}, ocimock.Response{}, err
			}
			expected := apmconfigsdk.CreateSpanFilterDetails{
				DisplayName:  common.String("mock-span-filter"),
				FilterText:   common.String(`service.name = "mock"`),
				Description:  common.String("mock span filter"),
				FreeformTags: map[string]string{"osok-mock": "create"},
			}
			if !reflect.DeepEqual(details, expected) {
				return apmconfigsdk.SpanFilter{}, ocimock.Response{}, fmt.Errorf("create Config details = %+v, want %+v", details, expected)
			}
			state := apmconfigsdk.SpanFilter{
				Id:           common.String(mockConfigID),
				TimeCreated:  &createdAt,
				TimeUpdated:  &createdAt,
				CreatedBy:    common.String("ocid1.user.oc1..mock"),
				UpdatedBy:    common.String("ocid1.user.oc1..mock"),
				Etag:         common.String("mock-etag-1"),
				DisplayName:  details.DisplayName,
				FilterText:   details.FilterText,
				Description:  details.Description,
				FreeformTags: details.FreeformTags,
				DefinedTags:  details.DefinedTags,
				InUseBy:      []apmconfigsdk.SpanFilterReference{},
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Read: func(request ocimock.Request, state apmconfigsdk.SpanFilter) (ocimock.Response, error) {
			if err := validateConfigDomain(request); err != nil {
				return ocimock.Response{}, err
			}
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		Update: func(request ocimock.Request, state apmconfigsdk.SpanFilter) (apmconfigsdk.SpanFilter, ocimock.Response, error) {
			if err := validateConfigDomain(request); err != nil {
				return apmconfigsdk.SpanFilter{}, ocimock.Response{}, err
			}
			var details apmconfigsdk.UpdateSpanFilterDetails
			if err := ocimock.DecodeDiscriminatedJSONRequest(request, &details, "configType", "SPAN_FILTER"); err != nil {
				return apmconfigsdk.SpanFilter{}, ocimock.Response{}, err
			}
			expected := apmconfigsdk.UpdateSpanFilterDetails{
				DisplayName:  common.String("mock-span-filter"),
				FilterText:   common.String(`service.name = "mock-updated"`),
				Description:  common.String("mock span filter updated"),
				FreeformTags: map[string]string{"osok-mock": "update"},
			}
			if !reflect.DeepEqual(details, expected) {
				return apmconfigsdk.SpanFilter{}, ocimock.Response{}, fmt.Errorf("update Config details = %+v, want %+v", details, expected)
			}
			state.DisplayName = details.DisplayName
			state.FilterText = details.FilterText
			state.Description = details.Description
			state.FreeformTags = details.FreeformTags
			state.DefinedTags = details.DefinedTags
			state.TimeUpdated = &updatedAt
			state.Etag = common.String("mock-etag-2")
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Delete: func(request ocimock.Request, _ apmconfigsdk.SpanFilter) (ocimock.Response, error) {
			if err := validateConfigDomain(request); err != nil {
				return ocimock.Response{}, err
			}
			if len(request.Body) != 0 {
				return ocimock.Response{}, fmt.Errorf("delete Config body = %s", request.Body)
			}
			return ocimock.EmptyResponse(http.StatusNoContent), nil
		},
		NotFound: func(request ocimock.Request) (ocimock.Response, error) {
			if err := validateConfigDomain(request); err != nil {
				return ocimock.Response{}, err
			}
			return ocimock.JSONResponse(http.StatusNotFound, map[string]any{"code": "NotAuthorizedOrNotFound", "message": "resource not found"})
		},
	})
}

func validateConfigDomain(request ocimock.Request) error {
	if got := request.URL.Query().Get("apmDomainId"); got != mockAPMDomainID {
		return fmt.Errorf("apmDomainId = %q, want %q", got, mockAPMDomainID)
	}
	return nil
}

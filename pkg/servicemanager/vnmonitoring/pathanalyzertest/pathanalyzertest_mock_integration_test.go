/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package pathanalyzertest

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	vnmonitoringsdk "github.com/oracle/oci-go-sdk/v65/vnmonitoring"
	vnmonitoringv1beta1 "github.com/oracle/oci-service-operator/api/vnmonitoring/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const mockPathAnalyzerTestID = "ocid1.pathanalyzertest.oc1..mock"

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - formal provider facts: formal/imports/vnmonitoring/pathanalyzertest.json
//   - repo-authored runtime: formal/controllers/vnmonitoring/pathanalyzertest/diagrams/runtime-lifecycle.yaml
//   - Terraform provider: terraform-provider-oci@eb653febb1ba internal/service/vn_monitoring/vn_monitoring_path_analyzer_test_resource.go
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/vnmonitoring
func TestMockIntegrationPathAnalyzerTestSynchronousCRUD(t *testing.T) {
	t.Parallel()

	resource := &vnmonitoringv1beta1.PathAnalyzerTest{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-path-analysis", Namespace: "default", UID: types.UID("mock-path-analysis-uid")},
		Spec: vnmonitoringv1beta1.PathAnalyzerTestSpec{
			CompartmentId: "ocid1.compartment.oc1..mock",
			DisplayName:   "mock-path-analysis",
			Protocol:      6,
			SourceEndpoint: vnmonitoringv1beta1.PathAnalyzerTestSourceEndpoint{
				Type: "IP_ADDRESS", Address: "10.0.0.10",
			},
			DestinationEndpoint: vnmonitoringv1beta1.PathAnalyzerTestDestinationEndpoint{
				Type: "IP_ADDRESS", Address: "10.0.1.20",
			},
			ProtocolParameters: vnmonitoringv1beta1.PathAnalyzerTestProtocolParameters{
				Type: "TCP", DestinationPort: 443,
			},
			QueryOptions: vnmonitoringv1beta1.PathAnalyzerTestQueryOptions{IsBiDirectionalAnalysis: true},
			FreeformTags: map[string]string{"osok-mock": "create"},
		},
	}
	responder, err := newPathAnalyzerTestMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://vnca.mock.invalid", BasePath: "20160918", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close PathAnalyzerTest OCI mock: %v", err)
		}
	})

	sdkClient := vnmonitoringsdk.VnMonitoringClient{BaseClient: session.BaseClient()}
	client := newPathAnalyzerTestServiceClientWithOCIClient(sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*vnmonitoringv1beta1.PathAnalyzerTest]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *vnmonitoringv1beta1.PathAnalyzerTest) error {
			if current.Status.Id != mockPathAnalyzerTestID ||
				current.Status.DisplayName != "mock-path-analysis" ||
				current.Status.LifecycleState != string(vnmonitoringsdk.PathAnalyzerTestLifecycleStateActive) ||
				current.Status.SourceEndpoint.Address != "10.0.0.10" ||
				current.Status.ProtocolParameters.DestinationPort != 443 ||
				!current.Status.QueryOptions.IsBiDirectionalAnalysis {
				return fmt.Errorf("created PathAnalyzerTest status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *vnmonitoringv1beta1.PathAnalyzerTest) {
			current.Spec.DisplayName = "mock-path-analysis-updated"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *vnmonitoringv1beta1.PathAnalyzerTest) error {
			if current.Status.DisplayName != current.Spec.DisplayName ||
				current.Status.FreeformTags["osok-mock"] != "update" ||
				current.Status.LifecycleState != string(vnmonitoringsdk.PathAnalyzerTestLifecycleStateActive) {
				return fmt.Errorf("updated PathAnalyzerTest status = %+v", current.Status)
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

func newPathAnalyzerTestMockResponder(resource *vnmonitoringv1beta1.PathAnalyzerTest) (*ocimock.CRUDResponder[vnmonitoringsdk.PathAnalyzerTest], error) {
	createdAt := common.SDKTime{Time: time.Date(2026, time.September, 4, 14, 0, 0, 0, time.UTC)}
	updatedAt := common.SDKTime{Time: createdAt.Time.Add(time.Minute)}
	deletedState := vnmonitoringsdk.PathAnalyzerTest{}
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[vnmonitoringsdk.PathAnalyzerTest]{
		CollectionPath:     "/20160918/pathAnalyzerTests",
		ItemPath:           "/20160918/pathAnalyzerTests/" + mockPathAnalyzerTestID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		Create: func(request ocimock.Request) (vnmonitoringsdk.PathAnalyzerTest, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero vnmonitoringsdk.PathAnalyzerTest
				return zero, ocimock.Response{}, err
			}
			var details vnmonitoringsdk.CreatePathAnalyzerTestDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return vnmonitoringsdk.PathAnalyzerTest{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return vnmonitoringsdk.PathAnalyzerTest{}, ocimock.Response{}, err
			}
			expected := mockCreatePathAnalyzerTestDetails(resource)
			if !reflect.DeepEqual(details, expected) {
				return vnmonitoringsdk.PathAnalyzerTest{}, ocimock.Response{}, fmt.Errorf("create PathAnalyzerTest details = %+v, want %+v", details, expected)
			}
			state := mockPathAnalyzerTestState(mockPathAnalyzerTestID, details, createdAt)
			deletedState = state
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Read: func(_ ocimock.Request, state vnmonitoringsdk.PathAnalyzerTest) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		Update: func(request ocimock.Request, state vnmonitoringsdk.PathAnalyzerTest) (vnmonitoringsdk.PathAnalyzerTest, ocimock.Response, error) {
			var details vnmonitoringsdk.UpdatePathAnalyzerTestDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return vnmonitoringsdk.PathAnalyzerTest{}, ocimock.Response{}, err
			}
			expected := mockUpdatePathAnalyzerTestDetails()
			if !reflect.DeepEqual(details, expected) {
				return vnmonitoringsdk.PathAnalyzerTest{}, ocimock.Response{}, fmt.Errorf("update PathAnalyzerTest details = %+v, want %+v", details, expected)
			}
			state.DisplayName = details.DisplayName
			state.Protocol = details.Protocol
			state.SourceEndpoint = details.SourceEndpoint
			state.DestinationEndpoint = details.DestinationEndpoint
			state.ProtocolParameters = details.ProtocolParameters
			state.QueryOptions = details.QueryOptions
			state.FreeformTags = details.FreeformTags
			state.DefinedTags = details.DefinedTags
			state.TimeUpdated = &updatedAt
			deletedState = state
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Delete: func(request ocimock.Request, state vnmonitoringsdk.PathAnalyzerTest) (ocimock.Response, error) {
			if len(request.Body) != 0 {
				return ocimock.Response{}, fmt.Errorf("delete PathAnalyzerTest body = %s", request.Body)
			}
			deletedState = state
			deletedState.LifecycleState = vnmonitoringsdk.PathAnalyzerTestLifecycleStateDeleted
			deletedState.TimeUpdated = &updatedAt
			return ocimock.EmptyResponse(http.StatusNoContent), nil
		},
		NotFound: func(_ ocimock.Request) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, deletedState)
		},
	})
}

func mockCreatePathAnalyzerTestDetails(resource *vnmonitoringv1beta1.PathAnalyzerTest) vnmonitoringsdk.CreatePathAnalyzerTestDetails {
	return vnmonitoringsdk.CreatePathAnalyzerTestDetails{
		CompartmentId:       common.String(resource.Spec.CompartmentId),
		Protocol:            common.Int(resource.Spec.Protocol),
		SourceEndpoint:      vnmonitoringsdk.IpAddressEndpoint{Address: common.String("10.0.0.10")},
		DestinationEndpoint: vnmonitoringsdk.IpAddressEndpoint{Address: common.String("10.0.1.20")},
		DisplayName:         common.String("mock-path-analysis"),
		ProtocolParameters:  vnmonitoringsdk.TcpProtocolParameters{DestinationPort: common.Int(443)},
		QueryOptions:        &vnmonitoringsdk.QueryOptions{IsBiDirectionalAnalysis: common.Bool(true)},
		FreeformTags:        map[string]string{"osok-mock": "create"},
	}
}

func mockUpdatePathAnalyzerTestDetails() vnmonitoringsdk.UpdatePathAnalyzerTestDetails {
	return vnmonitoringsdk.UpdatePathAnalyzerTestDetails{
		DisplayName:         common.String("mock-path-analysis-updated"),
		Protocol:            common.Int(6),
		SourceEndpoint:      vnmonitoringsdk.IpAddressEndpoint{Address: common.String("10.0.0.10")},
		DestinationEndpoint: vnmonitoringsdk.IpAddressEndpoint{Address: common.String("10.0.1.20")},
		ProtocolParameters:  vnmonitoringsdk.TcpProtocolParameters{DestinationPort: common.Int(443)},
		QueryOptions:        &vnmonitoringsdk.QueryOptions{IsBiDirectionalAnalysis: common.Bool(true)},
		FreeformTags:        map[string]string{"osok-mock": "update"},
		DefinedTags:         map[string]map[string]interface{}{},
	}
}

func mockPathAnalyzerTestState(id string, details vnmonitoringsdk.CreatePathAnalyzerTestDetails, timestamp common.SDKTime) vnmonitoringsdk.PathAnalyzerTest {
	return vnmonitoringsdk.PathAnalyzerTest{
		Id:                  common.String(id),
		DisplayName:         details.DisplayName,
		CompartmentId:       details.CompartmentId,
		Protocol:            details.Protocol,
		SourceEndpoint:      details.SourceEndpoint,
		DestinationEndpoint: details.DestinationEndpoint,
		QueryOptions:        details.QueryOptions,
		TimeCreated:         &timestamp,
		TimeUpdated:         &timestamp,
		LifecycleState:      vnmonitoringsdk.PathAnalyzerTestLifecycleStateActive,
		ProtocolParameters:  details.ProtocolParameters,
		FreeformTags:        details.FreeformTags,
		DefinedTags:         details.DefinedTags,
		SystemTags:          map[string]map[string]interface{}{},
	}
}

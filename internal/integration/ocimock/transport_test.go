/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package ocimock

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	budgetsdk "github.com/oracle/oci-go-sdk/v65/budget"
	"github.com/oracle/oci-go-sdk/v65/common"
)

type testResponder struct {
	respond func(Request) (Response, error)
	verify  func() error
}

func (r testResponder) Respond(request Request) (Response, error) {
	return r.respond(request)
}

func (r testResponder) Verify() error {
	if r.verify == nil {
		return nil
	}
	return r.verify()
}

func TestSessionRoutesRealSDKRequestAndDecodesTypedResponse(t *testing.T) {
	t.Parallel()

	called := false
	responder := testResponder{
		respond: func(request Request) (Response, error) {
			called = true
			if request.Method != http.MethodGet || request.URL.Path != "/20190111/budgets" {
				return Response{}, fmt.Errorf("request = %s %s", request.Method, request.URL.String())
			}
			if got := request.URL.Query().Get("compartmentId"); got != "ocid1.compartment.oc1..mock" {
				return Response{}, fmt.Errorf("compartmentId = %q", got)
			}
			return JSONResponse(http.StatusOK, []budgetsdk.BudgetSummary{{
				Id:             common.String("ocid1.budget.oc1..mock"),
				CompartmentId:  common.String("ocid1.compartment.oc1..mock"),
				DisplayName:    common.String("mock-budget"),
				Amount:         common.Float32(100),
				ResetPeriod:    budgetsdk.ResetPeriodMonthly,
				LifecycleState: budgetsdk.LifecycleStateActive,
				AlertRuleCount: common.Int(0),
			}})
		},
		verify: func() error {
			if !called {
				return fmt.Errorf("SDK request was not observed")
			}
			return nil
		},
	}
	session, err := Open(Options{
		Host:      "https://usage.mock.invalid",
		BasePath:  "20190111",
		Responder: responder,
	})
	if err != nil {
		t.Fatal(err)
	}
	client := budgetsdk.BudgetClient{BaseClient: session.BaseClient()}
	response, err := client.ListBudgets(context.Background(), budgetsdk.ListBudgetsRequest{
		CompartmentId: common.String("ocid1.compartment.oc1..mock"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Items) != 1 || response.Items[0].DisplayName == nil || *response.Items[0].DisplayName != "mock-budget" {
		t.Fatalf("typed SDK response = %+v", response.Items)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateMandatoryFieldsUsesSDKTags(t *testing.T) {
	t.Parallel()

	details := budgetsdk.CreateBudgetDetails{ResetPeriod: budgetsdk.ResetPeriodMonthly}
	err := ValidateMandatoryFields(details)
	if err == nil || !strings.Contains(err.Error(), "amount") || !strings.Contains(err.Error(), "compartmentId") {
		t.Fatalf("ValidateMandatoryFields() error = %v", err)
	}
	details.Amount = common.Float32(100)
	details.CompartmentId = common.String("ocid1.compartment.oc1..mock")
	if err := ValidateMandatoryFields(details); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeJSONRequestRejectsUnknownSDKFields(t *testing.T) {
	t.Parallel()

	request := Request{Method: http.MethodPost, URL: mustTestURL(t, "https://usage.mock.invalid/20190111/budgets"), Body: []byte(`{"amount":100,"compartmentId":"mock","resetPeriod":"MONTHLY","unknown":true}`)}
	var details budgetsdk.CreateBudgetDetails
	if err := DecodeJSONRequest(request, &details); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("DecodeJSONRequest() error = %v", err)
	}
}

func TestDecodeDiscriminatedJSONRequestStrictlyDecodesConcreteBody(t *testing.T) {
	t.Parallel()

	type details struct {
		Name *string `json:"name"`
	}
	request := Request{Method: http.MethodPost, URL: mustTestURL(t, "https://example.invalid/configs"), Body: []byte(`{"configType":"SPAN_FILTER","name":"example"}`)}
	var decoded details
	if err := DecodeDiscriminatedJSONRequest(request, &decoded, "configType", "SPAN_FILTER"); err != nil {
		t.Fatal(err)
	}
	if decoded.Name == nil || *decoded.Name != "example" {
		t.Fatalf("decoded details = %+v", decoded)
	}
}

func TestDecodeDiscriminatedJSONRequestRejectsWrongTypeAndUnknownFields(t *testing.T) {
	t.Parallel()

	type details struct {
		Name *string `json:"name"`
	}
	wrongType := Request{Method: http.MethodPost, URL: mustTestURL(t, "https://example.invalid/configs"), Body: []byte(`{"configType":"OPTIONS","name":"example"}`)}
	if err := DecodeDiscriminatedJSONRequest(wrongType, &details{}, "configType", "SPAN_FILTER"); err == nil || !strings.Contains(err.Error(), "want \"SPAN_FILTER\"") {
		t.Fatalf("wrong discriminator error = %v", err)
	}
	unknown := Request{Method: http.MethodPost, URL: mustTestURL(t, "https://example.invalid/configs"), Body: []byte(`{"configType":"SPAN_FILTER","name":"example","unknown":true}`)}
	if err := DecodeDiscriminatedJSONRequest(unknown, &details{}, "configType", "SPAN_FILTER"); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown field error = %v", err)
	}
}

func mustTestURL(t *testing.T, value string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

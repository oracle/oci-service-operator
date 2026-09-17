/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package ocimock

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/datalabelingservice"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type explicitFixture struct {
	Name  string            `json:"name"`
	Count int               `json:"count"`
	Tags  map[string]string `json:"tags,omitempty"`
}

func TestStateSequencePreservesOrder(t *testing.T) {
	t.Parallel()

	states := StateSequence("CREATING", "ACTIVE")
	if len(states) != 2 || states[0] != "CREATING" || states[1] != "ACTIVE" {
		t.Fatalf("StateSequence() = %v", states)
	}
}

func TestLifecycleStateSequenceClonesPendingStatesBeforeTerminal(t *testing.T) {
	t.Parallel()

	type resource struct {
		ID             string `json:"id"`
		LifecycleState string `json:"lifecycleState"`
	}
	terminal := resource{ID: "resource-1", LifecycleState: "ACTIVE"}
	states := LifecycleStateSequence(t, terminal, "CREATING", "UPDATING")
	if len(states) != 3 || states[0].LifecycleState != "CREATING" || states[1].LifecycleState != "UPDATING" || states[2] != terminal {
		t.Fatalf("lifecycle states = %+v", states)
	}
}

func TestLifecycleStatesPreservesOCIResponseFields(t *testing.T) {
	t.Parallel()

	baseline := datalabelingservice.Dataset{
		Id:             common.String("dataset-id"),
		CompartmentId:  common.String("compartment-id"),
		LifecycleState: datalabelingservice.DatasetLifecycleStateActive,
	}
	pending := LifecycleStates(t, baseline, "CREATING")
	if len(pending) != 1 || pending[0].Id == nil || *pending[0].Id != "dataset-id" || pending[0].CompartmentId == nil || *pending[0].CompartmentId != "compartment-id" || pending[0].LifecycleState != datalabelingservice.DatasetLifecycleStateCreating {
		t.Fatalf("pending Dataset = %+v", pending)
	}
}

func TestNewWorkRequestResponseSequenceReturnsPendingThenTerminal(t *testing.T) {
	t.Parallel()

	type workRequest struct {
		ID              string  `json:"id"`
		Status          string  `json:"status"`
		PercentComplete float32 `json:"percentComplete"`
		TimeFinished    *string `json:"timeFinished"`
	}
	finished := "2026-09-09T00:00:00Z"
	respond := NewWorkRequestResponseSequence(t, "IN_PROGRESS", workRequest{ID: "work-request-1", Status: "SUCCEEDED", PercentComplete: 100, TimeFinished: &finished})
	for index, want := range []string{"IN_PROGRESS", "SUCCEEDED", "SUCCEEDED"} {
		response, err := respond(Request{})
		if err != nil {
			t.Fatal(err)
		}
		var got workRequest
		if err := json.Unmarshal(response.Body, &got); err != nil {
			t.Fatal(err)
		}
		if got.ID != "work-request-1" || got.Status != want {
			t.Fatalf("response %d = %+v, want status %q", index, got, want)
		}
		if index == 0 && (got.PercentComplete != 0 || got.TimeFinished != nil) {
			t.Fatalf("pending response = %+v, want incomplete without finish time", got)
		}
	}
}

func TestMustJSONFixtureReturnsExplicitType(t *testing.T) {
	t.Parallel()
	fixture := MustJSONFixture[explicitFixture](t, "{\"name\":\"typed\",\"count\":2}")
	if fixture.Name != "typed" || fixture.Count != 2 || fixture.Tags != nil {
		t.Fatalf("fixture = %+v", fixture)
	}
}

func TestValidateJSONRequestComparesCompleteTypedValue(t *testing.T) {
	t.Parallel()
	request := Request{
		Method: http.MethodPost,
		URL:    mustTestURL(t, "https://mock.invalid/things"),
		Body:   []byte("{\"name\":\"typed\",\"count\":2}"),
	}
	if err := ValidateJSONRequest(request, explicitFixture{Name: "typed", Count: 2}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateJSONRequest(request, explicitFixture{Name: "typed", Count: 3}); err == nil {
		t.Fatal("ValidateJSONRequest() error = nil, want mismatch")
	}
}

func TestCompareJSONSubsetChecksDeclaredFieldsAndAllowsAdditionalFields(t *testing.T) {
	t.Parallel()
	type requestFixture struct {
		Name    *string `json:"name,omitempty"`
		Enabled *bool   `json:"enabled,omitempty"`
	}
	name := "typed"
	enabled := false
	actual := requestFixture{Name: &name, Enabled: &enabled}
	if err := CompareJSONSubset(actual, requestFixture{Name: &name}); err != nil {
		t.Fatal(err)
	}
	other := "other"
	if err := CompareJSONSubset(actual, requestFixture{Name: &other}); err == nil {
		t.Fatal("CompareJSONSubset() error = nil, want declared-field mismatch")
	}
}

func TestValidateRetryTokenMatchesResourceUID(t *testing.T) {
	resource := &metav1.PartialObjectMetadata{ObjectMeta: metav1.ObjectMeta{UID: types.UID("stable-resource-uid")}}
	request := Request{Method: http.MethodPost, URL: &url.URL{Path: "/resources"}, Header: http.Header{"Opc-Retry-Token": []string{"stable-resource-uid"}}}
	if err := ValidateRetryToken(request, resource); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRetryTokenRejectsMissingOrDifferentUID(t *testing.T) {
	request := Request{Method: http.MethodPost, URL: &url.URL{Path: "/resources"}, Header: make(http.Header)}
	if err := ValidateRetryToken(request, &metav1.PartialObjectMetadata{}); err == nil {
		t.Fatal("expected empty resource UID to fail")
	}
	resource := &metav1.PartialObjectMetadata{ObjectMeta: metav1.ObjectMeta{UID: types.UID("stable-resource-uid")}}
	if err := ValidateRetryToken(request, resource); err == nil {
		t.Fatal("expected missing retry token to fail")
	}
	request.Header.Set("opc-retry-token", "different-uid")
	if err := ValidateRetryToken(request, resource); err == nil {
		t.Fatal("expected different retry token to fail")
	}
}

func TestValidateRetryTokenValueSupportsPackageOwnedDeterministicTokens(t *testing.T) {
	t.Parallel()

	request := Request{Method: http.MethodPost, URL: &url.URL{Path: "/resources"}, Header: http.Header{"Opc-Retry-Token": []string{"scoped-token"}}}
	if err := ValidateRetryTokenValue(request, "scoped-token"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateRetryTokenValue(request, ""); err == nil {
		t.Fatal("expected empty deterministic token to fail")
	}
	if err := ValidateRetryTokenValue(request, "other-token"); err == nil {
		t.Fatal("expected different deterministic token to fail")
	}
}

func TestMustMergeJSONFixturePreservesOmittedTypedFields(t *testing.T) {
	t.Parallel()
	baseline := explicitFixture{Name: "created", Count: 1, Tags: map[string]string{"phase": "create", "retained": "yes"}}
	fixture := baseline
	MustMergeJSONFixture(t, &fixture, "{\"name\":\"updated\",\"tags\":{\"phase\":\"update\"}}")
	if fixture.Name != "updated" || fixture.Count != 1 || fixture.Tags["phase"] != "update" || fixture.Tags["retained"] != "yes" {
		t.Fatalf("fixture = %+v", fixture)
	}
	if baseline.Tags["phase"] != "create" {
		t.Fatalf("baseline was mutated: %+v", baseline)
	}
}

func TestMustOCIResponseFixtureAllowsAdditiveServiceFields(t *testing.T) {
	t.Parallel()
	fixture := MustOCIResponseFixture[explicitFixture](t, "{\"name\":\"typed\",\"count\":2,\"futureField\":true}")
	if fixture.Name != "typed" || fixture.Count != 2 || fixture.Tags != nil {
		t.Fatalf("fixture = %+v", fixture)
	}
}

func TestValidateDiscriminatedJSONRequestUsesConcreteDetails(t *testing.T) {
	t.Parallel()
	request := Request{
		Method: http.MethodPost,
		URL:    mustTestURL(t, "https://mock.invalid/things"),
		Body:   []byte("{\"type\":\"NAMED\",\"name\":\"typed\",\"count\":2}"),
	}
	if err := ValidateDiscriminatedJSONRequest(request, "type", "NAMED", explicitFixture{Name: "typed", Count: 2}); err != nil {
		t.Fatal(err)
	}
}

func TestValidateDiscriminatedJSONRequestSubsetAllowsAdditionalConcreteFields(t *testing.T) {
	t.Parallel()
	request := Request{
		Method: http.MethodPost,
		URL:    mustTestURL(t, "https://mock.invalid/things"),
		Body:   []byte("{\"type\":\"NAMED\",\"name\":\"typed\",\"count\":2}"),
	}
	if err := ValidateDiscriminatedJSONRequestSubset(request, "type", "NAMED", struct {
		Name string `json:"name"`
	}{Name: "typed"}); err != nil {
		t.Fatal(err)
	}
}

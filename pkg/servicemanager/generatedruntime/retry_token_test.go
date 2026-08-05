/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"context"
	"strings"
	"testing"
)

type retryTokenDetails struct {
	DisplayName string `json:"displayName,omitempty"`
	Enabled     bool   `json:"enabled,omitempty"`
}

type retryTokenCreateRequest struct {
	OpcRetryToken     *string           `contributesTo:"header" name:"opc-retry-token"`
	RetryTokenDetails retryTokenDetails `contributesTo:"body"`
}

type retryTokenUpdateRequest struct {
	ResourceId        *string           `contributesTo:"path" name:"resourceId"`
	OpcRetryToken     *string           `contributesTo:"header" name:"opc-retry-token"`
	RetryTokenDetails retryTokenDetails `contributesTo:"body"`
}

type retryTokenDeleteRequest struct {
	ResourceId    *string `contributesTo:"path" name:"resourceId"`
	OpcRetryToken *string `contributesTo:"header" name:"opc-retry-token"`
}

type retryTokenResponse struct{}

func TestServiceClientInvokeScopesDeterministicRetryTokensByOperationAndRequest(t *testing.T) {
	t.Parallel()

	var createToken string
	updateTokens := make([]string, 0, 2)
	deleteTokens := make([]string, 0, 2)

	createOp := &Operation{
		NewRequest: func() any { return &retryTokenCreateRequest{} },
		Fields: []RequestField{
			{FieldName: "RetryTokenDetails", RequestName: "RetryTokenDetails", Contribution: "body"},
		},
		Call: func(_ context.Context, request any) (any, error) {
			createToken = requiredRetryToken(t, request.(*retryTokenCreateRequest).OpcRetryToken)
			return retryTokenResponse{}, nil
		},
	}
	updateOp := &Operation{
		NewRequest: func() any { return &retryTokenUpdateRequest{} },
		Fields: []RequestField{
			{FieldName: "ResourceId", RequestName: "resourceId", Contribution: "path", PreferResourceID: true},
			{FieldName: "RetryTokenDetails", RequestName: "RetryTokenDetails", Contribution: "body"},
		},
		Call: func(_ context.Context, request any) (any, error) {
			updateTokens = append(updateTokens, requiredRetryToken(t, request.(*retryTokenUpdateRequest).OpcRetryToken))
			return retryTokenResponse{}, nil
		},
	}
	deleteOp := &Operation{
		NewRequest: func() any { return &retryTokenDeleteRequest{} },
		Fields: []RequestField{
			{FieldName: "ResourceId", RequestName: "resourceId", Contribution: "path", PreferResourceID: true},
		},
		Call: func(_ context.Context, request any) (any, error) {
			deleteTokens = append(deleteTokens, requiredRetryToken(t, request.(*retryTokenDeleteRequest).OpcRetryToken))
			return retryTokenResponse{}, nil
		},
	}

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:   "Thing",
		Create: createOp,
		Update: updateOp,
		Delete: deleteOp,
		BuildCreateBody: func(_ context.Context, resource *fakeResource, _ string) (any, error) {
			return retryTokenDetails{DisplayName: resource.Spec.DisplayName, Enabled: resource.Spec.Enabled}, nil
		},
		BuildUpdateBody: func(_ context.Context, resource *fakeResource, _ string, _ any) (any, bool, error) {
			return retryTokenDetails{DisplayName: resource.Spec.DisplayName, Enabled: resource.Spec.Enabled}, true, nil
		},
	})
	ctx := context.Background()
	resource := &fakeResource{
		Name:      "thing",
		Namespace: "default",
		UID:       "11111111-1111-1111-1111-111111111111",
		Spec: fakeSpec{
			DisplayName: "first",
			Enabled:     true,
		},
	}

	if _, err := client.invoke(ctx, createOp, resource, "", client.requestBuildOptions(ctx, resource.Namespace)); err != nil {
		t.Fatalf("invoke create error = %v", err)
	}
	if _, err := client.invoke(ctx, updateOp, resource, "ocid1.thing.oc1..target", client.requestBuildOptions(ctx, resource.Namespace)); err != nil {
		t.Fatalf("invoke update error = %v", err)
	}
	if _, err := client.invoke(ctx, deleteOp, resource, "ocid1.thing.oc1..target", requestBuildOptions{}); err != nil {
		t.Fatalf("invoke delete error = %v", err)
	}
	if _, err := client.invoke(ctx, deleteOp, resource, "ocid1.thing.oc1..target", requestBuildOptions{}); err != nil {
		t.Fatalf("invoke delete retry error = %v", err)
	}

	resource.Spec.DisplayName = "second"
	if _, err := client.invoke(ctx, updateOp, resource, "ocid1.thing.oc1..target", client.requestBuildOptions(ctx, resource.Namespace)); err != nil {
		t.Fatalf("invoke changed update error = %v", err)
	}

	if createToken != string(resource.UID) {
		t.Fatalf("create retry token = %q, want legacy resource UID token %q", createToken, resource.UID)
	}
	requireRetryTokenScope(t, updateTokens[0], "update")
	requireRetryTokenScope(t, deleteTokens[0], "delete")
	if createToken == updateTokens[0] || createToken == deleteTokens[0] || updateTokens[0] == deleteTokens[0] {
		t.Fatalf("retry tokens should differ across operations: create=%q update=%q delete=%q", createToken, updateTokens[0], deleteTokens[0])
	}
	if deleteTokens[1] != deleteTokens[0] {
		t.Fatalf("delete retry token = %q, want stable %q", deleteTokens[1], deleteTokens[0])
	}
	if updateTokens[1] == updateTokens[0] {
		t.Fatalf("changed update token = %q, want a new token for a different update request", updateTokens[1])
	}
}

func TestBuildRequestScopesGeneratedRetryTokenWhenRequested(t *testing.T) {
	t.Parallel()

	request := &retryTokenCreateRequest{}
	resource := &fakeResource{
		UID: "11111111-1111-1111-1111-111111111111",
	}

	err := buildRequest(
		request,
		resource,
		nil,
		"",
		[]RequestField{
			{FieldName: "RetryTokenDetails", RequestName: "RetryTokenDetails", Contribution: "body"},
		},
		nil,
		requestBuildOptions{RetryTokenScope: "create"},
		retryTokenDetails{DisplayName: "first", Enabled: true},
		true,
	)
	if err != nil {
		t.Fatalf("buildRequest() error = %v", err)
	}
	got := requiredRetryToken(t, request.OpcRetryToken)
	requireRetryTokenScope(t, got, "create")
	if got == string(resource.UID) {
		t.Fatalf("buildRequest() retry token = %q, want request-scoped token", got)
	}
}

func TestBuildRequestPreservesExplicitRetryToken(t *testing.T) {
	t.Parallel()

	token := "caller-supplied-token"
	request := &retryTokenCreateRequest{
		OpcRetryToken: &token,
	}

	err := buildRequest(
		request,
		nil,
		nil,
		"",
		[]RequestField{
			{FieldName: "RetryTokenDetails", RequestName: "RetryTokenDetails", Contribution: "body"},
		},
		nil,
		requestBuildOptions{RetryTokenScope: "create"},
		retryTokenDetails{DisplayName: "first", Enabled: true},
		true,
	)
	if err != nil {
		t.Fatalf("buildRequest() error = %v", err)
	}
	if got := requiredRetryToken(t, request.OpcRetryToken); got != token {
		t.Fatalf("buildRequest() retry token = %q, want explicit token %q preserved", got, token)
	}
}

func requiredRetryToken(t *testing.T, token *string) string {
	t.Helper()
	if token == nil || *token == "" {
		t.Fatal("request retry token is empty")
	}
	return *token
}

func requireRetryTokenScope(t *testing.T, token string, scope string) {
	t.Helper()
	if !strings.Contains(token, "-"+scope+"-") {
		t.Fatalf("retry token %q should include operation scope %q", token, scope)
	}
}

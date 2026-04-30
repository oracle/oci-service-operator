/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package scheduledquery

import (
	"context"
	"fmt"
	"strings"

	apmtracessdk "github.com/oracle/oci-go-sdk/v65/apmtraces"
	apmtracesv1beta1 "github.com/oracle/oci-service-operator/api/apmtraces/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

type scheduledQueryOCIClient interface {
	CreateScheduledQuery(context.Context, apmtracessdk.CreateScheduledQueryRequest) (apmtracessdk.CreateScheduledQueryResponse, error)
	GetScheduledQuery(context.Context, apmtracessdk.GetScheduledQueryRequest) (apmtracessdk.GetScheduledQueryResponse, error)
	ListScheduledQueries(context.Context, apmtracessdk.ListScheduledQueriesRequest) (apmtracessdk.ListScheduledQueriesResponse, error)
	UpdateScheduledQuery(context.Context, apmtracessdk.UpdateScheduledQueryRequest) (apmtracessdk.UpdateScheduledQueryResponse, error)
	DeleteScheduledQuery(context.Context, apmtracessdk.DeleteScheduledQueryRequest) (apmtracessdk.DeleteScheduledQueryResponse, error)
}

func init() {
	registerScheduledQueryRuntimeHooksMutator(func(_ *ScheduledQueryServiceManager, hooks *ScheduledQueryRuntimeHooks) {
		applyScheduledQueryRuntimeHooks(hooks)
	})
}

func applyScheduledQueryRuntimeHooks(hooks *ScheduledQueryRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedScheduledQueryRuntimeSemantics()
	hooks.Identity.GuardExistingBeforeCreate = guardScheduledQueryExistingBeforeCreate
	hooks.Create.Fields = scheduledQueryCreateFields()
	hooks.Get.Fields = scheduledQueryGetFields()
	hooks.List.Fields = scheduledQueryListFields()
	hooks.Update.Fields = scheduledQueryUpdateFields()
	hooks.Delete.Fields = scheduledQueryDeleteFields()
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, wrapScheduledQueryStatusMirrorClient)
}

func reviewedScheduledQueryRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newScheduledQueryRuntimeSemantics()
	semantics.List = &generatedruntime.ListSemantics{
		ResponseItemsField: "Items",
		MatchFields:        []string{"scheduledQueryName"},
	}
	semantics.Unsupported = nil
	return semantics
}

func newScheduledQueryServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client scheduledQueryOCIClient,
) ScheduledQueryServiceClient {
	hooks := newScheduledQueryRuntimeHooksWithOCIClient(client)
	applyScheduledQueryRuntimeHooks(&hooks)
	delegate := defaultScheduledQueryServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*apmtracesv1beta1.ScheduledQuery](
			buildScheduledQueryGeneratedRuntimeConfig(&ScheduledQueryServiceManager{Log: log}, hooks),
		),
	}
	return wrapScheduledQueryGeneratedClient(hooks, delegate)
}

func newScheduledQueryRuntimeHooksWithOCIClient(client scheduledQueryOCIClient) ScheduledQueryRuntimeHooks {
	return ScheduledQueryRuntimeHooks{
		Semantics:       newScheduledQueryRuntimeSemantics(),
		Identity:        generatedruntime.IdentityHooks[*apmtracesv1beta1.ScheduledQuery]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*apmtracesv1beta1.ScheduledQuery]{},
		StatusHooks:     generatedruntime.StatusHooks[*apmtracesv1beta1.ScheduledQuery]{},
		ParityHooks:     generatedruntime.ParityHooks[*apmtracesv1beta1.ScheduledQuery]{},
		Async:           generatedruntime.AsyncHooks[*apmtracesv1beta1.ScheduledQuery]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*apmtracesv1beta1.ScheduledQuery]{},
		Create: runtimeOperationHooks[apmtracessdk.CreateScheduledQueryRequest, apmtracessdk.CreateScheduledQueryResponse]{
			Fields: scheduledQueryCreateFields(),
			Call: func(ctx context.Context, request apmtracessdk.CreateScheduledQueryRequest) (apmtracessdk.CreateScheduledQueryResponse, error) {
				return client.CreateScheduledQuery(ctx, request)
			},
		},
		Get: runtimeOperationHooks[apmtracessdk.GetScheduledQueryRequest, apmtracessdk.GetScheduledQueryResponse]{
			Fields: scheduledQueryGetFields(),
			Call: func(ctx context.Context, request apmtracessdk.GetScheduledQueryRequest) (apmtracessdk.GetScheduledQueryResponse, error) {
				return client.GetScheduledQuery(ctx, request)
			},
		},
		List: runtimeOperationHooks[apmtracessdk.ListScheduledQueriesRequest, apmtracessdk.ListScheduledQueriesResponse]{
			Fields: scheduledQueryListFields(),
			Call: func(ctx context.Context, request apmtracessdk.ListScheduledQueriesRequest) (apmtracessdk.ListScheduledQueriesResponse, error) {
				return client.ListScheduledQueries(ctx, request)
			},
		},
		Update: runtimeOperationHooks[apmtracessdk.UpdateScheduledQueryRequest, apmtracessdk.UpdateScheduledQueryResponse]{
			Fields: scheduledQueryUpdateFields(),
			Call: func(ctx context.Context, request apmtracessdk.UpdateScheduledQueryRequest) (apmtracessdk.UpdateScheduledQueryResponse, error) {
				return client.UpdateScheduledQuery(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[apmtracessdk.DeleteScheduledQueryRequest, apmtracessdk.DeleteScheduledQueryResponse]{
			Fields: scheduledQueryDeleteFields(),
			Call: func(ctx context.Context, request apmtracessdk.DeleteScheduledQueryRequest) (apmtracessdk.DeleteScheduledQueryResponse, error) {
				return client.DeleteScheduledQuery(ctx, request)
			},
		},
		WrapGeneratedClient: []func(ScheduledQueryServiceClient) ScheduledQueryServiceClient{},
	}
}

func scheduledQueryCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "ApmDomainId",
			RequestName:  "apmDomainId",
			Contribution: "query",
			LookupPaths:  []string{"status.apmDomainId", "spec.apmDomainId", "apmDomainId"},
		},
		{FieldName: "CreateScheduledQueryDetails", RequestName: "CreateScheduledQueryDetails", Contribution: "body"},
	}
}

func scheduledQueryGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "ApmDomainId",
			RequestName:  "apmDomainId",
			Contribution: "query",
			LookupPaths:  []string{"status.apmDomainId", "spec.apmDomainId", "apmDomainId"},
		},
		{FieldName: "ScheduledQueryId", RequestName: "scheduledQueryId", Contribution: "path", PreferResourceID: true},
	}
}

func scheduledQueryListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "ApmDomainId",
			RequestName:  "apmDomainId",
			Contribution: "query",
			LookupPaths:  []string{"status.apmDomainId", "spec.apmDomainId", "apmDomainId"},
		},
		{
			FieldName:    "DisplayName",
			RequestName:  "displayName",
			Contribution: "query",
			LookupPaths:  []string{"status.scheduledQueryName", "spec.scheduledQueryName", "scheduledQueryName"},
		},
	}
}

func scheduledQueryUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "ApmDomainId",
			RequestName:  "apmDomainId",
			Contribution: "query",
			LookupPaths:  []string{"status.apmDomainId", "spec.apmDomainId", "apmDomainId"},
		},
		{FieldName: "ScheduledQueryId", RequestName: "scheduledQueryId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateScheduledQueryDetails", RequestName: "UpdateScheduledQueryDetails", Contribution: "body"},
	}
}

func scheduledQueryDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "ApmDomainId",
			RequestName:  "apmDomainId",
			Contribution: "query",
			LookupPaths:  []string{"status.apmDomainId", "spec.apmDomainId", "apmDomainId"},
		},
		{FieldName: "ScheduledQueryId", RequestName: "scheduledQueryId", Contribution: "path", PreferResourceID: true},
	}
}

func guardScheduledQueryExistingBeforeCreate(
	_ context.Context,
	resource *apmtracesv1beta1.ScheduledQuery,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("ScheduledQuery resource is nil")
	}
	if strings.TrimSpace(resource.Spec.ApmDomainId) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("ScheduledQuery spec.apmDomainId is required")
	}
	if strings.TrimSpace(resource.Spec.ScheduledQueryName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

type scheduledQueryStatusMirrorClient struct {
	delegate ScheduledQueryServiceClient
}

func wrapScheduledQueryStatusMirrorClient(delegate ScheduledQueryServiceClient) ScheduledQueryServiceClient {
	return scheduledQueryStatusMirrorClient{delegate: delegate}
}

func (c scheduledQueryStatusMirrorClient) CreateOrUpdate(
	ctx context.Context,
	resource *apmtracesv1beta1.ScheduledQuery,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err == nil && response.IsSuccessful {
		projectScheduledQueryRequestContext(resource)
	}
	return response, err
}

func (c scheduledQueryStatusMirrorClient) Delete(ctx context.Context, resource *apmtracesv1beta1.ScheduledQuery) (bool, error) {
	return c.delegate.Delete(ctx, resource)
}

func projectScheduledQueryRequestContext(resource *apmtracesv1beta1.ScheduledQuery) {
	if resource == nil {
		return
	}

	resource.Status.ApmDomainId = strings.TrimSpace(resource.Spec.ApmDomainId)
	if strings.TrimSpace(resource.Status.ScheduledQueryName) == "" {
		resource.Status.ScheduledQueryName = strings.TrimSpace(resource.Spec.ScheduledQueryName)
	}
}

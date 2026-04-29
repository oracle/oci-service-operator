/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package acceptedagreement

import (
	"context"
	"fmt"
	"strings"

	marketplacesdk "github.com/oracle/oci-go-sdk/v65/marketplace"
	marketplacev1beta1 "github.com/oracle/oci-service-operator/api/marketplace/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type acceptedAgreementOCIClient interface {
	CreateAcceptedAgreement(context.Context, marketplacesdk.CreateAcceptedAgreementRequest) (marketplacesdk.CreateAcceptedAgreementResponse, error)
	GetAcceptedAgreement(context.Context, marketplacesdk.GetAcceptedAgreementRequest) (marketplacesdk.GetAcceptedAgreementResponse, error)
	ListAcceptedAgreements(context.Context, marketplacesdk.ListAcceptedAgreementsRequest) (marketplacesdk.ListAcceptedAgreementsResponse, error)
	UpdateAcceptedAgreement(context.Context, marketplacesdk.UpdateAcceptedAgreementRequest) (marketplacesdk.UpdateAcceptedAgreementResponse, error)
	DeleteAcceptedAgreement(context.Context, marketplacesdk.DeleteAcceptedAgreementRequest) (marketplacesdk.DeleteAcceptedAgreementResponse, error)
}

type acceptedAgreementRuntimeClient struct {
	delegate AcceptedAgreementServiceClient
	log      loggerutil.OSOKLogger
	initErr  error
}

func init() {
	registerAcceptedAgreementRuntimeHooksMutator(func(manager *AcceptedAgreementServiceManager, hooks *AcceptedAgreementRuntimeHooks) {
		applyAcceptedAgreementRuntimeHooks(hooks)
		appendAcceptedAgreementRuntimeWrapper(manager, hooks)
	})
}

func applyAcceptedAgreementRuntimeHooks(hooks *AcceptedAgreementRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newAcceptedAgreementRuntimeSemanticsOverride()
	hooks.Create.Fields = acceptedAgreementCreateFields()
	hooks.Get.Fields = acceptedAgreementGetFields()
	hooks.List.Fields = acceptedAgreementListFields()
	hooks.Update.Fields = acceptedAgreementUpdateFields()
	hooks.Delete.Fields = acceptedAgreementDeleteFields()
}

func appendAcceptedAgreementRuntimeWrapper(manager *AcceptedAgreementServiceManager, hooks *AcceptedAgreementRuntimeHooks) {
	if manager == nil || hooks == nil {
		return
	}

	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate AcceptedAgreementServiceClient) AcceptedAgreementServiceClient {
		return newAcceptedAgreementRuntimeWrapper(manager, delegate)
	})
}

func newAcceptedAgreementRuntimeClient(
	manager *AcceptedAgreementServiceManager,
	client acceptedAgreementOCIClient,
	initErr error,
) *acceptedAgreementRuntimeClient {
	log := loggerutil.OSOKLogger{}
	if manager != nil {
		log = manager.Log
	}
	delegate := defaultAcceptedAgreementServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*marketplacev1beta1.AcceptedAgreement](
			newAcceptedAgreementRuntimeConfig(log, client, initErr),
		),
	}
	runtimeClient := newAcceptedAgreementRuntimeWrapper(manager, delegate)
	runtimeClient.initErr = initErr
	return runtimeClient
}

func newAcceptedAgreementRuntimeWrapper(
	manager *AcceptedAgreementServiceManager,
	delegate AcceptedAgreementServiceClient,
) *acceptedAgreementRuntimeClient {
	runtimeClient := &acceptedAgreementRuntimeClient{delegate: delegate}
	if manager == nil {
		return runtimeClient
	}

	runtimeClient.log = manager.Log
	if manager.Provider == nil {
		return runtimeClient
	}
	if _, err := marketplacesdk.NewMarketplaceClientWithConfigurationProvider(manager.Provider); err != nil {
		runtimeClient.initErr = fmt.Errorf("initialize AcceptedAgreement OCI client: %w", err)
	}
	return runtimeClient
}

func newAcceptedAgreementRuntimeConfig(
	log loggerutil.OSOKLogger,
	client acceptedAgreementOCIClient,
	initErr error,
) generatedruntime.Config[*marketplacev1beta1.AcceptedAgreement] {
	return generatedruntime.Config[*marketplacev1beta1.AcceptedAgreement]{
		Kind:      "AcceptedAgreement",
		SDKName:   "AcceptedAgreement",
		Log:       log,
		InitError: initErr,
		Semantics: newAcceptedAgreementRuntimeSemanticsOverride(),
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &marketplacesdk.CreateAcceptedAgreementRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.CreateAcceptedAgreement(ctx, *request.(*marketplacesdk.CreateAcceptedAgreementRequest))
			},
			Fields: acceptedAgreementCreateFields(),
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &marketplacesdk.GetAcceptedAgreementRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.GetAcceptedAgreement(ctx, *request.(*marketplacesdk.GetAcceptedAgreementRequest))
			},
			Fields: acceptedAgreementGetFields(),
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &marketplacesdk.ListAcceptedAgreementsRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.ListAcceptedAgreements(ctx, *request.(*marketplacesdk.ListAcceptedAgreementsRequest))
			},
			Fields: acceptedAgreementListFields(),
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &marketplacesdk.UpdateAcceptedAgreementRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.UpdateAcceptedAgreement(ctx, *request.(*marketplacesdk.UpdateAcceptedAgreementRequest))
			},
			Fields: acceptedAgreementUpdateFields(),
		},
		Delete: &generatedruntime.Operation{
			NewRequest: func() any { return &marketplacesdk.DeleteAcceptedAgreementRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.DeleteAcceptedAgreement(ctx, *request.(*marketplacesdk.DeleteAcceptedAgreementRequest))
			},
			Fields: acceptedAgreementDeleteFields(),
		},
	}
}

func newAcceptedAgreementRuntimeSemanticsOverride() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "marketplace",
		FormalSlug:    "acceptedagreement",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{},
			UpdatingStates:     []string{},
			ActiveStates:       []string{"ACTIVE"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"agreementId", "compartmentId", "displayName", "listingId", "packageVersion"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"definedTags", "displayName", "freeformTags"},
			ForceNew:      []string{"agreementId", "compartmentId", "listingId", "packageVersion", "signature"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func acceptedAgreementCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CreateAcceptedAgreementDetails",
			RequestName:  "CreateAcceptedAgreementDetails",
			Contribution: "body",
		},
	}
}

func acceptedAgreementGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:        "AcceptedAgreementId",
			RequestName:      "acceptedAgreementId",
			Contribution:     "path",
			PreferResourceID: true,
		},
	}
}

func acceptedAgreementListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
		},
		{
			FieldName:    "DisplayName",
			RequestName:  "displayName",
			Contribution: "query",
		},
		{
			FieldName:    "ListingId",
			RequestName:  "listingId",
			Contribution: "query",
		},
		{
			FieldName:    "PackageVersion",
			RequestName:  "packageVersion",
			Contribution: "query",
		},
		{
			FieldName:        "AcceptedAgreementId",
			RequestName:      "acceptedAgreementId",
			Contribution:     "query",
			PreferResourceID: true,
		},
		{
			FieldName:    "SortBy",
			RequestName:  "sortBy",
			Contribution: "query",
		},
		{
			FieldName:    "SortOrder",
			RequestName:  "sortOrder",
			Contribution: "query",
		},
		{
			FieldName:    "Limit",
			RequestName:  "limit",
			Contribution: "query",
		},
		{
			FieldName:    "Page",
			RequestName:  "page",
			Contribution: "query",
		},
	}
}

func acceptedAgreementUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:        "AcceptedAgreementId",
			RequestName:      "acceptedAgreementId",
			Contribution:     "path",
			PreferResourceID: true,
		},
		{
			FieldName:    "UpdateAcceptedAgreementDetails",
			RequestName:  "UpdateAcceptedAgreementDetails",
			Contribution: "body",
		},
	}
}

func acceptedAgreementDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:        "AcceptedAgreementId",
			RequestName:      "acceptedAgreementId",
			Contribution:     "path",
			PreferResourceID: true,
		},
		{
			FieldName:    "Signature",
			RequestName:  "signature",
			Contribution: "query",
		},
	}
}

func (c *acceptedAgreementRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *marketplacev1beta1.AcceptedAgreement,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("AcceptedAgreement delegate is not configured")
	}
	if c.initErr != nil {
		return c.delegate.CreateOrUpdate(ctx, resource, req)
	}

	if err := validateAcceptedAgreementSignature(resource); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
	}

	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err != nil {
		return response, err
	}

	stampAcceptedAgreementSignature(resource)
	markAcceptedAgreementActive(resource, c.log)
	response.ShouldRequeue = false
	response.RequeueDuration = 0
	return response, nil
}

func (c *acceptedAgreementRuntimeClient) Delete(ctx context.Context, resource *marketplacev1beta1.AcceptedAgreement) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("AcceptedAgreement delegate is not configured")
	}

	deleted, err := c.delegate.Delete(ctx, resource)
	if deleted {
		resource.Status.AppliedSignature = ""
	}
	return deleted, err
}

func validateAcceptedAgreementSignature(resource *marketplacev1beta1.AcceptedAgreement) error {
	applied := strings.TrimSpace(resource.Status.AppliedSignature)
	if applied == "" {
		return nil
	}
	if desired := strings.TrimSpace(resource.Spec.Signature); desired != applied {
		return fmt.Errorf("AcceptedAgreement formal semantics require replacement when signature changes")
	}
	return nil
}

func stampAcceptedAgreementSignature(resource *marketplacev1beta1.AcceptedAgreement) {
	resource.Status.AppliedSignature = strings.TrimSpace(resource.Spec.Signature)
}

func markAcceptedAgreementActive(resource *marketplacev1beta1.AcceptedAgreement, log loggerutil.OSOKLogger) {
	status := &resource.Status.OsokStatus
	servicemanager.ClearAsyncOperation(status)

	message := strings.TrimSpace(status.Message)
	if message == "" {
		message = strings.TrimSpace(resource.Status.DisplayName)
	}
	if message == "" {
		message = "OCI accepted agreement is present"
	}

	now := metav1.Now()
	if status.CreatedAt == nil && strings.TrimSpace(string(status.Ocid)) != "" {
		status.CreatedAt = &now
	}
	status.Message = message
	status.Reason = string(shared.Active)
	status.UpdatedAt = &now
	*status = util.UpdateOSOKStatusCondition(*status, shared.Active, v1.ConditionTrue, "", message, log)
}

func (c *acceptedAgreementRuntimeClient) fail(resource *marketplacev1beta1.AcceptedAgreement, err error) error {
	status := &resource.Status.OsokStatus
	servicemanager.ClearAsyncOperation(status)
	servicemanager.RecordErrorOpcRequestID(status, err)
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	now := metav1.Now()
	status.UpdatedAt = &now
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, v1.ConditionFalse, "", err.Error(), c.log)
	return err
}

var _ AcceptedAgreementServiceClient = (*acceptedAgreementRuntimeClient)(nil)

var _ acceptedAgreementOCIClient = (*marketplacesdk.MarketplaceClient)(nil)

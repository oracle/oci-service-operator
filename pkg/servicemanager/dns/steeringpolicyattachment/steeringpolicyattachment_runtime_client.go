/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package steeringpolicyattachment

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	dnssdk "github.com/oracle/oci-go-sdk/v65/dns"
	dnsv1beta1 "github.com/oracle/oci-service-operator/api/dns/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

type steeringPolicyAttachmentOCIClient interface {
	CreateSteeringPolicyAttachment(context.Context, dnssdk.CreateSteeringPolicyAttachmentRequest) (dnssdk.CreateSteeringPolicyAttachmentResponse, error)
	GetSteeringPolicyAttachment(context.Context, dnssdk.GetSteeringPolicyAttachmentRequest) (dnssdk.GetSteeringPolicyAttachmentResponse, error)
	ListSteeringPolicyAttachments(context.Context, dnssdk.ListSteeringPolicyAttachmentsRequest) (dnssdk.ListSteeringPolicyAttachmentsResponse, error)
	UpdateSteeringPolicyAttachment(context.Context, dnssdk.UpdateSteeringPolicyAttachmentRequest) (dnssdk.UpdateSteeringPolicyAttachmentResponse, error)
	DeleteSteeringPolicyAttachment(context.Context, dnssdk.DeleteSteeringPolicyAttachmentRequest) (dnssdk.DeleteSteeringPolicyAttachmentResponse, error)
	GetSteeringPolicy(context.Context, dnssdk.GetSteeringPolicyRequest) (dnssdk.GetSteeringPolicyResponse, error)
}

type ambiguousSteeringPolicyAttachmentNotFoundError struct {
	message      string
	opcRequestID string
}

func (e ambiguousSteeringPolicyAttachmentNotFoundError) Error() string {
	return e.message
}

func (e ambiguousSteeringPolicyAttachmentNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerSteeringPolicyAttachmentRuntimeHooksMutator(func(manager *SteeringPolicyAttachmentServiceManager, hooks *SteeringPolicyAttachmentRuntimeHooks) {
		client, initErr := newSteeringPolicyAttachmentSDKClient(manager)
		applySteeringPolicyAttachmentRuntimeHooks(hooks, client, initErr)
	})
}

func newSteeringPolicyAttachmentSDKClient(manager *SteeringPolicyAttachmentServiceManager) (steeringPolicyAttachmentOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("SteeringPolicyAttachment service manager is nil")
	}
	client, err := dnssdk.NewDnsClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applySteeringPolicyAttachmentRuntimeHooks(
	hooks *SteeringPolicyAttachmentRuntimeHooks,
	client steeringPolicyAttachmentOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = steeringPolicyAttachmentRuntimeSemantics()
	hooks.BuildCreateBody = buildSteeringPolicyAttachmentCreateBody
	hooks.List.Fields = steeringPolicyAttachmentListFields()
	hooks.Create.Call = func(ctx context.Context, request dnssdk.CreateSteeringPolicyAttachmentRequest) (dnssdk.CreateSteeringPolicyAttachmentResponse, error) {
		if err := ensureSteeringPolicyAttachmentOCIClient(client, initErr); err != nil {
			return dnssdk.CreateSteeringPolicyAttachmentResponse{}, err
		}
		request.Scope = defaultCreateSteeringPolicyAttachmentScope(request.Scope)
		return client.CreateSteeringPolicyAttachment(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request dnssdk.GetSteeringPolicyAttachmentRequest) (dnssdk.GetSteeringPolicyAttachmentResponse, error) {
		if err := ensureSteeringPolicyAttachmentOCIClient(client, initErr); err != nil {
			return dnssdk.GetSteeringPolicyAttachmentResponse{}, err
		}
		request.Scope = defaultGetSteeringPolicyAttachmentScope(request.Scope)
		response, err := client.GetSteeringPolicyAttachment(ctx, request)
		return response, conservativeSteeringPolicyAttachmentNotFoundError(err, "read")
	}
	hooks.List.Call = func(ctx context.Context, request dnssdk.ListSteeringPolicyAttachmentsRequest) (dnssdk.ListSteeringPolicyAttachmentsResponse, error) {
		return listSteeringPolicyAttachmentsAllPages(ctx, client, initErr, request)
	}
	hooks.Update.Call = func(ctx context.Context, request dnssdk.UpdateSteeringPolicyAttachmentRequest) (dnssdk.UpdateSteeringPolicyAttachmentResponse, error) {
		if err := ensureSteeringPolicyAttachmentOCIClient(client, initErr); err != nil {
			return dnssdk.UpdateSteeringPolicyAttachmentResponse{}, err
		}
		request.Scope = defaultUpdateSteeringPolicyAttachmentScope(request.Scope)
		return client.UpdateSteeringPolicyAttachment(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request dnssdk.DeleteSteeringPolicyAttachmentRequest) (dnssdk.DeleteSteeringPolicyAttachmentResponse, error) {
		if err := ensureSteeringPolicyAttachmentOCIClient(client, initErr); err != nil {
			return dnssdk.DeleteSteeringPolicyAttachmentResponse{}, err
		}
		request.Scope = defaultDeleteSteeringPolicyAttachmentScope(request.Scope)
		response, err := client.DeleteSteeringPolicyAttachment(ctx, request)
		return response, conservativeSteeringPolicyAttachmentNotFoundError(err, "delete")
	}
	hooks.ParityHooks.NormalizeDesiredState = normalizeSteeringPolicyAttachmentDesiredState
	hooks.DeleteHooks.HandleError = handleSteeringPolicyAttachmentDeleteError
}

func steeringPolicyAttachmentRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "dns",
		FormalSlug:    "steeringpolicyattachment",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(dnssdk.SteeringPolicyAttachmentLifecycleStateCreating)},
			ActiveStates:       []string{string(dnssdk.SteeringPolicyAttachmentLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(dnssdk.SteeringPolicyAttachmentLifecycleStateDeleting)},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"steeringPolicyId", "zoneId", "domainName"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"displayName"},
			ForceNew:      []string{"steeringPolicyId", "zoneId", "domainName"},
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

func steeringPolicyAttachmentListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "compartmentId"},
		},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Id", RequestName: "id", Contribution: "query"},
		{
			FieldName:    "SteeringPolicyId",
			RequestName:  "steeringPolicyId",
			Contribution: "query",
			LookupPaths:  []string{"status.steeringPolicyId", "spec.steeringPolicyId", "steeringPolicyId"},
		},
		{
			FieldName:    "ZoneId",
			RequestName:  "zoneId",
			Contribution: "query",
			LookupPaths:  []string{"status.zoneId", "spec.zoneId", "zoneId"},
		},
		{
			FieldName:    "Domain",
			RequestName:  "domain",
			Contribution: "query",
			LookupPaths:  []string{"status.domainName", "spec.domainName", "domainName"},
		},
	}
}

func buildSteeringPolicyAttachmentCreateBody(_ context.Context, resource *dnsv1beta1.SteeringPolicyAttachment, _ string) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("SteeringPolicyAttachment resource is nil")
	}
	normalizeSteeringPolicyAttachmentSpec(resource)
	if resource.Spec.SteeringPolicyId == "" {
		return nil, fmt.Errorf("steeringPolicyId is required")
	}
	if resource.Spec.ZoneId == "" {
		return nil, fmt.Errorf("zoneId is required")
	}
	if resource.Spec.DomainName == "" {
		return nil, fmt.Errorf("domainName is required")
	}

	body := dnssdk.CreateSteeringPolicyAttachmentDetails{
		SteeringPolicyId: common.String(resource.Spec.SteeringPolicyId),
		ZoneId:           common.String(resource.Spec.ZoneId),
		DomainName:       common.String(resource.Spec.DomainName),
	}
	if resource.Spec.DisplayName != "" {
		body.DisplayName = common.String(resource.Spec.DisplayName)
	}
	return body, nil
}

func normalizeSteeringPolicyAttachmentDesiredState(resource *dnsv1beta1.SteeringPolicyAttachment, _ any) {
	normalizeSteeringPolicyAttachmentSpec(resource)
}

func normalizeSteeringPolicyAttachmentSpec(resource *dnsv1beta1.SteeringPolicyAttachment) {
	if resource == nil {
		return
	}
	resource.Spec.SteeringPolicyId = strings.TrimSpace(resource.Spec.SteeringPolicyId)
	resource.Spec.ZoneId = strings.TrimSpace(resource.Spec.ZoneId)
	resource.Spec.DomainName = strings.TrimSpace(resource.Spec.DomainName)
	resource.Spec.DisplayName = strings.TrimSpace(resource.Spec.DisplayName)
}

func listSteeringPolicyAttachmentsAllPages(
	ctx context.Context,
	client steeringPolicyAttachmentOCIClient,
	initErr error,
	request dnssdk.ListSteeringPolicyAttachmentsRequest,
) (dnssdk.ListSteeringPolicyAttachmentsResponse, error) {
	if err := ensureSteeringPolicyAttachmentOCIClient(client, initErr); err != nil {
		return dnssdk.ListSteeringPolicyAttachmentsResponse{}, err
	}
	request.Scope = defaultListSteeringPolicyAttachmentScope(request.Scope)
	if request.CompartmentId == nil || strings.TrimSpace(*request.CompartmentId) == "" {
		compartmentID, err := resolveSteeringPolicyAttachmentCompartmentID(ctx, client, request.SteeringPolicyId)
		if err != nil {
			return dnssdk.ListSteeringPolicyAttachmentsResponse{}, err
		}
		request.CompartmentId = common.String(compartmentID)
	}

	var combined dnssdk.ListSteeringPolicyAttachmentsResponse
	for {
		response, err := client.ListSteeringPolicyAttachments(ctx, request)
		if err != nil {
			return combined, conservativeSteeringPolicyAttachmentNotFoundError(err, "list")
		}
		combined.Items = append(combined.Items, response.Items...)
		if combined.OpcRequestId == nil {
			combined.OpcRequestId = response.OpcRequestId
		}
		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			combined.OpcNextPage = nil
			break
		}
		request.Page = response.OpcNextPage
		combined.OpcNextPage = response.OpcNextPage
	}
	return combined, nil
}

func resolveSteeringPolicyAttachmentCompartmentID(
	ctx context.Context,
	client steeringPolicyAttachmentOCIClient,
	steeringPolicyID *string,
) (string, error) {
	if steeringPolicyID == nil || strings.TrimSpace(*steeringPolicyID) == "" {
		return "", fmt.Errorf("compartmentId is required to list SteeringPolicyAttachment when steeringPolicyId is empty")
	}
	response, err := client.GetSteeringPolicy(ctx, dnssdk.GetSteeringPolicyRequest{
		SteeringPolicyId: common.String(strings.TrimSpace(*steeringPolicyID)),
		Scope:            dnssdk.GetSteeringPolicyScopeGlobal,
	})
	if err != nil {
		return "", conservativeSteeringPolicyAttachmentNotFoundError(err, "resolve steering policy compartment")
	}
	if response.CompartmentId == nil || strings.TrimSpace(*response.CompartmentId) == "" {
		return "", fmt.Errorf("SteeringPolicyAttachment list compartment could not be resolved from steering policy %s", strings.TrimSpace(*steeringPolicyID))
	}
	return strings.TrimSpace(*response.CompartmentId), nil
}

func ensureSteeringPolicyAttachmentOCIClient(client steeringPolicyAttachmentOCIClient, initErr error) error {
	if initErr != nil {
		return fmt.Errorf("initialize SteeringPolicyAttachment OCI client: %w", initErr)
	}
	if client == nil {
		return fmt.Errorf("SteeringPolicyAttachment OCI client is not configured")
	}
	return nil
}

func handleSteeringPolicyAttachmentDeleteError(resource *dnsv1beta1.SteeringPolicyAttachment, err error) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return err
}

func conservativeSteeringPolicyAttachmentNotFoundError(err error, operation string) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	message := fmt.Sprintf("SteeringPolicyAttachment %s returned ambiguous 404 NotAuthorizedOrNotFound: %s", strings.TrimSpace(operation), err.Error())
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		return ambiguousSteeringPolicyAttachmentNotFoundError{message: message, opcRequestID: serviceErr.GetOpcRequestID()}
	}
	return ambiguousSteeringPolicyAttachmentNotFoundError{message: message}
}

func defaultCreateSteeringPolicyAttachmentScope(scope dnssdk.CreateSteeringPolicyAttachmentScopeEnum) dnssdk.CreateSteeringPolicyAttachmentScopeEnum {
	if scope == "" {
		return dnssdk.CreateSteeringPolicyAttachmentScopeGlobal
	}
	return scope
}

func defaultGetSteeringPolicyAttachmentScope(scope dnssdk.GetSteeringPolicyAttachmentScopeEnum) dnssdk.GetSteeringPolicyAttachmentScopeEnum {
	if scope == "" {
		return dnssdk.GetSteeringPolicyAttachmentScopeGlobal
	}
	return scope
}

func defaultListSteeringPolicyAttachmentScope(scope dnssdk.ListSteeringPolicyAttachmentsScopeEnum) dnssdk.ListSteeringPolicyAttachmentsScopeEnum {
	if scope == "" {
		return dnssdk.ListSteeringPolicyAttachmentsScopeGlobal
	}
	return scope
}

func defaultUpdateSteeringPolicyAttachmentScope(scope dnssdk.UpdateSteeringPolicyAttachmentScopeEnum) dnssdk.UpdateSteeringPolicyAttachmentScopeEnum {
	if scope == "" {
		return dnssdk.UpdateSteeringPolicyAttachmentScopeGlobal
	}
	return scope
}

func defaultDeleteSteeringPolicyAttachmentScope(scope dnssdk.DeleteSteeringPolicyAttachmentScopeEnum) dnssdk.DeleteSteeringPolicyAttachmentScopeEnum {
	if scope == "" {
		return dnssdk.DeleteSteeringPolicyAttachmentScopeGlobal
	}
	return scope
}

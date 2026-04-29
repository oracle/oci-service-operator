/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package tenancyattachment

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	resourceanalyticssdk "github.com/oracle/oci-go-sdk/v65/resourceanalytics"
	resourceanalyticsv1beta1 "github.com/oracle/oci-service-operator/api/resourceanalytics/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

type tenancyAttachmentOCIClient interface {
	CreateTenancyAttachment(context.Context, resourceanalyticssdk.CreateTenancyAttachmentRequest) (resourceanalyticssdk.CreateTenancyAttachmentResponse, error)
	GetTenancyAttachment(context.Context, resourceanalyticssdk.GetTenancyAttachmentRequest) (resourceanalyticssdk.GetTenancyAttachmentResponse, error)
	ListTenancyAttachments(context.Context, resourceanalyticssdk.ListTenancyAttachmentsRequest) (resourceanalyticssdk.ListTenancyAttachmentsResponse, error)
	UpdateTenancyAttachment(context.Context, resourceanalyticssdk.UpdateTenancyAttachmentRequest) (resourceanalyticssdk.UpdateTenancyAttachmentResponse, error)
	DeleteTenancyAttachment(context.Context, resourceanalyticssdk.DeleteTenancyAttachmentRequest) (resourceanalyticssdk.DeleteTenancyAttachmentResponse, error)
}

type ambiguousTenancyAttachmentNotFoundError struct {
	message      string
	opcRequestID string
}

func (e ambiguousTenancyAttachmentNotFoundError) Error() string {
	return e.message
}

func (e ambiguousTenancyAttachmentNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerTenancyAttachmentRuntimeHooksMutator(func(_ *TenancyAttachmentServiceManager, hooks *TenancyAttachmentRuntimeHooks) {
		applyTenancyAttachmentRuntimeHooks(hooks)
	})
}

func applyTenancyAttachmentRuntimeHooks(hooks *TenancyAttachmentRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = tenancyAttachmentRuntimeSemantics()
	hooks.BuildCreateBody = buildTenancyAttachmentCreateBody
	hooks.BuildUpdateBody = buildTenancyAttachmentUpdateBody
	hooks.List.Fields = tenancyAttachmentListFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listTenancyAttachmentsAllPages(hooks.List.Call)
	}
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateTenancyAttachmentCreateOnlyDrift
	hooks.DeleteHooks.HandleError = handleTenancyAttachmentDeleteError
	wrapTenancyAttachmentDeleteConfirmation(hooks)
}

func newTenancyAttachmentServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client tenancyAttachmentOCIClient,
) TenancyAttachmentServiceClient {
	hooks := newTenancyAttachmentRuntimeHooksWithOCIClient(client)
	applyTenancyAttachmentRuntimeHooks(&hooks)
	delegate := defaultTenancyAttachmentServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*resourceanalyticsv1beta1.TenancyAttachment](
			buildTenancyAttachmentGeneratedRuntimeConfig(&TenancyAttachmentServiceManager{Log: log}, hooks),
		),
	}
	return wrapTenancyAttachmentGeneratedClient(hooks, delegate)
}

func newTenancyAttachmentRuntimeHooksWithOCIClient(client tenancyAttachmentOCIClient) TenancyAttachmentRuntimeHooks {
	return TenancyAttachmentRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*resourceanalyticsv1beta1.TenancyAttachment]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*resourceanalyticsv1beta1.TenancyAttachment]{},
		StatusHooks:     generatedruntime.StatusHooks[*resourceanalyticsv1beta1.TenancyAttachment]{},
		ParityHooks:     generatedruntime.ParityHooks[*resourceanalyticsv1beta1.TenancyAttachment]{},
		Async:           generatedruntime.AsyncHooks[*resourceanalyticsv1beta1.TenancyAttachment]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*resourceanalyticsv1beta1.TenancyAttachment]{},
		Create: runtimeOperationHooks[resourceanalyticssdk.CreateTenancyAttachmentRequest, resourceanalyticssdk.CreateTenancyAttachmentResponse]{
			Fields: tenancyAttachmentCreateFields(),
			Call: func(ctx context.Context, request resourceanalyticssdk.CreateTenancyAttachmentRequest) (resourceanalyticssdk.CreateTenancyAttachmentResponse, error) {
				if client == nil {
					return resourceanalyticssdk.CreateTenancyAttachmentResponse{}, fmt.Errorf("tenancyattachment OCI client is nil")
				}
				return client.CreateTenancyAttachment(ctx, request)
			},
		},
		Get: runtimeOperationHooks[resourceanalyticssdk.GetTenancyAttachmentRequest, resourceanalyticssdk.GetTenancyAttachmentResponse]{
			Fields: tenancyAttachmentGetFields(),
			Call: func(ctx context.Context, request resourceanalyticssdk.GetTenancyAttachmentRequest) (resourceanalyticssdk.GetTenancyAttachmentResponse, error) {
				if client == nil {
					return resourceanalyticssdk.GetTenancyAttachmentResponse{}, fmt.Errorf("tenancyattachment OCI client is nil")
				}
				return client.GetTenancyAttachment(ctx, request)
			},
		},
		List: runtimeOperationHooks[resourceanalyticssdk.ListTenancyAttachmentsRequest, resourceanalyticssdk.ListTenancyAttachmentsResponse]{
			Fields: tenancyAttachmentListFields(),
			Call: func(ctx context.Context, request resourceanalyticssdk.ListTenancyAttachmentsRequest) (resourceanalyticssdk.ListTenancyAttachmentsResponse, error) {
				if client == nil {
					return resourceanalyticssdk.ListTenancyAttachmentsResponse{}, fmt.Errorf("tenancyattachment OCI client is nil")
				}
				return client.ListTenancyAttachments(ctx, request)
			},
		},
		Update: runtimeOperationHooks[resourceanalyticssdk.UpdateTenancyAttachmentRequest, resourceanalyticssdk.UpdateTenancyAttachmentResponse]{
			Fields: tenancyAttachmentUpdateFields(),
			Call: func(ctx context.Context, request resourceanalyticssdk.UpdateTenancyAttachmentRequest) (resourceanalyticssdk.UpdateTenancyAttachmentResponse, error) {
				if client == nil {
					return resourceanalyticssdk.UpdateTenancyAttachmentResponse{}, fmt.Errorf("tenancyattachment OCI client is nil")
				}
				return client.UpdateTenancyAttachment(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[resourceanalyticssdk.DeleteTenancyAttachmentRequest, resourceanalyticssdk.DeleteTenancyAttachmentResponse]{
			Fields: tenancyAttachmentDeleteFields(),
			Call: func(ctx context.Context, request resourceanalyticssdk.DeleteTenancyAttachmentRequest) (resourceanalyticssdk.DeleteTenancyAttachmentResponse, error) {
				if client == nil {
					return resourceanalyticssdk.DeleteTenancyAttachmentResponse{}, fmt.Errorf("tenancyattachment OCI client is nil")
				}
				return client.DeleteTenancyAttachment(ctx, request)
			},
		},
		WrapGeneratedClient: []func(TenancyAttachmentServiceClient) TenancyAttachmentServiceClient{},
	}
}

func tenancyAttachmentRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "resourceanalytics",
		FormalSlug:    "tenancyattachment",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(resourceanalyticssdk.TenancyAttachmentLifecycleStateCreating)},
			UpdatingStates:     []string{string(resourceanalyticssdk.TenancyAttachmentLifecycleStateUpdating)},
			ActiveStates:       []string{string(resourceanalyticssdk.TenancyAttachmentLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(resourceanalyticssdk.TenancyAttachmentLifecycleStateDeleting)},
			TerminalStates: []string{string(resourceanalyticssdk.TenancyAttachmentLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"resourceAnalyticsInstanceId", "tenancyId", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"description"},
			ForceNew:      []string{"resourceAnalyticsInstanceId", "tenancyId"},
			ConflictsWith: map[string][]string{},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
		},
	}
}

func tenancyAttachmentCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateTenancyAttachmentDetails", RequestName: "CreateTenancyAttachmentDetails", Contribution: "body"},
	}
}

func tenancyAttachmentGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "TenancyAttachmentId", RequestName: "tenancyAttachmentId", Contribution: "path", PreferResourceID: true},
	}
}

func tenancyAttachmentListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "ResourceAnalyticsInstanceId",
			RequestName:  "resourceAnalyticsInstanceId",
			Contribution: "query",
			LookupPaths:  []string{"status.resourceAnalyticsInstanceId", "spec.resourceAnalyticsInstanceId", "resourceAnalyticsInstanceId"},
		},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
	}
}

func tenancyAttachmentUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "TenancyAttachmentId", RequestName: "tenancyAttachmentId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateTenancyAttachmentDetails", RequestName: "UpdateTenancyAttachmentDetails", Contribution: "body"},
	}
}

func tenancyAttachmentDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "TenancyAttachmentId", RequestName: "tenancyAttachmentId", Contribution: "path", PreferResourceID: true},
	}
}

func buildTenancyAttachmentCreateBody(
	_ context.Context,
	resource *resourceanalyticsv1beta1.TenancyAttachment,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("tenancyattachment resource is nil")
	}
	if err := validateTenancyAttachmentSpec(resource.Spec); err != nil {
		return nil, err
	}

	details := resourceanalyticssdk.CreateTenancyAttachmentDetails{
		ResourceAnalyticsInstanceId: common.String(strings.TrimSpace(resource.Spec.ResourceAnalyticsInstanceId)),
		TenancyId:                   common.String(strings.TrimSpace(resource.Spec.TenancyId)),
	}
	if resource.Spec.Description != "" {
		details.Description = common.String(resource.Spec.Description)
	}
	return details, nil
}

func buildTenancyAttachmentUpdateBody(
	_ context.Context,
	resource *resourceanalyticsv1beta1.TenancyAttachment,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return resourceanalyticssdk.UpdateTenancyAttachmentDetails{}, false, fmt.Errorf("tenancyattachment resource is nil")
	}
	if err := validateTenancyAttachmentSpec(resource.Spec); err != nil {
		return resourceanalyticssdk.UpdateTenancyAttachmentDetails{}, false, err
	}
	if err := validateTenancyAttachmentCreateOnlyDrift(resource, currentResponse); err != nil {
		return resourceanalyticssdk.UpdateTenancyAttachmentDetails{}, false, err
	}

	current, err := tenancyAttachmentRuntimeBody(currentResponse)
	if err != nil {
		return resourceanalyticssdk.UpdateTenancyAttachmentDetails{}, false, err
	}

	if !tenancyAttachmentDescriptionDrift(resource.Spec.Description, current.Description) {
		return resourceanalyticssdk.UpdateTenancyAttachmentDetails{}, false, nil
	}
	return resourceanalyticssdk.UpdateTenancyAttachmentDetails{
		Description: common.String(resource.Spec.Description),
	}, true, nil
}

func validateTenancyAttachmentSpec(spec resourceanalyticsv1beta1.TenancyAttachmentSpec) error {
	var missing []string
	if strings.TrimSpace(spec.ResourceAnalyticsInstanceId) == "" {
		missing = append(missing, "resourceAnalyticsInstanceId")
	}
	if strings.TrimSpace(spec.TenancyId) == "" {
		missing = append(missing, "tenancyId")
	}
	if len(missing) != 0 {
		return fmt.Errorf("tenancyattachment spec is missing required field(s): %s", strings.Join(missing, ", "))
	}
	return nil
}

func validateTenancyAttachmentCreateOnlyDrift(
	resource *resourceanalyticsv1beta1.TenancyAttachment,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("tenancyattachment resource is nil")
	}
	current, err := tenancyAttachmentRuntimeBody(currentResponse)
	if err != nil {
		return err
	}

	var drift []string
	if tenancyAttachmentStringDrift(resource.Spec.ResourceAnalyticsInstanceId, current.ResourceAnalyticsInstanceId) {
		drift = append(drift, "resourceAnalyticsInstanceId")
	}
	if tenancyAttachmentStringDrift(resource.Spec.TenancyId, current.TenancyId) {
		drift = append(drift, "tenancyId")
	}
	if len(drift) == 0 {
		return nil
	}
	return fmt.Errorf("tenancyattachment create-only drift detected for %s; replace the resource or restore the desired spec before update", strings.Join(drift, ", "))
}

func tenancyAttachmentRuntimeBody(currentResponse any) (resourceanalyticssdk.TenancyAttachment, error) {
	if current, ok, err := tenancyAttachmentDirectRuntimeBody(currentResponse); ok || err != nil {
		return current, err
	}
	if current, ok, err := tenancyAttachmentResponseRuntimeBody(currentResponse); ok || err != nil {
		return current, err
	}
	return resourceanalyticssdk.TenancyAttachment{}, fmt.Errorf("unexpected current TenancyAttachment response type %T", currentResponse)
}

func tenancyAttachmentDirectRuntimeBody(currentResponse any) (resourceanalyticssdk.TenancyAttachment, bool, error) {
	switch current := currentResponse.(type) {
	case resourceanalyticssdk.TenancyAttachment:
		return current, true, nil
	case *resourceanalyticssdk.TenancyAttachment:
		body, err := dereferenceTenancyAttachmentRuntimeBody(current)
		return body, true, err
	case resourceanalyticssdk.TenancyAttachmentSummary:
		return tenancyAttachmentFromSummary(current), true, nil
	case *resourceanalyticssdk.TenancyAttachmentSummary:
		summary, err := dereferenceTenancyAttachmentRuntimeBody(current)
		if err != nil {
			return resourceanalyticssdk.TenancyAttachment{}, true, err
		}
		return tenancyAttachmentFromSummary(summary), true, nil
	default:
		return resourceanalyticssdk.TenancyAttachment{}, false, nil
	}
}

func tenancyAttachmentResponseRuntimeBody(currentResponse any) (resourceanalyticssdk.TenancyAttachment, bool, error) {
	switch current := currentResponse.(type) {
	case resourceanalyticssdk.CreateTenancyAttachmentResponse:
		return current.TenancyAttachment, true, nil
	case *resourceanalyticssdk.CreateTenancyAttachmentResponse:
		response, err := dereferenceTenancyAttachmentRuntimeBody(current)
		if err != nil {
			return resourceanalyticssdk.TenancyAttachment{}, true, err
		}
		return response.TenancyAttachment, true, nil
	case resourceanalyticssdk.GetTenancyAttachmentResponse:
		return current.TenancyAttachment, true, nil
	case *resourceanalyticssdk.GetTenancyAttachmentResponse:
		response, err := dereferenceTenancyAttachmentRuntimeBody(current)
		if err != nil {
			return resourceanalyticssdk.TenancyAttachment{}, true, err
		}
		return response.TenancyAttachment, true, nil
	default:
		return resourceanalyticssdk.TenancyAttachment{}, false, nil
	}
}

func dereferenceTenancyAttachmentRuntimeBody[T any](current *T) (T, error) {
	if current == nil {
		var zero T
		return zero, fmt.Errorf("current tenancyattachment response is nil")
	}
	return *current, nil
}

func tenancyAttachmentFromSummary(summary resourceanalyticssdk.TenancyAttachmentSummary) resourceanalyticssdk.TenancyAttachment {
	return resourceanalyticssdk.TenancyAttachment(summary)
}

func tenancyAttachmentStringDrift(spec string, current *string) bool {
	desired := strings.TrimSpace(spec)
	observed := strings.TrimSpace(tenancyAttachmentStringValue(current))
	return desired != "" && observed != "" && desired != observed
}

func tenancyAttachmentDescriptionDrift(spec string, current *string) bool {
	if current == nil {
		return spec != ""
	}
	return spec != *current
}

func listTenancyAttachmentsAllPages(
	call func(context.Context, resourceanalyticssdk.ListTenancyAttachmentsRequest) (resourceanalyticssdk.ListTenancyAttachmentsResponse, error),
) func(context.Context, resourceanalyticssdk.ListTenancyAttachmentsRequest) (resourceanalyticssdk.ListTenancyAttachmentsResponse, error) {
	return func(ctx context.Context, request resourceanalyticssdk.ListTenancyAttachmentsRequest) (resourceanalyticssdk.ListTenancyAttachmentsResponse, error) {
		var combined resourceanalyticssdk.ListTenancyAttachmentsResponse
		seenPages := map[string]struct{}{}

		for {
			response, err := call(ctx, request)
			if err != nil {
				return resourceanalyticssdk.ListTenancyAttachmentsResponse{}, err
			}
			combined.RawResponse = response.RawResponse
			if combined.OpcRequestId == nil {
				combined.OpcRequestId = response.OpcRequestId
			}
			combined.Items = append(combined.Items, response.Items...)

			nextPage := strings.TrimSpace(tenancyAttachmentStringValue(response.OpcNextPage))
			if nextPage == "" {
				combined.OpcNextPage = nil
				return combined, nil
			}
			if _, ok := seenPages[nextPage]; ok {
				return resourceanalyticssdk.ListTenancyAttachmentsResponse{}, fmt.Errorf("tenancyattachment list pagination repeated page token %q", nextPage)
			}
			seenPages[nextPage] = struct{}{}
			request.Page = response.OpcNextPage
			combined.OpcNextPage = response.OpcNextPage
		}
	}
}

func handleTenancyAttachmentDeleteError(resource *resourceanalyticsv1beta1.TenancyAttachment, err error) error {
	if err == nil {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if !isAmbiguousTenancyAttachmentNotFound(err) {
		return err
	}
	return ambiguousTenancyAttachmentNotFound("delete path", err)
}

func wrapTenancyAttachmentDeleteConfirmation(hooks *TenancyAttachmentRuntimeHooks) {
	if hooks == nil || hooks.Get.Call == nil {
		return
	}
	getTenancyAttachment := hooks.Get.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate TenancyAttachmentServiceClient) TenancyAttachmentServiceClient {
		return tenancyAttachmentDeleteConfirmationClient{
			delegate:             delegate,
			getTenancyAttachment: getTenancyAttachment,
		}
	})
}

type tenancyAttachmentDeleteConfirmationClient struct {
	delegate             TenancyAttachmentServiceClient
	getTenancyAttachment func(context.Context, resourceanalyticssdk.GetTenancyAttachmentRequest) (resourceanalyticssdk.GetTenancyAttachmentResponse, error)
}

func (c tenancyAttachmentDeleteConfirmationClient) CreateOrUpdate(
	ctx context.Context,
	resource *resourceanalyticsv1beta1.TenancyAttachment,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c tenancyAttachmentDeleteConfirmationClient) Delete(
	ctx context.Context,
	resource *resourceanalyticsv1beta1.TenancyAttachment,
) (bool, error) {
	if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c tenancyAttachmentDeleteConfirmationClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *resourceanalyticsv1beta1.TenancyAttachment,
) error {
	if c.getTenancyAttachment == nil || resource == nil {
		return nil
	}
	tenancyAttachmentID := trackedTenancyAttachmentID(resource)
	if tenancyAttachmentID == "" {
		return nil
	}

	_, err := c.getTenancyAttachment(ctx, resourceanalyticssdk.GetTenancyAttachmentRequest{
		TenancyAttachmentId: common.String(tenancyAttachmentID),
	})
	if err == nil {
		return nil
	}
	if !isAmbiguousTenancyAttachmentNotFound(err) {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return ambiguousTenancyAttachmentNotFound("delete confirmation", err)
}

func trackedTenancyAttachmentID(resource *resourceanalyticsv1beta1.TenancyAttachment) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func isAmbiguousTenancyAttachmentNotFound(err error) bool {
	if err == nil {
		return false
	}
	var ambiguous ambiguousTenancyAttachmentNotFoundError
	if errors.As(err, &ambiguous) {
		return true
	}
	return errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound()
}

func ambiguousTenancyAttachmentNotFound(operation string, err error) ambiguousTenancyAttachmentNotFoundError {
	var ambiguous ambiguousTenancyAttachmentNotFoundError
	if errors.As(err, &ambiguous) {
		return ambiguous
	}

	message := fmt.Sprintf("tenancyattachment %s returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %s", strings.TrimSpace(operation), err.Error())
	return ambiguousTenancyAttachmentNotFoundError{
		message:      message,
		opcRequestID: errorutil.OpcRequestID(err),
	}
}

func tenancyAttachmentStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

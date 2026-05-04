/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package attachment

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	marketplaceprivateoffersdk "github.com/oracle/oci-go-sdk/v65/marketplaceprivateoffer"
	marketplaceprivateofferv1beta1 "github.com/oracle/oci-service-operator/api/marketplaceprivateoffer/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	attachmentOfferIDAnnotation = "marketplaceprivateoffer.oracle.com/offer-id"
	attachmentFileSHA256Key     = "attachmentFileSHA256="
)

type attachmentOCIClient interface {
	CreateAttachment(context.Context, marketplaceprivateoffersdk.CreateAttachmentRequest) (marketplaceprivateoffersdk.CreateAttachmentResponse, error)
	GetAttachment(context.Context, marketplaceprivateoffersdk.GetAttachmentRequest) (marketplaceprivateoffersdk.GetAttachmentResponse, error)
	ListAttachments(context.Context, marketplaceprivateoffersdk.ListAttachmentsRequest) (marketplaceprivateoffersdk.ListAttachmentsResponse, error)
	DeleteAttachment(context.Context, marketplaceprivateoffersdk.DeleteAttachmentRequest) (marketplaceprivateoffersdk.DeleteAttachmentResponse, error)
}

type attachmentIdentity struct {
	offerID string
}

type ambiguousAttachmentNotFoundError struct {
	message      string
	opcRequestID string
}

func (e ambiguousAttachmentNotFoundError) Error() string {
	return e.message
}

func (e ambiguousAttachmentNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerAttachmentRuntimeHooksMutator(func(_ *AttachmentServiceManager, hooks *AttachmentRuntimeHooks) {
		applyAttachmentRuntimeHooks(hooks)
	})
}

func applyAttachmentRuntimeHooks(hooks *AttachmentRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = attachmentRuntimeSemantics()
	hooks.Identity = generatedruntime.IdentityHooks[*marketplaceprivateofferv1beta1.Attachment]{
		Resolve:       resolveAttachmentIdentity,
		RecordPath:    recordAttachmentPathIdentity,
		RecordTracked: recordAttachmentTrackedIdentity,
	}
	hooks.TrackedRecreate = generatedruntime.TrackedRecreateHooks[*marketplaceprivateofferv1beta1.Attachment]{
		ClearTrackedIdentity: clearAttachmentTrackedIdentity,
	}
	hooks.BuildCreateBody = buildAttachmentCreateBody
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateAttachmentCreateOnlyDrift
	hooks.DeleteHooks.HandleError = handleAttachmentDeleteError
	hooks.Create.Fields = attachmentCreateFields()
	hooks.Get.Fields = attachmentGetFields()
	hooks.List.Fields = attachmentListFields()
	hooks.Delete.Fields = attachmentDeleteFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listAttachmentsAllPages(hooks.List.Call)
	}
	wrapAttachmentLifecycleHooks(hooks)
}

func newAttachmentServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client attachmentOCIClient,
) AttachmentServiceClient {
	hooks := newAttachmentRuntimeHooksWithOCIClient(client)
	applyAttachmentRuntimeHooks(&hooks)
	delegate := defaultAttachmentServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*marketplaceprivateofferv1beta1.Attachment](
			buildAttachmentGeneratedRuntimeConfig(&AttachmentServiceManager{Log: log}, hooks),
		),
	}
	return wrapAttachmentGeneratedClient(hooks, delegate)
}

func newAttachmentRuntimeHooksWithOCIClient(client attachmentOCIClient) AttachmentRuntimeHooks {
	return AttachmentRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*marketplaceprivateofferv1beta1.Attachment]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*marketplaceprivateofferv1beta1.Attachment]{},
		StatusHooks:     generatedruntime.StatusHooks[*marketplaceprivateofferv1beta1.Attachment]{},
		ParityHooks:     generatedruntime.ParityHooks[*marketplaceprivateofferv1beta1.Attachment]{},
		Async:           generatedruntime.AsyncHooks[*marketplaceprivateofferv1beta1.Attachment]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*marketplaceprivateofferv1beta1.Attachment]{},
		Create: runtimeOperationHooks[marketplaceprivateoffersdk.CreateAttachmentRequest, marketplaceprivateoffersdk.CreateAttachmentResponse]{
			Fields: attachmentCreateFields(),
			Call: func(ctx context.Context, request marketplaceprivateoffersdk.CreateAttachmentRequest) (marketplaceprivateoffersdk.CreateAttachmentResponse, error) {
				if client == nil {
					return marketplaceprivateoffersdk.CreateAttachmentResponse{}, fmt.Errorf("attachment OCI client is nil")
				}
				return client.CreateAttachment(ctx, request)
			},
		},
		Get: runtimeOperationHooks[marketplaceprivateoffersdk.GetAttachmentRequest, marketplaceprivateoffersdk.GetAttachmentResponse]{
			Fields: attachmentGetFields(),
			Call: func(ctx context.Context, request marketplaceprivateoffersdk.GetAttachmentRequest) (marketplaceprivateoffersdk.GetAttachmentResponse, error) {
				if client == nil {
					return marketplaceprivateoffersdk.GetAttachmentResponse{}, fmt.Errorf("attachment OCI client is nil")
				}
				return client.GetAttachment(ctx, request)
			},
		},
		List: runtimeOperationHooks[marketplaceprivateoffersdk.ListAttachmentsRequest, marketplaceprivateoffersdk.ListAttachmentsResponse]{
			Fields: attachmentListFields(),
			Call: func(ctx context.Context, request marketplaceprivateoffersdk.ListAttachmentsRequest) (marketplaceprivateoffersdk.ListAttachmentsResponse, error) {
				if client == nil {
					return marketplaceprivateoffersdk.ListAttachmentsResponse{}, fmt.Errorf("attachment OCI client is nil")
				}
				return client.ListAttachments(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[marketplaceprivateoffersdk.DeleteAttachmentRequest, marketplaceprivateoffersdk.DeleteAttachmentResponse]{
			Fields: attachmentDeleteFields(),
			Call: func(ctx context.Context, request marketplaceprivateoffersdk.DeleteAttachmentRequest) (marketplaceprivateoffersdk.DeleteAttachmentResponse, error) {
				if client == nil {
					return marketplaceprivateoffersdk.DeleteAttachmentResponse{}, fmt.Errorf("attachment OCI client is nil")
				}
				return client.DeleteAttachment(ctx, request)
			},
		},
		WrapGeneratedClient: []func(AttachmentServiceClient) AttachmentServiceClient{},
	}
}

func attachmentRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "marketplaceprivateoffer",
		FormalSlug:        "attachment",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(marketplaceprivateoffersdk.AttachmentLifecycleStateCreating)},
			UpdatingStates:     []string{string(marketplaceprivateoffersdk.AttachmentLifecycleStateUpdating)},
			ActiveStates:       []string{string(marketplaceprivateoffersdk.AttachmentLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(marketplaceprivateoffersdk.AttachmentLifecycleStateDeleting)},
			TerminalStates: []string{string(marketplaceprivateoffersdk.AttachmentLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"offerId", "displayName", "type", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			ForceNew:      []string{"offerId"},
			ConflictsWith: map[string][]string{},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
		},
	}
}

func attachmentCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "OfferId",
			RequestName:  "offerId",
			Contribution: "path",
			LookupPaths:  attachmentOfferIDLookupPaths(),
		},
		{FieldName: "CreateAttachmentDetails", RequestName: "CreateAttachmentDetails", Contribution: "body"},
	}
}

func attachmentGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "OfferId",
			RequestName:  "offerId",
			Contribution: "path",
			LookupPaths:  attachmentOfferIDLookupPaths(),
		},
		{FieldName: "AttachmentId", RequestName: "attachmentId", Contribution: "path", PreferResourceID: true},
	}
}

func attachmentListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "OfferId",
			RequestName:  "offerId",
			Contribution: "path",
			LookupPaths:  attachmentOfferIDLookupPaths(),
		},
		{
			FieldName:    "DisplayName",
			RequestName:  "displayName",
			Contribution: "query",
			LookupPaths:  []string{"status.displayName", "spec.displayName", "displayName"},
		},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
	}
}

func attachmentDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "OfferId",
			RequestName:  "offerId",
			Contribution: "path",
			LookupPaths:  attachmentOfferIDLookupPaths(),
		},
		{FieldName: "AttachmentId", RequestName: "attachmentId", Contribution: "path", PreferResourceID: true},
	}
}

func attachmentOfferIDLookupPaths() []string {
	return []string{"status.offerId", "offerId"}
}

func resolveAttachmentIdentity(resource *marketplaceprivateofferv1beta1.Attachment) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("attachment resource is nil")
	}

	statusOfferID := strings.TrimSpace(resource.Status.OfferId)
	annotationOfferID := attachmentAnnotation(resource, attachmentOfferIDAnnotation)
	if statusOfferID != "" && annotationOfferID != "" && statusOfferID != annotationOfferID && resource.DeletionTimestamp.IsZero() {
		return nil, fmt.Errorf("attachment create-only parent offer annotation %q changed; create a replacement resource instead", attachmentOfferIDAnnotation)
	}

	offerID := firstNonEmptyAttachmentString(statusOfferID, annotationOfferID)
	if offerID == "" {
		if attachmentCurrentID(resource) == "" && !resource.DeletionTimestamp.IsZero() {
			return attachmentIdentity{}, nil
		}
		return nil, fmt.Errorf("attachment requires metadata annotation %q with the parent offer OCID because spec.offerId is not available", attachmentOfferIDAnnotation)
	}
	return attachmentIdentity{offerID: offerID}, nil
}

func recordAttachmentPathIdentity(resource *marketplaceprivateofferv1beta1.Attachment, identity any) {
	if resource == nil {
		return
	}
	typed, ok := identity.(attachmentIdentity)
	if !ok || strings.TrimSpace(typed.offerID) == "" {
		return
	}
	resource.Status.OfferId = typed.offerID
}

func recordAttachmentTrackedIdentity(resource *marketplaceprivateofferv1beta1.Attachment, identity any, resourceID string) {
	if resource == nil {
		return
	}
	if resourceID = strings.TrimSpace(resourceID); resourceID != "" {
		resource.Status.Id = resourceID
		resource.Status.OsokStatus.Ocid = shared.OCID(resourceID)
	}
	recordAttachmentPathIdentity(resource, identity)
}

func clearAttachmentTrackedIdentity(resource *marketplaceprivateofferv1beta1.Attachment) {
	if resource == nil {
		return
	}
	resource.Status.Id = ""
	resource.Status.OsokStatus.Ocid = ""
	resource.Status.OfferId = ""
}

func buildAttachmentCreateBody(
	_ context.Context,
	resource *marketplaceprivateofferv1beta1.Attachment,
	_ string,
) (any, error) {
	if resource == nil {
		return marketplaceprivateoffersdk.CreateAttachmentDetails{}, fmt.Errorf("attachment resource is nil")
	}
	if err := validateAttachmentSpec(resource.Spec); err != nil {
		return marketplaceprivateoffersdk.CreateAttachmentDetails{}, err
	}

	file, err := decodeAttachmentFile(resource.Spec.FileBase64Encoded)
	if err != nil {
		return marketplaceprivateoffersdk.CreateAttachmentDetails{}, err
	}
	attachmentType, err := attachmentType(resource.Spec.Type)
	if err != nil {
		return marketplaceprivateoffersdk.CreateAttachmentDetails{}, err
	}

	return marketplaceprivateoffersdk.CreateAttachmentDetails{
		FileBase64Encoded: file,
		DisplayName:       common.String(strings.TrimSpace(resource.Spec.DisplayName)),
		Type:              attachmentType,
	}, nil
}

func validateAttachmentSpec(spec marketplaceprivateofferv1beta1.AttachmentSpec) error {
	var missing []string
	if strings.TrimSpace(spec.FileBase64Encoded) == "" {
		missing = append(missing, "fileBase64Encoded")
	}
	if strings.TrimSpace(spec.DisplayName) == "" {
		missing = append(missing, "displayName")
	}
	if strings.TrimSpace(spec.Type) == "" {
		missing = append(missing, "type")
	}
	if len(missing) != 0 {
		return fmt.Errorf("attachment spec is missing required field(s): %s", strings.Join(missing, ", "))
	}
	if _, err := attachmentType(spec.Type); err != nil {
		return err
	}
	return nil
}

func validateAttachmentCreateOnlyDrift(
	resource *marketplaceprivateofferv1beta1.Attachment,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("attachment resource is nil")
	}
	current, err := attachmentRuntimeBody(currentResponse)
	if err != nil {
		return err
	}
	if err := validateAttachmentSpec(resource.Spec); err != nil {
		return err
	}

	drift, recordedFingerprint, err := attachmentCreateOnlyDriftFields(resource, current)
	if err != nil {
		return err
	}
	if len(drift) == 0 {
		return nil
	}
	return attachmentCreateOnlyDriftError{
		fields:              drift,
		recordedFingerprint: recordedFingerprint,
	}
}

func attachmentCreateOnlyDriftFields(
	resource *marketplaceprivateofferv1beta1.Attachment,
	current marketplaceprivateoffersdk.Attachment,
) ([]string, string, error) {
	drift := make([]string, 0, 4)
	desiredType, err := attachmentType(resource.Spec.Type)
	if err != nil {
		return nil, "", err
	}
	drift = appendAttachmentStringDrift(drift, "offerId", attachmentOfferIDForDrift(resource), attachmentString(current.OfferId))
	drift = appendAttachmentStringDrift(drift, "displayName", resource.Spec.DisplayName, attachmentString(current.DisplayName))
	drift = appendAttachmentStringDrift(drift, "type", string(desiredType), string(current.Type))

	recordedFingerprint, hasRecordedFingerprint := attachmentRecordedFileFingerprint(resource)
	if !hasRecordedFingerprint {
		return drift, "", nil
	}
	desiredFingerprint, err := attachmentFileFingerprint(resource.Spec.FileBase64Encoded)
	if err != nil {
		return nil, "", err
	}
	drift = appendAttachmentStringDrift(drift, "fileBase64Encoded", desiredFingerprint, recordedFingerprint)
	return drift, recordedFingerprint, nil
}

func appendAttachmentStringDrift(drift []string, field string, desired string, current string) []string {
	desired = strings.TrimSpace(desired)
	current = strings.TrimSpace(current)
	if desired != "" && current != "" && desired != current {
		return append(drift, field)
	}
	return drift
}

type attachmentCreateOnlyDriftError struct {
	fields              []string
	recordedFingerprint string
}

func (e attachmentCreateOnlyDriftError) Error() string {
	message := fmt.Sprintf("attachment has create-only drift for %s; replace the resource or restore the desired spec before reconcile", strings.Join(e.fields, ", "))
	if strings.TrimSpace(e.recordedFingerprint) == "" {
		return message
	}
	return message + "; " + attachmentFileSHA256Key + e.recordedFingerprint
}

func attachmentRuntimeBody(currentResponse any) (marketplaceprivateoffersdk.Attachment, error) {
	if current, ok, err := attachmentDirectRuntimeBody(currentResponse); ok || err != nil {
		return current, err
	}
	if current, ok, err := attachmentResponseRuntimeBody(currentResponse); ok || err != nil {
		return current, err
	}
	return marketplaceprivateoffersdk.Attachment{}, fmt.Errorf("unexpected current Attachment response type %T", currentResponse)
}

func attachmentDirectRuntimeBody(currentResponse any) (marketplaceprivateoffersdk.Attachment, bool, error) {
	switch current := currentResponse.(type) {
	case marketplaceprivateoffersdk.Attachment:
		return current, true, nil
	case *marketplaceprivateoffersdk.Attachment:
		body, err := dereferenceAttachmentRuntimeBody(current)
		return body, true, err
	case marketplaceprivateoffersdk.AttachmentSummary:
		return attachmentFromSummary(current), true, nil
	case *marketplaceprivateoffersdk.AttachmentSummary:
		summary, err := dereferenceAttachmentRuntimeBody(current)
		if err != nil {
			return marketplaceprivateoffersdk.Attachment{}, true, err
		}
		return attachmentFromSummary(summary), true, nil
	default:
		return marketplaceprivateoffersdk.Attachment{}, false, nil
	}
}

func attachmentResponseRuntimeBody(currentResponse any) (marketplaceprivateoffersdk.Attachment, bool, error) {
	switch current := currentResponse.(type) {
	case marketplaceprivateoffersdk.CreateAttachmentResponse:
		return current.Attachment, true, nil
	case *marketplaceprivateoffersdk.CreateAttachmentResponse:
		response, err := dereferenceAttachmentRuntimeBody(current)
		if err != nil {
			return marketplaceprivateoffersdk.Attachment{}, true, err
		}
		return response.Attachment, true, nil
	case marketplaceprivateoffersdk.GetAttachmentResponse:
		return current.Attachment, true, nil
	case *marketplaceprivateoffersdk.GetAttachmentResponse:
		response, err := dereferenceAttachmentRuntimeBody(current)
		if err != nil {
			return marketplaceprivateoffersdk.Attachment{}, true, err
		}
		return response.Attachment, true, nil
	default:
		return marketplaceprivateoffersdk.Attachment{}, false, nil
	}
}

func dereferenceAttachmentRuntimeBody[T any](current *T) (T, error) {
	if current == nil {
		var zero T
		return zero, fmt.Errorf("current Attachment response is nil")
	}
	return *current, nil
}

func attachmentFromSummary(summary marketplaceprivateoffersdk.AttachmentSummary) marketplaceprivateoffersdk.Attachment {
	return marketplaceprivateoffersdk.Attachment{
		Id:             summary.Id,
		OfferId:        summary.OfferId,
		DisplayName:    summary.DisplayName,
		Type:           summary.Type,
		LifecycleState: summary.LifecycleState,
		FreeformTags:   cloneAttachmentStringMap(summary.FreeformTags),
		DefinedTags:    cloneAttachmentDefinedTags(summary.DefinedTags),
		MimeType:       summary.MimeType,
	}
}

func listAttachmentsAllPages(
	call func(context.Context, marketplaceprivateoffersdk.ListAttachmentsRequest) (marketplaceprivateoffersdk.ListAttachmentsResponse, error),
) func(context.Context, marketplaceprivateoffersdk.ListAttachmentsRequest) (marketplaceprivateoffersdk.ListAttachmentsResponse, error) {
	return func(ctx context.Context, request marketplaceprivateoffersdk.ListAttachmentsRequest) (marketplaceprivateoffersdk.ListAttachmentsResponse, error) {
		var combined marketplaceprivateoffersdk.ListAttachmentsResponse
		seenPages := map[string]struct{}{}

		for {
			response, err := call(ctx, request)
			if err != nil {
				return marketplaceprivateoffersdk.ListAttachmentsResponse{}, err
			}
			combined.RawResponse = response.RawResponse
			if combined.OpcRequestId == nil {
				combined.OpcRequestId = response.OpcRequestId
			}
			combined.Items = append(combined.Items, response.Items...)

			nextPage := strings.TrimSpace(attachmentString(response.OpcNextPage))
			if nextPage == "" {
				combined.OpcNextPage = nil
				return combined, nil
			}
			if _, ok := seenPages[nextPage]; ok {
				return marketplaceprivateoffersdk.ListAttachmentsResponse{}, fmt.Errorf("attachment list pagination repeated page token %q", nextPage)
			}
			seenPages[nextPage] = struct{}{}
			request.Page = response.OpcNextPage
			combined.OpcNextPage = response.OpcNextPage
		}
	}
}

func handleAttachmentDeleteError(resource *marketplaceprivateofferv1beta1.Attachment, err error) error {
	if err == nil {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if !isAmbiguousAttachmentNotFound(err) {
		return err
	}
	return ambiguousAttachmentNotFound("delete path", err)
}

func wrapAttachmentLifecycleHooks(hooks *AttachmentRuntimeHooks) {
	if hooks == nil || hooks.Get.Call == nil {
		return
	}
	getAttachment := hooks.Get.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate AttachmentServiceClient) AttachmentServiceClient {
		return attachmentRuntimeClient{
			delegate:      delegate,
			getAttachment: getAttachment,
		}
	})
}

type attachmentRuntimeClient struct {
	delegate      AttachmentServiceClient
	getAttachment func(context.Context, marketplaceprivateoffersdk.GetAttachmentRequest) (marketplaceprivateoffersdk.GetAttachmentResponse, error)
}

func (c attachmentRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *marketplaceprivateofferv1beta1.Attachment,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("attachment generated runtime delegate is not configured")
	}
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("attachment resource is nil")
	}
	if err := validateAttachmentSpec(resource.Spec); err != nil {
		markAttachmentCreateOrUpdateFailure(resource, err)
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	if err := rejectMissingAttachmentFileFingerprint(resource); err != nil {
		markAttachmentCreateOrUpdateFailure(resource, err)
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err == nil && response.IsSuccessful {
		recordAttachmentFileFingerprint(resource)
	}
	return response, err
}

func (c attachmentRuntimeClient) Delete(
	ctx context.Context,
	resource *marketplaceprivateofferv1beta1.Attachment,
) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("attachment generated runtime delegate is not configured")
	}
	if resource != nil && attachmentCurrentID(resource) == "" && attachmentOfferID(resource) == "" {
		return true, nil
	}
	if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c attachmentRuntimeClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *marketplaceprivateofferv1beta1.Attachment,
) error {
	if c.getAttachment == nil || resource == nil {
		return nil
	}
	attachmentID := attachmentCurrentID(resource)
	offerID := attachmentOfferID(resource)
	if attachmentID == "" || offerID == "" {
		return nil
	}

	_, err := c.getAttachment(ctx, marketplaceprivateoffersdk.GetAttachmentRequest{
		OfferId:      common.String(offerID),
		AttachmentId: common.String(attachmentID),
	})
	if err == nil {
		return nil
	}
	if !isAmbiguousAttachmentNotFound(err) {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return ambiguousAttachmentNotFound("delete confirmation", err)
}

func isAmbiguousAttachmentNotFound(err error) bool {
	if err == nil {
		return false
	}
	var ambiguous ambiguousAttachmentNotFoundError
	if errors.As(err, &ambiguous) {
		return true
	}
	return errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound()
}

func ambiguousAttachmentNotFound(operation string, err error) ambiguousAttachmentNotFoundError {
	var ambiguous ambiguousAttachmentNotFoundError
	if errors.As(err, &ambiguous) {
		return ambiguous
	}

	return ambiguousAttachmentNotFoundError{
		message:      fmt.Sprintf("attachment %s returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %s", strings.TrimSpace(operation), err.Error()),
		opcRequestID: errorutil.OpcRequestID(err),
	}
}

func decodeAttachmentFile(fileBase64Encoded string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(fileBase64Encoded))
	if err != nil {
		return nil, fmt.Errorf("decode Attachment spec.fileBase64Encoded: %w", err)
	}
	if len(decoded) == 0 {
		return nil, fmt.Errorf("attachment spec.fileBase64Encoded decoded to empty content")
	}
	return decoded, nil
}

func attachmentType(value string) (marketplaceprivateoffersdk.AttachmentTypeEnum, error) {
	trimmed := strings.TrimSpace(value)
	attachmentType, ok := marketplaceprivateoffersdk.GetMappingAttachmentTypeEnum(trimmed)
	if !ok {
		return "", fmt.Errorf("unsupported Attachment type %q; supported values: %s", value, strings.Join(marketplaceprivateoffersdk.GetAttachmentTypeEnumStringValues(), ", "))
	}
	return attachmentType, nil
}

func rejectMissingAttachmentFileFingerprint(resource *marketplaceprivateofferv1beta1.Attachment) error {
	if resource == nil || attachmentCurrentID(resource) == "" {
		return nil
	}
	if _, ok := attachmentRecordedFileFingerprint(resource); ok {
		return nil
	}
	return fmt.Errorf("attachment cannot verify create-only spec.fileBase64Encoded drift because status.status.message is missing %s; replace the resource or restore the recorded fingerprint before reconcile", attachmentFileSHA256Key)
}

func markAttachmentCreateOrUpdateFailure(resource *marketplaceprivateofferv1beta1.Attachment, err error) {
	if resource == nil || err == nil {
		return
	}
	now := metav1.Now()
	fingerprint, hasFingerprint := attachmentRecordedFileFingerprint(resource)
	status := &resource.Status.OsokStatus
	status.Message = err.Error()
	if hasFingerprint {
		status.Message = attachmentMessageWithFileFingerprint(status.Message, fingerprint)
	}
	status.Reason = string(shared.Failed)
	status.UpdatedAt = &now
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, corev1.ConditionFalse, "", err.Error(), loggerutil.OSOKLogger{})
}

func recordAttachmentFileFingerprint(resource *marketplaceprivateofferv1beta1.Attachment) {
	if resource == nil {
		return
	}
	fingerprint, err := attachmentFileFingerprint(resource.Spec.FileBase64Encoded)
	if err != nil {
		return
	}
	resource.Status.OsokStatus.Message = attachmentMessageWithFileFingerprint(resource.Status.OsokStatus.Message, fingerprint)
}

func attachmentFileFingerprint(fileBase64Encoded string) (string, error) {
	decoded, err := decodeAttachmentFile(fileBase64Encoded)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(decoded)
	return hex.EncodeToString(sum[:]), nil
}

func attachmentMessageWithFileFingerprint(message string, fingerprint string) string {
	base := stripAttachmentFileFingerprint(message)
	marker := attachmentFileSHA256Key + fingerprint
	if base == "" {
		return marker
	}
	return base + "; " + marker
}

func attachmentRecordedFileFingerprint(resource *marketplaceprivateofferv1beta1.Attachment) (string, bool) {
	if resource == nil {
		return "", false
	}
	raw := resource.Status.OsokStatus.Message
	index := strings.LastIndex(raw, attachmentFileSHA256Key)
	if index < 0 {
		return "", false
	}
	start := index + len(attachmentFileSHA256Key)
	end := start
	for end < len(raw) && attachmentIsHexDigit(raw[end]) {
		end++
	}
	fingerprint := raw[start:end]
	if len(fingerprint) != sha256.Size*2 {
		return "", false
	}
	if _, err := hex.DecodeString(fingerprint); err != nil {
		return "", false
	}
	return fingerprint, true
}

func stripAttachmentFileFingerprint(raw string) string {
	raw = strings.TrimSpace(raw)
	index := strings.LastIndex(raw, attachmentFileSHA256Key)
	if index < 0 {
		return raw
	}
	prefix := strings.TrimSpace(strings.TrimRight(raw[:index], "; "))
	start := index + len(attachmentFileSHA256Key)
	end := start
	for end < len(raw) && attachmentIsHexDigit(raw[end]) {
		end++
	}
	suffix := strings.TrimSpace(strings.TrimLeft(raw[end:], "; "))
	return strings.TrimSpace(strings.Trim(prefix+"; "+suffix, "; "))
}

func attachmentIsHexDigit(value byte) bool {
	return (value >= '0' && value <= '9') ||
		(value >= 'a' && value <= 'f') ||
		(value >= 'A' && value <= 'F')
}

func attachmentCurrentID(resource *marketplaceprivateofferv1beta1.Attachment) string {
	if resource == nil {
		return ""
	}
	return firstNonEmptyAttachmentString(
		string(resource.Status.OsokStatus.Ocid),
		resource.Status.Id,
	)
}

func attachmentOfferID(resource *marketplaceprivateofferv1beta1.Attachment) string {
	if resource == nil {
		return ""
	}
	return firstNonEmptyAttachmentString(
		resource.Status.OfferId,
		attachmentAnnotation(resource, attachmentOfferIDAnnotation),
	)
}

func attachmentOfferIDForDrift(resource *marketplaceprivateofferv1beta1.Attachment) string {
	if resource == nil {
		return ""
	}
	return firstNonEmptyAttachmentString(resource.Status.OfferId, attachmentAnnotation(resource, attachmentOfferIDAnnotation))
}

func attachmentAnnotation(resource *marketplaceprivateofferv1beta1.Attachment, key string) string {
	if resource == nil || len(resource.Annotations) == 0 {
		return ""
	}
	return strings.TrimSpace(resource.Annotations[key])
}

func attachmentString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func firstNonEmptyAttachmentString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func cloneAttachmentStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func cloneAttachmentDefinedTags(input map[string]map[string]interface{}) map[string]map[string]interface{} {
	if input == nil {
		return nil
	}
	cloned := make(map[string]map[string]interface{}, len(input))
	for key, values := range input {
		if values == nil {
			cloned[key] = nil
			continue
		}
		child := make(map[string]interface{}, len(values))
		for childKey, childValue := range values {
			child[childKey] = childValue
		}
		cloned[key] = child
	}
	return cloned
}

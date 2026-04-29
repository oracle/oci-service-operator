/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package securityattributenamespace

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	securityattributesdk "github.com/oracle/oci-go-sdk/v65/securityattribute"
	securityattributev1beta1 "github.com/oracle/oci-service-operator/api/securityattribute/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const securityAttributeNamespaceDeletePendingMessage = "OCI SecurityAttributeNamespace delete is in progress"

type securityAttributeNamespaceOCIClient interface {
	CreateSecurityAttributeNamespace(context.Context, securityattributesdk.CreateSecurityAttributeNamespaceRequest) (securityattributesdk.CreateSecurityAttributeNamespaceResponse, error)
	GetSecurityAttributeNamespace(context.Context, securityattributesdk.GetSecurityAttributeNamespaceRequest) (securityattributesdk.GetSecurityAttributeNamespaceResponse, error)
	ListSecurityAttributeNamespaces(context.Context, securityattributesdk.ListSecurityAttributeNamespacesRequest) (securityattributesdk.ListSecurityAttributeNamespacesResponse, error)
	UpdateSecurityAttributeNamespace(context.Context, securityattributesdk.UpdateSecurityAttributeNamespaceRequest) (securityattributesdk.UpdateSecurityAttributeNamespaceResponse, error)
	DeleteSecurityAttributeNamespace(context.Context, securityattributesdk.DeleteSecurityAttributeNamespaceRequest) (securityattributesdk.DeleteSecurityAttributeNamespaceResponse, error)
}

type ambiguousSecurityAttributeNamespaceNotFoundError struct {
	message      string
	opcRequestID string
}

func (e ambiguousSecurityAttributeNamespaceNotFoundError) Error() string {
	return e.message
}

func (e ambiguousSecurityAttributeNamespaceNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerSecurityAttributeNamespaceRuntimeHooksMutator(func(_ *SecurityAttributeNamespaceServiceManager, hooks *SecurityAttributeNamespaceRuntimeHooks) {
		applySecurityAttributeNamespaceRuntimeHooks(hooks)
	})
}

func applySecurityAttributeNamespaceRuntimeHooks(hooks *SecurityAttributeNamespaceRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedSecurityAttributeNamespaceRuntimeSemantics()
	hooks.BuildCreateBody = buildSecurityAttributeNamespaceCreateBody
	hooks.BuildUpdateBody = buildSecurityAttributeNamespaceUpdateBody
	hooks.Identity.GuardExistingBeforeCreate = guardSecurityAttributeNamespaceExistingBeforeCreate
	hooks.List.Fields = securityAttributeNamespaceListFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listSecurityAttributeNamespacesAllPages(hooks.List.Call)
	}
	wrapSecurityAttributeNamespaceReadAndDeleteCalls(hooks)
	wrapSecurityAttributeNamespaceDeleteConfirmation(hooks)
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateSecurityAttributeNamespaceCreateOnlyDriftForResponse
	hooks.StatusHooks.MarkTerminating = markSecurityAttributeNamespaceTerminating
	hooks.DeleteHooks.HandleError = handleSecurityAttributeNamespaceDeleteError
	hooks.DeleteHooks.ApplyOutcome = applySecurityAttributeNamespaceDeleteOutcome
}

func newSecurityAttributeNamespaceServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client securityAttributeNamespaceOCIClient,
) SecurityAttributeNamespaceServiceClient {
	hooks := newSecurityAttributeNamespaceRuntimeHooksWithOCIClient(client)
	applySecurityAttributeNamespaceRuntimeHooks(&hooks)
	delegate := defaultSecurityAttributeNamespaceServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*securityattributev1beta1.SecurityAttributeNamespace](
			buildSecurityAttributeNamespaceGeneratedRuntimeConfig(&SecurityAttributeNamespaceServiceManager{Log: log}, hooks),
		),
	}
	return wrapSecurityAttributeNamespaceGeneratedClient(hooks, delegate)
}

func newSecurityAttributeNamespaceRuntimeHooksWithOCIClient(client securityAttributeNamespaceOCIClient) SecurityAttributeNamespaceRuntimeHooks {
	return SecurityAttributeNamespaceRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*securityattributev1beta1.SecurityAttributeNamespace]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*securityattributev1beta1.SecurityAttributeNamespace]{},
		StatusHooks:     generatedruntime.StatusHooks[*securityattributev1beta1.SecurityAttributeNamespace]{},
		ParityHooks:     generatedruntime.ParityHooks[*securityattributev1beta1.SecurityAttributeNamespace]{},
		Async:           generatedruntime.AsyncHooks[*securityattributev1beta1.SecurityAttributeNamespace]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*securityattributev1beta1.SecurityAttributeNamespace]{},
		Create: runtimeOperationHooks[securityattributesdk.CreateSecurityAttributeNamespaceRequest, securityattributesdk.CreateSecurityAttributeNamespaceResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateSecurityAttributeNamespaceDetails", RequestName: "CreateSecurityAttributeNamespaceDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request securityattributesdk.CreateSecurityAttributeNamespaceRequest) (securityattributesdk.CreateSecurityAttributeNamespaceResponse, error) {
				if client == nil {
					return securityattributesdk.CreateSecurityAttributeNamespaceResponse{}, fmt.Errorf("SecurityAttributeNamespace OCI client is nil")
				}
				return client.CreateSecurityAttributeNamespace(ctx, request)
			},
		},
		Get: runtimeOperationHooks[securityattributesdk.GetSecurityAttributeNamespaceRequest, securityattributesdk.GetSecurityAttributeNamespaceResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "SecurityAttributeNamespaceId", RequestName: "securityAttributeNamespaceId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request securityattributesdk.GetSecurityAttributeNamespaceRequest) (securityattributesdk.GetSecurityAttributeNamespaceResponse, error) {
				if client == nil {
					return securityattributesdk.GetSecurityAttributeNamespaceResponse{}, fmt.Errorf("SecurityAttributeNamespace OCI client is nil")
				}
				return client.GetSecurityAttributeNamespace(ctx, request)
			},
		},
		List: runtimeOperationHooks[securityattributesdk.ListSecurityAttributeNamespacesRequest, securityattributesdk.ListSecurityAttributeNamespacesResponse]{
			Fields: securityAttributeNamespaceListFields(),
			Call: func(ctx context.Context, request securityattributesdk.ListSecurityAttributeNamespacesRequest) (securityattributesdk.ListSecurityAttributeNamespacesResponse, error) {
				if client == nil {
					return securityattributesdk.ListSecurityAttributeNamespacesResponse{}, fmt.Errorf("SecurityAttributeNamespace OCI client is nil")
				}
				return client.ListSecurityAttributeNamespaces(ctx, request)
			},
		},
		Update: runtimeOperationHooks[securityattributesdk.UpdateSecurityAttributeNamespaceRequest, securityattributesdk.UpdateSecurityAttributeNamespaceResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "SecurityAttributeNamespaceId", RequestName: "securityAttributeNamespaceId", Contribution: "path", PreferResourceID: true}, {FieldName: "UpdateSecurityAttributeNamespaceDetails", RequestName: "UpdateSecurityAttributeNamespaceDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request securityattributesdk.UpdateSecurityAttributeNamespaceRequest) (securityattributesdk.UpdateSecurityAttributeNamespaceResponse, error) {
				if client == nil {
					return securityattributesdk.UpdateSecurityAttributeNamespaceResponse{}, fmt.Errorf("SecurityAttributeNamespace OCI client is nil")
				}
				return client.UpdateSecurityAttributeNamespace(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[securityattributesdk.DeleteSecurityAttributeNamespaceRequest, securityattributesdk.DeleteSecurityAttributeNamespaceResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "SecurityAttributeNamespaceId", RequestName: "securityAttributeNamespaceId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request securityattributesdk.DeleteSecurityAttributeNamespaceRequest) (securityattributesdk.DeleteSecurityAttributeNamespaceResponse, error) {
				if client == nil {
					return securityattributesdk.DeleteSecurityAttributeNamespaceResponse{}, fmt.Errorf("SecurityAttributeNamespace OCI client is nil")
				}
				return client.DeleteSecurityAttributeNamespace(ctx, request)
			},
		},
		WrapGeneratedClient: []func(SecurityAttributeNamespaceServiceClient) SecurityAttributeNamespaceServiceClient{},
	}
}

func reviewedSecurityAttributeNamespaceRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "securityattribute",
		FormalSlug:    "securityattributenamespace",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ActiveStates: []string{
				string(securityattributesdk.SecurityAttributeNamespaceLifecycleStateActive),
				string(securityattributesdk.SecurityAttributeNamespaceLifecycleStateInactive),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(securityattributesdk.SecurityAttributeNamespaceLifecycleStateDeleting)},
			TerminalStates: []string{string(securityattributesdk.SecurityAttributeNamespaceLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "name"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"description", "isRetired", "freeformTags", "definedTags"},
			ForceNew:      []string{"compartmentId", "name"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "SecurityAttributeNamespace", Action: "CreateSecurityAttributeNamespace"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "SecurityAttributeNamespace", Action: "UpdateSecurityAttributeNamespace"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "SecurityAttributeNamespace", Action: "DeleteSecurityAttributeNamespace"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "SecurityAttributeNamespace", Action: "GetSecurityAttributeNamespace"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "SecurityAttributeNamespace", Action: "GetSecurityAttributeNamespace"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "SecurityAttributeNamespace", Action: "GetSecurityAttributeNamespace"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func securityAttributeNamespaceListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "Name", RequestName: "name", Contribution: "query", LookupPaths: []string{"status.name", "spec.name", "metadataName", "name"}},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "CompartmentIdInSubtree", RequestName: "compartmentIdInSubtree", Contribution: "query"},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
	}
}

func buildSecurityAttributeNamespaceCreateBody(
	_ context.Context,
	resource *securityattributev1beta1.SecurityAttributeNamespace,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("SecurityAttributeNamespace resource is nil")
	}
	if err := validateSecurityAttributeNamespaceSpec(resource.Spec); err != nil {
		return nil, err
	}

	body := securityattributesdk.CreateSecurityAttributeNamespaceDetails{
		CompartmentId: common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		Name:          common.String(strings.TrimSpace(resource.Spec.Name)),
		Description:   common.String(strings.TrimSpace(resource.Spec.Description)),
	}
	if resource.Spec.FreeformTags != nil {
		body.FreeformTags = cloneSecurityAttributeNamespaceStringMap(resource.Spec.FreeformTags)
	}
	if resource.Spec.DefinedTags != nil {
		body.DefinedTags = *util.ConvertToOciDefinedTags(&resource.Spec.DefinedTags)
	}
	return body, nil
}

func buildSecurityAttributeNamespaceUpdateBody(
	_ context.Context,
	resource *securityattributev1beta1.SecurityAttributeNamespace,
	_ string,
	currentResponse any,
) (any, bool, error) {
	current, err := currentSecurityAttributeNamespaceForUpdate(resource, currentResponse)
	if err != nil {
		return securityattributesdk.UpdateSecurityAttributeNamespaceDetails{}, false, err
	}

	updateDetails, updateNeeded := securityAttributeNamespaceUpdateDetails(resource.Spec, current)
	return updateDetails, updateNeeded, nil
}

func currentSecurityAttributeNamespaceForUpdate(
	resource *securityattributev1beta1.SecurityAttributeNamespace,
	currentResponse any,
) (securityattributesdk.SecurityAttributeNamespace, error) {
	if resource == nil {
		return securityattributesdk.SecurityAttributeNamespace{}, fmt.Errorf("SecurityAttributeNamespace resource is nil")
	}
	if err := validateSecurityAttributeNamespaceSpec(resource.Spec); err != nil {
		return securityattributesdk.SecurityAttributeNamespace{}, err
	}

	current, ok := securityAttributeNamespaceFromResponse(currentResponse)
	if !ok {
		return securityattributesdk.SecurityAttributeNamespace{}, fmt.Errorf("current SecurityAttributeNamespace response does not expose a SecurityAttributeNamespace body")
	}
	if err := validateSecurityAttributeNamespaceCreateOnlyDrift(resource.Spec, current); err != nil {
		return securityattributesdk.SecurityAttributeNamespace{}, err
	}
	return current, nil
}

func securityAttributeNamespaceUpdateDetails(
	spec securityattributev1beta1.SecurityAttributeNamespaceSpec,
	current securityattributesdk.SecurityAttributeNamespace,
) (securityattributesdk.UpdateSecurityAttributeNamespaceDetails, bool) {
	updateDetails := securityattributesdk.UpdateSecurityAttributeNamespaceDetails{}
	updateNeeded := applySecurityAttributeNamespaceScalarUpdates(&updateDetails, spec, current)
	updateNeeded = applySecurityAttributeNamespaceTagUpdates(&updateDetails, spec, current) || updateNeeded
	return updateDetails, updateNeeded
}

func applySecurityAttributeNamespaceScalarUpdates(
	updateDetails *securityattributesdk.UpdateSecurityAttributeNamespaceDetails,
	spec securityattributev1beta1.SecurityAttributeNamespaceSpec,
	current securityattributesdk.SecurityAttributeNamespace,
) bool {
	updateNeeded := false
	if desired := strings.TrimSpace(spec.Description); !stringPtrEqual(current.Description, desired) {
		updateDetails.Description = common.String(desired)
		updateNeeded = true
	}
	if desired, ok := desiredSecurityAttributeNamespaceRetiredUpdate(spec.IsRetired, current.IsRetired); ok {
		updateDetails.IsRetired = common.Bool(desired)
		updateNeeded = true
	}
	return updateNeeded
}

func applySecurityAttributeNamespaceTagUpdates(
	updateDetails *securityattributesdk.UpdateSecurityAttributeNamespaceDetails,
	spec securityattributev1beta1.SecurityAttributeNamespaceSpec,
	current securityattributesdk.SecurityAttributeNamespace,
) bool {
	updateNeeded := false
	if spec.FreeformTags != nil {
		desired := cloneSecurityAttributeNamespaceStringMap(spec.FreeformTags)
		if !reflect.DeepEqual(current.FreeformTags, desired) {
			updateDetails.FreeformTags = desired
			updateNeeded = true
		}
	}
	if spec.DefinedTags != nil {
		desired := *util.ConvertToOciDefinedTags(&spec.DefinedTags)
		if !reflect.DeepEqual(current.DefinedTags, desired) {
			updateDetails.DefinedTags = desired
			updateNeeded = true
		}
	}
	return updateNeeded
}

func desiredSecurityAttributeNamespaceRetiredUpdate(spec bool, current *bool) (bool, bool) {
	if current == nil {
		return spec, spec
	}
	if *current == spec {
		return false, false
	}
	return spec, true
}

func validateSecurityAttributeNamespaceSpec(spec securityattributev1beta1.SecurityAttributeNamespaceSpec) error {
	var missing []string
	if strings.TrimSpace(spec.CompartmentId) == "" {
		missing = append(missing, "compartmentId")
	}
	if strings.TrimSpace(spec.Name) == "" {
		missing = append(missing, "name")
	}
	if strings.TrimSpace(spec.Description) == "" {
		missing = append(missing, "description")
	}
	if len(missing) != 0 {
		return fmt.Errorf("SecurityAttributeNamespace spec is missing required field(s): %s", strings.Join(missing, ", "))
	}
	return nil
}

func guardSecurityAttributeNamespaceExistingBeforeCreate(
	_ context.Context,
	resource *securityattributev1beta1.SecurityAttributeNamespace,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("SecurityAttributeNamespace resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" || strings.TrimSpace(resource.Spec.Name) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func validateSecurityAttributeNamespaceCreateOnlyDriftForResponse(
	resource *securityattributev1beta1.SecurityAttributeNamespace,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("SecurityAttributeNamespace resource is nil")
	}
	current, ok := securityAttributeNamespaceFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current SecurityAttributeNamespace response does not expose a SecurityAttributeNamespace body")
	}
	return validateSecurityAttributeNamespaceCreateOnlyDrift(resource.Spec, current)
}

func validateSecurityAttributeNamespaceCreateOnlyDrift(
	spec securityattributev1beta1.SecurityAttributeNamespaceSpec,
	current securityattributesdk.SecurityAttributeNamespace,
) error {
	var drift []string
	if !stringPtrEqual(current.CompartmentId, strings.TrimSpace(spec.CompartmentId)) {
		drift = append(drift, "compartmentId")
	}
	if !stringPtrEqual(current.Name, strings.TrimSpace(spec.Name)) {
		drift = append(drift, "name")
	}
	if len(drift) != 0 {
		return fmt.Errorf("SecurityAttributeNamespace create-only field drift is not supported: %s", strings.Join(drift, ", "))
	}
	return nil
}

func listSecurityAttributeNamespacesAllPages(
	call func(context.Context, securityattributesdk.ListSecurityAttributeNamespacesRequest) (securityattributesdk.ListSecurityAttributeNamespacesResponse, error),
) func(context.Context, securityattributesdk.ListSecurityAttributeNamespacesRequest) (securityattributesdk.ListSecurityAttributeNamespacesResponse, error) {
	return func(ctx context.Context, request securityattributesdk.ListSecurityAttributeNamespacesRequest) (securityattributesdk.ListSecurityAttributeNamespacesResponse, error) {
		return collectSecurityAttributeNamespacePages(ctx, request, call)
	}
}

func collectSecurityAttributeNamespacePages(
	ctx context.Context,
	request securityattributesdk.ListSecurityAttributeNamespacesRequest,
	call func(context.Context, securityattributesdk.ListSecurityAttributeNamespacesRequest) (securityattributesdk.ListSecurityAttributeNamespacesResponse, error),
) (securityattributesdk.ListSecurityAttributeNamespacesResponse, error) {
	var combined securityattributesdk.ListSecurityAttributeNamespacesResponse
	seenPages := map[string]bool{}
	for {
		response, err := call(ctx, request)
		if err != nil {
			return securityattributesdk.ListSecurityAttributeNamespacesResponse{}, err
		}
		appendSecurityAttributeNamespacePage(&combined, response)
		nextPage, ok, err := nextSecurityAttributeNamespacePage(response, seenPages)
		if err != nil {
			return combined, err
		}
		if !ok {
			combined.OpcNextPage = nil
			return combined, nil
		}
		request.Page = common.String(nextPage)
		combined.OpcNextPage = common.String(nextPage)
	}
}

func appendSecurityAttributeNamespacePage(
	combined *securityattributesdk.ListSecurityAttributeNamespacesResponse,
	response securityattributesdk.ListSecurityAttributeNamespacesResponse,
) {
	combined.RawResponse = response.RawResponse
	combined.OpcRequestId = response.OpcRequestId
	for _, item := range response.Items {
		if item.LifecycleState == securityattributesdk.SecurityAttributeNamespaceLifecycleStateDeleted {
			continue
		}
		combined.Items = append(combined.Items, item)
	}
}

func nextSecurityAttributeNamespacePage(
	response securityattributesdk.ListSecurityAttributeNamespacesResponse,
	seenPages map[string]bool,
) (string, bool, error) {
	if response.OpcNextPage == nil {
		return "", false, nil
	}
	nextPage := strings.TrimSpace(*response.OpcNextPage)
	if nextPage == "" {
		return "", false, nil
	}
	if seenPages[nextPage] {
		return "", false, fmt.Errorf("SecurityAttributeNamespace list pagination repeated page token %q", nextPage)
	}
	seenPages[nextPage] = true
	return nextPage, true, nil
}

func wrapSecurityAttributeNamespaceReadAndDeleteCalls(hooks *SecurityAttributeNamespaceRuntimeHooks) {
	if hooks == nil {
		return
	}
	if getCall := hooks.Get.Call; getCall != nil {
		hooks.Get.Call = func(ctx context.Context, request securityattributesdk.GetSecurityAttributeNamespaceRequest) (securityattributesdk.GetSecurityAttributeNamespaceResponse, error) {
			response, err := getCall(ctx, request)
			return response, conservativeSecurityAttributeNamespaceNotFoundError(err, "read")
		}
	}
	if listCall := hooks.List.Call; listCall != nil {
		hooks.List.Call = func(ctx context.Context, request securityattributesdk.ListSecurityAttributeNamespacesRequest) (securityattributesdk.ListSecurityAttributeNamespacesResponse, error) {
			response, err := listCall(ctx, request)
			return response, conservativeSecurityAttributeNamespaceNotFoundError(err, "list")
		}
	}
	if deleteCall := hooks.Delete.Call; deleteCall != nil {
		hooks.Delete.Call = func(ctx context.Context, request securityattributesdk.DeleteSecurityAttributeNamespaceRequest) (securityattributesdk.DeleteSecurityAttributeNamespaceResponse, error) {
			response, err := deleteCall(ctx, request)
			return response, conservativeSecurityAttributeNamespaceNotFoundError(err, "delete")
		}
	}
}

func wrapSecurityAttributeNamespaceDeleteConfirmation(hooks *SecurityAttributeNamespaceRuntimeHooks) {
	if hooks == nil || hooks.Get.Call == nil {
		return
	}
	getSecurityAttributeNamespace := hooks.Get.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate SecurityAttributeNamespaceServiceClient) SecurityAttributeNamespaceServiceClient {
		return securityAttributeNamespaceDeleteConfirmationClient{
			delegate:                      delegate,
			getSecurityAttributeNamespace: getSecurityAttributeNamespace,
		}
	})
}

type securityAttributeNamespaceDeleteConfirmationClient struct {
	delegate                      SecurityAttributeNamespaceServiceClient
	getSecurityAttributeNamespace func(context.Context, securityattributesdk.GetSecurityAttributeNamespaceRequest) (securityattributesdk.GetSecurityAttributeNamespaceResponse, error)
}

func (c securityAttributeNamespaceDeleteConfirmationClient) CreateOrUpdate(
	ctx context.Context,
	resource *securityattributev1beta1.SecurityAttributeNamespace,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c securityAttributeNamespaceDeleteConfirmationClient) Delete(
	ctx context.Context,
	resource *securityattributev1beta1.SecurityAttributeNamespace,
) (bool, error) {
	if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c securityAttributeNamespaceDeleteConfirmationClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *securityattributev1beta1.SecurityAttributeNamespace,
) error {
	if c.getSecurityAttributeNamespace == nil || resource == nil {
		return nil
	}
	namespaceID := trackedSecurityAttributeNamespaceID(resource)
	if namespaceID == "" {
		return nil
	}
	_, err := c.getSecurityAttributeNamespace(ctx, securityattributesdk.GetSecurityAttributeNamespaceRequest{
		SecurityAttributeNamespaceId: common.String(namespaceID),
	})
	if err == nil || !isAmbiguousSecurityAttributeNamespaceNotFound(err) {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("SecurityAttributeNamespace delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %w", err)
}

func trackedSecurityAttributeNamespaceID(resource *securityattributev1beta1.SecurityAttributeNamespace) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func handleSecurityAttributeNamespaceDeleteError(resource *securityattributev1beta1.SecurityAttributeNamespace, err error) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return err
}

func applySecurityAttributeNamespaceDeleteOutcome(
	resource *securityattributev1beta1.SecurityAttributeNamespace,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	lifecycleState := strings.ToUpper(securityAttributeNamespaceLifecycleState(response))
	switch lifecycleState {
	case "", string(securityattributesdk.SecurityAttributeNamespaceLifecycleStateDeleted),
		string(securityattributesdk.SecurityAttributeNamespaceLifecycleStateDeleting):
		return generatedruntime.DeleteOutcome{}, nil
	case string(securityattributesdk.SecurityAttributeNamespaceLifecycleStateActive),
		string(securityattributesdk.SecurityAttributeNamespaceLifecycleStateInactive):
		if stage == generatedruntime.DeleteConfirmStageAlreadyPending && !securityAttributeNamespaceDeleteAlreadyPending(resource) {
			return generatedruntime.DeleteOutcome{}, nil
		}
		markSecurityAttributeNamespaceTerminating(resource, response)
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
	default:
		return generatedruntime.DeleteOutcome{}, nil
	}
}

func securityAttributeNamespaceDeleteAlreadyPending(resource *securityattributev1beta1.SecurityAttributeNamespace) bool {
	if resource == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	return current != nil &&
		current.Phase == shared.OSOKAsyncPhaseDelete &&
		current.NormalizedClass == shared.OSOKAsyncClassPending
}

func markSecurityAttributeNamespaceTerminating(resource *securityattributev1beta1.SecurityAttributeNamespace, _ any) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = securityAttributeNamespaceDeletePendingMessage
	status.Reason = string(shared.Terminating)
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         securityAttributeNamespaceDeletePendingMessage,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(
		*status,
		shared.Terminating,
		v1.ConditionTrue,
		"",
		securityAttributeNamespaceDeletePendingMessage,
		loggerutil.OSOKLogger{},
	)
}

func conservativeSecurityAttributeNamespaceNotFoundError(err error, operation string) error {
	if err == nil {
		return err
	}
	var ambiguous ambiguousSecurityAttributeNamespaceNotFoundError
	if errors.As(err, &ambiguous) {
		return err
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	message := fmt.Sprintf("SecurityAttributeNamespace %s returned ambiguous 404 NotAuthorizedOrNotFound: %s", strings.TrimSpace(operation), err.Error())
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		return ambiguousSecurityAttributeNamespaceNotFoundError{message: message, opcRequestID: serviceErr.GetOpcRequestID()}
	}
	return ambiguousSecurityAttributeNamespaceNotFoundError{message: message}
}

func isAmbiguousSecurityAttributeNamespaceNotFound(err error) bool {
	if err == nil {
		return false
	}
	var ambiguous ambiguousSecurityAttributeNamespaceNotFoundError
	if errors.As(err, &ambiguous) {
		return true
	}
	return errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound()
}

func securityAttributeNamespaceFromResponse(response any) (securityattributesdk.SecurityAttributeNamespace, bool) {
	if current, ok := securityAttributeNamespaceBodyFromResponse(response); ok {
		return current, true
	}
	if current, ok := securityAttributeNamespaceFromSummaryResponse(response); ok {
		return current, true
	}
	return securityAttributeNamespaceFromOperationResponse(response)
}

func securityAttributeNamespaceBodyFromResponse(response any) (securityattributesdk.SecurityAttributeNamespace, bool) {
	switch current := response.(type) {
	case securityattributesdk.SecurityAttributeNamespace:
		return current, true
	case *securityattributesdk.SecurityAttributeNamespace:
		return securityAttributeNamespaceFromPointer(current)
	default:
		return securityattributesdk.SecurityAttributeNamespace{}, false
	}
}

func securityAttributeNamespaceFromSummaryResponse(response any) (securityattributesdk.SecurityAttributeNamespace, bool) {
	switch current := response.(type) {
	case securityattributesdk.SecurityAttributeNamespaceSummary:
		return securityAttributeNamespaceFromSummary(current), true
	case *securityattributesdk.SecurityAttributeNamespaceSummary:
		return securityAttributeNamespaceFromSummaryPointer(current)
	default:
		return securityattributesdk.SecurityAttributeNamespace{}, false
	}
}

func securityAttributeNamespaceFromOperationResponse(response any) (securityattributesdk.SecurityAttributeNamespace, bool) {
	switch current := response.(type) {
	case securityattributesdk.CreateSecurityAttributeNamespaceResponse:
		return current.SecurityAttributeNamespace, true
	case *securityattributesdk.CreateSecurityAttributeNamespaceResponse:
		return securityAttributeNamespaceFromCreateResponsePointer(current)
	case securityattributesdk.GetSecurityAttributeNamespaceResponse:
		return current.SecurityAttributeNamespace, true
	case *securityattributesdk.GetSecurityAttributeNamespaceResponse:
		return securityAttributeNamespaceFromGetResponsePointer(current)
	case securityattributesdk.UpdateSecurityAttributeNamespaceResponse:
		return current.SecurityAttributeNamespace, true
	case *securityattributesdk.UpdateSecurityAttributeNamespaceResponse:
		return securityAttributeNamespaceFromUpdateResponsePointer(current)
	default:
		return securityattributesdk.SecurityAttributeNamespace{}, false
	}
}

func securityAttributeNamespaceFromPointer(
	current *securityattributesdk.SecurityAttributeNamespace,
) (securityattributesdk.SecurityAttributeNamespace, bool) {
	if current == nil {
		return securityattributesdk.SecurityAttributeNamespace{}, false
	}
	return *current, true
}

func securityAttributeNamespaceFromSummaryPointer(
	current *securityattributesdk.SecurityAttributeNamespaceSummary,
) (securityattributesdk.SecurityAttributeNamespace, bool) {
	if current == nil {
		return securityattributesdk.SecurityAttributeNamespace{}, false
	}
	return securityAttributeNamespaceFromSummary(*current), true
}

func securityAttributeNamespaceFromCreateResponsePointer(
	current *securityattributesdk.CreateSecurityAttributeNamespaceResponse,
) (securityattributesdk.SecurityAttributeNamespace, bool) {
	if current == nil {
		return securityattributesdk.SecurityAttributeNamespace{}, false
	}
	return current.SecurityAttributeNamespace, true
}

func securityAttributeNamespaceFromGetResponsePointer(
	current *securityattributesdk.GetSecurityAttributeNamespaceResponse,
) (securityattributesdk.SecurityAttributeNamespace, bool) {
	if current == nil {
		return securityattributesdk.SecurityAttributeNamespace{}, false
	}
	return current.SecurityAttributeNamespace, true
}

func securityAttributeNamespaceFromUpdateResponsePointer(
	current *securityattributesdk.UpdateSecurityAttributeNamespaceResponse,
) (securityattributesdk.SecurityAttributeNamespace, bool) {
	if current == nil {
		return securityattributesdk.SecurityAttributeNamespace{}, false
	}
	return current.SecurityAttributeNamespace, true
}

func securityAttributeNamespaceFromSummary(summary securityattributesdk.SecurityAttributeNamespaceSummary) securityattributesdk.SecurityAttributeNamespace {
	return securityattributesdk.SecurityAttributeNamespace{
		Id:             summary.Id,
		CompartmentId:  summary.CompartmentId,
		Name:           summary.Name,
		Description:    summary.Description,
		IsRetired:      summary.IsRetired,
		TimeCreated:    summary.TimeCreated,
		FreeformTags:   cloneSecurityAttributeNamespaceStringMap(summary.FreeformTags),
		DefinedTags:    cloneSecurityAttributeNamespaceDefinedTagMap(summary.DefinedTags),
		SystemTags:     cloneSecurityAttributeNamespaceDefinedTagMap(summary.SystemTags),
		Mode:           append([]string(nil), summary.Mode...),
		LifecycleState: summary.LifecycleState,
	}
}

func securityAttributeNamespaceLifecycleState(response any) string {
	current, ok := securityAttributeNamespaceFromResponse(response)
	if !ok {
		return ""
	}
	return string(current.LifecycleState)
}

func cloneSecurityAttributeNamespaceStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneSecurityAttributeNamespaceDefinedTagMap(source map[string]map[string]interface{}) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	cloned := make(map[string]map[string]interface{}, len(source))
	for namespace, values := range source {
		if values == nil {
			cloned[namespace] = nil
			continue
		}
		clonedValues := make(map[string]interface{}, len(values))
		for key, value := range values {
			clonedValues[key] = value
		}
		cloned[namespace] = clonedValues
	}
	return cloned
}

func stringPtrEqual(current *string, desired string) bool {
	desired = strings.TrimSpace(desired)
	if current == nil {
		return desired == ""
	}
	return strings.TrimSpace(*current) == desired
}

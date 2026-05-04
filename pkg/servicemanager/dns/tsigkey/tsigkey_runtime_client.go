/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package tsigkey

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	dnssdk "github.com/oracle/oci-go-sdk/v65/dns"
	dnsv1beta1 "github.com/oracle/oci-service-operator/api/dns/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
)

type tsigKeyOCIClient interface {
	CreateTsigKey(context.Context, dnssdk.CreateTsigKeyRequest) (dnssdk.CreateTsigKeyResponse, error)
	GetTsigKey(context.Context, dnssdk.GetTsigKeyRequest) (dnssdk.GetTsigKeyResponse, error)
	ListTsigKeys(context.Context, dnssdk.ListTsigKeysRequest) (dnssdk.ListTsigKeysResponse, error)
	UpdateTsigKey(context.Context, dnssdk.UpdateTsigKeyRequest) (dnssdk.UpdateTsigKeyResponse, error)
	DeleteTsigKey(context.Context, dnssdk.DeleteTsigKeyRequest) (dnssdk.DeleteTsigKeyResponse, error)
}

type ambiguousTsigKeyNotFoundError struct {
	message      string
	opcRequestID string
}

func (e ambiguousTsigKeyNotFoundError) Error() string {
	return e.message
}

func (e ambiguousTsigKeyNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerTsigKeyRuntimeHooksMutator(func(manager *TsigKeyServiceManager, hooks *TsigKeyRuntimeHooks) {
		client, initErr := newTsigKeySDKClient(manager)
		applyTsigKeyRuntimeHooks(hooks, client, initErr)
	})
}

func newTsigKeySDKClient(manager *TsigKeyServiceManager) (tsigKeyOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("TsigKey service manager is nil")
	}
	client, err := dnssdk.NewDnsClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyTsigKeyRuntimeHooks(
	hooks *TsigKeyRuntimeHooks,
	client tsigKeyOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newTsigKeyRuntimeSemantics()
	hooks.BuildCreateBody = buildTsigKeyCreateBody
	hooks.BuildUpdateBody = buildTsigKeyUpdateBody
	hooks.Identity.GuardExistingBeforeCreate = guardTsigKeyExistingBeforeCreate
	hooks.List.Fields = tsigKeyListFields()
	hooks.List.Call = func(ctx context.Context, request dnssdk.ListTsigKeysRequest) (dnssdk.ListTsigKeysResponse, error) {
		return listTsigKeysAllPages(ctx, client, initErr, request)
	}
	hooks.Get.Call = func(ctx context.Context, request dnssdk.GetTsigKeyRequest) (dnssdk.GetTsigKeyResponse, error) {
		if initErr != nil {
			return dnssdk.GetTsigKeyResponse{}, fmt.Errorf("initialize TsigKey OCI client: %w", initErr)
		}
		if client == nil {
			return dnssdk.GetTsigKeyResponse{}, fmt.Errorf("TsigKey OCI client is not configured")
		}
		response, err := client.GetTsigKey(ctx, request)
		return response, conservativeTsigKeyNotFoundError(err, "read")
	}
	hooks.Update.Call = func(ctx context.Context, request dnssdk.UpdateTsigKeyRequest) (dnssdk.UpdateTsigKeyResponse, error) {
		if initErr != nil {
			return dnssdk.UpdateTsigKeyResponse{}, fmt.Errorf("initialize TsigKey OCI client: %w", initErr)
		}
		if client == nil {
			return dnssdk.UpdateTsigKeyResponse{}, fmt.Errorf("TsigKey OCI client is not configured")
		}
		return client.UpdateTsigKey(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request dnssdk.DeleteTsigKeyRequest) (dnssdk.DeleteTsigKeyResponse, error) {
		if initErr != nil {
			return dnssdk.DeleteTsigKeyResponse{}, fmt.Errorf("initialize TsigKey OCI client: %w", initErr)
		}
		if client == nil {
			return dnssdk.DeleteTsigKeyResponse{}, fmt.Errorf("TsigKey OCI client is not configured")
		}
		response, err := client.DeleteTsigKey(ctx, request)
		return response, conservativeTsigKeyNotFoundError(err, "delete")
	}
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedTsigKeyIdentity
	hooks.StatusHooks.ProjectStatus = projectTsigKeyStatus
	hooks.ParityHooks.NormalizeDesiredState = normalizeTsigKeyDesiredState
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateTsigKeyCreateOnlyDriftForResponse
	hooks.DeleteHooks.HandleError = handleTsigKeyDeleteError
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate TsigKeyServiceClient) TsigKeyServiceClient {
		return tsigKeyDeleteGuardClient{delegate: delegate, client: client, initErr: initErr}
	})
}

func newTsigKeyServiceClientWithOCIClient(log loggerutil.OSOKLogger, client tsigKeyOCIClient) TsigKeyServiceClient {
	manager := &TsigKeyServiceManager{Log: log}
	hooks := newTsigKeyRuntimeHooksWithOCIClient(client)
	applyTsigKeyRuntimeHooks(&hooks, client, nil)
	delegate := defaultTsigKeyServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*dnsv1beta1.TsigKey](
			buildTsigKeyGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapTsigKeyGeneratedClient(hooks, delegate)
}

func newTsigKeyRuntimeHooksWithOCIClient(client tsigKeyOCIClient) TsigKeyRuntimeHooks {
	return TsigKeyRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*dnsv1beta1.TsigKey]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*dnsv1beta1.TsigKey]{},
		StatusHooks:     generatedruntime.StatusHooks[*dnsv1beta1.TsigKey]{},
		ParityHooks:     generatedruntime.ParityHooks[*dnsv1beta1.TsigKey]{},
		Async:           generatedruntime.AsyncHooks[*dnsv1beta1.TsigKey]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*dnsv1beta1.TsigKey]{},
		Create: runtimeOperationHooks[dnssdk.CreateTsigKeyRequest, dnssdk.CreateTsigKeyResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateTsigKeyDetails", RequestName: "CreateTsigKeyDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request dnssdk.CreateTsigKeyRequest) (dnssdk.CreateTsigKeyResponse, error) {
				return client.CreateTsigKey(ctx, request)
			},
		},
		Get: runtimeOperationHooks[dnssdk.GetTsigKeyRequest, dnssdk.GetTsigKeyResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "TsigKeyId", RequestName: "tsigKeyId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request dnssdk.GetTsigKeyRequest) (dnssdk.GetTsigKeyResponse, error) {
				return client.GetTsigKey(ctx, request)
			},
		},
		List: runtimeOperationHooks[dnssdk.ListTsigKeysRequest, dnssdk.ListTsigKeysResponse]{
			Fields: tsigKeyListFields(),
			Call: func(ctx context.Context, request dnssdk.ListTsigKeysRequest) (dnssdk.ListTsigKeysResponse, error) {
				return client.ListTsigKeys(ctx, request)
			},
		},
		Update: runtimeOperationHooks[dnssdk.UpdateTsigKeyRequest, dnssdk.UpdateTsigKeyResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "TsigKeyId", RequestName: "tsigKeyId", Contribution: "path", PreferResourceID: true}, {FieldName: "UpdateTsigKeyDetails", RequestName: "UpdateTsigKeyDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request dnssdk.UpdateTsigKeyRequest) (dnssdk.UpdateTsigKeyResponse, error) {
				return client.UpdateTsigKey(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[dnssdk.DeleteTsigKeyRequest, dnssdk.DeleteTsigKeyResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "TsigKeyId", RequestName: "tsigKeyId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request dnssdk.DeleteTsigKeyRequest) (dnssdk.DeleteTsigKeyResponse, error) {
				return client.DeleteTsigKey(ctx, request)
			},
		},
		WrapGeneratedClient: []func(TsigKeyServiceClient) TsigKeyServiceClient{},
	}
}

func newTsigKeyRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "dns",
		FormalSlug:    "tsigkey",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(dnssdk.TsigKeyLifecycleStateCreating)},
			UpdatingStates:     []string{string(dnssdk.TsigKeyLifecycleStateUpdating)},
			ActiveStates:       []string{string(dnssdk.TsigKeyLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(dnssdk.TsigKeyLifecycleStateDeleting)},
			TerminalStates: []string{string(dnssdk.TsigKeyLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "name", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"freeformTags", "definedTags"},
			ForceNew:      []string{"algorithm", "compartmentId", "name", "secret"},
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

func tsigKeyListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "Name", RequestName: "name", Contribution: "query", LookupPaths: []string{"status.name", "spec.name", "name"}},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
	}
}

func buildTsigKeyCreateBody(_ context.Context, resource *dnsv1beta1.TsigKey, _ string) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("TsigKey resource is nil")
	}
	normalizeTsigKeySpec(resource)
	if err := validateTsigKeySpec(resource.Spec); err != nil {
		return nil, err
	}

	body := dnssdk.CreateTsigKeyDetails{
		Algorithm:     common.String(resource.Spec.Algorithm),
		Name:          common.String(resource.Spec.Name),
		CompartmentId: common.String(resource.Spec.CompartmentId),
		Secret:        common.String(resource.Spec.Secret),
	}
	if resource.Spec.FreeformTags != nil {
		body.FreeformTags = cloneTsigKeyStringMap(resource.Spec.FreeformTags)
	}
	if resource.Spec.DefinedTags != nil {
		body.DefinedTags = tsigKeyDefinedTagsFromSpec(resource.Spec.DefinedTags)
	}
	return body, nil
}

func buildTsigKeyUpdateBody(
	_ context.Context,
	resource *dnsv1beta1.TsigKey,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return dnssdk.UpdateTsigKeyDetails{}, false, fmt.Errorf("TsigKey resource is nil")
	}
	normalizeTsigKeySpec(resource)
	if err := validateTsigKeySpec(resource.Spec); err != nil {
		return dnssdk.UpdateTsigKeyDetails{}, false, err
	}

	current, ok := tsigKeyFromResponse(currentResponse)
	if !ok {
		return dnssdk.UpdateTsigKeyDetails{}, false, fmt.Errorf("current TsigKey response does not expose a TsigKey body")
	}
	if err := validateTsigKeyCreateOnlyDrift(resource.Spec, current); err != nil {
		return dnssdk.UpdateTsigKeyDetails{}, false, err
	}

	updateDetails := dnssdk.UpdateTsigKeyDetails{}
	updateNeeded := false

	desiredFreeformTags := desiredTsigKeyFreeformTagsForUpdate(resource.Spec.FreeformTags, current.FreeformTags)
	if !reflect.DeepEqual(current.FreeformTags, desiredFreeformTags) {
		updateDetails.FreeformTags = desiredFreeformTags
		updateNeeded = true
	}

	desiredDefinedTags := desiredTsigKeyDefinedTagsForUpdate(resource.Spec.DefinedTags, current.DefinedTags)
	if !reflect.DeepEqual(current.DefinedTags, desiredDefinedTags) {
		updateDetails.DefinedTags = desiredDefinedTags
		updateNeeded = true
	}

	if !updateNeeded {
		return dnssdk.UpdateTsigKeyDetails{}, false, nil
	}
	return updateDetails, true, nil
}

func guardTsigKeyExistingBeforeCreate(_ context.Context, resource *dnsv1beta1.TsigKey) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("TsigKey resource is nil")
	}
	normalizeTsigKeySpec(resource)
	if resource.Spec.CompartmentId == "" || resource.Spec.Name == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func validateTsigKeySpec(spec dnsv1beta1.TsigKeySpec) error {
	var missing []string
	if strings.TrimSpace(spec.Algorithm) == "" {
		missing = append(missing, "algorithm")
	}
	if strings.TrimSpace(spec.Name) == "" {
		missing = append(missing, "name")
	}
	if strings.TrimSpace(spec.CompartmentId) == "" {
		missing = append(missing, "compartmentId")
	}
	if strings.TrimSpace(spec.Secret) == "" {
		missing = append(missing, "secret")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("TsigKey spec is missing required field(s): %s", strings.Join(missing, ", "))
}

func normalizeTsigKeyDesiredState(resource *dnsv1beta1.TsigKey, _ any) {
	normalizeTsigKeySpec(resource)
}

func normalizeTsigKeySpec(resource *dnsv1beta1.TsigKey) {
	if resource == nil {
		return
	}
	resource.Spec.Algorithm = strings.TrimSpace(resource.Spec.Algorithm)
	resource.Spec.Name = strings.TrimSpace(resource.Spec.Name)
	resource.Spec.CompartmentId = strings.TrimSpace(resource.Spec.CompartmentId)
	resource.Spec.Secret = strings.TrimSpace(resource.Spec.Secret)
}

func validateTsigKeyCreateOnlyDriftForResponse(resource *dnsv1beta1.TsigKey, currentResponse any) error {
	if resource == nil {
		return fmt.Errorf("TsigKey resource is nil")
	}
	current, ok := tsigKeyFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current TsigKey response does not expose a TsigKey body")
	}
	return validateTsigKeyCreateOnlyDrift(resource.Spec, current)
}

func validateTsigKeyCreateOnlyDrift(spec dnsv1beta1.TsigKeySpec, current dnssdk.TsigKey) error {
	var drift []string
	if !tsigKeyStringPtrEqual(current.Algorithm, spec.Algorithm) {
		drift = append(drift, "algorithm")
	}
	if !tsigKeyStringPtrEqual(current.CompartmentId, spec.CompartmentId) {
		drift = append(drift, "compartmentId")
	}
	if !tsigKeyStringPtrEqual(current.Name, spec.Name) {
		drift = append(drift, "name")
	}
	if current.Secret != nil && strings.TrimSpace(*current.Secret) != "" && !tsigKeyStringPtrEqual(current.Secret, spec.Secret) {
		drift = append(drift, "secret")
	}
	if len(drift) == 0 {
		return nil
	}
	return fmt.Errorf("TsigKey create-only field drift is not supported: %s", strings.Join(drift, ", "))
}

func desiredTsigKeyFreeformTagsForUpdate(spec map[string]string, current map[string]string) map[string]string {
	if spec != nil {
		return cloneTsigKeyStringMap(spec)
	}
	if current != nil {
		return map[string]string{}
	}
	return nil
}

func desiredTsigKeyDefinedTagsForUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) map[string]map[string]interface{} {
	if spec != nil {
		return tsigKeyDefinedTagsFromSpec(spec)
	}
	if current != nil {
		return map[string]map[string]interface{}{}
	}
	return nil
}

func listTsigKeysAllPages(
	ctx context.Context,
	client tsigKeyOCIClient,
	initErr error,
	request dnssdk.ListTsigKeysRequest,
) (dnssdk.ListTsigKeysResponse, error) {
	if initErr != nil {
		return dnssdk.ListTsigKeysResponse{}, fmt.Errorf("initialize TsigKey OCI client: %w", initErr)
	}
	if client == nil {
		return dnssdk.ListTsigKeysResponse{}, fmt.Errorf("TsigKey OCI client is not configured")
	}

	var combined dnssdk.ListTsigKeysResponse
	for {
		response, err := client.ListTsigKeys(ctx, request)
		if err != nil {
			return dnssdk.ListTsigKeysResponse{}, conservativeTsigKeyNotFoundError(err, "list")
		}
		combined.RawResponse = response.RawResponse
		combined.OpcRequestId = response.OpcRequestId
		for _, item := range response.Items {
			if item.LifecycleState == dnssdk.TsigKeySummaryLifecycleStateDeleted {
				continue
			}
			combined.Items = append(combined.Items, item)
		}
		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			combined.OpcNextPage = nil
			return combined, nil
		}
		request.Page = response.OpcNextPage
		combined.OpcNextPage = response.OpcNextPage
	}
}

func projectTsigKeyStatus(resource *dnsv1beta1.TsigKey, response any) error {
	if resource == nil {
		return fmt.Errorf("TsigKey resource is nil")
	}

	current, ok := tsigKeyFromResponse(response)
	if !ok {
		return nil
	}
	resource.Status = dnsv1beta1.TsigKeyStatus{
		OsokStatus:     resource.Status.OsokStatus,
		Algorithm:      tsigKeyStringValue(current.Algorithm),
		Name:           tsigKeyStringValue(current.Name),
		CompartmentId:  tsigKeyStringValue(current.CompartmentId),
		FreeformTags:   cloneTsigKeyStringMap(current.FreeformTags),
		DefinedTags:    tsigKeyStatusDefinedTags(current.DefinedTags),
		Id:             tsigKeyStringValue(current.Id),
		Self:           tsigKeyStringValue(current.Self),
		TimeCreated:    tsigKeySDKTimeString(current.TimeCreated),
		LifecycleState: string(current.LifecycleState),
		TimeUpdated:    tsigKeySDKTimeString(current.TimeUpdated),
	}
	return nil
}

func clearTrackedTsigKeyIdentity(resource *dnsv1beta1.TsigKey) {
	if resource == nil {
		return
	}
	resource.Status = dnsv1beta1.TsigKeyStatus{}
}

func handleTsigKeyDeleteError(resource *dnsv1beta1.TsigKey, err error) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return err
}

type tsigKeyDeleteGuardClient struct {
	delegate TsigKeyServiceClient
	client   tsigKeyOCIClient
	initErr  error
}

func (c tsigKeyDeleteGuardClient) CreateOrUpdate(
	ctx context.Context,
	resource *dnsv1beta1.TsigKey,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c tsigKeyDeleteGuardClient) Delete(ctx context.Context, resource *dnsv1beta1.TsigKey) (bool, error) {
	if err := c.guardDeleteRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c tsigKeyDeleteGuardClient) guardDeleteRead(ctx context.Context, resource *dnsv1beta1.TsigKey) error {
	if resource == nil {
		return nil
	}
	id := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	if id == "" {
		id = strings.TrimSpace(resource.Status.Id)
	}
	if id == "" {
		return nil
	}
	if c.initErr != nil {
		return fmt.Errorf("initialize TsigKey OCI client: %w", c.initErr)
	}
	if c.client == nil {
		return fmt.Errorf("TsigKey OCI client is not configured")
	}
	_, err := c.client.GetTsigKey(ctx, dnssdk.GetTsigKeyRequest{TsigKeyId: common.String(id)})
	if err == nil || errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
		return nil
	}
	err = conservativeTsigKeyNotFoundError(err, "pre-delete read")
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return err
}

func conservativeTsigKeyNotFoundError(err error, operation string) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	message := fmt.Sprintf("TsigKey %s returned ambiguous 404 NotAuthorizedOrNotFound: %s", strings.TrimSpace(operation), err.Error())
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		return ambiguousTsigKeyNotFoundError{message: message, opcRequestID: serviceErr.GetOpcRequestID()}
	}
	return ambiguousTsigKeyNotFoundError{message: message}
}

func tsigKeyFromResponse(response any) (dnssdk.TsigKey, bool) {
	switch current := response.(type) {
	case dnssdk.CreateTsigKeyResponse:
		return current.TsigKey, true
	case dnssdk.GetTsigKeyResponse:
		return current.TsigKey, true
	case dnssdk.UpdateTsigKeyResponse:
		return current.TsigKey, true
	case dnssdk.TsigKey:
		return current, true
	case dnssdk.TsigKeySummary:
		return tsigKeyFromSummary(current), true
	default:
		return tsigKeyFromPointerResponse(response)
	}
}

func tsigKeyFromPointerResponse(response any) (dnssdk.TsigKey, bool) {
	switch current := response.(type) {
	case *dnssdk.CreateTsigKeyResponse:
		return tsigKeyFromCreateResponsePtr(current)
	case *dnssdk.GetTsigKeyResponse:
		return tsigKeyFromGetResponsePtr(current)
	case *dnssdk.UpdateTsigKeyResponse:
		return tsigKeyFromUpdateResponsePtr(current)
	case *dnssdk.TsigKey:
		return tsigKeyFromPtr(current)
	case *dnssdk.TsigKeySummary:
		return tsigKeyFromSummaryPtr(current)
	default:
		return dnssdk.TsigKey{}, false
	}
}

func tsigKeyFromCreateResponsePtr(response *dnssdk.CreateTsigKeyResponse) (dnssdk.TsigKey, bool) {
	if response == nil {
		return dnssdk.TsigKey{}, false
	}
	return response.TsigKey, true
}

func tsigKeyFromGetResponsePtr(response *dnssdk.GetTsigKeyResponse) (dnssdk.TsigKey, bool) {
	if response == nil {
		return dnssdk.TsigKey{}, false
	}
	return response.TsigKey, true
}

func tsigKeyFromUpdateResponsePtr(response *dnssdk.UpdateTsigKeyResponse) (dnssdk.TsigKey, bool) {
	if response == nil {
		return dnssdk.TsigKey{}, false
	}
	return response.TsigKey, true
}

func tsigKeyFromPtr(tsigKey *dnssdk.TsigKey) (dnssdk.TsigKey, bool) {
	if tsigKey == nil {
		return dnssdk.TsigKey{}, false
	}
	return *tsigKey, true
}

func tsigKeyFromSummaryPtr(summary *dnssdk.TsigKeySummary) (dnssdk.TsigKey, bool) {
	if summary == nil {
		return dnssdk.TsigKey{}, false
	}
	return tsigKeyFromSummary(*summary), true
}

func tsigKeyFromSummary(summary dnssdk.TsigKeySummary) dnssdk.TsigKey {
	lifecycleState := dnssdk.TsigKeyLifecycleStateEnum(summary.LifecycleState)
	return dnssdk.TsigKey{
		Algorithm:      summary.Algorithm,
		Name:           summary.Name,
		CompartmentId:  summary.CompartmentId,
		FreeformTags:   cloneTsigKeyStringMap(summary.FreeformTags),
		DefinedTags:    cloneTsigKeyDefinedTags(summary.DefinedTags),
		Id:             summary.Id,
		Self:           summary.Self,
		TimeCreated:    summary.TimeCreated,
		LifecycleState: lifecycleState,
	}
}

func tsigKeyDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
	if spec == nil {
		return nil
	}
	return *util.ConvertToOciDefinedTags(&spec)
}

func tsigKeyStatusDefinedTags(input map[string]map[string]interface{}) map[string]shared.MapValue {
	if input == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(input))
	for key, values := range input {
		if values == nil {
			converted[key] = nil
			continue
		}
		tagValues := make(shared.MapValue, len(values))
		for innerKey, innerValue := range values {
			tagValues[innerKey] = fmt.Sprint(innerValue)
		}
		converted[key] = tagValues
	}
	return converted
}

func cloneTsigKeyStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func cloneTsigKeyDefinedTags(input map[string]map[string]interface{}) map[string]map[string]interface{} {
	if input == nil {
		return nil
	}
	cloned := make(map[string]map[string]interface{}, len(input))
	for key, values := range input {
		if values == nil {
			cloned[key] = nil
			continue
		}
		inner := make(map[string]interface{}, len(values))
		for innerKey, innerValue := range values {
			inner[innerKey] = innerValue
		}
		cloned[key] = inner
	}
	return cloned
}

func tsigKeyStringPtrEqual(actual *string, expected string) bool {
	return strings.TrimSpace(tsigKeyStringValue(actual)) == strings.TrimSpace(expected)
}

func tsigKeyStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func tsigKeySDKTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.String()
}

/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package protecteddatabase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	recoverysdk "github.com/oracle/oci-go-sdk/v65/recovery"
	recoveryv1beta1 "github.com/oracle/oci-service-operator/api/recovery/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	protectedDatabaseDeletePendingMessage    = "OCI ProtectedDatabase delete is in progress"
	protectedDatabaseLegacyPasswordHashKey   = "osokPasswordSHA256="
	protectedDatabasePasswordStateHashKey    = "passwordSHA256"
	protectedDatabasePasswordStateSecretKind = "ProtectedDatabase"
)

var protectedDatabaseWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(recoverysdk.OperationStatusAccepted),
		string(recoverysdk.OperationStatusWaiting),
		string(recoverysdk.OperationStatusInProgress),
		string(recoverysdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(recoverysdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(recoverysdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(recoverysdk.OperationStatusCanceled)},
	CreateActionTokens:    []string{string(recoverysdk.ActionTypeCreated)},
	UpdateActionTokens:    []string{string(recoverysdk.ActionTypeUpdated)},
	DeleteActionTokens:    []string{string(recoverysdk.ActionTypeDeleted)},
}

type protectedDatabaseOCIClient interface {
	CreateProtectedDatabase(context.Context, recoverysdk.CreateProtectedDatabaseRequest) (recoverysdk.CreateProtectedDatabaseResponse, error)
	GetProtectedDatabase(context.Context, recoverysdk.GetProtectedDatabaseRequest) (recoverysdk.GetProtectedDatabaseResponse, error)
	ListProtectedDatabases(context.Context, recoverysdk.ListProtectedDatabasesRequest) (recoverysdk.ListProtectedDatabasesResponse, error)
	UpdateProtectedDatabase(context.Context, recoverysdk.UpdateProtectedDatabaseRequest) (recoverysdk.UpdateProtectedDatabaseResponse, error)
	ChangeProtectedDatabaseCompartment(context.Context, recoverysdk.ChangeProtectedDatabaseCompartmentRequest) (recoverysdk.ChangeProtectedDatabaseCompartmentResponse, error)
	ChangeProtectedDatabaseSubscription(context.Context, recoverysdk.ChangeProtectedDatabaseSubscriptionRequest) (recoverysdk.ChangeProtectedDatabaseSubscriptionResponse, error)
	DeleteProtectedDatabase(context.Context, recoverysdk.DeleteProtectedDatabaseRequest) (recoverysdk.DeleteProtectedDatabaseResponse, error)
	GetWorkRequest(context.Context, recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error)
}

type protectedDatabaseAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e protectedDatabaseAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e protectedDatabaseAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerProtectedDatabaseRuntimeHooksMutator(func(manager *ProtectedDatabaseServiceManager, hooks *ProtectedDatabaseRuntimeHooks) {
		client, initErr := newProtectedDatabaseSDKClient(manager)
		applyProtectedDatabaseRuntimeHooks(manager, hooks, client, initErr)
	})
}

func newProtectedDatabaseSDKClient(manager *ProtectedDatabaseServiceManager) (protectedDatabaseOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("ProtectedDatabase service manager is nil")
	}
	client, err := recoverysdk.NewDatabaseRecoveryClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyProtectedDatabaseRuntimeHooks(
	manager *ProtectedDatabaseServiceManager,
	hooks *ProtectedDatabaseRuntimeHooks,
	client protectedDatabaseOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = protectedDatabaseRuntimeSemantics()
	hooks.BuildCreateBody = buildProtectedDatabaseCreateBody
	hooks.BuildUpdateBody = buildProtectedDatabaseUpdateBody
	configureProtectedDatabaseOperationHooks(hooks, client, initErr)
	hooks.List.Fields = protectedDatabaseListFields()
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateProtectedDatabaseCreateOnlyDriftForResponse
	hooks.DeleteHooks.HandleError = handleProtectedDatabaseDeleteError
	hooks.StatusHooks.MarkTerminating = markProtectedDatabaseTerminating
	hooks.Async.Adapter = protectedDatabaseWorkRequestAsyncAdapter
	configureProtectedDatabaseAsyncHooks(hooks, client, initErr)
	wrapProtectedDatabaseChangeOperations(hooks, client, initErr)
	wrapProtectedDatabaseDeleteConfirmation(hooks)
	wrapProtectedDatabasePasswordTracking(manager, hooks)
}

func configureProtectedDatabaseOperationHooks(
	hooks *ProtectedDatabaseRuntimeHooks,
	client protectedDatabaseOCIClient,
	initErr error,
) {
	hooks.Create.Call = protectedDatabaseCreateCall(client, initErr)
	hooks.Get.Call = protectedDatabaseGetCall(client, initErr)
	hooks.List.Call = func(ctx context.Context, request recoverysdk.ListProtectedDatabasesRequest) (recoverysdk.ListProtectedDatabasesResponse, error) {
		return listProtectedDatabasesAllPages(ctx, client, initErr, request)
	}
	hooks.Update.Call = protectedDatabaseUpdateCall(client, initErr)
	hooks.Delete.Call = protectedDatabaseDeleteCall(client, initErr)
}

func configureProtectedDatabaseAsyncHooks(
	hooks *ProtectedDatabaseRuntimeHooks,
	client protectedDatabaseOCIClient,
	initErr error,
) {
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getProtectedDatabaseWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveProtectedDatabaseGeneratedWorkRequestAction
	hooks.Async.ResolvePhase = resolveProtectedDatabaseGeneratedWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverProtectedDatabaseIDFromGeneratedWorkRequest
	hooks.Async.Message = protectedDatabaseGeneratedWorkRequestMessage
}

func protectedDatabaseCreateCall(
	client protectedDatabaseOCIClient,
	initErr error,
) func(context.Context, recoverysdk.CreateProtectedDatabaseRequest) (recoverysdk.CreateProtectedDatabaseResponse, error) {
	return func(ctx context.Context, request recoverysdk.CreateProtectedDatabaseRequest) (recoverysdk.CreateProtectedDatabaseResponse, error) {
		if err := validateProtectedDatabaseOCIClient(client, initErr); err != nil {
			return recoverysdk.CreateProtectedDatabaseResponse{}, err
		}
		return client.CreateProtectedDatabase(ctx, request)
	}
}

func protectedDatabaseGetCall(
	client protectedDatabaseOCIClient,
	initErr error,
) func(context.Context, recoverysdk.GetProtectedDatabaseRequest) (recoverysdk.GetProtectedDatabaseResponse, error) {
	return func(ctx context.Context, request recoverysdk.GetProtectedDatabaseRequest) (recoverysdk.GetProtectedDatabaseResponse, error) {
		if err := validateProtectedDatabaseOCIClient(client, initErr); err != nil {
			return recoverysdk.GetProtectedDatabaseResponse{}, err
		}
		response, err := client.GetProtectedDatabase(ctx, request)
		return response, conservativeProtectedDatabaseNotFoundError(err, "read")
	}
}

func protectedDatabaseUpdateCall(
	client protectedDatabaseOCIClient,
	initErr error,
) func(context.Context, recoverysdk.UpdateProtectedDatabaseRequest) (recoverysdk.UpdateProtectedDatabaseResponse, error) {
	return func(ctx context.Context, request recoverysdk.UpdateProtectedDatabaseRequest) (recoverysdk.UpdateProtectedDatabaseResponse, error) {
		if err := validateProtectedDatabaseOCIClient(client, initErr); err != nil {
			return recoverysdk.UpdateProtectedDatabaseResponse{}, err
		}
		return client.UpdateProtectedDatabase(ctx, request)
	}
}

func protectedDatabaseDeleteCall(
	client protectedDatabaseOCIClient,
	initErr error,
) func(context.Context, recoverysdk.DeleteProtectedDatabaseRequest) (recoverysdk.DeleteProtectedDatabaseResponse, error) {
	return func(ctx context.Context, request recoverysdk.DeleteProtectedDatabaseRequest) (recoverysdk.DeleteProtectedDatabaseResponse, error) {
		if err := validateProtectedDatabaseOCIClient(client, initErr); err != nil {
			return recoverysdk.DeleteProtectedDatabaseResponse{}, err
		}
		response, err := client.DeleteProtectedDatabase(ctx, request)
		return response, conservativeProtectedDatabaseNotFoundError(err, "delete")
	}
}

func validateProtectedDatabaseOCIClient(client protectedDatabaseOCIClient, initErr error) error {
	if initErr != nil {
		return fmt.Errorf("initialize ProtectedDatabase OCI client: %w", initErr)
	}
	if client == nil {
		return fmt.Errorf("ProtectedDatabase OCI client is not configured")
	}
	return nil
}

func newProtectedDatabaseServiceClientWithOCIClientAndCredentialClient(
	log loggerutil.OSOKLogger,
	client protectedDatabaseOCIClient,
	credentialClient credhelper.CredentialClient,
) ProtectedDatabaseServiceClient {
	manager := &ProtectedDatabaseServiceManager{Log: log, CredentialClient: credentialClient}
	hooks := newProtectedDatabaseRuntimeHooksWithOCIClient(client)
	applyProtectedDatabaseRuntimeHooks(manager, &hooks, client, nil)
	delegate := defaultProtectedDatabaseServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*recoveryv1beta1.ProtectedDatabase](
			buildProtectedDatabaseGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapProtectedDatabaseGeneratedClient(hooks, delegate)
}

func newProtectedDatabaseRuntimeHooksWithOCIClient(client protectedDatabaseOCIClient) ProtectedDatabaseRuntimeHooks {
	hooks := newProtectedDatabaseDefaultRuntimeHooks(recoverysdk.DatabaseRecoveryClient{})
	hooks.Create.Call = func(ctx context.Context, request recoverysdk.CreateProtectedDatabaseRequest) (recoverysdk.CreateProtectedDatabaseResponse, error) {
		return client.CreateProtectedDatabase(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request recoverysdk.GetProtectedDatabaseRequest) (recoverysdk.GetProtectedDatabaseResponse, error) {
		return client.GetProtectedDatabase(ctx, request)
	}
	hooks.List.Call = func(ctx context.Context, request recoverysdk.ListProtectedDatabasesRequest) (recoverysdk.ListProtectedDatabasesResponse, error) {
		return client.ListProtectedDatabases(ctx, request)
	}
	hooks.Update.Call = func(ctx context.Context, request recoverysdk.UpdateProtectedDatabaseRequest) (recoverysdk.UpdateProtectedDatabaseResponse, error) {
		return client.UpdateProtectedDatabase(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request recoverysdk.DeleteProtectedDatabaseRequest) (recoverysdk.DeleteProtectedDatabaseResponse, error) {
		return client.DeleteProtectedDatabase(ctx, request)
	}
	return hooks
}

func protectedDatabaseChangeCall(
	client protectedDatabaseOCIClient,
	initErr error,
) protectedDatabaseChangeOperations {
	return protectedDatabaseChangeOperations{
		changeCompartment: func(ctx context.Context, request recoverysdk.ChangeProtectedDatabaseCompartmentRequest) (recoverysdk.ChangeProtectedDatabaseCompartmentResponse, error) {
			if err := validateProtectedDatabaseOCIClient(client, initErr); err != nil {
				return recoverysdk.ChangeProtectedDatabaseCompartmentResponse{}, err
			}
			return client.ChangeProtectedDatabaseCompartment(ctx, request)
		},
		changeSubscription: func(ctx context.Context, request recoverysdk.ChangeProtectedDatabaseSubscriptionRequest) (recoverysdk.ChangeProtectedDatabaseSubscriptionResponse, error) {
			if err := validateProtectedDatabaseOCIClient(client, initErr); err != nil {
				return recoverysdk.ChangeProtectedDatabaseSubscriptionResponse{}, err
			}
			return client.ChangeProtectedDatabaseSubscription(ctx, request)
		},
		getWorkRequest: func(ctx context.Context, workRequestID string) (any, error) {
			return getProtectedDatabaseWorkRequest(ctx, client, initErr, workRequestID)
		},
	}
}

type protectedDatabaseChangeOperations struct {
	changeCompartment  func(context.Context, recoverysdk.ChangeProtectedDatabaseCompartmentRequest) (recoverysdk.ChangeProtectedDatabaseCompartmentResponse, error)
	changeSubscription func(context.Context, recoverysdk.ChangeProtectedDatabaseSubscriptionRequest) (recoverysdk.ChangeProtectedDatabaseSubscriptionResponse, error)
	getWorkRequest     func(context.Context, string) (any, error)
}

func protectedDatabaseRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "recovery",
		FormalSlug:    "protecteddatabase",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "workrequest",
			Runtime:              "generatedruntime",
			FormalClassification: "workrequest",
			WorkRequest: &generatedruntime.WorkRequestSemantics{
				Source: "service-sdk",
				Phases: []string{"create", "update", "delete"},
			},
		},
		StatusProjection:  "required",
		SecretSideEffects: "controller-owned-password-state",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(recoverysdk.LifecycleStateCreating)},
			UpdatingStates:     []string{string(recoverysdk.LifecycleStateUpdating)},
			ActiveStates:       []string{string(recoverysdk.LifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy: "required",
			PendingStates: []string{
				string(recoverysdk.LifecycleStateDeleteScheduled),
				string(recoverysdk.LifecycleStateDeleting),
			},
			TerminalStates: []string{string(recoverysdk.LifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "dbUniqueName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"compartmentId",
				"displayName",
				"databaseSize",
				"databaseSizeInGBs",
				"password",
				"protectionPolicyId",
				"recoveryServiceSubnets",
				"isRedoLogsShipped",
				"subscriptionId",
				"freeformTags",
				"definedTags",
			},
			ForceNew: []string{
				"dbUniqueName",
				"databaseId",
				"changeRate",
				"compressionRatio",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{
				{Helper: "tfresource.CreateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "protectedDatabase", Action: "CREATED"},
			},
			Update: []generatedruntime.Hook{
				{Helper: "tfresource.UpdateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "protectedDatabase", Action: "UPDATED"},
			},
			Delete: []generatedruntime.Hook{
				{Helper: "tfresource.DeleteResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "protectedDatabase", Action: "DELETED"},
			},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetProtectedDatabase",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.CreateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "protectedDatabase", Action: "CREATED"},
			},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetProtectedDatabase",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.UpdateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "protectedDatabase", Action: "UPDATED"},
			},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.DeleteResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "protectedDatabase", Action: "DELETED"},
			},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func protectedDatabaseListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "spec.compartmentId", "compartmentId"},
		},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
	}
}

func buildProtectedDatabaseCreateBody(
	_ context.Context,
	resource *recoveryv1beta1.ProtectedDatabase,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("ProtectedDatabase resource is nil")
	}
	if err := validateProtectedDatabaseSpec(resource.Spec); err != nil {
		return nil, err
	}

	body := recoverysdk.CreateProtectedDatabaseDetails{
		DisplayName:            common.String(strings.TrimSpace(resource.Spec.DisplayName)),
		DbUniqueName:           common.String(strings.TrimSpace(resource.Spec.DbUniqueName)),
		Password:               common.String(resource.Spec.Password),
		ProtectionPolicyId:     common.String(strings.TrimSpace(resource.Spec.ProtectionPolicyId)),
		RecoveryServiceSubnets: protectedDatabaseRecoveryServiceSubnetInputs(resource.Spec.RecoveryServiceSubnets),
		CompartmentId:          common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
	}
	if err := setProtectedDatabaseCreateOptionalFields(&body, resource.Spec); err != nil {
		return nil, err
	}
	return body, nil
}

func setProtectedDatabaseCreateOptionalFields(
	body *recoverysdk.CreateProtectedDatabaseDetails,
	spec recoveryv1beta1.ProtectedDatabaseSpec,
) error {
	if body == nil {
		return fmt.Errorf("ProtectedDatabase create body is nil")
	}
	if err := setProtectedDatabaseCreateDatabaseSize(body, spec.DatabaseSize); err != nil {
		return err
	}
	setProtectedDatabaseCreateStorageFields(body, spec)
	setProtectedDatabaseCreateConnectionFields(body, spec)
	setProtectedDatabaseCreateTagFields(body, spec)
	return nil
}

func setProtectedDatabaseCreateDatabaseSize(
	body *recoverysdk.CreateProtectedDatabaseDetails,
	databaseSizeValue string,
) error {
	if value := strings.TrimSpace(databaseSizeValue); value != "" {
		databaseSize, err := protectedDatabaseSize(value)
		if err != nil {
			return err
		}
		body.DatabaseSize = databaseSize
	}
	return nil
}

func setProtectedDatabaseCreateStorageFields(
	body *recoverysdk.CreateProtectedDatabaseDetails,
	spec recoveryv1beta1.ProtectedDatabaseSpec,
) {
	if value := strings.TrimSpace(spec.DatabaseId); value != "" {
		body.DatabaseId = common.String(value)
	}
	if spec.DatabaseSizeInGBs != 0 {
		body.DatabaseSizeInGBs = common.Int(spec.DatabaseSizeInGBs)
	}
	if spec.ChangeRate != 0 {
		body.ChangeRate = common.Float64(spec.ChangeRate)
	}
	if spec.CompressionRatio != 0 {
		body.CompressionRatio = common.Float64(spec.CompressionRatio)
	}
	if spec.IsRedoLogsShipped {
		body.IsRedoLogsShipped = common.Bool(true)
	}
}

func setProtectedDatabaseCreateConnectionFields(
	body *recoverysdk.CreateProtectedDatabaseDetails,
	spec recoveryv1beta1.ProtectedDatabaseSpec,
) {
	if value := strings.TrimSpace(spec.SubscriptionId); value != "" {
		body.SubscriptionId = common.String(value)
	}
}

func setProtectedDatabaseCreateTagFields(
	body *recoverysdk.CreateProtectedDatabaseDetails,
	spec recoveryv1beta1.ProtectedDatabaseSpec,
) {
	if spec.FreeformTags != nil {
		body.FreeformTags = cloneProtectedDatabaseStringMap(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		body.DefinedTags = protectedDatabaseDefinedTags(spec.DefinedTags)
	}
}

func buildProtectedDatabaseUpdateBody(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectedDatabase,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("ProtectedDatabase resource is nil")
	}
	if err := validateProtectedDatabaseSpec(resource.Spec); err != nil {
		return nil, false, err
	}
	current, ok := protectedDatabaseFromResponse(currentResponse)
	if !ok {
		return nil, false, fmt.Errorf("current ProtectedDatabase response does not expose a ProtectedDatabase body")
	}
	if err := validateProtectedDatabaseCreateOnlyDrift(resource.Spec, current); err != nil {
		return nil, false, err
	}
	if err := validateProtectedDatabasePasswordTracking(ctx, resource); err != nil {
		return nil, false, err
	}

	body := recoverysdk.UpdateProtectedDatabaseDetails{}
	updateNeeded := false
	setProtectedDatabaseStringUpdate(&body.DisplayName, &updateNeeded, resource.Spec.DisplayName, current.DisplayName)
	if err := setProtectedDatabaseSizeUpdate(&body, &updateNeeded, resource.Spec.DatabaseSize, current.DatabaseSize); err != nil {
		return nil, false, err
	}
	setProtectedDatabaseIntUpdate(&body.DatabaseSizeInGBs, &updateNeeded, resource.Spec.DatabaseSizeInGBs, current.DatabaseSizeInGBs)
	setProtectedDatabasePasswordUpdate(ctx, &body, &updateNeeded, resource)
	setProtectedDatabaseStringUpdate(&body.ProtectionPolicyId, &updateNeeded, resource.Spec.ProtectionPolicyId, current.ProtectionPolicyId)
	if setProtectedDatabaseSubnetUpdate(&body, resource.Spec.RecoveryServiceSubnets, current.RecoveryServiceSubnets) {
		updateNeeded = true
	}
	setProtectedDatabaseBoolUpdate(&body.IsRedoLogsShipped, &updateNeeded, resource.Spec.IsRedoLogsShipped, current.IsRedoLogsShipped)
	setProtectedDatabaseTagUpdate(&body, &updateNeeded, resource.Spec, current)

	if !updateNeeded {
		return recoverysdk.UpdateProtectedDatabaseDetails{}, false, nil
	}
	return body, true, nil
}

func validateProtectedDatabaseSpec(spec recoveryv1beta1.ProtectedDatabaseSpec) error {
	var missing []string
	if strings.TrimSpace(spec.DisplayName) == "" {
		missing = append(missing, "displayName")
	}
	if strings.TrimSpace(spec.DbUniqueName) == "" {
		missing = append(missing, "dbUniqueName")
	}
	if strings.TrimSpace(spec.Password) == "" {
		missing = append(missing, "password")
	}
	if strings.TrimSpace(spec.ProtectionPolicyId) == "" {
		missing = append(missing, "protectionPolicyId")
	}
	if strings.TrimSpace(spec.CompartmentId) == "" {
		missing = append(missing, "compartmentId")
	}
	if len(spec.RecoveryServiceSubnets) == 0 {
		missing = append(missing, "recoveryServiceSubnets")
	}
	for index, subnet := range spec.RecoveryServiceSubnets {
		if strings.TrimSpace(subnet.RecoveryServiceSubnetId) == "" {
			missing = append(missing, fmt.Sprintf("recoveryServiceSubnets[%d].recoveryServiceSubnetId", index))
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("ProtectedDatabase spec missing required fields: %s", strings.Join(missing, ", "))
	}
	return nil
}

func protectedDatabaseSize(value string) (recoverysdk.DatabaseSizesEnum, error) {
	if databaseSize, ok := recoverysdk.GetMappingDatabaseSizesEnum(strings.TrimSpace(value)); ok {
		return databaseSize, nil
	}
	return "", fmt.Errorf("unsupported databaseSize %q", value)
}

func protectedDatabaseRecoveryServiceSubnetInputs(
	subnets []recoveryv1beta1.ProtectedDatabaseRecoveryServiceSubnet,
) []recoverysdk.RecoveryServiceSubnetInput {
	if subnets == nil {
		return nil
	}
	result := make([]recoverysdk.RecoveryServiceSubnetInput, 0, len(subnets))
	for _, subnet := range subnets {
		result = append(result, recoverysdk.RecoveryServiceSubnetInput{
			RecoveryServiceSubnetId: common.String(strings.TrimSpace(subnet.RecoveryServiceSubnetId)),
		})
	}
	return result
}

func setProtectedDatabaseStringUpdate(target **string, updateNeeded *bool, desired string, current *string) {
	value := strings.TrimSpace(desired)
	if value == "" {
		return
	}
	if current == nil || strings.TrimSpace(*current) != value {
		*target = common.String(value)
		*updateNeeded = true
	}
}

func setProtectedDatabaseSizeUpdate(
	body *recoverysdk.UpdateProtectedDatabaseDetails,
	updateNeeded *bool,
	desired string,
	current recoverysdk.DatabaseSizesEnum,
) error {
	value := strings.TrimSpace(desired)
	if value == "" {
		return nil
	}
	databaseSize, err := protectedDatabaseSize(value)
	if err != nil {
		return err
	}
	if current != databaseSize {
		body.DatabaseSize = databaseSize
		*updateNeeded = true
	}
	return nil
}

func setProtectedDatabaseIntUpdate(target **int, updateNeeded *bool, desired int, current *int) {
	if desired == 0 {
		return
	}
	if current == nil || *current != desired {
		*target = common.Int(desired)
		*updateNeeded = true
	}
}

func setProtectedDatabasePasswordUpdate(
	ctx context.Context,
	body *recoverysdk.UpdateProtectedDatabaseDetails,
	updateNeeded *bool,
	resource *recoveryv1beta1.ProtectedDatabase,
) {
	desired := resource.Spec.Password
	if strings.TrimSpace(desired) == "" {
		return
	}
	state, ok := protectedDatabasePasswordStateFromContext(ctx)
	if !ok || !state.hasRecorded || protectedDatabasePasswordHash(desired) == state.recordedHash {
		return
	}
	body.Password = common.String(desired)
	*updateNeeded = true
}

func setProtectedDatabaseBoolUpdate(target **bool, updateNeeded *bool, desired bool, current *bool) {
	if current == nil {
		if desired {
			*target = common.Bool(true)
			*updateNeeded = true
		}
		return
	}
	if *current != desired {
		*target = common.Bool(desired)
		*updateNeeded = true
	}
}

func setProtectedDatabaseSubnetUpdate(
	body *recoverysdk.UpdateProtectedDatabaseDetails,
	desired []recoveryv1beta1.ProtectedDatabaseRecoveryServiceSubnet,
	current []recoverysdk.RecoveryServiceSubnetDetails,
) bool {
	desiredSubnets := protectedDatabaseRecoveryServiceSubnetInputs(desired)
	changed := !reflect.DeepEqual(protectedDatabaseSubnetInputIDs(desiredSubnets), protectedDatabaseSubnetDetailIDs(current))
	if changed {
		body.RecoveryServiceSubnets = desiredSubnets
	}
	return changed
}

func setProtectedDatabaseTagUpdate(
	body *recoverysdk.UpdateProtectedDatabaseDetails,
	updateNeeded *bool,
	spec recoveryv1beta1.ProtectedDatabaseSpec,
	current recoverysdk.ProtectedDatabase,
) {
	if spec.FreeformTags != nil && !reflect.DeepEqual(current.FreeformTags, spec.FreeformTags) {
		desiredFreeformTags := cloneProtectedDatabaseStringMap(spec.FreeformTags)
		body.FreeformTags = desiredFreeformTags
		*updateNeeded = true
	}
	if spec.DefinedTags != nil {
		desiredDefinedTags := protectedDatabaseDefinedTags(spec.DefinedTags)
		if !reflect.DeepEqual(current.DefinedTags, desiredDefinedTags) {
			body.DefinedTags = desiredDefinedTags
			*updateNeeded = true
		}
	}
}

func validateProtectedDatabaseCreateOnlyDriftForResponse(
	resource *recoveryv1beta1.ProtectedDatabase,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("ProtectedDatabase resource is nil")
	}
	current, ok := protectedDatabaseFromResponse(currentResponse)
	if !ok {
		return nil
	}
	if err := validateProtectedDatabaseCreateOnlyDrift(resource.Spec, current); err != nil {
		return err
	}
	return nil
}

func validateProtectedDatabaseCreateOnlyDrift(
	spec recoveryv1beta1.ProtectedDatabaseSpec,
	current recoverysdk.ProtectedDatabase,
) error {
	if err := rejectProtectedDatabaseStringDrift("dbUniqueName", spec.DbUniqueName, current.DbUniqueName, true); err != nil {
		return err
	}
	if err := rejectProtectedDatabaseStringDrift("databaseId", spec.DatabaseId, current.DatabaseId, false); err != nil {
		return err
	}
	if err := rejectProtectedDatabaseFloatDrift("changeRate", spec.ChangeRate, current.ChangeRate); err != nil {
		return err
	}
	if err := rejectProtectedDatabaseFloatDrift("compressionRatio", spec.CompressionRatio, current.CompressionRatio); err != nil {
		return err
	}
	return nil
}

func rejectProtectedDatabaseStringDrift(field string, desired string, current *string, required bool) error {
	desired = strings.TrimSpace(desired)
	observed := ""
	if current != nil {
		observed = strings.TrimSpace(*current)
	}
	if required && desired == "" {
		return fmt.Errorf("ProtectedDatabase %s is required", field)
	}
	if !required && desired == "" {
		return nil
	}
	if desired == observed {
		return nil
	}
	if desired == "" && observed == "" {
		return nil
	}
	return fmt.Errorf("ProtectedDatabase formal semantics require replacement when %s changes", field)
}

func rejectProtectedDatabaseFloatDrift(field string, desired float64, current *float64) error {
	if desired == 0 {
		return nil
	}
	observed := 0.0
	if current != nil {
		observed = *current
	}
	if desired == observed {
		return nil
	}
	if desired == 0 && observed == 0 {
		return nil
	}
	return fmt.Errorf("ProtectedDatabase formal semantics require replacement when %s changes", field)
}

func validateProtectedDatabasePasswordTracking(ctx context.Context, resource *recoveryv1beta1.ProtectedDatabase) error {
	if resource == nil {
		return fmt.Errorf("ProtectedDatabase resource is nil")
	}
	if strings.TrimSpace(resource.Spec.Password) == "" || !protectedDatabaseHasTrackedIdentity(resource) {
		return nil
	}
	state, ok := protectedDatabasePasswordStateFromContext(ctx)
	if ok && state.loadErr != nil {
		return state.loadErr
	}
	if ok && state.hasRecorded {
		return nil
	}
	if resource.Status.OsokStatus.CreatedAt == nil {
		return nil
	}
	return fmt.Errorf("ProtectedDatabase password update cannot be validated because the controller-owned password tracking secret is missing")
}

func getProtectedDatabaseWorkRequest(
	ctx context.Context,
	client protectedDatabaseOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize ProtectedDatabase OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("ProtectedDatabase OCI client is not configured")
	}
	response, err := client.GetWorkRequest(ctx, recoverysdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveProtectedDatabaseGeneratedWorkRequestAction(workRequest any) (string, error) {
	protectedDatabaseWorkRequest, err := protectedDatabaseWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return resolveProtectedDatabaseWorkRequestAction(protectedDatabaseWorkRequest)
}

func resolveProtectedDatabaseGeneratedWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	protectedDatabaseWorkRequest, err := protectedDatabaseWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := protectedDatabaseWorkRequestPhaseFromOperationType(protectedDatabaseWorkRequest.OperationType)
	return phase, ok, nil
}

func recoverProtectedDatabaseIDFromGeneratedWorkRequest(
	_ *recoveryv1beta1.ProtectedDatabase,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	protectedDatabaseWorkRequest, err := protectedDatabaseWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return resolveProtectedDatabaseIDFromWorkRequest(
		protectedDatabaseWorkRequest,
		protectedDatabaseWorkRequestActionForPhase(phase),
	)
}

func protectedDatabaseWorkRequestFromAny(workRequest any) (recoverysdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case recoverysdk.WorkRequest:
		return current, nil
	case *recoverysdk.WorkRequest:
		if current == nil {
			return recoverysdk.WorkRequest{}, fmt.Errorf("ProtectedDatabase work request is nil")
		}
		return *current, nil
	default:
		return recoverysdk.WorkRequest{}, fmt.Errorf("unexpected ProtectedDatabase work request type %T", workRequest)
	}
}

func resolveProtectedDatabaseWorkRequestAction(workRequest recoverysdk.WorkRequest) (string, error) {
	var action string
	for _, resource := range workRequest.Resources {
		if !isProtectedDatabaseWorkRequestResource(resource) {
			continue
		}
		candidate := strings.TrimSpace(string(resource.ActionType))
		if candidate == "" || strings.EqualFold(candidate, string(recoverysdk.ActionTypeInProgress)) {
			continue
		}
		if action == "" {
			action = candidate
			continue
		}
		if action != candidate {
			return "", fmt.Errorf("ProtectedDatabase work request %s exposes conflicting ProtectedDatabase action types %q and %q", stringValue(workRequest.Id), action, candidate)
		}
	}
	return action, nil
}

func protectedDatabaseWorkRequestPhaseFromOperationType(operationType recoverysdk.OperationTypeEnum) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case recoverysdk.OperationTypeCreateProtectedDatabase:
		return shared.OSOKAsyncPhaseCreate, true
	case recoverysdk.OperationTypeUpdateProtectedDatabase, recoverysdk.OperationTypeMoveProtectedDatabase:
		return shared.OSOKAsyncPhaseUpdate, true
	case recoverysdk.OperationTypeDeleteProtectedDatabase:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func protectedDatabaseWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) recoverysdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return recoverysdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return recoverysdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return recoverysdk.ActionTypeDeleted
	default:
		return ""
	}
}

func resolveProtectedDatabaseIDFromWorkRequest(
	workRequest recoverysdk.WorkRequest,
	action recoverysdk.ActionTypeEnum,
) (string, error) {
	if id, ok := resolveProtectedDatabaseIDFromResources(workRequest.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resolveProtectedDatabaseIDFromResources(workRequest.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("ProtectedDatabase work request %s does not expose a ProtectedDatabase identifier", stringValue(workRequest.Id))
}

func resolveProtectedDatabaseIDFromResources(
	resources []recoverysdk.WorkRequestResource,
	action recoverysdk.ActionTypeEnum,
	preferProtectedDatabaseOnly bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferProtectedDatabaseOnly && !isProtectedDatabaseWorkRequestResource(resource) {
			continue
		}
		id := strings.TrimSpace(stringValue(resource.Identifier))
		if id == "" {
			continue
		}
		if candidate == "" {
			candidate = id
			continue
		}
		if candidate != id {
			return "", false
		}
	}
	return candidate, candidate != ""
}

func isProtectedDatabaseWorkRequestResource(resource recoverysdk.WorkRequestResource) bool {
	entityType := strings.ToLower(strings.TrimSpace(stringValue(resource.EntityType)))
	switch entityType {
	case "protecteddatabase", "protected_database", "protected-database", "protecteddatabases", "protected_database_resource":
		return true
	}
	if strings.Contains(entityType, "protecteddatabase") || strings.Contains(entityType, "protected_database") {
		return true
	}
	entityURI := strings.ToLower(strings.TrimSpace(stringValue(resource.EntityUri)))
	return strings.Contains(entityURI, "/protecteddatabases/")
}

func protectedDatabaseGeneratedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	protectedDatabaseWorkRequest, err := protectedDatabaseWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("ProtectedDatabase %s work request %s is %s", phase, stringValue(protectedDatabaseWorkRequest.Id), protectedDatabaseWorkRequest.Status)
}

func listProtectedDatabasesAllPages(
	ctx context.Context,
	client protectedDatabaseOCIClient,
	initErr error,
	request recoverysdk.ListProtectedDatabasesRequest,
) (recoverysdk.ListProtectedDatabasesResponse, error) {
	if initErr != nil {
		return recoverysdk.ListProtectedDatabasesResponse{}, fmt.Errorf("initialize ProtectedDatabase OCI client: %w", initErr)
	}
	if client == nil {
		return recoverysdk.ListProtectedDatabasesResponse{}, fmt.Errorf("ProtectedDatabase OCI client is not configured")
	}

	var combined recoverysdk.ListProtectedDatabasesResponse
	for {
		response, err := client.ListProtectedDatabases(ctx, request)
		if err != nil {
			return recoverysdk.ListProtectedDatabasesResponse{}, conservativeProtectedDatabaseNotFoundError(err, "list")
		}
		combined.RawResponse = response.RawResponse
		combined.OpcRequestId = response.OpcRequestId
		for _, item := range response.Items {
			if item.LifecycleState == recoverysdk.LifecycleStateDeleted {
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

func handleProtectedDatabaseDeleteError(resource *recoveryv1beta1.ProtectedDatabase, err error) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return err
}

func wrapProtectedDatabaseDeleteConfirmation(hooks *ProtectedDatabaseRuntimeHooks) {
	if hooks == nil || hooks.Get.Call == nil {
		return
	}
	getProtectedDatabase := hooks.Get.Call
	getWorkRequest := hooks.Async.GetWorkRequest
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ProtectedDatabaseServiceClient) ProtectedDatabaseServiceClient {
		return protectedDatabaseDeleteConfirmationClient{
			delegate:             delegate,
			getProtectedDatabase: getProtectedDatabase,
			getWorkRequest:       getWorkRequest,
		}
	})
}

func wrapProtectedDatabaseChangeOperations(
	hooks *ProtectedDatabaseRuntimeHooks,
	client protectedDatabaseOCIClient,
	initErr error,
) {
	if hooks == nil || hooks.Get.Call == nil {
		return
	}
	changeOperations := protectedDatabaseChangeCall(client, initErr)
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ProtectedDatabaseServiceClient) ProtectedDatabaseServiceClient {
		return protectedDatabaseChangeOperationClient{
			delegate:         delegate,
			getProtectedData: hooks.Get.Call,
			changes:          changeOperations,
		}
	})
}

func wrapProtectedDatabasePasswordTracking(manager *ProtectedDatabaseServiceManager, hooks *ProtectedDatabaseRuntimeHooks) {
	if hooks == nil {
		return
	}
	var credentialClient credhelper.CredentialClient
	if manager != nil {
		credentialClient = manager.CredentialClient
	}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ProtectedDatabaseServiceClient) ProtectedDatabaseServiceClient {
		return protectedDatabasePasswordTrackingClient{
			delegate:         delegate,
			credentialClient: credentialClient,
		}
	})
}

type protectedDatabasePasswordTrackingClient struct {
	delegate         ProtectedDatabaseServiceClient
	credentialClient credhelper.CredentialClient
}

func (c protectedDatabasePasswordTrackingClient) CreateOrUpdate(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectedDatabase,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	trackingCtx, state := c.preparePasswordTracking(ctx, resource)
	if state.loadErr != nil {
		return protectedDatabaseFailure(resource, state.loadErr)
	}

	response, err := c.delegate.CreateOrUpdate(trackingCtx, resource, req)
	stripProtectedDatabaseLegacyPasswordHash(resource)
	switch {
	case shouldRecordProtectedDatabasePasswordHash(resource, response, err):
		if syncErr := c.syncPasswordStateSecret(ctx, resource, protectedDatabasePasswordHash(resource.Spec.Password)); syncErr != nil {
			return protectedDatabaseFailure(resource, syncErr)
		}
	case state.hasRecorded:
		clearFailedProtectedDatabasePasswordUpdate(resource, state.recordedHash, err)
	}
	return response, err
}

func (c protectedDatabasePasswordTrackingClient) Delete(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectedDatabase,
) (bool, error) {
	deleted, err := c.delegate.Delete(ctx, resource)
	if err != nil || !deleted {
		return deleted, err
	}
	if err := c.deletePasswordStateSecret(ctx, resource); err != nil {
		return false, err
	}
	return deleted, nil
}

type protectedDatabasePasswordStateContextKey struct{}

type protectedDatabasePasswordState struct {
	recordedHash string
	hasRecorded  bool
	loadErr      error
}

func protectedDatabasePasswordStateFromContext(ctx context.Context) (protectedDatabasePasswordState, bool) {
	state, ok := ctx.Value(protectedDatabasePasswordStateContextKey{}).(protectedDatabasePasswordState)
	return state, ok
}

func contextWithProtectedDatabasePasswordState(
	ctx context.Context,
	state protectedDatabasePasswordState,
) context.Context {
	return context.WithValue(ctx, protectedDatabasePasswordStateContextKey{}, state)
}

func (c protectedDatabasePasswordTrackingClient) preparePasswordTracking(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectedDatabase,
) (context.Context, protectedDatabasePasswordState) {
	state := protectedDatabasePasswordState{}
	if !protectedDatabaseShouldTrackPassword(resource) {
		return contextWithProtectedDatabasePasswordState(ctx, state), state
	}
	legacyHash, hasLegacyHash := protectedDatabaseRecordedLegacyPasswordHash(resource)
	stripProtectedDatabaseLegacyPasswordHash(resource)
	if c.credentialClient == nil {
		state.loadErr = fmt.Errorf("ProtectedDatabase password tracking secret credential client is not configured")
		return contextWithProtectedDatabasePasswordState(ctx, state), state
	}

	state = c.loadPasswordTrackingState(ctx, resource, legacyHash, hasLegacyHash)
	return contextWithProtectedDatabasePasswordState(ctx, state), state
}

func protectedDatabaseShouldTrackPassword(resource *recoveryv1beta1.ProtectedDatabase) bool {
	return resource != nil && strings.TrimSpace(resource.Spec.Password) != ""
}

func (c protectedDatabasePasswordTrackingClient) loadPasswordTrackingState(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectedDatabase,
	legacyHash string,
	hasLegacyHash bool,
) protectedDatabasePasswordState {
	data, err := c.credentialClient.GetSecret(ctx, protectedDatabasePasswordStateSecretName(resource), resource.Namespace)
	switch {
	case err == nil:
		return c.passwordStateFromExistingSecret(ctx, resource, data, legacyHash, hasLegacyHash)
	case servicemanager.IsSecretNotFoundError(err):
		return c.passwordStateFromMissingSecret(ctx, resource, legacyHash, hasLegacyHash)
	default:
		return protectedDatabasePasswordState{
			loadErr: fmt.Errorf("read ProtectedDatabase password tracking secret: %w", err),
		}
	}
}

func (c protectedDatabasePasswordTrackingClient) passwordStateFromExistingSecret(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectedDatabase,
	data map[string][]byte,
	legacyHash string,
	hasLegacyHash bool,
) protectedDatabasePasswordState {
	state := protectedDatabasePasswordStateFromSecretData(resource, data)
	if state.hasRecorded || !hasLegacyHash {
		return state
	}
	return c.passwordStateFromLegacyHash(ctx, resource, legacyHash)
}

func (c protectedDatabasePasswordTrackingClient) passwordStateFromMissingSecret(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectedDatabase,
	legacyHash string,
	hasLegacyHash bool,
) protectedDatabasePasswordState {
	if !hasLegacyHash {
		return protectedDatabasePasswordState{}
	}
	return c.passwordStateFromLegacyHash(ctx, resource, legacyHash)
}

func (c protectedDatabasePasswordTrackingClient) passwordStateFromLegacyHash(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectedDatabase,
	legacyHash string,
) protectedDatabasePasswordState {
	if err := c.syncPasswordStateSecret(ctx, resource, legacyHash); err != nil {
		return protectedDatabasePasswordState{loadErr: err}
	}
	return protectedDatabasePasswordState{
		recordedHash: legacyHash,
		hasRecorded:  true,
	}
}

func protectedDatabasePasswordStateFromSecretData(
	resource *recoveryv1beta1.ProtectedDatabase,
	data map[string][]byte,
) protectedDatabasePasswordState {
	if !servicemanager.SecretOwnedBy(data, protectedDatabasePasswordStateSecretKind, protectedDatabasePasswordStateOwnerName(resource)) {
		return protectedDatabasePasswordState{
			loadErr: fmt.Errorf(
				"ProtectedDatabase password tracking secret %s/%s is not owned by this ProtectedDatabase",
				resource.Namespace,
				protectedDatabasePasswordStateSecretName(resource),
			),
		}
	}
	hash, ok := protectedDatabasePasswordHashFromBytes(data[protectedDatabasePasswordStateHashKey])
	return protectedDatabasePasswordState{recordedHash: hash, hasRecorded: ok}
}

func (c protectedDatabasePasswordTrackingClient) syncPasswordStateSecret(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectedDatabase,
	hash string,
) error {
	if resource == nil {
		return nil
	}
	if err := c.validatePasswordStateSecretWrite(hash); err != nil {
		return err
	}
	name := protectedDatabasePasswordStateSecretName(resource)
	data := protectedDatabasePasswordStateSecretData(resource, hash)
	labels := servicemanager.ManagedSecretLabels(protectedDatabasePasswordStateSecretKind, protectedDatabasePasswordStateOwnerName(resource))

	created, err := c.createPasswordStateSecret(ctx, resource, name, labels, data)
	if created || err != nil {
		return err
	}
	return c.updatePasswordStateSecretIfNeeded(ctx, resource, name, labels, data, hash)
}

func (c protectedDatabasePasswordTrackingClient) validatePasswordStateSecretWrite(hash string) error {
	if c.credentialClient == nil {
		return fmt.Errorf("ProtectedDatabase password tracking secret credential client is not configured")
	}
	if _, ok := protectedDatabasePasswordHashFromBytes([]byte(hash)); !ok {
		return fmt.Errorf("ProtectedDatabase password tracking secret hash is invalid")
	}
	return nil
}

func (c protectedDatabasePasswordTrackingClient) createPasswordStateSecret(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectedDatabase,
	name string,
	labels map[string]string,
	data map[string][]byte,
) (bool, error) {
	ok, err := c.credentialClient.CreateSecret(ctx, name, resource.Namespace, labels, data)
	if err == nil {
		return ok, nil
	}
	if apierrors.IsAlreadyExists(err) {
		return false, nil
	}
	return false, fmt.Errorf("create ProtectedDatabase password tracking secret: %w", err)
}

func (c protectedDatabasePasswordTrackingClient) updatePasswordStateSecretIfNeeded(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectedDatabase,
	name string,
	labels map[string]string,
	data map[string][]byte,
	hash string,
) error {
	existing, err := c.credentialClient.GetSecret(ctx, name, resource.Namespace)
	if err != nil {
		return fmt.Errorf("read existing ProtectedDatabase password tracking secret: %w", err)
	}
	state := protectedDatabasePasswordStateFromSecretData(resource, existing)
	if state.loadErr != nil {
		return state.loadErr
	}
	if state.hasRecorded && state.recordedHash == hash {
		return nil
	}
	if _, updateErr := c.credentialClient.UpdateSecret(ctx, name, resource.Namespace, labels, data); updateErr != nil {
		return fmt.Errorf("update ProtectedDatabase password tracking secret: %w", updateErr)
	}
	return nil
}

func (c protectedDatabasePasswordTrackingClient) deletePasswordStateSecret(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectedDatabase,
) error {
	if resource == nil || c.credentialClient == nil {
		return nil
	}
	_, err := servicemanager.DeleteOwnedSecretIfPresent(
		ctx,
		c.credentialClient,
		protectedDatabasePasswordStateSecretName(resource),
		resource.Namespace,
		protectedDatabasePasswordStateSecretKind,
		protectedDatabasePasswordStateOwnerName(resource),
	)
	if err != nil {
		return fmt.Errorf("delete ProtectedDatabase password tracking secret: %w", err)
	}
	return nil
}

type protectedDatabaseChangeOperationClient struct {
	delegate         ProtectedDatabaseServiceClient
	getProtectedData func(context.Context, recoverysdk.GetProtectedDatabaseRequest) (recoverysdk.GetProtectedDatabaseResponse, error)
	changes          protectedDatabaseChangeOperations
}

func (c protectedDatabaseChangeOperationClient) CreateOrUpdate(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectedDatabase,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return protectedDatabaseFailure(resource, fmt.Errorf("ProtectedDatabase generated delegate is not configured"))
	}
	if resource == nil || !protectedDatabaseHasTrackedIdentity(resource) || c.getProtectedData == nil {
		return c.delegate.CreateOrUpdate(ctx, resource, req)
	}
	if !protectedDatabaseStatusSuggestsChangeOperation(resource) {
		return c.delegate.CreateOrUpdate(ctx, resource, req)
	}

	current, err := c.readProtectedDatabase(ctx, trackedProtectedDatabaseID(resource))
	if err != nil {
		return c.delegate.CreateOrUpdate(ctx, resource, req)
	}
	if !protectedDatabaseNeedsChangeOperation(resource, current) || protectedDatabaseLifecycleBlocksChange(current) {
		return c.delegate.CreateOrUpdate(ctx, resource, req)
	}

	handled, response, err := c.applyChangeOperations(ctx, resource, current)
	if handled {
		return response, err
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c protectedDatabaseChangeOperationClient) Delete(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectedDatabase,
) (bool, error) {
	return c.delegate.Delete(ctx, resource)
}

func (c protectedDatabaseChangeOperationClient) readProtectedDatabase(
	ctx context.Context,
	protectedDatabaseID string,
) (recoverysdk.ProtectedDatabase, error) {
	response, err := c.getProtectedData(ctx, recoverysdk.GetProtectedDatabaseRequest{
		ProtectedDatabaseId: common.String(strings.TrimSpace(protectedDatabaseID)),
	})
	if err != nil {
		return recoverysdk.ProtectedDatabase{}, err
	}
	return response.ProtectedDatabase, nil
}

func (c protectedDatabaseChangeOperationClient) applyChangeOperations(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectedDatabase,
	current recoverysdk.ProtectedDatabase,
) (bool, servicemanager.OSOKResponse, error) {
	currentID := protectedDatabaseIDForChange(resource, current)
	if currentID == "" {
		return protectedDatabaseHandledFailure(resource, fmt.Errorf("ProtectedDatabase change operation requires a tracked protected database OCID"))
	}

	refreshed, handled, response, err := c.applyCompartmentChangeIfNeeded(ctx, resource, currentID, current)
	if handled || err != nil {
		return handled, response, err
	}
	current = refreshed
	return c.applySubscriptionChangeIfNeeded(ctx, resource, currentID, current)
}

func protectedDatabaseIDForChange(
	resource *recoveryv1beta1.ProtectedDatabase,
	current recoverysdk.ProtectedDatabase,
) string {
	if currentID := stringValue(current.Id); currentID != "" {
		return currentID
	}
	return trackedProtectedDatabaseID(resource)
}

func (c protectedDatabaseChangeOperationClient) applyCompartmentChangeIfNeeded(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectedDatabase,
	protectedDatabaseID string,
	current recoverysdk.ProtectedDatabase,
) (recoverysdk.ProtectedDatabase, bool, servicemanager.OSOKResponse, error) {
	if !protectedDatabaseCompartmentNeedsChange(resource, current) {
		return current, false, servicemanager.OSOKResponse{}, nil
	}
	handled, response, err := c.changeCompartment(ctx, resource, protectedDatabaseID)
	if handled || err != nil {
		return recoverysdk.ProtectedDatabase{}, handled, response, err
	}
	refreshed, err := c.readProtectedDatabase(ctx, protectedDatabaseID)
	if err != nil {
		handled, response, err := protectedDatabaseHandledFailure(resource, fmt.Errorf("confirm ProtectedDatabase compartment change: %w", err))
		return recoverysdk.ProtectedDatabase{}, handled, response, err
	}
	if protectedDatabaseCompartmentNeedsChange(resource, refreshed) || protectedDatabaseLifecycleBlocksChange(refreshed) {
		return recoverysdk.ProtectedDatabase{}, true, markProtectedDatabaseUpdatePending(resource, refreshed), nil
	}
	return refreshed, false, servicemanager.OSOKResponse{}, nil
}

func (c protectedDatabaseChangeOperationClient) applySubscriptionChangeIfNeeded(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectedDatabase,
	protectedDatabaseID string,
	current recoverysdk.ProtectedDatabase,
) (bool, servicemanager.OSOKResponse, error) {
	if !protectedDatabaseSubscriptionNeedsChange(resource, current) {
		return false, servicemanager.OSOKResponse{}, nil
	}
	handled, response, err := c.changeSubscription(ctx, resource, protectedDatabaseID)
	if handled || err != nil {
		return handled, response, err
	}
	refreshed, err := c.readProtectedDatabase(ctx, protectedDatabaseID)
	if err != nil {
		return protectedDatabaseHandledFailure(resource, fmt.Errorf("confirm ProtectedDatabase subscription change: %w", err))
	}
	if protectedDatabaseSubscriptionNeedsChange(resource, refreshed) || protectedDatabaseLifecycleBlocksChange(refreshed) {
		return true, markProtectedDatabaseUpdatePending(resource, refreshed), nil
	}
	return false, servicemanager.OSOKResponse{}, nil
}

func (c protectedDatabaseChangeOperationClient) changeCompartment(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectedDatabase,
	protectedDatabaseID string,
) (bool, servicemanager.OSOKResponse, error) {
	if c.changes.changeCompartment == nil {
		return protectedDatabaseHandledFailure(resource, fmt.Errorf("ProtectedDatabase compartment change operation is not configured"))
	}
	response, err := c.changes.changeCompartment(ctx, recoverysdk.ChangeProtectedDatabaseCompartmentRequest{
		ProtectedDatabaseId: common.String(protectedDatabaseID),
		ChangeProtectedDatabaseCompartmentDetails: recoverysdk.ChangeProtectedDatabaseCompartmentDetails{
			CompartmentId: common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		},
	})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return protectedDatabaseHandledFailure(resource, fmt.Errorf("change ProtectedDatabase compartment: %w", err))
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	return c.followChangeWorkRequest(ctx, resource, response.OpcWorkRequestId, shared.OSOKAsyncPhaseUpdate)
}

func (c protectedDatabaseChangeOperationClient) changeSubscription(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectedDatabase,
	protectedDatabaseID string,
) (bool, servicemanager.OSOKResponse, error) {
	if c.changes.changeSubscription == nil {
		return protectedDatabaseHandledFailure(resource, fmt.Errorf("ProtectedDatabase subscription change operation is not configured"))
	}
	response, err := c.changes.changeSubscription(ctx, recoverysdk.ChangeProtectedDatabaseSubscriptionRequest{
		ProtectedDatabaseId: common.String(protectedDatabaseID),
		ChangeProtectedDatabaseSubscriptionDetails: recoverysdk.ChangeProtectedDatabaseSubscriptionDetails{
			SubscriptionId: common.String(strings.TrimSpace(resource.Spec.SubscriptionId)),
		},
	})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return protectedDatabaseHandledFailure(resource, fmt.Errorf("change ProtectedDatabase subscription: %w", err))
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	return c.followChangeWorkRequest(ctx, resource, response.OpcWorkRequestId, shared.OSOKAsyncPhaseUpdate)
}

func (c protectedDatabaseChangeOperationClient) followChangeWorkRequest(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectedDatabase,
	workRequestID *string,
	fallbackPhase shared.OSOKAsyncPhase,
) (bool, servicemanager.OSOKResponse, error) {
	if strings.TrimSpace(stringValue(workRequestID)) == "" {
		return false, servicemanager.OSOKResponse{}, nil
	}

	if c.changes.getWorkRequest == nil {
		return protectedDatabaseHandledFailure(resource, fmt.Errorf("ProtectedDatabase work request operation is not configured"))
	}
	workRequest, err := c.changes.getWorkRequest(ctx, stringValue(workRequestID))
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return protectedDatabaseHandledFailure(resource, fmt.Errorf("get ProtectedDatabase change work request: %w", err))
	}
	typedWorkRequest, err := protectedDatabaseWorkRequestFromAny(workRequest)
	if err != nil {
		return protectedDatabaseHandledFailure(resource, err)
	}
	current, err := buildProtectedDatabaseChangeAsyncOperation(&resource.Status.OsokStatus, typedWorkRequest, fallbackPhase)
	if err != nil {
		return protectedDatabaseHandledFailure(resource, err)
	}
	class := current.NormalizedClass
	if class == shared.OSOKAsyncClassSucceeded {
		return false, servicemanager.OSOKResponse{}, nil
	}
	if class != shared.OSOKAsyncClassPending {
		response := applyProtectedDatabaseAsyncOperation(resource, current)
		return true, response, fmt.Errorf("ProtectedDatabase update work request %s finished with status %s", stringValue(typedWorkRequest.Id), typedWorkRequest.Status)
	}
	return true, applyProtectedDatabaseAsyncOperation(resource, current), nil
}

func protectedDatabaseNeedsChangeOperation(
	resource *recoveryv1beta1.ProtectedDatabase,
	current recoverysdk.ProtectedDatabase,
) bool {
	return protectedDatabaseCompartmentNeedsChange(resource, current) ||
		protectedDatabaseSubscriptionNeedsChange(resource, current)
}

func protectedDatabaseStatusSuggestsChangeOperation(resource *recoveryv1beta1.ProtectedDatabase) bool {
	if resource == nil {
		return false
	}
	if desired := strings.TrimSpace(resource.Spec.CompartmentId); desired != "" && desired != strings.TrimSpace(resource.Status.CompartmentId) {
		return true
	}
	if desired := strings.TrimSpace(resource.Spec.SubscriptionId); desired != "" && desired != strings.TrimSpace(resource.Status.SubscriptionId) {
		return true
	}
	return false
}

func protectedDatabaseCompartmentNeedsChange(
	resource *recoveryv1beta1.ProtectedDatabase,
	current recoverysdk.ProtectedDatabase,
) bool {
	if resource == nil {
		return false
	}
	desired := strings.TrimSpace(resource.Spec.CompartmentId)
	if desired == "" {
		return false
	}
	return desired != stringValue(current.CompartmentId)
}

func protectedDatabaseSubscriptionNeedsChange(
	resource *recoveryv1beta1.ProtectedDatabase,
	current recoverysdk.ProtectedDatabase,
) bool {
	if resource == nil {
		return false
	}
	desired := strings.TrimSpace(resource.Spec.SubscriptionId)
	if desired == "" {
		return false
	}
	return desired != stringValue(current.SubscriptionId)
}

func protectedDatabaseLifecycleBlocksChange(current recoverysdk.ProtectedDatabase) bool {
	switch strings.ToUpper(strings.TrimSpace(string(current.LifecycleState))) {
	case string(recoverysdk.LifecycleStateCreating),
		string(recoverysdk.LifecycleStateUpdating),
		string(recoverysdk.LifecycleStateDeleteScheduled),
		string(recoverysdk.LifecycleStateDeleting):
		return true
	default:
		return false
	}
}

func buildProtectedDatabaseChangeAsyncOperation(
	status *shared.OSOKStatus,
	workRequest recoverysdk.WorkRequest,
	fallbackPhase shared.OSOKAsyncPhase,
) (*shared.OSOKAsyncOperation, error) {
	rawAction, err := resolveProtectedDatabaseWorkRequestAction(workRequest)
	if err != nil {
		return nil, err
	}
	if phase, ok := protectedDatabaseWorkRequestPhaseFromOperationType(workRequest.OperationType); ok {
		fallbackPhase = phase
	}
	current, err := servicemanager.BuildWorkRequestAsyncOperation(status, protectedDatabaseWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(workRequest.Status),
		RawAction:        rawAction,
		RawOperationType: string(workRequest.OperationType),
		WorkRequestID:    stringValue(workRequest.Id),
		PercentComplete:  workRequest.PercentComplete,
		FallbackPhase:    fallbackPhase,
	})
	if err != nil {
		return nil, err
	}
	if message := protectedDatabaseGeneratedWorkRequestMessage(current.Phase, workRequest); message != "" {
		current.Message = message
	}
	return current, nil
}

func applyProtectedDatabaseAsyncOperation(
	resource *recoveryv1beta1.ProtectedDatabase,
	current *shared.OSOKAsyncOperation,
) servicemanager.OSOKResponse {
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}
	now := metav1.Now()
	current.UpdatedAt = &now
	projection := servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, loggerutil.OSOKLogger{})
	return servicemanager.OSOKResponse{
		IsSuccessful:    projection.Condition != shared.Failed,
		ShouldRequeue:   projection.ShouldRequeue,
		RequeueDuration: time.Minute,
	}
}

func markProtectedDatabaseUpdatePending(
	resource *recoveryv1beta1.ProtectedDatabase,
	current recoverysdk.ProtectedDatabase,
) servicemanager.OSOKResponse {
	projectProtectedDatabaseStatusFields(resource, current)
	message := "OCI ProtectedDatabase update is in progress"
	async := servicemanager.NewLifecycleAsyncOperation(
		&resource.Status.OsokStatus,
		string(current.LifecycleState),
		message,
		shared.OSOKAsyncPhaseUpdate,
	)
	if async == nil {
		now := metav1.Now()
		async = &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           shared.OSOKAsyncPhaseUpdate,
			RawStatus:       string(current.LifecycleState),
			NormalizedClass: shared.OSOKAsyncClassPending,
			Message:         message,
			UpdatedAt:       &now,
		}
	}
	return applyProtectedDatabaseAsyncOperation(resource, async)
}

func projectProtectedDatabaseStatusFields(
	resource *recoveryv1beta1.ProtectedDatabase,
	current recoverysdk.ProtectedDatabase,
) {
	if resource == nil {
		return
	}
	if id := stringValue(current.Id); id != "" {
		resource.Status.Id = id
		resource.Status.OsokStatus.Ocid = shared.OCID(id)
	}
	if compartmentID := stringValue(current.CompartmentId); compartmentID != "" {
		resource.Status.CompartmentId = compartmentID
	}
	if subscriptionID := stringValue(current.SubscriptionId); subscriptionID != "" {
		resource.Status.SubscriptionId = subscriptionID
	}
	resource.Status.LifecycleState = string(current.LifecycleState)
}

func protectedDatabaseFailure(
	resource *recoveryv1beta1.ProtectedDatabase,
	err error,
) (servicemanager.OSOKResponse, error) {
	if resource == nil || err == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	status := &resource.Status.OsokStatus
	servicemanager.RecordErrorOpcRequestID(status, err)
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	now := metav1.Now()
	status.UpdatedAt = &now
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, corev1.ConditionFalse, "", err.Error(), loggerutil.OSOKLogger{})
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func protectedDatabaseHandledFailure(
	resource *recoveryv1beta1.ProtectedDatabase,
	err error,
) (bool, servicemanager.OSOKResponse, error) {
	response, err := protectedDatabaseFailure(resource, err)
	return true, response, err
}

type protectedDatabaseDeleteConfirmationClient struct {
	delegate             ProtectedDatabaseServiceClient
	getProtectedDatabase func(context.Context, recoverysdk.GetProtectedDatabaseRequest) (recoverysdk.GetProtectedDatabaseResponse, error)
	getWorkRequest       func(context.Context, string) (any, error)
}

func (c protectedDatabaseDeleteConfirmationClient) CreateOrUpdate(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectedDatabase,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c protectedDatabaseDeleteConfirmationClient) Delete(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectedDatabase,
) (bool, error) {
	if handled, err := c.guardPendingWriteBeforeDelete(ctx, resource); handled || err != nil {
		return false, err
	}
	if protectedDatabasePendingDeleteWorkRequestID(resource) == "" {
		if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
			return false, err
		}
	}
	return c.delegate.Delete(ctx, resource)
}

func (c protectedDatabaseDeleteConfirmationClient) guardPendingWriteBeforeDelete(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectedDatabase,
) (bool, error) {
	current, ok := protectedDatabasePendingWriteWorkRequest(resource)
	if !ok {
		return false, nil
	}

	workRequest, err := c.fetchPendingWriteWorkRequest(ctx, current)
	if err != nil {
		return true, protectedDatabasePendingWriteDeleteError(resource, current, err)
	}
	next, err := buildProtectedDatabaseChangeAsyncOperation(&resource.Status.OsokStatus, workRequest, current.Phase)
	if err != nil {
		return true, protectedDatabasePendingWriteDeleteError(resource, current, err)
	}

	switch next.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		applyProtectedDatabaseAsyncOperation(resource, next)
		return true, nil
	case shared.OSOKAsyncClassSucceeded:
		if err := c.confirmPendingWriteReadback(ctx, resource, workRequest, next); err != nil {
			return true, err
		}
		return false, nil
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		err := fmt.Errorf("ProtectedDatabase %s work request %s finished with status %s; refusing delete until the write is resolved", next.Phase, next.WorkRequestID, next.RawStatus)
		return true, protectedDatabasePendingWriteDeleteError(resource, next, err)
	default:
		err := fmt.Errorf("ProtectedDatabase %s work request %s projected unsupported async class %s", next.Phase, next.WorkRequestID, next.NormalizedClass)
		return true, protectedDatabasePendingWriteDeleteError(resource, next, err)
	}
}

func (c protectedDatabaseDeleteConfirmationClient) fetchPendingWriteWorkRequest(
	ctx context.Context,
	current *shared.OSOKAsyncOperation,
) (recoverysdk.WorkRequest, error) {
	if c.getWorkRequest == nil {
		return recoverysdk.WorkRequest{}, fmt.Errorf("ProtectedDatabase work request operation is not configured")
	}
	workRequestID := strings.TrimSpace(current.WorkRequestID)
	if workRequestID == "" {
		return recoverysdk.WorkRequest{}, fmt.Errorf("ProtectedDatabase pending %s work request is missing a work request ID", current.Phase)
	}
	workRequest, err := c.getWorkRequest(ctx, workRequestID)
	if err != nil {
		return recoverysdk.WorkRequest{}, err
	}
	return protectedDatabaseWorkRequestFromAny(workRequest)
}

func (c protectedDatabaseDeleteConfirmationClient) confirmPendingWriteReadback(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectedDatabase,
	workRequest recoverysdk.WorkRequest,
	current *shared.OSOKAsyncOperation,
) error {
	protectedDatabaseID, err := protectedDatabaseIDForPendingWriteReadback(resource, workRequest, current.Phase)
	if err != nil {
		return protectedDatabasePendingWriteDeleteError(resource, current, err)
	}
	if c.getProtectedDatabase == nil {
		err := fmt.Errorf("ProtectedDatabase read operation is not configured")
		return protectedDatabasePendingWriteDeleteError(resource, current, err)
	}

	response, err := c.getProtectedDatabase(ctx, recoverysdk.GetProtectedDatabaseRequest{
		ProtectedDatabaseId: common.String(protectedDatabaseID),
	})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return fmt.Errorf("ProtectedDatabase %s work request %s succeeded but readback is not unambiguous: %w", current.Phase, current.WorkRequestID, err)
	}
	observedID := stringValue(response.Id)
	if observedID == "" {
		err := fmt.Errorf("ProtectedDatabase %s work request %s readback did not return a ProtectedDatabase id", current.Phase, current.WorkRequestID)
		return protectedDatabasePendingWriteDeleteError(resource, current, err)
	}
	if observedID != protectedDatabaseID {
		err := fmt.Errorf("ProtectedDatabase %s work request %s readback returned id %q, want %q", current.Phase, current.WorkRequestID, observedID, protectedDatabaseID)
		return protectedDatabasePendingWriteDeleteError(resource, current, err)
	}

	projectProtectedDatabaseStatusFields(resource, response.ProtectedDatabase)
	if resource.Status.OsokStatus.Async.Current != nil &&
		resource.Status.OsokStatus.Async.Current.Source == shared.OSOKAsyncSourceWorkRequest &&
		resource.Status.OsokStatus.Async.Current.Phase == current.Phase {
		resource.Status.OsokStatus.Async.Current = nil
	}
	return nil
}

func protectedDatabaseIDForPendingWriteReadback(
	resource *recoveryv1beta1.ProtectedDatabase,
	workRequest recoverysdk.WorkRequest,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	if id := trackedProtectedDatabaseID(resource); id != "" {
		return id, nil
	}
	id, err := resolveProtectedDatabaseIDFromWorkRequest(workRequest, protectedDatabaseWorkRequestActionForPhase(phase))
	if err != nil {
		return "", err
	}
	if id == "" {
		return "", fmt.Errorf("ProtectedDatabase %s work request %s did not expose a ProtectedDatabase identifier", phase, stringValue(workRequest.Id))
	}
	return id, nil
}

func protectedDatabasePendingWriteDeleteError(
	resource *recoveryv1beta1.ProtectedDatabase,
	current *shared.OSOKAsyncOperation,
	err error,
) error {
	if err == nil {
		return nil
	}
	if current != nil && resource != nil {
		applyProtectedDatabaseAsyncOperation(resource, current)
	}
	_, failureErr := protectedDatabaseFailure(resource, err)
	return failureErr
}

func (c protectedDatabaseDeleteConfirmationClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectedDatabase,
) error {
	if c.getProtectedDatabase == nil || resource == nil {
		return nil
	}
	protectedDatabaseID := trackedProtectedDatabaseID(resource)
	if protectedDatabaseID == "" {
		return nil
	}
	_, err := c.getProtectedDatabase(ctx, recoverysdk.GetProtectedDatabaseRequest{
		ProtectedDatabaseId: common.String(protectedDatabaseID),
	})
	if err == nil {
		return nil
	}
	if !isProtectedDatabaseAmbiguousNotFound(err) && !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("ProtectedDatabase delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to call delete: %w", err)
}

func trackedProtectedDatabaseID(resource *recoveryv1beta1.ProtectedDatabase) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func protectedDatabasePendingDeleteWorkRequestID(resource *recoveryv1beta1.ProtectedDatabase) string {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return ""
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != shared.OSOKAsyncSourceWorkRequest || current.Phase != shared.OSOKAsyncPhaseDelete {
		return ""
	}
	return strings.TrimSpace(current.WorkRequestID)
}

func protectedDatabasePendingWriteWorkRequest(resource *recoveryv1beta1.ProtectedDatabase) (*shared.OSOKAsyncOperation, bool) {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return nil, false
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != shared.OSOKAsyncSourceWorkRequest || current.NormalizedClass != shared.OSOKAsyncClassPending {
		return nil, false
	}
	switch current.Phase {
	case shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncPhaseUpdate:
		return current, true
	default:
		return nil, false
	}
}

func protectedDatabaseHasTrackedIdentity(resource *recoveryv1beta1.ProtectedDatabase) bool {
	return trackedProtectedDatabaseID(resource) != ""
}

func shouldRecordProtectedDatabasePasswordHash(
	resource *recoveryv1beta1.ProtectedDatabase,
	response servicemanager.OSOKResponse,
	err error,
) bool {
	if err != nil || !response.IsSuccessful || response.ShouldRequeue || !protectedDatabaseHasTrackedIdentity(resource) {
		return false
	}
	current := protectedDatabaseCurrentAsync(resource)
	return current == nil || current.NormalizedClass == shared.OSOKAsyncClassSucceeded
}

func clearFailedProtectedDatabasePasswordUpdate(
	resource *recoveryv1beta1.ProtectedDatabase,
	recorded string,
	err error,
) {
	if err == nil || resource == nil || protectedDatabasePasswordHash(resource.Spec.Password) == recorded {
		return
	}
	current := protectedDatabaseCurrentAsync(resource)
	if current == nil || current.Phase != shared.OSOKAsyncPhaseUpdate {
		return
	}
	switch current.NormalizedClass {
	case shared.OSOKAsyncClassFailed,
		shared.OSOKAsyncClassCanceled,
		shared.OSOKAsyncClassAttention,
		shared.OSOKAsyncClassUnknown:
		resource.Status.OsokStatus.Async.Current = nil
	}
}

func protectedDatabaseCurrentAsync(resource *recoveryv1beta1.ProtectedDatabase) *shared.OSOKAsyncOperation {
	if resource == nil {
		return nil
	}
	return resource.Status.OsokStatus.Async.Current
}

func stripProtectedDatabaseLegacyPasswordHash(resource *recoveryv1beta1.ProtectedDatabase) {
	if resource == nil {
		return
	}
	resource.Status.OsokStatus.Message = stripProtectedDatabaseLegacyPasswordHashFromMessage(resource.Status.OsokStatus.Message)
}

func protectedDatabaseRecordedLegacyPasswordHash(resource *recoveryv1beta1.ProtectedDatabase) (string, bool) {
	if resource == nil {
		return "", false
	}
	raw := strings.TrimSpace(resource.Status.OsokStatus.Message)
	index := strings.LastIndex(raw, protectedDatabaseLegacyPasswordHashKey)
	if index < 0 {
		return "", false
	}
	start := index + len(protectedDatabaseLegacyPasswordHashKey)
	end := start
	for end < len(raw) && isProtectedDatabaseHexDigit(raw[end]) {
		end++
	}
	return protectedDatabasePasswordHashFromBytes([]byte(raw[start:end]))
}

func protectedDatabasePasswordHash(password string) string {
	sum := sha256.Sum256([]byte(password))
	return hex.EncodeToString(sum[:])
}

func protectedDatabasePasswordHashFromBytes(raw []byte) (string, bool) {
	hash := strings.TrimSpace(string(raw))
	if len(hash) != sha256.Size*2 {
		return "", false
	}
	if _, err := hex.DecodeString(hash); err != nil {
		return "", false
	}
	return hash, true
}

func protectedDatabasePasswordStateSecretName(resource *recoveryv1beta1.ProtectedDatabase) string {
	if resource == nil {
		return ""
	}
	return strings.TrimSpace(resource.Name)
}

func protectedDatabasePasswordStateOwnerName(resource *recoveryv1beta1.ProtectedDatabase) string {
	if resource == nil {
		return ""
	}
	return strings.TrimSpace(resource.Name)
}

func protectedDatabasePasswordStateSecretData(resource *recoveryv1beta1.ProtectedDatabase, hash string) map[string][]byte {
	return servicemanager.AddManagedSecretData(
		map[string][]byte{protectedDatabasePasswordStateHashKey: []byte(hash)},
		protectedDatabasePasswordStateSecretKind,
		protectedDatabasePasswordStateOwnerName(resource),
	)
}

func stripProtectedDatabaseLegacyPasswordHashFromMessage(raw string) string {
	raw = strings.TrimSpace(raw)
	index := strings.LastIndex(raw, protectedDatabaseLegacyPasswordHashKey)
	if index < 0 {
		return raw
	}
	prefix := strings.TrimSpace(strings.TrimRight(raw[:index], "; "))
	start := index + len(protectedDatabaseLegacyPasswordHashKey)
	end := start
	for end < len(raw) && isProtectedDatabaseHexDigit(raw[end]) {
		end++
	}
	suffix := strings.TrimSpace(strings.TrimLeft(raw[end:], "; "))
	switch {
	case prefix == "":
		return suffix
	case suffix == "":
		return prefix
	default:
		return prefix + "; " + suffix
	}
}

func isProtectedDatabaseHexDigit(value byte) bool {
	return ('0' <= value && value <= '9') ||
		('a' <= value && value <= 'f') ||
		('A' <= value && value <= 'F')
}

func conservativeProtectedDatabaseNotFoundError(err error, operation string) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	message := fmt.Sprintf("ProtectedDatabase %s returned ambiguous 404 NotAuthorizedOrNotFound: %s", strings.TrimSpace(operation), err.Error())
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		return protectedDatabaseAmbiguousNotFoundError{message: message, opcRequestID: serviceErr.GetOpcRequestID()}
	}
	return protectedDatabaseAmbiguousNotFoundError{message: message}
}

func isProtectedDatabaseAmbiguousNotFound(err error) bool {
	var ambiguous protectedDatabaseAmbiguousNotFoundError
	return errors.As(err, &ambiguous)
}

func protectedDatabaseFromResponse(response any) (recoverysdk.ProtectedDatabase, bool) {
	if current, ok := protectedDatabaseFromOperationResponse(response); ok {
		return current, true
	}
	if current, ok := protectedDatabaseFromDatabaseValue(response); ok {
		return current, true
	}
	return protectedDatabaseFromSummaryValue(response)
}

func protectedDatabaseFromOperationResponse(response any) (recoverysdk.ProtectedDatabase, bool) {
	switch current := response.(type) {
	case recoverysdk.CreateProtectedDatabaseResponse:
		return current.ProtectedDatabase, true
	case *recoverysdk.CreateProtectedDatabaseResponse:
		if current == nil {
			return recoverysdk.ProtectedDatabase{}, false
		}
		return current.ProtectedDatabase, true
	case recoverysdk.GetProtectedDatabaseResponse:
		return current.ProtectedDatabase, true
	case *recoverysdk.GetProtectedDatabaseResponse:
		if current == nil {
			return recoverysdk.ProtectedDatabase{}, false
		}
		return current.ProtectedDatabase, true
	default:
		return recoverysdk.ProtectedDatabase{}, false
	}
}

func protectedDatabaseFromDatabaseValue(response any) (recoverysdk.ProtectedDatabase, bool) {
	switch current := response.(type) {
	case recoverysdk.ProtectedDatabase:
		return current, true
	case *recoverysdk.ProtectedDatabase:
		if current == nil {
			return recoverysdk.ProtectedDatabase{}, false
		}
		return *current, true
	default:
		return recoverysdk.ProtectedDatabase{}, false
	}
}

func protectedDatabaseFromSummaryValue(response any) (recoverysdk.ProtectedDatabase, bool) {
	switch current := response.(type) {
	case recoverysdk.ProtectedDatabaseSummary:
		return protectedDatabaseFromSummary(current), true
	case *recoverysdk.ProtectedDatabaseSummary:
		if current == nil {
			return recoverysdk.ProtectedDatabase{}, false
		}
		return protectedDatabaseFromSummary(*current), true
	default:
		return recoverysdk.ProtectedDatabase{}, false
	}
}

func protectedDatabaseFromSummary(summary recoverysdk.ProtectedDatabaseSummary) recoverysdk.ProtectedDatabase {
	return recoverysdk.ProtectedDatabase{
		Id:                     summary.Id,
		CompartmentId:          summary.CompartmentId,
		DbUniqueName:           summary.DbUniqueName,
		VpcUserName:            summary.VpcUserName,
		DatabaseSize:           summary.DatabaseSize,
		ProtectionPolicyId:     summary.ProtectionPolicyId,
		RecoveryServiceSubnets: summary.RecoveryServiceSubnets,
		DisplayName:            summary.DisplayName,
		PolicyLockedDateTime:   summary.PolicyLockedDateTime,
		DatabaseId:             summary.DatabaseId,
		TimeCreated:            summary.TimeCreated,
		TimeUpdated:            summary.TimeUpdated,
		LifecycleState:         summary.LifecycleState,
		Health:                 summary.Health,
		IsReadOnlyResource:     summary.IsReadOnlyResource,
		LifecycleDetails:       summary.LifecycleDetails,
		HealthDetails:          summary.HealthDetails,
		SubscriptionId:         summary.SubscriptionId,
		FreeformTags:           summary.FreeformTags,
		DefinedTags:            summary.DefinedTags,
		SystemTags:             summary.SystemTags,
	}
}

func protectedDatabaseSubnetInputIDs(subnets []recoverysdk.RecoveryServiceSubnetInput) []string {
	result := make([]string, 0, len(subnets))
	for _, subnet := range subnets {
		result = append(result, strings.TrimSpace(stringValue(subnet.RecoveryServiceSubnetId)))
	}
	return result
}

func protectedDatabaseSubnetDetailIDs(subnets []recoverysdk.RecoveryServiceSubnetDetails) []string {
	result := make([]string, 0, len(subnets))
	for _, subnet := range subnets {
		result = append(result, strings.TrimSpace(stringValue(subnet.RecoveryServiceSubnetId)))
	}
	return result
}

func markProtectedDatabaseTerminating(resource *recoveryv1beta1.ProtectedDatabase, response any) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = protectedDatabaseDeletePendingMessage
	status.Reason = string(shared.Terminating)
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		RawStatus:       protectedDatabaseLifecycleState(response),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         protectedDatabaseDeletePendingMessage,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(
		*status,
		shared.Terminating,
		corev1.ConditionTrue,
		"",
		protectedDatabaseDeletePendingMessage,
		loggerutil.OSOKLogger{},
	)
}

func protectedDatabaseLifecycleState(response any) string {
	current, ok := protectedDatabaseFromResponse(response)
	if !ok {
		return ""
	}
	return string(current.LifecycleState)
}

func cloneProtectedDatabaseStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func protectedDatabaseDefinedTags(source map[string]shared.MapValue) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	return *util.ConvertToOciDefinedTags(&source)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

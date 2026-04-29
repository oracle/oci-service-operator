/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	bastionsdk "github.com/oracle/oci-go-sdk/v65/bastion"
	"github.com/oracle/oci-go-sdk/v65/common"
	bastionv1beta1 "github.com/oracle/oci-service-operator/api/bastion/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

const sessionKind = "Session"

var sessionWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(bastionsdk.OperationStatusAccepted),
		string(bastionsdk.OperationStatusInProgress),
		string(bastionsdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(bastionsdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(bastionsdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(bastionsdk.OperationStatusCanceled)},
	CreateActionTokens: []string{
		string(bastionsdk.OperationTypeCreateSession),
		string(bastionsdk.ActionTypeCreated),
	},
	UpdateActionTokens: []string{
		"UPDATE_SESSION",
		string(bastionsdk.ActionTypeUpdated),
	},
	DeleteActionTokens: []string{
		string(bastionsdk.OperationTypeDeleteSession),
		string(bastionsdk.ActionTypeDeleted),
	},
}

type sessionOCIClient interface {
	CreateSession(context.Context, bastionsdk.CreateSessionRequest) (bastionsdk.CreateSessionResponse, error)
	GetSession(context.Context, bastionsdk.GetSessionRequest) (bastionsdk.GetSessionResponse, error)
	ListSessions(context.Context, bastionsdk.ListSessionsRequest) (bastionsdk.ListSessionsResponse, error)
	UpdateSession(context.Context, bastionsdk.UpdateSessionRequest) (bastionsdk.UpdateSessionResponse, error)
	DeleteSession(context.Context, bastionsdk.DeleteSessionRequest) (bastionsdk.DeleteSessionResponse, error)
	GetWorkRequest(context.Context, bastionsdk.GetWorkRequestRequest) (bastionsdk.GetWorkRequestResponse, error)
}

type ambiguousSessionNotFoundError struct {
	message      string
	opcRequestID string
}

func (e ambiguousSessionNotFoundError) Error() string {
	return e.message
}

func (e ambiguousSessionNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerSessionRuntimeHooksMutator(func(manager *SessionServiceManager, hooks *SessionRuntimeHooks) {
		client, initErr := newSessionSDKClient(manager)
		applySessionRuntimeHooks(manager, hooks, client, initErr)
	})
}

func newSessionSDKClient(manager *SessionServiceManager) (sessionOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("%s service manager is nil", sessionKind)
	}
	client, err := bastionsdk.NewBastionClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applySessionRuntimeHooks(
	_ *SessionServiceManager,
	hooks *SessionRuntimeHooks,
	workRequestClient sessionOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = sessionRuntimeSemantics()
	hooks.BuildCreateBody = buildSessionCreateBody
	hooks.BuildUpdateBody = buildSessionUpdateBody
	hooks.List.Fields = sessionListFields()
	if hooks.List.Call != nil {
		list := hooks.List.Call
		hooks.List.Call = func(ctx context.Context, request bastionsdk.ListSessionsRequest) (bastionsdk.ListSessionsResponse, error) {
			return listSessionPages(ctx, list, request)
		}
	}
	hooks.Identity.GuardExistingBeforeCreate = guardSessionExistingBeforeCreate
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedSessionIdentity
	hooks.DeleteHooks.HandleError = handleSessionDeleteError
	hooks.Async.Adapter = sessionWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getSessionWorkRequest(ctx, workRequestClient, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveSessionWorkRequestAction
	hooks.Async.RecoverResourceID = recoverSessionIDFromWorkRequest
	hooks.ParityHooks.NormalizeDesiredState = normalizeSessionDesiredState
	wrapSessionRuntimeClient(hooks, workRequestClient, initErr)
}

func newSessionServiceClientWithOCIClient(log loggerutil.OSOKLogger, client sessionOCIClient) SessionServiceClient {
	manager := &SessionServiceManager{Log: log}
	hooks := newSessionRuntimeHooksWithOCIClient(client)
	applySessionRuntimeHooks(manager, &hooks, client, nil)
	delegate := defaultSessionServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*bastionv1beta1.Session](
			buildSessionGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapSessionGeneratedClient(hooks, delegate)
}

func newSessionRuntimeHooksWithOCIClient(client sessionOCIClient) SessionRuntimeHooks {
	return SessionRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*bastionv1beta1.Session]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*bastionv1beta1.Session]{},
		StatusHooks:     generatedruntime.StatusHooks[*bastionv1beta1.Session]{},
		ParityHooks:     generatedruntime.ParityHooks[*bastionv1beta1.Session]{},
		Async:           generatedruntime.AsyncHooks[*bastionv1beta1.Session]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*bastionv1beta1.Session]{},
		Create: runtimeOperationHooks[bastionsdk.CreateSessionRequest, bastionsdk.CreateSessionResponse]{
			Fields: sessionCreateFields(),
			Call: func(ctx context.Context, request bastionsdk.CreateSessionRequest) (bastionsdk.CreateSessionResponse, error) {
				return client.CreateSession(ctx, request)
			},
		},
		Get: runtimeOperationHooks[bastionsdk.GetSessionRequest, bastionsdk.GetSessionResponse]{
			Fields: sessionGetFields(),
			Call: func(ctx context.Context, request bastionsdk.GetSessionRequest) (bastionsdk.GetSessionResponse, error) {
				return client.GetSession(ctx, request)
			},
		},
		List: runtimeOperationHooks[bastionsdk.ListSessionsRequest, bastionsdk.ListSessionsResponse]{
			Fields: sessionListFields(),
			Call: func(ctx context.Context, request bastionsdk.ListSessionsRequest) (bastionsdk.ListSessionsResponse, error) {
				return client.ListSessions(ctx, request)
			},
		},
		Update: runtimeOperationHooks[bastionsdk.UpdateSessionRequest, bastionsdk.UpdateSessionResponse]{
			Fields: sessionUpdateFields(),
			Call: func(ctx context.Context, request bastionsdk.UpdateSessionRequest) (bastionsdk.UpdateSessionResponse, error) {
				return client.UpdateSession(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[bastionsdk.DeleteSessionRequest, bastionsdk.DeleteSessionResponse]{
			Fields: sessionDeleteFields(),
			Call: func(ctx context.Context, request bastionsdk.DeleteSessionRequest) (bastionsdk.DeleteSessionResponse, error) {
				return client.DeleteSession(ctx, request)
			},
		},
		WrapGeneratedClient: []func(SessionServiceClient) SessionServiceClient{},
	}
}

func sessionRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "bastion",
		FormalSlug:    "session",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "workrequest",
			Runtime:              "generatedruntime",
			FormalClassification: "workrequest",
			WorkRequest: &generatedruntime.WorkRequestSemantics{
				Source: "service-sdk",
				Phases: []string{"create", "delete"},
			},
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(bastionsdk.SessionLifecycleStateCreating)},
			ActiveStates:       []string{string(bastionsdk.SessionLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(bastionsdk.SessionLifecycleStateDeleting)},
			TerminalStates: []string{string(bastionsdk.SessionLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"bastionId", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{"displayName"},
			Mutable:         []string{"displayName"},
			ForceNew: []string{
				"bastionId",
				"targetResourceDetails",
				"keyDetails",
				"keyType",
				"sessionTtlInSeconds",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{
				{Helper: "tfresource.CreateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "session", Action: "CREATED"},
			},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
			Delete: []generatedruntime.Hook{
				{Helper: "tfresource.DeleteResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "session", Action: "DELETED"},
			},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetSession",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.CreateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "session", Action: "CREATED"},
			},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.DeleteResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "session", Action: "DELETED"},
			},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func sessionCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateSessionDetails", RequestName: "CreateSessionDetails", Contribution: "body"},
	}
}

func sessionGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "SessionId", RequestName: "sessionId", Contribution: "path", PreferResourceID: true},
	}
}

func sessionListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "BastionId", RequestName: "bastionId", Contribution: "query", LookupPaths: []string{"status.bastionId", "spec.bastionId", "bastionId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.displayName", "spec.displayName", "displayName"}},
		{FieldName: "SessionId", RequestName: "sessionId", Contribution: "query", PreferResourceID: true},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
	}
}

func sessionUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "SessionId", RequestName: "sessionId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateSessionDetails", RequestName: "UpdateSessionDetails", Contribution: "body"},
	}
}

func sessionDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "SessionId", RequestName: "sessionId", Contribution: "path", PreferResourceID: true},
	}
}

func buildSessionCreateBody(
	_ context.Context,
	resource *bastionv1beta1.Session,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("%s resource is nil", sessionKind)
	}
	if err := validateSessionSpec(resource.Spec); err != nil {
		return nil, err
	}

	targetDetails, err := sessionCreateTargetResourceDetails(resource.Spec.TargetResourceDetails)
	if err != nil {
		return nil, err
	}
	body := bastionsdk.CreateSessionDetails{
		BastionId:             common.String(strings.TrimSpace(resource.Spec.BastionId)),
		TargetResourceDetails: targetDetails,
		KeyDetails: &bastionsdk.PublicKeyDetails{
			PublicKeyContent: common.String(resource.Spec.KeyDetails.PublicKeyContent),
		},
	}
	if displayName := strings.TrimSpace(resource.Spec.DisplayName); displayName != "" {
		body.DisplayName = common.String(displayName)
	}
	if keyType := strings.ToUpper(strings.TrimSpace(resource.Spec.KeyType)); keyType != "" {
		body.KeyType = bastionsdk.CreateSessionDetailsKeyTypeEnum(keyType)
	}
	if resource.Spec.SessionTtlInSeconds > 0 {
		body.SessionTtlInSeconds = common.Int(resource.Spec.SessionTtlInSeconds)
	}
	return body, nil
}

func buildSessionUpdateBody(
	_ context.Context,
	resource *bastionv1beta1.Session,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return bastionsdk.UpdateSessionDetails{}, false, fmt.Errorf("%s resource is nil", sessionKind)
	}
	if err := validateSessionSpec(resource.Spec); err != nil {
		return bastionsdk.UpdateSessionDetails{}, false, err
	}

	current, ok := sessionFromResponse(currentResponse)
	if !ok {
		return bastionsdk.UpdateSessionDetails{}, false, fmt.Errorf("current %s response does not expose a %s body", sessionKind, sessionKind)
	}
	displayName := strings.TrimSpace(resource.Spec.DisplayName)
	if displayName == "" || stringPointerValue(current.DisplayName) == displayName {
		return bastionsdk.UpdateSessionDetails{}, false, nil
	}
	return bastionsdk.UpdateSessionDetails{DisplayName: common.String(displayName)}, true, nil
}

func validateSessionSpec(spec bastionv1beta1.SessionSpec) error {
	var missing []string
	if strings.TrimSpace(spec.BastionId) == "" {
		missing = append(missing, "bastionId")
	}
	if strings.TrimSpace(spec.KeyDetails.PublicKeyContent) == "" {
		missing = append(missing, "keyDetails.publicKeyContent")
	}
	if strings.TrimSpace(spec.TargetResourceDetails.SessionType) == "" &&
		strings.TrimSpace(spec.TargetResourceDetails.JsonData) == "" {
		missing = append(missing, "targetResourceDetails.sessionType")
	}
	if len(missing) > 0 {
		return fmt.Errorf("%s spec is missing required field(s): %s", sessionKind, strings.Join(missing, ", "))
	}
	if spec.SessionTtlInSeconds < 0 {
		return fmt.Errorf("%s spec sessionTtlInSeconds must be non-negative", sessionKind)
	}
	if keyType := strings.ToUpper(strings.TrimSpace(spec.KeyType)); keyType != "" && keyType != string(bastionsdk.CreateSessionDetailsKeyTypePub) {
		return fmt.Errorf("%s spec keyType %q is unsupported", sessionKind, spec.KeyType)
	}
	return nil
}

func sessionCreateTargetResourceDetails(spec bastionv1beta1.SessionTargetResourceDetails) (bastionsdk.CreateSessionTargetResourceDetails, error) {
	details, err := resolvedSessionTargetDetails(spec)
	if err != nil {
		return nil, err
	}
	if details.TargetResourcePort < 0 {
		return nil, fmt.Errorf("%s spec targetResourceDetails.targetResourcePort must be non-negative", sessionKind)
	}

	switch sessionType := strings.ToUpper(strings.TrimSpace(details.SessionType)); sessionType {
	case "MANAGED_SSH":
		return managedSSHSessionTargetDetails(details)
	case "PORT_FORWARDING":
		return portForwardingSessionTargetDetails(details), nil
	case "DYNAMIC_PORT_FORWARDING":
		return bastionsdk.CreateDynamicPortForwardingSessionTargetResourceDetails{}, nil
	case "":
		return nil, fmt.Errorf("%s spec targetResourceDetails.sessionType is required", sessionKind)
	default:
		return nil, fmt.Errorf("%s spec targetResourceDetails.sessionType %q is unsupported", sessionKind, details.SessionType)
	}
}

func resolvedSessionTargetDetails(spec bastionv1beta1.SessionTargetResourceDetails) (bastionv1beta1.SessionTargetResourceDetails, error) {
	raw := strings.TrimSpace(spec.JsonData)
	if raw == "" {
		return spec, nil
	}

	var decoded bastionv1beta1.SessionTargetResourceDetails
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return bastionv1beta1.SessionTargetResourceDetails{}, fmt.Errorf("decode %s targetResourceDetails.jsonData: %w", sessionKind, err)
	}
	return overlaySessionTargetDetails(decoded, spec), nil
}

func overlaySessionTargetDetails(
	base bastionv1beta1.SessionTargetResourceDetails,
	overlay bastionv1beta1.SessionTargetResourceDetails,
) bastionv1beta1.SessionTargetResourceDetails {
	if strings.TrimSpace(overlay.SessionType) != "" {
		base.SessionType = overlay.SessionType
	}
	if strings.TrimSpace(overlay.TargetResourceOperatingSystemUserName) != "" {
		base.TargetResourceOperatingSystemUserName = overlay.TargetResourceOperatingSystemUserName
	}
	if strings.TrimSpace(overlay.TargetResourceId) != "" {
		base.TargetResourceId = overlay.TargetResourceId
	}
	if strings.TrimSpace(overlay.TargetResourcePrivateIpAddress) != "" {
		base.TargetResourcePrivateIpAddress = overlay.TargetResourcePrivateIpAddress
	}
	if overlay.TargetResourcePort != 0 {
		base.TargetResourcePort = overlay.TargetResourcePort
	}
	if strings.TrimSpace(overlay.TargetResourceFqdn) != "" {
		base.TargetResourceFqdn = overlay.TargetResourceFqdn
	}
	return base
}

func managedSSHSessionTargetDetails(details bastionv1beta1.SessionTargetResourceDetails) (bastionsdk.CreateManagedSshSessionTargetResourceDetails, error) {
	var missing []string
	if strings.TrimSpace(details.TargetResourceOperatingSystemUserName) == "" {
		missing = append(missing, "targetResourceDetails.targetResourceOperatingSystemUserName")
	}
	if strings.TrimSpace(details.TargetResourceId) == "" {
		missing = append(missing, "targetResourceDetails.targetResourceId")
	}
	if len(missing) > 0 {
		return bastionsdk.CreateManagedSshSessionTargetResourceDetails{}, fmt.Errorf("%s spec is missing required field(s): %s", sessionKind, strings.Join(missing, ", "))
	}

	body := bastionsdk.CreateManagedSshSessionTargetResourceDetails{
		TargetResourceOperatingSystemUserName: common.String(strings.TrimSpace(details.TargetResourceOperatingSystemUserName)),
		TargetResourceId:                      common.String(strings.TrimSpace(details.TargetResourceId)),
	}
	if value := strings.TrimSpace(details.TargetResourcePrivateIpAddress); value != "" {
		body.TargetResourcePrivateIpAddress = common.String(value)
	}
	if details.TargetResourcePort > 0 {
		body.TargetResourcePort = common.Int(details.TargetResourcePort)
	}
	return body, nil
}

func portForwardingSessionTargetDetails(details bastionv1beta1.SessionTargetResourceDetails) bastionsdk.CreatePortForwardingSessionTargetResourceDetails {
	body := bastionsdk.CreatePortForwardingSessionTargetResourceDetails{}
	if value := strings.TrimSpace(details.TargetResourceId); value != "" {
		body.TargetResourceId = common.String(value)
	}
	if value := strings.TrimSpace(details.TargetResourcePrivateIpAddress); value != "" {
		body.TargetResourcePrivateIpAddress = common.String(value)
	}
	if value := strings.TrimSpace(details.TargetResourceFqdn); value != "" {
		body.TargetResourceFqdn = common.String(value)
	}
	if details.TargetResourcePort > 0 {
		body.TargetResourcePort = common.Int(details.TargetResourcePort)
	}
	return body
}

func normalizeSessionDesiredState(resource *bastionv1beta1.Session, _ any) {
	if resource == nil {
		return
	}
	details, ok := normalizedSessionTargetDetails(resource.Spec.TargetResourceDetails)
	if ok {
		resource.Spec.TargetResourceDetails = details
	}
}

func validateSessionTargetJSONDataCreateOnlyDrift(resource *bastionv1beta1.Session, currentResponse any) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", sessionKind)
	}
	if strings.TrimSpace(resource.Spec.TargetResourceDetails.JsonData) == "" {
		return nil
	}
	desiredDetails, err := resolvedSessionTargetDetails(resource.Spec.TargetResourceDetails)
	if err != nil {
		return err
	}
	desiredDetails.JsonData = ""
	current, ok := sessionFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("%s current response %T does not expose a %s body", sessionKind, currentResponse, sessionKind)
	}
	if current.TargetResourceDetails == nil {
		return fmt.Errorf("%s current targetResourceDetails is empty; refusing to validate targetResourceDetails.jsonData drift", sessionKind)
	}
	currentDetails, ok := sessionTargetDetailsFromSDK(current.TargetResourceDetails)
	if !ok {
		return fmt.Errorf("%s current targetResourceDetails has unexpected type %T", sessionKind, current.TargetResourceDetails)
	}
	if path := sessionTargetDetailsDriftPath(desiredDetails, currentDetails); path != "" {
		return fmt.Errorf("%s formal semantics require replacement when targetResourceDetails.jsonData changes (%s)", sessionKind, path)
	}
	return nil
}

func normalizedSessionTargetDetails(spec bastionv1beta1.SessionTargetResourceDetails) (bastionv1beta1.SessionTargetResourceDetails, bool) {
	if strings.TrimSpace(spec.JsonData) == "" {
		return spec, false
	}
	details, err := resolvedSessionTargetDetails(spec)
	if err != nil {
		return spec, false
	}
	details.JsonData = ""
	return details, true
}

func sessionTargetDetailsFromSDK(details bastionsdk.TargetResourceDetails) (bastionv1beta1.SessionTargetResourceDetails, bool) {
	if managed, ok := managedSSHSessionTargetDetailsFromSDK(details); ok {
		return managed, true
	}
	if portForwarding, ok := portForwardingSessionTargetDetailsFromSDK(details); ok {
		return portForwarding, true
	}
	if _, ok := details.(bastionsdk.DynamicPortForwardingSessionTargetResourceDetails); ok {
		return bastionv1beta1.SessionTargetResourceDetails{SessionType: "DYNAMIC_PORT_FORWARDING"}, true
	}
	if value, ok := details.(*bastionsdk.DynamicPortForwardingSessionTargetResourceDetails); ok && value != nil {
		return bastionv1beta1.SessionTargetResourceDetails{SessionType: "DYNAMIC_PORT_FORWARDING"}, true
	}
	return bastionv1beta1.SessionTargetResourceDetails{}, false
}

func managedSSHSessionTargetDetailsFromSDK(details bastionsdk.TargetResourceDetails) (bastionv1beta1.SessionTargetResourceDetails, bool) {
	switch value := details.(type) {
	case bastionsdk.ManagedSshSessionTargetResourceDetails:
		return managedSSHSessionTargetDetailsFromValue(value), true
	case *bastionsdk.ManagedSshSessionTargetResourceDetails:
		if value != nil {
			return managedSSHSessionTargetDetailsFromValue(*value), true
		}
	}
	return bastionv1beta1.SessionTargetResourceDetails{}, false
}

func managedSSHSessionTargetDetailsFromValue(details bastionsdk.ManagedSshSessionTargetResourceDetails) bastionv1beta1.SessionTargetResourceDetails {
	return bastionv1beta1.SessionTargetResourceDetails{
		SessionType:                           "MANAGED_SSH",
		TargetResourceOperatingSystemUserName: stringPointerValue(details.TargetResourceOperatingSystemUserName),
		TargetResourceId:                      stringPointerValue(details.TargetResourceId),
		TargetResourcePrivateIpAddress:        stringPointerValue(details.TargetResourcePrivateIpAddress),
		TargetResourcePort:                    intPointerValue(details.TargetResourcePort),
	}
}

func portForwardingSessionTargetDetailsFromSDK(details bastionsdk.TargetResourceDetails) (bastionv1beta1.SessionTargetResourceDetails, bool) {
	switch value := details.(type) {
	case bastionsdk.PortForwardingSessionTargetResourceDetails:
		return portForwardingSessionTargetDetailsFromValue(value), true
	case *bastionsdk.PortForwardingSessionTargetResourceDetails:
		if value != nil {
			return portForwardingSessionTargetDetailsFromValue(*value), true
		}
	}
	return bastionv1beta1.SessionTargetResourceDetails{}, false
}

func portForwardingSessionTargetDetailsFromValue(details bastionsdk.PortForwardingSessionTargetResourceDetails) bastionv1beta1.SessionTargetResourceDetails {
	return bastionv1beta1.SessionTargetResourceDetails{
		SessionType:                    "PORT_FORWARDING",
		TargetResourceId:               stringPointerValue(details.TargetResourceId),
		TargetResourcePrivateIpAddress: stringPointerValue(details.TargetResourcePrivateIpAddress),
		TargetResourcePort:             intPointerValue(details.TargetResourcePort),
		TargetResourceFqdn:             stringPointerValue(details.TargetResourceFqdn),
	}
}

type sessionTargetDetailDriftCheck struct {
	path    string
	desired string
	current string
}

func sessionTargetDetailsDriftPath(desired bastionv1beta1.SessionTargetResourceDetails, current bastionv1beta1.SessionTargetResourceDetails) string {
	for _, check := range sessionTargetDetailDriftChecks(desired, current) {
		if check.desired != "" && check.desired != check.current {
			return check.path
		}
	}
	return ""
}

func sessionTargetDetailDriftChecks(
	desired bastionv1beta1.SessionTargetResourceDetails,
	current bastionv1beta1.SessionTargetResourceDetails,
) []sessionTargetDetailDriftCheck {
	checks := []sessionTargetDetailDriftCheck{
		{
			path:    "targetResourceDetails.sessionType",
			desired: strings.ToUpper(strings.TrimSpace(desired.SessionType)),
			current: strings.ToUpper(strings.TrimSpace(current.SessionType)),
		},
		{
			path:    "targetResourceDetails.targetResourceOperatingSystemUserName",
			desired: strings.TrimSpace(desired.TargetResourceOperatingSystemUserName),
			current: strings.TrimSpace(current.TargetResourceOperatingSystemUserName),
		},
		{
			path:    "targetResourceDetails.targetResourceId",
			desired: strings.TrimSpace(desired.TargetResourceId),
			current: strings.TrimSpace(current.TargetResourceId),
		},
		{
			path:    "targetResourceDetails.targetResourcePrivateIpAddress",
			desired: strings.TrimSpace(desired.TargetResourcePrivateIpAddress),
			current: strings.TrimSpace(current.TargetResourcePrivateIpAddress),
		},
		{
			path:    "targetResourceDetails.targetResourceFqdn",
			desired: strings.TrimSpace(desired.TargetResourceFqdn),
			current: strings.TrimSpace(current.TargetResourceFqdn),
		},
	}
	if desired.TargetResourcePort != 0 {
		checks = append(checks, sessionTargetDetailDriftCheck{
			path:    "targetResourceDetails.targetResourcePort",
			desired: fmt.Sprint(desired.TargetResourcePort),
			current: fmt.Sprint(current.TargetResourcePort),
		})
	}
	return checks
}

func listSessionPages(
	ctx context.Context,
	call func(context.Context, bastionsdk.ListSessionsRequest) (bastionsdk.ListSessionsResponse, error),
	request bastionsdk.ListSessionsRequest,
) (bastionsdk.ListSessionsResponse, error) {
	var combined bastionsdk.ListSessionsResponse
	for {
		response, err := call(ctx, request)
		if err != nil {
			return response, err
		}
		if combined.OpcRequestId == nil {
			combined.OpcRequestId = response.OpcRequestId
		}
		combined.RawResponse = response.RawResponse
		combined.Items = append(combined.Items, response.Items...)
		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			combined.OpcNextPage = nil
			return combined, nil
		}
		request.Page = response.OpcNextPage
	}
}

func guardSessionExistingBeforeCreate(
	_ context.Context,
	resource *bastionv1beta1.Session,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("%s resource is nil", sessionKind)
	}
	if strings.TrimSpace(resource.Spec.BastionId) == "" || strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func clearTrackedSessionIdentity(resource *bastionv1beta1.Session) {
	if resource == nil {
		return
	}
	resource.Status.Id = ""
	resource.Status.OsokStatus.Ocid = shared.OCID("")
}

func handleSessionDeleteError(resource *bastionv1beta1.Session, err error) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return ambiguousSessionError("delete", err)
	}
	return err
}

func wrapSessionRuntimeClient(hooks *SessionRuntimeHooks, workRequestClient sessionOCIClient, initErr error) {
	if hooks.Get.Call == nil {
		return
	}
	getSession := hooks.Get.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate SessionServiceClient) SessionServiceClient {
		return sessionRuntimeClient{
			delegate:          delegate,
			getSession:        getSession,
			workRequestClient: workRequestClient,
			initErr:           initErr,
		}
	})
}

type sessionRuntimeClient struct {
	delegate          SessionServiceClient
	getSession        func(context.Context, bastionsdk.GetSessionRequest) (bastionsdk.GetSessionResponse, error)
	workRequestClient sessionOCIClient
	initErr           error
}

func (c sessionRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *bastionv1beta1.Session,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if err := c.validateTrackedTargetJSONData(ctx, resource); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c sessionRuntimeClient) Delete(ctx context.Context, resource *bastionv1beta1.Session) (bool, error) {
	if handled, err := c.observePendingWriteWorkRequest(ctx, resource); handled || err != nil {
		return false, err
	}
	if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c sessionRuntimeClient) observePendingWriteWorkRequest(ctx context.Context, resource *bastionv1beta1.Session) (bool, error) {
	workRequestID, phase, ok := pendingSessionWriteWorkRequest(resource)
	if !ok {
		return false, nil
	}
	workRequest, err := getSessionWorkRequest(ctx, c.workRequestClient, c.initErr, workRequestID)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return true, err
	}
	current, err := sessionWorkRequestAsyncOperation(&resource.Status.OsokStatus, workRequest, phase)
	if err != nil {
		return true, err
	}
	_ = servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, loggerutil.OSOKLogger{})
	return true, nil
}

func (c sessionRuntimeClient) validateTrackedTargetJSONData(ctx context.Context, resource *bastionv1beta1.Session) error {
	if c.getSession == nil || resource == nil || strings.TrimSpace(resource.Spec.TargetResourceDetails.JsonData) == "" {
		return nil
	}
	sessionID := trackedSessionID(resource)
	if sessionID == "" {
		return nil
	}
	response, err := c.getSession(ctx, bastionsdk.GetSessionRequest{SessionId: common.String(sessionID)})
	if err != nil {
		return err
	}
	return validateSessionTargetJSONDataCreateOnlyDrift(resource, response)
}

func (c sessionRuntimeClient) rejectAuthShapedConfirmRead(ctx context.Context, resource *bastionv1beta1.Session) error {
	if c.getSession == nil || resource == nil {
		return nil
	}
	sessionID := trackedSessionID(resource)
	if sessionID == "" {
		return nil
	}
	_, err := c.getSession(ctx, bastionsdk.GetSessionRequest{SessionId: common.String(sessionID)})
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return ambiguousSessionError("delete confirmation", err)
}

func trackedSessionID(resource *bastionv1beta1.Session) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func pendingSessionWriteWorkRequest(resource *bastionv1beta1.Session) (string, shared.OSOKAsyncPhase, bool) {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return "", "", false
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != "" && current.Source != shared.OSOKAsyncSourceWorkRequest {
		return "", "", false
	}
	if current.NormalizedClass != shared.OSOKAsyncClassPending {
		return "", "", false
	}
	workRequestID := strings.TrimSpace(current.WorkRequestID)
	if workRequestID == "" {
		return "", "", false
	}
	switch current.Phase {
	case shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncPhaseUpdate:
		return workRequestID, current.Phase, true
	default:
		return "", "", false
	}
}

func sessionWorkRequestAsyncOperation(
	status *shared.OSOKStatus,
	workRequest any,
	fallbackPhase shared.OSOKAsyncPhase,
) (*shared.OSOKAsyncOperation, error) {
	wr, ok := sessionWorkRequestFromAny(workRequest)
	if !ok {
		return nil, fmt.Errorf("%s work request has unexpected type %T", sessionKind, workRequest)
	}
	return servicemanager.BuildWorkRequestAsyncOperation(status, sessionWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(wr.Status),
		RawAction:        string(wr.OperationType),
		RawOperationType: string(wr.OperationType),
		WorkRequestID:    stringPointerValue(wr.Id),
		PercentComplete:  wr.PercentComplete,
		FallbackPhase:    fallbackPhase,
	})
}

func ambiguousSessionError(operation string, err error) error {
	return ambiguousSessionNotFoundError{
		message:      fmt.Sprintf("%s %s returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", sessionKind, operation, err),
		opcRequestID: errorutil.OpcRequestID(err),
	}
}

func getSessionWorkRequest(
	ctx context.Context,
	client sessionOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize %s OCI client: %w", sessionKind, initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("%s OCI client is not configured", sessionKind)
	}
	response, err := client.GetWorkRequest(ctx, bastionsdk.GetWorkRequestRequest{WorkRequestId: common.String(strings.TrimSpace(workRequestID))})
	if err != nil {
		return nil, err
	}
	return response, nil
}

func resolveSessionWorkRequestAction(workRequest any) (string, error) {
	wr, ok := sessionWorkRequestFromAny(workRequest)
	if !ok {
		return "", fmt.Errorf("%s work request has unexpected type %T", sessionKind, workRequest)
	}
	return string(wr.OperationType), nil
}

func recoverSessionIDFromWorkRequest(
	_ *bastionv1beta1.Session,
	workRequest any,
	_ shared.OSOKAsyncPhase,
) (string, error) {
	wr, ok := sessionWorkRequestFromAny(workRequest)
	if !ok {
		return "", fmt.Errorf("%s work request has unexpected type %T", sessionKind, workRequest)
	}
	for _, resource := range wr.Resources {
		if id := stringPointerValue(resource.Identifier); id != "" {
			return id, nil
		}
	}
	return "", nil
}

func sessionWorkRequestFromAny(workRequest any) (bastionsdk.WorkRequest, bool) {
	switch value := workRequest.(type) {
	case bastionsdk.GetWorkRequestResponse:
		return value.WorkRequest, true
	case *bastionsdk.GetWorkRequestResponse:
		if value == nil {
			return bastionsdk.WorkRequest{}, false
		}
		return value.WorkRequest, true
	case bastionsdk.WorkRequest:
		return value, true
	case *bastionsdk.WorkRequest:
		if value == nil {
			return bastionsdk.WorkRequest{}, false
		}
		return *value, true
	default:
		return bastionsdk.WorkRequest{}, false
	}
}

func sessionFromResponse(response any) (bastionsdk.Session, bool) {
	if session, ok := sessionFromReadOrCreateResponse(response); ok {
		return session, true
	}
	if session, ok := sessionFromUpdateOrBody(response); ok {
		return session, true
	}
	return bastionsdk.Session{}, false
}

func sessionFromReadOrCreateResponse(response any) (bastionsdk.Session, bool) {
	switch value := response.(type) {
	case bastionsdk.GetSessionResponse:
		return value.Session, true
	case *bastionsdk.GetSessionResponse:
		if value == nil {
			return bastionsdk.Session{}, false
		}
		return value.Session, true
	case bastionsdk.CreateSessionResponse:
		return value.Session, true
	case *bastionsdk.CreateSessionResponse:
		if value == nil {
			return bastionsdk.Session{}, false
		}
		return value.Session, true
	default:
		return bastionsdk.Session{}, false
	}
}

func sessionFromUpdateOrBody(response any) (bastionsdk.Session, bool) {
	switch value := response.(type) {
	case bastionsdk.UpdateSessionResponse:
		return value.Session, true
	case *bastionsdk.UpdateSessionResponse:
		if value == nil {
			return bastionsdk.Session{}, false
		}
		return value.Session, true
	case bastionsdk.Session:
		return value, true
	case *bastionsdk.Session:
		if value == nil {
			return bastionsdk.Session{}, false
		}
		return *value, true
	default:
		return bastionsdk.Session{}, false
	}
}

func stringPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func intPointerValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

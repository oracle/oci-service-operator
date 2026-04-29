/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package authenticationprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	odasdk "github.com/oracle/oci-go-sdk/v65/oda"
	odav1beta1 "github.com/oracle/oci-service-operator/api/oda/v1beta1"
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

const authenticationProviderOdaInstanceIDAnnotation = "oda.oracle.com/oda-instance-id"

type authenticationProviderOCIClient interface {
	CreateAuthenticationProvider(context.Context, odasdk.CreateAuthenticationProviderRequest) (odasdk.CreateAuthenticationProviderResponse, error)
	GetAuthenticationProvider(context.Context, odasdk.GetAuthenticationProviderRequest) (odasdk.GetAuthenticationProviderResponse, error)
	ListAuthenticationProviders(context.Context, odasdk.ListAuthenticationProvidersRequest) (odasdk.ListAuthenticationProvidersResponse, error)
	UpdateAuthenticationProvider(context.Context, odasdk.UpdateAuthenticationProviderRequest) (odasdk.UpdateAuthenticationProviderResponse, error)
	DeleteAuthenticationProvider(context.Context, odasdk.DeleteAuthenticationProviderRequest) (odasdk.DeleteAuthenticationProviderResponse, error)
}

type authenticationProviderRuntimeClient struct {
	delegate AuthenticationProviderServiceClient
	hooks    AuthenticationProviderRuntimeHooks
	log      loggerutil.OSOKLogger
}

func init() {
	registerAuthenticationProviderRuntimeHooksMutator(func(manager *AuthenticationProviderServiceManager, hooks *AuthenticationProviderRuntimeHooks) {
		applyAuthenticationProviderRuntimeHooks(manager, hooks)
	})
}

func applyAuthenticationProviderRuntimeHooks(manager *AuthenticationProviderServiceManager, hooks *AuthenticationProviderRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newAuthenticationProviderRuntimeSemantics()
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate AuthenticationProviderServiceClient) AuthenticationProviderServiceClient {
		runtimeClient := &authenticationProviderRuntimeClient{
			delegate: delegate,
			hooks:    *hooks,
		}
		if manager != nil {
			runtimeClient.log = manager.Log
		}
		return runtimeClient
	})
}

func newAuthenticationProviderServiceClientWithOCIClient(log loggerutil.OSOKLogger, client authenticationProviderOCIClient) AuthenticationProviderServiceClient {
	hooks := newAuthenticationProviderRuntimeHooksWithOCIClient(client)
	applyAuthenticationProviderRuntimeHooks(&AuthenticationProviderServiceManager{Log: log}, &hooks)
	delegate := defaultAuthenticationProviderServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*odav1beta1.AuthenticationProvider](
			buildAuthenticationProviderGeneratedRuntimeConfig(&AuthenticationProviderServiceManager{Log: log}, hooks),
		),
	}
	return wrapAuthenticationProviderGeneratedClient(hooks, delegate)
}

func newAuthenticationProviderRuntimeHooksWithOCIClient(client authenticationProviderOCIClient) AuthenticationProviderRuntimeHooks {
	return AuthenticationProviderRuntimeHooks{
		Create: runtimeOperationHooks[odasdk.CreateAuthenticationProviderRequest, odasdk.CreateAuthenticationProviderResponse]{
			Call: func(ctx context.Context, request odasdk.CreateAuthenticationProviderRequest) (odasdk.CreateAuthenticationProviderResponse, error) {
				return client.CreateAuthenticationProvider(ctx, request)
			},
		},
		Get: runtimeOperationHooks[odasdk.GetAuthenticationProviderRequest, odasdk.GetAuthenticationProviderResponse]{
			Call: func(ctx context.Context, request odasdk.GetAuthenticationProviderRequest) (odasdk.GetAuthenticationProviderResponse, error) {
				return client.GetAuthenticationProvider(ctx, request)
			},
		},
		List: runtimeOperationHooks[odasdk.ListAuthenticationProvidersRequest, odasdk.ListAuthenticationProvidersResponse]{
			Call: func(ctx context.Context, request odasdk.ListAuthenticationProvidersRequest) (odasdk.ListAuthenticationProvidersResponse, error) {
				return client.ListAuthenticationProviders(ctx, request)
			},
		},
		Update: runtimeOperationHooks[odasdk.UpdateAuthenticationProviderRequest, odasdk.UpdateAuthenticationProviderResponse]{
			Call: func(ctx context.Context, request odasdk.UpdateAuthenticationProviderRequest) (odasdk.UpdateAuthenticationProviderResponse, error) {
				return client.UpdateAuthenticationProvider(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[odasdk.DeleteAuthenticationProviderRequest, odasdk.DeleteAuthenticationProviderResponse]{
			Call: func(ctx context.Context, request odasdk.DeleteAuthenticationProviderRequest) (odasdk.DeleteAuthenticationProviderResponse, error) {
				return client.DeleteAuthenticationProvider(ctx, request)
			},
		},
	}
}

func newAuthenticationProviderRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "oda",
		FormalSlug:    "authenticationprovider",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "handwritten",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{"CREATING"},
			UpdatingStates:     []string{"UPDATING"},
			ActiveStates:       []string{"ACTIVE"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{"DELETING"},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"odaInstanceId", "identityProvider", "name"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"authorizationEndpointUrl",
				"clientId",
				"definedTags",
				"freeformTags",
				"redirectUrl",
				"refreshTokenRetentionPeriodInDays",
				"revokeTokenEndpointUrl",
				"scopes",
				"shortAuthorizationCodeRequestUrl",
				"subjectClaim",
				"tokenEndpointUrl",
			},
			ForceNew:      []string{"grantType", "identityProvider", "name", "isVisible"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "resource-local", EntityType: "AuthenticationProvider", Action: "CreateAuthenticationProvider"}},
			Update: []generatedruntime.Hook{{Helper: "resource-local", EntityType: "AuthenticationProvider", Action: "UpdateAuthenticationProvider"}},
			Delete: []generatedruntime.Hook{{Helper: "resource-local", EntityType: "AuthenticationProvider", Action: "DeleteAuthenticationProvider"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "resource-local", EntityType: "AuthenticationProvider", Action: "GetAuthenticationProvider"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "resource-local", EntityType: "AuthenticationProvider", Action: "GetAuthenticationProvider"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "resource-local", EntityType: "AuthenticationProvider", Action: "GetAuthenticationProvider"}},
		},
		Unsupported: []generatedruntime.UnsupportedSemantic{
			{
				Category:      "clientSecret-update-drift",
				StopCondition: "clientSecret is write-only in OCI responses; update drift cannot be detected until the API exposes a last-applied secret hash or secret reference status.",
			},
		},
	}
}

func (c *authenticationProviderRuntimeClient) CreateOrUpdate(ctx context.Context, resource *odav1beta1.AuthenticationProvider, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("AuthenticationProvider resource is nil")
	}
	if c == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("AuthenticationProvider runtime client is not configured")
	}

	odaInstanceID, err := authenticationProviderOdaInstanceID(resource)
	if err != nil {
		return c.fail(resource, err)
	}

	current, found, err := c.resolveCurrent(ctx, resource, odaInstanceID)
	if err != nil {
		return c.fail(resource, err)
	}
	if !found {
		return c.create(ctx, resource, odaInstanceID)
	}

	if isAuthenticationProviderRetryableLifecycle(current.LifecycleState) {
		return c.finishWithLifecycle(resource, current), nil
	}
	if state := normalizedAuthenticationProviderLifecycle(current.LifecycleState); state != "" && state != string(odasdk.LifecycleStateActive) {
		return c.finishWithLifecycle(resource, current), nil
	}
	if err := validateAuthenticationProviderCreateOnlyDrift(resource, current); err != nil {
		return c.fail(resource, err)
	}

	updateDetails, updateNeeded := buildAuthenticationProviderUpdateDetails(resource, current)
	if updateNeeded {
		return c.update(ctx, resource, odaInstanceID, authenticationProviderIDFromSDK(current), updateDetails)
	}

	return c.finishWithLifecycle(resource, current), nil
}

func (c *authenticationProviderRuntimeClient) Delete(ctx context.Context, resource *odav1beta1.AuthenticationProvider) (bool, error) {
	if resource == nil {
		return false, fmt.Errorf("AuthenticationProvider resource is nil")
	}
	if c == nil {
		return false, fmt.Errorf("AuthenticationProvider runtime client is not configured")
	}

	odaInstanceID, parentErr := authenticationProviderOdaInstanceID(resource)
	currentID := currentAuthenticationProviderID(resource)
	if parentErr != nil {
		if currentID == "" {
			c.markDeleted(resource, "AuthenticationProvider has no tracked OCI identity")
			return true, nil
		}
		c.markFailure(resource, parentErr)
		return false, parentErr
	}

	current, found, err := c.resolveCurrent(ctx, resource, odaInstanceID)
	if err != nil {
		return false, c.markFailure(resource, err)
	}
	if !found {
		c.markDeleted(resource, "OCI AuthenticationProvider is already deleted")
		return true, nil
	}

	currentID = authenticationProviderIDFromSDK(current)
	if currentID == "" {
		err := fmt.Errorf("AuthenticationProvider delete could not resolve OCI resource ID")
		return false, c.markFailure(resource, err)
	}

	switch normalizedAuthenticationProviderLifecycle(current.LifecycleState) {
	case string(odasdk.LifecycleStateDeleting):
		c.projectAuthenticationProviderStatus(resource, current)
		c.markCondition(resource, shared.Terminating, string(current.LifecycleState), "AuthenticationProvider delete is in progress", true)
		return false, nil
	case string(odasdk.LifecycleStateDeleted):
		c.markDeleted(resource, "OCI AuthenticationProvider deleted")
		return true, nil
	}

	response, err := c.hooks.Delete.Call(ctx, odasdk.DeleteAuthenticationProviderRequest{
		OdaInstanceId:            common.String(odaInstanceID),
		AuthenticationProviderId: common.String(currentID),
	})
	if err != nil {
		if isAuthenticationProviderReadNotFound(err) {
			c.markDeleted(resource, "OCI AuthenticationProvider is already deleted")
			return true, nil
		}
		return false, c.markFailure(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)

	confirm, err := c.read(ctx, odaInstanceID, currentID)
	if err != nil {
		if isAuthenticationProviderReadNotFound(err) {
			c.markDeleted(resource, "OCI AuthenticationProvider deleted")
			return true, nil
		}
		return false, c.markFailure(resource, err)
	}
	if normalizedAuthenticationProviderLifecycle(confirm.LifecycleState) == string(odasdk.LifecycleStateDeleted) {
		c.markDeleted(resource, "OCI AuthenticationProvider deleted")
		return true, nil
	}

	c.projectAuthenticationProviderStatus(resource, confirm)
	c.markCondition(resource, shared.Terminating, string(confirm.LifecycleState), "AuthenticationProvider delete is in progress", true)
	return false, nil
}

func (c *authenticationProviderRuntimeClient) create(ctx context.Context, resource *odav1beta1.AuthenticationProvider, odaInstanceID string) (servicemanager.OSOKResponse, error) {
	response, err := c.hooks.Create.Call(ctx, odasdk.CreateAuthenticationProviderRequest{
		OdaInstanceId:                       common.String(odaInstanceID),
		CreateAuthenticationProviderDetails: buildAuthenticationProviderCreateDetails(resource.Spec),
	})
	if err != nil {
		return c.fail(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)

	current := response.AuthenticationProvider
	currentID := authenticationProviderIDFromSDK(current)
	if currentID == "" {
		return c.fail(resource, fmt.Errorf("AuthenticationProvider create response did not include an OCI resource ID"))
	}

	followUp, err := c.read(ctx, odaInstanceID, currentID)
	if err != nil {
		return c.fail(resource, err)
	}
	return c.finishWithLifecycle(resource, followUp), nil
}

func (c *authenticationProviderRuntimeClient) update(
	ctx context.Context,
	resource *odav1beta1.AuthenticationProvider,
	odaInstanceID string,
	authenticationProviderID string,
	details odasdk.UpdateAuthenticationProviderDetails,
) (servicemanager.OSOKResponse, error) {
	if authenticationProviderID == "" {
		return c.fail(resource, fmt.Errorf("AuthenticationProvider update could not resolve OCI resource ID"))
	}
	response, err := c.hooks.Update.Call(ctx, odasdk.UpdateAuthenticationProviderRequest{
		OdaInstanceId:                       common.String(odaInstanceID),
		AuthenticationProviderId:            common.String(authenticationProviderID),
		UpdateAuthenticationProviderDetails: details,
	})
	if err != nil {
		return c.fail(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)

	followUp, err := c.read(ctx, odaInstanceID, authenticationProviderID)
	if err != nil {
		return c.fail(resource, err)
	}
	return c.finishWithLifecycle(resource, followUp), nil
}

func (c *authenticationProviderRuntimeClient) resolveCurrent(
	ctx context.Context,
	resource *odav1beta1.AuthenticationProvider,
	odaInstanceID string,
) (odasdk.AuthenticationProvider, bool, error) {
	if currentID := currentAuthenticationProviderID(resource); currentID != "" {
		current, err := c.read(ctx, odaInstanceID, currentID)
		if err != nil {
			if isAuthenticationProviderReadNotFound(err) {
				return odasdk.AuthenticationProvider{}, false, nil
			}
			return odasdk.AuthenticationProvider{}, false, err
		}
		return current, true, nil
	}
	return c.resolveByList(ctx, resource, odaInstanceID)
}

func (c *authenticationProviderRuntimeClient) resolveByList(
	ctx context.Context,
	resource *odav1beta1.AuthenticationProvider,
	odaInstanceID string,
) (odasdk.AuthenticationProvider, bool, error) {
	response, err := c.hooks.List.Call(ctx, odasdk.ListAuthenticationProvidersRequest{
		OdaInstanceId:    common.String(odaInstanceID),
		IdentityProvider: odasdk.ListAuthenticationProvidersIdentityProviderEnum(resource.Spec.IdentityProvider),
		Name:             common.String(resource.Spec.Name),
	})
	if err != nil {
		return odasdk.AuthenticationProvider{}, false, err
	}

	var matchedID string
	for _, item := range response.Items {
		if !authenticationProviderSummaryMatches(resource, item) {
			continue
		}
		if normalizedAuthenticationProviderLifecycle(item.LifecycleState) == string(odasdk.LifecycleStateDeleted) {
			continue
		}
		itemID := stringValue(item.Id)
		if itemID == "" {
			continue
		}
		if matchedID != "" && matchedID != itemID {
			return odasdk.AuthenticationProvider{}, false, fmt.Errorf("multiple OCI AuthenticationProviders matched name %q and identityProvider %q", resource.Spec.Name, resource.Spec.IdentityProvider)
		}
		matchedID = itemID
	}
	if matchedID == "" {
		return odasdk.AuthenticationProvider{}, false, nil
	}

	current, err := c.read(ctx, odaInstanceID, matchedID)
	if err != nil {
		if isAuthenticationProviderReadNotFound(err) {
			return odasdk.AuthenticationProvider{}, false, nil
		}
		return odasdk.AuthenticationProvider{}, false, err
	}
	return current, true, nil
}

func (c *authenticationProviderRuntimeClient) read(ctx context.Context, odaInstanceID string, authenticationProviderID string) (odasdk.AuthenticationProvider, error) {
	response, err := c.hooks.Get.Call(ctx, odasdk.GetAuthenticationProviderRequest{
		OdaInstanceId:            common.String(odaInstanceID),
		AuthenticationProviderId: common.String(authenticationProviderID),
	})
	if err != nil {
		return odasdk.AuthenticationProvider{}, err
	}
	return response.AuthenticationProvider, nil
}

func (c *authenticationProviderRuntimeClient) finishWithLifecycle(resource *odav1beta1.AuthenticationProvider, current odasdk.AuthenticationProvider) servicemanager.OSOKResponse {
	c.projectAuthenticationProviderStatus(resource, current)

	state := normalizedAuthenticationProviderLifecycle(current.LifecycleState)
	message := authenticationProviderLifecycleMessage(current, "AuthenticationProvider lifecycle state "+state)
	switch state {
	case string(odasdk.LifecycleStateCreating):
		return c.markCondition(resource, shared.Provisioning, state, message, true)
	case string(odasdk.LifecycleStateUpdating):
		return c.markCondition(resource, shared.Updating, state, message, true)
	case string(odasdk.LifecycleStateDeleting):
		return c.markCondition(resource, shared.Terminating, state, message, true)
	case string(odasdk.LifecycleStateDeleted):
		return c.markCondition(resource, shared.Terminating, state, message, false)
	case string(odasdk.LifecycleStateActive):
		return c.markCondition(resource, shared.Active, state, message, false)
	default:
		return c.markCondition(resource, shared.Failed, state, fmt.Sprintf("formal lifecycle state %q is not modeled: %s", state, message), false)
	}
}

func (c *authenticationProviderRuntimeClient) projectAuthenticationProviderStatus(resource *odav1beta1.AuthenticationProvider, current odasdk.AuthenticationProvider) {
	status := &resource.Status
	status.Id = stringValue(current.Id)
	status.GrantType = string(current.GrantType)
	status.IdentityProvider = string(current.IdentityProvider)
	status.Name = stringValue(current.Name)
	status.TokenEndpointUrl = stringValue(current.TokenEndpointUrl)
	status.ClientId = stringValue(current.ClientId)
	status.Scopes = stringValue(current.Scopes)
	status.IsVisible = boolValue(current.IsVisible)
	status.LifecycleState = string(current.LifecycleState)
	status.TimeCreated = authenticationProviderSDKTimeString(current.TimeCreated)
	status.TimeUpdated = authenticationProviderSDKTimeString(current.TimeUpdated)
	status.AuthorizationEndpointUrl = stringValue(current.AuthorizationEndpointUrl)
	status.ShortAuthorizationCodeRequestUrl = stringValue(current.ShortAuthorizationCodeRequestUrl)
	status.RevokeTokenEndpointUrl = stringValue(current.RevokeTokenEndpointUrl)
	status.SubjectClaim = stringValue(current.SubjectClaim)
	status.RefreshTokenRetentionPeriodInDays = intValue(current.RefreshTokenRetentionPeriodInDays)
	status.RedirectUrl = stringValue(current.RedirectUrl)
	status.FreeformTags = cloneStringMap(current.FreeformTags)
	status.DefinedTags = authenticationProviderStatusDefinedTags(current.DefinedTags)

	now := metav1.Now()
	if status.Id != "" {
		status.OsokStatus.Ocid = shared.OCID(status.Id)
		if status.OsokStatus.CreatedAt == nil {
			status.OsokStatus.CreatedAt = &now
		}
	}
}

func (c *authenticationProviderRuntimeClient) fail(resource *odav1beta1.AuthenticationProvider, err error) (servicemanager.OSOKResponse, error) {
	return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
}

func (c *authenticationProviderRuntimeClient) markFailure(resource *odav1beta1.AuthenticationProvider, err error) error {
	if err == nil {
		return nil
	}
	status := &resource.Status.OsokStatus
	servicemanager.RecordErrorOpcRequestID(status, err)
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	now := metav1.Now()
	status.UpdatedAt = &now
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		NormalizedClass: shared.OSOKAsyncClassFailed,
		Message:         err.Error(),
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, v1.ConditionFalse, "", err.Error(), c.log)
	return err
}

func (c *authenticationProviderRuntimeClient) markDeleted(resource *odav1beta1.AuthenticationProvider, message string) {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	status.Async.Current = nil
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, c.log)
}

func (c *authenticationProviderRuntimeClient) markCondition(
	resource *odav1beta1.AuthenticationProvider,
	condition shared.OSOKConditionType,
	rawState string,
	message string,
	shouldRequeue bool,
) servicemanager.OSOKResponse {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.Message = message
	status.Reason = string(condition)
	status.UpdatedAt = &now

	if condition == shared.Provisioning || condition == shared.Updating || (condition == shared.Terminating && shouldRequeue) {
		status.Async.Current = &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           asyncPhaseForAuthenticationProviderCondition(condition),
			RawStatus:       rawState,
			NormalizedClass: shared.OSOKAsyncClassPending,
			Message:         message,
			UpdatedAt:       &now,
		}
	} else if condition == shared.Failed {
		status.Async.Current = &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			RawStatus:       rawState,
			NormalizedClass: shared.OSOKAsyncClassUnknown,
			Message:         message,
			UpdatedAt:       &now,
		}
	} else {
		status.Async.Current = nil
	}

	conditionStatus := v1.ConditionTrue
	if condition == shared.Failed {
		conditionStatus = v1.ConditionFalse
	}
	*status = util.UpdateOSOKStatusCondition(*status, condition, conditionStatus, "", message, c.log)
	return servicemanager.OSOKResponse{
		IsSuccessful:    condition != shared.Failed,
		ShouldRequeue:   shouldRequeue,
		RequeueDuration: time.Minute,
	}
}

func authenticationProviderOdaInstanceID(resource *odav1beta1.AuthenticationProvider) (string, error) {
	if resource == nil {
		return "", fmt.Errorf("AuthenticationProvider resource is nil")
	}
	odaInstanceID := strings.TrimSpace(resource.Annotations[authenticationProviderOdaInstanceIDAnnotation])
	if odaInstanceID == "" {
		return "", fmt.Errorf("AuthenticationProvider requires metadata annotation %q with the parent ODA instance OCID", authenticationProviderOdaInstanceIDAnnotation)
	}
	return odaInstanceID, nil
}

func currentAuthenticationProviderID(resource *odav1beta1.AuthenticationProvider) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); id != "" {
		return id
	}
	return strings.TrimSpace(resource.Status.Id)
}

func buildAuthenticationProviderCreateDetails(spec odav1beta1.AuthenticationProviderSpec) odasdk.CreateAuthenticationProviderDetails {
	details := odasdk.CreateAuthenticationProviderDetails{
		GrantType:        odasdk.AuthenticationGrantTypeEnum(spec.GrantType),
		IdentityProvider: odasdk.AuthenticationIdentityProviderEnum(spec.IdentityProvider),
		Name:             common.String(spec.Name),
		TokenEndpointUrl: common.String(spec.TokenEndpointUrl),
		ClientId:         common.String(spec.ClientId),
		ClientSecret:     common.String(spec.ClientSecret),
		Scopes:           common.String(spec.Scopes),
	}
	if spec.AuthorizationEndpointUrl != "" {
		details.AuthorizationEndpointUrl = common.String(spec.AuthorizationEndpointUrl)
	}
	if spec.ShortAuthorizationCodeRequestUrl != "" {
		details.ShortAuthorizationCodeRequestUrl = common.String(spec.ShortAuthorizationCodeRequestUrl)
	}
	if spec.RevokeTokenEndpointUrl != "" {
		details.RevokeTokenEndpointUrl = common.String(spec.RevokeTokenEndpointUrl)
	}
	if spec.SubjectClaim != "" {
		details.SubjectClaim = common.String(spec.SubjectClaim)
	}
	if spec.RefreshTokenRetentionPeriodInDays != 0 {
		details.RefreshTokenRetentionPeriodInDays = common.Int(spec.RefreshTokenRetentionPeriodInDays)
	}
	if spec.RedirectUrl != "" {
		details.RedirectUrl = common.String(spec.RedirectUrl)
	}
	if spec.IsVisible {
		details.IsVisible = common.Bool(spec.IsVisible)
	}
	if spec.FreeformTags != nil {
		details.FreeformTags = maps.Clone(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		details.DefinedTags = authenticationProviderDefinedTagsFromSpec(spec.DefinedTags)
	}
	return details
}

func buildAuthenticationProviderUpdateDetails(
	resource *odav1beta1.AuthenticationProvider,
	current odasdk.AuthenticationProvider,
) (odasdk.UpdateAuthenticationProviderDetails, bool) {
	spec := resource.Spec
	details := odasdk.UpdateAuthenticationProviderDetails{}
	updateNeeded := false

	if spec.TokenEndpointUrl != "" && spec.TokenEndpointUrl != stringValue(current.TokenEndpointUrl) {
		details.TokenEndpointUrl = common.String(spec.TokenEndpointUrl)
		updateNeeded = true
	}
	if spec.AuthorizationEndpointUrl != "" && spec.AuthorizationEndpointUrl != stringValue(current.AuthorizationEndpointUrl) {
		details.AuthorizationEndpointUrl = common.String(spec.AuthorizationEndpointUrl)
		updateNeeded = true
	}
	if spec.ShortAuthorizationCodeRequestUrl != "" && spec.ShortAuthorizationCodeRequestUrl != stringValue(current.ShortAuthorizationCodeRequestUrl) {
		details.ShortAuthorizationCodeRequestUrl = common.String(spec.ShortAuthorizationCodeRequestUrl)
		updateNeeded = true
	}
	if spec.RevokeTokenEndpointUrl != "" && spec.RevokeTokenEndpointUrl != stringValue(current.RevokeTokenEndpointUrl) {
		details.RevokeTokenEndpointUrl = common.String(spec.RevokeTokenEndpointUrl)
		updateNeeded = true
	}
	if spec.ClientId != "" && spec.ClientId != stringValue(current.ClientId) {
		details.ClientId = common.String(spec.ClientId)
		updateNeeded = true
	}
	if spec.Scopes != "" && spec.Scopes != stringValue(current.Scopes) {
		details.Scopes = common.String(spec.Scopes)
		updateNeeded = true
	}
	if spec.SubjectClaim != "" && spec.SubjectClaim != stringValue(current.SubjectClaim) {
		details.SubjectClaim = common.String(spec.SubjectClaim)
		updateNeeded = true
	}
	if spec.RefreshTokenRetentionPeriodInDays != 0 && spec.RefreshTokenRetentionPeriodInDays != intValue(current.RefreshTokenRetentionPeriodInDays) {
		details.RefreshTokenRetentionPeriodInDays = common.Int(spec.RefreshTokenRetentionPeriodInDays)
		updateNeeded = true
	}
	if spec.RedirectUrl != "" && spec.RedirectUrl != stringValue(current.RedirectUrl) {
		details.RedirectUrl = common.String(spec.RedirectUrl)
		updateNeeded = true
	}
	if spec.FreeformTags != nil && !maps.Equal(spec.FreeformTags, current.FreeformTags) {
		details.FreeformTags = maps.Clone(spec.FreeformTags)
		updateNeeded = true
	}
	if spec.DefinedTags != nil {
		desired := authenticationProviderDefinedTagsFromSpec(spec.DefinedTags)
		if !authenticationProviderJSONEqual(desired, current.DefinedTags) {
			details.DefinedTags = desired
			updateNeeded = true
		}
	}

	return details, updateNeeded
}

func validateAuthenticationProviderCreateOnlyDrift(resource *odav1beta1.AuthenticationProvider, current odasdk.AuthenticationProvider) error {
	spec := resource.Spec
	drift := []string{}
	if spec.GrantType != "" && current.GrantType != "" && spec.GrantType != string(current.GrantType) {
		drift = append(drift, "grantType")
	}
	if spec.IdentityProvider != "" && current.IdentityProvider != "" && spec.IdentityProvider != string(current.IdentityProvider) {
		drift = append(drift, "identityProvider")
	}
	if spec.Name != "" && stringValue(current.Name) != "" && spec.Name != stringValue(current.Name) {
		drift = append(drift, "name")
	}
	if spec.IsVisible && current.IsVisible != nil && spec.IsVisible != boolValue(current.IsVisible) {
		drift = append(drift, "isVisible")
	}
	if len(drift) == 0 {
		return nil
	}
	return fmt.Errorf("AuthenticationProvider create-only field drift detected for %s; recreate the resource instead of updating immutable fields", strings.Join(drift, ", "))
}

func authenticationProviderSummaryMatches(resource *odav1beta1.AuthenticationProvider, item odasdk.AuthenticationProviderSummary) bool {
	if resource == nil {
		return false
	}
	return stringValue(item.Name) == resource.Spec.Name &&
		string(item.IdentityProvider) == resource.Spec.IdentityProvider
}

func isAuthenticationProviderRetryableLifecycle(state odasdk.LifecycleStateEnum) bool {
	switch normalizedAuthenticationProviderLifecycle(state) {
	case string(odasdk.LifecycleStateCreating),
		string(odasdk.LifecycleStateUpdating),
		string(odasdk.LifecycleStateDeleting):
		return true
	default:
		return false
	}
}

func isAuthenticationProviderReadNotFound(err error) bool {
	classification := errorutil.ClassifyDeleteError(err)
	return classification.IsUnambiguousNotFound() || classification.IsAuthShapedNotFound()
}

func authenticationProviderIDFromSDK(current odasdk.AuthenticationProvider) string {
	return strings.TrimSpace(stringValue(current.Id))
}

func authenticationProviderLifecycleMessage(current odasdk.AuthenticationProvider, fallback string) string {
	if name := strings.TrimSpace(stringValue(current.Name)); name != "" {
		return fmt.Sprintf("AuthenticationProvider %s is %s", name, current.LifecycleState)
	}
	return fallback
}

func normalizedAuthenticationProviderLifecycle(state odasdk.LifecycleStateEnum) string {
	return strings.ToUpper(strings.TrimSpace(string(state)))
}

func asyncPhaseForAuthenticationProviderCondition(condition shared.OSOKConditionType) shared.OSOKAsyncPhase {
	switch condition {
	case shared.Provisioning:
		return shared.OSOKAsyncPhaseCreate
	case shared.Updating:
		return shared.OSOKAsyncPhaseUpdate
	case shared.Terminating:
		return shared.OSOKAsyncPhaseDelete
	default:
		return ""
	}
}

func authenticationProviderDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
	if spec == nil {
		return nil
	}
	return *util.ConvertToOciDefinedTags(&spec)
}

func authenticationProviderStatusDefinedTags(input map[string]map[string]interface{}) map[string]shared.MapValue {
	if len(input) == 0 {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(input))
	for namespace, values := range input {
		if len(values) == 0 {
			continue
		}
		tagValues := make(shared.MapValue, len(values))
		for key, value := range values {
			tagValues[key] = fmt.Sprint(value)
		}
		converted[namespace] = tagValues
	}
	if len(converted) == 0 {
		return nil
	}
	return converted
}

func authenticationProviderJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}

func authenticationProviderSDKTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Time.Format(time.RFC3339Nano)
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	return maps.Clone(input)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func intValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func boolValue(value *bool) bool {
	if value == nil {
		return false
	}
	return *value
}

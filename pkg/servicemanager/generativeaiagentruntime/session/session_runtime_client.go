/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package session

import (
	"context"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	generativeaiagentruntimesdk "github.com/oracle/oci-go-sdk/v65/generativeaiagentruntime"
	generativeaiagentruntimev1beta1 "github.com/oracle/oci-service-operator/api/generativeaiagentruntime/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const sessionKind = "Session"

type sessionOCIClient interface {
	CreateSession(context.Context, generativeaiagentruntimesdk.CreateSessionRequest) (generativeaiagentruntimesdk.CreateSessionResponse, error)
	GetSession(context.Context, generativeaiagentruntimesdk.GetSessionRequest) (generativeaiagentruntimesdk.GetSessionResponse, error)
	UpdateSession(context.Context, generativeaiagentruntimesdk.UpdateSessionRequest) (generativeaiagentruntimesdk.UpdateSessionResponse, error)
	DeleteSession(context.Context, generativeaiagentruntimesdk.DeleteSessionRequest) (generativeaiagentruntimesdk.DeleteSessionResponse, error)
}

type sessionPathIdentity struct {
	agentEndpointID string
}

type synchronousSessionServiceClient struct {
	delegate SessionServiceClient
	log      loggerutil.OSOKLogger
}

type sessionStatusMirrorClient struct {
	delegate SessionServiceClient
}

func init() {
	registerSessionRuntimeHooksMutator(func(manager *SessionServiceManager, hooks *SessionRuntimeHooks) {
		applySessionRuntimeHooks(manager, hooks)
	})
}

func applySessionRuntimeHooks(manager *SessionServiceManager, hooks *SessionRuntimeHooks) {
	if hooks == nil {
		return
	}

	log := loggerutil.OSOKLogger{}
	if manager != nil {
		log = manager.Log
	}

	hooks.Semantics = reviewedSessionRuntimeSemantics()
	hooks.BuildCreateBody = buildSessionCreateBody
	hooks.BuildUpdateBody = buildSessionUpdateBody
	hooks.Identity = generatedruntime.IdentityHooks[*generativeaiagentruntimev1beta1.Session]{
		Resolve:    resolveSessionPathIdentity,
		RecordPath: recordSessionPathIdentity,
	}
	hooks.Create.Fields = sessionCreateFields()
	hooks.Get.Fields = sessionGetFields()
	hooks.Update.Fields = sessionUpdateFields()
	hooks.Delete.Fields = sessionDeleteFields()
	hooks.WrapGeneratedClient = append(
		hooks.WrapGeneratedClient,
		func(delegate SessionServiceClient) SessionServiceClient {
			return &synchronousSessionServiceClient{
				delegate: delegate,
				log:      log,
			}
		},
		wrapSessionStatusMirrorClient,
	)
}

func newSessionServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client sessionOCIClient,
) SessionServiceClient {
	hooks := newSessionRuntimeHooksWithOCIClient(client)
	applySessionRuntimeHooks(&SessionServiceManager{Log: log}, &hooks)
	delegate := defaultSessionServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*generativeaiagentruntimev1beta1.Session](
			buildSessionGeneratedRuntimeConfig(&SessionServiceManager{Log: log}, hooks),
		),
	}
	return wrapSessionGeneratedClient(hooks, delegate)
}

func newSessionRuntimeHooksWithOCIClient(client sessionOCIClient) SessionRuntimeHooks {
	return SessionRuntimeHooks{
		Semantics:       reviewedSessionRuntimeSemantics(),
		Identity:        generatedruntime.IdentityHooks[*generativeaiagentruntimev1beta1.Session]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*generativeaiagentruntimev1beta1.Session]{},
		StatusHooks:     generatedruntime.StatusHooks[*generativeaiagentruntimev1beta1.Session]{},
		ParityHooks:     generatedruntime.ParityHooks[*generativeaiagentruntimev1beta1.Session]{},
		Async:           generatedruntime.AsyncHooks[*generativeaiagentruntimev1beta1.Session]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*generativeaiagentruntimev1beta1.Session]{},
		Create: runtimeOperationHooks[generativeaiagentruntimesdk.CreateSessionRequest, generativeaiagentruntimesdk.CreateSessionResponse]{
			Fields: sessionCreateFields(),
			Call: func(ctx context.Context, request generativeaiagentruntimesdk.CreateSessionRequest) (generativeaiagentruntimesdk.CreateSessionResponse, error) {
				return client.CreateSession(ctx, request)
			},
		},
		Get: runtimeOperationHooks[generativeaiagentruntimesdk.GetSessionRequest, generativeaiagentruntimesdk.GetSessionResponse]{
			Fields: sessionGetFields(),
			Call: func(ctx context.Context, request generativeaiagentruntimesdk.GetSessionRequest) (generativeaiagentruntimesdk.GetSessionResponse, error) {
				return client.GetSession(ctx, request)
			},
		},
		Update: runtimeOperationHooks[generativeaiagentruntimesdk.UpdateSessionRequest, generativeaiagentruntimesdk.UpdateSessionResponse]{
			Fields: sessionUpdateFields(),
			Call: func(ctx context.Context, request generativeaiagentruntimesdk.UpdateSessionRequest) (generativeaiagentruntimesdk.UpdateSessionResponse, error) {
				return client.UpdateSession(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[generativeaiagentruntimesdk.DeleteSessionRequest, generativeaiagentruntimesdk.DeleteSessionResponse]{
			Fields: sessionDeleteFields(),
			Call: func(ctx context.Context, request generativeaiagentruntimesdk.DeleteSessionRequest) (generativeaiagentruntimesdk.DeleteSessionResponse, error) {
				return client.DeleteSession(ctx, request)
			},
		},
		WrapGeneratedClient: []func(SessionServiceClient) SessionServiceClient{},
	}
}

func reviewedSessionRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newSessionRuntimeSemantics()
	semantics.Lifecycle = generatedruntime.LifecycleSemantics{}
	semantics.AuxiliaryOperations = nil
	semantics.Unsupported = nil
	return semantics
}

func sessionCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		sessionAgentEndpointIDField(),
		{FieldName: "CreateSessionDetails", RequestName: "CreateSessionDetails", Contribution: "body"},
	}
}

func sessionGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		sessionAgentEndpointIDField(),
		{FieldName: "SessionId", RequestName: "sessionId", Contribution: "path", PreferResourceID: true},
	}
}

func sessionUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		sessionAgentEndpointIDField(),
		{FieldName: "SessionId", RequestName: "sessionId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateSessionDetails", RequestName: "UpdateSessionDetails", Contribution: "body"},
	}
}

func sessionDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		sessionAgentEndpointIDField(),
		{FieldName: "SessionId", RequestName: "sessionId", Contribution: "path", PreferResourceID: true},
	}
}

func sessionAgentEndpointIDField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:    "AgentEndpointId",
		RequestName:  "agentEndpointId",
		Contribution: "path",
		LookupPaths:  []string{"status.agentEndpointId", "spec.agentEndpointId"},
	}
}

func resolveSessionPathIdentity(resource *generativeaiagentruntimev1beta1.Session) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("%s resource is nil", sessionKind)
	}

	agentEndpointID := firstNonEmptyTrim(resource.Status.AgentEndpointId, resource.Spec.AgentEndpointId)
	if agentEndpointID == "" {
		return nil, fmt.Errorf("%s spec.agentEndpointId is required", sessionKind)
	}

	return sessionPathIdentity{agentEndpointID: agentEndpointID}, nil
}

func recordSessionPathIdentity(resource *generativeaiagentruntimev1beta1.Session, identity any) {
	if resource == nil {
		return
	}

	resolved, ok := identity.(sessionPathIdentity)
	if !ok {
		return
	}
	resource.Status.AgentEndpointId = resolved.agentEndpointID
}

func wrapSessionStatusMirrorClient(delegate SessionServiceClient) SessionServiceClient {
	return sessionStatusMirrorClient{delegate: delegate}
}

func (c sessionStatusMirrorClient) CreateOrUpdate(
	ctx context.Context,
	resource *generativeaiagentruntimev1beta1.Session,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err == nil && response.IsSuccessful {
		projectSessionPathIdentity(resource)
	}
	return response, err
}

func (c sessionStatusMirrorClient) Delete(ctx context.Context, resource *generativeaiagentruntimev1beta1.Session) (bool, error) {
	return c.delegate.Delete(ctx, resource)
}

func projectSessionPathIdentity(resource *generativeaiagentruntimev1beta1.Session) {
	if resource == nil {
		return
	}

	if agentEndpointID := firstNonEmptyTrim(resource.Status.AgentEndpointId, resource.Spec.AgentEndpointId); agentEndpointID != "" {
		resource.Status.AgentEndpointId = agentEndpointID
	}
	if sessionID := firstNonEmptyTrim(resource.Status.SessionId, resource.Status.Id, string(resource.Status.OsokStatus.Ocid)); sessionID != "" {
		resource.Status.SessionId = sessionID
	}
}

func (c *synchronousSessionServiceClient) CreateOrUpdate(
	ctx context.Context,
	resource *generativeaiagentruntimev1beta1.Session,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err != nil || !response.IsSuccessful || !response.ShouldRequeue || resource == nil {
		return response, err
	}

	status := &resource.Status.OsokStatus
	if status.Async.Current != nil {
		return response, err
	}
	if status.Reason != string(shared.Provisioning) && status.Reason != string(shared.Updating) {
		return response, err
	}

	now := metav1.Now()
	servicemanager.ClearAsyncOperation(status)
	status.Reason = string(shared.Active)
	status.UpdatedAt = &now
	if strings.TrimSpace(status.Message) == "" {
		status.Message = firstNonEmptyTrim(
			resource.Status.DisplayName,
			resource.Spec.DisplayName,
			resource.Status.SessionId,
			resource.Status.Id,
		)
		if status.Message == "" {
			status.Message = "OCI Session is active"
		}
	}
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(
		resource.Status.OsokStatus,
		shared.Active,
		corev1.ConditionTrue,
		"",
		status.Message,
		c.log,
	)

	response.ShouldRequeue = false
	response.RequeueDuration = 0
	return response, nil
}

func (c *synchronousSessionServiceClient) Delete(
	ctx context.Context,
	resource *generativeaiagentruntimev1beta1.Session,
) (bool, error) {
	return c.delegate.Delete(ctx, resource)
}

func buildSessionCreateBody(
	_ context.Context,
	resource *generativeaiagentruntimev1beta1.Session,
	_ string,
) (any, error) {
	if err := validateSessionSpec(resource); err != nil {
		return nil, err
	}

	body := generativeaiagentruntimesdk.CreateSessionDetails{}
	if value := strings.TrimSpace(resource.Spec.DisplayName); value != "" {
		body.DisplayName = common.String(value)
	}
	if value := strings.TrimSpace(resource.Spec.Description); value != "" {
		body.Description = common.String(value)
	}
	return body, nil
}

func buildSessionUpdateBody(
	_ context.Context,
	resource *generativeaiagentruntimev1beta1.Session,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if err := validateSessionSpec(resource); err != nil {
		return generativeaiagentruntimesdk.UpdateSessionDetails{}, false, err
	}

	current, err := sessionFromResponse(currentResponse)
	if err != nil {
		return generativeaiagentruntimesdk.UpdateSessionDetails{}, false, err
	}

	body := generativeaiagentruntimesdk.UpdateSessionDetails{}
	updateNeeded := false

	if value, ok := sessionDesiredOptionalStringUpdate(resource.Spec.DisplayName, current.DisplayName); ok {
		body.DisplayName = value
		updateNeeded = true
	}
	if value, ok := sessionDesiredOptionalStringUpdate(resource.Spec.Description, current.Description); ok {
		body.Description = value
		updateNeeded = true
	}

	return body, updateNeeded, nil
}

func validateSessionSpec(resource *generativeaiagentruntimev1beta1.Session) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", sessionKind)
	}
	if strings.TrimSpace(resource.Spec.AgentEndpointId) == "" {
		return fmt.Errorf("%s spec.agentEndpointId is required", sessionKind)
	}
	return nil
}

func sessionFromResponse(currentResponse any) (generativeaiagentruntimesdk.Session, error) {
	switch current := currentResponse.(type) {
	case generativeaiagentruntimesdk.Session:
		return current, nil
	case *generativeaiagentruntimesdk.Session:
		if current == nil {
			return generativeaiagentruntimesdk.Session{}, fmt.Errorf("current %s response is nil", sessionKind)
		}
		return *current, nil
	case generativeaiagentruntimesdk.CreateSessionResponse:
		return current.Session, nil
	case *generativeaiagentruntimesdk.CreateSessionResponse:
		if current == nil {
			return generativeaiagentruntimesdk.Session{}, fmt.Errorf("current %s response is nil", sessionKind)
		}
		return current.Session, nil
	case generativeaiagentruntimesdk.GetSessionResponse:
		return current.Session, nil
	case *generativeaiagentruntimesdk.GetSessionResponse:
		if current == nil {
			return generativeaiagentruntimesdk.Session{}, fmt.Errorf("current %s response is nil", sessionKind)
		}
		return current.Session, nil
	case generativeaiagentruntimesdk.UpdateSessionResponse:
		return current.Session, nil
	case *generativeaiagentruntimesdk.UpdateSessionResponse:
		if current == nil {
			return generativeaiagentruntimesdk.Session{}, fmt.Errorf("current %s response is nil", sessionKind)
		}
		return current.Session, nil
	default:
		return generativeaiagentruntimesdk.Session{}, fmt.Errorf("unexpected current %s response type %T", sessionKind, currentResponse)
	}
}

func sessionDesiredOptionalStringUpdate(spec string, current *string) (*string, bool) {
	specValue := strings.TrimSpace(spec)
	currentValue := strings.TrimSpace(stringPointerValue(current))
	if specValue == currentValue {
		return nil, false
	}
	if specValue == "" && current == nil {
		return nil, false
	}
	return common.String(specValue), true
}

func stringPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func firstNonEmptyTrim(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

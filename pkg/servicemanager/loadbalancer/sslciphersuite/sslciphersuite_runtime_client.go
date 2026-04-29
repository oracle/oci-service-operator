/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package sslciphersuite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	loadbalancersdk "github.com/oracle/oci-go-sdk/v65/loadbalancer"
	loadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/loadbalancer/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const sslCipherSuiteLoadBalancerIDAnnotation = "loadbalancer.oracle.com/load-balancer-id"

var sslCipherSuiteWorkRequestAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens:   []string{string(loadbalancersdk.WorkRequestLifecycleStateAccepted), string(loadbalancersdk.WorkRequestLifecycleStateInProgress)},
	SucceededStatusTokens: []string{string(loadbalancersdk.WorkRequestLifecycleStateSucceeded)},
	FailedStatusTokens:    []string{string(loadbalancersdk.WorkRequestLifecycleStateFailed)},
	CreateActionTokens:    []string{"CreateSSLCipherSuite", "CreateSslCipherSuite"},
	UpdateActionTokens:    []string{"UpdateSSLCipherSuite", "UpdateSslCipherSuite"},
	DeleteActionTokens:    []string{"DeleteSSLCipherSuite", "DeleteSslCipherSuite"},
}

type sslCipherSuiteRuntimeOCIClient interface {
	CreateSSLCipherSuite(context.Context, loadbalancersdk.CreateSSLCipherSuiteRequest) (loadbalancersdk.CreateSSLCipherSuiteResponse, error)
	GetSSLCipherSuite(context.Context, loadbalancersdk.GetSSLCipherSuiteRequest) (loadbalancersdk.GetSSLCipherSuiteResponse, error)
	ListSSLCipherSuites(context.Context, loadbalancersdk.ListSSLCipherSuitesRequest) (loadbalancersdk.ListSSLCipherSuitesResponse, error)
	UpdateSSLCipherSuite(context.Context, loadbalancersdk.UpdateSSLCipherSuiteRequest) (loadbalancersdk.UpdateSSLCipherSuiteResponse, error)
	DeleteSSLCipherSuite(context.Context, loadbalancersdk.DeleteSSLCipherSuiteRequest) (loadbalancersdk.DeleteSSLCipherSuiteResponse, error)
	GetWorkRequest(context.Context, loadbalancersdk.GetWorkRequestRequest) (loadbalancersdk.GetWorkRequestResponse, error)
}

type sslCipherSuiteIdentity struct {
	loadBalancerID     string
	sslCipherSuiteName string
}

type sslCipherSuiteWorkRequestView struct {
	Id             string
	Status         string
	OperationType  string
	Message        string
	LoadBalancerId string
}

type sslCipherSuiteReadbackActiveClient struct {
	delegate SSLCipherSuiteServiceClient
	log      loggerutil.OSOKLogger
}

func init() {
	registerSSLCipherSuiteRuntimeHooksMutator(func(manager *SSLCipherSuiteServiceManager, hooks *SSLCipherSuiteRuntimeHooks) {
		client, initErr := newSSLCipherSuiteSDKClient(manager)
		var log loggerutil.OSOKLogger
		if manager != nil {
			log = manager.Log
		}
		applySSLCipherSuiteRuntimeHooks(hooks, client, initErr, log)
	})
}

func applySSLCipherSuiteRuntimeHooks(
	hooks *SSLCipherSuiteRuntimeHooks,
	client sslCipherSuiteRuntimeOCIClient,
	initErr error,
	log loggerutil.OSOKLogger,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newSSLCipherSuiteRuntimeSemantics()
	hooks.BuildCreateBody = func(
		_ context.Context,
		resource *loadbalancerv1beta1.SSLCipherSuite,
		_ string,
	) (any, error) {
		return buildSSLCipherSuiteCreateBody(resource)
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *loadbalancerv1beta1.SSLCipherSuite,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildSSLCipherSuiteUpdateBody(resource, currentResponse)
	}
	hooks.Identity = generatedruntime.IdentityHooks[*loadbalancerv1beta1.SSLCipherSuite]{
		Resolve: func(resource *loadbalancerv1beta1.SSLCipherSuite) (any, error) {
			return resolveSSLCipherSuiteIdentity(resource)
		},
		RecordPath: func(resource *loadbalancerv1beta1.SSLCipherSuite, identity any) {
			recordSSLCipherSuitePathIdentity(resource, identity.(sslCipherSuiteIdentity))
		},
		RecordTracked: func(resource *loadbalancerv1beta1.SSLCipherSuite, identity any, _ string) {
			recordSSLCipherSuiteTrackedIdentity(resource, identity.(sslCipherSuiteIdentity))
		},
		LookupExisting: func(context.Context, *loadbalancerv1beta1.SSLCipherSuite, any) (any, error) {
			return nil, nil
		},
	}
	hooks.Async = generatedruntime.AsyncHooks[*loadbalancerv1beta1.SSLCipherSuite]{
		Adapter: sslCipherSuiteWorkRequestAdapter,
		GetWorkRequest: func(ctx context.Context, workRequestID string) (any, error) {
			if initErr != nil {
				return nil, initErr
			}
			if client == nil {
				return nil, errors.New("SSLCipherSuite OCI client is nil")
			}
			response, err := client.GetWorkRequest(ctx, loadbalancersdk.GetWorkRequestRequest{
				WorkRequestId: common.String(workRequestID),
			})
			if err != nil {
				return nil, err
			}
			return sslCipherSuiteWorkRequestViewFromResponse(response), nil
		},
		ResolveAction: func(workRequest any) (string, error) {
			return workRequestOperationType(workRequest), nil
		},
		RecoverResourceID: func(resource *loadbalancerv1beta1.SSLCipherSuite, workRequest any, _ shared.OSOKAsyncPhase) (string, error) {
			return recoverSSLCipherSuiteLoadBalancerID(resource, workRequest), nil
		},
	}
	hooks.Create.Fields = sslCipherSuiteCreateFields()
	hooks.Get.Fields = sslCipherSuiteGetFields()
	hooks.List.Fields = sslCipherSuiteListFields()
	hooks.Update.Fields = sslCipherSuiteUpdateFields()
	hooks.Delete.Fields = sslCipherSuiteDeleteFields()
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate SSLCipherSuiteServiceClient) SSLCipherSuiteServiceClient {
		return &sslCipherSuiteReadbackActiveClient{
			delegate: delegate,
			log:      log,
		}
	})
}

func (c *sslCipherSuiteReadbackActiveClient) CreateOrUpdate(
	ctx context.Context,
	resource *loadbalancerv1beta1.SSLCipherSuite,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("SSLCipherSuite generated delegate is not configured")
	}

	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err != nil || !response.IsSuccessful || !response.ShouldRequeue || resource == nil {
		return response, err
	}
	if !shouldMarkSSLCipherSuiteReadbackActive(resource) {
		return response, err
	}

	status := &resource.Status.OsokStatus
	now := metav1.Now()
	servicemanager.ClearAsyncOperation(status)
	status.Reason = string(shared.Active)
	status.UpdatedAt = &now
	status.Message = firstNonEmptyTrim(status.Message, resource.Status.Name, resource.Spec.Name, resource.Name)
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

func (c *sslCipherSuiteReadbackActiveClient) Delete(ctx context.Context, resource *loadbalancerv1beta1.SSLCipherSuite) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("SSLCipherSuite generated delegate is not configured")
	}
	return c.delegate.Delete(ctx, resource)
}

func shouldMarkSSLCipherSuiteReadbackActive(resource *loadbalancerv1beta1.SSLCipherSuite) bool {
	if resource == nil {
		return false
	}
	status := resource.Status.OsokStatus
	if status.Async.Current != nil {
		return false
	}
	if status.Reason != string(shared.Provisioning) && status.Reason != string(shared.Updating) {
		return false
	}
	return firstNonEmptyTrim(string(status.Ocid), resource.Status.Name) != ""
}

func newSSLCipherSuiteSDKClient(manager *SSLCipherSuiteServiceManager) (sslCipherSuiteRuntimeOCIClient, error) {
	if manager == nil {
		return nil, errors.New("SSLCipherSuite service manager is nil")
	}
	client, err := loadbalancersdk.NewLoadBalancerClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, fmt.Errorf("initialize SSLCipherSuite OCI client: %w", err)
	}
	return client, nil
}

func newSSLCipherSuiteRuntimeHooksWithOCIClient(client sslCipherSuiteRuntimeOCIClient) SSLCipherSuiteRuntimeHooks {
	return SSLCipherSuiteRuntimeHooks{
		Semantics: newSSLCipherSuiteRuntimeSemantics(),
		Identity:  generatedruntime.IdentityHooks[*loadbalancerv1beta1.SSLCipherSuite]{},
		Read:      generatedruntime.ReadHooks{},
		Async: generatedruntime.AsyncHooks[*loadbalancerv1beta1.SSLCipherSuite]{
			Adapter: sslCipherSuiteWorkRequestAdapter,
		},
		Create: runtimeOperationHooks[loadbalancersdk.CreateSSLCipherSuiteRequest, loadbalancersdk.CreateSSLCipherSuiteResponse]{
			Fields: sslCipherSuiteCreateFields(),
			Call: func(ctx context.Context, request loadbalancersdk.CreateSSLCipherSuiteRequest) (loadbalancersdk.CreateSSLCipherSuiteResponse, error) {
				return client.CreateSSLCipherSuite(ctx, request)
			},
		},
		Get: runtimeOperationHooks[loadbalancersdk.GetSSLCipherSuiteRequest, loadbalancersdk.GetSSLCipherSuiteResponse]{
			Fields: sslCipherSuiteGetFields(),
			Call: func(ctx context.Context, request loadbalancersdk.GetSSLCipherSuiteRequest) (loadbalancersdk.GetSSLCipherSuiteResponse, error) {
				return client.GetSSLCipherSuite(ctx, request)
			},
		},
		List: runtimeOperationHooks[loadbalancersdk.ListSSLCipherSuitesRequest, loadbalancersdk.ListSSLCipherSuitesResponse]{
			Fields: sslCipherSuiteListFields(),
			Call: func(ctx context.Context, request loadbalancersdk.ListSSLCipherSuitesRequest) (loadbalancersdk.ListSSLCipherSuitesResponse, error) {
				return client.ListSSLCipherSuites(ctx, request)
			},
		},
		Update: runtimeOperationHooks[loadbalancersdk.UpdateSSLCipherSuiteRequest, loadbalancersdk.UpdateSSLCipherSuiteResponse]{
			Fields: sslCipherSuiteUpdateFields(),
			Call: func(ctx context.Context, request loadbalancersdk.UpdateSSLCipherSuiteRequest) (loadbalancersdk.UpdateSSLCipherSuiteResponse, error) {
				return client.UpdateSSLCipherSuite(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[loadbalancersdk.DeleteSSLCipherSuiteRequest, loadbalancersdk.DeleteSSLCipherSuiteResponse]{
			Fields: sslCipherSuiteDeleteFields(),
			Call: func(ctx context.Context, request loadbalancersdk.DeleteSSLCipherSuiteRequest) (loadbalancersdk.DeleteSSLCipherSuiteResponse, error) {
				return client.DeleteSSLCipherSuite(ctx, request)
			},
		},
		WrapGeneratedClient: []func(SSLCipherSuiteServiceClient) SSLCipherSuiteServiceClient{},
	}
}

func newSSLCipherSuiteRuntimeSemantics() *generatedruntime.Semantics {
	workRequestPendingStates := []string{
		string(loadbalancersdk.WorkRequestLifecycleStateAccepted),
		string(loadbalancersdk.WorkRequestLifecycleStateInProgress),
	}
	return &generatedruntime.Semantics{
		FormalService: "loadbalancer",
		FormalSlug:    "sslciphersuite",
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
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: workRequestPendingStates,
			UpdatingStates:     workRequestPendingStates,
			ActiveStates:       []string{string(loadbalancersdk.WorkRequestLifecycleStateSucceeded), "ACTIVE"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  append([]string{"DELETING"}, workRequestPendingStates...),
			TerminalStates: []string{"DELETED", string(loadbalancersdk.WorkRequestLifecycleStateSucceeded)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"name"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"ciphers"},
			ForceNew:      []string{"name"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "workrequest",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.WaitForWorkRequestWithErrorHandling"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "workrequest",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.WaitForWorkRequestWithErrorHandling"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func sslCipherSuiteCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		sslCipherSuiteLoadBalancerIDField(),
		{
			FieldName:    "CreateSslCipherSuiteDetails",
			RequestName:  "CreateSslCipherSuiteDetails",
			Contribution: "body",
		},
	}
}

func sslCipherSuiteGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		sslCipherSuiteLoadBalancerIDField(),
		sslCipherSuiteNameField(),
	}
}

func sslCipherSuiteListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		sslCipherSuiteLoadBalancerIDField(),
	}
}

func sslCipherSuiteUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		sslCipherSuiteLoadBalancerIDField(),
		sslCipherSuiteNameField(),
		{
			FieldName:    "UpdateSslCipherSuiteDetails",
			RequestName:  "UpdateSslCipherSuiteDetails",
			Contribution: "body",
		},
	}
}

func sslCipherSuiteDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		sslCipherSuiteLoadBalancerIDField(),
		sslCipherSuiteNameField(),
	}
}

func sslCipherSuiteLoadBalancerIDField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:        "LoadBalancerId",
		RequestName:      "loadBalancerId",
		Contribution:     "path",
		PreferResourceID: true,
		LookupPaths:      []string{"status.status.ocid"},
	}
}

func sslCipherSuiteNameField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:    "Name",
		RequestName:  "name",
		Contribution: "path",
		LookupPaths:  []string{"status.name", "spec.name", "name"},
	}
}

func buildSSLCipherSuiteCreateBody(resource *loadbalancerv1beta1.SSLCipherSuite) (loadbalancersdk.CreateSslCipherSuiteDetails, error) {
	if resource == nil {
		return loadbalancersdk.CreateSslCipherSuiteDetails{}, fmt.Errorf("SSLCipherSuite resource is nil")
	}
	return loadbalancersdk.CreateSslCipherSuiteDetails{
		Name:    stringPointer(firstNonEmptyTrim(resource.Spec.Name, resource.Name)),
		Ciphers: cloneStringSlice(resource.Spec.Ciphers),
	}, nil
}

func buildSSLCipherSuiteUpdateBody(
	resource *loadbalancerv1beta1.SSLCipherSuite,
	currentResponse any,
) (loadbalancersdk.UpdateSslCipherSuiteDetails, bool, error) {
	if resource == nil {
		return loadbalancersdk.UpdateSslCipherSuiteDetails{}, false, fmt.Errorf("SSLCipherSuite resource is nil")
	}

	desired := loadbalancersdk.UpdateSslCipherSuiteDetails{
		Ciphers: cloneStringSlice(resource.Spec.Ciphers),
	}
	currentSource, err := sslCipherSuiteUpdateSource(resource, currentResponse)
	if err != nil {
		return loadbalancersdk.UpdateSslCipherSuiteDetails{}, false, err
	}
	current, err := sslCipherSuiteUpdateDetailsFromValue(currentSource)
	if err != nil {
		return loadbalancersdk.UpdateSslCipherSuiteDetails{}, false, err
	}

	updateNeeded, err := sslCipherSuiteUpdateNeeded(desired, current)
	if err != nil {
		return loadbalancersdk.UpdateSslCipherSuiteDetails{}, false, err
	}
	if !updateNeeded {
		return loadbalancersdk.UpdateSslCipherSuiteDetails{}, false, nil
	}
	return desired, true, nil
}

func sslCipherSuiteUpdateSource(resource *loadbalancerv1beta1.SSLCipherSuite, currentResponse any) (any, error) {
	switch current := currentResponse.(type) {
	case nil:
		if resource == nil {
			return nil, fmt.Errorf("SSLCipherSuite resource is nil")
		}
		return resource.Status, nil
	case loadbalancersdk.SslCipherSuite:
		return current, nil
	case *loadbalancersdk.SslCipherSuite:
		if current == nil {
			return nil, fmt.Errorf("current SSLCipherSuite response is nil")
		}
		return *current, nil
	case loadbalancersdk.GetSSLCipherSuiteResponse:
		return current.SslCipherSuite, nil
	case *loadbalancersdk.GetSSLCipherSuiteResponse:
		if current == nil {
			return nil, fmt.Errorf("current SSLCipherSuite response is nil")
		}
		return current.SslCipherSuite, nil
	default:
		return currentResponse, nil
	}
}

func sslCipherSuiteUpdateDetailsFromValue(value any) (loadbalancersdk.UpdateSslCipherSuiteDetails, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return loadbalancersdk.UpdateSslCipherSuiteDetails{}, fmt.Errorf("marshal SSLCipherSuite update details source: %w", err)
	}

	var details loadbalancersdk.UpdateSslCipherSuiteDetails
	if err := json.Unmarshal(payload, &details); err != nil {
		return loadbalancersdk.UpdateSslCipherSuiteDetails{}, fmt.Errorf("decode SSLCipherSuite update details: %w", err)
	}
	return details, nil
}

func sslCipherSuiteUpdateNeeded(desired loadbalancersdk.UpdateSslCipherSuiteDetails, current loadbalancersdk.UpdateSslCipherSuiteDetails) (bool, error) {
	desiredPayload, err := json.Marshal(desired)
	if err != nil {
		return false, fmt.Errorf("marshal desired SSLCipherSuite update details: %w", err)
	}
	currentPayload, err := json.Marshal(current)
	if err != nil {
		return false, fmt.Errorf("marshal current SSLCipherSuite update details: %w", err)
	}
	return string(desiredPayload) != string(currentPayload), nil
}

func resolveSSLCipherSuiteIdentity(resource *loadbalancerv1beta1.SSLCipherSuite) (sslCipherSuiteIdentity, error) {
	if resource == nil {
		return sslCipherSuiteIdentity{}, fmt.Errorf("resolve SSLCipherSuite identity: resource is nil")
	}

	statusLoadBalancerID := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	annotationLoadBalancerID := strings.TrimSpace(resource.Annotations[sslCipherSuiteLoadBalancerIDAnnotation])
	if statusLoadBalancerID != "" && annotationLoadBalancerID != "" && statusLoadBalancerID != annotationLoadBalancerID {
		return sslCipherSuiteIdentity{}, fmt.Errorf(
			"resolve SSLCipherSuite identity: %s changed from recorded loadBalancerId %q to %q",
			sslCipherSuiteLoadBalancerIDAnnotation,
			statusLoadBalancerID,
			annotationLoadBalancerID,
		)
	}

	identity := sslCipherSuiteIdentity{
		loadBalancerID:     firstNonEmptyTrim(statusLoadBalancerID, annotationLoadBalancerID),
		sslCipherSuiteName: firstNonEmptyTrim(resource.Status.Name, resource.Spec.Name, resource.Name),
	}
	if identity.loadBalancerID == "" {
		return sslCipherSuiteIdentity{}, fmt.Errorf("resolve SSLCipherSuite identity: %s annotation is required", sslCipherSuiteLoadBalancerIDAnnotation)
	}
	if identity.sslCipherSuiteName == "" {
		return sslCipherSuiteIdentity{}, fmt.Errorf("resolve SSLCipherSuite identity: ssl cipher suite name is empty")
	}
	return identity, nil
}

func recordSSLCipherSuitePathIdentity(resource *loadbalancerv1beta1.SSLCipherSuite, identity sslCipherSuiteIdentity) {
	if resource == nil {
		return
	}
	resource.Status.Name = identity.sslCipherSuiteName
	// SSLCipherSuite has no child OCID in the Load Balancer API, so the runtime
	// records the parent loadBalancerId as the stable path identity for requests.
	resource.Status.OsokStatus.Ocid = shared.OCID(identity.loadBalancerID)
}

func recordSSLCipherSuiteTrackedIdentity(resource *loadbalancerv1beta1.SSLCipherSuite, identity sslCipherSuiteIdentity) {
	recordSSLCipherSuitePathIdentity(resource, identity)
}

func recoverSSLCipherSuiteLoadBalancerID(resource *loadbalancerv1beta1.SSLCipherSuite, workRequest any) string {
	if resource != nil {
		if currentID := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); currentID != "" {
			return currentID
		}
		if annotationID := strings.TrimSpace(resource.Annotations[sslCipherSuiteLoadBalancerIDAnnotation]); annotationID != "" {
			return annotationID
		}
	}
	if view, ok := workRequest.(sslCipherSuiteWorkRequestView); ok {
		return strings.TrimSpace(view.LoadBalancerId)
	}
	return ""
}

func sslCipherSuiteWorkRequestViewFromResponse(response loadbalancersdk.GetWorkRequestResponse) sslCipherSuiteWorkRequestView {
	workRequest := response.WorkRequest
	return sslCipherSuiteWorkRequestView{
		Id:             stringValue(workRequest.Id),
		Status:         string(workRequest.LifecycleState),
		OperationType:  stringValue(workRequest.Type),
		Message:        stringValue(workRequest.Message),
		LoadBalancerId: stringValue(workRequest.LoadBalancerId),
	}
}

func workRequestOperationType(workRequest any) string {
	if view, ok := workRequest.(sslCipherSuiteWorkRequestView); ok {
		return strings.TrimSpace(view.OperationType)
	}
	return ""
}

func firstNonEmptyTrim(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func cloneStringSlice(in []string) []string {
	if in == nil {
		return nil
	}
	return append([]string(nil), in...)
}

func stringPointer(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

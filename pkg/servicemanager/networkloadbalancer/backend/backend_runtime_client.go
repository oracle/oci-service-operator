/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package backend

import (
	"context"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	networkloadbalancersdk "github.com/oracle/oci-go-sdk/v65/networkloadbalancer"
	networkloadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/networkloadbalancer/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

var backendWorkRequestAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(networkloadbalancersdk.OperationStatusAccepted),
		string(networkloadbalancersdk.OperationStatusInProgress),
		string(networkloadbalancersdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(networkloadbalancersdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(networkloadbalancersdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(networkloadbalancersdk.OperationStatusCanceled)},
	CreateActionTokens:    []string{string(networkloadbalancersdk.OperationTypeCreateBackend)},
	UpdateActionTokens:    []string{string(networkloadbalancersdk.OperationTypeUpdateBackend)},
	DeleteActionTokens:    []string{string(networkloadbalancersdk.OperationTypeDeleteBackend)},
}

type backendRuntimeOCIClient interface {
	CreateBackend(context.Context, networkloadbalancersdk.CreateBackendRequest) (networkloadbalancersdk.CreateBackendResponse, error)
	GetBackend(context.Context, networkloadbalancersdk.GetBackendRequest) (networkloadbalancersdk.GetBackendResponse, error)
	ListBackends(context.Context, networkloadbalancersdk.ListBackendsRequest) (networkloadbalancersdk.ListBackendsResponse, error)
	UpdateBackend(context.Context, networkloadbalancersdk.UpdateBackendRequest) (networkloadbalancersdk.UpdateBackendResponse, error)
	DeleteBackend(context.Context, networkloadbalancersdk.DeleteBackendRequest) (networkloadbalancersdk.DeleteBackendResponse, error)
	GetWorkRequest(context.Context, networkloadbalancersdk.GetWorkRequestRequest) (networkloadbalancersdk.GetWorkRequestResponse, error)
}

type backendIdentity struct {
	networkLoadBalancerID string
	backendSetName        string
	backendName           string
}

func init() {
	registerBackendRuntimeHooksMutator(func(manager *BackendServiceManager, hooks *BackendRuntimeHooks) {
		client, initErr := newBackendRuntimeOCIClient(manager)
		applyBackendRuntimeHooks(hooks, client, initErr)
	})
}

func newBackendRuntimeOCIClient(manager *BackendServiceManager) (backendRuntimeOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("backend service manager is nil")
	}
	client, err := networkloadbalancersdk.NewNetworkLoadBalancerClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyBackendRuntimeHooks(hooks *BackendRuntimeHooks, client backendRuntimeOCIClient, initErr error) {
	if hooks == nil {
		return
	}

	getCall := hooks.Get.Call
	hooks.Semantics = newBackendRuntimeSemantics()
	hooks.Identity = generatedruntime.IdentityHooks[*networkloadbalancerv1beta1.Backend]{
		Resolve: func(resource *networkloadbalancerv1beta1.Backend) (any, error) {
			return resolveBackendIdentity(resource)
		},
		RecordPath: func(resource *networkloadbalancerv1beta1.Backend, identity any) {
			recordBackendPathIdentity(resource, identity.(backendIdentity))
		},
		RecordTracked: func(resource *networkloadbalancerv1beta1.Backend, identity any, _ string) {
			recordBackendTrackedIdentity(resource, identity.(backendIdentity))
		},
		LookupExisting: func(ctx context.Context, _ *networkloadbalancerv1beta1.Backend, identity any) (any, error) {
			return lookupExistingBackend(ctx, getCall, identity.(backendIdentity))
		},
		SeedSyntheticTrackedID: func(resource *networkloadbalancerv1beta1.Backend, identity any) func() {
			return seedSyntheticBackendOCID(resource, identity.(backendIdentity))
		},
	}
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedBackendIdentity
	hooks.Async = generatedruntime.AsyncHooks[*networkloadbalancerv1beta1.Backend]{
		Adapter: backendWorkRequestAdapter,
		GetWorkRequest: func(ctx context.Context, workRequestID string) (any, error) {
			return getBackendWorkRequest(ctx, client, initErr, workRequestID)
		},
		ResolveAction: func(workRequest any) (string, error) {
			return backendWorkRequestOperationType(workRequest), nil
		},
		RecoverResourceID: func(resource *networkloadbalancerv1beta1.Backend, _ any, _ shared.OSOKAsyncPhase) (string, error) {
			identity, err := resolveBackendIdentity(resource)
			if err != nil {
				return "", err
			}
			return identity.backendName, nil
		},
	}
	hooks.DeleteHooks.HandleError = handleBackendDeleteError
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate BackendServiceClient) BackendServiceClient {
		return backendDeleteGuardClient{
			delegate: delegate,
			client:   client,
			initErr:  initErr,
		}
	})
	hooks.Create.Fields = backendCreateFields()
	hooks.Get.Fields = backendGetFields()
	hooks.List.Fields = backendListFields()
	hooks.Update.Fields = backendUpdateFields()
	hooks.Delete.Fields = backendDeleteFields()
}

type backendDeleteGuardClient struct {
	delegate BackendServiceClient
	client   backendRuntimeOCIClient
	initErr  error
}

func (c backendDeleteGuardClient) CreateOrUpdate(
	ctx context.Context,
	resource *networkloadbalancerv1beta1.Backend,
	request ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if err := validateBackendTrackedAddressDrift(resource); err != nil {
		return failBackendCreateOrUpdate(resource, err)
	}
	return c.delegate.CreateOrUpdate(ctx, resource, request)
}

func (c backendDeleteGuardClient) Delete(ctx context.Context, resource *networkloadbalancerv1beta1.Backend) (bool, error) {
	if hasPendingBackendWriteWorkRequest(resource) {
		response, err := c.delegate.CreateOrUpdate(ctx, resource, ctrl.Request{})
		if err != nil {
			return false, err
		}
		if hasPendingBackendWriteWorkRequest(resource) || response.ShouldRequeue {
			return false, nil
		}
	}
	if hasCurrentBackendDeleteWorkRequest(resource) {
		if err := c.guardCompletedBackendDeleteWorkRequest(ctx, resource); err != nil {
			return false, err
		}
	} else {
		if err := c.guardBackendDelete(ctx, resource); err != nil {
			return false, err
		}
	}
	return c.delegate.Delete(ctx, resource)
}

func (c backendDeleteGuardClient) guardBackendDelete(ctx context.Context, resource *networkloadbalancerv1beta1.Backend) error {
	if c.initErr != nil || c.client == nil {
		return nil
	}
	identity, err := resolveBackendIdentity(resource)
	if err != nil {
		return nil
	}

	_, err = c.client.GetBackend(ctx, networkloadbalancersdk.GetBackendRequest{
		NetworkLoadBalancerId: common.String(identity.networkLoadBalancerID),
		BackendSetName:        common.String(identity.backendSetName),
		BackendName:           common.String(identity.backendName),
	})
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	return newBackendAmbiguousNotFoundError(
		resource,
		errorutil.OpcRequestID(err),
		"Backend pre-delete read returned NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed",
	)
}

func (c backendDeleteGuardClient) guardCompletedBackendDeleteWorkRequest(
	ctx context.Context,
	resource *networkloadbalancerv1beta1.Backend,
) error {
	workRequestID := currentBackendDeleteWorkRequestID(resource)
	if workRequestID == "" {
		return nil
	}

	workRequest, err := getBackendWorkRequest(ctx, c.client, c.initErr, workRequestID)
	if err != nil {
		if resource != nil {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		}
		return err
	}
	currentAsync, err := buildBackendWorkRequestOperation(resource, workRequest, shared.OSOKAsyncPhaseDelete)
	if err != nil {
		return err
	}
	if currentAsync.NormalizedClass != shared.OSOKAsyncClassSucceeded {
		return nil
	}
	return c.rejectAmbiguousDeleteConfirmation(ctx, resource)
}

func (c backendDeleteGuardClient) rejectAmbiguousDeleteConfirmation(
	ctx context.Context,
	resource *networkloadbalancerv1beta1.Backend,
) error {
	if c.initErr != nil {
		return c.initErr
	}
	if c.client == nil {
		return fmt.Errorf("backend OCI client is nil")
	}
	identity, err := resolveBackendIdentity(resource)
	if err != nil {
		return err
	}

	_, err = c.client.GetBackend(ctx, networkloadbalancersdk.GetBackendRequest{
		NetworkLoadBalancerId: common.String(identity.networkLoadBalancerID),
		BackendSetName:        common.String(identity.backendSetName),
		BackendName:           common.String(identity.backendName),
	})
	return backendDeleteConfirmationError(resource, err)
}

func getBackendWorkRequest(
	ctx context.Context,
	client backendRuntimeOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, initErr
	}
	if client == nil {
		return nil, fmt.Errorf("backend OCI client is nil")
	}
	response, err := client.GetWorkRequest(ctx, networkloadbalancersdk.GetWorkRequestRequest{
		WorkRequestId: common.String(workRequestID),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func hasCurrentBackendDeleteWorkRequest(resource *networkloadbalancerv1beta1.Backend) bool {
	return currentBackendDeleteWorkRequestID(resource) != ""
}

func currentBackendDeleteWorkRequestID(resource *networkloadbalancerv1beta1.Backend) string {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return ""
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != "" && current.Source != shared.OSOKAsyncSourceWorkRequest {
		return ""
	}
	if current.Phase != shared.OSOKAsyncPhaseDelete {
		return ""
	}
	return strings.TrimSpace(current.WorkRequestID)
}

func hasPendingBackendWriteWorkRequest(resource *networkloadbalancerv1beta1.Backend) bool {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != shared.OSOKAsyncSourceWorkRequest ||
		current.NormalizedClass != shared.OSOKAsyncClassPending ||
		strings.TrimSpace(current.WorkRequestID) == "" {
		return false
	}
	return current.Phase == shared.OSOKAsyncPhaseCreate || current.Phase == shared.OSOKAsyncPhaseUpdate
}

func newBackendRuntimeHooksWithOCIClient(client backendRuntimeOCIClient) BackendRuntimeHooks {
	return BackendRuntimeHooks{
		Semantics:       newBackendRuntimeSemantics(),
		Identity:        generatedruntime.IdentityHooks[*networkloadbalancerv1beta1.Backend]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*networkloadbalancerv1beta1.Backend]{},
		StatusHooks:     generatedruntime.StatusHooks[*networkloadbalancerv1beta1.Backend]{},
		ParityHooks:     generatedruntime.ParityHooks[*networkloadbalancerv1beta1.Backend]{},
		Async:           generatedruntime.AsyncHooks[*networkloadbalancerv1beta1.Backend]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*networkloadbalancerv1beta1.Backend]{},
		Create: runtimeOperationHooks[networkloadbalancersdk.CreateBackendRequest, networkloadbalancersdk.CreateBackendResponse]{
			Fields: backendCreateFields(),
			Call: func(ctx context.Context, request networkloadbalancersdk.CreateBackendRequest) (networkloadbalancersdk.CreateBackendResponse, error) {
				return client.CreateBackend(ctx, request)
			},
		},
		Get: runtimeOperationHooks[networkloadbalancersdk.GetBackendRequest, networkloadbalancersdk.GetBackendResponse]{
			Fields: backendGetFields(),
			Call: func(ctx context.Context, request networkloadbalancersdk.GetBackendRequest) (networkloadbalancersdk.GetBackendResponse, error) {
				return client.GetBackend(ctx, request)
			},
		},
		List: runtimeOperationHooks[networkloadbalancersdk.ListBackendsRequest, networkloadbalancersdk.ListBackendsResponse]{
			Fields: backendListFields(),
			Call: func(ctx context.Context, request networkloadbalancersdk.ListBackendsRequest) (networkloadbalancersdk.ListBackendsResponse, error) {
				return client.ListBackends(ctx, request)
			},
		},
		Update: runtimeOperationHooks[networkloadbalancersdk.UpdateBackendRequest, networkloadbalancersdk.UpdateBackendResponse]{
			Fields: backendUpdateFields(),
			Call: func(ctx context.Context, request networkloadbalancersdk.UpdateBackendRequest) (networkloadbalancersdk.UpdateBackendResponse, error) {
				return client.UpdateBackend(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[networkloadbalancersdk.DeleteBackendRequest, networkloadbalancersdk.DeleteBackendResponse]{
			Fields: backendDeleteFields(),
			Call: func(ctx context.Context, request networkloadbalancersdk.DeleteBackendRequest) (networkloadbalancersdk.DeleteBackendResponse, error) {
				return client.DeleteBackend(ctx, request)
			},
		},
		WrapGeneratedClient: []func(BackendServiceClient) BackendServiceClient{},
	}
}

func newBackendRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "networkloadbalancer",
		FormalSlug:    "backend",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "workrequest",
			Runtime:              "generatedruntime",
			FormalClassification: "workrequest",
			WorkRequest: &generatedruntime.WorkRequestSemantics{
				Source: "service-sdk",
				Phases: []string{
					string(shared.OSOKAsyncPhaseCreate),
					string(shared.OSOKAsyncPhaseUpdate),
					string(shared.OSOKAsyncPhaseDelete),
				},
			},
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{},
			UpdatingStates:     []string{},
			ActiveStates:       []string{},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"name"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"weight",
				"isBackup",
				"isDrain",
				"isOffline",
			},
			ForceNew: []string{
				"backendSetName",
				"ipAddress",
				"name",
				"networkLoadBalancerId",
				"port",
				"targetId",
			},
			ConflictsWith: map[string][]string{
				"ipAddress": {"targetId"},
				"targetId":  {"ipAddress"},
			},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "Backend", Action: "CreateBackend"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "Backend", Action: "UpdateBackend"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "Backend", Action: "DeleteBackend"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "WorkRequest", Action: "GetWorkRequest"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "WorkRequest", Action: "GetWorkRequest"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "WorkRequest", Action: "GetWorkRequest"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func backendCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		backendNetworkLoadBalancerIDField(),
		backendSetNameField(),
		{
			FieldName:    "CreateBackendDetails",
			RequestName:  "CreateBackendDetails",
			Contribution: "body",
		},
	}
}

func backendGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		backendNetworkLoadBalancerIDField(),
		backendSetNameField(),
		backendNameField(),
	}
}

func backendListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		backendNetworkLoadBalancerIDField(),
		backendSetNameField(),
	}
}

func backendUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		backendNetworkLoadBalancerIDField(),
		backendSetNameField(),
		backendNameField(),
		{
			FieldName:    "UpdateBackendDetails",
			RequestName:  "UpdateBackendDetails",
			Contribution: "body",
		},
	}
}

func backendDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		backendNetworkLoadBalancerIDField(),
		backendSetNameField(),
		backendNameField(),
	}
}

func backendNetworkLoadBalancerIDField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:    "NetworkLoadBalancerId",
		RequestName:  "networkLoadBalancerId",
		Contribution: "path",
		LookupPaths:  []string{"status.networkLoadBalancerId", "spec.networkLoadBalancerId"},
	}
}

func backendSetNameField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:    "BackendSetName",
		RequestName:  "backendSetName",
		Contribution: "path",
		LookupPaths:  []string{"status.backendSetName", "spec.backendSetName"},
	}
}

func backendNameField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:        "BackendName",
		RequestName:      "backendName",
		Contribution:     "path",
		PreferResourceID: true,
		LookupPaths:      []string{"status.name", "spec.name", "name"},
	}
}

func lookupExistingBackend(
	ctx context.Context,
	getCall func(context.Context, networkloadbalancersdk.GetBackendRequest) (networkloadbalancersdk.GetBackendResponse, error),
	identity backendIdentity,
) (any, error) {
	if getCall == nil {
		return nil, nil
	}

	return getCall(ctx, networkloadbalancersdk.GetBackendRequest{
		NetworkLoadBalancerId: common.String(identity.networkLoadBalancerID),
		BackendSetName:        common.String(identity.backendSetName),
		BackendName:           common.String(identity.backendName),
	})
}

func resolveBackendIdentity(resource *networkloadbalancerv1beta1.Backend) (backendIdentity, error) {
	if resource == nil {
		return backendIdentity{}, fmt.Errorf("resolve Backend identity: resource is nil")
	}
	identity := backendIdentity{
		networkLoadBalancerID: firstNonEmptyTrim(resource.Status.NetworkLoadBalancerId, resource.Spec.NetworkLoadBalancerId),
		backendSetName:        firstNonEmptyTrim(resource.Status.BackendSetName, resource.Spec.BackendSetName),
		backendName:           currentBackendName(resource),
	}
	if identity.networkLoadBalancerID == "" {
		return backendIdentity{}, fmt.Errorf("resolve Backend identity: networkLoadBalancerId is empty")
	}
	if identity.backendSetName == "" {
		return backendIdentity{}, fmt.Errorf("resolve Backend identity: backendSetName is empty")
	}
	if identity.backendName == "" {
		return backendIdentity{}, fmt.Errorf("resolve Backend identity: backend name is empty")
	}
	return identity, nil
}

func currentBackendName(resource *networkloadbalancerv1beta1.Backend) string {
	if resource == nil {
		return ""
	}
	if name := strings.TrimSpace(resource.Status.Name); name != "" {
		return name
	}
	if name := strings.TrimSpace(resource.Spec.Name); name != "" {
		return name
	}

	target := firstNonEmptyTrim(resource.Spec.TargetId, resource.Spec.IpAddress)
	if target == "" {
		return ""
	}
	return fmt.Sprintf("%s:%d", target, resource.Spec.Port)
}

func validateBackendTrackedAddressDrift(resource *networkloadbalancerv1beta1.Backend) error {
	if resource == nil || !hasTrackedBackendAddress(resource) {
		return nil
	}

	desiredValue, desiredField := desiredBackendAddress(resource)
	currentValue, currentField := trackedBackendAddress(resource)
	if currentValue == "" {
		return nil
	}
	if desiredValue == "" {
		return backendForceNewDriftError(currentField)
	}
	if desiredValue != currentValue {
		return backendForceNewDriftError(desiredField)
	}
	if currentField != "backendName" && desiredField != currentField {
		return backendForceNewDriftError(desiredField)
	}
	return nil
}

func hasTrackedBackendAddress(resource *networkloadbalancerv1beta1.Backend) bool {
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)) != "" ||
		strings.TrimSpace(resource.Status.Name) != "" ||
		strings.TrimSpace(resource.Status.IpAddress) != "" ||
		strings.TrimSpace(resource.Status.TargetId) != ""
}

func desiredBackendAddress(resource *networkloadbalancerv1beta1.Backend) (string, string) {
	if targetID := strings.TrimSpace(resource.Spec.TargetId); targetID != "" {
		return targetID, "targetId"
	}
	if ipAddress := strings.TrimSpace(resource.Spec.IpAddress); ipAddress != "" {
		return ipAddress, "ipAddress"
	}
	return "", ""
}

func trackedBackendAddress(resource *networkloadbalancerv1beta1.Backend) (string, string) {
	if targetID := strings.TrimSpace(resource.Status.TargetId); targetID != "" {
		return targetID, "targetId"
	}
	if ipAddress := strings.TrimSpace(resource.Status.IpAddress); ipAddress != "" {
		return ipAddress, "ipAddress"
	}
	if strings.TrimSpace(resource.Spec.Name) != "" {
		return "", ""
	}
	if address := addressFromTrackedBackendName(resource.Status.Name, resource.Status.Port); address != "" {
		return address, "backendName"
	}
	return "", ""
}

func addressFromTrackedBackendName(name string, port int) string {
	name = strings.TrimSpace(name)
	if name == "" || port == 0 {
		return ""
	}
	address, found := strings.CutSuffix(name, fmt.Sprintf(":%d", port))
	if !found {
		return ""
	}
	return strings.TrimSpace(address)
}

func backendForceNewDriftError(field string) error {
	if field == "" || field == "backendName" {
		field = "name"
	}
	return fmt.Errorf("backend formal semantics require replacement when %s changes", field)
}

func failBackendCreateOrUpdate(
	resource *networkloadbalancerv1beta1.Backend,
	err error,
) (servicemanager.OSOKResponse, error) {
	return servicemanager.OSOKResponse{IsSuccessful: false}, markBackendFailure(resource, err)
}

func markBackendFailure(resource *networkloadbalancerv1beta1.Backend, err error) error {
	if resource == nil || err == nil {
		return err
	}
	status := &resource.Status.OsokStatus
	servicemanager.RecordErrorOpcRequestID(status, err)
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	now := metav1.Now()
	status.UpdatedAt = &now
	if status.Async.Current != nil {
		current := *status.Async.Current
		current.NormalizedClass = shared.OSOKAsyncClassFailed
		current.Message = err.Error()
		current.UpdatedAt = &now
		status.Async.Current = &current
	}
	*status = util.UpdateOSOKStatusCondition(
		*status,
		shared.Failed,
		v1.ConditionFalse,
		"",
		err.Error(),
		servicemanager.RuntimeDeps{}.Log,
	)
	return err
}

func recordBackendPathIdentity(resource *networkloadbalancerv1beta1.Backend, identity backendIdentity) {
	if resource == nil {
		return
	}
	resource.Status.NetworkLoadBalancerId = identity.networkLoadBalancerID
	resource.Status.BackendSetName = identity.backendSetName
	resource.Status.Name = identity.backendName
}

func recordBackendTrackedIdentity(resource *networkloadbalancerv1beta1.Backend, identity backendIdentity) {
	recordBackendPathIdentity(resource, identity)
	resource.Status.OsokStatus.Ocid = shared.OCID(identity.backendName)
}

func clearTrackedBackendIdentity(resource *networkloadbalancerv1beta1.Backend) {
	if resource == nil {
		return
	}
	resource.Status.OsokStatus.Ocid = ""
	resource.Status.NetworkLoadBalancerId = ""
	resource.Status.BackendSetName = ""
	resource.Status.Name = ""
}

func seedSyntheticBackendOCID(resource *networkloadbalancerv1beta1.Backend, identity backendIdentity) func() {
	if resource == nil {
		return func() {}
	}

	previous := resource.Status.OsokStatus.Ocid
	resource.Status.OsokStatus.Ocid = shared.OCID(identity.backendName)
	return func() {
		resource.Status.OsokStatus.Ocid = previous
	}
}

func backendWorkRequestOperationType(workRequest any) string {
	switch wr := workRequest.(type) {
	case networkloadbalancersdk.WorkRequest:
		return string(wr.OperationType)
	case *networkloadbalancersdk.WorkRequest:
		if wr == nil {
			return ""
		}
		return string(wr.OperationType)
	default:
		return ""
	}
}

func buildBackendWorkRequestOperation(
	resource *networkloadbalancerv1beta1.Backend,
	workRequest any,
	fallbackPhase shared.OSOKAsyncPhase,
) (*shared.OSOKAsyncOperation, error) {
	current, err := backendWorkRequestFromAny(workRequest)
	if err != nil {
		return nil, err
	}

	var status *shared.OSOKStatus
	if resource != nil {
		status = &resource.Status.OsokStatus
	}
	return servicemanager.BuildWorkRequestAsyncOperation(status, backendWorkRequestAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(current.Status),
		RawAction:        string(current.OperationType),
		RawOperationType: string(current.OperationType),
		WorkRequestID:    firstNonEmptyTrim(backendStringValue(current.Id), currentBackendDeleteWorkRequestID(resource)),
		PercentComplete:  current.PercentComplete,
		FallbackPhase:    fallbackPhase,
	})
}

func backendWorkRequestFromAny(workRequest any) (networkloadbalancersdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case networkloadbalancersdk.WorkRequest:
		return current, nil
	case *networkloadbalancersdk.WorkRequest:
		if current == nil {
			return networkloadbalancersdk.WorkRequest{}, fmt.Errorf("backend work request is nil")
		}
		return *current, nil
	default:
		return networkloadbalancersdk.WorkRequest{}, fmt.Errorf("expected Backend work request, got %T", workRequest)
	}
}

func handleBackendDeleteError(resource *networkloadbalancerv1beta1.Backend, err error) error {
	if err == nil {
		return nil
	}
	classification := errorutil.ClassifyDeleteError(err)
	if !classification.IsAuthShapedNotFound() {
		return err
	}
	requestID := errorutil.OpcRequestID(err)
	return newBackendAmbiguousNotFoundError(
		resource,
		requestID,
		"Backend delete returned NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed",
	)
}

func backendDeleteConfirmationError(resource *networkloadbalancerv1beta1.Backend, err error) error {
	classification := errorutil.ClassifyDeleteError(err)
	if err == nil || classification.IsUnambiguousNotFound() {
		return nil
	}
	if !classification.IsAuthShapedNotFound() {
		return err
	}
	return newBackendAmbiguousNotFoundError(
		resource,
		errorutil.OpcRequestID(err),
		"Backend delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed",
	)
}

func newBackendAmbiguousNotFoundError(
	resource *networkloadbalancerv1beta1.Backend,
	requestID string,
	message string,
) backendAmbiguousNotFoundError {
	if resource != nil {
		servicemanager.SetOpcRequestID(&resource.Status.OsokStatus, requestID)
	}
	return backendAmbiguousNotFoundError{
		message:      message,
		opcRequestID: requestID,
	}
}

type backendAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e backendAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e backendAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func firstNonEmptyTrim(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func backendStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

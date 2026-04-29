/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package odaprivateendpointscanproxy

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
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

const (
	odaPrivateEndpointScanProxyRequeueDuration = time.Minute

	// OdaPrivateEndpointIDAnnotation is the resource-local compatibility input
	// for the required OCI parent path until the CRD exposes odaPrivateEndpointId.
	OdaPrivateEndpointIDAnnotation = "oda.oracle.com/oda-private-endpoint-id"
)

type odaPrivateEndpointScanProxyOCIClient interface {
	CreateOdaPrivateEndpointScanProxy(context.Context, odasdk.CreateOdaPrivateEndpointScanProxyRequest) (odasdk.CreateOdaPrivateEndpointScanProxyResponse, error)
	GetOdaPrivateEndpointScanProxy(context.Context, odasdk.GetOdaPrivateEndpointScanProxyRequest) (odasdk.GetOdaPrivateEndpointScanProxyResponse, error)
	ListOdaPrivateEndpointScanProxies(context.Context, odasdk.ListOdaPrivateEndpointScanProxiesRequest) (odasdk.ListOdaPrivateEndpointScanProxiesResponse, error)
	DeleteOdaPrivateEndpointScanProxy(context.Context, odasdk.DeleteOdaPrivateEndpointScanProxyRequest) (odasdk.DeleteOdaPrivateEndpointScanProxyResponse, error)
}

type odaPrivateEndpointScanProxyRuntimeClient struct {
	delegate OdaPrivateEndpointScanProxyServiceClient
	log      loggerutil.OSOKLogger
	client   odaPrivateEndpointScanProxyOCIClient
	initErr  error
}

var _ OdaPrivateEndpointScanProxyServiceClient = (*odaPrivateEndpointScanProxyRuntimeClient)(nil)

func init() {
	registerOdaPrivateEndpointScanProxyRuntimeHooksMutator(func(manager *OdaPrivateEndpointScanProxyServiceManager, hooks *OdaPrivateEndpointScanProxyRuntimeHooks) {
		client, initErr := newOdaPrivateEndpointScanProxySDKClient(manager)
		applyOdaPrivateEndpointScanProxyRuntimeHooks(manager, hooks, client, initErr)
	})
}

func newOdaPrivateEndpointScanProxySDKClient(manager *OdaPrivateEndpointScanProxyServiceManager) (odaPrivateEndpointScanProxyOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("OdaPrivateEndpointScanProxy service manager is nil")
	}
	client, err := odasdk.NewManagementClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyOdaPrivateEndpointScanProxyRuntimeHooks(
	manager *OdaPrivateEndpointScanProxyServiceManager,
	hooks *OdaPrivateEndpointScanProxyRuntimeHooks,
	client odaPrivateEndpointScanProxyOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedOdaPrivateEndpointScanProxyRuntimeSemantics()

	var log loggerutil.OSOKLogger
	if manager != nil {
		log = manager.Log
	}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate OdaPrivateEndpointScanProxyServiceClient) OdaPrivateEndpointScanProxyServiceClient {
		return &odaPrivateEndpointScanProxyRuntimeClient{
			delegate: delegate,
			log:      log,
			client:   client,
			initErr:  initErr,
		}
	})
}

func newOdaPrivateEndpointScanProxyServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client odaPrivateEndpointScanProxyOCIClient,
) OdaPrivateEndpointScanProxyServiceClient {
	manager := &OdaPrivateEndpointScanProxyServiceManager{Log: log}
	hooks := newOdaPrivateEndpointScanProxyRuntimeHooksWithOCIClient(client)
	applyOdaPrivateEndpointScanProxyRuntimeHooks(manager, &hooks, client, nil)
	delegate := defaultOdaPrivateEndpointScanProxyServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*odav1beta1.OdaPrivateEndpointScanProxy](
			buildOdaPrivateEndpointScanProxyGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapOdaPrivateEndpointScanProxyGeneratedClient(hooks, delegate)
}

func newOdaPrivateEndpointScanProxyRuntimeHooksWithOCIClient(client odaPrivateEndpointScanProxyOCIClient) OdaPrivateEndpointScanProxyRuntimeHooks {
	return OdaPrivateEndpointScanProxyRuntimeHooks{
		Semantics: reviewedOdaPrivateEndpointScanProxyRuntimeSemantics(),
		Create: runtimeOperationHooks[odasdk.CreateOdaPrivateEndpointScanProxyRequest, odasdk.CreateOdaPrivateEndpointScanProxyResponse]{
			Fields: odaPrivateEndpointScanProxyCreateFields(),
			Call: func(ctx context.Context, request odasdk.CreateOdaPrivateEndpointScanProxyRequest) (odasdk.CreateOdaPrivateEndpointScanProxyResponse, error) {
				return client.CreateOdaPrivateEndpointScanProxy(ctx, request)
			},
		},
		Get: runtimeOperationHooks[odasdk.GetOdaPrivateEndpointScanProxyRequest, odasdk.GetOdaPrivateEndpointScanProxyResponse]{
			Fields: odaPrivateEndpointScanProxyGetFields(),
			Call: func(ctx context.Context, request odasdk.GetOdaPrivateEndpointScanProxyRequest) (odasdk.GetOdaPrivateEndpointScanProxyResponse, error) {
				return client.GetOdaPrivateEndpointScanProxy(ctx, request)
			},
		},
		List: runtimeOperationHooks[odasdk.ListOdaPrivateEndpointScanProxiesRequest, odasdk.ListOdaPrivateEndpointScanProxiesResponse]{
			Fields: odaPrivateEndpointScanProxyListFields(),
			Call: func(ctx context.Context, request odasdk.ListOdaPrivateEndpointScanProxiesRequest) (odasdk.ListOdaPrivateEndpointScanProxiesResponse, error) {
				return client.ListOdaPrivateEndpointScanProxies(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[odasdk.DeleteOdaPrivateEndpointScanProxyRequest, odasdk.DeleteOdaPrivateEndpointScanProxyResponse]{
			Fields: odaPrivateEndpointScanProxyDeleteFields(),
			Call: func(ctx context.Context, request odasdk.DeleteOdaPrivateEndpointScanProxyRequest) (odasdk.DeleteOdaPrivateEndpointScanProxyResponse, error) {
				return client.DeleteOdaPrivateEndpointScanProxy(ctx, request)
			},
		},
		WrapGeneratedClient: []func(OdaPrivateEndpointScanProxyServiceClient) OdaPrivateEndpointScanProxyServiceClient{},
	}
}

func reviewedOdaPrivateEndpointScanProxyRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "oda",
		FormalSlug:        "odaprivateendpointscanproxy",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "handwritten",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{"CREATING"},
			ActiveStates:       []string{"ACTIVE"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{"DELETING"},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"scanListenerType", "protocol", "scanListenerInfos"},
		},
		Mutation: generatedruntime.MutationSemantics{
			ForceNew: []string{"scanListenerType", "protocol", "scanListenerInfos"},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func odaPrivateEndpointScanProxyCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "OdaPrivateEndpointId", RequestName: "odaPrivateEndpointId", Contribution: "path"},
		{FieldName: "CreateOdaPrivateEndpointScanProxyDetails", RequestName: "CreateOdaPrivateEndpointScanProxyDetails", Contribution: "body"},
	}
}

func odaPrivateEndpointScanProxyGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "OdaPrivateEndpointScanProxyId", RequestName: "odaPrivateEndpointScanProxyId", Contribution: "path", PreferResourceID: true},
		{FieldName: "OdaPrivateEndpointId", RequestName: "odaPrivateEndpointId", Contribution: "path"},
	}
}

func odaPrivateEndpointScanProxyListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "OdaPrivateEndpointId", RequestName: "odaPrivateEndpointId", Contribution: "path"},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func odaPrivateEndpointScanProxyDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "OdaPrivateEndpointScanProxyId", RequestName: "odaPrivateEndpointScanProxyId", Contribution: "path", PreferResourceID: true},
		{FieldName: "OdaPrivateEndpointId", RequestName: "odaPrivateEndpointId", Contribution: "path"},
	}
}

func (c *odaPrivateEndpointScanProxyRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *odav1beta1.OdaPrivateEndpointScanProxy,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if err := c.validateRuntime(resource); err != nil {
		return c.fail(resource, err)
	}

	parentID, err := odaPrivateEndpointScanProxyParentID(resource)
	if err != nil {
		return c.fail(resource, err)
	}

	currentID := odaPrivateEndpointScanProxyCurrentID(resource)
	if currentID == "" {
		return c.createOrBind(ctx, resource, parentID, req.Namespace)
	}

	current, err := c.get(ctx, parentID, currentID)
	if err != nil {
		if isOdaPrivateEndpointScanProxyReadNotFound(err) {
			return c.fail(resource, newOdaPrivateEndpointScanProxyParentDriftError(parentID, currentID, err))
		}
		return c.fail(resource, fmt.Errorf("get OdaPrivateEndpointScanProxy %q: %w", currentID, err))
	}

	if current.LifecycleState == odasdk.OdaPrivateEndpointScanProxyLifecycleStateDeleted {
		clearOdaPrivateEndpointScanProxyIdentity(resource)
		return c.createOrBind(ctx, resource, parentID, req.Namespace)
	}

	c.projectStatus(resource, current)
	switch current.LifecycleState {
	case odasdk.OdaPrivateEndpointScanProxyLifecycleStateCreating,
		odasdk.OdaPrivateEndpointScanProxyLifecycleStateDeleting,
		odasdk.OdaPrivateEndpointScanProxyLifecycleStateFailed:
		return c.applyLifecycle(resource, current, shared.Active)
	}

	if err := validateOdaPrivateEndpointScanProxyCreateOnlyDrift(resource, current); err != nil {
		return c.fail(resource, err)
	}
	return c.applyLifecycle(resource, current, shared.Active)
}

func (c *odaPrivateEndpointScanProxyRuntimeClient) Delete(
	ctx context.Context,
	resource *odav1beta1.OdaPrivateEndpointScanProxy,
) (bool, error) {
	if err := c.validateRuntime(resource); err != nil {
		c.markFailure(resource, err)
		return false, err
	}

	currentID := odaPrivateEndpointScanProxyCurrentID(resource)
	if currentID == "" {
		c.markDeleted(resource, "OCI resource identifier is not recorded")
		return true, nil
	}

	parentID, err := odaPrivateEndpointScanProxyParentID(resource)
	if err != nil {
		c.markFailure(resource, err)
		return false, err
	}

	current, err := c.get(ctx, parentID, currentID)
	if err != nil {
		if isOdaPrivateEndpointScanProxyDeleteNotFound(err) {
			err = newOdaPrivateEndpointScanProxyAmbiguousDeleteNotFoundError(parentID, currentID, err)
			c.markFailure(resource, err)
			return false, err
		}
		c.markFailure(resource, fmt.Errorf("get OdaPrivateEndpointScanProxy %q before delete: %w", currentID, err))
		return false, err
	}

	c.projectStatus(resource, current)
	switch current.LifecycleState {
	case odasdk.OdaPrivateEndpointScanProxyLifecycleStateDeleted:
		c.markDeleted(resource, "OCI resource deleted")
		return true, nil
	case odasdk.OdaPrivateEndpointScanProxyLifecycleStateDeleting:
		c.markTerminating(resource, "OCI delete is in progress")
		return false, nil
	}

	deleteResponse, err := c.client.DeleteOdaPrivateEndpointScanProxy(ctx, odasdk.DeleteOdaPrivateEndpointScanProxyRequest{
		OdaPrivateEndpointId:          common.String(parentID),
		OdaPrivateEndpointScanProxyId: common.String(currentID),
	})
	if err != nil {
		err = normalizeOdaPrivateEndpointScanProxyOCIError(err)
		if isOdaPrivateEndpointScanProxyDeleteNotFound(err) {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			c.markDeleted(resource, "OCI resource no longer exists")
			return true, nil
		}
		if !isOdaPrivateEndpointScanProxyRetryableDeleteConflict(err) {
			c.markFailure(resource, fmt.Errorf("delete OdaPrivateEndpointScanProxy %q: %w", currentID, err))
			return false, err
		}
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, deleteResponse)
	c.seedWorkRequest(resource, stringValue(deleteResponse.OpcWorkRequestId), shared.OSOKAsyncPhaseDelete)

	confirmed, err := c.get(ctx, parentID, currentID)
	if err != nil {
		if isOdaPrivateEndpointScanProxyDeleteNotFound(err) {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			c.markDeleted(resource, "OCI resource deleted")
			return true, nil
		}
		c.markFailure(resource, fmt.Errorf("confirm delete OdaPrivateEndpointScanProxy %q: %w", currentID, err))
		return false, err
	}
	c.projectStatus(resource, confirmed)
	switch confirmed.LifecycleState {
	case odasdk.OdaPrivateEndpointScanProxyLifecycleStateDeleted:
		c.markDeleted(resource, "OCI resource deleted")
		return true, nil
	case odasdk.OdaPrivateEndpointScanProxyLifecycleStateDeleting:
		c.markTerminating(resource, "OCI delete is in progress")
		return false, nil
	default:
		c.markTerminating(resource, "OCI delete request accepted")
		return false, nil
	}
}

func (c *odaPrivateEndpointScanProxyRuntimeClient) createOrBind(
	ctx context.Context,
	resource *odav1beta1.OdaPrivateEndpointScanProxy,
	parentID string,
	namespace string,
) (servicemanager.OSOKResponse, error) {
	details, err := buildCreateOdaPrivateEndpointScanProxyDetails(resource)
	if err != nil {
		return c.fail(resource, err)
	}

	existing, found, err := c.lookupExisting(ctx, parentID, details)
	if err != nil {
		return c.fail(resource, fmt.Errorf("list OdaPrivateEndpointScanProxy resources: %w", err))
	}
	if found {
		current := odaPrivateEndpointScanProxyFromSummary(existing)
		c.projectStatus(resource, current)
		return c.applyLifecycle(resource, current, shared.Active)
	}

	request := odasdk.CreateOdaPrivateEndpointScanProxyRequest{
		OdaPrivateEndpointId:                     common.String(parentID),
		CreateOdaPrivateEndpointScanProxyDetails: details,
	}
	if retryToken := odaPrivateEndpointScanProxyRetryToken(resource); retryToken != "" {
		request.OpcRetryToken = common.String(retryToken)
	}
	response, err := c.client.CreateOdaPrivateEndpointScanProxy(ctx, request)
	if err != nil {
		return c.fail(resource, fmt.Errorf("create OdaPrivateEndpointScanProxy in namespace %q: %w", namespace, normalizeOdaPrivateEndpointScanProxyOCIError(err)))
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	c.seedWorkRequest(resource, stringValue(response.OpcWorkRequestId), shared.OSOKAsyncPhaseCreate)
	c.projectStatus(resource, response.OdaPrivateEndpointScanProxy)
	return c.applyLifecycle(resource, response.OdaPrivateEndpointScanProxy, shared.Provisioning)
}

func (c *odaPrivateEndpointScanProxyRuntimeClient) lookupExisting(
	ctx context.Context,
	parentID string,
	details odasdk.CreateOdaPrivateEndpointScanProxyDetails,
) (odasdk.OdaPrivateEndpointScanProxySummary, bool, error) {
	var page *string
	for {
		response, err := c.client.ListOdaPrivateEndpointScanProxies(ctx, odasdk.ListOdaPrivateEndpointScanProxiesRequest{
			OdaPrivateEndpointId: common.String(parentID),
			Page:                 page,
		})
		if err != nil {
			return odasdk.OdaPrivateEndpointScanProxySummary{}, false, normalizeOdaPrivateEndpointScanProxyOCIError(err)
		}
		for _, item := range response.OdaPrivateEndpointScanProxyCollection.Items {
			if odaPrivateEndpointScanProxyDeletedOrDeleting(item.LifecycleState) {
				continue
			}
			if odaPrivateEndpointScanProxySummaryMatches(item, details) {
				return item, true, nil
			}
		}
		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			return odasdk.OdaPrivateEndpointScanProxySummary{}, false, nil
		}
		page = response.OpcNextPage
	}
}

func odaPrivateEndpointScanProxyRetryToken(resource *odav1beta1.OdaPrivateEndpointScanProxy) string {
	if resource == nil {
		return ""
	}
	if uid := strings.TrimSpace(string(resource.UID)); uid != "" {
		return uid
	}

	namespace := strings.TrimSpace(resource.Namespace)
	name := strings.TrimSpace(resource.Name)
	if namespace == "" && name == "" {
		return ""
	}

	sum := sha256.Sum256([]byte(namespace + "/" + name))
	return fmt.Sprintf("%x", sum[:16])
}

func (c *odaPrivateEndpointScanProxyRuntimeClient) get(
	ctx context.Context,
	parentID string,
	scanProxyID string,
) (odasdk.OdaPrivateEndpointScanProxy, error) {
	response, err := c.client.GetOdaPrivateEndpointScanProxy(ctx, odasdk.GetOdaPrivateEndpointScanProxyRequest{
		OdaPrivateEndpointId:          common.String(parentID),
		OdaPrivateEndpointScanProxyId: common.String(scanProxyID),
	})
	if err != nil {
		return odasdk.OdaPrivateEndpointScanProxy{}, normalizeOdaPrivateEndpointScanProxyOCIError(err)
	}
	return response.OdaPrivateEndpointScanProxy, nil
}

func (c *odaPrivateEndpointScanProxyRuntimeClient) validateRuntime(resource *odav1beta1.OdaPrivateEndpointScanProxy) error {
	if resource == nil {
		return fmt.Errorf("OdaPrivateEndpointScanProxy resource is nil")
	}
	if c.initErr != nil {
		return fmt.Errorf("initialize OdaPrivateEndpointScanProxy OCI client: %w", c.initErr)
	}
	if c.client == nil {
		return fmt.Errorf("OdaPrivateEndpointScanProxy OCI client is not configured")
	}
	return nil
}

func (c *odaPrivateEndpointScanProxyRuntimeClient) fail(
	resource *odav1beta1.OdaPrivateEndpointScanProxy,
	err error,
) (servicemanager.OSOKResponse, error) {
	c.markFailure(resource, err)
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func (c *odaPrivateEndpointScanProxyRuntimeClient) markFailure(resource *odav1beta1.OdaPrivateEndpointScanProxy, err error) {
	if resource == nil || err == nil {
		return
	}
	status := &resource.Status.OsokStatus
	servicemanager.RecordErrorOpcRequestID(status, err)
	now := metav1.Now()
	status.UpdatedAt = &now
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, v1.ConditionFalse, "", err.Error(), c.log)
}

func (c *odaPrivateEndpointScanProxyRuntimeClient) projectStatus(
	resource *odav1beta1.OdaPrivateEndpointScanProxy,
	current odasdk.OdaPrivateEndpointScanProxy,
) {
	if resource == nil {
		return
	}
	resource.Status.Id = stringValue(current.Id)
	resource.Status.ScanListenerType = string(current.ScanListenerType)
	resource.Status.Protocol = string(current.Protocol)
	resource.Status.ScanListenerInfos = apiScanListenerInfos(current.ScanListenerInfos)
	resource.Status.LifecycleState = string(current.LifecycleState)
	if current.TimeCreated != nil && !current.TimeCreated.Time.IsZero() {
		resource.Status.TimeCreated = current.TimeCreated.Time.Format(time.RFC3339Nano)
	}

	now := metav1.Now()
	status := &resource.Status.OsokStatus
	if resource.Status.Id != "" {
		status.Ocid = shared.OCID(resource.Status.Id)
		if status.CreatedAt == nil {
			status.CreatedAt = &now
		}
	}
	status.UpdatedAt = &now
}

func (c *odaPrivateEndpointScanProxyRuntimeClient) applyLifecycle(
	resource *odav1beta1.OdaPrivateEndpointScanProxy,
	current odasdk.OdaPrivateEndpointScanProxy,
	fallback shared.OSOKConditionType,
) (servicemanager.OSOKResponse, error) {
	message := odaPrivateEndpointScanProxyLifecycleMessage(current, fallback)
	switch current.LifecycleState {
	case "":
		return c.markCondition(resource, fallback, message, shouldOdaPrivateEndpointScanProxyRequeue(fallback)), nil
	case odasdk.OdaPrivateEndpointScanProxyLifecycleStateActive:
		return c.markCondition(resource, shared.Active, message, false), nil
	case odasdk.OdaPrivateEndpointScanProxyLifecycleStateCreating:
		return c.markLifecycleAsync(resource, string(current.LifecycleState), message, shared.OSOKAsyncPhaseCreate), nil
	case odasdk.OdaPrivateEndpointScanProxyLifecycleStateDeleting:
		return c.markLifecycleAsync(resource, string(current.LifecycleState), message, shared.OSOKAsyncPhaseDelete), nil
	case odasdk.OdaPrivateEndpointScanProxyLifecycleStateFailed:
		return c.markLifecycleAsync(resource, string(current.LifecycleState), message, failedOdaPrivateEndpointScanProxyAsyncPhase(resource)), nil
	case odasdk.OdaPrivateEndpointScanProxyLifecycleStateDeleted:
		return c.markLifecycleAsync(resource, string(current.LifecycleState), message, shared.OSOKAsyncPhaseDelete), nil
	default:
		return c.fail(resource, fmt.Errorf("OdaPrivateEndpointScanProxy lifecycle state %q is not modeled", current.LifecycleState))
	}
}

func failedOdaPrivateEndpointScanProxyAsyncPhase(resource *odav1beta1.OdaPrivateEndpointScanProxy) shared.OSOKAsyncPhase {
	if resource != nil &&
		resource.Status.OsokStatus.Async.Current != nil &&
		resource.Status.OsokStatus.Async.Current.Phase != "" {
		return resource.Status.OsokStatus.Async.Current.Phase
	}
	return shared.OSOKAsyncPhaseCreate
}

func (c *odaPrivateEndpointScanProxyRuntimeClient) markCondition(
	resource *odav1beta1.OdaPrivateEndpointScanProxy,
	condition shared.OSOKConditionType,
	message string,
	shouldRequeue bool,
) servicemanager.OSOKResponse {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	if resource.Status.Id != "" {
		status.Ocid = shared.OCID(resource.Status.Id)
		if status.CreatedAt == nil {
			status.CreatedAt = &now
		}
	}
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(condition)
	if condition == shared.Active {
		servicemanager.ClearAsyncOperation(status)
	}
	conditionStatus := v1.ConditionTrue
	if condition == shared.Failed {
		conditionStatus = v1.ConditionFalse
	}
	*status = util.UpdateOSOKStatusCondition(*status, condition, conditionStatus, "", message, c.log)

	return servicemanager.OSOKResponse{
		IsSuccessful:    condition != shared.Failed,
		ShouldRequeue:   shouldRequeue,
		RequeueDuration: odaPrivateEndpointScanProxyRequeueDuration,
	}
}

func (c *odaPrivateEndpointScanProxyRuntimeClient) markLifecycleAsync(
	resource *odav1beta1.OdaPrivateEndpointScanProxy,
	lifecycleState string,
	message string,
	phase shared.OSOKAsyncPhase,
) servicemanager.OSOKResponse {
	status := &resource.Status.OsokStatus
	current := servicemanager.NewLifecycleAsyncOperation(status, lifecycleState, message, phase)
	if current == nil {
		current = &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           phase,
			RawStatus:       strings.ToUpper(strings.TrimSpace(lifecycleState)),
			NormalizedClass: shared.OSOKAsyncClassPending,
			Message:         message,
		}
	}
	now := metav1.Now()
	if current.UpdatedAt == nil {
		current.UpdatedAt = &now
	}
	projection := servicemanager.ApplyAsyncOperation(status, current, c.log)
	if resource.Status.Id != "" {
		status.Ocid = shared.OCID(resource.Status.Id)
		if status.CreatedAt == nil {
			status.CreatedAt = &now
		}
	}
	return servicemanager.OSOKResponse{
		IsSuccessful:    projection.Condition != shared.Failed,
		ShouldRequeue:   projection.ShouldRequeue,
		RequeueDuration: odaPrivateEndpointScanProxyRequeueDuration,
	}
}

func (c *odaPrivateEndpointScanProxyRuntimeClient) seedWorkRequest(
	resource *odav1beta1.OdaPrivateEndpointScanProxy,
	workRequestID string,
	phase shared.OSOKAsyncPhase,
) {
	if resource == nil || strings.TrimSpace(workRequestID) == "" || phase == "" {
		return
	}
	now := metav1.Now()
	_ = servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           phase,
		WorkRequestID:   strings.TrimSpace(workRequestID),
		NormalizedClass: shared.OSOKAsyncClassPending,
		UpdatedAt:       &now,
	}, c.log)
}

func (c *odaPrivateEndpointScanProxyRuntimeClient) markTerminating(
	resource *odav1beta1.OdaPrivateEndpointScanProxy,
	message string,
) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	current := &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		RawStatus:       string(odasdk.OdaPrivateEndpointScanProxyLifecycleStateDeleting),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         message,
		UpdatedAt:       &now,
	}
	_ = servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, c.log)
}

func (c *odaPrivateEndpointScanProxyRuntimeClient) markDeleted(
	resource *odav1beta1.OdaPrivateEndpointScanProxy,
	message string,
) {
	if resource == nil {
		return
	}
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, c.log)
}

func buildCreateOdaPrivateEndpointScanProxyDetails(
	resource *odav1beta1.OdaPrivateEndpointScanProxy,
) (odasdk.CreateOdaPrivateEndpointScanProxyDetails, error) {
	if resource == nil {
		return odasdk.CreateOdaPrivateEndpointScanProxyDetails{}, fmt.Errorf("OdaPrivateEndpointScanProxy resource is nil")
	}

	scanListenerType, err := odaPrivateEndpointScanProxyScanListenerType(resource.Spec.ScanListenerType)
	if err != nil {
		return odasdk.CreateOdaPrivateEndpointScanProxyDetails{}, err
	}
	protocol, err := odaPrivateEndpointScanProxyProtocol(resource.Spec.Protocol)
	if err != nil {
		return odasdk.CreateOdaPrivateEndpointScanProxyDetails{}, err
	}
	if len(resource.Spec.ScanListenerInfos) == 0 {
		return odasdk.CreateOdaPrivateEndpointScanProxyDetails{}, fmt.Errorf("spec.scanListenerInfos must contain at least one listener")
	}

	return odasdk.CreateOdaPrivateEndpointScanProxyDetails{
		ScanListenerType:  scanListenerType,
		Protocol:          protocol,
		ScanListenerInfos: sdkScanListenerInfos(resource.Spec.ScanListenerInfos),
	}, nil
}

func odaPrivateEndpointScanProxyScanListenerType(value string) (odasdk.OdaPrivateEndpointScanProxyScanListenerTypeEnum, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("spec.scanListenerType is required")
	}
	if enum, ok := odasdk.GetMappingOdaPrivateEndpointScanProxyScanListenerTypeEnum(trimmed); ok {
		return enum, nil
	}
	return "", fmt.Errorf("spec.scanListenerType %q is not supported", value)
}

func odaPrivateEndpointScanProxyProtocol(value string) (odasdk.OdaPrivateEndpointScanProxyProtocolEnum, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("spec.protocol is required")
	}
	if enum, ok := odasdk.GetMappingOdaPrivateEndpointScanProxyProtocolEnum(trimmed); ok {
		return enum, nil
	}
	return "", fmt.Errorf("spec.protocol %q is not supported", value)
}

func sdkScanListenerInfos(in []odav1beta1.OdaPrivateEndpointScanProxyScanListenerInfo) []odasdk.ScanListenerInfo {
	out := make([]odasdk.ScanListenerInfo, 0, len(in))
	for _, item := range in {
		next := odasdk.ScanListenerInfo{}
		if value := strings.TrimSpace(item.ScanListenerFqdn); value != "" {
			next.ScanListenerFqdn = common.String(value)
		}
		if value := strings.TrimSpace(item.ScanListenerIp); value != "" {
			next.ScanListenerIp = common.String(value)
		}
		if item.ScanListenerPort != 0 {
			next.ScanListenerPort = common.Int(item.ScanListenerPort)
		}
		out = append(out, next)
	}
	return out
}

func apiScanListenerInfos(in []odasdk.ScanListenerInfo) []odav1beta1.OdaPrivateEndpointScanProxyScanListenerInfo {
	out := make([]odav1beta1.OdaPrivateEndpointScanProxyScanListenerInfo, 0, len(in))
	for _, item := range in {
		out = append(out, odav1beta1.OdaPrivateEndpointScanProxyScanListenerInfo{
			ScanListenerFqdn: stringValue(item.ScanListenerFqdn),
			ScanListenerIp:   stringValue(item.ScanListenerIp),
			ScanListenerPort: intValue(item.ScanListenerPort),
		})
	}
	return out
}

func validateOdaPrivateEndpointScanProxyCreateOnlyDrift(
	resource *odav1beta1.OdaPrivateEndpointScanProxy,
	current odasdk.OdaPrivateEndpointScanProxy,
) error {
	details, err := buildCreateOdaPrivateEndpointScanProxyDetails(resource)
	if err != nil {
		return err
	}
	if !odaPrivateEndpointScanProxyMatches(
		string(current.ScanListenerType),
		string(current.Protocol),
		current.ScanListenerInfos,
		details,
	) {
		return fmt.Errorf("OdaPrivateEndpointScanProxy does not support updates; spec.scanListenerType, spec.protocol, and spec.scanListenerInfos are create-only")
	}
	return nil
}

func odaPrivateEndpointScanProxySummaryMatches(
	current odasdk.OdaPrivateEndpointScanProxySummary,
	details odasdk.CreateOdaPrivateEndpointScanProxyDetails,
) bool {
	return odaPrivateEndpointScanProxyMatches(
		string(current.ScanListenerType),
		string(current.Protocol),
		current.ScanListenerInfos,
		details,
	)
}

func odaPrivateEndpointScanProxyMatches(
	currentScanListenerType string,
	currentProtocol string,
	currentInfos []odasdk.ScanListenerInfo,
	details odasdk.CreateOdaPrivateEndpointScanProxyDetails,
) bool {
	if !strings.EqualFold(strings.TrimSpace(currentScanListenerType), string(details.ScanListenerType)) {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(currentProtocol), string(details.Protocol)) {
		return false
	}
	return scanListenerInfosEqual(currentInfos, details.ScanListenerInfos)
}

func scanListenerInfosEqual(actual []odasdk.ScanListenerInfo, desired []odasdk.ScanListenerInfo) bool {
	if len(actual) != len(desired) {
		return false
	}
	for i := range actual {
		if strings.TrimSpace(stringValue(actual[i].ScanListenerFqdn)) != strings.TrimSpace(stringValue(desired[i].ScanListenerFqdn)) {
			return false
		}
		if strings.TrimSpace(stringValue(actual[i].ScanListenerIp)) != strings.TrimSpace(stringValue(desired[i].ScanListenerIp)) {
			return false
		}
		if intValue(actual[i].ScanListenerPort) != intValue(desired[i].ScanListenerPort) {
			return false
		}
	}
	return true
}

func odaPrivateEndpointScanProxyFromSummary(summary odasdk.OdaPrivateEndpointScanProxySummary) odasdk.OdaPrivateEndpointScanProxy {
	return odasdk.OdaPrivateEndpointScanProxy{
		Id:                summary.Id,
		ScanListenerType:  summary.ScanListenerType,
		Protocol:          summary.Protocol,
		ScanListenerInfos: summary.ScanListenerInfos,
		LifecycleState:    summary.LifecycleState,
		TimeCreated:       summary.TimeCreated,
	}
}

func odaPrivateEndpointScanProxyParentID(resource *odav1beta1.OdaPrivateEndpointScanProxy) (string, error) {
	if resource == nil {
		return "", fmt.Errorf("OdaPrivateEndpointScanProxy resource is nil")
	}
	value := strings.TrimSpace(resource.Annotations[OdaPrivateEndpointIDAnnotation])
	if value == "" {
		return "", fmt.Errorf("OdaPrivateEndpointScanProxy requires annotation %q with the parent ODA Private Endpoint OCID because v1beta1 spec does not expose odaPrivateEndpointId", OdaPrivateEndpointIDAnnotation)
	}
	return value, nil
}

func odaPrivateEndpointScanProxyCurrentID(resource *odav1beta1.OdaPrivateEndpointScanProxy) string {
	if resource == nil {
		return ""
	}
	if value := strings.TrimSpace(resource.Status.Id); value != "" {
		return value
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func newOdaPrivateEndpointScanProxyParentDriftError(parentID, scanProxyID string, cause error) error {
	message := fmt.Sprintf(
		"OdaPrivateEndpointScanProxy annotation %q is create-only after OCI resource %q is tracked; read under annotated parent %q returned not found, so refusing to clear identity or reconcile against a different parent",
		OdaPrivateEndpointIDAnnotation,
		scanProxyID,
		parentID,
	)
	if cause != nil {
		message = fmt.Sprintf("%s: %v", message, cause)
	}
	return errors.New(message)
}

func newOdaPrivateEndpointScanProxyAmbiguousDeleteNotFoundError(parentID, scanProxyID string, cause error) error {
	message := fmt.Sprintf(
		"OdaPrivateEndpointScanProxy delete confirmation for OCI resource %q is ambiguous because pre-delete read under annotated parent %q returned not found; keeping the finalizer because annotation %q is create-only after tracking",
		scanProxyID,
		parentID,
		OdaPrivateEndpointIDAnnotation,
	)
	if cause != nil {
		message = fmt.Sprintf("%s: %v", message, cause)
	}
	return errors.New(message)
}

func clearOdaPrivateEndpointScanProxyIdentity(resource *odav1beta1.OdaPrivateEndpointScanProxy) {
	if resource == nil {
		return
	}
	resource.Status.Id = ""
	resource.Status.OsokStatus.Ocid = ""
}

func odaPrivateEndpointScanProxyDeletedOrDeleting(state odasdk.OdaPrivateEndpointScanProxyLifecycleStateEnum) bool {
	return state == odasdk.OdaPrivateEndpointScanProxyLifecycleStateDeleted ||
		state == odasdk.OdaPrivateEndpointScanProxyLifecycleStateDeleting
}

func odaPrivateEndpointScanProxyLifecycleMessage(
	current odasdk.OdaPrivateEndpointScanProxy,
	fallback shared.OSOKConditionType,
) string {
	state := strings.TrimSpace(string(current.LifecycleState))
	if state == "" {
		return defaultOdaPrivateEndpointScanProxyConditionMessage(fallback)
	}
	id := strings.TrimSpace(stringValue(current.Id))
	if id == "" {
		id = "resource"
	}
	return fmt.Sprintf("OdaPrivateEndpointScanProxy %s is %s", id, state)
}

func defaultOdaPrivateEndpointScanProxyConditionMessage(condition shared.OSOKConditionType) string {
	switch condition {
	case shared.Provisioning:
		return "OCI OdaPrivateEndpointScanProxy provisioning is in progress"
	case shared.Terminating:
		return "OCI OdaPrivateEndpointScanProxy delete is in progress"
	case shared.Failed:
		return "OCI OdaPrivateEndpointScanProxy reconcile failed"
	default:
		return "OCI OdaPrivateEndpointScanProxy is active"
	}
}

func shouldOdaPrivateEndpointScanProxyRequeue(condition shared.OSOKConditionType) bool {
	return condition == shared.Provisioning || condition == shared.Updating || condition == shared.Terminating
}

func normalizeOdaPrivateEndpointScanProxyOCIError(err error) error {
	var serviceErr common.ServiceError
	if !errors.As(err, &serviceErr) {
		return err
	}
	if _, normalized := errorutil.NewServiceFailureFromResponse(
		serviceErr.GetCode(),
		serviceErr.GetHTTPStatusCode(),
		serviceErr.GetOpcRequestID(),
		serviceErr.GetMessage(),
	); normalized != nil {
		return normalized
	}
	return err
}

func isOdaPrivateEndpointScanProxyReadNotFound(err error) bool {
	return errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound()
}

func isOdaPrivateEndpointScanProxyDeleteNotFound(err error) bool {
	classification := errorutil.ClassifyDeleteError(err)
	return classification.IsUnambiguousNotFound() || classification.IsAuthShapedNotFound()
}

func isOdaPrivateEndpointScanProxyRetryableDeleteConflict(err error) bool {
	return errorutil.ClassifyDeleteError(err).IsConflict()
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

/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package certificate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	loadbalancersdk "github.com/oracle/oci-go-sdk/v65/loadbalancer"
	loadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/loadbalancer/v1beta1"
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
	CertificateLoadBalancerIDAnnotation = "loadbalancer.oracle.com/load-balancer-id"

	certificateLegacyLoadBalancerIDAnnotation = "loadbalancer.oracle.com/loadBalancerId"
	certificateSyntheticIDPrefix              = "loadbalancer/certificate/"
	certificateRequeueDuration                = time.Minute
	certificateDesiredFingerprintLength       = 16
)

var certificateWorkRequestAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens:   []string{string(loadbalancersdk.WorkRequestLifecycleStateAccepted), string(loadbalancersdk.WorkRequestLifecycleStateInProgress)},
	SucceededStatusTokens: []string{string(loadbalancersdk.WorkRequestLifecycleStateSucceeded)},
	FailedStatusTokens:    []string{string(loadbalancersdk.WorkRequestLifecycleStateFailed)},
	CreateActionTokens:    []string{"CreateCertificate"},
	DeleteActionTokens:    []string{"DeleteCertificate"},
}

type certificateRuntimeOCIClient interface {
	CreateCertificate(context.Context, loadbalancersdk.CreateCertificateRequest) (loadbalancersdk.CreateCertificateResponse, error)
	ListCertificates(context.Context, loadbalancersdk.ListCertificatesRequest) (loadbalancersdk.ListCertificatesResponse, error)
	DeleteCertificate(context.Context, loadbalancersdk.DeleteCertificateRequest) (loadbalancersdk.DeleteCertificateResponse, error)
	GetWorkRequest(context.Context, loadbalancersdk.GetWorkRequestRequest) (loadbalancersdk.GetWorkRequestResponse, error)
}

type certificateRuntimeClient struct {
	delegate CertificateServiceClient
	client   certificateRuntimeOCIClient
	log      loggerutil.OSOKLogger
	initErr  error
}

type certificateIdentity struct {
	loadBalancerID   string
	certificateName  string
	loadBalancerHash string
	fingerprint      string
}

type trackedCertificateIdentity struct {
	loadBalancerHash string
	certificateHash  string
	fingerprint      string
}

type certificateWorkRequestView struct {
	Id            string
	Status        string
	OperationType string
	Message       string
}

func init() {
	registerCertificateRuntimeHooksMutator(func(manager *CertificateServiceManager, hooks *CertificateRuntimeHooks) {
		client, initErr := newCertificateSDKClient(manager)
		applyCertificateRuntimeHooks(manager, hooks, client, initErr)
	})
}

func applyCertificateRuntimeHooks(
	manager *CertificateServiceManager,
	hooks *CertificateRuntimeHooks,
	client certificateRuntimeOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	log := loggerutil.OSOKLogger{}
	if manager != nil {
		log = manager.Log
	}

	hooks.Semantics = newCertificateRuntimeSemantics()
	hooks.Async = generatedruntime.AsyncHooks[*loadbalancerv1beta1.Certificate]{
		Adapter: certificateWorkRequestAdapter,
		GetWorkRequest: func(ctx context.Context, workRequestID string) (any, error) {
			if initErr != nil {
				return nil, initErr
			}
			if client == nil {
				return nil, errors.New("Certificate OCI client is nil")
			}
			response, err := client.GetWorkRequest(ctx, loadbalancersdk.GetWorkRequestRequest{
				WorkRequestId: common.String(workRequestID),
			})
			if err != nil {
				return nil, err
			}
			return certificateWorkRequestViewFromResponse(response), nil
		},
	}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate CertificateServiceClient) CertificateServiceClient {
		return &certificateRuntimeClient{
			delegate: delegate,
			client:   client,
			log:      log,
			initErr:  initErr,
		}
	})
}

func newCertificateSDKClient(manager *CertificateServiceManager) (certificateRuntimeOCIClient, error) {
	if manager == nil {
		return nil, errors.New("Certificate service manager is nil")
	}
	client, err := loadbalancersdk.NewLoadBalancerClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, fmt.Errorf("initialize Certificate OCI client: %w", err)
	}
	return client, nil
}

func newCertificateRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "loadbalancer",
		FormalSlug:    "certificate",
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
			ProvisioningStates: []string{string(loadbalancersdk.WorkRequestLifecycleStateAccepted), string(loadbalancersdk.WorkRequestLifecycleStateInProgress)},
			ActiveStates:       []string{string(loadbalancersdk.WorkRequestLifecycleStateSucceeded), "ACTIVE"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(loadbalancersdk.WorkRequestLifecycleStateAccepted), string(loadbalancersdk.WorkRequestLifecycleStateInProgress)},
			TerminalStates: []string{string(loadbalancersdk.WorkRequestLifecycleStateSucceeded), "DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"loadBalancerId", "certificateName"},
		},
		Mutation: generatedruntime.MutationSemantics{
			ForceNew:      []string{"loadBalancerId", "certificateName", "passphrase", "privateKey", "publicCertificate", "caCertificate"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "workrequest",
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func newCertificateServiceClientWithOCIClient(
	client certificateRuntimeOCIClient,
	log loggerutil.OSOKLogger,
	initErr error,
) CertificateServiceClient {
	return &certificateRuntimeClient{
		client:  client,
		log:     log,
		initErr: initErr,
	}
}

func (c *certificateRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *loadbalancerv1beta1.Certificate,
	_ ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if err := c.validateReady(resource); err != nil {
		return c.fail(resource, err), err
	}

	identity, err := resolveCertificateIdentity(resource)
	if err != nil {
		return c.fail(resource, err), err
	}

	if workRequestID, phase := currentCertificateWorkRequest(resource); workRequestID != "" {
		return c.resumeCreateOrUpdateWorkRequest(ctx, resource, identity, workRequestID, phase)
	}

	current, found, err := c.lookupCertificate(ctx, identity)
	if err != nil {
		return c.fail(resource, err), err
	}
	if found {
		if err := validateCertificateCreateOnlyDrift(resource, identity, current); err != nil {
			return c.fail(resource, err), err
		}
		return c.applyActiveCertificate(resource, identity, current, "Certificate is active"), nil
	}

	response, err := c.client.CreateCertificate(ctx, buildCreateCertificateRequest(resource, identity))
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return c.fail(resource, err), err
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	if strings.TrimSpace(stringValue(response.OpcWorkRequestId)) == "" {
		err := errors.New("Certificate create did not return an opc-work-request-id")
		return c.fail(resource, err), err
	}

	return c.markWorkRequest(resource, shared.OSOKAsyncPhaseCreate, stringValue(response.OpcWorkRequestId), "CreateCertificate", string(loadbalancersdk.WorkRequestLifecycleStateAccepted), "OCI create is in progress"), nil
}

func (c *certificateRuntimeClient) Delete(ctx context.Context, resource *loadbalancerv1beta1.Certificate) (bool, error) {
	if err := c.validateReady(resource); err != nil {
		c.markCondition(resource, shared.Failed, v1.ConditionFalse, err.Error())
		return false, err
	}
	if certificateLoadBalancerID(resource) == "" &&
		strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)) == "" &&
		resource.Status.OsokStatus.Async.Current == nil {
		c.markDeleted(resource, "Certificate was never bound to OCI")
		return true, nil
	}

	identity, err := resolveCertificateDeleteIdentity(resource)
	if err != nil {
		c.markCondition(resource, shared.Failed, v1.ConditionFalse, err.Error())
		return false, err
	}

	if workRequestID, phase := currentCertificateWorkRequest(resource); workRequestID != "" {
		if phase == shared.OSOKAsyncPhaseCreate {
			return c.resumeCreateWorkRequestForDelete(ctx, resource, identity, workRequestID)
		}
		return c.resumeDeleteWorkRequest(ctx, resource, identity, workRequestID, phase)
	}

	_, found, err := c.lookupCertificate(ctx, identity)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		c.markCondition(resource, shared.Failed, v1.ConditionFalse, err.Error())
		return false, err
	}
	if !found {
		c.markDeleted(resource, "Certificate is no longer present")
		return true, nil
	}

	return c.startDeleteCertificate(ctx, resource, identity)
}

func (c *certificateRuntimeClient) startDeleteCertificate(
	ctx context.Context,
	resource *loadbalancerv1beta1.Certificate,
	identity certificateIdentity,
) (bool, error) {
	response, err := c.client.DeleteCertificate(ctx, loadbalancersdk.DeleteCertificateRequest{
		LoadBalancerId:  common.String(identity.loadBalancerID),
		CertificateName: common.String(identity.certificateName),
	})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		if isCertificateNotFoundError(err) {
			c.markDeleted(resource, "Certificate is no longer present")
			return true, nil
		}
		c.markCondition(resource, shared.Failed, v1.ConditionFalse, err.Error())
		return false, err
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	if strings.TrimSpace(stringValue(response.OpcWorkRequestId)) == "" {
		err := errors.New("Certificate delete did not return an opc-work-request-id")
		c.markCondition(resource, shared.Failed, v1.ConditionFalse, err.Error())
		return false, err
	}

	c.markWorkRequest(resource, shared.OSOKAsyncPhaseDelete, stringValue(response.OpcWorkRequestId), "DeleteCertificate", string(loadbalancersdk.WorkRequestLifecycleStateAccepted), "OCI delete is in progress")
	return false, nil
}

func (c *certificateRuntimeClient) validateReady(resource *loadbalancerv1beta1.Certificate) error {
	if resource == nil {
		return errors.New("Certificate resource is nil")
	}
	if c.initErr != nil {
		return c.initErr
	}
	if c.client == nil {
		return errors.New("Certificate OCI client is nil")
	}
	return nil
}

func (c *certificateRuntimeClient) resumeCreateOrUpdateWorkRequest(
	ctx context.Context,
	resource *loadbalancerv1beta1.Certificate,
	identity certificateIdentity,
	workRequestID string,
	phase shared.OSOKAsyncPhase,
) (servicemanager.OSOKResponse, error) {
	if phase != "" && phase != shared.OSOKAsyncPhaseCreate {
		err := fmt.Errorf("Certificate reconcile cannot resume %s work request %s from create/update path", phase, workRequestID)
		return c.fail(resource, err), err
	}

	workRequest, err := c.fetchWorkRequest(ctx, workRequestID)
	if err != nil {
		return c.fail(resource, err), err
	}
	currentAsync, err := buildCertificateWorkRequestOperation(&resource.Status.OsokStatus, workRequest, shared.OSOKAsyncPhaseCreate)
	if err != nil {
		return c.fail(resource, err), err
	}

	switch currentAsync.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		return c.applyWorkRequestOperation(resource, currentAsync), nil
	case shared.OSOKAsyncClassSucceeded:
		current, found, err := c.lookupCertificate(ctx, identity)
		if err != nil {
			return c.failWorkRequest(resource, currentAsync, err)
		}
		if !found {
			message := fmt.Sprintf("Certificate create work request %s succeeded; waiting for Certificate %q to become readable", workRequestID, identity.certificateName)
			return c.applyWorkRequestOperationAs(resource, currentAsync, shared.OSOKAsyncClassPending, message), nil
		}
		if err := validateCertificateCreateOnlyDrift(resource, identity, current); err != nil {
			return c.failWorkRequest(resource, currentAsync, err)
		}
		servicemanager.ClearAsyncOperation(&resource.Status.OsokStatus)
		return c.applyActiveCertificate(resource, identity, current, "Certificate create completed"), nil
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		return c.failWorkRequest(resource, currentAsync, fmt.Errorf("Certificate create work request %s finished with status %s", workRequestID, currentAsync.RawStatus))
	default:
		err := fmt.Errorf("Certificate create work request %s projected unsupported async class %s", workRequestID, currentAsync.NormalizedClass)
		return c.failWorkRequest(resource, currentAsync, err)
	}
}

func (c *certificateRuntimeClient) resumeCreateWorkRequestForDelete(
	ctx context.Context,
	resource *loadbalancerv1beta1.Certificate,
	identity certificateIdentity,
	workRequestID string,
) (bool, error) {
	workRequest, err := c.fetchWorkRequest(ctx, workRequestID)
	if err != nil {
		c.markCondition(resource, shared.Failed, v1.ConditionFalse, err.Error())
		return false, err
	}
	currentAsync, err := buildCertificateWorkRequestOperation(&resource.Status.OsokStatus, workRequest, shared.OSOKAsyncPhaseCreate)
	if err != nil {
		c.markCondition(resource, shared.Failed, v1.ConditionFalse, err.Error())
		return false, err
	}

	switch currentAsync.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		message := fmt.Sprintf("Certificate create work request %s is still in progress; waiting before delete", workRequestID)
		c.applyWorkRequestOperationAs(resource, currentAsync, shared.OSOKAsyncClassPending, message)
		return false, nil
	case shared.OSOKAsyncClassSucceeded:
		current, found, err := c.lookupCertificate(ctx, identity)
		if err != nil {
			c.markCondition(resource, shared.Failed, v1.ConditionFalse, err.Error())
			return false, err
		}
		if !found {
			message := fmt.Sprintf("Certificate create work request %s succeeded; waiting for Certificate %q to become readable before delete", workRequestID, identity.certificateName)
			c.applyWorkRequestOperationAs(resource, currentAsync, shared.OSOKAsyncClassPending, message)
			return false, nil
		}
		projectCertificateStatus(resource, identity, current)
		servicemanager.ClearAsyncOperation(&resource.Status.OsokStatus)
		return c.startDeleteCertificate(ctx, resource, identity)
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		_, found, lookupErr := c.lookupCertificate(ctx, identity)
		if lookupErr != nil {
			c.markCondition(resource, shared.Failed, v1.ConditionFalse, lookupErr.Error())
			return false, lookupErr
		}
		if !found {
			c.markDeleted(resource, "Certificate create did not leave a readable OCI resource")
			return true, nil
		}
		return c.startDeleteCertificate(ctx, resource, identity)
	default:
		err := fmt.Errorf("Certificate create work request %s projected unsupported async class %s while deleting", workRequestID, currentAsync.NormalizedClass)
		c.markCondition(resource, shared.Failed, v1.ConditionFalse, err.Error())
		return false, err
	}
}

func (c *certificateRuntimeClient) resumeDeleteWorkRequest(
	ctx context.Context,
	resource *loadbalancerv1beta1.Certificate,
	identity certificateIdentity,
	workRequestID string,
	phase shared.OSOKAsyncPhase,
) (bool, error) {
	if phase != "" && phase != shared.OSOKAsyncPhaseDelete {
		err := fmt.Errorf("Certificate delete cannot resume %s work request %s from delete path", phase, workRequestID)
		c.markCondition(resource, shared.Failed, v1.ConditionFalse, err.Error())
		return false, err
	}

	workRequest, err := c.fetchWorkRequest(ctx, workRequestID)
	if err != nil {
		c.markCondition(resource, shared.Failed, v1.ConditionFalse, err.Error())
		return false, err
	}
	currentAsync, err := buildCertificateWorkRequestOperation(&resource.Status.OsokStatus, workRequest, shared.OSOKAsyncPhaseDelete)
	if err != nil {
		c.markCondition(resource, shared.Failed, v1.ConditionFalse, err.Error())
		return false, err
	}

	switch currentAsync.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		c.applyWorkRequestOperation(resource, currentAsync)
		return false, nil
	case shared.OSOKAsyncClassSucceeded:
		_, found, err := c.lookupCertificate(ctx, identity)
		if err != nil {
			c.markCondition(resource, shared.Failed, v1.ConditionFalse, err.Error())
			return false, err
		}
		if found {
			c.markWorkRequest(resource, shared.OSOKAsyncPhaseDelete, workRequestID, stringValue(workRequest.Type), string(workRequest.LifecycleState), "OCI delete completed; waiting for final confirmation")
			return false, nil
		}
		c.markDeleted(resource, "Certificate delete completed")
		return true, nil
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		err := fmt.Errorf("Certificate delete work request %s finished with status %s", workRequestID, currentAsync.RawStatus)
		c.applyWorkRequestOperation(resource, currentAsync)
		return false, err
	default:
		err := fmt.Errorf("Certificate delete work request %s projected unsupported async class %s", workRequestID, currentAsync.NormalizedClass)
		c.markCondition(resource, shared.Failed, v1.ConditionFalse, err.Error())
		return false, err
	}
}

func (c *certificateRuntimeClient) lookupCertificate(
	ctx context.Context,
	identity certificateIdentity,
) (loadbalancersdk.Certificate, bool, error) {
	response, err := c.client.ListCertificates(ctx, loadbalancersdk.ListCertificatesRequest{
		LoadBalancerId: common.String(identity.loadBalancerID),
	})
	if err != nil {
		if isCertificateNotFoundError(err) {
			return loadbalancersdk.Certificate{}, false, nil
		}
		return loadbalancersdk.Certificate{}, false, err
	}

	var matches []loadbalancersdk.Certificate
	for _, item := range response.Items {
		if strings.TrimSpace(stringValue(item.CertificateName)) == identity.certificateName {
			matches = append(matches, item)
		}
	}
	switch len(matches) {
	case 0:
		return loadbalancersdk.Certificate{}, false, nil
	case 1:
		return matches[0], true, nil
	default:
		return loadbalancersdk.Certificate{}, false, fmt.Errorf("Certificate list returned %d matches for certificateName %q", len(matches), identity.certificateName)
	}
}

func (c *certificateRuntimeClient) fetchWorkRequest(ctx context.Context, workRequestID string) (loadbalancersdk.WorkRequest, error) {
	workRequestID = strings.TrimSpace(workRequestID)
	if workRequestID == "" {
		return loadbalancersdk.WorkRequest{}, errors.New("Certificate work request ID is empty")
	}
	response, err := c.client.GetWorkRequest(ctx, loadbalancersdk.GetWorkRequestRequest{
		WorkRequestId: common.String(workRequestID),
	})
	if err != nil {
		return loadbalancersdk.WorkRequest{}, err
	}
	return response.WorkRequest, nil
}

func (c *certificateRuntimeClient) applyActiveCertificate(
	resource *loadbalancerv1beta1.Certificate,
	identity certificateIdentity,
	current loadbalancersdk.Certificate,
	message string,
) servicemanager.OSOKResponse {
	projectCertificateStatus(resource, identity, current)
	servicemanager.ClearAsyncOperation(&resource.Status.OsokStatus)
	c.markCondition(resource, shared.Active, v1.ConditionTrue, message)
	return servicemanager.OSOKResponse{IsSuccessful: true}
}

func (c *certificateRuntimeClient) fail(
	resource *loadbalancerv1beta1.Certificate,
	err error,
) servicemanager.OSOKResponse {
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		c.markCondition(resource, shared.Failed, v1.ConditionFalse, err.Error())
	}
	return servicemanager.OSOKResponse{IsSuccessful: false}
}

func (c *certificateRuntimeClient) failWorkRequest(
	resource *loadbalancerv1beta1.Certificate,
	current *shared.OSOKAsyncOperation,
	err error,
) (servicemanager.OSOKResponse, error) {
	if current != nil {
		current.NormalizedClass = shared.OSOKAsyncClassFailed
		if err != nil {
			current.Message = err.Error()
		}
		return c.applyWorkRequestOperation(resource, current), err
	}
	return c.fail(resource, err), err
}

func (c *certificateRuntimeClient) markWorkRequest(
	resource *loadbalancerv1beta1.Certificate,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	operationType string,
	rawStatus string,
	message string,
) servicemanager.OSOKResponse {
	current, err := servicemanager.BuildWorkRequestAsyncOperation(&resource.Status.OsokStatus, certificateWorkRequestAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        rawStatus,
		RawAction:        operationType,
		RawOperationType: operationType,
		WorkRequestID:    workRequestID,
		Message:          message,
		FallbackPhase:    phase,
	})
	if err != nil {
		return c.fail(resource, err)
	}
	return c.applyWorkRequestOperation(resource, current)
}

func (c *certificateRuntimeClient) applyWorkRequestOperation(
	resource *loadbalancerv1beta1.Certificate,
	current *shared.OSOKAsyncOperation,
) servicemanager.OSOKResponse {
	if current != nil && current.WorkRequestID == "" && resource.Status.OsokStatus.Async.Current != nil {
		current.WorkRequestID = resource.Status.OsokStatus.Async.Current.WorkRequestID
	}

	now := metav1.Now()
	if current != nil && current.UpdatedAt == nil {
		current.UpdatedAt = &now
	}
	if resource.Status.OsokStatus.Ocid == "" {
		if identity, err := resolveCertificateIdentity(resource); err == nil {
			resource.Status.OsokStatus.Ocid = certificateSyntheticOCID(identity)
		}
	}
	if resource.Status.OsokStatus.CreatedAt == nil && resource.Status.OsokStatus.Ocid != "" {
		resource.Status.OsokStatus.CreatedAt = &now
	}

	projection := servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, c.log)
	return servicemanager.OSOKResponse{
		IsSuccessful:    projection.Condition != shared.Failed,
		ShouldRequeue:   projection.ShouldRequeue,
		RequeueDuration: certificateRequeueDuration,
	}
}

func (c *certificateRuntimeClient) applyWorkRequestOperationAs(
	resource *loadbalancerv1beta1.Certificate,
	current *shared.OSOKAsyncOperation,
	class shared.OSOKAsyncNormalizedClass,
	message string,
) servicemanager.OSOKResponse {
	if current == nil {
		return c.applyWorkRequestOperation(resource, current)
	}
	next := *current
	next.NormalizedClass = class
	next.Message = strings.TrimSpace(message)
	next.UpdatedAt = nil
	return c.applyWorkRequestOperation(resource, &next)
}

func (c *certificateRuntimeClient) markCondition(
	resource *loadbalancerv1beta1.Certificate,
	condition shared.OSOKConditionType,
	status v1.ConditionStatus,
	message string,
) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	resource.Status.OsokStatus.UpdatedAt = &now
	if condition == shared.Active && resource.Status.OsokStatus.CreatedAt == nil && resource.Status.OsokStatus.Ocid != "" {
		resource.Status.OsokStatus.CreatedAt = &now
	}
	resource.Status.OsokStatus.Message = strings.TrimSpace(message)
	resource.Status.OsokStatus.Reason = string(condition)
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, condition, status, "", message, c.log)
}

func (c *certificateRuntimeClient) markDeleted(resource *loadbalancerv1beta1.Certificate, message string) {
	servicemanager.ClearAsyncOperation(&resource.Status.OsokStatus)
	now := metav1.Now()
	resource.Status.OsokStatus.DeletedAt = &now
	resource.Status.OsokStatus.UpdatedAt = &now
	resource.Status.OsokStatus.Message = strings.TrimSpace(message)
	resource.Status.OsokStatus.Reason = string(shared.Terminating)
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", message, c.log)
}

func resolveCertificateIdentity(resource *loadbalancerv1beta1.Certificate) (certificateIdentity, error) {
	if resource == nil {
		return certificateIdentity{}, errors.New("Certificate resource is nil")
	}
	loadBalancerID := certificateLoadBalancerID(resource)
	if loadBalancerID == "" {
		return certificateIdentity{}, fmt.Errorf("Certificate requires metadata annotation %q with the parent load balancer OCID because spec.loadBalancerId is not available", CertificateLoadBalancerIDAnnotation)
	}
	certificateName := strings.TrimSpace(resource.Spec.CertificateName)
	if certificateName == "" {
		return certificateIdentity{}, errors.New("Certificate spec.certificateName is empty")
	}
	identity := certificateIdentity{
		loadBalancerID:   loadBalancerID,
		certificateName:  certificateName,
		loadBalancerHash: shortHash(loadBalancerID),
		fingerprint:      certificateDesiredFingerprint(resource),
	}
	if err := validateTrackedCertificateIdentity(resource, identity); err != nil {
		return certificateIdentity{}, err
	}
	return identity, nil
}

func resolveCertificateDeleteIdentity(resource *loadbalancerv1beta1.Certificate) (certificateIdentity, error) {
	identity, err := resolveCertificateIdentity(resource)
	if err == nil {
		return identity, nil
	}
	if strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)) == "" {
		name := firstNonEmptyTrim(resource.Status.CertificateName, resource.Spec.CertificateName)
		if name == "" {
			return certificateIdentity{}, nil
		}
	}
	return certificateIdentity{}, err
}

func validateTrackedCertificateIdentity(resource *loadbalancerv1beta1.Certificate, identity certificateIdentity) error {
	tracked, ok := parseTrackedCertificateIdentity(resource.Status.OsokStatus.Ocid)
	if !ok {
		return nil
	}
	if tracked.loadBalancerHash != "" && tracked.loadBalancerHash != identity.loadBalancerHash {
		return fmt.Errorf("Certificate create-only parent load balancer annotation %q changed; create a replacement resource instead", CertificateLoadBalancerIDAnnotation)
	}
	if tracked.certificateHash != "" && tracked.certificateHash != shortHash(identity.certificateName) {
		return errors.New("Certificate create-only certificateName changed; create a replacement resource instead")
	}
	if tracked.fingerprint != "" && tracked.fingerprint != identity.fingerprint {
		return errors.New("Certificate create-only certificate inputs changed; create a replacement resource instead")
	}
	return nil
}

func validateCertificateCreateOnlyDrift(
	resource *loadbalancerv1beta1.Certificate,
	identity certificateIdentity,
	current loadbalancersdk.Certificate,
) error {
	if err := validateTrackedCertificateIdentity(resource, identity); err != nil {
		return err
	}
	if wanted := strings.TrimSpace(resource.Spec.PublicCertificate); wanted != "" && wanted != strings.TrimSpace(stringValue(current.PublicCertificate)) {
		return errors.New("Certificate create-only publicCertificate changed; create a replacement resource instead")
	}
	if wanted := strings.TrimSpace(resource.Spec.CaCertificate); wanted != "" && wanted != strings.TrimSpace(stringValue(current.CaCertificate)) {
		return errors.New("Certificate create-only caCertificate changed; create a replacement resource instead")
	}
	return nil
}

func projectCertificateStatus(
	resource *loadbalancerv1beta1.Certificate,
	identity certificateIdentity,
	current loadbalancersdk.Certificate,
) {
	resource.Status.CertificateName = firstNonEmptyTrim(stringValue(current.CertificateName), identity.certificateName)
	resource.Status.PublicCertificate = strings.TrimSpace(stringValue(current.PublicCertificate))
	resource.Status.CaCertificate = strings.TrimSpace(stringValue(current.CaCertificate))
	resource.Status.OsokStatus.Ocid = certificateSyntheticOCID(identity)
}

func buildCreateCertificateRequest(
	resource *loadbalancerv1beta1.Certificate,
	identity certificateIdentity,
) loadbalancersdk.CreateCertificateRequest {
	request := loadbalancersdk.CreateCertificateRequest{
		LoadBalancerId: common.String(identity.loadBalancerID),
		CreateCertificateDetails: loadbalancersdk.CreateCertificateDetails{
			CertificateName: common.String(identity.certificateName),
		},
	}
	if value := strings.TrimSpace(resource.Spec.Passphrase); value != "" {
		request.Passphrase = common.String(value)
	}
	if value := strings.TrimSpace(resource.Spec.PrivateKey); value != "" {
		request.PrivateKey = common.String(value)
	}
	if value := strings.TrimSpace(resource.Spec.PublicCertificate); value != "" {
		request.PublicCertificate = common.String(value)
	}
	if value := strings.TrimSpace(resource.Spec.CaCertificate); value != "" {
		request.CaCertificate = common.String(value)
	}
	return request
}

func buildCertificateWorkRequestOperation(
	status *shared.OSOKStatus,
	workRequest loadbalancersdk.WorkRequest,
	fallbackPhase shared.OSOKAsyncPhase,
) (*shared.OSOKAsyncOperation, error) {
	return servicemanager.BuildWorkRequestAsyncOperation(status, certificateWorkRequestAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(workRequest.LifecycleState),
		RawAction:        stringValue(workRequest.Type),
		RawOperationType: stringValue(workRequest.Type),
		WorkRequestID:    stringValue(workRequest.Id),
		Message:          stringValue(workRequest.Message),
		FallbackPhase:    fallbackPhase,
	})
}

func currentCertificateWorkRequest(resource *loadbalancerv1beta1.Certificate) (string, shared.OSOKAsyncPhase) {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return "", ""
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		return "", ""
	}
	return strings.TrimSpace(current.WorkRequestID), current.Phase
}

func certificateLoadBalancerID(resource *loadbalancerv1beta1.Certificate) string {
	if resource == nil {
		return ""
	}
	return annotationValue(resource.Annotations, CertificateLoadBalancerIDAnnotation, certificateLegacyLoadBalancerIDAnnotation)
}

func annotationValue(annotations map[string]string, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(annotations[key]); value != "" {
			return value
		}
	}
	return ""
}

func certificateSyntheticOCID(identity certificateIdentity) shared.OCID {
	nameHash := shortHash(identity.certificateName)
	return shared.OCID(fmt.Sprintf("%s%s/%s/%s", certificateSyntheticIDPrefix, identity.loadBalancerHash, nameHash, identity.fingerprint))
}

func parseTrackedCertificateIdentity(value shared.OCID) (trackedCertificateIdentity, bool) {
	raw := strings.TrimSpace(string(value))
	if raw == "" || !strings.HasPrefix(raw, certificateSyntheticIDPrefix) {
		return trackedCertificateIdentity{}, false
	}
	parts := strings.Split(strings.TrimPrefix(raw, certificateSyntheticIDPrefix), "/")
	if len(parts) != 3 {
		return trackedCertificateIdentity{}, false
	}
	return trackedCertificateIdentity{
		loadBalancerHash: strings.TrimSpace(parts[0]),
		certificateHash:  strings.TrimSpace(parts[1]),
		fingerprint:      strings.TrimSpace(parts[2]),
	}, true
}

func certificateDesiredFingerprint(resource *loadbalancerv1beta1.Certificate) string {
	if resource == nil {
		return ""
	}
	sum := sha256.New()
	for _, value := range []string{
		resource.Spec.CertificateName,
		resource.Spec.Passphrase,
		resource.Spec.PrivateKey,
		resource.Spec.PublicCertificate,
		resource.Spec.CaCertificate,
	} {
		sum.Write([]byte(strings.TrimSpace(value)))
		sum.Write([]byte{0})
	}
	return hex.EncodeToString(sum.Sum(nil))[:certificateDesiredFingerprintLength]
}

func shortHash(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:])[:16]
}

func certificateWorkRequestViewFromResponse(response loadbalancersdk.GetWorkRequestResponse) certificateWorkRequestView {
	return certificateWorkRequestView{
		Id:            stringValue(response.WorkRequest.Id),
		Status:        string(response.WorkRequest.LifecycleState),
		OperationType: stringValue(response.WorkRequest.Type),
		Message:       stringValue(response.WorkRequest.Message),
	}
}

func isCertificateNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	classification := errorutil.ClassifyDeleteError(err)
	return classification.IsUnambiguousNotFound()
}

func firstNonEmptyTrim(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

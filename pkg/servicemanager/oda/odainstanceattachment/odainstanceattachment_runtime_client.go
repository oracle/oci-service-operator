/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package odainstanceattachment

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
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

const odaInstanceAttachmentOdaInstanceIDAnnotation = "oda.oracle.com/oda-instance-id"

var odaInstanceAttachmentWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(odasdk.WorkRequestStatusAccepted),
		string(odasdk.WorkRequestStatusInProgress),
		string(odasdk.WorkRequestStatusCanceling),
	},
	SucceededStatusTokens: []string{string(odasdk.WorkRequestStatusSucceeded)},
	FailedStatusTokens:    []string{string(odasdk.WorkRequestStatusFailed)},
	CanceledStatusTokens:  []string{string(odasdk.WorkRequestStatusCanceled)},
	CreateActionTokens:    []string{string(odasdk.WorkRequestRequestActionCreateOdaInstanceAttachment)},
	UpdateActionTokens:    []string{string(odasdk.WorkRequestRequestActionUpdateOdaInstanceAttachment)},
	DeleteActionTokens:    []string{string(odasdk.WorkRequestRequestActionDeleteOdaInstanceAttachment)},
}

type odaInstanceAttachmentOCIClient interface {
	CreateOdaInstanceAttachment(context.Context, odasdk.CreateOdaInstanceAttachmentRequest) (odasdk.CreateOdaInstanceAttachmentResponse, error)
	GetOdaInstanceAttachment(context.Context, odasdk.GetOdaInstanceAttachmentRequest) (odasdk.GetOdaInstanceAttachmentResponse, error)
	ListOdaInstanceAttachments(context.Context, odasdk.ListOdaInstanceAttachmentsRequest) (odasdk.ListOdaInstanceAttachmentsResponse, error)
	UpdateOdaInstanceAttachment(context.Context, odasdk.UpdateOdaInstanceAttachmentRequest) (odasdk.UpdateOdaInstanceAttachmentResponse, error)
	DeleteOdaInstanceAttachment(context.Context, odasdk.DeleteOdaInstanceAttachmentRequest) (odasdk.DeleteOdaInstanceAttachmentResponse, error)
	GetWorkRequest(context.Context, odasdk.GetWorkRequestRequest) (odasdk.GetWorkRequestResponse, error)
}

type odaInstanceAttachmentRuntimeClient struct {
	delegate OdaInstanceAttachmentServiceClient
	hooks    OdaInstanceAttachmentRuntimeHooks
	log      loggerutil.OSOKLogger
	client   odaInstanceAttachmentOCIClient
	initErr  error
}

var _ OdaInstanceAttachmentServiceClient = (*odaInstanceAttachmentRuntimeClient)(nil)

func init() {
	registerOdaInstanceAttachmentRuntimeHooksMutator(func(manager *OdaInstanceAttachmentServiceManager, hooks *OdaInstanceAttachmentRuntimeHooks) {
		client, initErr := newOdaInstanceAttachmentSDKClient(manager)
		applyOdaInstanceAttachmentRuntimeHooks(manager, hooks, client, initErr)
	})
}

func newOdaInstanceAttachmentSDKClient(manager *OdaInstanceAttachmentServiceManager) (odaInstanceAttachmentOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("OdaInstanceAttachment service manager is nil")
	}
	client, err := odasdk.NewOdaClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyOdaInstanceAttachmentRuntimeHooks(
	manager *OdaInstanceAttachmentServiceManager,
	hooks *OdaInstanceAttachmentRuntimeHooks,
	client odaInstanceAttachmentOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newOdaInstanceAttachmentRuntimeSemantics()
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate OdaInstanceAttachmentServiceClient) OdaInstanceAttachmentServiceClient {
		runtimeClient := &odaInstanceAttachmentRuntimeClient{
			delegate: delegate,
			hooks:    *hooks,
			client:   client,
			initErr:  initErr,
		}
		if manager != nil {
			runtimeClient.log = manager.Log
		}
		return runtimeClient
	})
}

func newOdaInstanceAttachmentServiceClientWithOCIClient(log loggerutil.OSOKLogger, client odaInstanceAttachmentOCIClient) OdaInstanceAttachmentServiceClient {
	manager := &OdaInstanceAttachmentServiceManager{Log: log}
	hooks := newOdaInstanceAttachmentRuntimeHooksWithOCIClient(client)
	applyOdaInstanceAttachmentRuntimeHooks(manager, &hooks, client, nil)
	delegate := defaultOdaInstanceAttachmentServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*odav1beta1.OdaInstanceAttachment](
			buildOdaInstanceAttachmentGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapOdaInstanceAttachmentGeneratedClient(hooks, delegate)
}

func newOdaInstanceAttachmentRuntimeHooksWithOCIClient(client odaInstanceAttachmentOCIClient) OdaInstanceAttachmentRuntimeHooks {
	return OdaInstanceAttachmentRuntimeHooks{
		Create: runtimeOperationHooks[odasdk.CreateOdaInstanceAttachmentRequest, odasdk.CreateOdaInstanceAttachmentResponse]{
			Call: func(ctx context.Context, request odasdk.CreateOdaInstanceAttachmentRequest) (odasdk.CreateOdaInstanceAttachmentResponse, error) {
				return client.CreateOdaInstanceAttachment(ctx, request)
			},
		},
		Get: runtimeOperationHooks[odasdk.GetOdaInstanceAttachmentRequest, odasdk.GetOdaInstanceAttachmentResponse]{
			Call: func(ctx context.Context, request odasdk.GetOdaInstanceAttachmentRequest) (odasdk.GetOdaInstanceAttachmentResponse, error) {
				return client.GetOdaInstanceAttachment(ctx, request)
			},
		},
		List: runtimeOperationHooks[odasdk.ListOdaInstanceAttachmentsRequest, odasdk.ListOdaInstanceAttachmentsResponse]{
			Call: func(ctx context.Context, request odasdk.ListOdaInstanceAttachmentsRequest) (odasdk.ListOdaInstanceAttachmentsResponse, error) {
				return client.ListOdaInstanceAttachments(ctx, request)
			},
		},
		Update: runtimeOperationHooks[odasdk.UpdateOdaInstanceAttachmentRequest, odasdk.UpdateOdaInstanceAttachmentResponse]{
			Call: func(ctx context.Context, request odasdk.UpdateOdaInstanceAttachmentRequest) (odasdk.UpdateOdaInstanceAttachmentResponse, error) {
				return client.UpdateOdaInstanceAttachment(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[odasdk.DeleteOdaInstanceAttachmentRequest, odasdk.DeleteOdaInstanceAttachmentResponse]{
			Call: func(ctx context.Context, request odasdk.DeleteOdaInstanceAttachmentRequest) (odasdk.DeleteOdaInstanceAttachmentResponse, error) {
				return client.DeleteOdaInstanceAttachment(ctx, request)
			},
		},
	}
}

func newOdaInstanceAttachmentRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "oda",
		FormalSlug:    "odainstanceattachment",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "workrequest",
			Runtime:              "handwritten",
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
			ProvisioningStates: []string{"ATTACHING"},
			ActiveStates:       []string{"ACTIVE"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:        "required",
			PendingStates: []string{"DETACHING"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"odaInstanceId", "attachToId", "attachmentType", "owner.ownerServiceName", "owner.ownerServiceTenancy"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{"attachmentMetadata", "restrictedOperations", "owner", "freeformTags", "definedTags"},
			ForceNew: []string{
				"odaInstanceId",
				"attachToId",
				"attachmentType",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "resource-local", EntityType: "OdaInstanceAttachment", Action: "CreateOdaInstanceAttachment"}},
			Update: []generatedruntime.Hook{{Helper: "resource-local", EntityType: "OdaInstanceAttachment", Action: "UpdateOdaInstanceAttachment"}},
			Delete: []generatedruntime.Hook{{Helper: "resource-local", EntityType: "OdaInstanceAttachment", Action: "DeleteOdaInstanceAttachment"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetOdaInstanceAttachment/ListOdaInstanceAttachments",
			Hooks: []generatedruntime.Hook{
				{Helper: "resource-local", EntityType: "WorkRequest", Action: "GetWorkRequest"},
				{Helper: "resource-local", EntityType: "OdaInstanceAttachment", Action: "GetOdaInstanceAttachment"},
			},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetOdaInstanceAttachment",
			Hooks: []generatedruntime.Hook{
				{Helper: "resource-local", EntityType: "WorkRequest", Action: "GetWorkRequest"},
				{Helper: "resource-local", EntityType: "OdaInstanceAttachment", Action: "GetOdaInstanceAttachment"},
			},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetOdaInstanceAttachment/ListOdaInstanceAttachments confirm-delete",
			Hooks: []generatedruntime.Hook{
				{Helper: "resource-local", EntityType: "WorkRequest", Action: "GetWorkRequest"},
				{Helper: "resource-local", EntityType: "OdaInstanceAttachment", Action: "GetOdaInstanceAttachment"},
			},
		},
		Unsupported: []generatedruntime.UnsupportedSemantic{
			{
				Category:      "direct-generatedruntime-parent-and-workrequest-shape",
				StopCondition: "OdaInstanceId is supplied from resource-local annotation/status and create/update/delete return only opc-work-request-id; use this resource-local wrapper until generatedruntime supports that direct shape.",
			},
		},
	}
}

func (c *odaInstanceAttachmentRuntimeClient) CreateOrUpdate(ctx context.Context, resource *odav1beta1.OdaInstanceAttachment, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("OdaInstanceAttachment resource is nil")
	}
	if c == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("OdaInstanceAttachment runtime client is not configured")
	}
	if c.initErr != nil {
		return c.fail(resource, fmt.Errorf("initialize OdaInstanceAttachment OCI client: %w", c.initErr))
	}
	if c.client == nil {
		return c.fail(resource, fmt.Errorf("OdaInstanceAttachment OCI client is not configured"))
	}

	odaInstanceID, err := odaInstanceAttachmentOdaInstanceID(resource)
	if err != nil {
		return c.fail(resource, err)
	}
	if err := validateOdaInstanceAttachmentTrackedParent(resource, odaInstanceID); err != nil {
		return c.fail(resource, err)
	}

	if workRequestID, phase := currentOdaInstanceAttachmentWorkRequest(resource, ""); workRequestID != "" && phase != shared.OSOKAsyncPhaseDelete {
		return c.applyWorkRequest(ctx, resource, odaInstanceID, workRequestID, phase, currentOdaInstanceAttachmentID(resource))
	}

	current, found, err := c.resolveCurrent(ctx, resource, odaInstanceID)
	if err != nil {
		return c.fail(resource, err)
	}
	if !found {
		return c.create(ctx, resource, odaInstanceID)
	}

	if isOdaInstanceAttachmentRetryableLifecycle(current.LifecycleState) {
		return c.finishWithLifecycle(resource, current), nil
	}
	if state := normalizedOdaInstanceAttachmentLifecycle(current.LifecycleState); state != "" && state != string(odasdk.OdaInstanceAttachmentLifecycleStateActive) {
		return c.finishWithLifecycle(resource, current), nil
	}
	if err := validateOdaInstanceAttachmentCreateOnlyDrift(resource, odaInstanceID, current); err != nil {
		return c.fail(resource, err)
	}

	updateDetails, updateNeeded := buildOdaInstanceAttachmentUpdateDetails(resource, current)
	if updateNeeded {
		return c.update(ctx, resource, odaInstanceID, odaInstanceAttachmentIDFromSDK(current), updateDetails)
	}

	return c.finishWithLifecycle(resource, current), nil
}

func (c *odaInstanceAttachmentRuntimeClient) Delete(ctx context.Context, resource *odav1beta1.OdaInstanceAttachment) (bool, error) {
	if resource == nil {
		return false, fmt.Errorf("OdaInstanceAttachment resource is nil")
	}
	if c == nil {
		return false, fmt.Errorf("OdaInstanceAttachment runtime client is not configured")
	}
	if c.initErr != nil {
		return false, c.markFailure(resource, fmt.Errorf("initialize OdaInstanceAttachment OCI client: %w", c.initErr))
	}
	if c.client == nil {
		return false, c.markFailure(resource, fmt.Errorf("OdaInstanceAttachment OCI client is not configured"))
	}

	odaInstanceID, parentErr := odaInstanceAttachmentOdaInstanceID(resource)
	currentID := currentOdaInstanceAttachmentID(resource)
	if parentErr != nil {
		if currentID == "" {
			c.markDeleted(resource, "OdaInstanceAttachment has no tracked OCI identity")
			return true, nil
		}
		return false, c.markFailure(resource, parentErr)
	}

	if workRequestID, phase := currentOdaInstanceAttachmentWorkRequest(resource, shared.OSOKAsyncPhaseDelete); workRequestID != "" {
		if phase == shared.OSOKAsyncPhaseDelete {
			return c.confirmDeleteWorkRequest(ctx, resource, odaInstanceID, workRequestID, currentID)
		}
		if _, err := c.applyWorkRequest(ctx, resource, odaInstanceID, workRequestID, phase, currentID); err != nil {
			return false, err
		}
		return false, nil
	}

	current, found, err := c.resolveCurrent(ctx, resource, odaInstanceID)
	if err != nil {
		return false, c.markFailure(resource, err)
	}
	if !found {
		c.markDeleted(resource, "OCI OdaInstanceAttachment is already deleted")
		return true, nil
	}

	currentID = odaInstanceAttachmentIDFromSDK(current)
	if currentID == "" {
		err := fmt.Errorf("OdaInstanceAttachment delete could not resolve OCI resource ID")
		return false, c.markFailure(resource, err)
	}

	if normalizedOdaInstanceAttachmentLifecycle(current.LifecycleState) == string(odasdk.OdaInstanceAttachmentLifecycleStateDetaching) {
		c.projectOdaInstanceAttachmentStatus(resource, current)
		c.markCondition(resource, shared.Terminating, string(current.LifecycleState), "OdaInstanceAttachment delete is in progress", true)
		return false, nil
	}

	response, err := c.hooks.Delete.Call(ctx, odasdk.DeleteOdaInstanceAttachmentRequest{
		OdaInstanceId: common.String(odaInstanceID),
		AttachmentId:  common.String(currentID),
	})
	if err != nil {
		if isOdaInstanceAttachmentReadNotFound(err) {
			c.markDeleted(resource, "OCI OdaInstanceAttachment is already deleted")
			return true, nil
		}
		return false, c.markFailure(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)

	workRequestID := stringValue(response.OpcWorkRequestId)
	if workRequestID != "" {
		return c.confirmDeleteWorkRequest(ctx, resource, odaInstanceID, workRequestID, currentID)
	}
	return c.confirmDeleted(ctx, resource, odaInstanceID, currentID)
}

func (c *odaInstanceAttachmentRuntimeClient) create(
	ctx context.Context,
	resource *odav1beta1.OdaInstanceAttachment,
	odaInstanceID string,
) (servicemanager.OSOKResponse, error) {
	response, err := c.hooks.Create.Call(ctx, odasdk.CreateOdaInstanceAttachmentRequest{
		OdaInstanceId:                      common.String(odaInstanceID),
		CreateOdaInstanceAttachmentDetails: buildOdaInstanceAttachmentCreateDetails(resource.Spec),
	})
	if err != nil {
		return c.fail(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)

	workRequestID := stringValue(response.OpcWorkRequestId)
	if workRequestID == "" {
		return c.resolveAfterWrite(ctx, resource, odaInstanceID, shared.OSOKAsyncPhaseCreate, "")
	}
	return c.applyWorkRequest(ctx, resource, odaInstanceID, workRequestID, shared.OSOKAsyncPhaseCreate, "")
}

func (c *odaInstanceAttachmentRuntimeClient) update(
	ctx context.Context,
	resource *odav1beta1.OdaInstanceAttachment,
	odaInstanceID string,
	attachmentID string,
	details odasdk.UpdateOdaInstanceAttachmentDetails,
) (servicemanager.OSOKResponse, error) {
	if attachmentID == "" {
		return c.fail(resource, fmt.Errorf("OdaInstanceAttachment update could not resolve OCI resource ID"))
	}
	response, err := c.hooks.Update.Call(ctx, odasdk.UpdateOdaInstanceAttachmentRequest{
		OdaInstanceId:                      common.String(odaInstanceID),
		AttachmentId:                       common.String(attachmentID),
		UpdateOdaInstanceAttachmentDetails: details,
	})
	if err != nil {
		return c.fail(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)

	workRequestID := stringValue(response.OpcWorkRequestId)
	if workRequestID == "" {
		return c.resolveAfterWrite(ctx, resource, odaInstanceID, shared.OSOKAsyncPhaseUpdate, attachmentID)
	}
	return c.applyWorkRequest(ctx, resource, odaInstanceID, workRequestID, shared.OSOKAsyncPhaseUpdate, attachmentID)
}

func (c *odaInstanceAttachmentRuntimeClient) resolveCurrent(
	ctx context.Context,
	resource *odav1beta1.OdaInstanceAttachment,
	odaInstanceID string,
) (odasdk.OdaInstanceAttachment, bool, error) {
	if currentID := currentOdaInstanceAttachmentID(resource); currentID != "" {
		current, err := c.read(ctx, odaInstanceID, currentID)
		if err != nil {
			if isOdaInstanceAttachmentReadNotFound(err) {
				return odasdk.OdaInstanceAttachment{}, false, nil
			}
			return odasdk.OdaInstanceAttachment{}, false, err
		}
		return current, true, nil
	}
	return c.resolveByList(ctx, resource, odaInstanceID)
}

func (c *odaInstanceAttachmentRuntimeClient) resolveByList(
	ctx context.Context,
	resource *odav1beta1.OdaInstanceAttachment,
	odaInstanceID string,
) (odasdk.OdaInstanceAttachment, bool, error) {
	request := odasdk.ListOdaInstanceAttachmentsRequest{
		OdaInstanceId:        common.String(odaInstanceID),
		IncludeOwnerMetadata: common.Bool(true),
	}

	var matchedID string
	for {
		response, err := c.hooks.List.Call(ctx, request)
		if err != nil {
			return odasdk.OdaInstanceAttachment{}, false, err
		}

		for _, item := range response.Items {
			if !odaInstanceAttachmentSummaryMatches(resource, odaInstanceID, item) {
				continue
			}
			itemID := stringValue(item.Id)
			if itemID == "" {
				continue
			}
			if matchedID != "" && matchedID != itemID {
				return odasdk.OdaInstanceAttachment{}, false, fmt.Errorf(
					"multiple OCI OdaInstanceAttachments matched attachToId %q, attachmentType %q, and owner %q/%q",
					resource.Spec.AttachToId,
					resource.Spec.AttachmentType,
					resource.Spec.Owner.OwnerServiceName,
					resource.Spec.Owner.OwnerServiceTenancy,
				)
			}
			matchedID = itemID
		}

		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			break
		}
		request.Page = response.OpcNextPage
	}

	if matchedID == "" {
		return odasdk.OdaInstanceAttachment{}, false, nil
	}
	current, err := c.read(ctx, odaInstanceID, matchedID)
	if err != nil {
		if isOdaInstanceAttachmentReadNotFound(err) {
			return odasdk.OdaInstanceAttachment{}, false, nil
		}
		return odasdk.OdaInstanceAttachment{}, false, err
	}
	return current, true, nil
}

func (c *odaInstanceAttachmentRuntimeClient) read(ctx context.Context, odaInstanceID string, attachmentID string) (odasdk.OdaInstanceAttachment, error) {
	response, err := c.hooks.Get.Call(ctx, odasdk.GetOdaInstanceAttachmentRequest{
		OdaInstanceId:        common.String(odaInstanceID),
		AttachmentId:         common.String(attachmentID),
		IncludeOwnerMetadata: common.Bool(true),
	})
	if err != nil {
		return odasdk.OdaInstanceAttachment{}, err
	}
	return response.OdaInstanceAttachment, nil
}

func (c *odaInstanceAttachmentRuntimeClient) getWorkRequest(ctx context.Context, workRequestID string) (odasdk.WorkRequest, error) {
	if c.initErr != nil {
		return odasdk.WorkRequest{}, fmt.Errorf("initialize OdaInstanceAttachment OCI client: %w", c.initErr)
	}
	if c.client == nil {
		return odasdk.WorkRequest{}, fmt.Errorf("OdaInstanceAttachment OCI client is not configured")
	}
	response, err := c.client.GetWorkRequest(ctx, odasdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return odasdk.WorkRequest{}, err
	}
	return response.WorkRequest, nil
}

func (c *odaInstanceAttachmentRuntimeClient) applyWorkRequest(
	ctx context.Context,
	resource *odav1beta1.OdaInstanceAttachment,
	odaInstanceID string,
	workRequestID string,
	phase shared.OSOKAsyncPhase,
	fallbackAttachmentID string,
) (servicemanager.OSOKResponse, error) {
	workRequest, err := c.getWorkRequest(ctx, workRequestID)
	if err != nil {
		return c.fail(resource, err)
	}

	current, err := odaInstanceAttachmentWorkRequestAsyncOperation(resource, workRequest, phase)
	if err != nil {
		return c.fail(resource, err)
	}
	response := c.markAsyncOperation(resource, current)
	if current.NormalizedClass != shared.OSOKAsyncClassSucceeded {
		return response, nil
	}
	return c.resolveAfterWrite(ctx, resource, odaInstanceID, current.Phase, odaInstanceAttachmentIDFromWorkRequest(workRequest, current.Phase, fallbackAttachmentID))
}

func (c *odaInstanceAttachmentRuntimeClient) confirmDeleteWorkRequest(
	ctx context.Context,
	resource *odav1beta1.OdaInstanceAttachment,
	odaInstanceID string,
	workRequestID string,
	fallbackAttachmentID string,
) (bool, error) {
	workRequest, err := c.getWorkRequest(ctx, workRequestID)
	if err != nil {
		return false, c.markFailure(resource, err)
	}

	current, err := odaInstanceAttachmentWorkRequestAsyncOperation(resource, workRequest, shared.OSOKAsyncPhaseDelete)
	if err != nil {
		return false, c.markFailure(resource, err)
	}
	c.markAsyncOperation(resource, current)
	if current.NormalizedClass != shared.OSOKAsyncClassSucceeded {
		return false, nil
	}

	attachmentID := odaInstanceAttachmentIDFromWorkRequest(workRequest, shared.OSOKAsyncPhaseDelete, fallbackAttachmentID)
	return c.confirmDeleted(ctx, resource, odaInstanceID, attachmentID)
}

func (c *odaInstanceAttachmentRuntimeClient) resolveAfterWrite(
	ctx context.Context,
	resource *odav1beta1.OdaInstanceAttachment,
	odaInstanceID string,
	phase shared.OSOKAsyncPhase,
	attachmentID string,
) (servicemanager.OSOKResponse, error) {
	if attachmentID != "" {
		current, err := c.read(ctx, odaInstanceID, attachmentID)
		if err != nil {
			if isOdaInstanceAttachmentReadNotFound(err) {
				return c.fail(resource, fmt.Errorf("OdaInstanceAttachment %s work request completed but OCI attachment %q was not found", phase, attachmentID))
			}
			return c.fail(resource, err)
		}
		return c.finishWithLifecycle(resource, current), nil
	}

	current, found, err := c.resolveByList(ctx, resource, odaInstanceID)
	if err != nil {
		return c.fail(resource, err)
	}
	if !found {
		return c.fail(resource, fmt.Errorf("OdaInstanceAttachment %s completed but OCI attachment identity could not be resolved", phase))
	}
	return c.finishWithLifecycle(resource, current), nil
}

func (c *odaInstanceAttachmentRuntimeClient) confirmDeleted(
	ctx context.Context,
	resource *odav1beta1.OdaInstanceAttachment,
	odaInstanceID string,
	attachmentID string,
) (bool, error) {
	if attachmentID == "" {
		current, found, err := c.resolveByList(ctx, resource, odaInstanceID)
		if err != nil {
			return false, c.markFailure(resource, err)
		}
		if !found {
			c.markDeleted(resource, "OCI OdaInstanceAttachment deleted")
			return true, nil
		}
		attachmentID = odaInstanceAttachmentIDFromSDK(current)
	}

	current, err := c.read(ctx, odaInstanceID, attachmentID)
	if err != nil {
		if isOdaInstanceAttachmentReadNotFound(err) {
			c.markDeleted(resource, "OCI OdaInstanceAttachment deleted")
			return true, nil
		}
		return false, c.markFailure(resource, err)
	}

	if normalizedOdaInstanceAttachmentLifecycle(current.LifecycleState) == string(odasdk.OdaInstanceAttachmentLifecycleStateDetaching) {
		c.projectOdaInstanceAttachmentStatus(resource, current)
		c.markCondition(resource, shared.Terminating, string(current.LifecycleState), "OdaInstanceAttachment delete is in progress", true)
		return false, nil
	}

	c.projectOdaInstanceAttachmentStatus(resource, current)
	c.markDeleteConfirmationPending(resource, string(current.LifecycleState), "OdaInstanceAttachment delete is waiting for confirmation")
	return false, nil
}

func (c *odaInstanceAttachmentRuntimeClient) finishWithLifecycle(resource *odav1beta1.OdaInstanceAttachment, current odasdk.OdaInstanceAttachment) servicemanager.OSOKResponse {
	c.projectOdaInstanceAttachmentStatus(resource, current)

	state := normalizedOdaInstanceAttachmentLifecycle(current.LifecycleState)
	message := odaInstanceAttachmentLifecycleMessage(current, "OdaInstanceAttachment lifecycle state "+state)
	switch state {
	case string(odasdk.OdaInstanceAttachmentLifecycleStateAttaching):
		return c.markCondition(resource, shared.Provisioning, state, message, true)
	case string(odasdk.OdaInstanceAttachmentLifecycleStateDetaching):
		return c.markCondition(resource, shared.Terminating, state, message, true)
	case string(odasdk.OdaInstanceAttachmentLifecycleStateActive):
		return c.markCondition(resource, shared.Active, state, message, false)
	default:
		return c.markCondition(resource, shared.Failed, state, fmt.Sprintf("formal lifecycle state %q is not modeled as successful: %s", state, message), false)
	}
}

func (c *odaInstanceAttachmentRuntimeClient) projectOdaInstanceAttachmentStatus(resource *odav1beta1.OdaInstanceAttachment, current odasdk.OdaInstanceAttachment) {
	status := &resource.Status
	status.Id = stringValue(current.Id)
	status.InstanceId = stringValue(current.InstanceId)
	status.AttachToId = stringValue(current.AttachToId)
	status.AttachmentType = string(current.AttachmentType)
	status.LifecycleState = string(current.LifecycleState)
	status.AttachmentMetadata = stringValue(current.AttachmentMetadata)
	status.RestrictedOperations = slices.Clone(current.RestrictedOperations)
	status.Owner = odaInstanceAttachmentStatusOwnerFromSDK(current.Owner)
	status.TimeCreated = odaInstanceAttachmentSDKTimeString(current.TimeCreated)
	status.TimeLastUpdate = odaInstanceAttachmentSDKTimeString(current.TimeLastUpdate)
	status.FreeformTags = cloneStringMap(current.FreeformTags)
	status.DefinedTags = odaInstanceAttachmentStatusDefinedTags(current.DefinedTags)

	now := metav1.Now()
	if status.Id != "" {
		status.OsokStatus.Ocid = shared.OCID(status.Id)
		if status.OsokStatus.CreatedAt == nil {
			status.OsokStatus.CreatedAt = &now
		}
	}
}

func (c *odaInstanceAttachmentRuntimeClient) fail(resource *odav1beta1.OdaInstanceAttachment, err error) (servicemanager.OSOKResponse, error) {
	return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
}

func (c *odaInstanceAttachmentRuntimeClient) markFailure(resource *odav1beta1.OdaInstanceAttachment, err error) error {
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

func (c *odaInstanceAttachmentRuntimeClient) markDeleted(resource *odav1beta1.OdaInstanceAttachment, message string) {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	status.Async.Current = nil
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, c.log)
}

func (c *odaInstanceAttachmentRuntimeClient) markAsyncOperation(
	resource *odav1beta1.OdaInstanceAttachment,
	current *shared.OSOKAsyncOperation,
) servicemanager.OSOKResponse {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	if resource.Status.Id != "" {
		status.Ocid = shared.OCID(resource.Status.Id)
	}
	if status.Ocid != "" && status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	if current.UpdatedAt == nil {
		current.UpdatedAt = &now
	}
	projection := servicemanager.ApplyAsyncOperation(status, current, c.log)
	return servicemanager.OSOKResponse{
		IsSuccessful:    projection.Condition != shared.Failed,
		ShouldRequeue:   projection.ShouldRequeue,
		RequeueDuration: time.Minute,
	}
}

func (c *odaInstanceAttachmentRuntimeClient) markCondition(
	resource *odav1beta1.OdaInstanceAttachment,
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

	switch {
	case condition == shared.Provisioning || (condition == shared.Terminating && shouldRequeue):
		status.Async.Current = &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           odaInstanceAttachmentCurrentOrFallbackPhase(resource, asyncPhaseForOdaInstanceAttachmentCondition(condition)),
			RawStatus:       rawState,
			NormalizedClass: shared.OSOKAsyncClassPending,
			Message:         message,
			UpdatedAt:       &now,
		}
	case condition == shared.Failed:
		status.Async.Current = &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           odaInstanceAttachmentCurrentOrFallbackPhase(resource, ""),
			RawStatus:       rawState,
			NormalizedClass: odaInstanceAttachmentFailedLifecycleClass(rawState),
			Message:         message,
			UpdatedAt:       &now,
		}
	default:
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

func (c *odaInstanceAttachmentRuntimeClient) markDeleteConfirmationPending(
	resource *odav1beta1.OdaInstanceAttachment,
	rawState string,
	message string,
) {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.Message = message
	status.Reason = string(shared.Terminating)
	status.UpdatedAt = &now

	if status.Async.Current != nil &&
		status.Async.Current.Source == shared.OSOKAsyncSourceWorkRequest &&
		status.Async.Current.Phase == shared.OSOKAsyncPhaseDelete &&
		strings.TrimSpace(status.Async.Current.WorkRequestID) != "" {
		current := *status.Async.Current
		current.Message = message
		current.UpdatedAt = &now
		status.Async.Current = &current
	} else {
		status.Async.Current = &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           shared.OSOKAsyncPhaseDelete,
			RawStatus:       rawState,
			NormalizedClass: shared.OSOKAsyncClassPending,
			Message:         message,
			UpdatedAt:       &now,
		}
	}

	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, c.log)
}

func odaInstanceAttachmentOdaInstanceID(resource *odav1beta1.OdaInstanceAttachment) (string, error) {
	if resource == nil {
		return "", fmt.Errorf("OdaInstanceAttachment resource is nil")
	}
	if odaInstanceID := strings.TrimSpace(resource.Annotations[odaInstanceAttachmentOdaInstanceIDAnnotation]); odaInstanceID != "" {
		return odaInstanceID, nil
	}
	if currentOdaInstanceAttachmentID(resource) != "" {
		if odaInstanceID := strings.TrimSpace(resource.Status.InstanceId); odaInstanceID != "" {
			return odaInstanceID, nil
		}
	}
	return "", fmt.Errorf("OdaInstanceAttachment requires metadata annotation %q with the parent ODA instance OCID", odaInstanceAttachmentOdaInstanceIDAnnotation)
}

func currentOdaInstanceAttachmentID(resource *odav1beta1.OdaInstanceAttachment) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); id != "" {
		return id
	}
	return strings.TrimSpace(resource.Status.Id)
}

func currentOdaInstanceAttachmentWorkRequest(resource *odav1beta1.OdaInstanceAttachment, fallback shared.OSOKAsyncPhase) (string, shared.OSOKAsyncPhase) {
	if resource == nil {
		return "", ""
	}
	return servicemanager.ResolveTrackedWorkRequest(&resource.Status.OsokStatus, resource, servicemanager.WorkRequestLegacyBridge{}, fallback)
}

func buildOdaInstanceAttachmentCreateDetails(spec odav1beta1.OdaInstanceAttachmentSpec) odasdk.CreateOdaInstanceAttachmentDetails {
	details := odasdk.CreateOdaInstanceAttachmentDetails{
		AttachToId:     common.String(spec.AttachToId),
		AttachmentType: odasdk.CreateOdaInstanceAttachmentDetailsAttachmentTypeEnum(spec.AttachmentType),
		Owner:          odaInstanceAttachmentCreateOwnerFromSpec(spec.Owner),
	}
	if spec.AttachmentMetadata != "" {
		details.AttachmentMetadata = common.String(spec.AttachmentMetadata)
	}
	if spec.RestrictedOperations != nil {
		details.RestrictedOperations = slices.Clone(spec.RestrictedOperations)
	}
	if spec.FreeformTags != nil {
		details.FreeformTags = maps.Clone(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		details.DefinedTags = odaInstanceAttachmentDefinedTagsFromSpec(spec.DefinedTags)
	}
	return details
}

func buildOdaInstanceAttachmentUpdateDetails(
	resource *odav1beta1.OdaInstanceAttachment,
	current odasdk.OdaInstanceAttachment,
) (odasdk.UpdateOdaInstanceAttachmentDetails, bool) {
	spec := resource.Spec
	details := odasdk.UpdateOdaInstanceAttachmentDetails{
		AttachmentMetadata:   common.String(spec.AttachmentMetadata),
		RestrictedOperations: slices.Clone(spec.RestrictedOperations),
		Owner:                odaInstanceAttachmentCreateOwnerFromSpec(spec.Owner),
	}
	updateNeeded := false

	if spec.AttachmentMetadata != stringValue(current.AttachmentMetadata) {
		updateNeeded = true
	}
	if !slices.Equal(spec.RestrictedOperations, current.RestrictedOperations) {
		updateNeeded = true
	}
	if !odaInstanceAttachmentOwnerMatchesSpec(current.Owner, spec.Owner) {
		updateNeeded = true
	}
	if spec.FreeformTags != nil && !maps.Equal(spec.FreeformTags, current.FreeformTags) {
		details.FreeformTags = maps.Clone(spec.FreeformTags)
		updateNeeded = true
	}
	if spec.DefinedTags != nil {
		desired := odaInstanceAttachmentDefinedTagsFromSpec(spec.DefinedTags)
		if !odaInstanceAttachmentJSONEqual(desired, current.DefinedTags) {
			details.DefinedTags = desired
			updateNeeded = true
		}
	}

	return details, updateNeeded
}

func validateOdaInstanceAttachmentTrackedParent(resource *odav1beta1.OdaInstanceAttachment, odaInstanceID string) error {
	if resource == nil {
		return fmt.Errorf("OdaInstanceAttachment resource is nil")
	}
	statusInstanceID := strings.TrimSpace(resource.Status.InstanceId)
	if statusInstanceID == "" || statusInstanceID == odaInstanceID {
		return nil
	}
	return fmt.Errorf("OdaInstanceAttachment create-only field drift detected for odaInstanceId; recreate the resource instead of changing parent ODA instance annotation")
}

func validateOdaInstanceAttachmentCreateOnlyDrift(resource *odav1beta1.OdaInstanceAttachment, odaInstanceID string, current odasdk.OdaInstanceAttachment) error {
	spec := resource.Spec
	drift := []string{}
	if odaInstanceID != "" && stringValue(current.InstanceId) != "" && odaInstanceID != stringValue(current.InstanceId) {
		drift = append(drift, "odaInstanceId")
	}
	if spec.AttachToId != "" && stringValue(current.AttachToId) != "" && spec.AttachToId != stringValue(current.AttachToId) {
		drift = append(drift, "attachToId")
	}
	if spec.AttachmentType != "" && current.AttachmentType != "" && spec.AttachmentType != string(current.AttachmentType) {
		drift = append(drift, "attachmentType")
	}
	if len(drift) == 0 {
		return nil
	}
	return fmt.Errorf("OdaInstanceAttachment create-only field drift detected for %s; recreate the resource instead of updating immutable fields", strings.Join(drift, ", "))
}

func odaInstanceAttachmentSummaryMatches(resource *odav1beta1.OdaInstanceAttachment, odaInstanceID string, item odasdk.OdaInstanceAttachmentSummary) bool {
	if resource == nil {
		return false
	}
	return stringValue(item.InstanceId) == odaInstanceID &&
		stringValue(item.AttachToId) == resource.Spec.AttachToId &&
		string(item.AttachmentType) == resource.Spec.AttachmentType &&
		odaInstanceAttachmentOwnerMatchesSpec(item.Owner, resource.Spec.Owner)
}

func isOdaInstanceAttachmentRetryableLifecycle(state odasdk.OdaInstanceAttachmentLifecycleStateEnum) bool {
	switch normalizedOdaInstanceAttachmentLifecycle(state) {
	case string(odasdk.OdaInstanceAttachmentLifecycleStateAttaching),
		string(odasdk.OdaInstanceAttachmentLifecycleStateDetaching):
		return true
	default:
		return false
	}
}

func isOdaInstanceAttachmentReadNotFound(err error) bool {
	classification := errorutil.ClassifyDeleteError(err)
	return classification.IsUnambiguousNotFound() || classification.IsAuthShapedNotFound()
}

func odaInstanceAttachmentIDFromSDK(current odasdk.OdaInstanceAttachment) string {
	return strings.TrimSpace(stringValue(current.Id))
}

func odaInstanceAttachmentIDFromWorkRequest(
	workRequest odasdk.WorkRequest,
	phase shared.OSOKAsyncPhase,
	fallback string,
) string {
	if id := strings.TrimSpace(stringValue(workRequest.ResourceId)); id != "" {
		return id
	}

	for _, resource := range workRequest.Resources {
		if !odaInstanceAttachmentWorkRequestResourceActionMatchesPhase(resource.ResourceAction, phase) {
			continue
		}
		if id := strings.TrimSpace(stringValue(resource.ResourceId)); id != "" {
			return id
		}
	}
	for _, resource := range workRequest.Resources {
		if id := strings.TrimSpace(stringValue(resource.ResourceId)); id != "" {
			return id
		}
	}
	return strings.TrimSpace(fallback)
}

func odaInstanceAttachmentWorkRequestResourceActionMatchesPhase(action odasdk.WorkRequestResourceResourceActionEnum, phase shared.OSOKAsyncPhase) bool {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return action == odasdk.WorkRequestResourceResourceActionCreateOdaInstanceAttachment ||
			action == odasdk.WorkRequestResourceResourceActionCreate
	case shared.OSOKAsyncPhaseUpdate:
		return action == odasdk.WorkRequestResourceResourceActionUpdateOdaInstanceAttachment ||
			action == odasdk.WorkRequestResourceResourceActionUpdate
	case shared.OSOKAsyncPhaseDelete:
		return action == odasdk.WorkRequestResourceResourceActionDeleteOdaInstanceAttachment ||
			action == odasdk.WorkRequestResourceResourceActionDelete
	default:
		return false
	}
}

func odaInstanceAttachmentWorkRequestAsyncOperation(
	resource *odav1beta1.OdaInstanceAttachment,
	workRequest odasdk.WorkRequest,
	explicitPhase shared.OSOKAsyncPhase,
) (*shared.OSOKAsyncOperation, error) {
	var status *shared.OSOKStatus
	if resource != nil {
		status = &resource.Status.OsokStatus
	}

	current, err := servicemanager.BuildWorkRequestAsyncOperation(status, odaInstanceAttachmentWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:       string(workRequest.Status),
		RawAction:       string(workRequest.RequestAction),
		WorkRequestID:   stringValue(workRequest.Id),
		PercentComplete: workRequest.PercentComplete,
		Message:         stringValue(workRequest.StatusMessage),
		FallbackPhase:   odaInstanceAttachmentCurrentOrFallbackPhase(resource, explicitPhase),
	})
	if err != nil {
		return nil, err
	}
	current.Message = odaInstanceAttachmentWorkRequestMessage(current.Phase, workRequest)
	return current, nil
}

func odaInstanceAttachmentWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest odasdk.WorkRequest) string {
	if message := strings.TrimSpace(stringValue(workRequest.StatusMessage)); message != "" {
		return message
	}
	action := "reconcile"
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		action = "create"
	case shared.OSOKAsyncPhaseUpdate:
		action = "update"
	case shared.OSOKAsyncPhaseDelete:
		action = "delete"
	}
	if id := strings.TrimSpace(stringValue(workRequest.Id)); id != "" {
		return fmt.Sprintf("OdaInstanceAttachment %s work request %s is %s", action, id, workRequest.Status)
	}
	return fmt.Sprintf("OdaInstanceAttachment %s work request is %s", action, workRequest.Status)
}

func odaInstanceAttachmentLifecycleMessage(current odasdk.OdaInstanceAttachment, fallback string) string {
	if id := strings.TrimSpace(stringValue(current.Id)); id != "" {
		return fmt.Sprintf("OdaInstanceAttachment %s is %s", id, current.LifecycleState)
	}
	return fallback
}

func normalizedOdaInstanceAttachmentLifecycle(state odasdk.OdaInstanceAttachmentLifecycleStateEnum) string {
	return strings.ToUpper(strings.TrimSpace(string(state)))
}

func asyncPhaseForOdaInstanceAttachmentCondition(condition shared.OSOKConditionType) shared.OSOKAsyncPhase {
	switch condition {
	case shared.Terminating:
		return shared.OSOKAsyncPhaseDelete
	default:
		return shared.OSOKAsyncPhaseCreate
	}
}

func odaInstanceAttachmentCurrentOrFallbackPhase(resource *odav1beta1.OdaInstanceAttachment, fallback shared.OSOKAsyncPhase) shared.OSOKAsyncPhase {
	if resource != nil && resource.Status.OsokStatus.Async.Current != nil && resource.Status.OsokStatus.Async.Current.Phase != "" {
		return resource.Status.OsokStatus.Async.Current.Phase
	}
	return fallback
}

func odaInstanceAttachmentFailedLifecycleClass(rawState string) shared.OSOKAsyncNormalizedClass {
	if strings.Contains(strings.ToUpper(strings.TrimSpace(rawState)), "FAIL") {
		return shared.OSOKAsyncClassFailed
	}
	return shared.OSOKAsyncClassUnknown
}

func odaInstanceAttachmentCreateOwnerFromSpec(owner odav1beta1.OdaInstanceAttachmentOwner) *odasdk.OdaInstanceAttachmentOwner {
	return &odasdk.OdaInstanceAttachmentOwner{
		OwnerServiceName:    common.String(owner.OwnerServiceName),
		OwnerServiceTenancy: common.String(owner.OwnerServiceTenancy),
	}
}

func odaInstanceAttachmentStatusOwnerFromSDK(owner *odasdk.OdaInstanceOwner) odav1beta1.OdaInstanceAttachmentOwner {
	if owner == nil {
		return odav1beta1.OdaInstanceAttachmentOwner{}
	}
	return odav1beta1.OdaInstanceAttachmentOwner{
		OwnerServiceName:    stringValue(owner.OwnerServiceName),
		OwnerServiceTenancy: stringValue(owner.OwnerServiceTenancy),
	}
}

func odaInstanceAttachmentOwnerMatchesSpec(owner *odasdk.OdaInstanceOwner, spec odav1beta1.OdaInstanceAttachmentOwner) bool {
	if owner == nil {
		return spec.OwnerServiceName == "" && spec.OwnerServiceTenancy == ""
	}
	return stringValue(owner.OwnerServiceName) == spec.OwnerServiceName &&
		stringValue(owner.OwnerServiceTenancy) == spec.OwnerServiceTenancy
}

func odaInstanceAttachmentDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
	if spec == nil {
		return nil
	}
	return *util.ConvertToOciDefinedTags(&spec)
}

func odaInstanceAttachmentStatusDefinedTags(input map[string]map[string]interface{}) map[string]shared.MapValue {
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

func odaInstanceAttachmentJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}

func odaInstanceAttachmentSDKTimeString(value *common.SDKTime) string {
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

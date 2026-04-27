/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package digitalassistant

import (
	"context"
	"crypto/sha256"
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

const digitalAssistantOdaInstanceIDAnnotation = "oda.oracle.com/oda-instance-id"

var digitalAssistantWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(odasdk.WorkRequestStatusAccepted),
		string(odasdk.WorkRequestStatusInProgress),
		string(odasdk.WorkRequestStatusCanceling),
	},
	SucceededStatusTokens: []string{string(odasdk.WorkRequestStatusSucceeded)},
	FailedStatusTokens:    []string{string(odasdk.WorkRequestStatusFailed)},
	CanceledStatusTokens:  []string{string(odasdk.WorkRequestStatusCanceled)},
	CreateActionTokens: []string{
		string(odasdk.WorkRequestRequestActionCreateDigitalAssistant),
		string(odasdk.WorkRequestRequestActionCloneDigitalAssistant),
		string(odasdk.WorkRequestRequestActionExtendDigitalAssistant),
		string(odasdk.WorkRequestRequestActionVersionDigitalAssistant),
	},
}

type digitalAssistantOCIClient interface {
	CreateDigitalAssistant(context.Context, odasdk.CreateDigitalAssistantRequest) (odasdk.CreateDigitalAssistantResponse, error)
	GetDigitalAssistant(context.Context, odasdk.GetDigitalAssistantRequest) (odasdk.GetDigitalAssistantResponse, error)
	ListDigitalAssistants(context.Context, odasdk.ListDigitalAssistantsRequest) (odasdk.ListDigitalAssistantsResponse, error)
	UpdateDigitalAssistant(context.Context, odasdk.UpdateDigitalAssistantRequest) (odasdk.UpdateDigitalAssistantResponse, error)
	DeleteDigitalAssistant(context.Context, odasdk.DeleteDigitalAssistantRequest) (odasdk.DeleteDigitalAssistantResponse, error)
	GetWorkRequest(context.Context, odasdk.GetWorkRequestRequest) (odasdk.GetWorkRequestResponse, error)
}

type digitalAssistantManagementClient struct {
	management odasdk.ManagementClient
	work       odasdk.OdaClient
}

func (c digitalAssistantManagementClient) CreateDigitalAssistant(ctx context.Context, request odasdk.CreateDigitalAssistantRequest) (odasdk.CreateDigitalAssistantResponse, error) {
	return c.management.CreateDigitalAssistant(ctx, request)
}

func (c digitalAssistantManagementClient) GetDigitalAssistant(ctx context.Context, request odasdk.GetDigitalAssistantRequest) (odasdk.GetDigitalAssistantResponse, error) {
	return c.management.GetDigitalAssistant(ctx, request)
}

func (c digitalAssistantManagementClient) ListDigitalAssistants(ctx context.Context, request odasdk.ListDigitalAssistantsRequest) (odasdk.ListDigitalAssistantsResponse, error) {
	return c.management.ListDigitalAssistants(ctx, request)
}

func (c digitalAssistantManagementClient) UpdateDigitalAssistant(ctx context.Context, request odasdk.UpdateDigitalAssistantRequest) (odasdk.UpdateDigitalAssistantResponse, error) {
	return c.management.UpdateDigitalAssistant(ctx, request)
}

func (c digitalAssistantManagementClient) DeleteDigitalAssistant(ctx context.Context, request odasdk.DeleteDigitalAssistantRequest) (odasdk.DeleteDigitalAssistantResponse, error) {
	return c.management.DeleteDigitalAssistant(ctx, request)
}

func (c digitalAssistantManagementClient) GetWorkRequest(ctx context.Context, request odasdk.GetWorkRequestRequest) (odasdk.GetWorkRequestResponse, error) {
	return c.work.GetWorkRequest(ctx, request)
}

type digitalAssistantRuntimeClient struct {
	hooks   DigitalAssistantRuntimeHooks
	log     loggerutil.OSOKLogger
	client  digitalAssistantOCIClient
	initErr error
}

var _ DigitalAssistantServiceClient = (*digitalAssistantRuntimeClient)(nil)

func init() {
	registerDigitalAssistantRuntimeHooksMutator(func(manager *DigitalAssistantServiceManager, hooks *DigitalAssistantRuntimeHooks) {
		client, initErr := newDigitalAssistantSDKClient(manager)
		applyDigitalAssistantRuntimeHooks(manager, hooks, client, initErr)
	})
}

func newDigitalAssistantSDKClient(manager *DigitalAssistantServiceManager) (digitalAssistantOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("DigitalAssistant service manager is nil")
	}
	managementClient, managementErr := odasdk.NewManagementClientWithConfigurationProvider(manager.Provider)
	workClient, workErr := odasdk.NewOdaClientWithConfigurationProvider(manager.Provider)
	if managementErr != nil || workErr != nil {
		return nil, fmt.Errorf("initialize DigitalAssistant OCI clients: %w", joinErrors(managementErr, workErr))
	}
	return digitalAssistantManagementClient{management: managementClient, work: workClient}, nil
}

func joinErrors(errs ...error) error {
	var joined error
	for _, err := range errs {
		if err == nil {
			continue
		}
		if joined == nil {
			joined = err
			continue
		}
		joined = fmt.Errorf("%w; %w", joined, err)
	}
	return joined
}

func applyDigitalAssistantRuntimeHooks(
	manager *DigitalAssistantServiceManager,
	hooks *DigitalAssistantRuntimeHooks,
	client digitalAssistantOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newDigitalAssistantRuntimeSemantics()
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(DigitalAssistantServiceClient) DigitalAssistantServiceClient {
		runtimeClient := &digitalAssistantRuntimeClient{
			hooks:   *hooks,
			client:  client,
			initErr: initErr,
		}
		if manager != nil {
			runtimeClient.log = manager.Log
		}
		return runtimeClient
	})
}

func newDigitalAssistantServiceClientWithOCIClient(log loggerutil.OSOKLogger, client digitalAssistantOCIClient) DigitalAssistantServiceClient {
	manager := &DigitalAssistantServiceManager{Log: log}
	hooks := newDigitalAssistantRuntimeHooksWithOCIClient(client)
	applyDigitalAssistantRuntimeHooks(manager, &hooks, client, nil)
	delegate := defaultDigitalAssistantServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*odav1beta1.DigitalAssistant](
			buildDigitalAssistantGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapDigitalAssistantGeneratedClient(hooks, delegate)
}

func newDigitalAssistantRuntimeHooksWithOCIClient(client digitalAssistantOCIClient) DigitalAssistantRuntimeHooks {
	return DigitalAssistantRuntimeHooks{
		Create: runtimeOperationHooks[odasdk.CreateDigitalAssistantRequest, odasdk.CreateDigitalAssistantResponse]{
			Call: func(ctx context.Context, request odasdk.CreateDigitalAssistantRequest) (odasdk.CreateDigitalAssistantResponse, error) {
				return client.CreateDigitalAssistant(ctx, request)
			},
		},
		Get: runtimeOperationHooks[odasdk.GetDigitalAssistantRequest, odasdk.GetDigitalAssistantResponse]{
			Call: func(ctx context.Context, request odasdk.GetDigitalAssistantRequest) (odasdk.GetDigitalAssistantResponse, error) {
				return client.GetDigitalAssistant(ctx, request)
			},
		},
		List: runtimeOperationHooks[odasdk.ListDigitalAssistantsRequest, odasdk.ListDigitalAssistantsResponse]{
			Call: func(ctx context.Context, request odasdk.ListDigitalAssistantsRequest) (odasdk.ListDigitalAssistantsResponse, error) {
				return client.ListDigitalAssistants(ctx, request)
			},
		},
		Update: runtimeOperationHooks[odasdk.UpdateDigitalAssistantRequest, odasdk.UpdateDigitalAssistantResponse]{
			Call: func(ctx context.Context, request odasdk.UpdateDigitalAssistantRequest) (odasdk.UpdateDigitalAssistantResponse, error) {
				return client.UpdateDigitalAssistant(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[odasdk.DeleteDigitalAssistantRequest, odasdk.DeleteDigitalAssistantResponse]{
			Call: func(ctx context.Context, request odasdk.DeleteDigitalAssistantRequest) (odasdk.DeleteDigitalAssistantResponse, error) {
				return client.DeleteDigitalAssistant(ctx, request)
			},
		},
	}
}

func newDigitalAssistantRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "oda",
		FormalSlug:    "digitalassistant",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "workrequest",
			Runtime:              "handwritten",
			FormalClassification: "workrequest",
			WorkRequest: &generatedruntime.WorkRequestSemantics{
				Source: "service-sdk",
				Phases: []string{
					string(shared.OSOKAsyncPhaseCreate),
				},
			},
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(odasdk.LifecycleStateCreating)},
			UpdatingStates:     []string{string(odasdk.LifecycleStateUpdating)},
			ActiveStates:       []string{string(odasdk.LifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(odasdk.LifecycleStateDeleting)},
			TerminalStates: []string{string(odasdk.LifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"odaInstanceId", "name", "version"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{"category", "description", "freeformTags", "definedTags"},
			ForceNew: []string{
				"jsonData",
				"kind",
				"id",
				"name",
				"displayName",
				"version",
				"platformVersion",
				"multilingualMode",
				"primaryLanguageTag",
				"nativeLanguageTags",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "resource-local", EntityType: "DigitalAssistant", Action: "CreateDigitalAssistant"}},
			Update: []generatedruntime.Hook{{Helper: "resource-local", EntityType: "DigitalAssistant", Action: "UpdateDigitalAssistant"}},
			Delete: []generatedruntime.Hook{{Helper: "resource-local", EntityType: "DigitalAssistant", Action: "DeleteDigitalAssistant"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetDigitalAssistant/ListDigitalAssistants",
			Hooks: []generatedruntime.Hook{
				{Helper: "resource-local", EntityType: "WorkRequest", Action: "GetWorkRequest"},
				{Helper: "resource-local", EntityType: "DigitalAssistant", Action: "GetDigitalAssistant"},
				{Helper: "resource-local", EntityType: "DigitalAssistant", Action: "ListDigitalAssistants"},
			},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "resource-local", EntityType: "DigitalAssistant", Action: "GetDigitalAssistant"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "resource-local", EntityType: "DigitalAssistant", Action: "GetDigitalAssistant"}},
		},
	}
}

func (c *digitalAssistantRuntimeClient) CreateOrUpdate(ctx context.Context, resource *odav1beta1.DigitalAssistant, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("DigitalAssistant resource is nil")
	}
	if c == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("DigitalAssistant runtime client is not configured")
	}
	if err := c.ensureConfigured(); err != nil {
		return c.fail(resource, err)
	}
	if err := validateDigitalAssistantDesiredSpec(resource); err != nil {
		return c.fail(resource, err)
	}

	odaInstanceID, err := digitalAssistantOdaInstanceID(resource)
	if err != nil {
		return c.fail(resource, err)
	}
	if workRequestID, phase := currentDigitalAssistantWorkRequest(resource, ""); workRequestID != "" && phase != shared.OSOKAsyncPhaseDelete {
		return c.applyWorkRequest(ctx, resource, odaInstanceID, workRequestID, phase, currentDigitalAssistantID(resource))
	}

	current, found, err := c.resolveCurrent(ctx, resource, odaInstanceID)
	if err != nil {
		return c.fail(resource, err)
	}
	if !found {
		return c.create(ctx, resource, odaInstanceID)
	}

	if digitalAssistantLifecycleBlocksMutation(current.LifecycleState) {
		return c.finishWithLifecycle(resource, current), nil
	}
	if err := validateDigitalAssistantCreateOnlyDrift(resource, current); err != nil {
		return c.fail(resource, err)
	}

	updateDetails, updateNeeded := buildDigitalAssistantUpdateDetails(resource, current)
	if updateNeeded {
		return c.update(ctx, resource, odaInstanceID, digitalAssistantIDFromSDK(current), updateDetails)
	}

	return c.finishWithLifecycle(resource, current), nil
}

func (c *digitalAssistantRuntimeClient) Delete(ctx context.Context, resource *odav1beta1.DigitalAssistant) (bool, error) {
	if resource == nil {
		return false, fmt.Errorf("DigitalAssistant resource is nil")
	}
	if c == nil {
		return false, fmt.Errorf("DigitalAssistant runtime client is not configured")
	}
	if err := c.ensureConfigured(); err != nil {
		return false, c.markFailure(resource, err)
	}

	odaInstanceID, parentErr := digitalAssistantOdaInstanceID(resource)
	currentID := currentDigitalAssistantID(resource)
	if parentErr != nil {
		if currentID == "" {
			c.markDeleted(resource, "DigitalAssistant has no tracked OCI identity")
			return true, nil
		}
		c.markFailure(resource, parentErr)
		return false, parentErr
	}
	if workRequestID, phase := currentDigitalAssistantWorkRequest(resource, ""); workRequestID != "" {
		if phase == shared.OSOKAsyncPhaseDelete {
			return c.confirmDeleteWorkRequest(ctx, resource, odaInstanceID, workRequestID, currentID)
		}
		response, err := c.applyWorkRequest(ctx, resource, odaInstanceID, workRequestID, phase, currentID)
		if err != nil {
			return false, err
		}
		if response.ShouldRequeue || currentDigitalAssistantID(resource) == "" {
			return false, nil
		}
	}

	current, found, err := c.resolveCurrent(ctx, resource, odaInstanceID)
	if err != nil {
		return false, c.markFailure(resource, err)
	}
	if !found {
		c.markDeleted(resource, "OCI DigitalAssistant is already deleted")
		return true, nil
	}

	currentID = digitalAssistantIDFromSDK(current)
	if currentID == "" {
		err := fmt.Errorf("DigitalAssistant delete could not resolve OCI resource ID")
		return false, c.markFailure(resource, err)
	}

	switch normalizedDigitalAssistantLifecycle(current.LifecycleState) {
	case string(odasdk.LifecycleStateDeleting):
		c.projectDigitalAssistantStatus(resource, current)
		c.markCondition(resource, shared.Terminating, string(current.LifecycleState), "DigitalAssistant delete is in progress", true)
		return false, nil
	case string(odasdk.LifecycleStateDeleted):
		c.markDeleted(resource, "OCI DigitalAssistant deleted")
		return true, nil
	}

	response, err := c.hooks.Delete.Call(ctx, odasdk.DeleteDigitalAssistantRequest{
		OdaInstanceId:      common.String(odaInstanceID),
		DigitalAssistantId: common.String(currentID),
	})
	if err != nil {
		if isDigitalAssistantReadNotFound(err) {
			c.markDeleted(resource, "OCI DigitalAssistant is already deleted")
			return true, nil
		}
		return false, c.markFailure(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)

	return c.confirmDeleted(ctx, resource, odaInstanceID, currentID)
}

func (c *digitalAssistantRuntimeClient) ensureConfigured() error {
	if c.initErr != nil {
		return c.initErr
	}
	if c.client == nil {
		return fmt.Errorf("DigitalAssistant OCI client is not configured")
	}
	return nil
}

func (c *digitalAssistantRuntimeClient) create(ctx context.Context, resource *odav1beta1.DigitalAssistant, odaInstanceID string) (servicemanager.OSOKResponse, error) {
	details, err := buildDigitalAssistantCreateDetails(resource)
	if err != nil {
		return c.fail(resource, err)
	}

	response, err := c.hooks.Create.Call(ctx, odasdk.CreateDigitalAssistantRequest{
		OdaInstanceId:                 common.String(odaInstanceID),
		CreateDigitalAssistantDetails: details,
		OpcRetryToken:                 digitalAssistantRetryToken(resource),
	})
	if err != nil {
		return c.fail(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)

	workRequestID := strings.TrimSpace(stringValue(response.OpcWorkRequestId))
	if workRequestID != "" {
		return c.applyWorkRequest(ctx, resource, odaInstanceID, workRequestID, shared.OSOKAsyncPhaseCreate, "")
	}

	current, found, err := c.resolveByList(ctx, resource, odaInstanceID)
	if err != nil {
		return c.fail(resource, err)
	}
	if !found {
		return c.markCondition(resource, shared.Provisioning, string(odasdk.LifecycleStateCreating), "DigitalAssistant create request accepted", true), nil
	}
	return c.finishWithLifecycle(resource, current), nil
}

func (c *digitalAssistantRuntimeClient) update(
	ctx context.Context,
	resource *odav1beta1.DigitalAssistant,
	odaInstanceID string,
	digitalAssistantID string,
	details odasdk.UpdateDigitalAssistantDetails,
) (servicemanager.OSOKResponse, error) {
	if digitalAssistantID == "" {
		return c.fail(resource, fmt.Errorf("DigitalAssistant update could not resolve OCI resource ID"))
	}
	response, err := c.hooks.Update.Call(ctx, odasdk.UpdateDigitalAssistantRequest{
		OdaInstanceId:                 common.String(odaInstanceID),
		DigitalAssistantId:            common.String(digitalAssistantID),
		UpdateDigitalAssistantDetails: details,
	})
	if err != nil {
		return c.fail(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)

	followUp, err := c.read(ctx, odaInstanceID, digitalAssistantID)
	if err != nil {
		return c.fail(resource, err)
	}
	return c.finishWithLifecycle(resource, followUp), nil
}

func (c *digitalAssistantRuntimeClient) resolveCurrent(
	ctx context.Context,
	resource *odav1beta1.DigitalAssistant,
	odaInstanceID string,
) (odasdk.DigitalAssistant, bool, error) {
	if currentID := currentDigitalAssistantID(resource); currentID != "" {
		current, err := c.read(ctx, odaInstanceID, currentID)
		if err != nil {
			if isDigitalAssistantReadNotFound(err) {
				return odasdk.DigitalAssistant{}, false, nil
			}
			return odasdk.DigitalAssistant{}, false, err
		}
		return current, true, nil
	}
	return c.resolveByList(ctx, resource, odaInstanceID)
}

func (c *digitalAssistantRuntimeClient) resolveByList(
	ctx context.Context,
	resource *odav1beta1.DigitalAssistant,
	odaInstanceID string,
) (odasdk.DigitalAssistant, bool, error) {
	name, version := digitalAssistantDesiredNameVersion(resource)
	if name == "" {
		return odasdk.DigitalAssistant{}, false, nil
	}
	request := odasdk.ListDigitalAssistantsRequest{
		OdaInstanceId: common.String(odaInstanceID),
		Name:          common.String(name),
	}
	if version != "" {
		request.Version = common.String(version)
	}

	var matchedID string
	for {
		response, err := c.hooks.List.Call(ctx, request)
		if err != nil {
			return odasdk.DigitalAssistant{}, false, err
		}

		for _, item := range response.Items {
			if !digitalAssistantSummaryMatches(resource, item) {
				continue
			}
			if normalizedDigitalAssistantLifecycle(item.LifecycleState) == string(odasdk.LifecycleStateDeleted) {
				continue
			}
			itemID := stringValue(item.Id)
			if itemID == "" {
				continue
			}
			if matchedID != "" && matchedID != itemID {
				return odasdk.DigitalAssistant{}, false, fmt.Errorf("multiple OCI DigitalAssistants matched name %q", name)
			}
			matchedID = itemID
		}

		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			break
		}
		request.Page = response.OpcNextPage
	}
	if matchedID == "" {
		return odasdk.DigitalAssistant{}, false, nil
	}

	current, err := c.read(ctx, odaInstanceID, matchedID)
	if err != nil {
		if isDigitalAssistantReadNotFound(err) {
			return odasdk.DigitalAssistant{}, false, nil
		}
		return odasdk.DigitalAssistant{}, false, err
	}
	return current, true, nil
}

func (c *digitalAssistantRuntimeClient) read(ctx context.Context, odaInstanceID string, digitalAssistantID string) (odasdk.DigitalAssistant, error) {
	response, err := c.hooks.Get.Call(ctx, odasdk.GetDigitalAssistantRequest{
		OdaInstanceId:      common.String(odaInstanceID),
		DigitalAssistantId: common.String(digitalAssistantID),
	})
	if err != nil {
		return odasdk.DigitalAssistant{}, err
	}
	return response.DigitalAssistant, nil
}

func (c *digitalAssistantRuntimeClient) getWorkRequest(ctx context.Context, workRequestID string) (odasdk.WorkRequest, error) {
	if c.initErr != nil {
		return odasdk.WorkRequest{}, c.initErr
	}
	if c.client == nil {
		return odasdk.WorkRequest{}, fmt.Errorf("DigitalAssistant OCI client is not configured")
	}
	response, err := c.client.GetWorkRequest(ctx, odasdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return odasdk.WorkRequest{}, err
	}
	return response.WorkRequest, nil
}

func (c *digitalAssistantRuntimeClient) applyWorkRequest(
	ctx context.Context,
	resource *odav1beta1.DigitalAssistant,
	odaInstanceID string,
	workRequestID string,
	phase shared.OSOKAsyncPhase,
	fallbackDigitalAssistantID string,
) (servicemanager.OSOKResponse, error) {
	workRequest, err := c.getWorkRequest(ctx, workRequestID)
	if err != nil {
		return c.fail(resource, err)
	}

	current, err := digitalAssistantWorkRequestAsyncOperation(resource, workRequest, phase)
	if err != nil {
		return c.fail(resource, err)
	}
	digitalAssistantID := digitalAssistantIDFromWorkRequest(workRequest, current.Phase, fallbackDigitalAssistantID)
	if current.Phase != shared.OSOKAsyncPhaseDelete && digitalAssistantID != "" {
		resource.Status.Id = digitalAssistantID
		resource.Status.OsokStatus.Ocid = shared.OCID(digitalAssistantID)
	}
	response := c.markAsyncOperation(resource, current)
	if current.NormalizedClass != shared.OSOKAsyncClassSucceeded {
		return response, nil
	}
	return c.resolveAfterWrite(ctx, resource, odaInstanceID, current.Phase, digitalAssistantID)
}

func (c *digitalAssistantRuntimeClient) confirmDeleteWorkRequest(
	ctx context.Context,
	resource *odav1beta1.DigitalAssistant,
	odaInstanceID string,
	workRequestID string,
	fallbackDigitalAssistantID string,
) (bool, error) {
	workRequest, err := c.getWorkRequest(ctx, workRequestID)
	if err != nil {
		return false, c.markFailure(resource, err)
	}
	current, err := digitalAssistantWorkRequestAsyncOperation(resource, workRequest, shared.OSOKAsyncPhaseDelete)
	if err != nil {
		return false, c.markFailure(resource, err)
	}
	c.markAsyncOperation(resource, current)
	if current.NormalizedClass != shared.OSOKAsyncClassSucceeded {
		return false, nil
	}
	return c.confirmDeleted(ctx, resource, odaInstanceID, digitalAssistantIDFromWorkRequest(workRequest, shared.OSOKAsyncPhaseDelete, fallbackDigitalAssistantID))
}

func (c *digitalAssistantRuntimeClient) resolveAfterWrite(
	ctx context.Context,
	resource *odav1beta1.DigitalAssistant,
	odaInstanceID string,
	phase shared.OSOKAsyncPhase,
	digitalAssistantID string,
) (servicemanager.OSOKResponse, error) {
	if digitalAssistantID != "" {
		current, err := c.read(ctx, odaInstanceID, digitalAssistantID)
		if err != nil {
			if isDigitalAssistantReadNotFound(err) {
				return c.fail(resource, fmt.Errorf("DigitalAssistant %s work request completed but OCI DigitalAssistant %q was not found", phase, digitalAssistantID))
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
		return c.fail(resource, fmt.Errorf("DigitalAssistant %s completed but OCI identity could not be resolved", phase))
	}
	return c.finishWithLifecycle(resource, current), nil
}

func (c *digitalAssistantRuntimeClient) confirmDeleted(
	ctx context.Context,
	resource *odav1beta1.DigitalAssistant,
	odaInstanceID string,
	digitalAssistantID string,
) (bool, error) {
	if digitalAssistantID == "" {
		current, found, err := c.resolveByList(ctx, resource, odaInstanceID)
		if err != nil {
			return false, c.markFailure(resource, err)
		}
		if !found {
			c.markDeleted(resource, "OCI DigitalAssistant deleted")
			return true, nil
		}
		digitalAssistantID = digitalAssistantIDFromSDK(current)
	}

	current, err := c.read(ctx, odaInstanceID, digitalAssistantID)
	if err != nil {
		if isDigitalAssistantReadNotFound(err) {
			c.markDeleted(resource, "OCI DigitalAssistant deleted")
			return true, nil
		}
		return false, c.markFailure(resource, err)
	}
	if normalizedDigitalAssistantLifecycle(current.LifecycleState) == string(odasdk.LifecycleStateDeleted) {
		c.markDeleted(resource, "OCI DigitalAssistant deleted")
		return true, nil
	}

	c.projectDigitalAssistantStatus(resource, current)
	c.markCondition(resource, shared.Terminating, string(current.LifecycleState), "DigitalAssistant delete is in progress", true)
	return false, nil
}

func (c *digitalAssistantRuntimeClient) finishWithLifecycle(resource *odav1beta1.DigitalAssistant, current odasdk.DigitalAssistant) servicemanager.OSOKResponse {
	c.projectDigitalAssistantStatus(resource, current)

	state := normalizedDigitalAssistantLifecycle(current.LifecycleState)
	message := digitalAssistantLifecycleMessage(current, "DigitalAssistant lifecycle state "+state)
	switch state {
	case string(odasdk.LifecycleStateCreating):
		return c.markCondition(resource, shared.Provisioning, state, message, true)
	case string(odasdk.LifecycleStateUpdating):
		return c.markCondition(resource, shared.Updating, state, message, true)
	case string(odasdk.LifecycleStateDeleting):
		return c.markCondition(resource, shared.Terminating, state, message, true)
	case string(odasdk.LifecycleStateActive):
		return c.markCondition(resource, shared.Active, state, message, false)
	case string(odasdk.LifecycleStateDeleted),
		string(odasdk.LifecycleStateFailed),
		string(odasdk.LifecycleStateInactive):
		return c.markCondition(resource, shared.Failed, state, message, false)
	default:
		return c.markCondition(resource, shared.Failed, state, fmt.Sprintf("formal lifecycle state %q is not modeled: %s", state, message), false)
	}
}

func (c *digitalAssistantRuntimeClient) projectDigitalAssistantStatus(resource *odav1beta1.DigitalAssistant, current odasdk.DigitalAssistant) {
	status := &resource.Status
	status.Id = stringValue(current.Id)
	status.Name = stringValue(current.Name)
	status.Version = stringValue(current.Version)
	status.DisplayName = stringValue(current.DisplayName)
	status.LifecycleState = string(current.LifecycleState)
	status.LifecycleDetails = string(current.LifecycleDetails)
	status.PlatformVersion = stringValue(current.PlatformVersion)
	status.TimeCreated = digitalAssistantSDKTimeString(current.TimeCreated)
	status.TimeUpdated = digitalAssistantSDKTimeString(current.TimeUpdated)
	status.Category = stringValue(current.Category)
	status.Description = stringValue(current.Description)
	status.Namespace = stringValue(current.Namespace)
	status.DialogVersion = stringValue(current.DialogVersion)
	status.BaseId = stringValue(current.BaseId)
	status.MultilingualMode = string(current.MultilingualMode)
	status.PrimaryLanguageTag = stringValue(current.PrimaryLanguageTag)
	status.NativeLanguageTags = slices.Clone(current.NativeLanguageTags)
	status.FreeformTags = cloneStringMap(current.FreeformTags)
	status.DefinedTags = digitalAssistantStatusDefinedTags(current.DefinedTags)

	now := metav1.Now()
	if status.Id != "" {
		status.OsokStatus.Ocid = shared.OCID(status.Id)
		if status.OsokStatus.CreatedAt == nil {
			status.OsokStatus.CreatedAt = &now
		}
	}
}

func (c *digitalAssistantRuntimeClient) fail(resource *odav1beta1.DigitalAssistant, err error) (servicemanager.OSOKResponse, error) {
	return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
}

func (c *digitalAssistantRuntimeClient) markFailure(resource *odav1beta1.DigitalAssistant, err error) error {
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

func (c *digitalAssistantRuntimeClient) markDeleted(resource *odav1beta1.DigitalAssistant, message string) {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	status.Async.Current = nil
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, c.log)
}

func (c *digitalAssistantRuntimeClient) markAsyncOperation(
	resource *odav1beta1.DigitalAssistant,
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

func (c *digitalAssistantRuntimeClient) markCondition(
	resource *odav1beta1.DigitalAssistant,
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
	case condition == shared.Provisioning || condition == shared.Updating || (condition == shared.Terminating && shouldRequeue):
		status.Async.Current = &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           digitalAssistantCurrentOrFallbackPhase(resource, asyncPhaseForDigitalAssistantCondition(condition)),
			RawStatus:       rawState,
			NormalizedClass: shared.OSOKAsyncClassPending,
			Message:         message,
			UpdatedAt:       &now,
		}
	case condition == shared.Failed:
		status.Async.Current = &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           digitalAssistantCurrentOrFallbackPhase(resource, ""),
			RawStatus:       rawState,
			NormalizedClass: shared.OSOKAsyncClassFailed,
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

func validateDigitalAssistantDesiredSpec(resource *odav1beta1.DigitalAssistant) error {
	if resource == nil {
		return fmt.Errorf("DigitalAssistant resource is nil")
	}
	if strings.TrimSpace(resource.Spec.JsonData) != "" {
		_, err := digitalAssistantCreateDetailsFromJSON(resource.Spec.JsonData)
		return err
	}

	kind, err := normalizedDigitalAssistantKind(resource.Spec.Kind)
	if err != nil {
		return err
	}
	if _, err := normalizedDigitalAssistantMultilingualMode(resource.Spec.MultilingualMode); err != nil {
		return err
	}
	switch kind {
	case "NEW":
		if strings.TrimSpace(resource.Spec.Name) == "" {
			return fmt.Errorf("DigitalAssistant spec.name is required for NEW create")
		}
		if strings.TrimSpace(resource.Spec.DisplayName) == "" {
			return fmt.Errorf("DigitalAssistant spec.displayName is required for NEW create")
		}
	case "CLONE", "EXTEND":
		if strings.TrimSpace(resource.Spec.Id) == "" {
			return fmt.Errorf("DigitalAssistant spec.id is required for %s create", kind)
		}
		if strings.TrimSpace(resource.Spec.Name) == "" {
			return fmt.Errorf("DigitalAssistant spec.name is required for %s create", kind)
		}
		if strings.TrimSpace(resource.Spec.DisplayName) == "" {
			return fmt.Errorf("DigitalAssistant spec.displayName is required for %s create", kind)
		}
	case "VERSION":
		if strings.TrimSpace(resource.Spec.Id) == "" {
			return fmt.Errorf("DigitalAssistant spec.id is required for VERSION create")
		}
		if strings.TrimSpace(resource.Spec.Version) == "" {
			return fmt.Errorf("DigitalAssistant spec.version is required for VERSION create")
		}
	}
	return nil
}

func digitalAssistantOdaInstanceID(resource *odav1beta1.DigitalAssistant) (string, error) {
	if resource == nil {
		return "", fmt.Errorf("DigitalAssistant resource is nil")
	}
	odaInstanceID := strings.TrimSpace(resource.Annotations[digitalAssistantOdaInstanceIDAnnotation])
	if odaInstanceID == "" {
		return "", fmt.Errorf("DigitalAssistant requires metadata annotation %q with the parent ODA instance OCID", digitalAssistantOdaInstanceIDAnnotation)
	}
	return odaInstanceID, nil
}

func currentDigitalAssistantID(resource *odav1beta1.DigitalAssistant) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); id != "" {
		return id
	}
	return strings.TrimSpace(resource.Status.Id)
}

func currentDigitalAssistantWorkRequest(resource *odav1beta1.DigitalAssistant, fallback shared.OSOKAsyncPhase) (string, shared.OSOKAsyncPhase) {
	if resource == nil {
		return "", ""
	}
	return servicemanager.ResolveTrackedWorkRequest(&resource.Status.OsokStatus, resource, servicemanager.WorkRequestLegacyBridge{}, fallback)
}

func buildDigitalAssistantCreateDetails(resource *odav1beta1.DigitalAssistant) (odasdk.CreateDigitalAssistantDetails, error) {
	if raw := strings.TrimSpace(resource.Spec.JsonData); raw != "" {
		return digitalAssistantCreateDetailsFromJSON(raw)
	}

	spec := resource.Spec
	kind, err := normalizedDigitalAssistantKind(spec.Kind)
	if err != nil {
		return nil, err
	}
	commonFields, err := digitalAssistantCommonCreateFields(spec)
	if err != nil {
		return nil, err
	}

	switch kind {
	case "NEW":
		return odasdk.CreateNewDigitalAssistantDetails{
			Name:               common.String(strings.TrimSpace(spec.Name)),
			DisplayName:        common.String(strings.TrimSpace(spec.DisplayName)),
			Category:           commonFields.Category,
			Description:        commonFields.Description,
			PlatformVersion:    commonFields.PlatformVersion,
			PrimaryLanguageTag: commonFields.PrimaryLanguageTag,
			FreeformTags:       commonFields.FreeformTags,
			DefinedTags:        commonFields.DefinedTags,
			Version:            optionalString(spec.Version),
			NativeLanguageTags: slices.Clone(spec.NativeLanguageTags),
			MultilingualMode:   commonFields.MultilingualMode,
		}, nil
	case "VERSION":
		return odasdk.CreateDigitalAssistantVersionDetails{
			Id:                 common.String(strings.TrimSpace(spec.Id)),
			Version:            common.String(strings.TrimSpace(spec.Version)),
			Category:           commonFields.Category,
			Description:        commonFields.Description,
			PlatformVersion:    commonFields.PlatformVersion,
			PrimaryLanguageTag: commonFields.PrimaryLanguageTag,
			FreeformTags:       commonFields.FreeformTags,
			DefinedTags:        commonFields.DefinedTags,
			MultilingualMode:   commonFields.MultilingualMode,
		}, nil
	case "CLONE":
		return odasdk.CloneDigitalAssistantDetails{
			Id:                 common.String(strings.TrimSpace(spec.Id)),
			Name:               common.String(strings.TrimSpace(spec.Name)),
			DisplayName:        common.String(strings.TrimSpace(spec.DisplayName)),
			Category:           commonFields.Category,
			Description:        commonFields.Description,
			PlatformVersion:    commonFields.PlatformVersion,
			PrimaryLanguageTag: commonFields.PrimaryLanguageTag,
			FreeformTags:       commonFields.FreeformTags,
			DefinedTags:        commonFields.DefinedTags,
			Version:            optionalString(spec.Version),
			MultilingualMode:   commonFields.MultilingualMode,
		}, nil
	case "EXTEND":
		return odasdk.ExtendDigitalAssistantDetails{
			Id:                 common.String(strings.TrimSpace(spec.Id)),
			Name:               common.String(strings.TrimSpace(spec.Name)),
			DisplayName:        common.String(strings.TrimSpace(spec.DisplayName)),
			Category:           commonFields.Category,
			Description:        commonFields.Description,
			PlatformVersion:    commonFields.PlatformVersion,
			PrimaryLanguageTag: commonFields.PrimaryLanguageTag,
			FreeformTags:       commonFields.FreeformTags,
			DefinedTags:        commonFields.DefinedTags,
			Version:            optionalString(spec.Version),
			MultilingualMode:   commonFields.MultilingualMode,
		}, nil
	default:
		return nil, fmt.Errorf("DigitalAssistant create kind %q is not supported", kind)
	}
}

type digitalAssistantCommonFields struct {
	Category           *string
	Description        *string
	PlatformVersion    *string
	PrimaryLanguageTag *string
	FreeformTags       map[string]string
	DefinedTags        map[string]map[string]interface{}
	MultilingualMode   odasdk.BotMultilingualModeEnum
}

func digitalAssistantCommonCreateFields(spec odav1beta1.DigitalAssistantSpec) (digitalAssistantCommonFields, error) {
	mode, err := normalizedDigitalAssistantMultilingualMode(spec.MultilingualMode)
	if err != nil {
		return digitalAssistantCommonFields{}, err
	}
	return digitalAssistantCommonFields{
		Category:           optionalString(spec.Category),
		Description:        optionalString(spec.Description),
		PlatformVersion:    optionalString(spec.PlatformVersion),
		PrimaryLanguageTag: optionalString(spec.PrimaryLanguageTag),
		FreeformTags:       cloneStringMap(spec.FreeformTags),
		DefinedTags:        digitalAssistantDefinedTagsFromSpec(spec.DefinedTags),
		MultilingualMode:   odasdk.BotMultilingualModeEnum(mode),
	}, nil
}

func digitalAssistantCreateDetailsFromJSON(raw string) (odasdk.CreateDigitalAssistantDetails, error) {
	payload := []byte(strings.TrimSpace(raw))
	var discriminator struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(payload, &discriminator); err != nil {
		return nil, fmt.Errorf("decode DigitalAssistant jsonData discriminator: %w", err)
	}
	kind, err := normalizedDigitalAssistantKindForField(discriminator.Kind, "jsonData.kind")
	if err != nil {
		return nil, err
	}

	switch kind {
	case "NEW":
		var details odasdk.CreateNewDigitalAssistantDetails
		if err := json.Unmarshal(payload, &details); err != nil {
			return nil, fmt.Errorf("decode DigitalAssistant NEW jsonData: %w", err)
		}
		if stringValue(details.Name) == "" || stringValue(details.DisplayName) == "" {
			return nil, fmt.Errorf("DigitalAssistant NEW jsonData requires name and displayName")
		}
		return normalizeDigitalAssistantCreateDetails(details, true)
	case "VERSION":
		var details odasdk.CreateDigitalAssistantVersionDetails
		if err := json.Unmarshal(payload, &details); err != nil {
			return nil, fmt.Errorf("decode DigitalAssistant VERSION jsonData: %w", err)
		}
		if stringValue(details.Id) == "" || stringValue(details.Version) == "" {
			return nil, fmt.Errorf("DigitalAssistant VERSION jsonData requires id and version")
		}
		return normalizeDigitalAssistantCreateDetails(details, true)
	case "CLONE":
		var details odasdk.CloneDigitalAssistantDetails
		if err := json.Unmarshal(payload, &details); err != nil {
			return nil, fmt.Errorf("decode DigitalAssistant CLONE jsonData: %w", err)
		}
		if stringValue(details.Id) == "" || stringValue(details.Name) == "" || stringValue(details.DisplayName) == "" {
			return nil, fmt.Errorf("DigitalAssistant CLONE jsonData requires id, name, and displayName")
		}
		return normalizeDigitalAssistantCreateDetails(details, true)
	case "EXTEND":
		var details odasdk.ExtendDigitalAssistantDetails
		if err := json.Unmarshal(payload, &details); err != nil {
			return nil, fmt.Errorf("decode DigitalAssistant EXTEND jsonData: %w", err)
		}
		if stringValue(details.Id) == "" || stringValue(details.Name) == "" || stringValue(details.DisplayName) == "" {
			return nil, fmt.Errorf("DigitalAssistant EXTEND jsonData requires id, name, and displayName")
		}
		return normalizeDigitalAssistantCreateDetails(details, true)
	default:
		return nil, fmt.Errorf("DigitalAssistant create kind %q is not supported", kind)
	}
}

type digitalAssistantDesiredCreateFields struct {
	FromJSON           bool
	Kind               string
	Id                 string
	Name               string
	DisplayName        string
	Version            string
	Category           string
	Description        string
	PlatformVersion    string
	MultilingualMode   string
	PrimaryLanguageTag string
	NativeLanguageTags []string
	FreeformTags       map[string]string
	DefinedTags        map[string]map[string]interface{}
}

func digitalAssistantDesiredCreateFieldsForResource(resource *odav1beta1.DigitalAssistant) (digitalAssistantDesiredCreateFields, error) {
	if resource == nil {
		return digitalAssistantDesiredCreateFields{}, fmt.Errorf("DigitalAssistant resource is nil")
	}
	details, err := buildDigitalAssistantCreateDetails(resource)
	if err != nil {
		return digitalAssistantDesiredCreateFields{}, err
	}
	return digitalAssistantDesiredCreateFieldsFromDetails(details, strings.TrimSpace(resource.Spec.JsonData) != "")
}

func digitalAssistantDesiredCreateFieldsFromDetails(details odasdk.CreateDigitalAssistantDetails, fromJSON bool) (digitalAssistantDesiredCreateFields, error) {
	details, err := normalizeDigitalAssistantCreateDetails(details, fromJSON)
	if err != nil {
		return digitalAssistantDesiredCreateFields{}, err
	}
	desired := digitalAssistantDesiredCreateFields{
		FromJSON:           fromJSON,
		Category:           stringValue(details.GetCategory()),
		Description:        stringValue(details.GetDescription()),
		PlatformVersion:    stringValue(details.GetPlatformVersion()),
		MultilingualMode:   string(details.GetMultilingualMode()),
		PrimaryLanguageTag: stringValue(details.GetPrimaryLanguageTag()),
		FreeformTags:       cloneStringMap(details.GetFreeformTags()),
		DefinedTags:        cloneNestedMap(details.GetDefinedTags()),
	}

	switch typed := details.(type) {
	case odasdk.CreateNewDigitalAssistantDetails:
		desired.Kind = "NEW"
		desired.Name = stringValue(typed.Name)
		desired.DisplayName = stringValue(typed.DisplayName)
		desired.Version = stringValue(typed.Version)
		desired.NativeLanguageTags = slices.Clone(typed.NativeLanguageTags)
	case odasdk.CreateDigitalAssistantVersionDetails:
		desired.Kind = "VERSION"
		desired.Id = stringValue(typed.Id)
		desired.Version = stringValue(typed.Version)
	case odasdk.CloneDigitalAssistantDetails:
		desired.Kind = "CLONE"
		desired.Id = stringValue(typed.Id)
		desired.Name = stringValue(typed.Name)
		desired.DisplayName = stringValue(typed.DisplayName)
		desired.Version = stringValue(typed.Version)
	case odasdk.ExtendDigitalAssistantDetails:
		desired.Kind = "EXTEND"
		desired.Id = stringValue(typed.Id)
		desired.Name = stringValue(typed.Name)
		desired.DisplayName = stringValue(typed.DisplayName)
		desired.Version = stringValue(typed.Version)
	default:
		source := "spec.kind"
		if fromJSON {
			source = "jsonData.kind"
		}
		return digitalAssistantDesiredCreateFields{}, fmt.Errorf("DigitalAssistant %s resolved to unsupported create body %T", source, details)
	}
	return desired, nil
}

func normalizeDigitalAssistantCreateDetails(details odasdk.CreateDigitalAssistantDetails, fromJSON bool) (odasdk.CreateDigitalAssistantDetails, error) {
	field := "spec.multilingualMode"
	if fromJSON {
		field = "jsonData.multilingualMode"
	}

	switch typed := details.(type) {
	case odasdk.CreateNewDigitalAssistantDetails:
		mode, err := normalizedDigitalAssistantMultilingualModeForField(string(typed.MultilingualMode), field)
		if err != nil {
			return nil, err
		}
		typed.MultilingualMode = odasdk.BotMultilingualModeEnum(mode)
		return typed, nil
	case odasdk.CreateDigitalAssistantVersionDetails:
		mode, err := normalizedDigitalAssistantMultilingualModeForField(string(typed.MultilingualMode), field)
		if err != nil {
			return nil, err
		}
		typed.MultilingualMode = odasdk.BotMultilingualModeEnum(mode)
		return typed, nil
	case odasdk.CloneDigitalAssistantDetails:
		mode, err := normalizedDigitalAssistantMultilingualModeForField(string(typed.MultilingualMode), field)
		if err != nil {
			return nil, err
		}
		typed.MultilingualMode = odasdk.BotMultilingualModeEnum(mode)
		return typed, nil
	case odasdk.ExtendDigitalAssistantDetails:
		mode, err := normalizedDigitalAssistantMultilingualModeForField(string(typed.MultilingualMode), field)
		if err != nil {
			return nil, err
		}
		typed.MultilingualMode = odasdk.BotMultilingualModeEnum(mode)
		return typed, nil
	default:
		return details, nil
	}
}

func buildDigitalAssistantUpdateDetails(resource *odav1beta1.DigitalAssistant, current odasdk.DigitalAssistant) (odasdk.UpdateDigitalAssistantDetails, bool) {
	spec := resource.Spec
	details := odasdk.UpdateDigitalAssistantDetails{}
	updateNeeded := false

	if spec.Category != "" && spec.Category != stringValue(current.Category) {
		details.Category = common.String(spec.Category)
		updateNeeded = true
	}
	if spec.Description != "" && spec.Description != stringValue(current.Description) {
		details.Description = common.String(spec.Description)
		updateNeeded = true
	}
	if spec.FreeformTags != nil && !maps.Equal(spec.FreeformTags, current.FreeformTags) {
		details.FreeformTags = maps.Clone(spec.FreeformTags)
		updateNeeded = true
	}
	if spec.DefinedTags != nil {
		desired := digitalAssistantDefinedTagsFromSpec(spec.DefinedTags)
		if !digitalAssistantJSONEqual(desired, current.DefinedTags) {
			details.DefinedTags = desired
			updateNeeded = true
		}
	}
	return details, updateNeeded
}

func validateDigitalAssistantCreateOnlyDrift(resource *odav1beta1.DigitalAssistant, current odasdk.DigitalAssistant) error {
	desired, err := digitalAssistantDesiredCreateFieldsForResource(resource)
	if err != nil {
		return err
	}
	drift := []string{}
	if desired.Name != "" && stringValue(current.Name) != "" && desired.Name != stringValue(current.Name) {
		drift = append(drift, desired.fieldPath("name"))
	}
	if desired.DisplayName != "" && stringValue(current.DisplayName) != "" && desired.DisplayName != stringValue(current.DisplayName) {
		drift = append(drift, desired.fieldPath("displayName"))
	}
	if desired.Version != "" && stringValue(current.Version) != "" && desired.Version != stringValue(current.Version) {
		drift = append(drift, desired.fieldPath("version"))
	}
	if desired.PlatformVersion != "" && stringValue(current.PlatformVersion) != "" && desired.PlatformVersion != stringValue(current.PlatformVersion) {
		drift = append(drift, desired.fieldPath("platformVersion"))
	}
	currentMultilingualMode := ""
	if desired.MultilingualMode != "" && current.MultilingualMode != "" {
		currentMultilingualMode, err = normalizedDigitalAssistantMultilingualModeForField(string(current.MultilingualMode), "observed multilingualMode")
		if err != nil {
			return err
		}
	}
	if desired.MultilingualMode != "" && currentMultilingualMode != "" && desired.MultilingualMode != currentMultilingualMode {
		drift = append(drift, desired.fieldPath("multilingualMode"))
	}
	if desired.PrimaryLanguageTag != "" && stringValue(current.PrimaryLanguageTag) != "" && desired.PrimaryLanguageTag != stringValue(current.PrimaryLanguageTag) {
		drift = append(drift, desired.fieldPath("primaryLanguageTag"))
	}
	if len(desired.NativeLanguageTags) > 0 && len(current.NativeLanguageTags) > 0 && !slices.Equal(desired.NativeLanguageTags, current.NativeLanguageTags) {
		drift = append(drift, desired.fieldPath("nativeLanguageTags"))
	}
	if desired.Id != "" && stringValue(current.BaseId) != "" && desired.Id != stringValue(current.BaseId) {
		drift = append(drift, desired.fieldPath("id"))
	}
	if desired.FromJSON {
		drift = append(drift, validateDigitalAssistantJSONDrift(desired, current)...)
	}
	if len(drift) == 0 {
		return nil
	}
	return fmt.Errorf("DigitalAssistant create-only field drift detected for %s; recreate the resource instead of updating immutable fields", strings.Join(drift, ", "))
}

func validateDigitalAssistantJSONDrift(desired digitalAssistantDesiredCreateFields, current odasdk.DigitalAssistant) []string {
	drift := []string{}
	if desired.Category != "" && stringValue(current.Category) != "" && desired.Category != stringValue(current.Category) {
		drift = append(drift, desired.fieldPath("category"))
	}
	if desired.Description != "" && stringValue(current.Description) != "" && desired.Description != stringValue(current.Description) {
		drift = append(drift, desired.fieldPath("description"))
	}
	if len(desired.FreeformTags) > 0 && len(current.FreeformTags) > 0 && !maps.Equal(desired.FreeformTags, current.FreeformTags) {
		drift = append(drift, desired.fieldPath("freeformTags"))
	}
	if len(desired.DefinedTags) > 0 && len(current.DefinedTags) > 0 && !digitalAssistantJSONEqual(desired.DefinedTags, current.DefinedTags) {
		drift = append(drift, desired.fieldPath("definedTags"))
	}
	return drift
}

func (d digitalAssistantDesiredCreateFields) fieldPath(field string) string {
	if d.FromJSON {
		return "jsonData." + field
	}
	return field
}

func digitalAssistantDesiredNameVersion(resource *odav1beta1.DigitalAssistant) (string, string) {
	if resource == nil {
		return "", ""
	}
	desired, err := digitalAssistantDesiredCreateFieldsForResource(resource)
	if err != nil {
		return strings.TrimSpace(resource.Spec.Name), strings.TrimSpace(resource.Spec.Version)
	}
	return desired.Name, desired.Version
}

func digitalAssistantSummaryMatches(resource *odav1beta1.DigitalAssistant, item odasdk.DigitalAssistantSummary) bool {
	name, version := digitalAssistantDesiredNameVersion(resource)
	if name != "" && name != stringValue(item.Name) {
		return false
	}
	if version != "" && version != stringValue(item.Version) {
		return false
	}
	return true
}

func digitalAssistantLifecycleBlocksMutation(state odasdk.LifecycleStateEnum) bool {
	switch normalizedDigitalAssistantLifecycle(state) {
	case string(odasdk.LifecycleStateCreating),
		string(odasdk.LifecycleStateUpdating),
		string(odasdk.LifecycleStateDeleting),
		string(odasdk.LifecycleStateDeleted),
		string(odasdk.LifecycleStateFailed),
		string(odasdk.LifecycleStateInactive):
		return true
	default:
		return false
	}
}

func isDigitalAssistantReadNotFound(err error) bool {
	classification := errorutil.ClassifyDeleteError(err)
	return classification.IsUnambiguousNotFound() || classification.IsAuthShapedNotFound()
}

func digitalAssistantIDFromSDK(current odasdk.DigitalAssistant) string {
	return strings.TrimSpace(stringValue(current.Id))
}

func digitalAssistantIDFromWorkRequest(workRequest odasdk.WorkRequest, phase shared.OSOKAsyncPhase, fallback string) string {
	if id := strings.TrimSpace(stringValue(workRequest.ResourceId)); id != "" {
		return id
	}
	for _, resource := range workRequest.Resources {
		if !digitalAssistantWorkRequestResourceActionMatchesPhase(resource.ResourceAction, phase) {
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

func digitalAssistantWorkRequestResourceActionMatchesPhase(action odasdk.WorkRequestResourceResourceActionEnum, phase shared.OSOKAsyncPhase) bool {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return action == odasdk.WorkRequestResourceResourceActionCreate
	case shared.OSOKAsyncPhaseUpdate:
		return action == odasdk.WorkRequestResourceResourceActionUpdate
	case shared.OSOKAsyncPhaseDelete:
		return action == odasdk.WorkRequestResourceResourceActionDelete
	default:
		return false
	}
}

func digitalAssistantWorkRequestAsyncOperation(
	resource *odav1beta1.DigitalAssistant,
	workRequest odasdk.WorkRequest,
	explicitPhase shared.OSOKAsyncPhase,
) (*shared.OSOKAsyncOperation, error) {
	var status *shared.OSOKStatus
	if resource != nil {
		status = &resource.Status.OsokStatus
	}

	current, err := servicemanager.BuildWorkRequestAsyncOperation(status, digitalAssistantWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:       string(workRequest.Status),
		RawAction:       string(workRequest.RequestAction),
		WorkRequestID:   stringValue(workRequest.Id),
		PercentComplete: workRequest.PercentComplete,
		Message:         stringValue(workRequest.StatusMessage),
		FallbackPhase:   digitalAssistantCurrentOrFallbackPhase(resource, explicitPhase),
	})
	if err != nil {
		return nil, err
	}
	current.Message = digitalAssistantWorkRequestMessage(current.Phase, workRequest)
	return current, nil
}

func digitalAssistantWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest odasdk.WorkRequest) string {
	if message := strings.TrimSpace(stringValue(workRequest.StatusMessage)); message != "" {
		return message
	}
	action := "reconcile"
	if phase == shared.OSOKAsyncPhaseCreate {
		action = "create"
	}
	if id := strings.TrimSpace(stringValue(workRequest.Id)); id != "" {
		return fmt.Sprintf("DigitalAssistant %s work request %s is %s", action, id, workRequest.Status)
	}
	return fmt.Sprintf("DigitalAssistant %s work request is %s", action, workRequest.Status)
}

func digitalAssistantLifecycleMessage(current odasdk.DigitalAssistant, fallback string) string {
	name := strings.TrimSpace(stringValue(current.Name))
	if name == "" {
		name = strings.TrimSpace(stringValue(current.Id))
	}
	if name != "" {
		return fmt.Sprintf("DigitalAssistant %s is %s", name, current.LifecycleState)
	}
	return fallback
}

func normalizedDigitalAssistantLifecycle(state odasdk.LifecycleStateEnum) string {
	return strings.ToUpper(strings.TrimSpace(string(state)))
}

func normalizedDigitalAssistantKind(value string) (string, error) {
	return normalizedDigitalAssistantKindForField(value, "spec.kind")
}

func normalizedDigitalAssistantKindForField(value string, field string) (string, error) {
	kind := strings.ToUpper(strings.TrimSpace(value))
	if kind == "" {
		kind = "NEW"
	}
	switch kind {
	case "NEW", "VERSION", "CLONE", "EXTEND":
		return kind, nil
	default:
		return "", fmt.Errorf("DigitalAssistant %s %q is not supported", field, value)
	}
}

func normalizedDigitalAssistantMultilingualMode(value string) (string, error) {
	return normalizedDigitalAssistantMultilingualModeForField(value, "spec.multilingualMode")
}

func normalizedDigitalAssistantMultilingualModeForField(value string, field string) (string, error) {
	mode := strings.ToUpper(strings.TrimSpace(value))
	if mode == "" {
		return "", nil
	}
	if _, ok := odasdk.GetMappingBotMultilingualModeEnum(mode); !ok {
		return "", fmt.Errorf("DigitalAssistant %s %q is not supported", field, value)
	}
	return mode, nil
}

func asyncPhaseForDigitalAssistantCondition(condition shared.OSOKConditionType) shared.OSOKAsyncPhase {
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

func digitalAssistantCurrentOrFallbackPhase(resource *odav1beta1.DigitalAssistant, fallback shared.OSOKAsyncPhase) shared.OSOKAsyncPhase {
	if resource != nil && resource.Status.OsokStatus.Async.Current != nil && resource.Status.OsokStatus.Async.Current.Phase != "" {
		return resource.Status.OsokStatus.Async.Current.Phase
	}
	return fallback
}

func digitalAssistantRetryToken(resource *odav1beta1.DigitalAssistant) *string {
	if resource == nil {
		return nil
	}
	if uid := strings.TrimSpace(string(resource.UID)); uid != "" {
		return common.String(uid)
	}
	sum := sha256.Sum256([]byte(strings.TrimSpace(resource.Namespace) + "/" + strings.TrimSpace(resource.Name)))
	return common.String(fmt.Sprintf("%x", sum[:16]))
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return common.String(value)
}

func digitalAssistantDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
	if spec == nil {
		return nil
	}
	return *util.ConvertToOciDefinedTags(&spec)
}

func digitalAssistantStatusDefinedTags(input map[string]map[string]interface{}) map[string]shared.MapValue {
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

func digitalAssistantJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}

func digitalAssistantSDKTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Time.Format(time.RFC3339Nano)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	return maps.Clone(input)
}

func cloneNestedMap(input map[string]map[string]interface{}) map[string]map[string]interface{} {
	if len(input) == 0 {
		return nil
	}
	output := make(map[string]map[string]interface{}, len(input))
	for key, value := range input {
		output[key] = maps.Clone(value)
	}
	return output
}

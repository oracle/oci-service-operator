/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package skill

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	odasdk "github.com/oracle/oci-go-sdk/v65/oda"
	odav1beta1 "github.com/oracle/oci-service-operator/api/oda/v1beta1"
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
	skillOdaInstanceIDAnnotation       = "oda.oracle.com/oda-instance-id"
	skillLegacyOdaInstanceIDAnnotation = "oda.oracle.com/odaInstanceId"

	skillKindNew     = "NEW"
	skillKindClone   = "CLONE"
	skillKindExtend  = "EXTEND"
	skillKindVersion = "VERSION"

	skillRequeueDuration = time.Minute
)

type skillOCIClient interface {
	CreateSkill(context.Context, odasdk.CreateSkillRequest) (odasdk.CreateSkillResponse, error)
	GetSkill(context.Context, odasdk.GetSkillRequest) (odasdk.GetSkillResponse, error)
	ListSkills(context.Context, odasdk.ListSkillsRequest) (odasdk.ListSkillsResponse, error)
	UpdateSkill(context.Context, odasdk.UpdateSkillRequest) (odasdk.UpdateSkillResponse, error)
	DeleteSkill(context.Context, odasdk.DeleteSkillRequest) (odasdk.DeleteSkillResponse, error)
	GetWorkRequest(context.Context, odasdk.GetWorkRequestRequest) (odasdk.GetWorkRequestResponse, error)
}

type skillSDKClients struct {
	management odasdk.ManagementClient
	oda        odasdk.OdaClient
}

type skillDesiredState struct {
	odaInstanceID      string
	kind               string
	kindSet            bool
	id                 string
	idSet              bool
	name               string
	displayName        string
	version            string
	category           string
	categorySet        bool
	description        string
	descriptionSet     bool
	platformVersion    string
	dialogVersion      string
	multilingualMode   string
	primaryLanguageTag string
	freeformTags       map[string]string
	definedTags        map[string]shared.MapValue
	nativeLanguageTags []string
}

type skillRuntimeClient struct {
	delegate SkillServiceClient
	client   skillOCIClient
	log      loggerutil.OSOKLogger
	initErr  error
}

var _ SkillServiceClient = (*skillRuntimeClient)(nil)

func init() {
	registerSkillRuntimeHooksMutator(func(manager *SkillServiceManager, hooks *SkillRuntimeHooks) {
		client, err := newSkillSDKClient(manager)
		applySkillRuntimeHooks(manager, hooks, client, err)
	})
}

func newSkillSDKClient(manager *SkillServiceManager) (skillOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("Skill service manager is nil")
	}
	managementClient, err := odasdk.NewManagementClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	odaClient, err := odasdk.NewOdaClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return skillSDKClients{
		management: managementClient,
		oda:        odaClient,
	}, nil
}

func (c skillSDKClients) CreateSkill(ctx context.Context, request odasdk.CreateSkillRequest) (odasdk.CreateSkillResponse, error) {
	return c.management.CreateSkill(ctx, request)
}

func (c skillSDKClients) GetSkill(ctx context.Context, request odasdk.GetSkillRequest) (odasdk.GetSkillResponse, error) {
	return c.management.GetSkill(ctx, request)
}

func (c skillSDKClients) ListSkills(ctx context.Context, request odasdk.ListSkillsRequest) (odasdk.ListSkillsResponse, error) {
	return c.management.ListSkills(ctx, request)
}

func (c skillSDKClients) UpdateSkill(ctx context.Context, request odasdk.UpdateSkillRequest) (odasdk.UpdateSkillResponse, error) {
	return c.management.UpdateSkill(ctx, request)
}

func (c skillSDKClients) DeleteSkill(ctx context.Context, request odasdk.DeleteSkillRequest) (odasdk.DeleteSkillResponse, error) {
	return c.management.DeleteSkill(ctx, request)
}

func (c skillSDKClients) GetWorkRequest(ctx context.Context, request odasdk.GetWorkRequestRequest) (odasdk.GetWorkRequestResponse, error) {
	return c.oda.GetWorkRequest(ctx, request)
}

func applySkillRuntimeHooks(
	manager *SkillServiceManager,
	hooks *SkillRuntimeHooks,
	client skillOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}
	hooks.Semantics = newSkillRuntimeSemantics()
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate SkillServiceClient) SkillServiceClient {
		return newSkillRuntimeClient(manager, delegate, client, initErr)
	})
}

func newSkillRuntimeClient(
	manager *SkillServiceManager,
	delegate SkillServiceClient,
	client skillOCIClient,
	initErr error,
) *skillRuntimeClient {
	runtimeClient := &skillRuntimeClient{
		delegate: delegate,
		client:   client,
		initErr:  initErr,
	}
	if manager != nil {
		runtimeClient.log = manager.Log
	}
	return runtimeClient
}

func newSkillServiceClientWithOCIClient(log loggerutil.OSOKLogger, client skillOCIClient) SkillServiceClient {
	manager := &SkillServiceManager{Log: log}
	hooks := SkillRuntimeHooks{}
	applySkillRuntimeHooks(manager, &hooks, client, nil)
	delegate := defaultSkillServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*odav1beta1.Skill](
			buildSkillGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapSkillGeneratedClient(hooks, delegate)
}

func newSkillRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "oda",
		FormalSlug:    "skill",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "workrequest+lifecycle",
			Runtime:              "handwritten",
			FormalClassification: "workrequest+lifecycle",
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
			MatchFields:        []string{"name", "version"},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{"category", "description", "freeformTags", "definedTags"},
			Mutable:         []string{"category", "description", "freeformTags", "definedTags"},
			ForceNew: []string{
				"kind",
				"id",
				"name",
				"displayName",
				"version",
				"platformVersion",
				"dialogVersion",
				"multilingualMode",
				"primaryLanguageTag",
				"nativeLanguageTags",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "resource-local", EntityType: "Skill", Action: "CreateSkill"}},
			Update: []generatedruntime.Hook{{Helper: "resource-local", EntityType: "Skill", Action: "UpdateSkill"}},
			Delete: []generatedruntime.Hook{{Helper: "resource-local", EntityType: "Skill", Action: "DeleteSkill"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "work-request-then-read",
			Hooks:    []generatedruntime.Hook{{Helper: "resource-local", EntityType: "Skill", Action: "GetWorkRequest"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "resource-local", EntityType: "Skill", Action: "GetSkill"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "resource-local", EntityType: "Skill", Action: "GetSkill"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{
			{
				Phase:            string(shared.OSOKAsyncPhaseCreate),
				MethodName:       "GetWorkRequest",
				RequestTypeName:  "GetWorkRequestRequest",
				ResponseTypeName: "GetWorkRequestResponse",
			},
		},
	}
}

func (c *skillRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *odav1beta1.Skill,
	_ ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if err := c.ensureClient(); err != nil {
		return c.fail(resource, err)
	}
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("Skill resource is nil")
	}

	desired, err := resolveDesiredSkillState(resource)
	if err != nil {
		return c.fail(resource, err)
	}
	if workRequestID, phase := currentSkillWorkRequest(resource); workRequestID != "" {
		return c.pollCreateWorkRequest(ctx, resource, desired, workRequestID, phase)
	}

	current, found, err := c.resolveCurrent(ctx, resource, desired)
	if err != nil {
		return c.fail(resource, err)
	}
	if !found {
		return c.create(ctx, resource, desired)
	}

	if skillLifecycleBlocksMutation(normalizeSkillLifecycle(current.LifecycleState)) {
		return c.finishWithLifecycle(resource, current, shared.OSOKAsyncPhaseUpdate), nil
	}
	if err := validateSkillCreateOnlyDrift(desired, current); err != nil {
		return c.fail(resource, err)
	}

	updateDetails, updateNeeded := buildSkillUpdateDetails(desired, current)
	if !updateNeeded {
		return c.finishWithLifecycle(resource, current, shared.OSOKAsyncPhaseUpdate), nil
	}

	currentID := skillIDFromSDK(current)
	if currentID == "" {
		currentID = currentSkillID(resource)
	}
	if currentID == "" {
		return c.fail(resource, fmt.Errorf("Skill update could not resolve OCI resource ID"))
	}
	response, err := c.client.UpdateSkill(ctx, odasdk.UpdateSkillRequest{
		OdaInstanceId:      common.String(desired.odaInstanceID),
		SkillId:            common.String(currentID),
		UpdateSkillDetails: updateDetails,
	})
	if err != nil {
		return c.fail(resource, fmt.Errorf("update Skill %q: %w", skillIdentityLabel(desired), err))
	}
	c.recordResponseRequestID(resource, response)

	refreshed, err := c.read(ctx, desired.odaInstanceID, currentID)
	if err != nil {
		if skillIsNotFound(err) {
			return c.markPendingLifecycleOperation(
				resource,
				shared.OSOKAsyncPhaseUpdate,
				"OCI Skill update request accepted",
				string(odasdk.LifecycleStateUpdating),
			), nil
		}
		return c.fail(resource, fmt.Errorf("confirm Skill %q update: %w", skillIdentityLabel(desired), err))
	}
	return c.finishWithLifecycle(resource, refreshed, shared.OSOKAsyncPhaseUpdate), nil
}

func (c *skillRuntimeClient) Delete(ctx context.Context, resource *odav1beta1.Skill) (bool, error) {
	if err := c.ensureClient(); err != nil {
		return false, err
	}
	if resource == nil {
		return false, fmt.Errorf("Skill resource is nil")
	}

	odaInstanceID := skillOdaInstanceID(resource)
	currentID := currentSkillID(resource)
	if odaInstanceID == "" {
		if currentID == "" {
			c.markDeleted(resource, "No tracked OCI Skill identity recorded; skipping OCI delete")
			return true, nil
		}
		err := fmt.Errorf(
			"Skill delete requires metadata annotation %q with the parent ODA instance OCID",
			skillOdaInstanceIDAnnotation,
		)
		return false, c.markFailure(resource, err)
	}

	var current odasdk.Skill
	var found bool
	var err error
	if currentID != "" {
		current, err = c.read(ctx, odaInstanceID, currentID)
		if err != nil {
			if skillIsNotFound(err) {
				c.recordErrorRequestID(resource, err)
				c.markDeleted(resource, "OCI Skill deleted")
				return true, nil
			}
			return false, fmt.Errorf("read Skill %q before delete: %w", currentID, err)
		}
		found = true
	} else {
		identity := deleteSkillIdentity(resource, odaInstanceID)
		if !identity.hasLookupKey() {
			resolved, resolvedOdaInstanceID, handled, err := c.resolvePendingCreateWorkRequestForDelete(ctx, resource, identity)
			if handled {
				if err != nil || skillIDFromSDK(resolved) == "" {
					return false, err
				}
				current = resolved
				currentID = skillIDFromSDK(current)
				if resolvedOdaInstanceID != "" {
					odaInstanceID = resolvedOdaInstanceID
				}
				found = true
			} else {
				c.markDeleted(resource, "No tracked OCI Skill identity recorded; skipping OCI delete")
				return true, nil
			}
		}
		if !found {
			current, found, err = c.resolveByList(ctx, identity)
			if err != nil {
				return false, fmt.Errorf("resolve Skill before delete: %w", err)
			}
		}
		if !found {
			resolved, resolvedOdaInstanceID, handled, err := c.resolvePendingCreateWorkRequestForDelete(ctx, resource, identity)
			if handled {
				if err != nil || skillIDFromSDK(resolved) == "" {
					return false, err
				}
				current = resolved
				currentID = skillIDFromSDK(current)
				if resolvedOdaInstanceID != "" {
					odaInstanceID = resolvedOdaInstanceID
				}
				found = true
			} else {
				c.markDeleted(resource, "OCI Skill deleted")
				return true, nil
			}
		}
		if !found {
			c.markDeleted(resource, "No tracked OCI Skill identity recorded; skipping OCI delete")
			return true, nil
		}
		currentID = skillIDFromSDK(current)
	}
	if err := c.projectStatus(resource, current); err != nil {
		return false, err
	}

	switch normalizeSkillLifecycle(current.LifecycleState) {
	case string(odasdk.LifecycleStateDeleted):
		c.markDeleted(resource, "OCI Skill deleted")
		return true, nil
	case string(odasdk.LifecycleStateDeleting):
		return false, c.markTerminating(resource, "OCI Skill delete is in progress", string(current.LifecycleState))
	}

	response, err := c.client.DeleteSkill(ctx, odasdk.DeleteSkillRequest{
		OdaInstanceId: common.String(odaInstanceID),
		SkillId:       common.String(currentID),
	})
	if err != nil {
		if skillIsNotFound(err) {
			c.recordErrorRequestID(resource, err)
			c.markDeleted(resource, "OCI Skill deleted")
			return true, nil
		}
		return false, fmt.Errorf("delete Skill %q: %w", currentID, err)
	}
	c.recordResponseRequestID(resource, response)

	refreshed, err := c.read(ctx, odaInstanceID, currentID)
	if err != nil {
		if skillIsNotFound(err) {
			c.recordErrorRequestID(resource, err)
			c.markDeleted(resource, "OCI Skill deleted")
			return true, nil
		}
		return false, fmt.Errorf("confirm Skill %q delete: %w", currentID, err)
	}
	if err := c.projectStatus(resource, refreshed); err != nil {
		return false, err
	}
	if normalizeSkillLifecycle(refreshed.LifecycleState) == string(odasdk.LifecycleStateDeleted) {
		c.markDeleted(resource, "OCI Skill deleted")
		return true, nil
	}

	return false, c.markTerminating(resource, "OCI Skill delete is in progress", string(refreshed.LifecycleState))
}

func (c *skillRuntimeClient) resolvePendingCreateWorkRequestForDelete(
	ctx context.Context,
	resource *odav1beta1.Skill,
	identity skillDesiredState,
) (odasdk.Skill, string, bool, error) {
	workRequestID, phase := currentSkillCreateWorkRequest(resource)
	if workRequestID == "" {
		return odasdk.Skill{}, "", false, nil
	}

	response, err := c.client.GetWorkRequest(ctx, odasdk.GetWorkRequestRequest{
		WorkRequestId: common.String(workRequestID),
	})
	if err != nil {
		return odasdk.Skill{}, "", true, fmt.Errorf("get Skill create work request %q before delete: %w", workRequestID, err)
	}
	c.recordResponseRequestID(resource, response)

	workRequest := response.WorkRequest
	current, err := c.buildWorkRequestOperation(resource, workRequest, workRequestID, phase)
	if err != nil {
		return odasdk.Skill{}, "", true, c.markFailure(resource, err)
	}
	switch current.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		_ = c.markWorkRequestOperation(resource, current)
		return odasdk.Skill{}, "", true, nil
	case shared.OSOKAsyncClassSucceeded:
		if current != nil {
			_ = c.markWorkRequestOperation(resource, current)
		}
		return c.resolveSucceededCreateWorkRequestForDelete(ctx, resource, identity, workRequest, workRequestID)
	default:
		_ = c.markWorkRequestOperation(resource, current)
		return odasdk.Skill{}, "", true, nil
	}
}

func (c *skillRuntimeClient) resolveSucceededCreateWorkRequestForDelete(
	ctx context.Context,
	resource *odav1beta1.Skill,
	identity skillDesiredState,
	workRequest odasdk.WorkRequest,
	workRequestID string,
) (odasdk.Skill, string, bool, error) {
	odaInstanceID := strings.TrimSpace(identity.odaInstanceID)
	if odaInstanceID == "" {
		odaInstanceID = strings.TrimSpace(stringValue(workRequest.OdaInstanceId))
	}

	skillID := skillIDFromWorkRequest(workRequest)
	if skillID != "" {
		if odaInstanceID == "" {
			err := fmt.Errorf("Skill create work request %q resolved Skill %q but no parent ODA instance ID is available for delete", workRequestID, skillID)
			return odasdk.Skill{}, "", true, c.markFailure(resource, err)
		}
		refreshed, err := c.read(ctx, odaInstanceID, skillID)
		if err == nil {
			return refreshed, odaInstanceID, true, nil
		}
		if !skillIsNotFound(err) {
			return odasdk.Skill{}, "", true, fmt.Errorf("read Skill %q after create work request before delete: %w", skillID, err)
		}
	}

	if identity.odaInstanceID == "" {
		identity.odaInstanceID = odaInstanceID
	}
	if identity.hasLookupKey() && identity.odaInstanceID != "" {
		refreshed, found, err := c.resolveByList(ctx, identity)
		if err != nil {
			return odasdk.Skill{}, "", true, fmt.Errorf("resolve Skill after create work request before delete: %w", err)
		}
		if found {
			return refreshed, identity.odaInstanceID, true, nil
		}
	}

	err := fmt.Errorf("Skill create work request %q succeeded but no Skill identity could be resolved during delete", workRequestID)
	return odasdk.Skill{}, "", true, c.markFailure(resource, err)
}

func (c *skillRuntimeClient) create(
	ctx context.Context,
	resource *odav1beta1.Skill,
	desired skillDesiredState,
) (servicemanager.OSOKResponse, error) {
	if err := validateSkillCreateInputs(desired); err != nil {
		return c.fail(resource, err)
	}

	body, err := buildSkillCreateDetails(desired)
	if err != nil {
		return c.fail(resource, err)
	}
	request := odasdk.CreateSkillRequest{
		OdaInstanceId:      common.String(desired.odaInstanceID),
		CreateSkillDetails: body,
	}
	if resource.UID != "" {
		request.OpcRetryToken = common.String(string(resource.UID))
	}

	response, err := c.client.CreateSkill(ctx, request)
	if err != nil {
		return c.fail(resource, fmt.Errorf("create Skill %q: %w", skillIdentityLabel(desired), err))
	}
	c.recordResponseRequestID(resource, response)

	if workRequestID := stringValue(response.OpcWorkRequestId); workRequestID != "" {
		return c.markPendingWorkRequest(resource, workRequestID, shared.OSOKAsyncPhaseCreate, "OCI Skill create work request accepted"), nil
	}

	current, found, err := c.resolveByListForDesired(ctx, desired)
	if err != nil {
		return c.fail(resource, fmt.Errorf("confirm Skill %q create: %w", skillIdentityLabel(desired), err))
	}
	if !found {
		return c.markPendingLifecycleOperation(
			resource,
			shared.OSOKAsyncPhaseCreate,
			"OCI Skill create request accepted",
			string(odasdk.LifecycleStateCreating),
		), nil
	}
	return c.finishWithLifecycle(resource, current, shared.OSOKAsyncPhaseCreate), nil
}

func (c *skillRuntimeClient) pollCreateWorkRequest(
	ctx context.Context,
	resource *odav1beta1.Skill,
	desired skillDesiredState,
	workRequestID string,
	phase shared.OSOKAsyncPhase,
) (servicemanager.OSOKResponse, error) {
	response, err := c.client.GetWorkRequest(ctx, odasdk.GetWorkRequestRequest{
		WorkRequestId: common.String(workRequestID),
	})
	if err != nil {
		return c.fail(resource, fmt.Errorf("get Skill work request %q: %w", workRequestID, err))
	}
	c.recordResponseRequestID(resource, response)

	workRequest := response.WorkRequest
	current, err := c.buildWorkRequestOperation(resource, workRequest, workRequestID, phase)
	if err != nil {
		return c.fail(resource, err)
	}
	switch current.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		return c.markWorkRequestOperation(resource, current), nil
	case shared.OSOKAsyncClassSucceeded:
		return c.finishSucceededCreateWorkRequest(ctx, resource, desired, workRequest, current)
	default:
		return c.markWorkRequestOperation(resource, current), nil
	}
}

func (c *skillRuntimeClient) finishSucceededCreateWorkRequest(
	ctx context.Context,
	resource *odav1beta1.Skill,
	desired skillDesiredState,
	workRequest odasdk.WorkRequest,
	current *shared.OSOKAsyncOperation,
) (servicemanager.OSOKResponse, error) {
	if current != nil {
		_ = c.markWorkRequestOperation(resource, current)
	}

	skillID := skillIDFromWorkRequest(workRequest)
	if skillID != "" {
		refreshed, err := c.read(ctx, desired.odaInstanceID, skillID)
		if err == nil {
			return c.finishWithLifecycle(resource, refreshed, shared.OSOKAsyncPhaseCreate), nil
		}
		if !skillIsNotFound(err) {
			return c.fail(resource, fmt.Errorf("read Skill %q after work request: %w", skillID, err))
		}
	}

	refreshed, found, err := c.resolveByListForDesired(ctx, desired)
	if err != nil {
		return c.fail(resource, fmt.Errorf("resolve Skill after work request: %w", err))
	}
	if !found {
		return c.fail(resource, fmt.Errorf("Skill create work request %q succeeded but no Skill matched name/version", current.WorkRequestID))
	}
	return c.finishWithLifecycle(resource, refreshed, shared.OSOKAsyncPhaseCreate), nil
}

func (c *skillRuntimeClient) resolveCurrent(
	ctx context.Context,
	resource *odav1beta1.Skill,
	desired skillDesiredState,
) (odasdk.Skill, bool, error) {
	if currentID := currentSkillID(resource); currentID != "" {
		current, err := c.read(ctx, desired.odaInstanceID, currentID)
		if err != nil {
			if skillIsNotFound(err) {
				return odasdk.Skill{}, false, fmt.Errorf(
					"tracked Skill %q was not found under odaInstanceId %q; refusing to create a replacement because Skill identity fields are create-only",
					currentID,
					desired.odaInstanceID,
				)
			}
			return odasdk.Skill{}, false, fmt.Errorf("get Skill %q: %w", currentID, err)
		}
		return current, true, nil
	}
	return c.resolveByListForDesired(ctx, desired)
}

func (c *skillRuntimeClient) resolveByListForDesired(
	ctx context.Context,
	desired skillDesiredState,
) (odasdk.Skill, bool, error) {
	lookup, err := c.resolveListLookupIdentity(ctx, desired)
	if err != nil {
		return odasdk.Skill{}, false, err
	}
	return c.resolveByList(ctx, lookup)
}

func (c *skillRuntimeClient) resolveListLookupIdentity(
	ctx context.Context,
	desired skillDesiredState,
) (skillDesiredState, error) {
	if strings.TrimSpace(desired.name) != "" {
		return desired, nil
	}
	if desired.kind != skillKindVersion || strings.TrimSpace(desired.id) == "" {
		return desired, nil
	}

	source, err := c.read(ctx, desired.odaInstanceID, desired.id)
	if err != nil {
		return skillDesiredState{}, fmt.Errorf("resolve source Skill %q for version lookup: %w", desired.id, err)
	}
	sourceName := strings.TrimSpace(stringValue(source.Name))
	if sourceName == "" {
		return skillDesiredState{}, fmt.Errorf("resolve source Skill %q for version lookup: OCI response did not include name", desired.id)
	}
	desired.name = sourceName
	return desired, nil
}

func (c *skillRuntimeClient) resolveByList(
	ctx context.Context,
	desired skillDesiredState,
) (odasdk.Skill, bool, error) {
	if !desired.hasLookupKey() {
		return odasdk.Skill{}, false, nil
	}

	request := odasdk.ListSkillsRequest{
		OdaInstanceId: common.String(desired.odaInstanceID),
	}
	if desired.name != "" {
		request.Name = common.String(desired.name)
	}
	if desired.version != "" {
		request.Version = common.String(desired.version)
	}

	var matchedID string
	for {
		response, err := c.client.ListSkills(ctx, request)
		if err != nil {
			if skillIsNotFound(err) {
				return odasdk.Skill{}, false, nil
			}
			return odasdk.Skill{}, false, err
		}
		for _, item := range response.Items {
			if !skillSummaryMatches(desired, item) {
				continue
			}
			if normalizeSkillLifecycle(item.LifecycleState) == string(odasdk.LifecycleStateDeleted) {
				continue
			}
			itemID := strings.TrimSpace(stringValue(item.Id))
			if itemID == "" {
				continue
			}
			if matchedID != "" && matchedID != itemID {
				return odasdk.Skill{}, false, fmt.Errorf("multiple OCI Skills matched name/version %q/%q", desired.name, desired.version)
			}
			matchedID = itemID
		}
		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			break
		}
		request.Page = response.OpcNextPage
	}

	if matchedID == "" {
		return odasdk.Skill{}, false, nil
	}
	current, err := c.read(ctx, desired.odaInstanceID, matchedID)
	if err != nil {
		if skillIsNotFound(err) {
			return odasdk.Skill{}, false, nil
		}
		return odasdk.Skill{}, false, err
	}
	return current, true, nil
}

func (c *skillRuntimeClient) read(ctx context.Context, odaInstanceID string, skillID string) (odasdk.Skill, error) {
	response, err := c.client.GetSkill(ctx, odasdk.GetSkillRequest{
		OdaInstanceId: common.String(odaInstanceID),
		SkillId:       common.String(skillID),
	})
	if err != nil {
		return odasdk.Skill{}, err
	}
	return response.Skill, nil
}

func (c *skillRuntimeClient) finishWithLifecycle(
	resource *odav1beta1.Skill,
	current odasdk.Skill,
	fallbackPhase shared.OSOKAsyncPhase,
) servicemanager.OSOKResponse {
	if err := c.projectStatus(resource, current); err != nil {
		response, _ := c.fail(resource, err)
		return response
	}

	state := normalizeSkillLifecycle(current.LifecycleState)
	message := skillLifecycleMessage(current)
	switch state {
	case string(odasdk.LifecycleStateActive):
		return c.markCondition(resource, shared.Active, state, message, false)
	case string(odasdk.LifecycleStateCreating):
		return c.markPendingLifecycleOperation(resource, shared.OSOKAsyncPhaseCreate, message, state)
	case string(odasdk.LifecycleStateUpdating):
		return c.markPendingLifecycleOperation(resource, shared.OSOKAsyncPhaseUpdate, message, state)
	case string(odasdk.LifecycleStateDeleting):
		return c.markPendingLifecycleOperation(resource, shared.OSOKAsyncPhaseDelete, message, state)
	case string(odasdk.LifecycleStateDeleted):
		return c.markAsyncOperation(resource, &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           shared.OSOKAsyncPhaseDelete,
			RawStatus:       state,
			NormalizedClass: shared.OSOKAsyncClassSucceeded,
			Message:         message,
		})
	case string(odasdk.LifecycleStateFailed), string(odasdk.LifecycleStateInactive):
		return c.markAsyncOperation(resource, &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           fallbackPhase,
			RawStatus:       state,
			NormalizedClass: shared.OSOKAsyncClassFailed,
			Message:         message,
		})
	default:
		return c.markCondition(
			resource,
			shared.Failed,
			state,
			fmt.Sprintf("formal lifecycle state %q is not modeled: %s", state, message),
			false,
		)
	}
}

func (c *skillRuntimeClient) markCondition(
	resource *odav1beta1.Skill,
	condition shared.OSOKConditionType,
	rawState string,
	message string,
	shouldRequeue bool,
) servicemanager.OSOKResponse {
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: condition != shared.Failed}
	}

	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.Message = message
	status.Reason = string(condition)
	status.UpdatedAt = &now
	if condition == shared.Active {
		servicemanager.ClearAsyncOperation(status)
	} else if condition == shared.Failed {
		status.Async.Current = &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			RawStatus:       rawState,
			NormalizedClass: shared.OSOKAsyncClassFailed,
			Message:         message,
			UpdatedAt:       &now,
		}
	}

	conditionStatus := v1.ConditionTrue
	if condition == shared.Failed {
		conditionStatus = v1.ConditionFalse
	}
	*status = util.UpdateOSOKStatusCondition(*status, condition, conditionStatus, "", message, c.log)
	return servicemanager.OSOKResponse{
		IsSuccessful:    condition != shared.Failed,
		ShouldRequeue:   shouldRequeue,
		RequeueDuration: skillRequeueDuration,
	}
}

func (c *skillRuntimeClient) markAsyncOperation(
	resource *odav1beta1.Skill,
	current *shared.OSOKAsyncOperation,
) servicemanager.OSOKResponse {
	if resource == nil || current == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}
	now := metav1.Now()
	if current.UpdatedAt == nil {
		current.UpdatedAt = &now
	}
	projection := servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, c.log)
	return servicemanager.OSOKResponse{
		IsSuccessful:    projection.Condition != shared.Failed,
		ShouldRequeue:   projection.ShouldRequeue,
		RequeueDuration: skillRequeueDuration,
	}
}

func (c *skillRuntimeClient) markPendingLifecycleOperation(
	resource *odav1beta1.Skill,
	phase shared.OSOKAsyncPhase,
	message string,
	rawStatus string,
) servicemanager.OSOKResponse {
	return c.markAsyncOperation(resource, &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           phase,
		RawStatus:       normalizeSkillLifecycleString(rawStatus),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         strings.TrimSpace(message),
	})
}

func (c *skillRuntimeClient) markPendingWorkRequest(
	resource *odav1beta1.Skill,
	workRequestID string,
	phase shared.OSOKAsyncPhase,
	message string,
) servicemanager.OSOKResponse {
	return c.markWorkRequestOperation(resource, &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           phase,
		WorkRequestID:   strings.TrimSpace(workRequestID),
		RawStatus:       string(odasdk.WorkRequestStatusAccepted),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         strings.TrimSpace(message),
	})
}

func (c *skillRuntimeClient) markWorkRequestOperation(
	resource *odav1beta1.Skill,
	current *shared.OSOKAsyncOperation,
) servicemanager.OSOKResponse {
	return c.markAsyncOperation(resource, current)
}

func (c *skillRuntimeClient) buildWorkRequestOperation(
	resource *odav1beta1.Skill,
	workRequest odasdk.WorkRequest,
	fallbackWorkRequestID string,
	fallbackPhase shared.OSOKAsyncPhase,
) (*shared.OSOKAsyncOperation, error) {
	if resource == nil {
		return nil, fmt.Errorf("Skill resource is nil")
	}
	workRequestID := strings.TrimSpace(stringValue(workRequest.Id))
	if workRequestID == "" {
		workRequestID = strings.TrimSpace(fallbackWorkRequestID)
	}
	message := strings.TrimSpace(stringValue(workRequest.StatusMessage))
	if message == "" {
		message = fmt.Sprintf("Skill %s work request %s is %s", fallbackPhase, workRequestID, workRequest.Status)
	}
	return servicemanager.BuildWorkRequestAsyncOperation(
		&resource.Status.OsokStatus,
		skillWorkRequestAdapter(),
		servicemanager.WorkRequestAsyncInput{
			RawStatus:       string(workRequest.Status),
			RawAction:       string(workRequest.RequestAction),
			WorkRequestID:   workRequestID,
			PercentComplete: workRequest.PercentComplete,
			Message:         message,
			FallbackPhase:   fallbackPhase,
		},
	)
}

func (c *skillRuntimeClient) markTerminating(resource *odav1beta1.Skill, message string, rawStatus string) error {
	c.markPendingLifecycleOperation(resource, shared.OSOKAsyncPhaseDelete, message, rawStatus)
	return nil
}

func (c *skillRuntimeClient) markDeleted(resource *odav1beta1.Skill, message string) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, c.log)
}

func (c *skillRuntimeClient) fail(
	resource *odav1beta1.Skill,
	err error,
) (servicemanager.OSOKResponse, error) {
	if err != nil {
		_ = c.markFailure(resource, err)
	}
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func (c *skillRuntimeClient) markFailure(resource *odav1beta1.Skill, err error) error {
	if resource == nil || err == nil {
		return err
	}
	status := &resource.Status.OsokStatus
	servicemanager.RecordErrorOpcRequestID(status, err)
	now := metav1.Now()
	status.UpdatedAt = &now
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	if status.Async.Current != nil {
		current := *status.Async.Current
		current.NormalizedClass = shared.OSOKAsyncClassFailed
		current.Message = err.Error()
		current.UpdatedAt = &now
		_ = servicemanager.ApplyAsyncOperation(status, &current, c.log)
		return err
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, v1.ConditionFalse, "", err.Error(), c.log)
	return err
}

func (c *skillRuntimeClient) projectStatus(resource *odav1beta1.Skill, current odasdk.Skill) error {
	if resource == nil {
		return fmt.Errorf("Skill resource is nil")
	}
	status := &resource.Status
	status.Id = stringValue(current.Id)
	status.Name = stringValue(current.Name)
	status.Version = stringValue(current.Version)
	status.DisplayName = stringValue(current.DisplayName)
	status.LifecycleState = string(current.LifecycleState)
	status.LifecycleDetails = string(current.LifecycleDetails)
	status.PlatformVersion = stringValue(current.PlatformVersion)
	status.TimeCreated = skillSDKTimeString(current.TimeCreated)
	status.TimeUpdated = skillSDKTimeString(current.TimeUpdated)
	status.Category = stringValue(current.Category)
	status.Description = stringValue(current.Description)
	status.Namespace = stringValue(current.Namespace)
	status.DialogVersion = stringValue(current.DialogVersion)
	status.BaseId = stringValue(current.BaseId)
	status.MultilingualMode = string(current.MultilingualMode)
	status.PrimaryLanguageTag = stringValue(current.PrimaryLanguageTag)
	status.NativeLanguageTags = cloneStringSlice(current.NativeLanguageTags)
	status.FreeformTags = cloneStringMap(current.FreeformTags)
	status.DefinedTags = skillStatusDefinedTags(current.DefinedTags)

	now := metav1.Now()
	if status.Id != "" {
		status.OsokStatus.Ocid = shared.OCID(status.Id)
		if status.OsokStatus.CreatedAt == nil {
			status.OsokStatus.CreatedAt = &now
		}
	}
	return nil
}

func (c *skillRuntimeClient) recordResponseRequestID(resource *odav1beta1.Skill, response any) {
	if resource == nil {
		return
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
}

func (c *skillRuntimeClient) recordErrorRequestID(resource *odav1beta1.Skill, err error) {
	if resource == nil {
		return
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
}

func (c *skillRuntimeClient) ensureClient() error {
	if c == nil {
		return fmt.Errorf("Skill runtime client is not configured")
	}
	if c.initErr != nil {
		return fmt.Errorf("initialize Skill OCI client: %w", c.initErr)
	}
	if c.client == nil {
		return fmt.Errorf("Skill OCI client is not configured")
	}
	return nil
}

func resolveDesiredSkillState(resource *odav1beta1.Skill) (skillDesiredState, error) {
	if resource == nil {
		return skillDesiredState{}, fmt.Errorf("Skill resource is nil")
	}
	spec := resource.Spec
	desired := skillDesiredState{
		odaInstanceID:      skillOdaInstanceID(resource),
		kind:               strings.TrimSpace(spec.Kind),
		kindSet:            strings.TrimSpace(spec.Kind) != "",
		id:                 strings.TrimSpace(spec.Id),
		idSet:              strings.TrimSpace(spec.Id) != "",
		name:               strings.TrimSpace(spec.Name),
		displayName:        strings.TrimSpace(spec.DisplayName),
		version:            strings.TrimSpace(spec.Version),
		category:           spec.Category,
		categorySet:        strings.TrimSpace(spec.Category) != "",
		description:        spec.Description,
		descriptionSet:     strings.TrimSpace(spec.Description) != "",
		platformVersion:    strings.TrimSpace(spec.PlatformVersion),
		dialogVersion:      strings.TrimSpace(spec.DialogVersion),
		multilingualMode:   strings.TrimSpace(spec.MultilingualMode),
		primaryLanguageTag: strings.TrimSpace(spec.PrimaryLanguageTag),
		freeformTags:       maps.Clone(spec.FreeformTags),
		definedTags:        cloneDefinedTags(spec.DefinedTags),
		nativeLanguageTags: cloneStringSlice(spec.NativeLanguageTags),
	}
	if err := mergeSkillJSONData(&desired, spec.JsonData); err != nil {
		return skillDesiredState{}, err
	}
	if desired.odaInstanceID == "" {
		return skillDesiredState{}, fmt.Errorf("Skill requires metadata annotation %q with the parent ODA instance OCID", skillOdaInstanceIDAnnotation)
	}
	kind, err := normalizeSkillKind(desired.kind)
	if err != nil {
		return skillDesiredState{}, err
	}
	desired.kind = kind
	if desired.multilingualMode != "" {
		multilingualMode, ok := odasdk.GetMappingBotMultilingualModeEnum(desired.multilingualMode)
		if !ok {
			return skillDesiredState{}, fmt.Errorf("Skill spec.multilingualMode %q is not supported", desired.multilingualMode)
		}
		desired.multilingualMode = string(multilingualMode)
	}
	return desired, nil
}

func mergeSkillJSONData(desired *skillDesiredState, rawJSON string) error {
	rawJSON = strings.TrimSpace(rawJSON)
	if rawJSON == "" {
		return nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(rawJSON), &raw); err != nil {
		return fmt.Errorf("decode Skill spec.jsonData: %w", err)
	}

	if err := mergeStringFieldWithPresence(raw, "kind", &desired.kind, &desired.kindSet); err != nil {
		return err
	}
	if err := mergeStringFieldWithPresence(raw, "id", &desired.id, &desired.idSet); err != nil {
		return err
	}
	if err := mergeStringField(raw, "name", &desired.name); err != nil {
		return err
	}
	if err := mergeStringField(raw, "displayName", &desired.displayName); err != nil {
		return err
	}
	if err := mergeStringField(raw, "version", &desired.version); err != nil {
		return err
	}
	if err := mergeStringFieldWithPresence(raw, "category", &desired.category, &desired.categorySet); err != nil {
		return err
	}
	if err := mergeStringFieldWithPresence(raw, "description", &desired.description, &desired.descriptionSet); err != nil {
		return err
	}
	if err := mergeStringField(raw, "platformVersion", &desired.platformVersion); err != nil {
		return err
	}
	if err := mergeStringField(raw, "dialogVersion", &desired.dialogVersion); err != nil {
		return err
	}
	if err := mergeStringField(raw, "multilingualMode", &desired.multilingualMode); err != nil {
		return err
	}
	if err := mergeStringField(raw, "primaryLanguageTag", &desired.primaryLanguageTag); err != nil {
		return err
	}
	if desired.freeformTags == nil {
		if err := mergeStringMapField(raw, "freeformTags", &desired.freeformTags); err != nil {
			return err
		}
	}
	if desired.definedTags == nil {
		if err := mergeDefinedTagsField(raw, "definedTags", &desired.definedTags); err != nil {
			return err
		}
	}
	if len(desired.nativeLanguageTags) == 0 {
		if err := mergeStringSliceField(raw, "nativeLanguageTags", &desired.nativeLanguageTags); err != nil {
			return err
		}
	}
	return nil
}

func mergeStringField(raw map[string]json.RawMessage, key string, target *string) error {
	if strings.TrimSpace(*target) != "" {
		return nil
	}
	payload, ok := raw[key]
	if !ok {
		return nil
	}
	var value string
	if err := json.Unmarshal(payload, &value); err != nil {
		return fmt.Errorf("decode Skill spec.jsonData.%s: %w", key, err)
	}
	*target = strings.TrimSpace(value)
	return nil
}

func mergeStringFieldWithPresence(raw map[string]json.RawMessage, key string, target *string, present *bool) error {
	if strings.TrimSpace(*target) != "" {
		return nil
	}
	payload, ok := raw[key]
	if !ok {
		return nil
	}
	var value string
	if err := json.Unmarshal(payload, &value); err != nil {
		return fmt.Errorf("decode Skill spec.jsonData.%s: %w", key, err)
	}
	*target = strings.TrimSpace(value)
	if present != nil {
		*present = true
	}
	return nil
}

func mergeStringMapField(raw map[string]json.RawMessage, key string, target *map[string]string) error {
	payload, ok := raw[key]
	if !ok {
		return nil
	}
	var value map[string]string
	if err := json.Unmarshal(payload, &value); err != nil {
		return fmt.Errorf("decode Skill spec.jsonData.%s: %w", key, err)
	}
	*target = value
	return nil
}

func mergeDefinedTagsField(raw map[string]json.RawMessage, key string, target *map[string]shared.MapValue) error {
	payload, ok := raw[key]
	if !ok {
		return nil
	}
	var value map[string]shared.MapValue
	if err := json.Unmarshal(payload, &value); err != nil {
		return fmt.Errorf("decode Skill spec.jsonData.%s: %w", key, err)
	}
	*target = value
	return nil
}

func mergeStringSliceField(raw map[string]json.RawMessage, key string, target *[]string) error {
	payload, ok := raw[key]
	if !ok {
		return nil
	}
	var value []string
	if err := json.Unmarshal(payload, &value); err != nil {
		return fmt.Errorf("decode Skill spec.jsonData.%s: %w", key, err)
	}
	*target = value
	return nil
}

func skillOdaInstanceID(resource *odav1beta1.Skill) string {
	if resource == nil {
		return ""
	}
	for _, key := range []string{skillOdaInstanceIDAnnotation, skillLegacyOdaInstanceIDAnnotation} {
		if value := strings.TrimSpace(resource.Annotations[key]); value != "" {
			return value
		}
	}
	return ""
}

func validateSkillCreateInputs(desired skillDesiredState) error {
	var missing []string
	switch desired.kind {
	case skillKindNew:
		if desired.id != "" {
			return fmt.Errorf("Skill spec.id is only valid when spec.kind is CLONE, EXTEND, or VERSION")
		}
		if desired.name == "" {
			missing = append(missing, "spec.name")
		}
		if desired.displayName == "" {
			missing = append(missing, "spec.displayName")
		}
		if desired.version == "" {
			missing = append(missing, "spec.version")
		}
	case skillKindClone, skillKindExtend:
		if desired.id == "" {
			missing = append(missing, "spec.id")
		}
		if desired.name == "" {
			missing = append(missing, "spec.name")
		}
		if desired.displayName == "" {
			missing = append(missing, "spec.displayName")
		}
	case skillKindVersion:
		if desired.id == "" {
			missing = append(missing, "spec.id")
		}
		if desired.version == "" {
			missing = append(missing, "spec.version")
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("Skill %s create requires %s", desired.kind, strings.Join(missing, ", "))
	}
	return nil
}

func buildSkillCreateDetails(desired skillDesiredState) (odasdk.CreateSkillDetails, error) {
	switch desired.kind {
	case skillKindNew:
		details := odasdk.CreateNewSkillDetails{
			Name:        common.String(desired.name),
			DisplayName: common.String(desired.displayName),
			Version:     common.String(desired.version),
		}
		applyNewSkillCreateCommon(&details, desired)
		return details, nil
	case skillKindClone:
		details := odasdk.CloneSkillDetails{
			Id:          common.String(desired.id),
			Name:        common.String(desired.name),
			DisplayName: common.String(desired.displayName),
		}
		if desired.version != "" {
			details.Version = common.String(desired.version)
		}
		applyCloneSkillCreateCommon(&details, desired)
		return details, nil
	case skillKindExtend:
		details := odasdk.ExtendSkillDetails{
			Id:          common.String(desired.id),
			Name:        common.String(desired.name),
			DisplayName: common.String(desired.displayName),
		}
		if desired.version != "" {
			details.Version = common.String(desired.version)
		}
		applyExtendSkillCreateCommon(&details, desired)
		return details, nil
	case skillKindVersion:
		details := odasdk.CreateSkillVersionDetails{
			Id:      common.String(desired.id),
			Version: common.String(desired.version),
		}
		applyVersionSkillCreateCommon(&details, desired)
		return details, nil
	default:
		return nil, fmt.Errorf("unsupported Skill kind %q", desired.kind)
	}
}

func applyNewSkillCreateCommon(details *odasdk.CreateNewSkillDetails, desired skillDesiredState) {
	details.Category = optionalString(desired.category)
	details.Description = optionalString(desired.description)
	details.PlatformVersion = optionalString(desired.platformVersion)
	details.DialogVersion = optionalString(desired.dialogVersion)
	details.PrimaryLanguageTag = optionalString(desired.primaryLanguageTag)
	details.FreeformTags = cloneStringMap(desired.freeformTags)
	details.DefinedTags = skillDefinedTagsFromSpec(desired.definedTags)
	details.NativeLanguageTags = cloneStringSlice(desired.nativeLanguageTags)
	if desired.multilingualMode != "" {
		details.MultilingualMode = odasdk.BotMultilingualModeEnum(desired.multilingualMode)
	}
}

func applyCloneSkillCreateCommon(details *odasdk.CloneSkillDetails, desired skillDesiredState) {
	details.Category = optionalString(desired.category)
	details.Description = optionalString(desired.description)
	details.PlatformVersion = optionalString(desired.platformVersion)
	details.DialogVersion = optionalString(desired.dialogVersion)
	details.PrimaryLanguageTag = optionalString(desired.primaryLanguageTag)
	details.FreeformTags = cloneStringMap(desired.freeformTags)
	details.DefinedTags = skillDefinedTagsFromSpec(desired.definedTags)
	if desired.multilingualMode != "" {
		details.MultilingualMode = odasdk.BotMultilingualModeEnum(desired.multilingualMode)
	}
}

func applyExtendSkillCreateCommon(details *odasdk.ExtendSkillDetails, desired skillDesiredState) {
	details.Category = optionalString(desired.category)
	details.Description = optionalString(desired.description)
	details.PlatformVersion = optionalString(desired.platformVersion)
	details.DialogVersion = optionalString(desired.dialogVersion)
	details.PrimaryLanguageTag = optionalString(desired.primaryLanguageTag)
	details.FreeformTags = cloneStringMap(desired.freeformTags)
	details.DefinedTags = skillDefinedTagsFromSpec(desired.definedTags)
	if desired.multilingualMode != "" {
		details.MultilingualMode = odasdk.BotMultilingualModeEnum(desired.multilingualMode)
	}
}

func applyVersionSkillCreateCommon(details *odasdk.CreateSkillVersionDetails, desired skillDesiredState) {
	details.Category = optionalString(desired.category)
	details.Description = optionalString(desired.description)
	details.PlatformVersion = optionalString(desired.platformVersion)
	details.DialogVersion = optionalString(desired.dialogVersion)
	details.PrimaryLanguageTag = optionalString(desired.primaryLanguageTag)
	details.FreeformTags = cloneStringMap(desired.freeformTags)
	details.DefinedTags = skillDefinedTagsFromSpec(desired.definedTags)
	if desired.multilingualMode != "" {
		details.MultilingualMode = odasdk.BotMultilingualModeEnum(desired.multilingualMode)
	}
}

func validateSkillCreateOnlyDrift(desired skillDesiredState, current odasdk.Skill) error {
	currentBaseID := strings.TrimSpace(stringValue(current.BaseId))
	if currentBaseID != "" {
		if desired.idSet && desired.id != currentBaseID {
			return fmt.Errorf("Skill create-only field drift detected for id; recreate the resource instead of updating immutable source fields")
		}
		if desired.kindSet && desired.kind != skillKindExtend {
			return fmt.Errorf("Skill create-only field drift detected for kind; recreate the resource instead of updating immutable source fields")
		}
	}

	checks := []struct {
		field   string
		desired string
		current string
	}{
		{field: "name", desired: desired.name, current: stringValue(current.Name)},
		{field: "displayName", desired: desired.displayName, current: stringValue(current.DisplayName)},
		{field: "version", desired: desired.version, current: stringValue(current.Version)},
		{field: "platformVersion", desired: desired.platformVersion, current: stringValue(current.PlatformVersion)},
		{field: "dialogVersion", desired: desired.dialogVersion, current: stringValue(current.DialogVersion)},
		{field: "multilingualMode", desired: desired.multilingualMode, current: string(current.MultilingualMode)},
		{field: "primaryLanguageTag", desired: desired.primaryLanguageTag, current: stringValue(current.PrimaryLanguageTag)},
	}
	for _, check := range checks {
		if check.desired == "" || check.current == "" || check.desired == check.current {
			continue
		}
		return fmt.Errorf("Skill create-only field drift detected for %s; recreate the resource instead of updating immutable fields", check.field)
	}
	if len(desired.nativeLanguageTags) > 0 && len(current.NativeLanguageTags) > 0 && !slices.Equal(desired.nativeLanguageTags, current.NativeLanguageTags) {
		return fmt.Errorf("Skill create-only field drift detected for nativeLanguageTags; recreate the resource instead of updating immutable fields")
	}
	return nil
}

func buildSkillUpdateDetails(desired skillDesiredState, current odasdk.Skill) (odasdk.UpdateSkillDetails, bool) {
	details := odasdk.UpdateSkillDetails{}
	updateNeeded := false
	if desired.categorySet && desired.category != stringValue(current.Category) {
		details.Category = common.String(desired.category)
		updateNeeded = true
	}
	if desired.descriptionSet && desired.description != stringValue(current.Description) {
		details.Description = common.String(desired.description)
		updateNeeded = true
	}
	if desired.freeformTags != nil && !maps.Equal(desired.freeformTags, current.FreeformTags) {
		details.FreeformTags = cloneStringMap(desired.freeformTags)
		updateNeeded = true
	}
	if desired.definedTags != nil {
		definedTags := skillDefinedTagsFromSpec(desired.definedTags)
		if !skillJSONEqual(definedTags, current.DefinedTags) {
			details.DefinedTags = definedTags
			updateNeeded = true
		}
	}
	return details, updateNeeded
}

func currentSkillID(resource *odav1beta1.Skill) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); id != "" {
		return id
	}
	return strings.TrimSpace(resource.Status.Id)
}

func currentSkillWorkRequest(resource *odav1beta1.Skill) (string, shared.OSOKAsyncPhase) {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return "", ""
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		return "", ""
	}
	return strings.TrimSpace(current.WorkRequestID), current.Phase
}

func currentSkillCreateWorkRequest(resource *odav1beta1.Skill) (string, shared.OSOKAsyncPhase) {
	workRequestID, phase := currentSkillWorkRequest(resource)
	if workRequestID == "" || phase != shared.OSOKAsyncPhaseCreate {
		return "", ""
	}
	return workRequestID, phase
}

func deleteSkillIdentity(resource *odav1beta1.Skill, odaInstanceID string) skillDesiredState {
	if resource == nil {
		return skillDesiredState{}
	}
	name := strings.TrimSpace(resource.Spec.Name)
	if name == "" {
		name = strings.TrimSpace(resource.Status.Name)
	}
	version := strings.TrimSpace(resource.Spec.Version)
	if version == "" {
		version = strings.TrimSpace(resource.Status.Version)
	}
	return skillDesiredState{
		odaInstanceID: odaInstanceID,
		name:          name,
		version:       version,
	}
}

func (desired skillDesiredState) hasLookupKey() bool {
	return strings.TrimSpace(desired.name) != ""
}

func skillSummaryMatches(desired skillDesiredState, item odasdk.SkillSummary) bool {
	if desired.name == "" {
		return false
	}
	if desired.name != strings.TrimSpace(stringValue(item.Name)) {
		return false
	}
	if desired.version != "" && desired.version != strings.TrimSpace(stringValue(item.Version)) {
		return false
	}
	return true
}

func skillLifecycleBlocksMutation(state string) bool {
	switch state {
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

func skillIDFromSDK(current odasdk.Skill) string {
	return strings.TrimSpace(stringValue(current.Id))
}

func skillIDFromWorkRequest(workRequest odasdk.WorkRequest) string {
	if id := strings.TrimSpace(stringValue(workRequest.ResourceId)); id != "" {
		return id
	}
	for _, resource := range workRequest.Resources {
		if resource.ResourceAction != odasdk.WorkRequestResourceResourceActionCreate {
			continue
		}
		if id := strings.TrimSpace(stringValue(resource.ResourceId)); id != "" {
			return id
		}
	}
	return ""
}

func skillWorkRequestAdapter() servicemanager.WorkRequestAsyncAdapter {
	return servicemanager.WorkRequestAsyncAdapter{
		PendingStatusTokens: []string{
			string(odasdk.WorkRequestStatusAccepted),
			string(odasdk.WorkRequestStatusInProgress),
			string(odasdk.WorkRequestStatusCanceling),
		},
		SucceededStatusTokens: []string{string(odasdk.WorkRequestStatusSucceeded)},
		FailedStatusTokens:    []string{string(odasdk.WorkRequestStatusFailed)},
		CanceledStatusTokens:  []string{string(odasdk.WorkRequestStatusCanceled)},
		CreateActionTokens: []string{
			string(odasdk.WorkRequestRequestActionCreateSkill),
			string(odasdk.WorkRequestRequestActionCloneSkill),
			string(odasdk.WorkRequestRequestActionExtendSkill),
			string(odasdk.WorkRequestRequestActionVersionSkill),
		},
	}
}

func normalizeSkillKind(value string) (string, error) {
	kind := strings.ToUpper(strings.TrimSpace(value))
	if kind == "" {
		return skillKindNew, nil
	}
	switch kind {
	case skillKindNew, skillKindClone, skillKindExtend, skillKindVersion:
		return kind, nil
	default:
		return "", fmt.Errorf("unsupported Skill spec.kind %q", value)
	}
}

func normalizeSkillLifecycle(state odasdk.LifecycleStateEnum) string {
	return normalizeSkillLifecycleString(string(state))
}

func normalizeSkillLifecycleString(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func skillLifecycleMessage(current odasdk.Skill) string {
	name := strings.TrimSpace(stringValue(current.DisplayName))
	if name == "" {
		name = strings.TrimSpace(stringValue(current.Name))
	}
	state := normalizeSkillLifecycle(current.LifecycleState)
	if name != "" && state != "" {
		return fmt.Sprintf("Skill %s is %s", name, state)
	}
	if state != "" {
		return "Skill lifecycle state " + state
	}
	return "OCI Skill is active"
}

func skillIdentityLabel(desired skillDesiredState) string {
	switch {
	case desired.name != "" && desired.version != "":
		return desired.name + "/" + desired.version
	case desired.name != "":
		return desired.name
	case desired.version != "":
		return desired.version
	default:
		return desired.kind
	}
}

func skillIsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return servicemanager.IsNotFoundServiceError(err) ||
		servicemanager.IsNotFoundErrorString(err) ||
		errors.Is(err, errSkillNotFound)
}

var errSkillNotFound = errors.New("skill not found")

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return common.String(value)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		if input == nil {
			return nil
		}
		return map[string]string{}
	}
	return maps.Clone(input)
}

func cloneStringSlice(input []string) []string {
	if len(input) == 0 {
		if input == nil {
			return nil
		}
		return []string{}
	}
	return slices.Clone(input)
}

func cloneDefinedTags(input map[string]shared.MapValue) map[string]shared.MapValue {
	if input == nil {
		return nil
	}
	output := make(map[string]shared.MapValue, len(input))
	for namespace, values := range input {
		if values == nil {
			output[namespace] = nil
			continue
		}
		output[namespace] = maps.Clone(values)
	}
	return output
}

func skillDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
	if spec == nil {
		return nil
	}
	return *util.ConvertToOciDefinedTags(&spec)
}

func skillStatusDefinedTags(input map[string]map[string]interface{}) map[string]shared.MapValue {
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

func skillJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return reflect.DeepEqual(left, right)
	}
	return string(leftPayload) == string(rightPayload)
}

func skillSDKTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Time.Format(time.RFC3339Nano)
}

/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package zone

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	dnssdk "github.com/oracle/oci-go-sdk/v65/dns"
	dnsv1beta1 "github.com/oracle/oci-service-operator/api/dns/v1beta1"
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

const zoneRequeueDuration = time.Minute
const zoneDynectCreateOnlyReasonPrefix = "ZoneDynectCreateOnlyInputs:v1:"

type zoneOCIClient interface {
	CreateZone(context.Context, dnssdk.CreateZoneRequest) (dnssdk.CreateZoneResponse, error)
	GetZone(context.Context, dnssdk.GetZoneRequest) (dnssdk.GetZoneResponse, error)
	ListZones(context.Context, dnssdk.ListZonesRequest) (dnssdk.ListZonesResponse, error)
	UpdateZone(context.Context, dnssdk.UpdateZoneRequest) (dnssdk.UpdateZoneResponse, error)
	DeleteZone(context.Context, dnssdk.DeleteZoneRequest) (dnssdk.DeleteZoneResponse, error)
}

type ambiguousZoneNotFoundError struct {
	message      string
	opcRequestID string
}

func (e ambiguousZoneNotFoundError) Error() string {
	return e.message
}

func (e ambiguousZoneNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerZoneRuntimeHooksMutator(func(manager *ZoneServiceManager, hooks *ZoneRuntimeHooks) {
		client, initErr := newZoneSDKClient(manager)
		applyZoneRuntimeHooks(manager, hooks, client, initErr)
	})
}

func newZoneSDKClient(manager *ZoneServiceManager) (zoneOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("zone service manager is nil")
	}
	client, err := dnssdk.NewDnsClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyZoneRuntimeHooks(
	manager *ZoneServiceManager,
	hooks *ZoneRuntimeHooks,
	client zoneOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newZoneRuntimeSemantics()
	hooks.BuildCreateBody = buildZoneCreateBody
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *dnsv1beta1.Zone,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildZoneUpdateBody(resource, currentResponse)
	}
	hooks.Create.Fields = zoneCreateFields()
	hooks.Create.Call = func(ctx context.Context, request dnssdk.CreateZoneRequest) (dnssdk.CreateZoneResponse, error) {
		if err := zoneClientReady(client, initErr); err != nil {
			return dnssdk.CreateZoneResponse{}, err
		}
		return client.CreateZone(ctx, request)
	}
	hooks.Get.Fields = zoneGetFields()
	hooks.Get.Call = func(ctx context.Context, request dnssdk.GetZoneRequest) (dnssdk.GetZoneResponse, error) {
		if err := zoneClientReady(client, initErr); err != nil {
			return dnssdk.GetZoneResponse{}, err
		}
		response, err := client.GetZone(ctx, request)
		return response, conservativeZoneNotFoundError(err, "read")
	}
	hooks.List.Fields = zoneListFields()
	hooks.List.Call = func(ctx context.Context, request dnssdk.ListZonesRequest) (dnssdk.ListZonesResponse, error) {
		return listZonesAllPages(ctx, client, initErr, request)
	}
	hooks.Update.Fields = zoneUpdateFields()
	hooks.Update.Call = func(ctx context.Context, request dnssdk.UpdateZoneRequest) (dnssdk.UpdateZoneResponse, error) {
		if err := zoneClientReady(client, initErr); err != nil {
			return dnssdk.UpdateZoneResponse{}, err
		}
		return client.UpdateZone(ctx, request)
	}
	hooks.Delete.Fields = zoneDeleteFields()
	hooks.Delete.Call = func(ctx context.Context, request dnssdk.DeleteZoneRequest) (dnssdk.DeleteZoneResponse, error) {
		if err := zoneClientReady(client, initErr); err != nil {
			return dnssdk.DeleteZoneResponse{}, err
		}
		response, err := client.DeleteZone(ctx, request)
		return response, conservativeZoneNotFoundError(err, "delete")
	}
	hooks.DeleteHooks.HandleError = handleZoneDeleteError
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ZoneServiceClient) ZoneServiceClient {
		log := loggerutil.OSOKLogger{}
		if manager != nil {
			log = manager.Log
		}
		return zoneRuntimeClient{
			delegate: delegate,
			client:   client,
			initErr:  initErr,
			log:      log,
		}
	})
}

func newZoneServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client zoneOCIClient,
	initErr error,
) ZoneServiceClient {
	manager := &ZoneServiceManager{Log: log}
	hooks := newZoneRuntimeHooksWithOCIClient(client)
	applyZoneRuntimeHooks(manager, &hooks, client, initErr)
	delegate := defaultZoneServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*dnsv1beta1.Zone](
			buildZoneGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapZoneGeneratedClient(hooks, delegate)
}

func newZoneRuntimeHooksWithOCIClient(client zoneOCIClient) ZoneRuntimeHooks {
	return ZoneRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*dnsv1beta1.Zone]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*dnsv1beta1.Zone]{},
		StatusHooks:     generatedruntime.StatusHooks[*dnsv1beta1.Zone]{},
		ParityHooks:     generatedruntime.ParityHooks[*dnsv1beta1.Zone]{},
		Async:           generatedruntime.AsyncHooks[*dnsv1beta1.Zone]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*dnsv1beta1.Zone]{},
		Create: runtimeOperationHooks[dnssdk.CreateZoneRequest, dnssdk.CreateZoneResponse]{
			Fields: zoneCreateFields(),
			Call: func(ctx context.Context, request dnssdk.CreateZoneRequest) (dnssdk.CreateZoneResponse, error) {
				return client.CreateZone(ctx, request)
			},
		},
		Get: runtimeOperationHooks[dnssdk.GetZoneRequest, dnssdk.GetZoneResponse]{
			Fields: zoneGetFields(),
			Call: func(ctx context.Context, request dnssdk.GetZoneRequest) (dnssdk.GetZoneResponse, error) {
				return client.GetZone(ctx, request)
			},
		},
		List: runtimeOperationHooks[dnssdk.ListZonesRequest, dnssdk.ListZonesResponse]{
			Fields: zoneListFields(),
			Call: func(ctx context.Context, request dnssdk.ListZonesRequest) (dnssdk.ListZonesResponse, error) {
				return client.ListZones(ctx, request)
			},
		},
		Update: runtimeOperationHooks[dnssdk.UpdateZoneRequest, dnssdk.UpdateZoneResponse]{
			Fields: zoneUpdateFields(),
			Call: func(ctx context.Context, request dnssdk.UpdateZoneRequest) (dnssdk.UpdateZoneResponse, error) {
				return client.UpdateZone(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[dnssdk.DeleteZoneRequest, dnssdk.DeleteZoneResponse]{
			Fields: zoneDeleteFields(),
			Call: func(ctx context.Context, request dnssdk.DeleteZoneRequest) (dnssdk.DeleteZoneResponse, error) {
				return client.DeleteZone(ctx, request)
			},
		},
		WrapGeneratedClient: []func(ZoneServiceClient) ZoneServiceClient{},
	}
}

type zoneRuntimeClient struct {
	delegate ZoneServiceClient
	client   zoneOCIClient
	initErr  error
	log      loggerutil.OSOKLogger
}

var _ ZoneServiceClient = zoneRuntimeClient{}

func (c zoneRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *dnsv1beta1.Zone,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if err := zoneClientReady(c.client, c.initErr); err != nil {
		return c.fail(resource, err)
	}
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("zone resource is nil")
	}

	current, exists, err := c.resolveZone(ctx, resource)
	if err != nil {
		return c.fail(resource, err)
	}
	if exists {
		return c.reconcileExistingZone(ctx, resource, current)
	}
	return c.createZone(ctx, resource, req)
}

func (c zoneRuntimeClient) Delete(ctx context.Context, resource *dnsv1beta1.Zone) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("zone generated runtime delegate is not configured")
	}
	if err := c.rejectAuthShapedDeleteConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c zoneRuntimeClient) rejectAuthShapedDeleteConfirmRead(ctx context.Context, resource *dnsv1beta1.Zone) error {
	if resource == nil {
		return nil
	}
	currentID := currentZoneID(resource)
	if currentID == "" {
		return nil
	}
	if err := zoneClientReady(c.client, c.initErr); err != nil {
		return err
	}
	_, err := c.client.GetZone(ctx, dnssdk.GetZoneRequest{
		ZoneNameOrId:  common.String(currentID),
		Scope:         dnssdk.GetZoneScopeEnum(resource.Spec.Scope),
		ViewId:        optionalString(resource.Spec.ViewId),
		CompartmentId: optionalString(resource.Spec.CompartmentId),
	})
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	err = conservativeZoneNotFoundError(err, "delete confirmation")
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("zone delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", err)
}

func (c zoneRuntimeClient) resolveZone(ctx context.Context, resource *dnsv1beta1.Zone) (dnssdk.Zone, bool, error) {
	if currentID := currentZoneID(resource); currentID != "" {
		current, found, err := c.getZone(ctx, resource, currentID)
		if err != nil {
			return dnssdk.Zone{}, false, err
		}
		if found {
			return current, true, nil
		}
		return dnssdk.Zone{}, false, validateZoneTrackedReadMiss(resource, currentID)
	}

	summary, found, err := c.findExistingZone(ctx, resource)
	if err != nil || !found {
		return dnssdk.Zone{}, false, err
	}
	currentID := stringValue(summary.Id)
	if currentID == "" {
		return zoneFromSummary(summary), true, nil
	}
	current, found, err := c.getZone(ctx, resource, currentID)
	if err != nil {
		return dnssdk.Zone{}, false, err
	}
	if !found {
		return zoneFromSummary(summary), true, nil
	}
	return current, true, nil
}

func (c zoneRuntimeClient) reconcileExistingZone(
	ctx context.Context,
	resource *dnsv1beta1.Zone,
	current dnssdk.Zone,
) (servicemanager.OSOKResponse, error) {
	wasTracked := currentZoneID(resource) != ""
	if zoneLifecyclePreventsUpdate(current) {
		return c.projectSuccess(resource, current, shared.Active)
	}

	if err := validateZoneForceNew(resource, current, wasTracked); err != nil {
		return c.fail(resource, err)
	}

	details, updateNeeded, err := buildZoneUpdateBody(resource, dnssdk.GetZoneResponse{Zone: current})
	if err != nil {
		return c.fail(resource, err)
	}
	if !updateNeeded {
		return c.projectSuccess(resource, current, shared.Active)
	}

	currentID := stringValue(current.Id)
	if currentID == "" {
		return c.fail(resource, fmt.Errorf("zone update requires an OCI resource identifier"))
	}
	response, err := c.client.UpdateZone(ctx, dnssdk.UpdateZoneRequest{
		ZoneNameOrId:      common.String(currentID),
		Scope:             dnssdk.UpdateZoneScopeEnum(resource.Spec.Scope),
		ViewId:            optionalString(resource.Spec.ViewId),
		CompartmentId:     optionalString(resource.Spec.CompartmentId),
		UpdateZoneDetails: details,
	})
	if err != nil {
		return c.fail(resource, normalizeZoneOCIError(err))
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	c.seedZoneOpeningWorkRequestID(resource, response, shared.OSOKAsyncPhaseUpdate)

	refreshed, found, err := c.getZone(ctx, resource, currentID)
	if err != nil {
		return c.fail(resource, err)
	}
	if !found {
		refreshed = response.Zone
	}
	return c.projectSuccess(resource, refreshed, shared.Updating)
}

func (c zoneRuntimeClient) createZone(
	ctx context.Context,
	resource *dnsv1beta1.Zone,
	_ ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	body, err := buildZoneCreateBody(ctx, resource, resource.Namespace)
	if err != nil {
		return c.fail(resource, err)
	}
	details, ok := body.(dnssdk.CreateZoneBaseDetails)
	if !ok {
		return c.fail(resource, fmt.Errorf("zone create body %T does not implement dns.CreateZoneBaseDetails", body))
	}

	response, err := c.client.CreateZone(ctx, dnssdk.CreateZoneRequest{
		CreateZoneDetails: details,
		CompartmentId:     optionalString(resource.Spec.CompartmentId),
		Scope:             dnssdk.CreateZoneScopeEnum(resource.Spec.Scope),
		ViewId:            optionalString(resource.Spec.ViewId),
		OpcRetryToken:     zoneRetryToken(resource),
	})
	if err != nil {
		return c.fail(resource, normalizeZoneOCIError(err))
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	c.seedZoneOpeningWorkRequestID(resource, response, shared.OSOKAsyncPhaseCreate)

	createdID := stringValue(response.Id)
	if createdID == "" {
		return c.projectSuccess(resource, response.Zone, shared.Provisioning)
	}
	refreshed, found, err := c.getZone(ctx, resource, createdID)
	if err != nil {
		return c.fail(resource, err)
	}
	if !found {
		refreshed = response.Zone
	}
	return c.projectSuccess(resource, refreshed, shared.Provisioning)
}

func (c zoneRuntimeClient) getZone(
	ctx context.Context,
	resource *dnsv1beta1.Zone,
	zoneNameOrID string,
) (dnssdk.Zone, bool, error) {
	response, err := c.client.GetZone(ctx, dnssdk.GetZoneRequest{
		ZoneNameOrId:  common.String(zoneNameOrID),
		Scope:         dnssdk.GetZoneScopeEnum(resource.Spec.Scope),
		ViewId:        optionalString(resource.Spec.ViewId),
		CompartmentId: optionalString(resource.Spec.CompartmentId),
	})
	if err != nil {
		err = conservativeZoneNotFoundError(err, "read")
		if zoneIsUnambiguousNotFound(err) {
			return dnssdk.Zone{}, false, nil
		}
		return dnssdk.Zone{}, false, normalizeZoneOCIError(err)
	}
	return response.Zone, true, nil
}

func (c zoneRuntimeClient) findExistingZone(
	ctx context.Context,
	resource *dnsv1beta1.Zone,
) (dnssdk.ZoneSummary, bool, error) {
	response, err := listZonesAllPages(ctx, c.client, c.initErr, dnssdk.ListZonesRequest{
		CompartmentId: optionalString(resource.Spec.CompartmentId),
		Name:          optionalString(resource.Spec.Name),
		Scope:         dnssdk.ListZonesScopeEnum(resource.Spec.Scope),
		ViewId:        optionalString(resource.Spec.ViewId),
	})
	if err != nil {
		if zoneIsUnambiguousNotFound(err) {
			return dnssdk.ZoneSummary{}, false, nil
		}
		return dnssdk.ZoneSummary{}, false, normalizeZoneOCIError(err)
	}

	var matches []dnssdk.ZoneSummary
	for _, item := range response.Items {
		if zoneSummaryDeleted(item) || !zoneSummaryMatches(resource, item) {
			continue
		}
		matches = append(matches, item)
	}
	switch len(matches) {
	case 0:
		return dnssdk.ZoneSummary{}, false, nil
	case 1:
		return matches[0], true, nil
	default:
		return dnssdk.ZoneSummary{}, false, fmt.Errorf("zone list response returned multiple matching resources")
	}
}

func (c zoneRuntimeClient) projectSuccess(
	resource *dnsv1beta1.Zone,
	zone dnssdk.Zone,
	fallback shared.OSOKConditionType,
) (servicemanager.OSOKResponse, error) {
	if err := projectZoneStatus(resource, zone); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	status := &resource.Status.OsokStatus
	now := metav1.Now()
	if status.Ocid != "" && status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	status.UpdatedAt = &now

	condition, phase, asyncClass, shouldRequeue := zoneLifecycleProjection(zone, fallback)
	message := zoneLifecycleMessage(condition, string(zone.LifecycleState))
	status.Message = message
	status.Reason = string(condition)

	if shouldRequeue {
		current := &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           phase,
			RawStatus:       string(zone.LifecycleState),
			NormalizedClass: asyncClass,
			Message:         message,
			UpdatedAt:       &now,
		}
		projection := servicemanager.ApplyAsyncOperation(status, current, c.log)
		recordZoneAppliedDynectFingerprint(resource, projection.Condition)
		return servicemanager.OSOKResponse{
			IsSuccessful:    projection.Condition != shared.Failed,
			ShouldRequeue:   projection.ShouldRequeue,
			RequeueDuration: zoneRequeueDurationFor(projection.ShouldRequeue),
		}, nil
	} else {
		servicemanager.ClearAsyncOperation(status)
	}

	conditionStatus := v1.ConditionTrue
	if condition == shared.Failed {
		conditionStatus = v1.ConditionFalse
	}
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, condition, conditionStatus, "", message, c.log)
	recordZoneAppliedDynectFingerprint(resource, condition)
	return servicemanager.OSOKResponse{
		IsSuccessful:    condition != shared.Failed,
		ShouldRequeue:   shouldRequeue,
		RequeueDuration: zoneRequeueDurationFor(shouldRequeue),
	}, nil
}

func (c zoneRuntimeClient) seedZoneOpeningWorkRequestID(
	resource *dnsv1beta1.Zone,
	response any,
	phase shared.OSOKAsyncPhase,
) {
	if resource == nil || phase == "" {
		return
	}
	workRequestID := zoneResponseWorkRequestID(response)
	if workRequestID == "" {
		return
	}

	now := metav1.Now()
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           phase,
		WorkRequestID:   workRequestID,
		NormalizedClass: shared.OSOKAsyncClassPending,
		UpdatedAt:       &now,
	}, c.log)
}

func (c zoneRuntimeClient) fail(resource *dnsv1beta1.Zone, err error) (servicemanager.OSOKResponse, error) {
	err = normalizeZoneOCIError(err)
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	status := &resource.Status.OsokStatus
	servicemanager.RecordErrorOpcRequestID(status, err)
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	now := metav1.Now()
	status.UpdatedAt = &now
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Failed, v1.ConditionFalse, "", err.Error(), c.log)
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func newZoneRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "dns",
		FormalSlug:    "zone",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(dnssdk.ZoneLifecycleStateCreating)},
			UpdatingStates:     []string{string(dnssdk.ZoneLifecycleStateUpdating)},
			ActiveStates:       []string{string(dnssdk.ZoneLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(dnssdk.ZoneLifecycleStateDeleting)},
			TerminalStates: []string{string(dnssdk.ZoneLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "name", "scope", "viewId", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{"freeformTags", "definedTags", "resolutionMode", "dnssecState", "externalMasters", "externalDownstreams"},
			Mutable:         []string{"freeformTags", "definedTags", "resolutionMode", "dnssecState", "externalMasters", "externalDownstreams"},
			ForceNew:        []string{"name", "compartmentId", "viewId", "zoneType", "scope", "jsonData", "migrationSource", "dynectMigrationDetails"},
			ConflictsWith:   map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func zoneCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateZoneDetails", RequestName: "createZoneDetails", Contribution: "body"},
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
		{FieldName: "Scope", RequestName: "scope", Contribution: "query"},
		{FieldName: "ViewId", RequestName: "viewId", Contribution: "query"},
	}
}

func zoneGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ZoneNameOrId", RequestName: "zoneNameOrId", Contribution: "path", PreferResourceID: true},
		{FieldName: "Scope", RequestName: "scope", Contribution: "query"},
		{FieldName: "ViewId", RequestName: "viewId", Contribution: "query"},
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
	}
}

func zoneListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Name", RequestName: "name", Contribution: "query"},
		{FieldName: "Scope", RequestName: "scope", Contribution: "query"},
		{FieldName: "ViewId", RequestName: "viewId", Contribution: "query"},
	}
}

func zoneUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ZoneNameOrId", RequestName: "zoneNameOrId", Contribution: "path", PreferResourceID: true},
		{FieldName: "Scope", RequestName: "scope", Contribution: "query"},
		{FieldName: "ViewId", RequestName: "viewId", Contribution: "query"},
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
		{FieldName: "UpdateZoneDetails", RequestName: "updateZoneDetails", Contribution: "body"},
	}
}

func zoneDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ZoneNameOrId", RequestName: "zoneNameOrId", Contribution: "path", PreferResourceID: true},
		{FieldName: "Scope", RequestName: "scope", Contribution: "query"},
		{FieldName: "ViewId", RequestName: "viewId", Contribution: "query"},
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
	}
}

func buildZoneCreateBody(_ context.Context, resource *dnsv1beta1.Zone, _ string) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("zone resource is nil")
	}
	if raw := strings.TrimSpace(resource.Spec.JsonData); raw != "" {
		return zoneCreateBodyFromJSON(raw)
	}

	switch strings.ToUpper(strings.TrimSpace(resource.Spec.MigrationSource)) {
	case "", string(dnssdk.CreateZoneBaseDetailsMigrationSourceNone):
		return zoneCreateDetailsFromSpec(resource.Spec), nil
	case string(dnssdk.CreateZoneBaseDetailsMigrationSourceDynect):
		return zoneCreateMigratedDynectDetailsFromSpec(resource.Spec), nil
	default:
		return nil, fmt.Errorf("unsupported Zone migrationSource %q", resource.Spec.MigrationSource)
	}
}

func zoneCreateBodyFromJSON(raw string) (dnssdk.CreateZoneBaseDetails, error) {
	var discriminator struct {
		MigrationSource string `json:"migrationSource"`
	}
	if err := json.Unmarshal([]byte(raw), &discriminator); err != nil {
		return nil, fmt.Errorf("decode Zone jsonData discriminator: %w", err)
	}

	switch strings.ToUpper(strings.TrimSpace(discriminator.MigrationSource)) {
	case "", string(dnssdk.CreateZoneBaseDetailsMigrationSourceNone):
		var details dnssdk.CreateZoneDetails
		if err := json.Unmarshal([]byte(raw), &details); err != nil {
			return nil, fmt.Errorf("decode Zone NONE jsonData: %w", err)
		}
		return details, nil
	case string(dnssdk.CreateZoneBaseDetailsMigrationSourceDynect):
		var details dnssdk.CreateMigratedDynectZoneDetails
		if err := json.Unmarshal([]byte(raw), &details); err != nil {
			return nil, fmt.Errorf("decode Zone DYNECT jsonData: %w", err)
		}
		return details, nil
	default:
		return nil, fmt.Errorf("unsupported Zone jsonData migrationSource %q", discriminator.MigrationSource)
	}
}

func zoneCreateDetailsFromSpec(spec dnsv1beta1.ZoneSpec) dnssdk.CreateZoneDetails {
	details := dnssdk.CreateZoneDetails{
		Name:                common.String(spec.Name),
		CompartmentId:       common.String(spec.CompartmentId),
		ZoneType:            dnssdk.CreateZoneDetailsZoneTypeEnum(spec.ZoneType),
		Scope:               dnssdk.ScopeEnum(spec.Scope),
		ResolutionMode:      dnssdk.ZoneResolutionModeEnum(spec.ResolutionMode),
		DnssecState:         dnssdk.ZoneDnssecStateEnum(spec.DnssecState),
		ExternalMasters:     zoneExternalMastersFromSpec(spec.ExternalMasters),
		ExternalDownstreams: zoneExternalDownstreamsFromSpec(spec.ExternalDownstreams),
	}
	if spec.FreeformTags != nil {
		details.FreeformTags = cloneZoneFreeformTags(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		details.DefinedTags = zoneDefinedTagsFromSpec(spec.DefinedTags)
	}
	if strings.TrimSpace(spec.ViewId) != "" {
		details.ViewId = common.String(spec.ViewId)
	}
	return details
}

func zoneCreateMigratedDynectDetailsFromSpec(spec dnsv1beta1.ZoneSpec) dnssdk.CreateMigratedDynectZoneDetails {
	details := dnssdk.CreateMigratedDynectZoneDetails{
		Name:                   common.String(spec.Name),
		CompartmentId:          common.String(spec.CompartmentId),
		DynectMigrationDetails: zoneDynectMigrationDetailsFromSpec(spec.DynectMigrationDetails),
	}
	if spec.FreeformTags != nil {
		details.FreeformTags = cloneZoneFreeformTags(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		details.DefinedTags = zoneDefinedTagsFromSpec(spec.DefinedTags)
	}
	return details
}

func buildZoneUpdateBody(resource *dnsv1beta1.Zone, currentResponse any) (dnssdk.UpdateZoneDetails, bool, error) {
	if resource == nil {
		return dnssdk.UpdateZoneDetails{}, false, fmt.Errorf("zone resource is nil")
	}

	current, ok := zoneFromResponse(currentResponse)
	if !ok {
		return dnssdk.UpdateZoneDetails{}, false, fmt.Errorf("zone current response %T does not expose a Zone body", currentResponse)
	}

	details := dnssdk.UpdateZoneDetails{}
	updateNeeded := applyZoneFreeformTagsUpdate(&details, current, resource.Spec.FreeformTags)
	if applyZoneDefinedTagsUpdate(&details, current, resource.Spec.DefinedTags) {
		updateNeeded = true
	}
	if applyZoneResolutionModeUpdate(&details, current, resource.Spec.ResolutionMode) {
		updateNeeded = true
	}
	if applyZoneDnssecStateUpdate(&details, current, resource.Spec.DnssecState) {
		updateNeeded = true
	}
	if applyZoneExternalMastersUpdate(&details, current, resource.Spec.ExternalMasters) {
		updateNeeded = true
	}
	if applyZoneExternalDownstreamsUpdate(&details, current, resource.Spec.ExternalDownstreams) {
		updateNeeded = true
	}
	if updateNeeded {
		completeZoneUpdateCollections(&details, current, resource.Spec)
	}

	return details, updateNeeded, nil
}

func completeZoneUpdateCollections(
	details *dnssdk.UpdateZoneDetails,
	current dnssdk.Zone,
	spec dnsv1beta1.ZoneSpec,
) {
	if spec.FreeformTags != nil {
		details.FreeformTags = cloneZoneFreeformTags(spec.FreeformTags)
	} else {
		details.FreeformTags = cloneZoneFreeformTags(current.FreeformTags)
	}
	if spec.DefinedTags != nil {
		details.DefinedTags = zoneDefinedTagsFromSpec(spec.DefinedTags)
	} else {
		details.DefinedTags = cloneZoneDefinedTags(current.DefinedTags)
	}
	if spec.ExternalMasters != nil {
		details.ExternalMasters = zoneExternalMastersFromSpec(spec.ExternalMasters)
	} else {
		details.ExternalMasters = cloneZoneExternalMasters(current.ExternalMasters)
	}
	if spec.ExternalDownstreams != nil {
		details.ExternalDownstreams = zoneExternalDownstreamsFromSpec(spec.ExternalDownstreams)
	} else {
		details.ExternalDownstreams = cloneZoneExternalDownstreams(current.ExternalDownstreams)
	}
}

func applyZoneFreeformTagsUpdate(details *dnssdk.UpdateZoneDetails, current dnssdk.Zone, desired map[string]string) bool {
	if desired == nil || reflect.DeepEqual(current.FreeformTags, desired) {
		return false
	}
	details.FreeformTags = cloneZoneFreeformTags(desired)
	return true
}

func applyZoneDefinedTagsUpdate(details *dnssdk.UpdateZoneDetails, current dnssdk.Zone, desired map[string]shared.MapValue) bool {
	if desired == nil {
		return false
	}
	converted := zoneDefinedTagsFromSpec(desired)
	if reflect.DeepEqual(current.DefinedTags, converted) {
		return false
	}
	details.DefinedTags = converted
	return true
}

func applyZoneResolutionModeUpdate(details *dnssdk.UpdateZoneDetails, current dnssdk.Zone, desired string) bool {
	if desired == "" || string(current.ResolutionMode) == desired {
		return false
	}
	details.ResolutionMode = dnssdk.ZoneResolutionModeEnum(desired)
	return true
}

func applyZoneDnssecStateUpdate(details *dnssdk.UpdateZoneDetails, current dnssdk.Zone, desired string) bool {
	if desired == "" || string(current.DnssecState) == desired {
		return false
	}
	details.DnssecState = dnssdk.ZoneDnssecStateEnum(desired)
	return true
}

func applyZoneExternalMastersUpdate(
	details *dnssdk.UpdateZoneDetails,
	current dnssdk.Zone,
	desiredSpec []dnsv1beta1.ZoneExternalMaster,
) bool {
	if desiredSpec == nil {
		return false
	}
	desired := zoneExternalMastersFromSpec(desiredSpec)
	if zoneExternalMastersEqual(current.ExternalMasters, desired) {
		return false
	}
	details.ExternalMasters = desired
	return true
}

func applyZoneExternalDownstreamsUpdate(
	details *dnssdk.UpdateZoneDetails,
	current dnssdk.Zone,
	desiredSpec []dnsv1beta1.ZoneExternalDownstream,
) bool {
	if desiredSpec == nil {
		return false
	}
	desired := zoneExternalDownstreamsFromSpec(desiredSpec)
	if zoneExternalDownstreamsEqual(current.ExternalDownstreams, desired) {
		return false
	}
	details.ExternalDownstreams = desired
	return true
}

func projectZoneStatus(resource *dnsv1beta1.Zone, zone dnssdk.Zone) error {
	payload, err := json.Marshal(zone)
	if err != nil {
		return fmt.Errorf("marshal Zone OCI response body: %w", err)
	}
	if err := json.Unmarshal(payload, &resource.Status); err != nil {
		return fmt.Errorf("project Zone OCI response body into status: %w", err)
	}
	if id := stringValue(zone.Id); id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(id)
	}
	return nil
}

func validateZoneForceNew(resource *dnsv1beta1.Zone, current dnssdk.Zone, wasTracked bool) error {
	for _, field := range []struct {
		name    string
		desired string
		current string
		strict  bool
	}{
		{name: "name", desired: resource.Spec.Name, current: stringValue(current.Name)},
		{name: "compartmentId", desired: resource.Spec.CompartmentId, current: stringValue(current.CompartmentId)},
		{name: "viewId", desired: resource.Spec.ViewId, current: stringValue(current.ViewId), strict: true},
		{name: "zoneType", desired: resource.Spec.ZoneType, current: string(current.ZoneType)},
		{name: "scope", desired: resource.Spec.Scope, current: string(current.Scope)},
	} {
		if !zoneForceNewFieldChanged(field.desired, field.current, field.strict) {
			continue
		}
		return zoneReplacementRequiredError(field.name)
	}
	if !zoneCreateOnlyInputConfigured(resource.Spec) {
		if zoneAppliedDynectFingerprint(resource) != "" {
			return zoneReplacementRequiredError("migrationSource")
		}
		return nil
	}
	return validateZoneCreateOnlyReadback(resource, current, wasTracked)
}

func validateZoneTrackedReadMiss(resource *dnsv1beta1.Zone, currentID string) error {
	if err := validateZoneTrackedStatusForceNew(resource); err != nil {
		return err
	}
	if err := validateZoneTrackedStatusCreateOnly(resource, currentID); err != nil {
		return err
	}
	return fmt.Errorf("tracked zone %q was not found using desired scope, viewId, and compartmentId; refusing list/create fallback", currentID)
}

func validateZoneTrackedStatusForceNew(resource *dnsv1beta1.Zone) error {
	if resource == nil {
		return nil
	}
	for _, field := range []struct {
		name    string
		desired string
		current string
		known   bool
		strict  bool
	}{
		{name: "name", desired: resource.Spec.Name, current: resource.Status.Name, known: strings.TrimSpace(resource.Status.Name) != ""},
		{name: "compartmentId", desired: resource.Spec.CompartmentId, current: resource.Status.CompartmentId, known: strings.TrimSpace(resource.Status.CompartmentId) != ""},
		{name: "zoneType", desired: resource.Spec.ZoneType, current: resource.Status.ZoneType, known: strings.TrimSpace(resource.Status.ZoneType) != ""},
		{name: "scope", desired: resource.Spec.Scope, current: resource.Status.Scope, known: strings.TrimSpace(resource.Status.Scope) != ""},
		{name: "viewId", desired: resource.Spec.ViewId, current: resource.Status.ViewId, known: zoneTrackedViewIDKnown(resource), strict: true},
	} {
		if !field.known || !zoneForceNewFieldChanged(field.desired, field.current, field.strict) {
			continue
		}
		return zoneReplacementRequiredError(field.name)
	}
	return nil
}

func validateZoneTrackedStatusCreateOnly(resource *dnsv1beta1.Zone, currentID string) error {
	if resource == nil || !zoneCreateOnlyInputConfigured(resource.Spec) {
		if resource != nil && zoneAppliedDynectFingerprint(resource) != "" {
			return zoneReplacementRequiredError("migrationSource")
		}
		return nil
	}
	dynectAllowed, err := validateZoneDynectCreateOnlyFingerprint(resource, true)
	if err != nil {
		return err
	}
	if !dynectAllowed {
		if err := validateZoneUnreadableCreateOnlyInputs(resource.Spec); err != nil {
			return err
		}
	}
	if !zoneStatusHasCreateOnlyReadback(resource) {
		return nil
	}
	return validateZoneCreateOnlyReadback(resource, zoneFromStatus(resource, currentID), true)
}

func zoneForceNewFieldChanged(desired string, current string, strict bool) bool {
	desired = strings.TrimSpace(desired)
	current = strings.TrimSpace(current)
	if desired == "" && current == "" {
		return false
	}
	if !strict && (desired == "" || current == "") {
		return false
	}
	return desired != current
}

func zoneReplacementRequiredError(field string) error {
	return fmt.Errorf("zone formal semantics require replacement when %s changes", field)
}

func zoneTrackedViewIDKnown(resource *dnsv1beta1.Zone) bool {
	if strings.TrimSpace(resource.Status.ViewId) != "" {
		return true
	}
	return strings.TrimSpace(resource.Status.Scope) != ""
}

func zoneStatusHasCreateOnlyReadback(resource *dnsv1beta1.Zone) bool {
	status := resource.Status
	return zoneStatusHasCreateOnlyScalarReadback(status) || zoneStatusHasCreateOnlyCollectionReadback(status)
}

func zoneStatusHasCreateOnlyScalarReadback(status dnsv1beta1.ZoneStatus) bool {
	for _, value := range []string{
		status.Name,
		status.CompartmentId,
		status.ZoneType,
		status.Scope,
		status.ViewId,
		status.ResolutionMode,
		status.DnssecState,
	} {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}

func zoneStatusHasCreateOnlyCollectionReadback(status dnsv1beta1.ZoneStatus) bool {
	return status.FreeformTags != nil ||
		status.DefinedTags != nil ||
		status.ExternalMasters != nil ||
		status.ExternalDownstreams != nil
}

func zoneFromStatus(resource *dnsv1beta1.Zone, currentID string) dnssdk.Zone {
	status := resource.Status
	return dnssdk.Zone{
		Name:                optionalString(status.Name),
		ZoneType:            dnssdk.ZoneZoneTypeEnum(status.ZoneType),
		CompartmentId:       optionalString(status.CompartmentId),
		Scope:               dnssdk.ScopeEnum(status.Scope),
		FreeformTags:        cloneZoneFreeformTags(status.FreeformTags),
		DefinedTags:         zoneDefinedTagsFromSpec(status.DefinedTags),
		ResolutionMode:      dnssdk.ZoneResolutionModeEnum(status.ResolutionMode),
		DnssecState:         dnssdk.ZoneDnssecStateEnum(status.DnssecState),
		ExternalMasters:     zoneExternalMastersFromSpec(status.ExternalMasters),
		ExternalDownstreams: zoneExternalDownstreamsFromSpec(status.ExternalDownstreams),
		Id:                  optionalString(currentID),
		ViewId:              optionalString(status.ViewId),
	}
}

func validateZoneCreateOnlyReadback(resource *dnsv1beta1.Zone, current dnssdk.Zone, wasTracked bool) error {
	dynectAllowed, err := validateZoneDynectCreateOnlyFingerprint(resource, wasTracked)
	if err != nil {
		return err
	}
	if !dynectAllowed {
		if err := validateZoneUnreadableCreateOnlyInputs(resource.Spec); err != nil {
			return err
		}
	}
	body, err := buildZoneCreateBody(context.Background(), resource, resource.Namespace)
	if err != nil {
		return err
	}
	details, ok := body.(dnssdk.CreateZoneBaseDetails)
	if !ok {
		return fmt.Errorf("zone create body %T does not implement dns.CreateZoneBaseDetails", body)
	}
	if driftPaths := zoneCreateOnlyDriftPaths(resource.Spec, details, current); len(driftPaths) > 0 {
		return fmt.Errorf("zone formal semantics require replacement when %s changes", strings.Join(driftPaths, ", "))
	}
	return nil
}

func zoneCreateOnlyInputConfigured(spec dnsv1beta1.ZoneSpec) bool {
	return strings.TrimSpace(spec.JsonData) != "" ||
		strings.TrimSpace(spec.MigrationSource) != "" ||
		zoneDynectMigrationDetailsConfigured(spec.DynectMigrationDetails)
}

func validateZoneDynectCreateOnlyFingerprint(resource *dnsv1beta1.Zone, wasTracked bool) (bool, error) {
	desired, hasDynect, err := zoneDynectCreateOnlyFingerprint(resource.Spec)
	if err != nil {
		return false, err
	}
	applied := zoneAppliedDynectFingerprint(resource)
	if applied != "" {
		if !hasDynect {
			return false, zoneReplacementRequiredError("migrationSource")
		}
		if desired != applied {
			return false, zoneReplacementRequiredError("migrationSource, dynectMigrationDetails")
		}
		return true, nil
	}
	if hasDynect && wasTracked {
		return false, validateZoneUnreadableCreateOnlyInputs(resource.Spec)
	}
	return hasDynect, nil
}

func zoneDynectCreateOnlyFingerprint(spec dnsv1beta1.ZoneSpec) (string, bool, error) {
	if raw := strings.TrimSpace(spec.JsonData); raw != "" {
		body, err := zoneCreateBodyFromJSON(raw)
		if err != nil {
			return "", false, err
		}
		details, ok := body.(dnssdk.CreateMigratedDynectZoneDetails)
		if !ok {
			return "", false, nil
		}
		return zoneDynectCreateOnlyDetailsFingerprint(details.DynectMigrationDetails)
	}

	if strings.ToUpper(strings.TrimSpace(spec.MigrationSource)) != string(dnssdk.CreateZoneBaseDetailsMigrationSourceDynect) {
		return "", false, nil
	}
	return zoneDynectCreateOnlyDetailsFingerprint(zoneDynectMigrationDetailsFromSpec(spec.DynectMigrationDetails))
}

func zoneDynectCreateOnlyDetailsFingerprint(details *dnssdk.DynectMigrationDetails) (string, bool, error) {
	payload := struct {
		MigrationSource        string                         `json:"migrationSource"`
		DynectMigrationDetails *dnssdk.DynectMigrationDetails `json:"dynectMigrationDetails,omitempty"`
	}{
		MigrationSource:        string(dnssdk.CreateZoneBaseDetailsMigrationSourceDynect),
		DynectMigrationDetails: details,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", false, fmt.Errorf("marshal Zone Dynect create-only fingerprint: %w", err)
	}
	return sha256Sum(string(encoded)), true, nil
}

func zoneAppliedDynectFingerprint(resource *dnsv1beta1.Zone) string {
	if resource == nil {
		return ""
	}
	conditions := resource.Status.OsokStatus.Conditions
	for i := len(conditions) - 1; i >= 0; i-- {
		reason := strings.TrimSpace(conditions[i].Reason)
		if strings.HasPrefix(reason, zoneDynectCreateOnlyReasonPrefix) {
			return strings.TrimPrefix(reason, zoneDynectCreateOnlyReasonPrefix)
		}
	}
	return ""
}

func recordZoneAppliedDynectFingerprint(resource *dnsv1beta1.Zone, conditionType shared.OSOKConditionType) {
	if resource == nil {
		return
	}
	fingerprint, ok, err := zoneDynectCreateOnlyFingerprint(resource.Spec)
	if err != nil || !ok {
		return
	}
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		return
	}
	index := len(conditions) - 1
	for i := len(conditions) - 1; i >= 0; i-- {
		if conditions[i].Type == conditionType {
			index = i
			break
		}
	}
	conditions[index].Reason = zoneDynectCreateOnlyReasonPrefix + fingerprint
	resource.Status.OsokStatus.Conditions = conditions
}

func validateZoneUnreadableCreateOnlyInputs(spec dnsv1beta1.ZoneSpec) error {
	driftPaths, err := zoneUnreadableCreateOnlyDriftPaths(spec)
	if err != nil {
		return err
	}
	if len(driftPaths) == 0 {
		return nil
	}
	return fmt.Errorf("zone formal semantics require replacement when %s changes", strings.Join(driftPaths, ", "))
}

func zoneUnreadableCreateOnlyDriftPaths(spec dnsv1beta1.ZoneSpec) ([]string, error) {
	if raw := strings.TrimSpace(spec.JsonData); raw != "" {
		return zoneUnreadableJSONDataDriftPaths(raw)
	}

	var paths []string
	switch strings.ToUpper(strings.TrimSpace(spec.MigrationSource)) {
	case "", string(dnssdk.CreateZoneBaseDetailsMigrationSourceNone):
	case string(dnssdk.CreateZoneBaseDetailsMigrationSourceDynect):
		paths = append(paths, "migrationSource")
	default:
		return nil, fmt.Errorf("unsupported Zone migrationSource %q", spec.MigrationSource)
	}
	if zoneDynectMigrationDetailsConfigured(spec.DynectMigrationDetails) {
		paths = append(paths, "dynectMigrationDetails")
	}
	return paths, nil
}

func zoneUnreadableJSONDataDriftPaths(raw string) ([]string, error) {
	var discriminator struct {
		MigrationSource        string          `json:"migrationSource"`
		DynectMigrationDetails json.RawMessage `json:"dynectMigrationDetails"`
	}
	if err := json.Unmarshal([]byte(raw), &discriminator); err != nil {
		return nil, fmt.Errorf("decode Zone jsonData discriminator: %w", err)
	}

	var paths []string
	switch strings.ToUpper(strings.TrimSpace(discriminator.MigrationSource)) {
	case "", string(dnssdk.CreateZoneBaseDetailsMigrationSourceNone):
	case string(dnssdk.CreateZoneBaseDetailsMigrationSourceDynect):
		paths = append(paths, "jsonData.migrationSource")
	default:
		return nil, fmt.Errorf("unsupported Zone jsonData migrationSource %q", discriminator.MigrationSource)
	}
	if len(discriminator.DynectMigrationDetails) > 0 && string(discriminator.DynectMigrationDetails) != "null" {
		paths = append(paths, "jsonData.dynectMigrationDetails")
	}
	return paths, nil
}

func zoneCreateOnlyDriftPaths(spec dnsv1beta1.ZoneSpec, body dnssdk.CreateZoneBaseDetails, current dnssdk.Zone) []string {
	prefix := zoneCreateOnlyDriftPrefix(spec)
	switch details := body.(type) {
	case dnssdk.CreateZoneDetails:
		return zoneCreateDetailsDriftPaths(prefix, spec, details, current)
	case dnssdk.CreateMigratedDynectZoneDetails:
		return zoneMigratedDynectCreateDetailsDriftPaths(prefix, spec, details, current)
	default:
		return []string{prefix}
	}
}

func zoneCreateOnlyDriftPrefix(spec dnsv1beta1.ZoneSpec) string {
	if strings.TrimSpace(spec.JsonData) != "" {
		return "jsonData"
	}
	if zoneDynectMigrationDetailsConfigured(spec.DynectMigrationDetails) {
		return "dynectMigrationDetails"
	}
	return "migrationSource"
}

func zoneCreateDetailsDriftPaths(
	prefix string,
	spec dnsv1beta1.ZoneSpec,
	details dnssdk.CreateZoneDetails,
	current dnssdk.Zone,
) []string {
	var paths []string
	paths = append(paths, zoneCreateDetailsBaseDriftPaths(prefix, details, current)...)
	paths = append(paths, zoneCreateDetailsMutableDriftPaths(prefix, spec, details, current)...)
	return compactZoneDriftPaths(paths)
}

func zoneCreateDetailsBaseDriftPaths(
	prefix string,
	details dnssdk.CreateZoneDetails,
	current dnssdk.Zone,
) []string {
	return []string{
		zoneStringCreateOnlyDriftPath(prefix, "name", stringValue(details.Name), stringValue(current.Name)),
		zoneStringCreateOnlyDriftPath(prefix, "compartmentId", stringValue(details.CompartmentId), stringValue(current.CompartmentId)),
		zoneStringCreateOnlyDriftPath(prefix, "viewId", stringValue(details.ViewId), stringValue(current.ViewId)),
		zoneStringCreateOnlyDriftPath(prefix, "zoneType", string(details.ZoneType), string(current.ZoneType)),
		zoneStringCreateOnlyDriftPath(prefix, "scope", string(details.Scope), string(current.Scope)),
	}
}

func zoneCreateDetailsMutableDriftPaths(
	prefix string,
	spec dnsv1beta1.ZoneSpec,
	details dnssdk.CreateZoneDetails,
	current dnssdk.Zone,
) []string {
	var paths []string
	paths = append(paths, zoneCreateDetailsScalarDriftPaths(prefix, spec, details, current)...)
	paths = append(paths, zoneCreateDetailsCollectionDriftPaths(prefix, spec, details, current)...)
	return paths
}

func zoneCreateDetailsScalarDriftPaths(
	prefix string,
	spec dnsv1beta1.ZoneSpec,
	details dnssdk.CreateZoneDetails,
	current dnssdk.Zone,
) []string {
	var paths []string
	if spec.ResolutionMode == "" {
		paths = append(paths, zoneStringCreateOnlyDriftPath(prefix, "resolutionMode", string(details.ResolutionMode), string(current.ResolutionMode)))
	}
	if spec.DnssecState == "" {
		paths = append(paths, zoneStringCreateOnlyDriftPath(prefix, "dnssecState", string(details.DnssecState), string(current.DnssecState)))
	}
	return paths
}

func zoneCreateDetailsCollectionDriftPaths(
	prefix string,
	spec dnsv1beta1.ZoneSpec,
	details dnssdk.CreateZoneDetails,
	current dnssdk.Zone,
) []string {
	return []string{
		zoneFreeformTagsDriftPath(prefix, spec, details, current),
		zoneDefinedTagsDriftPath(prefix, spec, details, current),
		zoneExternalMastersDriftPath(prefix, spec, details, current),
		zoneExternalDownstreamsDriftPath(prefix, spec, details, current),
	}
}

func zoneFreeformTagsDriftPath(
	prefix string,
	spec dnsv1beta1.ZoneSpec,
	details dnssdk.CreateZoneDetails,
	current dnssdk.Zone,
) string {
	if spec.FreeformTags == nil && details.FreeformTags != nil && !reflect.DeepEqual(details.FreeformTags, current.FreeformTags) {
		return prefix + ".freeformTags"
	}
	return ""
}

func zoneDefinedTagsDriftPath(
	prefix string,
	spec dnsv1beta1.ZoneSpec,
	details dnssdk.CreateZoneDetails,
	current dnssdk.Zone,
) string {
	if spec.DefinedTags == nil && details.DefinedTags != nil && !reflect.DeepEqual(details.DefinedTags, current.DefinedTags) {
		return prefix + ".definedTags"
	}
	return ""
}

func zoneExternalMastersDriftPath(
	prefix string,
	spec dnsv1beta1.ZoneSpec,
	details dnssdk.CreateZoneDetails,
	current dnssdk.Zone,
) string {
	if spec.ExternalMasters == nil && details.ExternalMasters != nil && !zoneExternalMastersEqual(details.ExternalMasters, current.ExternalMasters) {
		return prefix + ".externalMasters"
	}
	return ""
}

func zoneExternalDownstreamsDriftPath(
	prefix string,
	spec dnsv1beta1.ZoneSpec,
	details dnssdk.CreateZoneDetails,
	current dnssdk.Zone,
) string {
	if spec.ExternalDownstreams == nil && details.ExternalDownstreams != nil && !zoneExternalDownstreamsEqual(details.ExternalDownstreams, current.ExternalDownstreams) {
		return prefix + ".externalDownstreams"
	}
	return ""
}

func zoneMigratedDynectCreateDetailsDriftPaths(
	prefix string,
	spec dnsv1beta1.ZoneSpec,
	details dnssdk.CreateMigratedDynectZoneDetails,
	current dnssdk.Zone,
) []string {
	var paths []string
	paths = append(paths,
		zoneStringCreateOnlyDriftPath(prefix, "name", stringValue(details.Name), stringValue(current.Name)),
		zoneStringCreateOnlyDriftPath(prefix, "compartmentId", stringValue(details.CompartmentId), stringValue(current.CompartmentId)),
	)
	if spec.FreeformTags == nil && details.FreeformTags != nil && !reflect.DeepEqual(details.FreeformTags, current.FreeformTags) {
		paths = append(paths, prefix+".freeformTags")
	}
	if spec.DefinedTags == nil && details.DefinedTags != nil && !reflect.DeepEqual(details.DefinedTags, current.DefinedTags) {
		paths = append(paths, prefix+".definedTags")
	}
	return compactZoneDriftPaths(paths)
}

func zoneStringCreateOnlyDriftPath(prefix string, field string, desired string, current string) string {
	if strings.TrimSpace(desired) == "" {
		return ""
	}
	if desired == current {
		return ""
	}
	return prefix + "." + field
}

func compactZoneDriftPaths(paths []string) []string {
	compacted := make([]string, 0, len(paths))
	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			continue
		}
		compacted = append(compacted, path)
	}
	return compacted
}

func zoneRequeueDurationFor(shouldRequeue bool) time.Duration {
	if shouldRequeue {
		return zoneRequeueDuration
	}
	return 0
}

func zoneDynectMigrationDetailsConfigured(details dnsv1beta1.ZoneDynectMigrationDetails) bool {
	return strings.TrimSpace(details.CustomerName) != "" ||
		strings.TrimSpace(details.Username) != "" ||
		strings.TrimSpace(details.Password) != "" ||
		len(details.HttpRedirectReplacements) > 0
}

func zoneLifecycleProjection(
	zone dnssdk.Zone,
	fallback shared.OSOKConditionType,
) (shared.OSOKConditionType, shared.OSOKAsyncPhase, shared.OSOKAsyncNormalizedClass, bool) {
	switch strings.ToUpper(string(zone.LifecycleState)) {
	case string(dnssdk.ZoneLifecycleStateCreating):
		return shared.Provisioning, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassPending, true
	case string(dnssdk.ZoneLifecycleStateUpdating):
		return shared.Updating, shared.OSOKAsyncPhaseUpdate, shared.OSOKAsyncClassPending, true
	case string(dnssdk.ZoneLifecycleStateDeleting):
		return shared.Terminating, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending, true
	case string(dnssdk.ZoneLifecycleStateDeleted):
		return shared.Terminating, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassSucceeded, true
	case string(dnssdk.ZoneLifecycleStateFailed):
		return shared.Failed, "", shared.OSOKAsyncClassFailed, false
	case string(dnssdk.ZoneLifecycleStateActive):
		return shared.Active, "", shared.OSOKAsyncClassSucceeded, false
	default:
		return fallback, "", shared.OSOKAsyncClassUnknown, false
	}
}

func zoneLifecyclePreventsUpdate(zone dnssdk.Zone) bool {
	switch strings.ToUpper(string(zone.LifecycleState)) {
	case string(dnssdk.ZoneLifecycleStateCreating),
		string(dnssdk.ZoneLifecycleStateUpdating),
		string(dnssdk.ZoneLifecycleStateDeleting),
		string(dnssdk.ZoneLifecycleStateDeleted),
		string(dnssdk.ZoneLifecycleStateFailed):
		return true
	default:
		return false
	}
}

func zoneLifecycleMessage(condition shared.OSOKConditionType, lifecycle string) string {
	lifecycle = strings.TrimSpace(lifecycle)
	if lifecycle == "" {
		return fmt.Sprintf("OCI Zone is %s", condition)
	}
	return fmt.Sprintf("OCI Zone lifecycle state is %s", lifecycle)
}

func currentZoneID(resource *dnsv1beta1.Zone) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); id != "" {
		return id
	}
	return strings.TrimSpace(resource.Status.Id)
}

func zoneSummaryMatches(resource *dnsv1beta1.Zone, summary dnssdk.ZoneSummary) bool {
	spec := resource.Spec
	for _, field := range []struct {
		desired string
		current string
	}{
		{desired: spec.Name, current: stringValue(summary.Name)},
		{desired: spec.CompartmentId, current: stringValue(summary.CompartmentId)},
		{desired: spec.Scope, current: string(summary.Scope)},
		{desired: spec.ViewId, current: stringValue(summary.ViewId)},
		{desired: spec.ZoneType, current: string(summary.ZoneType)},
	} {
		if field.desired != "" && field.desired != field.current {
			return false
		}
	}
	return true
}

func zoneSummaryDeleted(summary dnssdk.ZoneSummary) bool {
	switch strings.ToUpper(string(summary.LifecycleState)) {
	case string(dnssdk.ZoneSummaryLifecycleStateDeleted), string(dnssdk.ZoneSummaryLifecycleStateDeleting):
		return true
	default:
		return false
	}
}

func zoneRetryToken(resource *dnsv1beta1.Zone) *string {
	if resource == nil {
		return nil
	}
	if uid := strings.TrimSpace(string(resource.UID)); uid != "" {
		return common.String(uid)
	}
	source := strings.TrimSpace(resource.Namespace + "/" + resource.Name)
	if source == "/" {
		return nil
	}
	sum := sha256Sum(source)
	return common.String(sum)
}

func sha256Sum(source string) string {
	sum := sha256.Sum256([]byte(source))
	return fmt.Sprintf("%x", sum[:16])
}

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

func intValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func zoneResponseWorkRequestID(response any) string {
	switch typed := response.(type) {
	case dnssdk.CreateZoneResponse:
		return stringValue(typed.OpcWorkRequestId)
	case *dnssdk.CreateZoneResponse:
		if typed == nil {
			return ""
		}
		return stringValue(typed.OpcWorkRequestId)
	case dnssdk.UpdateZoneResponse:
		return stringValue(typed.OpcWorkRequestId)
	case *dnssdk.UpdateZoneResponse:
		if typed == nil {
			return ""
		}
		return stringValue(typed.OpcWorkRequestId)
	case dnssdk.DeleteZoneResponse:
		return stringValue(typed.OpcWorkRequestId)
	case *dnssdk.DeleteZoneResponse:
		if typed == nil {
			return ""
		}
		return stringValue(typed.OpcWorkRequestId)
	default:
		return ""
	}
}

func zoneIsUnambiguousNotFound(err error) bool {
	return errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound()
}

func normalizeZoneOCIError(err error) error {
	if err == nil {
		return nil
	}
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		if _, ok := err.(common.ServiceError); ok {
			_, normalized := errorutil.OciErrorTypeResponse(err)
			if normalized != nil {
				return normalized
			}
		}
		if _, normalized := errorutil.NewServiceFailureFromResponse(
			serviceErr.GetCode(),
			serviceErr.GetHTTPStatusCode(),
			serviceErr.GetOpcRequestID(),
			serviceErr.GetMessage(),
		); normalized != nil {
			return normalized
		}
	}
	return err
}

func listZonesAllPages(
	ctx context.Context,
	client zoneOCIClient,
	initErr error,
	request dnssdk.ListZonesRequest,
) (dnssdk.ListZonesResponse, error) {
	if err := zoneClientReady(client, initErr); err != nil {
		return dnssdk.ListZonesResponse{}, err
	}

	var combined dnssdk.ListZonesResponse
	for {
		response, err := client.ListZones(ctx, request)
		if err != nil {
			return dnssdk.ListZonesResponse{}, conservativeZoneNotFoundError(err, "list")
		}
		combined.RawResponse = response.RawResponse
		combined.OpcRequestId = response.OpcRequestId
		combined.OpcTotalItems = response.OpcTotalItems
		combined.Items = append(combined.Items, response.Items...)
		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			combined.OpcNextPage = nil
			return combined, nil
		}
		request.Page = response.OpcNextPage
		combined.OpcNextPage = response.OpcNextPage
	}
}

func handleZoneDeleteError(resource *dnsv1beta1.Zone, err error) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return err
}

func conservativeZoneNotFoundError(err error, operation string) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	message := fmt.Sprintf("Zone %s returned ambiguous 404 NotAuthorizedOrNotFound: %s", strings.TrimSpace(operation), err.Error())
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		return ambiguousZoneNotFoundError{message: message, opcRequestID: serviceErr.GetOpcRequestID()}
	}
	return ambiguousZoneNotFoundError{message: message}
}

func zoneClientReady(client zoneOCIClient, initErr error) error {
	if initErr != nil {
		return fmt.Errorf("initialize Zone OCI client: %w", initErr)
	}
	if client == nil {
		return fmt.Errorf("zone OCI client is not configured")
	}
	return nil
}

func zoneFromResponse(response any) (dnssdk.Zone, bool) {
	value := reflect.ValueOf(response)
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return dnssdk.Zone{}, false
		}
		return zoneFromResponse(value.Elem().Interface())
	}

	switch current := response.(type) {
	case dnssdk.CreateZoneResponse:
		return current.Zone, true
	case dnssdk.GetZoneResponse:
		return current.Zone, true
	case dnssdk.UpdateZoneResponse:
		return current.Zone, true
	case dnssdk.Zone:
		return current, true
	case dnssdk.ZoneSummary:
		return zoneFromSummary(current), true
	default:
		return dnssdk.Zone{}, false
	}
}

func zoneFromSummary(summary dnssdk.ZoneSummary) dnssdk.Zone {
	return dnssdk.Zone{
		Name:            summary.Name,
		ZoneType:        dnssdk.ZoneZoneTypeEnum(summary.ZoneType),
		CompartmentId:   summary.CompartmentId,
		Scope:           summary.Scope,
		FreeformTags:    summary.FreeformTags,
		DefinedTags:     summary.DefinedTags,
		ResolutionMode:  summary.ResolutionMode,
		DnssecState:     summary.DnssecState,
		Self:            summary.Self,
		Id:              summary.Id,
		TimeCreated:     summary.TimeCreated,
		Version:         summary.Version,
		Serial:          summary.Serial,
		LifecycleState:  dnssdk.ZoneLifecycleStateEnum(summary.LifecycleState),
		IsProtected:     summary.IsProtected,
		ViewId:          summary.ViewId,
		DnssecConfig:    summary.DnssecConfig,
		Nameservers:     nil,
		ExternalMasters: nil,
	}
}

func zoneExternalMastersFromSpec(source []dnsv1beta1.ZoneExternalMaster) []dnssdk.ExternalMaster {
	if source == nil {
		return nil
	}
	converted := make([]dnssdk.ExternalMaster, 0, len(source))
	for _, item := range source {
		master := dnssdk.ExternalMaster{
			Address: common.String(item.Address),
		}
		if item.Port != 0 {
			master.Port = common.Int(item.Port)
		}
		if strings.TrimSpace(item.TsigKeyId) != "" {
			master.TsigKeyId = common.String(item.TsigKeyId)
		}
		converted = append(converted, master)
	}
	return converted
}

func zoneExternalDownstreamsFromSpec(source []dnsv1beta1.ZoneExternalDownstream) []dnssdk.ExternalDownstream {
	if source == nil {
		return nil
	}
	converted := make([]dnssdk.ExternalDownstream, 0, len(source))
	for _, item := range source {
		downstream := dnssdk.ExternalDownstream{
			Address: common.String(item.Address),
		}
		if item.Port != 0 {
			downstream.Port = common.Int(item.Port)
		}
		if strings.TrimSpace(item.TsigKeyId) != "" {
			downstream.TsigKeyId = common.String(item.TsigKeyId)
		}
		converted = append(converted, downstream)
	}
	return converted
}

func zoneExternalMastersEqual(left []dnssdk.ExternalMaster, right []dnssdk.ExternalMaster) bool {
	return reflect.DeepEqual(canonicalZoneExternalMasters(left), canonicalZoneExternalMasters(right))
}

func canonicalZoneExternalMasters(source []dnssdk.ExternalMaster) []zoneExternalTransferEndpoint {
	if len(source) == 0 {
		return nil
	}
	canonical := make([]zoneExternalTransferEndpoint, 0, len(source))
	for _, item := range source {
		canonical = append(canonical, zoneExternalTransferEndpoint{
			Address:   stringValue(item.Address),
			Port:      canonicalZoneTransferPort(item.Port),
			TsigKeyID: stringValue(item.TsigKeyId),
		})
	}
	return canonical
}

func zoneExternalDownstreamsEqual(left []dnssdk.ExternalDownstream, right []dnssdk.ExternalDownstream) bool {
	return reflect.DeepEqual(canonicalZoneExternalDownstreams(left), canonicalZoneExternalDownstreams(right))
}

func canonicalZoneExternalDownstreams(source []dnssdk.ExternalDownstream) []zoneExternalTransferEndpoint {
	if len(source) == 0 {
		return nil
	}
	canonical := make([]zoneExternalTransferEndpoint, 0, len(source))
	for _, item := range source {
		canonical = append(canonical, zoneExternalTransferEndpoint{
			Address:   stringValue(item.Address),
			Port:      canonicalZoneTransferPort(item.Port),
			TsigKeyID: stringValue(item.TsigKeyId),
		})
	}
	return canonical
}

type zoneExternalTransferEndpoint struct {
	Address   string
	Port      int
	TsigKeyID string
}

func canonicalZoneTransferPort(port *int) int {
	value := intValue(port)
	if value == 53 {
		return 0
	}
	return value
}

func zoneDynectMigrationDetailsFromSpec(source dnsv1beta1.ZoneDynectMigrationDetails) *dnssdk.DynectMigrationDetails {
	return &dnssdk.DynectMigrationDetails{
		CustomerName:             common.String(source.CustomerName),
		Username:                 common.String(source.Username),
		Password:                 common.String(source.Password),
		HttpRedirectReplacements: zoneMigrationReplacementsFromSpec(source.HttpRedirectReplacements),
	}
}

func zoneMigrationReplacementsFromSpec(
	source map[string][]dnsv1beta1.ZoneDynectMigrationDetailsHttpRedirectReplacements,
) map[string][]dnssdk.MigrationReplacement {
	if source == nil {
		return nil
	}
	converted := make(map[string][]dnssdk.MigrationReplacement, len(source))
	for domain, replacements := range source {
		next := make([]dnssdk.MigrationReplacement, 0, len(replacements))
		for _, item := range replacements {
			replacement := dnssdk.MigrationReplacement{
				Rtype: common.String(item.Rtype),
				Ttl:   common.Int(item.Ttl),
				Rdata: common.String(item.Rdata),
			}
			if strings.TrimSpace(item.SubstituteRtype) != "" {
				replacement.SubstituteRtype = common.String(item.SubstituteRtype)
			}
			next = append(next, replacement)
		}
		converted[domain] = next
	}
	return converted
}

func zoneDefinedTagsFromSpec(source map[string]shared.MapValue) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	return *util.ConvertToOciDefinedTags(&source)
}

func cloneZoneDefinedTags(source map[string]map[string]interface{}) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	clone := make(map[string]map[string]interface{}, len(source))
	for namespace, values := range source {
		if values == nil {
			clone[namespace] = nil
			continue
		}
		nested := make(map[string]interface{}, len(values))
		for key, value := range values {
			nested[key] = value
		}
		clone[namespace] = nested
	}
	return clone
}

func cloneZoneFreeformTags(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func cloneZoneExternalMasters(source []dnssdk.ExternalMaster) []dnssdk.ExternalMaster {
	if source == nil {
		return nil
	}
	clone := make([]dnssdk.ExternalMaster, 0, len(source))
	for _, item := range source {
		master := dnssdk.ExternalMaster{
			Address: optionalString(stringValue(item.Address)),
		}
		if item.Port != nil {
			master.Port = common.Int(intValue(item.Port))
		}
		if strings.TrimSpace(stringValue(item.TsigKeyId)) != "" {
			master.TsigKeyId = optionalString(stringValue(item.TsigKeyId))
		}
		clone = append(clone, master)
	}
	return clone
}

func cloneZoneExternalDownstreams(source []dnssdk.ExternalDownstream) []dnssdk.ExternalDownstream {
	if source == nil {
		return nil
	}
	clone := make([]dnssdk.ExternalDownstream, 0, len(source))
	for _, item := range source {
		downstream := dnssdk.ExternalDownstream{
			Address: optionalString(stringValue(item.Address)),
		}
		if item.Port != nil {
			downstream.Port = common.Int(intValue(item.Port))
		}
		if strings.TrimSpace(stringValue(item.TsigKeyId)) != "" {
			downstream.TsigKeyId = optionalString(stringValue(item.TsigKeyId))
		}
		clone = append(clone, downstream)
	}
	return clone
}

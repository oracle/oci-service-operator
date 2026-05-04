/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package monitoredregion

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	resourceanalyticssdk "github.com/oracle/oci-go-sdk/v65/resourceanalytics"
	resourceanalyticsv1beta1 "github.com/oracle/oci-service-operator/api/resourceanalytics/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type monitoredRegionOCIClient interface {
	CreateMonitoredRegion(context.Context, resourceanalyticssdk.CreateMonitoredRegionRequest) (resourceanalyticssdk.CreateMonitoredRegionResponse, error)
	GetMonitoredRegion(context.Context, resourceanalyticssdk.GetMonitoredRegionRequest) (resourceanalyticssdk.GetMonitoredRegionResponse, error)
	ListMonitoredRegions(context.Context, resourceanalyticssdk.ListMonitoredRegionsRequest) (resourceanalyticssdk.ListMonitoredRegionsResponse, error)
	DeleteMonitoredRegion(context.Context, resourceanalyticssdk.DeleteMonitoredRegionRequest) (resourceanalyticssdk.DeleteMonitoredRegionResponse, error)
}

type ambiguousMonitoredRegionNotFoundError struct {
	message      string
	opcRequestID string
}

func (e ambiguousMonitoredRegionNotFoundError) Error() string {
	return e.message
}

func (e ambiguousMonitoredRegionNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerMonitoredRegionRuntimeHooksMutator(func(_ *MonitoredRegionServiceManager, hooks *MonitoredRegionRuntimeHooks) {
		applyMonitoredRegionRuntimeHooks(hooks)
	})
}

func applyMonitoredRegionRuntimeHooks(hooks *MonitoredRegionRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = monitoredRegionRuntimeSemantics()
	hooks.BuildCreateBody = buildMonitoredRegionCreateBody
	hooks.List.Fields = monitoredRegionListFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listMonitoredRegionsAllPages(hooks.List.Call)
	}
	hooks.Identity.GuardExistingBeforeCreate = guardMonitoredRegionExistingBeforeCreate
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateMonitoredRegionCreateOnlyDrift
	hooks.DeleteHooks.HandleError = handleMonitoredRegionDeleteError
	wrapMonitoredRegionDeleteConfirmation(hooks)
}

func newMonitoredRegionServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client monitoredRegionOCIClient,
) MonitoredRegionServiceClient {
	hooks := newMonitoredRegionRuntimeHooksWithOCIClient(client)
	applyMonitoredRegionRuntimeHooks(&hooks)
	delegate := defaultMonitoredRegionServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*resourceanalyticsv1beta1.MonitoredRegion](
			buildMonitoredRegionGeneratedRuntimeConfig(&MonitoredRegionServiceManager{Log: log}, hooks),
		),
	}
	return wrapMonitoredRegionGeneratedClient(hooks, delegate)
}

func newMonitoredRegionRuntimeHooksWithOCIClient(client monitoredRegionOCIClient) MonitoredRegionRuntimeHooks {
	return MonitoredRegionRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*resourceanalyticsv1beta1.MonitoredRegion]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*resourceanalyticsv1beta1.MonitoredRegion]{},
		StatusHooks:     generatedruntime.StatusHooks[*resourceanalyticsv1beta1.MonitoredRegion]{},
		ParityHooks:     generatedruntime.ParityHooks[*resourceanalyticsv1beta1.MonitoredRegion]{},
		Async:           generatedruntime.AsyncHooks[*resourceanalyticsv1beta1.MonitoredRegion]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*resourceanalyticsv1beta1.MonitoredRegion]{},
		Create: runtimeOperationHooks[resourceanalyticssdk.CreateMonitoredRegionRequest, resourceanalyticssdk.CreateMonitoredRegionResponse]{
			Fields: monitoredRegionCreateFields(),
			Call: func(ctx context.Context, request resourceanalyticssdk.CreateMonitoredRegionRequest) (resourceanalyticssdk.CreateMonitoredRegionResponse, error) {
				if client == nil {
					return resourceanalyticssdk.CreateMonitoredRegionResponse{}, fmt.Errorf("monitoredregion OCI client is nil")
				}
				return client.CreateMonitoredRegion(ctx, request)
			},
		},
		Get: runtimeOperationHooks[resourceanalyticssdk.GetMonitoredRegionRequest, resourceanalyticssdk.GetMonitoredRegionResponse]{
			Fields: monitoredRegionGetFields(),
			Call: func(ctx context.Context, request resourceanalyticssdk.GetMonitoredRegionRequest) (resourceanalyticssdk.GetMonitoredRegionResponse, error) {
				if client == nil {
					return resourceanalyticssdk.GetMonitoredRegionResponse{}, fmt.Errorf("monitoredregion OCI client is nil")
				}
				return client.GetMonitoredRegion(ctx, request)
			},
		},
		List: runtimeOperationHooks[resourceanalyticssdk.ListMonitoredRegionsRequest, resourceanalyticssdk.ListMonitoredRegionsResponse]{
			Fields: monitoredRegionListFields(),
			Call: func(ctx context.Context, request resourceanalyticssdk.ListMonitoredRegionsRequest) (resourceanalyticssdk.ListMonitoredRegionsResponse, error) {
				if client == nil {
					return resourceanalyticssdk.ListMonitoredRegionsResponse{}, fmt.Errorf("monitoredregion OCI client is nil")
				}
				return client.ListMonitoredRegions(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[resourceanalyticssdk.DeleteMonitoredRegionRequest, resourceanalyticssdk.DeleteMonitoredRegionResponse]{
			Fields: monitoredRegionDeleteFields(),
			Call: func(ctx context.Context, request resourceanalyticssdk.DeleteMonitoredRegionRequest) (resourceanalyticssdk.DeleteMonitoredRegionResponse, error) {
				if client == nil {
					return resourceanalyticssdk.DeleteMonitoredRegionResponse{}, fmt.Errorf("monitoredregion OCI client is nil")
				}
				return client.DeleteMonitoredRegion(ctx, request)
			},
		},
		WrapGeneratedClient: []func(MonitoredRegionServiceClient) MonitoredRegionServiceClient{},
	}
}

func monitoredRegionRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "resourceanalytics",
		FormalSlug:        "monitoredregion",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(resourceanalyticssdk.MonitoredRegionLifecycleStateCreating)},
			UpdatingStates:     []string{string(resourceanalyticssdk.MonitoredRegionLifecycleStateUpdating)},
			ActiveStates:       []string{string(resourceanalyticssdk.MonitoredRegionLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(resourceanalyticssdk.MonitoredRegionLifecycleStateDeleting)},
			TerminalStates: []string{string(resourceanalyticssdk.MonitoredRegionLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"resourceAnalyticsInstanceId", "regionId", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			ForceNew:      []string{"resourceAnalyticsInstanceId", "regionId"},
			ConflictsWith: map[string][]string{},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
		},
	}
}

func monitoredRegionCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateMonitoredRegionDetails", RequestName: "CreateMonitoredRegionDetails", Contribution: "body"},
	}
}

func monitoredRegionGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "MonitoredRegionId", RequestName: "monitoredRegionId", Contribution: "path", PreferResourceID: true},
	}
}

func monitoredRegionListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "ResourceAnalyticsInstanceId",
			RequestName:  "resourceAnalyticsInstanceId",
			Contribution: "query",
			LookupPaths:  []string{"spec.resourceAnalyticsInstanceId", "status.resourceAnalyticsInstanceId", "resourceAnalyticsInstanceId"},
		},
		{
			FieldName:        "Id",
			RequestName:      "id",
			Contribution:     "query",
			PreferResourceID: true,
			LookupPaths:      []string{"status.id", "status.status.ocid", "id", "ocid"},
		},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
	}
}

func monitoredRegionDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "MonitoredRegionId", RequestName: "monitoredRegionId", Contribution: "path", PreferResourceID: true},
	}
}

func buildMonitoredRegionCreateBody(
	_ context.Context,
	resource *resourceanalyticsv1beta1.MonitoredRegion,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("monitoredregion resource is nil")
	}
	if err := validateMonitoredRegionSpec(resource.Spec); err != nil {
		return nil, err
	}

	return resourceanalyticssdk.CreateMonitoredRegionDetails{
		ResourceAnalyticsInstanceId: common.String(strings.TrimSpace(resource.Spec.ResourceAnalyticsInstanceId)),
		RegionId:                    common.String(strings.TrimSpace(resource.Spec.RegionId)),
	}, nil
}

func validateMonitoredRegionSpec(spec resourceanalyticsv1beta1.MonitoredRegionSpec) error {
	var missing []string
	if strings.TrimSpace(spec.ResourceAnalyticsInstanceId) == "" {
		missing = append(missing, "resourceAnalyticsInstanceId")
	}
	if strings.TrimSpace(spec.RegionId) == "" {
		missing = append(missing, "regionId")
	}
	if len(missing) != 0 {
		return fmt.Errorf("monitoredregion spec is missing required field(s): %s", strings.Join(missing, ", "))
	}
	return nil
}

func guardMonitoredRegionExistingBeforeCreate(
	_ context.Context,
	resource *resourceanalyticsv1beta1.MonitoredRegion,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("monitoredregion resource is nil")
	}
	if err := validateMonitoredRegionSpec(resource.Spec); err != nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, err
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func validateMonitoredRegionCreateOnlyDrift(
	resource *resourceanalyticsv1beta1.MonitoredRegion,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("monitoredregion resource is nil")
	}
	current, err := monitoredRegionRuntimeBody(currentResponse)
	if err != nil {
		return err
	}

	var drift []string
	if monitoredRegionStringDrift(resource.Spec.ResourceAnalyticsInstanceId, current.ResourceAnalyticsInstanceId) {
		drift = append(drift, "resourceAnalyticsInstanceId")
	}
	if monitoredRegionStringDrift(resource.Spec.RegionId, current.RegionId) {
		drift = append(drift, "regionId")
	}
	if len(drift) == 0 {
		return nil
	}
	return fmt.Errorf("monitoredregion create-only drift detected for %s; replace the resource or restore the desired spec before reconcile", strings.Join(drift, ", "))
}

func monitoredRegionRuntimeBody(currentResponse any) (resourceanalyticssdk.MonitoredRegion, error) {
	if current, ok, err := monitoredRegionDirectRuntimeBody(currentResponse); ok || err != nil {
		return current, err
	}
	if current, ok, err := monitoredRegionResponseRuntimeBody(currentResponse); ok || err != nil {
		return current, err
	}
	return resourceanalyticssdk.MonitoredRegion{}, fmt.Errorf("unexpected current MonitoredRegion response type %T", currentResponse)
}

func monitoredRegionDirectRuntimeBody(currentResponse any) (resourceanalyticssdk.MonitoredRegion, bool, error) {
	switch current := currentResponse.(type) {
	case resourceanalyticssdk.MonitoredRegion:
		return current, true, nil
	case *resourceanalyticssdk.MonitoredRegion:
		body, err := dereferenceMonitoredRegionRuntimeBody(current)
		return body, true, err
	case resourceanalyticssdk.MonitoredRegionSummary:
		return monitoredRegionFromSummary(current), true, nil
	case *resourceanalyticssdk.MonitoredRegionSummary:
		summary, err := dereferenceMonitoredRegionRuntimeBody(current)
		if err != nil {
			return resourceanalyticssdk.MonitoredRegion{}, true, err
		}
		return monitoredRegionFromSummary(summary), true, nil
	default:
		return resourceanalyticssdk.MonitoredRegion{}, false, nil
	}
}

func monitoredRegionResponseRuntimeBody(currentResponse any) (resourceanalyticssdk.MonitoredRegion, bool, error) {
	switch current := currentResponse.(type) {
	case resourceanalyticssdk.CreateMonitoredRegionResponse:
		return current.MonitoredRegion, true, nil
	case *resourceanalyticssdk.CreateMonitoredRegionResponse:
		response, err := dereferenceMonitoredRegionRuntimeBody(current)
		if err != nil {
			return resourceanalyticssdk.MonitoredRegion{}, true, err
		}
		return response.MonitoredRegion, true, nil
	case resourceanalyticssdk.GetMonitoredRegionResponse:
		return current.MonitoredRegion, true, nil
	case *resourceanalyticssdk.GetMonitoredRegionResponse:
		response, err := dereferenceMonitoredRegionRuntimeBody(current)
		if err != nil {
			return resourceanalyticssdk.MonitoredRegion{}, true, err
		}
		return response.MonitoredRegion, true, nil
	default:
		return resourceanalyticssdk.MonitoredRegion{}, false, nil
	}
}

func dereferenceMonitoredRegionRuntimeBody[T any](current *T) (T, error) {
	if current == nil {
		var zero T
		return zero, fmt.Errorf("current monitoredregion response is nil")
	}
	return *current, nil
}

func monitoredRegionFromSummary(summary resourceanalyticssdk.MonitoredRegionSummary) resourceanalyticssdk.MonitoredRegion {
	return resourceanalyticssdk.MonitoredRegion(summary)
}

func monitoredRegionStringDrift(spec string, current *string) bool {
	desired := strings.TrimSpace(spec)
	observed := strings.TrimSpace(monitoredRegionStringValue(current))
	return desired != "" && observed != "" && desired != observed
}

func listMonitoredRegionsAllPages(
	call func(context.Context, resourceanalyticssdk.ListMonitoredRegionsRequest) (resourceanalyticssdk.ListMonitoredRegionsResponse, error),
) func(context.Context, resourceanalyticssdk.ListMonitoredRegionsRequest) (resourceanalyticssdk.ListMonitoredRegionsResponse, error) {
	return func(ctx context.Context, request resourceanalyticssdk.ListMonitoredRegionsRequest) (resourceanalyticssdk.ListMonitoredRegionsResponse, error) {
		var combined resourceanalyticssdk.ListMonitoredRegionsResponse
		seenPages := map[string]struct{}{}

		for {
			response, err := call(ctx, request)
			if err != nil {
				return resourceanalyticssdk.ListMonitoredRegionsResponse{}, err
			}
			combined.RawResponse = response.RawResponse
			if combined.OpcRequestId == nil {
				combined.OpcRequestId = response.OpcRequestId
			}
			combined.Items = append(combined.Items, response.Items...)

			nextPage := strings.TrimSpace(monitoredRegionStringValue(response.OpcNextPage))
			if nextPage == "" {
				combined.OpcNextPage = nil
				return combined, nil
			}
			if _, ok := seenPages[nextPage]; ok {
				return resourceanalyticssdk.ListMonitoredRegionsResponse{}, fmt.Errorf("monitoredregion list pagination repeated page token %q", nextPage)
			}
			seenPages[nextPage] = struct{}{}
			request.Page = response.OpcNextPage
			combined.OpcNextPage = response.OpcNextPage
		}
	}
}

func handleMonitoredRegionDeleteError(resource *resourceanalyticsv1beta1.MonitoredRegion, err error) error {
	if err == nil {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if !isAmbiguousMonitoredRegionNotFound(err) {
		return err
	}
	return ambiguousMonitoredRegionNotFound("delete path", err)
}

func wrapMonitoredRegionDeleteConfirmation(hooks *MonitoredRegionRuntimeHooks) {
	if hooks == nil || hooks.Get.Call == nil {
		return
	}
	getMonitoredRegion := hooks.Get.Call
	listMonitoredRegions := hooks.List.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate MonitoredRegionServiceClient) MonitoredRegionServiceClient {
		return monitoredRegionDeleteConfirmationClient{
			delegate:             delegate,
			getMonitoredRegion:   getMonitoredRegion,
			listMonitoredRegions: listMonitoredRegions,
		}
	})
}

type monitoredRegionDeleteConfirmationClient struct {
	delegate             MonitoredRegionServiceClient
	getMonitoredRegion   func(context.Context, resourceanalyticssdk.GetMonitoredRegionRequest) (resourceanalyticssdk.GetMonitoredRegionResponse, error)
	listMonitoredRegions func(context.Context, resourceanalyticssdk.ListMonitoredRegionsRequest) (resourceanalyticssdk.ListMonitoredRegionsResponse, error)
}

func (c monitoredRegionDeleteConfirmationClient) CreateOrUpdate(
	ctx context.Context,
	resource *resourceanalyticsv1beta1.MonitoredRegion,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c monitoredRegionDeleteConfirmationClient) Delete(
	ctx context.Context,
	resource *resourceanalyticsv1beta1.MonitoredRegion,
) (bool, error) {
	if resource == nil {
		return false, fmt.Errorf("monitoredregion resource is nil")
	}
	if err := validateMonitoredRegionSpec(resource.Spec); err != nil {
		markMonitoredRegionFailed(resource, err)
		return false, err
	}
	if deleted, err, handled := c.deleteUntrackedByIdentity(ctx, resource); handled {
		return deleted, err
	}
	if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c monitoredRegionDeleteConfirmationClient) deleteUntrackedByIdentity(
	ctx context.Context,
	resource *resourceanalyticsv1beta1.MonitoredRegion,
) (bool, error, bool) {
	if trackedMonitoredRegionID(resource) != "" || c.listMonitoredRegions == nil {
		return false, nil, false
	}

	response, err := c.listMonitoredRegions(ctx, resourceanalyticssdk.ListMonitoredRegionsRequest{
		ResourceAnalyticsInstanceId: common.String(strings.TrimSpace(resource.Spec.ResourceAnalyticsInstanceId)),
	})
	if err != nil {
		if isAmbiguousMonitoredRegionNotFound(err) {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			return false, ambiguousMonitoredRegionNotFound("delete confirmation", err), true
		}
		return false, err, true
	}

	matches := monitoredRegionListMatches(resource, response.Items)
	switch len(matches) {
	case 0:
		markMonitoredRegionDeleted(resource, "MonitoredRegion delete confirmation did not find a matching OCI monitored region")
		return true, nil, true
	case 1:
		monitoredRegionID := monitoredRegionStringValue(matches[0].Id)
		if monitoredRegionID == "" {
			return false, fmt.Errorf("monitoredregion delete confirmation resolved a match without an OCID"), true
		}
		if err := c.rejectAuthShapedConfirmReadByID(ctx, resource, monitoredRegionID); err != nil {
			return false, err, true
		}
		recordTrackedMonitoredRegionID(resource, monitoredRegionID)
		return false, nil, false
	default:
		return false, fmt.Errorf(
			"monitoredregion list response returned multiple matching resources for resourceAnalyticsInstanceId %q and regionId %q",
			resource.Spec.ResourceAnalyticsInstanceId,
			resource.Spec.RegionId,
		), true
	}
}

func (c monitoredRegionDeleteConfirmationClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *resourceanalyticsv1beta1.MonitoredRegion,
) error {
	if c.getMonitoredRegion == nil || resource == nil {
		return nil
	}
	monitoredRegionID := trackedMonitoredRegionID(resource)
	if monitoredRegionID == "" {
		return nil
	}
	return c.rejectAuthShapedConfirmReadByID(ctx, resource, monitoredRegionID)
}

func (c monitoredRegionDeleteConfirmationClient) rejectAuthShapedConfirmReadByID(
	ctx context.Context,
	resource *resourceanalyticsv1beta1.MonitoredRegion,
	monitoredRegionID string,
) error {
	_, err := c.getMonitoredRegion(ctx, resourceanalyticssdk.GetMonitoredRegionRequest{
		MonitoredRegionId: common.String(monitoredRegionID),
	})
	if err == nil {
		return nil
	}
	if !isAmbiguousMonitoredRegionNotFound(err) {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return ambiguousMonitoredRegionNotFound("delete confirmation", err)
}

func monitoredRegionListMatches(
	resource *resourceanalyticsv1beta1.MonitoredRegion,
	items []resourceanalyticssdk.MonitoredRegionSummary,
) []resourceanalyticssdk.MonitoredRegionSummary {
	if resource == nil {
		return nil
	}
	matches := make([]resourceanalyticssdk.MonitoredRegionSummary, 0, len(items))
	for _, item := range items {
		if monitoredRegionSummaryMatchesSpec(item, resource.Spec) {
			matches = append(matches, item)
		}
	}
	return matches
}

func monitoredRegionSummaryMatchesSpec(
	summary resourceanalyticssdk.MonitoredRegionSummary,
	spec resourceanalyticsv1beta1.MonitoredRegionSpec,
) bool {
	if strings.TrimSpace(spec.ResourceAnalyticsInstanceId) != "" &&
		monitoredRegionStringValue(summary.ResourceAnalyticsInstanceId) != strings.TrimSpace(spec.ResourceAnalyticsInstanceId) {
		return false
	}
	if strings.TrimSpace(spec.RegionId) != "" && monitoredRegionStringValue(summary.RegionId) != strings.TrimSpace(spec.RegionId) {
		return false
	}
	return true
}

func recordTrackedMonitoredRegionID(resource *resourceanalyticsv1beta1.MonitoredRegion, monitoredRegionID string) {
	if resource == nil || strings.TrimSpace(monitoredRegionID) == "" {
		return
	}
	resource.Status.Id = strings.TrimSpace(monitoredRegionID)
	resource.Status.OsokStatus.Ocid = shared.OCID(strings.TrimSpace(monitoredRegionID))
}

func trackedMonitoredRegionID(resource *resourceanalyticsv1beta1.MonitoredRegion) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func markMonitoredRegionFailed(resource *resourceanalyticsv1beta1.MonitoredRegion, err error) {
	if resource == nil || err == nil {
		return
	}

	status := &resource.Status.OsokStatus
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	now := metav1.Now()
	status.UpdatedAt = &now
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(*status, shared.Failed, corev1.ConditionFalse, "", err.Error(), loggerutil.OSOKLogger{})
}

func markMonitoredRegionDeleted(resource *resourceanalyticsv1beta1.MonitoredRegion, message string) {
	if resource == nil {
		return
	}

	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	status.Async.Current = nil
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", message, loggerutil.OSOKLogger{})
}

func isAmbiguousMonitoredRegionNotFound(err error) bool {
	if err == nil {
		return false
	}
	var ambiguous ambiguousMonitoredRegionNotFoundError
	if errors.As(err, &ambiguous) {
		return true
	}
	return errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound()
}

func ambiguousMonitoredRegionNotFound(operation string, err error) ambiguousMonitoredRegionNotFoundError {
	var ambiguous ambiguousMonitoredRegionNotFoundError
	if errors.As(err, &ambiguous) {
		return ambiguous
	}

	message := fmt.Sprintf("monitoredregion %s returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %s", strings.TrimSpace(operation), err.Error())
	return ambiguousMonitoredRegionNotFoundError{
		message:      message,
		opcRequestID: errorutil.OpcRequestID(err),
	}
}

func monitoredRegionStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

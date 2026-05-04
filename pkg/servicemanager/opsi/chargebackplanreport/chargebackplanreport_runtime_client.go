/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package chargebackplanreport

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	opsisdk "github.com/oracle/oci-go-sdk/v65/opsi"
	opsiv1beta1 "github.com/oracle/oci-service-operator/api/opsi/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	chargebackPlanReportResourceIDAnnotation   = "opsi.oracle.com/resource-id"
	chargebackPlanReportResourceTypeAnnotation = "opsi.oracle.com/resource-type"

	chargebackPlanReportWorkRequestEntityType = "chargebackPlanReport"
)

type chargebackPlanReportRuntimeOCIClient interface {
	CreateChargebackPlanReport(context.Context, opsisdk.CreateChargebackPlanReportRequest) (opsisdk.CreateChargebackPlanReportResponse, error)
	GetChargebackPlanReport(context.Context, opsisdk.GetChargebackPlanReportRequest) (opsisdk.GetChargebackPlanReportResponse, error)
	ListChargebackPlanReports(context.Context, opsisdk.ListChargebackPlanReportsRequest) (opsisdk.ListChargebackPlanReportsResponse, error)
	UpdateChargebackPlanReport(context.Context, opsisdk.UpdateChargebackPlanReportRequest) (opsisdk.UpdateChargebackPlanReportResponse, error)
	DeleteChargebackPlanReport(context.Context, opsisdk.DeleteChargebackPlanReportRequest) (opsisdk.DeleteChargebackPlanReportResponse, error)
	GetWorkRequest(context.Context, opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error)
}

type chargebackPlanReportIdentity struct {
	resourceID   string
	resourceType string
}

type chargebackPlanReportAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

type chargebackPlanReportRuntimeHookDeps struct {
	client  chargebackPlanReportRuntimeOCIClient
	initErr error
}

func (e chargebackPlanReportAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e chargebackPlanReportAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

var chargebackPlanReportWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(opsisdk.OperationStatusAccepted),
		string(opsisdk.OperationStatusInProgress),
		string(opsisdk.OperationStatusWaiting),
		string(opsisdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(opsisdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(opsisdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(opsisdk.OperationStatusCanceled)},
	CreateActionTokens:    []string{string(opsisdk.ActionTypeCreated), string(opsisdk.OperationTypeCreateChargeBack)},
	UpdateActionTokens:    []string{string(opsisdk.ActionTypeUpdated), string(opsisdk.OperationTypeUpdateChargeBack)},
	DeleteActionTokens:    []string{string(opsisdk.ActionTypeDeleted), string(opsisdk.OperationTypeDeleteChargeBack)},
}

func init() {
	registerChargebackPlanReportRuntimeHooksMutator(func(manager *ChargebackPlanReportServiceManager, hooks *ChargebackPlanReportRuntimeHooks) {
		client, initErr := newChargebackPlanReportRuntimeOCIClient(manager)
		configureChargebackPlanReportRuntimeHooks(hooks, client, initErr, manager.Log)
	})
}

func newChargebackPlanReportRuntimeOCIClient(manager *ChargebackPlanReportServiceManager) (chargebackPlanReportRuntimeOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("chargeback plan report service manager is nil")
	}
	return opsisdk.NewOperationsInsightsClientWithConfigurationProvider(manager.Provider)
}

func configureChargebackPlanReportRuntimeHooks(
	hooks *ChargebackPlanReportRuntimeHooks,
	client chargebackPlanReportRuntimeOCIClient,
	initErr error,
	log loggerutil.OSOKLogger,
) {
	if hooks == nil {
		return
	}
	if client == nil && initErr == nil {
		initErr = fmt.Errorf("chargeback plan report OCI client is nil")
	}

	hooks.Semantics = chargebackPlanReportRuntimeSemantics()
	hooks.BuildCreateBody = buildChargebackPlanReportCreateBody
	hooks.BuildUpdateBody = buildChargebackPlanReportUpdateBodyHook

	deps := chargebackPlanReportRuntimeHookDeps{client: client, initErr: initErr}
	configureChargebackPlanReportOperationHooks(hooks, deps)
	configureChargebackPlanReportIdentityHooks(hooks, deps)
	hooks.Identity.Resolve = resolveChargebackPlanReportIdentity
	hooks.Identity.RecordPath = recordChargebackPlanReportIdentityPath
	hooks.Identity.RecordTracked = recordTrackedChargebackPlanReportIdentity
	hooks.Identity.SeedSyntheticTrackedID = seedChargebackPlanReportTrackedID
	hooks.StatusHooks.ProjectStatus = projectChargebackPlanReportStatus
	hooks.DeleteHooks.HandleError = handleChargebackPlanReportDeleteError
	configureChargebackPlanReportAsyncHooks(hooks, deps)
	configureChargebackPlanReportClientWrappers(hooks, log)
}

func buildChargebackPlanReportUpdateBodyHook(
	_ context.Context,
	resource *opsiv1beta1.ChargebackPlanReport,
	_ string,
	currentResponse any,
) (any, bool, error) {
	return buildChargebackPlanReportUpdateBody(resource, currentResponse)
}

func configureChargebackPlanReportOperationHooks(hooks *ChargebackPlanReportRuntimeHooks, deps chargebackPlanReportRuntimeHookDeps) {
	hooks.Create.Fields = chargebackPlanReportRequestFields(hooks.Create.Fields)
	hooks.Create.Call = deps.create
	hooks.Get.Fields = chargebackPlanReportRequestFields(hooks.Get.Fields)
	hooks.Get.Call = deps.get
	hooks.List.Fields = chargebackPlanReportRequestFields(hooks.List.Fields)
	hooks.List.Call = deps.list
	hooks.Update.Fields = chargebackPlanReportRequestFields(hooks.Update.Fields)
	hooks.Update.Call = deps.update
	hooks.Delete.Fields = chargebackPlanReportRequestFields(hooks.Delete.Fields)
	hooks.Delete.Call = deps.delete
}

func configureChargebackPlanReportIdentityHooks(hooks *ChargebackPlanReportRuntimeHooks, deps chargebackPlanReportRuntimeHookDeps) {
	hooks.Identity.LookupExisting = deps.lookupExisting
}

func configureChargebackPlanReportAsyncHooks(hooks *ChargebackPlanReportRuntimeHooks, deps chargebackPlanReportRuntimeHookDeps) {
	hooks.Async.Adapter = chargebackPlanReportWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = deps.getWorkRequest
	hooks.Async.ResolveAction = resolveChargebackPlanReportWorkRequestAction
	hooks.Async.RecoverResourceID = recoverChargebackPlanReportIDFromWorkRequest
}

func configureChargebackPlanReportClientWrappers(hooks *ChargebackPlanReportRuntimeHooks, log loggerutil.OSOKLogger) {
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ChargebackPlanReportServiceClient) ChargebackPlanReportServiceClient {
		return chargebackPlanReportDeleteSafetyClient{
			delegate:       delegate,
			getWorkRequest: hooks.Async.GetWorkRequest,
			log:            log,
		}
	})
}

func (d chargebackPlanReportRuntimeHookDeps) create(ctx context.Context, request opsisdk.CreateChargebackPlanReportRequest) (opsisdk.CreateChargebackPlanReportResponse, error) {
	if d.initErr != nil {
		return opsisdk.CreateChargebackPlanReportResponse{}, d.initErr
	}
	return d.client.CreateChargebackPlanReport(ctx, request)
}

func (d chargebackPlanReportRuntimeHookDeps) get(ctx context.Context, request opsisdk.GetChargebackPlanReportRequest) (opsisdk.GetChargebackPlanReportResponse, error) {
	if d.initErr != nil {
		return opsisdk.GetChargebackPlanReportResponse{}, d.initErr
	}
	response, err := d.client.GetChargebackPlanReport(ctx, request)
	return response, normalizeChargebackPlanReportReadError(err)
}

func (d chargebackPlanReportRuntimeHookDeps) list(ctx context.Context, request opsisdk.ListChargebackPlanReportsRequest) (opsisdk.ListChargebackPlanReportsResponse, error) {
	if d.initErr != nil {
		return opsisdk.ListChargebackPlanReportsResponse{}, d.initErr
	}
	response, err := listChargebackPlanReportsAllPages(ctx, d.client.ListChargebackPlanReports, request)
	return response, normalizeChargebackPlanReportReadError(err)
}

func (d chargebackPlanReportRuntimeHookDeps) update(ctx context.Context, request opsisdk.UpdateChargebackPlanReportRequest) (opsisdk.UpdateChargebackPlanReportResponse, error) {
	if d.initErr != nil {
		return opsisdk.UpdateChargebackPlanReportResponse{}, d.initErr
	}
	return d.client.UpdateChargebackPlanReport(ctx, request)
}

func (d chargebackPlanReportRuntimeHookDeps) delete(ctx context.Context, request opsisdk.DeleteChargebackPlanReportRequest) (opsisdk.DeleteChargebackPlanReportResponse, error) {
	if d.initErr != nil {
		return opsisdk.DeleteChargebackPlanReportResponse{}, d.initErr
	}
	return d.client.DeleteChargebackPlanReport(ctx, request)
}

func (d chargebackPlanReportRuntimeHookDeps) lookupExisting(
	ctx context.Context,
	resource *opsiv1beta1.ChargebackPlanReport,
	identity any,
) (any, error) {
	if d.initErr != nil {
		return nil, d.initErr
	}
	return lookupExistingChargebackPlanReport(ctx, d.client, resource, identity)
}

func (d chargebackPlanReportRuntimeHookDeps) getWorkRequest(ctx context.Context, workRequestID string) (any, error) {
	if d.initErr != nil {
		return nil, d.initErr
	}
	response, err := d.client.GetWorkRequest(ctx, opsisdk.GetWorkRequestRequest{WorkRequestId: common.String(strings.TrimSpace(workRequestID))})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func chargebackPlanReportRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "opsi",
		FormalSlug:        "chargebackplanreport",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "workrequest",
			Runtime:              "generatedruntime",
			FormalClassification: "workrequest",
			WorkRequest: &generatedruntime.WorkRequestSemantics{
				Source: "service-sdk",
				Phases: []string{"create", "update", "delete"},
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{"DELETING"},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"reportName"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{"reportName", "reportProperties"},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func chargebackPlanReportRequestFields(fields []generatedruntime.RequestField) []generatedruntime.RequestField {
	updated := append([]generatedruntime.RequestField(nil), fields...)
	for index := range updated {
		switch updated[index].FieldName {
		case "Id":
			updated[index].LookupPaths = []string{"resourceId"}
		case "ResourceType":
			updated[index].LookupPaths = []string{"resourceType"}
		}
	}
	return updated
}

func resolveChargebackPlanReportIdentity(resource *opsiv1beta1.ChargebackPlanReport) (any, error) {
	identity, err := operationalChargebackPlanReportIdentity(resource)
	if err != nil {
		return nil, err
	}
	return identity, nil
}

func operationalChargebackPlanReportIdentity(resource *opsiv1beta1.ChargebackPlanReport) (chargebackPlanReportIdentity, error) {
	if resource == nil {
		return chargebackPlanReportIdentity{}, fmt.Errorf("chargeback plan report resource is nil")
	}
	if hasTrackedChargebackPlanReport(resource) && strings.TrimSpace(resource.Status.ResourceId) != "" && strings.TrimSpace(resource.Status.ResourceType) != "" {
		return chargebackPlanReportIdentity{
			resourceID:   strings.TrimSpace(resource.Status.ResourceId),
			resourceType: strings.TrimSpace(resource.Status.ResourceType),
		}, nil
	}

	identity, found, err := chargebackPlanReportAnnotationIdentity(resource)
	if err != nil {
		return chargebackPlanReportIdentity{}, err
	}
	if found {
		return identity, nil
	}

	identity = chargebackPlanReportIdentity{
		resourceID:   strings.TrimSpace(resource.Status.ResourceId),
		resourceType: strings.TrimSpace(resource.Status.ResourceType),
	}
	if identity.resourceID == "" {
		return chargebackPlanReportIdentity{}, fmt.Errorf("chargeback plan report requires %q annotation before OCI create or bind", chargebackPlanReportResourceIDAnnotation)
	}
	if identity.resourceType == "" {
		return chargebackPlanReportIdentity{}, fmt.Errorf("chargeback plan report requires %q annotation before OCI create or bind", chargebackPlanReportResourceTypeAnnotation)
	}
	return normalizeChargebackPlanReportIdentity(identity)
}

func chargebackPlanReportAnnotationIdentity(resource *opsiv1beta1.ChargebackPlanReport) (chargebackPlanReportIdentity, bool, error) {
	annotations := resource.GetAnnotations()
	resourceID := strings.TrimSpace(annotations[chargebackPlanReportResourceIDAnnotation])
	resourceType := strings.TrimSpace(annotations[chargebackPlanReportResourceTypeAnnotation])
	if resourceID == "" && resourceType == "" {
		return chargebackPlanReportIdentity{}, false, nil
	}
	if resourceID == "" {
		return chargebackPlanReportIdentity{}, true, fmt.Errorf("chargeback plan report annotation %q is required when %q is set", chargebackPlanReportResourceIDAnnotation, chargebackPlanReportResourceTypeAnnotation)
	}
	if resourceType == "" {
		return chargebackPlanReportIdentity{}, true, fmt.Errorf("chargeback plan report annotation %q is required when %q is set", chargebackPlanReportResourceTypeAnnotation, chargebackPlanReportResourceIDAnnotation)
	}
	identity, err := normalizeChargebackPlanReportIdentity(chargebackPlanReportIdentity{
		resourceID:   resourceID,
		resourceType: resourceType,
	})
	return identity, true, err
}

func normalizeChargebackPlanReportIdentity(identity chargebackPlanReportIdentity) (chargebackPlanReportIdentity, error) {
	identity.resourceID = strings.TrimSpace(identity.resourceID)
	identity.resourceType = strings.TrimSpace(identity.resourceType)
	if identity.resourceID == "" {
		return chargebackPlanReportIdentity{}, fmt.Errorf("chargeback plan report source resourceId is required")
	}
	resourceType, ok := opsisdk.GetMappingChargebackPlanReportResourceTypeEnum(identity.resourceType)
	if !ok || resourceType == "" {
		return chargebackPlanReportIdentity{}, fmt.Errorf("chargeback plan report resourceType %q is unsupported; supported values are %s", identity.resourceType, strings.Join(opsisdk.GetChargebackPlanReportResourceTypeEnumStringValues(), ", "))
	}
	identity.resourceType = string(resourceType)
	return identity, nil
}

func recordChargebackPlanReportIdentityPath(resource *opsiv1beta1.ChargebackPlanReport, identity any) {
	if resource == nil {
		return
	}
	if reportID := strings.TrimSpace(resource.Status.ReportId); reportID != "" && resource.Status.OsokStatus.Ocid == "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(reportID)
	}

	typed, ok := identity.(chargebackPlanReportIdentity)
	if !ok {
		return
	}
	if strings.TrimSpace(resource.Status.ResourceId) == "" {
		resource.Status.ResourceId = typed.resourceID
	}
	if strings.TrimSpace(resource.Status.ResourceType) == "" {
		resource.Status.ResourceType = typed.resourceType
	}
}

func recordTrackedChargebackPlanReportIdentity(resource *opsiv1beta1.ChargebackPlanReport, identity any, resourceID string) {
	if resource == nil {
		return
	}
	reportID := strings.TrimSpace(resourceID)
	if reportID == "" {
		reportID = strings.TrimSpace(resource.Status.ReportId)
	}
	if reportID != "" {
		resource.Status.ReportId = reportID
		resource.Status.OsokStatus.Ocid = shared.OCID(reportID)
	}

	if typed, ok := identity.(chargebackPlanReportIdentity); ok {
		resource.Status.ResourceId = typed.resourceID
		resource.Status.ResourceType = typed.resourceType
	}
}

func hasTrackedChargebackPlanReport(resource *opsiv1beta1.ChargebackPlanReport) bool {
	if resource == nil {
		return false
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)) != "" || strings.TrimSpace(resource.Status.ReportId) != ""
}

func lookupExistingChargebackPlanReport(
	ctx context.Context,
	client chargebackPlanReportRuntimeOCIClient,
	resource *opsiv1beta1.ChargebackPlanReport,
	identity any,
) (any, error) {
	if client == nil {
		return nil, fmt.Errorf("chargeback plan report OCI client is nil")
	}
	typed, reportName, err := chargebackPlanReportLookupTarget(resource, identity)
	if err != nil || reportName == "" {
		return nil, err
	}

	response, err := listChargebackPlanReportsForIdentity(ctx, client, typed)
	if err != nil {
		return nil, err
	}
	return bindMatchingChargebackPlanReport(resource, typed, matchingChargebackPlanReportSummaries(response.Items, reportName, typed))
}

func chargebackPlanReportLookupTarget(
	resource *opsiv1beta1.ChargebackPlanReport,
	identity any,
) (chargebackPlanReportIdentity, string, error) {
	if resource == nil {
		return chargebackPlanReportIdentity{}, "", fmt.Errorf("chargeback plan report resource is nil")
	}
	typed, ok := identity.(chargebackPlanReportIdentity)
	if !ok {
		return chargebackPlanReportIdentity{}, "", fmt.Errorf("chargeback plan report identity has unexpected type %T", identity)
	}
	return typed, strings.TrimSpace(resource.Spec.ReportName), nil
}

func listChargebackPlanReportsForIdentity(
	ctx context.Context,
	client chargebackPlanReportRuntimeOCIClient,
	identity chargebackPlanReportIdentity,
) (opsisdk.ListChargebackPlanReportsResponse, error) {
	response, err := listChargebackPlanReportsAllPages(ctx, client.ListChargebackPlanReports, opsisdk.ListChargebackPlanReportsRequest{
		Id:           common.String(identity.resourceID),
		ResourceType: common.String(identity.resourceType),
	})
	if err != nil {
		return opsisdk.ListChargebackPlanReportsResponse{}, normalizeChargebackPlanReportReadError(err)
	}
	return response, nil
}

func matchingChargebackPlanReportSummaries(
	items []opsisdk.ChargebackPlanReportSummary,
	reportName string,
	identity chargebackPlanReportIdentity,
) []opsisdk.ChargebackPlanReportSummary {
	var matches []opsisdk.ChargebackPlanReportSummary
	for _, item := range items {
		if chargebackPlanReportSummaryMatches(item, reportName, identity) {
			matches = append(matches, item)
		}
	}
	return matches
}

func chargebackPlanReportSummaryMatches(
	item opsisdk.ChargebackPlanReportSummary,
	reportName string,
	identity chargebackPlanReportIdentity,
) bool {
	readback := chargebackPlanReportFromSummary(item)
	if readback.reportName != reportName {
		return false
	}
	if readback.resourceID != "" && readback.resourceID != identity.resourceID {
		return false
	}
	if readback.resourceType != "" && !strings.EqualFold(readback.resourceType, identity.resourceType) {
		return false
	}
	return true
}

func bindMatchingChargebackPlanReport(
	resource *opsiv1beta1.ChargebackPlanReport,
	identity chargebackPlanReportIdentity,
	matches []opsisdk.ChargebackPlanReportSummary,
) (any, error) {
	switch len(matches) {
	case 0:
		return nil, nil
	case 1:
		if err := projectChargebackPlanReportStatus(resource, matches[0]); err != nil {
			return nil, err
		}
		recordTrackedChargebackPlanReportIdentity(resource, identity, stringPtrValue(matches[0].ReportId))
		return matches[0], nil
	default:
		return nil, fmt.Errorf("chargeback plan report list returned multiple reports named %q for source %q", resource.Spec.ReportName, identity.resourceID)
	}
}

func seedChargebackPlanReportTrackedID(resource *opsiv1beta1.ChargebackPlanReport, _ any) func() {
	if resource == nil {
		return nil
	}
	reportID := strings.TrimSpace(resource.Status.ReportId)
	if reportID == "" {
		reportID = strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	}
	if reportID == "" {
		return nil
	}
	previousReportID := resource.Status.ReportId
	previousOcid := resource.Status.OsokStatus.Ocid
	resource.Status.ReportId = reportID
	resource.Status.OsokStatus.Ocid = shared.OCID(reportID)
	return func() {
		resource.Status.ReportId = previousReportID
		resource.Status.OsokStatus.Ocid = previousOcid
	}
}

func buildChargebackPlanReportCreateBody(
	_ context.Context,
	resource *opsiv1beta1.ChargebackPlanReport,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("chargeback plan report resource is nil")
	}
	properties, err := buildChargebackPlanReportProperties(resource.Spec.ReportProperties)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(resource.Spec.ReportName) == "" {
		return nil, fmt.Errorf("chargeback plan report spec.reportName is required")
	}
	return opsisdk.CreateChargebackPlanReportDetails{
		ReportName:       common.String(resource.Spec.ReportName),
		ReportProperties: &properties,
	}, nil
}

func buildChargebackPlanReportUpdateBody(resource *opsiv1beta1.ChargebackPlanReport, currentResponse any) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("chargeback plan report resource is nil")
	}
	properties, err := buildChargebackPlanReportProperties(resource.Spec.ReportProperties)
	if err != nil {
		return nil, false, err
	}
	if strings.TrimSpace(resource.Spec.ReportName) == "" {
		return nil, false, fmt.Errorf("chargeback plan report spec.reportName is required")
	}

	if current, ok := chargebackPlanReportFromResponse(currentResponse); ok {
		if err := validateChargebackPlanReportIdentityDrift(resource, current); err != nil {
			return nil, false, err
		}
		if chargebackPlanReportMatchesDesired(resource, properties, current) {
			return nil, false, nil
		}
	}

	return opsisdk.UpdateChargebackPlanReportDetails{
		ReportName:       common.String(resource.Spec.ReportName),
		ReportProperties: &properties,
	}, true, nil
}

func buildChargebackPlanReportProperties(spec opsiv1beta1.ChargebackPlanReportReportProperties) (opsisdk.ReportPropertyDetails, error) {
	if strings.TrimSpace(spec.AnalysisTimeInterval) == "" {
		return opsisdk.ReportPropertyDetails{}, fmt.Errorf("chargeback plan report spec.reportProperties.analysisTimeInterval is required")
	}
	start, err := parseChargebackPlanReportTime("spec.reportProperties.timeIntervalStart", spec.TimeIntervalStart)
	if err != nil {
		return opsisdk.ReportPropertyDetails{}, err
	}
	end, err := parseChargebackPlanReportTime("spec.reportProperties.timeIntervalEnd", spec.TimeIntervalEnd)
	if err != nil {
		return opsisdk.ReportPropertyDetails{}, err
	}
	groupBy, err := decodeChargebackPlanReportGroupBy(spec.GroupBy)
	if err != nil {
		return opsisdk.ReportPropertyDetails{}, err
	}

	return opsisdk.ReportPropertyDetails{
		AnalysisTimeInterval: common.String(spec.AnalysisTimeInterval),
		TimeIntervalStart:    &common.SDKTime{Time: start},
		TimeIntervalEnd:      &common.SDKTime{Time: end},
		GroupBy:              &groupBy,
	}, nil
}

func parseChargebackPlanReportTime(fieldName string, value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("chargeback plan report %s is required", fieldName)
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("chargeback plan report %s must be RFC3339: %w", fieldName, err)
	}
	return parsed, nil
}

func decodeChargebackPlanReportGroupBy(value shared.JSONValue) (any, error) {
	if len(value.Raw) == 0 {
		return nil, fmt.Errorf("chargeback plan report spec.reportProperties.groupBy is required")
	}
	var decoded any
	if err := json.Unmarshal(value.Raw, &decoded); err != nil {
		return nil, fmt.Errorf("chargeback plan report spec.reportProperties.groupBy must be valid JSON: %w", err)
	}
	if decoded == nil {
		return nil, fmt.Errorf("chargeback plan report spec.reportProperties.groupBy is required")
	}
	return decoded, nil
}

func validateChargebackPlanReportIdentityDrift(resource *opsiv1beta1.ChargebackPlanReport, current chargebackPlanReportReadback) error {
	desired, found, err := chargebackPlanReportAnnotationIdentity(resource)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	if current.resourceID != "" && current.resourceID != desired.resourceID {
		return fmt.Errorf("chargeback plan report source resourceId is create-only; current %q, desired %q", current.resourceID, desired.resourceID)
	}
	if current.resourceType != "" && !strings.EqualFold(current.resourceType, desired.resourceType) {
		return fmt.Errorf("chargeback plan report source resourceType is create-only; current %q, desired %q", current.resourceType, desired.resourceType)
	}
	return nil
}

func chargebackPlanReportMatchesDesired(
	resource *opsiv1beta1.ChargebackPlanReport,
	desiredProperties opsisdk.ReportPropertyDetails,
	current chargebackPlanReportReadback,
) bool {
	if strings.TrimSpace(current.reportName) != strings.TrimSpace(resource.Spec.ReportName) {
		return false
	}
	if current.reportProperties == nil {
		return false
	}
	return jsonValuesEqual(desiredProperties, *current.reportProperties)
}

func jsonValuesEqual(a any, b any) bool {
	var left any
	var right any
	leftPayload, err := json.Marshal(a)
	if err != nil {
		return false
	}
	rightPayload, err := json.Marshal(b)
	if err != nil {
		return false
	}
	if err := json.Unmarshal(leftPayload, &left); err != nil {
		return false
	}
	if err := json.Unmarshal(rightPayload, &right); err != nil {
		return false
	}
	return reflect.DeepEqual(left, right)
}

type chargebackPlanReportReadback struct {
	reportID          string
	reportName        string
	resourceID        string
	resourceType      string
	timeCreated       *common.SDKTime
	timeUpdated       *common.SDKTime
	reportProperties  *opsisdk.ReportPropertyDetails
	opcRequestCarrier any
}

func chargebackPlanReportFromResponse(response any) (chargebackPlanReportReadback, bool) {
	if readback, ok := chargebackPlanReportFromCreateResponse(response); ok {
		return readback, true
	}
	if readback, ok := chargebackPlanReportFromGetResponse(response); ok {
		return readback, true
	}
	if readback, ok := chargebackPlanReportFromReportResponse(response); ok {
		return readback, true
	}
	return chargebackPlanReportFromSummaryResponse(response)
}

func chargebackPlanReportFromCreateResponse(response any) (chargebackPlanReportReadback, bool) {
	switch typed := response.(type) {
	case opsisdk.CreateChargebackPlanReportResponse:
		readback := chargebackPlanReportFromSDK(typed.ChargebackPlanReport)
		readback.opcRequestCarrier = typed
		return readback, true
	case *opsisdk.CreateChargebackPlanReportResponse:
		if typed != nil {
			return chargebackPlanReportFromCreateResponse(*typed)
		}
	}
	return chargebackPlanReportReadback{}, false
}

func chargebackPlanReportFromGetResponse(response any) (chargebackPlanReportReadback, bool) {
	switch typed := response.(type) {
	case opsisdk.GetChargebackPlanReportResponse:
		readback := chargebackPlanReportFromSDK(typed.ChargebackPlanReport)
		readback.opcRequestCarrier = typed
		return readback, true
	case *opsisdk.GetChargebackPlanReportResponse:
		if typed != nil {
			return chargebackPlanReportFromGetResponse(*typed)
		}
	}
	return chargebackPlanReportReadback{}, false
}

func chargebackPlanReportFromReportResponse(response any) (chargebackPlanReportReadback, bool) {
	switch typed := response.(type) {
	case opsisdk.ChargebackPlanReport:
		return chargebackPlanReportFromSDK(typed), true
	case *opsisdk.ChargebackPlanReport:
		if typed != nil {
			return chargebackPlanReportFromSDK(*typed), true
		}
	}
	return chargebackPlanReportReadback{}, false
}

func chargebackPlanReportFromSummaryResponse(response any) (chargebackPlanReportReadback, bool) {
	switch typed := response.(type) {
	case opsisdk.ChargebackPlanReportSummary:
		return chargebackPlanReportFromSummary(typed), true
	case *opsisdk.ChargebackPlanReportSummary:
		if typed != nil {
			return chargebackPlanReportFromSummary(*typed), true
		}
	}
	return chargebackPlanReportReadback{}, false
}

func chargebackPlanReportFromSDK(report opsisdk.ChargebackPlanReport) chargebackPlanReportReadback {
	return chargebackPlanReportReadback{
		reportID:         stringPtrValue(report.ReportId),
		reportName:       stringPtrValue(report.ReportName),
		resourceID:       stringPtrValue(report.ResourceId),
		resourceType:     string(report.ResourceType),
		timeCreated:      report.TimeCreated,
		timeUpdated:      report.TimeUpdated,
		reportProperties: report.ReportProperties,
	}
}

func chargebackPlanReportFromSummary(summary opsisdk.ChargebackPlanReportSummary) chargebackPlanReportReadback {
	return chargebackPlanReportReadback{
		reportID:         stringPtrValue(summary.ReportId),
		reportName:       stringPtrValue(summary.ReportName),
		resourceID:       stringPtrValue(summary.ResourceId),
		resourceType:     string(summary.ResourceType),
		timeCreated:      summary.TimeCreated,
		timeUpdated:      summary.TimeUpdated,
		reportProperties: summary.ReportProperties,
	}
}

func projectChargebackPlanReportStatus(resource *opsiv1beta1.ChargebackPlanReport, response any) error {
	if resource == nil {
		return fmt.Errorf("chargeback plan report resource is nil")
	}
	readback, ok := chargebackPlanReportFromResponse(response)
	if !ok {
		return nil
	}
	if requestID := servicemanager.ResponseOpcRequestID(readback.opcRequestCarrier); requestID != "" {
		servicemanager.SetOpcRequestID(&resource.Status.OsokStatus, requestID)
	}
	projectChargebackPlanReportIdentityStatus(resource, readback)
	projectChargebackPlanReportTimeStatus(resource, readback)
	projectChargebackPlanReportPropertiesStatus(resource, readback)
	return nil
}

func projectChargebackPlanReportIdentityStatus(resource *opsiv1beta1.ChargebackPlanReport, readback chargebackPlanReportReadback) {
	if readback.reportID != "" {
		resource.Status.ReportId = readback.reportID
		resource.Status.OsokStatus.Ocid = shared.OCID(readback.reportID)
	}
	if readback.reportName != "" {
		resource.Status.ReportName = readback.reportName
	}
	if readback.resourceID != "" {
		resource.Status.ResourceId = readback.resourceID
	}
	if readback.resourceType != "" {
		resource.Status.ResourceType = readback.resourceType
	}
}

func projectChargebackPlanReportTimeStatus(resource *opsiv1beta1.ChargebackPlanReport, readback chargebackPlanReportReadback) {
	if readback.timeCreated != nil {
		resource.Status.TimeCreated = readback.timeCreated.Format(time.RFC3339Nano)
	}
	if readback.timeUpdated != nil {
		resource.Status.TimeUpdated = readback.timeUpdated.Format(time.RFC3339Nano)
	}
}

func projectChargebackPlanReportPropertiesStatus(resource *opsiv1beta1.ChargebackPlanReport, readback chargebackPlanReportReadback) {
	if readback.reportProperties != nil {
		resource.Status.ReportProperties = projectChargebackPlanReportProperties(readback.reportProperties)
	}
}

func projectChargebackPlanReportProperties(properties *opsisdk.ReportPropertyDetails) opsiv1beta1.ChargebackPlanReportReportProperties {
	if properties == nil {
		return opsiv1beta1.ChargebackPlanReportReportProperties{}
	}
	projected := opsiv1beta1.ChargebackPlanReportReportProperties{
		AnalysisTimeInterval: stringPtrValue(properties.AnalysisTimeInterval),
	}
	if properties.TimeIntervalStart != nil {
		projected.TimeIntervalStart = properties.TimeIntervalStart.Format(time.RFC3339Nano)
	}
	if properties.TimeIntervalEnd != nil {
		projected.TimeIntervalEnd = properties.TimeIntervalEnd.Format(time.RFC3339Nano)
	}
	if properties.GroupBy != nil {
		if payload, err := json.Marshal(properties.GroupBy); err == nil {
			projected.GroupBy = shared.JSONValue{Raw: payload}
		}
	}
	return projected
}

func listChargebackPlanReportsAllPages(
	ctx context.Context,
	call func(context.Context, opsisdk.ListChargebackPlanReportsRequest) (opsisdk.ListChargebackPlanReportsResponse, error),
	request opsisdk.ListChargebackPlanReportsRequest,
) (opsisdk.ListChargebackPlanReportsResponse, error) {
	if call == nil {
		return opsisdk.ListChargebackPlanReportsResponse{}, fmt.Errorf("chargeback plan report ListChargebackPlanReports call is not configured")
	}

	var combined opsisdk.ListChargebackPlanReportsResponse
	for {
		response, err := call(ctx, request)
		if err != nil {
			return opsisdk.ListChargebackPlanReportsResponse{}, err
		}
		if combined.RawResponse == nil {
			combined.RawResponse = response.RawResponse
		}
		if combined.OpcRequestId == nil {
			combined.OpcRequestId = response.OpcRequestId
		}
		combined.Items = append(combined.Items, response.Items...)
		nextPage := strings.TrimSpace(stringPtrValue(response.OpcNextPage))
		if nextPage == "" {
			combined.OpcNextPage = nil
			return combined, nil
		}
		request.Page = common.String(nextPage)
		combined.OpcNextPage = common.String(nextPage)
	}
}

func handleChargebackPlanReportDeleteError(resource *opsiv1beta1.ChargebackPlanReport, err error) error {
	if err == nil {
		return nil
	}
	classification := errorutil.ClassifyDeleteError(err)
	if !classification.IsAuthShapedNotFound() {
		return err
	}
	requestID := errorutil.OpcRequestID(err)
	if resource != nil {
		servicemanager.SetOpcRequestID(&resource.Status.OsokStatus, requestID)
	}
	return chargebackPlanReportAmbiguousNotFoundError{
		message:      "chargeback plan report delete returned ambiguous 404 NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed",
		opcRequestID: requestID,
	}
}

func normalizeChargebackPlanReportReadError(err error) error {
	if err == nil {
		return nil
	}
	classification := errorutil.ClassifyDeleteError(err)
	if !classification.IsAuthShapedNotFound() {
		return err
	}
	return chargebackPlanReportAmbiguousNotFoundError{
		message:      "chargeback plan report read returned ambiguous 404 NotAuthorizedOrNotFound",
		opcRequestID: errorutil.OpcRequestID(err),
	}
}

func recoverChargebackPlanReportIDFromWorkRequest(
	_ *opsiv1beta1.ChargebackPlanReport,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	workRequestValue, ok := chargebackPlanReportWorkRequestFromAny(workRequest)
	if !ok {
		return "", nil
	}
	if reportID, ok := chargebackPlanReportIDFromWorkRequestResources(workRequestValue.Resources, chargebackPlanReportWorkRequestActionForPhase(phase)); ok {
		return reportID, nil
	}
	if reportID, ok := chargebackPlanReportIDFromWorkRequestResources(workRequestValue.Resources, ""); ok {
		return reportID, nil
	}
	return "", nil
}

func resolveChargebackPlanReportWorkRequestAction(workRequest any) (string, error) {
	workRequestValue, ok := chargebackPlanReportWorkRequestFromAny(workRequest)
	if !ok {
		return "", nil
	}

	var action string
	for _, resource := range workRequestValue.Resources {
		if !isChargebackPlanReportWorkRequestResource(resource) {
			continue
		}
		candidate := strings.TrimSpace(string(resource.ActionType))
		if candidate == "" || isChargebackPlanReportIgnorableWorkRequestAction(resource.ActionType) {
			continue
		}
		if action == "" {
			action = candidate
			continue
		}
		if action != candidate {
			return "", fmt.Errorf("chargeback plan report work request %s exposes conflicting action types %q and %q", stringPtrValue(workRequestValue.Id), action, candidate)
		}
	}
	return action, nil
}

func chargebackPlanReportIDFromWorkRequestResources(resources []opsisdk.WorkRequestResource, action opsisdk.ActionTypeEnum) (string, bool) {
	for _, resource := range resources {
		if !isChargebackPlanReportWorkRequestResource(resource) {
			continue
		}
		if isChargebackPlanReportIgnorableWorkRequestAction(resource.ActionType) {
			continue
		}
		if action != "" && resource.ActionType != action {
			continue
		}
		if identifier := strings.TrimSpace(stringPtrValue(resource.Identifier)); identifier != "" {
			return identifier, true
		}
	}
	return "", false
}

func chargebackPlanReportWorkRequestFromAny(workRequest any) (opsisdk.WorkRequest, bool) {
	if typed, ok := workRequest.(opsisdk.WorkRequest); ok {
		return typed, true
	}
	if typed, ok := workRequest.(*opsisdk.WorkRequest); ok && typed != nil {
		return *typed, true
	}
	return opsisdk.WorkRequest{}, false
}

func chargebackPlanReportWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) opsisdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return opsisdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return opsisdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return opsisdk.ActionTypeDeleted
	default:
		return ""
	}
}

func isChargebackPlanReportIgnorableWorkRequestAction(action opsisdk.ActionTypeEnum) bool {
	return action == opsisdk.ActionTypeInProgress || action == opsisdk.ActionTypeRelated
}

func isChargebackPlanReportWorkRequestResource(resource opsisdk.WorkRequestResource) bool {
	entityType := strings.TrimSpace(stringPtrValue(resource.EntityType))
	if entityType == "" {
		return true
	}
	return normalizeChargebackPlanReportWorkRequestEntity(entityType) == normalizeChargebackPlanReportWorkRequestEntity(chargebackPlanReportWorkRequestEntityType)
}

func normalizeChargebackPlanReportWorkRequestEntity(value string) string {
	replacer := strings.NewReplacer("_", "", "-", "", " ", "")
	return replacer.Replace(strings.ToLower(strings.TrimSpace(value)))
}

type chargebackPlanReportDeleteSafetyClient struct {
	delegate       ChargebackPlanReportServiceClient
	getWorkRequest func(context.Context, string) (any, error)
	log            loggerutil.OSOKLogger
}

func (c chargebackPlanReportDeleteSafetyClient) CreateOrUpdate(
	ctx context.Context,
	resource *opsiv1beta1.ChargebackPlanReport,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c chargebackPlanReportDeleteSafetyClient) Delete(ctx context.Context, resource *opsiv1beta1.ChargebackPlanReport) (bool, error) {
	if deleted, handled, err := c.handleWriteWorkRequestBeforeDelete(ctx, resource); handled {
		return deleted, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c chargebackPlanReportDeleteSafetyClient) handleWriteWorkRequestBeforeDelete(
	ctx context.Context,
	resource *opsiv1beta1.ChargebackPlanReport,
) (bool, bool, error) {
	workRequestID, phase := currentChargebackPlanReportWriteWorkRequest(resource)
	if workRequestID == "" {
		return false, false, nil
	}
	if c.getWorkRequest == nil {
		return false, true, fmt.Errorf("chargeback plan report work request polling is not configured")
	}

	workRequest, err := c.pollWriteWorkRequest(ctx, resource, workRequestID)
	if err != nil {
		return false, true, err
	}
	current, err := buildChargebackPlanReportWorkRequestAsyncOperation(&resource.Status.OsokStatus, workRequest, phase)
	if err != nil {
		return false, true, err
	}
	return c.handlePolledWriteWorkRequest(ctx, resource, workRequestID, phase, workRequest, current)
}

func (c chargebackPlanReportDeleteSafetyClient) pollWriteWorkRequest(
	ctx context.Context,
	resource *opsiv1beta1.ChargebackPlanReport,
	workRequestID string,
) (any, error) {
	workRequest, err := c.getWorkRequest(ctx, workRequestID)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return nil, err
	}
	return workRequest, nil
}

func (c chargebackPlanReportDeleteSafetyClient) handlePolledWriteWorkRequest(
	ctx context.Context,
	resource *opsiv1beta1.ChargebackPlanReport,
	workRequestID string,
	phase shared.OSOKAsyncPhase,
	workRequest any,
	current *shared.OSOKAsyncOperation,
) (bool, bool, error) {
	switch current.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		return c.waitForWriteWorkRequest(resource, workRequestID, phase, current)
	case shared.OSOKAsyncClassSucceeded:
		return c.deleteAfterSucceededWriteWorkRequest(ctx, resource, phase, workRequest)
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		return c.failFinishedWriteWorkRequest(resource, workRequestID, phase, current)
	default:
		return false, true, fmt.Errorf("chargeback plan report %s work request %s projected unsupported async class %s before delete", phase, workRequestID, current.NormalizedClass)
	}
}

func (c chargebackPlanReportDeleteSafetyClient) waitForWriteWorkRequest(
	resource *opsiv1beta1.ChargebackPlanReport,
	workRequestID string,
	phase shared.OSOKAsyncPhase,
	current *shared.OSOKAsyncOperation,
) (bool, bool, error) {
	current.Message = fmt.Sprintf("chargeback plan report %s work request %s is still in progress; waiting before delete", phase, workRequestID)
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, c.log)
	return false, true, nil
}

func (c chargebackPlanReportDeleteSafetyClient) deleteAfterSucceededWriteWorkRequest(
	ctx context.Context,
	resource *opsiv1beta1.ChargebackPlanReport,
	phase shared.OSOKAsyncPhase,
	workRequest any,
) (bool, bool, error) {
	recordSucceededChargebackPlanReportWorkRequestID(resource, workRequest, phase)
	servicemanager.ClearAsyncOperation(&resource.Status.OsokStatus)
	deleted, err := c.delegate.Delete(ctx, resource)
	return deleted, true, err
}

func recordSucceededChargebackPlanReportWorkRequestID(
	resource *opsiv1beta1.ChargebackPlanReport,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) {
	reportID, err := recoverChargebackPlanReportIDFromWorkRequest(resource, workRequest, phase)
	if err != nil || reportID == "" {
		return
	}
	identity, err := operationalChargebackPlanReportIdentity(resource)
	if err != nil {
		return
	}
	recordTrackedChargebackPlanReportIdentity(resource, identity, reportID)
}

func (c chargebackPlanReportDeleteSafetyClient) failFinishedWriteWorkRequest(
	resource *opsiv1beta1.ChargebackPlanReport,
	workRequestID string,
	phase shared.OSOKAsyncPhase,
	current *shared.OSOKAsyncOperation,
) (bool, bool, error) {
	err := fmt.Errorf("chargeback plan report %s work request %s finished with status %s before delete", phase, workRequestID, current.RawStatus)
	current.Message = err.Error()
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, c.log)
	return false, true, err
}

func buildChargebackPlanReportWorkRequestAsyncOperation(
	status *shared.OSOKStatus,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (*shared.OSOKAsyncOperation, error) {
	workRequestValue, ok := chargebackPlanReportWorkRequestFromAny(workRequest)
	if !ok {
		return nil, fmt.Errorf("chargeback plan report work request has unexpected type %T", workRequest)
	}
	action, err := resolveChargebackPlanReportWorkRequestAction(workRequestValue)
	if err != nil {
		return nil, err
	}
	return servicemanager.BuildWorkRequestAsyncOperation(status, chargebackPlanReportWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(workRequestValue.Status),
		RawAction:        action,
		RawOperationType: string(workRequestValue.OperationType),
		WorkRequestID:    stringPtrValue(workRequestValue.Id),
		PercentComplete:  workRequestValue.PercentComplete,
		FallbackPhase:    phase,
	})
}

func currentChargebackPlanReportWriteWorkRequest(resource *opsiv1beta1.ChargebackPlanReport) (string, shared.OSOKAsyncPhase) {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return "", ""
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		return "", ""
	}
	switch current.Phase {
	case shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncPhaseUpdate:
		if workRequestID := strings.TrimSpace(current.WorkRequestID); workRequestID != "" {
			return workRequestID, current.Phase
		}
	}
	return "", ""
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

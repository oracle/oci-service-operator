/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package licenserecord

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	licensemanagersdk "github.com/oracle/oci-go-sdk/v65/licensemanager"
	licensemanagerv1beta1 "github.com/oracle/oci-service-operator/api/licensemanager/v1beta1"
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

const (
	LicenseRecordProductLicenseIDAnnotation = "licensemanager.oracle.com/product-license-id"

	licenseRecordLegacyProductLicenseIDAnnotation = "licensemanager.oracle.com/productLicenseId"
	licenseRecordDeletePendingMessage             = "OCI LicenseRecord delete is in progress"
)

type licenseRecordOCIClient interface {
	CreateLicenseRecord(context.Context, licensemanagersdk.CreateLicenseRecordRequest) (licensemanagersdk.CreateLicenseRecordResponse, error)
	GetLicenseRecord(context.Context, licensemanagersdk.GetLicenseRecordRequest) (licensemanagersdk.GetLicenseRecordResponse, error)
	ListLicenseRecords(context.Context, licensemanagersdk.ListLicenseRecordsRequest) (licensemanagersdk.ListLicenseRecordsResponse, error)
	UpdateLicenseRecord(context.Context, licensemanagersdk.UpdateLicenseRecordRequest) (licensemanagersdk.UpdateLicenseRecordResponse, error)
	DeleteLicenseRecord(context.Context, licensemanagersdk.DeleteLicenseRecordRequest) (licensemanagersdk.DeleteLicenseRecordResponse, error)
}

type licenseRecordIdentity struct {
	productLicenseID string
}

type licenseRecordAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e licenseRecordAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e licenseRecordAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerLicenseRecordRuntimeHooksMutator(func(_ *LicenseRecordServiceManager, hooks *LicenseRecordRuntimeHooks) {
		applyLicenseRecordRuntimeHooks(hooks)
	})
}

func applyLicenseRecordRuntimeHooks(hooks *LicenseRecordRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = licenseRecordRuntimeSemantics()
	hooks.BuildCreateBody = buildLicenseRecordCreateBody
	hooks.BuildUpdateBody = buildLicenseRecordUpdateBody
	hooks.Identity.Resolve = resolveLicenseRecordIdentity
	hooks.Identity.RecordPath = recordLicenseRecordIdentityPath
	hooks.List.Fields = licenseRecordListFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listLicenseRecordsAllPages(hooks.List.Call)
	}
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateLicenseRecordCreateOnlyDrift
	hooks.DeleteHooks.HandleError = handleLicenseRecordDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyLicenseRecordDeleteOutcome
	hooks.StatusHooks.MarkTerminating = markLicenseRecordTerminating
	wrapLicenseRecordDeleteGuard(hooks)
}

func newLicenseRecordServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client licenseRecordOCIClient,
) LicenseRecordServiceClient {
	manager := &LicenseRecordServiceManager{Log: log}
	hooks := newLicenseRecordRuntimeHooksWithOCIClient(client)
	applyLicenseRecordRuntimeHooks(&hooks)
	delegate := defaultLicenseRecordServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*licensemanagerv1beta1.LicenseRecord](
			buildLicenseRecordGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapLicenseRecordGeneratedClient(hooks, delegate)
}

func newLicenseRecordRuntimeHooksWithOCIClient(client licenseRecordOCIClient) LicenseRecordRuntimeHooks {
	hooks := newLicenseRecordDefaultRuntimeHooks(licensemanagersdk.LicenseManagerClient{})
	hooks.Create.Call = func(ctx context.Context, request licensemanagersdk.CreateLicenseRecordRequest) (licensemanagersdk.CreateLicenseRecordResponse, error) {
		if client == nil {
			return licensemanagersdk.CreateLicenseRecordResponse{}, fmt.Errorf("LicenseRecord OCI client is not configured")
		}
		return client.CreateLicenseRecord(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request licensemanagersdk.GetLicenseRecordRequest) (licensemanagersdk.GetLicenseRecordResponse, error) {
		if client == nil {
			return licensemanagersdk.GetLicenseRecordResponse{}, fmt.Errorf("LicenseRecord OCI client is not configured")
		}
		return client.GetLicenseRecord(ctx, request)
	}
	hooks.List.Call = func(ctx context.Context, request licensemanagersdk.ListLicenseRecordsRequest) (licensemanagersdk.ListLicenseRecordsResponse, error) {
		if client == nil {
			return licensemanagersdk.ListLicenseRecordsResponse{}, fmt.Errorf("LicenseRecord OCI client is not configured")
		}
		return client.ListLicenseRecords(ctx, request)
	}
	hooks.Update.Call = func(ctx context.Context, request licensemanagersdk.UpdateLicenseRecordRequest) (licensemanagersdk.UpdateLicenseRecordResponse, error) {
		if client == nil {
			return licensemanagersdk.UpdateLicenseRecordResponse{}, fmt.Errorf("LicenseRecord OCI client is not configured")
		}
		return client.UpdateLicenseRecord(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request licensemanagersdk.DeleteLicenseRecordRequest) (licensemanagersdk.DeleteLicenseRecordResponse, error) {
		if client == nil {
			return licensemanagersdk.DeleteLicenseRecordResponse{}, fmt.Errorf("LicenseRecord OCI client is not configured")
		}
		return client.DeleteLicenseRecord(ctx, request)
	}
	return hooks
}

func licenseRecordRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "licensemanager",
		FormalSlug:    "licenserecord",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ActiveStates: []string{
				string(licensemanagersdk.LifeCycleStateActive),
				string(licensemanagersdk.LifeCycleStateInactive),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{"DELETING"},
			TerminalStates: []string{string(licensemanagersdk.LifeCycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"productLicenseId", "displayName", "productId"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"displayName",
				"isPerpetual",
				"isUnlimited",
				"expirationDate",
				"supportEndDate",
				"licenseCount",
				"productId",
				"freeformTags",
				"definedTags",
			},
			ForceNew:      []string{"productLicenseId"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "LicenseRecord", Action: "CreateLicenseRecord"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "LicenseRecord", Action: "UpdateLicenseRecord"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "LicenseRecord", Action: "DeleteLicenseRecord"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "LicenseRecord", Action: "GetLicenseRecord"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "LicenseRecord", Action: "GetLicenseRecord"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "LicenseRecord", Action: "GetLicenseRecord"}},
		},
	}
}

func licenseRecordListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "ProductLicenseId",
			RequestName:  "productLicenseId",
			Contribution: "query",
			LookupPaths:  []string{"status.productLicenseId", "productLicenseId"},
		},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func resolveLicenseRecordIdentity(resource *licensemanagerv1beta1.LicenseRecord) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("LicenseRecord resource is nil")
	}
	annotatedProductLicenseID := licenseRecordProductLicenseIDAnnotation(resource)
	recordedProductLicenseID := strings.TrimSpace(resource.Status.ProductLicenseId)
	if annotatedProductLicenseID != "" && recordedProductLicenseID != "" && annotatedProductLicenseID != recordedProductLicenseID {
		return nil, fmt.Errorf("LicenseRecord create-only parent product license annotation %q changed; create a replacement resource instead", LicenseRecordProductLicenseIDAnnotation)
	}
	productLicenseID := firstNonEmptyString(annotatedProductLicenseID, recordedProductLicenseID)
	if productLicenseID == "" && trackedLicenseRecordID(resource) == "" {
		return nil, fmt.Errorf("LicenseRecord requires metadata annotation %q with the parent product license OCID because spec.productLicenseId is not available", LicenseRecordProductLicenseIDAnnotation)
	}
	return licenseRecordIdentity{productLicenseID: productLicenseID}, nil
}

func recordLicenseRecordIdentityPath(resource *licensemanagerv1beta1.LicenseRecord, identity any) {
	if resource == nil {
		return
	}
	resolved, ok := identity.(licenseRecordIdentity)
	if !ok || strings.TrimSpace(resolved.productLicenseID) == "" {
		return
	}
	resource.Status.ProductLicenseId = strings.TrimSpace(resolved.productLicenseID)
}

func buildLicenseRecordCreateBody(_ context.Context, resource *licensemanagerv1beta1.LicenseRecord, _ string) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("LicenseRecord resource is nil")
	}
	if strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return nil, fmt.Errorf("displayName is required")
	}
	body := licensemanagersdk.CreateLicenseRecordDetails{
		DisplayName: common.String(strings.TrimSpace(resource.Spec.DisplayName)),
		IsPerpetual: common.Bool(resource.Spec.IsPerpetual),
		IsUnlimited: common.Bool(resource.Spec.IsUnlimited),
	}
	if err := setLicenseRecordCreateOptionalFields(&body, resource.Spec); err != nil {
		return nil, err
	}
	return body, nil
}

func setLicenseRecordCreateOptionalFields(
	body *licensemanagersdk.CreateLicenseRecordDetails,
	spec licensemanagerv1beta1.LicenseRecordSpec,
) error {
	if value := strings.TrimSpace(spec.ExpirationDate); value != "" {
		parsed, err := licenseRecordOptionalSDKTime("expirationDate", value)
		if err != nil {
			return err
		}
		body.ExpirationDate = parsed
	}
	if value := strings.TrimSpace(spec.SupportEndDate); value != "" {
		parsed, err := licenseRecordOptionalSDKTime("supportEndDate", value)
		if err != nil {
			return err
		}
		body.SupportEndDate = parsed
	}
	if spec.LicenseCount != 0 {
		body.LicenseCount = common.Int(spec.LicenseCount)
	}
	if value := strings.TrimSpace(spec.ProductId); value != "" {
		body.ProductId = common.String(value)
	}
	if spec.FreeformTags != nil {
		body.FreeformTags = cloneLicenseRecordStringMap(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		body.DefinedTags = licenseRecordDefinedTags(spec.DefinedTags)
	}
	return nil
}

func buildLicenseRecordUpdateBody(
	_ context.Context,
	resource *licensemanagerv1beta1.LicenseRecord,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("LicenseRecord resource is nil")
	}
	current, ok := licenseRecordBodyFromResponse(currentResponse)
	if !ok {
		return nil, false, fmt.Errorf("current LicenseRecord response does not expose a LicenseRecord body")
	}

	displayName := strings.TrimSpace(resource.Spec.DisplayName)
	if displayName == "" {
		return nil, false, fmt.Errorf("displayName is required")
	}

	body := licenseRecordRequiredUpdateBody(resource.Spec, displayName)
	updateNeeded := licenseRecordRequiredFieldsNeedUpdate(current, resource.Spec, displayName)
	if err := setLicenseRecordUpdateOptionalFields(&body, &updateNeeded, resource.Spec, current); err != nil {
		return nil, false, err
	}
	return body, updateNeeded, nil
}

func licenseRecordRequiredUpdateBody(
	spec licensemanagerv1beta1.LicenseRecordSpec,
	displayName string,
) licensemanagersdk.UpdateLicenseRecordDetails {
	return licensemanagersdk.UpdateLicenseRecordDetails{
		DisplayName: common.String(displayName),
		IsPerpetual: common.Bool(spec.IsPerpetual),
		IsUnlimited: common.Bool(spec.IsUnlimited),
	}
}

func licenseRecordRequiredFieldsNeedUpdate(
	current licensemanagersdk.LicenseRecord,
	spec licensemanagerv1beta1.LicenseRecordSpec,
	displayName string,
) bool {
	return current.DisplayName == nil ||
		strings.TrimSpace(*current.DisplayName) != displayName ||
		current.IsPerpetual == nil ||
		*current.IsPerpetual != spec.IsPerpetual ||
		current.IsUnlimited == nil ||
		*current.IsUnlimited != spec.IsUnlimited
}

func setLicenseRecordUpdateOptionalFields(
	body *licensemanagersdk.UpdateLicenseRecordDetails,
	updateNeeded *bool,
	spec licensemanagerv1beta1.LicenseRecordSpec,
	current licensemanagersdk.LicenseRecord,
) error {
	changed, err := setLicenseRecordUpdateSDKTime(
		&body.ExpirationDate,
		"expirationDate",
		spec.ExpirationDate,
		current.ExpirationDate,
	)
	if err != nil {
		return err
	}
	recordLicenseRecordUpdateNeeded(updateNeeded, changed)

	changed, err = setLicenseRecordUpdateSDKTime(
		&body.SupportEndDate,
		"supportEndDate",
		spec.SupportEndDate,
		current.SupportEndDate,
	)
	if err != nil {
		return err
	}
	recordLicenseRecordUpdateNeeded(updateNeeded, changed)
	recordLicenseRecordUpdateNeeded(updateNeeded, setLicenseRecordUpdateInt(&body.LicenseCount, spec.LicenseCount, current.LicenseCount))
	recordLicenseRecordUpdateNeeded(updateNeeded, setLicenseRecordUpdateString(&body.ProductId, spec.ProductId, current.ProductId))
	recordLicenseRecordUpdateNeeded(updateNeeded, setLicenseRecordUpdateStringMap(&body.FreeformTags, spec.FreeformTags, current.FreeformTags))
	recordLicenseRecordUpdateNeeded(updateNeeded, setLicenseRecordUpdateDefinedTags(&body.DefinedTags, spec.DefinedTags, current.DefinedTags))
	return nil
}

func setLicenseRecordUpdateSDKTime(
	target **common.SDKTime,
	fieldName string,
	desiredValue string,
	current *common.SDKTime,
) (bool, error) {
	value := strings.TrimSpace(desiredValue)
	if value == "" {
		return false, nil
	}
	desired, err := licenseRecordOptionalSDKTime(fieldName, value)
	if err != nil {
		return false, err
	}
	*target = desired
	return current == nil || !current.Equal(desired.Time), nil
}

func setLicenseRecordUpdateInt(target **int, desiredValue int, current *int) bool {
	if desiredValue == 0 {
		return false
	}
	*target = common.Int(desiredValue)
	return current == nil || *current != desiredValue
}

func setLicenseRecordUpdateString(target **string, desiredValue string, current *string) bool {
	value := strings.TrimSpace(desiredValue)
	if value == "" {
		return false
	}
	*target = common.String(value)
	return current == nil || strings.TrimSpace(*current) != value
}

func setLicenseRecordUpdateStringMap(
	target *map[string]string,
	desiredValue map[string]string,
	current map[string]string,
) bool {
	if desiredValue == nil {
		return false
	}
	desired := cloneLicenseRecordStringMap(desiredValue)
	*target = desired
	return !reflect.DeepEqual(current, desired)
}

func setLicenseRecordUpdateDefinedTags(
	target *map[string]map[string]interface{},
	desiredValue map[string]shared.MapValue,
	current map[string]map[string]interface{},
) bool {
	if desiredValue == nil {
		return false
	}
	desired := licenseRecordDefinedTags(desiredValue)
	*target = desired
	return !reflect.DeepEqual(current, desired)
}

func recordLicenseRecordUpdateNeeded(updateNeeded *bool, changed bool) {
	if changed {
		*updateNeeded = true
	}
}

func validateLicenseRecordCreateOnlyDrift(resource *licensemanagerv1beta1.LicenseRecord, currentResponse any) error {
	if resource == nil {
		return fmt.Errorf("LicenseRecord resource is nil")
	}
	current, ok := licenseRecordBodyFromResponse(currentResponse)
	if !ok {
		return nil
	}
	desiredProductLicenseID := firstNonEmptyString(
		licenseRecordProductLicenseIDAnnotation(resource),
		resource.Status.ProductLicenseId,
	)
	return rejectLicenseRecordParentDrift(desiredProductLicenseID, current.ProductLicenseId)
}

func rejectLicenseRecordParentDrift(desired string, observed *string) error {
	desired = strings.TrimSpace(desired)
	current := strings.TrimSpace(stringValue(observed))
	if desired == "" || current == "" || desired == current {
		return nil
	}
	return fmt.Errorf("LicenseRecord formal semantics require replacement when productLicenseId changes")
}

func licenseRecordBodyFromResponse(response any) (licensemanagersdk.LicenseRecord, bool) {
	if body, ok := licenseRecordMutationResponseBody(response); ok {
		return body, true
	}
	return licenseRecordReadResponseBody(response)
}

func licenseRecordMutationResponseBody(response any) (licensemanagersdk.LicenseRecord, bool) {
	switch current := response.(type) {
	case licensemanagersdk.CreateLicenseRecordResponse:
		return current.LicenseRecord, true
	case *licensemanagersdk.CreateLicenseRecordResponse:
		return licenseRecordCreateResponseBody(current)
	case licensemanagersdk.UpdateLicenseRecordResponse:
		return current.LicenseRecord, true
	case *licensemanagersdk.UpdateLicenseRecordResponse:
		return licenseRecordUpdateResponseBody(current)
	default:
		return licensemanagersdk.LicenseRecord{}, false
	}
}

func licenseRecordReadResponseBody(response any) (licensemanagersdk.LicenseRecord, bool) {
	switch current := response.(type) {
	case licensemanagersdk.GetLicenseRecordResponse:
		return current.LicenseRecord, true
	case *licensemanagersdk.GetLicenseRecordResponse:
		return licenseRecordGetResponseBody(current)
	case licensemanagersdk.LicenseRecord:
		return current, true
	case *licensemanagersdk.LicenseRecord:
		return licenseRecordPointerBody(current)
	default:
		return licenseRecordSummaryResponseBody(response)
	}
}

func licenseRecordSummaryResponseBody(response any) (licensemanagersdk.LicenseRecord, bool) {
	switch current := response.(type) {
	case licensemanagersdk.LicenseRecordSummary:
		return licenseRecordFromSummary(current), true
	case *licensemanagersdk.LicenseRecordSummary:
		if current == nil {
			return licensemanagersdk.LicenseRecord{}, false
		}
		return licenseRecordFromSummary(*current), true
	default:
		return licensemanagersdk.LicenseRecord{}, false
	}
}

func licenseRecordCreateResponseBody(response *licensemanagersdk.CreateLicenseRecordResponse) (licensemanagersdk.LicenseRecord, bool) {
	if response == nil {
		return licensemanagersdk.LicenseRecord{}, false
	}
	return response.LicenseRecord, true
}

func licenseRecordGetResponseBody(response *licensemanagersdk.GetLicenseRecordResponse) (licensemanagersdk.LicenseRecord, bool) {
	if response == nil {
		return licensemanagersdk.LicenseRecord{}, false
	}
	return response.LicenseRecord, true
}

func licenseRecordUpdateResponseBody(response *licensemanagersdk.UpdateLicenseRecordResponse) (licensemanagersdk.LicenseRecord, bool) {
	if response == nil {
		return licensemanagersdk.LicenseRecord{}, false
	}
	return response.LicenseRecord, true
}

func licenseRecordPointerBody(response *licensemanagersdk.LicenseRecord) (licensemanagersdk.LicenseRecord, bool) {
	if response == nil {
		return licensemanagersdk.LicenseRecord{}, false
	}
	return *response, true
}

func licenseRecordFromSummary(summary licensemanagersdk.LicenseRecordSummary) licensemanagersdk.LicenseRecord {
	return licensemanagersdk.LicenseRecord{
		Id:               summary.Id,
		DisplayName:      summary.DisplayName,
		IsUnlimited:      summary.IsUnlimited,
		IsPerpetual:      summary.IsPerpetual,
		LifecycleState:   summary.LifecycleState,
		ProductLicenseId: summary.ProductLicenseId,
		CompartmentId:    summary.CompartmentId,
		ProductId:        summary.ProductId,
		LicenseCount:     summary.LicenseCount,
		ExpirationDate:   summary.ExpirationDate,
		SupportEndDate:   summary.SupportEndDate,
		TimeCreated:      summary.TimeCreated,
		TimeUpdated:      summary.TimeUpdated,
		LicenseUnit:      summary.LicenseUnit,
		ProductLicense:   summary.ProductLicense,
		FreeformTags:     summary.FreeformTags,
		DefinedTags:      summary.DefinedTags,
		SystemTags:       summary.SystemTags,
	}
}

func listLicenseRecordsAllPages(
	call func(context.Context, licensemanagersdk.ListLicenseRecordsRequest) (licensemanagersdk.ListLicenseRecordsResponse, error),
) func(context.Context, licensemanagersdk.ListLicenseRecordsRequest) (licensemanagersdk.ListLicenseRecordsResponse, error) {
	return func(ctx context.Context, request licensemanagersdk.ListLicenseRecordsRequest) (licensemanagersdk.ListLicenseRecordsResponse, error) {
		var combined licensemanagersdk.ListLicenseRecordsResponse
		for {
			response, err := call(ctx, request)
			if err != nil {
				return licensemanagersdk.ListLicenseRecordsResponse{}, err
			}
			combined.RawResponse = response.RawResponse
			combined.OpcRequestId = response.OpcRequestId
			combined.Items = append(combined.Items, response.Items...)
			if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
				combined.OpcNextPage = nil
				return combined, nil
			}
			request.Page = response.OpcNextPage
			combined.OpcNextPage = response.OpcNextPage
		}
	}
}

func handleLicenseRecordDeleteError(resource *licensemanagerv1beta1.LicenseRecord, err error) error {
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
	return licenseRecordAmbiguousNotFoundError{
		message:      "LicenseRecord delete returned ambiguous 404 NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed",
		opcRequestID: requestID,
	}
}

func applyLicenseRecordDeleteOutcome(
	resource *licensemanagerv1beta1.LicenseRecord,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	lifecycleState := strings.ToUpper(licenseRecordLifecycleState(response))
	switch lifecycleState {
	case "":
		if stage == generatedruntime.DeleteConfirmStageAfterRequest || licenseRecordDeleteAlreadyRequested(resource) {
			markLicenseRecordTerminating(resource, response)
			return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
		}
	case string(licensemanagersdk.LifeCycleStateActive), string(licensemanagersdk.LifeCycleStateInactive):
		if stage == generatedruntime.DeleteConfirmStageAfterRequest || licenseRecordDeleteAlreadyRequested(resource) {
			markLicenseRecordTerminating(resource, response)
			return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
		}
	case "DELETING":
		markLicenseRecordTerminating(resource, response)
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
	}
	return generatedruntime.DeleteOutcome{}, nil
}

func wrapLicenseRecordDeleteGuard(hooks *LicenseRecordRuntimeHooks) {
	if hooks == nil || hooks.Get.Call == nil {
		return
	}
	getLicenseRecord := hooks.Get.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate LicenseRecordServiceClient) LicenseRecordServiceClient {
		return licenseRecordDeleteGuardClient{
			delegate:         delegate,
			getLicenseRecord: getLicenseRecord,
		}
	})
}

type licenseRecordDeleteGuardClient struct {
	delegate         LicenseRecordServiceClient
	getLicenseRecord func(context.Context, licensemanagersdk.GetLicenseRecordRequest) (licensemanagersdk.GetLicenseRecordResponse, error)
}

func (c licenseRecordDeleteGuardClient) CreateOrUpdate(
	ctx context.Context,
	resource *licensemanagerv1beta1.LicenseRecord,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c licenseRecordDeleteGuardClient) Delete(
	ctx context.Context,
	resource *licensemanagerv1beta1.LicenseRecord,
) (bool, error) {
	if err := c.rejectAuthShapedPreDeleteConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c licenseRecordDeleteGuardClient) rejectAuthShapedPreDeleteConfirmRead(
	ctx context.Context,
	resource *licensemanagerv1beta1.LicenseRecord,
) error {
	if c.getLicenseRecord == nil || resource == nil {
		return nil
	}
	recordID := trackedLicenseRecordID(resource)
	if recordID == "" {
		return nil
	}
	_, err := c.getLicenseRecord(ctx, licensemanagersdk.GetLicenseRecordRequest{LicenseRecordId: common.String(recordID)})
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("LicenseRecord delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to call delete: %w", err)
}

func markLicenseRecordTerminating(resource *licensemanagerv1beta1.LicenseRecord, response any) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = licenseRecordDeletePendingMessage
	status.Reason = string(shared.Terminating)
	rawStatus := licenseRecordLifecycleState(response)
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		RawStatus:       rawStatus,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         licenseRecordDeletePendingMessage,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(
		*status,
		shared.Terminating,
		corev1.ConditionTrue,
		"",
		licenseRecordDeletePendingMessage,
		loggerutil.OSOKLogger{},
	)
}

func licenseRecordDeleteAlreadyRequested(resource *licensemanagerv1beta1.LicenseRecord) bool {
	if resource == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	if current != nil && current.Phase == shared.OSOKAsyncPhaseDelete {
		return true
	}
	if resource.Status.OsokStatus.Reason == string(shared.Terminating) {
		return true
	}
	conditions := resource.Status.OsokStatus.Conditions
	return len(conditions) > 0 && conditions[len(conditions)-1].Type == shared.Terminating
}

func licenseRecordLifecycleState(response any) string {
	current, ok := licenseRecordBodyFromResponse(response)
	if !ok {
		return ""
	}
	return string(current.LifecycleState)
}

func licenseRecordProductLicenseIDAnnotation(resource *licensemanagerv1beta1.LicenseRecord) string {
	if resource == nil {
		return ""
	}
	return annotationValue(resource.Annotations, LicenseRecordProductLicenseIDAnnotation, licenseRecordLegacyProductLicenseIDAnnotation)
}

func annotationValue(annotations map[string]string, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(annotations[key]); value != "" {
			return value
		}
	}
	return ""
}

func trackedLicenseRecordID(resource *licensemanagerv1beta1.LicenseRecord) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func licenseRecordOptionalSDKTime(fieldName string, value string) (*common.SDKTime, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		parsed, err = time.Parse("2006-01-02", value)
	}
	if err != nil {
		return nil, fmt.Errorf("%s must be RFC3339 or YYYY-MM-DD: %w", fieldName, err)
	}
	return &common.SDKTime{Time: parsed}, nil
}

func cloneLicenseRecordStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func licenseRecordDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(tags))
	for namespace, values := range tags {
		converted[namespace] = make(map[string]interface{}, len(values))
		for key, value := range values {
			converted[namespace][key] = value
		}
	}
	return converted
}

func firstNonEmptyString(values ...string) string {
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

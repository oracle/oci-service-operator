/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package importedpackage

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	odasdk "github.com/oracle/oci-go-sdk/v65/oda"
	odav1beta1 "github.com/oracle/oci-service-operator/api/oda/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const importedPackageOdaInstanceIDAnnotation = "oda.oracle.com/oda-instance-id"

type importedPackageIdentity struct {
	odaInstanceID    string
	trackedPackageID string
	desiredPackageID string
}

type importedPackageRuntimeBody struct {
	ID               string                            `json:"id,omitempty"`
	OdaInstanceID    *string                           `json:"odaInstanceId,omitempty"`
	CurrentPackageID *string                           `json:"currentPackageId,omitempty"`
	Name             *string                           `json:"name,omitempty"`
	DisplayName      *string                           `json:"displayName,omitempty"`
	Version          *string                           `json:"version,omitempty"`
	SDKStatus        string                            `json:"sdkStatus,omitempty"`
	LifecycleState   string                            `json:"lifecycleState,omitempty"`
	TimeCreated      *common.SDKTime                   `json:"timeCreated,omitempty"`
	TimeUpdated      *common.SDKTime                   `json:"timeUpdated,omitempty"`
	StatusMessage    *string                           `json:"statusMessage,omitempty"`
	ParameterValues  map[string]string                 `json:"parameterValues,omitempty"`
	FreeformTags     map[string]string                 `json:"freeformTags,omitempty"`
	DefinedTags      map[string]map[string]interface{} `json:"definedTags,omitempty"`
}

type importedPackageCreateResponse struct {
	ImportedPackage  importedPackageRuntimeBody `presentIn:"body"`
	OpcRequestId     *string                    `presentIn:"header" name:"opc-request-id"`
	OpcWorkRequestId *string                    `presentIn:"header" name:"opc-work-request-id"`
}

type importedPackageGetResponse struct {
	ImportedPackage importedPackageRuntimeBody `presentIn:"body"`
	OpcRequestId    *string                    `presentIn:"header" name:"opc-request-id"`
	Etag            *string                    `presentIn:"header" name:"etag"`
}

type importedPackageUpdateResponse struct {
	ImportedPackage  importedPackageRuntimeBody `presentIn:"body"`
	OpcRequestId     *string                    `presentIn:"header" name:"opc-request-id"`
	OpcWorkRequestId *string                    `presentIn:"header" name:"opc-work-request-id"`
}

type importedPackageListResponse struct {
	ImportedPackages importedPackageCollection `presentIn:"body"`
	OpcRequestId     *string                   `presentIn:"header" name:"opc-request-id"`
	OpcNextPage      *string                   `presentIn:"header" name:"opc-next-page"`
}

type importedPackageCollection struct {
	Items []importedPackageRuntimeBody `json:"items"`
}

type importedPackageDeleteResponse struct {
	OpcRequestId     *string `presentIn:"header" name:"opc-request-id"`
	OpcWorkRequestId *string `presentIn:"header" name:"opc-work-request-id"`
}

func init() {
	registerImportedPackageRuntimeHooksMutator(func(manager *ImportedPackageServiceManager, hooks *ImportedPackageRuntimeHooks) {
		applyImportedPackageRuntimeHooks(manager, hooks)
	})
}

func applyImportedPackageRuntimeHooks(manager *ImportedPackageServiceManager, hooks *ImportedPackageRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newImportedPackageRuntimeSemantics()
	hooks.Identity = generatedruntime.IdentityHooks[*odav1beta1.ImportedPackage]{
		Resolve:       resolveImportedPackageIdentity,
		RecordPath:    recordImportedPackagePathIdentity,
		RecordTracked: recordImportedPackageTrackedIdentity,
	}
	hooks.TrackedRecreate = generatedruntime.TrackedRecreateHooks[*odav1beta1.ImportedPackage]{
		ClearTrackedIdentity: clearImportedPackageTrackedIdentity,
	}
	hooks.BuildCreateBody = buildImportedPackageCreateBody
	hooks.BuildUpdateBody = buildImportedPackageUpdateBody
	hooks.DeleteHooks = generatedruntime.DeleteHooks[*odav1beta1.ImportedPackage]{
		ApplyOutcome: applyImportedPackageDeleteOutcome,
	}
	hooks.Get.Fields = importedPackageGetFields()
	hooks.List.Fields = importedPackageListFields()
	hooks.Update.Fields = importedPackageUpdateFields()
	hooks.Delete.Fields = importedPackageDeleteFields()
	hooks.Create.Fields = importedPackageCreateFields()
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(ImportedPackageServiceClient) ImportedPackageServiceClient {
		return newImportedPackageAdaptedGeneratedClient(manager, *hooks)
	})
}

func newImportedPackageRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "oda",
		FormalSlug:    "importedpackage",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(odasdk.ImportedPackageStatusOperationPending)},
			ActiveStates:       []string{string(odasdk.ImportedPackageStatusReady)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(odasdk.ImportedPackageStatusOperationPending)},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"odaInstanceId", "currentPackageId"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"currentPackageId", "parameterValues", "freeformTags", "definedTags"},
			ConflictsWith: map[string][]string{},
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

func newImportedPackageAdaptedGeneratedClient(
	manager *ImportedPackageServiceManager,
	hooks ImportedPackageRuntimeHooks,
) ImportedPackageServiceClient {
	var log = servicemanager.RuntimeDeps{}.Log
	if manager != nil {
		log = manager.Log
	}

	config := generatedruntime.Config[*odav1beta1.ImportedPackage]{
		Kind:            "ImportedPackage",
		SDKName:         "ImportedPackage",
		Log:             log,
		Semantics:       hooks.Semantics,
		Identity:        hooks.Identity,
		Read:            hooks.Read,
		TrackedRecreate: hooks.TrackedRecreate,
		StatusHooks:     hooks.StatusHooks,
		ParityHooks:     hooks.ParityHooks,
		Async:           hooks.Async,
		DeleteHooks:     hooks.DeleteHooks,
		BuildCreateBody: hooks.BuildCreateBody,
		BuildUpdateBody: hooks.BuildUpdateBody,
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &odasdk.CreateImportedPackageRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), hooks.Create.Fields...),
			Call: func(ctx context.Context, request any) (any, error) {
				response, err := hooks.Create.Call(ctx, *request.(*odasdk.CreateImportedPackageRequest))
				if err != nil {
					return nil, err
				}
				return adaptImportedPackageCreateResponse(response), nil
			},
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &odasdk.GetImportedPackageRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), hooks.Get.Fields...),
			Call: func(ctx context.Context, request any) (any, error) {
				response, err := hooks.Get.Call(ctx, *request.(*odasdk.GetImportedPackageRequest))
				if err != nil {
					return nil, err
				}
				return adaptImportedPackageGetResponse(response), nil
			},
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &odasdk.ListImportedPackagesRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), hooks.List.Fields...),
			Call: func(ctx context.Context, request any) (any, error) {
				response, err := hooks.List.Call(ctx, *request.(*odasdk.ListImportedPackagesRequest))
				if err != nil {
					return nil, err
				}
				return adaptImportedPackageListResponse(response), nil
			},
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &odasdk.UpdateImportedPackageRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), hooks.Update.Fields...),
			Call: func(ctx context.Context, request any) (any, error) {
				response, err := hooks.Update.Call(ctx, *request.(*odasdk.UpdateImportedPackageRequest))
				if err != nil {
					return nil, err
				}
				return adaptImportedPackageUpdateResponse(response), nil
			},
		},
		Delete: &generatedruntime.Operation{
			NewRequest: func() any { return &odasdk.DeleteImportedPackageRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), hooks.Delete.Fields...),
			Call: func(ctx context.Context, request any) (any, error) {
				response, err := hooks.Delete.Call(ctx, *request.(*odasdk.DeleteImportedPackageRequest))
				if err != nil {
					return nil, err
				}
				return adaptImportedPackageDeleteResponse(response), nil
			},
		},
	}

	return defaultImportedPackageServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*odav1beta1.ImportedPackage](config),
	}
}

func importedPackageCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "OdaInstanceId", RequestName: "odaInstanceId", Contribution: "path", LookupPaths: []string{"status.odaInstanceId", "odaInstanceId"}},
		{FieldName: "CreateImportedPackageDetails", RequestName: "CreateImportedPackageDetails", Contribution: "body"},
	}
}

func importedPackageGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "OdaInstanceId", RequestName: "odaInstanceId", Contribution: "path", LookupPaths: []string{"status.odaInstanceId", "odaInstanceId"}},
		{FieldName: "PackageId", RequestName: "packageId", Contribution: "path", PreferResourceID: true, LookupPaths: importedPackageIDLookupPaths()},
	}
}

func importedPackageListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "OdaInstanceId", RequestName: "odaInstanceId", Contribution: "path", LookupPaths: []string{"status.odaInstanceId", "odaInstanceId"}},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func importedPackageUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "OdaInstanceId", RequestName: "odaInstanceId", Contribution: "path", LookupPaths: []string{"status.odaInstanceId", "odaInstanceId"}},
		{FieldName: "PackageId", RequestName: "packageId", Contribution: "path", PreferResourceID: true, LookupPaths: importedPackageIDLookupPaths()},
		{FieldName: "UpdateImportedPackageDetails", RequestName: "UpdateImportedPackageDetails", Contribution: "body"},
	}
}

func importedPackageDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "OdaInstanceId", RequestName: "odaInstanceId", Contribution: "path", LookupPaths: []string{"status.odaInstanceId", "odaInstanceId"}},
		{FieldName: "PackageId", RequestName: "packageId", Contribution: "path", PreferResourceID: true, LookupPaths: importedPackageIDLookupPaths()},
	}
}

func importedPackageIDLookupPaths() []string {
	return []string{"status.status.ocid", "status.currentPackageId", "currentPackageId", "spec.currentPackageId"}
}

func resolveImportedPackageIdentity(resource *odav1beta1.ImportedPackage) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("ImportedPackage resource is nil")
	}

	statusOdaInstanceID := strings.TrimSpace(resource.Status.OdaInstanceId)
	annotationOdaInstanceID := strings.TrimSpace(resource.Annotations[importedPackageOdaInstanceIDAnnotation])
	switch {
	case statusOdaInstanceID != "" && annotationOdaInstanceID != "" && statusOdaInstanceID != annotationOdaInstanceID:
		return nil, fmt.Errorf("ImportedPackage cannot change odaInstanceId from %q to %q; create a replacement resource instead", statusOdaInstanceID, annotationOdaInstanceID)
	case statusOdaInstanceID != "":
	case annotationOdaInstanceID != "":
		statusOdaInstanceID = annotationOdaInstanceID
	default:
		return nil, fmt.Errorf("ImportedPackage requires %s annotation until odaInstanceId is promoted into the CR spec", importedPackageOdaInstanceIDAnnotation)
	}

	desiredPackageID := strings.TrimSpace(resource.Spec.CurrentPackageId)
	trackedPackageID := firstNonEmptyImportedPackageString(string(resource.Status.OsokStatus.Ocid), resource.Status.CurrentPackageId, desiredPackageID)
	if trackedPackageID == "" {
		return nil, fmt.Errorf("ImportedPackage requires spec.currentPackageId")
	}

	return importedPackageIdentity{
		odaInstanceID:    statusOdaInstanceID,
		trackedPackageID: trackedPackageID,
		desiredPackageID: desiredPackageID,
	}, nil
}

func recordImportedPackagePathIdentity(resource *odav1beta1.ImportedPackage, identity any) {
	importedPackageID, ok := identity.(importedPackageIdentity)
	if !ok || resource == nil {
		return
	}
	resource.Status.OdaInstanceId = importedPackageID.odaInstanceID
	if resource.Status.CurrentPackageId == "" {
		resource.Status.CurrentPackageId = importedPackageID.trackedPackageID
	}
	if resource.Status.OsokStatus.Ocid == "" && resource.Status.CurrentPackageId != "" && resource.Status.CurrentPackageId != importedPackageID.desiredPackageID {
		resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.CurrentPackageId)
	}
}

func recordImportedPackageTrackedIdentity(resource *odav1beta1.ImportedPackage, identity any, resourceID string) {
	if resource == nil {
		return
	}
	importedPackageID, _ := identity.(importedPackageIdentity)
	resourceID = firstNonEmptyImportedPackageString(resourceID, importedPackageID.desiredPackageID, importedPackageID.trackedPackageID)
	if resourceID == "" {
		return
	}
	resource.Status.OsokStatus.Ocid = shared.OCID(resourceID)
	resource.Status.CurrentPackageId = resourceID
	if importedPackageID.odaInstanceID != "" {
		resource.Status.OdaInstanceId = importedPackageID.odaInstanceID
	}
}

func seedImportedPackageTrackedID(resource *odav1beta1.ImportedPackage, identity any) func() {
	if resource == nil {
		return nil
	}
	importedPackageID, ok := identity.(importedPackageIdentity)
	if !ok || importedPackageID.trackedPackageID == "" {
		return nil
	}

	previousOcid := resource.Status.OsokStatus.Ocid
	previousCurrentPackageID := resource.Status.CurrentPackageId
	resource.Status.OsokStatus.Ocid = shared.OCID(importedPackageID.trackedPackageID)
	if resource.Status.CurrentPackageId == "" {
		resource.Status.CurrentPackageId = importedPackageID.trackedPackageID
	}
	return func() {
		resource.Status.OsokStatus.Ocid = previousOcid
		resource.Status.CurrentPackageId = previousCurrentPackageID
	}
}

func clearImportedPackageTrackedIdentity(resource *odav1beta1.ImportedPackage) {
	if resource == nil {
		return
	}
	resource.Status.OsokStatus.Ocid = ""
	resource.Status.CurrentPackageId = ""
}

func buildImportedPackageCreateBody(_ context.Context, resource *odav1beta1.ImportedPackage, _ string) (any, error) {
	packageID := strings.TrimSpace(resource.Spec.CurrentPackageId)
	if packageID == "" {
		return nil, fmt.Errorf("spec.currentPackageId is required")
	}
	details := odasdk.CreateImportedPackageDetails{
		CurrentPackageId: common.String(packageID),
		ParameterValues:  cloneImportedPackageStringMap(resource.Spec.ParameterValues),
		FreeformTags:     cloneImportedPackageStringMap(resource.Spec.FreeformTags),
	}
	if resource.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&resource.Spec.DefinedTags)
	}
	return details, nil
}

func buildImportedPackageUpdateBody(_ context.Context, resource *odav1beta1.ImportedPackage, _ string, currentResponse any) (any, bool, error) {
	packageID := strings.TrimSpace(resource.Spec.CurrentPackageId)
	if packageID == "" {
		return nil, false, fmt.Errorf("spec.currentPackageId is required")
	}

	current, ok := importedPackageRuntimeBodyFromResponse(currentResponse)
	if !ok {
		return nil, false, fmt.Errorf("current ImportedPackage response is missing package details")
	}

	updateNeeded := false
	if packageID != stringFromPointer(current.CurrentPackageID) {
		updateNeeded = true
	}
	if resource.Spec.ParameterValues != nil && !reflect.DeepEqual(resource.Spec.ParameterValues, current.ParameterValues) {
		updateNeeded = true
	}
	if resource.Spec.FreeformTags != nil && !reflect.DeepEqual(resource.Spec.FreeformTags, current.FreeformTags) {
		updateNeeded = true
	}
	if resource.Spec.DefinedTags != nil && !reflect.DeepEqual(resource.Spec.DefinedTags, importedPackageSharedDefinedTags(current.DefinedTags)) {
		updateNeeded = true
	}
	if !updateNeeded {
		return nil, false, nil
	}

	parameterValues := cloneImportedPackageStringMap(resource.Spec.ParameterValues)
	if parameterValues == nil {
		parameterValues = cloneImportedPackageStringMap(current.ParameterValues)
	}
	if parameterValues == nil {
		parameterValues = map[string]string{}
	}

	details := odasdk.UpdateImportedPackageDetails{
		CurrentPackageId: common.String(packageID),
		ParameterValues:  parameterValues,
	}
	if resource.Spec.FreeformTags != nil {
		details.FreeformTags = cloneImportedPackageStringMap(resource.Spec.FreeformTags)
	}
	if resource.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&resource.Spec.DefinedTags)
	}
	return details, true, nil
}

func applyImportedPackageDeleteOutcome(resource *odav1beta1.ImportedPackage, response any, stage generatedruntime.DeleteConfirmStage) (generatedruntime.DeleteOutcome, error) {
	if stage != generatedruntime.DeleteConfirmStageAfterRequest {
		return generatedruntime.DeleteOutcome{}, nil
	}

	current, ok := importedPackageRuntimeBodyFromResponse(response)
	if !ok || !strings.EqualFold(current.LifecycleState, string(odasdk.ImportedPackageStatusReady)) {
		return generatedruntime.DeleteOutcome{}, nil
	}

	now := metav1.Now()
	message := "OCI resource delete is in progress"
	status := &resource.Status.OsokStatus
	status.Message = message
	status.Reason = string(shared.Terminating)
	status.UpdatedAt = &now
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         message,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, servicemanager.RuntimeDeps{}.Log)
	return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
}

func adaptImportedPackageCreateResponse(response odasdk.CreateImportedPackageResponse) importedPackageCreateResponse {
	return importedPackageCreateResponse{
		ImportedPackage:  adaptImportedPackageBody(response.ImportedPackage),
		OpcRequestId:     response.OpcRequestId,
		OpcWorkRequestId: response.OpcWorkRequestId,
	}
}

func adaptImportedPackageGetResponse(response odasdk.GetImportedPackageResponse) importedPackageGetResponse {
	return importedPackageGetResponse{
		ImportedPackage: adaptImportedPackageBody(response.ImportedPackage),
		OpcRequestId:    response.OpcRequestId,
		Etag:            response.Etag,
	}
}

func adaptImportedPackageUpdateResponse(response odasdk.UpdateImportedPackageResponse) importedPackageUpdateResponse {
	return importedPackageUpdateResponse{
		ImportedPackage:  adaptImportedPackageBody(response.ImportedPackage),
		OpcRequestId:     response.OpcRequestId,
		OpcWorkRequestId: response.OpcWorkRequestId,
	}
}

func adaptImportedPackageListResponse(response odasdk.ListImportedPackagesResponse) importedPackageListResponse {
	items := make([]importedPackageRuntimeBody, 0, len(response.Items))
	for _, item := range response.Items {
		items = append(items, adaptImportedPackageSummaryBody(item))
	}
	return importedPackageListResponse{
		ImportedPackages: importedPackageCollection{Items: items},
		OpcRequestId:     response.OpcRequestId,
		OpcNextPage:      response.OpcNextPage,
	}
}

func adaptImportedPackageDeleteResponse(response odasdk.DeleteImportedPackageResponse) importedPackageDeleteResponse {
	return importedPackageDeleteResponse{
		OpcRequestId:     response.OpcRequestId,
		OpcWorkRequestId: response.OpcWorkRequestId,
	}
}

func adaptImportedPackageBody(pkg odasdk.ImportedPackage) importedPackageRuntimeBody {
	status := string(pkg.Status)
	currentPackageID := stringFromPointer(pkg.CurrentPackageId)
	return importedPackageRuntimeBody{
		ID:               currentPackageID,
		OdaInstanceID:    pkg.OdaInstanceId,
		CurrentPackageID: pkg.CurrentPackageId,
		Name:             pkg.Name,
		DisplayName:      pkg.DisplayName,
		Version:          pkg.Version,
		SDKStatus:        status,
		LifecycleState:   status,
		TimeCreated:      pkg.TimeCreated,
		TimeUpdated:      pkg.TimeUpdated,
		StatusMessage:    pkg.StatusMessage,
		ParameterValues:  cloneImportedPackageStringMap(pkg.ParameterValues),
		FreeformTags:     cloneImportedPackageStringMap(pkg.FreeformTags),
		DefinedTags:      cloneImportedPackageOCIDefinedTags(pkg.DefinedTags),
	}
}

func adaptImportedPackageSummaryBody(pkg odasdk.ImportedPackageSummary) importedPackageRuntimeBody {
	status := string(pkg.Status)
	currentPackageID := stringFromPointer(pkg.CurrentPackageId)
	return importedPackageRuntimeBody{
		ID:               currentPackageID,
		OdaInstanceID:    pkg.OdaInstanceId,
		CurrentPackageID: pkg.CurrentPackageId,
		Name:             pkg.Name,
		DisplayName:      pkg.DisplayName,
		Version:          pkg.Version,
		SDKStatus:        status,
		LifecycleState:   status,
		TimeCreated:      pkg.TimeCreated,
		TimeUpdated:      pkg.TimeUpdated,
		FreeformTags:     cloneImportedPackageStringMap(pkg.FreeformTags),
		DefinedTags:      cloneImportedPackageOCIDefinedTags(pkg.DefinedTags),
	}
}

func importedPackageRuntimeBodyFromResponse(response any) (importedPackageRuntimeBody, bool) {
	switch typed := response.(type) {
	case importedPackageRuntimeBody:
		return typed, true
	case *importedPackageRuntimeBody:
		if typed == nil {
			return importedPackageRuntimeBody{}, false
		}
		return *typed, true
	case importedPackageCreateResponse:
		return typed.ImportedPackage, true
	case *importedPackageCreateResponse:
		if typed == nil {
			return importedPackageRuntimeBody{}, false
		}
		return typed.ImportedPackage, true
	case importedPackageGetResponse:
		return typed.ImportedPackage, true
	case *importedPackageGetResponse:
		if typed == nil {
			return importedPackageRuntimeBody{}, false
		}
		return typed.ImportedPackage, true
	case importedPackageUpdateResponse:
		return typed.ImportedPackage, true
	case *importedPackageUpdateResponse:
		if typed == nil {
			return importedPackageRuntimeBody{}, false
		}
		return typed.ImportedPackage, true
	default:
		return importedPackageRuntimeBody{}, false
	}
}

func firstNonEmptyImportedPackageString(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func stringFromPointer(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func cloneImportedPackageStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func cloneImportedPackageOCIDefinedTags(source map[string]map[string]interface{}) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	clone := make(map[string]map[string]interface{}, len(source))
	for namespace, tags := range source {
		clone[namespace] = make(map[string]interface{}, len(tags))
		for key, value := range tags {
			clone[namespace][key] = value
		}
	}
	return clone
}

func importedPackageSharedDefinedTags(source map[string]map[string]interface{}) map[string]shared.MapValue {
	if source == nil {
		return nil
	}
	clone := make(map[string]shared.MapValue, len(source))
	for namespace, tags := range source {
		clone[namespace] = make(shared.MapValue, len(tags))
		for key, value := range tags {
			clone[namespace][key] = fmt.Sprint(value)
		}
	}
	return clone
}

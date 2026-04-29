/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dataset

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	datalabelingservicesdk "github.com/oracle/oci-go-sdk/v65/datalabelingservice"
	datalabelingservicev1beta1 "github.com/oracle/oci-service-operator/api/datalabelingservice/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

type datasetOCIClient interface {
	CreateDataset(context.Context, datalabelingservicesdk.CreateDatasetRequest) (datalabelingservicesdk.CreateDatasetResponse, error)
	GetDataset(context.Context, datalabelingservicesdk.GetDatasetRequest) (datalabelingservicesdk.GetDatasetResponse, error)
	ListDatasets(context.Context, datalabelingservicesdk.ListDatasetsRequest) (datalabelingservicesdk.ListDatasetsResponse, error)
	UpdateDataset(context.Context, datalabelingservicesdk.UpdateDatasetRequest) (datalabelingservicesdk.UpdateDatasetResponse, error)
	DeleteDataset(context.Context, datalabelingservicesdk.DeleteDatasetRequest) (datalabelingservicesdk.DeleteDatasetResponse, error)
}

func init() {
	registerDatasetRuntimeHooksMutator(func(_ *DatasetServiceManager, hooks *DatasetRuntimeHooks) {
		applyDatasetRuntimeHooks(hooks)
	})
}

func applyDatasetRuntimeHooks(hooks *DatasetRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedDatasetRuntimeSemantics()
	hooks.Identity.GuardExistingBeforeCreate = guardDatasetExistingBeforeCreate
	hooks.BuildCreateBody = func(
		ctx context.Context,
		resource *datalabelingservicev1beta1.Dataset,
		namespace string,
	) (any, error) {
		return buildDatasetCreateDetails(ctx, resource, namespace)
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *datalabelingservicev1beta1.Dataset,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildDatasetUpdateBody(resource, currentResponse)
	}
	hooks.Create.Fields = datasetCreateFields()
	hooks.Get.Fields = datasetGetFields()
	hooks.List.Fields = datasetListFields()
	hooks.Update.Fields = datasetUpdateFields()
	hooks.Delete.Fields = datasetDeleteFields()
}

func newDatasetServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client datasetOCIClient,
) DatasetServiceClient {
	return defaultDatasetServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*datalabelingservicev1beta1.Dataset](
			newDatasetRuntimeConfig(log, client),
		),
	}
}

func newDatasetRuntimeConfig(
	log loggerutil.OSOKLogger,
	client datasetOCIClient,
) generatedruntime.Config[*datalabelingservicev1beta1.Dataset] {
	hooks := newDatasetRuntimeHooksWithOCIClient(client)
	applyDatasetRuntimeHooks(&hooks)
	return buildDatasetGeneratedRuntimeConfig(&DatasetServiceManager{Log: log}, hooks)
}

func newDatasetRuntimeHooksWithOCIClient(client datasetOCIClient) DatasetRuntimeHooks {
	return DatasetRuntimeHooks{
		Semantics:       newDatasetRuntimeSemantics(),
		Identity:        generatedruntime.IdentityHooks[*datalabelingservicev1beta1.Dataset]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*datalabelingservicev1beta1.Dataset]{},
		StatusHooks:     generatedruntime.StatusHooks[*datalabelingservicev1beta1.Dataset]{},
		ParityHooks:     generatedruntime.ParityHooks[*datalabelingservicev1beta1.Dataset]{},
		Async:           generatedruntime.AsyncHooks[*datalabelingservicev1beta1.Dataset]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*datalabelingservicev1beta1.Dataset]{},
		Create: runtimeOperationHooks[datalabelingservicesdk.CreateDatasetRequest, datalabelingservicesdk.CreateDatasetResponse]{
			Fields: datasetCreateFields(),
			Call: func(ctx context.Context, request datalabelingservicesdk.CreateDatasetRequest) (datalabelingservicesdk.CreateDatasetResponse, error) {
				return client.CreateDataset(ctx, request)
			},
		},
		Get: runtimeOperationHooks[datalabelingservicesdk.GetDatasetRequest, datalabelingservicesdk.GetDatasetResponse]{
			Fields: datasetGetFields(),
			Call: func(ctx context.Context, request datalabelingservicesdk.GetDatasetRequest) (datalabelingservicesdk.GetDatasetResponse, error) {
				return client.GetDataset(ctx, request)
			},
		},
		List: runtimeOperationHooks[datalabelingservicesdk.ListDatasetsRequest, datalabelingservicesdk.ListDatasetsResponse]{
			Fields: datasetListFields(),
			Call: func(ctx context.Context, request datalabelingservicesdk.ListDatasetsRequest) (datalabelingservicesdk.ListDatasetsResponse, error) {
				return client.ListDatasets(ctx, request)
			},
		},
		Update: runtimeOperationHooks[datalabelingservicesdk.UpdateDatasetRequest, datalabelingservicesdk.UpdateDatasetResponse]{
			Fields: datasetUpdateFields(),
			Call: func(ctx context.Context, request datalabelingservicesdk.UpdateDatasetRequest) (datalabelingservicesdk.UpdateDatasetResponse, error) {
				return client.UpdateDataset(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[datalabelingservicesdk.DeleteDatasetRequest, datalabelingservicesdk.DeleteDatasetResponse]{
			Fields: datasetDeleteFields(),
			Call: func(ctx context.Context, request datalabelingservicesdk.DeleteDatasetRequest) (datalabelingservicesdk.DeleteDatasetResponse, error) {
				return client.DeleteDataset(ctx, request)
			},
		},
	}
}

func reviewedDatasetRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newDatasetRuntimeSemantics()
	semantics.List = &generatedruntime.ListSemantics{
		ResponseItemsField: "Items",
		MatchFields:        []string{"annotationFormat", "compartmentId", "displayName", "id"},
	}
	semantics.AuxiliaryOperations = nil
	return semantics
}

func datasetCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateDatasetDetails", RequestName: "CreateDatasetDetails", Contribution: "body"},
	}
}

func datasetGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "DatasetId", RequestName: "datasetId", Contribution: "path", PreferResourceID: true},
	}
}

func datasetListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "spec.compartmentId", "compartmentId"},
		},
		{
			FieldName:    "AnnotationFormat",
			RequestName:  "annotationFormat",
			Contribution: "query",
			LookupPaths:  []string{"status.annotationFormat", "spec.annotationFormat", "annotationFormat"},
		},
		{
			FieldName:    "DisplayName",
			RequestName:  "displayName",
			Contribution: "query",
			LookupPaths:  []string{"status.displayName", "spec.displayName", "displayName"},
		},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
	}
}

func datasetUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "DatasetId", RequestName: "datasetId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateDatasetDetails", RequestName: "UpdateDatasetDetails", Contribution: "body"},
	}
}

func datasetDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "DatasetId", RequestName: "datasetId", Contribution: "path", PreferResourceID: true},
	}
}

func guardDatasetExistingBeforeCreate(
	_ context.Context,
	resource *datalabelingservicev1beta1.Dataset,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
	}
	if strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildDatasetCreateDetails(
	ctx context.Context,
	resource *datalabelingservicev1beta1.Dataset,
	namespace string,
) (datalabelingservicesdk.CreateDatasetDetails, error) {
	if resource == nil {
		return datalabelingservicesdk.CreateDatasetDetails{}, fmt.Errorf("dataset resource is nil")
	}

	resolvedSpec, err := generatedruntime.ResolveSpecValueWithBoolFields(resource, ctx, nil, namespace)
	if err != nil {
		return datalabelingservicesdk.CreateDatasetDetails{}, err
	}

	payload, err := json.Marshal(resolvedSpec)
	if err != nil {
		return datalabelingservicesdk.CreateDatasetDetails{}, fmt.Errorf("marshal resolved dataset spec: %w", err)
	}

	var details datalabelingservicesdk.CreateDatasetDetails
	if err := json.Unmarshal(payload, &details); err != nil {
		return datalabelingservicesdk.CreateDatasetDetails{}, fmt.Errorf("decode dataset create request body: %w", err)
	}

	return details, nil
}

func buildDatasetUpdateBody(
	resource *datalabelingservicev1beta1.Dataset,
	currentResponse any,
) (datalabelingservicesdk.UpdateDatasetDetails, bool, error) {
	if resource == nil {
		return datalabelingservicesdk.UpdateDatasetDetails{}, false, fmt.Errorf("dataset resource is nil")
	}

	current, err := datasetRuntimeBody(currentResponse)
	if err != nil {
		return datalabelingservicesdk.UpdateDatasetDetails{}, false, err
	}

	details := datalabelingservicesdk.UpdateDatasetDetails{}
	updateNeeded := false

	if desired, ok := datasetDesiredStringUpdate(resource.Spec.DisplayName, current.DisplayName); ok {
		details.DisplayName = desired
		updateNeeded = true
	}
	if desired, ok := datasetDesiredStringUpdate(resource.Spec.Description, current.Description); ok {
		details.Description = desired
		updateNeeded = true
	}
	if desired, ok := datasetDesiredStringUpdate(resource.Spec.LabelingInstructions, current.LabelingInstructions); ok {
		details.LabelingInstructions = desired
		updateNeeded = true
	}
	if desired, ok := datasetDesiredFreeformTagsUpdate(resource.Spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok := datasetDesiredDefinedTagsUpdate(resource.Spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = desired
		updateNeeded = true
	}

	return details, updateNeeded, nil
}

func datasetRuntimeBody(currentResponse any) (datalabelingservicesdk.Dataset, error) {
	switch current := currentResponse.(type) {
	case datalabelingservicesdk.Dataset:
		return current, nil
	case *datalabelingservicesdk.Dataset:
		if current == nil {
			return datalabelingservicesdk.Dataset{}, fmt.Errorf("current Dataset response is nil")
		}
		return *current, nil
	case datalabelingservicesdk.DatasetSummary:
		return datasetFromSummary(current), nil
	case *datalabelingservicesdk.DatasetSummary:
		if current == nil {
			return datalabelingservicesdk.Dataset{}, fmt.Errorf("current Dataset response is nil")
		}
		return datasetFromSummary(*current), nil
	case datalabelingservicesdk.CreateDatasetResponse:
		return current.Dataset, nil
	case *datalabelingservicesdk.CreateDatasetResponse:
		if current == nil {
			return datalabelingservicesdk.Dataset{}, fmt.Errorf("current Dataset response is nil")
		}
		return current.Dataset, nil
	case datalabelingservicesdk.GetDatasetResponse:
		return current.Dataset, nil
	case *datalabelingservicesdk.GetDatasetResponse:
		if current == nil {
			return datalabelingservicesdk.Dataset{}, fmt.Errorf("current Dataset response is nil")
		}
		return current.Dataset, nil
	case datalabelingservicesdk.UpdateDatasetResponse:
		return current.Dataset, nil
	case *datalabelingservicesdk.UpdateDatasetResponse:
		if current == nil {
			return datalabelingservicesdk.Dataset{}, fmt.Errorf("current Dataset response is nil")
		}
		return current.Dataset, nil
	default:
		return datalabelingservicesdk.Dataset{}, fmt.Errorf("unexpected current Dataset response type %T", currentResponse)
	}
}

func datasetFromSummary(summary datalabelingservicesdk.DatasetSummary) datalabelingservicesdk.Dataset {
	return datalabelingservicesdk.Dataset{
		Id:                   summary.Id,
		CompartmentId:        summary.CompartmentId,
		TimeCreated:          summary.TimeCreated,
		TimeUpdated:          summary.TimeUpdated,
		LifecycleState:       summary.LifecycleState,
		AnnotationFormat:     summary.AnnotationFormat,
		DatasetFormatDetails: summary.DatasetFormatDetails,
		DisplayName:          summary.DisplayName,
		LifecycleDetails:     summary.LifecycleDetails,
		FreeformTags:         summary.FreeformTags,
		DefinedTags:          summary.DefinedTags,
		SystemTags:           summary.SystemTags,
	}
}

func datasetDesiredStringUpdate(spec string, current *string) (*string, bool) {
	currentValue := ""
	if current != nil {
		currentValue = *current
	}
	if spec == currentValue {
		return nil, false
	}
	if spec == "" && current == nil {
		return nil, false
	}
	return common.String(spec), true
}

func datasetDesiredFreeformTagsUpdate(
	spec map[string]string,
	current map[string]string,
) (map[string]string, bool) {
	if spec == nil {
		return nil, false
	}
	if len(spec) == 0 && len(current) == 0 {
		return nil, false
	}
	if maps.Equal(spec, current) {
		return nil, false
	}
	return spec, true
}

func datasetDesiredDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}

	desired := datasetDefinedTagsFromSpec(spec)
	if len(desired) == 0 && len(current) == 0 {
		return nil, false
	}
	if reflect.DeepEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func datasetDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
	if spec == nil {
		return nil
	}

	converted := make(map[string]map[string]interface{}, len(spec))
	for namespace, values := range spec {
		inner := make(map[string]interface{}, len(values))
		for key, value := range values {
			inner[key] = value
		}
		converted[namespace] = inner
	}
	return converted
}

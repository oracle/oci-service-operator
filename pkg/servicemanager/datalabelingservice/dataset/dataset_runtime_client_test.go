/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dataset

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	datalabelingservicesdk "github.com/oracle/oci-go-sdk/v65/datalabelingservice"
	datalabelingservicev1beta1 "github.com/oracle/oci-service-operator/api/datalabelingservice/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeDatasetOCIClient struct {
	createFn func(context.Context, datalabelingservicesdk.CreateDatasetRequest) (datalabelingservicesdk.CreateDatasetResponse, error)
	getFn    func(context.Context, datalabelingservicesdk.GetDatasetRequest) (datalabelingservicesdk.GetDatasetResponse, error)
	listFn   func(context.Context, datalabelingservicesdk.ListDatasetsRequest) (datalabelingservicesdk.ListDatasetsResponse, error)
	updateFn func(context.Context, datalabelingservicesdk.UpdateDatasetRequest) (datalabelingservicesdk.UpdateDatasetResponse, error)
	deleteFn func(context.Context, datalabelingservicesdk.DeleteDatasetRequest) (datalabelingservicesdk.DeleteDatasetResponse, error)
}

func (f *fakeDatasetOCIClient) CreateDataset(
	ctx context.Context,
	req datalabelingservicesdk.CreateDatasetRequest,
) (datalabelingservicesdk.CreateDatasetResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return datalabelingservicesdk.CreateDatasetResponse{}, nil
}

func (f *fakeDatasetOCIClient) GetDataset(
	ctx context.Context,
	req datalabelingservicesdk.GetDatasetRequest,
) (datalabelingservicesdk.GetDatasetResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return datalabelingservicesdk.GetDatasetResponse{}, nil
}

func (f *fakeDatasetOCIClient) ListDatasets(
	ctx context.Context,
	req datalabelingservicesdk.ListDatasetsRequest,
) (datalabelingservicesdk.ListDatasetsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return datalabelingservicesdk.ListDatasetsResponse{}, nil
}

func (f *fakeDatasetOCIClient) UpdateDataset(
	ctx context.Context,
	req datalabelingservicesdk.UpdateDatasetRequest,
) (datalabelingservicesdk.UpdateDatasetResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return datalabelingservicesdk.UpdateDatasetResponse{}, nil
}

func (f *fakeDatasetOCIClient) DeleteDataset(
	ctx context.Context,
	req datalabelingservicesdk.DeleteDatasetRequest,
) (datalabelingservicesdk.DeleteDatasetResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return datalabelingservicesdk.DeleteDatasetResponse{}, nil
}

type datasetRequestBodyBuilder interface {
	HTTPRequest(
		method string,
		path string,
		binaryRequestBody *common.OCIReadSeekCloser,
		extraHeaders map[string]string,
	) (http.Request, error)
}

func testDatasetClient(fake *fakeDatasetOCIClient) DatasetServiceClient {
	return newDatasetServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: logr.Discard()},
		fake,
	)
}

func TestDatasetRuntimeHooksUseReviewedSemantics(t *testing.T) {
	t.Parallel()

	hooks := newDatasetRuntimeHooksWithOCIClient(&fakeDatasetOCIClient{})
	applyDatasetRuntimeHooks(&hooks)

	if !reflect.DeepEqual(hooks.Create.Fields, datasetCreateFields()) {
		t.Fatalf("create fields = %#v, want %#v", hooks.Create.Fields, datasetCreateFields())
	}
	if !reflect.DeepEqual(hooks.Get.Fields, datasetGetFields()) {
		t.Fatalf("get fields = %#v, want %#v", hooks.Get.Fields, datasetGetFields())
	}
	if !reflect.DeepEqual(hooks.List.Fields, datasetListFields()) {
		t.Fatalf("list fields = %#v, want %#v", hooks.List.Fields, datasetListFields())
	}
	if !reflect.DeepEqual(hooks.Update.Fields, datasetUpdateFields()) {
		t.Fatalf("update fields = %#v, want %#v", hooks.Update.Fields, datasetUpdateFields())
	}
	if !reflect.DeepEqual(hooks.Delete.Fields, datasetDeleteFields()) {
		t.Fatalf("delete fields = %#v, want %#v", hooks.Delete.Fields, datasetDeleteFields())
	}
	if hooks.Semantics == nil {
		t.Fatal("semantics = nil, want reviewed semantics")
	}
	if got := hooks.Semantics.List; got == nil {
		t.Fatal("semantics.list = nil, want reviewed list semantics")
	} else if !reflect.DeepEqual(got.MatchFields, []string{"annotationFormat", "compartmentId", "displayName", "id"}) {
		t.Fatalf("semantics.list.matchFields = %#v, want %#v", got.MatchFields, []string{"annotationFormat", "compartmentId", "displayName", "id"})
	}
	if len(hooks.Semantics.AuxiliaryOperations) != 0 {
		t.Fatalf("semantics.auxiliaryOperations = %#v, want none for published runtime", hooks.Semantics.AuxiliaryOperations)
	}
	if hooks.Identity.GuardExistingBeforeCreate == nil {
		t.Fatal("identity guard = nil, want reviewed pre-create reuse guard")
	}
	if hooks.BuildCreateBody == nil {
		t.Fatal("build create body = nil, want reviewed create builder")
	}
	if hooks.BuildUpdateBody == nil {
		t.Fatal("build update body = nil, want reviewed update builder")
	}

	resource := newDatasetTestResource()
	resource.Spec.DisplayName = ""
	decision, err := hooks.Identity.GuardExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("GuardExistingBeforeCreate() error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionSkip {
		t.Fatalf("GuardExistingBeforeCreate() = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionSkip)
	}
}

func TestDatasetCreateOrUpdateCreatesConcretePolymorphicBodyAndTracksLifecycle(t *testing.T) {
	t.Parallel()

	resource := newDatasetTestResource()
	createdID := "ocid1.dataset.oc1..created"

	var createRequest datalabelingservicesdk.CreateDatasetRequest
	var getRequest datalabelingservicesdk.GetDatasetRequest

	client := testDatasetClient(&fakeDatasetOCIClient{
		createFn: func(_ context.Context, req datalabelingservicesdk.CreateDatasetRequest) (datalabelingservicesdk.CreateDatasetResponse, error) {
			createRequest = req
			return datalabelingservicesdk.CreateDatasetResponse{
				OpcRequestId:     common.String("opc-create-1"),
				OpcWorkRequestId: common.String("wr-create-1"),
				Dataset:          observedDatasetFromSpec(createdID, resource.Spec, "CREATING"),
			}, nil
		},
		getFn: func(_ context.Context, req datalabelingservicesdk.GetDatasetRequest) (datalabelingservicesdk.GetDatasetResponse, error) {
			getRequest = req
			return datalabelingservicesdk.GetDatasetResponse{
				Dataset: observedDatasetFromSpec(createdID, resource.Spec, "CREATING"),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success while create is still CREATING")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should requeue while lifecycle remains CREATING")
	}

	sourceDetails, ok := createRequest.CreateDatasetDetails.DatasetSourceDetails.(datalabelingservicesdk.ObjectStorageSourceDetails)
	if !ok {
		t.Fatalf("datasetSourceDetails type = %T, want datalabelingservice.ObjectStorageSourceDetails", createRequest.CreateDatasetDetails.DatasetSourceDetails)
	}
	if sourceDetails.Namespace == nil || *sourceDetails.Namespace != resource.Spec.DatasetSourceDetails.Namespace {
		t.Fatalf("datasetSourceDetails.Namespace = %#v, want %q", sourceDetails.Namespace, resource.Spec.DatasetSourceDetails.Namespace)
	}

	formatDetails, ok := createRequest.CreateDatasetDetails.DatasetFormatDetails.(datalabelingservicesdk.TextDatasetFormatDetails)
	if !ok {
		t.Fatalf("datasetFormatDetails type = %T, want datalabelingservice.TextDatasetFormatDetails", createRequest.CreateDatasetDetails.DatasetFormatDetails)
	}
	textMetadata, ok := formatDetails.TextFileTypeMetadata.(datalabelingservicesdk.DelimitedFileTypeMetadata)
	if !ok {
		t.Fatalf("textFileTypeMetadata type = %T, want datalabelingservice.DelimitedFileTypeMetadata", formatDetails.TextFileTypeMetadata)
	}
	if textMetadata.ColumnIndex == nil || resource.Spec.DatasetFormatDetails.TextFileTypeMetadata == nil || *textMetadata.ColumnIndex != *resource.Spec.DatasetFormatDetails.TextFileTypeMetadata.ColumnIndex {
		t.Fatalf("textFileTypeMetadata.ColumnIndex = %#v, want %v", textMetadata.ColumnIndex, resource.Spec.DatasetFormatDetails.TextFileTypeMetadata)
	}

	if createRequest.CreateDatasetDetails.InitialImportDatasetConfiguration == nil {
		t.Fatal("initialImportDatasetConfiguration = nil, want concrete import configuration")
	}
	if _, ok := createRequest.CreateDatasetDetails.InitialImportDatasetConfiguration.ImportMetadataPath.(datalabelingservicesdk.ObjectStorageImportMetadataPath); !ok {
		t.Fatalf(
			"importMetadataPath type = %T, want datalabelingservice.ObjectStorageImportMetadataPath",
			createRequest.CreateDatasetDetails.InitialImportDatasetConfiguration.ImportMetadataPath,
		)
	}
	if createRequest.CreateDatasetDetails.InitialImportDatasetConfiguration.ImportFormat == nil ||
		createRequest.CreateDatasetDetails.InitialImportDatasetConfiguration.ImportFormat.Name != datalabelingservicesdk.ImportFormatNameJsonlConsolidated {
		t.Fatalf(
			"importFormat = %#v, want JSONL_CONSOLIDATED",
			createRequest.CreateDatasetDetails.InitialImportDatasetConfiguration.ImportFormat,
		)
	}

	if getRequest.DatasetId == nil || *getRequest.DatasetId != createdID {
		t.Fatalf("get request datasetId = %#v, want %q", getRequest.DatasetId, createdID)
	}

	body := datasetSerializedRequestBody(t, createRequest, http.MethodPost, "/datasets")
	for _, want := range []string{
		`"sourceType":"OBJECT_STORAGE"`,
		`"formatType":"TEXT"`,
		`"formatType":"DELIMITED"`,
		`"imports/preannotated.jsonl"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("request body %s does not contain %s", body, want)
		}
	}
	if strings.Contains(body, `"jsonData"`) {
		t.Fatalf("request body unexpectedly exposed jsonData helper field: %s", body)
	}

	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", got, "opc-create-1")
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want lifecycle tracker")
	}
	if current.Source != shared.OSOKAsyncSourceLifecycle {
		t.Fatalf("status.async.current.source = %q, want %q", current.Source, shared.OSOKAsyncSourceLifecycle)
	}
	if current.Phase != shared.OSOKAsyncPhaseCreate {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, shared.OSOKAsyncPhaseCreate)
	}
	if current.WorkRequestID != "wr-create-1" {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", current.WorkRequestID, "wr-create-1")
	}
	if current.RawStatus != "CREATING" {
		t.Fatalf("status.async.current.rawStatus = %q, want %q", current.RawStatus, "CREATING")
	}
	if got := resource.Status.LifecycleState; got != "CREATING" {
		t.Fatalf("status.lifecycleState = %q, want %q", got, "CREATING")
	}
	requireDatasetTrailingCondition(t, resource, shared.Provisioning)
}

func TestDatasetCreateOrUpdateReusesExistingActiveDatasetByReviewedListLookup(t *testing.T) {
	t.Parallel()

	resource := newDatasetTestResource()
	existingID := "ocid1.dataset.oc1..existing"

	createCalled := false
	var listRequest datalabelingservicesdk.ListDatasetsRequest
	var getRequest datalabelingservicesdk.GetDatasetRequest

	client := testDatasetClient(&fakeDatasetOCIClient{
		createFn: func(context.Context, datalabelingservicesdk.CreateDatasetRequest) (datalabelingservicesdk.CreateDatasetResponse, error) {
			createCalled = true
			t.Fatal("CreateDataset should not be called when a reusable ACTIVE match exists")
			return datalabelingservicesdk.CreateDatasetResponse{}, nil
		},
		listFn: func(_ context.Context, req datalabelingservicesdk.ListDatasetsRequest) (datalabelingservicesdk.ListDatasetsResponse, error) {
			listRequest = req
			return datalabelingservicesdk.ListDatasetsResponse{
				DatasetCollection: datalabelingservicesdk.DatasetCollection{
					Items: []datalabelingservicesdk.DatasetSummary{
						observedDatasetSummaryFromSpec(existingID, resource.Spec, "ACTIVE"),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, req datalabelingservicesdk.GetDatasetRequest) (datalabelingservicesdk.GetDatasetResponse, error) {
			getRequest = req
			return datalabelingservicesdk.GetDatasetResponse{
				Dataset: observedDatasetFromSpec(existingID, resource.Spec, "ACTIVE"),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if createCalled {
		t.Fatal("CreateDataset was called unexpectedly")
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success for reusable ACTIVE resource")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue for reusable ACTIVE resource")
	}

	if listRequest.CompartmentId == nil || *listRequest.CompartmentId != resource.Spec.CompartmentId {
		t.Fatalf("list compartmentId = %#v, want %q", listRequest.CompartmentId, resource.Spec.CompartmentId)
	}
	if listRequest.AnnotationFormat == nil || *listRequest.AnnotationFormat != resource.Spec.AnnotationFormat {
		t.Fatalf("list annotationFormat = %#v, want %q", listRequest.AnnotationFormat, resource.Spec.AnnotationFormat)
	}
	if listRequest.DisplayName == nil || *listRequest.DisplayName != resource.Spec.DisplayName {
		t.Fatalf("list displayName = %#v, want %q", listRequest.DisplayName, resource.Spec.DisplayName)
	}
	if listRequest.LifecycleState != "" {
		t.Fatalf("list lifecycleState = %q, want empty reviewed lookup filter", listRequest.LifecycleState)
	}
	if listRequest.Id != nil {
		t.Fatalf("list id = %#v, want nil before an OCI identifier is tracked", listRequest.Id)
	}
	if getRequest.DatasetId == nil || *getRequest.DatasetId != existingID {
		t.Fatalf("get request datasetId = %#v, want %q", getRequest.DatasetId, existingID)
	}
	if got := resource.Status.Id; got != existingID {
		t.Fatalf("status.id = %q, want %q", got, existingID)
	}
	requireDatasetTrailingCondition(t, resource, shared.Active)
}

func TestBuildDatasetUpdateBodySupportsClearingOptionalFields(t *testing.T) {
	t.Parallel()

	resource := newDatasetTestResource()
	resource.Spec.DisplayName = ""
	resource.Spec.Description = ""
	resource.Spec.LabelingInstructions = ""
	resource.Spec.FreeformTags = map[string]string{}
	resource.Spec.DefinedTags = map[string]shared.MapValue{}

	current := observedDatasetFromSpec("ocid1.dataset.oc1..existing", newDatasetTestResource().Spec, "ACTIVE")

	details, updateNeeded, err := buildDatasetUpdateBody(resource, current)
	if err != nil {
		t.Fatalf("buildDatasetUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildDatasetUpdateBody() updateNeeded = false, want clear intent")
	}
	if details.DisplayName == nil || *details.DisplayName != "" {
		t.Fatalf("DisplayName = %#v, want explicit empty string clear", details.DisplayName)
	}
	if details.Description == nil || *details.Description != "" {
		t.Fatalf("Description = %#v, want explicit empty string clear", details.Description)
	}
	if details.LabelingInstructions == nil || *details.LabelingInstructions != "" {
		t.Fatalf("LabelingInstructions = %#v, want explicit empty string clear", details.LabelingInstructions)
	}
	if details.FreeformTags == nil || len(details.FreeformTags) != 0 {
		t.Fatalf("FreeformTags = %#v, want explicit empty map clear", details.FreeformTags)
	}
	if details.DefinedTags == nil || len(details.DefinedTags) != 0 {
		t.Fatalf("DefinedTags = %#v, want explicit empty map clear", details.DefinedTags)
	}

	body := datasetSerializedRequestBody(t, datalabelingservicesdk.UpdateDatasetRequest{
		DatasetId:            common.String("ocid1.dataset.oc1..existing"),
		UpdateDatasetDetails: details,
	}, http.MethodPut, "/datasets/ocid1.dataset.oc1..existing")

	for _, want := range []string{
		`"displayName":""`,
		`"description":""`,
		`"labelingInstructions":""`,
		`"freeformTags":{}`,
		`"definedTags":{}`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("request body %s does not contain %s", body, want)
		}
	}
}

func TestBuildDatasetCreateDetailsPreservesZeroColumnIndex(t *testing.T) {
	t.Parallel()

	resource := newDatasetTestResource()
	resource.Spec.DatasetFormatDetails.TextFileTypeMetadata.ColumnIndex = ptr(0)

	details, err := buildDatasetCreateDetails(context.Background(), resource, resource.Namespace)
	if err != nil {
		t.Fatalf("buildDatasetCreateDetails() error = %v", err)
	}

	formatDetails, ok := details.DatasetFormatDetails.(datalabelingservicesdk.TextDatasetFormatDetails)
	if !ok {
		t.Fatalf("datasetFormatDetails type = %T, want datalabelingservice.TextDatasetFormatDetails", details.DatasetFormatDetails)
	}
	textMetadata, ok := formatDetails.TextFileTypeMetadata.(datalabelingservicesdk.DelimitedFileTypeMetadata)
	if !ok {
		t.Fatalf("textFileTypeMetadata type = %T, want datalabelingservice.DelimitedFileTypeMetadata", formatDetails.TextFileTypeMetadata)
	}
	if textMetadata.ColumnIndex == nil || *textMetadata.ColumnIndex != 0 {
		t.Fatalf("textFileTypeMetadata.ColumnIndex = %#v, want explicit zero", textMetadata.ColumnIndex)
	}

	body := datasetSerializedRequestBody(t, datalabelingservicesdk.CreateDatasetRequest{
		CreateDatasetDetails: details,
	}, http.MethodPost, "/datasets")
	if !strings.Contains(body, `"columnIndex":0`) {
		t.Fatalf("request body %s does not contain explicit zero columnIndex", body)
	}
}

func TestDatasetCreateOrUpdateRejectsUnsupportedLabelSetDrift(t *testing.T) {
	t.Parallel()

	existingID := "ocid1.dataset.oc1..existing"
	resource := newExistingDatasetTestResource(existingID)
	resource.Spec.LabelSet.Items = []datalabelingservicev1beta1.DatasetLabelSetItem{
		{Name: "cat"},
		{Name: "dog"},
	}

	client := testDatasetClient(&fakeDatasetOCIClient{
		getFn: func(_ context.Context, req datalabelingservicesdk.GetDatasetRequest) (datalabelingservicesdk.GetDatasetResponse, error) {
			if req.DatasetId == nil || *req.DatasetId != existingID {
				t.Fatalf("get request datasetId = %#v, want %q", req.DatasetId, existingID)
			}
			return datalabelingservicesdk.GetDatasetResponse{
				Dataset: observedDatasetFromSpec(existingID, newDatasetTestResource().Spec, "ACTIVE"),
			}, nil
		},
		updateFn: func(context.Context, datalabelingservicesdk.UpdateDatasetRequest) (datalabelingservicesdk.UpdateDatasetResponse, error) {
			t.Fatal("UpdateDataset should not be called when labelSet drift is out of scope")
			return datalabelingservicesdk.UpdateDatasetResponse{}, nil
		},
	})

	_, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want unsupported labelSet drift failure")
	}
	if !strings.Contains(err.Error(), "labelSet") {
		t.Fatalf("CreateOrUpdate() error = %v, want labelSet drift detail", err)
	}
}

func datasetSerializedRequestBody(
	t *testing.T,
	request datasetRequestBodyBuilder,
	method string,
	path string,
) string {
	t.Helper()

	httpRequest, err := request.HTTPRequest(method, path, nil, nil)
	if err != nil {
		t.Fatalf("HTTPRequest() error = %v", err)
	}

	body, err := io.ReadAll(httpRequest.Body)
	if err != nil {
		t.Fatalf("ReadAll(request.Body) error = %v", err)
	}

	return string(body)
}

func newDatasetTestResource() *datalabelingservicev1beta1.Dataset {
	return &datalabelingservicev1beta1.Dataset{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dataset-sample",
			Namespace: "default",
		},
		Spec: datalabelingservicev1beta1.DatasetSpec{
			CompartmentId:    "ocid1.compartment.oc1..example",
			AnnotationFormat: "SINGLE_LABEL",
			DatasetSourceDetails: datalabelingservicev1beta1.DatasetCreateSourceDetails{
				SourceType: "OBJECT_STORAGE",
				Namespace:  "datasetns",
				Bucket:     "dataset-bucket",
				Prefix:     "records/",
			},
			DatasetFormatDetails: datalabelingservicev1beta1.DatasetCreateFormatDetails{
				FormatType: "TEXT",
				TextFileTypeMetadata: &datalabelingservicev1beta1.DatasetCreateTextFileTypeMetadata{
					FormatType:      "DELIMITED",
					ColumnIndex:     ptr(2),
					ColumnName:      "text",
					ColumnDelimiter: ",",
					LineDelimiter:   "\n",
					EscapeCharacter: "\\",
				},
			},
			LabelSet: datalabelingservicev1beta1.DatasetLabelSet{
				Items: []datalabelingservicev1beta1.DatasetLabelSetItem{
					{Name: "cat"},
				},
			},
			DisplayName:          "dataset-alpha",
			Description:          "dataset description",
			LabelingInstructions: "label carefully",
			FreeformTags: map[string]string{
				"env": "test",
			},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {
					"CostCenter": "42",
				},
			},
			InitialRecordGenerationConfiguration: datalabelingservicev1beta1.DatasetInitialRecordGenerationConfiguration{
				Limit: 25,
			},
			InitialImportDatasetConfiguration: datalabelingservicev1beta1.DatasetCreateInitialImportDatasetConfiguration{
				ImportFormat: datalabelingservicev1beta1.DatasetInitialImportDatasetConfigurationImportFormat{
					Name: "JSONL_CONSOLIDATED",
				},
				ImportMetadataPath: datalabelingservicev1beta1.DatasetCreateImportMetadataPath{
					SourceType: "OBJECT_STORAGE",
					Namespace:  "datasetns",
					Bucket:     "dataset-bucket",
					Path:       "imports/preannotated.jsonl",
				},
			},
		},
	}
}

func newExistingDatasetTestResource(existingID string) *datalabelingservicev1beta1.Dataset {
	resource := newDatasetTestResource()
	resource.Status = datalabelingservicev1beta1.DatasetStatus{
		OsokStatus: shared.OSOKStatus{
			Ocid: shared.OCID(existingID),
		},
		Id: existingID,
	}
	return resource
}

func observedDatasetFromSpec(
	id string,
	spec datalabelingservicev1beta1.DatasetSpec,
	lifecycleState string,
) datalabelingservicesdk.Dataset {
	now := &common.SDKTime{Time: time.Unix(1713240000, 0).UTC()}

	return datalabelingservicesdk.Dataset{
		Id:                                   common.String(id),
		CompartmentId:                        common.String(spec.CompartmentId),
		TimeCreated:                          now,
		TimeUpdated:                          now,
		LifecycleState:                       datalabelingservicesdk.DatasetLifecycleStateEnum(lifecycleState),
		AnnotationFormat:                     common.String(spec.AnnotationFormat),
		DatasetSourceDetails:                 observedDatasetSourceDetails(spec.DatasetSourceDetails),
		DatasetFormatDetails:                 observedDatasetFormatDetails(spec.DatasetFormatDetails),
		LabelSet:                             observedDatasetLabelSet(spec.LabelSet),
		DisplayName:                          pointerOrNil(spec.DisplayName),
		Description:                          pointerOrNil(spec.Description),
		InitialRecordGenerationConfiguration: observedDatasetInitialRecordGenerationConfiguration(spec.InitialRecordGenerationConfiguration),
		InitialImportDatasetConfiguration:    observedDatasetInitialImportDatasetConfiguration(spec.InitialImportDatasetConfiguration),
		LabelingInstructions:                 pointerOrNil(spec.LabelingInstructions),
		FreeformTags:                         cloneStringMap(spec.FreeformTags),
		DefinedTags:                          datasetDefinedTagsFromSpec(spec.DefinedTags),
		SystemTags:                           datasetSystemTagsForTest(),
	}
}

func observedDatasetSummaryFromSpec(
	id string,
	spec datalabelingservicev1beta1.DatasetSpec,
	lifecycleState string,
) datalabelingservicesdk.DatasetSummary {
	now := &common.SDKTime{Time: time.Unix(1713240000, 0).UTC()}

	return datalabelingservicesdk.DatasetSummary{
		Id:                   common.String(id),
		CompartmentId:        common.String(spec.CompartmentId),
		TimeCreated:          now,
		TimeUpdated:          now,
		LifecycleState:       datalabelingservicesdk.DatasetLifecycleStateEnum(lifecycleState),
		AnnotationFormat:     common.String(spec.AnnotationFormat),
		DatasetFormatDetails: observedDatasetFormatDetails(spec.DatasetFormatDetails),
		DisplayName:          pointerOrNil(spec.DisplayName),
		LifecycleDetails:     common.String("summary details"),
		FreeformTags:         cloneStringMap(spec.FreeformTags),
		DefinedTags:          datasetDefinedTagsFromSpec(spec.DefinedTags),
		SystemTags:           datasetSystemTagsForTest(),
	}
}

func observedDatasetSourceDetails(
	spec datalabelingservicev1beta1.DatasetCreateSourceDetails,
) datalabelingservicesdk.DatasetSourceDetails {
	if spec.SourceType != "OBJECT_STORAGE" {
		return nil
	}
	return datalabelingservicesdk.ObjectStorageSourceDetails{
		Namespace: common.String(spec.Namespace),
		Bucket:    common.String(spec.Bucket),
		Prefix:    pointerOrNil(spec.Prefix),
	}
}

func observedDatasetFormatDetails(
	spec datalabelingservicev1beta1.DatasetCreateFormatDetails,
) datalabelingservicesdk.DatasetFormatDetails {
	switch spec.FormatType {
	case "DOCUMENT":
		return datalabelingservicesdk.DocumentDatasetFormatDetails{}
	case "IMAGE":
		return datalabelingservicesdk.ImageDatasetFormatDetails{}
	case "TEXT":
		return datalabelingservicesdk.TextDatasetFormatDetails{
			TextFileTypeMetadata: observedDatasetTextFileTypeMetadata(spec.TextFileTypeMetadata),
		}
	default:
		return nil
	}
}

func observedDatasetTextFileTypeMetadata(
	spec *datalabelingservicev1beta1.DatasetCreateTextFileTypeMetadata,
) datalabelingservicesdk.TextFileTypeMetadata {
	if spec == nil {
		return nil
	}
	if spec.FormatType != "DELIMITED" {
		return nil
	}
	return datalabelingservicesdk.DelimitedFileTypeMetadata{
		ColumnIndex:     spec.ColumnIndex,
		ColumnName:      pointerOrNil(spec.ColumnName),
		ColumnDelimiter: pointerOrNil(spec.ColumnDelimiter),
		LineDelimiter:   pointerOrNil(spec.LineDelimiter),
		EscapeCharacter: pointerOrNil(spec.EscapeCharacter),
	}
}

func observedDatasetLabelSet(
	spec datalabelingservicev1beta1.DatasetLabelSet,
) *datalabelingservicesdk.LabelSet {
	items := make([]datalabelingservicesdk.Label, 0, len(spec.Items))
	for _, item := range spec.Items {
		items = append(items, datalabelingservicesdk.Label{Name: pointerOrNil(item.Name)})
	}
	return &datalabelingservicesdk.LabelSet{Items: items}
}

func observedDatasetInitialRecordGenerationConfiguration(
	spec datalabelingservicev1beta1.DatasetInitialRecordGenerationConfiguration,
) *datalabelingservicesdk.InitialRecordGenerationConfiguration {
	if spec.Limit == 0 {
		return nil
	}
	return &datalabelingservicesdk.InitialRecordGenerationConfiguration{
		Limit: ptr(spec.Limit),
	}
}

func observedDatasetInitialImportDatasetConfiguration(
	spec datalabelingservicev1beta1.DatasetCreateInitialImportDatasetConfiguration,
) *datalabelingservicesdk.InitialImportDatasetConfiguration {
	if spec.ImportFormat.Name == "" {
		return nil
	}
	return &datalabelingservicesdk.InitialImportDatasetConfiguration{
		ImportFormat: &datalabelingservicesdk.ImportFormat{
			Name:    datalabelingservicesdk.ImportFormatNameEnum(spec.ImportFormat.Name),
			Version: datalabelingservicesdk.ImportFormatVersionEnum(spec.ImportFormat.Version),
		},
		ImportMetadataPath: observedDatasetImportMetadataPath(spec.ImportMetadataPath),
	}
}

func observedDatasetImportMetadataPath(
	spec datalabelingservicev1beta1.DatasetCreateImportMetadataPath,
) datalabelingservicesdk.ImportMetadataPath {
	if spec.SourceType != "OBJECT_STORAGE" {
		return nil
	}
	return datalabelingservicesdk.ObjectStorageImportMetadataPath{
		Namespace: common.String(spec.Namespace),
		Bucket:    common.String(spec.Bucket),
		Path:      common.String(spec.Path),
	}
}

func datasetSystemTagsForTest() map[string]map[string]interface{} {
	return map[string]map[string]interface{}{
		"orcl-cloud": {
			"key": "value",
		},
	}
}

func requireDatasetTrailingCondition(
	t *testing.T,
	resource *datalabelingservicev1beta1.Dataset,
	want shared.OSOKConditionType,
) {
	t.Helper()

	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatal("status.conditions = empty, want trailing condition")
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("trailing condition = %q, want %q", got, want)
	}
}

func pointerOrNil(value string) *string {
	if value == "" {
		return nil
	}
	return common.String(value)
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func ptr[T any](value T) *T {
	return &value
}

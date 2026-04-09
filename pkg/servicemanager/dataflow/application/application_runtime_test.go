/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package application

import (
	"context"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	dataflowsdk "github.com/oracle/oci-go-sdk/v65/dataflow"
	dataflowv1beta1 "github.com/oracle/oci-service-operator/api/dataflow/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeApplicationOCIClient struct {
	createFn func(context.Context, dataflowsdk.CreateApplicationRequest) (dataflowsdk.CreateApplicationResponse, error)
	getFn    func(context.Context, dataflowsdk.GetApplicationRequest) (dataflowsdk.GetApplicationResponse, error)
	updateFn func(context.Context, dataflowsdk.UpdateApplicationRequest) (dataflowsdk.UpdateApplicationResponse, error)
	deleteFn func(context.Context, dataflowsdk.DeleteApplicationRequest) (dataflowsdk.DeleteApplicationResponse, error)
}

func (f *fakeApplicationOCIClient) CreateApplication(ctx context.Context, req dataflowsdk.CreateApplicationRequest) (dataflowsdk.CreateApplicationResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return dataflowsdk.CreateApplicationResponse{}, nil
}

func (f *fakeApplicationOCIClient) GetApplication(ctx context.Context, req dataflowsdk.GetApplicationRequest) (dataflowsdk.GetApplicationResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return dataflowsdk.GetApplicationResponse{}, nil
}

func (f *fakeApplicationOCIClient) UpdateApplication(ctx context.Context, req dataflowsdk.UpdateApplicationRequest) (dataflowsdk.UpdateApplicationResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return dataflowsdk.UpdateApplicationResponse{}, nil
}

func (f *fakeApplicationOCIClient) DeleteApplication(ctx context.Context, req dataflowsdk.DeleteApplicationRequest) (dataflowsdk.DeleteApplicationResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return dataflowsdk.DeleteApplicationResponse{}, nil
}

type fakeApplicationServiceError struct {
	statusCode int
	code       string
	message    string
}

func (f fakeApplicationServiceError) Error() string          { return f.message }
func (f fakeApplicationServiceError) GetHTTPStatusCode() int { return f.statusCode }
func (f fakeApplicationServiceError) GetMessage() string     { return f.message }
func (f fakeApplicationServiceError) GetCode() string        { return f.code }
func (f fakeApplicationServiceError) GetOpcRequestID() string {
	return ""
}

func newTestManager(client applicationOCIClient) *ApplicationServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	manager := NewApplicationServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), nil, nil, log, nil)
	if client != nil {
		manager.WithClient(newApplicationServiceClientWithOCIClient(manager, client, nil))
	}
	return manager
}

func makeSpecApplication() *dataflowv1beta1.Application {
	return &dataflowv1beta1.Application{
		Spec: dataflowv1beta1.ApplicationSpec{
			CompartmentId: "ocid1.compartment.oc1..example",
			DisplayName:   "test-application",
			DriverShape:   "VM.Standard.E4.Flex",
			ExecutorShape: "VM.Standard.E4.Flex",
			Language:      "PYTHON",
			NumExecutors:  2,
			SparkVersion:  "3.5.0",
			Type:          "BATCH",
			FileUri:       "oci://bucket@app/main.py",
		},
	}
}

func makeSDKApplication(id, displayName string, state dataflowsdk.ApplicationLifecycleStateEnum) dataflowsdk.Application {
	created := common.SDKTime{Time: time.Date(2026, 4, 3, 20, 0, 0, 0, time.UTC)}
	updated := common.SDKTime{Time: time.Date(2026, 4, 3, 20, 1, 0, 0, time.UTC)}
	return dataflowsdk.Application{
		Id:                   common.String(id),
		CompartmentId:        common.String("ocid1.compartment.oc1..example"),
		DisplayName:          common.String(displayName),
		DriverShape:          common.String("VM.Standard.E4.Flex"),
		ExecutorShape:        common.String("VM.Standard.E4.Flex"),
		FileUri:              common.String("oci://bucket@app/main.py"),
		Language:             dataflowsdk.ApplicationLanguagePython,
		LifecycleState:       state,
		NumExecutors:         common.Int(2),
		OwnerPrincipalId:     common.String("ocid1.user.oc1..owner"),
		SparkVersion:         common.String("3.5.0"),
		TimeCreated:          &created,
		TimeUpdated:          &updated,
		Type:                 dataflowsdk.ApplicationTypeBatch,
		PoolId:               common.String("ocid1.pool.oc1..example"),
		PrivateEndpointId:    common.String("ocid1.privateendpoint.oc1..example"),
		MaxDurationInMinutes: common.Int64(60),
		IdleTimeoutInMinutes: common.Int64(30),
		FreeformTags:         map[string]string{"env": "dev"},
		DefinedTags: map[string]map[string]interface{}{
			"Operations": {"CostCenter": "42"},
		},
	}
}

func TestBuildCreateApplicationDetails_RequiredAndOptionalFields(t *testing.T) {
	spec := makeSpecApplication().Spec
	spec.ArchiveUri = "oci://bucket@app/deps.zip"
	spec.Arguments = []string{"--input", "oci://bucket@app/input.csv"}
	spec.ApplicationLogConfig = dataflowv1beta1.ApplicationLogConfig{
		LogGroupId: "ocid1.loggroup.oc1..example",
		LogId:      "ocid1.log.oc1..example",
	}
	spec.ClassName = "com.example.Main"
	spec.Configuration = map[string]string{"spark.app.name": "demo"}
	spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "42"}}
	spec.Description = "application description"
	spec.DriverShapeConfig = dataflowv1beta1.ApplicationDriverShapeConfig{Ocpus: 1, MemoryInGBs: 16}
	spec.ExecutorShapeConfig = dataflowv1beta1.ApplicationExecutorShapeConfig{Ocpus: 2, MemoryInGBs: 32}
	spec.FreeformTags = map[string]string{"env": "dev"}
	spec.LogsBucketUri = "oci://bucket@app/logs/"
	spec.MetastoreId = "ocid1.metastore.oc1..example"
	spec.Parameters = []dataflowv1beta1.ApplicationParameter{{Name: "input_file", Value: "data.csv"}}
	spec.PoolId = "ocid1.pool.oc1..example"
	spec.PrivateEndpointId = "ocid1.privateendpoint.oc1..example"
	spec.WarehouseBucketUri = "oci://bucket@app/warehouse/"
	spec.MaxDurationInMinutes = 90
	spec.IdleTimeoutInMinutes = 45

	details, err := buildCreateApplicationDetails(spec)

	assert.NoError(t, err)
	assert.Equal(t, common.String("ocid1.compartment.oc1..example"), details.CompartmentId)
	assert.Equal(t, common.String("test-application"), details.DisplayName)
	assert.Equal(t, dataflowsdk.ApplicationLanguagePython, details.Language)
	assert.Equal(t, dataflowsdk.ApplicationTypeBatch, details.Type)
	assert.Equal(t, []string{"--input", "oci://bucket@app/input.csv"}, details.Arguments)
	assert.Equal(t, "ocid1.loggroup.oc1..example", *details.ApplicationLogConfig.LogGroupId)
	assert.Equal(t, "com.example.Main", *details.ClassName)
	assert.Equal(t, map[string]string{"spark.app.name": "demo"}, details.Configuration)
	assert.Equal(t, "42", details.DefinedTags["Operations"]["CostCenter"])
	assert.Equal(t, float32(1), *details.DriverShapeConfig.Ocpus)
	assert.Equal(t, float32(32), *details.ExecutorShapeConfig.MemoryInGBs)
	assert.Equal(t, map[string]string{"env": "dev"}, details.FreeformTags)
	assert.Equal(t, "ocid1.privateendpoint.oc1..example", *details.PrivateEndpointId)
	assert.Equal(t, int64(90), *details.MaxDurationInMinutes)
	assert.Equal(t, int64(45), *details.IdleTimeoutInMinutes)
}

func TestBuildCreateApplicationDetails_OmitsZeroValueOptionalNestedFields(t *testing.T) {
	details, err := buildCreateApplicationDetails(makeSpecApplication().Spec)

	assert.NoError(t, err)
	assert.Nil(t, details.ApplicationLogConfig)
	assert.Nil(t, details.DriverShapeConfig)
	assert.Nil(t, details.ExecutorShapeConfig)
}

func TestBuildCreateApplicationDetails_ExecutePrecedenceOmitsSubordinateFields(t *testing.T) {
	spec := makeSpecApplication().Spec
	spec.Execute = "--class com.example.Main oci://bucket@app/main.jar 10"
	spec.ClassName = "com.example.Other"
	spec.FileUri = "oci://bucket@app/other.jar"
	spec.Arguments = []string{"--ignored"}
	spec.Configuration = map[string]string{"spark.app.name": "ignored"}
	spec.Parameters = []dataflowv1beta1.ApplicationParameter{{Name: "ignored", Value: "value"}}

	details, err := buildCreateApplicationDetails(spec)

	assert.NoError(t, err)
	assert.Equal(t, spec.Execute, *details.Execute)
	assert.Nil(t, details.ClassName)
	assert.Nil(t, details.FileUri)
	assert.Nil(t, details.Arguments)
	assert.Nil(t, details.Configuration)
	assert.Nil(t, details.Parameters)
}

func TestBuildCreateApplicationDetails_RejectsInvalidEnums(t *testing.T) {
	spec := makeSpecApplication().Spec
	spec.Language = "RUST"

	_, err := buildCreateApplicationDetails(spec)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported Application language")

	spec = makeSpecApplication().Spec
	spec.Type = "PIPELINE"

	_, err = buildCreateApplicationDetails(spec)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported Application type")
}

func TestCreateOrUpdate_CreateSuccessAndStatusProjection(t *testing.T) {
	var captured dataflowsdk.CreateApplicationRequest
	manager := newTestManager(&fakeApplicationOCIClient{
		createFn: func(_ context.Context, req dataflowsdk.CreateApplicationRequest) (dataflowsdk.CreateApplicationResponse, error) {
			captured = req
			app := makeSDKApplication("ocid1.dataflowapplication.oc1..create", "test-application", dataflowsdk.ApplicationLifecycleStateActive)
			app.ApplicationLogConfig = &dataflowsdk.ApplicationLogConfig{
				LogGroupId: common.String("ocid1.loggroup.oc1..example"),
				LogId:      common.String("ocid1.log.oc1..example"),
			}
			app.DriverShapeConfig = &dataflowsdk.ShapeConfig{Ocpus: common.Float32(1), MemoryInGBs: common.Float32(16)}
			app.ExecutorShapeConfig = &dataflowsdk.ShapeConfig{Ocpus: common.Float32(2), MemoryInGBs: common.Float32(32)}
			app.ArchiveUri = common.String("oci://bucket@app/deps.zip")
			app.ClassName = common.String("com.example.Main")
			app.Configuration = map[string]string{"spark.app.name": "demo"}
			app.Description = common.String("application description")
			app.LogsBucketUri = common.String("oci://bucket@app/logs/")
			app.MetastoreId = common.String("ocid1.metastore.oc1..example")
			app.OwnerUserName = common.String("example-user")
			app.Parameters = []dataflowsdk.ApplicationParameter{{Name: common.String("input_file"), Value: common.String("data.csv")}}
			app.WarehouseBucketUri = common.String("oci://bucket@app/warehouse/")
			return dataflowsdk.CreateApplicationResponse{Application: app}, nil
		},
	})

	resource := makeSpecApplication()
	resource.Spec.ArchiveUri = "oci://bucket@app/deps.zip"
	resource.Spec.ApplicationLogConfig = dataflowv1beta1.ApplicationLogConfig{LogGroupId: "ocid1.loggroup.oc1..example", LogId: "ocid1.log.oc1..example"}
	resource.Spec.ClassName = "com.example.Main"
	resource.Spec.Configuration = map[string]string{"spark.app.name": "demo"}
	resource.Spec.Description = "application description"
	resource.Spec.DriverShapeConfig = dataflowv1beta1.ApplicationDriverShapeConfig{Ocpus: 1, MemoryInGBs: 16}
	resource.Spec.ExecutorShapeConfig = dataflowv1beta1.ApplicationExecutorShapeConfig{Ocpus: 2, MemoryInGBs: 32}
	resource.Spec.FreeformTags = map[string]string{"env": "dev"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "42"}}
	resource.Spec.LogsBucketUri = "oci://bucket@app/logs/"
	resource.Spec.MetastoreId = "ocid1.metastore.oc1..example"
	resource.Spec.Parameters = []dataflowv1beta1.ApplicationParameter{{Name: "input_file", Value: "data.csv"}}
	resource.Spec.PoolId = "ocid1.pool.oc1..example"
	resource.Spec.PrivateEndpointId = "ocid1.privateendpoint.oc1..example"
	resource.Spec.WarehouseBucketUri = "oci://bucket@app/warehouse/"
	resource.Spec.MaxDurationInMinutes = 60
	resource.Spec.IdleTimeoutInMinutes = 30

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, resp.ShouldRequeue)
	assert.Equal(t, common.String("ocid1.compartment.oc1..example"), captured.CompartmentId)
	assert.Equal(t, common.String("test-application"), captured.DisplayName)
	assert.Equal(t, dataflowsdk.ApplicationLanguagePython, captured.Language)
	assert.Equal(t, dataflowsdk.ApplicationTypeBatch, captured.Type)
	assert.Equal(t, "ocid1.dataflowapplication.oc1..create", string(resource.Status.OsokStatus.Ocid))
	assert.Equal(t, "ACTIVE", resource.Status.LifecycleState)
	assert.Equal(t, "test-application", resource.Status.DisplayName)
	assert.Equal(t, "com.example.Main", resource.Status.ClassName)
	assert.Equal(t, "ocid1.privateendpoint.oc1..example", resource.Status.PrivateEndpointId)
	assert.Equal(t, "example-user", resource.Status.OwnerUserName)
	assert.Equal(t, int64(60), resource.Status.MaxDurationInMinutes)
}

func TestCreateOrUpdate_UpdateMutableFields(t *testing.T) {
	var captured dataflowsdk.UpdateApplicationRequest
	manager := newTestManager(&fakeApplicationOCIClient{
		getFn: func(_ context.Context, _ dataflowsdk.GetApplicationRequest) (dataflowsdk.GetApplicationResponse, error) {
			app := makeSDKApplication("ocid1.dataflowapplication.oc1..existing", "old-name", dataflowsdk.ApplicationLifecycleStateActive)
			app.DriverShape = common.String("VM.Standard3.Flex")
			app.ExecutorShape = common.String("VM.Standard3.Flex")
			app.NumExecutors = common.Int(1)
			app.FileUri = common.String("oci://bucket@app/old.py")
			app.PoolId = common.String("ocid1.pool.oc1..old")
			app.PrivateEndpointId = common.String("ocid1.privateendpoint.oc1..old")
			app.FreeformTags = map[string]string{"env": "old"}
			app.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "7"}}
			return dataflowsdk.GetApplicationResponse{Application: app}, nil
		},
		updateFn: func(_ context.Context, req dataflowsdk.UpdateApplicationRequest) (dataflowsdk.UpdateApplicationResponse, error) {
			captured = req
			app := makeSDKApplication("ocid1.dataflowapplication.oc1..existing", "new-name", dataflowsdk.ApplicationLifecycleStateActive)
			app.DriverShape = common.String("VM.Standard.E5.Flex")
			app.ExecutorShape = common.String("VM.Standard.E5.Flex")
			app.NumExecutors = common.Int(4)
			app.FileUri = common.String("oci://bucket@app/new.py")
			app.PoolId = common.String("ocid1.pool.oc1..new")
			app.PrivateEndpointId = common.String("ocid1.privateendpoint.oc1..new")
			app.FreeformTags = map[string]string{"env": "dev"}
			app.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}}
			return dataflowsdk.UpdateApplicationResponse{Application: app}, nil
		},
	})

	resource := makeSpecApplication()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.dataflowapplication.oc1..existing")
	resource.Spec.DisplayName = "new-name"
	resource.Spec.DriverShape = "VM.Standard.E5.Flex"
	resource.Spec.ExecutorShape = "VM.Standard.E5.Flex"
	resource.Spec.NumExecutors = 4
	resource.Spec.FileUri = "oci://bucket@app/new.py"
	resource.Spec.FreeformTags = map[string]string{"env": "dev"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "42"}}
	resource.Spec.PoolId = "ocid1.pool.oc1..new"
	resource.Spec.PrivateEndpointId = "ocid1.privateendpoint.oc1..new"

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, "ocid1.dataflowapplication.oc1..existing", *captured.ApplicationId)
	assert.Equal(t, "new-name", *captured.DisplayName)
	assert.Equal(t, "VM.Standard.E5.Flex", *captured.DriverShape)
	assert.Equal(t, "VM.Standard.E5.Flex", *captured.ExecutorShape)
	assert.Equal(t, 4, *captured.NumExecutors)
	assert.Equal(t, "oci://bucket@app/new.py", *captured.FileUri)
	assert.Equal(t, "ocid1.pool.oc1..new", *captured.PoolId)
	assert.Equal(t, "ocid1.privateendpoint.oc1..new", *captured.PrivateEndpointId)
	assert.Equal(t, map[string]string{"env": "dev"}, captured.FreeformTags)
	assert.Equal(t, "new-name", resource.Status.DisplayName)
}

func TestCreateOrUpdate_RejectsCreateOnlyDrift(t *testing.T) {
	updateCalls := 0
	manager := newTestManager(&fakeApplicationOCIClient{
		getFn: func(_ context.Context, _ dataflowsdk.GetApplicationRequest) (dataflowsdk.GetApplicationResponse, error) {
			app := makeSDKApplication("ocid1.dataflowapplication.oc1..existing", "test-application", dataflowsdk.ApplicationLifecycleStateActive)
			app.CompartmentId = common.String("ocid1.compartment.oc1..different")
			app.Type = dataflowsdk.ApplicationTypeStreaming
			return dataflowsdk.GetApplicationResponse{Application: app}, nil
		},
		updateFn: func(_ context.Context, _ dataflowsdk.UpdateApplicationRequest) (dataflowsdk.UpdateApplicationResponse, error) {
			updateCalls++
			return dataflowsdk.UpdateApplicationResponse{}, nil
		},
	})

	resource := makeSpecApplication()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.dataflowapplication.oc1..existing")

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.Contains(t, err.Error(), "create-only field drift")
	assert.Contains(t, err.Error(), "compartmentId")
	assert.Contains(t, err.Error(), "type")
	assert.Equal(t, 0, updateCalls)
}

func TestCreateOrUpdate_ExecutePrecedenceSkipsSubordinateDrift(t *testing.T) {
	updateCalls := 0
	manager := newTestManager(&fakeApplicationOCIClient{
		getFn: func(_ context.Context, _ dataflowsdk.GetApplicationRequest) (dataflowsdk.GetApplicationResponse, error) {
			app := makeSDKApplication("ocid1.dataflowapplication.oc1..existing", "test-application", dataflowsdk.ApplicationLifecycleStateActive)
			app.Execute = common.String("--class com.example.Main oci://bucket@app/main.jar 10")
			app.ClassName = common.String("server-derived-class")
			app.FileUri = common.String("oci://bucket@app/derived.jar")
			app.Arguments = []string{"server-derived"}
			app.Configuration = map[string]string{"spark.app.name": "server-derived"}
			app.Parameters = []dataflowsdk.ApplicationParameter{{Name: common.String("server"), Value: common.String("derived")}}
			return dataflowsdk.GetApplicationResponse{Application: app}, nil
		},
		updateFn: func(_ context.Context, _ dataflowsdk.UpdateApplicationRequest) (dataflowsdk.UpdateApplicationResponse, error) {
			updateCalls++
			return dataflowsdk.UpdateApplicationResponse{}, nil
		},
	})

	resource := makeSpecApplication()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.dataflowapplication.oc1..existing")
	resource.Spec.Execute = "--class com.example.Main oci://bucket@app/main.jar 10"
	resource.Spec.ClassName = "spec-class"
	resource.Spec.FileUri = "oci://bucket@app/spec.py"
	resource.Spec.Arguments = []string{"spec-arg"}
	resource.Spec.Configuration = map[string]string{"spark.app.name": "spec"}
	resource.Spec.Parameters = []dataflowv1beta1.ApplicationParameter{{Name: "spec", Value: "value"}}

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, 0, updateCalls)
}

func TestCreateOrUpdate_ClearsStatusFieldsOmittedByUpdateResponse(t *testing.T) {
	var captured dataflowsdk.UpdateApplicationRequest
	manager := newTestManager(&fakeApplicationOCIClient{
		getFn: func(_ context.Context, _ dataflowsdk.GetApplicationRequest) (dataflowsdk.GetApplicationResponse, error) {
			app := makeSDKApplication("ocid1.dataflowapplication.oc1..existing", "old-name", dataflowsdk.ApplicationLifecycleStateActive)
			app.ClassName = common.String("com.example.Stale")
			app.Configuration = map[string]string{"spark.app.name": "stale"}
			app.LogsBucketUri = common.String("oci://bucket@app/stale-logs/")
			app.Parameters = []dataflowsdk.ApplicationParameter{{Name: common.String("stale"), Value: common.String("value")}}
			return dataflowsdk.GetApplicationResponse{Application: app}, nil
		},
		updateFn: func(_ context.Context, req dataflowsdk.UpdateApplicationRequest) (dataflowsdk.UpdateApplicationResponse, error) {
			captured = req
			app := makeSDKApplication("ocid1.dataflowapplication.oc1..existing", "new-name", dataflowsdk.ApplicationLifecycleStateActive)
			return dataflowsdk.UpdateApplicationResponse{Application: app}, nil
		},
	})

	resource := makeSpecApplication()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.dataflowapplication.oc1..existing")
	resource.Spec.DisplayName = "new-name"

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, "new-name", *captured.DisplayName)
	assert.Equal(t, "new-name", resource.Status.DisplayName)
	assert.Empty(t, resource.Status.ClassName)
	assert.Nil(t, resource.Status.Configuration)
	assert.Empty(t, resource.Status.LogsBucketUri)
	assert.Nil(t, resource.Status.Parameters)
}

func TestCreateOrUpdate_RecreatesOnExplicitNotFound(t *testing.T) {
	getCalls := 0
	createCalls := 0
	manager := newTestManager(&fakeApplicationOCIClient{
		getFn: func(_ context.Context, _ dataflowsdk.GetApplicationRequest) (dataflowsdk.GetApplicationResponse, error) {
			getCalls++
			return dataflowsdk.GetApplicationResponse{}, fakeApplicationServiceError{
				statusCode: 404,
				code:       "NotFound",
				message:    "missing",
			}
		},
		createFn: func(_ context.Context, req dataflowsdk.CreateApplicationRequest) (dataflowsdk.CreateApplicationResponse, error) {
			createCalls++
			assert.Equal(t, common.String("ocid1.compartment.oc1..example"), req.CompartmentId)
			return dataflowsdk.CreateApplicationResponse{
				Application: makeSDKApplication("ocid1.dataflowapplication.oc1..recreated", "test-application", dataflowsdk.ApplicationLifecycleStateActive),
			}, nil
		},
	})

	resource := makeSpecApplication()
	resource.Status.Id = "ocid1.dataflowapplication.oc1..existing"
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.dataflowapplication.oc1..existing")
	oldCreatedAt := common.SDKTime{Time: time.Now().Add(-time.Hour)}

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, 1, getCalls)
	assert.Equal(t, 1, createCalls)
	assert.Equal(t, "ocid1.dataflowapplication.oc1..recreated", string(resource.Status.OsokStatus.Ocid))
	assert.NotEqual(t, oldCreatedAt.Time.Format(time.RFC3339Nano), resource.Status.TimeCreated)
}

func TestCreateOrUpdate_RecreatesWhenTrackedApplicationTurnsDeleted(t *testing.T) {
	getCalls := 0
	createCalls := 0
	manager := newTestManager(&fakeApplicationOCIClient{
		getFn: func(_ context.Context, _ dataflowsdk.GetApplicationRequest) (dataflowsdk.GetApplicationResponse, error) {
			getCalls++
			if getCalls == 1 {
				return dataflowsdk.GetApplicationResponse{
					Application: makeSDKApplication("ocid1.dataflowapplication.oc1..existing", "test-application", dataflowsdk.ApplicationLifecycleStateActive),
				}, nil
			}
			return dataflowsdk.GetApplicationResponse{
				Application: makeSDKApplication("ocid1.dataflowapplication.oc1..existing", "test-application", dataflowsdk.ApplicationLifecycleStateDeleted),
			}, nil
		},
		createFn: func(_ context.Context, req dataflowsdk.CreateApplicationRequest) (dataflowsdk.CreateApplicationResponse, error) {
			createCalls++
			assert.Equal(t, common.String("ocid1.compartment.oc1..example"), req.CompartmentId)
			return dataflowsdk.CreateApplicationResponse{
				Application: makeSDKApplication("ocid1.dataflowapplication.oc1..recreated", "test-application", dataflowsdk.ApplicationLifecycleStateActive),
			}, nil
		},
		updateFn: func(_ context.Context, _ dataflowsdk.UpdateApplicationRequest) (dataflowsdk.UpdateApplicationResponse, error) {
			t.Fatal("UpdateApplication should not be called once the tracked application turns DELETED")
			return dataflowsdk.UpdateApplicationResponse{}, nil
		},
	})

	resource := makeSpecApplication()
	resource.Status.Id = "ocid1.dataflowapplication.oc1..existing"
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.dataflowapplication.oc1..existing")

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, resp.ShouldRequeue)
	assert.Equal(t, 2, getCalls)
	assert.Equal(t, 1, createCalls)
	assert.Equal(t, "ocid1.dataflowapplication.oc1..recreated", string(resource.Status.OsokStatus.Ocid))
	assert.Equal(t, "ACTIVE", resource.Status.LifecycleState)
}

func TestCreateOrUpdate_TreatsInactiveAsSuccessfulObservedState(t *testing.T) {
	updateCalls := 0
	manager := newTestManager(&fakeApplicationOCIClient{
		getFn: func(_ context.Context, _ dataflowsdk.GetApplicationRequest) (dataflowsdk.GetApplicationResponse, error) {
			return dataflowsdk.GetApplicationResponse{
				Application: makeSDKApplication("ocid1.dataflowapplication.oc1..inactive", "test-application", dataflowsdk.ApplicationLifecycleStateInactive),
			}, nil
		},
		updateFn: func(_ context.Context, _ dataflowsdk.UpdateApplicationRequest) (dataflowsdk.UpdateApplicationResponse, error) {
			updateCalls++
			return dataflowsdk.UpdateApplicationResponse{}, nil
		},
	})

	resource := makeSpecApplication()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.dataflowapplication.oc1..inactive")

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, resp.ShouldRequeue)
	assert.Equal(t, 0, updateCalls)
	assert.Equal(t, "INACTIVE", resource.Status.LifecycleState)
	assert.Equal(t, string(shared.Active), resource.Status.OsokStatus.Reason)
}

func TestDelete_ConfirmsDeletionOnNotFound(t *testing.T) {
	manager := newTestManager(&fakeApplicationOCIClient{
		deleteFn: func(_ context.Context, req dataflowsdk.DeleteApplicationRequest) (dataflowsdk.DeleteApplicationResponse, error) {
			assert.Equal(t, "ocid1.dataflowapplication.oc1..delete", *req.ApplicationId)
			return dataflowsdk.DeleteApplicationResponse{}, nil
		},
		getFn: func(_ context.Context, _ dataflowsdk.GetApplicationRequest) (dataflowsdk.GetApplicationResponse, error) {
			return dataflowsdk.GetApplicationResponse{}, fakeApplicationServiceError{
				statusCode: 404,
				code:       "NotFound",
				message:    "not found",
			}
		},
	})

	resource := makeSpecApplication()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.dataflowapplication.oc1..delete")

	done, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.True(t, done)
	assert.Equal(t, string(shared.Terminating), resource.Status.OsokStatus.Reason)
	assert.NotNil(t, resource.Status.OsokStatus.DeletedAt)
}

func TestDelete_StaysTerminatingWhileOCIStillReportsDeleted(t *testing.T) {
	manager := newTestManager(&fakeApplicationOCIClient{
		deleteFn: func(_ context.Context, req dataflowsdk.DeleteApplicationRequest) (dataflowsdk.DeleteApplicationResponse, error) {
			assert.Equal(t, "ocid1.dataflowapplication.oc1..delete", *req.ApplicationId)
			return dataflowsdk.DeleteApplicationResponse{}, nil
		},
		getFn: func(_ context.Context, _ dataflowsdk.GetApplicationRequest) (dataflowsdk.GetApplicationResponse, error) {
			return dataflowsdk.GetApplicationResponse{
				Application: makeSDKApplication("ocid1.dataflowapplication.oc1..delete", "test-application", dataflowsdk.ApplicationLifecycleStateDeleted),
			}, nil
		},
	})

	resource := makeSpecApplication()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.dataflowapplication.oc1..delete")

	done, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.False(t, done)
	assert.Equal(t, "DELETED", resource.Status.LifecycleState)
	assert.Equal(t, string(shared.Terminating), resource.Status.OsokStatus.Reason)
	assert.Nil(t, resource.Status.OsokStatus.DeletedAt)
}

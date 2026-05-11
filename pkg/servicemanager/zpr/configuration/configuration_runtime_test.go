package configuration

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	zprsdk "github.com/oracle/oci-go-sdk/v65/zpr"
	zprv1beta1 "github.com/oracle/oci-service-operator/api/zpr/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

const testConfigurationID = "ocid1.zprconfiguration.oc1..runtime"

type fakeConfigurationOCIClient struct {
	createFn         func(context.Context, zprsdk.CreateConfigurationRequest) (zprsdk.CreateConfigurationResponse, error)
	getFn            func(context.Context, zprsdk.GetConfigurationRequest) (zprsdk.GetConfigurationResponse, error)
	getWorkRequestFn func(context.Context, zprsdk.GetZprConfigurationWorkRequestRequest) (zprsdk.GetZprConfigurationWorkRequestResponse, error)

	createRequests         []zprsdk.CreateConfigurationRequest
	getRequests            []zprsdk.GetConfigurationRequest
	getWorkRequestRequests []zprsdk.GetZprConfigurationWorkRequestRequest
}

func (f *fakeConfigurationOCIClient) CreateConfiguration(ctx context.Context, req zprsdk.CreateConfigurationRequest) (zprsdk.CreateConfigurationResponse, error) {
	f.createRequests = append(f.createRequests, req)
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return zprsdk.CreateConfigurationResponse{}, nil
}

func (f *fakeConfigurationOCIClient) GetConfiguration(ctx context.Context, req zprsdk.GetConfigurationRequest) (zprsdk.GetConfigurationResponse, error) {
	f.getRequests = append(f.getRequests, req)
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return zprsdk.GetConfigurationResponse{}, nil
}

func (f *fakeConfigurationOCIClient) GetZprConfigurationWorkRequest(ctx context.Context, req zprsdk.GetZprConfigurationWorkRequestRequest) (zprsdk.GetZprConfigurationWorkRequestResponse, error) {
	f.getWorkRequestRequests = append(f.getWorkRequestRequests, req)
	if f.getWorkRequestFn != nil {
		return f.getWorkRequestFn(ctx, req)
	}
	return zprsdk.GetZprConfigurationWorkRequestResponse{}, nil
}

func newConfigurationTestClient(client configurationOCIClient) ConfigurationServiceClient {
	if client == nil {
		client = &fakeConfigurationOCIClient{}
	}

	hooks := ConfigurationRuntimeHooks{
		Semantics:       newConfigurationRuntimeSemantics(),
		Identity:        generatedruntime.IdentityHooks[*zprv1beta1.Configuration]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*zprv1beta1.Configuration]{},
		StatusHooks:     generatedruntime.StatusHooks[*zprv1beta1.Configuration]{},
		ParityHooks:     generatedruntime.ParityHooks[*zprv1beta1.Configuration]{},
		Async:           generatedruntime.AsyncHooks[*zprv1beta1.Configuration]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*zprv1beta1.Configuration]{},
		Create: runtimeOperationHooks[zprsdk.CreateConfigurationRequest, zprsdk.CreateConfigurationResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "CreateConfigurationDetails", RequestName: "CreateConfigurationDetails", Contribution: "body"},
			},
			Call: func(ctx context.Context, request zprsdk.CreateConfigurationRequest) (zprsdk.CreateConfigurationResponse, error) {
				return client.CreateConfiguration(ctx, request)
			},
		},
		Get: runtimeOperationHooks[zprsdk.GetConfigurationRequest, zprsdk.GetConfigurationResponse]{
			Fields: configurationGetFields(),
			Call: func(ctx context.Context, request zprsdk.GetConfigurationRequest) (zprsdk.GetConfigurationResponse, error) {
				return client.GetConfiguration(ctx, request)
			},
		},
		WrapGeneratedClient: []func(ConfigurationServiceClient) ConfigurationServiceClient{},
	}
	applyConfigurationRuntimeHooks(&hooks, client, nil)

	delegate := defaultConfigurationServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*zprv1beta1.Configuration](
			buildConfigurationGeneratedRuntimeConfig(
				&ConfigurationServiceManager{Log: loggerutil.OSOKLogger{Logger: logr.Discard()}},
				hooks,
			),
		),
	}
	return wrapConfigurationGeneratedClient(hooks, delegate)
}

func newConfigurationTestResource() *zprv1beta1.Configuration {
	return &zprv1beta1.Configuration{
		Spec: zprv1beta1.ConfigurationSpec{
			CompartmentId: "ocid1.tenancy.oc1..example",
			ZprStatus:     "ENABLED",
			FreeformTags: map[string]string{
				"env": "dev",
			},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func makeSDKConfiguration(id string, state zprsdk.ConfigurationLifecycleStateEnum) zprsdk.Configuration {
	return zprsdk.Configuration{
		Id:               common.String(id),
		CompartmentId:    common.String("ocid1.tenancy.oc1..example"),
		ZprStatus:        zprsdk.ConfigurationZprStatusEnabled,
		TimeCreated:      &common.SDKTime{Time: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)},
		TimeUpdated:      &common.SDKTime{Time: time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC)},
		LifecycleState:   state,
		LifecycleDetails: common.String("ready"),
		FreeformTags:     map[string]string{"env": "dev"},
		DefinedTags:      map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		SystemTags:       map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "true"}},
	}
}

func makeConfigurationWorkRequest(
	id string,
	status zprsdk.WorkRequestStatusEnum,
	operation zprsdk.OperationTypeEnum,
	action zprsdk.ActionTypeEnum,
	configurationID string,
) zprsdk.WorkRequest {
	return zprsdk.WorkRequest{
		Id:            common.String(id),
		Status:        status,
		OperationType: operation,
		CompartmentId: common.String("ocid1.tenancy.oc1..example"),
		Resources: []zprsdk.WorkRequestResource{
			{
				EntityType: common.String("configuration"),
				ActionType: action,
				Identifier: common.String(configurationID),
				EntityUri:  common.String("/configuration"),
			},
		},
		PercentComplete: common.Float32(100),
		TimeAccepted:    &common.SDKTime{Time: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)},
		TimeUpdated:     &common.SDKTime{Time: time.Date(2026, 5, 1, 0, 1, 0, 0, time.UTC)},
	}
}

func TestConfigurationRuntimeCreateFollowsWorkRequestToGetConfiguration(t *testing.T) {
	t.Parallel()

	resource := newConfigurationTestResource()
	fake := &fakeConfigurationOCIClient{}
	fake.createFn = func(_ context.Context, req zprsdk.CreateConfigurationRequest) (zprsdk.CreateConfigurationResponse, error) {
		if got := stringValue(req.CreateConfigurationDetails.CompartmentId); got != resource.Spec.CompartmentId {
			t.Fatalf("CreateConfiguration compartmentId = %q, want %q", got, resource.Spec.CompartmentId)
		}
		if req.CreateConfigurationDetails.ZprStatus != zprsdk.ConfigurationZprStatusEnabled {
			t.Fatalf("CreateConfiguration zprStatus = %q, want %q", req.CreateConfigurationDetails.ZprStatus, zprsdk.ConfigurationZprStatusEnabled)
		}
		if got := req.CreateConfigurationDetails.FreeformTags["env"]; got != "dev" {
			t.Fatalf("CreateConfiguration freeformTags[env] = %q, want dev", got)
		}
		return zprsdk.CreateConfigurationResponse{
			OpcWorkRequestId: common.String("wr-create-1"),
			OpcRequestId:     common.String("req-create-1"),
		}, nil
	}
	fake.getFn = func(_ context.Context, req zprsdk.GetConfigurationRequest) (zprsdk.GetConfigurationResponse, error) {
		if got := stringValue(req.CompartmentId); got != resource.Spec.CompartmentId {
			t.Fatalf("GetConfiguration compartmentId = %q, want %q", got, resource.Spec.CompartmentId)
		}
		if len(fake.createRequests) == 0 {
			return zprsdk.GetConfigurationResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "not found")
		}
		return zprsdk.GetConfigurationResponse{
			Configuration: makeSDKConfiguration(testConfigurationID, zprsdk.ConfigurationLifecycleStateActive),
			OpcRequestId:  common.String("req-get-1"),
		}, nil
	}
	fake.getWorkRequestFn = func(_ context.Context, req zprsdk.GetZprConfigurationWorkRequestRequest) (zprsdk.GetZprConfigurationWorkRequestResponse, error) {
		if got := stringValue(req.WorkRequestId); got != "wr-create-1" {
			t.Fatalf("GetZprConfigurationWorkRequest workRequestId = %q, want wr-create-1", got)
		}
		return zprsdk.GetZprConfigurationWorkRequestResponse{
			WorkRequest: makeConfigurationWorkRequest(
				"wr-create-1",
				zprsdk.WorkRequestStatusSucceeded,
				zprsdk.OperationTypeCreateZprConfiguration,
				zprsdk.ActionTypeCreated,
				testConfigurationID,
			),
			OpcRequestId: common.String("req-wr-1"),
		}, nil
	}

	client := newConfigurationTestClient(fake)
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = true, want false after successful follow-up read")
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("CreateConfiguration calls = %d, want 1", len(fake.createRequests))
	}
	if len(fake.getWorkRequestRequests) != 1 {
		t.Fatalf("GetZprConfigurationWorkRequest calls = %d, want 1", len(fake.getWorkRequestRequests))
	}
	if len(fake.getRequests) != 2 {
		t.Fatalf("GetConfiguration calls = %d, want 2", len(fake.getRequests))
	}
	if got := resource.Status.Id; got != testConfigurationID {
		t.Fatalf("status.id = %q, want %q", got, testConfigurationID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testConfigurationID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testConfigurationID)
	}
	if got := resource.Status.ZprStatus; got != string(zprsdk.ConfigurationZprStatusEnabled) {
		t.Fatalf("status.zprStatus = %q, want %q", got, zprsdk.ConfigurationZprStatusEnabled)
	}
	if got := resource.Status.LifecycleState; got != string(zprsdk.ConfigurationLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want %q", got, zprsdk.ConfigurationLifecycleStateActive)
	}
}

func TestConfigurationRuntimePendingWorkRequestRequeuesWithoutReadback(t *testing.T) {
	t.Parallel()

	resource := newConfigurationTestResource()
	fake := &fakeConfigurationOCIClient{}
	fake.createFn = func(_ context.Context, _ zprsdk.CreateConfigurationRequest) (zprsdk.CreateConfigurationResponse, error) {
		return zprsdk.CreateConfigurationResponse{
			OpcWorkRequestId: common.String("wr-create-pending"),
			OpcRequestId:     common.String("req-create-pending"),
		}, nil
	}
	fake.getFn = func(_ context.Context, _ zprsdk.GetConfigurationRequest) (zprsdk.GetConfigurationResponse, error) {
		if len(fake.createRequests) == 0 {
			return zprsdk.GetConfigurationResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "not found")
		}
		t.Fatal("GetConfiguration should not be called for readback while the create work request is still pending")
		return zprsdk.GetConfigurationResponse{}, nil
	}
	fake.getWorkRequestFn = func(_ context.Context, _ zprsdk.GetZprConfigurationWorkRequestRequest) (zprsdk.GetZprConfigurationWorkRequestResponse, error) {
		return zprsdk.GetZprConfigurationWorkRequestResponse{
			WorkRequest: makeConfigurationWorkRequest(
				"wr-create-pending",
				zprsdk.WorkRequestStatusInProgress,
				zprsdk.OperationTypeCreateZprConfiguration,
				zprsdk.ActionTypeCreated,
				testConfigurationID,
			),
		}, nil
	}

	client := newConfigurationTestClient(fake)
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should stay successful while the create work request is pending")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = false, want true while the create work request is pending")
	}
	if len(fake.getRequests) != 1 {
		t.Fatalf("GetConfiguration calls = %d, want 1 pre-create singleton lookup only", len(fake.getRequests))
	}
	if current := resource.Status.OsokStatus.Async.Current; current == nil {
		t.Fatal("status.status.async.current = nil, want tracked pending create work request")
	} else {
		if current.Phase != shared.OSOKAsyncPhaseCreate {
			t.Fatalf("status.status.async.current.phase = %q, want %q", current.Phase, shared.OSOKAsyncPhaseCreate)
		}
		if current.WorkRequestID != "wr-create-pending" {
			t.Fatalf("status.status.async.current.workRequestId = %q, want wr-create-pending", current.WorkRequestID)
		}
	}
}

func TestConfigurationRuntimeLookupExistingUsesGetConfigurationWithoutCreate(t *testing.T) {
	t.Parallel()

	resource := newConfigurationTestResource()
	fake := &fakeConfigurationOCIClient{
		createFn: func(context.Context, zprsdk.CreateConfigurationRequest) (zprsdk.CreateConfigurationResponse, error) {
			t.Fatal("CreateConfiguration should not be called when the singleton already exists")
			return zprsdk.CreateConfigurationResponse{}, nil
		},
		getFn: func(_ context.Context, req zprsdk.GetConfigurationRequest) (zprsdk.GetConfigurationResponse, error) {
			if got := stringValue(req.CompartmentId); got != resource.Spec.CompartmentId {
				t.Fatalf("GetConfiguration compartmentId = %q, want %q", got, resource.Spec.CompartmentId)
			}
			return zprsdk.GetConfigurationResponse{
				Configuration: makeSDKConfiguration(testConfigurationID, zprsdk.ConfigurationLifecycleStateActive),
			}, nil
		},
	}

	client := newConfigurationTestClient(fake)
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success when binding an existing singleton")
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateConfiguration calls = %d, want 0", len(fake.createRequests))
	}
	if len(fake.getRequests) != 2 {
		t.Fatalf("GetConfiguration calls = %d, want 2 singleton reads", len(fake.getRequests))
	}
	if got := resource.Status.Id; got != testConfigurationID {
		t.Fatalf("status.id = %q, want %q", got, testConfigurationID)
	}
}

func TestConfigurationRuntimeRecreatesAfterStaleTrackedIdentityNotFound(t *testing.T) {
	t.Parallel()

	resource := newConfigurationTestResource()
	resource.Status.Id = "ocid1.zprconfiguration.oc1..stale"
	resource.Status.CompartmentId = resource.Spec.CompartmentId
	resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)

	getCalls := 0
	fake := &fakeConfigurationOCIClient{
		createFn: func(_ context.Context, _ zprsdk.CreateConfigurationRequest) (zprsdk.CreateConfigurationResponse, error) {
			return zprsdk.CreateConfigurationResponse{
				OpcWorkRequestId: common.String("wr-create-stale"),
			}, nil
		},
		getFn: func(_ context.Context, _ zprsdk.GetConfigurationRequest) (zprsdk.GetConfigurationResponse, error) {
			getCalls++
			if getCalls == 1 {
				return zprsdk.GetConfigurationResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "not found")
			}
			return zprsdk.GetConfigurationResponse{
				Configuration: makeSDKConfiguration(testConfigurationID, zprsdk.ConfigurationLifecycleStateActive),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, _ zprsdk.GetZprConfigurationWorkRequestRequest) (zprsdk.GetZprConfigurationWorkRequestResponse, error) {
			return zprsdk.GetZprConfigurationWorkRequestResponse{
				WorkRequest: makeConfigurationWorkRequest(
					"wr-create-stale",
					zprsdk.WorkRequestStatusSucceeded,
					zprsdk.OperationTypeCreateZprConfiguration,
					zprsdk.ActionTypeCreated,
					testConfigurationID,
				),
			}, nil
		},
	}

	client := newConfigurationTestClient(fake)
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success after recreating a stale tracked singleton")
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("CreateConfiguration calls = %d, want 1", len(fake.createRequests))
	}
	if len(fake.getWorkRequestRequests) != 1 {
		t.Fatalf("GetZprConfigurationWorkRequest calls = %d, want 1", len(fake.getWorkRequestRequests))
	}
	if got := resource.Status.Id; got != testConfigurationID {
		t.Fatalf("status.id = %q, want %q", got, testConfigurationID)
	}
}

func TestConfigurationRuntimeTrackedCompartmentDriftFailsInsteadOfRecreating(t *testing.T) {
	t.Parallel()

	resource := newConfigurationTestResource()
	resource.Status.Id = "ocid1.zprconfiguration.oc1..tracked"
	resource.Status.CompartmentId = "ocid1.tenancy.oc1..tracked"
	resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)
	resource.Spec.CompartmentId = "ocid1.tenancy.oc1..desired"

	fake := &fakeConfigurationOCIClient{
		createFn: func(context.Context, zprsdk.CreateConfigurationRequest) (zprsdk.CreateConfigurationResponse, error) {
			t.Fatal("CreateConfiguration should not be called when tracked compartmentId drift requires replacement")
			return zprsdk.CreateConfigurationResponse{}, nil
		},
		getFn: func(_ context.Context, req zprsdk.GetConfigurationRequest) (zprsdk.GetConfigurationResponse, error) {
			if got := stringValue(req.CompartmentId); got != resource.Status.CompartmentId {
				t.Fatalf("GetConfiguration compartmentId = %q, want tracked status compartment %q", got, resource.Status.CompartmentId)
			}
			return zprsdk.GetConfigurationResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "not found")
		},
		getWorkRequestFn: func(context.Context, zprsdk.GetZprConfigurationWorkRequestRequest) (zprsdk.GetZprConfigurationWorkRequestResponse, error) {
			t.Fatal("GetZprConfigurationWorkRequest should not be called when tracked compartmentId drift requires replacement")
			return zprsdk.GetZprConfigurationWorkRequestResponse{}, nil
		},
	}

	client := newConfigurationTestClient(fake)
	_, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "require replacement when compartmentId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartmentId replacement failure", err)
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateConfiguration calls = %d, want 0", len(fake.createRequests))
	}
	if len(fake.getRequests) != 1 {
		t.Fatalf("GetConfiguration calls = %d, want 1 tracked-compartment probe", len(fake.getRequests))
	}
	if len(fake.getWorkRequestRequests) != 0 {
		t.Fatalf("GetZprConfigurationWorkRequest calls = %d, want 0", len(fake.getWorkRequestRequests))
	}
}

func TestConfigurationRuntimeTrackedDeletedCompartmentDriftFailsInsteadOfRecreating(t *testing.T) {
	t.Parallel()

	resource := newConfigurationTestResource()
	resource.Status.Id = "ocid1.zprconfiguration.oc1..tracked"
	resource.Status.CompartmentId = "ocid1.tenancy.oc1..tracked"
	resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)
	resource.Spec.CompartmentId = "ocid1.tenancy.oc1..desired"

	fake := &fakeConfigurationOCIClient{
		createFn: func(context.Context, zprsdk.CreateConfigurationRequest) (zprsdk.CreateConfigurationResponse, error) {
			t.Fatal("CreateConfiguration should not be called when tracked deleted compartmentId drift requires replacement")
			return zprsdk.CreateConfigurationResponse{}, nil
		},
		getFn: func(_ context.Context, req zprsdk.GetConfigurationRequest) (zprsdk.GetConfigurationResponse, error) {
			if got := stringValue(req.CompartmentId); got != resource.Status.CompartmentId {
				t.Fatalf("GetConfiguration compartmentId = %q, want tracked status compartment %q", got, resource.Status.CompartmentId)
			}
			return zprsdk.GetConfigurationResponse{
				Configuration: makeSDKConfiguration(resource.Status.Id, zprsdk.ConfigurationLifecycleStateDeleted),
			}, nil
		},
		getWorkRequestFn: func(context.Context, zprsdk.GetZprConfigurationWorkRequestRequest) (zprsdk.GetZprConfigurationWorkRequestResponse, error) {
			t.Fatal("GetZprConfigurationWorkRequest should not be called when tracked deleted compartmentId drift requires replacement")
			return zprsdk.GetZprConfigurationWorkRequestResponse{}, nil
		},
	}

	client := newConfigurationTestClient(fake)
	_, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "require replacement when compartmentId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartmentId replacement failure", err)
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateConfiguration calls = %d, want 0", len(fake.createRequests))
	}
	if len(fake.getRequests) != 1 {
		t.Fatalf("GetConfiguration calls = %d, want 1 tracked-compartment probe", len(fake.getRequests))
	}
	if len(fake.getWorkRequestRequests) != 0 {
		t.Fatalf("GetZprConfigurationWorkRequest calls = %d, want 0", len(fake.getWorkRequestRequests))
	}
}

func TestConfigurationRuntimeRejectsUnsupportedFreeformTagDrift(t *testing.T) {
	t.Parallel()

	resource := newConfigurationTestResource()
	resource.Spec.FreeformTags = map[string]string{}

	fake := &fakeConfigurationOCIClient{
		createFn: func(context.Context, zprsdk.CreateConfigurationRequest) (zprsdk.CreateConfigurationResponse, error) {
			t.Fatal("CreateConfiguration should not be called when binding an existing singleton with unsupported drift")
			return zprsdk.CreateConfigurationResponse{}, nil
		},
		getFn: func(context.Context, zprsdk.GetConfigurationRequest) (zprsdk.GetConfigurationResponse, error) {
			return zprsdk.GetConfigurationResponse{
				Configuration: makeSDKConfiguration(testConfigurationID, zprsdk.ConfigurationLifecycleStateActive),
			}, nil
		},
	}

	client := newConfigurationTestClient(fake)
	_, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "reject unsupported update drift for freeformTags") {
		t.Fatalf("CreateOrUpdate() error = %v, want unsupported freeformTags drift failure", err)
	}
}

func TestConfigurationRuntimeDeleteIsLocalCleanupOnly(t *testing.T) {
	t.Parallel()

	fake := &fakeConfigurationOCIClient{}
	client := newConfigurationTestClient(fake)

	deleted, err := client.Delete(context.Background(), newConfigurationTestResource())
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() = false, want CR-local cleanup success")
	}
	if len(fake.createRequests) != 0 || len(fake.getRequests) != 0 || len(fake.getWorkRequestRequests) != 0 {
		t.Fatalf("OCI calls create/get/getWorkRequest = %d/%d/%d, want 0/0/0", len(fake.createRequests), len(fake.getRequests), len(fake.getWorkRequestRequests))
	}
}

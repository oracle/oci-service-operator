package functions

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	ocifunctions "github.com/oracle/oci-go-sdk/v65/functions"
	functionsv1beta1 "github.com/oracle/oci-service-operator/api/functions/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeFunctionsManagementClient struct {
	createApplicationFn func(context.Context, ocifunctions.CreateApplicationRequest) (ocifunctions.CreateApplicationResponse, error)
	getApplicationFn    func(context.Context, ocifunctions.GetApplicationRequest) (ocifunctions.GetApplicationResponse, error)
	listApplicationsFn  func(context.Context, ocifunctions.ListApplicationsRequest) (ocifunctions.ListApplicationsResponse, error)
	updateApplicationFn func(context.Context, ocifunctions.UpdateApplicationRequest) (ocifunctions.UpdateApplicationResponse, error)
	deleteApplicationFn func(context.Context, ocifunctions.DeleteApplicationRequest) (ocifunctions.DeleteApplicationResponse, error)
	createFunctionFn    func(context.Context, ocifunctions.CreateFunctionRequest) (ocifunctions.CreateFunctionResponse, error)
	getFunctionFn       func(context.Context, ocifunctions.GetFunctionRequest) (ocifunctions.GetFunctionResponse, error)
	listFunctionsFn     func(context.Context, ocifunctions.ListFunctionsRequest) (ocifunctions.ListFunctionsResponse, error)
	updateFunctionFn    func(context.Context, ocifunctions.UpdateFunctionRequest) (ocifunctions.UpdateFunctionResponse, error)
	deleteFunctionFn    func(context.Context, ocifunctions.DeleteFunctionRequest) (ocifunctions.DeleteFunctionResponse, error)
}

func (f *fakeFunctionsManagementClient) CreateApplication(ctx context.Context, req ocifunctions.CreateApplicationRequest) (ocifunctions.CreateApplicationResponse, error) {
	if f.createApplicationFn != nil {
		return f.createApplicationFn(ctx, req)
	}
	return ocifunctions.CreateApplicationResponse{}, nil
}

func (f *fakeFunctionsManagementClient) GetApplication(ctx context.Context, req ocifunctions.GetApplicationRequest) (ocifunctions.GetApplicationResponse, error) {
	if f.getApplicationFn != nil {
		return f.getApplicationFn(ctx, req)
	}
	return ocifunctions.GetApplicationResponse{}, nil
}

func (f *fakeFunctionsManagementClient) ListApplications(ctx context.Context, req ocifunctions.ListApplicationsRequest) (ocifunctions.ListApplicationsResponse, error) {
	if f.listApplicationsFn != nil {
		return f.listApplicationsFn(ctx, req)
	}
	return ocifunctions.ListApplicationsResponse{}, nil
}

func (f *fakeFunctionsManagementClient) UpdateApplication(ctx context.Context, req ocifunctions.UpdateApplicationRequest) (ocifunctions.UpdateApplicationResponse, error) {
	if f.updateApplicationFn != nil {
		return f.updateApplicationFn(ctx, req)
	}
	return ocifunctions.UpdateApplicationResponse{}, nil
}

func (f *fakeFunctionsManagementClient) DeleteApplication(ctx context.Context, req ocifunctions.DeleteApplicationRequest) (ocifunctions.DeleteApplicationResponse, error) {
	if f.deleteApplicationFn != nil {
		return f.deleteApplicationFn(ctx, req)
	}
	return ocifunctions.DeleteApplicationResponse{}, nil
}

func (f *fakeFunctionsManagementClient) CreateFunction(ctx context.Context, req ocifunctions.CreateFunctionRequest) (ocifunctions.CreateFunctionResponse, error) {
	if f.createFunctionFn != nil {
		return f.createFunctionFn(ctx, req)
	}
	return ocifunctions.CreateFunctionResponse{}, nil
}

func (f *fakeFunctionsManagementClient) GetFunction(ctx context.Context, req ocifunctions.GetFunctionRequest) (ocifunctions.GetFunctionResponse, error) {
	if f.getFunctionFn != nil {
		return f.getFunctionFn(ctx, req)
	}
	return ocifunctions.GetFunctionResponse{}, nil
}

func (f *fakeFunctionsManagementClient) ListFunctions(ctx context.Context, req ocifunctions.ListFunctionsRequest) (ocifunctions.ListFunctionsResponse, error) {
	if f.listFunctionsFn != nil {
		return f.listFunctionsFn(ctx, req)
	}
	return ocifunctions.ListFunctionsResponse{}, nil
}

func (f *fakeFunctionsManagementClient) UpdateFunction(ctx context.Context, req ocifunctions.UpdateFunctionRequest) (ocifunctions.UpdateFunctionResponse, error) {
	if f.updateFunctionFn != nil {
		return f.updateFunctionFn(ctx, req)
	}
	return ocifunctions.UpdateFunctionResponse{}, nil
}

func (f *fakeFunctionsManagementClient) DeleteFunction(ctx context.Context, req ocifunctions.DeleteFunctionRequest) (ocifunctions.DeleteFunctionResponse, error) {
	if f.deleteFunctionFn != nil {
		return f.deleteFunctionFn(ctx, req)
	}
	return ocifunctions.DeleteFunctionResponse{}, nil
}

var _ FunctionsManagementClientInterface = (*fakeFunctionsManagementClient)(nil)

type fakeFunctionsCredentialClient struct {
	deleteSecretFn func(context.Context, string, string) (bool, error)
	deleteCalls    int
}

func (f *fakeFunctionsCredentialClient) CreateSecret(context.Context, string, string, map[string]string, map[string][]byte) (bool, error) {
	return true, nil
}

func (f *fakeFunctionsCredentialClient) DeleteSecret(ctx context.Context, name string, namespace string) (bool, error) {
	f.deleteCalls++
	if f.deleteSecretFn != nil {
		return f.deleteSecretFn(ctx, name, namespace)
	}
	return true, nil
}

func (f *fakeFunctionsCredentialClient) GetSecret(context.Context, string, string) (map[string][]byte, error) {
	return nil, nil
}

func (f *fakeFunctionsCredentialClient) UpdateSecret(context.Context, string, string, map[string]string, map[string][]byte) (bool, error) {
	return true, nil
}

var _ credhelper.CredentialClient = (*fakeFunctionsCredentialClient)(nil)

type recordingFunctionsCredentialClient struct {
	createSecretFn          func(context.Context, string, string, map[string]string, map[string][]byte) (bool, error)
	deleteSecretFn          func(context.Context, string, string) (bool, error)
	deleteSecretIfCurrentFn func(context.Context, string, string, credhelper.SecretRecord) (bool, error)
	getSecretFn             func(context.Context, string, string) (map[string][]byte, error)
	getSecretRecordFn       func(context.Context, string, string) (credhelper.SecretRecord, error)
	updateSecretFn          func(context.Context, string, string, map[string]string, map[string][]byte) (bool, error)
	updateSecretIfCurrentFn func(context.Context, string, string, credhelper.SecretRecord, map[string]string, map[string][]byte) (bool, error)
	createCalls             int
	deleteCalls             int
	deleteIfCurrentCalls    int
	getCalls                int
	getRecordCalls          int
	updateCalls             int
	updateIfCurrentCalls    int
}

func (f *recordingFunctionsCredentialClient) CreateSecret(
	ctx context.Context,
	name string,
	namespace string,
	labels map[string]string,
	data map[string][]byte,
) (bool, error) {
	f.createCalls++
	if f.createSecretFn != nil {
		return f.createSecretFn(ctx, name, namespace, labels, data)
	}
	return true, nil
}

func (f *recordingFunctionsCredentialClient) DeleteSecret(ctx context.Context, name string, namespace string) (bool, error) {
	f.deleteCalls++
	if f.deleteSecretFn != nil {
		return f.deleteSecretFn(ctx, name, namespace)
	}
	return true, nil
}

func (f *recordingFunctionsCredentialClient) DeleteSecretIfCurrent(
	ctx context.Context,
	name string,
	namespace string,
	current credhelper.SecretRecord,
) (bool, error) {
	f.deleteIfCurrentCalls++
	if f.deleteSecretIfCurrentFn != nil {
		return f.deleteSecretIfCurrentFn(ctx, name, namespace, current)
	}
	return true, nil
}

func (f *recordingFunctionsCredentialClient) GetSecret(ctx context.Context, name string, namespace string) (map[string][]byte, error) {
	f.getCalls++
	if f.getSecretFn != nil {
		return f.getSecretFn(ctx, name, namespace)
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, name)
}

func (f *recordingFunctionsCredentialClient) GetSecretRecord(
	ctx context.Context,
	name string,
	namespace string,
) (credhelper.SecretRecord, error) {
	f.getRecordCalls++
	if f.getSecretRecordFn != nil {
		return f.getSecretRecordFn(ctx, name, namespace)
	}
	return credhelper.SecretRecord{}, apierrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, name)
}

func (f *recordingFunctionsCredentialClient) UpdateSecret(
	ctx context.Context,
	name string,
	namespace string,
	labels map[string]string,
	data map[string][]byte,
) (bool, error) {
	f.updateCalls++
	if f.updateSecretFn != nil {
		return f.updateSecretFn(ctx, name, namespace, labels, data)
	}
	return true, nil
}

func (f *recordingFunctionsCredentialClient) UpdateSecretIfCurrent(
	ctx context.Context,
	name string,
	namespace string,
	current credhelper.SecretRecord,
	labels map[string]string,
	data map[string][]byte,
) (bool, error) {
	f.updateIfCurrentCalls++
	if f.updateSecretIfCurrentFn != nil {
		return f.updateSecretIfCurrentFn(ctx, name, namespace, current, labels, data)
	}
	return true, nil
}

var _ credhelper.CredentialClient = (*recordingFunctionsCredentialClient)(nil)
var _ credhelper.SecretRecordReader = (*recordingFunctionsCredentialClient)(nil)
var _ credhelper.GuardedSecretMutator = (*recordingFunctionsCredentialClient)(nil)

func TestIsFunctionsNotFoundTreatsAll404sAsMissing(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "namespace-scoped 404", err: errortest.NewServiceError(404, "NamespaceNotFound", "namespace missing"), want: true},
		{name: "auth-shaped 404", err: errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "missing"), want: true},
		{name: "forbidden", err: errortest.NewServiceError(403, "NotAuthorized", "forbidden"), want: false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := isFunctionsNotFound(tc.err); got != tc.want {
				t.Fatalf("isFunctionsNotFound() = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestFunctionsApplicationCreateOrUpdateTracked404FallsBackToCreate(t *testing.T) {
	t.Parallel()

	getCalls := 0
	createCalls := 0
	client := &fakeFunctionsManagementClient{
		getApplicationFn: func(_ context.Context, req ocifunctions.GetApplicationRequest) (ocifunctions.GetApplicationResponse, error) {
			getCalls++
			if got := safeFunctionsString(req.ApplicationId); got != "ocid1.application.oc1..tracked" {
				t.Fatalf("GetApplicationRequest.ApplicationId = %q, want tracked id", got)
			}
			return ocifunctions.GetApplicationResponse{}, errortest.NewServiceError(404, "NamespaceNotFound", "application namespace missing")
		},
		listApplicationsFn: func(_ context.Context, req ocifunctions.ListApplicationsRequest) (ocifunctions.ListApplicationsResponse, error) {
			if got := safeFunctionsString(req.DisplayName); got != "sample-app" {
				t.Fatalf("ListApplicationsRequest.DisplayName = %q, want sample-app", got)
			}
			return ocifunctions.ListApplicationsResponse{}, nil
		},
		createApplicationFn: func(_ context.Context, req ocifunctions.CreateApplicationRequest) (ocifunctions.CreateApplicationResponse, error) {
			createCalls++
			if got := safeFunctionsString(req.CreateApplicationDetails.DisplayName); got != "sample-app" {
				t.Fatalf("CreateApplicationDetails.DisplayName = %q, want sample-app", got)
			}
			return ocifunctions.CreateApplicationResponse{
				OpcRequestId: common.String("opc-create-app-1"),
				Application: ocifunctions.Application{
					Id:             common.String("ocid1.application.oc1..created"),
					DisplayName:    common.String("sample-app"),
					LifecycleState: ocifunctions.ApplicationLifecycleStateCreating,
				},
			}, nil
		},
	}

	manager := &FunctionsApplicationServiceManager{
		Log:       testFunctionsLogger(),
		ociClient: client,
	}

	resource := testApplicationResource()
	resource.Status.Id = "ocid1.application.oc1..tracked"
	resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeueing create", response)
	}
	if getCalls != 1 {
		t.Fatalf("GetApplication() calls = %d, want 1", getCalls)
	}
	if createCalls != 1 {
		t.Fatalf("CreateApplication() calls = %d, want 1", createCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != "ocid1.application.oc1..created" {
		t.Fatalf("status.ocid = %q, want created OCID", got)
	}
	if got := resource.Status.Id; got != "ocid1.application.oc1..created" {
		t.Fatalf("status.id = %q, want created OCID", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-app-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", got, "opc-create-app-1")
	}
	if len(resource.Status.OsokStatus.Conditions) == 0 || resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type != shared.Provisioning {
		t.Fatalf("status.conditions = %#v, want trailing Provisioning condition", resource.Status.OsokStatus.Conditions)
	}
}

func TestFunctionsApplicationCreateOrUpdateUpdatesMutableConfig(t *testing.T) {
	t.Parallel()

	updateCalls := 0
	client := &fakeFunctionsManagementClient{
		getApplicationFn: func(_ context.Context, req ocifunctions.GetApplicationRequest) (ocifunctions.GetApplicationResponse, error) {
			if got := safeFunctionsString(req.ApplicationId); got != "ocid1.application.oc1..tracked" {
				t.Fatalf("GetApplicationRequest.ApplicationId = %q, want tracked id", got)
			}
			return ocifunctions.GetApplicationResponse{
				Application: ocifunctions.Application{
					Id:             common.String("ocid1.application.oc1..tracked"),
					CompartmentId:  common.String("ocid1.compartment.oc1..example"),
					DisplayName:    common.String("sample-app"),
					SubnetIds:      []string{"ocid1.subnet.oc1..example"},
					Config:         map[string]string{"mode": "old"},
					Shape:          ocifunctions.ApplicationShapeX86,
					LifecycleState: ocifunctions.ApplicationLifecycleStateActive,
				},
			}, nil
		},
		updateApplicationFn: func(_ context.Context, req ocifunctions.UpdateApplicationRequest) (ocifunctions.UpdateApplicationResponse, error) {
			updateCalls++
			if got := safeFunctionsString(req.ApplicationId); got != "ocid1.application.oc1..tracked" {
				t.Fatalf("UpdateApplicationRequest.ApplicationId = %q, want tracked id", got)
			}
			if got := req.UpdateApplicationDetails.Config["mode"]; got != "new" {
				t.Fatalf("UpdateApplicationDetails.Config[mode] = %q, want new", got)
			}
			if req.UpdateApplicationDetails.NetworkSecurityGroupIds != nil {
				t.Fatalf("UpdateApplicationDetails.NetworkSecurityGroupIds = %#v, want nil when unchanged", req.UpdateApplicationDetails.NetworkSecurityGroupIds)
			}
			return ocifunctions.UpdateApplicationResponse{
				OpcRequestId: common.String("opc-update-app-1"),
				Application: ocifunctions.Application{
					Id:             common.String("ocid1.application.oc1..tracked"),
					CompartmentId:  common.String("ocid1.compartment.oc1..example"),
					DisplayName:    common.String("sample-app"),
					SubnetIds:      []string{"ocid1.subnet.oc1..example"},
					Config:         map[string]string{"mode": "new"},
					Shape:          ocifunctions.ApplicationShapeX86,
					LifecycleState: ocifunctions.ApplicationLifecycleStateActive,
				},
			}, nil
		},
	}

	manager := &FunctionsApplicationServiceManager{
		Log:       testFunctionsLogger(),
		ociClient: client,
	}

	resource := testApplicationResource()
	resource.Spec.Config = map[string]string{"mode": "new"}
	resource.Status.Id = "ocid1.application.oc1..tracked"
	resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeueing update", response)
	}
	if updateCalls != 1 {
		t.Fatalf("UpdateApplication() calls = %d, want 1", updateCalls)
	}
	if got := resource.Status.Config["mode"]; got != "new" {
		t.Fatalf("status.config[mode] = %q, want new", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-app-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", got, "opc-update-app-1")
	}
	requireTrailingFunctionsCondition(t, resource.Status.OsokStatus, shared.Active)
}

func TestFunctionsApplicationCreateOrUpdateRejectsCreateOnlyDrift(t *testing.T) {
	t.Parallel()

	updateCalls := 0
	client := &fakeFunctionsManagementClient{
		getApplicationFn: func(_ context.Context, _ ocifunctions.GetApplicationRequest) (ocifunctions.GetApplicationResponse, error) {
			return ocifunctions.GetApplicationResponse{
				Application: ocifunctions.Application{
					Id:             common.String("ocid1.application.oc1..tracked"),
					CompartmentId:  common.String("ocid1.compartment.oc1..existing"),
					DisplayName:    common.String("existing-app"),
					SubnetIds:      []string{"ocid1.subnet.oc1..existing"},
					Shape:          ocifunctions.ApplicationShapeX86,
					LifecycleState: ocifunctions.ApplicationLifecycleStateActive,
				},
			}, nil
		},
		updateApplicationFn: func(context.Context, ocifunctions.UpdateApplicationRequest) (ocifunctions.UpdateApplicationResponse, error) {
			updateCalls++
			return ocifunctions.UpdateApplicationResponse{}, nil
		},
	}

	manager := &FunctionsApplicationServiceManager{
		Log:       testFunctionsLogger(),
		ociClient: client,
	}

	resource := testApplicationResource()
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..desired"
	resource.Spec.DisplayName = "desired-app"
	resource.Spec.SubnetIds = []string{"ocid1.subnet.oc1..desired"}
	resource.Spec.Shape = string(ocifunctions.ApplicationShapeArm)
	resource.Status.Id = "ocid1.application.oc1..tracked"
	resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful drift rejection", response)
	}
	for _, want := range []string{"create-only field drift", "compartmentId", "displayName", "subnetIds", "shape"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("CreateOrUpdate() error = %q, want substring %q", err.Error(), want)
		}
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateApplication() calls = %d, want 0 after create-only drift rejection", updateCalls)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Failed) {
		t.Fatalf("status.reason = %q, want Failed", got)
	}
	requireTrailingFunctionsCondition(t, resource.Status.OsokStatus, shared.Failed)
}

func TestFunctionsApplicationCreateOrUpdateSkipsUpdateWhileCreating(t *testing.T) {
	t.Parallel()

	updateCalls := 0
	client := &fakeFunctionsManagementClient{
		getApplicationFn: func(_ context.Context, _ ocifunctions.GetApplicationRequest) (ocifunctions.GetApplicationResponse, error) {
			return ocifunctions.GetApplicationResponse{
				Application: ocifunctions.Application{
					Id:             common.String("ocid1.application.oc1..tracked"),
					CompartmentId:  common.String("ocid1.compartment.oc1..example"),
					DisplayName:    common.String("sample-app"),
					SubnetIds:      []string{"ocid1.subnet.oc1..example"},
					Config:         map[string]string{"mode": "old"},
					LifecycleState: ocifunctions.ApplicationLifecycleStateCreating,
				},
			}, nil
		},
		updateApplicationFn: func(context.Context, ocifunctions.UpdateApplicationRequest) (ocifunctions.UpdateApplicationResponse, error) {
			updateCalls++
			return ocifunctions.UpdateApplicationResponse{}, nil
		},
	}

	manager := &FunctionsApplicationServiceManager{
		Log:       testFunctionsLogger(),
		ociClient: client,
	}

	resource := testApplicationResource()
	resource.Spec.Config = map[string]string{"mode": "new"}
	resource.Status.Id = "ocid1.application.oc1..tracked"
	resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue while creating", response)
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateApplication() calls = %d, want 0 while lifecycle is CREATING", updateCalls)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Provisioning) {
		t.Fatalf("status.reason = %q, want Provisioning", got)
	}
	requireTrailingFunctionsCondition(t, resource.Status.OsokStatus, shared.Provisioning)
}

func TestFunctionsApplicationLifecycleClassification(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		state       ocifunctions.ApplicationLifecycleStateEnum
		wantReason  shared.OSOKConditionType
		wantSuccess bool
		wantRequeue bool
	}{
		{name: "active", state: ocifunctions.ApplicationLifecycleStateActive, wantReason: shared.Active, wantSuccess: true},
		{name: "inactive", state: ocifunctions.ApplicationLifecycleStateInactive, wantReason: shared.Active, wantSuccess: true},
		{name: "creating", state: ocifunctions.ApplicationLifecycleStateCreating, wantReason: shared.Provisioning, wantSuccess: true, wantRequeue: true},
		{name: "updating", state: ocifunctions.ApplicationLifecycleStateUpdating, wantReason: shared.Updating, wantSuccess: true, wantRequeue: true},
		{name: "deleting", state: ocifunctions.ApplicationLifecycleStateDeleting, wantReason: shared.Terminating, wantSuccess: true, wantRequeue: true},
		{name: "failed", state: ocifunctions.ApplicationLifecycleStateFailed, wantReason: shared.Failed},
		{name: "deleted", state: ocifunctions.ApplicationLifecycleStateDeleted, wantReason: shared.Failed},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			status := shared.OSOKStatus{}
			response := reconcileFunctionsApplicationLifecycle(&status, &ocifunctions.Application{
				Id:             common.String("ocid1.application.oc1..tracked"),
				DisplayName:    common.String("sample-app"),
				LifecycleState: tc.state,
			}, testFunctionsLogger())

			if response.IsSuccessful != tc.wantSuccess {
				t.Fatalf("IsSuccessful = %t, want %t", response.IsSuccessful, tc.wantSuccess)
			}
			if response.ShouldRequeue != tc.wantRequeue {
				t.Fatalf("ShouldRequeue = %t, want %t", response.ShouldRequeue, tc.wantRequeue)
			}
			if got := status.Reason; got != string(tc.wantReason) {
				t.Fatalf("status.reason = %q, want %q", got, tc.wantReason)
			}
			requireTrailingFunctionsCondition(t, status, tc.wantReason)
		})
	}
}

func TestFunctionsApplicationDeleteTreatsRead404AsDeleted(t *testing.T) {
	t.Parallel()

	deleteCalls := 0
	getCalls := 0
	client := &fakeFunctionsManagementClient{
		deleteApplicationFn: func(_ context.Context, req ocifunctions.DeleteApplicationRequest) (ocifunctions.DeleteApplicationResponse, error) {
			deleteCalls++
			if got := safeFunctionsString(req.ApplicationId); got != "ocid1.application.oc1..tracked" {
				t.Fatalf("DeleteApplicationRequest.ApplicationId = %q, want tracked id", got)
			}
			return ocifunctions.DeleteApplicationResponse{}, nil
		},
		getApplicationFn: func(_ context.Context, req ocifunctions.GetApplicationRequest) (ocifunctions.GetApplicationResponse, error) {
			getCalls++
			if got := safeFunctionsString(req.ApplicationId); got != "ocid1.application.oc1..tracked" {
				t.Fatalf("GetApplicationRequest.ApplicationId = %q, want tracked id", got)
			}
			return ocifunctions.GetApplicationResponse{}, errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "application missing")
		},
	}

	manager := &FunctionsApplicationServiceManager{
		Log:       testFunctionsLogger(),
		ociClient: client,
	}

	resource := testApplicationResource()
	resource.Status.Id = "ocid1.application.oc1..tracked"
	resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)

	deleted, err := manager.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() should treat a reread 404 as deleted")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteApplication() calls = %d, want 1", deleteCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetApplication() calls = %d, want 1", getCalls)
	}
	if resource.Status.Id != "" {
		t.Fatalf("status.id = %q, want cleared tracked id", resource.Status.Id)
	}
	if resource.Status.OsokStatus.Ocid != "" {
		t.Fatalf("status.ocid = %q, want cleared tracked ocid", resource.Status.OsokStatus.Ocid)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, "opc-request-id")
	}
}

func TestFunctionsApplicationDeleteMarksTerminatingUntilReadbackConfirmsGone(t *testing.T) {
	t.Parallel()

	deleteCalls := 0
	getCalls := 0
	client := &fakeFunctionsManagementClient{
		deleteApplicationFn: func(_ context.Context, req ocifunctions.DeleteApplicationRequest) (ocifunctions.DeleteApplicationResponse, error) {
			deleteCalls++
			if got := safeFunctionsString(req.ApplicationId); got != "ocid1.application.oc1..tracked" {
				t.Fatalf("DeleteApplicationRequest.ApplicationId = %q, want tracked id", got)
			}
			return ocifunctions.DeleteApplicationResponse{}, nil
		},
		getApplicationFn: func(_ context.Context, req ocifunctions.GetApplicationRequest) (ocifunctions.GetApplicationResponse, error) {
			getCalls++
			if got := safeFunctionsString(req.ApplicationId); got != "ocid1.application.oc1..tracked" {
				t.Fatalf("GetApplicationRequest.ApplicationId = %q, want tracked id", got)
			}
			return ocifunctions.GetApplicationResponse{
				Application: ocifunctions.Application{
					Id:             common.String("ocid1.application.oc1..tracked"),
					DisplayName:    common.String("sample-app"),
					LifecycleState: ocifunctions.ApplicationLifecycleStateDeleting,
				},
			}, nil
		},
	}

	manager := &FunctionsApplicationServiceManager{
		Log:       testFunctionsLogger(),
		ociClient: client,
	}

	resource := testApplicationResource()
	resource.Status.Id = "ocid1.application.oc1..tracked"
	resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)

	deleted, err := manager.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() should keep the finalizer while OCI still returns the application")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteApplication() calls = %d, want 1", deleteCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetApplication() calls = %d, want 1", getCalls)
	}
	if resource.Status.Id != "ocid1.application.oc1..tracked" {
		t.Fatalf("status.id = %q, want retained tracked id", resource.Status.Id)
	}
	if resource.Status.OsokStatus.Ocid != shared.OCID("ocid1.application.oc1..tracked") {
		t.Fatalf("status.ocid = %q, want retained tracked ocid", resource.Status.OsokStatus.Ocid)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Terminating) {
		t.Fatalf("status.reason = %q, want Terminating", got)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete progress timestamp")
	}
	requireTrailingFunctionsCondition(t, resource.Status.OsokStatus, shared.Terminating)
}

func TestFunctionsFunctionDeleteTreatsDelete404AsDeletedAndIgnoresMissingSecret(t *testing.T) {
	t.Parallel()

	credentials := &fakeFunctionsCredentialClient{
		deleteSecretFn: func(_ context.Context, name string, namespace string) (bool, error) {
			if name != "sample-function" || namespace != "default" {
				t.Fatalf("DeleteSecret(%q, %q), want sample-function/default", name, namespace)
			}
			return false, apierrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, name)
		},
	}
	deleteCalls := 0
	client := &fakeFunctionsManagementClient{
		deleteFunctionFn: func(_ context.Context, req ocifunctions.DeleteFunctionRequest) (ocifunctions.DeleteFunctionResponse, error) {
			deleteCalls++
			if got := safeFunctionsString(req.FunctionId); got != "ocid1.fnfunc.oc1..tracked" {
				t.Fatalf("DeleteFunctionRequest.FunctionId = %q, want tracked id", got)
			}
			return ocifunctions.DeleteFunctionResponse{}, errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "function missing")
		},
	}

	manager := &FunctionsFunctionServiceManager{
		CredentialClient: credentials,
		Log:              testFunctionsLogger(),
		ociClient:        client,
	}

	resource := testFunctionResource()
	resource.Status.Id = "ocid1.fnfunc.oc1..tracked"
	resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)

	deleted, err := manager.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() should treat a delete 404 as deleted")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteFunction() calls = %d, want 1", deleteCalls)
	}
	if credentials.deleteCalls != 1 {
		t.Fatalf("DeleteSecret() calls = %d, want 1", credentials.deleteCalls)
	}
	if resource.Status.Id != "" {
		t.Fatalf("status.id = %q, want cleared tracked id", resource.Status.Id)
	}
	if resource.Status.OsokStatus.Ocid != "" {
		t.Fatalf("status.ocid = %q, want cleared tracked ocid", resource.Status.OsokStatus.Ocid)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, "opc-request-id")
	}
}

func TestFunctionsFunctionCreateOrUpdateBadRequestCapturesOCIErrorCode(t *testing.T) {
	t.Parallel()

	_, badRequestErr := errorutil.NewServiceFailureFromResponse(
		"InvalidParameter",
		400,
		"opc-request-id",
		"memory is invalid",
	)
	client := &fakeFunctionsManagementClient{
		listFunctionsFn: func(_ context.Context, req ocifunctions.ListFunctionsRequest) (ocifunctions.ListFunctionsResponse, error) {
			if got := safeFunctionsString(req.DisplayName); got != "sample-function" {
				t.Fatalf("ListFunctionsRequest.DisplayName = %q, want sample-function", got)
			}
			return ocifunctions.ListFunctionsResponse{}, nil
		},
		createFunctionFn: func(_ context.Context, req ocifunctions.CreateFunctionRequest) (ocifunctions.CreateFunctionResponse, error) {
			if got := safeFunctionsString(req.CreateFunctionDetails.DisplayName); got != "sample-function" {
				t.Fatalf("CreateFunctionDetails.DisplayName = %q, want sample-function", got)
			}
			return ocifunctions.CreateFunctionResponse{}, badRequestErr
		},
	}

	manager := &FunctionsFunctionServiceManager{
		Log:       testFunctionsLogger(),
		ociClient: client,
	}

	resource := testFunctionResource()

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want bad request failure")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful response", response)
	}
	if got := resource.Status.OsokStatus.Message; got != "InvalidParameter" {
		t.Fatalf("status.message = %q, want OCI error code InvalidParameter", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want %q", got, "opc-request-id")
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Failed) {
		t.Fatalf("status.reason = %q, want Failed", got)
	}
	if len(resource.Status.OsokStatus.Conditions) == 0 || resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type != shared.Failed {
		t.Fatalf("status.conditions = %#v, want trailing Failed condition", resource.Status.OsokStatus.Conditions)
	}
}

func TestFunctionsFunctionCreateOrUpdateReusesPagedListMatch(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.fnfunc.oc1..existing"

	listCalls := 0
	getCalls := 0
	createCalls := 0
	updateCalls := 0
	client := &fakeFunctionsManagementClient{
		listFunctionsFn: func(_ context.Context, req ocifunctions.ListFunctionsRequest) (ocifunctions.ListFunctionsResponse, error) {
			listCalls++
			if got := safeFunctionsString(req.ApplicationId); got != "ocid1.application.oc1..example" {
				t.Fatalf("ListFunctionsRequest.ApplicationId = %q, want application id", got)
			}
			if got := safeFunctionsString(req.DisplayName); got != "sample-function" {
				t.Fatalf("ListFunctionsRequest.DisplayName = %q, want sample-function", got)
			}
			if req.Limit == nil || *req.Limit != 50 {
				t.Fatalf("ListFunctionsRequest.Limit = %v, want 50", req.Limit)
			}
			if listCalls == 1 {
				return ocifunctions.ListFunctionsResponse{
					Items: []ocifunctions.FunctionSummary{
						testFunctionSummary("ocid1.fnfunc.oc1..deleted", ocifunctions.FunctionLifecycleStateDeleted),
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			if got := safeFunctionsString(req.Page); got != "page-2" {
				t.Fatalf("ListFunctionsRequest.Page = %q, want page-2", got)
			}
			return ocifunctions.ListFunctionsResponse{
				Items: []ocifunctions.FunctionSummary{
					testFunctionSummary(existingID, ocifunctions.FunctionLifecycleStateActive),
				},
			}, nil
		},
		getFunctionFn: func(_ context.Context, req ocifunctions.GetFunctionRequest) (ocifunctions.GetFunctionResponse, error) {
			getCalls++
			if got := safeFunctionsString(req.FunctionId); got != existingID {
				t.Fatalf("GetFunctionRequest.FunctionId = %q, want existing id", got)
			}
			return ocifunctions.GetFunctionResponse{
				Function: testFunction(existingID, ocifunctions.FunctionLifecycleStateActive),
			}, nil
		},
		createFunctionFn: func(context.Context, ocifunctions.CreateFunctionRequest) (ocifunctions.CreateFunctionResponse, error) {
			createCalls++
			t.Fatal("CreateFunction() should not be called for a reusable list match")
			return ocifunctions.CreateFunctionResponse{}, nil
		},
		updateFunctionFn: func(context.Context, ocifunctions.UpdateFunctionRequest) (ocifunctions.UpdateFunctionResponse, error) {
			updateCalls++
			t.Fatal("UpdateFunction() should not be called when mutable fields do not drift")
			return ocifunctions.UpdateFunctionResponse{}, nil
		},
	}

	manager := &FunctionsFunctionServiceManager{
		Log:       testFunctionsLogger(),
		ociClient: client,
	}

	resource := testFunctionResource()
	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful steady state", response)
	}
	if listCalls != 2 {
		t.Fatalf("ListFunctions() calls = %d, want 2", listCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetFunction() calls = %d, want 1", getCalls)
	}
	if createCalls != 0 {
		t.Fatalf("CreateFunction() calls = %d, want 0", createCalls)
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateFunction() calls = %d, want 0", updateCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != existingID {
		t.Fatalf("status.ocid = %q, want existing OCID", got)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Active) {
		t.Fatalf("status.reason = %q, want Active", got)
	}
}

func TestFunctionsFunctionCreateOrUpdateUpdatesMutableFields(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.fnfunc.oc1..existing"

	updateCalls := 0
	client := &fakeFunctionsManagementClient{
		getFunctionFn: func(_ context.Context, req ocifunctions.GetFunctionRequest) (ocifunctions.GetFunctionResponse, error) {
			if got := safeFunctionsString(req.FunctionId); got != existingID {
				t.Fatalf("GetFunctionRequest.FunctionId = %q, want existing id", got)
			}
			current := testFunction(existingID, ocifunctions.FunctionLifecycleStateActive)
			current.MemoryInMBs = common.Int64(128)
			current.Image = common.String("phx.ocir.io/tenant/functions/sample-function:0.9.0")
			return ocifunctions.GetFunctionResponse{Function: current}, nil
		},
		updateFunctionFn: func(_ context.Context, req ocifunctions.UpdateFunctionRequest) (ocifunctions.UpdateFunctionResponse, error) {
			updateCalls++
			if got := safeFunctionsString(req.FunctionId); got != existingID {
				t.Fatalf("UpdateFunctionRequest.FunctionId = %q, want existing id", got)
			}
			if req.UpdateFunctionDetails.MemoryInMBs == nil || *req.UpdateFunctionDetails.MemoryInMBs != 256 {
				t.Fatalf("UpdateFunctionDetails.MemoryInMBs = %v, want 256", req.UpdateFunctionDetails.MemoryInMBs)
			}
			if got := safeFunctionsString(req.UpdateFunctionDetails.Image); got != "phx.ocir.io/tenant/functions/sample-function:1.0.0" {
				t.Fatalf("UpdateFunctionDetails.Image = %q, want desired image", got)
			}
			updated := testFunction(existingID, ocifunctions.FunctionLifecycleStateUpdating)
			return ocifunctions.UpdateFunctionResponse{
				OpcRequestId: common.String("opc-update-function-1"),
				Function:     updated,
			}, nil
		},
	}

	manager := &FunctionsFunctionServiceManager{
		Log:       testFunctionsLogger(),
		ociClient: client,
	}

	resource := testFunctionResource()
	resource.Status.Id = existingID
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful updating requeue", response)
	}
	if updateCalls != 1 {
		t.Fatalf("UpdateFunction() calls = %d, want 1", updateCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-function-1" {
		t.Fatalf("status.opcRequestId = %q, want update request id", got)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Updating) {
		t.Fatalf("status.reason = %q, want Updating", got)
	}
}

func TestFunctionsFunctionCreateOrUpdateUpdatesTraceConfigToFalse(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.fnfunc.oc1..existing"

	updateCalls := 0
	client := &fakeFunctionsManagementClient{
		getFunctionFn: func(_ context.Context, req ocifunctions.GetFunctionRequest) (ocifunctions.GetFunctionResponse, error) {
			if got := safeFunctionsString(req.FunctionId); got != existingID {
				t.Fatalf("GetFunctionRequest.FunctionId = %q, want existing id", got)
			}
			current := testFunction(existingID, ocifunctions.FunctionLifecycleStateActive)
			current.TraceConfig = &ocifunctions.FunctionTraceConfig{IsEnabled: common.Bool(true)}
			return ocifunctions.GetFunctionResponse{Function: current}, nil
		},
		updateFunctionFn: func(_ context.Context, req ocifunctions.UpdateFunctionRequest) (ocifunctions.UpdateFunctionResponse, error) {
			updateCalls++
			if got := safeFunctionsString(req.FunctionId); got != existingID {
				t.Fatalf("UpdateFunctionRequest.FunctionId = %q, want existing id", got)
			}
			if req.UpdateFunctionDetails.TraceConfig == nil || req.UpdateFunctionDetails.TraceConfig.IsEnabled == nil {
				t.Fatalf("UpdateFunctionDetails.TraceConfig.IsEnabled = nil, want false")
			}
			if *req.UpdateFunctionDetails.TraceConfig.IsEnabled {
				t.Fatalf("UpdateFunctionDetails.TraceConfig.IsEnabled = true, want false")
			}
			if req.UpdateFunctionDetails.Image != nil {
				t.Fatalf("UpdateFunctionDetails.Image = %q, want omitted unchanged image", safeFunctionsString(req.UpdateFunctionDetails.Image))
			}
			if req.UpdateFunctionDetails.MemoryInMBs != nil {
				t.Fatalf("UpdateFunctionDetails.MemoryInMBs = %d, want omitted unchanged memory", *req.UpdateFunctionDetails.MemoryInMBs)
			}

			updated := testFunction(existingID, ocifunctions.FunctionLifecycleStateUpdating)
			updated.TraceConfig = &ocifunctions.FunctionTraceConfig{IsEnabled: common.Bool(false)}
			return ocifunctions.UpdateFunctionResponse{
				OpcRequestId: common.String("opc-update-function-trace"),
				Function:     updated,
			}, nil
		},
	}

	manager := &FunctionsFunctionServiceManager{
		Log:       testFunctionsLogger(),
		ociClient: client,
	}

	resource := testFunctionResource()
	resource.Spec.TraceConfig.IsEnabled = false
	resource.Status.Id = existingID
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful updating requeue", response)
	}
	if updateCalls != 1 {
		t.Fatalf("UpdateFunction() calls = %d, want 1", updateCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-function-trace" {
		t.Fatalf("status.opcRequestId = %q, want update request id", got)
	}
}

func TestFunctionsFunctionCreateOrUpdateCreatesEndpointSecretWhenActive(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.fnfunc.oc1..existing"
	const invokeEndpoint = "https://invoke.functions.example.com/20181201/functions/ocid1.fnfunc.oc1..existing/actions/invoke"

	credentials := &recordingFunctionsCredentialClient{
		getSecretRecordFn: func(_ context.Context, name string, namespace string) (credhelper.SecretRecord, error) {
			requireFunctionSecretTarget(t, "GetSecretRecord", name, namespace)
			return credhelper.SecretRecord{}, apierrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, name)
		},
		createSecretFn: func(_ context.Context, name string, namespace string, labels map[string]string, data map[string][]byte) (bool, error) {
			requireFunctionSecretTarget(t, "CreateSecret", name, namespace)
			requireFunctionOwnedSecretLabels(t, labels, "function-uid")
			requireFunctionSecretData(t, data, existingID, invokeEndpoint)
			return true, nil
		},
		updateSecretIfCurrentFn: func(context.Context, string, string, credhelper.SecretRecord, map[string]string, map[string][]byte) (bool, error) {
			t.Fatal("UpdateSecretIfCurrent() should not be called when the endpoint secret is missing")
			return false, nil
		},
	}
	client := &fakeFunctionsManagementClient{
		getFunctionFn: func(_ context.Context, req ocifunctions.GetFunctionRequest) (ocifunctions.GetFunctionResponse, error) {
			if got := safeFunctionsString(req.FunctionId); got != existingID {
				t.Fatalf("GetFunctionRequest.FunctionId = %q, want existing id", got)
			}
			current := testFunction(existingID, ocifunctions.FunctionLifecycleStateActive)
			current.InvokeEndpoint = common.String(invokeEndpoint)
			return ocifunctions.GetFunctionResponse{Function: current}, nil
		},
		updateFunctionFn: func(context.Context, ocifunctions.UpdateFunctionRequest) (ocifunctions.UpdateFunctionResponse, error) {
			t.Fatal("UpdateFunction() should not be called when mutable fields do not drift")
			return ocifunctions.UpdateFunctionResponse{}, nil
		},
	}

	manager := &FunctionsFunctionServiceManager{
		CredentialClient: credentials,
		Log:              testFunctionsLogger(),
		ociClient:        client,
	}

	resource := testFunctionResource()
	resource.Status.Id = existingID
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful steady state", response)
	}
	if credentials.getRecordCalls != 1 {
		t.Fatalf("GetSecretRecord() calls = %d, want 1", credentials.getRecordCalls)
	}
	if credentials.createCalls != 1 {
		t.Fatalf("CreateSecret() calls = %d, want 1", credentials.createCalls)
	}
	if credentials.updateIfCurrentCalls != 0 {
		t.Fatalf("UpdateSecretIfCurrent() calls = %d, want 0", credentials.updateIfCurrentCalls)
	}
}

func TestFunctionsFunctionCreateOrUpdateUpdatesOwnedEndpointSecretWhenActive(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.fnfunc.oc1..existing"
	const invokeEndpoint = "https://invoke.functions.example.com/20181201/functions/ocid1.fnfunc.oc1..existing/actions/invoke"

	credentials := &recordingFunctionsCredentialClient{
		getSecretRecordFn: func(_ context.Context, name string, namespace string) (credhelper.SecretRecord, error) {
			requireFunctionSecretTarget(t, "GetSecretRecord", name, namespace)
			return credhelper.SecretRecord{
				UID: types.UID("endpoint-secret-uid"),
				Labels: map[string]string{
					functionsFunctionSecretOwnerUIDLabel: "function-uid",
				},
				Data: map[string][]byte{
					"functionId":     []byte(existingID),
					"invokeEndpoint": []byte("https://stale.functions.example.com"),
				},
			}, nil
		},
		createSecretFn: func(context.Context, string, string, map[string]string, map[string][]byte) (bool, error) {
			t.Fatal("CreateSecret() should not be called when an owned endpoint secret exists")
			return false, nil
		},
		updateSecretIfCurrentFn: func(
			_ context.Context,
			name string,
			namespace string,
			current credhelper.SecretRecord,
			labels map[string]string,
			data map[string][]byte,
		) (bool, error) {
			requireFunctionSecretTarget(t, "UpdateSecretIfCurrent", name, namespace)
			if current.UID != types.UID("endpoint-secret-uid") {
				t.Fatalf("current UID = %q, want endpoint-secret-uid", current.UID)
			}
			requireFunctionOwnedSecretLabels(t, labels, "function-uid")
			requireFunctionSecretData(t, data, existingID, invokeEndpoint)
			return true, nil
		},
	}
	client := &fakeFunctionsManagementClient{
		getFunctionFn: func(_ context.Context, req ocifunctions.GetFunctionRequest) (ocifunctions.GetFunctionResponse, error) {
			if got := safeFunctionsString(req.FunctionId); got != existingID {
				t.Fatalf("GetFunctionRequest.FunctionId = %q, want existing id", got)
			}
			current := testFunction(existingID, ocifunctions.FunctionLifecycleStateActive)
			current.InvokeEndpoint = common.String(invokeEndpoint)
			return ocifunctions.GetFunctionResponse{Function: current}, nil
		},
		updateFunctionFn: func(context.Context, ocifunctions.UpdateFunctionRequest) (ocifunctions.UpdateFunctionResponse, error) {
			t.Fatal("UpdateFunction() should not be called when mutable fields do not drift")
			return ocifunctions.UpdateFunctionResponse{}, nil
		},
	}

	manager := &FunctionsFunctionServiceManager{
		CredentialClient: credentials,
		Log:              testFunctionsLogger(),
		ociClient:        client,
	}

	resource := testFunctionResource()
	resource.Status.Id = existingID
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful steady state", response)
	}
	if credentials.getRecordCalls != 1 {
		t.Fatalf("GetSecretRecord() calls = %d, want 1", credentials.getRecordCalls)
	}
	if credentials.createCalls != 0 {
		t.Fatalf("CreateSecret() calls = %d, want 0", credentials.createCalls)
	}
	if credentials.updateIfCurrentCalls != 1 {
		t.Fatalf("UpdateSecretIfCurrent() calls = %d, want 1", credentials.updateIfCurrentCalls)
	}
}

func TestFunctionsFunctionCreateOrUpdateRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.fnfunc.oc1..existing"

	client := &fakeFunctionsManagementClient{
		getFunctionFn: func(_ context.Context, req ocifunctions.GetFunctionRequest) (ocifunctions.GetFunctionResponse, error) {
			if got := safeFunctionsString(req.FunctionId); got != existingID {
				t.Fatalf("GetFunctionRequest.FunctionId = %q, want existing id", got)
			}
			current := testFunction(existingID, ocifunctions.FunctionLifecycleStateActive)
			current.DisplayName = common.String("other-function")
			return ocifunctions.GetFunctionResponse{Function: current}, nil
		},
		updateFunctionFn: func(context.Context, ocifunctions.UpdateFunctionRequest) (ocifunctions.UpdateFunctionResponse, error) {
			t.Fatal("UpdateFunction() should not be called for create-only drift")
			return ocifunctions.UpdateFunctionResponse{}, nil
		},
	}

	manager := &FunctionsFunctionServiceManager{
		Log:       testFunctionsLogger(),
		ociClient: client,
	}

	resource := testFunctionResource()
	resource.Status.Id = existingID
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if !strings.Contains(err.Error(), "create-only field drift") || !strings.Contains(err.Error(), "displayName") {
		t.Fatalf("CreateOrUpdate() error = %v, want displayName create-only drift", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful response", response)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Failed) {
		t.Fatalf("status.reason = %q, want Failed", got)
	}
}

func TestFunctionsFunctionCreateOrUpdateRejectsOmittedSourceDetailsDriftBeforeMutableUpdate(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.fnfunc.oc1..existing"

	updateCalls := 0
	client := &fakeFunctionsManagementClient{
		getFunctionFn: func(_ context.Context, req ocifunctions.GetFunctionRequest) (ocifunctions.GetFunctionResponse, error) {
			if got := safeFunctionsString(req.FunctionId); got != existingID {
				t.Fatalf("GetFunctionRequest.FunctionId = %q, want existing id", got)
			}
			current := testFunction(existingID, ocifunctions.FunctionLifecycleStateActive)
			current.MemoryInMBs = common.Int64(128)
			current.SourceDetails = ocifunctions.PreBuiltFunctionSourceDetails{
				PbfListingId: common.String("ocid1.fnprebuiltlisting.oc1..existing"),
			}
			return ocifunctions.GetFunctionResponse{Function: current}, nil
		},
		updateFunctionFn: func(context.Context, ocifunctions.UpdateFunctionRequest) (ocifunctions.UpdateFunctionResponse, error) {
			updateCalls++
			t.Fatal("UpdateFunction() should not be called when omitted sourceDetails drifts")
			return ocifunctions.UpdateFunctionResponse{}, nil
		},
	}

	manager := &FunctionsFunctionServiceManager{
		Log:       testFunctionsLogger(),
		ociClient: client,
	}

	resource := testFunctionResource()
	resource.Status.Id = existingID
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want sourceDetails create-only drift rejection")
	}
	if !strings.Contains(err.Error(), "create-only field drift") || !strings.Contains(err.Error(), "sourceDetails") {
		t.Fatalf("CreateOrUpdate() error = %v, want sourceDetails create-only drift", err)
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateFunction() calls = %d, want 0", updateCalls)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful response", response)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Failed) {
		t.Fatalf("status.reason = %q, want Failed", got)
	}
}

func TestFunctionsFunctionCreateOrUpdateLifecycleMapping(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		state       ocifunctions.FunctionLifecycleStateEnum
		wantReason  shared.OSOKConditionType
		wantSuccess bool
		wantRequeue bool
	}{
		{name: "creating", state: ocifunctions.FunctionLifecycleStateCreating, wantReason: shared.Provisioning, wantSuccess: true, wantRequeue: true},
		{name: "active", state: ocifunctions.FunctionLifecycleStateActive, wantReason: shared.Active, wantSuccess: true, wantRequeue: false},
		{name: "inactive", state: ocifunctions.FunctionLifecycleStateInactive, wantReason: shared.Active, wantSuccess: true, wantRequeue: false},
		{name: "updating", state: ocifunctions.FunctionLifecycleStateUpdating, wantReason: shared.Updating, wantSuccess: true, wantRequeue: true},
		{name: "deleting", state: ocifunctions.FunctionLifecycleStateDeleting, wantReason: shared.Terminating, wantSuccess: true, wantRequeue: true},
		{name: "failed", state: ocifunctions.FunctionLifecycleStateFailed, wantReason: shared.Failed, wantSuccess: false, wantRequeue: false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			status := &shared.OSOKStatus{}
			response := reconcileFunctionsFunctionLifecycle(status, &ocifunctions.Function{
				Id:             common.String("ocid1.fnfunc.oc1..lifecycle"),
				DisplayName:    common.String("sample-function"),
				LifecycleState: tc.state,
			}, testFunctionsLogger())

			if response.IsSuccessful != tc.wantSuccess {
				t.Fatalf("IsSuccessful = %t, want %t", response.IsSuccessful, tc.wantSuccess)
			}
			if response.ShouldRequeue != tc.wantRequeue {
				t.Fatalf("ShouldRequeue = %t, want %t", response.ShouldRequeue, tc.wantRequeue)
			}
			if status.Reason != string(tc.wantReason) {
				t.Fatalf("status.reason = %q, want %q", status.Reason, tc.wantReason)
			}
		})
	}
}

func TestFunctionsFunctionDeleteKeepsFinalizerWhileDeleting(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.fnfunc.oc1..tracked"

	credentials := &fakeFunctionsCredentialClient{}
	deleteCalls := 0
	client := &fakeFunctionsManagementClient{
		deleteFunctionFn: func(_ context.Context, req ocifunctions.DeleteFunctionRequest) (ocifunctions.DeleteFunctionResponse, error) {
			deleteCalls++
			if got := safeFunctionsString(req.FunctionId); got != existingID {
				t.Fatalf("DeleteFunctionRequest.FunctionId = %q, want tracked id", got)
			}
			return ocifunctions.DeleteFunctionResponse{OpcRequestId: common.String("opc-delete-function-1")}, nil
		},
		getFunctionFn: func(_ context.Context, req ocifunctions.GetFunctionRequest) (ocifunctions.GetFunctionResponse, error) {
			if got := safeFunctionsString(req.FunctionId); got != existingID {
				t.Fatalf("GetFunctionRequest.FunctionId = %q, want tracked id", got)
			}
			return ocifunctions.GetFunctionResponse{
				Function: testFunction(existingID, ocifunctions.FunctionLifecycleStateDeleting),
			}, nil
		},
	}

	manager := &FunctionsFunctionServiceManager{
		CredentialClient: credentials,
		Log:              testFunctionsLogger(),
		ociClient:        client,
	}

	resource := testFunctionResource()
	resource.Status.Id = existingID
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)

	deleted, err := manager.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() should keep the finalizer while OCI reports DELETING")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteFunction() calls = %d, want 1", deleteCalls)
	}
	if credentials.deleteCalls != 0 {
		t.Fatalf("DeleteSecret() calls = %d, want 0 before delete confirmation", credentials.deleteCalls)
	}
	if got := resource.Status.LifecycleState; got != "DELETING" {
		t.Fatalf("status.lifecycleState = %q, want DELETING", got)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Terminating) {
		t.Fatalf("status.reason = %q, want Terminating", got)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt = %#v, want nil while delete is pending", resource.Status.OsokStatus.DeletedAt)
	}
}

func TestFunctionsFunctionDeleteUsesListFallbackWhenStatusIDMissing(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.fnfunc.oc1..listed"

	credentials := &fakeFunctionsCredentialClient{}
	getCalls := 0
	deleteCalls := 0
	client := &fakeFunctionsManagementClient{
		listFunctionsFn: func(_ context.Context, req ocifunctions.ListFunctionsRequest) (ocifunctions.ListFunctionsResponse, error) {
			if got := safeFunctionsString(req.ApplicationId); got != "ocid1.application.oc1..example" {
				t.Fatalf("ListFunctionsRequest.ApplicationId = %q, want application id", got)
			}
			return ocifunctions.ListFunctionsResponse{
				Items: []ocifunctions.FunctionSummary{
					testFunctionSummary(existingID, ocifunctions.FunctionLifecycleStateActive),
				},
			}, nil
		},
		getFunctionFn: func(_ context.Context, req ocifunctions.GetFunctionRequest) (ocifunctions.GetFunctionResponse, error) {
			getCalls++
			if got := safeFunctionsString(req.FunctionId); got != existingID {
				t.Fatalf("GetFunctionRequest.FunctionId = %q, want listed id", got)
			}
			if getCalls == 1 {
				return ocifunctions.GetFunctionResponse{
					Function: testFunction(existingID, ocifunctions.FunctionLifecycleStateActive),
				}, nil
			}
			return ocifunctions.GetFunctionResponse{}, errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "function missing")
		},
		deleteFunctionFn: func(_ context.Context, req ocifunctions.DeleteFunctionRequest) (ocifunctions.DeleteFunctionResponse, error) {
			deleteCalls++
			if got := safeFunctionsString(req.FunctionId); got != existingID {
				t.Fatalf("DeleteFunctionRequest.FunctionId = %q, want listed id", got)
			}
			return ocifunctions.DeleteFunctionResponse{OpcRequestId: common.String("opc-delete-function-1")}, nil
		},
	}

	manager := &FunctionsFunctionServiceManager{
		CredentialClient: credentials,
		Log:              testFunctionsLogger(),
		ociClient:        client,
	}

	resource := testFunctionResource()

	deleted, err := manager.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() should release the finalizer after fallback delete confirmation")
	}
	if getCalls != 2 {
		t.Fatalf("GetFunction() calls = %d, want 2", getCalls)
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteFunction() calls = %d, want 1", deleteCalls)
	}
	if credentials.deleteCalls != 1 {
		t.Fatalf("DeleteSecret() calls = %d, want 1 after delete confirmation", credentials.deleteCalls)
	}
	if resource.Status.Id != "" {
		t.Fatalf("status.id = %q, want cleared tracked id", resource.Status.Id)
	}
	if resource.Status.OsokStatus.Ocid != "" {
		t.Fatalf("status.ocid = %q, want cleared tracked ocid", resource.Status.OsokStatus.Ocid)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete timestamp")
	}
}

func requireFunctionSecretTarget(t *testing.T, action string, name string, namespace string) {
	t.Helper()
	if name != "sample-function" || namespace != "default" {
		t.Fatalf("%s() target = %s/%s, want default/sample-function", action, namespace, name)
	}
}

func requireFunctionOwnedSecretLabels(t *testing.T, labels map[string]string, wantUID string) {
	t.Helper()
	if got := labels[functionsFunctionSecretOwnerUIDLabel]; got != wantUID {
		t.Fatalf("secret owner label = %q, want %q", got, wantUID)
	}
}

func requireFunctionSecretData(t *testing.T, data map[string][]byte, wantFunctionID string, wantInvokeEndpoint string) {
	t.Helper()
	if got := string(data["functionId"]); got != wantFunctionID {
		t.Fatalf("secret functionId = %q, want %q", got, wantFunctionID)
	}
	if got := string(data["invokeEndpoint"]); got != wantInvokeEndpoint {
		t.Fatalf("secret invokeEndpoint = %q, want %q", got, wantInvokeEndpoint)
	}
}

func testFunctionsLogger() loggerutil.OSOKLogger {
	return loggerutil.OSOKLogger{Logger: logr.Discard()}
}

func requireTrailingFunctionsCondition(t *testing.T, status shared.OSOKStatus, want shared.OSOKConditionType) {
	t.Helper()

	if len(status.Conditions) == 0 {
		t.Fatalf("status.conditions = nil, want trailing %s condition", want)
	}
	if got := status.Conditions[len(status.Conditions)-1].Type; got != want {
		t.Fatalf("trailing status condition = %s, want %s", got, want)
	}
}

func testApplicationResource() *functionsv1beta1.Application {
	return &functionsv1beta1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sample-app",
			Namespace: "default",
		},
		Spec: functionsv1beta1.ApplicationSpec{
			CompartmentId: "ocid1.compartment.oc1..example",
			DisplayName:   "sample-app",
			SubnetIds:     []string{"ocid1.subnet.oc1..example"},
		},
	}
}

func testFunctionResource() *functionsv1beta1.Function {
	return &functionsv1beta1.Function{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sample-function",
			Namespace: "default",
			UID:       types.UID("function-uid"),
		},
		Spec: functionsv1beta1.FunctionSpec{
			DisplayName:   "sample-function",
			ApplicationId: "ocid1.application.oc1..example",
			MemoryInMBs:   256,
			Image:         "phx.ocir.io/tenant/functions/sample-function:1.0.0",
		},
	}
}

func testFunction(id string, lifecycleState ocifunctions.FunctionLifecycleStateEnum) ocifunctions.Function {
	return ocifunctions.Function{
		Id:             common.String(id),
		DisplayName:    common.String("sample-function"),
		ApplicationId:  common.String("ocid1.application.oc1..example"),
		CompartmentId:  common.String("ocid1.compartment.oc1..example"),
		Image:          common.String("phx.ocir.io/tenant/functions/sample-function:1.0.0"),
		MemoryInMBs:    common.Int64(256),
		LifecycleState: lifecycleState,
	}
}

func testFunctionSummary(id string, lifecycleState ocifunctions.FunctionLifecycleStateEnum) ocifunctions.FunctionSummary {
	return ocifunctions.FunctionSummary{
		Id:             common.String(id),
		DisplayName:    common.String("sample-function"),
		ApplicationId:  common.String("ocid1.application.oc1..example"),
		CompartmentId:  common.String("ocid1.compartment.oc1..example"),
		Image:          common.String("phx.ocir.io/tenant/functions/sample-function:1.0.0"),
		MemoryInMBs:    common.Int64(256),
		LifecycleState: lifecycleState,
	}
}

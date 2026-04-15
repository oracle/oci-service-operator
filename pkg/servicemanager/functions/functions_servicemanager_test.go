package functions

import (
	"context"
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

func testFunctionsLogger() loggerutil.OSOKLogger {
	return loggerutil.OSOKLogger{Logger: logr.Discard()}
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

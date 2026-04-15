package instance

import (
	"context"
	"testing"

	coresdk "github.com/oracle/oci-go-sdk/v65/core"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestInstancePlainGeneratedRuntimeCreateErrorMatrix(t *testing.T) {
	t.Parallel()

	errortest.RunGeneratedRuntimePlainCreateMatrix(t, func(_ *testing.T, candidate errortest.CommonErrorCase) errortest.GeneratedRuntimePlainMutationResult {
		manager := newInstanceTestManager(&fakeInstanceOCIClient{
			launchFn: func(context.Context, coresdk.LaunchInstanceRequest) (coresdk.LaunchInstanceResponse, error) {
				return coresdk.LaunchInstanceResponse{}, errortest.NewServiceErrorFromCase(candidate)
			},
		})
		resource := makeSpecInstance()

		response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
		return errortest.GeneratedRuntimePlainMutationResult{
			Response:     response,
			Err:          err,
			StatusReason: resource.Status.OsokStatus.Reason,
		}
	})
}

func TestInstancePlainGeneratedRuntimeDeleteErrorMatrix(t *testing.T) {
	t.Parallel()

	errortest.RunGeneratedRuntimePlainDeleteMatrix(t, func(t *testing.T, candidate errortest.CommonErrorCase) errortest.GeneratedRuntimePlainDeleteResult {
		getCalls := 0

		manager := newInstanceTestManager(&fakeInstanceOCIClient{
			terminateFn: func(context.Context, coresdk.TerminateInstanceRequest) (coresdk.TerminateInstanceResponse, error) {
				return coresdk.TerminateInstanceResponse{}, errortest.NewServiceErrorFromCase(candidate)
			},
			getFn: func(context.Context, coresdk.GetInstanceRequest) (coresdk.GetInstanceResponse, error) {
				getCalls++
				lifecycleState := coresdk.InstanceLifecycleStateRunning
				if getCalls > 1 && errortest.GeneratedRuntimePlainDeleteRequiresConfirmRead(candidate) {
					lifecycleState = coresdk.InstanceLifecycleStateTerminating
				}
				return coresdk.GetInstanceResponse{
					Instance: makeSDKInstance("ocid1.instance.oc1..matrix", lifecycleState, map[string]string{
						"matrix": "delete",
					}),
				}, nil
			},
		})

		resource := makeSpecInstance()
		resource.Status.Id = "ocid1.instance.oc1..matrix"
		resource.Status.OsokStatus.Ocid = "ocid1.instance.oc1..matrix"

		deleted, err := manager.Delete(context.Background(), resource)
		return errortest.GeneratedRuntimePlainDeleteResult{
			Deleted:      deleted,
			Err:          err,
			StatusReason: resource.Status.OsokStatus.Reason,
		}
	})
}

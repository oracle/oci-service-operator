package nodepool

import (
	"context"
	"testing"

	containerenginesdk "github.com/oracle/oci-go-sdk/v65/containerengine"
	containerenginev1beta1 "github.com/oracle/oci-service-operator/api/containerengine/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestNodePoolPlainGeneratedRuntimeCreateErrorMatrix(t *testing.T) {
	t.Parallel()

	errortest.RunGeneratedRuntimePlainCreateMatrix(t, func(_ *testing.T, candidate errortest.CommonErrorCase) errortest.GeneratedRuntimePlainMutationResult {
		manager := newNodePoolRuntimeTestManager(generatedruntime.Config[*containerenginev1beta1.NodePool]{
			Create: &generatedruntime.Operation{
				NewRequest: func() any { return &containerenginesdk.CreateNodePoolRequest{} },
				Call: func(_ context.Context, _ any) (any, error) {
					return containerenginesdk.CreateNodePoolResponse{}, errortest.NewServiceErrorFromCase(candidate)
				},
				Fields: nodePoolCreateFields(),
			},
		})

		resource := newNodePoolTestResource()
		response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
		return errortest.GeneratedRuntimePlainMutationResult{
			Response:     response,
			Err:          err,
			StatusReason: resource.Status.OsokStatus.Reason,
		}
	})
}

func TestNodePoolPlainGeneratedRuntimeDeleteErrorMatrix(t *testing.T) {
	t.Parallel()

	errortest.RunGeneratedRuntimePlainDeleteMatrix(t, func(_ *testing.T, candidate errortest.CommonErrorCase) errortest.GeneratedRuntimePlainDeleteResult {
		const existingID = "ocid1.nodepool.oc1..matrix"

		resource := newExistingNodePoolTestResource(existingID)
		getCalls := 0

		manager := newNodePoolRuntimeTestManager(generatedruntime.Config[*containerenginev1beta1.NodePool]{
			Get: &generatedruntime.Operation{
				NewRequest: func() any { return &containerenginesdk.GetNodePoolRequest{} },
				Call: func(_ context.Context, _ any) (any, error) {
					getCalls++
					lifecycleState := "ACTIVE"
					if getCalls > 1 && errortest.GeneratedRuntimePlainDeleteRequiresConfirmRead(candidate) {
						lifecycleState = "DELETING"
					}
					return containerenginesdk.GetNodePoolResponse{
						NodePool: observedNodePoolFromSpec(existingID, resource.Spec, lifecycleState),
					}, nil
				},
				Fields: nodePoolGetFields(),
			},
			Delete: &generatedruntime.Operation{
				NewRequest: func() any { return &containerenginesdk.DeleteNodePoolRequest{} },
				Call: func(_ context.Context, _ any) (any, error) {
					return containerenginesdk.DeleteNodePoolResponse{}, errortest.NewServiceErrorFromCase(candidate)
				},
				Fields: nodePoolDeleteFields(),
			},
		})

		deleted, err := manager.Delete(context.Background(), resource)
		return errortest.GeneratedRuntimePlainDeleteResult{
			Deleted:      deleted,
			Err:          err,
			StatusReason: resource.Status.OsokStatus.Reason,
		}
	})
}

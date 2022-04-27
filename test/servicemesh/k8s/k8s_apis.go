/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package k8s

import (
	"context"
	"time"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	meshCommons "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
)

// K8sAPIs supports common k8s operations for each kind of resource.
// It will be added to the test framework to facilitate tests.
type K8sAPIs interface {
	Get(ctx context.Context, key types.NamespacedName, obj client.Object) error
	Create(ctx context.Context, obj client.Object) error
	Update(ctx context.Context, oldObj client.Object, newObj client.Object) error
	Delete(ctx context.Context, obj client.Object) error
	WaitUntilDeleted(ctx context.Context, obj client.Object) error
}

type defaultK8sAPIs struct {
	k8sClient client.Client
}

func NewDefaultK8sAPIs(k8sClient client.Client) K8sAPIs {
	return &defaultK8sAPIs{k8sClient: k8sClient}
}

func (h *defaultK8sAPIs) Get(ctx context.Context, key types.NamespacedName, obj client.Object) error {
	err := h.k8sClient.Get(ctx, key, obj)
	if err != nil {
		return err
	}
	return nil
}

func (h *defaultK8sAPIs) Create(ctx context.Context, obj client.Object) error {
	err := h.k8sClient.Create(ctx, obj)
	if err != nil {
		return err
	}

	if _, err := h.waitUntilActive(ctx, obj); err != nil {
		return err
	}

	return nil
}

func (h *defaultK8sAPIs) Update(ctx context.Context, newObj client.Object, oldObj client.Object) error {
	err := h.k8sClient.Patch(ctx, newObj, client.MergeFrom(oldObj))
	if err != nil {
		return err
	}

	if _, err := h.waitUntilActive(ctx, newObj); err != nil {
		return err
	}

	return nil
}

func (h *defaultK8sAPIs) Delete(ctx context.Context, obj client.Object) error {
	err := h.k8sClient.Delete(ctx, obj)
	if err != nil {
		return err
	}

	if err := h.WaitUntilDeleted(ctx, obj); err != nil {
		return err
	}

	return nil
}

func (h *defaultK8sAPIs) waitUntilActive(ctx context.Context, obj client.Object) (client.Object, error) {
	key := newNamespacedName(obj)
	observedObj := obj
	return observedObj, wait.PollImmediateUntil(meshCommons.PollInterval, func() (bool, error) {
		// Sometimes there's a delay in the resource showing up
		// We need to make sure the resource is in k8s before we send test requests to the reconciler
		for i := 0; i < 5; i++ {
			if err := h.k8sClient.Get(ctx, key, observedObj); err != nil {
				if i >= 5 {
					return false, err
				}
			}

			time.Sleep(100 * time.Millisecond)
		}

		if IsValidMeshResource(obj.GetObjectKind().GroupVersionKind().Kind) {
			if !isActive(obj) {
				return false, nil
			}
		}
		return true, nil
	}, ctx.Done())
}

func (h *defaultK8sAPIs) WaitUntilDeleted(ctx context.Context, obj client.Object) error {
	key := newNamespacedName(obj)
	observedObj := obj
	return wait.PollImmediateUntil(meshCommons.PollInterval, func() (bool, error) {
		if err := h.k8sClient.Get(ctx, key, observedObj); err != nil {
			if kerrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	}, ctx.Done())
}

func newNamespacedName(obj metav1.Object) types.NamespacedName {
	return types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}
}

func isActive(obj client.Object) bool {

	if string(meshCommons.Mesh) == obj.GetObjectKind().GroupVersionKind().Kind {
		mesh := obj.(*servicemeshapi.Mesh)
		for _, condition := range mesh.Status.Conditions {
			if condition.Status == metav1.ConditionTrue {
				return true
			}
		}
		return false
	}

	if string(meshCommons.VirtualServiceRouteTable) == obj.GetObjectKind().GroupVersionKind().Kind {
		virtualServiceRouteTable := obj.(*servicemeshapi.VirtualServiceRouteTable)
		for _, condition := range virtualServiceRouteTable.Status.Conditions {
			if condition.Status == metav1.ConditionTrue {
				return true
			}
		}
		return false
	}

	if string(meshCommons.VirtualService) == obj.GetObjectKind().GroupVersionKind().Kind {
		virtualService := obj.(*servicemeshapi.VirtualService)
		for _, condition := range virtualService.Status.Conditions {
			if condition.Status == metav1.ConditionTrue {
				return true
			}
		}
		return false
	}

	if string(meshCommons.VirtualDeploymentBinding) == obj.GetObjectKind().GroupVersionKind().Kind {
		virtualDeploymentBinding := obj.(*servicemeshapi.VirtualDeploymentBinding)
		for _, condition := range virtualDeploymentBinding.Status.Conditions {
			if condition.Status == metav1.ConditionTrue {
				return true
			}
		}
		return false
	}

	if string(meshCommons.VirtualDeployment) == obj.GetObjectKind().GroupVersionKind().Kind {
		virtualDeployment := obj.(*servicemeshapi.VirtualDeployment)
		for _, condition := range virtualDeployment.Status.Conditions {
			if condition.Status == metav1.ConditionTrue {
				return true
			}
		}
		return false
	}

	if string(meshCommons.AccessPolicy) == obj.GetObjectKind().GroupVersionKind().Kind {
		accessPolicy := obj.(*servicemeshapi.AccessPolicy)
		for _, condition := range accessPolicy.Status.Conditions {
			if condition.Status == metav1.ConditionTrue {
				return true
			}
		}
		return false
	}

	if string(meshCommons.IngressGatewayRouteTable) == obj.GetObjectKind().GroupVersionKind().Kind {
		ingressGatewayRouteTable := obj.(*servicemeshapi.IngressGatewayRouteTable)
		for _, condition := range ingressGatewayRouteTable.Status.Conditions {
			if condition.Status == metav1.ConditionTrue {
				return true
			}
		}
		return false
	}

	if string(meshCommons.IngressGatewayDeployment) == obj.GetObjectKind().GroupVersionKind().Kind {
		ingressGatewayDeployment := obj.(*servicemeshapi.IngressGatewayDeployment)
		for _, condition := range ingressGatewayDeployment.Status.Conditions {
			if condition.Status == metav1.ConditionTrue {
				return true
			}
		}
		return false
	}

	if string(meshCommons.IngressGateway) == obj.GetObjectKind().GroupVersionKind().Kind {
		ingressGateway := obj.(*servicemeshapi.IngressGateway)
		for _, condition := range ingressGateway.Status.Conditions {
			if condition.Status == metav1.ConditionTrue {
				return true
			}
		}
		return false
	}

	return false
}

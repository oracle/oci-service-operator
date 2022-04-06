/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package virtualdeployment

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/oracle/oci-service-operator/api/v1beta1"
	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
)

func NewVd(id string, compartment string, logging servicemeshapi.AccessLogging) *servicemeshapi.VirtualDeployment {
	vd := servicemeshapi.VirtualDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: servicemeshapi.VirtualDeploymentSpec{
			CompartmentId: "my-compartment",
			VirtualService: servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Name: "my-vs",
				},
			},
		},
	}
	if id != "" {
		vd.Status.VirtualDeploymentId = api.OCID(id)
	}
	if compartment != "" {
		vd.Spec.CompartmentId = api.OCID(compartment)
	}
	vd.Spec.AccessLogging = &logging
	return &vd
}

func NewTestEnvVd(name string, namespace string) *servicemeshapi.VirtualDeployment {
	now := metav1.Now()
	return &servicemeshapi.VirtualDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: servicemeshapi.VirtualDeploymentSpec{
			CompartmentId: "my-compartment",
			VirtualService: servicemeshapi.RefOrId{
				Id: "my-service-id",
				ResourceRef: &servicemeshapi.ResourceRef{
					Name:      "my-vs",
					Namespace: namespace,
				},
			},
			Listener: []servicemeshapi.Listener{
				{
					Protocol: "HTTP",
					Port:     80,
				},
			},
			AccessLogging: &servicemeshapi.AccessLogging{
				IsEnabled: true,
			},
			ServiceDiscovery: servicemeshapi.ServiceDiscovery{
				Type: servicemeshapi.ServiceDiscoveryTypeDns,
			},
			TagResources: api.TagResources{
				FreeFormTags: map[string]string{"freeformTag1": "value1"},
				DefinedTags: map[string]api.MapValue{
					"definedTag1": {"foo": "bar"},
				},
			},
		},
		Status: servicemeshapi.ServiceMeshStatus{
			VirtualDeploymentId: "vd",
			Conditions: []servicemeshapi.ServiceMeshCondition{
				{
					Type: servicemeshapi.ServiceMeshActive,
					ResourceCondition: servicemeshapi.ResourceCondition{
						Status:             metav1.ConditionTrue,
						LastTransitionTime: &now,
						Reason:             "Successful",
						Message:            "",
					},
				},
			},
		},
	}
}

func WaitUntilVirtualDeploymentActive(ctx context.Context, client client.Client, vd *servicemeshapi.VirtualDeployment) (*servicemeshapi.VirtualDeployment, error) {
	observedVd := &servicemeshapi.VirtualDeployment{}
	return observedVd, wait.PollImmediateUntil(commons.PollInterval, func() (bool, error) {

		// sometimes there's a delay in the resource showing up
		for i := 0; i < 5; i++ {
			if err := client.Get(ctx, types.NamespacedName{Namespace: vd.Namespace, Name: vd.Name}, observedVd); err != nil {
				if i >= 5 {
					return false, err
				}
			}
			time.Sleep(100 * time.Millisecond)
		}

		for _, condition := range observedVd.Status.Conditions {
			if condition.Type == servicemeshapi.ServiceMeshActive && condition.Status == metav1.ConditionTrue {
				return true, nil
			}
		}

		return false, nil
	}, ctx.Done())
}

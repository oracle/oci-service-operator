/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package virtualdeploymentbinding

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/oracle/oci-service-operator/api/v1beta1"
	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
)

func NewVdbWithVdRef(name string, namespace string, virtualDeployment string, service string) *servicemeshapi.VirtualDeploymentBinding {
	return NewVdbWithVdRefWithServiceNamespace(name, namespace, virtualDeployment, service, namespace)
}

func NewVdbWithVdRefWithServiceNamespace(name string, namespace string, virtualDeployment string, service string, serviceNameSpace string) *servicemeshapi.VirtualDeploymentBinding {

	virtualDeploymentRefOrId := servicemeshapi.RefOrId{
		ResourceRef: &servicemeshapi.ResourceRef{
			Name: servicemeshapi.Name(virtualDeployment),
		},
	}

	target := servicemeshapi.Target{
		Service: servicemeshapi.Service{
			ServiceRef: servicemeshapi.ResourceRef{
				Name: servicemeshapi.Name(service),
			},
		},
	}

	if serviceNameSpace != "" {
		target = servicemeshapi.Target{
			Service: servicemeshapi.Service{
				ServiceRef: servicemeshapi.ResourceRef{
					Name:      servicemeshapi.Name(service),
					Namespace: serviceNameSpace,
				},
			},
		}
	}

	return &servicemeshapi.VirtualDeploymentBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: servicemeshapi.VirtualDeploymentBindingSpec{
			VirtualDeployment: virtualDeploymentRefOrId,
			Target:            target,
		},
	}
}

func NewVdbWithVdOcid(name string, namespace string, vdOcid string, service string) *servicemeshapi.VirtualDeploymentBinding {

	virtualDeploymentRefOrId := servicemeshapi.RefOrId{
		Id: api.OCID(vdOcid),
	}

	target := servicemeshapi.Target{
		Service: servicemeshapi.Service{
			ServiceRef: servicemeshapi.ResourceRef{
				Name: servicemeshapi.Name(service),
			},
		},
	}

	return &servicemeshapi.VirtualDeploymentBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: servicemeshapi.VirtualDeploymentBindingSpec{
			VirtualDeployment: virtualDeploymentRefOrId,
			Target:            target,
		},
	}
}

func CreateVirtualDeploymentBinding(ctx context.Context, client client.Client, key types.NamespacedName, vdb *servicemeshapi.VirtualDeploymentBinding) error {
	err := client.Get(ctx, key, vdb)
	if err != nil {
		err = client.Create(ctx, vdb)
	}
	return err
}

func UpdateVirtualDeploymentBinding(ctx context.Context, client client.Client, vdb *servicemeshapi.VirtualDeploymentBinding) error {
	return client.Update(ctx, vdb)
}

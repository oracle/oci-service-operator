/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package virtualdeploymentbinding

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"context"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	podUtil "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/pod"

	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ListVDB returns a list of virtual deployment binding for all namespaces
func ListVDB(ctx context.Context, r client.Reader) (*servicemeshapi.VirtualDeploymentBindingList, error) {
	virtualDeploymentBindingList := &servicemeshapi.VirtualDeploymentBindingList{}
	listOpts := &client.ListOptions{}
	err := r.List(ctx, virtualDeploymentBindingList, listOpts)
	return virtualDeploymentBindingList, err
}

// GetVDB returns the vdb object for a given name and namespace
func GetVDB(ctx context.Context, r client.Client, namespace string, name string, vdb *servicemeshapi.VirtualDeploymentBinding) error {
	return r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, vdb)
}

// FilterVDBsByServiceRef return a filtered list of VDBs from the input list that refers to the given service
func FilterVDBsByServiceRef(vdbs *servicemeshapi.VirtualDeploymentBindingList, serviceName string, serviceNamespace string) []servicemeshapi.VirtualDeploymentBinding {
	filteredVDBs := []servicemeshapi.VirtualDeploymentBinding{}
	for i := range vdbs.Items {
		serviceRef := &vdbs.Items[i].Spec.Target.Service.ServiceRef
		serviceRefNamespace := GetServiceRefNamespace(&vdbs.Items[i])
		if serviceRef.Name == servicemeshapi.Name(serviceName) && serviceRefNamespace == serviceNamespace {
			filteredVDBs = append(filteredVDBs, vdbs.Items[i])
		}
	}
	return filteredVDBs
}

// GetServiceRefNamespace returns the namespace of the serviceRef from the VDB resource
func GetServiceRefNamespace(vdb *servicemeshapi.VirtualDeploymentBinding) string {
	serviceRefNamespace := vdb.Spec.Target.Service.ServiceRef.Namespace
	if serviceRefNamespace == "" {
		serviceRefNamespace = vdb.Namespace
	}
	return serviceRefNamespace
}

// GetVDBForPod checks for which vdb does the "vdb.Spec.Target.Service.MatchLabels" match the given Pod labels
func GetVDBForPod(vdbs []servicemeshapi.VirtualDeploymentBinding, podLabels labels.Set) *servicemeshapi.VirtualDeploymentBinding {
	for _, vdb := range vdbs {
		vdbLabels := labels.Set(vdb.Spec.Target.Service.MatchLabels)
		if podUtil.MatchLabels(podLabels, vdbLabels) {
			return &vdb
		}
	}
	return nil
}

func IsVirtualDeploymentBindingActive(virtualDeploymentBinding *servicemeshapi.VirtualDeploymentBinding) bool {
	if virtualDeploymentBinding.Status.Conditions == nil {
		return false
	}
	for _, condition := range virtualDeploymentBinding.Status.Conditions {
		if condition.Status != metav1.ConditionTrue {
			return false
		}
	}
	return true
}

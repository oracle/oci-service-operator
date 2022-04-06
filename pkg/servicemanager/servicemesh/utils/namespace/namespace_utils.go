/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package namespace

import (
	"context"
	"strings"

	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ListServiceMeshEnabledNamespaces returns a list of namespaces that contain service proxy injection label
func ListServiceMeshEnabledNamespaces(ctx context.Context, cl client.Client) (*corev1.NamespaceList, error) {
	serviceMeshEnabledNamespaceList := &corev1.NamespaceList{}
	listOpts := []client.ListOption{
		client.HasLabels{commons.ProxyInjectionLabel},
	}
	err := cl.List(ctx, serviceMeshEnabledNamespaceList, listOpts...)
	return serviceMeshEnabledNamespaceList, err
}

// Get NamespacedName and valid boolean for a given string,
func NewNamespacedNameFromString(s string) (types.NamespacedName, bool) {
	nn := types.NamespacedName{}
	result := strings.Split(s, "/")
	if len(result) == 2 {
		nn.Namespace = result[0]
		nn.Name = result[1]
		return nn, true
	}
	return nn, false
}

func NewNamespacedName(obj metav1.Object) types.NamespacedName {
	return types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}
}

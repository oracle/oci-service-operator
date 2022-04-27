/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package equality

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
)

func TestIgnoreFakeClientPopulatedFields(t *testing.T) {
	tests := []struct {
		name       string
		meshLeft   *servicemeshapi.Mesh
		meshRight  *servicemeshapi.Mesh
		wantEquals bool
	}{
		{
			name: "objects should be equal if only TypeMeta, ObjectMeta.ResourceVersion and time diffs",
			meshLeft: &servicemeshapi.Mesh{
				TypeMeta: v1.TypeMeta{
					Kind:       "mesh",
					APIVersion: "servicemesh.oraclecloud.com/v1alpha1",
				},
				ObjectMeta: v1.ObjectMeta{
					ResourceVersion: "0",
					Annotations: map[string]string{
						"k": "v1",
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:             metav1.ConditionFalse,
								LastTransitionTime: &metav1.Time{Time: time.Now()},
							},
						},
					},
				},
			},
			meshRight: &servicemeshapi.Mesh{
				TypeMeta: v1.TypeMeta{
					Kind:       "mesh",
					APIVersion: "servicemesh.oraclecloud.com/servicemeshapi",
				},
				ObjectMeta: v1.ObjectMeta{
					ResourceVersion: "1",
					Annotations: map[string]string{
						"k": "v1",
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:             metav1.ConditionFalse,
								LastTransitionTime: &metav1.Time{Time: time.Now().Add(100 * time.Minute)},
							},
						},
					},
				},
			},
			wantEquals: true,
		},
		{
			name: "objects shouldn't be equal if more fields than TypeMeta and ObjectMeta.ResourceVersion diffs",
			meshLeft: &servicemeshapi.Mesh{
				TypeMeta: v1.TypeMeta{
					Kind:       "mesh",
					APIVersion: "servicemesh.oraclecloud.com/v1alpha1",
				},
				ObjectMeta: v1.ObjectMeta{
					ResourceVersion: "0",
					Annotations: map[string]string{
						"k": "v1",
					},
				},
			},
			meshRight: &servicemeshapi.Mesh{
				TypeMeta: v1.TypeMeta{
					Kind:       "mesh",
					APIVersion: "servicemesh.oraclecloud.com/servicemeshapi",
				},
				ObjectMeta: v1.ObjectMeta{
					ResourceVersion: "1",
					Annotations: map[string]string{
						"k": "v2",
					},
				},
			},
			wantEquals: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := IgnoreFakeClientPopulatedFields()
			gotEquals := cmp.Equal(tt.meshLeft, tt.meshRight, opts)
			assert.Equal(t, tt.wantEquals, gotEquals)
		})
	}
}

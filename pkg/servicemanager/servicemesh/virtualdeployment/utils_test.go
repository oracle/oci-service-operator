/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package virtualdeployment

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	sdk "github.com/oracle/oci-go-sdk/v65/servicemesh"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
)

func TestIsServiceMeshActive(t *testing.T) {
	type args struct {
		virtualDeployment *servicemeshapi.VirtualDeployment
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "virtualDeployment has true ServiceMeshActive condition",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "virtualDeployment has false ServiceMeshActive condition",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionFalse,
								},
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "virtualDeployment has unknown ServiceMeshActive condition",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionUnknown,
								},
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "virtualDeployment doesn't have ServiceMeshActive condition",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: []servicemeshapi.ServiceMeshCondition{},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsVdActiveK8s(tt.args.virtualDeployment)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetVDActiveStatus(t *testing.T) {
	type args struct {
		vd *servicemeshapi.VirtualDeployment
	}
	tests := []struct {
		name string
		args args
		want metav1.ConditionStatus
	}{
		{
			name: "status active present",
			args: args{
				vd: &servicemeshapi.VirtualDeployment{
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionFalse,
								},
							},
						},
					},
				},
			},
			want: metav1.ConditionFalse,
		},
		{
			name: "status active not present",
			args: args{
				vd: &servicemeshapi.VirtualDeployment{
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{},
						},
					},
				},
			},
			want: metav1.ConditionUnknown,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetVDActiveStatus(tt.args.vd); got != tt.want {
				t.Errorf("GetVDActiveStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsVdActiveCp(t *testing.T) {
	type args struct {
		vd *sdk.VirtualDeployment
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "active state",
			args: args{
				vd: &sdk.VirtualDeployment{LifecycleState: sdk.VirtualDeploymentLifecycleStateActive},
			},
			want: true,
		},
		{
			name: "state not active",
			args: args{
				vd: &sdk.VirtualDeployment{LifecycleState: sdk.VirtualDeploymentLifecycleStateFailed},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsVdActiveCp(tt.args.vd); got != tt.want {
				t.Errorf("IsVdActiveCp() = %v, want %v", got, tt.want)
			}
		})
	}
}

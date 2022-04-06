/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package commons

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
)

func TestGetConditionState(t *testing.T) {
	tests := []struct {
		name   string
		state  string
		expect metav1.ConditionStatus
	}{
		{
			name:   "condition true",
			state:  Active,
			expect: metav1.ConditionTrue,
		},
		{
			name:   "condition unknown",
			state:  Creating,
			expect: metav1.ConditionUnknown,
		},
		{
			name:   "condition false",
			state:  Deleted,
			expect: metav1.ConditionFalse,
		},
		{
			name:   "condition unknown",
			state:  Deleting,
			expect: metav1.ConditionUnknown,
		},
		{
			name:   "condition unknown",
			state:  Updating,
			expect: metav1.ConditionUnknown,
		},
		{
			name:   "condition false",
			state:  Failed,
			expect: metav1.ConditionFalse,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetConditionStatus(tt.state)
			assert.Equal(t, tt.expect, got)
		})
	}
}

func TestGetReason(t *testing.T) {
	tests := []struct {
		name   string
		status metav1.ConditionStatus
		expect ResourceConditionReason
	}{
		{
			name:   "successful",
			status: metav1.ConditionTrue,
			expect: Successful,
		},
		{
			name:   "state changed for unknown",
			status: metav1.ConditionUnknown,
			expect: LifecycleStateChanged,
		},
		{
			name:   "state changed for false",
			status: metav1.ConditionFalse,
			expect: LifecycleStateChanged,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetReason(tt.status)
			assert.Equal(t, tt.expect, got)
		})
	}
}

func TestGetMessage(t *testing.T) {
	tests := []struct {
		name   string
		state  string
		expect ResourceConditionMessage
	}{
		{
			name:   "condition true",
			state:  Active,
			expect: ResourceActive,
		},
		{
			name:   "condition unknown",
			state:  Creating,
			expect: ResourceCreating,
		},
		{
			name:   "condition false",
			state:  Deleted,
			expect: ResourceDeleted,
		},
		{
			name:   "condition unknown",
			state:  Deleting,
			expect: ResourceDeleting,
		},
		{
			name:   "condition unknown",
			state:  Updating,
			expect: ResourceUpdating,
		},
		{
			name:   "condition false",
			state:  Failed,
			expect: ResourceFailed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetMessage(tt.state)
			assert.Equal(t, tt.expect, got)
		})
	}
}

func TestGetCondition(t *testing.T) {
	type args struct {
		mesh          *servicemeshapi.Mesh
		conditionType servicemeshapi.ServiceMeshConditionType
	}
	tests := []struct {
		name   string
		args   args
		expect *servicemeshapi.ServiceMeshCondition
	}{
		{
			name: "condition found",
			args: args{
				mesh: &servicemeshapi.Mesh{
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
				conditionType: servicemeshapi.ServiceMeshActive,
			},
			expect: &servicemeshapi.ServiceMeshCondition{
				Type: servicemeshapi.ServiceMeshActive,
				ResourceCondition: servicemeshapi.ResourceCondition{
					Status: metav1.ConditionFalse,
				},
			},
		},
		{
			name: "condition not found",
			args: args{
				mesh: &servicemeshapi.Mesh{
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: []servicemeshapi.ServiceMeshCondition{},
					},
				},
				conditionType: servicemeshapi.ServiceMeshActive,
			},
			expect: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetServiceMeshCondition(&tt.args.mesh.Status, tt.args.conditionType)
			assert.Equal(t, tt.expect, got)
		})
	}
}

func TestUpdateCondition(t *testing.T) {
	type args struct {
		mesh          *servicemeshapi.Mesh
		conditionType servicemeshapi.ServiceMeshConditionType
		status        metav1.ConditionStatus
		reason        string
		message       string
		generation    int64
	}
	tests := []struct {
		name         string
		args         args
		expectedMesh *servicemeshapi.Mesh
		wantChanged  bool
	}{
		{
			name: "condition updated by modify condition status",
			args: args{
				mesh: &servicemeshapi.Mesh{
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
				conditionType: servicemeshapi.ServiceMeshActive,
				status:        metav1.ConditionTrue,
			},
			expectedMesh: &servicemeshapi.Mesh{
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
			wantChanged: true,
		},
		{
			name: "condition updated by modify condition reason",
			args: args{
				mesh: &servicemeshapi.Mesh{
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
				conditionType: servicemeshapi.ServiceMeshActive,
				status:        metav1.ConditionTrue,
				reason:        "reason",
			},
			expectedMesh: &servicemeshapi.Mesh{
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
								Reason: "reason",
							},
						},
					},
				},
			},
			wantChanged: true,
		},
		{
			name: "condition updated by modify condition message",
			args: args{
				mesh: &servicemeshapi.Mesh{
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
				conditionType: servicemeshapi.ServiceMeshActive,
				status:        metav1.ConditionTrue,
				message:       "message",
			},
			expectedMesh: &servicemeshapi.Mesh{
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:  metav1.ConditionTrue,
								Message: "message",
							},
						},
					},
				},
			},
			wantChanged: true,
		},
		{
			name: "condition updated by modify condition generation",
			args: args{
				mesh: &servicemeshapi.Mesh{
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status:             metav1.ConditionTrue,
									ObservedGeneration: 1,
								},
							},
						},
					},
				},
				conditionType: servicemeshapi.ServiceMeshActive,
				status:        metav1.ConditionTrue,
				generation:    0,
			},
			expectedMesh: &servicemeshapi.Mesh{
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:             metav1.ConditionTrue,
								ObservedGeneration: 0,
							},
						},
					},
				},
			},
			wantChanged: true,
		},
		{
			name: "condition updated by modify condition status/generation/reason/message",
			args: args{
				mesh: &servicemeshapi.Mesh{
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status:             metav1.ConditionFalse,
									ObservedGeneration: 1,
								},
							},
						},
					},
				},
				conditionType: servicemeshapi.ServiceMeshActive,
				status:        metav1.ConditionTrue,
				reason:        "reason",
				message:       "message",
				generation:    0,
			},
			expectedMesh: &servicemeshapi.Mesh{
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:             metav1.ConditionTrue,
								ObservedGeneration: 0,
								Reason:             "reason",
								Message:            "message",
							},
						},
					},
				},
			},
			wantChanged: true,
		},
		{
			name: "condition updated by new condition",
			args: args{
				mesh: &servicemeshapi.Mesh{
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: nil,
					},
				},
				conditionType: servicemeshapi.ServiceMeshActive,
				status:        metav1.ConditionTrue,
				reason:        "reason",
				message:       "message",
			},
			expectedMesh: &servicemeshapi.Mesh{
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:  metav1.ConditionTrue,
								Reason:  "reason",
								Message: "message",
							},
						},
					},
				},
			},
			wantChanged: true,
		},
		{
			name: "condition unmodified",
			args: args{
				mesh: &servicemeshapi.Mesh{
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status:  metav1.ConditionTrue,
									Reason:  "reason",
									Message: "message",
								},
							},
						},
					},
				},
				conditionType: servicemeshapi.ServiceMeshActive,
				status:        metav1.ConditionTrue,
				reason:        "reason",
				message:       "message",
			},
			expectedMesh: &servicemeshapi.Mesh{
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:  metav1.ConditionTrue,
								Reason:  "reason",
								Message: "message",
							},
						},
					},
				},
			},
			wantChanged: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotChanged := UpdateServiceMeshCondition(&tt.args.mesh.Status, tt.args.conditionType, tt.args.status, tt.args.reason, tt.args.message, tt.args.generation)
			opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
			assert.True(t, cmp.Equal(tt.expectedMesh, tt.args.mesh, opts), "diff", cmp.Diff(tt.expectedMesh, tt.args.mesh, opts))
			assert.Equal(t, tt.wantChanged, gotChanged)
		})
	}
}

func TestGetVirtualDeploymentBindingConditionReason(t *testing.T) {
	tests := []struct {
		name   string
		status metav1.ConditionStatus
		expect ResourceConditionReason
	}{
		{
			name:   "successful",
			status: metav1.ConditionTrue,
			expect: Successful,
		},
		{
			name:   "state changed for unknown",
			status: metav1.ConditionUnknown,
			expect: DependenciesNotResolved,
		},
		{
			name:   "state changed for false",
			status: metav1.ConditionFalse,
			expect: DependenciesNotResolved,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetVirtualDeploymentBindingConditionReason(tt.status)
			assert.Equal(t, tt.expect, got)
		})
	}
}

func TestGetVirtualDeploymentBindingConditionMessage(t *testing.T) {
	tests := []struct {
		name   string
		state  string
		expect ResourceConditionMessageVDB
	}{
		{
			name:   "condition true",
			state:  Active,
			expect: ResourceActiveVDB,
		},
		{
			name:   "condition unknown",
			state:  Creating,
			expect: ResourceCreatingVDB,
		},
		{
			name:   "condition false",
			state:  Deleted,
			expect: ResourceDeletedVDB,
		},
		{
			name:   "condition unknown",
			state:  Deleting,
			expect: ResourceDeletingVDB,
		},
		{
			name:   "condition unknown",
			state:  Updating,
			expect: ResourceUpdatingVDB,
		},
		{
			name:   "condition false",
			state:  Failed,
			expect: ResourceFailedVDB,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetVirtualDeploymentBindingConditionMessage(tt.state)
			assert.Equal(t, tt.expect, got)
		})
	}
}

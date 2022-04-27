/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package validator

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	sdk "github.com/oracle/oci-go-sdk/v65/servicemesh"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	meshErrors "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/errors"
)

func TestValidateMeshK8s(t *testing.T) {
	tests := []struct {
		name    string
		mesh    *servicemeshapi.Mesh
		wantErr error
	}{
		{
			name: "mesh is active",
			mesh: &servicemeshapi.Mesh{
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId: "mesh-id",
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
						{
							Type: servicemeshapi.ServiceMeshDependenciesActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
						{
							Type: servicemeshapi.ServiceMeshConfigured,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
			},
		},
		{
			name: "mesh is inactive",
			mesh: &servicemeshapi.Mesh{
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId: "mesh-id",
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionFalse,
							},
						},
						{
							Type: servicemeshapi.ServiceMeshDependenciesActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
						{
							Type: servicemeshapi.ServiceMeshConfigured,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
			},
			wantErr: errors.New("mesh status condition ServiceMeshActive is not yet satisfied"),
		},
		{
			name: "mesh dependencies only active",
			mesh: &servicemeshapi.Mesh{
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId: "mesh-id",
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshDependenciesActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
			},
			wantErr: errors.New("mesh status condition is not yet satisfied"),
		},
		{
			name: "mesh dependencies is not active",
			mesh: &servicemeshapi.Mesh{
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId: "mesh-id",
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshDependenciesActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionFalse,
							},
						},
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
						{
							Type: servicemeshapi.ServiceMeshConfigured,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
			},
			wantErr: errors.New("mesh status condition ServiceMeshDependenciesActive is not yet satisfied"),
		},
		{
			name: "mesh configured only active",
			mesh: &servicemeshapi.Mesh{
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId: "mesh-id",
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshConfigured,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
			},
			wantErr: errors.New("mesh status condition is not yet satisfied"),
		},
		{
			name: "mesh configured is not true",
			mesh: &servicemeshapi.Mesh{
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId: "mesh-id",
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshDependenciesActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
						{
							Type: servicemeshapi.ServiceMeshConfigured,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionFalse,
							},
						},
					},
				},
			},
			wantErr: errors.New("mesh status condition ServiceMeshConfigured is not yet satisfied"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMeshK8s(tt.mesh)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateMeshCp(t *testing.T) {
	tests := []struct {
		name    string
		mesh    *sdk.Mesh
		wantErr error
	}{
		{
			name: "mesh is active",
			mesh: &sdk.Mesh{
				LifecycleState: sdk.MeshLifecycleStateActive,
			},
		},
		{
			name: "mesh is inactive",
			mesh: &sdk.Mesh{
				LifecycleState: sdk.MeshLifecycleStateCreating,
			},
			wantErr: errors.New("mesh is not active yet"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMeshCp(tt.mesh)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateVSK8s(t *testing.T) {
	tests := []struct {
		name           string
		virtualService *servicemeshapi.VirtualService
		wantErr        error
	}{
		{
			name: "virtual service is active",
			virtualService: &servicemeshapi.VirtualService{
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
						{
							Type: servicemeshapi.ServiceMeshDependenciesActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
						{
							Type: servicemeshapi.ServiceMeshConfigured,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
			},
		},
		{
			name: "virtual service dependencies only active",
			virtualService: &servicemeshapi.VirtualService{
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshDependenciesActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
			},
			wantErr: errors.New("virtual service status condition is not yet satisfied"),
		},
		{
			name: "virtual service configured",
			virtualService: &servicemeshapi.VirtualService{
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshConfigured,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
			},
			wantErr: errors.New("virtual service status condition is not yet satisfied"),
		},
		{
			name: "virtual service is inactive",
			virtualService: &servicemeshapi.VirtualService{
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionFalse,
							},
						},
						{
							Type: servicemeshapi.ServiceMeshDependenciesActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
						{
							Type: servicemeshapi.ServiceMeshConfigured,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
			},
			wantErr: meshErrors.NewRequeueOnError(errors.New("virtual service status condition ServiceMeshActive is not yet satisfied")),
		},
		{
			name: "virtual service dependencies is not active",
			virtualService: &servicemeshapi.VirtualService{
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshDependenciesActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionFalse,
							},
						},
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
						{
							Type: servicemeshapi.ServiceMeshConfigured,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
			},
			wantErr: meshErrors.NewRequeueOnError(errors.New("virtual service status condition ServiceMeshDependenciesActive is not yet satisfied")),
		},
		{
			name: "virtual service is not configures",
			virtualService: &servicemeshapi.VirtualService{
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshDependenciesActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
						{
							Type: servicemeshapi.ServiceMeshConfigured,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionFalse,
							},
						},
					},
				},
			},
			wantErr: meshErrors.NewRequeueOnError(errors.New("virtual service status condition ServiceMeshConfigured is not yet satisfied")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVSK8s(tt.virtualService)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateVSCp(t *testing.T) {
	tests := []struct {
		name           string
		virtualService *sdk.VirtualService
		wantErr        error
	}{
		{
			name: "virtual service is active",
			virtualService: &sdk.VirtualService{
				LifecycleState: sdk.VirtualServiceLifecycleStateActive,
			},
		},
		{
			name: "virtual service is inactive",
			virtualService: &sdk.VirtualService{
				LifecycleState: sdk.VirtualServiceLifecycleStateCreating,
			},
			wantErr: meshErrors.NewRequeueOnError(errors.New("virtual service is not active yet")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVSCp(tt.virtualService)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateVDK8s(t *testing.T) {
	tests := []struct {
		name              string
		virtualDeployment *servicemeshapi.VirtualDeployment
		wantErr           error
	}{
		{
			name: "virtual deployment is active",
			virtualDeployment: &servicemeshapi.VirtualDeployment{
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
						{
							Type: servicemeshapi.ServiceMeshDependenciesActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
						{
							Type: servicemeshapi.ServiceMeshConfigured,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
			},
		},
		{
			name: "virtual deployment is inactive",
			virtualDeployment: &servicemeshapi.VirtualDeployment{
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionFalse,
							},
						},
						{
							Type: servicemeshapi.ServiceMeshDependenciesActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
						{
							Type: servicemeshapi.ServiceMeshConfigured,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
			},
			wantErr: meshErrors.NewRequeueOnError(errors.New("virtual deployment status condition ServiceMeshActive is not yet satisfied")),
		},
		{
			name: "virtual deployment dependencies only active",
			virtualDeployment: &servicemeshapi.VirtualDeployment{
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshDependenciesActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
			},
			wantErr: errors.New("virtual deployment status condition is not yet satisfied"),
		},
		{
			name: "virtual deployment dependencies is not active",
			virtualDeployment: &servicemeshapi.VirtualDeployment{
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshDependenciesActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionFalse,
							},
						},
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
						{
							Type: servicemeshapi.ServiceMeshConfigured,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
			},
			wantErr: meshErrors.NewRequeueOnError(errors.New("virtual deployment status condition ServiceMeshDependenciesActive is not yet satisfied")),
		},
		{
			name: "virtual deployment configured",
			virtualDeployment: &servicemeshapi.VirtualDeployment{
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshConfigured,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
			},
			wantErr: errors.New("virtual deployment status condition is not yet satisfied"),
		},
		{
			name: "virtual deployment not configured",
			virtualDeployment: &servicemeshapi.VirtualDeployment{
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshDependenciesActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
						{
							Type: servicemeshapi.ServiceMeshConfigured,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionFalse,
							},
						},
					},
				},
			},
			wantErr: meshErrors.NewRequeueOnError(errors.New("virtual deployment status condition ServiceMeshConfigured is not yet satisfied")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVDK8s(tt.virtualDeployment)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateVDCp(t *testing.T) {
	tests := []struct {
		name              string
		virtualDeployment *sdk.VirtualDeployment
		wantErr           error
	}{
		{
			name: "virtual deployment is active",
			virtualDeployment: &sdk.VirtualDeployment{
				LifecycleState: sdk.VirtualDeploymentLifecycleStateActive,
			},
		},
		{
			name: "virtual deployment is inactive",
			virtualDeployment: &sdk.VirtualDeployment{
				LifecycleState: sdk.VirtualDeploymentLifecycleStateCreating,
			},
			wantErr: meshErrors.NewRequeueOnError(errors.New("virtual deployment is not active yet")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVDCp(tt.virtualDeployment)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateIGK8s(t *testing.T) {
	tests := []struct {
		name           string
		ingressGateway *servicemeshapi.IngressGateway
		wantErr        error
	}{
		{
			name: "ingress gateway is active",
			ingressGateway: &servicemeshapi.IngressGateway{
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
						{
							Type: servicemeshapi.ServiceMeshDependenciesActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
						{
							Type: servicemeshapi.ServiceMeshConfigured,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
			},
		},
		{
			name: "ingress gateway is inactive",
			ingressGateway: &servicemeshapi.IngressGateway{
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionFalse,
							},
						},
						{
							Type: servicemeshapi.ServiceMeshDependenciesActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
						{
							Type: servicemeshapi.ServiceMeshConfigured,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
			},
			wantErr: meshErrors.NewRequeueOnError(errors.New("ingress gateway status condition ServiceMeshActive is not yet satisfied")),
		},
		{
			name: "ingress gateway dependencies only active",
			ingressGateway: &servicemeshapi.IngressGateway{
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshDependenciesActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
			},
			wantErr: errors.New("ingress gateway status condition is not yet satisfied"),
		},
		{
			name: "ingress gateway dependencies is not active",
			ingressGateway: &servicemeshapi.IngressGateway{
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshDependenciesActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionFalse,
							},
						},
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
						{
							Type: servicemeshapi.ServiceMeshConfigured,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
			},
			wantErr: meshErrors.NewRequeueOnError(errors.New("ingress gateway status condition ServiceMeshDependenciesActive is not yet satisfied")),
		},
		{
			name: "ingress gateway configured",
			ingressGateway: &servicemeshapi.IngressGateway{
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshConfigured,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
			},
			wantErr: errors.New("ingress gateway status condition is not yet satisfied"),
		},
		{
			name: "ingress gateway not configured",
			ingressGateway: &servicemeshapi.IngressGateway{
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshDependenciesActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
						{
							Type: servicemeshapi.ServiceMeshConfigured,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionFalse,
							},
						},
					},
				},
			},
			wantErr: meshErrors.NewRequeueOnError(errors.New("ingress gateway status condition ServiceMeshConfigured is not yet satisfied")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIGK8s(tt.ingressGateway)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateIGCp(t *testing.T) {
	tests := []struct {
		name           string
		ingressGateway *sdk.IngressGateway
		wantErr        error
	}{
		{
			name: "ingress gateway is active",
			ingressGateway: &sdk.IngressGateway{
				LifecycleState: sdk.IngressGatewayLifecycleStateActive,
			},
		},
		{
			name: "ingress gateway is inactive",
			ingressGateway: &sdk.IngressGateway{
				LifecycleState: sdk.IngressGatewayLifecycleStateCreating,
			},
			wantErr: meshErrors.NewRequeueOnError(errors.New("ingress gateway is not active yet")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIGCp(tt.ingressGateway)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

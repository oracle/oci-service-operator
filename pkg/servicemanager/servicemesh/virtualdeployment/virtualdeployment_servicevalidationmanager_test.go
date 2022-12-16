/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package virtualdeployment

import (
	"context"
	"errors"
	sdk "github.com/oracle/oci-go-sdk/v65/servicemesh"
	api "github.com/oracle/oci-service-operator/api/v1beta1"
	meshErrors "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	serviceMeshManager "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/manager"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	meshMocks "github.com/oracle/oci-service-operator/test/mocks/servicemesh"
)

var (
	vdName         = servicemeshapi.Name("my-vd")
	vdName1        = servicemeshapi.Name("my-vd-1")
	vdDescription1 = servicemeshapi.Description("This is Virtual Deployment 1")
)

func Test_VirtualDeploymentValidateCreate(t *testing.T) {
	type args struct {
		virtualDeployment *servicemeshapi.VirtualDeployment
	}
	type expectation struct {
		allowed bool
		reason  string
	}
	type fields struct {
		ResolveResourceRef             func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef
		ResolveVirtualServiceReference func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error)
		ResolveVirtualServiceById      func(ctx context.Context, virtualServiceId *api.OCID) (*sdk.VirtualService, error)
	}
	tests := []struct {
		name    string
		args    args
		fields  fields
		wantErr expectation
	}{
		{
			name: "Spec contains only virtual service References",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-vs",
							},
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
					vsRef := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vsRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Spec contains only virtual service OCID",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-ocid",
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
					vsRef := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vsRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Spec contains both virtual service OCID and Ref",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-vs",
							},
							Id: "my-vs-ocid",
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.VirtualServiceReferenceIsNotUnique)},
		},
		{
			name: "Spec not contains OCID and Virtual service Ref",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{},
					},
				},
			},
			wantErr: expectation{false, string(commons.VirtualServiceReferenceIsEmpty)},
		},
		{
			name: "Spec contains metadata name longer than 190 characters",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "Thisisaverylongnametotestthefunctionthereisnopointinhavingthislongofanamebutmaybesometimeswewillhavesuchalongnameandweneedtoensurethatourservicecanhandlethisnamewhenitisoverhundredandninetycharacters",
						Namespace: "my-namespace",
					},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-vs",
							},
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
					vsRef := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vsRef, nil
				},
			},
			wantErr: expectation{false, string(commons.MetadataNameLengthExceeded)},
		},
		{
			name: "Spec contains empty service discovery and empty listener",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-ocid",
						},
						Listener: []servicemeshapi.Listener{},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
					vsRef := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
						Spec: servicemeshapi.VirtualServiceSpec{
							Hosts: []string{"host"},
						},
					}
					return vsRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Spec contains disabled service discovery and empty listener",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-ocid",
						},
						Listener: []servicemeshapi.Listener{},
						ServiceDiscovery: &servicemeshapi.ServiceDiscovery{
							Type: "DISABLED",
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
					vsRef := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
						Spec: servicemeshapi.VirtualServiceSpec{
							Hosts: []string{"host"},
						},
					}
					return vsRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Spec contains empty hostname and empty listener for type DNS",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-ocid",
						},
						ServiceDiscovery: &servicemeshapi.ServiceDiscovery{
							Type: "DNS",
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
					vsRef := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vsRef, nil
				},
			},
			wantErr: expectation{false, "hostname cannot be empty when service discovery type is DNS"},
		},
		{
			name: "Spec contains empty hostname for type DNS",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-ocid",
						},
						ServiceDiscovery: &servicemeshapi.ServiceDiscovery{
							Type: "DNS",
						},
						Listener: []servicemeshapi.Listener{
							{
								Protocol: "HTTP",
								Port:     9080,
							},
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
					vsRef := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vsRef, nil
				},
			},
			wantErr: expectation{false, "hostname cannot be empty when service discovery type is DNS"},
		},
		{
			name: "Spec contains hostname for type DISABLED",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-ocid",
						},
						ServiceDiscovery: &servicemeshapi.ServiceDiscovery{
							Type:     "DISABLED",
							Hostname: "my-hostname",
						},
						Listener: []servicemeshapi.Listener{
							{
								Protocol: "HTTP",
								Port:     9080,
							},
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
					vsRef := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vsRef, nil
				},
			},
			wantErr: expectation{false, "hostname should be empty when service discovery type is DISABLED"},
		},
		{
			name: "Spec contains hostname for type DNS with VS contains no matching host",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-ocid",
						},
						ServiceDiscovery: &servicemeshapi.ServiceDiscovery{
							Type:     "DNS",
							Hostname: "my-hostname",
						},
						Listener: []servicemeshapi.Listener{
							{
								Protocol: "HTTP",
								Port:     9080,
							},
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualServiceById: func(ctx context.Context, virtualServiceId *api.OCID) (*sdk.VirtualService, error) {
					vsId := "vs-1"
					compartmentId := "compartment-id"
					meshId := "mesh-id"
					vs := &sdk.VirtualService{Id: &vsId, CompartmentId: &compartmentId, MeshId: &meshId}
					return vs, nil
				},
			},
			wantErr: expectation{false, "parent virtualService doesn't have any host"},
		},
		{
			name: "Spec contains hostname for type DNS without listener",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-ocid",
						},
						ServiceDiscovery: &servicemeshapi.ServiceDiscovery{
							Type:     "DNS",
							Hostname: "my-hostname",
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
					vsRef := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vsRef, nil
				},
			},
			wantErr: expectation{false, "service discovery and listeners should be provided together or be both empty"},
		},
		{
			name: "Spec contains listener without service discovery",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-ocid",
						},
						Listener: []servicemeshapi.Listener{
							{
								Protocol: "HTTP",
								Port:     9080,
							},
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
					vsRef := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
						Spec: servicemeshapi.VirtualServiceSpec{
							Hosts: []string{"my-hostname"},
						},
					}
					return vsRef, nil
				},
			},
			wantErr: expectation{false, "service discovery and listeners should be provided together or be both empty"},
		},
		{
			name: "Spec contains hostname for type DNS with VS contains matching host",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-ocid",
						},
						ServiceDiscovery: &servicemeshapi.ServiceDiscovery{
							Type:     "DNS",
							Hostname: "my-hostname",
						},
						Listener: []servicemeshapi.Listener{
							{
								Protocol: "HTTP",
								Port:     9080,
							},
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualServiceById: func(ctx context.Context, virtualServiceId *api.OCID) (*sdk.VirtualService, error) {
					vsId := "vs-1"
					compartmentId := "compartment-id"
					meshId := "mesh-id"
					vs := &sdk.VirtualService{Id: &vsId, CompartmentId: &compartmentId, MeshId: &meshId, Hosts: []string{"my-host"}}
					return vs, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Virtual service References is deleting",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-vs",
							},
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
					currentTime := metav1.Now()
					vsRef := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: &currentTime,
						},
					}
					return vsRef, nil
				},
			},
			wantErr: expectation{false, string(commons.VirtualServiceReferenceIsDeleting)},
		},
		{
			name: "Virtual service References cannot found",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-vs",
							},
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
					return nil, errors.New("CANNOT FETCH")
				},
			},
			wantErr: expectation{false, string(commons.VirtualServiceReferenceNotFound)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrller := gomock.NewController(t)
			defer ctrller.Finish()
			resolver := meshMocks.NewMockResolver(ctrller)
			meshValidator := NewVirtualDeploymentValidator(resolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VD")})
			meshServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(meshValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VD")})
			if tt.fields.ResolveVirtualServiceReference != nil {
				resolver.EXPECT().ResolveVirtualServiceReference(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveVirtualServiceReference).AnyTimes()
			}

			if tt.fields.ResolveResourceRef != nil {
				resolver.EXPECT().ResolveResourceRef(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveResourceRef).AnyTimes()
			}

			if tt.fields.ResolveVirtualServiceById != nil {
				resolver.EXPECT().ResolveVirtualServiceById(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveVirtualServiceById).AnyTimes()
			}

			response := meshServiceValidationManager.ValidateCreateRequest(ctx, tt.args.virtualDeployment)
			if tt.wantErr.reason != "" {
				tt.wantErr.reason = meshErrors.GetValidationErrorMessage(tt.args.virtualDeployment, tt.wantErr.reason)
			}
			assert.Equal(t, tt.wantErr.allowed, response.Allowed)
			assert.Equal(t, tt.wantErr.reason, string(response.Result.Reason))
		})
	}
}

func Test_VirtualDeploymentValidateUpdate(t *testing.T) {
	type args struct {
		virtualDeployment    *servicemeshapi.VirtualDeployment
		oldVirtualDeployment *servicemeshapi.VirtualDeployment
	}
	type expectation struct {
		allowed bool
		reason  string
	}
	type fields struct {
		ResolveResourceRef             func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef
		ResolveVirtualServiceReference func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error)
		ResolveVirtualServiceById      func(ctx context.Context, virtualServiceId *api.OCID) (*sdk.VirtualService, error)
	}
	tests := []struct {
		name    string
		args    args
		fields  fields
		wantErr expectation
	}{
		{
			name: "Update when state is not Active",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
				},
				oldVirtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
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
			},
			wantErr: expectation{false, string(commons.NotActiveOnUpdate)},
		},
		{
			name: "Update when ServiceMeshDependenciesActive state is unknown",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
				},
				oldVirtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
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
									Status: metav1.ConditionUnknown,
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
			wantErr: expectation{false, string(commons.DependenciesIsUnknownOnUpdate)},
		},
		{
			name: "Update when ServiceMeshConfigured state is unknown",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
				},
				oldVirtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
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
									Status: metav1.ConditionUnknown,
								},
							},
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.UnknownStateOnUpdate)},
		},
		{
			name: "Update when virtual service Ref is changed",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-vs-1",
							},
						},
					},
				},
				oldVirtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-vs",
							},
						},
					},
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
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.VirtualServiceReferenceIsImmutable)},
		},
		{
			name: "Update when virtual service OCID is changed",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
					},
				},
				oldVirtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id",
						},
					},
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
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.VirtualServiceReferenceIsImmutable)},
		},
		{
			name: "Update when spec is not changed",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId",
					},
				},
				oldVirtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId",
					},
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
						},
					},
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Update when spec name is not supplied in new VD whereas old VD has a spec name",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId",
					},
				},
				oldVirtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId",
						Name:          &vdName,
					},
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
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.NameIsImmutable)},
		},
		{
			name: "Update when spec name is supplied in new VD whereas old VD does not have a spec name",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId",
						Name:          &vdName1,
					},
				},
				oldVirtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId",
					},
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
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.NameIsImmutable)},
		},
		{
			name: "Update when spec name is changed from old VD's spec name",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId",
						Name:          &vdName1,
					},
				},
				oldVirtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId",
						Name:          &vdName,
					},
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
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.NameIsImmutable)},
		},
		{
			name: "Update when spec name is not supplied in both old and new VD",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId",
					},
				},
				oldVirtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId",
					},
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
						},
					},
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Update to add hostname and listener when VS has hosts",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						ServiceDiscovery: &servicemeshapi.ServiceDiscovery{
							Type:     "DNS",
							Hostname: "hostname",
						},
						Listener: []servicemeshapi.Listener{
							{
								Protocol: "HTTP",
								Port:     9080,
							},
						},
					},
				},
				oldVirtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId",
					},
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
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualServiceById: func(ctx context.Context, virtualServiceId *api.OCID) (*sdk.VirtualService, error) {
					vsId := "vs-1"
					compartmentId := "compartment-id"
					meshId := "mesh-id"
					vs := &sdk.VirtualService{Id: &vsId, CompartmentId: &compartmentId, MeshId: &meshId, Hosts: []string{"my-host"}}
					return vs, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Update to add hostname and listener when VS doesn't have hosts",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						ServiceDiscovery: &servicemeshapi.ServiceDiscovery{
							Type:     "DNS",
							Hostname: "hostname",
						},
						Listener: []servicemeshapi.Listener{
							{
								Protocol: "HTTP",
								Port:     9080,
							},
						},
					},
				},
				oldVirtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId",
					},
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
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualServiceById: func(ctx context.Context, virtualServiceId *api.OCID) (*sdk.VirtualService, error) {
					vsId := "vs-1"
					compartmentId := "compartment-id"
					meshId := "mesh-id"
					vs := &sdk.VirtualService{Id: &vsId, CompartmentId: &compartmentId, MeshId: &meshId, Hosts: []string{}}
					return vs, nil
				},
			},
			wantErr: expectation{false, "parent virtualService doesn't have any host"},
		},
		{
			name: "Update to an egress only virtual deployment",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						ServiceDiscovery: &servicemeshapi.ServiceDiscovery{
							Type: "DISABLED",
						},
						Listener: []servicemeshapi.Listener{},
					},
				},
				oldVirtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId",
						ServiceDiscovery: &servicemeshapi.ServiceDiscovery{
							Type:     "DNS",
							Hostname: "hostname",
						},
						Listener: []servicemeshapi.Listener{
							{
								Protocol: "HTTP",
								Port:     9080,
							},
						},
					},
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
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualServiceById: func(ctx context.Context, virtualServiceId *api.OCID) (*sdk.VirtualService, error) {
					vsId := "vs-1"
					compartmentId := "compartment-id"
					meshId := "mesh-id"
					vs := &sdk.VirtualService{Id: &vsId, CompartmentId: &compartmentId, MeshId: &meshId, Hosts: []string{"my-host"}}
					return vs, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Update to set Disabled service discovery with hostname",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						ServiceDiscovery: &servicemeshapi.ServiceDiscovery{
							Type:     "DISABLED",
							Hostname: "hostname",
						},
						Listener: []servicemeshapi.Listener{},
					},
				},
				oldVirtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId",
						ServiceDiscovery: &servicemeshapi.ServiceDiscovery{
							Type:     "DNS",
							Hostname: "hostname",
						},
						Listener: []servicemeshapi.Listener{
							{
								Protocol: "HTTP",
								Port:     9080,
							},
						},
					},
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
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualServiceById: func(ctx context.Context, virtualServiceId *api.OCID) (*sdk.VirtualService, error) {
					vsId := "vs-1"
					compartmentId := "compartment-id"
					meshId := "mesh-id"
					vs := &sdk.VirtualService{Id: &vsId, CompartmentId: &compartmentId, MeshId: &meshId, Hosts: []string{"my-host"}}
					return vs, nil
				},
			},
			wantErr: expectation{false, "hostname should be empty when service discovery type is DISABLED"},
		},
		{
			name: "Update when only adding service discovery without adding listener",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						ServiceDiscovery: &servicemeshapi.ServiceDiscovery{
							Type:     "DNS",
							Hostname: "hostname",
						},
					},
				},
				oldVirtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId",
					},
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
						},
					},
				},
			},
			wantErr: expectation{false, "service discovery and listeners should be provided together or be both empty"},
		},
		{
			name: "Update when only adding listener without adding service discovery",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						Listener: []servicemeshapi.Listener{
							{
								Protocol: "HTTP",
								Port:     9080,
							},
						},
					},
				},
				oldVirtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId",
					},
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
						},
					},
				},
			},
			wantErr: expectation{false, "service discovery and listeners should be provided together or be both empty"},
		},
		{
			name: "Spec contains empty hostname for type DNS",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						ServiceDiscovery: &servicemeshapi.ServiceDiscovery{
							Type: "DNS",
						},
					},
				},
				oldVirtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
					},
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
						},
					},
				},
			},
			wantErr: expectation{false, "hostname cannot be empty when service discovery type is DNS"},
		},
		{
			name: "Update when description is changed",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId",
						Description:   &vdDescription1,
					},
				},
				oldVirtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId",
						Description:   &vdDescription,
					},
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
						},
					},
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Update when spec is changed by not changing the compartment id",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId",
						ServiceDiscovery: &servicemeshapi.ServiceDiscovery{
							Type:     "DNS",
							Hostname: "my-host-name",
						},
						Listener: []servicemeshapi.Listener{
							{
								Protocol: "HTTP",
								Port:     9080,
							},
						},
					},
				},
				oldVirtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId",
					},
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
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualServiceById: func(ctx context.Context, virtualServiceId *api.OCID) (*sdk.VirtualService, error) {
					vsId := "vs-1"
					compartmentId := "compartment-id"
					meshId := "mesh-id"
					vs := &sdk.VirtualService{Id: &vsId, CompartmentId: &compartmentId, MeshId: &meshId, Hosts: []string{"my-host"}}
					return vs, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Update when both service discovery and compartmentId is changed",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId-1",
						ServiceDiscovery: &servicemeshapi.ServiceDiscovery{
							Type:     "DNS",
							Hostname: "host-name",
						},
						Listener: []servicemeshapi.Listener{
							{
								Protocol: "HTTP",
								Port:     9080,
							},
						},
					},
				},
				oldVirtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId",
						ServiceDiscovery: &servicemeshapi.ServiceDiscovery{
							Type:     "DNS",
							Hostname: "old-host-name",
						},
						Listener: []servicemeshapi.Listener{
							{
								Protocol: "HTTP",
								Port:     9080,
							},
						},
					},
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
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualServiceById: func(ctx context.Context, virtualServiceId *api.OCID) (*sdk.VirtualService, error) {
					vsId := "vs-1"
					compartmentId := "compartment-id"
					meshId := "mesh-id"
					vs := &sdk.VirtualService{Id: &vsId, CompartmentId: &compartmentId, MeshId: &meshId, Hosts: []string{"my-host"}}
					return vs, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Update when both spec name and compartmentId is changed",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId-1",
						Name:          &vdName,
					},
				},
				oldVirtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId",
					},
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
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.NameIsImmutable)},
		},
		{
			name: "Update when both listeners and compartmentId is changed",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId-1",
						ServiceDiscovery: &servicemeshapi.ServiceDiscovery{
							Type:     "DNS",
							Hostname: "old-host-name",
						},
						Listener: []servicemeshapi.Listener{
							{
								Protocol: "HTTP",
								Port:     9080,
							},
						},
					},
				},
				oldVirtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId",
						ServiceDiscovery: &servicemeshapi.ServiceDiscovery{
							Type:     "DNS",
							Hostname: "old-host-name",
						},
						Listener: []servicemeshapi.Listener{
							{
								Protocol: "HTTP",
								Port:     8080,
							},
						},
					},
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
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualServiceById: func(ctx context.Context, virtualServiceId *api.OCID) (*sdk.VirtualService, error) {
					vsId := "vs-1"
					compartmentId := "compartment-id"
					meshId := "mesh-id"
					vs := &sdk.VirtualService{Id: &vsId, CompartmentId: &compartmentId, MeshId: &meshId, Hosts: []string{"my-host"}}
					return vs, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Update when both access logging and compartmentId is changed",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId-1",
						AccessLogging: &servicemeshapi.AccessLogging{
							IsEnabled: false,
						},
					},
				},
				oldVirtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId",
					},
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
						},
					},
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Update when status is added",
			args: args{
				virtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						CompartmentId: "my-compId-1",
					},
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
				oldVirtualDeployment: &servicemeshapi.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualDeploymentSpec{
						CompartmentId: "my-compId",
					},
				},
			},
			wantErr: expectation{true, ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrller := gomock.NewController(t)
			defer ctrller.Finish()
			resolver := meshMocks.NewMockResolver(ctrller)
			meshValidator := NewVirtualDeploymentValidator(resolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VD")})
			meshServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(meshValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VD")})

			if tt.fields.ResolveVirtualServiceReference != nil {
				resolver.EXPECT().ResolveVirtualServiceReference(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveVirtualServiceReference).AnyTimes()
			}

			if tt.fields.ResolveResourceRef != nil {
				resolver.EXPECT().ResolveResourceRef(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveResourceRef).AnyTimes()
			}

			if tt.fields.ResolveVirtualServiceById != nil {
				resolver.EXPECT().ResolveVirtualServiceById(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveVirtualServiceById).AnyTimes()
			}

			response := meshServiceValidationManager.ValidateUpdateRequest(ctx, tt.args.virtualDeployment, tt.args.oldVirtualDeployment)
			if tt.wantErr.reason != "" {
				tt.wantErr.reason = meshErrors.GetValidationErrorMessage(tt.args.virtualDeployment, tt.wantErr.reason)
			}
			assert.Equal(t, tt.wantErr.allowed, response.Allowed)
			assert.Equal(t, tt.wantErr.reason, string(response.Result.Reason))
		})
	}
}

func Test_VirtualDeploymentValidateDelete(t *testing.T) {
	type args struct {
		virtualDeployment *servicemeshapi.VirtualDeployment
	}
	type expectation struct {
		allowed bool
		reason  string
	}
	tests := []struct {
		name    string
		args    args
		wantErr expectation
	}{
		{
			name: "State is True",
			args: args{
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
								Type: servicemeshapi.ServiceMeshConfigured,
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
						},
					},
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "State is False",
			args: args{
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
								Type: servicemeshapi.ServiceMeshConfigured,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
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
			wantErr: expectation{true, ""},
		},
		{
			name: "State is Unknown",
			args: args{
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
								Type: servicemeshapi.ServiceMeshConfigured,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
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
			wantErr: expectation{false, string(commons.UnknownStatusOnDelete)},
		},
		{
			name: "Active is Unknown and configured is False",
			args: args{
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
								Type: servicemeshapi.ServiceMeshConfigured,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionFalse,
								},
							},
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
			wantErr: expectation{true, ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			resolver := meshMocks.NewMockResolver(mockCtrl)
			meshValidator := NewVirtualDeploymentValidator(resolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VD")})
			meshServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(meshValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VD")})
			response := meshServiceValidationManager.ValidateDeleteRequest(ctx, tt.args.virtualDeployment)
			if tt.wantErr.reason != "" {
				tt.wantErr.reason = meshErrors.GetValidationErrorMessage(tt.args.virtualDeployment, tt.wantErr.reason)
			}
			assert.Equal(t, tt.wantErr.allowed, response.Allowed)
			assert.Equal(t, tt.wantErr.reason, string(response.Result.Reason))
		})
	}
}

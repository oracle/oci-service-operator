/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package virtualdeploymentbinding

import (
	"context"
	"errors"
	meshErrors "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	serviceMeshManager "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/manager"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	meshMocks "github.com/oracle/oci-service-operator/test/mocks/servicemesh"
)

func Test_VirtualDeploymentBindingValidateCreate(t *testing.T) {
	type args struct {
		VirtualDeploymentBinding *servicemeshapi.VirtualDeploymentBinding
	}
	type expectation struct {
		allowed bool
		reason  string
	}
	type fields struct {
		ResolveResourceRef                func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef
		ResolveVirtualDeploymentReference func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error)
		ResolveServiceReference           func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*corev1.Service, error)
	}
	tests := []struct {
		name    string
		args    args
		fields  fields
		wantErr expectation
	}{
		{
			name: "Spec contains only virtual deployment References",
			args: args{
				VirtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					Spec: servicemeshapi.VirtualDeploymentBindingSpec{
						VirtualDeployment: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-vd",
							},
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualDeploymentReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
					vdRef := &servicemeshapi.VirtualDeployment{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vdRef, nil
				},
				ResolveServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*corev1.Service, error) {
					serviceRef := &corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return serviceRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Spec contains only virtual deployment OCID",
			args: args{
				VirtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					Spec: servicemeshapi.VirtualDeploymentBindingSpec{
						VirtualDeployment: servicemeshapi.RefOrId{
							Id: "my-vd-ocid",
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualDeploymentReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
					vdRef := &servicemeshapi.VirtualDeployment{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vdRef, nil
				},
				ResolveServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*corev1.Service, error) {
					serviceRef := &corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return serviceRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Spec contains both virtual deployment OCID and Ref",
			args: args{
				VirtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					Spec: servicemeshapi.VirtualDeploymentBindingSpec{
						VirtualDeployment: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-vd",
							},
							Id: "my-vd-ocid",
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.VirtualDeploymentReferenceIsNotUnique)},
		},
		{
			name: "Spec not contains both virtual deployment OCID and Ref",
			args: args{
				VirtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					Spec: servicemeshapi.VirtualDeploymentBindingSpec{
						VirtualDeployment: servicemeshapi.RefOrId{},
					},
				},
			},
			wantErr: expectation{false, string(commons.VirtualDeploymentReferenceIsEmpty)},
		},
		{
			name: "Spec contains metadata name longer than 190 characters",
			args: args{
				VirtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "Thisisaverylongnametotestthefunctionthereisnopointinhavingthislongofanamebutmaybesometimeswewillhavesuchalongnameandweneedtoensurethatourservicecanhandlethisnamewhenitisoverhundredandninetycharacters",
						Namespace: "my-namespace",
					},
					Spec: servicemeshapi.VirtualDeploymentBindingSpec{
						VirtualDeployment: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-vd",
							},
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualDeploymentReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
					vdRef := &servicemeshapi.VirtualDeployment{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vdRef, nil
				},
				ResolveServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*corev1.Service, error) {
					serviceRef := &corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return serviceRef, nil
				},
			},
			wantErr: expectation{false, string(commons.MetadataNameLengthExceeded)},
		},
		{
			name: "Spec refers to deleting virtual deployment",
			args: args{
				VirtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					Spec: servicemeshapi.VirtualDeploymentBindingSpec{
						VirtualDeployment: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-vd",
							},
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualDeploymentReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
					currentTime := metav1.Now()
					vdRef := &servicemeshapi.VirtualDeployment{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: &currentTime,
						},
					}
					return vdRef, nil
				},
			},
			wantErr: expectation{false, string(commons.VirtualDeploymentReferenceIsDeleting)},
		},
		{
			name: "Spec refers to non-exist virtual deployment",
			args: args{
				VirtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					Spec: servicemeshapi.VirtualDeploymentBindingSpec{
						VirtualDeployment: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-vd",
							},
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualDeploymentReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
					return nil, errors.New("CANNOT FETCH")
				},
			},
			wantErr: expectation{false, string(commons.VirtualDeploymentReferenceNotFound)},
		},
		{
			name: "Spec refers to deleting kubernetes service",
			args: args{
				VirtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					Spec: servicemeshapi.VirtualDeploymentBindingSpec{
						VirtualDeployment: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-vd",
							},
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualDeploymentReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
					vdRef := &servicemeshapi.VirtualDeployment{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vdRef, nil
				},
				ResolveServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*corev1.Service, error) {
					currentTime := metav1.Now()
					serviceRef := &corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: &currentTime,
						},
					}
					return serviceRef, nil
				},
			},
			wantErr: expectation{false, string(commons.KubernetesServiceReferenceIsDeleting)},
		},
		{
			name: "Spec refers to non-exist kubernetes service",
			args: args{
				VirtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					Spec: servicemeshapi.VirtualDeploymentBindingSpec{
						VirtualDeployment: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-vd",
							},
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualDeploymentReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
					vdRef := &servicemeshapi.VirtualDeployment{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vdRef, nil
				},
				ResolveServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*corev1.Service, error) {
					return nil, errors.New("CANNOT FETCH")
				},
			},
			wantErr: expectation{false, string(commons.KubernetesServiceReferenceNotFound)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrller := gomock.NewController(t)
			defer ctrller.Finish()
			resolver := meshMocks.NewMockResolver(ctrller)
			meshValidator := NewVirtualDeploymentBindingValidator(resolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VDB")})
			meshServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(meshValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VDB")})
			if tt.fields.ResolveVirtualDeploymentReference != nil {
				resolver.EXPECT().ResolveVirtualDeploymentReference(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveVirtualDeploymentReference).AnyTimes()
			}

			if tt.fields.ResolveResourceRef != nil {
				resolver.EXPECT().ResolveResourceRef(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveResourceRef).AnyTimes()
			}

			if tt.fields.ResolveServiceReference != nil {
				resolver.EXPECT().ResolveServiceReferenceWithApiReader(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveServiceReference).AnyTimes()
			}

			response := meshServiceValidationManager.ValidateCreateRequest(ctx, tt.args.VirtualDeploymentBinding)
			if tt.wantErr.reason != "" {
				tt.wantErr.reason = meshErrors.GetValidationErrorMessage(tt.args.VirtualDeploymentBinding, tt.wantErr.reason)
			}
			assert.Equal(t, tt.wantErr.allowed, response.Allowed)
			assert.Equal(t, tt.wantErr.reason, string(response.Result.Reason))
		})
	}
}

func Test_VirtualDeploymentBindingValidateUpdate(t *testing.T) {
	type args struct {
		virtualDeploymentBinding    *servicemeshapi.VirtualDeploymentBinding
		oldVirtualDeploymentBinding *servicemeshapi.VirtualDeploymentBinding
	}
	type expectation struct {
		allowed bool
		reason  string
	}
	type fields struct {
		ResolveResourceRef                func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef
		ResolveVirtualDeploymentReference func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error)
		ResolveServiceReference           func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*corev1.Service, error)
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
				virtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
				},
				oldVirtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
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
				virtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
				},
				oldVirtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
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
				virtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
				},
				oldVirtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
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
			name: "Update when spec is not changed",
			args: args{
				virtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualDeploymentBindingSpec{
						VirtualDeployment: servicemeshapi.RefOrId{
							Id: "my-vd-id-1",
						},
					},
				},
				oldVirtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualDeploymentBindingSpec{
						VirtualDeployment: servicemeshapi.RefOrId{
							Id: "my-vd-id-1",
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
				ResolveVirtualDeploymentReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
					vdRef := &servicemeshapi.VirtualDeployment{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vdRef, nil
				},
				ResolveServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*corev1.Service, error) {
					serviceRef := &corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return serviceRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Update when status is added",
			args: args{
				virtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualDeploymentBindingSpec{
						VirtualDeployment: servicemeshapi.RefOrId{
							Id: "my-vd-id-1",
						},
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
				oldVirtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualDeploymentBindingSpec{
						VirtualDeployment: servicemeshapi.RefOrId{
							Id: "my-vd-id-1",
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualDeploymentReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
					vdRef := &servicemeshapi.VirtualDeployment{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vdRef, nil
				},
				ResolveServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*corev1.Service, error) {
					serviceRef := &corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return serviceRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Spec contains only virtual deployment References",
			args: args{
				virtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualDeploymentBindingSpec{
						VirtualDeployment: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-vd",
							},
						},
					},
				},
				oldVirtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualDeploymentBindingSpec{
						VirtualDeployment: servicemeshapi.RefOrId{
							Id: "my-vd-id-1",
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
				ResolveVirtualDeploymentReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
					vdRef := &servicemeshapi.VirtualDeployment{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vdRef, nil
				},
				ResolveServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*corev1.Service, error) {
					serviceRef := &corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return serviceRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Spec contains only virtual deployment OCID",
			args: args{
				virtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualDeploymentBindingSpec{
						VirtualDeployment: servicemeshapi.RefOrId{
							Id: "my-vd-id-1",
						},
					},
				},
				oldVirtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualDeploymentBindingSpec{
						VirtualDeployment: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-vd",
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
				ResolveVirtualDeploymentReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
					vdRef := &servicemeshapi.VirtualDeployment{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vdRef, nil
				},
				ResolveServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*corev1.Service, error) {
					serviceRef := &corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return serviceRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Spec contains both virtual deployment OCID and Ref",
			args: args{
				virtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualDeploymentBindingSpec{
						VirtualDeployment: servicemeshapi.RefOrId{
							Id: "my-vd-id-1",
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-vd",
							},
						},
					},
				},
				oldVirtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualDeploymentBindingSpec{
						VirtualDeployment: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-vd",
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
			wantErr: expectation{false, string(commons.VirtualDeploymentReferenceIsNotUnique)},
		},
		{
			name: "Spec not contains both virtual deployment OCID and Ref",
			args: args{
				virtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualDeploymentBindingSpec{
						VirtualDeployment: servicemeshapi.RefOrId{},
					},
				},
				oldVirtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualDeploymentBindingSpec{
						VirtualDeployment: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-vd",
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
			wantErr: expectation{false, string(commons.VirtualDeploymentReferenceIsEmpty)},
		},
		{
			name: "Spec refers to deleting virtual deployment",
			args: args{
				virtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualDeploymentBindingSpec{
						VirtualDeployment: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-vd",
							},
						},
					},
				},
				oldVirtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualDeploymentBindingSpec{
						VirtualDeployment: servicemeshapi.RefOrId{
							Id: "my-vd-id-1",
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
				ResolveVirtualDeploymentReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
					currentTime := metav1.Now()
					vdRef := &servicemeshapi.VirtualDeployment{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: &currentTime,
						},
					}
					return vdRef, nil
				},
			},
			wantErr: expectation{false, string(commons.VirtualDeploymentReferenceIsDeleting)},
		},
		{
			name: "Spec refers to non-exist virtual deployment",
			args: args{
				virtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualDeploymentBindingSpec{
						VirtualDeployment: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-vd",
							},
						},
					},
				},
				oldVirtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualDeploymentBindingSpec{
						VirtualDeployment: servicemeshapi.RefOrId{
							Id: "my-vd-id-1",
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
				ResolveVirtualDeploymentReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
					return nil, errors.New("CANNOT FETCH")
				},
			},
			wantErr: expectation{false, string(commons.VirtualDeploymentReferenceNotFound)},
		},
		{
			name: "Spec refers to deleting kubernetes service",
			args: args{
				virtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualDeploymentBindingSpec{
						VirtualDeployment: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-vd",
							},
						},
					},
				},
				oldVirtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualDeploymentBindingSpec{
						VirtualDeployment: servicemeshapi.RefOrId{
							Id: "my-vd-id-1",
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
				ResolveVirtualDeploymentReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
					vdRef := &servicemeshapi.VirtualDeployment{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vdRef, nil
				},
				ResolveServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*corev1.Service, error) {
					currentTime := metav1.Now()
					serviceRef := &corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: &currentTime,
						},
					}
					return serviceRef, nil
				},
			},
			wantErr: expectation{false, string(commons.KubernetesServiceReferenceIsDeleting)},
		},
		{
			name: "Spec refers to non-exist kubernetes service",
			args: args{
				virtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualDeploymentBindingSpec{
						VirtualDeployment: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-vd",
							},
						},
					},
				},
				oldVirtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualDeploymentBindingSpec{
						VirtualDeployment: servicemeshapi.RefOrId{
							Id: "my-vd-id-1",
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
				ResolveVirtualDeploymentReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
					vdRef := &servicemeshapi.VirtualDeployment{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vdRef, nil
				},
				ResolveServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*corev1.Service, error) {
					return nil, errors.New("CANNOT FETCH")
				},
			},
			wantErr: expectation{false, string(commons.KubernetesServiceReferenceNotFound)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrller := gomock.NewController(t)
			defer ctrller.Finish()
			resolver := meshMocks.NewMockResolver(ctrller)
			meshValidator := NewVirtualDeploymentBindingValidator(resolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VDB")})
			meshServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(meshValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VDB")})
			if tt.fields.ResolveVirtualDeploymentReference != nil {
				resolver.EXPECT().ResolveVirtualDeploymentReference(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveVirtualDeploymentReference).AnyTimes()
			}

			if tt.fields.ResolveResourceRef != nil {
				resolver.EXPECT().ResolveResourceRef(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveResourceRef).AnyTimes()
			}

			if tt.fields.ResolveServiceReference != nil {
				resolver.EXPECT().ResolveServiceReferenceWithApiReader(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveServiceReference).AnyTimes()
			}

			response := meshServiceValidationManager.ValidateUpdateRequest(ctx, tt.args.virtualDeploymentBinding, tt.args.oldVirtualDeploymentBinding)
			if tt.wantErr.reason != "" {
				tt.wantErr.reason = meshErrors.GetValidationErrorMessage(tt.args.virtualDeploymentBinding, tt.wantErr.reason)
			}
			assert.Equal(t, tt.wantErr.allowed, response.Allowed)
			assert.Equal(t, tt.wantErr.reason, string(response.Result.Reason))
		})
	}
}

func Test_VirtualDeploymentBindingValidateDelete(t *testing.T) {
	type args struct {
		VirtualDeploymentBinding *servicemeshapi.VirtualDeploymentBinding
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
				VirtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
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
				VirtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
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
				VirtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
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
				VirtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
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
			ctrller := gomock.NewController(t)
			defer ctrller.Finish()
			resolver := meshMocks.NewMockResolver(ctrller)
			meshValidator := NewVirtualDeploymentBindingValidator(resolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VDB")})
			meshServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(meshValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VDB")})
			response := meshServiceValidationManager.ValidateDeleteRequest(ctx, tt.args.VirtualDeploymentBinding)
			if tt.wantErr.reason != "" {
				tt.wantErr.reason = meshErrors.GetValidationErrorMessage(tt.args.VirtualDeploymentBinding, tt.wantErr.reason)
			}
			assert.Equal(t, tt.wantErr.allowed, response.Allowed)
			assert.Equal(t, tt.wantErr.reason, string(response.Result.Reason))
		})
	}
}

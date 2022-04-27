/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package ingressgatewaydeployment

import (
	"context"
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

func Test_IngressGatewayDeploymentValidateCreate(t *testing.T) {

	type args struct {
		ingressGatewayDeployment *servicemeshapi.IngressGatewayDeployment
	}
	type expectation struct {
		allowed bool
		reason  string
	}
	type fields struct {
		ResolveResourceRef             func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef
		ResolveIngressGatewayReference func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.IngressGateway, error)
	}
	tests := []struct {
		name    string
		args    args
		fields  fields
		wantErr expectation
	}{
		{
			name: "Spec contains Invalid pod scaling config",
			args: args{
				ingressGatewayDeployment: &servicemeshapi.IngressGatewayDeployment{
					Spec: servicemeshapi.IngressGatewayDeploymentSpec{
						IngressGateway: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-ig",
							},
						},
						Deployment: servicemeshapi.IngressDeployment{
							Autoscaling: &servicemeshapi.Autoscaling{
								MinPods: 5,
								MaxPods: 3,
							},
						},
						Ports: []servicemeshapi.GatewayListener{},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveIngressGatewayReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.IngressGateway, error) {
					igRef := &servicemeshapi.IngressGateway{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return igRef, nil
				},
			},
			wantErr: expectation{false, string(commons.IngressGatewayDeploymentInvalidMaxPod)},
		},
		{
			name: "Spec contains Invalid port config contains multiple protocol",
			args: args{
				ingressGatewayDeployment: &servicemeshapi.IngressGatewayDeployment{
					Spec: servicemeshapi.IngressGatewayDeploymentSpec{
						IngressGateway: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-ig",
							},
						},
						Deployment: servicemeshapi.IngressDeployment{
							Autoscaling: &servicemeshapi.Autoscaling{
								MinPods: 3,
								MaxPods: 3,
							},
						},
						Ports: []servicemeshapi.GatewayListener{
							{
								Protocol:    corev1.ProtocolTCP,
								Port:        &[]int32{8080}[0],
								ServicePort: &[]int32{8080}[0],
							},
							{
								Protocol:    corev1.ProtocolUDP,
								Port:        &[]int32{8081}[0],
								ServicePort: &[]int32{8081}[0],
							},
						},
						Service: &servicemeshapi.IngressGatewayService{
							Type: corev1.ServiceTypeLoadBalancer,
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveIngressGatewayReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.IngressGateway, error) {
					igRef := &servicemeshapi.IngressGateway{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return igRef, nil
				},
			},
			wantErr: expectation{false, string(commons.IngressGatewayDeploymentPortsWithMultipleProtocols)},
		},
		{
			name: "Spec contains multiple ports without names",
			args: args{
				ingressGatewayDeployment: &servicemeshapi.IngressGatewayDeployment{
					Spec: servicemeshapi.IngressGatewayDeploymentSpec{
						IngressGateway: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-ig",
							},
						},
						Deployment: servicemeshapi.IngressDeployment{
							Autoscaling: &servicemeshapi.Autoscaling{
								MinPods: 3,
								MaxPods: 3,
							},
						},
						Ports: []servicemeshapi.GatewayListener{
							{
								Name:        "hello",
								Protocol:    corev1.ProtocolTCP,
								Port:        &[]int32{8080}[0],
								ServicePort: &[]int32{8080}[0],
							},
							{
								Protocol:    corev1.ProtocolTCP,
								Port:        &[]int32{8081}[0],
								ServicePort: &[]int32{8081}[0],
							},
						},
						Service: &servicemeshapi.IngressGatewayService{
							Type: corev1.ServiceTypeLoadBalancer,
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveIngressGatewayReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.IngressGateway, error) {
					igRef := &servicemeshapi.IngressGateway{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return igRef, nil
				},
			},
			wantErr: expectation{false, string(commons.IngressGatewayDeploymentWithMultiplePortEmptyName)},
		},
		{
			name: "Spec contains multiple ports non unique names",
			args: args{
				ingressGatewayDeployment: &servicemeshapi.IngressGatewayDeployment{
					Spec: servicemeshapi.IngressGatewayDeploymentSpec{
						IngressGateway: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-ig",
							},
						},
						Deployment: servicemeshapi.IngressDeployment{
							Autoscaling: &servicemeshapi.Autoscaling{
								MinPods: 3,
								MaxPods: 3,
							},
						},
						Ports: []servicemeshapi.GatewayListener{
							{
								Name:        "hello",
								Protocol:    corev1.ProtocolTCP,
								Port:        &[]int32{8080}[0],
								ServicePort: &[]int32{8080}[0],
							},
							{
								Name:        "hello",
								Protocol:    corev1.ProtocolTCP,
								Port:        &[]int32{8081}[0],
								ServicePort: &[]int32{8081}[0],
							},
						},
						Service: &servicemeshapi.IngressGatewayService{
							Type: corev1.ServiceTypeLoadBalancer,
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveIngressGatewayReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.IngressGateway, error) {
					igRef := &servicemeshapi.IngressGateway{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return igRef, nil
				},
			},
			wantErr: expectation{false, string(commons.IngressGatewayDeploymentPortsWithNonUniqueNames)},
		},
		{
			name: "Spec without service and service ports",
			args: args{
				ingressGatewayDeployment: &servicemeshapi.IngressGatewayDeployment{
					Spec: servicemeshapi.IngressGatewayDeploymentSpec{
						IngressGateway: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-ig",
							},
						},
						Deployment: servicemeshapi.IngressDeployment{
							Autoscaling: &servicemeshapi.Autoscaling{
								MinPods: 3,
								MaxPods: 3,
							},
						},
						Ports: []servicemeshapi.GatewayListener{
							{
								Protocol:    corev1.ProtocolTCP,
								Port:        &[]int32{8080}[0],
								ServicePort: &[]int32{8080}[0],
							},
							{
								Protocol:    corev1.ProtocolTCP,
								Port:        &[]int32{8081}[0],
								ServicePort: &[]int32{8081}[0],
							},
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveIngressGatewayReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.IngressGateway, error) {
					igRef := &servicemeshapi.IngressGateway{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return igRef, nil
				},
			},
			wantErr: expectation{false, string(commons.IngressGatewayDeploymentRedundantServicePorts)},
		},
		{
			name: "Spec contains Valid Multiple port config",
			args: args{
				ingressGatewayDeployment: &servicemeshapi.IngressGatewayDeployment{
					Spec: servicemeshapi.IngressGatewayDeploymentSpec{
						IngressGateway: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-ig",
							},
						},
						Deployment: servicemeshapi.IngressDeployment{
							Autoscaling: &servicemeshapi.Autoscaling{
								MinPods: 3,
								MaxPods: 3,
							},
						},
						Ports: []servicemeshapi.GatewayListener{
							{
								Name:        "hello",
								Protocol:    corev1.ProtocolTCP,
								Port:        &[]int32{8080}[0],
								ServicePort: &[]int32{8080}[0],
							},
							{
								Name:        "world",
								Protocol:    corev1.ProtocolTCP,
								Port:        &[]int32{8081}[0],
								ServicePort: &[]int32{8081}[0],
							},
						},
						Service: &servicemeshapi.IngressGatewayService{
							Type: corev1.ServiceTypeLoadBalancer,
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveIngressGatewayReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.IngressGateway, error) {
					igRef := &servicemeshapi.IngressGateway{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return igRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Spec contains metadata name longer than 190 characters",
			args: args{
				ingressGatewayDeployment: &servicemeshapi.IngressGatewayDeployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "Thisisaverylongnametotestthefunctionthereisnopointinhavingthislongofanamebutmaybesometimeswewillhavesuchalongnameandweneedtoensurethatourservicecanhandlethisnamewhenitisoverhundredandninetycharacters",
						Namespace: "my-namespace",
					},
					Spec: servicemeshapi.IngressGatewayDeploymentSpec{
						IngressGateway: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-ig",
							},
						},
						Deployment: servicemeshapi.IngressDeployment{
							Autoscaling: &servicemeshapi.Autoscaling{
								MinPods: 3,
								MaxPods: 3,
							},
						},
						Ports: []servicemeshapi.GatewayListener{
							{
								Name:        "hello",
								Protocol:    corev1.ProtocolTCP,
								Port:        &[]int32{8080}[0],
								ServicePort: &[]int32{8080}[0],
							},
							{
								Name:        "world",
								Protocol:    corev1.ProtocolTCP,
								Port:        &[]int32{8081}[0],
								ServicePort: &[]int32{8081}[0],
							},
						},
						Service: &servicemeshapi.IngressGatewayService{
							Type: corev1.ServiceTypeLoadBalancer,
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveIngressGatewayReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.IngressGateway, error) {
					igRef := &servicemeshapi.IngressGateway{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return igRef, nil
				},
			},
			wantErr: expectation{false, string(commons.MetadataNameLengthExceeded)},
		},
		{
			name: "Spec contains Valid Single port config",
			args: args{
				ingressGatewayDeployment: &servicemeshapi.IngressGatewayDeployment{
					Spec: servicemeshapi.IngressGatewayDeploymentSpec{
						IngressGateway: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-ig",
							},
						},
						Deployment: servicemeshapi.IngressDeployment{
							Autoscaling: &servicemeshapi.Autoscaling{
								MinPods: 3,
								MaxPods: 3,
							},
						},
						Ports: []servicemeshapi.GatewayListener{
							{
								Protocol:    corev1.ProtocolTCP,
								Port:        &[]int32{8080}[0],
								ServicePort: &[]int32{8080}[0],
							},
						},
						Service: &servicemeshapi.IngressGatewayService{
							Type: corev1.ServiceTypeLoadBalancer,
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveIngressGatewayReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.IngressGateway, error) {
					igRef := &servicemeshapi.IngressGateway{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return igRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Spec without reference",
			args: args{
				ingressGatewayDeployment: &servicemeshapi.IngressGatewayDeployment{
					Spec: servicemeshapi.IngressGatewayDeploymentSpec{
						IngressGateway: servicemeshapi.RefOrId{},
						Deployment: servicemeshapi.IngressDeployment{
							Autoscaling: &servicemeshapi.Autoscaling{
								MinPods: 5,
								MaxPods: 5,
							},
						},
						Ports: []servicemeshapi.GatewayListener{},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveIngressGatewayReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.IngressGateway, error) {
					igRef := &servicemeshapi.IngressGateway{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return igRef, nil
				},
			},
			wantErr: expectation{false, string(commons.IngressGatewayReferenceIsEmpty)},
		},
		{
			name: "Spec with name reference",
			args: args{
				ingressGatewayDeployment: &servicemeshapi.IngressGatewayDeployment{
					Spec: servicemeshapi.IngressGatewayDeploymentSpec{
						IngressGateway: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-ig",
							},
						},
						Deployment: servicemeshapi.IngressDeployment{
							Autoscaling: &servicemeshapi.Autoscaling{
								MinPods: 5,
								MaxPods: 5,
							},
						},
						Ports: []servicemeshapi.GatewayListener{},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveIngressGatewayReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.IngressGateway, error) {
					igRef := &servicemeshapi.IngressGateway{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return igRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Spec with both ocid and ingress reference",
			args: args{
				ingressGatewayDeployment: &servicemeshapi.IngressGatewayDeployment{
					Spec: servicemeshapi.IngressGatewayDeploymentSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "123",
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-ig",
							},
						},
						Deployment: servicemeshapi.IngressDeployment{
							Autoscaling: &servicemeshapi.Autoscaling{
								MinPods: 5,
								MaxPods: 5,
							},
						},
						Ports: []servicemeshapi.GatewayListener{},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveIngressGatewayReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.IngressGateway, error) {
					igRef := &servicemeshapi.IngressGateway{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return igRef, nil
				},
			},
			wantErr: expectation{false, string(commons.IngressGatewayReferenceIsNotUnique)},
		},
		{
			name: "Spec refers to deleting ingress gateway",
			args: args{
				ingressGatewayDeployment: &servicemeshapi.IngressGatewayDeployment{
					Spec: servicemeshapi.IngressGatewayDeploymentSpec{
						IngressGateway: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-ig",
							},
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveIngressGatewayReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.IngressGateway, error) {
					currentTime := metav1.Now()
					igRef := &servicemeshapi.IngressGateway{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: &currentTime,
						},
					}
					return igRef, nil
				},
			},
			wantErr: expectation{false, string(commons.IngressGatewayReferenceIsDeleting)},
		},
		{
			name: "Spec with  ocid ",
			args: args{
				ingressGatewayDeployment: &servicemeshapi.IngressGatewayDeployment{
					Spec: servicemeshapi.IngressGatewayDeploymentSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "123",
						},
						Deployment: servicemeshapi.IngressDeployment{
							Autoscaling: &servicemeshapi.Autoscaling{
								MinPods: 5,
								MaxPods: 5,
							},
						},
						Ports: []servicemeshapi.GatewayListener{},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveIngressGatewayReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.IngressGateway, error) {
					igRef := &servicemeshapi.IngressGateway{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return igRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Spec with duplicate secrets",
			args: args{
				ingressGatewayDeployment: &servicemeshapi.IngressGatewayDeployment{
					Spec: servicemeshapi.IngressGatewayDeploymentSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "123",
						},
						Deployment: servicemeshapi.IngressDeployment{
							Autoscaling: &servicemeshapi.Autoscaling{
								MinPods: 5,
								MaxPods: 5,
							},
						},
						Ports: []servicemeshapi.GatewayListener{},
						Secrets: []servicemeshapi.SecretReference{
							{SecretName: "bookinfo-tls-secret"},
							{SecretName: "bookinfo-tls-secret"},
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveIngressGatewayReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.IngressGateway, error) {
					igRef := &servicemeshapi.IngressGateway{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return igRef, nil
				},
			},
			wantErr: expectation{false, "spec.ingressgatewaydeployment has duplicate secret bookinfo-tls-secret"},
		},
		{
			name: "Spec with valid secrets",
			args: args{
				ingressGatewayDeployment: &servicemeshapi.IngressGatewayDeployment{
					Spec: servicemeshapi.IngressGatewayDeploymentSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "123",
						},
						Deployment: servicemeshapi.IngressDeployment{
							Autoscaling: &servicemeshapi.Autoscaling{
								MinPods: 5,
								MaxPods: 5,
							},
						},
						Ports: []servicemeshapi.GatewayListener{},
						Secrets: []servicemeshapi.SecretReference{
							{SecretName: "bookinfo-tls-secret"},
							{SecretName: "bookinfo-tls-secret2"},
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveIngressGatewayReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.IngressGateway, error) {
					igRef := &servicemeshapi.IngressGateway{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return igRef, nil
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
			meshValidator := NewIngressGatewayDeploymentValidator(resolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("IGD")})
			meshServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(meshValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("IGD")})

			if tt.fields.ResolveIngressGatewayReference != nil {
				resolver.EXPECT().ResolveIngressGatewayReference(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveIngressGatewayReference).AnyTimes()
			}

			if tt.fields.ResolveResourceRef != nil {
				resolver.EXPECT().ResolveResourceRef(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveResourceRef).AnyTimes()
			}

			response := meshServiceValidationManager.ValidateCreateRequest(ctx, tt.args.ingressGatewayDeployment)
			if tt.wantErr.reason != "" {
				tt.wantErr.reason = meshErrors.GetValidationErrorMessage(tt.args.ingressGatewayDeployment, tt.wantErr.reason)
			}
			assert.Equal(t, tt.wantErr.allowed, response.Allowed)
			assert.Equal(t, tt.wantErr.reason, string(response.Result.Reason))

		})
	}

}

func Test_IngressGatewayDeploymentValidateDelete(t *testing.T) {
	type args struct {
		ingressGatewayDeployment *servicemeshapi.IngressGatewayDeployment
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
				ingressGatewayDeployment: &servicemeshapi.IngressGatewayDeployment{
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
				ingressGatewayDeployment: &servicemeshapi.IngressGatewayDeployment{
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
				ingressGatewayDeployment: &servicemeshapi.IngressGatewayDeployment{
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
				ingressGatewayDeployment: &servicemeshapi.IngressGatewayDeployment{
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
			meshValidator := NewIngressGatewayDeploymentValidator(resolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("IGD")})
			meshServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(meshValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("IGD")})
			response := meshServiceValidationManager.ValidateDeleteRequest(ctx, tt.args.ingressGatewayDeployment)
			if tt.wantErr.reason != "" {
				tt.wantErr.reason = meshErrors.GetValidationErrorMessage(tt.args.ingressGatewayDeployment, tt.wantErr.reason)
			}
			assert.Equal(t, tt.wantErr.allowed, response.Allowed)
			assert.Equal(t, tt.wantErr.reason, string(response.Result.Reason))
		})
	}
}

func Test_IngressGatewayDeploymentValidateUpdate(t *testing.T) {
	type args struct {
		ingressGatewayDeployment    *servicemeshapi.IngressGatewayDeployment
		oldIngressGatewayDeployment *servicemeshapi.IngressGatewayDeployment
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
			name: "Update when state is not Active",
			args: args{
				ingressGatewayDeployment: &servicemeshapi.IngressGatewayDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
				},
				oldIngressGatewayDeployment: &servicemeshapi.IngressGatewayDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
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
									Status: metav1.ConditionFalse,
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
				ingressGatewayDeployment: &servicemeshapi.IngressGatewayDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
				},
				oldIngressGatewayDeployment: &servicemeshapi.IngressGatewayDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshDependenciesActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionUnknown,
								},
							},
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionFalse,
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
				ingressGatewayDeployment: &servicemeshapi.IngressGatewayDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
				},
				oldIngressGatewayDeployment: &servicemeshapi.IngressGatewayDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
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
									Status: metav1.ConditionFalse,
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
			name: "Update when Ingress Ref is changed",
			args: args{
				ingressGatewayDeployment: &servicemeshapi.IngressGatewayDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.IngressGatewayDeploymentSpec{
						IngressGateway: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-ig-1",
							},
						},
						Deployment: servicemeshapi.IngressDeployment{
							Autoscaling: &servicemeshapi.Autoscaling{
								MinPods: 5,
								MaxPods: 3,
							},
						},
						Ports: []servicemeshapi.GatewayListener{},
					},
				},
				oldIngressGatewayDeployment: &servicemeshapi.IngressGatewayDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.IngressGatewayDeploymentSpec{
						IngressGateway: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-ig",
							},
						},
						Deployment: servicemeshapi.IngressDeployment{
							Autoscaling: &servicemeshapi.Autoscaling{
								MinPods: 5,
								MaxPods: 3,
							},
						},
						Ports: []servicemeshapi.GatewayListener{},
					},
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
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.IngressGatewayReferenceIsImmutable)},
		},
		{
			name: "Update when OCID",
			args: args{
				ingressGatewayDeployment: &servicemeshapi.IngressGatewayDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.IngressGatewayDeploymentSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "1",
						},
						Deployment: servicemeshapi.IngressDeployment{
							Autoscaling: &servicemeshapi.Autoscaling{
								MinPods: 5,
								MaxPods: 3,
							},
						},
						Ports: []servicemeshapi.GatewayListener{},
					},
				},
				oldIngressGatewayDeployment: &servicemeshapi.IngressGatewayDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.IngressGatewayDeploymentSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "2",
						},
						Deployment: servicemeshapi.IngressDeployment{
							Autoscaling: &servicemeshapi.Autoscaling{
								MinPods: 5,
								MaxPods: 3,
							},
						},
						Ports: []servicemeshapi.GatewayListener{},
					},
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
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.IngressGatewayReferenceIsImmutable)},
		},

		{
			name: "Update when status is added",
			args: args{
				ingressGatewayDeployment: &servicemeshapi.IngressGatewayDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec:       servicemeshapi.IngressGatewayDeploymentSpec{},
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
				oldIngressGatewayDeployment: &servicemeshapi.IngressGatewayDeployment{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec:       servicemeshapi.IngressGatewayDeploymentSpec{},
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
			meshValidator := NewIngressGatewayDeploymentValidator(resolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("IGD")})
			meshServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(meshValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("IGD")})
			response := meshServiceValidationManager.ValidateUpdateRequest(ctx, tt.args.ingressGatewayDeployment, tt.args.oldIngressGatewayDeployment)
			if tt.wantErr.reason != "" {
				tt.wantErr.reason = meshErrors.GetValidationErrorMessage(tt.args.ingressGatewayDeployment, tt.wantErr.reason)
			}
			assert.Equal(t, tt.wantErr.allowed, response.Allowed)
			assert.Equal(t, tt.wantErr.reason, string(response.Result.Reason))

		})
	}
}

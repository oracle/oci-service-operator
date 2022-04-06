/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package validations

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/oracle/oci-service-operator/api/v1beta1"
	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	meshMocks "github.com/oracle/oci-service-operator/test/mocks/servicemesh"
)

func Test_IsRefNonUnique(t *testing.T) {
	type args struct {
		resourceRef *servicemeshapi.ResourceRef
		id          api.OCID
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Non-unique reference",
			args: args{
				resourceRef: &servicemeshapi.ResourceRef{
					Name: "ref-name",
				},
				id: "ref-ocid",
			},
			wantErr: true,
		},
		{
			name: "Reference contains only name",
			args: args{
				resourceRef: &servicemeshapi.ResourceRef{
					Name: "ref-name",
				},
			},
			wantErr: false,
		},
		{
			name: "Reference contains only OCID",
			args: args{
				id:          "ref-ocid",
				resourceRef: nil,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantErr, IsRefNotUnique(tt.args.resourceRef, tt.args.id))
		})
	}
}

func Test_IsRefEmpty(t *testing.T) {
	type args struct {
		resourceRef *servicemeshapi.ResourceRef
		id          api.OCID
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Mesh reference contains both name and OCID",
			args: args{
				id: "ref-ocid",
				resourceRef: &servicemeshapi.ResourceRef{
					Name: "ref-name",
				},
			},
			wantErr: false,
		},
		{
			name: "Mesh reference contains only name",
			args: args{
				resourceRef: &servicemeshapi.ResourceRef{
					Name: "ref-name",
				},
			},
			wantErr: false,
		},
		{
			name: "Mesh reference contains only OCID",
			args: args{
				id: "ref-ocid",
			},
			wantErr: false,
		},
		{
			name: "Empty reference",
			args: args{
				resourceRef: nil,
				id:          "",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantErr, IsRefEmpty(tt.args.resourceRef, tt.args.id))
		})
	}
}

func Test_ValidateMeshRef(t *testing.T) {
	type args struct {
		meshRef *servicemeshapi.RefOrId
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "Non-unique mesh reference",
			args: args{
				meshRef: &servicemeshapi.RefOrId{
					Id:          "ref-id",
					ResourceRef: &servicemeshapi.ResourceRef{Name: "ref-name"},
				},
			},
			wantErr: errors.New(string(commons.MeshReferenceIsNotUnique)),
		},
		{
			name: "Empty mesh reference",
			args: args{
				meshRef: &servicemeshapi.RefOrId{},
			},
			wantErr: errors.New(string(commons.MeshReferenceIsEmpty)),
		},
		{
			name: "Mesh reference contains only name",
			args: args{
				meshRef: &servicemeshapi.RefOrId{
					ResourceRef: &servicemeshapi.ResourceRef{Name: "ref-name"},
				},
			},
			wantErr: nil,
		},
		{
			name: "Mesh reference contains only OCID",
			args: args{
				meshRef: &servicemeshapi.RefOrId{
					Id: "ref-id",
				},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMeshRef(tt.args.meshRef)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_ValidateVSRef(t *testing.T) {
	type args struct {
		vsRef *servicemeshapi.RefOrId
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "Non-unique mesh reference",
			args: args{
				vsRef: &servicemeshapi.RefOrId{
					Id:          "ref-id",
					ResourceRef: &servicemeshapi.ResourceRef{Name: "ref-name"},
				},
			},
			wantErr: errors.New(string(commons.VirtualServiceReferenceIsNotUnique)),
		},
		{
			name: "Empty mesh reference",
			args: args{
				vsRef: &servicemeshapi.RefOrId{
					ResourceRef: nil,
				},
			},
			wantErr: errors.New(string(commons.VirtualServiceReferenceIsEmpty)),
		},
		{
			name: "Mesh reference contains only name",
			args: args{
				vsRef: &servicemeshapi.RefOrId{
					ResourceRef: &servicemeshapi.ResourceRef{Name: "ref-name"},
				},
			},
			wantErr: nil,
		},
		{
			name: "Mesh reference contains only OCID",
			args: args{
				vsRef: &servicemeshapi.RefOrId{
					Id: "ref-id",
				},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVSRef(tt.args.vsRef)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_ValidateVDRef(t *testing.T) {
	type args struct {
		vdRef *servicemeshapi.RefOrId
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "Non-unique VD reference",
			args: args{
				vdRef: &servicemeshapi.RefOrId{
					Id:          "ref-id",
					ResourceRef: &servicemeshapi.ResourceRef{Name: "ref-name"},
				},
			},
			wantErr: errors.New(string(commons.VirtualDeploymentReferenceIsNotUnique)),
		},
		{
			name: "Empty VD reference",
			args: args{
				vdRef: &servicemeshapi.RefOrId{
					ResourceRef: nil,
				},
			},
			wantErr: errors.New(string(commons.VirtualDeploymentReferenceIsEmpty)),
		},
		{
			name: "VD reference contains only name",
			args: args{
				vdRef: &servicemeshapi.RefOrId{
					ResourceRef: &servicemeshapi.ResourceRef{Name: "ref-name"},
				},
			},
			wantErr: nil,
		},
		{
			name: "VD reference contains only OCID",
			args: args{
				vdRef: &servicemeshapi.RefOrId{
					Id: "ref-id",
				},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVDRef(tt.args.vdRef)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_ValidateIGRef(t *testing.T) {
	type args struct {
		igRef *servicemeshapi.RefOrId
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "Non-unique mesh reference",
			args: args{
				igRef: &servicemeshapi.RefOrId{
					Id:          "ref-id",
					ResourceRef: &servicemeshapi.ResourceRef{Name: "ref-name"},
				},
			},
			wantErr: errors.New(string(commons.IngressGatewayReferenceIsNotUnique)),
		},
		{
			name: "Empty mesh reference",
			args: args{
				igRef: &servicemeshapi.RefOrId{},
			},
			wantErr: errors.New(string(commons.IngressGatewayReferenceIsEmpty)),
		},
		{
			name: "Mesh reference contains only name",
			args: args{
				igRef: &servicemeshapi.RefOrId{
					ResourceRef: &servicemeshapi.ResourceRef{Name: "ref-name"},
				},
			},
			wantErr: nil,
		},
		{
			name: "Mesh reference contains only OCID",
			args: args{
				igRef: &servicemeshapi.RefOrId{
					Id: "ref-id",
				},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIGRef(tt.args.igRef)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_IsVSPresent(t *testing.T) {
	type args struct {
		vsRef *servicemeshapi.ResourceRef
		meta  *metav1.ObjectMeta
	}
	type expectation struct {
		allowed bool
		reason  string
	}
	type fields struct {
		ResolveResourceRef             func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef
		ResolveVirtualServiceReference func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error)
	}
	tests := []struct {
		name    string
		args    args
		fields  fields
		wantErr expectation
	}{
		{
			name: "Valid VS reference",
			args: args{
				vsRef: &servicemeshapi.ResourceRef{Name: "ref-name"},
				meta: &metav1.ObjectMeta{
					Name:      "meta-name",
					Namespace: "meta-namespace",
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
			name: "Deleting VS reference",
			args: args{
				vsRef: &servicemeshapi.ResourceRef{},
				meta: &metav1.ObjectMeta{
					Name:      "meta-name",
					Namespace: "meta-namespace",
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
			name: "VS reference contains only name",
			args: args{
				vsRef: &servicemeshapi.ResourceRef{Name: "ref-name"},
				meta: &metav1.ObjectMeta{
					Name:      "meta-name",
					Namespace: "meta-namespace",
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
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			resolver := meshMocks.NewMockResolver(ctrl)

			if tt.fields.ResolveResourceRef != nil {
				resolver.EXPECT().ResolveResourceRef(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveResourceRef).AnyTimes()
			}

			if tt.fields.ResolveVirtualServiceReference != nil {
				resolver.EXPECT().ResolveVirtualServiceReference(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveVirtualServiceReference).AnyTimes()
			}

			allowed, reason := IsVSPresent(resolver, ctx, tt.args.vsRef, tt.args.meta)
			assert.Equal(t, tt.wantErr.allowed, allowed)
			assert.Equal(t, tt.wantErr.reason, reason)
		})
	}
}

func Test_IsVDPresent(t *testing.T) {
	type args struct {
		vdRef *servicemeshapi.ResourceRef
		meta  *metav1.ObjectMeta
	}
	type expectation struct {
		allowed bool
		reason  string
	}
	type fields struct {
		ResolveResourceRef                func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef
		ResolveVirtualDeploymentReference func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error)
	}
	tests := []struct {
		name    string
		args    args
		fields  fields
		wantErr expectation
	}{
		{
			name: "Valid VD reference",
			args: args{
				vdRef: &servicemeshapi.ResourceRef{Name: "ref-name"},
				meta: &metav1.ObjectMeta{
					Name:      "meta-name",
					Namespace: "meta-namespace",
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
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Deleting VD reference",
			args: args{
				vdRef: &servicemeshapi.ResourceRef{},
				meta: &metav1.ObjectMeta{
					Name:      "meta-name",
					Namespace: "meta-namespace",
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
			name: "VD reference contains only name",
			args: args{
				vdRef: &servicemeshapi.ResourceRef{Name: "ref-name"},
				meta: &metav1.ObjectMeta{
					Name:      "meta-name",
					Namespace: "meta-namespace",
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			resolver := meshMocks.NewMockResolver(ctrl)

			if tt.fields.ResolveResourceRef != nil {
				resolver.EXPECT().ResolveResourceRef(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveResourceRef).AnyTimes()
			}

			if tt.fields.ResolveVirtualDeploymentReference != nil {
				resolver.EXPECT().ResolveVirtualDeploymentReference(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveVirtualDeploymentReference).AnyTimes()
			}

			allowed, reason := IsVDPresent(resolver, ctx, tt.args.vdRef, tt.args.meta)
			assert.Equal(t, tt.wantErr.allowed, allowed)
			assert.Equal(t, tt.wantErr.reason, reason)
		})
	}
}

func Test_IsIGPresent(t *testing.T) {
	type args struct {
		igRef *servicemeshapi.ResourceRef
		meta  *metav1.ObjectMeta
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
			name: "Valid IG reference",
			args: args{
				igRef: &servicemeshapi.ResourceRef{Name: "ref-name"},
				meta: &metav1.ObjectMeta{
					Name:      "meta-name",
					Namespace: "meta-namespace",
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
			name: "Deleting IG reference",
			args: args{
				igRef: &servicemeshapi.ResourceRef{},
				meta: &metav1.ObjectMeta{
					Name:      "meta-name",
					Namespace: "meta-namespace",
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
			name: "IG reference contains only name",
			args: args{
				igRef: &servicemeshapi.ResourceRef{Name: "ref-name"},
				meta: &metav1.ObjectMeta{
					Name:      "meta-name",
					Namespace: "meta-namespace",
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveIngressGatewayReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.IngressGateway, error) {
					return nil, errors.New("CANNOT FETCH")
				},
			},
			wantErr: expectation{false, string(commons.IngressGatewayReferenceNotFound)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			resolver := meshMocks.NewMockResolver(ctrl)

			if tt.fields.ResolveResourceRef != nil {
				resolver.EXPECT().ResolveResourceRef(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveResourceRef).AnyTimes()
			}

			if tt.fields.ResolveIngressGatewayReference != nil {
				resolver.EXPECT().ResolveIngressGatewayReference(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveIngressGatewayReference).AnyTimes()
			}

			allowed, reason := IsIGPresent(resolver, ctx, tt.args.igRef, tt.args.meta)
			assert.Equal(t, tt.wantErr.allowed, allowed)
			assert.Equal(t, tt.wantErr.reason, reason)
		})
	}
}

func Test_IsServicePresent(t *testing.T) {
	type args struct {
		serviceRef *servicemeshapi.ResourceRef
		meta       *metav1.ObjectMeta
	}
	type expectation struct {
		allowed bool
		reason  string
	}
	type fields struct {
		ResolveResourceRef      func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef
		ResolveServiceReference func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*corev1.Service, error)
	}
	tests := []struct {
		name    string
		args    args
		fields  fields
		wantErr expectation
	}{
		{
			name: "Valid Service reference",
			args: args{
				serviceRef: &servicemeshapi.ResourceRef{Name: "ref-name"},
				meta: &metav1.ObjectMeta{
					Name:      "meta-name",
					Namespace: "meta-namespace",
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
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
			name: "Deleting Service reference",
			args: args{
				serviceRef: &servicemeshapi.ResourceRef{},
				meta: &metav1.ObjectMeta{
					Name:      "meta-name",
					Namespace: "meta-namespace",
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
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
			name: "Service reference contains only name",
			args: args{
				serviceRef: &servicemeshapi.ResourceRef{Name: "ref-name"},
				meta: &metav1.ObjectMeta{
					Name:      "meta-name",
					Namespace: "meta-namespace",
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
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
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			resolver := meshMocks.NewMockResolver(ctrl)

			if tt.fields.ResolveResourceRef != nil {
				resolver.EXPECT().ResolveResourceRef(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveResourceRef).AnyTimes()
			}

			if tt.fields.ResolveServiceReference != nil {
				resolver.EXPECT().ResolveServiceReferenceWithApiReader(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveServiceReference).AnyTimes()
			}

			allowed, reason := IsServicePresent(resolver, ctx, tt.args.serviceRef, tt.args.meta)
			assert.Equal(t, tt.wantErr.allowed, allowed)
			assert.Equal(t, tt.wantErr.reason, reason)
		})
	}
}

func Test_IsSpecNameChanged(t *testing.T) {
	type args struct {
		name    *servicemeshapi.Name
		oldName *servicemeshapi.Name
	}
	name1 := servicemeshapi.Name("name-1")
	name2 := servicemeshapi.Name("name-2")

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Same name as old name",
			args: args{
				name:    &name1,
				oldName: &name1,
			},
			want: false,
		},
		{
			name: "Different name from old name",
			args: args{
				name:    &name2,
				oldName: &name1,
			},
			want: true,
		},
		{
			name: "New name was null when old was given",
			args: args{
				name:    nil,
				oldName: &name1,
			},
			want: true,
		},
		{
			name: "New name was given when old was null",
			args: args{
				name:    &name2,
				oldName: nil,
			},
			want: true,
		},
		{
			name: "No names were given",
			args: args{
				name:    nil,
				oldName: nil,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSpecNameChanged(tt.args.name, tt.args.oldName)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_IsMetadataNameValid(t *testing.T) {
	type args struct {
		name string
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
			name: "Valid name",
			args: args{
				name: "my-name",
			},
			wantErr: expectation{
				allowed: true,
				reason:  "",
			},
		},
		{
			name: "Invalid name",
			args: args{
				name: "Thisisaverylongnametotestthefunctionthereisnopointinhavingthislongofanamebutmaybesometimeswewillhavesuchalongnameandweneedtoensurethatourservicecanhandlethisnamewhenitisoverhundredandninetycharacters",
			},
			wantErr: expectation{
				allowed: false,
				reason:  string(commons.MetadataNameLengthExceeded),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, reason := IsMetadataNameValid(tt.args.name)
			assert.Equal(t, tt.wantErr.allowed, allowed)
			assert.Equal(t, tt.wantErr.reason, reason)
		})
	}
}

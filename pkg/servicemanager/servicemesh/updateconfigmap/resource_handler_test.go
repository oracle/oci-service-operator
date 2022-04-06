/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package updateconfigmap

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/equality"
	mocks "github.com/oracle/oci-service-operator/test/mocks/servicemesh"
	. "github.com/oracle/oci-service-operator/test/servicemesh/k8s"
)

func TestUpdateLatestProxyVersion(t *testing.T) {
	type fields struct {
		GetProxyDetails func(ctx context.Context) (*string, error)
	}
	tests := []struct {
		name      string
		fields    fields
		configMap *corev1.ConfigMap
		want      *corev1.ConfigMap
		wantErr   error
	}{
		{
			name: "Config map with outdated Proxy version",
			fields: fields{
				GetProxyDetails: func(ctx context.Context) (*string, error) {
					newProxyVersion := "sm-proxy-image-v1.17.0.2"
					return common.String(newProxyVersion), nil
				},
			},
			configMap: NewConfigMap(commons.OsokNamespace, commons.MeshConfigMapName, map[string]string{
				commons.CpEndpointInMeshConfigMap:  "http://servicemeshtest.us-ashburn-1.oci.oc-test.com",
				commons.MdsEndpointInMeshConfigMap: "http://144.25.97.129:443", commons.ProxyLabelInMeshConfigMap: "sm-proxy-image-v1.17.0.1"}),
			want: NewConfigMap(commons.OsokNamespace, commons.MeshConfigMapName, map[string]string{
				commons.CpEndpointInMeshConfigMap:  "http://servicemeshtest.us-ashburn-1.oci.oc-test.com",
				commons.MdsEndpointInMeshConfigMap: "http://144.25.97.129:443", commons.ProxyLabelInMeshConfigMap: "sm-proxy-image-v1.17.0.2"}),
			wantErr: nil,
		},
		{
			name: "Config map with same Proxy version",
			fields: fields{
				GetProxyDetails: func(ctx context.Context) (*string, error) {
					newProxyVersion := "sm-proxy-image-v1.17.0.1"
					return common.String(newProxyVersion), nil
				},
			},
			configMap: NewConfigMap(commons.OsokNamespace, commons.MeshConfigMapName, map[string]string{
				commons.CpEndpointInMeshConfigMap:  "http://servicemeshtest.us-ashburn-1.oci.oc-test.com",
				commons.MdsEndpointInMeshConfigMap: "http://144.25.97.129:443", commons.ProxyLabelInMeshConfigMap: "sm-proxy-image-v1.17.0.1"}),
			want: NewConfigMap(commons.OsokNamespace, commons.MeshConfigMapName, map[string]string{
				commons.CpEndpointInMeshConfigMap:  "http://servicemeshtest.us-ashburn-1.oci.oc-test.com",
				commons.MdsEndpointInMeshConfigMap: "http://144.25.97.129:443", commons.ProxyLabelInMeshConfigMap: "sm-proxy-image-v1.17.0.1"}),
			wantErr: nil,
		},
		{
			name:   "Config map with autoUpdate disabled and valid version",
			fields: fields{},
			configMap: NewConfigMap(commons.OsokNamespace, commons.MeshConfigMapName, map[string]string{
				commons.ProxyLabelInMeshConfigMap: "sm-proxy-image-v1.17.0.1",
				commons.AutoUpdateProxyVersion:    "false"}),
			want: NewConfigMap(commons.OsokNamespace, commons.MeshConfigMapName, map[string]string{
				commons.ProxyLabelInMeshConfigMap: "sm-proxy-image-v1.17.0.1",
				commons.AutoUpdateProxyVersion:    "false"}),
			wantErr: nil,
		},
		{
			name: "Config map with autoUpdate enabled and valid version",
			fields: fields{
				GetProxyDetails: func(ctx context.Context) (*string, error) {
					newProxyVersion := "sm-proxy-image-v1.17.0.2"
					return common.String(newProxyVersion), nil
				},
			},
			configMap: NewConfigMap(commons.OsokNamespace, commons.MeshConfigMapName, map[string]string{
				commons.ProxyLabelInMeshConfigMap: "sm-proxy-image-v1.17.0.1",
				commons.AutoUpdateProxyVersion:    "true"}),
			want: NewConfigMap(commons.OsokNamespace, commons.MeshConfigMapName, map[string]string{
				commons.ProxyLabelInMeshConfigMap: "sm-proxy-image-v1.17.0.2",
				commons.AutoUpdateProxyVersion:    "true"}),
			wantErr: nil,
		},
		{
			name: "Config map with autoUpdate disabled and no version",
			fields: fields{
				GetProxyDetails: func(ctx context.Context) (*string, error) {
					newProxyVersion := "sm-proxy-image-v1.17.0.2"
					return common.String(newProxyVersion), nil
				},
			},
			configMap: NewConfigMap(commons.OsokNamespace, commons.MeshConfigMapName, map[string]string{
				commons.AutoUpdateProxyVersion: "false"}),
			want: NewConfigMap(commons.OsokNamespace, commons.MeshConfigMapName, map[string]string{
				commons.ProxyLabelInMeshConfigMap: "sm-proxy-image-v1.17.0.2",
				commons.AutoUpdateProxyVersion:    "false"}),
			wantErr: nil,
		},
		{
			name: "Config map with outdated Proxy version and extra fields in data",
			fields: fields{
				GetProxyDetails: func(ctx context.Context) (*string, error) {
					newProxyVersion := "sm-proxy-image-v1.17.0.2"
					return common.String(newProxyVersion), nil
				},
			},
			configMap: NewConfigMap(commons.OsokNamespace, commons.MeshConfigMapName, map[string]string{
				commons.CpEndpointInMeshConfigMap:  "http://servicemeshtest.us-ashburn-1.oci.oc-test.com",
				commons.MdsEndpointInMeshConfigMap: "http://144.25.97.129:443", commons.ProxyLabelInMeshConfigMap: "sm-proxy-image-v1.17.0.1", "extraFieldKey": "extraFieldValue"}),
			want: NewConfigMap(commons.OsokNamespace, commons.MeshConfigMapName, map[string]string{
				commons.CpEndpointInMeshConfigMap:  "http://servicemeshtest.us-ashburn-1.oci.oc-test.com",
				commons.MdsEndpointInMeshConfigMap: "http://144.25.97.129:443", commons.ProxyLabelInMeshConfigMap: "sm-proxy-image-v1.17.0.2", "extraFieldKey": "extraFieldValue"}),
			wantErr: nil,
		},
		{
			name: "Error while polling the control plane",
			fields: fields{
				GetProxyDetails: func(ctx context.Context) (*string, error) {
					return nil, errors.New("error in getting proxy details from the control plane client")
				},
			},
			configMap: NewConfigMap(commons.OsokNamespace, commons.MeshConfigMapName, map[string]string{
				commons.CpEndpointInMeshConfigMap:  "http://servicemeshtest.us-ashburn-1.oci.oc-test.com",
				commons.MdsEndpointInMeshConfigMap: "http://144.25.97.129:443", commons.ProxyLabelInMeshConfigMap: "sm-proxy-image-v1.17.0.1"}),
			want: NewConfigMap(commons.OsokNamespace, commons.MeshConfigMapName, map[string]string{
				commons.CpEndpointInMeshConfigMap:  "http://servicemeshtest.us-ashburn-1.oci.oc-test.com",
				commons.MdsEndpointInMeshConfigMap: "http://144.25.97.129:443", commons.ProxyLabelInMeshConfigMap: "sm-proxy-image-v1.17.0.2"}),
			wantErr: errors.New("error in getting proxy details from the control plane client"),
		},
		{
			name: "Config map with no initial data",
			fields: fields{
				GetProxyDetails: func(ctx context.Context) (*string, error) {
					newProxyVersion := "sm-proxy-image-v1.17.0.2"
					return common.String(newProxyVersion), nil
				},
			},
			configMap: NewConfigMap(commons.OsokNamespace, commons.MeshConfigMapName, nil),
			want: NewConfigMap(commons.OsokNamespace, commons.MeshConfigMapName, map[string]string{
				commons.ProxyLabelInMeshConfigMap: "sm-proxy-image-v1.17.0.2"}),
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			err := clientgoscheme.AddToScheme(k8sSchema)
			assert.NoError(t, err)
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).WithObjects(tt.configMap).Build()
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			smClient := mocks.NewMockServiceMeshClient(mockCtrl)

			h := NewDefaultResourceHandler(k8sClient, smClient,
				loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("resourceHandler").WithName("UpdateConfigMap")})

			if tt.fields.GetProxyDetails != nil {
				smClient.EXPECT().GetProxyDetails(gomock.Any()).DoAndReturn(tt.fields.GetProxyDetails)
			} else {
				smClient.EXPECT().GetProxyDetails(gomock.Any()).Times(0)
			}

			err = h.UpdateLatestProxyVersion(ctx, tt.configMap)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				opts := equality.IgnoreFakeClientPopulatedFields()
				_ = k8sClient.Get(ctx, types.NamespacedName{Namespace: commons.OsokNamespace, Name: commons.MeshConfigMapName}, tt.configMap)
				assert.True(t, cmp.Equal(tt.want, tt.configMap, opts), "diff", cmp.Diff(tt.want, tt.configMap, opts))
			}
		})
	}
}

func TestUpdateCpClientHost(t *testing.T) {
	type fields struct {
		CPSetClientHost func(endpoint string)
	}
	tests := []struct {
		name      string
		fields    fields
		configMap *corev1.ConfigMap
		wantErr   error
	}{
		{
			name: "Config map with valid CP endpoint",
			fields: fields{
				CPSetClientHost: func(endpoint string) {},
			},
			configMap: NewConfigMap(commons.OsokNamespace, commons.MeshConfigMapName, map[string]string{
				commons.CpEndpointInMeshConfigMap: "http://servicemeshtest.us-ashburn-1.oci.oc-test.com"}),
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sSchema := runtime.NewScheme()
			err := clientgoscheme.AddToScheme(k8sSchema)
			assert.NoError(t, err)
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).WithObjects(tt.configMap).Build()
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			smClient := mocks.NewMockServiceMeshClient(mockCtrl)

			h := NewDefaultResourceHandler(k8sClient, smClient,
				loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("resourceHandler").WithName("UpdateConfigMap")})

			if tt.fields.CPSetClientHost != nil {
				smClient.EXPECT().SetClientHost(tt.configMap.Data[commons.CpEndpointInMeshConfigMap])
			}

			h.UpdateServiceMeshClientHost(tt.configMap)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

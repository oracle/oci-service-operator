/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package conversions_test

import (
	"testing"

	sdk "github.com/oracle/oci-go-sdk/v65/servicemesh"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/conversions"
)

var (
	vdName        = "my-virtualdeployment"
	vdSpecName    = v1beta1.Name(vdName)
	vsId          = api.OCID("my-service-id")
	vdDescription = v1beta1.Description("This is Virtual Deployment")
)

func TestConvert_CRD_VirtualDeployment_To_SDK_VirtualDeployment(t *testing.T) {
	defaultPort := 80
	isLoggingEnabled := true

	defaultCrdVirtualDeployment := v1beta1.VirtualDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vdName,
			Namespace: "my-namespace-id",
		},
		Spec: v1beta1.VirtualDeploymentSpec{
			CompartmentId: "my-compartment",
			Name:          &vdSpecName,
			Description:   &vdDescription,
			VirtualService: v1beta1.RefOrId{
				Id: "my-service-id",
			},
			Listener: []v1beta1.Listener{
				{
					Protocol: "HTTP",
					Port:     80,
				},
			},
			AccessLogging: &v1beta1.AccessLogging{
				IsEnabled: true,
			},
			ServiceDiscovery: v1beta1.ServiceDiscovery{
				Type:     v1beta1.ServiceDiscoveryTypeDns,
				Hostname: "oracle.com",
			},
			TagResources: api.TagResources{
				FreeFormTags: map[string]string{"freeformTag1": "value1"},
				DefinedTags: map[string]api.MapValue{
					"definedTag1": {"foo": "bar"},
				},
			},
		},
		Status: v1beta1.ServiceMeshStatus{
			VirtualDeploymentId: "vdId",
		},
	}
	type args struct {
		crdObj *v1beta1.VirtualDeployment
		sdkObj *sdk.VirtualDeployment
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *sdk.VirtualDeployment
	}{
		{
			name: "Happy Path",
			args: args{
				crdObj: &defaultCrdVirtualDeployment,
				sdkObj: &sdk.VirtualDeployment{
					FreeformTags: map[string]string{"freeformTag2": "value2"},
					DefinedTags: map[string]map[string]interface{}{
						"definedTag2": {"key": "val"},
					},
				},
			},
			wantSDKObj: &sdk.VirtualDeployment{
				Id:               conversions.String("vdId"),
				Name:             conversions.String(vdName),
				Description:      conversions.String("This is Virtual Deployment"),
				CompartmentId:    conversions.String("my-compartment"),
				VirtualServiceId: conversions.String("my-service-id"),
				Listeners: []sdk.VirtualDeploymentListener{
					{
						Protocol: sdk.VirtualDeploymentListenerProtocolHttp,
						Port:     &defaultPort,
					},
				},
				ServiceDiscovery: sdk.DnsServiceDiscoveryConfiguration{
					Hostname: conversions.String("oracle.com"),
				},
				FreeformTags: map[string]string{
					"freeformTag1": "value1",
					"freeformTag2": "value2",
				},
				DefinedTags: map[string]map[string]interface{}{
					"definedTag1": {"foo": "bar"},
					"definedTag2": {"key": "val"},
				},
				AccessLogging: &sdk.AccessLoggingConfiguration{
					IsEnabled: &isLoggingEnabled,
				},
			},
		},
		{
			name: "virtual deployment with no access logging supplied",
			args: args{
				crdObj: &v1beta1.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      vdName,
						Namespace: "my-namespace-id",
					},
					Spec: v1beta1.VirtualDeploymentSpec{
						CompartmentId: "my-compartment",
						Name:          &vdSpecName,
						Description:   &vdDescription,
						VirtualService: v1beta1.RefOrId{
							Id: "my-service-id",
						},
						Listener: []v1beta1.Listener{
							{
								Protocol: "HTTP",
								Port:     80,
							},
						},
						ServiceDiscovery: v1beta1.ServiceDiscovery{
							Type:     v1beta1.ServiceDiscoveryTypeDns,
							Hostname: "oracle.com",
						},
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{"freeformTag1": "value1"},
							DefinedTags: map[string]api.MapValue{
								"definedTag1": {"foo": "bar"},
							},
						},
					},
					Status: v1beta1.ServiceMeshStatus{
						VirtualDeploymentId: "vdId",
					},
				},
				sdkObj: &sdk.VirtualDeployment{},
			},
			wantSDKObj: &sdk.VirtualDeployment{
				Id:               conversions.String("vdId"),
				Name:             conversions.String(vdName),
				Description:      conversions.String("This is Virtual Deployment"),
				CompartmentId:    conversions.String("my-compartment"),
				VirtualServiceId: conversions.String("my-service-id"),
				Listeners: []sdk.VirtualDeploymentListener{
					{
						Protocol: sdk.VirtualDeploymentListenerProtocolHttp,
						Port:     &defaultPort,
					},
				},
				ServiceDiscovery: sdk.DnsServiceDiscoveryConfiguration{
					Hostname: conversions.String("oracle.com"),
				},
				FreeformTags: map[string]string{"freeformTag1": "value1"},
				DefinedTags: map[string]map[string]interface{}{
					"definedTag1": {"foo": "bar"},
				},
			},
		},
		{
			name: "virtual deployment with no spec name supplied",
			args: args{
				crdObj: &v1beta1.VirtualDeployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      vdName,
						Namespace: "my-namespace-id",
					},
					Spec: v1beta1.VirtualDeploymentSpec{
						CompartmentId: "my-compartment",
						VirtualService: v1beta1.RefOrId{
							Id: "my-service-id",
						},
						Listener: []v1beta1.Listener{
							{
								Protocol: "HTTP",
								Port:     80,
							},
						},
						ServiceDiscovery: v1beta1.ServiceDiscovery{
							Type:     v1beta1.ServiceDiscoveryTypeDns,
							Hostname: "oracle.com",
						},
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{"freeformTag1": "value1"},
							DefinedTags: map[string]api.MapValue{
								"definedTag1": {"foo": "bar"},
							},
						},
					},
					Status: v1beta1.ServiceMeshStatus{
						VirtualDeploymentId: "vdId",
					},
				},
				sdkObj: &sdk.VirtualDeployment{},
			},
			wantSDKObj: &sdk.VirtualDeployment{
				Id:               conversions.String("vdId"),
				Name:             conversions.String("my-namespace-id/" + vdName),
				CompartmentId:    conversions.String("my-compartment"),
				VirtualServiceId: conversions.String("my-service-id"),
				Listeners: []sdk.VirtualDeploymentListener{
					{
						Protocol: sdk.VirtualDeploymentListenerProtocolHttp,
						Port:     &defaultPort,
					},
				},
				ServiceDiscovery: sdk.DnsServiceDiscoveryConfiguration{
					Hostname: conversions.String("oracle.com"),
				},
				FreeformTags: map[string]string{"freeformTag1": "value1"},
				DefinedTags: map[string]map[string]interface{}{
					"definedTag1": {"foo": "bar"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conversions.ConvertCrdVirtualDeploymentToSdkVirtualDeployment(tt.args.crdObj, tt.args.sdkObj, &vsId)
			assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
		})
	}
}

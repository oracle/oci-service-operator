/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package accesspolicy

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	sdk "github.com/oracle/oci-go-sdk/v65/servicemesh"
	api "github.com/oracle/oci-service-operator/api/v1beta1"
	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/equality"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/namespace"
	"github.com/oracle/oci-service-operator/test/servicemesh/functional"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var (
	testEnvFramework functional.TestEnvFramework
	testEnv          *envtest.Environment
	config           *rest.Config
	ctx              context.Context
)

func beforeEach(t *testing.T) *functional.Framework {
	ctx = context.Background()
	testEnvFramework = functional.NewDefaultTestEnvFramework()
	testEnv, config = testEnvFramework.SetupTestEnv()
	framework := testEnvFramework.SetupTestFramework(t, config)
	framework.CreateNamespace(ctx, "test-namespace")
	return framework
}

func afterEach(f *functional.Framework) {
	testEnvFramework.CleanUpTestFramework(f)
	testEnvFramework.CleanUpTestEnv(testEnv)
}

func TestAccessPolicy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Service Mesh Functional Tests during build.")
	}
	type args struct {
		compartmentId  string
		freeformTags   map[string]string
		definedTagsSdk map[string]map[string]interface{}
		definedTags    map[string]api.MapValue
	}
	tests := []struct {
		name           string
		args           args
		expectedStatus *servicemeshapi.ServiceMeshStatus
		expectedErr    error
	}{
		{
			name: "create accessPolicy",
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId:         api.OCID("my-mesh-id"),
				AccessPolicyId: api.OCID("my-accesspolicy-id"),
				RefIdForRules: []map[string]api.OCID{
					{
						"destination": "my-virtualservice-id",
						"source":      "my-virtualservice-id",
					},
				},
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshDependenciesActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            string(commons.DependenciesResolved),
							ObservedGeneration: 1,
						},
					},
					{
						Type: servicemeshapi.ServiceMeshConfigured,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            string(commons.ResourceConfigured),
							ObservedGeneration: 1,
						},
					},
					{
						Type: servicemeshapi.ServiceMeshActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            string(commons.ResourceActive),
							ObservedGeneration: 1,
						},
					},
				},
			},
		},
		{
			name: "change accessPolicy compartmentId",
			args: args{
				compartmentId: "compartment-id-2",
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId:         api.OCID("my-mesh-id"),
				AccessPolicyId: api.OCID("my-accesspolicy-id"),
				RefIdForRules: []map[string]api.OCID{
					{
						"destination": "my-virtualservice-id",
						"source":      "my-virtualservice-id",
					},
				},
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshDependenciesActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            string(commons.DependenciesResolved),
							ObservedGeneration: 2,
						},
					},
					{
						Type: servicemeshapi.ServiceMeshConfigured,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            string(commons.ResourceConfigured),
							ObservedGeneration: 2,
						},
					},
					{
						Type: servicemeshapi.ServiceMeshActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            string(commons.ResourceActive),
							ObservedGeneration: 2,
						},
					},
				},
			},
		},
		{
			name: "update accessPolicy freeformTags",
			args: args{
				freeformTags: map[string]string{"freeformTag2": "value2"},
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId:         api.OCID("my-mesh-id"),
				AccessPolicyId: api.OCID("my-accesspolicy-id"),
				RefIdForRules: []map[string]api.OCID{
					{
						"destination": "my-virtualservice-id",
						"source":      "my-virtualservice-id",
					},
				},
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshDependenciesActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            string(commons.DependenciesResolved),
							ObservedGeneration: 2,
						},
					},
					{
						Type: servicemeshapi.ServiceMeshConfigured,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            string(commons.ResourceConfigured),
							ObservedGeneration: 2,
						},
					},
					{
						Type: servicemeshapi.ServiceMeshActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            string(commons.ResourceActive),
							ObservedGeneration: 2,
						},
					},
				},
			},
		},
		{
			name: "update accessPolicy definedTags",
			args: args{
				definedTagsSdk: map[string]map[string]interface{}{
					"definedTag2": {"foo2": "bar2"},
				},
				definedTags: map[string]api.MapValue{
					"definedTag2": {"foo2": "bar2"},
				},
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId:         api.OCID("my-mesh-id"),
				AccessPolicyId: api.OCID("my-accesspolicy-id"),
				RefIdForRules: []map[string]api.OCID{
					{
						"destination": "my-virtualservice-id",
						"source":      "my-virtualservice-id",
					},
				},
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshDependenciesActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            string(commons.DependenciesResolved),
							ObservedGeneration: 2,
						},
					},
					{
						Type: servicemeshapi.ServiceMeshConfigured,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            string(commons.ResourceConfigured),
							ObservedGeneration: 2,
						},
					},
					{
						Type: servicemeshapi.ServiceMeshActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            string(commons.ResourceActive),
							ObservedGeneration: 2,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		framework := beforeEach(t)
		time.Sleep(2 * time.Second)
		t.Run(tt.name, func(t *testing.T) {
			// Create the accessPolicy
			mesh := functional.GetApiMesh()
			framework.MeshClient.EXPECT().CreateMesh(gomock.Any(), gomock.Any(), gomock.Any()).Return(functional.GetSdkMesh(sdk.MeshLifecycleStateActive), nil).AnyTimes()
			framework.MeshClient.EXPECT().DeleteMesh(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			assert.NoError(t, framework.K8sAPIs.Create(ctx, mesh))

			// Create the virtualService
			virtualService := functional.GetApiVirtualService()
			framework.MeshClient.EXPECT().CreateVirtualService(gomock.Any(), gomock.Any(), gomock.Any()).Return(functional.GetSdkVirtualService(sdk.VirtualServiceLifecycleStateActive), nil).AnyTimes()
			framework.MeshClient.EXPECT().DeleteVirtualService(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			assert.NoError(t, framework.K8sAPIs.Create(ctx, virtualService))

			// Create the accessPolicy
			accessPolicy := functional.GetApiAccessPolicy()
			framework.MeshClient.EXPECT().CreateAccessPolicy(gomock.Any(), gomock.Any(), gomock.Any()).Return(functional.GetSdkAccessPolicy(sdk.AccessPolicyLifecycleStateActive), nil).AnyTimes()
			framework.MeshClient.EXPECT().DeleteAccessPolicy(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			assert.NoError(t, framework.K8sAPIs.Create(ctx, accessPolicy))

			// Change compartmentId
			if len(tt.args.compartmentId) > 0 {
				sdkAccessPolicy0 := functional.GetSdkAccessPolicy(sdk.AccessPolicyLifecycleStateActive)
				sdkAccessPolicy1 := functional.GetSdkAccessPolicy(sdk.AccessPolicyLifecycleStateUpdating)
				sdkAccessPolicy1.CompartmentId = &tt.args.compartmentId
				sdkAccessPolicy2 := functional.GetSdkAccessPolicy(sdk.AccessPolicyLifecycleStateActive)
				sdkAccessPolicy2.CompartmentId = &tt.args.compartmentId
				sdkAccessPolicy3 := functional.GetSdkAccessPolicy(sdk.AccessPolicyLifecycleStateUpdating)
				sdkAccessPolicy3.CompartmentId = &tt.args.compartmentId
				sdkAccessPolicy4 := functional.GetSdkAccessPolicy(sdk.AccessPolicyLifecycleStateActive)
				sdkAccessPolicy4.CompartmentId = &tt.args.compartmentId
				gomock.InOrder(
					framework.MeshClient.EXPECT().GetAccessPolicy(gomock.Any(), gomock.Any()).Return(sdkAccessPolicy0, nil).Times(1),
					framework.MeshClient.EXPECT().GetAccessPolicy(gomock.Any(), gomock.Any()).Return(sdkAccessPolicy1, nil).Times(1),
					framework.MeshClient.EXPECT().GetAccessPolicy(gomock.Any(), gomock.Any()).Return(sdkAccessPolicy2, nil).Times(1),
					framework.MeshClient.EXPECT().GetAccessPolicy(gomock.Any(), gomock.Any()).Return(sdkAccessPolicy3, nil).Times(1),
					framework.MeshClient.EXPECT().GetAccessPolicy(gomock.Any(), gomock.Any()).Return(sdkAccessPolicy4, nil).Times(1),
				)
				framework.MeshClient.EXPECT().ChangeAccessPolicyCompartment(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				framework.MeshClient.EXPECT().UpdateAccessPolicy(gomock.Any(), gomock.Any()).Return(nil)
				oldAccessPolicy := accessPolicy.DeepCopy()
				accessPolicy.Spec.CompartmentId = api.OCID(tt.args.compartmentId)
				err := framework.K8sAPIs.Update(ctx, accessPolicy, oldAccessPolicy)
				assert.NoError(t, err)
				assert.NoError(t, waitUntilSettled(ctx, framework, accessPolicy))
			}

			// Update freeformTags
			if tt.args.freeformTags != nil {
				sdkAccessPolicy0 := functional.GetSdkAccessPolicy(sdk.AccessPolicyLifecycleStateActive)
				sdkAccessPolicy1 := functional.GetSdkAccessPolicy(sdk.AccessPolicyLifecycleStateUpdating)
				sdkAccessPolicy1.FreeformTags = tt.args.freeformTags
				sdkAccessPolicy2 := functional.GetSdkAccessPolicy(sdk.AccessPolicyLifecycleStateActive)
				sdkAccessPolicy2.FreeformTags = tt.args.freeformTags
				gomock.InOrder(
					framework.MeshClient.EXPECT().GetAccessPolicy(gomock.Any(), gomock.Any()).Return(sdkAccessPolicy0, nil).Times(1),
					framework.MeshClient.EXPECT().GetAccessPolicy(gomock.Any(), gomock.Any()).Return(sdkAccessPolicy1, nil).Times(1),
					framework.MeshClient.EXPECT().GetAccessPolicy(gomock.Any(), gomock.Any()).Return(sdkAccessPolicy2, nil).Times(1),
				)
				framework.MeshClient.EXPECT().UpdateAccessPolicy(gomock.Any(), gomock.Any()).Return(nil)
				oldAccessPolicy := accessPolicy.DeepCopy()
				accessPolicy.Spec.FreeFormTags = tt.args.freeformTags
				err := framework.K8sAPIs.Update(ctx, accessPolicy, oldAccessPolicy)
				assert.NoError(t, err)
				assert.NoError(t, waitUntilSettled(ctx, framework, accessPolicy))
			}

			// Update definedTags
			if tt.args.definedTags != nil {
				sdkAccessPolicy0 := functional.GetSdkAccessPolicy(sdk.AccessPolicyLifecycleStateActive)
				sdkAccessPolicy1 := functional.GetSdkAccessPolicy(sdk.AccessPolicyLifecycleStateUpdating)
				sdkAccessPolicy1.DefinedTags = tt.args.definedTagsSdk
				sdkAccessPolicy2 := functional.GetSdkAccessPolicy(sdk.AccessPolicyLifecycleStateActive)
				sdkAccessPolicy2.DefinedTags = tt.args.definedTagsSdk
				gomock.InOrder(
					framework.MeshClient.EXPECT().GetAccessPolicy(gomock.Any(), gomock.Any()).Return(sdkAccessPolicy0, nil).Times(1),
					framework.MeshClient.EXPECT().GetAccessPolicy(gomock.Any(), gomock.Any()).Return(sdkAccessPolicy1, nil).Times(1),
					framework.MeshClient.EXPECT().GetAccessPolicy(gomock.Any(), gomock.Any()).Return(sdkAccessPolicy2, nil).Times(1),
				)
				framework.MeshClient.EXPECT().UpdateAccessPolicy(gomock.Any(), gomock.Any()).Return(nil)
				oldAccessPolicy := accessPolicy.DeepCopy()
				accessPolicy.Spec.DefinedTags = tt.args.definedTags
				err := framework.K8sAPIs.Update(ctx, accessPolicy, oldAccessPolicy)
				assert.NoError(t, err)
				assert.NoError(t, waitUntilSettled(ctx, framework, accessPolicy))
			}

			curAccessPolicy := &servicemeshapi.AccessPolicy{}
			opts := equality.IgnoreFakeClientPopulatedFields()
			key := namespace.NewNamespacedName(accessPolicy)

			assert.NoError(t, framework.K8sAPIs.Get(ctx, key, curAccessPolicy))
			if tt.expectedStatus != nil {
				assert.True(t, cmp.Equal(*tt.expectedStatus, curAccessPolicy.Status, opts), "diff", cmp.Diff(*tt.expectedStatus, curAccessPolicy.Status, opts))
			}

			assert.NoError(t, framework.K8sAPIs.Delete(ctx, accessPolicy))

		})
		afterEach(framework)
	}
}

func waitUntilSettled(ctx context.Context, framework *functional.Framework, accessPolicy *servicemeshapi.AccessPolicy) error {
	observedAccessPolicy := &servicemeshapi.AccessPolicy{}
	key := namespace.NewNamespacedName(accessPolicy)
	return wait.PollImmediateUntil(commons.PollInterval, func() (bool, error) {
		if err := framework.K8sClient.Get(ctx, key, observedAccessPolicy); err != nil {
			return false, err
		}
		if observedAccessPolicy != nil && commons.GetServiceMeshCondition(&observedAccessPolicy.Status, servicemeshapi.ServiceMeshActive).Status == metav1.ConditionTrue {
			return true, nil
		}
		return false, nil
	}, ctx.Done())
}

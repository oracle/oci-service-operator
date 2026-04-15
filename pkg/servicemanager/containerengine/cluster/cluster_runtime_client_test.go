package cluster

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	containerenginesdk "github.com/oracle/oci-go-sdk/v65/containerengine"
	containerenginev1beta1 "github.com/oracle/oci-service-operator/api/containerengine/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestBuildClusterCreateDetailsPreservesNestedFalseAndPolymorphicOptions(t *testing.T) {
	t.Parallel()

	resource := &containerenginev1beta1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-sample",
			Namespace: "default",
		},
		Spec: containerenginev1beta1.ClusterSpec{
			Name:              "cluster-sample",
			CompartmentId:     "ocid1.compartment.oc1..example",
			VcnId:             "ocid1.vcn.oc1..example",
			KubernetesVersion: "v1.30.1",
			EndpointConfig: containerenginev1beta1.ClusterEndpointConfig{
				SubnetId:          "ocid1.subnet.oc1..example",
				NsgIds:            []string{"ocid1.nsg.oc1..example"},
				IsPublicIpEnabled: false,
			},
			FreeformTags: map[string]string{
				"team": "oke",
			},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {
					"CostCenter": "42",
				},
			},
			Options: containerenginev1beta1.ClusterOptions{
				ServiceLbSubnetIds: []string{"ocid1.subnet.oc1..service"},
				AddOns: containerenginev1beta1.ClusterOptionsAddOns{
					IsKubernetesDashboardEnabled: true,
					IsTillerEnabled:              false,
				},
			},
			ImagePolicyConfig: containerenginev1beta1.ClusterImagePolicyConfig{
				IsPolicyEnabled: false,
				KeyDetails: []containerenginev1beta1.ClusterImagePolicyConfigKeyDetail{
					{KmsKeyId: "ocid1.key.oc1..example"},
				},
			},
			ClusterPodNetworkOptions: []containerenginev1beta1.ClusterPodNetworkOption{
				{CniType: "OCI_VCN_IP_NATIVE"},
				{CniType: "FLANNEL_OVERLAY"},
			},
		},
	}

	details, err := buildClusterCreateDetails(resource)
	if err != nil {
		t.Fatalf("buildClusterCreateDetails() error = %v", err)
	}

	request := containerenginesdk.CreateClusterRequest{
		CreateClusterDetails: details,
	}
	httpRequest, err := request.HTTPRequest(http.MethodPost, "/clusters", nil, nil)
	if err != nil {
		t.Fatalf("HTTPRequest() error = %v", err)
	}

	body, err := io.ReadAll(httpRequest.Body)
	if err != nil {
		t.Fatalf("ReadAll(request.Body) error = %v", err)
	}
	got := string(body)

	wantSnippets := []string{
		`"isPublicIpEnabled":false`,
		`"isTillerEnabled":false`,
		`"isPolicyEnabled":false`,
		`"cniType":"OCI_VCN_IP_NATIVE"`,
		`"cniType":"FLANNEL_OVERLAY"`,
		`"definedTags":{"Operations":{"CostCenter":"42"}}`,
	}
	for _, want := range wantSnippets {
		if !strings.Contains(got, want) {
			t.Fatalf("request body %s does not contain %s", got, want)
		}
	}

	if strings.Contains(got, `"jsonData"`) {
		t.Fatalf("request body unexpectedly exposes jsonData helper fields: %s", got)
	}
}

func TestBuildClusterCreateDetailsOmitsEmptyBlocks(t *testing.T) {
	t.Parallel()

	resource := &containerenginev1beta1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-sample",
			Namespace: "default",
		},
		Spec: containerenginev1beta1.ClusterSpec{
			Name:              "cluster-sample",
			CompartmentId:     "ocid1.compartment.oc1..example",
			VcnId:             "ocid1.vcn.oc1..example",
			KubernetesVersion: "v1.30.1",
		},
	}

	details, err := buildClusterCreateDetails(resource)
	if err != nil {
		t.Fatalf("buildClusterCreateDetails() error = %v", err)
	}
	if details.EndpointConfig != nil {
		t.Fatalf("EndpointConfig = %#v, want nil", details.EndpointConfig)
	}
	if details.Options != nil {
		t.Fatalf("Options = %#v, want nil", details.Options)
	}
	if details.ImagePolicyConfig != nil {
		t.Fatalf("ImagePolicyConfig = %#v, want nil", details.ImagePolicyConfig)
	}
	if len(details.ClusterPodNetworkOptions) != 0 {
		t.Fatalf("ClusterPodNetworkOptions = %#v, want empty", details.ClusterPodNetworkOptions)
	}

	request := containerenginesdk.CreateClusterRequest{
		CreateClusterDetails: details,
	}
	httpRequest, err := request.HTTPRequest(http.MethodPost, "/clusters", nil, nil)
	if err != nil {
		t.Fatalf("HTTPRequest() error = %v", err)
	}

	body, err := io.ReadAll(httpRequest.Body)
	if err != nil {
		t.Fatalf("ReadAll(request.Body) error = %v", err)
	}
	got := string(body)

	unexpectedSnippets := []string{
		`"endpointConfig"`,
		`"options"`,
		`"imagePolicyConfig"`,
		`"clusterPodNetworkOptions"`,
	}
	for _, unexpected := range unexpectedSnippets {
		if strings.Contains(got, unexpected) {
			t.Fatalf("request body unexpectedly contains %s: %s", unexpected, got)
		}
	}
}

func TestBuildClusterCreateDetailsRejectsUnsupportedPodNetworkOptionType(t *testing.T) {
	t.Parallel()

	resource := &containerenginev1beta1.Cluster{
		Spec: containerenginev1beta1.ClusterSpec{
			Name:              "cluster-sample",
			CompartmentId:     "ocid1.compartment.oc1..example",
			VcnId:             "ocid1.vcn.oc1..example",
			KubernetesVersion: "v1.30.1",
			ClusterPodNetworkOptions: []containerenginev1beta1.ClusterPodNetworkOption{
				{CniType: "UNKNOWN"},
			},
		},
	}

	_, err := buildClusterCreateDetails(resource)
	if err == nil {
		t.Fatal("buildClusterCreateDetails() error = nil, want unsupported cniType failure")
	}
	if !strings.Contains(err.Error(), `unsupported cniType "UNKNOWN"`) {
		t.Fatalf("buildClusterCreateDetails() error = %v, want unsupported cniType context", err)
	}
}

func TestClusterCreateOrUpdateClassifiesObservedLifecycleStates(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		lifecycleState string
		wantSuccessful bool
		wantRequeue    bool
		wantCondition  shared.OSOKConditionType
	}{
		{
			name:           "active",
			lifecycleState: "ACTIVE",
			wantSuccessful: true,
			wantRequeue:    false,
			wantCondition:  shared.Active,
		},
		{
			name:           "creating",
			lifecycleState: "CREATING",
			wantSuccessful: true,
			wantRequeue:    true,
			wantCondition:  shared.Provisioning,
		},
		{
			name:           "updating",
			lifecycleState: "UPDATING",
			wantSuccessful: true,
			wantRequeue:    true,
			wantCondition:  shared.Updating,
		},
		{
			name:           "failed",
			lifecycleState: "FAILED",
			wantSuccessful: false,
			wantRequeue:    false,
			wantCondition:  shared.Failed,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			const existingID = "ocid1.cluster.oc1..existing"

			resource := newExistingClusterTestResource(existingID)
			var getRequest containerenginesdk.GetClusterRequest

			manager := newClusterRuntimeTestManager(generatedruntime.Config[*containerenginev1beta1.Cluster]{
				Get: &generatedruntime.Operation{
					NewRequest: func() any { return &containerenginesdk.GetClusterRequest{} },
					Call: func(_ context.Context, request any) (any, error) {
						getRequest = *request.(*containerenginesdk.GetClusterRequest)
						return containerenginesdk.GetClusterResponse{
							Cluster: observedClusterFromSpec(existingID, resource.Spec, tc.lifecycleState),
						}, nil
					},
					Fields: clusterGetFields(),
				},
			})

			response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			if err != nil {
				t.Fatalf("CreateOrUpdate() error = %v", err)
			}
			if response.IsSuccessful != tc.wantSuccessful {
				t.Fatalf("CreateOrUpdate() IsSuccessful = %t, want %t", response.IsSuccessful, tc.wantSuccessful)
			}
			if response.ShouldRequeue != tc.wantRequeue {
				t.Fatalf("CreateOrUpdate() ShouldRequeue = %t, want %t", response.ShouldRequeue, tc.wantRequeue)
			}
			requireClusterIDPointer(t, "get request clusterId", getRequest.ClusterId, existingID)
			requireClusterCondition(t, resource, tc.wantCondition)
			requireClusterOCID(t, resource, existingID)
			if resource.Status.LifecycleState != tc.lifecycleState {
				t.Fatalf("status.lifecycleState = %q, want %q", resource.Status.LifecycleState, tc.lifecycleState)
			}
		})
	}
}

func TestClusterCreateOrUpdateReusesMatchingListEntry(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.cluster.oc1..existing"

	resource := newClusterTestResource()
	createCalled := false
	var listRequest containerenginesdk.ListClustersRequest
	var getRequest containerenginesdk.GetClusterRequest

	manager := newClusterRuntimeTestManager(generatedruntime.Config[*containerenginev1beta1.Cluster]{
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &containerenginesdk.CreateClusterRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				createCalled = true
				t.Fatal("Create() should not be called when list lookup finds a reusable cluster")
				return nil, nil
			},
			Fields: clusterCreateFields(),
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &containerenginesdk.GetClusterRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getRequest = *request.(*containerenginesdk.GetClusterRequest)
				return containerenginesdk.GetClusterResponse{
					Cluster: observedClusterFromSpec(existingID, resource.Spec, "ACTIVE"),
				}, nil
			},
			Fields: clusterGetFields(),
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &containerenginesdk.ListClustersRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				listRequest = *request.(*containerenginesdk.ListClustersRequest)
				return containerenginesdk.ListClustersResponse{
					Items: []containerenginesdk.ClusterSummary{
						observedClusterSummaryFromSpec(existingID, resource.Spec, "ACTIVE"),
					},
				}, nil
			},
			Fields: clusterListFields(),
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should succeed when list lookup reuses an existing cluster")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue for ACTIVE list reuse")
	}
	if createCalled {
		t.Fatal("Create() should not be called when list lookup finds a reusable cluster")
	}
	requireClusterIDPointer(t, "list request compartmentId", listRequest.CompartmentId, resource.Spec.CompartmentId)
	requireClusterIDPointer(t, "list request name", listRequest.Name, resource.Spec.Name)
	requireClusterIDPointer(t, "get request clusterId", getRequest.ClusterId, existingID)
	requireClusterOCID(t, resource, existingID)
	if resource.Status.Id != existingID {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, existingID)
	}
}

func TestClusterCreateOrUpdateFallsBackFromStaleTrackedIDToListWithoutLifecycleQuery(t *testing.T) {
	t.Parallel()

	const staleID = "ocid1.cluster.oc1..stale"
	const replacementID = "ocid1.cluster.oc1..replacement"

	resource := newExistingClusterTestResource(staleID)
	resource.Status.LifecycleState = "ACTIVE"

	var listRequest containerenginesdk.ListClustersRequest

	manager := newClusterRuntimeTestManager(generatedruntime.Config[*containerenginev1beta1.Cluster]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &containerenginesdk.GetClusterRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getRequest := request.(*containerenginesdk.GetClusterRequest)
				gotClusterID := ""
				if getRequest.ClusterId != nil {
					gotClusterID = *getRequest.ClusterId
				}
				if gotClusterID != staleID {
					t.Fatalf("get request clusterId = %q, want %q", gotClusterID, staleID)
				}
				return nil, errortest.NewServiceError(404, "NotFound", "cluster not found")
			},
			Fields: clusterGetFields(),
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &containerenginesdk.ListClustersRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				listRequest = *request.(*containerenginesdk.ListClustersRequest)
				return containerenginesdk.ListClustersResponse{
					Items: []containerenginesdk.ClusterSummary{
						observedClusterSummaryFromSpec(replacementID, resource.Spec, "ACTIVE"),
					},
				}, nil
			},
			Fields: clusterListFields(),
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should succeed when the list fallback rebinds a replacement cluster")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue when fallback list reuse observes ACTIVE")
	}
	if len(listRequest.LifecycleState) != 0 {
		t.Fatalf("list request lifecycleState = %#v, want empty after stale tracked ID fallback", listRequest.LifecycleState)
	}
	requireClusterIDPointer(t, "list request compartmentId", listRequest.CompartmentId, resource.Spec.CompartmentId)
	requireClusterIDPointer(t, "list request name", listRequest.Name, resource.Spec.Name)
	requireClusterOCID(t, resource, replacementID)
	if resource.Status.Id != replacementID {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, replacementID)
	}
}

func TestClusterCreateOrUpdateDoesNotReuseMismatchedListEntries(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		summary containerenginesdk.ClusterSummary
	}{
		{
			name: "name mismatch",
			summary: containerenginesdk.ClusterSummary{
				Id:             common.String("ocid1.cluster.oc1..wrong-name"),
				Name:           common.String("other-cluster"),
				CompartmentId:  common.String("ocid1.compartment.oc1..example"),
				LifecycleState: containerenginesdk.ClusterLifecycleStateActive,
			},
		},
		{
			name: "compartment mismatch",
			summary: containerenginesdk.ClusterSummary{
				Id:             common.String("ocid1.cluster.oc1..wrong-compartment"),
				Name:           common.String("cluster-sample"),
				CompartmentId:  common.String("ocid1.compartment.oc1..other"),
				LifecycleState: containerenginesdk.ClusterLifecycleStateActive,
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			const createdID = "ocid1.cluster.oc1..created"

			resource := newClusterTestResource()
			createCalled := false
			listCalls := 0
			listRequests := make([]containerenginesdk.ListClustersRequest, 0, 2)
			var createRequest containerenginesdk.CreateClusterRequest

			manager := newClusterRuntimeTestManager(generatedruntime.Config[*containerenginev1beta1.Cluster]{
				Create: &generatedruntime.Operation{
					NewRequest: func() any { return &containerenginesdk.CreateClusterRequest{} },
					Call: func(_ context.Context, request any) (any, error) {
						createCalled = true
						createRequest = *request.(*containerenginesdk.CreateClusterRequest)
						return containerenginesdk.CreateClusterResponse{}, nil
					},
					Fields: clusterCreateFields(),
				},
				List: &generatedruntime.Operation{
					NewRequest: func() any { return &containerenginesdk.ListClustersRequest{} },
					Call: func(_ context.Context, request any) (any, error) {
						listRequests = append(listRequests, *request.(*containerenginesdk.ListClustersRequest))
						listCalls++
						if listCalls == 1 {
							return containerenginesdk.ListClustersResponse{
								Items: []containerenginesdk.ClusterSummary{tc.summary},
							}, nil
						}
						return containerenginesdk.ListClustersResponse{
							Items: []containerenginesdk.ClusterSummary{
								observedClusterSummaryFromSpec(createdID, resource.Spec, "ACTIVE"),
							},
						}, nil
					},
					Fields: clusterListFields(),
				},
			})

			response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			if err != nil {
				t.Fatalf("CreateOrUpdate() error = %v", err)
			}
			if !response.IsSuccessful {
				t.Fatal("CreateOrUpdate() should succeed after creating a replacement for the mismatched list entry")
			}
			if response.ShouldRequeue {
				t.Fatal("CreateOrUpdate() should not requeue when the follow-up list sees ACTIVE state")
			}
			if !createCalled {
				t.Fatal("Create() should be called when the list response does not match the cluster identity keys")
			}
			if len(listRequests) != 2 {
				t.Fatalf("list calls = %d, want 2", len(listRequests))
			}
			for index, request := range listRequests {
				requireClusterIDPointer(t, "list request compartmentId", request.CompartmentId, resource.Spec.CompartmentId)
				requireClusterIDPointer(t, "list request name", request.Name, resource.Spec.Name)
				if index > 1 {
					t.Fatalf("unexpected list request index %d", index)
				}
			}
			requireClusterIDPointer(t, "create request name", createRequest.Name, resource.Spec.Name)
			requireClusterIDPointer(t, "create request compartmentId", createRequest.CompartmentId, resource.Spec.CompartmentId)
			requireClusterIDPointer(t, "create request vcnId", createRequest.VcnId, resource.Spec.VcnId)
			requireClusterOCID(t, resource, createdID)
			if resource.Status.Id != createdID {
				t.Fatalf("status.id = %q, want %q", resource.Status.Id, createdID)
			}
		})
	}
}

func TestClusterCreateOrUpdateUpdatesMutableFields(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.cluster.oc1..existing"

	resource := newExistingClusterTestResource(existingID)
	resource.Spec.Name = "renamed-cluster"

	getCalls := 0
	var updateRequest containerenginesdk.UpdateClusterRequest

	manager := newClusterRuntimeTestManager(generatedruntime.Config[*containerenginev1beta1.Cluster]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &containerenginesdk.GetClusterRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getCalls++
				requireClusterIDPointer(t, "get request clusterId", request.(*containerenginesdk.GetClusterRequest).ClusterId, existingID)

				live := observedClusterFromSpec(existingID, resource.Spec, "ACTIVE")
				if getCalls == 1 {
					live.Name = common.String("current-cluster")
				}
				return containerenginesdk.GetClusterResponse{Cluster: live}, nil
			},
			Fields: clusterGetFields(),
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &containerenginesdk.UpdateClusterRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				updateRequest = *request.(*containerenginesdk.UpdateClusterRequest)
				return containerenginesdk.UpdateClusterResponse{}, nil
			},
			Fields: clusterUpdateFields(),
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should succeed when a mutable cluster field changes")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue when the update follow-up sees ACTIVE state")
	}
	if getCalls != 2 {
		t.Fatalf("get calls = %d, want 2", getCalls)
	}
	requireClusterIDPointer(t, "update request clusterId", updateRequest.ClusterId, existingID)
	requireClusterIDPointer(t, "update request name", updateRequest.Name, resource.Spec.Name)
	requireClusterOCID(t, resource, existingID)
	if resource.Status.Name != resource.Spec.Name {
		t.Fatalf("status.name = %q, want %q", resource.Status.Name, resource.Spec.Name)
	}
	requireClusterCondition(t, resource, shared.Active)
}

func TestClusterCreateOrUpdateProjectsMutableFalseBoolUpdate(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.cluster.oc1..existing"
	const kmsKeyID = "ocid1.key.oc1..cluster"

	resource := newExistingClusterTestResource(existingID)
	resource.Spec.ImagePolicyConfig = containerenginev1beta1.ClusterImagePolicyConfig{
		IsPolicyEnabled: false,
		KeyDetails: []containerenginev1beta1.ClusterImagePolicyConfigKeyDetail{
			{KmsKeyId: kmsKeyID},
		},
	}

	getCalls := 0
	updateCalled := false
	var updateRequest containerenginesdk.UpdateClusterRequest

	manager := newClusterRuntimeTestManager(generatedruntime.Config[*containerenginev1beta1.Cluster]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &containerenginesdk.GetClusterRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getCalls++
				requireClusterIDPointer(t, "get request clusterId", request.(*containerenginesdk.GetClusterRequest).ClusterId, existingID)

				live := observedClusterFromSpec(existingID, resource.Spec, "ACTIVE")
				live.ImagePolicyConfig = &containerenginesdk.ImagePolicyConfig{
					IsPolicyEnabled: common.Bool(getCalls == 1),
					KeyDetails: []containerenginesdk.KeyDetails{
						{KmsKeyId: common.String(kmsKeyID)},
					},
				}
				return containerenginesdk.GetClusterResponse{Cluster: live}, nil
			},
			Fields: clusterGetFields(),
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &containerenginesdk.UpdateClusterRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				updateCalled = true
				updateRequest = *request.(*containerenginesdk.UpdateClusterRequest)
				return containerenginesdk.UpdateClusterResponse{}, nil
			},
			Fields: clusterUpdateFields(),
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should succeed when image policy is toggled from true to false")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue when the update follow-up sees ACTIVE state")
	}
	if !updateCalled {
		t.Fatal("Update() should be called when image policy is toggled from true to false")
	}
	requireClusterIDPointer(t, "update request clusterId", updateRequest.ClusterId, existingID)
	if updateRequest.ImagePolicyConfig == nil {
		t.Fatal("update request imagePolicyConfig = nil, want nested mutable bool projected")
	}
	if updateRequest.ImagePolicyConfig.IsPolicyEnabled == nil || *updateRequest.ImagePolicyConfig.IsPolicyEnabled {
		t.Fatalf("update request imagePolicyConfig.isPolicyEnabled = %v, want false", updateRequest.ImagePolicyConfig.IsPolicyEnabled)
	}
	if len(updateRequest.ImagePolicyConfig.KeyDetails) != 1 {
		t.Fatalf("update request imagePolicyConfig.keyDetails = %#v, want single key detail", updateRequest.ImagePolicyConfig.KeyDetails)
	}
	requireClusterIDPointer(t, "update request imagePolicyConfig.keyDetails[0].kmsKeyId", updateRequest.ImagePolicyConfig.KeyDetails[0].KmsKeyId, kmsKeyID)
	requireClusterCondition(t, resource, shared.Active)
}

func TestClusterCreateOrUpdateRejectsImmutableDrift(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		mutateSpec    func(*containerenginev1beta1.Cluster)
		mutateLive    func(*containerenginesdk.Cluster)
		wantErrorPart string
	}{
		{
			name: "compartment id drift",
			mutateSpec: func(resource *containerenginev1beta1.Cluster) {
				resource.Spec.CompartmentId = "ocid1.compartment.oc1..desired"
			},
			mutateLive: func(live *containerenginesdk.Cluster) {
				live.CompartmentId = common.String("ocid1.compartment.oc1..live")
			},
			wantErrorPart: "Cluster formal semantics require replacement when compartmentId changes",
		},
		{
			name: "vcn id drift",
			mutateSpec: func(resource *containerenginev1beta1.Cluster) {
				resource.Spec.VcnId = "ocid1.vcn.oc1..desired"
			},
			mutateLive: func(live *containerenginesdk.Cluster) {
				live.VcnId = common.String("ocid1.vcn.oc1..live")
			},
			wantErrorPart: "Cluster formal semantics require replacement when vcnId changes",
		},
		{
			name: "endpoint public ip true to false drift",
			mutateSpec: func(resource *containerenginev1beta1.Cluster) {
				resource.Spec.EndpointConfig = containerenginev1beta1.ClusterEndpointConfig{
					SubnetId:          "ocid1.subnet.oc1..desired",
					IsPublicIpEnabled: false,
				}
			},
			mutateLive: func(live *containerenginesdk.Cluster) {
				live.EndpointConfig = &containerenginesdk.ClusterEndpointConfig{
					SubnetId:          common.String("ocid1.subnet.oc1..desired"),
					IsPublicIpEnabled: common.Bool(true),
				}
			},
			wantErrorPart: "Cluster formal semantics require replacement when endpointConfig.isPublicIpEnabled changes",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			const existingID = "ocid1.cluster.oc1..existing"

			resource := newExistingClusterTestResource(existingID)
			tc.mutateSpec(resource)

			updateCalled := false
			manager := newClusterRuntimeTestManager(generatedruntime.Config[*containerenginev1beta1.Cluster]{
				Get: &generatedruntime.Operation{
					NewRequest: func() any { return &containerenginesdk.GetClusterRequest{} },
					Call: func(_ context.Context, request any) (any, error) {
						requireClusterIDPointer(t, "get request clusterId", request.(*containerenginesdk.GetClusterRequest).ClusterId, existingID)
						live := observedClusterFromSpec(existingID, newClusterTestResource().Spec, "ACTIVE")
						tc.mutateLive(&live)
						return containerenginesdk.GetClusterResponse{Cluster: live}, nil
					},
					Fields: clusterGetFields(),
				},
				Update: &generatedruntime.Operation{
					NewRequest: func() any { return &containerenginesdk.UpdateClusterRequest{} },
					Call: func(_ context.Context, _ any) (any, error) {
						updateCalled = true
						t.Fatal("Update() should not be called when immutable cluster fields drift")
						return nil, nil
					},
					Fields: clusterUpdateFields(),
				},
			})

			response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			if err == nil {
				t.Fatal("CreateOrUpdate() error = nil, want immutable-drift rejection")
			}
			if !strings.Contains(err.Error(), tc.wantErrorPart) {
				t.Fatalf("CreateOrUpdate() error = %v, want %q", err, tc.wantErrorPart)
			}
			if response.IsSuccessful {
				t.Fatal("CreateOrUpdate() should report failure when immutable cluster fields drift")
			}
			if updateCalled {
				t.Fatal("Update() should not be called when immutable cluster fields drift")
			}
			requireClusterCondition(t, resource, shared.Failed)
		})
	}
}

func TestClusterDeleteConfirmsDeleteOutcomes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name             string
		confirmedState   string
		wantDeleted      bool
		wantMessage      string
		wantDeletedStamp bool
	}{
		{
			name:             "deleting remains pending",
			confirmedState:   "DELETING",
			wantDeleted:      false,
			wantMessage:      "OCI resource delete is in progress",
			wantDeletedStamp: false,
		},
		{
			name:             "deleted confirms removal",
			confirmedState:   "DELETED",
			wantDeleted:      true,
			wantMessage:      "OCI resource deleted",
			wantDeletedStamp: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			const existingID = "ocid1.cluster.oc1..existing"

			resource := newExistingClusterTestResource(existingID)
			getCalls := 0
			var deleteRequest containerenginesdk.DeleteClusterRequest

			manager := newClusterRuntimeTestManager(generatedruntime.Config[*containerenginev1beta1.Cluster]{
				Get: &generatedruntime.Operation{
					NewRequest: func() any { return &containerenginesdk.GetClusterRequest{} },
					Call: func(_ context.Context, request any) (any, error) {
						getCalls++
						requireClusterIDPointer(t, "get request clusterId", request.(*containerenginesdk.GetClusterRequest).ClusterId, existingID)
						state := "ACTIVE"
						if getCalls > 1 {
							state = tc.confirmedState
						}
						return containerenginesdk.GetClusterResponse{
							Cluster: observedClusterFromSpec(existingID, resource.Spec, state),
						}, nil
					},
					Fields: clusterGetFields(),
				},
				Delete: &generatedruntime.Operation{
					NewRequest: func() any { return &containerenginesdk.DeleteClusterRequest{} },
					Call: func(_ context.Context, request any) (any, error) {
						deleteRequest = *request.(*containerenginesdk.DeleteClusterRequest)
						return containerenginesdk.DeleteClusterResponse{}, nil
					},
					Fields: clusterDeleteFields(),
				},
			})

			deleted, err := manager.Delete(context.Background(), resource)
			if err != nil {
				t.Fatalf("Delete() error = %v", err)
			}
			if deleted != tc.wantDeleted {
				t.Fatalf("Delete() deleted = %t, want %t", deleted, tc.wantDeleted)
			}
			if getCalls != 2 {
				t.Fatalf("get calls = %d, want 2", getCalls)
			}
			requireClusterIDPointer(t, "delete request clusterId", deleteRequest.ClusterId, existingID)
			requireClusterCondition(t, resource, shared.Terminating)
			if resource.Status.LifecycleState != tc.confirmedState {
				t.Fatalf("status.lifecycleState = %q, want %q", resource.Status.LifecycleState, tc.confirmedState)
			}
			if resource.Status.OsokStatus.Message != tc.wantMessage {
				t.Fatalf("status.message = %q, want %q", resource.Status.OsokStatus.Message, tc.wantMessage)
			}
			if tc.wantDeletedStamp {
				if resource.Status.OsokStatus.DeletedAt == nil {
					t.Fatal("status.deletedAt = nil, want delete confirmation timestamp")
				}
			} else if resource.Status.OsokStatus.DeletedAt != nil {
				t.Fatalf("status.deletedAt = %v, want nil while delete is still pending", resource.Status.OsokStatus.DeletedAt)
			}
		})
	}
}

func newClusterRuntimeTestManager(cfg generatedruntime.Config[*containerenginev1beta1.Cluster]) *ClusterServiceManager {
	if cfg.Kind == "" {
		cfg.Kind = "Cluster"
	}
	if cfg.SDKName == "" {
		cfg.SDKName = "Cluster"
	}
	if cfg.Semantics == nil {
		cfg.Semantics = testClusterRuntimeSemantics()
	}
	if cfg.BuildCreateBody == nil {
		cfg.BuildCreateBody = func(_ context.Context, resource *containerenginev1beta1.Cluster, _ string) (any, error) {
			return buildClusterCreateDetails(resource)
		}
	}
	if cfg.BuildUpdateBody == nil {
		cfg.BuildUpdateBody = func(
			ctx context.Context,
			resource *containerenginev1beta1.Cluster,
			namespace string,
			currentResponse any,
		) (any, bool, error) {
			return buildClusterUpdateBody(ctx, nil, resource, namespace, currentResponse)
		}
	}

	return &ClusterServiceManager{
		client: defaultClusterServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*containerenginev1beta1.Cluster](cfg),
		},
	}
}

func testClusterRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{"CREATING"},
			UpdatingStates:     []string{"UPDATING"},
			ActiveStates:       []string{"ACTIVE"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{"DELETING"},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "lifecycleState", "name"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"definedTags",
				"freeformTags",
				"imagePolicyConfig.isPolicyEnabled",
				"imagePolicyConfig.keyDetails.kmsKeyId",
				"kubernetesVersion",
				"name",
				"options.admissionControllerOptions.isPodSecurityPolicyEnabled",
				"options.persistentVolumeConfig.definedTags",
				"options.persistentVolumeConfig.freeformTags",
				"options.serviceLbConfig.definedTags",
				"options.serviceLbConfig.freeformTags",
				"type",
			},
			ForceNew: []string{
				"clusterPodNetworkOptions.cniType",
				"clusterPodNetworkOptions.jsonData",
				"compartmentId",
				"endpointConfig.isPublicIpEnabled",
				"endpointConfig.nsgIds",
				"endpointConfig.subnetId",
				"kmsKeyId",
				"options.addOns.isKubernetesDashboardEnabled",
				"options.addOns.isTillerEnabled",
				"options.kubernetesNetworkConfig.podsCidr",
				"options.kubernetesNetworkConfig.servicesCidr",
				"options.serviceLbSubnetIds",
				"vcnId",
			},
			ConflictsWith: map[string][]string{},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
		},
	}
}

func newClusterTestResource() *containerenginev1beta1.Cluster {
	return &containerenginev1beta1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-sample",
			Namespace: "default",
		},
		Spec: containerenginev1beta1.ClusterSpec{
			Name:              "cluster-sample",
			CompartmentId:     "ocid1.compartment.oc1..example",
			VcnId:             "ocid1.vcn.oc1..example",
			KubernetesVersion: "v1.30.1",
		},
	}
}

func newExistingClusterTestResource(existingID string) *containerenginev1beta1.Cluster {
	resource := newClusterTestResource()
	resource.Status = containerenginev1beta1.ClusterStatus{
		OsokStatus: shared.OSOKStatus{
			Ocid: shared.OCID(existingID),
		},
		Id: existingID,
	}
	return resource
}

func observedClusterFromSpec(
	id string,
	spec containerenginev1beta1.ClusterSpec,
	lifecycleState string,
) containerenginesdk.Cluster {
	lifecycleDetails := "cluster " + strings.ToLower(lifecycleState) + " details"
	return containerenginesdk.Cluster{
		Id:                common.String(id),
		Name:              common.String(spec.Name),
		CompartmentId:     common.String(spec.CompartmentId),
		VcnId:             common.String(spec.VcnId),
		KubernetesVersion: common.String(spec.KubernetesVersion),
		LifecycleState:    containerenginesdk.ClusterLifecycleStateEnum(lifecycleState),
		LifecycleDetails:  common.String(lifecycleDetails),
	}
}

func observedClusterSummaryFromSpec(
	id string,
	spec containerenginev1beta1.ClusterSpec,
	lifecycleState string,
) containerenginesdk.ClusterSummary {
	lifecycleDetails := "cluster " + strings.ToLower(lifecycleState) + " details"
	return containerenginesdk.ClusterSummary{
		Id:                common.String(id),
		Name:              common.String(spec.Name),
		CompartmentId:     common.String(spec.CompartmentId),
		VcnId:             common.String(spec.VcnId),
		KubernetesVersion: common.String(spec.KubernetesVersion),
		LifecycleState:    containerenginesdk.ClusterLifecycleStateEnum(lifecycleState),
		LifecycleDetails:  common.String(lifecycleDetails),
	}
}

func clusterCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateClusterDetails", RequestName: "CreateClusterDetails", Contribution: "body"},
	}
}

func clusterGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ClusterId", RequestName: "clusterId", Contribution: "path", PreferResourceID: true},
	}
}

func clusterListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
		{FieldName: "Name", RequestName: "name", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func clusterUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ClusterId", RequestName: "clusterId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateClusterDetails", RequestName: "UpdateClusterDetails", Contribution: "body"},
	}
}

func clusterDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ClusterId", RequestName: "clusterId", Contribution: "path", PreferResourceID: true},
	}
}

func requireClusterCondition(t *testing.T, resource *containerenginev1beta1.Cluster, want shared.OSOKConditionType) {
	t.Helper()

	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 || conditions[len(conditions)-1].Type != want {
		t.Fatalf("status.conditions = %#v, want trailing %s condition", conditions, want)
	}
}

func requireClusterOCID(t *testing.T, resource *containerenginev1beta1.Cluster, want string) {
	t.Helper()

	if got := string(resource.Status.OsokStatus.Ocid); got != want {
		t.Fatalf("status.ocid = %q, want %q", got, want)
	}
}

func requireClusterIDPointer(t *testing.T, label string, got *string, want string) {
	t.Helper()

	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %q", label, got, want)
	}
}

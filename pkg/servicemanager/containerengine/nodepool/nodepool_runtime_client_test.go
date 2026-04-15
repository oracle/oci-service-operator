package nodepool

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	containerenginesdk "github.com/oracle/oci-go-sdk/v65/containerengine"
	containerenginev1beta1 "github.com/oracle/oci-service-operator/api/containerengine/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestBuildNodePoolCreateDetailsPreservesPolymorphicDetailsAndFalseBooleans(t *testing.T) {
	t.Parallel()

	resource := newNodePoolTestResource()

	details, err := buildNodePoolCreateDetails(context.Background(), resource, resource.Namespace)
	if err != nil {
		t.Fatalf("buildNodePoolCreateDetails() error = %v", err)
	}

	request := containerenginesdk.CreateNodePoolRequest{
		CreateNodePoolDetails: details,
	}
	got := nodePoolSerializedRequestBody(t, request, http.MethodPost, "/nodePools")

	wantSnippets := []string{
		`"sourceType":"IMAGE"`,
		`"imageId":"ocid1.image.oc1..nodepool"`,
		`"cniType":"OCI_VCN_IP_NATIVE"`,
		`"type":"TERMINATE"`,
		`"isPreserveBootVolume":false`,
		`"isPvEncryptionInTransitEnabled":false`,
		`"isForceDeleteAfterGraceDuration":false`,
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

func TestBuildNodePoolCreateDetailsOmitsDeprecatedPlacementFieldsWhenNodeConfigDetailsPresent(t *testing.T) {
	t.Parallel()

	resource := newNodePoolTestResource()

	details, err := buildNodePoolCreateDetails(context.Background(), resource, resource.Namespace)
	if err != nil {
		t.Fatalf("buildNodePoolCreateDetails() error = %v", err)
	}
	if details.SubnetIds != nil {
		t.Fatalf("buildNodePoolCreateDetails() SubnetIds = %#v, want nil when nodeConfigDetails is set", details.SubnetIds)
	}
	if details.QuantityPerSubnet != nil {
		t.Fatalf("buildNodePoolCreateDetails() QuantityPerSubnet = %#v, want nil when nodeConfigDetails is set", details.QuantityPerSubnet)
	}

	body := nodePoolSerializedRequestBody(t, containerenginesdk.CreateNodePoolRequest{
		CreateNodePoolDetails: details,
	}, http.MethodPost, "/nodePools")

	if !strings.Contains(body, `"nodeConfigDetails"`) {
		t.Fatalf("request body %s does not contain nodeConfigDetails", body)
	}
	for _, field := range []string{`"subnetIds"`, `"quantityPerSubnet"`} {
		if strings.Contains(body, field) {
			t.Fatalf("request body unexpectedly serialized %s: %s", field, body)
		}
	}
}

func TestSanitizeCreateNodePoolRequestClearsSubnetIDsWhenNodeConfigDetailsPresent(t *testing.T) {
	t.Parallel()

	req := &containerenginesdk.CreateNodePoolRequest{
		CreateNodePoolDetails: containerenginesdk.CreateNodePoolDetails{
			SubnetIds:         []string{},
			QuantityPerSubnet: common.Int(1),
			NodeConfigDetails: &containerenginesdk.CreateNodePoolNodeConfigDetails{
				Size: common.Int(1),
				PlacementConfigs: []containerenginesdk.NodePoolPlacementConfigDetails{
					{
						AvailabilityDomain: common.String("PHX-AD-1"),
						SubnetId:           common.String("ocid1.subnet.oc1..worker"),
					},
				},
			},
		},
	}

	sanitizeCreateNodePoolRequest(req)

	if req.CreateNodePoolDetails.SubnetIds != nil {
		t.Fatalf("sanitizeCreateNodePoolRequest() SubnetIds = %#v, want nil", req.CreateNodePoolDetails.SubnetIds)
	}
	if req.CreateNodePoolDetails.QuantityPerSubnet == nil || *req.CreateNodePoolDetails.QuantityPerSubnet != 1 {
		t.Fatalf("sanitizeCreateNodePoolRequest() QuantityPerSubnet = %#v, want preserved 1", req.CreateNodePoolDetails.QuantityPerSubnet)
	}
}

func TestBuildNodePoolCreateDetailsPreservesLegacyPlacementFieldsWithoutNodeConfigDetails(t *testing.T) {
	t.Parallel()

	resource := newNodePoolTestResource()
	resource.Spec.NodeConfigDetails = containerenginev1beta1.NodePoolNodeConfigDetails{}
	resource.Spec.SubnetIds = []string{"ocid1.subnet.oc1..legacy"}
	resource.Spec.QuantityPerSubnet = 2

	details, err := buildNodePoolCreateDetails(context.Background(), resource, resource.Namespace)
	if err != nil {
		t.Fatalf("buildNodePoolCreateDetails() error = %v", err)
	}
	if details.NodeConfigDetails != nil {
		t.Fatalf("buildNodePoolCreateDetails() NodeConfigDetails = %#v, want nil for legacy subnetIds path", details.NodeConfigDetails)
	}
	if len(details.SubnetIds) != 1 || details.SubnetIds[0] != "ocid1.subnet.oc1..legacy" {
		t.Fatalf("buildNodePoolCreateDetails() SubnetIds = %#v, want legacy subnet preserved", details.SubnetIds)
	}
	if details.QuantityPerSubnet == nil || *details.QuantityPerSubnet != 2 {
		t.Fatalf("buildNodePoolCreateDetails() QuantityPerSubnet = %#v, want 2", details.QuantityPerSubnet)
	}

	body := nodePoolSerializedRequestBody(t, containerenginesdk.CreateNodePoolRequest{
		CreateNodePoolDetails: details,
	}, http.MethodPost, "/nodePools")

	for _, want := range []string{`"subnetIds":["ocid1.subnet.oc1..legacy"]`, `"quantityPerSubnet":2`} {
		if !strings.Contains(body, want) {
			t.Fatalf("request body %s does not contain %s", body, want)
		}
	}
	if strings.Contains(body, `"nodeConfigDetails"`) {
		t.Fatalf("request body unexpectedly serialized nodeConfigDetails: %s", body)
	}
}

func TestBuildNodePoolCreateDetailsOmitsEmptyPreemptibleNodeConfigAndDerivedEmptyNsgIDs(t *testing.T) {
	t.Parallel()

	resource := newNodePoolTestResource()
	resource.Spec.NodeConfigDetails.NsgIds = nil
	resource.Spec.NodeConfigDetails.PlacementConfigs[0].PreemptibleNodeConfig = containerenginev1beta1.NodePoolNodeConfigDetailsPlacementConfigPreemptibleNodeConfig{}

	details, err := buildNodePoolCreateDetails(context.Background(), resource, resource.Namespace)
	if err != nil {
		t.Fatalf("buildNodePoolCreateDetails() error = %v", err)
	}
	if details.NodeConfigDetails == nil {
		t.Fatal("buildNodePoolCreateDetails() NodeConfigDetails = nil, want active nodeConfigDetails path")
	}
	if details.NodeConfigDetails.NsgIds != nil {
		t.Fatalf("buildNodePoolCreateDetails() NodeConfigDetails.NsgIds = %#v, want nil", details.NodeConfigDetails.NsgIds)
	}
	if got := details.NodeConfigDetails.PlacementConfigs[0].PreemptibleNodeConfig; got != nil {
		t.Fatalf("buildNodePoolCreateDetails() PlacementConfigs[0].PreemptibleNodeConfig = %#v, want nil", got)
	}

	body := nodePoolSerializedRequestBody(t, containerenginesdk.CreateNodePoolRequest{
		CreateNodePoolDetails: details,
	}, http.MethodPost, "/nodePools")

	for _, field := range []string{`"preemptibleNodeConfig"`, `"preemptionAction"`, `"nsgIds"`} {
		if strings.Contains(body, field) {
			t.Fatalf("request body unexpectedly serialized %s: %s", field, body)
		}
	}
}

func TestBuildNodePoolCreateDetailsPreservesExplicitEmptyNsgIDs(t *testing.T) {
	t.Parallel()

	resource := newNodePoolTestResource()
	resource.Spec.NodeConfigDetails.NsgIds = []string{}

	details, err := buildNodePoolCreateDetails(context.Background(), resource, resource.Namespace)
	if err != nil {
		t.Fatalf("buildNodePoolCreateDetails() error = %v", err)
	}
	if details.NodeConfigDetails == nil {
		t.Fatal("buildNodePoolCreateDetails() NodeConfigDetails = nil, want active nodeConfigDetails path")
	}
	if details.NodeConfigDetails.NsgIds == nil {
		t.Fatal("buildNodePoolCreateDetails() NodeConfigDetails.NsgIds = nil, want explicit empty slice preserved")
	}
	if len(details.NodeConfigDetails.NsgIds) != 0 {
		t.Fatalf("buildNodePoolCreateDetails() NodeConfigDetails.NsgIds = %#v, want empty slice", details.NodeConfigDetails.NsgIds)
	}

	body := nodePoolSerializedRequestBody(t, containerenginesdk.CreateNodePoolRequest{
		CreateNodePoolDetails: details,
	}, http.MethodPost, "/nodePools")

	if !strings.Contains(body, `"nsgIds":[]`) {
		t.Fatalf("request body %s does not preserve explicit empty nsgIds intent", body)
	}
}

func TestBuildNodePoolCreateDetailsRejectsNonEmptyUnsupportedPreemptibleNodeConfig(t *testing.T) {
	t.Parallel()

	resource := newNodePoolTestResource()
	resource.Spec.NodeConfigDetails.PlacementConfigs[0].PreemptibleNodeConfig =
		containerenginev1beta1.NodePoolNodeConfigDetailsPlacementConfigPreemptibleNodeConfig{
			PreemptionAction: containerenginev1beta1.NodePoolNodeConfigDetailsPlacementConfigPreemptibleNodeConfigPreemptionAction{
				IsPreserveBootVolume: true,
			},
		}

	_, err := buildNodePoolCreateDetails(context.Background(), resource, resource.Namespace)
	if err == nil {
		t.Fatal("buildNodePoolCreateDetails() error = nil, want unsupported preemptible action failure")
	}
	if !strings.Contains(err.Error(), "unsupported nodeConfigDetails.placementConfigs[0].preemptibleNodeConfig.preemptionAction type") {
		t.Fatalf("buildNodePoolCreateDetails() error = %v, want unsupported preemptible action context", err)
	}
}

func TestBuildNodePoolCreateDetailsRejectsUnsupportedNodeSourceType(t *testing.T) {
	t.Parallel()

	resource := newNodePoolTestResource()
	resource.Spec.NodeSourceDetails = containerenginev1beta1.NodePoolNodeSourceDetails{
		SourceType: "UNKNOWN",
		ImageId:    "ocid1.image.oc1..nodepool",
	}

	_, err := buildNodePoolCreateDetails(context.Background(), resource, resource.Namespace)
	if err == nil {
		t.Fatal("buildNodePoolCreateDetails() error = nil, want unsupported node source failure")
	}
	if !strings.Contains(err.Error(), "unsupported nodeSourceDetails type") {
		t.Fatalf("buildNodePoolCreateDetails() error = %v, want unsupported node source context", err)
	}
}

func TestBuildNodePoolUpdateBodyDetectsMutableDrift(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.nodepool.oc1..existing"

	resource := newNodePoolTestResource()
	currentDetails, err := buildNodePoolUpdateDetails(context.Background(), resource, resource.Namespace)
	if err != nil {
		t.Fatalf("buildNodePoolUpdateDetails() error = %v", err)
	}
	current := observedNodePoolFromUpdateDetails(t, existingID, resource.Spec, currentDetails, "ACTIVE")

	_, updateNeeded, err := buildNodePoolUpdateBody(context.Background(), resource, resource.Namespace, current)
	if err != nil {
		t.Fatalf("buildNodePoolUpdateBody() error = %v", err)
	}
	if updateNeeded {
		t.Fatal("buildNodePoolUpdateBody() reported drift for matching live state")
	}

	resource.Spec.NodeShape = "VM.Standard.E5.Flex"

	details, updateNeeded, err := buildNodePoolUpdateBody(context.Background(), resource, resource.Namespace, current)
	if err != nil {
		t.Fatalf("buildNodePoolUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildNodePoolUpdateBody() updateNeeded = false, want true after mutable drift")
	}
	if details.NodeShape == nil || *details.NodeShape != "VM.Standard.E5.Flex" {
		t.Fatalf("buildNodePoolUpdateBody() nodeShape = %#v, want VM.Standard.E5.Flex", details.NodeShape)
	}
}

func TestBuildNodePoolUpdateBodyDetectsExplicitEmptyNodeConfigNsgIDsDrift(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.nodepool.oc1..existing"

	resource := newNodePoolTestResource()
	resource.Spec.NodeConfigDetails.NsgIds = []string{}

	currentResource := newNodePoolTestResource()
	currentDetails, err := buildNodePoolUpdateDetails(context.Background(), currentResource, currentResource.Namespace)
	if err != nil {
		t.Fatalf("buildNodePoolUpdateDetails() error = %v", err)
	}
	current := observedNodePoolFromUpdateDetails(t, existingID, currentResource.Spec, currentDetails, "ACTIVE")

	details, updateNeeded, err := buildNodePoolUpdateBody(context.Background(), resource, resource.Namespace, current)
	if err != nil {
		t.Fatalf("buildNodePoolUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildNodePoolUpdateBody() updateNeeded = false, want true for explicit empty nsgIds drift")
	}
	if details.NodeConfigDetails == nil {
		t.Fatal("buildNodePoolUpdateBody() NodeConfigDetails = nil, want active nodeConfigDetails path")
	}
	if details.NodeConfigDetails.NsgIds == nil || len(details.NodeConfigDetails.NsgIds) != 0 {
		t.Fatalf("buildNodePoolUpdateBody() NodeConfigDetails.NsgIds = %#v, want explicit empty slice", details.NodeConfigDetails.NsgIds)
	}

	body := nodePoolSerializedRequestBody(t, containerenginesdk.UpdateNodePoolRequest{
		NodePoolId:            common.String(existingID),
		UpdateNodePoolDetails: details,
	}, http.MethodPut, "/nodePools/"+existingID)

	if !strings.Contains(body, `"nsgIds":[]`) {
		t.Fatalf("request body %s does not preserve explicit empty nsgIds intent", body)
	}
}

func TestBuildNodePoolUpdateDetailsOmitsDeprecatedPlacementFieldsWhenNodeConfigDetailsPresent(t *testing.T) {
	t.Parallel()

	resource := newNodePoolTestResource()

	details, err := buildNodePoolUpdateDetails(context.Background(), resource, resource.Namespace)
	if err != nil {
		t.Fatalf("buildNodePoolUpdateDetails() error = %v", err)
	}
	if details.SubnetIds != nil {
		t.Fatalf("buildNodePoolUpdateDetails() SubnetIds = %#v, want nil when nodeConfigDetails is set", details.SubnetIds)
	}
	if details.QuantityPerSubnet != nil {
		t.Fatalf("buildNodePoolUpdateDetails() QuantityPerSubnet = %#v, want nil when nodeConfigDetails is set", details.QuantityPerSubnet)
	}

	body := nodePoolSerializedRequestBody(t, containerenginesdk.UpdateNodePoolRequest{
		NodePoolId:            common.String("ocid1.nodepool.oc1..example"),
		UpdateNodePoolDetails: details,
	}, http.MethodPut, "/nodePools/ocid1.nodepool.oc1..example")

	if !strings.Contains(body, `"nodeConfigDetails"`) {
		t.Fatalf("request body %s does not contain nodeConfigDetails", body)
	}
	for _, field := range []string{`"subnetIds"`, `"quantityPerSubnet"`} {
		if strings.Contains(body, field) {
			t.Fatalf("request body unexpectedly serialized %s: %s", field, body)
		}
	}
}

func TestBuildNodePoolUpdateDetailsOmitsEmptyPreemptibleNodeConfigAndDerivedEmptyNsgIDs(t *testing.T) {
	t.Parallel()

	resource := newNodePoolTestResource()
	resource.Spec.NodeConfigDetails.NsgIds = nil
	resource.Spec.NodeConfigDetails.PlacementConfigs[0].PreemptibleNodeConfig = containerenginev1beta1.NodePoolNodeConfigDetailsPlacementConfigPreemptibleNodeConfig{}

	details, err := buildNodePoolUpdateDetails(context.Background(), resource, resource.Namespace)
	if err != nil {
		t.Fatalf("buildNodePoolUpdateDetails() error = %v", err)
	}
	if details.NodeConfigDetails == nil {
		t.Fatal("buildNodePoolUpdateDetails() NodeConfigDetails = nil, want active nodeConfigDetails path")
	}
	if details.NodeConfigDetails.NsgIds != nil {
		t.Fatalf("buildNodePoolUpdateDetails() NodeConfigDetails.NsgIds = %#v, want nil", details.NodeConfigDetails.NsgIds)
	}
	if got := details.NodeConfigDetails.PlacementConfigs[0].PreemptibleNodeConfig; got != nil {
		t.Fatalf("buildNodePoolUpdateDetails() PlacementConfigs[0].PreemptibleNodeConfig = %#v, want nil", got)
	}

	body := nodePoolSerializedRequestBody(t, containerenginesdk.UpdateNodePoolRequest{
		NodePoolId:            common.String("ocid1.nodepool.oc1..example"),
		UpdateNodePoolDetails: details,
	}, http.MethodPut, "/nodePools/ocid1.nodepool.oc1..example")

	for _, field := range []string{`"preemptibleNodeConfig"`, `"preemptionAction"`, `"nsgIds"`} {
		if strings.Contains(body, field) {
			t.Fatalf("request body unexpectedly serialized %s: %s", field, body)
		}
	}
}

func TestBuildNodePoolUpdateDetailsPreservesExplicitEmptyNsgIDs(t *testing.T) {
	t.Parallel()

	resource := newNodePoolTestResource()
	resource.Spec.NodeConfigDetails.NsgIds = []string{}

	details, err := buildNodePoolUpdateDetails(context.Background(), resource, resource.Namespace)
	if err != nil {
		t.Fatalf("buildNodePoolUpdateDetails() error = %v", err)
	}
	if details.NodeConfigDetails == nil {
		t.Fatal("buildNodePoolUpdateDetails() NodeConfigDetails = nil, want active nodeConfigDetails path")
	}
	if details.NodeConfigDetails.NsgIds == nil {
		t.Fatal("buildNodePoolUpdateDetails() NodeConfigDetails.NsgIds = nil, want explicit empty slice preserved")
	}
	if len(details.NodeConfigDetails.NsgIds) != 0 {
		t.Fatalf("buildNodePoolUpdateDetails() NodeConfigDetails.NsgIds = %#v, want empty slice", details.NodeConfigDetails.NsgIds)
	}

	body := nodePoolSerializedRequestBody(t, containerenginesdk.UpdateNodePoolRequest{
		NodePoolId:            common.String("ocid1.nodepool.oc1..example"),
		UpdateNodePoolDetails: details,
	}, http.MethodPut, "/nodePools/ocid1.nodepool.oc1..example")

	if !strings.Contains(body, `"nsgIds":[]`) {
		t.Fatalf("request body %s does not preserve explicit empty nsgIds intent", body)
	}
}

func TestBuildNodePoolUpdateDetailsRejectsNonEmptyUnsupportedPreemptibleNodeConfig(t *testing.T) {
	t.Parallel()

	resource := newNodePoolTestResource()
	resource.Spec.NodeConfigDetails.PlacementConfigs[0].PreemptibleNodeConfig =
		containerenginev1beta1.NodePoolNodeConfigDetailsPlacementConfigPreemptibleNodeConfig{
			PreemptionAction: containerenginev1beta1.NodePoolNodeConfigDetailsPlacementConfigPreemptibleNodeConfigPreemptionAction{
				IsPreserveBootVolume: true,
			},
		}

	_, err := buildNodePoolUpdateDetails(context.Background(), resource, resource.Namespace)
	if err == nil {
		t.Fatal("buildNodePoolUpdateDetails() error = nil, want unsupported preemptible action failure")
	}
	if !strings.Contains(err.Error(), "unsupported nodeConfigDetails.placementConfigs[0].preemptibleNodeConfig.preemptionAction type") {
		t.Fatalf("buildNodePoolUpdateDetails() error = %v, want unsupported preemptible action context", err)
	}
}

func TestNodePoolCreateOrUpdateClassifiesObservedLifecycleStates(t *testing.T) {
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
			name:           "inactive",
			lifecycleState: "INACTIVE",
			wantSuccessful: true,
			wantRequeue:    false,
			wantCondition:  shared.Active,
		},
		{
			name:           "needs attention",
			lifecycleState: "NEEDS_ATTENTION",
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

			const existingID = "ocid1.nodepool.oc1..existing"

			resource := newExistingNodePoolTestResource(existingID)
			var getRequest containerenginesdk.GetNodePoolRequest

			manager := newNodePoolRuntimeTestManager(generatedruntime.Config[*containerenginev1beta1.NodePool]{
				Get: &generatedruntime.Operation{
					NewRequest: func() any { return &containerenginesdk.GetNodePoolRequest{} },
					Call: func(_ context.Context, request any) (any, error) {
						getRequest = *request.(*containerenginesdk.GetNodePoolRequest)
						return containerenginesdk.GetNodePoolResponse{
							NodePool: observedNodePoolFromSpec(existingID, resource.Spec, tc.lifecycleState),
						}, nil
					},
					Fields: nodePoolGetFields(),
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
			if getRequest.NodePoolId == nil || *getRequest.NodePoolId != existingID {
				t.Fatalf("GetNodePoolRequest.NodePoolId = %v, want %s", getRequest.NodePoolId, existingID)
			}
			if got := resource.Status.OsokStatus.Reason; got != string(tc.wantCondition) {
				t.Fatalf("status.reason = %q, want %q", got, tc.wantCondition)
			}
		})
	}
}

func TestNodePoolCreateOrUpdateReusesListMatchesForSeededSuccessStates(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		lifecycleState string
	}{
		{
			name:           "inactive",
			lifecycleState: "INACTIVE",
		},
		{
			name:           "needs attention",
			lifecycleState: "NEEDS_ATTENTION",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			const existingID = "ocid1.nodepool.oc1..existing"

			resource := newNodePoolTestResource()
			createCalled := false
			getCalled := false
			var listRequest containerenginesdk.ListNodePoolsRequest

			manager := newNodePoolRuntimeTestManager(generatedruntime.Config[*containerenginev1beta1.NodePool]{
				Create: &generatedruntime.Operation{
					NewRequest: func() any { return &containerenginesdk.CreateNodePoolRequest{} },
					Call: func(_ context.Context, _ any) (any, error) {
						createCalled = true
						return containerenginesdk.CreateNodePoolResponse{}, nil
					},
					Fields: nodePoolCreateFields(),
				},
				Get: &generatedruntime.Operation{
					NewRequest: func() any { return &containerenginesdk.GetNodePoolRequest{} },
					Call: func(_ context.Context, request any) (any, error) {
						getCalled = true
						getRequest := request.(*containerenginesdk.GetNodePoolRequest)
						if getRequest.NodePoolId == nil || *getRequest.NodePoolId != existingID {
							t.Fatalf("GetNodePoolRequest.NodePoolId = %v, want %s", getRequest.NodePoolId, existingID)
						}
						return containerenginesdk.GetNodePoolResponse{
							NodePool: observedNodePoolFromSpec(existingID, resource.Spec, tc.lifecycleState),
						}, nil
					},
					Fields: nodePoolGetFields(),
				},
				List: &generatedruntime.Operation{
					NewRequest: func() any { return &containerenginesdk.ListNodePoolsRequest{} },
					Call: func(_ context.Context, request any) (any, error) {
						listRequest = *request.(*containerenginesdk.ListNodePoolsRequest)
						return containerenginesdk.ListNodePoolsResponse{
							Items: []containerenginesdk.NodePoolSummary{
								observedNodePoolSummaryFromSpec(existingID, resource.Spec, tc.lifecycleState),
							},
						}, nil
					},
					Fields: nodePoolListFields(),
				},
			})

			response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			if err != nil {
				t.Fatalf("CreateOrUpdate() error = %v", err)
			}
			if !response.IsSuccessful {
				t.Fatal("CreateOrUpdate() should report success when seeded lifecycle is reusable")
			}
			if createCalled {
				t.Fatal("CreateNodePool() should not be called when list lookup found a reusable seeded-state match")
			}
			if !getCalled {
				t.Fatal("GetNodePool() should be called to reread the reusable list match")
			}
			if listRequest.CompartmentId == nil || *listRequest.CompartmentId != resource.Spec.CompartmentId {
				t.Fatalf("ListNodePoolsRequest.CompartmentId = %v, want %s", listRequest.CompartmentId, resource.Spec.CompartmentId)
			}
			if listRequest.ClusterId == nil || *listRequest.ClusterId != resource.Spec.ClusterId {
				t.Fatalf("ListNodePoolsRequest.ClusterId = %v, want %s", listRequest.ClusterId, resource.Spec.ClusterId)
			}
			if listRequest.Name == nil || *listRequest.Name != resource.Spec.Name {
				t.Fatalf("ListNodePoolsRequest.Name = %v, want %s", listRequest.Name, resource.Spec.Name)
			}
			if got := string(resource.Status.OsokStatus.Ocid); got != existingID {
				t.Fatalf("status.ocid = %q, want %q", got, existingID)
			}
		})
	}
}

func TestNewNodePoolServiceClientAllowsPrimaryListSemantics(t *testing.T) {
	t.Parallel()

	manager := &NodePoolServiceManager{
		Provider: newNodePoolTestConfigurationProvider(t),
		Log:      loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
	}

	client := newNodePoolServiceClient(manager)
	response, err := client.CreateOrUpdate(context.Background(), (*containerenginev1beta1.NodePool)(nil), ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want resource validation failure")
	}
	if strings.Contains(err.Error(), "unsupported list auxiliary operation ListNodePools") {
		t.Fatalf("CreateOrUpdate() error = %v, want primary list semantics without auxiliary-operation blocker", err)
	}
	if !strings.Contains(err.Error(), "expected pointer resource") {
		t.Fatalf("CreateOrUpdate() error = %v, want nil resource validation after successful client construction", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should not report success for a nil NodePool resource")
	}
}

type nodePoolRequestBodyBuilder interface {
	HTTPRequest(string, string, *common.OCIReadSeekCloser, map[string]string) (http.Request, error)
}

func nodePoolSerializedRequestBody(t *testing.T, request nodePoolRequestBodyBuilder, method string, path string) string {
	t.Helper()

	httpRequest, err := request.HTTPRequest(method, path, nil, nil)
	if err != nil {
		t.Fatalf("HTTPRequest() error = %v", err)
	}

	body, err := io.ReadAll(httpRequest.Body)
	if err != nil {
		t.Fatalf("ReadAll(request.Body) error = %v", err)
	}

	return string(body)
}

func newNodePoolTestConfigurationProvider(t *testing.T) common.ConfigurationProvider {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	return common.NewRawConfigurationProvider(
		"ocid1.tenancy.oc1..example",
		"ocid1.user.oc1..example",
		"us-ashburn-1",
		"20:3b:97:13:55:1c",
		string(privateKeyPEM),
		nil,
	)
}

func newNodePoolRuntimeTestManager(cfg generatedruntime.Config[*containerenginev1beta1.NodePool]) *NodePoolServiceManager {
	if cfg.Kind == "" {
		cfg.Kind = "NodePool"
	}
	if cfg.SDKName == "" {
		cfg.SDKName = "NodePool"
	}
	if cfg.Semantics == nil {
		cfg.Semantics = testNodePoolRuntimeSemantics()
	}
	if cfg.BuildCreateBody == nil {
		cfg.BuildCreateBody = func(ctx context.Context, resource *containerenginev1beta1.NodePool, namespace string) (any, error) {
			return buildNodePoolCreateDetails(ctx, resource, namespace)
		}
	}
	if cfg.BuildUpdateBody == nil {
		cfg.BuildUpdateBody = func(
			ctx context.Context,
			resource *containerenginev1beta1.NodePool,
			namespace string,
			currentResponse any,
		) (any, bool, error) {
			return buildNodePoolUpdateBody(ctx, resource, namespace, currentResponse)
		}
	}

	return &NodePoolServiceManager{
		client: defaultNodePoolServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*containerenginev1beta1.NodePool](cfg),
		},
	}
}

func testNodePoolRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{"CREATING"},
			UpdatingStates:     []string{"UPDATING"},
			ActiveStates:       []string{"ACTIVE", "INACTIVE", "NEEDS_ATTENTION"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{"DELETING"},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "clusterId", "name", "lifecycleState"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"definedTags",
				"freeformTags",
				"initialNodeLabels",
				"kubernetesVersion",
				"name",
				"nodeConfigDetails",
				"nodeEvictionNodePoolSettings",
				"nodeMetadata",
				"nodePoolCyclingDetails",
				"nodeShape",
				"nodeShapeConfig",
				"nodeSourceDetails",
				"sshPublicKey",
			},
			ForceNew: []string{
				"clusterId",
				"compartmentId",
				"nodeImageName",
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

func newNodePoolTestResource() *containerenginev1beta1.NodePool {
	return &containerenginev1beta1.NodePool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nodepool-sample",
			Namespace: "default",
		},
		Spec: containerenginev1beta1.NodePoolSpec{
			CompartmentId:     "ocid1.compartment.oc1..example",
			ClusterId:         "ocid1.cluster.oc1..example",
			Name:              "nodepool-sample",
			NodeShape:         "VM.Standard.E4.Flex",
			KubernetesVersion: "v1.30.1",
			NodeMetadata: map[string]string{
				"user-data": "cloud-init",
			},
			NodeSourceDetails: containerenginev1beta1.NodePoolNodeSourceDetails{
				SourceType:          "IMAGE",
				ImageId:             "ocid1.image.oc1..nodepool",
				BootVolumeSizeInGBs: 80,
			},
			NodeShapeConfig: containerenginev1beta1.NodePoolNodeShapeConfig{
				Ocpus:       2,
				MemoryInGBs: 16,
			},
			InitialNodeLabels: []containerenginev1beta1.NodePoolInitialNodeLabel{
				{
					Key:   "pool",
					Value: "workers",
				},
			},
			SshPublicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDc nodepool@test",
			NodeConfigDetails: containerenginev1beta1.NodePoolNodeConfigDetails{
				Size: 3,
				PlacementConfigs: []containerenginev1beta1.NodePoolNodeConfigDetailsPlacementConfig{
					{
						AvailabilityDomain:    "PHX-AD-1",
						SubnetId:              "ocid1.subnet.oc1..worker",
						CapacityReservationId: "ocid1.capacityreservation.oc1..example",
						PreemptibleNodeConfig: containerenginev1beta1.NodePoolNodeConfigDetailsPlacementConfigPreemptibleNodeConfig{
							PreemptionAction: containerenginev1beta1.NodePoolNodeConfigDetailsPlacementConfigPreemptibleNodeConfigPreemptionAction{
								Type:                 "TERMINATE",
								IsPreserveBootVolume: false,
							},
						},
						FaultDomains: []string{"FAULT-DOMAIN-1"},
					},
				},
				NsgIds:                         []string{"ocid1.nsg.oc1..worker"},
				IsPvEncryptionInTransitEnabled: false,
				FreeformTags: map[string]string{
					"team": "oke",
				},
				DefinedTags: map[string]shared.MapValue{
					"Operations": {
						"CostCenter": "42",
					},
				},
				NodePoolPodNetworkOptionDetails: containerenginev1beta1.NodePoolNodeConfigDetailsNodePoolPodNetworkOptionDetails{
					CniType:        "OCI_VCN_IP_NATIVE",
					PodSubnetIds:   []string{"ocid1.subnet.oc1..pods"},
					MaxPodsPerNode: 31,
					PodNsgIds:      []string{"ocid1.nsg.oc1..pods"},
				},
			},
			FreeformTags: map[string]string{
				"env": "test",
			},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {
					"CostCenter": "42",
				},
			},
			NodeEvictionNodePoolSettings: containerenginev1beta1.NodePoolNodeEvictionNodePoolSettings{
				EvictionGraceDuration:           "PT30M",
				IsForceDeleteAfterGraceDuration: false,
			},
			NodePoolCyclingDetails: containerenginev1beta1.NodePoolCyclingDetails{
				MaximumUnavailable:   "0",
				MaximumSurge:         "1",
				IsNodeCyclingEnabled: false,
			},
		},
	}
}

func newExistingNodePoolTestResource(existingID string) *containerenginev1beta1.NodePool {
	resource := newNodePoolTestResource()
	resource.Status = containerenginev1beta1.NodePoolStatus{
		OsokStatus: shared.OSOKStatus{
			Ocid: shared.OCID(existingID),
		},
		Id: existingID,
	}
	return resource
}

func observedNodePoolFromSpec(
	id string,
	spec containerenginev1beta1.NodePoolSpec,
	lifecycleState string,
) containerenginesdk.NodePool {
	lifecycleDetails := "nodepool " + strings.ToLower(lifecycleState) + " details"
	return containerenginesdk.NodePool{
		Id:                common.String(id),
		CompartmentId:     common.String(spec.CompartmentId),
		ClusterId:         common.String(spec.ClusterId),
		Name:              common.String(spec.Name),
		KubernetesVersion: common.String(spec.KubernetesVersion),
		NodeShape:         common.String(spec.NodeShape),
		LifecycleState:    containerenginesdk.NodePoolLifecycleStateEnum(lifecycleState),
		LifecycleDetails:  common.String(lifecycleDetails),
	}
}

func observedNodePoolSummaryFromSpec(
	id string,
	spec containerenginev1beta1.NodePoolSpec,
	lifecycleState string,
) containerenginesdk.NodePoolSummary {
	lifecycleDetails := "nodepool " + strings.ToLower(lifecycleState) + " details"
	return containerenginesdk.NodePoolSummary{
		Id:                common.String(id),
		CompartmentId:     common.String(spec.CompartmentId),
		ClusterId:         common.String(spec.ClusterId),
		Name:              common.String(spec.Name),
		KubernetesVersion: common.String(spec.KubernetesVersion),
		NodeShape:         common.String(spec.NodeShape),
		LifecycleState:    containerenginesdk.NodePoolLifecycleStateEnum(lifecycleState),
		LifecycleDetails:  common.String(lifecycleDetails),
	}
}

func observedNodePoolFromUpdateDetails(
	t *testing.T,
	id string,
	spec containerenginev1beta1.NodePoolSpec,
	details containerenginesdk.UpdateNodePoolDetails,
	lifecycleState string,
) containerenginesdk.NodePool {
	t.Helper()

	payload, err := json.Marshal(details)
	if err != nil {
		t.Fatalf("json.Marshal(update details) error = %v", err)
	}

	var current containerenginesdk.NodePool
	if err := json.Unmarshal(payload, &current); err != nil {
		t.Fatalf("json.Unmarshal(update details into current response) error = %v", err)
	}

	current.Id = common.String(id)
	current.CompartmentId = common.String(spec.CompartmentId)
	current.ClusterId = common.String(spec.ClusterId)
	if current.Name == nil {
		current.Name = common.String(spec.Name)
	}
	if current.NodeShape == nil {
		current.NodeShape = common.String(spec.NodeShape)
	}
	current.LifecycleState = containerenginesdk.NodePoolLifecycleStateEnum(lifecycleState)
	current.LifecycleDetails = common.String("nodepool " + strings.ToLower(lifecycleState) + " details")

	return current
}

func nodePoolCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateNodePoolDetails", RequestName: "CreateNodePoolDetails", Contribution: "body"},
	}
}

func nodePoolGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "NodePoolId", RequestName: "nodePoolId", Contribution: "path", PreferResourceID: true},
	}
}

func nodePoolListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
		{FieldName: "ClusterId", RequestName: "clusterId", Contribution: "query"},
		{FieldName: "Name", RequestName: "name", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
	}
}

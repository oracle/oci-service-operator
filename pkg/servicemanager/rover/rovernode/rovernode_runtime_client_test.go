package rovernode

import (
	"context"
	"io"
	"maps"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	roversdk "github.com/oracle/oci-go-sdk/v65/rover"
	roverv1beta1 "github.com/oracle/oci-service-operator/api/rover/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestApplyRoverNodeRuntimeHooksOverridesGeneratedDefaults(t *testing.T) {
	t.Parallel()

	hooks := newRoverNodeDefaultRuntimeHooks(roversdk.RoverNodeClient{})
	applyRoverNodeRuntimeHooks(&hooks)

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed semantics")
	}
	if len(hooks.Semantics.AuxiliaryOperations) != 0 {
		t.Fatalf("hooks.Semantics.AuxiliaryOperations = %#v, want none", hooks.Semantics.AuxiliaryOperations)
	}
	if got, want := hooks.Semantics.List.MatchFields, []string{"compartmentId", "displayName", "serialNumber", "shape"}; !equalStringSlices(got, want) {
		t.Fatalf("hooks.Semantics.List.MatchFields = %#v, want %#v", got, want)
	}
	if got, want := hooks.List.Fields, reviewedRoverNodeListFields(); !equalRequestFields(got, want) {
		t.Fatalf("hooks.List.Fields = %#v, want %#v", got, want)
	}
	if hooks.Identity.GuardExistingBeforeCreate == nil {
		t.Fatal("hooks.Identity.GuardExistingBeforeCreate = nil, want explicit serialNumber guard")
	}
	if hooks.ParityHooks.NormalizeDesiredState == nil {
		t.Fatal("hooks.ParityHooks.NormalizeDesiredState = nil, want provider-managed field normalization")
	}
	if hooks.BuildUpdateBody == nil {
		t.Fatal("hooks.BuildUpdateBody = nil, want reviewed update builder")
	}

	resource := newRoverNodeTestResource()
	resource.Spec.LifecycleState = "ACTIVE"
	resource.Spec.LifecycleStateDetails = "provider-managed"
	resource.Spec.SystemTags = map[string]shared.MapValue{
		"orcl-cloud": {
			"free-tier-retained": "true",
		},
	}
	resource.Spec.NodeWorkloads[0].WorkRequestId = "wr-custom-image-export"

	hooks.ParityHooks.NormalizeDesiredState(resource, nil)

	if resource.Spec.LifecycleState != "" {
		t.Fatalf("resource.Spec.LifecycleState = %q, want empty string", resource.Spec.LifecycleState)
	}
	if resource.Spec.LifecycleStateDetails != "" {
		t.Fatalf("resource.Spec.LifecycleStateDetails = %q, want empty string", resource.Spec.LifecycleStateDetails)
	}
	if resource.Spec.SystemTags != nil {
		t.Fatalf("resource.Spec.SystemTags = %#v, want nil", resource.Spec.SystemTags)
	}
	if got := resource.Spec.NodeWorkloads[0].WorkRequestId; got != "" {
		t.Fatalf("resource.Spec.NodeWorkloads[0].WorkRequestId = %q, want empty string", got)
	}
}

func TestGuardRoverNodeExistingBeforeCreateSkipsReuseWithoutSerialNumber(t *testing.T) {
	t.Parallel()

	resource := newRoverNodeTestResource()
	resource.Spec.SerialNumber = "   "

	decision, err := guardRoverNodeExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardRoverNodeExistingBeforeCreate() error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionSkip {
		t.Fatalf("guardRoverNodeExistingBeforeCreate() = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionSkip)
	}
}

func TestWrapRoverNodeListCallEnforceStandalone(t *testing.T) {
	t.Parallel()

	var seen roversdk.ListRoverNodesRequest
	call := wrapRoverNodeListCallEnforceStandalone(func(_ context.Context, request roversdk.ListRoverNodesRequest) (roversdk.ListRoverNodesResponse, error) {
		seen = request
		return roversdk.ListRoverNodesResponse{}, nil
	})

	_, err := call(context.Background(), roversdk.ListRoverNodesRequest{})
	if err != nil {
		t.Fatalf("wrapped list call error = %v", err)
	}
	if seen.NodeType != roversdk.ListRoverNodesNodeTypeStandalone {
		t.Fatalf("wrapped ListRoverNodesRequest.NodeType = %q, want %q", seen.NodeType, roversdk.ListRoverNodesNodeTypeStandalone)
	}
}

func TestBuildRoverNodeUpdateBodySupportsClearingOptionalFields(t *testing.T) {
	t.Parallel()

	currentSpec := newRoverNodeTestResource().Spec
	resource := newRoverNodeTestResource()
	resource.Spec.PointOfContact = ""
	resource.Spec.PointOfContactPhoneNumber = ""
	resource.Spec.DataValidationCode = ""
	resource.Spec.SuperUserPassword = ""
	resource.Spec.UnlockPassphrase = ""
	resource.Spec.FreeformTags = map[string]string{}
	resource.Spec.DefinedTags = map[string]shared.MapValue{}

	current := observedRoverNodeFromSpec("ocid1.rovernode.oc1..existing", currentSpec, "ACTIVE")

	details, updateNeeded, err := buildRoverNodeUpdateBody(resource, current)
	if err != nil {
		t.Fatalf("buildRoverNodeUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildRoverNodeUpdateBody() updateNeeded = false, want clear intent")
	}
	if details.PointOfContact == nil || *details.PointOfContact != "" {
		t.Fatalf("PointOfContact = %#v, want explicit empty string clear", details.PointOfContact)
	}
	if details.PointOfContactPhoneNumber == nil || *details.PointOfContactPhoneNumber != "" {
		t.Fatalf("PointOfContactPhoneNumber = %#v, want explicit empty string clear", details.PointOfContactPhoneNumber)
	}
	if details.DataValidationCode == nil || *details.DataValidationCode != "" {
		t.Fatalf("DataValidationCode = %#v, want explicit empty string clear", details.DataValidationCode)
	}
	if details.SuperUserPassword == nil || *details.SuperUserPassword != "" {
		t.Fatalf("SuperUserPassword = %#v, want explicit empty string clear", details.SuperUserPassword)
	}
	if details.UnlockPassphrase == nil || *details.UnlockPassphrase != "" {
		t.Fatalf("UnlockPassphrase = %#v, want explicit empty string clear", details.UnlockPassphrase)
	}
	if details.FreeformTags == nil || len(details.FreeformTags) != 0 {
		t.Fatalf("FreeformTags = %#v, want explicit empty map clear", details.FreeformTags)
	}
	if details.DefinedTags == nil || len(details.DefinedTags) != 0 {
		t.Fatalf("DefinedTags = %#v, want explicit empty map clear", details.DefinedTags)
	}

	body := roverNodeSerializedRequestBody(t, roversdk.UpdateRoverNodeRequest{
		RoverNodeId:            common.String("ocid1.rovernode.oc1..existing"),
		UpdateRoverNodeDetails: details,
	}, http.MethodPut, "/roverNodes/ocid1.rovernode.oc1..existing")

	for _, want := range []string{
		`"pointOfContact":""`,
		`"pointOfContactPhoneNumber":""`,
		`"dataValidationCode":""`,
		`"superUserPassword":""`,
		`"unlockPassphrase":""`,
		`"freeformTags":{}`,
		`"definedTags":{}`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("request body %s does not contain %s", body, want)
		}
	}
}

func TestRoverNodeCreateOrUpdateCreatesWhenListCandidateSerialNumberDiffers(t *testing.T) {
	t.Parallel()

	resource := newRoverNodeTestResource()
	resource.Spec.LifecycleState = "ACTIVE"
	resource.Spec.LifecycleStateDetails = "provider-managed"
	resource.Spec.SystemTags = map[string]shared.MapValue{
		"orcl-cloud": {
			"free-tier-retained": "true",
		},
	}
	resource.Spec.NodeWorkloads[0].WorkRequestId = "wr-custom-image-export"

	createSpec := resource.Spec
	mismatchedSpec := resource.Spec
	mismatchedSpec.SerialNumber = "SN-99999"

	defaultHooks := newRoverNodeDefaultRuntimeHooks(roversdk.RoverNodeClient{})

	var createRequest roversdk.CreateRoverNodeRequest
	createCalled := false
	listCalled := false

	manager := newRoverNodeRuntimeTestManager(generatedruntime.Config[*roverv1beta1.RoverNode]{
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &roversdk.ListRoverNodesRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), reviewedRoverNodeListFields()...),
			Call: func(_ context.Context, request any) (any, error) {
				listCalled = true
				listRequest := *request.(*roversdk.ListRoverNodesRequest)
				if listRequest.NodeType != roversdk.ListRoverNodesNodeTypeStandalone {
					t.Fatalf("ListRoverNodesRequest.NodeType = %q, want %q for standalone-only reviewed scope", listRequest.NodeType, roversdk.ListRoverNodesNodeTypeStandalone)
				}
				if listRequest.LifecycleState != "" {
					t.Fatalf("ListRoverNodesRequest.LifecycleState = %q, want empty string after reviewed list-field override", listRequest.LifecycleState)
				}
				return roversdk.ListRoverNodesResponse{
					RoverNodeCollection: roversdk.RoverNodeCollection{
						Items: []roversdk.RoverNodeSummary{
							observedRoverNodeSummaryFromSpec("ocid1.rovernode.oc1..existing", mismatchedSpec, "ACTIVE"),
						},
					},
				}, nil
			},
		},
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &roversdk.CreateRoverNodeRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), defaultHooks.Create.Fields...),
			Call: func(_ context.Context, request any) (any, error) {
				createCalled = true
				createRequest = *request.(*roversdk.CreateRoverNodeRequest)
				return roversdk.CreateRoverNodeResponse{
					RoverNode:    observedRoverNodeFromSpec("ocid1.rovernode.oc1..created", createSpec, "CREATING"),
					OpcRequestId: common.String("opc-create"),
				}, nil
			},
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !listCalled {
		t.Fatal("List() should be called before create")
	}
	if !createCalled {
		t.Fatal("Create() should be called when the only list candidate mismatches serialNumber")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = false, want true while lifecycle remains CREATING")
	}

	requireRoverNodeOpcRequestID(t, resource, "opc-create")
	requireRoverNodeAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, "CREATING", shared.OSOKAsyncClassPending, "")

	body := roverNodeSerializedRequestBody(t, createRequest, http.MethodPost, "/roverNodes")
	for _, unwanted := range []string{`"lifecycleState"`, `"lifecycleStateDetails"`, `"systemTags"`, `"workRequestId"`} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("request body %s unexpectedly contains %s", body, unwanted)
		}
	}
}

func TestRoverNodeCreateOrUpdateCreatesWithoutReuseWhenSerialNumberIsBlank(t *testing.T) {
	t.Parallel()

	resource := newRoverNodeTestResource()
	resource.Spec.SerialNumber = ""

	defaultHooks := newRoverNodeDefaultRuntimeHooks(roversdk.RoverNodeClient{})

	createCalled := false
	preCreateListCalled := false

	manager := newRoverNodeRuntimeTestManager(generatedruntime.Config[*roverv1beta1.RoverNode]{
		Identity: generatedruntime.IdentityHooks[*roverv1beta1.RoverNode]{
			GuardExistingBeforeCreate: guardRoverNodeExistingBeforeCreate,
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &roversdk.ListRoverNodesRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), reviewedRoverNodeListFields()...),
			Call: func(_ context.Context, request any) (any, error) {
				if !createCalled {
					preCreateListCalled = true
				}
				return roversdk.ListRoverNodesResponse{
					RoverNodeCollection: roversdk.RoverNodeCollection{
						Items: []roversdk.RoverNodeSummary{
							observedRoverNodeSummaryFromSpec("ocid1.rovernode.oc1..existing", newRoverNodeTestResource().Spec, "ACTIVE"),
						},
					},
				}, nil
			},
		},
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &roversdk.CreateRoverNodeRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), defaultHooks.Create.Fields...),
			Call: func(_ context.Context, _ any) (any, error) {
				createCalled = true
				return roversdk.CreateRoverNodeResponse{
					RoverNode:    observedRoverNodeFromSpec("ocid1.rovernode.oc1..created", resource.Spec, "CREATING"),
					OpcRequestId: common.String("opc-create"),
				}, nil
			},
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if preCreateListCalled {
		t.Fatal("List() should not be used for pre-create reuse when desired serialNumber is blank")
	}
	if !createCalled {
		t.Fatal("Create() should be called when desired serialNumber is blank")
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true after create path succeeds")
	}
}

func TestRoverNodeCreateOrUpdateUpdatesUniqueExactSummaryMatch(t *testing.T) {
	t.Parallel()

	currentSpec := newRoverNodeTestResource().Spec
	resource := newRoverNodeTestResource()
	resource.Spec.PointOfContact = "Updated Contact"

	defaultHooks := newRoverNodeDefaultRuntimeHooks(roversdk.RoverNodeClient{})

	var updateRequest roversdk.UpdateRoverNodeRequest
	createCalled := false
	updateCalled := false
	getCalls := 0

	manager := newRoverNodeRuntimeTestManager(generatedruntime.Config[*roverv1beta1.RoverNode]{
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &roversdk.ListRoverNodesRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), reviewedRoverNodeListFields()...),
			Call: func(_ context.Context, request any) (any, error) {
				listRequest := *request.(*roversdk.ListRoverNodesRequest)
				if listRequest.NodeType != roversdk.ListRoverNodesNodeTypeStandalone {
					t.Fatalf("ListRoverNodesRequest.NodeType = %q, want %q for standalone-only reviewed scope", listRequest.NodeType, roversdk.ListRoverNodesNodeTypeStandalone)
				}
				if listRequest.LifecycleState != "" {
					t.Fatalf("ListRoverNodesRequest.LifecycleState = %q, want empty string after reviewed list-field override", listRequest.LifecycleState)
				}
				return roversdk.ListRoverNodesResponse{
					RoverNodeCollection: roversdk.RoverNodeCollection{
						Items: []roversdk.RoverNodeSummary{
							observedRoverNodeSummaryFromSpec("ocid1.rovernode.oc1..existing", currentSpec, "ACTIVE"),
						},
					},
				}, nil
			},
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &roversdk.GetRoverNodeRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), defaultHooks.Get.Fields...),
			Call: func(_ context.Context, request any) (any, error) {
				getCalls++
				getRequest := *request.(*roversdk.GetRoverNodeRequest)
				if getRequest.RoverNodeId == nil || *getRequest.RoverNodeId != "ocid1.rovernode.oc1..existing" {
					t.Fatalf("GetRoverNodeRequest.RoverNodeId = %#v, want existing rover node ID", getRequest.RoverNodeId)
				}
				lifecycleState := "ACTIVE"
				if getCalls > 1 {
					lifecycleState = "UPDATING"
				}
				return roversdk.GetRoverNodeResponse{
					RoverNode:    observedRoverNodeFromSpec("ocid1.rovernode.oc1..existing", currentSpec, lifecycleState),
					OpcRequestId: common.String("opc-get"),
				}, nil
			},
		},
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &roversdk.CreateRoverNodeRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), defaultHooks.Create.Fields...),
			Call: func(_ context.Context, _ any) (any, error) {
				createCalled = true
				return nil, nil
			},
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &roversdk.UpdateRoverNodeRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), defaultHooks.Update.Fields...),
			Call: func(_ context.Context, request any) (any, error) {
				updateCalled = true
				updateRequest = *request.(*roversdk.UpdateRoverNodeRequest)
				return roversdk.UpdateRoverNodeResponse{
					RoverNode:    observedRoverNodeFromSpec("ocid1.rovernode.oc1..existing", resource.Spec, "UPDATING"),
					OpcRequestId: common.String("opc-update"),
				}, nil
			},
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if createCalled {
		t.Fatal("Create() should not be called when a unique exact list candidate is reusable")
	}
	if !updateCalled {
		t.Fatal("Update() should be called when reviewed mutable drift exists on the reused node")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = false, want true while lifecycle remains UPDATING")
	}
	if updateRequest.UpdateRoverNodeDetails.PointOfContact == nil || *updateRequest.UpdateRoverNodeDetails.PointOfContact != resource.Spec.PointOfContact {
		t.Fatalf("UpdateRoverNodeDetails.PointOfContact = %#v, want %q", updateRequest.UpdateRoverNodeDetails.PointOfContact, resource.Spec.PointOfContact)
	}
	if updateRequest.UpdateRoverNodeDetails.DisplayName != nil {
		t.Fatalf("UpdateRoverNodeDetails.DisplayName = %#v, want nil for unchanged field", updateRequest.UpdateRoverNodeDetails.DisplayName)
	}

	requireRoverNodeOpcRequestID(t, resource, "opc-update")
	requireRoverNodeAsyncCurrent(t, resource, shared.OSOKAsyncPhaseUpdate, "UPDATING", shared.OSOKAsyncClassPending, "")
}

type roverNodeRequestBodyBuilder interface {
	HTTPRequest(string, string, *common.OCIReadSeekCloser, map[string]string) (http.Request, error)
}

func roverNodeSerializedRequestBody(
	t *testing.T,
	request roverNodeRequestBodyBuilder,
	method string,
	path string,
) string {
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

func newRoverNodeRuntimeTestManager(
	cfg generatedruntime.Config[*roverv1beta1.RoverNode],
) *RoverNodeServiceManager {
	if cfg.Kind == "" {
		cfg.Kind = "RoverNode"
	}
	if cfg.SDKName == "" {
		cfg.SDKName = "RoverNode"
	}
	if cfg.Semantics == nil {
		cfg.Semantics = reviewedRoverNodeRuntimeSemantics()
	}
	if cfg.Identity.GuardExistingBeforeCreate == nil {
		cfg.Identity.GuardExistingBeforeCreate = guardRoverNodeExistingBeforeCreate
	}
	if cfg.ParityHooks.NormalizeDesiredState == nil {
		cfg.ParityHooks.NormalizeDesiredState = normalizeRoverNodeDesiredState
	}
	if cfg.BuildUpdateBody == nil {
		cfg.BuildUpdateBody = func(
			_ context.Context,
			resource *roverv1beta1.RoverNode,
			_ string,
			currentResponse any,
		) (any, bool, error) {
			return buildRoverNodeUpdateBody(resource, currentResponse)
		}
	}
	if cfg.List != nil && cfg.List.Call != nil {
		call := cfg.List.Call
		cfg.List.Call = func(ctx context.Context, request any) (any, error) {
			listRequest := request.(*roversdk.ListRoverNodesRequest)
			listRequest.NodeType = roversdk.ListRoverNodesNodeTypeStandalone
			return call(ctx, request)
		}
	}

	return &RoverNodeServiceManager{
		client: defaultRoverNodeServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*roverv1beta1.RoverNode](cfg),
		},
	}
}

func newRoverNodeTestResource() *roverv1beta1.RoverNode {
	return &roverv1beta1.RoverNode{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rovernode-sample",
			Namespace: "default",
		},
		Spec: roverv1beta1.RoverNodeSpec{
			DisplayName:   "rovernode-sample",
			CompartmentId: "ocid1.compartment.oc1..example",
			Shape:         "ROVER.MINI",
			CustomerShippingAddress: roverv1beta1.RoverNodeCustomerShippingAddress{
				Addressee:      "Alice Example",
				Address1:       "1 Main Street",
				CityOrLocality: "Austin",
				StateOrRegion:  "TX",
				Zipcode:        "78701",
				Country:        "US",
				PhoneNumber:    "555-0100",
				Email:          "alice@example.com",
			},
			NodeWorkloads: []roverv1beta1.RoverNodeNodeWorkload{
				{
					CompartmentId: "ocid1.compartment.oc1..workload",
					Id:            "ocid1.workload.oc1..example",
					WorkloadType:  "OBJECT_STORAGE",
					Name:          "sample-workload",
					Size:          "25TB",
				},
			},
			SuperUserPassword:         "TempPassw0rd!",
			UnlockPassphrase:          "Unlock123!",
			PointOfContact:            "Alice Example",
			PointOfContactPhoneNumber: "555-0100",
			ShippingPreference:        "ORACLE_SHIPPED",
			ShippingVendor:            "UPS",
			TimePickupExpected:        "2026-05-01T12:00:00Z",
			PublicKey:                 "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDexample",
			TimeReturnWindowStarts:    "2026-05-02T12:00:00Z",
			TimeReturnWindowEnds:      "2026-05-03T12:00:00Z",
			EnclosureType:             "NON_RUGGADIZED",
			SerialNumber:              "SN-12345",
			OracleShippingTrackingUrl: "https://tracking.example.com/rovernode",
			IsImportRequested:         true,
			ImportCompartmentId:       "ocid1.compartment.oc1..import",
			ImportFileBucket:          "import-bucket",
			DataValidationCode:        "validate-123",
			MasterKeyId:               "ocid1.key.oc1..master",
			CertificateAuthorityId:    "ocid1.certificateauthority.oc1..ca",
			TimeCertValidityEnd:       "2026-06-01T12:00:00Z",
			CommonName:                "rovernode.example.internal",
			CertCompartmentId:         "ocid1.compartment.oc1..cert",
			CertKeyAlgorithm:          "RSA2048",
			CertSignatureAlgorithm:    "SHA256_WITH_RSA",
			FreeformTags:              map[string]string{"env": "test"},
			DefinedTags:               map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func observedRoverNodeFromSpec(
	id string,
	spec roverv1beta1.RoverNodeSpec,
	lifecycleState string,
) roversdk.RoverNode {
	now := &common.SDKTime{Time: time.Unix(1713240000, 0).UTC()}

	address, err := roverNodeShippingAddressFromSpec(spec.CustomerShippingAddress)
	if err != nil {
		panic(err)
	}
	workloads, err := roverNodeWorkloadsFromSpec(spec.NodeWorkloads)
	if err != nil {
		panic(err)
	}
	definedTags, err := roverNodeDefinedTagsFromSpec(spec.DefinedTags)
	if err != nil {
		panic(err)
	}

	return roversdk.RoverNode{
		Id:                        common.String(id),
		CompartmentId:             common.String(spec.CompartmentId),
		DisplayName:               common.String(spec.DisplayName),
		LifecycleState:            roversdk.LifecycleStateEnum(lifecycleState),
		NodeType:                  roversdk.NodeTypeStandalone,
		Shape:                     pointerOrNil(spec.Shape),
		EnclosureType:             roversdk.EnclosureTypeEnum(spec.EnclosureType),
		SerialNumber:              pointerOrNil(spec.SerialNumber),
		TimeCreated:               now,
		LifecycleStateDetails:     ptr("lifecycle details"),
		CustomerShippingAddress:   address,
		NodeWorkloads:             workloads,
		TimeCustomerReceieved:     now,
		TimeCustomerReturned:      now,
		DeliveryTrackingInfo:      ptr("tracking-123"),
		SuperUserPassword:         pointerOrNil(spec.SuperUserPassword),
		UnlockPassphrase:          pointerOrNil(spec.UnlockPassphrase),
		PointOfContact:            pointerOrNil(spec.PointOfContact),
		PointOfContactPhoneNumber: pointerOrNil(spec.PointOfContactPhoneNumber),
		ShippingPreference:        roversdk.RoverNodeShippingPreferenceEnum(spec.ShippingPreference),
		ShippingVendor:            pointerOrNil(spec.ShippingVendor),
		TimePickupExpected:        sdkTimeOrNil(spec.TimePickupExpected),
		PublicKey:                 pointerOrNil(spec.PublicKey),
		TimeReturnWindowStarts:    sdkTimeOrNil(spec.TimeReturnWindowStarts),
		OracleShippingTrackingUrl: pointerOrNil(spec.OracleShippingTrackingUrl),
		TimeReturnWindowEnds:      sdkTimeOrNil(spec.TimeReturnWindowEnds),
		ReturnShippingLabelUri:    ptr("https://downloads.example.com/return-label"),
		IsImportRequested:         common.Bool(spec.IsImportRequested),
		ImportCompartmentId:       pointerOrNil(spec.ImportCompartmentId),
		ImportFileBucket:          pointerOrNil(spec.ImportFileBucket),
		DataValidationCode:        pointerOrNil(spec.DataValidationCode),
		ImageExportPar:            ptr("https://downloads.example.com/par"),
		MasterKeyId:               pointerOrNil(spec.MasterKeyId),
		CertificateAuthorityId:    pointerOrNil(spec.CertificateAuthorityId),
		TimeCertValidityEnd:       sdkTimeOrNil(spec.TimeCertValidityEnd),
		CommonName:                pointerOrNil(spec.CommonName),
		CertCompartmentId:         pointerOrNil(spec.CertCompartmentId),
		CertificateVersionNumber:  ptr("1"),
		CertificateId:             ptr("ocid1.certificate.oc1..leaf"),
		CertKeyAlgorithm:          roversdk.CertKeyAlgorithmEnum(spec.CertKeyAlgorithm),
		CertSignatureAlgorithm:    roversdk.CertSignatureAlgorithmEnum(spec.CertSignatureAlgorithm),
		Tags:                      ptr("alpha"),
		FreeformTags:              cloneStringMap(spec.FreeformTags),
		DefinedTags:               definedTags,
		SystemTags:                roverNodeSystemTagsForTest(),
	}
}

func observedRoverNodeSummaryFromSpec(
	id string,
	spec roverv1beta1.RoverNodeSpec,
	lifecycleState string,
) roversdk.RoverNodeSummary {
	now := &common.SDKTime{Time: time.Unix(1713240000, 0).UTC()}

	return roversdk.RoverNodeSummary{
		Id:                    common.String(id),
		CompartmentId:         common.String(spec.CompartmentId),
		LifecycleState:        roversdk.LifecycleStateEnum(lifecycleState),
		SerialNumber:          pointerOrNil(spec.SerialNumber),
		NodeType:              roversdk.NodeTypeStandalone,
		Shape:                 pointerOrNil(spec.Shape),
		DisplayName:           common.String(spec.DisplayName),
		TimeCreated:           now,
		LifecycleStateDetails: ptr("summary lifecycle details"),
		FreeformTags:          cloneStringMap(spec.FreeformTags),
		DefinedTags:           map[string]map[string]interface{}{},
		SystemTags:            roverNodeSystemTagsForTest(),
	}
}

func roverNodeSystemTagsForTest() map[string]map[string]interface{} {
	return map[string]map[string]interface{}{
		"orcl-cloud": {
			"free-tier-retained": "true",
		},
	}
}

func requireRoverNodeOpcRequestID(t *testing.T, resource *roverv1beta1.RoverNode, want string) {
	t.Helper()

	if got := resource.Status.OsokStatus.OpcRequestID; got != want {
		t.Fatalf("status.opcRequestId = %q, want %q", got, want)
	}
}

func requireRoverNodeAsyncCurrent(
	t *testing.T,
	resource *roverv1beta1.RoverNode,
	phase shared.OSOKAsyncPhase,
	rawStatus string,
	class shared.OSOKAsyncNormalizedClass,
	workRequestID string,
) {
	t.Helper()

	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want lifecycle tracker")
	}
	if current.Source != shared.OSOKAsyncSourceLifecycle {
		t.Fatalf("status.async.current.source = %q, want %q", current.Source, shared.OSOKAsyncSourceLifecycle)
	}
	if current.Phase != phase {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, phase)
	}
	if current.WorkRequestID != workRequestID {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", current.WorkRequestID, workRequestID)
	}
	if current.RawStatus != rawStatus {
		t.Fatalf("status.async.current.rawStatus = %q, want %q", current.RawStatus, rawStatus)
	}
	if current.NormalizedClass != class {
		t.Fatalf("status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, class)
	}
}

func pointerOrNil(value string) *string {
	if value == "" {
		return nil
	}
	return common.String(value)
}

func sdkTimeOrNil(value string) *common.SDKTime {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return &common.SDKTime{Time: parsed.UTC()}
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	return maps.Clone(values)
}

func equalStringSlices(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func equalRequestFields(left []generatedruntime.RequestField, right []generatedruntime.RequestField) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i].FieldName != right[i].FieldName ||
			left[i].RequestName != right[i].RequestName ||
			left[i].Contribution != right[i].Contribution ||
			left[i].PreferResourceID != right[i].PreferResourceID ||
			!equalStringSlices(left[i].LookupPaths, right[i].LookupPaths) {
			return false
		}
	}
	return true
}

func ptr[T any](value T) *T {
	return &value
}

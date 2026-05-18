package rovercluster

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

func TestApplyRoverClusterRuntimeHooksOverridesGeneratedDefaults(t *testing.T) {
	t.Parallel()

	hooks := newRoverClusterDefaultRuntimeHooks(roversdk.RoverClusterClient{})
	applyRoverClusterRuntimeHooks(&hooks)

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed semantics")
	}
	if len(hooks.Semantics.AuxiliaryOperations) != 0 {
		t.Fatalf("hooks.Semantics.AuxiliaryOperations = %#v, want none", hooks.Semantics.AuxiliaryOperations)
	}
	if got, want := hooks.Semantics.List.MatchFields, []string{"clusterSize", "clusterType", "compartmentId", "displayName"}; !equalStringSlices(got, want) {
		t.Fatalf("hooks.Semantics.List.MatchFields = %#v, want %#v", got, want)
	}
	if got, want := hooks.List.Fields, reviewedRoverClusterListFields(); !equalRequestFields(got, want) {
		t.Fatalf("hooks.List.Fields = %#v, want %#v", got, want)
	}
	if hooks.ParityHooks.NormalizeDesiredState == nil {
		t.Fatal("hooks.ParityHooks.NormalizeDesiredState = nil, want provider-managed field normalization")
	}
	if hooks.BuildUpdateBody == nil {
		t.Fatal("hooks.BuildUpdateBody = nil, want reviewed update builder")
	}

	resource := newRoverClusterTestResource()
	resource.Spec.LifecycleState = "ACTIVE"
	resource.Spec.LifecycleStateDetails = "provider-managed"
	resource.Spec.SystemTags = map[string]shared.MapValue{
		"orcl-cloud": {
			"free-tier-retained": "true",
		},
	}
	resource.Spec.ClusterWorkloads[0].WorkRequestId = "wr-custom-image-export"

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
	if got := resource.Spec.ClusterWorkloads[0].WorkRequestId; got != "" {
		t.Fatalf("resource.Spec.ClusterWorkloads[0].WorkRequestId = %q, want empty string", got)
	}
}

func TestBuildRoverClusterUpdateBodySupportsClearingOptionalFields(t *testing.T) {
	t.Parallel()

	currentSpec := newRoverClusterTestResource().Spec
	resource := newRoverClusterTestResource()
	resource.Spec.PointOfContact = ""
	resource.Spec.PointOfContactPhoneNumber = ""
	resource.Spec.DataValidationCode = ""
	resource.Spec.SuperUserPassword = ""
	resource.Spec.UnlockPassphrase = ""
	resource.Spec.FreeformTags = map[string]string{}
	resource.Spec.DefinedTags = map[string]shared.MapValue{}

	current := observedRoverClusterFromSpec("ocid1.rovercluster.oc1..existing", currentSpec, "ACTIVE")

	details, updateNeeded, err := buildRoverClusterUpdateBody(resource, current)
	if err != nil {
		t.Fatalf("buildRoverClusterUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildRoverClusterUpdateBody() updateNeeded = false, want clear intent")
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

	body := roverClusterSerializedRequestBody(t, roversdk.UpdateRoverClusterRequest{
		RoverClusterId:            common.String("ocid1.rovercluster.oc1..existing"),
		UpdateRoverClusterDetails: details,
	}, http.MethodPut, "/roverClusters/ocid1.rovercluster.oc1..existing")

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

func TestRoverClusterCreateOrUpdateCreatesWhenListCandidateClusterSizeDiffers(t *testing.T) {
	t.Parallel()

	resource := newRoverClusterTestResource()
	resource.Spec.LifecycleState = "ACTIVE"
	resource.Spec.LifecycleStateDetails = "provider-managed"
	resource.Spec.SystemTags = map[string]shared.MapValue{
		"orcl-cloud": {
			"free-tier-retained": "true",
		},
	}
	resource.Spec.ClusterWorkloads[0].WorkRequestId = "wr-custom-image-export"

	createSpec := resource.Spec
	mismatchedSpec := resource.Spec
	mismatchedSpec.ClusterSize++

	defaultHooks := newRoverClusterDefaultRuntimeHooks(roversdk.RoverClusterClient{})

	var createRequest roversdk.CreateRoverClusterRequest
	createCalled := false
	listCalled := false

	manager := newRoverClusterRuntimeTestManager(generatedruntime.Config[*roverv1beta1.RoverCluster]{
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &roversdk.ListRoverClustersRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), reviewedRoverClusterListFields()...),
			Call: func(_ context.Context, request any) (any, error) {
				listCalled = true
				listRequest := *request.(*roversdk.ListRoverClustersRequest)
				if listRequest.LifecycleState != "" {
					t.Fatalf("ListRoverClustersRequest.LifecycleState = %q, want empty string after reviewed list-field override", listRequest.LifecycleState)
				}
				return roversdk.ListRoverClustersResponse{
					RoverClusterCollection: roversdk.RoverClusterCollection{
						Items: []roversdk.RoverClusterSummary{
							observedRoverClusterSummaryFromSpec("ocid1.rovercluster.oc1..existing", mismatchedSpec, "ACTIVE"),
						},
					},
				}, nil
			},
		},
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &roversdk.CreateRoverClusterRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), defaultHooks.Create.Fields...),
			Call: func(_ context.Context, request any) (any, error) {
				createCalled = true
				createRequest = *request.(*roversdk.CreateRoverClusterRequest)
				return roversdk.CreateRoverClusterResponse{
					RoverCluster: observedRoverClusterFromSpec("ocid1.rovercluster.oc1..created", createSpec, "CREATING"),
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
		t.Fatal("Create() should be called when the only list candidate mismatches clusterSize")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = false, want true while lifecycle remains CREATING")
	}

	requireRoverClusterOpcRequestID(t, resource, "opc-create")
	requireRoverClusterAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, "CREATING", shared.OSOKAsyncClassPending, "")

	body := roverClusterSerializedRequestBody(t, createRequest, http.MethodPost, "/roverClusters")
	for _, unwanted := range []string{`"lifecycleState"`, `"lifecycleStateDetails"`, `"systemTags"`, `"workRequestId"`} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("request body %s unexpectedly contains %s", body, unwanted)
		}
	}
}

func TestRoverClusterCreateOrUpdateUpdatesUniqueExactSummaryMatch(t *testing.T) {
	t.Parallel()

	currentSpec := newRoverClusterTestResource().Spec
	resource := newRoverClusterTestResource()
	resource.Spec.PointOfContact = "Updated Contact"

	defaultHooks := newRoverClusterDefaultRuntimeHooks(roversdk.RoverClusterClient{})

	var updateRequest roversdk.UpdateRoverClusterRequest
	createCalled := false
	updateCalled := false
	getCalls := 0

	manager := newRoverClusterRuntimeTestManager(generatedruntime.Config[*roverv1beta1.RoverCluster]{
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &roversdk.ListRoverClustersRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), reviewedRoverClusterListFields()...),
			Call: func(_ context.Context, request any) (any, error) {
				listRequest := *request.(*roversdk.ListRoverClustersRequest)
				if listRequest.LifecycleState != "" {
					t.Fatalf("ListRoverClustersRequest.LifecycleState = %q, want empty string after reviewed list-field override", listRequest.LifecycleState)
				}
				return roversdk.ListRoverClustersResponse{
					RoverClusterCollection: roversdk.RoverClusterCollection{
						Items: []roversdk.RoverClusterSummary{
							observedRoverClusterSummaryFromSpec("ocid1.rovercluster.oc1..existing", currentSpec, "ACTIVE"),
						},
					},
				}, nil
			},
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &roversdk.GetRoverClusterRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), defaultHooks.Get.Fields...),
			Call: func(_ context.Context, request any) (any, error) {
				getCalls++
				getRequest := *request.(*roversdk.GetRoverClusterRequest)
				if getRequest.RoverClusterId == nil || *getRequest.RoverClusterId != "ocid1.rovercluster.oc1..existing" {
					t.Fatalf("GetRoverClusterRequest.RoverClusterId = %#v, want existing rover cluster ID", getRequest.RoverClusterId)
				}
				lifecycleState := "ACTIVE"
				if getCalls > 1 {
					lifecycleState = "UPDATING"
				}
				return roversdk.GetRoverClusterResponse{
					RoverCluster: observedRoverClusterFromSpec("ocid1.rovercluster.oc1..existing", currentSpec, lifecycleState),
					OpcRequestId: common.String("opc-get"),
				}, nil
			},
		},
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &roversdk.CreateRoverClusterRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), defaultHooks.Create.Fields...),
			Call: func(_ context.Context, _ any) (any, error) {
				createCalled = true
				return nil, nil
			},
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &roversdk.UpdateRoverClusterRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), defaultHooks.Update.Fields...),
			Call: func(_ context.Context, request any) (any, error) {
				updateCalled = true
				updateRequest = *request.(*roversdk.UpdateRoverClusterRequest)
				return roversdk.UpdateRoverClusterResponse{
					RoverCluster: observedRoverClusterFromSpec("ocid1.rovercluster.oc1..existing", resource.Spec, "UPDATING"),
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
		t.Fatal("Update() should be called when reviewed mutable drift exists on the reused cluster")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = false, want true while lifecycle remains UPDATING")
	}
	if updateRequest.UpdateRoverClusterDetails.PointOfContact == nil || *updateRequest.UpdateRoverClusterDetails.PointOfContact != resource.Spec.PointOfContact {
		t.Fatalf("UpdateRoverClusterDetails.PointOfContact = %#v, want %q", updateRequest.UpdateRoverClusterDetails.PointOfContact, resource.Spec.PointOfContact)
	}
	if updateRequest.UpdateRoverClusterDetails.DisplayName != nil {
		t.Fatalf("UpdateRoverClusterDetails.DisplayName = %#v, want nil for unchanged field", updateRequest.UpdateRoverClusterDetails.DisplayName)
	}

	requireRoverClusterOpcRequestID(t, resource, "opc-update")
	requireRoverClusterAsyncCurrent(t, resource, shared.OSOKAsyncPhaseUpdate, "UPDATING", shared.OSOKAsyncClassPending, "")
}

type roverClusterRequestBodyBuilder interface {
	HTTPRequest(string, string, *common.OCIReadSeekCloser, map[string]string) (http.Request, error)
}

func roverClusterSerializedRequestBody(
	t *testing.T,
	request roverClusterRequestBodyBuilder,
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

func newRoverClusterRuntimeTestManager(
	cfg generatedruntime.Config[*roverv1beta1.RoverCluster],
) *RoverClusterServiceManager {
	if cfg.Kind == "" {
		cfg.Kind = "RoverCluster"
	}
	if cfg.SDKName == "" {
		cfg.SDKName = "RoverCluster"
	}
	if cfg.Semantics == nil {
		cfg.Semantics = reviewedRoverClusterRuntimeSemantics()
	}
	if cfg.ParityHooks.NormalizeDesiredState == nil {
		cfg.ParityHooks.NormalizeDesiredState = normalizeRoverClusterDesiredState
	}
	if cfg.BuildUpdateBody == nil {
		cfg.BuildUpdateBody = func(
			_ context.Context,
			resource *roverv1beta1.RoverCluster,
			_ string,
			currentResponse any,
		) (any, bool, error) {
			return buildRoverClusterUpdateBody(resource, currentResponse)
		}
	}

	return &RoverClusterServiceManager{
		client: defaultRoverClusterServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*roverv1beta1.RoverCluster](cfg),
		},
	}
}

func newRoverClusterTestResource() *roverv1beta1.RoverCluster {
	return &roverv1beta1.RoverCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rovercluster-sample",
			Namespace: "default",
		},
		Spec: roverv1beta1.RoverClusterSpec{
			DisplayName:   "rovercluster-sample",
			CompartmentId: "ocid1.compartment.oc1..example",
			ClusterSize:   5,
			CustomerShippingAddress: roverv1beta1.RoverClusterCustomerShippingAddress{
				Addressee:      "Alice Example",
				Address1:       "1 Main Street",
				CityOrLocality: "Austin",
				StateOrRegion:  "TX",
				Zipcode:        "78701",
				Country:        "US",
				PhoneNumber:    "555-0100",
				Email:          "alice@example.com",
			},
			ClusterWorkloads: []roverv1beta1.RoverClusterClusterWorkload{
				{
					CompartmentId: "ocid1.compartment.oc1..workload",
					Id:            "ocid1.workload.oc1..example",
					WorkloadType:  "OBJECT_STORAGE",
					Name:          "sample-workload",
					Size:          "25TB",
				},
			},
			ClusterType:               "STANDALONE",
			SuperUserPassword:         "TempPassw0rd!",
			EnclosureType:             "RUGGADIZED",
			UnlockPassphrase:          "Unlock123!",
			PointOfContact:            "Alice Example",
			PointOfContactPhoneNumber: "555-0100",
			ShippingPreference:        "ORACLE_SHIPPED",
			ShippingVendor:            "UPS",
			TimePickupExpected:        "2026-05-01T12:00:00Z",
			OracleShippingTrackingUrl: "https://tracking.example.com/rovercluster",
			SubscriptionId:            "subscription-123",
			IsImportRequested:         true,
			ImportCompartmentId:       "ocid1.compartment.oc1..import",
			ImportFileBucket:          "import-bucket",
			DataValidationCode:        "validate-123",
			MasterKeyId:               "ocid1.key.oc1..master",
			FreeformTags:              map[string]string{"env": "test"},
			DefinedTags:               map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func observedRoverClusterFromSpec(
	id string,
	spec roverv1beta1.RoverClusterSpec,
	lifecycleState string,
) roversdk.RoverCluster {
	now := &common.SDKTime{Time: time.Unix(1713240000, 0).UTC()}

	address, err := roverClusterShippingAddressFromSpec(spec.CustomerShippingAddress)
	if err != nil {
		panic(err)
	}
	workloads, err := roverClusterWorkloadsFromSpec(spec.ClusterWorkloads)
	if err != nil {
		panic(err)
	}
	definedTags, err := roverClusterDefinedTagsFromSpec(spec.DefinedTags)
	if err != nil {
		panic(err)
	}

	return roversdk.RoverCluster{
		Id:                        common.String(id),
		CompartmentId:             common.String(spec.CompartmentId),
		DisplayName:               common.String(spec.DisplayName),
		ClusterSize:               common.Int(spec.ClusterSize),
		LifecycleState:            roversdk.LifecycleStateEnum(lifecycleState),
		TimeCreated:               now,
		LifecycleStateDetails:     ptr("lifecycle details"),
		CustomerShippingAddress:   address,
		Nodes:                     nil,
		EnclosureType:             roversdk.EnclosureTypeEnum(spec.EnclosureType),
		TimeCustomerReceived:      now,
		TimeCustomerReturned:      now,
		DeliveryTrackingInfo:      ptr("tracking-123"),
		ClusterWorkloads:          workloads,
		ClusterType:               roversdk.ClusterTypeEnum(spec.ClusterType),
		SubscriptionId:            pointerOrNil(spec.SubscriptionId),
		ExteriorDoorCode:          ptr("1111"),
		InteriorAlarmDisarmCode:   ptr("2222"),
		SuperUserPassword:         pointerOrNil(spec.SuperUserPassword),
		UnlockPassphrase:          pointerOrNil(spec.UnlockPassphrase),
		PointOfContact:            pointerOrNil(spec.PointOfContact),
		PointOfContactPhoneNumber: pointerOrNil(spec.PointOfContactPhoneNumber),
		ShippingPreference:        roversdk.RoverClusterShippingPreferenceEnum(spec.ShippingPreference),
		OracleShippingTrackingUrl: pointerOrNil(spec.OracleShippingTrackingUrl),
		ShippingVendor:            pointerOrNil(spec.ShippingVendor),
		TimePickupExpected:        sdkTimeOrNil(spec.TimePickupExpected),
		TimeReturnWindowStarts:    now,
		TimeReturnWindowEnds:      now,
		ReturnShippingLabelUri:    ptr("https://downloads.example.com/return-label"),
		IsImportRequested:         common.Bool(spec.IsImportRequested),
		ImportCompartmentId:       pointerOrNil(spec.ImportCompartmentId),
		ImportFileBucket:          pointerOrNil(spec.ImportFileBucket),
		DataValidationCode:        pointerOrNil(spec.DataValidationCode),
		ImageExportPar:            ptr("https://downloads.example.com/par"),
		MasterKeyId:               pointerOrNil(spec.MasterKeyId),
		FreeformTags:              cloneStringMap(spec.FreeformTags),
		DefinedTags:               definedTags,
		SystemTags:                roverClusterSystemTagsForTest(),
	}
}

func observedRoverClusterSummaryFromSpec(
	id string,
	spec roverv1beta1.RoverClusterSpec,
	lifecycleState string,
) roversdk.RoverClusterSummary {
	now := &common.SDKTime{Time: time.Unix(1713240000, 0).UTC()}

	return roversdk.RoverClusterSummary{
		Id:                    common.String(id),
		CompartmentId:         common.String(spec.CompartmentId),
		LifecycleState:        roversdk.LifecycleStateEnum(lifecycleState),
		DisplayName:           common.String(spec.DisplayName),
		TimeCreated:           now,
		ClusterSize:           common.Int(spec.ClusterSize),
		ClusterType:           roversdk.ClusterTypeEnum(spec.ClusterType),
		LifecycleStateDetails: ptr("summary lifecycle details"),
		FreeformTags:          cloneStringMap(spec.FreeformTags),
		DefinedTags:           map[string]map[string]interface{}{},
		SystemTags:            roverClusterSystemTagsForTest(),
	}
}

func roverClusterSystemTagsForTest() map[string]map[string]interface{} {
	return map[string]map[string]interface{}{
		"orcl-cloud": {
			"free-tier-retained": "true",
		},
	}
}

func requireRoverClusterOpcRequestID(t *testing.T, resource *roverv1beta1.RoverCluster, want string) {
	t.Helper()

	if got := resource.Status.OsokStatus.OpcRequestID; got != want {
		t.Fatalf("status.opcRequestId = %q, want %q", got, want)
	}
}

func requireRoverClusterAsyncCurrent(
	t *testing.T,
	resource *roverv1beta1.RoverCluster,
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

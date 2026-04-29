package analyticsinstance

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	analyticssdk "github.com/oracle/oci-go-sdk/v65/analytics"
	"github.com/oracle/oci-go-sdk/v65/common"
	analyticsv1beta1 "github.com/oracle/oci-service-operator/api/analytics/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestAnalyticsInstanceCreateRequestPreservesPolymorphicNetworkEndpointDetails(t *testing.T) {
	t.Parallel()

	resource := newAnalyticsInstanceTestResource()
	resource.Spec.NetworkEndpointDetails = analyticsv1beta1.AnalyticsInstanceNetworkEndpointDetails{
		NetworkEndpointType: "PUBLIC",
		WhitelistedIps:      []string{"10.0.0.0/24"},
		WhitelistedVcns: []analyticsv1beta1.AnalyticsInstanceNetworkEndpointDetailsWhitelistedVcn{
			{
				Id:             "ocid1.vcn.oc1..allowed",
				WhitelistedIps: []string{"10.0.1.0/24"},
			},
		},
		WhitelistedServices: []string{"ALL"},
	}

	var createRequest analyticssdk.CreateAnalyticsInstanceRequest

	manager := newAnalyticsInstanceRuntimeTestManager(generatedruntime.Config[*analyticsv1beta1.AnalyticsInstance]{
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &analyticssdk.CreateAnalyticsInstanceRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				createRequest = *request.(*analyticssdk.CreateAnalyticsInstanceRequest)
				return analyticssdk.CreateAnalyticsInstanceResponse{
					AnalyticsInstance: observedAnalyticsInstanceFromSpec(
						"ocid1.analyticsinstance.oc1..created",
						resource.Spec,
						"ACTIVE",
					),
				}, nil
			},
			Fields: analyticsInstanceCreateFields(),
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}

	publicDetails, ok := createRequest.CreateAnalyticsInstanceDetails.NetworkEndpointDetails.(analyticssdk.PublicEndpointDetails)
	if !ok {
		t.Fatalf("network endpoint details type = %T, want analytics.PublicEndpointDetails", createRequest.CreateAnalyticsInstanceDetails.NetworkEndpointDetails)
	}
	if len(publicDetails.WhitelistedIps) != 1 || publicDetails.WhitelistedIps[0] != "10.0.0.0/24" {
		t.Fatalf("WhitelistedIps = %#v, want public ingress rule preserved", publicDetails.WhitelistedIps)
	}
	if len(publicDetails.WhitelistedVcns) != 1 || publicDetails.WhitelistedVcns[0].Id == nil || *publicDetails.WhitelistedVcns[0].Id != "ocid1.vcn.oc1..allowed" {
		t.Fatalf("WhitelistedVcns = %#v, want VCN allowlist preserved", publicDetails.WhitelistedVcns)
	}
	if len(publicDetails.WhitelistedServices) != 1 || publicDetails.WhitelistedServices[0] != analyticssdk.AccessControlServiceTypeAll {
		t.Fatalf("WhitelistedServices = %#v, want ALL service preserved", publicDetails.WhitelistedServices)
	}

	body := analyticsInstanceSerializedRequestBody(t, createRequest, http.MethodPost, "/analyticsInstances")
	for _, want := range []string{
		`"networkEndpointType":"PUBLIC"`,
		`"whitelistedIps":["10.0.0.0/24"]`,
		`"whitelistedServices":["ALL"]`,
	} {
		if !contains(body, want) {
			t.Fatalf("request body %s does not contain %s", body, want)
		}
	}
	if contains(body, `"jsonData"`) {
		t.Fatalf("request body unexpectedly exposed jsonData helper field: %s", body)
	}
}

func TestApplyAnalyticsInstanceRuntimeHooksOverridesGeneratedDefaults(t *testing.T) {
	t.Parallel()

	hooks := newAnalyticsInstanceDefaultRuntimeHooks(analyticssdk.AnalyticsClient{})
	applyAnalyticsInstanceRuntimeHooks(&hooks)

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed semantics")
	}
	if len(hooks.Semantics.AuxiliaryOperations) != 0 {
		t.Fatalf("hooks.Semantics.AuxiliaryOperations = %#v, want cleared reviewed semantics", hooks.Semantics.AuxiliaryOperations)
	}
	if len(hooks.Semantics.Lifecycle.UpdatingStates) != 1 || hooks.Semantics.Lifecycle.UpdatingStates[0] != "UPDATING" {
		t.Fatalf("hooks.Semantics.Lifecycle.UpdatingStates = %#v, want [\"UPDATING\"]", hooks.Semantics.Lifecycle.UpdatingStates)
	}
	if !containsStringSlice(hooks.Semantics.Mutation.Mutable, "updateChannel") {
		t.Fatalf("hooks.Semantics.Mutation.Mutable = %#v, want updateChannel in reviewed mutable surface", hooks.Semantics.Mutation.Mutable)
	}
	if hooks.ParityHooks.NormalizeDesiredState == nil {
		t.Fatal("hooks.ParityHooks.NormalizeDesiredState = nil, want create-only normalization hook")
	}
	if hooks.BuildUpdateBody == nil {
		t.Fatal("hooks.BuildUpdateBody = nil, want reviewed update builder")
	}

	resource := newAnalyticsInstanceTestResource()
	resource.Spec.Description = "updated analytics description"

	body, updateNeeded, err := hooks.BuildUpdateBody(
		context.Background(),
		resource,
		resource.Namespace,
		observedAnalyticsInstanceFromSpec(
			"ocid1.analyticsinstance.oc1..existing",
			newAnalyticsInstanceTestResource().Spec,
			"ACTIVE",
		),
	)
	if err != nil {
		t.Fatalf("hooks.BuildUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("hooks.BuildUpdateBody() updateNeeded = false, want true after reviewed drift")
	}

	details, ok := body.(analyticssdk.UpdateAnalyticsInstanceDetails)
	if !ok {
		t.Fatalf("hooks.BuildUpdateBody() body type = %T, want analytics.UpdateAnalyticsInstanceDetails", body)
	}
	if details.Description == nil || *details.Description != resource.Spec.Description {
		t.Fatalf("hooks.BuildUpdateBody() description = %#v, want %q", details.Description, resource.Spec.Description)
	}
}

func TestAnalyticsInstanceCreateRequestIncludesRefreshedContractFields(t *testing.T) {
	t.Parallel()

	resource := newAnalyticsInstanceTestResource()
	resource.Spec.UpdateChannel = "EARLY"
	resource.Spec.DomainId = "ocid1.domain.oc1..example"
	resource.Spec.AdminUser = "analytics.admin@example.com"
	resource.Spec.FeatureBundle = "FAW_FREE"

	var createRequest analyticssdk.CreateAnalyticsInstanceRequest

	manager := newAnalyticsInstanceRuntimeTestManager(generatedruntime.Config[*analyticsv1beta1.AnalyticsInstance]{
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &analyticssdk.CreateAnalyticsInstanceRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				createRequest = *request.(*analyticssdk.CreateAnalyticsInstanceRequest)
				return analyticssdk.CreateAnalyticsInstanceResponse{
					AnalyticsInstance: observedAnalyticsInstanceFromSpec(
						"ocid1.analyticsinstance.oc1..created",
						resource.Spec,
						"ACTIVE",
					),
				}, nil
			},
			Fields: analyticsInstanceCreateFields(),
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}

	if got := createRequest.CreateAnalyticsInstanceDetails.UpdateChannel; got != analyticssdk.UpdateChannelEarly {
		t.Fatalf("CreateAnalyticsInstanceDetails.UpdateChannel = %q, want %q", got, analyticssdk.UpdateChannelEarly)
	}
	if createRequest.CreateAnalyticsInstanceDetails.DomainId == nil || *createRequest.CreateAnalyticsInstanceDetails.DomainId != resource.Spec.DomainId {
		t.Fatalf("CreateAnalyticsInstanceDetails.DomainId = %#v, want %q", createRequest.CreateAnalyticsInstanceDetails.DomainId, resource.Spec.DomainId)
	}
	if createRequest.CreateAnalyticsInstanceDetails.AdminUser == nil || *createRequest.CreateAnalyticsInstanceDetails.AdminUser != resource.Spec.AdminUser {
		t.Fatalf("CreateAnalyticsInstanceDetails.AdminUser = %#v, want %q", createRequest.CreateAnalyticsInstanceDetails.AdminUser, resource.Spec.AdminUser)
	}
	if got := createRequest.CreateAnalyticsInstanceDetails.FeatureBundle; got != analyticssdk.FeatureBundleFawFree {
		t.Fatalf("CreateAnalyticsInstanceDetails.FeatureBundle = %q, want %q", got, analyticssdk.FeatureBundleFawFree)
	}

	body := analyticsInstanceSerializedRequestBody(t, createRequest, http.MethodPost, "/analyticsInstances")
	for _, want := range []string{
		`"updateChannel":"EARLY"`,
		`"domainId":"ocid1.domain.oc1..example"`,
		`"adminUser":"analytics.admin@example.com"`,
		`"featureBundle":"FAW_FREE"`,
	} {
		if !contains(body, want) {
			t.Fatalf("request body %s does not contain %s", body, want)
		}
	}
}

func TestBuildAnalyticsInstanceUpdateBodySupportsClearingOptionalFields(t *testing.T) {
	t.Parallel()

	resource := newAnalyticsInstanceTestResource()
	resource.Spec.Description = ""
	resource.Spec.EmailNotification = ""
	resource.Spec.FreeformTags = map[string]string{}
	resource.Spec.DefinedTags = map[string]shared.MapValue{}

	current := observedAnalyticsInstanceFromSpec("ocid1.analyticsinstance.oc1..existing", newAnalyticsInstanceTestResource().Spec, "ACTIVE")

	details, updateNeeded, err := buildAnalyticsInstanceUpdateBody(resource, current)
	if err != nil {
		t.Fatalf("buildAnalyticsInstanceUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildAnalyticsInstanceUpdateBody() updateNeeded = false, want clear intent")
	}
	if details.Description == nil || *details.Description != "" {
		t.Fatalf("Description = %#v, want explicit empty string clear", details.Description)
	}
	if details.EmailNotification == nil || *details.EmailNotification != "" {
		t.Fatalf("EmailNotification = %#v, want explicit empty string clear", details.EmailNotification)
	}
	if details.FreeformTags == nil || len(details.FreeformTags) != 0 {
		t.Fatalf("FreeformTags = %#v, want explicit empty map clear", details.FreeformTags)
	}
	if details.DefinedTags == nil || len(details.DefinedTags) != 0 {
		t.Fatalf("DefinedTags = %#v, want explicit empty map clear", details.DefinedTags)
	}

	body := analyticsInstanceSerializedRequestBody(t, analyticssdk.UpdateAnalyticsInstanceRequest{
		AnalyticsInstanceId:            common.String("ocid1.analyticsinstance.oc1..existing"),
		UpdateAnalyticsInstanceDetails: details,
	}, http.MethodPut, "/analyticsInstances/ocid1.analyticsinstance.oc1..existing")

	for _, want := range []string{
		`"description":""`,
		`"emailNotification":""`,
		`"freeformTags":{}`,
		`"definedTags":{}`,
	} {
		if !contains(body, want) {
			t.Fatalf("request body %s does not contain %s", body, want)
		}
	}
}

func TestBuildAnalyticsInstanceUpdateBodyIncludesUpdateChannel(t *testing.T) {
	t.Parallel()

	resource := newAnalyticsInstanceTestResource()
	resource.Spec.UpdateChannel = "EARLY"

	currentSpec := newAnalyticsInstanceTestResource().Spec
	currentSpec.UpdateChannel = "REGULAR"
	current := observedAnalyticsInstanceFromSpec("ocid1.analyticsinstance.oc1..existing", currentSpec, "ACTIVE")

	details, updateNeeded, err := buildAnalyticsInstanceUpdateBody(resource, current)
	if err != nil {
		t.Fatalf("buildAnalyticsInstanceUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildAnalyticsInstanceUpdateBody() updateNeeded = false, want updateChannel drift")
	}
	if details.UpdateChannel != analyticssdk.UpdateChannelEarly {
		t.Fatalf("UpdateChannel = %q, want %q", details.UpdateChannel, analyticssdk.UpdateChannelEarly)
	}

	body := analyticsInstanceSerializedRequestBody(t, analyticssdk.UpdateAnalyticsInstanceRequest{
		AnalyticsInstanceId:            common.String("ocid1.analyticsinstance.oc1..existing"),
		UpdateAnalyticsInstanceDetails: details,
	}, http.MethodPut, "/analyticsInstances/ocid1.analyticsinstance.oc1..existing")
	if !contains(body, `"updateChannel":"EARLY"`) {
		t.Fatalf("request body %s does not contain updateChannel update", body)
	}
}

func TestAnalyticsInstanceCreatePendingProjectsSharedAsyncBreadcrumbs(t *testing.T) {
	t.Parallel()

	const createdID = "ocid1.analyticsinstance.oc1..created"

	resource := newAnalyticsInstanceTestResource()

	manager := newAnalyticsInstanceRuntimeTestManager(generatedruntime.Config[*analyticsv1beta1.AnalyticsInstance]{
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &analyticssdk.CreateAnalyticsInstanceRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				return analyticssdk.CreateAnalyticsInstanceResponse{
					AnalyticsInstance: observedAnalyticsInstanceFromSpec(createdID, resource.Spec, "CREATING"),
					OpcRequestId:      common.String("opc-create-1"),
					OpcWorkRequestId:  common.String("wr-create-1"),
				}, nil
			},
			Fields: analyticsInstanceCreateFields(),
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success while create response stays CREATING")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should keep requeueing while create response stays CREATING")
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != createdID {
		t.Fatalf("status.ocid = %q, want %q", got, createdID)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Provisioning) {
		t.Fatalf("status.reason = %q, want %q", got, shared.Provisioning)
	}
	requireAnalyticsInstanceOpcRequestID(t, resource, "opc-create-1")
	requireAnalyticsInstanceAsyncCurrent(
		t,
		resource,
		shared.OSOKAsyncPhaseCreate,
		"CREATING",
		shared.OSOKAsyncClassPending,
		"wr-create-1",
	)
}

func TestAnalyticsInstanceUpdatePendingProjectsSharedAsyncBreadcrumbs(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.analyticsinstance.oc1..existing"

	resource := newExistingAnalyticsInstanceTestResource(existingID)
	resource.Spec.Description = "updated analytics description"
	var updateRequest analyticssdk.UpdateAnalyticsInstanceRequest

	manager := newAnalyticsInstanceRuntimeTestManager(generatedruntime.Config[*analyticsv1beta1.AnalyticsInstance]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &analyticssdk.GetAnalyticsInstanceRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getRequest := request.(*analyticssdk.GetAnalyticsInstanceRequest)
				if getRequest.AnalyticsInstanceId == nil || *getRequest.AnalyticsInstanceId != existingID {
					t.Fatalf("GetAnalyticsInstanceRequest.AnalyticsInstanceId = %v, want %s", getRequest.AnalyticsInstanceId, existingID)
				}
				return analyticssdk.GetAnalyticsInstanceResponse{
					AnalyticsInstance: observedAnalyticsInstanceFromSpec(
						existingID,
						newAnalyticsInstanceTestResource().Spec,
						"ACTIVE",
					),
				}, nil
			},
			Fields: analyticsInstanceGetFields(),
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &analyticssdk.UpdateAnalyticsInstanceRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				updateRequest = *request.(*analyticssdk.UpdateAnalyticsInstanceRequest)
				return analyticssdk.UpdateAnalyticsInstanceResponse{
					AnalyticsInstance: observedAnalyticsInstanceFromSpec(existingID, resource.Spec, "UPDATING"),
					OpcRequestId:      common.String("opc-update-1"),
				}, nil
			},
			Fields: analyticsInstanceUpdateFields(),
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success while update response stays UPDATING")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should keep requeueing while update response stays UPDATING")
	}
	if updateRequest.AnalyticsInstanceId == nil || *updateRequest.AnalyticsInstanceId != existingID {
		t.Fatalf("UpdateAnalyticsInstanceRequest.AnalyticsInstanceId = %v, want %s", updateRequest.AnalyticsInstanceId, existingID)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Updating) {
		t.Fatalf("status.reason = %q, want %q", got, shared.Updating)
	}
	requireAnalyticsInstanceOpcRequestID(t, resource, "opc-update-1")
	requireAnalyticsInstanceAsyncCurrent(
		t,
		resource,
		shared.OSOKAsyncPhaseUpdate,
		"UPDATING",
		shared.OSOKAsyncClassPending,
		"",
	)
}

func TestAnalyticsInstanceCreateOrUpdateClassifiesReviewedLifecycleStates(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.analyticsinstance.oc1..existing"

	testCases := []struct {
		name        string
		lifecycle   string
		wantReason  string
		wantRequeue bool
		wantAsync   *shared.OSOKAsyncPhase
	}{
		{
			name:        "updating requeues",
			lifecycle:   "UPDATING",
			wantReason:  string(shared.Updating),
			wantRequeue: true,
			wantAsync:   ptr(shared.OSOKAsyncPhaseUpdate),
		},
		{
			name:        "inactive settles active",
			lifecycle:   "INACTIVE",
			wantReason:  string(shared.Active),
			wantRequeue: false,
			wantAsync:   nil,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resource := newExistingAnalyticsInstanceTestResource(existingID)

			manager := newAnalyticsInstanceRuntimeTestManager(generatedruntime.Config[*analyticsv1beta1.AnalyticsInstance]{
				Get: &generatedruntime.Operation{
					NewRequest: func() any { return &analyticssdk.GetAnalyticsInstanceRequest{} },
					Call: func(_ context.Context, request any) (any, error) {
						if request.(*analyticssdk.GetAnalyticsInstanceRequest).AnalyticsInstanceId == nil ||
							*request.(*analyticssdk.GetAnalyticsInstanceRequest).AnalyticsInstanceId != existingID {
							t.Fatalf("get request AnalyticsInstanceId = %v, want %s", request.(*analyticssdk.GetAnalyticsInstanceRequest).AnalyticsInstanceId, existingID)
						}
						return analyticssdk.GetAnalyticsInstanceResponse{
							AnalyticsInstance: observedAnalyticsInstanceFromSpec(existingID, resource.Spec, tc.lifecycle),
						}, nil
					},
					Fields: analyticsInstanceGetFields(),
				},
			})

			response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			if err != nil {
				t.Fatalf("CreateOrUpdate() error = %v", err)
			}
			if !response.IsSuccessful {
				t.Fatal("CreateOrUpdate() should report success")
			}
			if response.ShouldRequeue != tc.wantRequeue {
				t.Fatalf("CreateOrUpdate() ShouldRequeue = %t, want %t", response.ShouldRequeue, tc.wantRequeue)
			}
			if resource.Status.OsokStatus.Reason != tc.wantReason {
				t.Fatalf("status reason = %q, want %q", resource.Status.OsokStatus.Reason, tc.wantReason)
			}
			if resource.Status.LifecycleState != tc.lifecycle {
				t.Fatalf("status lifecycleState = %q, want %q", resource.Status.LifecycleState, tc.lifecycle)
			}
			if tc.wantAsync == nil {
				if resource.Status.OsokStatus.Async.Current != nil {
					t.Fatalf("status.async.current = %#v, want nil for steady INACTIVE state", resource.Status.OsokStatus.Async.Current)
				}
				return
			}
			requireAnalyticsInstanceAsyncCurrent(t, resource, *tc.wantAsync, tc.lifecycle, shared.OSOKAsyncClassPending, "")
		})
	}
}

func TestAnalyticsInstanceCreateOrUpdateRejectsUnsupportedCompartmentDrift(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.analyticsinstance.oc1..existing"

	resource := newExistingAnalyticsInstanceTestResource(existingID)
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..new"

	current := observedAnalyticsInstanceFromSpec(existingID, newAnalyticsInstanceTestResource().Spec, "ACTIVE")

	manager := newAnalyticsInstanceRuntimeTestManager(generatedruntime.Config[*analyticsv1beta1.AnalyticsInstance]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &analyticssdk.GetAnalyticsInstanceRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				return analyticssdk.GetAnalyticsInstanceResponse{AnalyticsInstance: current}, nil
			},
			Fields: analyticsInstanceGetFields(),
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &analyticssdk.UpdateAnalyticsInstanceRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				t.Fatal("Update() should not be called for unsupported compartment drift")
				return nil, nil
			},
			Fields: analyticsInstanceUpdateFields(),
		},
	})

	_, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !contains(err.Error(), "reject unsupported update drift for compartmentId") {
		t.Fatalf("CreateOrUpdate() error = %v, want unsupported compartment drift failure", err)
	}
}

func TestAnalyticsInstanceCreateOrUpdateReusesInactiveListMatch(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.analyticsinstance.oc1..reused"

	resource := newAnalyticsInstanceTestResource()
	createCalled := false

	manager := newAnalyticsInstanceRuntimeTestManager(generatedruntime.Config[*analyticsv1beta1.AnalyticsInstance]{
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &analyticssdk.CreateAnalyticsInstanceRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				createCalled = true
				t.Fatal("Create() should not be called when an INACTIVE list match is reusable")
				return nil, nil
			},
			Fields: analyticsInstanceCreateFields(),
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &analyticssdk.GetAnalyticsInstanceRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				return analyticssdk.GetAnalyticsInstanceResponse{
					AnalyticsInstance: observedAnalyticsInstanceFromSpec(existingID, resource.Spec, "INACTIVE"),
				}, nil
			},
			Fields: analyticsInstanceGetFields(),
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &analyticssdk.ListAnalyticsInstancesRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				return analyticssdk.ListAnalyticsInstancesResponse{
					Items: []analyticssdk.AnalyticsInstanceSummary{
						observedAnalyticsInstanceSummaryFromSpec(existingID, resource.Spec, "INACTIVE"),
					},
				}, nil
			},
			Fields: analyticsInstanceListFields(),
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue once the reusable AnalyticsInstance is INACTIVE")
	}
	if createCalled {
		t.Fatal("Create() was called unexpectedly")
	}
	if resource.Status.OsokStatus.Ocid != shared.OCID(existingID) {
		t.Fatalf("status.ocid = %q, want %q", resource.Status.OsokStatus.Ocid, existingID)
	}
}

func TestAnalyticsInstanceCreateOrUpdateIgnoresUnobservedCreateOnlyFieldsAfterCreate(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.analyticsinstance.oc1..existing"

	resource := newExistingAnalyticsInstanceTestResource(existingID)
	resource.Spec.AdminUser = "analytics.admin@example.com"
	resource.Spec.IdcsAccessToken = "opaque-access-token"
	updateCalled := false

	manager := newAnalyticsInstanceRuntimeTestManager(generatedruntime.Config[*analyticsv1beta1.AnalyticsInstance]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &analyticssdk.GetAnalyticsInstanceRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				if request.(*analyticssdk.GetAnalyticsInstanceRequest).AnalyticsInstanceId == nil ||
					*request.(*analyticssdk.GetAnalyticsInstanceRequest).AnalyticsInstanceId != existingID {
					t.Fatalf("get request AnalyticsInstanceId = %v, want %s", request.(*analyticssdk.GetAnalyticsInstanceRequest).AnalyticsInstanceId, existingID)
				}
				return analyticssdk.GetAnalyticsInstanceResponse{
					AnalyticsInstance: observedAnalyticsInstanceFromSpec(
						existingID,
						newAnalyticsInstanceTestResource().Spec,
						"ACTIVE",
					),
				}, nil
			},
			Fields: analyticsInstanceGetFields(),
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &analyticssdk.UpdateAnalyticsInstanceRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				updateCalled = true
				t.Fatal("Update() should not be called for unobserved create-only fields")
				return nil, nil
			},
			Fields: analyticsInstanceUpdateFields(),
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue for steady ACTIVE state")
	}
	if updateCalled {
		t.Fatal("Update() was called unexpectedly")
	}
	if resource.Status.OsokStatus.Reason != string(shared.Active) {
		t.Fatalf("status reason = %q, want %q", resource.Status.OsokStatus.Reason, shared.Active)
	}
}

func TestAnalyticsInstanceCreateOrUpdateProjectsRefreshedStatusFields(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.analyticsinstance.oc1..existing"

	resource := newExistingAnalyticsInstanceTestResource(existingID)
	resource.Spec.UpdateChannel = "EARLY"
	resource.Spec.DomainId = "ocid1.domain.oc1..example"
	resource.Spec.FeatureBundle = "FAW_FREE"

	manager := newAnalyticsInstanceRuntimeTestManager(generatedruntime.Config[*analyticsv1beta1.AnalyticsInstance]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &analyticssdk.GetAnalyticsInstanceRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				if request.(*analyticssdk.GetAnalyticsInstanceRequest).AnalyticsInstanceId == nil ||
					*request.(*analyticssdk.GetAnalyticsInstanceRequest).AnalyticsInstanceId != existingID {
					t.Fatalf("get request AnalyticsInstanceId = %v, want %s", request.(*analyticssdk.GetAnalyticsInstanceRequest).AnalyticsInstanceId, existingID)
				}
				return analyticssdk.GetAnalyticsInstanceResponse{
					AnalyticsInstance: observedAnalyticsInstanceFromSpec(existingID, resource.Spec, "ACTIVE"),
				}, nil
			},
			Fields: analyticsInstanceGetFields(),
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &analyticssdk.UpdateAnalyticsInstanceRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				t.Fatal("Update() should not be called when refreshed status fields already match")
				return nil, nil
			},
			Fields: analyticsInstanceUpdateFields(),
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if resource.Status.UpdateChannel != resource.Spec.UpdateChannel {
		t.Fatalf("status.updateChannel = %q, want %q", resource.Status.UpdateChannel, resource.Spec.UpdateChannel)
	}
	if resource.Status.DomainId != resource.Spec.DomainId {
		t.Fatalf("status.domainId = %q, want %q", resource.Status.DomainId, resource.Spec.DomainId)
	}
	if resource.Status.FeatureBundle != resource.Spec.FeatureBundle {
		t.Fatalf("status.featureBundle = %q, want %q", resource.Status.FeatureBundle, resource.Spec.FeatureBundle)
	}
	if resource.Status.SystemTags["orcl-cloud"]["key"] != "value" {
		t.Fatalf("status.systemTags = %#v, want orcl-cloud.key=value", resource.Status.SystemTags)
	}
}

func TestAnalyticsInstanceDeletePendingProjectsSharedAsyncBreadcrumbs(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.analyticsinstance.oc1..existing"

	resource := newExistingAnalyticsInstanceTestResource(existingID)
	getCalls := 0
	var deleteRequest analyticssdk.DeleteAnalyticsInstanceRequest

	manager := newAnalyticsInstanceRuntimeTestManager(generatedruntime.Config[*analyticsv1beta1.AnalyticsInstance]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &analyticssdk.GetAnalyticsInstanceRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				getCalls++
				lifecycleState := "ACTIVE"
				if getCalls > 1 {
					lifecycleState = "DELETING"
				}
				return analyticssdk.GetAnalyticsInstanceResponse{
					AnalyticsInstance: observedAnalyticsInstanceFromSpec(existingID, resource.Spec, lifecycleState),
				}, nil
			},
			Fields: analyticsInstanceGetFields(),
		},
		Delete: &generatedruntime.Operation{
			NewRequest: func() any { return &analyticssdk.DeleteAnalyticsInstanceRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				deleteRequest = *request.(*analyticssdk.DeleteAnalyticsInstanceRequest)
				return analyticssdk.DeleteAnalyticsInstanceResponse{
					OpcRequestId:     common.String("opc-delete-1"),
					OpcWorkRequestId: common.String("wr-delete-1"),
				}, nil
			},
			Fields: analyticsInstanceDeleteFields(),
		},
	})

	deleted, err := manager.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want pending delete confirmation while lifecycle is DELETING")
	}
	if deleteRequest.AnalyticsInstanceId == nil || *deleteRequest.AnalyticsInstanceId != existingID {
		t.Fatalf("DeleteAnalyticsInstanceRequest.AnalyticsInstanceId = %v, want %s", deleteRequest.AnalyticsInstanceId, existingID)
	}
	if resource.Status.OsokStatus.Reason != string(shared.Terminating) {
		t.Fatalf("status reason = %q, want %q", resource.Status.OsokStatus.Reason, shared.Terminating)
	}
	if resource.Status.LifecycleState != "DELETING" {
		t.Fatalf("status lifecycleState = %q, want %q", resource.Status.LifecycleState, "DELETING")
	}
	requireAnalyticsInstanceOpcRequestID(t, resource, "opc-delete-1")
	requireAnalyticsInstanceAsyncCurrent(
		t,
		resource,
		shared.OSOKAsyncPhaseDelete,
		"",
		shared.OSOKAsyncClassPending,
		"wr-delete-1",
	)
}

type analyticsInstanceRequestBodyBuilder interface {
	HTTPRequest(string, string, *common.OCIReadSeekCloser, map[string]string) (http.Request, error)
}

func analyticsInstanceSerializedRequestBody(
	t *testing.T,
	request analyticsInstanceRequestBodyBuilder,
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

func newAnalyticsInstanceRuntimeTestManager(
	cfg generatedruntime.Config[*analyticsv1beta1.AnalyticsInstance],
) *AnalyticsInstanceServiceManager {
	if cfg.Kind == "" {
		cfg.Kind = "AnalyticsInstance"
	}
	if cfg.SDKName == "" {
		cfg.SDKName = "AnalyticsInstance"
	}
	if cfg.Semantics == nil {
		cfg.Semantics = reviewedAnalyticsInstanceRuntimeSemantics()
	}
	if cfg.BuildUpdateBody == nil {
		cfg.BuildUpdateBody = func(
			_ context.Context,
			resource *analyticsv1beta1.AnalyticsInstance,
			_ string,
			currentResponse any,
		) (any, bool, error) {
			return buildAnalyticsInstanceUpdateBody(resource, currentResponse)
		}
	}

	return &AnalyticsInstanceServiceManager{
		client: defaultAnalyticsInstanceServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*analyticsv1beta1.AnalyticsInstance](cfg),
		},
	}
}

func newAnalyticsInstanceTestResource() *analyticsv1beta1.AnalyticsInstance {
	return &analyticsv1beta1.AnalyticsInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "analytics-sample",
			Namespace: "default",
		},
		Spec: analyticsv1beta1.AnalyticsInstanceSpec{
			Name:              "analytics-sample",
			CompartmentId:     "ocid1.compartment.oc1..example",
			FeatureSet:        "ENTERPRISE_ANALYTICS",
			Capacity:          analyticsv1beta1.AnalyticsInstanceCapacity{CapacityType: "OLPU_COUNT", CapacityValue: 2},
			LicenseType:       "LICENSE_INCLUDED",
			Description:       "analytics description",
			EmailNotification: "ops@example.com",
			DefinedTags: map[string]shared.MapValue{
				"Operations": {
					"CostCenter": "42",
				},
			},
			FreeformTags: map[string]string{
				"env": "test",
			},
		},
	}
}

func newExistingAnalyticsInstanceTestResource(existingID string) *analyticsv1beta1.AnalyticsInstance {
	resource := newAnalyticsInstanceTestResource()
	resource.Status = analyticsv1beta1.AnalyticsInstanceStatus{
		OsokStatus: shared.OSOKStatus{
			Ocid: shared.OCID(existingID),
		},
		Id: existingID,
	}
	return resource
}

func observedAnalyticsInstanceFromSpec(
	id string,
	spec analyticsv1beta1.AnalyticsInstanceSpec,
	lifecycleState string,
) analyticssdk.AnalyticsInstance {
	now := &common.SDKTime{Time: time.Unix(1713240000, 0).UTC()}

	return analyticssdk.AnalyticsInstance{
		Id:                     common.String(id),
		Name:                   common.String(spec.Name),
		CompartmentId:          common.String(spec.CompartmentId),
		LifecycleState:         analyticssdk.AnalyticsInstanceLifecycleStateEnum(lifecycleState),
		FeatureSet:             analyticssdk.FeatureSetEnum(spec.FeatureSet),
		Capacity:               observedAnalyticsInstanceCapacity(spec.Capacity),
		NetworkEndpointDetails: observedAnalyticsInstanceNetworkEndpoint(spec.NetworkEndpointDetails),
		TimeCreated:            now,
		Description:            pointerOrNil(spec.Description),
		LicenseType:            analyticssdk.LicenseTypeEnum(spec.LicenseType),
		EmailNotification:      pointerOrNil(spec.EmailNotification),
		UpdateChannel:          analyticssdk.UpdateChannelEnum(spec.UpdateChannel),
		DefinedTags:            analyticsDefinedTagsFromSpec(spec.DefinedTags),
		FreeformTags:           cloneStringMap(spec.FreeformTags),
		SystemTags:             analyticsSystemTagsForTest(),
		KmsKeyId:               pointerOrNil(spec.KmsKeyId),
		ServiceUrl:             common.String("https://analytics.example.com"),
		TimeUpdated:            now,
		FeatureBundle:          analyticssdk.FeatureBundleEnum(spec.FeatureBundle),
		DomainId:               pointerOrNil(spec.DomainId),
	}
}

func observedAnalyticsInstanceSummaryFromSpec(
	id string,
	spec analyticsv1beta1.AnalyticsInstanceSpec,
	lifecycleState string,
) analyticssdk.AnalyticsInstanceSummary {
	now := &common.SDKTime{Time: time.Unix(1713240000, 0).UTC()}

	return analyticssdk.AnalyticsInstanceSummary{
		Id:                     common.String(id),
		Name:                   common.String(spec.Name),
		CompartmentId:          common.String(spec.CompartmentId),
		LifecycleState:         analyticssdk.AnalyticsInstanceLifecycleStateEnum(lifecycleState),
		FeatureSet:             analyticssdk.FeatureSetEnum(spec.FeatureSet),
		Capacity:               observedAnalyticsInstanceCapacity(spec.Capacity),
		NetworkEndpointDetails: observedAnalyticsInstanceNetworkEndpoint(spec.NetworkEndpointDetails),
		TimeCreated:            now,
		Description:            pointerOrNil(spec.Description),
		LicenseType:            analyticssdk.LicenseTypeEnum(spec.LicenseType),
		EmailNotification:      pointerOrNil(spec.EmailNotification),
		DefinedTags:            analyticsDefinedTagsFromSpec(spec.DefinedTags),
		FreeformTags:           cloneStringMap(spec.FreeformTags),
		SystemTags:             analyticsSystemTagsForTest(),
		ServiceUrl:             common.String("https://analytics.example.com"),
		TimeUpdated:            now,
	}
}

func analyticsSystemTagsForTest() map[string]map[string]interface{} {
	return map[string]map[string]interface{}{
		"orcl-cloud": {
			"key": "value",
		},
	}
}

func observedAnalyticsInstanceCapacity(
	spec analyticsv1beta1.AnalyticsInstanceCapacity,
) *analyticssdk.Capacity {
	return &analyticssdk.Capacity{
		CapacityType:  analyticssdk.CapacityTypeEnum(spec.CapacityType),
		CapacityValue: common.Int(spec.CapacityValue),
	}
}

func observedAnalyticsInstanceNetworkEndpoint(
	spec analyticsv1beta1.AnalyticsInstanceNetworkEndpointDetails,
) analyticssdk.NetworkEndpointDetails {
	switch spec.NetworkEndpointType {
	case "PUBLIC":
		whitelistedVcns := make([]analyticssdk.VirtualCloudNetwork, 0, len(spec.WhitelistedVcns))
		for _, vcn := range spec.WhitelistedVcns {
			whitelistedVcns = append(whitelistedVcns, analyticssdk.VirtualCloudNetwork{
				Id:             common.String(vcn.Id),
				WhitelistedIps: append([]string(nil), vcn.WhitelistedIps...),
			})
		}

		whitelistedServices := make([]analyticssdk.AccessControlServiceTypeEnum, 0, len(spec.WhitelistedServices))
		for _, service := range spec.WhitelistedServices {
			whitelistedServices = append(whitelistedServices, analyticssdk.AccessControlServiceTypeEnum(service))
		}

		return analyticssdk.PublicEndpointDetails{
			WhitelistedIps:      append([]string(nil), spec.WhitelistedIps...),
			WhitelistedVcns:     whitelistedVcns,
			WhitelistedServices: whitelistedServices,
		}
	case "PRIVATE":
		return analyticssdk.PrivateEndpointDetails{
			VcnId:                   pointerOrNil(spec.VcnId),
			SubnetId:                pointerOrNil(spec.SubnetId),
			NetworkSecurityGroupIds: append([]string(nil), spec.NetworkSecurityGroupIds...),
		}
	default:
		return nil
	}
}

func requireAnalyticsInstanceOpcRequestID(t *testing.T, resource *analyticsv1beta1.AnalyticsInstance, want string) {
	t.Helper()

	if got := resource.Status.OsokStatus.OpcRequestID; got != want {
		t.Fatalf("status.opcRequestId = %q, want %q", got, want)
	}
}

func requireAnalyticsInstanceAsyncCurrent(
	t *testing.T,
	resource *analyticsv1beta1.AnalyticsInstance,
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

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func contains(haystack string, needle string) bool {
	return strings.Contains(haystack, needle)
}

func containsStringSlice(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func ptr[T any](value T) *T {
	return &value
}

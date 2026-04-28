package steeringpolicy

import (
	"context"
	"strings"
	"testing"

	dnssdk "github.com/oracle/oci-go-sdk/v65/dns"
	dnsv1beta1 "github.com/oracle/oci-service-operator/api/dns/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testCompartmentID      = "ocid1.compartment.oc1..steering"
	testSteeringPolicyID   = "ocid1.dnssteeringpolicy.oc1..policy"
	testSteeringPolicyName = "test-steering-policy"
)

type fakeSteeringPolicyOCIClient struct {
	createRequests  []dnssdk.CreateSteeringPolicyRequest
	createResponses []dnssdk.CreateSteeringPolicyResponse
	createErrors    []error
	getRequests     []dnssdk.GetSteeringPolicyRequest
	getResponses    []dnssdk.GetSteeringPolicyResponse
	getErrors       []error
	listRequests    []dnssdk.ListSteeringPoliciesRequest
	listResponses   []dnssdk.ListSteeringPoliciesResponse
	listErrors      []error
	updateRequests  []dnssdk.UpdateSteeringPolicyRequest
	updateResponses []dnssdk.UpdateSteeringPolicyResponse
	updateErrors    []error
	deleteRequests  []dnssdk.DeleteSteeringPolicyRequest
	deleteResponses []dnssdk.DeleteSteeringPolicyResponse
	deleteErrors    []error
}

func (f *fakeSteeringPolicyOCIClient) CreateSteeringPolicy(_ context.Context, request dnssdk.CreateSteeringPolicyRequest) (dnssdk.CreateSteeringPolicyResponse, error) {
	f.createRequests = append(f.createRequests, request)
	index := len(f.createRequests) - 1
	if err := indexedError(f.createErrors, index); err != nil {
		return dnssdk.CreateSteeringPolicyResponse{}, err
	}
	if index < len(f.createResponses) {
		return f.createResponses[index], nil
	}
	return dnssdk.CreateSteeringPolicyResponse{}, nil
}

func (f *fakeSteeringPolicyOCIClient) GetSteeringPolicy(_ context.Context, request dnssdk.GetSteeringPolicyRequest) (dnssdk.GetSteeringPolicyResponse, error) {
	f.getRequests = append(f.getRequests, request)
	index := len(f.getRequests) - 1
	if err := indexedError(f.getErrors, index); err != nil {
		return dnssdk.GetSteeringPolicyResponse{}, err
	}
	if index < len(f.getResponses) {
		return f.getResponses[index], nil
	}
	return dnssdk.GetSteeringPolicyResponse{}, nil
}

func (f *fakeSteeringPolicyOCIClient) ListSteeringPolicies(_ context.Context, request dnssdk.ListSteeringPoliciesRequest) (dnssdk.ListSteeringPoliciesResponse, error) {
	f.listRequests = append(f.listRequests, request)
	index := len(f.listRequests) - 1
	if err := indexedError(f.listErrors, index); err != nil {
		return dnssdk.ListSteeringPoliciesResponse{}, err
	}
	if index < len(f.listResponses) {
		return f.listResponses[index], nil
	}
	return dnssdk.ListSteeringPoliciesResponse{}, nil
}

func (f *fakeSteeringPolicyOCIClient) UpdateSteeringPolicy(_ context.Context, request dnssdk.UpdateSteeringPolicyRequest) (dnssdk.UpdateSteeringPolicyResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	index := len(f.updateRequests) - 1
	if err := indexedError(f.updateErrors, index); err != nil {
		return dnssdk.UpdateSteeringPolicyResponse{}, err
	}
	if index < len(f.updateResponses) {
		return f.updateResponses[index], nil
	}
	return dnssdk.UpdateSteeringPolicyResponse{}, nil
}

func (f *fakeSteeringPolicyOCIClient) DeleteSteeringPolicy(_ context.Context, request dnssdk.DeleteSteeringPolicyRequest) (dnssdk.DeleteSteeringPolicyResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	index := len(f.deleteRequests) - 1
	if err := indexedError(f.deleteErrors, index); err != nil {
		return dnssdk.DeleteSteeringPolicyResponse{}, err
	}
	if index < len(f.deleteResponses) {
		return f.deleteResponses[index], nil
	}
	return dnssdk.DeleteSteeringPolicyResponse{}, nil
}

func indexedError(errors []error, index int) error {
	if index < len(errors) {
		return errors[index]
	}
	return nil
}

func TestSteeringPolicyRuntimeSemanticsAndBodyShaping(t *testing.T) {
	hooks := newSteeringPolicyDefaultRuntimeHooks(dnssdk.DnsClient{})
	applySteeringPolicyRuntimeHooks(&hooks)

	requireSteeringPolicyRuntimeSemantics(t, hooks.Semantics)

	resource := newTestSteeringPolicy()
	resource.Spec.Answers[0].IsDisabled = false
	resource.Spec.Rules = []dnsv1beta1.SteeringPolicyRule{
		{
			RuleType: "FILTER",
			DefaultAnswerData: []dnsv1beta1.SteeringPolicyRuleDefaultAnswerData{
				{AnswerCondition: "answer.name == 'primary'", ShouldKeep: false},
			},
		},
		{
			RuleType: "WEIGHTED",
			JsonData: `{"ruleType":"WEIGHTED","defaultAnswerData":[{"answerCondition":"answer.pool == 'blue'","value":10}]}`,
		},
	}

	body, err := buildSteeringPolicyCreateBody(resource)
	if err != nil {
		t.Fatalf("buildSteeringPolicyCreateBody() error = %v", err)
	}
	requireSteeringPolicyCreateBodyPreservesRuleValues(t, body)
}

func requireSteeringPolicyRuntimeSemantics(t *testing.T, semantics *generatedruntime.Semantics) {
	t.Helper()

	if semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed semantics")
	}
	if got := semantics.Delete.Policy; got != "required" {
		t.Fatalf("Delete.Policy = %q, want required", got)
	}
	if !containsString(semantics.Mutation.Mutable, "rules") {
		t.Fatalf("Mutation.Mutable = %#v, want rules", semantics.Mutation.Mutable)
	}
	if !containsString(semantics.Mutation.ForceNew, "compartmentId") {
		t.Fatalf("Mutation.ForceNew = %#v, want compartmentId", semantics.Mutation.ForceNew)
	}
}

func requireSteeringPolicyCreateBodyPreservesRuleValues(t *testing.T, body dnssdk.CreateSteeringPolicyDetails) {
	t.Helper()

	if body.Answers[0].IsDisabled == nil || *body.Answers[0].IsDisabled {
		t.Fatalf("answer.isDisabled = %#v, want explicit false", body.Answers[0].IsDisabled)
	}
	filterRule, ok := body.Rules[0].(dnssdk.SteeringPolicyFilterRule)
	if !ok {
		t.Fatalf("rule[0] type = %T, want SteeringPolicyFilterRule", body.Rules[0])
	}
	if filterRule.DefaultAnswerData[0].ShouldKeep == nil || *filterRule.DefaultAnswerData[0].ShouldKeep {
		t.Fatalf("filter shouldKeep = %#v, want explicit false", filterRule.DefaultAnswerData[0].ShouldKeep)
	}
	weightedRule, ok := body.Rules[1].(dnssdk.SteeringPolicyWeightedRule)
	if !ok {
		t.Fatalf("rule[1] type = %T, want SteeringPolicyWeightedRule", body.Rules[1])
	}
	if weightedRule.DefaultAnswerData[0].Value == nil || *weightedRule.DefaultAnswerData[0].Value != 10 {
		t.Fatalf("weighted value = %#v, want 10", weightedRule.DefaultAnswerData[0].Value)
	}
}

func TestSteeringPolicyCreateOrUpdateCreatesAndReadsBack(t *testing.T) {
	resource := newTestSteeringPolicy()
	client := &fakeSteeringPolicyOCIClient{
		listResponses: []dnssdk.ListSteeringPoliciesResponse{{}},
		createResponses: []dnssdk.CreateSteeringPolicyResponse{{
			SteeringPolicy: sdkSteeringPolicyFromResource(resource, testSteeringPolicyID, dnssdk.SteeringPolicyLifecycleStateCreating),
			OpcRequestId:   stringPointer("opc-create-1"),
		}},
		getResponses: []dnssdk.GetSteeringPolicyResponse{{
			SteeringPolicy: sdkSteeringPolicyFromResource(resource, testSteeringPolicyID, dnssdk.SteeringPolicyLifecycleStateActive),
		}},
	}
	runtimeClient := newTestSteeringPolicyServiceClient(client)

	response, err := runtimeClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	requireCreateOrUpdateResponse(t, response.IsSuccessful, response.ShouldRequeue)
	requireSteeringPolicyCreateRequest(t, client.createRequests)
	requireSteeringPolicyActiveStatus(t, resource, "opc-create-1")
}

func requireCreateOrUpdateResponse(t *testing.T, isSuccessful bool, shouldRequeue bool) {
	t.Helper()

	if !isSuccessful || shouldRequeue {
		t.Fatalf("CreateOrUpdate() response successful=%t shouldRequeue=%t, want successful non-requeue", isSuccessful, shouldRequeue)
	}
}

func requireSteeringPolicyCreateRequest(t *testing.T, requests []dnssdk.CreateSteeringPolicyRequest) {
	t.Helper()

	if len(requests) != 1 {
		t.Fatalf("CreateSteeringPolicy calls = %d, want 1", len(requests))
	}
	createRequest := requests[0]
	if createRequest.OpcRetryToken == nil || strings.TrimSpace(*createRequest.OpcRetryToken) == "" {
		t.Fatal("CreateSteeringPolicy OpcRetryToken is empty")
	}
	if got := stringPointerValue(createRequest.CompartmentId); got != testCompartmentID {
		t.Fatalf("CreateSteeringPolicy compartmentId = %q, want %q", got, testCompartmentID)
	}
}

func requireSteeringPolicyActiveStatus(t *testing.T, resource *dnsv1beta1.SteeringPolicy, opcRequestID string) {
	t.Helper()

	if resource.Status.OsokStatus.Ocid != shared.OCID(testSteeringPolicyID) {
		t.Fatalf("status.ocid = %q, want %q", resource.Status.OsokStatus.Ocid, testSteeringPolicyID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != opcRequestID {
		t.Fatalf("status.opcRequestId = %q, want %s", got, opcRequestID)
	}
	if got := resource.Status.LifecycleState; got != string(dnssdk.SteeringPolicyLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
}

func TestSteeringPolicyCreateOrUpdateBindsExistingAcrossListPages(t *testing.T) {
	resource := newTestSteeringPolicy()
	client := &fakeSteeringPolicyOCIClient{
		listResponses: []dnssdk.ListSteeringPoliciesResponse{
			{OpcNextPage: stringPointer("page-2")},
			{Items: []dnssdk.SteeringPolicySummary{sdkSteeringPolicySummary(resource, testSteeringPolicyID, dnssdk.SteeringPolicySummaryLifecycleStateActive)}},
		},
		getResponses: []dnssdk.GetSteeringPolicyResponse{{
			SteeringPolicy: sdkSteeringPolicyFromResource(resource, testSteeringPolicyID, dnssdk.SteeringPolicyLifecycleStateActive),
		}},
	}
	runtimeClient := newTestSteeringPolicyServiceClient(client)

	if _, err := runtimeClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{}); err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("CreateSteeringPolicy calls = %d, want 0 for bind", len(client.createRequests))
	}
	if len(client.listRequests) != 2 {
		t.Fatalf("ListSteeringPolicies calls = %d, want 2", len(client.listRequests))
	}
	if got := stringPointerValue(client.listRequests[1].Page); got != "page-2" {
		t.Fatalf("second ListSteeringPolicies page = %q, want page-2", got)
	}
	if resource.Status.OsokStatus.Ocid != shared.OCID(testSteeringPolicyID) {
		t.Fatalf("status.ocid = %q, want bound OCID", resource.Status.OsokStatus.Ocid)
	}
}

func TestSteeringPolicyCreateOrUpdateNoopDoesNotUpdate(t *testing.T) {
	resource := newTestSteeringPolicy()
	resource.Status.OsokStatus.Ocid = shared.OCID(testSteeringPolicyID)
	client := &fakeSteeringPolicyOCIClient{
		getResponses: []dnssdk.GetSteeringPolicyResponse{{
			SteeringPolicy: sdkSteeringPolicyFromResource(resource, testSteeringPolicyID, dnssdk.SteeringPolicyLifecycleStateActive),
		}},
	}
	runtimeClient := newTestSteeringPolicyServiceClient(client)

	if _, err := runtimeClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{}); err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("UpdateSteeringPolicy calls = %d, want 0", len(client.updateRequests))
	}
}

func TestSteeringPolicyCreateOrUpdateNoopNormalizesAnswerRdata(t *testing.T) {
	resource := newTestSteeringPolicy()
	resource.Spec.Answers[0].Rtype = "AAAA"
	resource.Spec.Answers[0].Rdata = "2001:0db8:0000:0000:0000:0000:0000:0001"
	resource.Status.OsokStatus.Ocid = shared.OCID(testSteeringPolicyID)
	current := sdkSteeringPolicyFromResource(resource, testSteeringPolicyID, dnssdk.SteeringPolicyLifecycleStateActive)
	current.Answers[0].Rdata = stringPointer("2001:db8::1")
	client := &fakeSteeringPolicyOCIClient{
		getResponses: []dnssdk.GetSteeringPolicyResponse{{SteeringPolicy: current}},
	}
	runtimeClient := newTestSteeringPolicyServiceClient(client)

	if _, err := runtimeClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{}); err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("UpdateSteeringPolicy calls = %d, want 0 for OCI-normalized equivalent rdata", len(client.updateRequests))
	}
}

func TestSteeringPolicyCreateOrUpdateMutableUpdate(t *testing.T) {
	resource := newTestSteeringPolicy()
	resource.Spec.Ttl = 60
	resource.Status.OsokStatus.Ocid = shared.OCID(testSteeringPolicyID)
	current := sdkSteeringPolicyFromResource(resource, testSteeringPolicyID, dnssdk.SteeringPolicyLifecycleStateActive)
	*current.Ttl = 30
	updated := sdkSteeringPolicyFromResource(resource, testSteeringPolicyID, dnssdk.SteeringPolicyLifecycleStateActive)
	client := &fakeSteeringPolicyOCIClient{
		getResponses: []dnssdk.GetSteeringPolicyResponse{
			{SteeringPolicy: current},
			{SteeringPolicy: updated},
		},
		updateResponses: []dnssdk.UpdateSteeringPolicyResponse{{
			SteeringPolicy: updated,
			OpcRequestId:   stringPointer("opc-update-1"),
		}},
	}
	runtimeClient := newTestSteeringPolicyServiceClient(client)

	if _, err := runtimeClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{}); err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if len(client.updateRequests) != 1 {
		t.Fatalf("UpdateSteeringPolicy calls = %d, want 1", len(client.updateRequests))
	}
	updateDetails := client.updateRequests[0].UpdateSteeringPolicyDetails
	if updateDetails.Ttl == nil || *updateDetails.Ttl != 60 {
		t.Fatalf("UpdateSteeringPolicy ttl = %#v, want 60", updateDetails.Ttl)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-update-1", got)
	}
}

func TestSteeringPolicyCreateOrUpdateRejectsCompartmentDriftBeforeUpdate(t *testing.T) {
	resource := newTestSteeringPolicy()
	resource.Status.OsokStatus.Ocid = shared.OCID(testSteeringPolicyID)
	current := sdkSteeringPolicyFromResource(resource, testSteeringPolicyID, dnssdk.SteeringPolicyLifecycleStateActive)
	current.CompartmentId = stringPointer("ocid1.compartment.oc1..old")
	client := &fakeSteeringPolicyOCIClient{
		getResponses: []dnssdk.GetSteeringPolicyResponse{{SteeringPolicy: current}},
	}
	runtimeClient := newTestSteeringPolicyServiceClient(client)

	_, err := runtimeClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want compartment drift rejection")
	}
	if !strings.Contains(err.Error(), "compartmentId") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartmentId drift", err)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("UpdateSteeringPolicy calls = %d, want 0 after drift rejection", len(client.updateRequests))
	}
}

func TestSteeringPolicyDeleteConfirmation(t *testing.T) {
	t.Run("retains finalizer while lifecycle is deleting", func(t *testing.T) {
		resource := newTestSteeringPolicy()
		resource.Status.OsokStatus.Ocid = shared.OCID(testSteeringPolicyID)
		client := &fakeSteeringPolicyOCIClient{
			getResponses: []dnssdk.GetSteeringPolicyResponse{
				{SteeringPolicy: sdkSteeringPolicyFromResource(resource, testSteeringPolicyID, dnssdk.SteeringPolicyLifecycleStateActive)},
				{SteeringPolicy: sdkSteeringPolicyFromResource(resource, testSteeringPolicyID, dnssdk.SteeringPolicyLifecycleStateDeleting)},
			},
			deleteResponses: []dnssdk.DeleteSteeringPolicyResponse{{OpcRequestId: stringPointer("opc-delete-1")}},
		}
		runtimeClient := newTestSteeringPolicyServiceClient(client)

		deleted, err := runtimeClient.Delete(context.Background(), resource)
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
		if deleted {
			t.Fatal("Delete() deleted = true, want false while readback is DELETING")
		}
		if got := resource.Status.OsokStatus.Reason; got != string(shared.Terminating) {
			t.Fatalf("status.reason = %q, want Terminating", got)
		}
		if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-1" {
			t.Fatalf("status.opcRequestId = %q, want opc-delete-1", got)
		}
	})

	t.Run("releases finalizer after unambiguous not found", func(t *testing.T) {
		resource := newTestSteeringPolicy()
		resource.Status.OsokStatus.Ocid = shared.OCID(testSteeringPolicyID)
		client := &fakeSteeringPolicyOCIClient{
			getResponses: []dnssdk.GetSteeringPolicyResponse{{
				SteeringPolicy: sdkSteeringPolicyFromResource(resource, testSteeringPolicyID, dnssdk.SteeringPolicyLifecycleStateActive),
			}},
			getErrors: []error{
				nil,
				errortest.NewServiceError(404, errorutil.NotFound, "missing"),
			},
			deleteResponses: []dnssdk.DeleteSteeringPolicyResponse{{OpcRequestId: stringPointer("opc-delete-1")}},
		}
		runtimeClient := newTestSteeringPolicyServiceClient(client)

		deleted, err := runtimeClient.Delete(context.Background(), resource)
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
		if !deleted {
			t.Fatal("Delete() deleted = false, want true after unambiguous not found")
		}
		if resource.Status.OsokStatus.DeletedAt == nil {
			t.Fatal("status.deletedAt = nil, want deletion timestamp")
		}
	})
}

func TestSteeringPolicyDeleteAuthShapedNotFoundRemainsFatal(t *testing.T) {
	resource := newTestSteeringPolicy()
	resource.Status.OsokStatus.Ocid = shared.OCID(testSteeringPolicyID)
	client := &fakeSteeringPolicyOCIClient{
		getResponses: []dnssdk.GetSteeringPolicyResponse{{
			SteeringPolicy: sdkSteeringPolicyFromResource(resource, testSteeringPolicyID, dnssdk.SteeringPolicyLifecycleStateActive),
		}},
		deleteErrors: []error{errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "auth ambiguity")},
	}
	runtimeClient := newTestSteeringPolicyServiceClient(client)

	deleted, err := runtimeClient.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped not found to remain fatal")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped not found")
	}
	if !strings.Contains(err.Error(), "authorization-shaped not found") {
		t.Fatalf("Delete() error = %v, want conservative auth-shaped message", err)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want surfaced error request id", got)
	}
}

func TestSteeringPolicyDeleteAuthShapedConfirmReadRemainsFatal(t *testing.T) {
	resource := newTestSteeringPolicy()
	resource.Status.OsokStatus.Ocid = shared.OCID(testSteeringPolicyID)
	client := &fakeSteeringPolicyOCIClient{
		getResponses: []dnssdk.GetSteeringPolicyResponse{{
			SteeringPolicy: sdkSteeringPolicyFromResource(resource, testSteeringPolicyID, dnssdk.SteeringPolicyLifecycleStateActive),
		}},
		getErrors: []error{
			nil,
			errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "auth ambiguity"),
		},
		deleteResponses: []dnssdk.DeleteSteeringPolicyResponse{{OpcRequestId: stringPointer("opc-delete-1")}},
	}
	runtimeClient := newTestSteeringPolicyServiceClient(client)

	deleted, err := runtimeClient.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped confirm read to remain fatal")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped confirm read")
	}
	if !strings.Contains(err.Error(), "authorization-shaped not found") {
		t.Fatalf("Delete() error = %v, want conservative auth-shaped message", err)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want surfaced error request id", got)
	}
}

func newTestSteeringPolicyServiceClient(fake *fakeSteeringPolicyOCIClient) defaultSteeringPolicyServiceClient {
	hooks := newSteeringPolicyDefaultRuntimeHooks(dnssdk.DnsClient{})
	hooks.Create.Call = fake.CreateSteeringPolicy
	hooks.Get.Call = fake.GetSteeringPolicy
	hooks.List.Call = fake.ListSteeringPolicies
	hooks.Update.Call = fake.UpdateSteeringPolicy
	hooks.Delete.Call = fake.DeleteSteeringPolicy
	applySteeringPolicyRuntimeHooks(&hooks)

	return defaultSteeringPolicyServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*dnsv1beta1.SteeringPolicy](
			buildSteeringPolicyGeneratedRuntimeConfig(&SteeringPolicyServiceManager{}, hooks),
		),
	}
}

func newTestSteeringPolicy() *dnsv1beta1.SteeringPolicy {
	return &dnsv1beta1.SteeringPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: testSteeringPolicyName, Namespace: "default"},
		Spec: dnsv1beta1.SteeringPolicySpec{
			CompartmentId: testCompartmentID,
			DisplayName:   testSteeringPolicyName,
			Template:      string(dnssdk.SteeringPolicyTemplateFailover),
			Ttl:           30,
			FreeformTags:  map[string]string{"env": "test"},
			DefinedTags:   map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
			Answers: []dnsv1beta1.SteeringPolicyAnswer{
				{Name: "primary", Rtype: "A", Rdata: "192.0.2.10", Pool: "blue"},
			},
			Rules: []dnsv1beta1.SteeringPolicyRule{
				{
					RuleType: "FILTER",
					DefaultAnswerData: []dnsv1beta1.SteeringPolicyRuleDefaultAnswerData{
						{AnswerCondition: "answer.name == 'primary'", ShouldKeep: true},
					},
				},
				{
					RuleType:     "LIMIT",
					DefaultCount: 1,
				},
			},
		},
	}
}

func sdkSteeringPolicyFromResource(
	resource *dnsv1beta1.SteeringPolicy,
	id string,
	lifecycle dnssdk.SteeringPolicyLifecycleStateEnum,
) dnssdk.SteeringPolicy {
	rules, err := steeringPolicySDKRules(resource.Spec.Rules)
	if err != nil {
		panic(err)
	}
	return dnssdk.SteeringPolicy{
		CompartmentId:        stringPointer(resource.Spec.CompartmentId),
		DisplayName:          stringPointer(resource.Spec.DisplayName),
		Ttl:                  intPointerNonZero(resource.Spec.Ttl),
		Template:             dnssdk.SteeringPolicyTemplateEnum(resource.Spec.Template),
		FreeformTags:         cloneStringMap(resource.Spec.FreeformTags),
		DefinedTags:          steeringPolicyDefinedTags(resource.Spec.DefinedTags),
		Answers:              steeringPolicySDKAnswers(resource.Spec.Answers),
		Rules:                rules,
		Id:                   stringPointer(id),
		LifecycleState:       lifecycle,
		HealthCheckMonitorId: stringPointer(resource.Spec.HealthCheckMonitorId),
	}
}

func sdkSteeringPolicySummary(
	resource *dnsv1beta1.SteeringPolicy,
	id string,
	lifecycle dnssdk.SteeringPolicySummaryLifecycleStateEnum,
) dnssdk.SteeringPolicySummary {
	return dnssdk.SteeringPolicySummary{
		CompartmentId:  stringPointer(resource.Spec.CompartmentId),
		DisplayName:    stringPointer(resource.Spec.DisplayName),
		Ttl:            intPointerNonZero(resource.Spec.Ttl),
		Template:       dnssdk.SteeringPolicySummaryTemplateEnum(resource.Spec.Template),
		FreeformTags:   cloneStringMap(resource.Spec.FreeformTags),
		DefinedTags:    steeringPolicyDefinedTags(resource.Spec.DefinedTags),
		Id:             stringPointer(id),
		LifecycleState: lifecycle,
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

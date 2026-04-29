/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package skill

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	odasdk "github.com/oracle/oci-go-sdk/v65/oda"
	odav1beta1 "github.com/oracle/oci-service-operator/api/oda/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestSkillRuntimeSemanticsEncodesLifecycleContract(t *testing.T) {
	semantics := newSkillRuntimeSemantics()
	if semantics.FormalService != "oda" || semantics.FormalSlug != "skill" {
		t.Fatalf("formal binding = %s/%s, want oda/skill", semantics.FormalService, semantics.FormalSlug)
	}
	if semantics.Async == nil || semantics.Async.Strategy != "workrequest+lifecycle" || semantics.Async.Runtime != "handwritten" {
		t.Fatalf("async semantics = %#v, want handwritten workrequest+lifecycle", semantics.Async)
	}
	if !containsString(semantics.Lifecycle.ProvisioningStates, string(odasdk.LifecycleStateCreating)) {
		t.Fatalf("provisioning states = %v, want CREATING", semantics.Lifecycle.ProvisioningStates)
	}
	if !containsString(semantics.Lifecycle.UpdatingStates, string(odasdk.LifecycleStateUpdating)) {
		t.Fatalf("updating states = %v, want UPDATING", semantics.Lifecycle.UpdatingStates)
	}
	if !containsString(semantics.Lifecycle.ActiveStates, string(odasdk.LifecycleStateActive)) {
		t.Fatalf("active states = %v, want ACTIVE", semantics.Lifecycle.ActiveStates)
	}
	if containsString(semantics.Lifecycle.ActiveStates, string(odasdk.LifecycleStateInactive)) {
		t.Fatalf("active states = %v, did not expect INACTIVE for formal success_condition=active", semantics.Lifecycle.ActiveStates)
	}
	if !containsString(semantics.Delete.PendingStates, string(odasdk.LifecycleStateDeleting)) ||
		!containsString(semantics.Delete.TerminalStates, string(odasdk.LifecycleStateDeleted)) {
		t.Fatalf("delete states pending=%v terminal=%v, want DELETING/DELETED", semantics.Delete.PendingStates, semantics.Delete.TerminalStates)
	}
	for _, field := range []string{"category", "description", "freeformTags", "definedTags"} {
		if !containsString(semantics.Mutation.Mutable, field) {
			t.Fatalf("mutable fields = %v, want %s", semantics.Mutation.Mutable, field)
		}
	}
	for _, field := range []string{"kind", "id", "name", "displayName", "version"} {
		if !containsString(semantics.Mutation.ForceNew, field) {
			t.Fatalf("force-new fields = %v, want %s", semantics.Mutation.ForceNew, field)
		}
	}
	if semantics.CreateFollowUp.Strategy != "work-request-then-read" ||
		semantics.UpdateFollowUp.Strategy != "read-after-write" ||
		semantics.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf(
			"follow-up strategies create=%q update=%q delete=%q",
			semantics.CreateFollowUp.Strategy,
			semantics.UpdateFollowUp.Strategy,
			semantics.DeleteFollowUp.Strategy,
		)
	}
}

func TestBuildSkillCreateDetailsSupportsPolymorphicKinds(t *testing.T) {
	tests := []struct {
		name string
		kind string
		want any
	}{
		{name: "new", kind: skillKindNew, want: odasdk.CreateNewSkillDetails{}},
		{name: "clone", kind: skillKindClone, want: odasdk.CloneSkillDetails{}},
		{name: "extend", kind: skillKindExtend, want: odasdk.ExtendSkillDetails{}},
		{name: "version", kind: skillKindVersion, want: odasdk.CreateSkillVersionDetails{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desired := skillDesiredState{
				kind:        tt.kind,
				id:          "source-skill",
				name:        "SkillName",
				displayName: "Skill Display",
				version:     "1.0",
			}
			if tt.kind == skillKindNew {
				desired.id = ""
			}

			body, err := buildSkillCreateDetails(desired)
			if err != nil {
				t.Fatalf("buildSkillCreateDetails() error = %v", err)
			}
			switch tt.want.(type) {
			case odasdk.CreateNewSkillDetails:
				if _, ok := body.(odasdk.CreateNewSkillDetails); !ok {
					t.Fatalf("body type = %T, want CreateNewSkillDetails", body)
				}
			case odasdk.CloneSkillDetails:
				if _, ok := body.(odasdk.CloneSkillDetails); !ok {
					t.Fatalf("body type = %T, want CloneSkillDetails", body)
				}
			case odasdk.ExtendSkillDetails:
				if _, ok := body.(odasdk.ExtendSkillDetails); !ok {
					t.Fatalf("body type = %T, want ExtendSkillDetails", body)
				}
			case odasdk.CreateSkillVersionDetails:
				if _, ok := body.(odasdk.CreateSkillVersionDetails); !ok {
					t.Fatalf("body type = %T, want CreateSkillVersionDetails", body)
				}
			}
		})
	}
}

func TestSkillServiceClientCreatesWithParentAnnotationAndWorkRequest(t *testing.T) {
	fake := newFakeSkillOCIClient(t)
	fake.listResults = append(fake.listResults, listSkillsResult{response: odasdk.ListSkillsResponse{}})
	fake.createResults = append(fake.createResults, createSkillResult{
		response: odasdk.CreateSkillResponse{
			OpcWorkRequestId: common.String("wr-create-1"),
			OpcRequestId:     common.String("req-create-1"),
		},
	})
	client := newSkillServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	resource := testSkill()

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue", response)
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("create requests = %d, want 1", len(fake.createRequests))
	}
	create := fake.createRequests[0]
	if got := stringValue(create.OdaInstanceId); got != "oda-1" {
		t.Fatalf("create OdaInstanceId = %q, want oda-1", got)
	}
	if got := stringValue(create.OpcRetryToken); got != "uid-1" {
		t.Fatalf("create retry token = %q, want uid-1", got)
	}
	body, ok := create.CreateSkillDetails.(odasdk.CreateNewSkillDetails)
	if !ok {
		t.Fatalf("create body type = %T, want CreateNewSkillDetails", create.CreateSkillDetails)
	}
	if got := stringValue(body.Name); got != "SkillName" {
		t.Fatalf("create name = %q, want SkillName", got)
	}
	if got := stringValue(body.DisplayName); got != "Skill Display" {
		t.Fatalf("create displayName = %q, want Skill Display", got)
	}
	if got := stringValue(body.Version); got != "1.0" {
		t.Fatalf("create version = %q, want 1.0", got)
	}
	if resource.Status.OsokStatus.OpcRequestID != "req-create-1" {
		t.Fatalf("opc request id = %q, want req-create-1", resource.Status.OsokStatus.OpcRequestID)
	}
	assertSkillAsync(t, resource, shared.OSOKAsyncSourceWorkRequest, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassPending)
	if got := resource.Status.OsokStatus.Async.Current.WorkRequestID; got != "wr-create-1" {
		t.Fatalf("work request id = %q, want wr-create-1", got)
	}
}

func TestSkillServiceClientPollsSucceededCreateWorkRequestAndProjectsStatus(t *testing.T) {
	fake := newFakeSkillOCIClient(t)
	fake.workRequestResults = append(fake.workRequestResults, getWorkRequestResult{
		response: odasdk.GetWorkRequestResponse{
			OpcRequestId: common.String("req-wr-1"),
			WorkRequest: odasdk.WorkRequest{
				Id:              common.String("wr-create-1"),
				ResourceId:      common.String("skill-1"),
				RequestAction:   odasdk.WorkRequestRequestActionCreateSkill,
				Status:          odasdk.WorkRequestStatusSucceeded,
				PercentComplete: float32Ptr(100),
			},
		},
	})
	fake.getResults = append(fake.getResults, getSkillResult{
		response: getSkillResponse(skillSDK("skill-1", "SkillName", "1.0", "Skill Display", odasdk.LifecycleStateActive, "cat", "desc")),
	})
	client := newSkillServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	resource := testSkill()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseCreate,
		WorkRequestID:   "wr-create-1",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful without requeue", response)
	}
	if len(fake.workRequestRequests) != 1 {
		t.Fatalf("work request calls = %d, want 1", len(fake.workRequestRequests))
	}
	if got := stringValue(fake.getRequests[0].SkillId); got != "skill-1" {
		t.Fatalf("get SkillId = %q, want skill-1", got)
	}
	assertSkillStatus(t, resource, "skill-1", "SkillName", "1.0", "Skill Display", "ACTIVE", shared.Active)
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("async current = %#v, want nil after active create", resource.Status.OsokStatus.Async.Current)
	}
}

func TestSkillServiceClientBindsExistingWithoutCreate(t *testing.T) {
	fake := newFakeSkillOCIClient(t)
	fake.listResults = append(fake.listResults, listSkillsResult{
		response: odasdk.ListSkillsResponse{
			SkillCollection: odasdk.SkillCollection{
				Items: []odasdk.SkillSummary{
					skillSummary("skill-1", "SkillName", "1.0", "Skill Display", odasdk.LifecycleStateActive),
				},
			},
		},
	})
	fake.getResults = append(fake.getResults, getSkillResult{
		response: getSkillResponse(skillSDK("skill-1", "SkillName", "1.0", "Skill Display", odasdk.LifecycleStateActive, "cat", "desc")),
	})
	client := newSkillServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	resource := testSkill()

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful without requeue", response)
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("create requests = %d, want 0", len(fake.createRequests))
	}
	assertSkillStatus(t, resource, "skill-1", "SkillName", "1.0", "Skill Display", "ACTIVE", shared.Active)
}

func TestSkillServiceClientVersionCreateDoesNotBindByVersionAlone(t *testing.T) {
	fake := newFakeSkillOCIClient(t)
	fake.getResults = append(fake.getResults, getSkillResult{
		response: getSkillResponse(skillSDK("source-skill", "SkillName", "1.0", "Skill Display", odasdk.LifecycleStateActive, "cat", "desc")),
	})
	fake.listResults = append(fake.listResults, listSkillsResult{
		response: odasdk.ListSkillsResponse{
			SkillCollection: odasdk.SkillCollection{
				Items: []odasdk.SkillSummary{
					skillSummary("unrelated-skill", "OtherSkill", "2.0", "Other Skill", odasdk.LifecycleStateActive),
				},
			},
		},
	})
	fake.createResults = append(fake.createResults, createSkillResult{
		response: odasdk.CreateSkillResponse{
			OpcWorkRequestId: common.String("wr-version-1"),
			OpcRequestId:     common.String("req-version-1"),
		},
	})
	client := newSkillServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	resource := testSkill()
	resource.Spec.Kind = skillKindVersion
	resource.Spec.Id = "source-skill"
	resource.Spec.Name = ""
	resource.Spec.DisplayName = ""
	resource.Spec.Version = "2.0"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue", response)
	}
	if len(fake.getRequests) != 1 {
		t.Fatalf("get requests = %d, want 1 source Skill lookup", len(fake.getRequests))
	}
	if got := stringValue(fake.getRequests[0].SkillId); got != "source-skill" {
		t.Fatalf("source SkillId = %q, want source-skill", got)
	}
	if len(fake.listRequests) != 1 {
		t.Fatalf("list requests = %d, want 1", len(fake.listRequests))
	}
	if got := stringValue(fake.listRequests[0].Name); got != "SkillName" {
		t.Fatalf("list name = %q, want source SkillName", got)
	}
	if got := stringValue(fake.listRequests[0].Version); got != "2.0" {
		t.Fatalf("list version = %q, want 2.0", got)
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("create requests = %d, want 1", len(fake.createRequests))
	}
	body, ok := fake.createRequests[0].CreateSkillDetails.(odasdk.CreateSkillVersionDetails)
	if !ok {
		t.Fatalf("create body type = %T, want CreateSkillVersionDetails", fake.createRequests[0].CreateSkillDetails)
	}
	if got := stringValue(body.Id); got != "source-skill" {
		t.Fatalf("create version source id = %q, want source-skill", got)
	}
	if got := stringValue(body.Version); got != "2.0" {
		t.Fatalf("create version = %q, want 2.0", got)
	}
	if resource.Status.Id == "unrelated-skill" || string(resource.Status.OsokStatus.Ocid) == "unrelated-skill" {
		t.Fatalf("resource bound unrelated Skill by version alone: status=%#v", resource.Status)
	}
}

func TestSkillServiceClientUpdatesSupportedMutableDrift(t *testing.T) {
	fake := newFakeSkillOCIClient(t)
	fake.getResults = append(fake.getResults,
		getSkillResult{response: getSkillResponse(skillSDK("skill-1", "SkillName", "1.0", "Skill Display", odasdk.LifecycleStateActive, "old-cat", "old-desc"))},
		getSkillResult{response: getSkillResponse(skillSDK("skill-1", "SkillName", "1.0", "Skill Display", odasdk.LifecycleStateActive, "new-cat", "new-desc"))},
	)
	client := newSkillServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	resource := testSkill()
	resource.Status.OsokStatus.Ocid = shared.OCID("skill-1")
	resource.Spec.Category = "new-cat"
	resource.Spec.Description = "new-desc"
	resource.Spec.FreeformTags = map[string]string{"env": "test"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"ns": {"key": "value"}}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful without requeue", response)
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("update requests = %d, want 1", len(fake.updateRequests))
	}
	update := fake.updateRequests[0]
	if got := stringValue(update.OdaInstanceId); got != "oda-1" {
		t.Fatalf("update OdaInstanceId = %q, want oda-1", got)
	}
	if got := stringValue(update.SkillId); got != "skill-1" {
		t.Fatalf("update SkillId = %q, want skill-1", got)
	}
	if got := stringValue(update.UpdateSkillDetails.Category); got != "new-cat" {
		t.Fatalf("update category = %q, want new-cat", got)
	}
	if got := stringValue(update.UpdateSkillDetails.Description); got != "new-desc" {
		t.Fatalf("update description = %q, want new-desc", got)
	}
	if got := update.UpdateSkillDetails.FreeformTags["env"]; got != "test" {
		t.Fatalf("update freeform tag env = %q, want test", got)
	}
	if got := update.UpdateSkillDetails.DefinedTags["ns"]["key"]; got != "value" {
		t.Fatalf("update defined tag ns.key = %#v, want value", got)
	}
	assertSkillStatus(t, resource, "skill-1", "SkillName", "1.0", "Skill Display", "ACTIVE", shared.Active)
}

func TestSkillServiceClientUpdatesTagsWithoutClearingOmittedOptionalStrings(t *testing.T) {
	fake := newFakeSkillOCIClient(t)
	current := skillSDK("skill-1", "SkillName", "1.0", "Skill Display", odasdk.LifecycleStateActive, "old-cat", "old-desc")
	refreshed := skillSDK("skill-1", "SkillName", "1.0", "Skill Display", odasdk.LifecycleStateActive, "old-cat", "old-desc")
	refreshed.FreeformTags = map[string]string{"env": "test"}
	fake.getResults = append(fake.getResults,
		getSkillResult{response: getSkillResponse(current)},
		getSkillResult{response: getSkillResponse(refreshed)},
	)
	client := newSkillServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	resource := testSkill()
	resource.Status.OsokStatus.Ocid = shared.OCID("skill-1")
	resource.Spec.Category = ""
	resource.Spec.Description = ""
	resource.Spec.FreeformTags = map[string]string{"env": "test"}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful without requeue", response)
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("update requests = %d, want 1", len(fake.updateRequests))
	}
	update := fake.updateRequests[0]
	if update.UpdateSkillDetails.Category != nil {
		t.Fatalf("update category = %q, want omitted", stringValue(update.UpdateSkillDetails.Category))
	}
	if update.UpdateSkillDetails.Description != nil {
		t.Fatalf("update description = %q, want omitted", stringValue(update.UpdateSkillDetails.Description))
	}
	if got := update.UpdateSkillDetails.FreeformTags["env"]; got != "test" {
		t.Fatalf("update freeform tag env = %q, want test", got)
	}
}

func TestSkillServiceClientRejectsSourceDriftBeforeUpdate(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(*odav1beta1.Skill)
		wantText string
	}{
		{
			name: "source id",
			mutate: func(resource *odav1beta1.Skill) {
				resource.Spec.Kind = skillKindExtend
				resource.Spec.Id = "source-new"
			},
			wantText: "id",
		},
		{
			name: "source kind",
			mutate: func(resource *odav1beta1.Skill) {
				resource.Spec.Kind = skillKindClone
				resource.Spec.Id = "source-old"
			},
			wantText: "kind",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := newFakeSkillOCIClient(t)
			current := skillSDK("skill-1", "SkillName", "1.0", "Skill Display", odasdk.LifecycleStateActive, "old-cat", "desc")
			current.BaseId = common.String("source-old")
			fake.getResults = append(fake.getResults, getSkillResult{
				response: getSkillResponse(current),
			})
			client := newSkillServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
			resource := testSkill()
			resource.Status.OsokStatus.Ocid = shared.OCID("skill-1")
			resource.Spec.Category = "new-cat"
			tt.mutate(resource)

			response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			if err == nil || !strings.Contains(err.Error(), tt.wantText) {
				t.Fatalf("CreateOrUpdate() error = %v, want %s drift error", err, tt.wantText)
			}
			if response.IsSuccessful {
				t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
			}
			if len(fake.updateRequests) != 0 {
				t.Fatalf("update requests = %d, want 0", len(fake.updateRequests))
			}
			assertSkillCondition(t, resource, shared.Failed)
		})
	}
}

func TestSkillServiceClientRejectsDisplayNameDriftBeforeUpdate(t *testing.T) {
	fake := newFakeSkillOCIClient(t)
	fake.getResults = append(fake.getResults, getSkillResult{
		response: getSkillResponse(skillSDK("skill-1", "SkillName", "1.0", "Old Display", odasdk.LifecycleStateActive, "cat", "desc")),
	})
	client := newSkillServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	resource := testSkill()
	resource.Status.OsokStatus.Ocid = shared.OCID("skill-1")
	resource.Spec.DisplayName = "New Display"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "displayName") {
		t.Fatalf("CreateOrUpdate() error = %v, want displayName drift error", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0", len(fake.updateRequests))
	}
	assertSkillCondition(t, resource, shared.Failed)
}

func TestSkillServiceClientMapsFailedLifecycleToFailure(t *testing.T) {
	fake := newFakeSkillOCIClient(t)
	fake.getResults = append(fake.getResults, getSkillResult{
		response: getSkillResponse(skillSDK("skill-1", "SkillName", "1.0", "Skill Display", odasdk.LifecycleStateFailed, "cat", "desc")),
	})
	client := newSkillServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	resource := testSkill()
	resource.Status.OsokStatus.Ocid = shared.OCID("skill-1")

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want failed without requeue", response)
	}
	assertSkillStatus(t, resource, "skill-1", "SkillName", "1.0", "Skill Display", "FAILED", shared.Failed)
	assertSkillAsync(t, resource, shared.OSOKAsyncSourceLifecycle, shared.OSOKAsyncPhaseUpdate, shared.OSOKAsyncClassFailed)
}

func TestSkillServiceClientDeleteKeepsFinalizerUntilConfirmed(t *testing.T) {
	fake := newFakeSkillOCIClient(t)
	fake.getResults = append(fake.getResults,
		getSkillResult{response: getSkillResponse(skillSDK("skill-1", "SkillName", "1.0", "Skill Display", odasdk.LifecycleStateActive, "cat", "desc"))},
		getSkillResult{response: getSkillResponse(skillSDK("skill-1", "SkillName", "1.0", "Skill Display", odasdk.LifecycleStateDeleting, "cat", "desc"))},
	)
	fake.deleteResults = append(fake.deleteResults, deleteSkillResult{
		response: odasdk.DeleteSkillResponse{OpcRequestId: common.String("req-delete-1")},
	})
	client := newSkillServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	resource := testSkill()
	resource.Status.OsokStatus.Ocid = shared.OCID("skill-1")

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatalf("Delete() deleted = true, want false until readback confirms delete")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("delete requests = %d, want 1", len(fake.deleteRequests))
	}
	if got := stringValue(fake.deleteRequests[0].SkillId); got != "skill-1" {
		t.Fatalf("delete SkillId = %q, want skill-1", got)
	}
	assertSkillAsync(t, resource, shared.OSOKAsyncSourceLifecycle, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending)
	assertSkillCondition(t, resource, shared.Terminating)
}

func TestSkillServiceClientDeleteRetainsFinalizerWhileCreateWorkRequestPendingAndListMisses(t *testing.T) {
	fake := newFakeSkillOCIClient(t)
	fake.listResults = append(fake.listResults, listSkillsResult{
		response: odasdk.ListSkillsResponse{},
	})
	fake.workRequestResults = append(fake.workRequestResults, getWorkRequestResult{
		response: odasdk.GetWorkRequestResponse{
			OpcRequestId: common.String("req-wr-pending"),
			WorkRequest: odasdk.WorkRequest{
				Id:              common.String("wr-create-1"),
				RequestAction:   odasdk.WorkRequestRequestActionCreateSkill,
				Status:          odasdk.WorkRequestStatusInProgress,
				PercentComplete: float32Ptr(35),
			},
		},
	})
	client := newSkillServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	resource := testSkill()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseCreate,
		WorkRequestID:   "wr-create-1",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatalf("Delete() deleted = true, want false while create work request is still pending")
	}
	if len(fake.listRequests) != 1 {
		t.Fatalf("list requests = %d, want 1 initial delete lookup", len(fake.listRequests))
	}
	if len(fake.workRequestRequests) != 1 {
		t.Fatalf("work request calls = %d, want 1", len(fake.workRequestRequests))
	}
	if got := stringValue(fake.workRequestRequests[0].WorkRequestId); got != "wr-create-1" {
		t.Fatalf("work request id = %q, want wr-create-1", got)
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("delete requests = %d, want 0 before create resolves", len(fake.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("DeletedAt = %v, want nil while create work request is pending", resource.Status.OsokStatus.DeletedAt)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != "" {
		t.Fatalf("status.ocid = %q, want empty until create resolves", got)
	}
	assertSkillAsync(t, resource, shared.OSOKAsyncSourceWorkRequest, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassPending)
	assertSkillCondition(t, resource, shared.Provisioning)
}

func TestSkillServiceClientDeletePollsSucceededCreateWorkRequestThenDeletesResolvedSkill(t *testing.T) {
	fake := newFakeSkillOCIClient(t)
	fake.listResults = append(fake.listResults, listSkillsResult{
		response: odasdk.ListSkillsResponse{},
	})
	fake.workRequestResults = append(fake.workRequestResults, getWorkRequestResult{
		response: odasdk.GetWorkRequestResponse{
			OpcRequestId: common.String("req-wr-succeeded"),
			WorkRequest: odasdk.WorkRequest{
				Id:              common.String("wr-create-1"),
				OdaInstanceId:   common.String("oda-1"),
				ResourceId:      common.String("skill-1"),
				RequestAction:   odasdk.WorkRequestRequestActionCreateSkill,
				Status:          odasdk.WorkRequestStatusSucceeded,
				PercentComplete: float32Ptr(100),
			},
		},
	})
	fake.getResults = append(fake.getResults,
		getSkillResult{response: getSkillResponse(skillSDK("skill-1", "SkillName", "1.0", "Skill Display", odasdk.LifecycleStateActive, "cat", "desc"))},
		getSkillResult{response: getSkillResponse(skillSDK("skill-1", "SkillName", "1.0", "Skill Display", odasdk.LifecycleStateDeleting, "cat", "desc"))},
	)
	fake.deleteResults = append(fake.deleteResults, deleteSkillResult{
		response: odasdk.DeleteSkillResponse{OpcRequestId: common.String("req-delete-1")},
	})
	client := newSkillServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	resource := testSkill()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseCreate,
		WorkRequestID:   "wr-create-1",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatalf("Delete() deleted = true, want false until delete readback confirms removal")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("delete requests = %d, want 1 after create work request resolves", len(fake.deleteRequests))
	}
	if got := stringValue(fake.deleteRequests[0].SkillId); got != "skill-1" {
		t.Fatalf("delete SkillId = %q, want skill-1", got)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != "skill-1" {
		t.Fatalf("status.ocid = %q, want skill-1", got)
	}
	assertSkillAsync(t, resource, shared.OSOKAsyncSourceLifecycle, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending)
	assertSkillCondition(t, resource, shared.Terminating)
}

func TestSkillServiceClientDeleteReleasesFinalizerAfterReadbackNotFound(t *testing.T) {
	fake := newFakeSkillOCIClient(t)
	fake.getResults = append(fake.getResults,
		getSkillResult{response: getSkillResponse(skillSDK("skill-1", "SkillName", "1.0", "Skill Display", odasdk.LifecycleStateActive, "cat", "desc"))},
		getSkillResult{err: errSkillNotFound},
	)
	fake.deleteResults = append(fake.deleteResults, deleteSkillResult{
		response: odasdk.DeleteSkillResponse{OpcRequestId: common.String("req-delete-1")},
	})
	client := newSkillServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	resource := testSkill()
	resource.Status.OsokStatus.Ocid = shared.OCID("skill-1")

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatalf("Delete() deleted = false, want true after readback not found")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatalf("status.deletedAt = nil, want timestamp")
	}
	assertSkillCondition(t, resource, shared.Terminating)
}

func TestSkillServiceClientRequiresParentAnnotation(t *testing.T) {
	fake := newFakeSkillOCIClient(t)
	client := newSkillServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	resource := testSkill()
	resource.Annotations = nil

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), skillOdaInstanceIDAnnotation) {
		t.Fatalf("CreateOrUpdate() error = %v, want parent annotation error", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if len(fake.listRequests)+len(fake.createRequests)+len(fake.updateRequests) != 0 {
		t.Fatalf("OCI calls made before annotation validation: list=%d create=%d update=%d", len(fake.listRequests), len(fake.createRequests), len(fake.updateRequests))
	}
	assertSkillCondition(t, resource, shared.Failed)
}

type createSkillResult struct {
	response odasdk.CreateSkillResponse
	err      error
}

type getSkillResult struct {
	response odasdk.GetSkillResponse
	err      error
}

type listSkillsResult struct {
	response odasdk.ListSkillsResponse
	err      error
}

type updateSkillResult struct {
	response odasdk.UpdateSkillResponse
	err      error
}

type deleteSkillResult struct {
	response odasdk.DeleteSkillResponse
	err      error
}

type getWorkRequestResult struct {
	response odasdk.GetWorkRequestResponse
	err      error
}

type fakeSkillOCIClient struct {
	t *testing.T

	createRequests      []odasdk.CreateSkillRequest
	getRequests         []odasdk.GetSkillRequest
	listRequests        []odasdk.ListSkillsRequest
	updateRequests      []odasdk.UpdateSkillRequest
	deleteRequests      []odasdk.DeleteSkillRequest
	workRequestRequests []odasdk.GetWorkRequestRequest

	createResults      []createSkillResult
	getResults         []getSkillResult
	listResults        []listSkillsResult
	updateResults      []updateSkillResult
	deleteResults      []deleteSkillResult
	workRequestResults []getWorkRequestResult
}

func newFakeSkillOCIClient(t *testing.T) *fakeSkillOCIClient {
	t.Helper()
	return &fakeSkillOCIClient{t: t}
}

func (f *fakeSkillOCIClient) CreateSkill(_ context.Context, request odasdk.CreateSkillRequest) (odasdk.CreateSkillResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if len(f.createResults) == 0 {
		f.t.Fatalf("unexpected CreateSkill request: %#v", request)
	}
	result := f.createResults[0]
	f.createResults = f.createResults[1:]
	return result.response, result.err
}

func (f *fakeSkillOCIClient) GetSkill(_ context.Context, request odasdk.GetSkillRequest) (odasdk.GetSkillResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if len(f.getResults) == 0 {
		f.t.Fatalf("unexpected GetSkill request: %#v", request)
	}
	result := f.getResults[0]
	f.getResults = f.getResults[1:]
	return result.response, result.err
}

func (f *fakeSkillOCIClient) ListSkills(_ context.Context, request odasdk.ListSkillsRequest) (odasdk.ListSkillsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if len(f.listResults) == 0 {
		f.t.Fatalf("unexpected ListSkills request: %#v", request)
	}
	result := f.listResults[0]
	f.listResults = f.listResults[1:]
	return result.response, result.err
}

func (f *fakeSkillOCIClient) UpdateSkill(_ context.Context, request odasdk.UpdateSkillRequest) (odasdk.UpdateSkillResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if len(f.updateResults) == 0 {
		return odasdk.UpdateSkillResponse{}, nil
	}
	result := f.updateResults[0]
	f.updateResults = f.updateResults[1:]
	return result.response, result.err
}

func (f *fakeSkillOCIClient) DeleteSkill(_ context.Context, request odasdk.DeleteSkillRequest) (odasdk.DeleteSkillResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if len(f.deleteResults) == 0 {
		f.t.Fatalf("unexpected DeleteSkill request: %#v", request)
	}
	result := f.deleteResults[0]
	f.deleteResults = f.deleteResults[1:]
	return result.response, result.err
}

func (f *fakeSkillOCIClient) GetWorkRequest(_ context.Context, request odasdk.GetWorkRequestRequest) (odasdk.GetWorkRequestResponse, error) {
	f.workRequestRequests = append(f.workRequestRequests, request)
	if len(f.workRequestResults) == 0 {
		f.t.Fatalf("unexpected GetWorkRequest request: %#v", request)
	}
	result := f.workRequestResults[0]
	f.workRequestResults = f.workRequestResults[1:]
	return result.response, result.err
}

func testSkill() *odav1beta1.Skill {
	return &odav1beta1.Skill{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "skill",
			Namespace: "default",
			UID:       "uid-1",
			Annotations: map[string]string{
				skillOdaInstanceIDAnnotation: "oda-1",
			},
		},
		Spec: odav1beta1.SkillSpec{
			Kind:        skillKindNew,
			Name:        "SkillName",
			DisplayName: "Skill Display",
			Version:     "1.0",
			Category:    "cat",
			Description: "desc",
		},
	}
}

func skillSDK(id, name, version, displayName string, state odasdk.LifecycleStateEnum, category string, description string) odasdk.Skill {
	return odasdk.Skill{
		Id:               common.String(id),
		Name:             common.String(name),
		Version:          common.String(version),
		DisplayName:      common.String(displayName),
		LifecycleState:   state,
		LifecycleDetails: odasdk.BotPublishStatePublished,
		PlatformVersion:  common.String("22.12"),
		Category:         common.String(category),
		Description:      common.String(description),
		FreeformTags:     map[string]string{"live": "true"},
		DefinedTags:      map[string]map[string]interface{}{"live": {"key": "value"}},
	}
}

func skillSummary(id, name, version, displayName string, state odasdk.LifecycleStateEnum) odasdk.SkillSummary {
	return odasdk.SkillSummary{
		Id:               common.String(id),
		Name:             common.String(name),
		Version:          common.String(version),
		DisplayName:      common.String(displayName),
		LifecycleState:   state,
		LifecycleDetails: odasdk.BotPublishStatePublished,
		Category:         common.String("cat"),
		Namespace:        common.String("ns"),
		PlatformVersion:  common.String("22.12"),
	}
}

func getSkillResponse(current odasdk.Skill) odasdk.GetSkillResponse {
	return odasdk.GetSkillResponse{Skill: current}
}

func assertSkillStatus(
	t *testing.T,
	resource *odav1beta1.Skill,
	id string,
	name string,
	version string,
	displayName string,
	lifecycle string,
	condition shared.OSOKConditionType,
) {
	t.Helper()
	if resource.Status.Id != id {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, id)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != id {
		t.Fatalf("status.status.ocid = %q, want %q", got, id)
	}
	if resource.Status.Name != name {
		t.Fatalf("status.name = %q, want %q", resource.Status.Name, name)
	}
	if resource.Status.Version != version {
		t.Fatalf("status.version = %q, want %q", resource.Status.Version, version)
	}
	if resource.Status.DisplayName != displayName {
		t.Fatalf("status.displayName = %q, want %q", resource.Status.DisplayName, displayName)
	}
	if resource.Status.LifecycleState != lifecycle {
		t.Fatalf("status.lifecycleState = %q, want %q", resource.Status.LifecycleState, lifecycle)
	}
	assertSkillCondition(t, resource, condition)
}

func assertSkillCondition(t *testing.T, resource *odav1beta1.Skill, condition shared.OSOKConditionType) {
	t.Helper()
	for _, observed := range resource.Status.OsokStatus.Conditions {
		if observed.Type == condition {
			return
		}
	}
	t.Fatalf("condition %q not found in %#v", condition, resource.Status.OsokStatus.Conditions)
}

func assertSkillAsync(
	t *testing.T,
	resource *odav1beta1.Skill,
	source shared.OSOKAsyncSource,
	phase shared.OSOKAsyncPhase,
	class shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatalf("async current = nil, want source=%s phase=%s class=%s", source, phase, class)
	}
	if current.Source != source || current.Phase != phase || current.NormalizedClass != class {
		t.Fatalf("async current = %#v, want source=%s phase=%s class=%s", current, source, phase, class)
	}
}

func float32Ptr(value float32) *float32 {
	return &value
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

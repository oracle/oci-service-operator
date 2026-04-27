/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package skillparameter

import (
	"context"
	"errors"
	"strings"
	"testing"

	odasdk "github.com/oracle/oci-go-sdk/v65/oda"
	odav1beta1 "github.com/oracle/oci-service-operator/api/oda/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestSkillParameterRuntimeSemanticsEncodesLifecycleContract(t *testing.T) {
	semantics := newSkillParameterRuntimeSemantics()
	if semantics.FormalService != "oda" || semantics.FormalSlug != "skillparameter" {
		t.Fatalf("formal binding = %s/%s, want oda/skillparameter", semantics.FormalService, semantics.FormalSlug)
	}
	if semantics.Async == nil || semantics.Async.Strategy != "lifecycle" || semantics.Async.Runtime != "generatedruntime" {
		t.Fatalf("async semantics = %#v, want generatedruntime lifecycle", semantics.Async)
	}
	if !containsString(semantics.Lifecycle.ProvisioningStates, "CREATING") {
		t.Fatalf("provisioning states = %v, want CREATING", semantics.Lifecycle.ProvisioningStates)
	}
	if !containsString(semantics.Lifecycle.UpdatingStates, "UPDATING") {
		t.Fatalf("updating states = %v, want UPDATING", semantics.Lifecycle.UpdatingStates)
	}
	if !containsString(semantics.Lifecycle.ActiveStates, "ACTIVE") {
		t.Fatalf("active states = %v, want ACTIVE", semantics.Lifecycle.ActiveStates)
	}
	if !containsString(semantics.Lifecycle.ActiveStates, "INACTIVE") {
		t.Fatalf("active states = %v, want INACTIVE", semantics.Lifecycle.ActiveStates)
	}
	if !containsString(semantics.Delete.PendingStates, "DELETING") || !containsString(semantics.Delete.TerminalStates, "DELETED") {
		t.Fatalf("delete states pending=%v terminal=%v, want DELETING/DELETED", semantics.Delete.PendingStates, semantics.Delete.TerminalStates)
	}
	for _, field := range []string{"description", "displayName", "value"} {
		if !containsString(semantics.Mutation.Mutable, field) {
			t.Fatalf("mutable fields = %v, want %s", semantics.Mutation.Mutable, field)
		}
	}
	for _, field := range []string{"name", "type"} {
		if !containsString(semantics.Mutation.ForceNew, field) {
			t.Fatalf("force-new fields = %v, want %s", semantics.Mutation.ForceNew, field)
		}
	}
	if semantics.CreateFollowUp.Strategy != "read-after-write" ||
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

func TestSkillParameterServiceClientCreatesAndProjectsStatus(t *testing.T) {
	fake := newFakeSkillParameterOCIClient(t)
	fake.listResults = append(fake.listResults, listSkillParametersResult{
		response: odasdk.ListSkillParametersResponse{},
	})
	fake.getResults = append(fake.getResults,
		getSkillParameterResult{err: errSkillParameterNotFound},
		getSkillParameterResult{response: getSkillParameterResponse("param", "Display", "STRING", "value", odasdk.LifecycleStateActive, "desc")},
	)
	client := newSkillParameterServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	resource := testSkillParameter()

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful without requeue", response)
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("create requests = %d, want 1", len(fake.createRequests))
	}
	create := fake.createRequests[0]
	if got := stringValue(create.OdaInstanceId); got != "oda-1" {
		t.Fatalf("create OdaInstanceId = %q, want oda-1", got)
	}
	if got := stringValue(create.SkillId); got != "skill-1" {
		t.Fatalf("create SkillId = %q, want skill-1", got)
	}
	if got := stringValue(create.CreateSkillParameterDetails.Name); got != "param" {
		t.Fatalf("create name = %q, want param", got)
	}
	if got := string(create.CreateSkillParameterDetails.Type); got != "STRING" {
		t.Fatalf("create type = %q, want STRING", got)
	}
	assertSkillParameterStatus(t, resource, "param", "Display", "STRING", "value", "ACTIVE", "desc", shared.Active)
	assertTrackedSkillParameterFingerprint(t, resource, skillParameterIdentity{odaInstanceID: "oda-1", skillID: "skill-1", name: "param"})
}

func TestSkillParameterTrackedIdentityIsBoundedForLongParents(t *testing.T) {
	fake := newFakeSkillParameterOCIClient(t)
	fake.listResults = append(fake.listResults, listSkillParametersResult{
		response: odasdk.ListSkillParametersResponse{},
	})

	longOdaInstanceID := "ocid1.odainstance.oc1.." + strings.Repeat("a", 220)
	longSkillID := "ocid1.odaskill.oc1.." + strings.Repeat("b", 220)
	longName := "param-" + strings.Repeat("c", 120)
	fake.getResults = append(fake.getResults,
		getSkillParameterResult{err: errSkillParameterNotFound},
		getSkillParameterResult{response: getSkillParameterResponse(longName, "Display", "STRING", "value", odasdk.LifecycleStateActive, "desc")},
	)
	client := newSkillParameterServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	resource := testSkillParameter()
	resource.Annotations[skillParameterOdaInstanceIDAnnotation] = longOdaInstanceID
	resource.Annotations[skillParameterSkillIDAnnotation] = longSkillID
	resource.Spec.Name = longName

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful", response)
	}

	trackedID := string(resource.Status.OsokStatus.Ocid)
	if len(trackedID) > 255 {
		t.Fatalf("status.ocid length = %d, want <= 255: %q", len(trackedID), trackedID)
	}
	for _, rawPart := range []string{longOdaInstanceID, longSkillID, longName} {
		if strings.Contains(trackedID, rawPart) {
			t.Fatalf("status.ocid = %q contains raw identity part %q", trackedID, rawPart)
		}
	}
	assertTrackedSkillParameterFingerprint(t, resource, skillParameterIdentity{odaInstanceID: longOdaInstanceID, skillID: longSkillID, name: longName})
}

func TestSkillParameterServiceClientBindsExistingWithoutCreate(t *testing.T) {
	fake := newFakeSkillParameterOCIClient(t)
	fake.listResults = append(fake.listResults, listSkillParametersResult{
		response: odasdk.ListSkillParametersResponse{
			SkillParameterCollection: odasdk.SkillParameterCollection{
				Items: []odasdk.SkillParameterSummary{
					skillParameterSummary("param", "Bound", "STRING", "live", odasdk.LifecycleStateActive, "bound"),
				},
			},
		},
	})
	fake.getResults = append(fake.getResults, getSkillParameterResult{
		response: getSkillParameterResponse("param", "Bound", "STRING", "live", odasdk.LifecycleStateActive, "bound"),
	})
	client := newSkillParameterServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	resource := testSkillParameter()
	resource.Spec.DisplayName = "Bound"
	resource.Spec.Value = "live"
	resource.Spec.Description = "bound"

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
	assertSkillParameterStatus(t, resource, "param", "Bound", "STRING", "live", "ACTIVE", "bound", shared.Active)
}

func TestSkillParameterServiceClientBindsInactiveExistingWithoutCreate(t *testing.T) {
	fake := newFakeSkillParameterOCIClient(t)
	fake.listResults = append(fake.listResults, listSkillParametersResult{
		response: odasdk.ListSkillParametersResponse{
			SkillParameterCollection: odasdk.SkillParameterCollection{
				Items: []odasdk.SkillParameterSummary{
					skillParameterSummary("param", "Bound", "STRING", "live", odasdk.LifecycleStateInactive, "bound"),
				},
			},
		},
	})
	fake.getResults = append(fake.getResults, getSkillParameterResult{
		response: getSkillParameterResponse("param", "Bound", "STRING", "live", odasdk.LifecycleStateInactive, "bound"),
	})
	client := newSkillParameterServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	resource := testSkillParameter()
	resource.Spec.DisplayName = "Bound"
	resource.Spec.Value = "live"
	resource.Spec.Description = "bound"

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
	if len(fake.getRequests) != 1 {
		t.Fatalf("get requests = %d, want 1", len(fake.getRequests))
	}
	assertSkillParameterStatus(t, resource, "param", "Bound", "STRING", "live", "INACTIVE", "bound", shared.Active)
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("async current = %#v, want nil for steady INACTIVE lifecycle", resource.Status.OsokStatus.Async.Current)
	}
}

func TestSkillParameterServiceClientUpdatesSupportedMutableDrift(t *testing.T) {
	fake := newFakeSkillParameterOCIClient(t)
	fake.listResults = append(fake.listResults, listSkillParametersResult{
		response: odasdk.ListSkillParametersResponse{
			SkillParameterCollection: odasdk.SkillParameterCollection{
				Items: []odasdk.SkillParameterSummary{
					skillParameterSummary("param", "Old", "STRING", "old", odasdk.LifecycleStateActive, "old-desc"),
				},
			},
		},
	})
	fake.getResults = append(fake.getResults,
		getSkillParameterResult{response: getSkillParameterResponse("param", "Old", "STRING", "old", odasdk.LifecycleStateActive, "old-desc")},
		getSkillParameterResult{response: getSkillParameterResponse("param", "New", "STRING", "new", odasdk.LifecycleStateActive, "new-desc")},
	)
	client := newSkillParameterServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	resource := testSkillParameter()
	resource.Spec.DisplayName = "New"
	resource.Spec.Value = "new"
	resource.Spec.Description = "new-desc"

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
	if got := stringValue(update.ParameterName); got != "param" {
		t.Fatalf("update ParameterName = %q, want param", got)
	}
	if got := stringValue(update.UpdateSkillParameterDetails.DisplayName); got != "New" {
		t.Fatalf("update displayName = %q, want New", got)
	}
	if got := stringValue(update.UpdateSkillParameterDetails.Description); got != "new-desc" {
		t.Fatalf("update description = %q, want new-desc", got)
	}
	if got := stringValue(update.UpdateSkillParameterDetails.Value); got != "new" {
		t.Fatalf("update value = %q, want new", got)
	}
	assertSkillParameterStatus(t, resource, "param", "New", "STRING", "new", "ACTIVE", "new-desc", shared.Active)
}

func TestSkillParameterServiceClientUpdatesInactiveMutableDrift(t *testing.T) {
	fake := newFakeSkillParameterOCIClient(t)
	fake.listResults = append(fake.listResults, listSkillParametersResult{
		response: odasdk.ListSkillParametersResponse{
			SkillParameterCollection: odasdk.SkillParameterCollection{
				Items: []odasdk.SkillParameterSummary{
					skillParameterSummary("param", "Old", "STRING", "old", odasdk.LifecycleStateInactive, "old-desc"),
				},
			},
		},
	})
	fake.getResults = append(fake.getResults,
		getSkillParameterResult{response: getSkillParameterResponse("param", "Old", "STRING", "old", odasdk.LifecycleStateInactive, "old-desc")},
		getSkillParameterResult{response: getSkillParameterResponse("param", "New", "STRING", "new", odasdk.LifecycleStateInactive, "new-desc")},
	)
	client := newSkillParameterServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	resource := testSkillParameter()
	resource.Spec.DisplayName = "New"
	resource.Spec.Value = "new"
	resource.Spec.Description = "new-desc"

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
	if got := stringValue(update.ParameterName); got != "param" {
		t.Fatalf("update ParameterName = %q, want param", got)
	}
	if got := stringValue(update.UpdateSkillParameterDetails.DisplayName); got != "New" {
		t.Fatalf("update displayName = %q, want New", got)
	}
	if got := stringValue(update.UpdateSkillParameterDetails.Description); got != "new-desc" {
		t.Fatalf("update description = %q, want new-desc", got)
	}
	if got := stringValue(update.UpdateSkillParameterDetails.Value); got != "new" {
		t.Fatalf("update value = %q, want new", got)
	}
	assertSkillParameterStatus(t, resource, "param", "New", "STRING", "new", "INACTIVE", "new-desc", shared.Active)
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("async current = %#v, want nil for steady INACTIVE lifecycle", resource.Status.OsokStatus.Async.Current)
	}
}

func TestSkillParameterServiceClientRejectsCreateOnlyTypeDriftBeforeUpdate(t *testing.T) {
	fake := newFakeSkillParameterOCIClient(t)
	fake.listResults = append(fake.listResults, listSkillParametersResult{
		response: odasdk.ListSkillParametersResponse{
			SkillParameterCollection: odasdk.SkillParameterCollection{
				Items: []odasdk.SkillParameterSummary{
					skillParameterSummary("param", "Display", "STRING", "value", odasdk.LifecycleStateActive, "desc"),
				},
			},
		},
	})
	fake.getResults = append(fake.getResults, getSkillParameterResult{
		response: getSkillParameterResponse("param", "Display", "STRING", "value", odasdk.LifecycleStateActive, "desc"),
	})
	client := newSkillParameterServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	resource := testSkillParameter()
	resource.Spec.Type = "INTEGER"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want immutable type drift error")
	}
	if !strings.Contains(err.Error(), "type is immutable") {
		t.Fatalf("CreateOrUpdate() error = %v, want immutable type detail", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0", len(fake.updateRequests))
	}
	if got := lastConditionType(resource); got != shared.Failed {
		t.Fatalf("last condition = %s, want Failed", got)
	}
}

func TestSkillParameterServiceClientRequeuesIntermediateLifecycle(t *testing.T) {
	fake := newFakeSkillParameterOCIClient(t)
	fake.listResults = append(fake.listResults, listSkillParametersResult{
		response: odasdk.ListSkillParametersResponse{
			SkillParameterCollection: odasdk.SkillParameterCollection{
				Items: []odasdk.SkillParameterSummary{
					skillParameterSummary("param", "Display", "STRING", "value", odasdk.LifecycleStateCreating, "desc"),
				},
			},
		},
	})
	fake.getResults = append(fake.getResults, getSkillParameterResult{
		response: getSkillParameterResponse("param", "Display", "STRING", "value", odasdk.LifecycleStateCreating, "desc"),
	})
	client := newSkillParameterServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	resource := testSkillParameter()

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue", response)
	}
	if got := lastConditionType(resource); got != shared.Provisioning {
		t.Fatalf("last condition = %s, want Provisioning", got)
	}
	if resource.Status.OsokStatus.Async.Current == nil ||
		resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseCreate ||
		resource.Status.OsokStatus.Async.Current.RawStatus != "CREATING" {
		t.Fatalf("async current = %#v, want create/CREATING", resource.Status.OsokStatus.Async.Current)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0", len(fake.updateRequests))
	}
}

func TestSkillParameterServiceClientFailsTerminalFailureLifecycle(t *testing.T) {
	fake := newFakeSkillParameterOCIClient(t)
	fake.listResults = append(fake.listResults, listSkillParametersResult{
		response: odasdk.ListSkillParametersResponse{
			SkillParameterCollection: odasdk.SkillParameterCollection{
				Items: []odasdk.SkillParameterSummary{
					skillParameterSummary("param", "Display", "STRING", "value", odasdk.LifecycleStateFailed, "desc"),
				},
			},
		},
	})
	fake.getResults = append(fake.getResults, getSkillParameterResult{
		response: getSkillParameterResponse("param", "Display", "STRING", "value", odasdk.LifecycleStateFailed, "desc"),
	})
	client := newSkillParameterServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	resource := testSkillParameter()

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful without requeue", response)
	}
	if got := lastConditionType(resource); got != shared.Failed {
		t.Fatalf("last condition = %s, want Failed", got)
	}
	if resource.Status.OsokStatus.Async.Current == nil ||
		resource.Status.OsokStatus.Async.Current.NormalizedClass != shared.OSOKAsyncClassFailed {
		t.Fatalf("async current = %#v, want failed lifecycle", resource.Status.OsokStatus.Async.Current)
	}
}

func TestSkillParameterDeleteKeepsFinalizerWhileDeleting(t *testing.T) {
	fake := newFakeSkillParameterOCIClient(t)
	fake.getResults = append(fake.getResults,
		getSkillParameterResult{response: getSkillParameterResponse("param", "Display", "STRING", "value", odasdk.LifecycleStateActive, "desc")},
		getSkillParameterResult{response: getSkillParameterResponse("param", "Display", "STRING", "value", odasdk.LifecycleStateDeleting, "desc")},
	)
	client := newSkillParameterServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	resource := testSkillParameter()

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while OCI reports DELETING")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("delete requests = %d, want 1", len(fake.deleteRequests))
	}
	if got := stringValue(fake.deleteRequests[0].ParameterName); got != "param" {
		t.Fatalf("delete ParameterName = %q, want param", got)
	}
	if got := lastConditionType(resource); got != shared.Terminating {
		t.Fatalf("last condition = %s, want Terminating", got)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("DeletedAt = %v, want nil while delete is pending", resource.Status.OsokStatus.DeletedAt)
	}
}

func TestSkillParameterDeleteReleasesFinalizerWhenNotFound(t *testing.T) {
	fake := newFakeSkillParameterOCIClient(t)
	fake.getResults = append(fake.getResults, getSkillParameterResult{err: errSkillParameterNotFound})
	client := newSkillParameterServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	resource := testSkillParameter()

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true when OCI read is not found")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("delete requests = %d, want 0", len(fake.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("DeletedAt = nil, want deletion timestamp")
	}
	if got := lastConditionType(resource); got != shared.Terminating {
		t.Fatalf("last condition = %s, want Terminating", got)
	}
}

func TestSkillParameterDeleteReleasesFinalizerWhenNoTrackedIdentityAndParentsMissing(t *testing.T) {
	fake := newFakeSkillParameterOCIClient(t)
	client := newSkillParameterServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	resource := testSkillParameter()
	resource.Annotations = nil

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true when no OCI identity was ever recorded")
	}
	if len(fake.getRequests) != 0 || len(fake.deleteRequests) != 0 {
		t.Fatalf("OCI requests get=%d delete=%d, want none", len(fake.getRequests), len(fake.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("DeletedAt = nil, want deletion timestamp")
	}
	if !strings.Contains(resource.Status.OsokStatus.Message, "No tracked OCI SkillParameter identity recorded") {
		t.Fatalf("status message = %q, want no tracked identity detail", resource.Status.OsokStatus.Message)
	}
}

func TestSkillParameterRequiresParentAnnotations(t *testing.T) {
	fake := newFakeSkillParameterOCIClient(t)
	client := newSkillParameterServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	resource := testSkillParameter()
	resource.Annotations = nil

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want missing annotation error")
	}
	if !strings.Contains(err.Error(), skillParameterOdaInstanceIDAnnotation) ||
		!strings.Contains(err.Error(), skillParameterSkillIDAnnotation) {
		t.Fatalf("CreateOrUpdate() error = %v, want both parent annotation names", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if len(fake.listRequests) != 0 {
		t.Fatalf("list requests = %d, want 0", len(fake.listRequests))
	}
	if got := lastConditionType(resource); got != shared.Failed {
		t.Fatalf("last condition = %s, want Failed", got)
	}
}

func TestSkillParameterRejectsTrackedIdentityDrift(t *testing.T) {
	fake := newFakeSkillParameterOCIClient(t)
	client := newSkillParameterServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	resource := testSkillParameter()
	resource.Status.OsokStatus.Ocid = skillParameterIdentity{odaInstanceID: "oda-old", skillID: "skill-1", name: "param"}.syntheticOCID()

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want immutable identity drift error")
	}
	if !strings.Contains(err.Error(), "identity is immutable") {
		t.Fatalf("CreateOrUpdate() error = %v, want immutable identity detail", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if len(fake.listRequests) != 0 {
		t.Fatalf("list requests = %d, want 0", len(fake.listRequests))
	}
}

func TestSkillParameterDeleteRequiresParentsWhenOnlyBoundedTrackedIdentityExists(t *testing.T) {
	fake := newFakeSkillParameterOCIClient(t)
	client := newSkillParameterServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	resource := testSkillParameter()
	resource.Annotations = nil
	resource.Status.OsokStatus.Ocid = skillParameterIdentity{odaInstanceID: "oda-old", skillID: "skill-old", name: "old-param"}.syntheticOCID()

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want bounded identity parent annotation error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if !strings.Contains(err.Error(), "bounded identity fingerprint") {
		t.Fatalf("Delete() error = %v, want bounded identity detail", err)
	}
	if len(fake.getRequests) != 0 || len(fake.deleteRequests) != 0 {
		t.Fatalf("OCI requests get=%d delete=%d, want none", len(fake.getRequests), len(fake.deleteRequests))
	}
	if got := lastConditionType(resource); got != shared.Failed {
		t.Fatalf("last condition = %s, want Failed", got)
	}
}

func TestSkillParameterDeleteRejectsDriftedAnnotationsWhenBoundedTrackedIdentityExists(t *testing.T) {
	fake := newFakeSkillParameterOCIClient(t)
	client := newSkillParameterServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	resource := testSkillParameter()
	resource.Status.OsokStatus.Ocid = skillParameterIdentity{
		odaInstanceID: "oda-old",
		skillID:       "skill-old",
		name:          "old-param",
	}.syntheticOCID()
	resource.Annotations[skillParameterOdaInstanceIDAnnotation] = "oda-new"
	resource.Annotations[skillParameterSkillIDAnnotation] = "skill-new"
	resource.Spec.Name = "new-param"

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want immutable identity drift error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if !strings.Contains(err.Error(), "identity is immutable") {
		t.Fatalf("Delete() error = %v, want immutable identity detail", err)
	}
	if len(fake.getRequests) != 0 || len(fake.deleteRequests) != 0 {
		t.Fatalf("OCI requests get=%d delete=%d, want none", len(fake.getRequests), len(fake.deleteRequests))
	}
	if got := lastConditionType(resource); got != shared.Failed {
		t.Fatalf("last condition = %s, want Failed", got)
	}
}

func TestSkillParameterDeleteUsesLegacyTrackedIdentity(t *testing.T) {
	fake := newFakeSkillParameterOCIClient(t)
	fake.getResults = append(fake.getResults, getSkillParameterResult{err: errSkillParameterNotFound})
	client := newSkillParameterServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	resource := testSkillParameter()
	resource.Annotations = nil
	resource.Status.OsokStatus.Ocid = legacySyntheticSkillParameterOCID(skillParameterIdentity{odaInstanceID: "oda-old", skillID: "skill-old", name: "old-param"})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true")
	}
	if len(fake.getRequests) != 1 {
		t.Fatalf("get requests = %d, want 1", len(fake.getRequests))
	}
	get := fake.getRequests[0]
	if stringValue(get.OdaInstanceId) != "oda-old" ||
		stringValue(get.SkillId) != "skill-old" ||
		stringValue(get.ParameterName) != "old-param" {
		t.Fatalf(
			"get identity = %q/%q/%q, want tracked old identity",
			stringValue(get.OdaInstanceId),
			stringValue(get.SkillId),
			stringValue(get.ParameterName),
		)
	}
}

type fakeSkillParameterOCIClient struct {
	t testing.TB

	listResults   []listSkillParametersResult
	getResults    []getSkillParameterResult
	createResults []createSkillParameterResult
	updateResults []updateSkillParameterResult
	deleteResults []deleteSkillParameterResult

	listRequests   []odasdk.ListSkillParametersRequest
	getRequests    []odasdk.GetSkillParameterRequest
	createRequests []odasdk.CreateSkillParameterRequest
	updateRequests []odasdk.UpdateSkillParameterRequest
	deleteRequests []odasdk.DeleteSkillParameterRequest
}

type listSkillParametersResult struct {
	response odasdk.ListSkillParametersResponse
	err      error
}

type getSkillParameterResult struct {
	response odasdk.GetSkillParameterResponse
	err      error
}

type createSkillParameterResult struct {
	response odasdk.CreateSkillParameterResponse
	err      error
}

type updateSkillParameterResult struct {
	response odasdk.UpdateSkillParameterResponse
	err      error
}

type deleteSkillParameterResult struct {
	response odasdk.DeleteSkillParameterResponse
	err      error
}

func newFakeSkillParameterOCIClient(t testing.TB) *fakeSkillParameterOCIClient {
	return &fakeSkillParameterOCIClient{t: t}
}

func (f *fakeSkillParameterOCIClient) CreateSkillParameter(
	_ context.Context,
	request odasdk.CreateSkillParameterRequest,
) (odasdk.CreateSkillParameterResponse, error) {
	f.t.Helper()
	f.createRequests = append(f.createRequests, request)
	if len(f.createResults) == 0 {
		return createSkillParameterResponse(
			stringValue(request.CreateSkillParameterDetails.Name),
			stringValue(request.CreateSkillParameterDetails.DisplayName),
			string(request.CreateSkillParameterDetails.Type),
			stringValue(request.CreateSkillParameterDetails.Value),
			odasdk.LifecycleStateCreating,
			stringValue(request.CreateSkillParameterDetails.Description),
		), nil
	}
	result := f.createResults[0]
	f.createResults = f.createResults[1:]
	return result.response, result.err
}

func (f *fakeSkillParameterOCIClient) GetSkillParameter(
	_ context.Context,
	request odasdk.GetSkillParameterRequest,
) (odasdk.GetSkillParameterResponse, error) {
	f.t.Helper()
	f.getRequests = append(f.getRequests, request)
	if len(f.getResults) == 0 {
		f.t.Fatalf("unexpected GetSkillParameter request: %#v", request)
	}
	result := f.getResults[0]
	f.getResults = f.getResults[1:]
	return result.response, result.err
}

func (f *fakeSkillParameterOCIClient) ListSkillParameters(
	_ context.Context,
	request odasdk.ListSkillParametersRequest,
) (odasdk.ListSkillParametersResponse, error) {
	f.t.Helper()
	f.listRequests = append(f.listRequests, request)
	if len(f.listResults) == 0 {
		f.t.Fatalf("unexpected ListSkillParameters request: %#v", request)
	}
	result := f.listResults[0]
	f.listResults = f.listResults[1:]
	return result.response, result.err
}

func (f *fakeSkillParameterOCIClient) UpdateSkillParameter(
	_ context.Context,
	request odasdk.UpdateSkillParameterRequest,
) (odasdk.UpdateSkillParameterResponse, error) {
	f.t.Helper()
	f.updateRequests = append(f.updateRequests, request)
	if len(f.updateResults) == 0 {
		return updateSkillParameterResponse(
			stringValue(request.ParameterName),
			stringValue(request.UpdateSkillParameterDetails.DisplayName),
			"STRING",
			stringValue(request.UpdateSkillParameterDetails.Value),
			odasdk.LifecycleStateUpdating,
			stringValue(request.UpdateSkillParameterDetails.Description),
		), nil
	}
	result := f.updateResults[0]
	f.updateResults = f.updateResults[1:]
	return result.response, result.err
}

func (f *fakeSkillParameterOCIClient) DeleteSkillParameter(
	_ context.Context,
	request odasdk.DeleteSkillParameterRequest,
) (odasdk.DeleteSkillParameterResponse, error) {
	f.t.Helper()
	f.deleteRequests = append(f.deleteRequests, request)
	if len(f.deleteResults) == 0 {
		return odasdk.DeleteSkillParameterResponse{}, nil
	}
	result := f.deleteResults[0]
	f.deleteResults = f.deleteResults[1:]
	return result.response, result.err
}

func testSkillParameter() *odav1beta1.SkillParameter {
	return &odav1beta1.SkillParameter{
		ObjectMeta: metav1.ObjectMeta{
			Name: "param-cr",
			Annotations: map[string]string{
				skillParameterOdaInstanceIDAnnotation: "oda-1",
				skillParameterSkillIDAnnotation:       "skill-1",
			},
		},
		Spec: odav1beta1.SkillParameterSpec{
			Name:        "param",
			DisplayName: "Display",
			Type:        "STRING",
			Value:       "value",
			Description: "desc",
		},
	}
}

func getSkillParameterResponse(
	name string,
	displayName string,
	parameterType string,
	value string,
	lifecycleState odasdk.LifecycleStateEnum,
	description string,
) odasdk.GetSkillParameterResponse {
	return odasdk.GetSkillParameterResponse{
		SkillParameter: skillParameter(name, displayName, parameterType, value, lifecycleState, description),
	}
}

func createSkillParameterResponse(
	name string,
	displayName string,
	parameterType string,
	value string,
	lifecycleState odasdk.LifecycleStateEnum,
	description string,
) odasdk.CreateSkillParameterResponse {
	return odasdk.CreateSkillParameterResponse{
		SkillParameter: skillParameter(name, displayName, parameterType, value, lifecycleState, description),
	}
}

func updateSkillParameterResponse(
	name string,
	displayName string,
	parameterType string,
	value string,
	lifecycleState odasdk.LifecycleStateEnum,
	description string,
) odasdk.UpdateSkillParameterResponse {
	return odasdk.UpdateSkillParameterResponse{
		SkillParameter: skillParameter(name, displayName, parameterType, value, lifecycleState, description),
	}
}

func skillParameter(
	name string,
	displayName string,
	parameterType string,
	value string,
	lifecycleState odasdk.LifecycleStateEnum,
	description string,
) odasdk.SkillParameter {
	return odasdk.SkillParameter{
		Name:           stringPtr(name),
		DisplayName:    stringPtr(displayName),
		Type:           odasdk.ParameterTypeEnum(parameterType),
		Value:          stringPtr(value),
		LifecycleState: lifecycleState,
		Description:    stringPtr(description),
	}
}

func skillParameterSummary(
	name string,
	displayName string,
	parameterType string,
	value string,
	lifecycleState odasdk.LifecycleStateEnum,
	description string,
) odasdk.SkillParameterSummary {
	return odasdk.SkillParameterSummary{
		Name:           stringPtr(name),
		DisplayName:    stringPtr(displayName),
		Type:           odasdk.ParameterTypeEnum(parameterType),
		Value:          stringPtr(value),
		LifecycleState: lifecycleState,
		Description:    stringPtr(description),
	}
}

func assertSkillParameterStatus(
	t *testing.T,
	resource *odav1beta1.SkillParameter,
	name string,
	displayName string,
	parameterType string,
	value string,
	lifecycleState string,
	description string,
	condition shared.OSOKConditionType,
) {
	t.Helper()
	if resource.Status.Name != name ||
		resource.Status.DisplayName != displayName ||
		resource.Status.Type != parameterType ||
		resource.Status.Value != value ||
		resource.Status.LifecycleState != lifecycleState ||
		resource.Status.Description != description {
		t.Fatalf("status = %#v, want %q/%q/%q/%q/%q/%q", resource.Status, name, displayName, parameterType, value, lifecycleState, description)
	}
	if got := lastConditionType(resource); got != condition {
		t.Fatalf("last condition = %s, want %s", got, condition)
	}
}

func assertTrackedSkillParameterFingerprint(
	t *testing.T,
	resource *odav1beta1.SkillParameter,
	want skillParameterIdentity,
) {
	t.Helper()
	got, ok := trackedSkillParameterFingerprint(resource)
	if !ok {
		t.Fatalf("tracked identity fingerprint missing from status ocid %q", resource.Status.OsokStatus.Ocid)
	}
	if got != want.fingerprint() {
		t.Fatalf("tracked identity fingerprint = %q, want %q", got, want.fingerprint())
	}
	if len(resource.Status.OsokStatus.Ocid) > 255 {
		t.Fatalf("status.ocid length = %d, want <= 255", len(resource.Status.OsokStatus.Ocid))
	}
}

func lastConditionType(resource *odav1beta1.SkillParameter) shared.OSOKConditionType {
	if resource == nil || len(resource.Status.OsokStatus.Conditions) == 0 {
		return ""
	}
	return resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func TestSkillParameterNotFoundClassifierRecognizesStringErrors(t *testing.T) {
	if !skillParameterIsNotFound(errors.New("404 NotFound")) {
		t.Fatal("skillParameterIsNotFound() = false, want true for 404 NotFound string")
	}
}

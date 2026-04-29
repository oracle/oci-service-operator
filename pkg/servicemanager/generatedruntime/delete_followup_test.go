/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"context"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"strings"
	"testing"
)

func TestServiceClientDeleteConfirmDeleteReadUsesMatrix(t *testing.T) {
	t.Parallel()
	registration := errortest.ReviewedRegistrationForFamily(t, "opensearch", "OpensearchCluster", errortest.APIErrorCoverageFamilyGeneratedRuntimeFollowUp)
	if !strings.Contains(registration.Deviation, "confirm-delete") {
		t.Fatalf("reviewed registration = %s, want explicit confirm-delete note", errortest.DescribeReviewedRegistration(registration))
	}
	focused := errortest.FocusedAsyncFollowUpCases(t)
	cases := []errortest.AsyncFollowUpMatrixCase{{Candidate: focused["notfound"], WantDeleted: true}, {Candidate: focused["auth404"], WantDeleted: true}, {Candidate: focused["conflict"], WantErrorType: focused["conflict"].NormalizedType}, {Candidate: focused["internal"], WantErrorType: focused["internal"].NormalizedType}, {Candidate: focused["unavailable"], WantErrorType: focused["unavailable"].NormalizedType}}
	errortest.RunAsyncFollowUpMatrix(t, cases, func(t *testing.T, candidate errortest.CommonErrorCase) errortest.AsyncFollowUpResult {
		getCalls := 0
		client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "OpensearchCluster", SDKName: "OpensearchCluster", Semantics: &Semantics{Delete: DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING"}, TerminalStates: []string{"DELETED"}}, List: &ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"displayName"}}, DeleteFollowUp: FollowUpSemantics{Strategy: "confirm-delete", Hooks: []Hook{{Helper: "tfresource.DeleteResource"}}}}, Delete: &Operation{NewRequest: func() any {
			return &fakeDeleteThingRequest{}
		}, Call: func(_ context.Context, _ any) (any, error) {
			return fakeDeleteThingResponse{}, nil
		}, Fields: []RequestField{{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true}}}, Get: &Operation{NewRequest: func() any {
			return &fakeGetThingRequest{}
		}, Call: func(_ context.Context, _ any) (any, error) {
			getCalls++
			if getCalls == 1 {
				return fakeGetThingResponse{Thing: fakeThing{Id: "ocid1.opensearchcluster.oc1..existing", DisplayName: "cluster-sample", LifecycleState: "ACTIVE"}}, nil
			}
			return nil, errortest.NewServiceErrorFromCase(candidate)
		}, Fields: []RequestField{{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true}}}, List: &Operation{NewRequest: func() any {
			return &fakeListThingRequest{}
		}, Call: func(_ context.Context, _ any) (any, error) {
			return fakeListThingResponse{Collection: fakeThingCollection{Items: nil}}, nil
		}, Fields: []RequestField{{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query"}}}})
		resource := &fakeResource{Spec: fakeSpec{DisplayName: "cluster-sample"}, Status: fakeStatus{OsokStatus: shared.OSOKStatus{Ocid: "ocid1.opensearchcluster.oc1..existing"}, Id: "ocid1.opensearchcluster.oc1..existing"}}
		deleted, err := client.Delete(context.Background(), resource)
		if err == nil && deleted && resource.Status.OsokStatus.DeletedAt == nil {
			t.Fatal("status.deletedAt should be set after confirmed delete")
		}
		return errortest.AsyncFollowUpResult{Err: err, Deleted: deleted}
	})
}

func TestServiceClientDeleteFollowUpPreservesSeededWorkRequestID(t *testing.T) {
	t.Parallel()
	getCalls := 0
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "OpensearchCluster", SDKName: "OpensearchCluster", Semantics: &Semantics{Delete: DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING"}, TerminalStates: []string{"DELETED"}}, DeleteFollowUp: FollowUpSemantics{Strategy: "confirm-delete", Hooks: []Hook{{Helper: "tfresource.DeleteResource"}}}}, Delete: &Operation{NewRequest: func() any {
		return &fakeDeleteThingRequest{}
	}, Call: func(_ context.Context, _ any) (any, error) {
		return fakeDeleteThingResponse{OpcWorkRequestId: stringPtr("wr-delete-followup")}, nil
	}, Fields: []RequestField{{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true}}}, Get: &Operation{NewRequest: func() any {
		return &fakeGetThingRequest{}
	}, Call: func(_ context.Context, _ any) (any, error) {
		getCalls++
		lifecycleState := "ACTIVE"
		if getCalls > 1 {
			lifecycleState = "DELETING"
		}
		return fakeGetThingResponse{Thing: fakeThing{Id: "ocid1.opensearchcluster.oc1..existing", DisplayName: "cluster-sample", LifecycleState: lifecycleState}}, nil
	}, Fields: []RequestField{{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true}}}})
	resource := &fakeResource{Spec: fakeSpec{DisplayName: "cluster-sample"}, Status: fakeStatus{OsokStatus: shared.OSOKStatus{Ocid: "ocid1.opensearchcluster.oc1..existing"}, Id: "ocid1.opensearchcluster.oc1..existing"}}
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() should keep waiting while confirm-delete follow-up stays DELETING")
	}
	requireCurrentWorkRequestID(t, resource, "wr-delete-followup")
	requireCurrentAsyncSource(t, resource, shared.OSOKAsyncSourceLifecycle)
	requireCurrentAsyncPhase(t, resource, shared.OSOKAsyncPhaseDelete)
	requireTrailingCondition(t, resource, shared.Terminating)
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt should stay empty while delete follow-up remains pending")
	}
}

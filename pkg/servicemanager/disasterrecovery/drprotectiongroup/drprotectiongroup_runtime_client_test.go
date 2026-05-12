/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package drprotectiongroup

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	disasterrecoverysdk "github.com/oracle/oci-go-sdk/v65/disasterrecovery"
	disasterrecoveryv1beta1 "github.com/oracle/oci-service-operator/api/disasterrecovery/v1beta1"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type drProtectionGroupRequestBodyBuilder interface {
	HTTPRequest(
		method string,
		path string,
		binaryRequestBody *common.OCIReadSeekCloser,
		extraHeaders map[string]string,
	) (http.Request, error)
}

func TestNormalizeDrProtectionGroupDesiredStateClearsListFilters(t *testing.T) {
	t.Parallel()

	resource := newDrProtectionGroupTestResource()
	resource.Spec.LifecycleState = "ACTIVE"
	resource.Spec.Role = "PRIMARY"
	resource.Spec.LifecycleSubState = "DR_PLAN_EXECUTION_IN_PROGRESS"

	normalizeDrProtectionGroupDesiredState(resource, nil)

	if resource.Spec.LifecycleState != "" {
		t.Fatalf("resource.Spec.LifecycleState = %q, want empty string", resource.Spec.LifecycleState)
	}
	if resource.Spec.Role != "" {
		t.Fatalf("resource.Spec.Role = %q, want empty string", resource.Spec.Role)
	}
	if resource.Spec.LifecycleSubState != "" {
		t.Fatalf("resource.Spec.LifecycleSubState = %q, want empty string", resource.Spec.LifecycleSubState)
	}
}

func TestBuildDrProtectionGroupCreateBodyIncludesAssociationAndPolymorphicMemberJSONData(t *testing.T) {
	t.Parallel()

	resource := newDrProtectionGroupTestResource()
	resource.Spec.Association = disasterrecoveryv1beta1.DrProtectionGroupAssociation{
		Role:       "STANDBY",
		PeerId:     "ocid1.drprotectiongroup.oc1..peer",
		PeerRegion: "us-ashburn-1",
	}
	resource.Spec.Members = []disasterrecoveryv1beta1.DrProtectionGroupMember{
		{
			MemberId:   "ocid1.instance.oc1..member",
			MemberType: "COMPUTE_INSTANCE",
			JsonData:   `{"isMovable":false}`,
		},
	}
	resource.Spec.FreeformTags = map[string]string{"managed-by": "osok"}

	details, err := buildDrProtectionGroupCreateBody(resource)
	if err != nil {
		t.Fatalf("buildDrProtectionGroupCreateBody() error = %v", err)
	}
	if details.Association == nil {
		t.Fatal("details.Association = nil, want populated association details")
	}
	if got := string(details.Association.Role); got != "STANDBY" {
		t.Fatalf("details.Association.Role = %q, want STANDBY", got)
	}
	if got := len(details.Members); got != 1 {
		t.Fatalf("len(details.Members) = %d, want 1", got)
	}

	member, ok := details.Members[0].(disasterrecoverysdk.CreateDrProtectionGroupMemberComputeInstanceDetails)
	if !ok {
		t.Fatalf("details.Members[0] type = %T, want CreateDrProtectionGroupMemberComputeInstanceDetails", details.Members[0])
	}
	if member.IsMovable == nil || *member.IsMovable {
		t.Fatalf("member.IsMovable = %#v, want explicit false", member.IsMovable)
	}

	body := drProtectionGroupSerializedRequestBody(t, disasterrecoverysdk.CreateDrProtectionGroupRequest{
		CreateDrProtectionGroupDetails: details,
	}, http.MethodPost, "/drProtectionGroups")
	for _, want := range []string{
		`"role":"STANDBY"`,
		`"peerId":"ocid1.drprotectiongroup.oc1..peer"`,
		`"peerRegion":"us-ashburn-1"`,
		`"memberType":"COMPUTE_INSTANCE"`,
		`"memberId":"ocid1.instance.oc1..member"`,
		`"isMovable":false`,
		`"managed-by":"osok"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("request body %s does not contain %s", body, want)
		}
	}
}

func TestBuildDrProtectionGroupUpdateBodyClearsTagsAndMembers(t *testing.T) {
	t.Parallel()

	resource := newDrProtectionGroupTestResource()

	current := disasterrecoverysdk.GetDrProtectionGroupResponse{
		DrProtectionGroup: disasterrecoverysdk.DrProtectionGroup{
			Id:             common.String("ocid1.drprotectiongroup.oc1..current"),
			CompartmentId:  common.String(resource.Spec.CompartmentId),
			DisplayName:    common.String(resource.Spec.DisplayName),
			Role:           disasterrecoverysdk.DrProtectionGroupRolePrimary,
			TimeCreated:    sdkTimePtr(time.Date(2026, time.May, 12, 10, 0, 0, 0, time.UTC)),
			TimeUpdated:    sdkTimePtr(time.Date(2026, time.May, 12, 10, 5, 0, 0, time.UTC)),
			LifecycleState: disasterrecoverysdk.DrProtectionGroupLifecycleStateActive,
			LogLocation: &disasterrecoverysdk.ObjectStorageLogLocation{
				Namespace: common.String(resource.Spec.LogLocation.Namespace),
				Bucket:    common.String(resource.Spec.LogLocation.Bucket),
				Object:    common.String("current/object"),
			},
			Members: []disasterrecoverysdk.DrProtectionGroupMember{
				disasterrecoverysdk.DrProtectionGroupMemberDatabase{
					MemberId:              common.String("ocid1.database.oc1..member"),
					PasswordVaultSecretId: common.String("ocid1.vaultsecret.oc1..secret"),
				},
			},
			FreeformTags: map[string]string{"env": "prod"},
		},
	}

	details, updateNeeded, err := buildDrProtectionGroupUpdateBody(resource, current)
	if err != nil {
		t.Fatalf("buildDrProtectionGroupUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildDrProtectionGroupUpdateBody() updateNeeded = false, want explicit clear")
	}

	body := drProtectionGroupSerializedRequestBody(t, disasterrecoverysdk.UpdateDrProtectionGroupRequest{
		DrProtectionGroupId:            common.String("ocid1.drprotectiongroup.oc1..current"),
		UpdateDrProtectionGroupDetails: details,
	}, http.MethodPut, "/drProtectionGroups/ocid1.drprotectiongroup.oc1..current")
	for _, want := range []string{
		`"members":[]`,
		`"freeformTags":{}`,
		`"definedTags":{}`,
		`"namespace":"tenant-a"`,
		`"bucket":"dr-logs"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("request body %s does not contain %s", body, want)
		}
	}
	if strings.Contains(body, `"object":"current/object"`) {
		t.Fatalf("request body %s unexpectedly contains logLocation.object", body)
	}
}

func TestBuildDrProtectionGroupUpdateBodyNoDiffWhenObservedMatchesDesired(t *testing.T) {
	t.Parallel()

	resource := newDrProtectionGroupTestResource()
	resource.Spec.Members = []disasterrecoveryv1beta1.DrProtectionGroupMember{
		{
			MemberId:   "ocid1.instance.oc1..member",
			MemberType: "COMPUTE_INSTANCE",
			JsonData:   `{"isMovable":false}`,
		},
	}

	current := disasterrecoverysdk.GetDrProtectionGroupResponse{
		DrProtectionGroup: disasterrecoverysdk.DrProtectionGroup{
			Id:             common.String("ocid1.drprotectiongroup.oc1..current"),
			CompartmentId:  common.String(resource.Spec.CompartmentId),
			DisplayName:    common.String(resource.Spec.DisplayName),
			Role:           disasterrecoverysdk.DrProtectionGroupRolePrimary,
			TimeCreated:    sdkTimePtr(time.Date(2026, time.May, 12, 10, 0, 0, 0, time.UTC)),
			TimeUpdated:    sdkTimePtr(time.Date(2026, time.May, 12, 10, 5, 0, 0, time.UTC)),
			LifecycleState: disasterrecoverysdk.DrProtectionGroupLifecycleStateActive,
			LogLocation: &disasterrecoverysdk.ObjectStorageLogLocation{
				Namespace: common.String(resource.Spec.LogLocation.Namespace),
				Bucket:    common.String(resource.Spec.LogLocation.Bucket),
			},
			Members: []disasterrecoverysdk.DrProtectionGroupMember{
				disasterrecoverysdk.DrProtectionGroupMemberComputeInstance{
					MemberId:  common.String("ocid1.instance.oc1..member"),
					IsMovable: common.Bool(false),
				},
			},
		},
	}

	_, updateNeeded, err := buildDrProtectionGroupUpdateBody(resource, current)
	if err != nil {
		t.Fatalf("buildDrProtectionGroupUpdateBody() error = %v", err)
	}
	if updateNeeded {
		t.Fatal("buildDrProtectionGroupUpdateBody() updateNeeded = true, want false for matching observed state")
	}
}

func TestValidateDrProtectionGroupCreateOnlyDriftRequiresAssociationMatch(t *testing.T) {
	t.Parallel()

	resource := newDrProtectionGroupTestResource()
	resource.Spec.Association = disasterrecoveryv1beta1.DrProtectionGroupAssociation{
		Role:       "STANDBY",
		PeerId:     "ocid1.drprotectiongroup.oc1..peer",
		PeerRegion: "us-ashburn-1",
	}

	current := disasterrecoverysdk.GetDrProtectionGroupResponse{
		DrProtectionGroup: disasterrecoverysdk.DrProtectionGroup{
			Id:             common.String("ocid1.drprotectiongroup.oc1..current"),
			CompartmentId:  common.String(resource.Spec.CompartmentId),
			DisplayName:    common.String(resource.Spec.DisplayName),
			Role:           disasterrecoverysdk.DrProtectionGroupRolePrimary,
			TimeCreated:    sdkTimePtr(time.Date(2026, time.May, 12, 10, 0, 0, 0, time.UTC)),
			TimeUpdated:    sdkTimePtr(time.Date(2026, time.May, 12, 10, 5, 0, 0, time.UTC)),
			LifecycleState: disasterrecoverysdk.DrProtectionGroupLifecycleStateActive,
			PeerId:         common.String("ocid1.drprotectiongroup.oc1..different"),
			PeerRegion:     common.String("us-phoenix-1"),
			LogLocation: &disasterrecoverysdk.ObjectStorageLogLocation{
				Namespace: common.String(resource.Spec.LogLocation.Namespace),
				Bucket:    common.String(resource.Spec.LogLocation.Bucket),
			},
		},
	}

	err := validateDrProtectionGroupCreateOnlyDriftForResponse(resource, current)
	if err == nil {
		t.Fatal("validateDrProtectionGroupCreateOnlyDriftForResponse() error = nil, want create-only association drift")
	}
}

func TestResolveAndRecoverDrProtectionGroupWorkRequest(t *testing.T) {
	t.Parallel()

	workRequest := disasterrecoverysdk.WorkRequest{
		OperationType: disasterrecoverysdk.OperationTypeUpdateDrProtectionGroup,
		Status:        disasterrecoverysdk.OperationStatusInProgress,
		Id:            common.String("wr-update-drpg"),
		Resources: []disasterrecoverysdk.WorkRequestResource{
			{
				EntityType: common.String("DrProtectionGroup"),
				ActionType: disasterrecoverysdk.ActionTypeUpdated,
				Identifier: common.String("ocid1.drprotectiongroup.oc1..current"),
			},
		},
	}

	action, err := resolveDrProtectionGroupGeneratedWorkRequestAction(workRequest)
	if err != nil {
		t.Fatalf("resolveDrProtectionGroupGeneratedWorkRequestAction() error = %v", err)
	}
	if action != string(disasterrecoverysdk.OperationTypeUpdateDrProtectionGroup) {
		t.Fatalf("action = %q, want %q", action, disasterrecoverysdk.OperationTypeUpdateDrProtectionGroup)
	}

	phase, ok, err := resolveDrProtectionGroupGeneratedWorkRequestPhase(workRequest)
	if err != nil {
		t.Fatalf("resolveDrProtectionGroupGeneratedWorkRequestPhase() error = %v", err)
	}
	if !ok || phase != shared.OSOKAsyncPhaseUpdate {
		t.Fatalf("phase = (%q, %t), want (%q, true)", phase, ok, shared.OSOKAsyncPhaseUpdate)
	}

	resourceID, err := recoverDrProtectionGroupIDFromGeneratedWorkRequest(nil, workRequest, shared.OSOKAsyncPhaseUpdate)
	if err != nil {
		t.Fatalf("recoverDrProtectionGroupIDFromGeneratedWorkRequest() error = %v", err)
	}
	if resourceID != "ocid1.drprotectiongroup.oc1..current" {
		t.Fatalf("resourceID = %q, want ocid1.drprotectiongroup.oc1..current", resourceID)
	}
}

func newDrProtectionGroupTestResource() *disasterrecoveryv1beta1.DrProtectionGroup {
	return &disasterrecoveryv1beta1.DrProtectionGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "drprotectiongroup-sample",
			Namespace: "default",
		},
		Spec: disasterrecoveryv1beta1.DrProtectionGroupSpec{
			CompartmentId: "ocid1.compartment.oc1..example",
			DisplayName:   "drprotectiongroup-sample",
			LogLocation: disasterrecoveryv1beta1.DrProtectionGroupLogLocation{
				Namespace: "tenant-a",
				Bucket:    "dr-logs",
			},
		},
	}
}

func sdkTimePtr(t time.Time) *common.SDKTime {
	value := common.SDKTime{Time: t}
	return &value
}

func drProtectionGroupSerializedRequestBody(
	t *testing.T,
	request drProtectionGroupRequestBodyBuilder,
	method string,
	path string,
) string {
	t.Helper()

	httpRequest, err := request.HTTPRequest(method, path, nil, nil)
	if err != nil {
		t.Fatalf("HTTPRequest() error = %v", err)
	}
	if httpRequest.Body == nil {
		return ""
	}

	body, err := io.ReadAll(httpRequest.Body)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}
	return string(body)
}

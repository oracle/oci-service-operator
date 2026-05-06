package lustrefilesystem

import (
	"context"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	lustrefilestoragesdk "github.com/oracle/oci-go-sdk/v65/lustrefilestorage"
	lustrefilestoragev1beta1 "github.com/oracle/oci-service-operator/api/lustrefilestorage/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestLustreFileSystemCreateRequestOmitsUnsetOptionalMaintenanceWindow(t *testing.T) {
	t.Parallel()

	const createdID = "ocid1.lustrefilesystem.oc1..created"

	resource := newLustreFileSystemTestResource()
	var createRequest lustrefilestoragesdk.CreateLustreFileSystemRequest

	manager := newLustreFileSystemRuntimeTestManager(generatedruntime.Config[*lustrefilestoragev1beta1.LustreFileSystem]{
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &lustrefilestoragesdk.CreateLustreFileSystemRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				createRequest = *request.(*lustrefilestoragesdk.CreateLustreFileSystemRequest)
				return lustrefilestoragesdk.CreateLustreFileSystemResponse{
					LustreFileSystem: observedLustreFileSystemFromSpec(createdID, resource.Spec, "CREATING", nil),
				}, nil
			},
			Fields: lustreFileSystemCreateFields(),
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &lustrefilestoragesdk.GetLustreFileSystemRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getRequest := *request.(*lustrefilestoragesdk.GetLustreFileSystemRequest)
				if getRequest.LustreFileSystemId == nil || *getRequest.LustreFileSystemId != createdID {
					t.Fatalf("GetLustreFileSystemRequest.LustreFileSystemId = %v, want %s", getRequest.LustreFileSystemId, createdID)
				}
				return lustrefilestoragesdk.GetLustreFileSystemResponse{
					LustreFileSystem: observedLustreFileSystemFromSpec(createdID, resource.Spec, "ACTIVE", nil),
				}, nil
			},
			Fields: lustreFileSystemGetFields(),
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success after the ACTIVE reread")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = true, want false after the ACTIVE reread")
	}
	if createRequest.DisplayName == nil || *createRequest.DisplayName != resource.Spec.DisplayName {
		t.Fatalf("CreateLustreFileSystemRequest.DisplayName = %v, want %q", createRequest.DisplayName, resource.Spec.DisplayName)
	}
	if createRequest.RootSquashConfiguration == nil {
		t.Fatal("CreateLustreFileSystemRequest.RootSquashConfiguration = nil, want populated root squash details")
	}
	if createRequest.RootSquashConfiguration.IdentitySquash != lustrefilestoragesdk.RootSquashConfigurationIdentitySquashRoot {
		t.Fatalf(
			"CreateLustreFileSystemRequest.RootSquashConfiguration.IdentitySquash = %q, want %q",
			createRequest.RootSquashConfiguration.IdentitySquash,
			lustrefilestoragesdk.RootSquashConfigurationIdentitySquashRoot,
		)
	}
	if createRequest.MaintenanceWindow != nil {
		t.Fatalf("CreateLustreFileSystemRequest.MaintenanceWindow = %#v, want nil when spec omitted the optional block", createRequest.MaintenanceWindow)
	}
}

func TestLustreFileSystemCreateOrUpdateClassifiesObservedLifecycleStates(t *testing.T) {
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

			const existingID = "ocid1.lustrefilesystem.oc1..existing"

			resource := newExistingLustreFileSystemTestResource(existingID)
			manager := newLustreFileSystemRuntimeTestManager(generatedruntime.Config[*lustrefilestoragev1beta1.LustreFileSystem]{
				Get: &generatedruntime.Operation{
					NewRequest: func() any { return &lustrefilestoragesdk.GetLustreFileSystemRequest{} },
					Call: func(_ context.Context, request any) (any, error) {
						getRequest := *request.(*lustrefilestoragesdk.GetLustreFileSystemRequest)
						if getRequest.LustreFileSystemId == nil || *getRequest.LustreFileSystemId != existingID {
							t.Fatalf("GetLustreFileSystemRequest.LustreFileSystemId = %v, want %s", getRequest.LustreFileSystemId, existingID)
						}
						return lustrefilestoragesdk.GetLustreFileSystemResponse{
							LustreFileSystem: observedLustreFileSystemFromSpec(existingID, resource.Spec, tc.lifecycleState, nil),
						}, nil
					},
					Fields: lustreFileSystemGetFields(),
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
			if got := resource.Status.OsokStatus.Reason; got != string(tc.wantCondition) {
				t.Fatalf("status.reason = %q, want %q", got, tc.wantCondition)
			}
		})
	}
}

func TestLustreFileSystemCreateOrUpdateDoesNotUpdateWhenOptionalMaintenanceWindowIsOmitted(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.lustrefilesystem.oc1..existing"

	resource := newExistingLustreFileSystemTestResource(existingID)
	updateCalls := 0

	manager := newLustreFileSystemRuntimeTestManager(generatedruntime.Config[*lustrefilestoragev1beta1.LustreFileSystem]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &lustrefilestoragesdk.GetLustreFileSystemRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getRequest := *request.(*lustrefilestoragesdk.GetLustreFileSystemRequest)
				if getRequest.LustreFileSystemId == nil || *getRequest.LustreFileSystemId != existingID {
					t.Fatalf("GetLustreFileSystemRequest.LustreFileSystemId = %v, want %s", getRequest.LustreFileSystemId, existingID)
				}
				return lustrefilestoragesdk.GetLustreFileSystemResponse{
					LustreFileSystem: observedLustreFileSystemFromSpec(existingID, resource.Spec, "ACTIVE", defaultSDKMaintenanceWindow()),
				}, nil
			},
			Fields: lustreFileSystemGetFields(),
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &lustrefilestoragesdk.UpdateLustreFileSystemRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				updateCalls++
				return lustrefilestoragesdk.UpdateLustreFileSystemResponse{}, nil
			},
			Fields: lustreFileSystemUpdateFields(),
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success when the live state already matches the spec")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = true, want false for a matching ACTIVE resource")
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateLustreFileSystem() called %d times, want 0 when optional maintenanceWindow was omitted from spec", updateCalls)
	}
}

func TestLustreFileSystemCreateOrUpdateReusesInactiveSeededListMatch(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.lustrefilesystem.oc1..seeded"

	resource := newLustreFileSystemTestResource()
	createCalled := false
	var listRequest lustrefilestoragesdk.ListLustreFileSystemsRequest

	manager := newLustreFileSystemRuntimeTestManager(generatedruntime.Config[*lustrefilestoragev1beta1.LustreFileSystem]{
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &lustrefilestoragesdk.CreateLustreFileSystemRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				createCalled = true
				return lustrefilestoragesdk.CreateLustreFileSystemResponse{}, nil
			},
			Fields: lustreFileSystemCreateFields(),
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &lustrefilestoragesdk.GetLustreFileSystemRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getRequest := *request.(*lustrefilestoragesdk.GetLustreFileSystemRequest)
				if getRequest.LustreFileSystemId == nil || *getRequest.LustreFileSystemId != existingID {
					t.Fatalf("GetLustreFileSystemRequest.LustreFileSystemId = %v, want %s", getRequest.LustreFileSystemId, existingID)
				}
				return lustrefilestoragesdk.GetLustreFileSystemResponse{
					LustreFileSystem: observedLustreFileSystemFromSpec(existingID, resource.Spec, "INACTIVE", nil),
				}, nil
			},
			Fields: lustreFileSystemGetFields(),
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &lustrefilestoragesdk.ListLustreFileSystemsRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				listRequest = *request.(*lustrefilestoragesdk.ListLustreFileSystemsRequest)
				return lustrefilestoragesdk.ListLustreFileSystemsResponse{
					LustreFileSystemCollection: lustrefilestoragesdk.LustreFileSystemCollection{
						Items: []lustrefilestoragesdk.LustreFileSystemSummary{
							observedLustreFileSystemSummaryFromSpec(existingID, resource.Spec, "INACTIVE"),
						},
					},
				}, nil
			},
			Fields: lustreFileSystemListFields(),
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success when the inactive seeded match is reusable")
	}
	if createCalled {
		t.Fatal("CreateLustreFileSystem() should not be called when list lookup found a reusable inactive seeded match")
	}
	if listRequest.CompartmentId == nil || *listRequest.CompartmentId != resource.Spec.CompartmentId {
		t.Fatalf("ListLustreFileSystemsRequest.CompartmentId = %v, want %s", listRequest.CompartmentId, resource.Spec.CompartmentId)
	}
	if listRequest.AvailabilityDomain == nil || *listRequest.AvailabilityDomain != resource.Spec.AvailabilityDomain {
		t.Fatalf("ListLustreFileSystemsRequest.AvailabilityDomain = %v, want %s", listRequest.AvailabilityDomain, resource.Spec.AvailabilityDomain)
	}
	if listRequest.DisplayName == nil || *listRequest.DisplayName != resource.Spec.DisplayName {
		t.Fatalf("ListLustreFileSystemsRequest.DisplayName = %v, want %s", listRequest.DisplayName, resource.Spec.DisplayName)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != existingID {
		t.Fatalf("status.ocid = %q, want %q", got, existingID)
	}
}

func TestLustreFileSystemDeleteTreatsNotFoundAsSuccess(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.lustrefilesystem.oc1..existing"

	resource := newExistingLustreFileSystemTestResource(existingID)
	manager := newLustreFileSystemRuntimeTestManager(generatedruntime.Config[*lustrefilestoragev1beta1.LustreFileSystem]{
		Delete: &generatedruntime.Operation{
			NewRequest: func() any { return &lustrefilestoragesdk.DeleteLustreFileSystemRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				deleteRequest := *request.(*lustrefilestoragesdk.DeleteLustreFileSystemRequest)
				if deleteRequest.LustreFileSystemId == nil || *deleteRequest.LustreFileSystemId != existingID {
					t.Fatalf("DeleteLustreFileSystemRequest.LustreFileSystemId = %v, want %s", deleteRequest.LustreFileSystemId, existingID)
				}
				return lustrefilestoragesdk.DeleteLustreFileSystemResponse{}, nil
			},
			Fields: lustreFileSystemDeleteFields(),
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &lustrefilestoragesdk.GetLustreFileSystemRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getRequest := *request.(*lustrefilestoragesdk.GetLustreFileSystemRequest)
				if getRequest.LustreFileSystemId == nil || *getRequest.LustreFileSystemId != existingID {
					t.Fatalf("GetLustreFileSystemRequest.LustreFileSystemId = %v, want %s", getRequest.LustreFileSystemId, existingID)
				}
				return lustrefilestoragesdk.GetLustreFileSystemResponse{}, fakeLustreFileSystemServiceError{
					statusCode: 404,
					code:       "NotAuthorizedOrNotFound",
					message:    "lustre file system not found",
				}
			},
			Fields: lustreFileSystemGetFields(),
		},
	})

	deleted, err := manager.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() = false, want true after GetLustreFileSystem reports NotFound")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want a deletion timestamp after successful delete confirmation")
	}
}

type fakeLustreFileSystemServiceError struct {
	statusCode int
	code       string
	message    string
}

func (f fakeLustreFileSystemServiceError) Error() string          { return f.message }
func (f fakeLustreFileSystemServiceError) GetHTTPStatusCode() int { return f.statusCode }
func (f fakeLustreFileSystemServiceError) GetMessage() string     { return f.message }
func (f fakeLustreFileSystemServiceError) GetCode() string        { return f.code }
func (f fakeLustreFileSystemServiceError) GetOpcRequestID() string {
	return ""
}

func newLustreFileSystemRuntimeTestManager(
	cfg generatedruntime.Config[*lustrefilestoragev1beta1.LustreFileSystem],
) *LustreFileSystemServiceManager {
	if cfg.Kind == "" {
		cfg.Kind = "LustreFileSystem"
	}
	if cfg.SDKName == "" {
		cfg.SDKName = "LustreFileSystem"
	}
	if cfg.Semantics == nil {
		cfg.Semantics = testLustreFileSystemRuntimeSemantics()
	}

	return &LustreFileSystemServiceManager{
		client: defaultLustreFileSystemServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*lustrefilestoragev1beta1.LustreFileSystem](cfg),
		},
	}
}

func testLustreFileSystemRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{"CREATING"},
			UpdatingStates:     []string{"UPDATING"},
			ActiveStates:       []string{"ACTIVE", "INACTIVE"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{"DELETING"},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"availabilityDomain", "compartmentId", "displayName", "id", "lifecycleState"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"capacityInGbs",
				"definedTags",
				"displayName",
				"fileSystemDescription",
				"freeformTags",
				"kmsKeyId",
				"maintenanceWindow",
				"nsgIds",
				"rootSquashConfiguration",
			},
			ForceNew: []string{
				"availabilityDomain",
				"clusterPlacementGroupId",
				"compartmentId",
				"fileSystemName",
				"performanceTier",
				"subnetId",
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

func newLustreFileSystemTestResource() *lustrefilestoragev1beta1.LustreFileSystem {
	return &lustrefilestoragev1beta1.LustreFileSystem{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "lustrefilesystem-sample",
			Namespace: "default",
		},
		Spec: lustrefilestoragev1beta1.LustreFileSystemSpec{
			CompartmentId:      "ocid1.compartment.oc1..example",
			AvailabilityDomain: "PHX-AD-1",
			FileSystemName:     "lustre01",
			CapacityInGBs:      5120,
			SubnetId:           "ocid1.subnet.oc1..example",
			PerformanceTier:    "MBPS_PER_TB_125",
			RootSquashConfiguration: lustrefilestoragev1beta1.LustreFileSystemRootSquashConfiguration{
				IdentitySquash:   "ROOT",
				SquashUid:        65534,
				SquashGid:        65534,
				ClientExceptions: []string{"10.0.0.10@tcp"},
			},
			DisplayName:           "lustre-sample",
			FileSystemDescription: "seeded lustre file system",
			FreeformTags: map[string]string{
				"env": "test",
			},
			NsgIds:   []string{"ocid1.nsg.oc1..example"},
			KmsKeyId: "ocid1.key.oc1..example",
		},
	}
}

func newExistingLustreFileSystemTestResource(existingID string) *lustrefilestoragev1beta1.LustreFileSystem {
	resource := newLustreFileSystemTestResource()
	resource.Status = lustrefilestoragev1beta1.LustreFileSystemStatus{
		OsokStatus: shared.OSOKStatus{
			Ocid: shared.OCID(existingID),
		},
		Id: existingID,
	}
	return resource
}

func observedLustreFileSystemFromSpec(
	id string,
	spec lustrefilestoragev1beta1.LustreFileSystemSpec,
	lifecycleState string,
	maintenanceWindow *lustrefilestoragesdk.MaintenanceWindow,
) lustrefilestoragesdk.LustreFileSystem {
	capacity := spec.CapacityInGBs
	squashUid := spec.RootSquashConfiguration.SquashUid
	squashGid := spec.RootSquashConfiguration.SquashGid

	return lustrefilestoragesdk.LustreFileSystem{
		Id:                 common.String(id),
		CompartmentId:      common.String(spec.CompartmentId),
		AvailabilityDomain: common.String(spec.AvailabilityDomain),
		DisplayName:        common.String(spec.DisplayName),
		FileSystemDescription: func() *string {
			if spec.FileSystemDescription == "" {
				return nil
			}
			return common.String(spec.FileSystemDescription)
		}(),
		TimeCreated:              &common.SDKTime{Time: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)},
		TimeUpdated:              &common.SDKTime{Time: time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC)},
		LifecycleState:           lustrefilestoragesdk.LustreFileSystemLifecycleStateEnum(lifecycleState),
		FreeformTags:             map[string]string{"env": "test"},
		DefinedTags:              map[string]map[string]interface{}{},
		SystemTags:               map[string]map[string]interface{}{},
		CapacityInGBs:            &capacity,
		SubnetId:                 common.String(spec.SubnetId),
		PerformanceTier:          lustrefilestoragesdk.LustreFileSystemPerformanceTierEnum(spec.PerformanceTier),
		ManagementServiceAddress: common.String("10.0.0.4"),
		FileSystemName:           common.String(spec.FileSystemName),
		Lnet:                     common.String("tcp"),
		MajorVersion:             common.String("2.15"),
		MaintenanceWindow:        maintenanceWindow,
		RootSquashConfiguration: &lustrefilestoragesdk.RootSquashConfiguration{
			IdentitySquash:   lustrefilestoragesdk.RootSquashConfigurationIdentitySquashEnum(spec.RootSquashConfiguration.IdentitySquash),
			SquashUid:        &squashUid,
			SquashGid:        &squashGid,
			ClientExceptions: append([]string(nil), spec.RootSquashConfiguration.ClientExceptions...),
		},
		LifecycleDetails: common.String("lustre file system " + lifecycleState),
		NsgIds:           append([]string(nil), spec.NsgIds...),
		KmsKeyId:         common.String(spec.KmsKeyId),
	}
}

func observedLustreFileSystemSummaryFromSpec(
	id string,
	spec lustrefilestoragev1beta1.LustreFileSystemSpec,
	lifecycleState string,
) lustrefilestoragesdk.LustreFileSystemSummary {
	capacity := spec.CapacityInGBs
	squashUid := spec.RootSquashConfiguration.SquashUid
	squashGid := spec.RootSquashConfiguration.SquashGid

	return lustrefilestoragesdk.LustreFileSystemSummary{
		Id:                       common.String(id),
		CompartmentId:            common.String(spec.CompartmentId),
		AvailabilityDomain:       common.String(spec.AvailabilityDomain),
		DisplayName:              common.String(spec.DisplayName),
		FileSystemDescription:    common.String(spec.FileSystemDescription),
		TimeCreated:              &common.SDKTime{Time: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)},
		TimeUpdated:              &common.SDKTime{Time: time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC)},
		LifecycleState:           lustrefilestoragesdk.LustreFileSystemLifecycleStateEnum(lifecycleState),
		FreeformTags:             map[string]string{"env": "test"},
		DefinedTags:              map[string]map[string]interface{}{},
		SystemTags:               map[string]map[string]interface{}{},
		FileSystemName:           common.String(spec.FileSystemName),
		CapacityInGBs:            &capacity,
		SubnetId:                 common.String(spec.SubnetId),
		PerformanceTier:          lustrefilestoragesdk.LustreFileSystemSummaryPerformanceTierEnum(spec.PerformanceTier),
		TimeBillingCycleEnd:      &common.SDKTime{Time: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)},
		ManagementServiceAddress: common.String("10.0.0.4"),
		Lnet:                     common.String("tcp"),
		MajorVersion:             common.String("2.15"),
		RootSquashConfiguration: &lustrefilestoragesdk.RootSquashConfiguration{
			IdentitySquash:   lustrefilestoragesdk.RootSquashConfigurationIdentitySquashEnum(spec.RootSquashConfiguration.IdentitySquash),
			SquashUid:        &squashUid,
			SquashGid:        &squashGid,
			ClientExceptions: append([]string(nil), spec.RootSquashConfiguration.ClientExceptions...),
		},
		LifecycleDetails: common.String("lustre file system " + lifecycleState),
		NsgIds:           append([]string(nil), spec.NsgIds...),
		KmsKeyId:         common.String(spec.KmsKeyId),
	}
}

func defaultSDKMaintenanceWindow() *lustrefilestoragesdk.MaintenanceWindow {
	timeStart := "22:00"
	return &lustrefilestoragesdk.MaintenanceWindow{
		DayOfWeek: lustrefilestoragesdk.MaintenanceWindowDayOfWeekMonday,
		TimeStart: &timeStart,
	}
}

func lustreFileSystemCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateLustreFileSystemDetails", RequestName: "CreateLustreFileSystemDetails", Contribution: "body"},
	}
}

func lustreFileSystemGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "LustreFileSystemId", RequestName: "lustreFileSystemId", Contribution: "path", PreferResourceID: true},
	}
}

func lustreFileSystemListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
		{FieldName: "AvailabilityDomain", RequestName: "availabilityDomain", Contribution: "query"},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query"},
		{FieldName: "Id", RequestName: "id", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func lustreFileSystemUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "LustreFileSystemId", RequestName: "lustreFileSystemId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateLustreFileSystemDetails", RequestName: "UpdateLustreFileSystemDetails", Contribution: "body"},
	}
}

func lustreFileSystemDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "LustreFileSystemId", RequestName: "lustreFileSystemId", Contribution: "path", PreferResourceID: true},
	}
}

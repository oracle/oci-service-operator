package script

import (
	"context"
	"maps"
	"strings"
	"testing"
	"time"

	apmsyntheticssdk "github.com/oracle/oci-go-sdk/v65/apmsynthetics"
	"github.com/oracle/oci-go-sdk/v65/common"
	apmsyntheticsv1beta1 "github.com/oracle/oci-service-operator/api/apmsynthetics/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type stubScriptOCIClient struct {
	create func(context.Context, apmsyntheticssdk.CreateScriptRequest) (apmsyntheticssdk.CreateScriptResponse, error)
	get    func(context.Context, apmsyntheticssdk.GetScriptRequest) (apmsyntheticssdk.GetScriptResponse, error)
	list   func(context.Context, apmsyntheticssdk.ListScriptsRequest) (apmsyntheticssdk.ListScriptsResponse, error)
	update func(context.Context, apmsyntheticssdk.UpdateScriptRequest) (apmsyntheticssdk.UpdateScriptResponse, error)
	delete func(context.Context, apmsyntheticssdk.DeleteScriptRequest) (apmsyntheticssdk.DeleteScriptResponse, error)
}

func (s stubScriptOCIClient) CreateScript(
	ctx context.Context,
	req apmsyntheticssdk.CreateScriptRequest,
) (apmsyntheticssdk.CreateScriptResponse, error) {
	if s.create == nil {
		return apmsyntheticssdk.CreateScriptResponse{}, nil
	}
	return s.create(ctx, req)
}

func (s stubScriptOCIClient) GetScript(
	ctx context.Context,
	req apmsyntheticssdk.GetScriptRequest,
) (apmsyntheticssdk.GetScriptResponse, error) {
	if s.get == nil {
		return apmsyntheticssdk.GetScriptResponse{}, nil
	}
	return s.get(ctx, req)
}

func (s stubScriptOCIClient) ListScripts(
	ctx context.Context,
	req apmsyntheticssdk.ListScriptsRequest,
) (apmsyntheticssdk.ListScriptsResponse, error) {
	if s.list == nil {
		return apmsyntheticssdk.ListScriptsResponse{}, nil
	}
	return s.list(ctx, req)
}

func (s stubScriptOCIClient) UpdateScript(
	ctx context.Context,
	req apmsyntheticssdk.UpdateScriptRequest,
) (apmsyntheticssdk.UpdateScriptResponse, error) {
	if s.update == nil {
		return apmsyntheticssdk.UpdateScriptResponse{}, nil
	}
	return s.update(ctx, req)
}

func (s stubScriptOCIClient) DeleteScript(
	ctx context.Context,
	req apmsyntheticssdk.DeleteScriptRequest,
) (apmsyntheticssdk.DeleteScriptResponse, error) {
	if s.delete == nil {
		return apmsyntheticssdk.DeleteScriptResponse{}, nil
	}
	return s.delete(ctx, req)
}

func TestApplyScriptRuntimeHooksUsesReviewedNoLifecycleContract(t *testing.T) {
	t.Parallel()

	hooks := newScriptDefaultRuntimeHooks(apmsyntheticssdk.ApmSyntheticClient{})
	applyScriptRuntimeHooks(&ScriptServiceManager{}, &hooks)

	if hooks.Semantics == nil || hooks.Semantics.List == nil {
		t.Fatal("hooks.Semantics.List = nil, want reviewed list semantics")
	}
	if got := hooks.Semantics.List.MatchFields; len(got) != 2 || got[0] != "displayName" || got[1] != "contentType" {
		t.Fatalf("hooks.Semantics.List.MatchFields = %#v, want [displayName contentType]", got)
	}
	if len(hooks.Semantics.Lifecycle.ProvisioningStates) != 0 {
		t.Fatalf("hooks.Semantics.Lifecycle.ProvisioningStates = %#v, want none", hooks.Semantics.Lifecycle.ProvisioningStates)
	}
	if hooks.Identity.GuardExistingBeforeCreate == nil {
		t.Fatal("hooks.Identity.GuardExistingBeforeCreate = nil, want reviewed guard")
	}
	if hooks.BuildUpdateBody == nil {
		t.Fatal("hooks.BuildUpdateBody = nil, want typed update builder")
	}
	if len(hooks.WrapGeneratedClient) != 2 {
		t.Fatalf("len(hooks.WrapGeneratedClient) = %d, want 2 wrappers", len(hooks.WrapGeneratedClient))
	}
	if got := hooks.List.Fields[1].LookupPaths; len(got) != 3 || got[0] != "status.displayName" || got[1] != "spec.displayName" || got[2] != "displayName" {
		t.Fatalf("hooks.List.Fields[1].LookupPaths = %#v, want status-first displayName lookup", got)
	}
	if got := hooks.List.Fields[2].LookupPaths; len(got) != 3 || got[0] != "status.contentType" || got[1] != "spec.contentType" || got[2] != "contentType" {
		t.Fatalf("hooks.List.Fields[2].LookupPaths = %#v, want status-first contentType lookup", got)
	}
}

func TestGuardScriptExistingBeforeCreate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		resource *apmsyntheticsv1beta1.Script
		want     generatedruntime.ExistingBeforeCreateDecision
		wantErr  string
	}{
		{
			name:     "nil resource fails",
			resource: nil,
			want:     generatedruntime.ExistingBeforeCreateDecisionFail,
			wantErr:  "resource is nil",
		},
		{
			name: "missing domain fails",
			resource: &apmsyntheticsv1beta1.Script{
				Spec: apmsyntheticsv1beta1.ScriptSpec{
					DisplayName: "script-a",
					ContentType: "JS",
				},
			},
			want:    generatedruntime.ExistingBeforeCreateDecisionFail,
			wantErr: "spec.apmDomainId is required",
		},
		{
			name: "missing display name skips",
			resource: &apmsyntheticsv1beta1.Script{
				Spec: apmsyntheticsv1beta1.ScriptSpec{
					ApmDomainId: "ocid1.apmdomain.oc1..example",
					ContentType: "JS",
				},
			},
			want: generatedruntime.ExistingBeforeCreateDecisionSkip,
		},
		{
			name: "missing content type skips",
			resource: &apmsyntheticsv1beta1.Script{
				Spec: apmsyntheticsv1beta1.ScriptSpec{
					ApmDomainId: "ocid1.apmdomain.oc1..example",
					DisplayName: "script-a",
				},
			},
			want: generatedruntime.ExistingBeforeCreateDecisionSkip,
		},
		{
			name:     "domain name and type allow lookup",
			resource: newScriptResource(),
			want:     generatedruntime.ExistingBeforeCreateDecisionAllow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := guardScriptExistingBeforeCreate(context.Background(), tt.resource)
			if got != tt.want {
				t.Fatalf("guardScriptExistingBeforeCreate() = %q, want %q", got, tt.want)
			}
			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatalf("guardScriptExistingBeforeCreate() error = %v, want nil", err)
			case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
				t.Fatalf("guardScriptExistingBeforeCreate() error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestNewScriptServiceClientWithOCIClientUsesScopedReuseAndMirrorsDomain(t *testing.T) {
	t.Parallel()

	resource := newScriptResource()
	existingID := "ocid1.script.oc1..existing"
	var listRequest apmsyntheticssdk.ListScriptsRequest
	var getRequest apmsyntheticssdk.GetScriptRequest
	createCalled := false

	client := newScriptServiceClientWithOCIClient(
		loggerutil.OSOKLogger{},
		stubScriptOCIClient{
			create: func(context.Context, apmsyntheticssdk.CreateScriptRequest) (apmsyntheticssdk.CreateScriptResponse, error) {
				createCalled = true
				return apmsyntheticssdk.CreateScriptResponse{}, nil
			},
			list: func(_ context.Context, req apmsyntheticssdk.ListScriptsRequest) (apmsyntheticssdk.ListScriptsResponse, error) {
				listRequest = req
				return apmsyntheticssdk.ListScriptsResponse{
					ScriptCollection: apmsyntheticssdk.ScriptCollection{
						Items: []apmsyntheticssdk.ScriptSummary{
							{
								Id:                    common.String(existingID),
								DisplayName:           common.String(resource.Spec.DisplayName),
								ContentType:           apmsyntheticssdk.ContentTypesEnum(resource.Spec.ContentType),
								MonitorStatusCountMap: &apmsyntheticssdk.MonitorStatusCountMap{},
							},
						},
					},
				}, nil
			},
			get: func(_ context.Context, req apmsyntheticssdk.GetScriptRequest) (apmsyntheticssdk.GetScriptResponse, error) {
				getRequest = req
				return apmsyntheticssdk.GetScriptResponse{
					Script: scriptSDK(existingID, resource),
				}, nil
			},
		},
	)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if createCalled {
		t.Fatal("CreateOrUpdate() invoked CreateScript, want existing-before-create reuse")
	}
	if listRequest.ApmDomainId == nil || *listRequest.ApmDomainId != resource.Spec.ApmDomainId {
		t.Fatalf("ListScriptsRequest.ApmDomainId = %#v, want %q", listRequest.ApmDomainId, resource.Spec.ApmDomainId)
	}
	if listRequest.DisplayName == nil || *listRequest.DisplayName != resource.Spec.DisplayName {
		t.Fatalf("ListScriptsRequest.DisplayName = %#v, want %q", listRequest.DisplayName, resource.Spec.DisplayName)
	}
	if listRequest.ContentType == nil || *listRequest.ContentType != resource.Spec.ContentType {
		t.Fatalf("ListScriptsRequest.ContentType = %#v, want %q", listRequest.ContentType, resource.Spec.ContentType)
	}
	if getRequest.ApmDomainId == nil || *getRequest.ApmDomainId != resource.Spec.ApmDomainId {
		t.Fatalf("GetScriptRequest.ApmDomainId = %#v, want %q", getRequest.ApmDomainId, resource.Spec.ApmDomainId)
	}
	if getRequest.ScriptId == nil || *getRequest.ScriptId != existingID {
		t.Fatalf("GetScriptRequest.ScriptId = %#v, want %q", getRequest.ScriptId, existingID)
	}
	if resource.Status.ApmDomainId != resource.Spec.ApmDomainId {
		t.Fatalf("Status.ApmDomainId = %q, want mirrored domain %q", resource.Status.ApmDomainId, resource.Spec.ApmDomainId)
	}
}

func TestNewScriptServiceClientWithOCIClientSettlesSynchronousCreateAsActive(t *testing.T) {
	t.Parallel()

	resource := newScriptResource()
	createdID := "ocid1.script.oc1..created"
	var createRequest apmsyntheticssdk.CreateScriptRequest
	var getRequest apmsyntheticssdk.GetScriptRequest

	client := newScriptServiceClientWithOCIClient(
		loggerutil.OSOKLogger{},
		stubScriptOCIClient{
			create: func(_ context.Context, req apmsyntheticssdk.CreateScriptRequest) (apmsyntheticssdk.CreateScriptResponse, error) {
				createRequest = req
				return apmsyntheticssdk.CreateScriptResponse{
					Script: scriptSDK(createdID, resource),
				}, nil
			},
			get: func(_ context.Context, req apmsyntheticssdk.GetScriptRequest) (apmsyntheticssdk.GetScriptResponse, error) {
				getRequest = req
				return apmsyntheticssdk.GetScriptResponse{
					Script: scriptSDK(createdID, resource),
				}, nil
			},
		},
	)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = true, want synchronous active completion")
	}
	if createRequest.ApmDomainId == nil || *createRequest.ApmDomainId != resource.Spec.ApmDomainId {
		t.Fatalf("CreateScriptRequest.ApmDomainId = %#v, want %q", createRequest.ApmDomainId, resource.Spec.ApmDomainId)
	}
	if getRequest.ApmDomainId == nil || *getRequest.ApmDomainId != resource.Spec.ApmDomainId {
		t.Fatalf("GetScriptRequest.ApmDomainId = %#v, want %q", getRequest.ApmDomainId, resource.Spec.ApmDomainId)
	}
	if getRequest.ScriptId == nil || *getRequest.ScriptId != createdID {
		t.Fatalf("GetScriptRequest.ScriptId = %#v, want %q", getRequest.ScriptId, createdID)
	}
	if resource.Status.OsokStatus.Reason != string(shared.Active) {
		t.Fatalf("status.reason = %q, want %q", resource.Status.OsokStatus.Reason, shared.Active)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want cleared tracker", resource.Status.OsokStatus.Async.Current)
	}
	if resource.Status.ApmDomainId != resource.Spec.ApmDomainId {
		t.Fatalf("Status.ApmDomainId = %q, want mirrored domain %q", resource.Status.ApmDomainId, resource.Spec.ApmDomainId)
	}
}

func TestNewScriptServiceClientWithOCIClientUsesTrackedDomainForForceNewValidation(t *testing.T) {
	t.Parallel()

	resource := newScriptResource()
	resource.Spec.ApmDomainId = "ocid1.apmdomain.oc1..new"
	resource.Status.ApmDomainId = "ocid1.apmdomain.oc1..tracked"
	resource.Status.OsokStatus.Ocid = "ocid1.script.oc1..tracked"

	var getRequest apmsyntheticssdk.GetScriptRequest
	client := newScriptServiceClientWithOCIClient(
		loggerutil.OSOKLogger{},
		stubScriptOCIClient{
			get: func(_ context.Context, req apmsyntheticssdk.GetScriptRequest) (apmsyntheticssdk.GetScriptResponse, error) {
				getRequest = req
				current := scriptSDK("ocid1.script.oc1..tracked", resource)
				return apmsyntheticssdk.GetScriptResponse{Script: current}, nil
			},
		},
	)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "require replacement when apmDomainId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want force-new apmDomainId error", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = true, want failed force-new validation")
	}
	if getRequest.ApmDomainId == nil || *getRequest.ApmDomainId != resource.Status.ApmDomainId {
		t.Fatalf("GetScriptRequest.ApmDomainId = %#v, want tracked domain %q", getRequest.ApmDomainId, resource.Status.ApmDomainId)
	}
}

func TestBuildScriptUpdateDetailsIgnoresEquivalentParameterInfoShape(t *testing.T) {
	t.Parallel()

	resource := newScriptResource()
	details, updateNeeded, err := buildScriptUpdateDetails(resource, scriptSDK("ocid1.script.oc1..same", resource))
	if err != nil {
		t.Fatalf("buildScriptUpdateDetails() error = %v", err)
	}
	if updateNeeded {
		t.Fatalf("buildScriptUpdateDetails() updateNeeded = true, want false with equivalent ScriptParameterInfo shape: %#v", details)
	}
}

func TestBuildScriptUpdateDetailsDetectsMutableDrift(t *testing.T) {
	t.Parallel()

	resource := newScriptResource()
	resource.Spec.ContentFileName = ""
	resource.Spec.Parameters = []apmsyntheticsv1beta1.ScriptParameter{}
	resource.Spec.FreeformTags = map[string]string{}
	resource.Spec.DefinedTags = map[string]shared.MapValue{}

	current := scriptSDK("ocid1.script.oc1..drift", newScriptResource())
	current.DisplayName = common.String("different-name")

	details, updateNeeded, err := buildScriptUpdateDetails(resource, current)
	if err != nil {
		t.Fatalf("buildScriptUpdateDetails() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildScriptUpdateDetails() updateNeeded = false, want true")
	}
	if details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName {
		t.Fatalf("UpdateScriptDetails.DisplayName = %#v, want %q", details.DisplayName, resource.Spec.DisplayName)
	}
	if details.ContentFileName == nil || *details.ContentFileName != "" {
		t.Fatalf("UpdateScriptDetails.ContentFileName = %#v, want explicit empty string clear", details.ContentFileName)
	}
	if details.Parameters == nil || len(details.Parameters) != 0 {
		t.Fatalf("UpdateScriptDetails.Parameters = %#v, want explicit empty slice clear", details.Parameters)
	}
	if details.FreeformTags == nil || len(details.FreeformTags) != 0 {
		t.Fatalf("UpdateScriptDetails.FreeformTags = %#v, want explicit empty map clear", details.FreeformTags)
	}
	if details.DefinedTags == nil || len(details.DefinedTags) != 0 {
		t.Fatalf("UpdateScriptDetails.DefinedTags = %#v, want explicit empty map clear", details.DefinedTags)
	}
}

func newScriptResource() *apmsyntheticsv1beta1.Script {
	return &apmsyntheticsv1beta1.Script{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "script-sample",
			Namespace: "default",
		},
		Spec: apmsyntheticsv1beta1.ScriptSpec{
			ApmDomainId:     "ocid1.apmdomain.oc1..example",
			DisplayName:     "script-sample",
			ContentType:     "JS",
			Content:         "console.log('sample');",
			ContentFileName: "script.js",
			Parameters: []apmsyntheticsv1beta1.ScriptParameter{
				{
					ParamName:  "username",
					ParamValue: "alice",
				},
			},
			FreeformTags: map[string]string{
				"managed-by": "osok",
			},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {
					"CostCenter": "42",
				},
			},
		},
	}
}

func scriptSDK(id string, resource *apmsyntheticsv1beta1.Script) apmsyntheticssdk.Script {
	now := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	parameters := make([]apmsyntheticssdk.ScriptParameterInfo, 0, len(resource.Spec.Parameters))
	for _, parameter := range resource.Spec.Parameters {
		parameters = append(parameters, apmsyntheticssdk.ScriptParameterInfo{
			ScriptParameter: &apmsyntheticssdk.ScriptParameter{
				ParamName:  common.String(parameter.ParamName),
				ParamValue: common.String(parameter.ParamValue),
				IsSecret:   common.Bool(parameter.IsSecret),
			},
			IsOverwritten: common.Bool(true),
		})
	}
	definedTags, err := scriptDefinedTagsToSDK(resource.Spec.DefinedTags)
	if err != nil {
		panic(err)
	}

	return apmsyntheticssdk.Script{
		Id:                    common.String(id),
		DisplayName:           common.String(resource.Spec.DisplayName),
		ContentType:           apmsyntheticssdk.ContentTypesEnum(resource.Spec.ContentType),
		MonitorStatusCountMap: &apmsyntheticssdk.MonitorStatusCountMap{},
		Content:               common.String(resource.Spec.Content),
		TimeUploaded:          &now,
		ContentSizeInBytes:    common.Int(len(resource.Spec.Content)),
		ContentFileName:       common.String(resource.Spec.ContentFileName),
		Parameters:            parameters,
		TimeCreated:           &now,
		TimeUpdated:           &now,
		FreeformTags:          maps.Clone(resource.Spec.FreeformTags),
		DefinedTags:           definedTags,
	}
}

func TestScriptDeleteUsesMirroredDomainDuringConfirmDelete(t *testing.T) {
	t.Parallel()

	resource := newScriptResource()
	resource.Spec.ApmDomainId = "ocid1.apmdomain.oc1..new"
	resource.Status.ApmDomainId = "ocid1.apmdomain.oc1..tracked"
	resource.Status.OsokStatus.Ocid = "ocid1.script.oc1..tracked"

	getCalls := 0
	var deleteRequest apmsyntheticssdk.DeleteScriptRequest

	client := newScriptServiceClientWithOCIClient(
		loggerutil.OSOKLogger{},
		stubScriptOCIClient{
			get: func(_ context.Context, req apmsyntheticssdk.GetScriptRequest) (apmsyntheticssdk.GetScriptResponse, error) {
				getCalls++
				if req.ApmDomainId == nil || *req.ApmDomainId != resource.Status.ApmDomainId {
					t.Fatalf("GetScriptRequest.ApmDomainId = %#v, want tracked domain %q", req.ApmDomainId, resource.Status.ApmDomainId)
				}
				if getCalls == 1 {
					return apmsyntheticssdk.GetScriptResponse{
						Script: scriptSDK("ocid1.script.oc1..tracked", newScriptResource()),
					}, nil
				}
				return apmsyntheticssdk.GetScriptResponse{}, errortest.NewServiceError(404, "NotFound", "missing")
			},
			delete: func(_ context.Context, req apmsyntheticssdk.DeleteScriptRequest) (apmsyntheticssdk.DeleteScriptResponse, error) {
				deleteRequest = req
				return apmsyntheticssdk.DeleteScriptResponse{}, nil
			},
		},
	)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() = false, want confirmed delete")
	}
	if deleteRequest.ApmDomainId == nil || *deleteRequest.ApmDomainId != resource.Status.ApmDomainId {
		t.Fatalf("DeleteScriptRequest.ApmDomainId = %#v, want tracked domain %q", deleteRequest.ApmDomainId, resource.Status.ApmDomainId)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete timestamp")
	}
}

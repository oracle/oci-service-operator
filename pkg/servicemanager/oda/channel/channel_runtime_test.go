/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package channel

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	odasdk "github.com/oracle/oci-go-sdk/v65/oda"
	odav1beta1 "github.com/oracle/oci-service-operator/api/oda/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testOdaInstanceID = "ocid1.odainstance.oc1..parent"
	testChannelID     = "ocid1.odachannel.oc1..channel"
)

func TestChannelCreateUsesParentIdentityAndConcreteBody(t *testing.T) {
	resource := newTestChannel()
	fake := &fakeChannelOperations{}
	fake.listFn = func(_ context.Context, request odasdk.ListChannelsRequest) (odasdk.ListChannelsResponse, error) {
		if got := stringPtrValue(request.OdaInstanceId); got != testOdaInstanceID {
			t.Fatalf("list OdaInstanceId = %q, want %q", got, testOdaInstanceID)
		}
		if got := stringPtrValue(request.Name); got != resource.Spec.Name {
			t.Fatalf("list Name = %q, want %q", got, resource.Spec.Name)
		}
		return odasdk.ListChannelsResponse{}, nil
	}
	fake.createFn = func(_ context.Context, request odasdk.CreateChannelRequest) (odasdk.CreateChannelResponse, error) {
		if got := stringPtrValue(request.OdaInstanceId); got != testOdaInstanceID {
			t.Fatalf("create OdaInstanceId = %q, want %q", got, testOdaInstanceID)
		}
		body, ok := request.CreateChannelDetails.(odasdk.CreateWebChannelDetails)
		if !ok {
			t.Fatalf("create body type = %T, want oda.CreateWebChannelDetails", request.CreateChannelDetails)
		}
		if got := stringPtrValue(body.Name); got != resource.Spec.Name {
			t.Fatalf("create body name = %q, want %q", got, resource.Spec.Name)
		}
		if body.IsClientAuthenticationEnabled == nil || *body.IsClientAuthenticationEnabled {
			t.Fatalf("create body should preserve explicit false isClientAuthenticationEnabled, got %#v", body.IsClientAuthenticationEnabled)
		}
		return odasdk.CreateChannelResponse{CreateChannelResult: newCreateWebChannelResult(testChannelID, resource.Spec.Name, odasdk.LifecycleStateCreating)}, nil
	}
	fake.getFn = func(_ context.Context, request odasdk.GetChannelRequest) (odasdk.GetChannelResponse, error) {
		if got := stringPtrValue(request.OdaInstanceId); got != testOdaInstanceID {
			t.Fatalf("get OdaInstanceId = %q, want %q", got, testOdaInstanceID)
		}
		if got := stringPtrValue(request.ChannelId); got != testChannelID {
			t.Fatalf("get ChannelId = %q, want %q", got, testChannelID)
		}
		return odasdk.GetChannelResponse{Channel: newWebChannel(testChannelID, resource.Spec.Name, odasdk.LifecycleStateActive, "")}, nil
	}

	response, err := newTestChannelClient(t, fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: resource.Name}})
	if err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("response = %#v, want successful without requeue", response)
	}
	if fake.createCalls != 1 {
		t.Fatalf("create calls = %d, want 1", fake.createCalls)
	}
	if resource.Status.Id != testChannelID {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, testChannelID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testChannelID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testChannelID)
	}
	if resource.Status.LifecycleState != string(odasdk.LifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", resource.Status.LifecycleState)
	}
}

func TestChannelBindExistingFromList(t *testing.T) {
	resource := newTestChannel()
	fake := &fakeChannelOperations{}
	fake.listFn = func(_ context.Context, request odasdk.ListChannelsRequest) (odasdk.ListChannelsResponse, error) {
		return odasdk.ListChannelsResponse{ChannelCollection: odasdk.ChannelCollection{Items: []odasdk.ChannelSummary{
			newChannelSummary(testChannelID, resource.Spec.Name, odasdk.LifecycleStateActive),
		}}}, nil
	}
	fake.getFn = func(_ context.Context, request odasdk.GetChannelRequest) (odasdk.GetChannelResponse, error) {
		return odasdk.GetChannelResponse{Channel: newWebChannel(stringPtrValue(request.ChannelId), resource.Spec.Name, odasdk.LifecycleStateActive, resource.Spec.Description)}, nil
	}

	response, err := newTestChannelClient(t, fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("response = %#v, want successful without requeue", response)
	}
	if fake.createCalls != 0 {
		t.Fatalf("create calls = %d, want 0", fake.createCalls)
	}
	if resource.Status.Id != testChannelID {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, testChannelID)
	}
}

func TestChannelUpdateMutableField(t *testing.T) {
	resource := newTestChannel()
	resource.Status.Id = testChannelID
	resource.Status.OsokStatus.Ocid = shared.OCID(testChannelID)
	resource.Spec.Description = "new description"
	fake := &fakeChannelOperations{}
	fake.getFn = func(_ context.Context, request odasdk.GetChannelRequest) (odasdk.GetChannelResponse, error) {
		if fake.getCalls == 1 {
			return odasdk.GetChannelResponse{Channel: newWebChannel(testChannelID, resource.Spec.Name, odasdk.LifecycleStateActive, "old description")}, nil
		}
		return odasdk.GetChannelResponse{Channel: newWebChannel(testChannelID, resource.Spec.Name, odasdk.LifecycleStateUpdating, resource.Spec.Description)}, nil
	}
	fake.updateFn = func(_ context.Context, request odasdk.UpdateChannelRequest) (odasdk.UpdateChannelResponse, error) {
		if got := stringPtrValue(request.OdaInstanceId); got != testOdaInstanceID {
			t.Fatalf("update OdaInstanceId = %q, want %q", got, testOdaInstanceID)
		}
		if got := stringPtrValue(request.ChannelId); got != testChannelID {
			t.Fatalf("update ChannelId = %q, want %q", got, testChannelID)
		}
		body, ok := request.UpdateChannelDetails.(odasdk.UpdateWebChannelDetails)
		if !ok {
			t.Fatalf("update body type = %T, want oda.UpdateWebChannelDetails", request.UpdateChannelDetails)
		}
		if got := stringPtrValue(body.Description); got != resource.Spec.Description {
			t.Fatalf("update description = %q, want %q", got, resource.Spec.Description)
		}
		return odasdk.UpdateChannelResponse{Channel: newWebChannel(testChannelID, resource.Spec.Name, odasdk.LifecycleStateUpdating, resource.Spec.Description)}, nil
	}

	response, err := newTestChannelClient(t, fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("response = %#v, want successful requeue while updating", response)
	}
	if fake.updateCalls != 1 {
		t.Fatalf("update calls = %d, want 1", fake.updateCalls)
	}
	if resource.Status.OsokStatus.Async.Current == nil || resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseUpdate {
		t.Fatalf("status async current = %#v, want update lifecycle tracker", resource.Status.OsokStatus.Async.Current)
	}
}

func TestChannelRejectsCreateOnlyTypeDrift(t *testing.T) {
	resource := newTestChannel()
	resource.Status.Id = testChannelID
	resource.Status.OsokStatus.Ocid = shared.OCID(testChannelID)
	resource.Spec.Type = string(odasdk.ChannelTypeFacebook)
	fake := &fakeChannelOperations{}
	fake.getFn = func(context.Context, odasdk.GetChannelRequest) (odasdk.GetChannelResponse, error) {
		return odasdk.GetChannelResponse{Channel: newWebChannel(testChannelID, resource.Spec.Name, odasdk.LifecycleStateActive, "")}, nil
	}

	response, err := newTestChannelClient(t, fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "type is create-only") {
		t.Fatalf("CreateOrUpdate error = %v, want create-only type drift error", err)
	}
	if response.IsSuccessful {
		t.Fatalf("response = %#v, want unsuccessful", response)
	}
	if fake.updateCalls != 0 {
		t.Fatalf("update calls = %d, want 0", fake.updateCalls)
	}
}

func TestChannelRejectsUnsupportedTypeSpecificDriftBeforeUpdate(t *testing.T) {
	resource := newTestChannel()
	resource.Status.Id = testChannelID
	resource.Status.OsokStatus.Ocid = shared.OCID(testChannelID)
	resource.Spec.AccountSID = "twilio-account"
	fake := &fakeChannelOperations{}
	fake.getFn = func(context.Context, odasdk.GetChannelRequest) (odasdk.GetChannelResponse, error) {
		return odasdk.GetChannelResponse{Channel: newWebChannel(testChannelID, resource.Spec.Name, odasdk.LifecycleStateActive, "")}, nil
	}
	fake.updateFn = func(context.Context, odasdk.UpdateChannelRequest) (odasdk.UpdateChannelResponse, error) {
		t.Fatal("update should not be called when unsupported drift is present")
		return odasdk.UpdateChannelResponse{}, nil
	}

	response, err := newTestChannelClient(t, fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), `does not support spec field(s): accountSID`) {
		t.Fatalf("CreateOrUpdate error = %v, want unsupported field drift error", err)
	}
	if response.IsSuccessful {
		t.Fatalf("response = %#v, want unsuccessful", response)
	}
	if fake.updateCalls != 0 {
		t.Fatalf("update calls = %d, want 0", fake.updateCalls)
	}
}

func TestChannelLifecycleFailedProjectsFailure(t *testing.T) {
	resource := newTestChannel()
	resource.Status.Id = testChannelID
	resource.Status.OsokStatus.Ocid = shared.OCID(testChannelID)
	fake := &fakeChannelOperations{}
	fake.getFn = func(context.Context, odasdk.GetChannelRequest) (odasdk.GetChannelResponse, error) {
		return odasdk.GetChannelResponse{Channel: newWebChannel(testChannelID, resource.Spec.Name, odasdk.LifecycleStateFailed, "")}, nil
	}

	response, err := newTestChannelClient(t, fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}
	if response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("response = %#v, want unsuccessful without requeue", response)
	}
	if got := latestCondition(resource.Status.OsokStatus, shared.Failed); got == nil || got.Status != v1.ConditionFalse {
		t.Fatalf("failed condition = %#v, want false condition", got)
	}
}

func TestChannelRejectsTrackedGetMissBeforeReplacement(t *testing.T) {
	resource := newTestChannel()
	resource.Status.Id = testChannelID
	resource.Status.OsokStatus.Ocid = shared.OCID(testChannelID)
	fake := &fakeChannelOperations{}
	fake.getFn = func(context.Context, odasdk.GetChannelRequest) (odasdk.GetChannelResponse, error) {
		return odasdk.GetChannelResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "missing")
	}

	response, err := newTestChannelClient(t, fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "refusing to create a replacement") {
		t.Fatalf("CreateOrUpdate error = %v, want replacement guard error", err)
	}
	if response.IsSuccessful {
		t.Fatalf("response = %#v, want unsuccessful", response)
	}
	if fake.listCalls != 0 || fake.createCalls != 0 {
		t.Fatalf("list/create calls = %d/%d, want no fallback replacement", fake.listCalls, fake.createCalls)
	}
}

func TestChannelDeleteKeepsFinalizerUntilConfirmed(t *testing.T) {
	resource := newTestChannel()
	resource.Status.Id = testChannelID
	resource.Status.OsokStatus.Ocid = shared.OCID(testChannelID)
	fake := &fakeChannelOperations{}
	fake.getFn = func(context.Context, odasdk.GetChannelRequest) (odasdk.GetChannelResponse, error) {
		if fake.getCalls == 1 {
			return odasdk.GetChannelResponse{Channel: newWebChannel(testChannelID, resource.Spec.Name, odasdk.LifecycleStateActive, "")}, nil
		}
		return odasdk.GetChannelResponse{Channel: newWebChannel(testChannelID, resource.Spec.Name, odasdk.LifecycleStateDeleting, "")}, nil
	}

	deleted, err := newTestChannelClient(t, fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if deleted {
		t.Fatalf("deleted = true, want false while delete is pending")
	}
	if fake.deleteCalls != 1 {
		t.Fatalf("delete calls = %d, want 1", fake.deleteCalls)
	}
	if resource.Status.OsokStatus.Async.Current == nil || resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status async current = %#v, want delete lifecycle tracker", resource.Status.OsokStatus.Async.Current)
	}
}

func TestChannelDeleteReleasesFinalizerOnConfirmedNotFound(t *testing.T) {
	resource := newTestChannel()
	resource.Status.Id = testChannelID
	resource.Status.OsokStatus.Ocid = shared.OCID(testChannelID)
	fake := &fakeChannelOperations{}
	fake.getFn = func(context.Context, odasdk.GetChannelRequest) (odasdk.GetChannelResponse, error) {
		if fake.getCalls == 1 {
			return odasdk.GetChannelResponse{Channel: newWebChannel(testChannelID, resource.Spec.Name, odasdk.LifecycleStateActive, "")}, nil
		}
		return odasdk.GetChannelResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "missing")
	}

	deleted, err := newTestChannelClient(t, fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if !deleted {
		t.Fatalf("deleted = false, want true after confirm read not found")
	}
	if fake.deleteCalls != 1 {
		t.Fatalf("delete calls = %d, want 1", fake.deleteCalls)
	}
}

func newTestChannelClient(t *testing.T, fake *fakeChannelOperations) ChannelServiceClient {
	t.Helper()
	manager := &ChannelServiceManager{Log: loggerutil.OSOKLogger{}}
	hooks := ChannelRuntimeHooks{
		Create: runtimeOperationHooks[odasdk.CreateChannelRequest, odasdk.CreateChannelResponse]{Call: fake.create},
		Get:    runtimeOperationHooks[odasdk.GetChannelRequest, odasdk.GetChannelResponse]{Call: fake.get},
		List:   runtimeOperationHooks[odasdk.ListChannelsRequest, odasdk.ListChannelsResponse]{Call: fake.list},
		Update: runtimeOperationHooks[odasdk.UpdateChannelRequest, odasdk.UpdateChannelResponse]{Call: fake.update},
		Delete: runtimeOperationHooks[odasdk.DeleteChannelRequest, odasdk.DeleteChannelResponse]{Call: fake.delete},
	}
	applyChannelRuntimeHooks(manager, &hooks)
	delegate := defaultChannelServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*odav1beta1.Channel](buildChannelGeneratedRuntimeConfig(manager, hooks)),
	}
	return wrapChannelGeneratedClient(hooks, delegate)
}

func newTestChannel() *odav1beta1.Channel {
	return &odav1beta1.Channel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "channel",
			Namespace: "default",
			UID:       types.UID("channel-uid"),
		},
		Spec: odav1beta1.ChannelSpec{
			OdaInstanceId:                 testOdaInstanceID,
			Name:                          "web_channel",
			Type:                          string(odasdk.ChannelTypeWeb),
			Description:                   "desired description",
			IsClientAuthenticationEnabled: false,
			AllowedDomains:                "*",
			BotId:                         "bot",
		},
	}
}

type fakeChannelOperations struct {
	createFn func(context.Context, odasdk.CreateChannelRequest) (odasdk.CreateChannelResponse, error)
	getFn    func(context.Context, odasdk.GetChannelRequest) (odasdk.GetChannelResponse, error)
	listFn   func(context.Context, odasdk.ListChannelsRequest) (odasdk.ListChannelsResponse, error)
	updateFn func(context.Context, odasdk.UpdateChannelRequest) (odasdk.UpdateChannelResponse, error)
	deleteFn func(context.Context, odasdk.DeleteChannelRequest) (odasdk.DeleteChannelResponse, error)

	createCalls int
	getCalls    int
	listCalls   int
	updateCalls int
	deleteCalls int
}

func (f *fakeChannelOperations) create(ctx context.Context, request odasdk.CreateChannelRequest) (odasdk.CreateChannelResponse, error) {
	f.createCalls++
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return odasdk.CreateChannelResponse{CreateChannelResult: newCreateWebChannelResult(testChannelID, "web_channel", odasdk.LifecycleStateActive)}, nil
}

func (f *fakeChannelOperations) get(ctx context.Context, request odasdk.GetChannelRequest) (odasdk.GetChannelResponse, error) {
	f.getCalls++
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return odasdk.GetChannelResponse{Channel: newWebChannel(stringPtrValue(request.ChannelId), "web_channel", odasdk.LifecycleStateActive, "")}, nil
}

func (f *fakeChannelOperations) list(ctx context.Context, request odasdk.ListChannelsRequest) (odasdk.ListChannelsResponse, error) {
	f.listCalls++
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return odasdk.ListChannelsResponse{}, nil
}

func (f *fakeChannelOperations) update(ctx context.Context, request odasdk.UpdateChannelRequest) (odasdk.UpdateChannelResponse, error) {
	f.updateCalls++
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return odasdk.UpdateChannelResponse{Channel: newWebChannel(stringPtrValue(request.ChannelId), "web_channel", odasdk.LifecycleStateActive, "")}, nil
}

func (f *fakeChannelOperations) delete(ctx context.Context, request odasdk.DeleteChannelRequest) (odasdk.DeleteChannelResponse, error) {
	f.deleteCalls++
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return odasdk.DeleteChannelResponse{}, nil
}

func newCreateWebChannelResult(id string, name string, state odasdk.LifecycleStateEnum) odasdk.CreateWebChannelResult {
	return odasdk.CreateWebChannelResult{
		Id:                            common.String(id),
		Name:                          common.String(name),
		Category:                      odasdk.ChannelCategoryBot,
		LifecycleState:                state,
		IsClientAuthenticationEnabled: common.Bool(false),
	}
}

func newWebChannel(id string, name string, state odasdk.LifecycleStateEnum, description string) odasdk.WebChannel {
	return odasdk.WebChannel{
		Id:                            common.String(id),
		Name:                          common.String(name),
		Category:                      odasdk.ChannelCategoryBot,
		LifecycleState:                state,
		Description:                   common.String(description),
		IsClientAuthenticationEnabled: common.Bool(false),
		AllowedDomains:                common.String("*"),
		BotId:                         common.String("bot"),
	}
}

func newChannelSummary(id string, name string, state odasdk.LifecycleStateEnum) odasdk.ChannelSummary {
	return odasdk.ChannelSummary{
		Id:             common.String(id),
		Name:           common.String(name),
		Category:       odasdk.ChannelCategoryBot,
		Type:           odasdk.ChannelTypeWeb,
		LifecycleState: state,
	}
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func latestCondition(status shared.OSOKStatus, conditionType shared.OSOKConditionType) *shared.OSOKCondition {
	for i := len(status.Conditions) - 1; i >= 0; i-- {
		if status.Conditions[i].Type == conditionType {
			return &status.Conditions[i]
		}
	}
	return nil
}

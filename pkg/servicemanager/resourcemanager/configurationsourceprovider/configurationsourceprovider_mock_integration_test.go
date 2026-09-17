/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package configurationsourceprovider

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	sdksvc "github.com/oracle/oci-go-sdk/v65/resourcemanager"
	apiv1beta1 "github.com/oracle/oci-service-operator/api/resourcemanager/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, generated production service manager,
// reviewed formal metadata, and package-owned typed OCI fixtures.
func TestMockIntegrationConfigurationSourceProviderCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.ConfigurationSourceProvider](t, `
{
  "metadata": {"name": "mock-configurationsourceprovider", "namespace": "default"},
  "spec": {
  "accessToken": "mock-accesstoken",
  "apiEndpoint": "mock-apiendpoint",
  "compartmentId": "<ocid:9>",
  "configSourceProviderType": "GITLAB_ACCESS_TOKEN",
  "displayName": "mock-displayname-initial"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-configurationsourceprovider")
	resource.Status = apiv1beta1.ConfigurationSourceProviderStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateGitlabAccessTokenConfigurationSourceProviderDetails](t, `{
  "accessToken": "mock-accesstoken",
  "apiEndpoint": "mock-apiendpoint",
  "compartmentId": "<ocid:9>",
  "displayName": "mock-displayname-initial"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateGitlabAccessTokenConfigurationSourceProviderDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.GitlabAccessTokenConfigurationSourceProvider](t, `{
  "accessToken": "mock-accesstoken",
  "apiEndpoint": "mock-apiendpoint",
  "compartmentId": "<ocid:9>",
  "displayName": "mock-displayname-initial",
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "resourceId": "<ocid:1>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.GitlabAccessTokenConfigurationSourceProvider](t, `{
  "accessToken": "mock-accesstoken",
  "apiEndpoint": "mock-apiendpoint",
  "compartmentId": "<ocid:9>",
  "displayName": "mock-displayname-updated",
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "resourceId": "<ocid:1>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)

	creatingState := createdState
	creatingState.LifecycleState = "CREATING"
	updatingState := updatedState
	updatingState.LifecycleState = "UPDATING"
	deletingState := updatedState
	deletingState.LifecycleState = "DELETING"

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.GitlabAccessTokenConfigurationSourceProvider, sdksvc.CreateGitlabAccessTokenConfigurationSourceProviderDetails, sdksvc.UpdateGitlabAccessTokenConfigurationSourceProviderDetails]{
		CollectionPath: "/20180917/configurationSourceProviders", ItemPath: "/20180917/configurationSourceProviders/<ocid:1>",
		CreatePath: "/20180917/configurationSourceProviders", CreateMethod: http.MethodPost,
		UpdatePath: "/20180917/configurationSourceProviders/<ocid:1>", UpdateMethod: http.MethodPut,
		DeletePath: "/20180917/configurationSourceProviders/<ocid:1>", DeleteMethod: http.MethodDelete,
		Operations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},

		CreatedState:      &creatingState,
		CreatedReadStates: ocimock.StateSequence(creatingState, createdState),

		UpdatedState:      &updatingState,
		UpdatedReadStates: ocimock.StateSequence(updatingState, updatedState),
		DeletedState:      &deletingState,
		DeletedReadStates: ocimock.StateSequence(deletingState),
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotAuthorizedOrNotFound",

		ValidateCreateRaw: func(request ocimock.Request) error {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				return err
			}
			return ocimock.ValidateDiscriminatedJSONRequestSubset(request, "configSourceProviderType", "GITLAB_ACCESS_TOKEN", createRequest)
		},
		ValidateUpdateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateDiscriminatedJSONRequestSubset(request, "configSourceProviderType", "GITLAB_ACCESS_TOKEN", updateRequest)
		},
		AdditionalRoutes: []ocimock.Route{},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://resourcemanager.us-ashburn-1.oci.oraclecloud.com", BasePath: "20180917", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := sdksvc.ResourceManagerClient{BaseClient: session.BaseClient()}
	manager := &ConfigurationSourceProviderServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newConfigurationSourceProviderRuntimeHooks(manager, sdkClient)
	client := wrapConfigurationSourceProviderGeneratedClient(hooks, defaultConfigurationSourceProviderServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.ConfigurationSourceProvider](buildConfigurationSourceProviderGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.ConfigurationSourceProvider]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.ConfigurationSourceProvider) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created ConfigurationSourceProvider status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.ConfigurationSourceProvider) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.ConfigurationSourceProvider) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated ConfigurationSourceProvider status = %+v", current.Status)
			}
			return nil
		},
		ValidateStable: func(*apiv1beta1.ConfigurationSourceProvider) error {
			if got := responder.OperationCounts()[ocimock.OperationUpdate]; got != 1 {
				return fmt.Errorf("stable ConfigurationSourceProvider update calls = %d, want 1", got)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
}

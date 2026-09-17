/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package termversion

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	marketplacepublishersdk "github.com/oracle/oci-go-sdk/v65/marketplacepublisher"
	marketplacepublisherv1beta1 "github.com/oracle/oci-service-operator/api/marketplacepublisher/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

const mockTermVersionID = "ocid1.termversion.oc1..mock"

func TestMockIntegrationTermVersionLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := newTermVersionResource()
	ocimock.InitializeResource(resource, "mock-termversion")
	updatedSpec := resource.Spec
	updatedSpec.DisplayName = "publisher-term-v1-updated"
	createdState := marketplacepublishersdk.TermVersion{
		Id:             common.String(mockTermVersionID),
		DisplayName:    common.String("publisher-term-v1"),
		LifecycleState: marketplacepublishersdk.TermVersionLifecycleStateActive,
	}
	updatedState := createdState
	updatedState.DisplayName = common.String("publisher-term-v1-updated")
	updateRequest := marketplacepublishersdk.UpdateTermVersionDetails{DisplayName: updatedState.DisplayName}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		marketplacepublishersdk.TermVersion,
		struct{},
		marketplacepublishersdk.UpdateTermVersionDetails,
	]{
		CollectionPath:    "/20241201/termVersions",
		ItemPath:          "/20241201/termVersions/" + mockTermVersionID,
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreatedState:      &createdState,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		ListShape:         ocimock.ListShapeItems,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      http.StatusCreated,
		UpdateStatus:      http.StatusOK,
		DeleteStatus:      http.StatusNoContent,
		NotFoundCode:      "NotFound",
		ValidateCreateRaw: func(request ocimock.Request) error {
			if string(request.Body) != testTermContent {
				return fmt.Errorf("create TermVersion content = %q", request.Body)
			}
			if request.Header.Get("content-type") != "application/octet-stream" ||
				request.Header.Get("display-name") != resource.Spec.DisplayName ||
				request.Header.Get("term-id") != testTermID ||
				request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create TermVersion headers = %v", request.Header)
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ marketplacepublishersdk.TermVersion) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete TermVersion body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://marketplacepublisher.mock.invalid", BasePath: "20241201", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close TermVersion OCI mock: %v", err)
		}
	})

	sdkClient := marketplacepublishersdk.MarketplacePublisherClient{BaseClient: session.BaseClient()}
	credentials := fakeTermVersionCredentialClient{secrets: map[string]map[string][]byte{"default/term-content": {termVersionDefaultContentSecretKey: []byte(testTermContent)}}}
	client := newTestTermVersionClient(sdkClient, credentials)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*marketplacepublisherv1beta1.TermVersion]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *marketplacepublisherv1beta1.TermVersion) error {
			if current.Status.Id != mockTermVersionID ||
				current.Status.DisplayName != "publisher-term-v1" ||
				current.Status.LifecycleState != string(marketplacepublishersdk.TermVersionLifecycleStateActive) {
				return fmt.Errorf("created TermVersion status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *marketplacepublisherv1beta1.TermVersion) {
			current.Spec = updatedSpec
		},
		ValidateUpdated: func(current *marketplacepublisherv1beta1.TermVersion) error {
			if current.Status.Id != mockTermVersionID ||
				current.Status.DisplayName != current.Spec.DisplayName ||
				current.Status.LifecycleState != string(marketplacepublishersdk.TermVersionLifecycleStateActive) {
				return fmt.Errorf("updated TermVersion status = %+v", current.Status)
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

/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package channel

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	odasdk "github.com/oracle/oci-go-sdk/v65/oda"
	odav1beta1 "github.com/oracle/oci-service-operator/api/oda/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestMockIntegrationChannelCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := newTestChannel()
	resource.Spec.OdaInstanceId = "<ocid:1>"
	updatedSpec := resource.Spec
	updatedSpec.Description = "updated description"
	createDetails := odasdk.CreateWebChannelDetails{Name: common.String(resource.Spec.Name), IsClientAuthenticationEnabled: common.Bool(false), Description: common.String(resource.Spec.Description), AllowedDomains: common.String(resource.Spec.AllowedDomains), BotId: common.String(resource.Spec.BotId)}
	updateDetails := odasdk.UpdateWebChannelDetails{Name: common.String(updatedSpec.Name), Description: common.String(updatedSpec.Description), IsClientAuthenticationEnabled: common.Bool(false), AllowedDomains: common.String(updatedSpec.AllowedDomains), BotId: common.String(updatedSpec.BotId)}
	created := newWebChannel("<ocid:2>", resource.Spec.Name, odasdk.LifecycleStateActive, resource.Spec.Description)
	updated := newWebChannel("<ocid:2>", updatedSpec.Name, odasdk.LifecycleStateActive, updatedSpec.Description)

	responder, err := ocimock.NewCRUDResponder(ocimock.CRUDOptions[odasdk.WebChannel]{
		CollectionPath: "/20190506/odaInstances/<ocid:1>/channels", ItemPath: "/20190506/odaInstances/<ocid:1>/channels/<ocid:2>",
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete}, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true,
		List: func(_ ocimock.Request, present bool, state odasdk.WebChannel) (ocimock.Response, error) {
			items := []odasdk.WebChannel{}
			if present {
				items = append(items, state)
			}
			return ocimock.JSONResponse(http.StatusOK, map[string]any{"items": items})
		},
		Create: func(request ocimock.Request) (odasdk.WebChannel, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero odasdk.WebChannel
				return zero, ocimock.Response{}, err
			}
			if err := ocimock.ValidateDiscriminatedJSONRequest(request, "type", "WEB", createDetails); err != nil {
				return odasdk.WebChannel{}, ocimock.Response{}, err
			}
			response, err := ocimock.JSONResponse(http.StatusCreated, created)
			return created, response, err
		},
		Read: func(_ ocimock.Request, state odasdk.WebChannel) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		Update: func(request ocimock.Request, _ odasdk.WebChannel) (odasdk.WebChannel, ocimock.Response, error) {
			if err := ocimock.ValidateDiscriminatedJSONRequest(request, "type", "WEB", updateDetails); err != nil {
				return odasdk.WebChannel{}, ocimock.Response{}, err
			}
			response, err := ocimock.JSONResponse(http.StatusOK, updated)
			return updated, response, err
		},
		Delete: func(_ ocimock.Request, _ odasdk.WebChannel) (ocimock.Response, error) {
			return ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oda.mock.invalid", BasePath: "20190506", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := odasdk.ManagementClient{BaseClient: session.BaseClient()}
	manager := &ChannelServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newChannelDefaultRuntimeHooks(sdkClient)
	applyChannelRuntimeHooks(manager, &hooks)
	client := wrapChannelGeneratedClient(hooks, defaultChannelServiceClient{ServiceClient: generatedruntime.NewServiceClient[*odav1beta1.Channel](buildChannelGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*odav1beta1.Channel]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *odav1beta1.Channel) error {
			if current.Status.Id != "<ocid:2>" || current.Status.Name != current.Spec.Name || current.Status.Type != current.Spec.Type || current.Status.LifecycleState != "ACTIVE" {
				return fmt.Errorf("created Channel status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *odav1beta1.Channel) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *odav1beta1.Channel) error {
			if current.Status.Description != current.Spec.Description || current.Status.Name != current.Spec.Name || current.Status.LifecycleState != "ACTIVE" {
				return fmt.Errorf("updated Channel status = %+v", current.Status)
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

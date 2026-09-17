/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package compartment

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	identitysdk "github.com/oracle/oci-go-sdk/v65/identity"
	identityv1beta1 "github.com/oracle/oci-service-operator/api/identity/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

func newMockCompartmentClient(sdkClient identitysdk.IdentityClient) CompartmentServiceClient {
	manager := &CompartmentServiceManager{Log: loggerutil.OSOKLogger{Logger: logr.Discard()}}
	hooks := newCompartmentRuntimeHooks(manager, sdkClient)
	delegate := wrapCompartmentGeneratedClient(hooks, defaultCompartmentServiceClient{ServiceClient: generatedruntime.NewServiceClient[*identityv1beta1.Compartment](buildCompartmentGeneratedRuntimeConfig(manager, hooks))})
	return compartmentOrphanDeleteClient{
		delegate: delegate,
		deleteCompartment: func(ctx context.Context, compartmentID shared.OCID) error {
			_, err := sdkClient.DeleteCompartment(ctx, identitysdk.DeleteCompartmentRequest{CompartmentId: common.String(string(compartmentID))})
			return err
		},
		loadCompartment: func(ctx context.Context, compartmentID shared.OCID) (*identitysdk.Compartment, error) {
			response, err := sdkClient.GetCompartment(ctx, identitysdk.GetCompartmentRequest{CompartmentId: common.String(string(compartmentID))})
			if err != nil {
				return nil, err
			}
			return &response.Compartment, nil
		},
		listCompartments: func(ctx context.Context, parentID shared.OCID, name string) ([]identitysdk.Compartment, error) {
			response, err := sdkClient.ListCompartments(ctx, identitysdk.ListCompartmentsRequest{CompartmentId: common.String(string(parentID)), Name: common.String(name)})
			if err != nil {
				return nil, err
			}
			return response.Items, nil
		},
	}
}

/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package compartment

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	identitysdk "github.com/oracle/oci-go-sdk/v65/identity"
	identityv1beta1 "github.com/oracle/oci-service-operator/api/identity/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

func init() {
	generatedFactory := newCompartmentServiceClient
	newCompartmentServiceClient = func(manager *CompartmentServiceManager) CompartmentServiceClient {
		return newCompartmentOrphanDeleteClient(manager, generatedFactory(manager))
	}
}

type compartmentOrphanDeleteClient struct {
	delegate          CompartmentServiceClient
	deleteCompartment func(context.Context, shared.OCID) error
	loadCompartment   func(context.Context, shared.OCID) (*identitysdk.Compartment, error)
}

var _ CompartmentServiceClient = compartmentOrphanDeleteClient{}

func newCompartmentOrphanDeleteClient(manager *CompartmentServiceManager, delegate CompartmentServiceClient) CompartmentServiceClient {
	client := compartmentOrphanDeleteClient{delegate: delegate}
	client.deleteCompartment = func(ctx context.Context, compartmentID shared.OCID) error {
		sdkClient, err := identitysdk.NewIdentityClientWithConfigurationProvider(manager.Provider)
		if err != nil {
			return fmt.Errorf("initialize Compartment delete OCI client: %w", err)
		}

		_, err = sdkClient.DeleteCompartment(ctx, identitysdk.DeleteCompartmentRequest{
			CompartmentId: common.String(string(compartmentID)),
		})
		return err
	}
	client.loadCompartment = func(ctx context.Context, compartmentID shared.OCID) (*identitysdk.Compartment, error) {
		sdkClient, err := identitysdk.NewIdentityClientWithConfigurationProvider(manager.Provider)
		if err != nil {
			return nil, fmt.Errorf("initialize Compartment get OCI client: %w", err)
		}

		response, err := sdkClient.GetCompartment(ctx, identitysdk.GetCompartmentRequest{
			CompartmentId: common.String(string(compartmentID)),
		})
		if err != nil {
			return nil, err
		}
		return &response.Compartment, nil
	}
	return client
}

func (c compartmentOrphanDeleteClient) CreateOrUpdate(
	ctx context.Context,
	resource *identityv1beta1.Compartment,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("compartment orphan delete delegate is not configured")
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c compartmentOrphanDeleteClient) Delete(ctx context.Context, resource *identityv1beta1.Compartment) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("compartment orphan delete delegate is not configured")
	}

	compartmentID := compartmentDeleteCurrentID(resource)
	if compartmentID == "" || c.deleteCompartment == nil {
		return c.delegate.Delete(ctx, resource)
	}

	if c.loadCompartment != nil {
		liveCompartment, err := c.loadCompartment(ctx, shared.OCID(compartmentID))
		if err == nil && shouldOrphanCompartmentDelete(liveCompartment.LifecycleState) {
			return true, nil
		}
		if err != nil && compartmentDeleteIsNotFound(err) {
			return true, nil
		}
	}

	err := c.deleteCompartment(ctx, shared.OCID(compartmentID))
	if err == nil || compartmentDeleteIsNotFound(err) {
		return true, nil
	}

	if c.loadCompartment != nil {
		liveCompartment, liveErr := c.loadCompartment(ctx, shared.OCID(compartmentID))
		switch {
		case liveErr == nil && shouldOrphanCompartmentDelete(liveCompartment.LifecycleState):
			return true, nil
		case liveErr != nil && compartmentDeleteIsNotFound(liveErr):
			return true, nil
		}
	}

	if compartmentDeleteIsConflict(err) {
		return false, nil
	}
	return false, err
}

func compartmentDeleteCurrentID(resource *identityv1beta1.Compartment) string {
	if resource == nil {
		return ""
	}
	if ocid := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); ocid != "" {
		return ocid
	}
	return strings.TrimSpace(resource.Status.Id)
}

func shouldOrphanCompartmentDelete(state identitysdk.CompartmentLifecycleStateEnum) bool {
	switch state {
	case identitysdk.CompartmentLifecycleStateDeleting,
		identitysdk.CompartmentLifecycleStateDeleted:
		return true
	default:
		return false
	}
}

func compartmentDeleteIsNotFound(err error) bool {
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		if serviceErr.GetHTTPStatusCode() == 404 {
			return true
		}
		switch serviceErr.GetCode() {
		case "NotFound", "NotAuthorizedOrNotFound":
			return true
		}
	}

	message := err.Error()
	return strings.Contains(message, "http status code: 404") ||
		strings.Contains(message, "NotFound") ||
		strings.Contains(message, "NotAuthorizedOrNotFound")
}

func compartmentDeleteIsConflict(err error) bool {
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		return serviceErr.GetHTTPStatusCode() == 409
	}

	var conflictErr errorutil.ConflictOciError
	if errors.As(err, &conflictErr) {
		return true
	}

	return strings.Contains(err.Error(), "http status code: 409")
}

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
	registerCompartmentRuntimeHooksMutator(func(manager *CompartmentServiceManager, hooks *CompartmentRuntimeHooks) {
		appendCompartmentOrphanDeleteRuntimeWrapper(manager, hooks)
	})
}

type compartmentOrphanDeleteClient struct {
	delegate          CompartmentServiceClient
	deleteCompartment func(context.Context, shared.OCID) error
	loadCompartment   func(context.Context, shared.OCID) (*identitysdk.Compartment, error)
	listCompartments  func(context.Context, shared.OCID, string) ([]identitysdk.Compartment, error)
}

var _ CompartmentServiceClient = compartmentOrphanDeleteClient{}

func appendCompartmentOrphanDeleteRuntimeWrapper(manager *CompartmentServiceManager, hooks *CompartmentRuntimeHooks) {
	if manager == nil || hooks == nil {
		return
	}

	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate CompartmentServiceClient) CompartmentServiceClient {
		return newCompartmentOrphanDeleteClient(manager, delegate)
	})
}

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
	client.listCompartments = func(ctx context.Context, parentID shared.OCID, name string) ([]identitysdk.Compartment, error) {
		sdkClient, err := identitysdk.NewIdentityClientWithConfigurationProvider(manager.Provider)
		if err != nil {
			return nil, fmt.Errorf("initialize Compartment list OCI client: %w", err)
		}

		var compartments []identitysdk.Compartment
		request := identitysdk.ListCompartmentsRequest{
			CompartmentId: common.String(string(parentID)),
			Name:          common.String(name),
		}
		for {
			response, err := sdkClient.ListCompartments(ctx, request)
			if err != nil {
				return nil, err
			}
			compartments = append(compartments, response.Items...)
			if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
				return compartments, nil
			}
			request.Page = response.OpcNextPage
		}
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
			found, confirmErr := c.confirmCompartmentByScopedList(ctx, resource, compartmentID)
			if confirmErr != nil {
				return false, confirmErr
			}
			if found != nil && shouldOrphanCompartmentDelete(found.LifecycleState) {
				return true, nil
			}
		}
	}

	err := c.deleteCompartment(ctx, shared.OCID(compartmentID))
	if err == nil {
		return true, nil
	}
	if compartmentDeleteIsNotFound(err) {
		found, confirmErr := c.confirmCompartmentByScopedList(ctx, resource, compartmentID)
		if confirmErr != nil {
			return false, confirmErr
		}
		if found != nil && shouldOrphanCompartmentDelete(found.LifecycleState) {
			return true, nil
		}
		if found == nil {
			return false, fmt.Errorf("Compartment delete returned ambiguous 404 and scoped list has not yet confirmed %s", compartmentID)
		}
		return false, fmt.Errorf("Compartment delete returned ambiguous 404 while scoped list still contains %s in state %s", compartmentID, found.LifecycleState)
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

func (c compartmentOrphanDeleteClient) confirmCompartmentByScopedList(
	ctx context.Context,
	resource *identityv1beta1.Compartment,
	compartmentID string,
) (*identitysdk.Compartment, error) {
	if c.listCompartments == nil || resource == nil {
		return nil, fmt.Errorf("Compartment %s returned ambiguous 404 and scoped list confirmation is unavailable", compartmentID)
	}
	parentID := strings.TrimSpace(resource.Spec.CompartmentId)
	name := strings.TrimSpace(resource.Spec.Name)
	if parentID == "" || name == "" {
		return nil, fmt.Errorf("Compartment %s returned ambiguous 404 without parent/name identity for scoped list confirmation", compartmentID)
	}
	items, err := c.listCompartments(ctx, shared.OCID(parentID), name)
	if err != nil {
		return nil, fmt.Errorf("confirm Compartment %s after ambiguous 404: %w", compartmentID, err)
	}
	for i := range items {
		if items[i].Id != nil && strings.TrimSpace(*items[i].Id) == compartmentID {
			return &items[i], nil
		}
	}
	return nil, nil
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

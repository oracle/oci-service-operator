/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package distributeddatabaseprivateendpoint

import (
	"context"
	"fmt"
	"reflect"

	"github.com/oracle/oci-go-sdk/v65/common"
	distributeddatabasesdk "github.com/oracle/oci-go-sdk/v65/distributeddatabase"
	distributeddatabasev1beta1 "github.com/oracle/oci-service-operator/api/distributeddatabase/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
)

func init() {
	registerDistributedDatabasePrivateEndpointRuntimeHooksMutator(func(_ *DistributedDatabasePrivateEndpointServiceManager, hooks *DistributedDatabasePrivateEndpointRuntimeHooks) {
		applyDistributedDatabasePrivateEndpointRuntimeHooks(hooks)
	})
}

func applyDistributedDatabasePrivateEndpointRuntimeHooks(hooks *DistributedDatabasePrivateEndpointRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedDistributedDatabasePrivateEndpointRuntimeSemantics()
	hooks.List.Fields = reviewedDistributedDatabasePrivateEndpointListFields()
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *distributeddatabasev1beta1.DistributedDatabasePrivateEndpoint,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildDistributedDatabasePrivateEndpointUpdateBody(resource, currentResponse)
	}
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedDistributedDatabasePrivateEndpointIdentity
}

func reviewedDistributedDatabasePrivateEndpointRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newDistributedDatabasePrivateEndpointRuntimeSemantics()
	semantics.List = &generatedruntime.ListSemantics{
		ResponseItemsField: "Items",
		MatchFields:        []string{"compartmentId", "displayName"},
	}
	semantics.AuxiliaryOperations = nil
	return semantics
}

func reviewedDistributedDatabasePrivateEndpointListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query"},
	}
}

func buildDistributedDatabasePrivateEndpointUpdateBody(
	resource *distributeddatabasev1beta1.DistributedDatabasePrivateEndpoint,
	currentResponse any,
) (distributeddatabasesdk.UpdateDistributedDatabasePrivateEndpointDetails, bool, error) {
	if resource == nil {
		return distributeddatabasesdk.UpdateDistributedDatabasePrivateEndpointDetails{}, false, fmt.Errorf("DistributedDatabasePrivateEndpoint resource is nil")
	}

	current, ok := distributedDatabasePrivateEndpointFromResponse(currentResponse)
	if !ok {
		return distributeddatabasesdk.UpdateDistributedDatabasePrivateEndpointDetails{}, false, fmt.Errorf("current DistributedDatabasePrivateEndpoint response does not expose a DistributedDatabasePrivateEndpoint body")
	}

	updateDetails := distributeddatabasesdk.UpdateDistributedDatabasePrivateEndpointDetails{}
	updateNeeded := false

	if desired, ok := desiredDistributedDatabasePrivateEndpointStringUpdate(resource.Spec.DisplayName, current.DisplayName); ok {
		updateDetails.DisplayName = desired
		updateNeeded = true
	}
	if desired, ok := desiredDistributedDatabasePrivateEndpointStringUpdate(resource.Spec.Description, current.Description); ok {
		updateDetails.Description = desired
		updateNeeded = true
	}

	desiredNsgIDs := desiredDistributedDatabasePrivateEndpointNsgIDsForUpdate(resource.Spec.NsgIds, current.NsgIds)
	if !reflect.DeepEqual(current.NsgIds, desiredNsgIDs) {
		updateDetails.NsgIds = desiredNsgIDs
		updateNeeded = true
	}

	desiredFreeformTags := desiredDistributedDatabasePrivateEndpointFreeformTagsForUpdate(resource.Spec.FreeformTags, current.FreeformTags)
	if !reflect.DeepEqual(current.FreeformTags, desiredFreeformTags) {
		updateDetails.FreeformTags = desiredFreeformTags
		updateNeeded = true
	}

	desiredDefinedTags := desiredDistributedDatabasePrivateEndpointDefinedTagsForUpdate(resource.Spec.DefinedTags, current.DefinedTags)
	if !reflect.DeepEqual(current.DefinedTags, desiredDefinedTags) {
		updateDetails.DefinedTags = desiredDefinedTags
		updateNeeded = true
	}

	if !updateNeeded {
		return distributeddatabasesdk.UpdateDistributedDatabasePrivateEndpointDetails{}, false, nil
	}
	return updateDetails, true, nil
}

func distributedDatabasePrivateEndpointFromResponse(response any) (distributeddatabasesdk.DistributedDatabasePrivateEndpoint, bool) {
	switch current := response.(type) {
	case distributeddatabasesdk.CreateDistributedDatabasePrivateEndpointResponse:
		return current.DistributedDatabasePrivateEndpoint, true
	case *distributeddatabasesdk.CreateDistributedDatabasePrivateEndpointResponse:
		if current == nil {
			return distributeddatabasesdk.DistributedDatabasePrivateEndpoint{}, false
		}
		return current.DistributedDatabasePrivateEndpoint, true
	case distributeddatabasesdk.GetDistributedDatabasePrivateEndpointResponse:
		return current.DistributedDatabasePrivateEndpoint, true
	case *distributeddatabasesdk.GetDistributedDatabasePrivateEndpointResponse:
		if current == nil {
			return distributeddatabasesdk.DistributedDatabasePrivateEndpoint{}, false
		}
		return current.DistributedDatabasePrivateEndpoint, true
	case distributeddatabasesdk.UpdateDistributedDatabasePrivateEndpointResponse:
		return current.DistributedDatabasePrivateEndpoint, true
	case *distributeddatabasesdk.UpdateDistributedDatabasePrivateEndpointResponse:
		if current == nil {
			return distributeddatabasesdk.DistributedDatabasePrivateEndpoint{}, false
		}
		return current.DistributedDatabasePrivateEndpoint, true
	case distributeddatabasesdk.DistributedDatabasePrivateEndpoint:
		return current, true
	case *distributeddatabasesdk.DistributedDatabasePrivateEndpoint:
		if current == nil {
			return distributeddatabasesdk.DistributedDatabasePrivateEndpoint{}, false
		}
		return *current, true
	case distributeddatabasesdk.DistributedDatabasePrivateEndpointSummary:
		return distributedDatabasePrivateEndpointFromSummary(current), true
	case *distributeddatabasesdk.DistributedDatabasePrivateEndpointSummary:
		if current == nil {
			return distributeddatabasesdk.DistributedDatabasePrivateEndpoint{}, false
		}
		return distributedDatabasePrivateEndpointFromSummary(*current), true
	default:
		return distributeddatabasesdk.DistributedDatabasePrivateEndpoint{}, false
	}
}

func distributedDatabasePrivateEndpointFromSummary(
	summary distributeddatabasesdk.DistributedDatabasePrivateEndpointSummary,
) distributeddatabasesdk.DistributedDatabasePrivateEndpoint {
	return distributeddatabasesdk.DistributedDatabasePrivateEndpoint{
		Id:                                     summary.Id,
		CompartmentId:                          summary.CompartmentId,
		SubnetId:                               summary.SubnetId,
		VcnId:                                  summary.VcnId,
		DisplayName:                            summary.DisplayName,
		TimeCreated:                            summary.TimeCreated,
		TimeUpdated:                            summary.TimeUpdated,
		LifecycleState:                         summary.LifecycleState,
		Description:                            summary.Description,
		NsgIds:                                 summary.NsgIds,
		LifecycleDetails:                       summary.LifecycleDetails,
		FreeformTags:                           summary.FreeformTags,
		DefinedTags:                            summary.DefinedTags,
		SystemTags:                             summary.SystemTags,
		GloballyDistributedDatabases:           nil,
		GloballyDistributedAutonomousDatabases: nil,
	}
}

func clearTrackedDistributedDatabasePrivateEndpointIdentity(
	resource *distributeddatabasev1beta1.DistributedDatabasePrivateEndpoint,
) {
	if resource == nil {
		return
	}
	resource.Status.Id = ""
	resource.Status.OsokStatus = shared.OSOKStatus{}
}

func desiredDistributedDatabasePrivateEndpointStringUpdate(spec string, current *string) (*string, bool) {
	currentValue := ""
	if current != nil {
		currentValue = *current
	}
	if spec == currentValue {
		return nil, false
	}
	if spec == "" && current == nil {
		return nil, false
	}
	return common.String(spec), true
}

func desiredDistributedDatabasePrivateEndpointNsgIDsForUpdate(spec []string, current []string) []string {
	if spec != nil {
		return append([]string{}, spec...)
	}
	if current != nil {
		return []string{}
	}
	return nil
}

func desiredDistributedDatabasePrivateEndpointFreeformTagsForUpdate(spec map[string]string, current map[string]string) map[string]string {
	if spec != nil {
		return cloneDistributedDatabasePrivateEndpointStringMap(spec)
	}
	if current != nil {
		return map[string]string{}
	}
	return nil
}

func desiredDistributedDatabasePrivateEndpointDefinedTagsForUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) map[string]map[string]interface{} {
	if spec != nil {
		return *util.ConvertToOciDefinedTags(&spec)
	}
	if current != nil {
		return map[string]map[string]interface{}{}
	}
	return nil
}

func cloneDistributedDatabasePrivateEndpointStringMap(spec map[string]string) map[string]string {
	if spec == nil {
		return nil
	}
	out := make(map[string]string, len(spec))
	for key, value := range spec {
		out[key] = value
	}
	return out
}

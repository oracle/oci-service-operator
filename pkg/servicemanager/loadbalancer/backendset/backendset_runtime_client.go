/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package backendset

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	loadbalancersdk "github.com/oracle/oci-go-sdk/v65/loadbalancer"
	loadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/loadbalancer/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

type backendSetRuntimeOCIClient interface {
	CreateBackendSet(context.Context, loadbalancersdk.CreateBackendSetRequest) (loadbalancersdk.CreateBackendSetResponse, error)
	GetBackendSet(context.Context, loadbalancersdk.GetBackendSetRequest) (loadbalancersdk.GetBackendSetResponse, error)
	ListBackendSets(context.Context, loadbalancersdk.ListBackendSetsRequest) (loadbalancersdk.ListBackendSetsResponse, error)
	UpdateBackendSet(context.Context, loadbalancersdk.UpdateBackendSetRequest) (loadbalancersdk.UpdateBackendSetResponse, error)
	DeleteBackendSet(context.Context, loadbalancersdk.DeleteBackendSetRequest) (loadbalancersdk.DeleteBackendSetResponse, error)
}

type backendSetIdentity struct {
	loadBalancerID string
	backendSetName string
}

func init() {
	registerBackendSetRuntimeHooksMutator(func(_ *BackendSetServiceManager, hooks *BackendSetRuntimeHooks) {
		applyBackendSetRuntimeHooks(hooks)
	})
}

func applyBackendSetRuntimeHooks(hooks *BackendSetRuntimeHooks) {
	if hooks == nil {
		return
	}

	getCall := hooks.Get.Call
	hooks.Semantics = newBackendSetRuntimeSemantics()
	hooks.BuildUpdateBody = func(
		ctx context.Context,
		resource *loadbalancerv1beta1.BackendSet,
		namespace string,
		currentResponse any,
	) (any, bool, error) {
		return buildBackendSetUpdateBody(ctx, resource, namespace, currentResponse)
	}
	hooks.Identity = generatedruntime.IdentityHooks[*loadbalancerv1beta1.BackendSet]{
		Resolve: func(resource *loadbalancerv1beta1.BackendSet) (any, error) {
			return resolveBackendSetIdentity(resource)
		},
		RecordPath: func(resource *loadbalancerv1beta1.BackendSet, identity any) {
			recordBackendSetPathIdentity(resource, identity.(backendSetIdentity))
		},
		RecordTracked: func(resource *loadbalancerv1beta1.BackendSet, identity any, _ string) {
			recordBackendSetTrackedIdentity(resource, identity.(backendSetIdentity))
		},
		LookupExisting: func(ctx context.Context, resource *loadbalancerv1beta1.BackendSet, identity any) (any, error) {
			if backendSetHasTrackedID(resource) {
				return nil, nil
			}
			return lookupExistingBackendSet(ctx, getCall, identity.(backendSetIdentity))
		},
		SeedSyntheticTrackedID: func(resource *loadbalancerv1beta1.BackendSet, identity any) func() {
			return seedSyntheticBackendSetOCID(resource, identity.(backendSetIdentity))
		},
	}
	hooks.Create.Fields = backendSetCreateFields()
	hooks.Get.Fields = backendSetGetFields()
	hooks.List.Fields = backendSetListFields()
	hooks.Update.Fields = backendSetUpdateFields()
	hooks.Delete.Fields = backendSetDeleteFields()
}

func newBackendSetRuntimeHooksWithOCIClient(client backendSetRuntimeOCIClient) BackendSetRuntimeHooks {
	return BackendSetRuntimeHooks{
		Semantics: newBackendSetRuntimeSemantics(),
		Identity:  generatedruntime.IdentityHooks[*loadbalancerv1beta1.BackendSet]{},
		Read:      generatedruntime.ReadHooks{},
		Create: runtimeOperationHooks[loadbalancersdk.CreateBackendSetRequest, loadbalancersdk.CreateBackendSetResponse]{
			Fields: backendSetCreateFields(),
			Call: func(ctx context.Context, request loadbalancersdk.CreateBackendSetRequest) (loadbalancersdk.CreateBackendSetResponse, error) {
				return client.CreateBackendSet(ctx, request)
			},
		},
		Get: runtimeOperationHooks[loadbalancersdk.GetBackendSetRequest, loadbalancersdk.GetBackendSetResponse]{
			Fields: backendSetGetFields(),
			Call: func(ctx context.Context, request loadbalancersdk.GetBackendSetRequest) (loadbalancersdk.GetBackendSetResponse, error) {
				return client.GetBackendSet(ctx, request)
			},
		},
		List: runtimeOperationHooks[loadbalancersdk.ListBackendSetsRequest, loadbalancersdk.ListBackendSetsResponse]{
			Fields: backendSetListFields(),
			Call: func(ctx context.Context, request loadbalancersdk.ListBackendSetsRequest) (loadbalancersdk.ListBackendSetsResponse, error) {
				return client.ListBackendSets(ctx, request)
			},
		},
		Update: runtimeOperationHooks[loadbalancersdk.UpdateBackendSetRequest, loadbalancersdk.UpdateBackendSetResponse]{
			Fields: backendSetUpdateFields(),
			Call: func(ctx context.Context, request loadbalancersdk.UpdateBackendSetRequest) (loadbalancersdk.UpdateBackendSetResponse, error) {
				return client.UpdateBackendSet(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[loadbalancersdk.DeleteBackendSetRequest, loadbalancersdk.DeleteBackendSetResponse]{
			Fields: backendSetDeleteFields(),
			Call: func(ctx context.Context, request loadbalancersdk.DeleteBackendSetRequest) (loadbalancersdk.DeleteBackendSetResponse, error) {
				return client.DeleteBackendSet(ctx, request)
			},
		},
		WrapGeneratedClient: []func(BackendSetServiceClient) BackendSetServiceClient{},
	}
}

func backendSetCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		backendSetLoadBalancerIDField(),
		{
			FieldName:    "CreateBackendSetDetails",
			RequestName:  "CreateBackendSetDetails",
			Contribution: "body",
		},
	}
}

func backendSetGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		backendSetLoadBalancerIDField(),
		backendSetNameField(),
	}
}

func backendSetListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		backendSetLoadBalancerIDField(),
	}
}

func backendSetUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		backendSetLoadBalancerIDField(),
		backendSetNameField(),
		{
			FieldName:    "UpdateBackendSetDetails",
			RequestName:  "UpdateBackendSetDetails",
			Contribution: "body",
		},
	}
}

func backendSetDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		backendSetLoadBalancerIDField(),
		backendSetNameField(),
	}
}

func buildBackendSetUpdateBody(
	ctx context.Context,
	resource *loadbalancerv1beta1.BackendSet,
	namespace string,
	currentResponse any,
) (loadbalancersdk.UpdateBackendSetDetails, bool, error) {
	if resource == nil {
		return loadbalancersdk.UpdateBackendSetDetails{}, false, fmt.Errorf("backendset resource is nil")
	}

	resolvedSpec, err := generatedruntime.ResolveSpecValue(resource, ctx, nil, namespace)
	if err != nil {
		return loadbalancersdk.UpdateBackendSetDetails{}, false, err
	}
	resolvedSpec = overlayBackendSetExistingBoolFields(reflect.ValueOf(resource.Spec), resolvedSpec)

	desired, err := backendSetUpdateDetailsFromValue(resolvedSpec)
	if err != nil {
		return loadbalancersdk.UpdateBackendSetDetails{}, false, fmt.Errorf("build desired BackendSet update details: %w", err)
	}

	currentSource, err := backendSetUpdateSource(resource, currentResponse)
	if err != nil {
		return loadbalancersdk.UpdateBackendSetDetails{}, false, err
	}

	current, err := backendSetUpdateDetailsFromValue(currentSource)
	if err != nil {
		return loadbalancersdk.UpdateBackendSetDetails{}, false, fmt.Errorf("build current BackendSet update details: %w", err)
	}

	updateNeeded, err := backendSetUpdateNeeded(desired, current)
	if err != nil {
		return loadbalancersdk.UpdateBackendSetDetails{}, false, err
	}
	if !updateNeeded {
		return loadbalancersdk.UpdateBackendSetDetails{}, false, nil
	}

	return desired, true, nil
}

func backendSetLoadBalancerIDField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:    "LoadBalancerId",
		RequestName:  "loadBalancerId",
		Contribution: "path",
		LookupPaths:  []string{"status.loadBalancerId", "spec.loadBalancerId"},
	}
}

func backendSetNameField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:    "BackendSetName",
		RequestName:  "backendSetName",
		Contribution: "path",
		LookupPaths:  []string{"status.name", "spec.name", "name"},
	}
}

func lookupExistingBackendSet(
	ctx context.Context,
	getCall func(context.Context, loadbalancersdk.GetBackendSetRequest) (loadbalancersdk.GetBackendSetResponse, error),
	identity backendSetIdentity,
) (any, error) {
	if getCall == nil {
		return nil, nil
	}

	return getCall(ctx, loadbalancersdk.GetBackendSetRequest{
		LoadBalancerId: common.String(identity.loadBalancerID),
		BackendSetName: common.String(identity.backendSetName),
	})
}

func resolveBackendSetIdentity(resource *loadbalancerv1beta1.BackendSet) (backendSetIdentity, error) {
	identity := backendSetIdentity{
		loadBalancerID: firstNonEmptyTrim(resource.Status.LoadBalancerId, resource.Spec.LoadBalancerId),
		backendSetName: firstNonEmptyTrim(resource.Status.Name, resource.Spec.Name, resource.Name),
	}
	if identity.loadBalancerID == "" {
		return backendSetIdentity{}, fmt.Errorf("resolve BackendSet identity: loadBalancerId is empty")
	}
	if identity.backendSetName == "" {
		return backendSetIdentity{}, fmt.Errorf("resolve BackendSet identity: backend set name is empty")
	}
	return identity, nil
}

func recordBackendSetPathIdentity(resource *loadbalancerv1beta1.BackendSet, identity backendSetIdentity) {
	if resource == nil {
		return
	}
	resource.Status.LoadBalancerId = identity.loadBalancerID
	resource.Status.Name = identity.backendSetName
}

func recordBackendSetTrackedIdentity(resource *loadbalancerv1beta1.BackendSet, identity backendSetIdentity) {
	recordBackendSetPathIdentity(resource, identity)
	if !backendSetHasTrackedID(resource) {
		resource.Status.OsokStatus.Ocid = backendSetSyntheticOCID(identity)
	}
}

func backendSetHasTrackedID(resource *loadbalancerv1beta1.BackendSet) bool {
	if resource == nil {
		return false
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)) != ""
}

func seedSyntheticBackendSetOCID(resource *loadbalancerv1beta1.BackendSet, identity backendSetIdentity) func() {
	if resource == nil {
		return func() {}
	}

	previous := resource.Status.OsokStatus.Ocid
	resource.Status.OsokStatus.Ocid = backendSetSyntheticOCID(identity)
	return func() {
		resource.Status.OsokStatus.Ocid = previous
	}
}

func backendSetSyntheticOCID(identity backendSetIdentity) shared.OCID {
	return shared.OCID("backendset/" + identity.backendSetName)
}

func backendSetUpdateSource(resource *loadbalancerv1beta1.BackendSet, currentResponse any) (any, error) {
	switch current := currentResponse.(type) {
	case nil:
		if resource == nil {
			return nil, fmt.Errorf("backendset resource is nil")
		}
		return resource.Status, nil
	case loadbalancersdk.BackendSet:
		return current, nil
	case *loadbalancersdk.BackendSet:
		if current == nil {
			return nil, fmt.Errorf("current BackendSet response is nil")
		}
		return *current, nil
	case loadbalancersdk.GetBackendSetResponse:
		return current.BackendSet, nil
	case *loadbalancersdk.GetBackendSetResponse:
		if current == nil {
			return nil, fmt.Errorf("current BackendSet response is nil")
		}
		return current.BackendSet, nil
	default:
		return currentResponse, nil
	}
}

func backendSetUpdateDetailsFromValue(value any) (loadbalancersdk.UpdateBackendSetDetails, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return loadbalancersdk.UpdateBackendSetDetails{}, fmt.Errorf("marshal BackendSet update details source: %w", err)
	}

	var details loadbalancersdk.UpdateBackendSetDetails
	if err := json.Unmarshal(payload, &details); err != nil {
		return loadbalancersdk.UpdateBackendSetDetails{}, fmt.Errorf("decode BackendSet update details: %w", err)
	}
	return details, nil
}

func backendSetUpdateNeeded(desired loadbalancersdk.UpdateBackendSetDetails, current loadbalancersdk.UpdateBackendSetDetails) (bool, error) {
	desiredComparable, err := cloneBackendSetUpdateDetails(desired)
	if err != nil {
		return false, err
	}
	currentComparable, err := cloneBackendSetUpdateDetails(current)
	if err != nil {
		return false, err
	}
	normalizeBackendSetOptionalFalseBools(reflect.ValueOf(&desiredComparable))
	normalizeBackendSetOptionalFalseBools(reflect.ValueOf(&currentComparable))

	desiredPayload, err := json.Marshal(desiredComparable)
	if err != nil {
		return false, fmt.Errorf("marshal desired BackendSet update details: %w", err)
	}
	currentPayload, err := json.Marshal(currentComparable)
	if err != nil {
		return false, fmt.Errorf("marshal current BackendSet update details: %w", err)
	}
	return string(desiredPayload) != string(currentPayload), nil
}

func cloneBackendSetUpdateDetails(details loadbalancersdk.UpdateBackendSetDetails) (loadbalancersdk.UpdateBackendSetDetails, error) {
	payload, err := json.Marshal(details)
	if err != nil {
		return loadbalancersdk.UpdateBackendSetDetails{}, fmt.Errorf("marshal BackendSet update details clone: %w", err)
	}

	var cloned loadbalancersdk.UpdateBackendSetDetails
	if err := json.Unmarshal(payload, &cloned); err != nil {
		return loadbalancersdk.UpdateBackendSetDetails{}, fmt.Errorf("decode BackendSet update details clone: %w", err)
	}
	return cloned, nil
}

func overlayBackendSetExistingBoolFields(value reflect.Value, decoded any) any {
	overlaid, _ := overlayBackendSetExistingBoolFieldsValue(value, decoded)
	return overlaid
}

func overlayBackendSetExistingBoolFieldsValue(value reflect.Value, decoded any) (any, bool) {
	value, ok := indirectBackendSetValue(value)
	if !ok {
		return decoded, decoded != nil
	}

	switch value.Kind() {
	case reflect.Struct:
		decodedMap, ok := decoded.(map[string]any)
		if !ok || decodedMap == nil {
			return decoded, decoded != nil
		}
		hasAny := len(decodedMap) > 0
		valueType := value.Type()
		for i := 0; i < value.NumField(); i++ {
			fieldType := valueType.Field(i)
			if !fieldType.IsExported() {
				continue
			}

			jsonName := backendSetJSONFieldName(fieldType)
			if jsonName == "" {
				continue
			}

			childDecoded, exists := decodedMap[jsonName]
			if !exists {
				continue
			}

			fieldValue := value.Field(i)
			indirectField, ok := indirectBackendSetValue(fieldValue)
			if !ok {
				continue
			}

			switch indirectField.Kind() {
			case reflect.Bool:
				decodedMap[jsonName] = indirectField.Bool()
				hasAny = true
			case reflect.Struct, reflect.Slice, reflect.Array:
				child, childHasAny := overlayBackendSetExistingBoolFieldsValue(fieldValue, childDecoded)
				if childHasAny {
					decodedMap[jsonName] = child
					hasAny = true
				}
			}
		}
		return decodedMap, hasAny
	case reflect.Slice, reflect.Array:
		decodedSlice, ok := decoded.([]any)
		if !ok {
			return decoded, decoded != nil
		}
		hasAny := len(decodedSlice) > 0
		limit := value.Len()
		if len(decodedSlice) < limit {
			limit = len(decodedSlice)
		}
		for i := 0; i < limit; i++ {
			child, childHasAny := overlayBackendSetExistingBoolFieldsValue(value.Index(i), decodedSlice[i])
			if childHasAny {
				decodedSlice[i] = child
				hasAny = true
			}
		}
		return decodedSlice, hasAny
	default:
		return decoded, decoded != nil
	}
}

func normalizeBackendSetOptionalFalseBools(value reflect.Value) {
	if !value.IsValid() {
		return
	}

	switch value.Kind() {
	case reflect.Pointer:
		if value.IsNil() {
			return
		}
		if value.Elem().Kind() == reflect.Bool {
			if !value.Elem().Bool() && value.CanSet() {
				value.Set(reflect.Zero(value.Type()))
			}
			return
		}
		normalizeBackendSetOptionalFalseBools(value.Elem())
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			normalizeBackendSetOptionalFalseBools(value.Field(i))
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			normalizeBackendSetOptionalFalseBools(value.Index(i))
		}
	}
}

func indirectBackendSetValue(value reflect.Value) (reflect.Value, bool) {
	for value.IsValid() && (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) {
		if value.IsNil() {
			return reflect.Value{}, false
		}
		value = value.Elem()
	}
	return value, value.IsValid()
}

func backendSetJSONFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" || tag == "-" {
		return ""
	}
	return strings.Split(tag, ",")[0]
}

func firstNonEmptyTrim(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

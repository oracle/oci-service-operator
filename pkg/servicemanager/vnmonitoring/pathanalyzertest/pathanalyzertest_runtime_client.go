/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package pathanalyzertest

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	vnmonitoringsdk "github.com/oracle/oci-go-sdk/v65/vnmonitoring"
	vnmonitoringv1beta1 "github.com/oracle/oci-service-operator/api/vnmonitoring/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

type pathAnalyzerTestOCIClient interface {
	CreatePathAnalyzerTest(context.Context, vnmonitoringsdk.CreatePathAnalyzerTestRequest) (vnmonitoringsdk.CreatePathAnalyzerTestResponse, error)
	GetPathAnalyzerTest(context.Context, vnmonitoringsdk.GetPathAnalyzerTestRequest) (vnmonitoringsdk.GetPathAnalyzerTestResponse, error)
	ListPathAnalyzerTests(context.Context, vnmonitoringsdk.ListPathAnalyzerTestsRequest) (vnmonitoringsdk.ListPathAnalyzerTestsResponse, error)
	UpdatePathAnalyzerTest(context.Context, vnmonitoringsdk.UpdatePathAnalyzerTestRequest) (vnmonitoringsdk.UpdatePathAnalyzerTestResponse, error)
	DeletePathAnalyzerTest(context.Context, vnmonitoringsdk.DeletePathAnalyzerTestRequest) (vnmonitoringsdk.DeletePathAnalyzerTestResponse, error)
}

type pathAnalyzerTestIdentity struct {
	compartmentID           string
	displayName             string
	protocol                int
	sourceEndpoint          string
	destinationEndpoint     string
	protocolParameters      string
	isBiDirectionalAnalysis bool
}

type pathAnalyzerTestComparable struct {
	displayName             string
	protocol                int
	sourceEndpoint          string
	destinationEndpoint     string
	protocolParameters      string
	isBiDirectionalAnalysis bool
	freeformTags            map[string]string
	definedTags             string
}

func init() {
	registerPathAnalyzerTestRuntimeHooksMutator(func(manager *PathAnalyzerTestServiceManager, hooks *PathAnalyzerTestRuntimeHooks) {
		applyPathAnalyzerTestRuntimeHooks(manager, hooks)
	})
}

func applyPathAnalyzerTestRuntimeHooks(_ *PathAnalyzerTestServiceManager, hooks *PathAnalyzerTestRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedPathAnalyzerTestRuntimeSemantics()
	hooks.Identity.Resolve = resolvePathAnalyzerTestIdentity
	hooks.Identity.GuardExistingBeforeCreate = guardPathAnalyzerTestExistingBeforeCreate
	hooks.List.Fields = pathAnalyzerTestListFields()
	wrapPathAnalyzerTestListPages(hooks)
	hooks.Identity.LookupExisting = lookupExistingPathAnalyzerTest(hooks.List.Call)
	hooks.BuildCreateBody = func(
		ctx context.Context,
		resource *vnmonitoringv1beta1.PathAnalyzerTest,
		namespace string,
	) (any, error) {
		return buildPathAnalyzerTestCreateBody(ctx, resource, namespace)
	}
	hooks.BuildUpdateBody = func(
		ctx context.Context,
		resource *vnmonitoringv1beta1.PathAnalyzerTest,
		namespace string,
		currentResponse any,
	) (any, bool, error) {
		return buildPathAnalyzerTestUpdateBody(ctx, resource, namespace, currentResponse)
	}
}

func newPathAnalyzerTestServiceClientWithOCIClient(client pathAnalyzerTestOCIClient) PathAnalyzerTestServiceClient {
	hooks := newPathAnalyzerTestRuntimeHooksWithOCIClient(client)
	delegate := defaultPathAnalyzerTestServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*vnmonitoringv1beta1.PathAnalyzerTest](
			buildPathAnalyzerTestGeneratedRuntimeConfig(&PathAnalyzerTestServiceManager{}, hooks),
		),
	}
	return wrapPathAnalyzerTestGeneratedClient(hooks, delegate)
}

func newPathAnalyzerTestRuntimeHooksWithOCIClient(client pathAnalyzerTestOCIClient) PathAnalyzerTestRuntimeHooks {
	hooks := newPathAnalyzerTestDefaultRuntimeHooks(vnmonitoringsdk.VnMonitoringClient{})
	if client != nil {
		hooks.Create.Call = client.CreatePathAnalyzerTest
		hooks.Get.Call = client.GetPathAnalyzerTest
		hooks.List.Call = client.ListPathAnalyzerTests
		hooks.Update.Call = client.UpdatePathAnalyzerTest
		hooks.Delete.Call = client.DeletePathAnalyzerTest
	}
	applyPathAnalyzerTestRuntimeHooks(nil, &hooks)
	return hooks
}

func reviewedPathAnalyzerTestRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newPathAnalyzerTestRuntimeSemantics()
	semantics.Delete = generatedruntime.DeleteSemantics{
		Policy:         "required",
		PendingStates:  []string{},
		TerminalStates: []string{"DELETED", "NOT_FOUND"},
	}
	semantics.List = &generatedruntime.ListSemantics{
		ResponseItemsField: "Items",
		MatchFields: []string{
			"compartmentId",
			"displayName",
			"protocol",
			"sourceEndpoint",
			"destinationEndpoint",
			"protocolParameters",
			"queryOptions",
		},
	}
	semantics.Hooks = generatedruntime.HookSet{
		Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "PathAnalyzerTest", Action: "CreatePathAnalyzerTest"}},
		Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "PathAnalyzerTest", Action: "UpdatePathAnalyzerTest"}},
		Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "PathAnalyzerTest", Action: "DeletePathAnalyzerTest"}},
	}
	semantics.CreateFollowUp = generatedruntime.FollowUpSemantics{
		Strategy: "read-after-write",
		Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "PathAnalyzerTest", Action: "GetPathAnalyzerTest"}},
	}
	semantics.UpdateFollowUp = generatedruntime.FollowUpSemantics{
		Strategy: "read-after-write",
		Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "PathAnalyzerTest", Action: "GetPathAnalyzerTest"}},
	}
	semantics.DeleteFollowUp = generatedruntime.FollowUpSemantics{
		Strategy: "confirm-delete",
		Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "PathAnalyzerTest", Action: "GetPathAnalyzerTest"}},
	}
	semantics.AuxiliaryOperations = nil
	semantics.Unsupported = nil
	return semantics
}

func pathAnalyzerTestListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "spec.compartmentId", "compartmentId"},
		},
		{
			FieldName:    "DisplayName",
			RequestName:  "displayName",
			Contribution: "query",
			LookupPaths:  []string{"status.displayName", "spec.displayName", "displayName"},
		},
	}
}

func wrapPathAnalyzerTestListPages(hooks *PathAnalyzerTestRuntimeHooks) {
	if hooks == nil || hooks.List.Call == nil {
		return
	}

	call := hooks.List.Call
	hooks.List.Call = func(ctx context.Context, request vnmonitoringsdk.ListPathAnalyzerTestsRequest) (vnmonitoringsdk.ListPathAnalyzerTestsResponse, error) {
		return listPathAnalyzerTestPages(ctx, call, request)
	}
}

func listPathAnalyzerTestPages(
	ctx context.Context,
	call func(context.Context, vnmonitoringsdk.ListPathAnalyzerTestsRequest) (vnmonitoringsdk.ListPathAnalyzerTestsResponse, error),
	request vnmonitoringsdk.ListPathAnalyzerTestsRequest,
) (vnmonitoringsdk.ListPathAnalyzerTestsResponse, error) {
	var combined vnmonitoringsdk.ListPathAnalyzerTestsResponse
	for {
		response, err := call(ctx, request)
		if err != nil {
			return response, err
		}
		if combined.OpcRequestId == nil {
			combined.OpcRequestId = response.OpcRequestId
		}
		combined.RawResponse = response.RawResponse
		combined.Items = append(combined.Items, response.Items...)
		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			combined.OpcNextPage = nil
			return combined, nil
		}
		request.Page = response.OpcNextPage
	}
}

func guardPathAnalyzerTestExistingBeforeCreate(
	_ context.Context,
	resource *vnmonitoringv1beta1.PathAnalyzerTest,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("PathAnalyzerTest resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("PathAnalyzerTest spec.compartmentId is required")
	}
	if strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func resolvePathAnalyzerTestIdentity(resource *vnmonitoringv1beta1.PathAnalyzerTest) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("PathAnalyzerTest resource is nil")
	}
	if strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return nil, nil
	}

	sourceEndpoint, err := pathAnalyzerTestSourceEndpointFromSpec(resource.Spec.SourceEndpoint)
	if err != nil {
		return nil, err
	}
	destinationEndpoint, err := pathAnalyzerTestDestinationEndpointFromSpec(resource.Spec.DestinationEndpoint)
	if err != nil {
		return nil, err
	}
	protocolParameters, err := pathAnalyzerTestProtocolParametersFromSpec(resource.Spec.ProtocolParameters)
	if err != nil {
		return nil, err
	}
	sourceEndpointJSON, err := pathAnalyzerTestCanonicalJSONString(sourceEndpoint)
	if err != nil {
		return nil, err
	}
	destinationEndpointJSON, err := pathAnalyzerTestCanonicalJSONString(destinationEndpoint)
	if err != nil {
		return nil, err
	}
	protocolParametersJSON, err := pathAnalyzerTestCanonicalJSONString(protocolParameters)
	if err != nil {
		return nil, err
	}

	return pathAnalyzerTestIdentity{
		compartmentID:           strings.TrimSpace(resource.Spec.CompartmentId),
		displayName:             resource.Spec.DisplayName,
		protocol:                resource.Spec.Protocol,
		sourceEndpoint:          sourceEndpointJSON,
		destinationEndpoint:     destinationEndpointJSON,
		protocolParameters:      protocolParametersJSON,
		isBiDirectionalAnalysis: resource.Spec.QueryOptions.IsBiDirectionalAnalysis,
	}, nil
}

func lookupExistingPathAnalyzerTest(
	listCall func(context.Context, vnmonitoringsdk.ListPathAnalyzerTestsRequest) (vnmonitoringsdk.ListPathAnalyzerTestsResponse, error),
) func(context.Context, *vnmonitoringv1beta1.PathAnalyzerTest, any) (any, error) {
	return func(
		ctx context.Context,
		resource *vnmonitoringv1beta1.PathAnalyzerTest,
		identity any,
	) (any, error) {
		if listCall == nil || resource == nil {
			return nil, nil
		}

		wanted, ok := identity.(pathAnalyzerTestIdentity)
		if !ok {
			return nil, fmt.Errorf("PathAnalyzerTest identity has unexpected type %T", identity)
		}

		request := vnmonitoringsdk.ListPathAnalyzerTestsRequest{
			CompartmentId: common.String(wanted.compartmentID),
			DisplayName:   common.String(wanted.displayName),
		}
		response, err := listCall(ctx, request)
		if err != nil {
			return nil, err
		}

		matches := make([]vnmonitoringsdk.PathAnalyzerTestSummary, 0, len(response.Items))
		for _, item := range response.Items {
			matched, err := pathAnalyzerTestIdentityMatchesSummary(wanted, item)
			if err != nil {
				return nil, err
			}
			if matched {
				matches = append(matches, item)
			}
		}

		switch len(matches) {
		case 0:
			return nil, nil
		case 1:
			return matches[0], nil
		default:
			return nil, fmt.Errorf("PathAnalyzerTest list response returned multiple matching resources")
		}
	}
}

func pathAnalyzerTestIdentityMatchesSummary(
	wanted pathAnalyzerTestIdentity,
	current vnmonitoringsdk.PathAnalyzerTestSummary,
) (bool, error) {
	sourceEndpointJSON, err := pathAnalyzerTestCanonicalJSONString(current.SourceEndpoint)
	if err != nil {
		return false, err
	}
	destinationEndpointJSON, err := pathAnalyzerTestCanonicalJSONString(current.DestinationEndpoint)
	if err != nil {
		return false, err
	}
	protocolParametersJSON, err := pathAnalyzerTestCanonicalJSONString(current.ProtocolParameters)
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(wanted.compartmentID) == stringValue(current.CompartmentId) &&
		wanted.displayName == stringValue(current.DisplayName) &&
		wanted.protocol == intValue(current.Protocol) &&
		wanted.sourceEndpoint == sourceEndpointJSON &&
		wanted.destinationEndpoint == destinationEndpointJSON &&
		wanted.protocolParameters == protocolParametersJSON &&
		wanted.isBiDirectionalAnalysis == queryOptionsValue(current.QueryOptions), nil
}

func buildPathAnalyzerTestCreateBody(
	_ context.Context,
	resource *vnmonitoringv1beta1.PathAnalyzerTest,
	_ string,
) (vnmonitoringsdk.CreatePathAnalyzerTestDetails, error) {
	if resource == nil {
		return vnmonitoringsdk.CreatePathAnalyzerTestDetails{}, fmt.Errorf("PathAnalyzerTest resource is nil")
	}

	sourceEndpoint, err := pathAnalyzerTestSourceEndpointFromSpec(resource.Spec.SourceEndpoint)
	if err != nil {
		return vnmonitoringsdk.CreatePathAnalyzerTestDetails{}, err
	}
	destinationEndpoint, err := pathAnalyzerTestDestinationEndpointFromSpec(resource.Spec.DestinationEndpoint)
	if err != nil {
		return vnmonitoringsdk.CreatePathAnalyzerTestDetails{}, err
	}
	protocolParameters, err := pathAnalyzerTestProtocolParametersFromSpec(resource.Spec.ProtocolParameters)
	if err != nil {
		return vnmonitoringsdk.CreatePathAnalyzerTestDetails{}, err
	}
	definedTags, err := pathAnalyzerTestDefinedTagsFromSpec(resource.Spec.DefinedTags, false)
	if err != nil {
		return vnmonitoringsdk.CreatePathAnalyzerTestDetails{}, err
	}

	details := vnmonitoringsdk.CreatePathAnalyzerTestDetails{
		CompartmentId:       common.String(resource.Spec.CompartmentId),
		Protocol:            common.Int(resource.Spec.Protocol),
		SourceEndpoint:      sourceEndpoint,
		DestinationEndpoint: destinationEndpoint,
		QueryOptions: &vnmonitoringsdk.QueryOptions{
			IsBiDirectionalAnalysis: common.Bool(resource.Spec.QueryOptions.IsBiDirectionalAnalysis),
		},
	}
	if strings.TrimSpace(resource.Spec.DisplayName) != "" {
		details.DisplayName = common.String(resource.Spec.DisplayName)
	}
	if protocolParameters != nil {
		details.ProtocolParameters = protocolParameters
	}
	if len(resource.Spec.FreeformTags) != 0 {
		details.FreeformTags = maps.Clone(resource.Spec.FreeformTags)
	}
	if len(definedTags) != 0 {
		details.DefinedTags = definedTags
	}

	return details, nil
}

func buildPathAnalyzerTestUpdateBody(
	_ context.Context,
	resource *vnmonitoringv1beta1.PathAnalyzerTest,
	_ string,
	currentResponse any,
) (vnmonitoringsdk.UpdatePathAnalyzerTestDetails, bool, error) {
	if resource == nil {
		return vnmonitoringsdk.UpdatePathAnalyzerTestDetails{}, false, fmt.Errorf("PathAnalyzerTest resource is nil")
	}

	current, err := pathAnalyzerTestRuntimeBody(currentResponse)
	if err != nil {
		return vnmonitoringsdk.UpdatePathAnalyzerTestDetails{}, false, err
	}

	sourceEndpoint, err := pathAnalyzerTestSourceEndpointFromSpec(resource.Spec.SourceEndpoint)
	if err != nil {
		return vnmonitoringsdk.UpdatePathAnalyzerTestDetails{}, false, err
	}
	destinationEndpoint, err := pathAnalyzerTestDestinationEndpointFromSpec(resource.Spec.DestinationEndpoint)
	if err != nil {
		return vnmonitoringsdk.UpdatePathAnalyzerTestDetails{}, false, err
	}
	protocolParameters, err := pathAnalyzerTestProtocolParametersFromSpec(resource.Spec.ProtocolParameters)
	if err != nil {
		return vnmonitoringsdk.UpdatePathAnalyzerTestDetails{}, false, err
	}
	definedTags, err := pathAnalyzerTestDefinedTagsFromSpec(resource.Spec.DefinedTags, true)
	if err != nil {
		return vnmonitoringsdk.UpdatePathAnalyzerTestDetails{}, false, err
	}

	details := vnmonitoringsdk.UpdatePathAnalyzerTestDetails{
		DisplayName:         common.String(resource.Spec.DisplayName),
		Protocol:            common.Int(resource.Spec.Protocol),
		SourceEndpoint:      sourceEndpoint,
		DestinationEndpoint: destinationEndpoint,
		ProtocolParameters:  protocolParameters,
		QueryOptions: &vnmonitoringsdk.QueryOptions{
			IsBiDirectionalAnalysis: common.Bool(resource.Spec.QueryOptions.IsBiDirectionalAnalysis),
		},
		FreeformTags: cloneStringMapOrEmpty(resource.Spec.FreeformTags),
		DefinedTags:  definedTags,
	}

	desired := pathAnalyzerTestComparableFromSpec(resource.Spec, sourceEndpoint, destinationEndpoint, protocolParameters, definedTags)
	currentComparable, err := pathAnalyzerTestComparableFromCurrent(current)
	if err != nil {
		return vnmonitoringsdk.UpdatePathAnalyzerTestDetails{}, false, err
	}

	return details, !desired.equal(currentComparable), nil
}

func pathAnalyzerTestComparableFromSpec(
	spec vnmonitoringv1beta1.PathAnalyzerTestSpec,
	sourceEndpoint vnmonitoringsdk.Endpoint,
	destinationEndpoint vnmonitoringsdk.Endpoint,
	protocolParameters vnmonitoringsdk.ProtocolParameters,
	definedTags map[string]map[string]interface{},
) pathAnalyzerTestComparable {
	sourceEndpointJSON, _ := pathAnalyzerTestCanonicalJSONString(sourceEndpoint)
	destinationEndpointJSON, _ := pathAnalyzerTestCanonicalJSONString(destinationEndpoint)
	protocolParametersJSON, _ := pathAnalyzerTestCanonicalJSONString(protocolParameters)
	definedTagsJSON, _ := pathAnalyzerTestCanonicalJSONString(definedTags)

	return pathAnalyzerTestComparable{
		displayName:             spec.DisplayName,
		protocol:                spec.Protocol,
		sourceEndpoint:          sourceEndpointJSON,
		destinationEndpoint:     destinationEndpointJSON,
		protocolParameters:      protocolParametersJSON,
		isBiDirectionalAnalysis: spec.QueryOptions.IsBiDirectionalAnalysis,
		freeformTags:            maps.Clone(spec.FreeformTags),
		definedTags:             definedTagsJSON,
	}
}

func pathAnalyzerTestComparableFromCurrent(
	current vnmonitoringsdk.PathAnalyzerTest,
) (pathAnalyzerTestComparable, error) {
	sourceEndpointJSON, err := pathAnalyzerTestCanonicalJSONString(current.SourceEndpoint)
	if err != nil {
		return pathAnalyzerTestComparable{}, err
	}
	destinationEndpointJSON, err := pathAnalyzerTestCanonicalJSONString(current.DestinationEndpoint)
	if err != nil {
		return pathAnalyzerTestComparable{}, err
	}
	protocolParametersJSON, err := pathAnalyzerTestCanonicalJSONString(current.ProtocolParameters)
	if err != nil {
		return pathAnalyzerTestComparable{}, err
	}
	definedTagsJSON, err := pathAnalyzerTestCanonicalJSONString(current.DefinedTags)
	if err != nil {
		return pathAnalyzerTestComparable{}, err
	}

	return pathAnalyzerTestComparable{
		displayName:             stringValue(current.DisplayName),
		protocol:                intValue(current.Protocol),
		sourceEndpoint:          sourceEndpointJSON,
		destinationEndpoint:     destinationEndpointJSON,
		protocolParameters:      protocolParametersJSON,
		isBiDirectionalAnalysis: queryOptionsValue(current.QueryOptions),
		freeformTags:            maps.Clone(current.FreeformTags),
		definedTags:             definedTagsJSON,
	}, nil
}

func (c pathAnalyzerTestComparable) equal(other pathAnalyzerTestComparable) bool {
	return c.displayName == other.displayName &&
		c.protocol == other.protocol &&
		c.sourceEndpoint == other.sourceEndpoint &&
		c.destinationEndpoint == other.destinationEndpoint &&
		c.protocolParameters == other.protocolParameters &&
		c.isBiDirectionalAnalysis == other.isBiDirectionalAnalysis &&
		maps.Equal(c.freeformTags, other.freeformTags) &&
		c.definedTags == other.definedTags
}

func pathAnalyzerTestRuntimeBody(currentResponse any) (vnmonitoringsdk.PathAnalyzerTest, error) {
	switch current := currentResponse.(type) {
	case vnmonitoringsdk.PathAnalyzerTest:
		return current, nil
	case *vnmonitoringsdk.PathAnalyzerTest:
		if current == nil {
			return vnmonitoringsdk.PathAnalyzerTest{}, fmt.Errorf("current PathAnalyzerTest response is nil")
		}
		return *current, nil
	case vnmonitoringsdk.PathAnalyzerTestSummary:
		return vnmonitoringsdk.PathAnalyzerTest{
			Id:                  current.Id,
			DisplayName:         current.DisplayName,
			CompartmentId:       current.CompartmentId,
			Protocol:            current.Protocol,
			SourceEndpoint:      current.SourceEndpoint,
			DestinationEndpoint: current.DestinationEndpoint,
			QueryOptions:        current.QueryOptions,
			TimeCreated:         current.TimeCreated,
			TimeUpdated:         current.TimeUpdated,
			LifecycleState:      current.LifecycleState,
			ProtocolParameters:  current.ProtocolParameters,
			FreeformTags:        current.FreeformTags,
			DefinedTags:         current.DefinedTags,
			SystemTags:          current.SystemTags,
		}, nil
	case *vnmonitoringsdk.PathAnalyzerTestSummary:
		if current == nil {
			return vnmonitoringsdk.PathAnalyzerTest{}, fmt.Errorf("current PathAnalyzerTest response is nil")
		}
		return pathAnalyzerTestRuntimeBody(*current)
	case vnmonitoringsdk.CreatePathAnalyzerTestResponse:
		return current.PathAnalyzerTest, nil
	case *vnmonitoringsdk.CreatePathAnalyzerTestResponse:
		if current == nil {
			return vnmonitoringsdk.PathAnalyzerTest{}, fmt.Errorf("current PathAnalyzerTest response is nil")
		}
		return current.PathAnalyzerTest, nil
	case vnmonitoringsdk.GetPathAnalyzerTestResponse:
		return current.PathAnalyzerTest, nil
	case *vnmonitoringsdk.GetPathAnalyzerTestResponse:
		if current == nil {
			return vnmonitoringsdk.PathAnalyzerTest{}, fmt.Errorf("current PathAnalyzerTest response is nil")
		}
		return current.PathAnalyzerTest, nil
	case vnmonitoringsdk.UpdatePathAnalyzerTestResponse:
		return current.PathAnalyzerTest, nil
	case *vnmonitoringsdk.UpdatePathAnalyzerTestResponse:
		if current == nil {
			return vnmonitoringsdk.PathAnalyzerTest{}, fmt.Errorf("current PathAnalyzerTest response is nil")
		}
		return current.PathAnalyzerTest, nil
	default:
		return vnmonitoringsdk.PathAnalyzerTest{}, fmt.Errorf("unexpected current PathAnalyzerTest response type %T", currentResponse)
	}
}

func pathAnalyzerTestSourceEndpointFromSpec(
	spec vnmonitoringv1beta1.PathAnalyzerTestSourceEndpoint,
) (vnmonitoringsdk.Endpoint, error) {
	return pathAnalyzerTestEndpointFromSpecJSON(spec.JsonData, spec)
}

func pathAnalyzerTestDestinationEndpointFromSpec(
	spec vnmonitoringv1beta1.PathAnalyzerTestDestinationEndpoint,
) (vnmonitoringsdk.Endpoint, error) {
	return pathAnalyzerTestEndpointFromSpecJSON(spec.JsonData, spec)
}

func pathAnalyzerTestEndpointFromSpecJSON(rawJSON string, fallback any) (vnmonitoringsdk.Endpoint, error) {
	payload, err := pathAnalyzerTestPayload(rawJSON, fallback)
	if err != nil {
		return nil, err
	}
	if len(payload) == 0 {
		return nil, fmt.Errorf("PathAnalyzerTest endpoint payload is empty")
	}

	endpointType, err := pathAnalyzerTestJSONFieldString(payload, "type")
	if err != nil {
		return nil, fmt.Errorf("decode PathAnalyzerTest endpoint type: %w", err)
	}

	switch endpointType {
	case "COMPUTE_INSTANCE":
		var endpoint vnmonitoringsdk.ComputeInstanceEndpoint
		return endpoint, json.Unmarshal(payload, &endpoint)
	case "IP_ADDRESS":
		var endpoint vnmonitoringsdk.IpAddressEndpoint
		return endpoint, json.Unmarshal(payload, &endpoint)
	case "LOAD_BALANCER":
		var endpoint vnmonitoringsdk.LoadBalancerEndpoint
		return endpoint, json.Unmarshal(payload, &endpoint)
	case "LOAD_BALANCER_LISTENER":
		var endpoint vnmonitoringsdk.LoadBalancerListenerEndpoint
		return endpoint, json.Unmarshal(payload, &endpoint)
	case "NETWORK_LOAD_BALANCER":
		var endpoint vnmonitoringsdk.NetworkLoadBalancerEndpoint
		return endpoint, json.Unmarshal(payload, &endpoint)
	case "NETWORK_LOAD_BALANCER_LISTENER":
		var endpoint vnmonitoringsdk.NetworkLoadBalancerListenerEndpoint
		return endpoint, json.Unmarshal(payload, &endpoint)
	case "ON_PREM":
		var endpoint vnmonitoringsdk.OnPremEndpoint
		return endpoint, json.Unmarshal(payload, &endpoint)
	case "PRIVATE_SERVICE_ACCESS":
		var endpoint vnmonitoringsdk.PrivateServiceAccessEndpoint
		return endpoint, json.Unmarshal(payload, &endpoint)
	case "SUBNET":
		var endpoint vnmonitoringsdk.SubnetEndpoint
		return endpoint, json.Unmarshal(payload, &endpoint)
	case "VLAN":
		var endpoint vnmonitoringsdk.VlanEndpoint
		return endpoint, json.Unmarshal(payload, &endpoint)
	case "VNIC":
		var endpoint vnmonitoringsdk.VnicEndpoint
		return endpoint, json.Unmarshal(payload, &endpoint)
	default:
		return nil, fmt.Errorf("unsupported PathAnalyzerTest endpoint type %q", endpointType)
	}
}

func pathAnalyzerTestProtocolParametersFromSpec(
	spec vnmonitoringv1beta1.PathAnalyzerTestProtocolParameters,
) (vnmonitoringsdk.ProtocolParameters, error) {
	payload, err := pathAnalyzerTestPayload(spec.JsonData, spec)
	if err != nil {
		return nil, err
	}
	if len(payload) == 0 {
		return nil, nil
	}

	protocolType, err := pathAnalyzerTestJSONFieldString(payload, "type")
	if err != nil {
		return nil, fmt.Errorf("decode PathAnalyzerTest protocolParameters type: %w", err)
	}

	switch protocolType {
	case "TCP":
		var parameters vnmonitoringsdk.TcpProtocolParameters
		return parameters, json.Unmarshal(payload, &parameters)
	case "UDP":
		var parameters vnmonitoringsdk.UdpProtocolParameters
		return parameters, json.Unmarshal(payload, &parameters)
	case "ICMP":
		var parameters vnmonitoringsdk.IcmpProtocolParameters
		return parameters, json.Unmarshal(payload, &parameters)
	default:
		return nil, fmt.Errorf("unsupported PathAnalyzerTest protocolParameters type %q", protocolType)
	}
}

func pathAnalyzerTestPayload(rawJSON string, fallback any) ([]byte, error) {
	if raw := strings.TrimSpace(rawJSON); raw != "" {
		return []byte(raw), nil
	}

	payload, err := json.Marshal(fallback)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(payload)) == "{}" {
		return nil, nil
	}
	return payload, nil
}

func pathAnalyzerTestJSONFieldString(payload []byte, key string) (string, error) {
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return "", err
	}
	raw, ok := decoded[key]
	if !ok || raw == nil {
		return "", fmt.Errorf("missing %q", key)
	}
	value, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("%q is %T, want string", key, raw)
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("%q is empty", key)
	}
	return trimmed, nil
}

func pathAnalyzerTestDefinedTagsFromSpec(
	spec map[string]shared.MapValue,
	emptyToClear bool,
) (map[string]map[string]interface{}, error) {
	if len(spec) == 0 {
		if emptyToClear {
			return map[string]map[string]interface{}{}, nil
		}
		return nil, nil
	}

	payload, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("marshal PathAnalyzerTest definedTags: %w", err)
	}
	var tags map[string]map[string]interface{}
	if err := json.Unmarshal(payload, &tags); err != nil {
		return nil, fmt.Errorf("decode PathAnalyzerTest definedTags: %w", err)
	}
	return tags, nil
}

func cloneStringMapOrEmpty(values map[string]string) map[string]string {
	if len(values) == 0 {
		return map[string]string{}
	}
	return maps.Clone(values)
}

func pathAnalyzerTestCanonicalJSONString(value any) (string, error) {
	if value == nil {
		return "", nil
	}

	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	if len(payload) == 0 || string(payload) == "null" || string(payload) == "{}" || string(payload) == "[]" {
		return "", nil
	}

	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return "", err
	}
	normalized, err := json.Marshal(decoded)
	if err != nil {
		return "", err
	}
	if string(normalized) == "null" || string(normalized) == "{}" || string(normalized) == "[]" {
		return "", nil
	}
	return string(normalized), nil
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func intValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func queryOptionsValue(options *vnmonitoringsdk.QueryOptions) bool {
	if options == nil || options.IsBiDirectionalAnalysis == nil {
		return false
	}
	return *options.IsBiDirectionalAnalysis
}

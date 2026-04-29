/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	containerenginesdk "github.com/oracle/oci-go-sdk/v65/containerengine"
	containerenginev1beta1 "github.com/oracle/oci-service-operator/api/containerengine/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

func init() {
	registerClusterRuntimeHooksMutator(func(manager *ClusterServiceManager, hooks *ClusterRuntimeHooks) {
		applyClusterRuntimeHooks(manager, hooks)
	})
}

func applyClusterRuntimeHooks(manager *ClusterServiceManager, hooks *ClusterRuntimeHooks) {
	if hooks == nil {
		return
	}

	var credentialClient credhelper.CredentialClient
	if manager != nil {
		credentialClient = manager.CredentialClient
	}

	hooks.Semantics = reviewedClusterRuntimeSemantics()
	hooks.BuildCreateBody = func(_ context.Context, resource *containerenginev1beta1.Cluster, _ string) (any, error) {
		return buildClusterCreateDetails(resource)
	}
	hooks.Get.Fields = reviewedClusterGetFields()
	if hooks.Get.Call != nil {
		originalGet := hooks.Get.Call
		hooks.Get.Call = func(ctx context.Context, request containerenginesdk.GetClusterRequest) (containerenginesdk.GetClusterResponse, error) {
			request.ShouldIncludeOidcConfigFile = common.Bool(true)
			return originalGet(ctx, request)
		}
	}
	hooks.List.Fields = reviewedClusterListFields()
	hooks.BuildUpdateBody = func(
		ctx context.Context,
		resource *containerenginev1beta1.Cluster,
		namespace string,
		currentResponse any,
	) (any, bool, error) {
		return buildClusterUpdateBody(ctx, credentialClient, resource, namespace, currentResponse)
	}
}

func reviewedClusterRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newClusterRuntimeSemantics()
	semantics.Mutation = generatedruntime.MutationSemantics{
		Mutable:       reviewedClusterMutableFields(),
		ForceNew:      reviewedClusterForceNewFields(),
		ConflictsWith: map[string][]string{},
	}
	return semantics
}

func reviewedClusterMutableFields() []string {
	return []string{
		"definedTags",
		"freeformTags",
		"imagePolicyConfig.isPolicyEnabled",
		"imagePolicyConfig.keyDetails.kmsKeyId",
		"kubernetesVersion",
		"name",
		"options.admissionControllerOptions.isPodSecurityPolicyEnabled",
		"options.openIdConnectDiscovery.isOpenIdConnectDiscoveryEnabled",
		"options.openIdConnectTokenAuthenticationConfig.caCertificate",
		"options.openIdConnectTokenAuthenticationConfig.clientId",
		"options.openIdConnectTokenAuthenticationConfig.configurationFile",
		"options.openIdConnectTokenAuthenticationConfig.groupsClaim",
		"options.openIdConnectTokenAuthenticationConfig.groupsPrefix",
		"options.openIdConnectTokenAuthenticationConfig.isOpenIdConnectAuthEnabled",
		"options.openIdConnectTokenAuthenticationConfig.issuerUrl",
		"options.openIdConnectTokenAuthenticationConfig.requiredClaims.key",
		"options.openIdConnectTokenAuthenticationConfig.requiredClaims.value",
		"options.openIdConnectTokenAuthenticationConfig.signingAlgorithms",
		"options.openIdConnectTokenAuthenticationConfig.usernameClaim",
		"options.openIdConnectTokenAuthenticationConfig.usernamePrefix",
		"options.persistentVolumeConfig.definedTags",
		"options.persistentVolumeConfig.freeformTags",
		"options.serviceLbConfig.backendNsgIds",
		"options.serviceLbConfig.definedTags",
		"options.serviceLbConfig.freeformTags",
		"type",
	}
}

func reviewedClusterForceNewFields() []string {
	return []string{
		"clusterPodNetworkOptions.cniType",
		"clusterPodNetworkOptions.jsonData",
		"compartmentId",
		"endpointConfig.isPublicIpEnabled",
		"endpointConfig.nsgIds",
		"endpointConfig.subnetId",
		"kmsKeyId",
		"options.addOns.isKubernetesDashboardEnabled",
		"options.addOns.isTillerEnabled",
		"options.kubernetesNetworkConfig.podsCidr",
		"options.kubernetesNetworkConfig.servicesCidr",
		"options.serviceLbSubnetIds",
		"vcnId",
	}
}

func reviewedClusterGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ClusterId", RequestName: "clusterId", Contribution: "path", PreferResourceID: true},
		{FieldName: "ShouldIncludeOidcConfigFile", RequestName: "shouldIncludeOidcConfigFile", Contribution: "query"},
	}
}

func reviewedClusterListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
		{FieldName: "Name", RequestName: "name", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func buildClusterCreateDetails(resource *containerenginev1beta1.Cluster) (containerenginesdk.CreateClusterDetails, error) {
	if resource == nil {
		return containerenginesdk.CreateClusterDetails{}, fmt.Errorf("cluster resource is nil")
	}

	spec := resource.Spec
	details := containerenginesdk.CreateClusterDetails{
		Name:              common.String(spec.Name),
		CompartmentId:     common.String(spec.CompartmentId),
		VcnId:             common.String(spec.VcnId),
		KubernetesVersion: common.String(spec.KubernetesVersion),
	}

	if endpointConfig := buildClusterEndpointConfigDetails(spec.EndpointConfig); endpointConfig != nil {
		details.EndpointConfig = endpointConfig
	}
	if spec.KmsKeyId != "" {
		details.KmsKeyId = common.String(spec.KmsKeyId)
	}
	if len(spec.FreeformTags) > 0 {
		details.FreeformTags = copyClusterStringMap(spec.FreeformTags)
	}
	definedTags, err := convertClusterDefinedTags(spec.DefinedTags)
	if err != nil {
		return containerenginesdk.CreateClusterDetails{}, fmt.Errorf("convert cluster definedTags: %w", err)
	}
	if len(definedTags) > 0 {
		details.DefinedTags = definedTags
	}
	options, err := buildClusterCreateOptions(spec.Options)
	if err != nil {
		return containerenginesdk.CreateClusterDetails{}, err
	}
	if options != nil {
		details.Options = options
	}
	if imagePolicyConfig := buildClusterImagePolicyConfig(spec.ImagePolicyConfig); imagePolicyConfig != nil {
		details.ImagePolicyConfig = imagePolicyConfig
	}
	clusterPodNetworkOptions, err := buildClusterPodNetworkOptions(spec.ClusterPodNetworkOptions)
	if err != nil {
		return containerenginesdk.CreateClusterDetails{}, err
	}
	if len(clusterPodNetworkOptions) > 0 {
		details.ClusterPodNetworkOptions = clusterPodNetworkOptions
	}
	if spec.Type != "" {
		details.Type = containerenginesdk.ClusterTypeEnum(spec.Type)
	}

	return details, nil
}

func buildClusterUpdateBody(
	ctx context.Context,
	credentialClient credhelper.CredentialClient,
	resource *containerenginev1beta1.Cluster,
	namespace string,
	currentResponse any,
) (containerenginesdk.UpdateClusterDetails, bool, error) {
	resolvedValues, details, err := buildClusterResolvedUpdateDetails(ctx, credentialClient, resource, namespace)
	if err != nil {
		return containerenginesdk.UpdateClusterDetails{}, false, err
	}

	currentDetails, err := buildCurrentClusterUpdateDetails(currentResponse)
	if err != nil {
		return containerenginesdk.UpdateClusterDetails{}, false, err
	}
	explicitClearDrift := applyClusterExplicitMutableClears(resolvedValues, resource.Spec, currentDetails, &details)

	desiredValues, err := clusterJSONMap(details)
	if err != nil {
		return containerenginesdk.UpdateClusterDetails{}, false, fmt.Errorf("project desired cluster update body: %w", err)
	}
	if len(desiredValues) == 0 && !explicitClearDrift {
		return details, false, nil
	}
	currentValues, err := clusterJSONMap(currentDetails)
	if err != nil {
		return containerenginesdk.UpdateClusterDetails{}, false, fmt.Errorf("project current cluster update body: %w", err)
	}

	return details, explicitClearDrift || !clusterMapSubsetEqual(desiredValues, currentValues), nil
}

func buildClusterUpdateDetails(
	ctx context.Context,
	credentialClient credhelper.CredentialClient,
	resource *containerenginev1beta1.Cluster,
	namespace string,
) (containerenginesdk.UpdateClusterDetails, error) {
	_, details, err := buildClusterResolvedUpdateDetails(ctx, credentialClient, resource, namespace)
	return details, err
}

func buildClusterResolvedUpdateDetails(
	ctx context.Context,
	credentialClient credhelper.CredentialClient,
	resource *containerenginev1beta1.Cluster,
	namespace string,
) (map[string]any, containerenginesdk.UpdateClusterDetails, error) {
	if resource == nil {
		return nil, containerenginesdk.UpdateClusterDetails{}, fmt.Errorf("cluster resource is nil")
	}

	resolvedSpec, err := generatedruntime.ResolveSpecValueWithBoolFields(resource, ctx, credentialClient, namespace)
	if err != nil {
		return nil, containerenginesdk.UpdateClusterDetails{}, err
	}

	payload, err := json.Marshal(resolvedSpec)
	if err != nil {
		return nil, containerenginesdk.UpdateClusterDetails{}, fmt.Errorf("marshal resolved cluster spec: %w", err)
	}

	var details containerenginesdk.UpdateClusterDetails
	if err := json.Unmarshal(payload, &details); err != nil {
		return nil, containerenginesdk.UpdateClusterDetails{}, fmt.Errorf("decode cluster update request body: %w", err)
	}
	resolvedValues, _ := resolvedSpec.(map[string]any)
	applyClusterExplicitEmptySliceValues(resource.Spec, &details)

	return resolvedValues, details, nil
}

func buildCurrentClusterUpdateDetails(currentResponse any) (containerenginesdk.UpdateClusterDetails, error) {
	body, err := clusterRuntimeResponseBody(currentResponse)
	if err != nil {
		return containerenginesdk.UpdateClusterDetails{}, err
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return containerenginesdk.UpdateClusterDetails{}, fmt.Errorf("marshal current cluster response: %w", err)
	}

	var details containerenginesdk.UpdateClusterDetails
	if err := json.Unmarshal(payload, &details); err != nil {
		return containerenginesdk.UpdateClusterDetails{}, fmt.Errorf("decode current cluster update body: %w", err)
	}

	return details, nil
}

func clusterRuntimeResponseBody(currentResponse any) (any, error) {
	switch current := currentResponse.(type) {
	case containerenginesdk.Cluster:
		return current, nil
	case *containerenginesdk.Cluster:
		if current == nil {
			return nil, fmt.Errorf("current Cluster response is nil")
		}
		return *current, nil
	case containerenginesdk.ClusterSummary:
		return current, nil
	case *containerenginesdk.ClusterSummary:
		if current == nil {
			return nil, fmt.Errorf("current Cluster response is nil")
		}
		return *current, nil
	case containerenginesdk.GetClusterResponse:
		return current.Cluster, nil
	case *containerenginesdk.GetClusterResponse:
		if current == nil {
			return nil, fmt.Errorf("current Cluster response is nil")
		}
		return current.Cluster, nil
	default:
		return nil, fmt.Errorf("unexpected current Cluster response type %T", currentResponse)
	}
}

func buildClusterEndpointConfigDetails(spec containerenginev1beta1.ClusterEndpointConfig) *containerenginesdk.CreateClusterEndpointConfigDetails {
	if spec.SubnetId == "" && len(spec.NsgIds) == 0 && !spec.IsPublicIpEnabled {
		return nil
	}

	details := &containerenginesdk.CreateClusterEndpointConfigDetails{
		IsPublicIpEnabled: common.Bool(spec.IsPublicIpEnabled),
	}
	if spec.SubnetId != "" {
		details.SubnetId = common.String(spec.SubnetId)
	}
	if len(spec.NsgIds) > 0 {
		details.NsgIds = append([]string(nil), spec.NsgIds...)
	}
	return details
}

func buildClusterCreateOptions(spec containerenginev1beta1.ClusterOptions) (*containerenginesdk.ClusterCreateOptions, error) {
	kubernetesNetworkConfig := buildClusterKubernetesNetworkConfig(spec.KubernetesNetworkConfig)
	addOns := buildClusterAddOnOptions(spec.AddOns)
	admissionControllerOptions := buildClusterAdmissionControllerOptions(spec.AdmissionControllerOptions)
	persistentVolumeConfig, err := buildClusterPersistentVolumeConfigDetails(spec.PersistentVolumeConfig)
	if err != nil {
		return nil, fmt.Errorf("build cluster persistentVolumeConfig: %w", err)
	}
	serviceLBConfig, err := buildClusterServiceLBConfigDetails(spec.ServiceLbConfig)
	if err != nil {
		return nil, fmt.Errorf("build cluster serviceLbConfig: %w", err)
	}
	openIDConnectTokenAuth := buildClusterOpenIDConnectTokenAuthenticationConfig(spec.OpenIdConnectTokenAuthenticationConfig)
	openIDConnectDiscovery := buildClusterOpenIDConnectDiscovery(spec.OpenIdConnectDiscovery)

	if len(spec.ServiceLbSubnetIds) == 0 &&
		kubernetesNetworkConfig == nil &&
		addOns == nil &&
		admissionControllerOptions == nil &&
		persistentVolumeConfig == nil &&
		serviceLBConfig == nil &&
		openIDConnectTokenAuth == nil &&
		openIDConnectDiscovery == nil {
		return nil, nil
	}

	details := &containerenginesdk.ClusterCreateOptions{
		KubernetesNetworkConfig:                kubernetesNetworkConfig,
		AddOns:                                 addOns,
		AdmissionControllerOptions:             admissionControllerOptions,
		PersistentVolumeConfig:                 persistentVolumeConfig,
		ServiceLbConfig:                        serviceLBConfig,
		OpenIdConnectTokenAuthenticationConfig: openIDConnectTokenAuth,
		OpenIdConnectDiscovery:                 openIDConnectDiscovery,
	}
	if len(spec.ServiceLbSubnetIds) > 0 {
		details.ServiceLbSubnetIds = append([]string(nil), spec.ServiceLbSubnetIds...)
	}
	return details, nil
}

func buildClusterKubernetesNetworkConfig(
	spec containerenginev1beta1.ClusterOptionsKubernetesNetworkConfig,
) *containerenginesdk.KubernetesNetworkConfig {
	if spec.PodsCidr == "" && spec.ServicesCidr == "" {
		return nil
	}

	details := &containerenginesdk.KubernetesNetworkConfig{}
	if spec.PodsCidr != "" {
		details.PodsCidr = common.String(spec.PodsCidr)
	}
	if spec.ServicesCidr != "" {
		details.ServicesCidr = common.String(spec.ServicesCidr)
	}
	return details
}

func buildClusterAddOnOptions(spec containerenginev1beta1.ClusterOptionsAddOns) *containerenginesdk.AddOnOptions {
	if !spec.IsKubernetesDashboardEnabled && !spec.IsTillerEnabled {
		return nil
	}

	return &containerenginesdk.AddOnOptions{
		IsKubernetesDashboardEnabled: common.Bool(spec.IsKubernetesDashboardEnabled),
		IsTillerEnabled:              common.Bool(spec.IsTillerEnabled),
	}
}

func buildClusterAdmissionControllerOptions(
	spec containerenginev1beta1.ClusterOptionsAdmissionControllerOptions,
) *containerenginesdk.AdmissionControllerOptions {
	if !spec.IsPodSecurityPolicyEnabled {
		return nil
	}

	return &containerenginesdk.AdmissionControllerOptions{
		IsPodSecurityPolicyEnabled: common.Bool(spec.IsPodSecurityPolicyEnabled),
	}
}

func buildClusterPersistentVolumeConfigDetails(
	spec containerenginev1beta1.ClusterOptionsPersistentVolumeConfig,
) (*containerenginesdk.PersistentVolumeConfigDetails, error) {
	if len(spec.FreeformTags) == 0 && len(spec.DefinedTags) == 0 {
		return nil, nil
	}

	definedTags, err := convertClusterDefinedTags(spec.DefinedTags)
	if err != nil {
		return nil, err
	}

	return &containerenginesdk.PersistentVolumeConfigDetails{
		FreeformTags: copyClusterStringMap(spec.FreeformTags),
		DefinedTags:  definedTags,
	}, nil
}

func buildClusterServiceLBConfigDetails(
	spec containerenginev1beta1.ClusterOptionsServiceLbConfig,
) (*containerenginesdk.ServiceLbConfigDetails, error) {
	if len(spec.FreeformTags) == 0 && len(spec.DefinedTags) == 0 && len(spec.BackendNsgIds) == 0 {
		return nil, nil
	}

	definedTags, err := convertClusterDefinedTags(spec.DefinedTags)
	if err != nil {
		return nil, err
	}

	return &containerenginesdk.ServiceLbConfigDetails{
		FreeformTags:  copyClusterStringMap(spec.FreeformTags),
		DefinedTags:   definedTags,
		BackendNsgIds: append([]string(nil), spec.BackendNsgIds...),
	}, nil
}

func buildClusterOpenIDConnectTokenAuthenticationConfig(
	spec containerenginev1beta1.ClusterOptionsOpenIdConnectTokenAuthenticationConfig,
) *containerenginesdk.OpenIdConnectTokenAuthenticationConfig {
	if !spec.IsOpenIdConnectAuthEnabled &&
		spec.IssuerUrl == "" &&
		spec.ClientId == "" &&
		spec.UsernameClaim == "" &&
		spec.UsernamePrefix == "" &&
		spec.GroupsClaim == "" &&
		spec.GroupsPrefix == "" &&
		len(spec.RequiredClaims) == 0 &&
		spec.CaCertificate == "" &&
		len(spec.SigningAlgorithms) == 0 &&
		spec.ConfigurationFile == "" {
		return nil
	}

	details := &containerenginesdk.OpenIdConnectTokenAuthenticationConfig{
		IsOpenIdConnectAuthEnabled: common.Bool(spec.IsOpenIdConnectAuthEnabled),
	}
	if spec.IssuerUrl != "" {
		details.IssuerUrl = common.String(spec.IssuerUrl)
	}
	if spec.ClientId != "" {
		details.ClientId = common.String(spec.ClientId)
	}
	if spec.UsernameClaim != "" {
		details.UsernameClaim = common.String(spec.UsernameClaim)
	}
	if spec.UsernamePrefix != "" {
		details.UsernamePrefix = common.String(spec.UsernamePrefix)
	}
	if spec.GroupsClaim != "" {
		details.GroupsClaim = common.String(spec.GroupsClaim)
	}
	if spec.GroupsPrefix != "" {
		details.GroupsPrefix = common.String(spec.GroupsPrefix)
	}
	requiredClaims := buildClusterOIDCRequiredClaims(spec.RequiredClaims)
	if len(requiredClaims) > 0 {
		details.RequiredClaims = requiredClaims
	}
	if spec.CaCertificate != "" {
		details.CaCertificate = common.String(spec.CaCertificate)
	}
	if len(spec.SigningAlgorithms) > 0 {
		details.SigningAlgorithms = append([]string(nil), spec.SigningAlgorithms...)
	}
	if spec.ConfigurationFile != "" {
		details.ConfigurationFile = common.String(spec.ConfigurationFile)
	}
	return details
}

func buildClusterOIDCRequiredClaims(
	spec []containerenginev1beta1.ClusterOptionsOpenIdConnectTokenAuthenticationConfigRequiredClaim,
) []containerenginesdk.KeyValue {
	if len(spec) == 0 {
		return nil
	}

	claims := make([]containerenginesdk.KeyValue, 0, len(spec))
	for _, claim := range spec {
		if claim.Key == "" && claim.Value == "" {
			continue
		}
		claims = append(claims, containerenginesdk.KeyValue{
			Key:   common.String(claim.Key),
			Value: common.String(claim.Value),
		})
	}
	if len(claims) == 0 {
		return nil
	}
	return claims
}

func buildClusterOpenIDConnectDiscovery(
	spec containerenginev1beta1.ClusterOptionsOpenIdConnectDiscovery,
) *containerenginesdk.OpenIdConnectDiscovery {
	if !spec.IsOpenIdConnectDiscoveryEnabled {
		return nil
	}

	return &containerenginesdk.OpenIdConnectDiscovery{
		IsOpenIdConnectDiscoveryEnabled: common.Bool(spec.IsOpenIdConnectDiscoveryEnabled),
	}
}

func applyClusterExplicitEmptySliceValues(
	spec containerenginev1beta1.ClusterSpec,
	details *containerenginesdk.UpdateClusterDetails,
) {
	if details == nil {
		return
	}

	if spec.Options.ServiceLbConfig.BackendNsgIds != nil && len(spec.Options.ServiceLbConfig.BackendNsgIds) == 0 {
		ensureClusterUpdateServiceLBConfig(details).BackendNsgIds = []string{}
	}

	if spec.Options.OpenIdConnectTokenAuthenticationConfig.RequiredClaims != nil &&
		len(spec.Options.OpenIdConnectTokenAuthenticationConfig.RequiredClaims) == 0 {
		ensureClusterUpdateOIDCTokenAuthenticationConfig(details).RequiredClaims = []containerenginesdk.KeyValue{}
	}

	if spec.Options.OpenIdConnectTokenAuthenticationConfig.SigningAlgorithms != nil &&
		len(spec.Options.OpenIdConnectTokenAuthenticationConfig.SigningAlgorithms) == 0 {
		ensureClusterUpdateOIDCTokenAuthenticationConfig(details).SigningAlgorithms = []string{}
	}
}

func applyClusterExplicitMutableClears(
	resolvedValues map[string]any,
	spec containerenginev1beta1.ClusterSpec,
	current containerenginesdk.UpdateClusterDetails,
	details *containerenginesdk.UpdateClusterDetails,
) bool {
	if details == nil {
		return false
	}

	updateNeeded := false

	if spec.Options.ServiceLbConfig.BackendNsgIds != nil &&
		len(spec.Options.ServiceLbConfig.BackendNsgIds) == 0 &&
		len(clusterCurrentServiceLBBackendNsgIDs(current)) != 0 {
		ensureClusterUpdateServiceLBConfig(details).BackendNsgIds = []string{}
		updateNeeded = true
	}

	if spec.Options.OpenIdConnectTokenAuthenticationConfig.RequiredClaims != nil &&
		len(spec.Options.OpenIdConnectTokenAuthenticationConfig.RequiredClaims) == 0 &&
		len(clusterCurrentOIDCRequiredClaims(current)) != 0 {
		ensureClusterUpdateOIDCTokenAuthenticationConfig(details).RequiredClaims = []containerenginesdk.KeyValue{}
		updateNeeded = true
	}

	if spec.Options.OpenIdConnectTokenAuthenticationConfig.SigningAlgorithms != nil &&
		len(spec.Options.OpenIdConnectTokenAuthenticationConfig.SigningAlgorithms) == 0 &&
		len(clusterCurrentOIDCSigningAlgorithms(current)) != 0 {
		ensureClusterUpdateOIDCTokenAuthenticationConfig(details).SigningAlgorithms = []string{}
		updateNeeded = true
	}

	if !clusterResolvedSpecHasPath(resolvedValues, "options.openIdConnectTokenAuthenticationConfig") {
		return updateNeeded
	}

	currentOIDC := clusterCurrentOIDCTokenAuthenticationConfig(current)
	if currentOIDC == nil {
		return updateNeeded
	}

	desiredOIDC := spec.Options.OpenIdConnectTokenAuthenticationConfig
	typedOIDC := ensureClusterUpdateOIDCTokenAuthenticationConfig(details)

	if desiredOIDC.IssuerUrl == "" && clusterStringPtrHasValue(currentOIDC.IssuerUrl) {
		typedOIDC.IssuerUrl = common.String("")
		updateNeeded = true
	}
	if desiredOIDC.ClientId == "" && clusterStringPtrHasValue(currentOIDC.ClientId) {
		typedOIDC.ClientId = common.String("")
		updateNeeded = true
	}
	if desiredOIDC.UsernameClaim == "" && clusterStringPtrHasValue(currentOIDC.UsernameClaim) {
		typedOIDC.UsernameClaim = common.String("")
		updateNeeded = true
	}
	if desiredOIDC.UsernamePrefix == "" && clusterStringPtrHasValue(currentOIDC.UsernamePrefix) {
		typedOIDC.UsernamePrefix = common.String("")
		updateNeeded = true
	}
	if desiredOIDC.GroupsClaim == "" && clusterStringPtrHasValue(currentOIDC.GroupsClaim) {
		typedOIDC.GroupsClaim = common.String("")
		updateNeeded = true
	}
	if desiredOIDC.GroupsPrefix == "" && clusterStringPtrHasValue(currentOIDC.GroupsPrefix) {
		typedOIDC.GroupsPrefix = common.String("")
		updateNeeded = true
	}
	if desiredOIDC.CaCertificate == "" && clusterStringPtrHasValue(currentOIDC.CaCertificate) {
		typedOIDC.CaCertificate = common.String("")
		updateNeeded = true
	}
	if desiredOIDC.ConfigurationFile == "" && clusterStringPtrHasValue(currentOIDC.ConfigurationFile) {
		typedOIDC.ConfigurationFile = common.String("")
		updateNeeded = true
	}

	return updateNeeded
}

func ensureClusterUpdateOptions(details *containerenginesdk.UpdateClusterDetails) *containerenginesdk.UpdateClusterOptionsDetails {
	if details.Options == nil {
		details.Options = &containerenginesdk.UpdateClusterOptionsDetails{}
	}
	return details.Options
}

func ensureClusterUpdateServiceLBConfig(details *containerenginesdk.UpdateClusterDetails) *containerenginesdk.ServiceLbConfigDetails {
	options := ensureClusterUpdateOptions(details)
	if options.ServiceLbConfig == nil {
		options.ServiceLbConfig = &containerenginesdk.ServiceLbConfigDetails{}
	}
	return options.ServiceLbConfig
}

func ensureClusterUpdateOIDCTokenAuthenticationConfig(
	details *containerenginesdk.UpdateClusterDetails,
) *containerenginesdk.OpenIdConnectTokenAuthenticationConfig {
	options := ensureClusterUpdateOptions(details)
	if options.OpenIdConnectTokenAuthenticationConfig == nil {
		options.OpenIdConnectTokenAuthenticationConfig = &containerenginesdk.OpenIdConnectTokenAuthenticationConfig{}
	}
	return options.OpenIdConnectTokenAuthenticationConfig
}

func clusterCurrentServiceLBBackendNsgIDs(current containerenginesdk.UpdateClusterDetails) []string {
	if current.Options == nil || current.Options.ServiceLbConfig == nil {
		return nil
	}
	return current.Options.ServiceLbConfig.BackendNsgIds
}

func clusterCurrentOIDCTokenAuthenticationConfig(
	current containerenginesdk.UpdateClusterDetails,
) *containerenginesdk.OpenIdConnectTokenAuthenticationConfig {
	if current.Options == nil {
		return nil
	}
	return current.Options.OpenIdConnectTokenAuthenticationConfig
}

func clusterCurrentOIDCRequiredClaims(current containerenginesdk.UpdateClusterDetails) []containerenginesdk.KeyValue {
	currentOIDC := clusterCurrentOIDCTokenAuthenticationConfig(current)
	if currentOIDC == nil {
		return nil
	}
	return currentOIDC.RequiredClaims
}

func clusterCurrentOIDCSigningAlgorithms(current containerenginesdk.UpdateClusterDetails) []string {
	currentOIDC := clusterCurrentOIDCTokenAuthenticationConfig(current)
	if currentOIDC == nil {
		return nil
	}
	return currentOIDC.SigningAlgorithms
}

func clusterStringPtrHasValue(value *string) bool {
	return value != nil && *value != ""
}

func clusterResolvedSpecHasPath(values map[string]any, path string) bool {
	path = strings.TrimSpace(path)
	if len(values) == 0 || path == "" {
		return false
	}

	current := any(values)
	for _, segment := range strings.Split(path, ".") {
		next, ok := current.(map[string]any)
		if !ok {
			return false
		}
		current, ok = next[strings.TrimSpace(segment)]
		if !ok {
			return false
		}
	}
	return true
}

func buildClusterImagePolicyConfig(
	spec containerenginev1beta1.ClusterImagePolicyConfig,
) *containerenginesdk.CreateImagePolicyConfigDetails {
	if !spec.IsPolicyEnabled && len(spec.KeyDetails) == 0 {
		return nil
	}

	details := &containerenginesdk.CreateImagePolicyConfigDetails{
		IsPolicyEnabled: common.Bool(spec.IsPolicyEnabled),
	}
	if len(spec.KeyDetails) > 0 {
		keyDetails := make([]containerenginesdk.KeyDetails, 0, len(spec.KeyDetails))
		for _, keyDetail := range spec.KeyDetails {
			if keyDetail.KmsKeyId == "" {
				continue
			}
			keyDetails = append(keyDetails, containerenginesdk.KeyDetails{
				KmsKeyId: common.String(keyDetail.KmsKeyId),
			})
		}
		if len(keyDetails) > 0 {
			details.KeyDetails = keyDetails
		}
	}
	return details
}

func buildClusterPodNetworkOptions(
	spec []containerenginev1beta1.ClusterPodNetworkOption,
) ([]containerenginesdk.ClusterPodNetworkOptionDetails, error) {
	if len(spec) == 0 {
		return nil, nil
	}

	details := make([]containerenginesdk.ClusterPodNetworkOptionDetails, 0, len(spec))
	for index, option := range spec {
		detail, err := buildClusterPodNetworkOption(option)
		if err != nil {
			return nil, fmt.Errorf("build clusterPodNetworkOptions[%d]: %w", index, err)
		}
		if detail != nil {
			details = append(details, detail)
		}
	}

	if len(details) == 0 {
		return nil, nil
	}
	return details, nil
}

func buildClusterPodNetworkOption(
	spec containerenginev1beta1.ClusterPodNetworkOption,
) (containerenginesdk.ClusterPodNetworkOptionDetails, error) {
	rawJSON := strings.TrimSpace(spec.JsonData)
	cniType := strings.TrimSpace(spec.CniType)
	if cniType == "" && rawJSON != "" {
		var discriminator struct {
			CniType string `json:"cniType"`
		}
		if err := json.Unmarshal([]byte(rawJSON), &discriminator); err != nil {
			return nil, fmt.Errorf("parse clusterPodNetworkOptions discriminator: %w", err)
		}
		cniType = strings.TrimSpace(discriminator.CniType)
	}

	switch strings.ToUpper(cniType) {
	case "":
		if rawJSON == "" {
			return nil, nil
		}
		return nil, fmt.Errorf("missing cniType")
	case "FLANNEL_OVERLAY":
		var detail containerenginesdk.FlannelOverlayClusterPodNetworkOptionDetails
		if rawJSON != "" {
			if err := json.Unmarshal([]byte(rawJSON), &detail); err != nil {
				return nil, fmt.Errorf("decode FLANNEL_OVERLAY jsonData: %w", err)
			}
		}
		return detail, nil
	case "OCI_VCN_IP_NATIVE":
		var detail containerenginesdk.OciVcnIpNativeClusterPodNetworkOptionDetails
		if rawJSON != "" {
			if err := json.Unmarshal([]byte(rawJSON), &detail); err != nil {
				return nil, fmt.Errorf("decode OCI_VCN_IP_NATIVE jsonData: %w", err)
			}
		}
		return detail, nil
	default:
		return nil, fmt.Errorf("unsupported cniType %q", cniType)
	}
}

func convertClusterDefinedTags(tags map[string]shared.MapValue) (map[string]map[string]interface{}, error) {
	if len(tags) == 0 {
		return nil, nil
	}

	payload, err := json.Marshal(tags)
	if err != nil {
		return nil, err
	}

	var converted map[string]map[string]interface{}
	if err := json.Unmarshal(payload, &converted); err != nil {
		return nil, err
	}
	return converted, nil
}

func copyClusterStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}

	copied := make(map[string]string, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return copied
}

func clusterJSONMap(value any) (map[string]any, error) {
	if value == nil {
		return nil, nil
	}

	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, err
	}

	pruned, ok := pruneClusterJSONValue(decoded)
	if !ok {
		return nil, nil
	}

	values, ok := pruned.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("cluster JSON projection is %T, want map[string]any", pruned)
	}
	return values, nil
}

func pruneClusterJSONValue(value any) (any, bool) {
	switch concrete := value.(type) {
	case nil:
		return nil, false
	case map[string]any:
		pruned := make(map[string]any, len(concrete))
		for key, child := range concrete {
			prunedChild, ok := pruneClusterJSONValue(child)
			if !ok {
				continue
			}
			pruned[key] = prunedChild
		}
		if len(pruned) == 0 {
			return nil, false
		}
		return pruned, true
	case []any:
		pruned := make([]any, 0, len(concrete))
		for _, child := range concrete {
			prunedChild, ok := pruneClusterJSONValue(child)
			if !ok {
				continue
			}
			pruned = append(pruned, prunedChild)
		}
		if len(pruned) == 0 {
			return nil, false
		}
		return pruned, true
	case string:
		if strings.TrimSpace(concrete) == "" {
			return nil, false
		}
		return concrete, true
	case float64:
		if concrete == 0 {
			return nil, false
		}
		return concrete, true
	default:
		return concrete, true
	}
}

func clusterMapSubsetEqual(want map[string]any, got map[string]any) bool {
	for key, wantValue := range want {
		gotValue, ok := got[key]
		if !ok {
			return false
		}
		if !clusterJSONValueEqual(wantValue, gotValue) {
			return false
		}
	}
	return true
}

func clusterJSONValueEqual(left any, right any) bool {
	leftMap, leftIsMap := left.(map[string]any)
	rightMap, rightIsMap := right.(map[string]any)
	switch {
	case leftIsMap && rightIsMap:
		return clusterMapSubsetEqual(leftMap, rightMap)
	case leftIsMap || rightIsMap:
		return false
	}

	leftSlice, leftIsSlice := left.([]any)
	rightSlice, rightIsSlice := right.([]any)
	switch {
	case leftIsSlice && rightIsSlice:
		if len(leftSlice) != len(rightSlice) {
			return false
		}
		for i := range leftSlice {
			if !clusterJSONValueEqual(leftSlice[i], rightSlice[i]) {
				return false
			}
		}
		return true
	case leftIsSlice || rightIsSlice:
		return false
	}

	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}

/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package securitylist

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	coresdk "github.com/oracle/oci-go-sdk/v65/core"
	corev1beta1 "github.com/oracle/oci-service-operator/api/core/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	securityListRequeueDuration      = time.Minute
	securityListLifecycleStateUpdate = coresdk.SecurityListLifecycleStateEnum("UPDATING")
)

var (
	securityListSDKContractOnce sync.Once
	securityListSDKContractErr  error
)

type securityListOCIClient interface {
	CreateSecurityList(ctx context.Context, request coresdk.CreateSecurityListRequest) (coresdk.CreateSecurityListResponse, error)
	GetSecurityList(ctx context.Context, request coresdk.GetSecurityListRequest) (coresdk.GetSecurityListResponse, error)
	UpdateSecurityList(ctx context.Context, request coresdk.UpdateSecurityListRequest) (coresdk.UpdateSecurityListResponse, error)
	DeleteSecurityList(ctx context.Context, request coresdk.DeleteSecurityListRequest) (coresdk.DeleteSecurityListResponse, error)
}

type securityListRuntimeClient struct {
	manager *SecurityListServiceManager
	client  securityListOCIClient
	initErr error
}

type normalizedSecurityRule struct {
	endpointType  string
	endpoint      string
	protocol      string
	typeQualifier string
	stateless     bool
	description   string
	icmpType      int
	hasIcmpType   bool
	icmpCode      int
	hasIcmpCode   bool
	tcpDstMin     int
	tcpDstMax     int
	hasTCPDst     bool
	tcpSrcMin     int
	tcpSrcMax     int
	hasTCPSrc     bool
	udpDstMin     int
	udpDstMax     int
	hasUDPDst     bool
	udpSrcMin     int
	udpSrcMax     int
	hasUDPSrc     bool
}

func init() {
	newSecurityListServiceClient = func(manager *SecurityListServiceManager) SecurityListServiceClient {
		sdkClient, err := coresdk.NewVirtualNetworkClientWithConfigurationProvider(manager.Provider)
		runtimeClient := &securityListRuntimeClient{
			manager: manager,
			client:  sdkClient,
		}
		if err != nil {
			runtimeClient.initErr = fmt.Errorf("initialize SecurityList OCI client: %w", err)
		}
		return runtimeClient
	}
}

func (c *securityListRuntimeClient) CreateOrUpdate(ctx context.Context, resource *corev1beta1.SecurityList, _ ctrl.Request) (servicemanager.OSOKResponse, error) {
	if c.initErr != nil {
		return c.fail(resource, c.initErr)
	}
	if err := validateSecurityListSDKContract(); err != nil {
		return c.fail(resource, err)
	}

	trackedID := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	if trackedID == "" {
		return c.create(ctx, resource)
	}

	current, err := c.get(ctx, trackedID)
	if err != nil {
		if isSecurityListNotFoundOCI(err) {
			c.clearTrackedIdentity(resource)
			return c.create(ctx, resource)
		}
		return c.fail(resource, normalizeSecurityListOCIError(err))
	}

	if err := c.projectStatus(resource, current); err != nil {
		return c.fail(resource, err)
	}

	switch current.LifecycleState {
	case coresdk.SecurityListLifecycleStateProvisioning, securityListLifecycleStateUpdate, coresdk.SecurityListLifecycleStateTerminating, coresdk.SecurityListLifecycleStateTerminated:
		return c.applyLifecycle(resource, current)
	}

	updateRequest, updateNeeded, err := c.buildUpdateRequest(resource, current)
	if err != nil {
		return c.fail(resource, err)
	}

	if updateNeeded {
		response, err := c.client.UpdateSecurityList(ctx, updateRequest)
		if err != nil {
			return c.fail(resource, normalizeSecurityListOCIError(err))
		}
		current = response.SecurityList
	}

	if err := c.projectStatus(resource, current); err != nil {
		return c.fail(resource, err)
	}
	return c.applyLifecycle(resource, current)
}

func (c *securityListRuntimeClient) Delete(ctx context.Context, resource *corev1beta1.SecurityList) (bool, error) {
	if c.initErr != nil {
		return false, c.initErr
	}
	if err := validateSecurityListSDKContract(); err != nil {
		return false, err
	}

	trackedID := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	if trackedID == "" {
		c.markDeleted(resource, "OCI resource identifier is not recorded")
		return true, nil
	}

	deleteRequest := coresdk.DeleteSecurityListRequest{
		SecurityListId: common.String(trackedID),
	}
	if _, err := c.client.DeleteSecurityList(ctx, deleteRequest); err != nil {
		if isSecurityListNotFoundOCI(err) {
			c.markDeleted(resource, "OCI resource no longer exists")
			return true, nil
		}
		return false, normalizeSecurityListOCIError(err)
	}

	current, err := c.get(ctx, trackedID)
	if err != nil {
		if isSecurityListNotFoundOCI(err) {
			c.markDeleted(resource, "OCI resource deleted")
			return true, nil
		}
		return false, normalizeSecurityListOCIError(err)
	}

	if err := c.projectStatus(resource, current); err != nil {
		return false, err
	}
	c.markTerminating(resource, current)
	return false, nil
}

func (c *securityListRuntimeClient) create(ctx context.Context, resource *corev1beta1.SecurityList) (servicemanager.OSOKResponse, error) {
	request := coresdk.CreateSecurityListRequest{
		CreateSecurityListDetails: buildCreateSecurityListDetails(resource.Spec),
	}

	response, err := c.client.CreateSecurityList(ctx, request)
	if err != nil {
		return c.fail(resource, normalizeSecurityListOCIError(err))
	}

	if err := c.projectStatus(resource, response.SecurityList); err != nil {
		return c.fail(resource, err)
	}
	return c.applyLifecycle(resource, response.SecurityList)
}

func (c *securityListRuntimeClient) get(ctx context.Context, ocid string) (coresdk.SecurityList, error) {
	response, err := c.client.GetSecurityList(ctx, coresdk.GetSecurityListRequest{
		SecurityListId: common.String(ocid),
	})
	if err != nil {
		return coresdk.SecurityList{}, err
	}
	return response.SecurityList, nil
}

func (c *securityListRuntimeClient) buildUpdateRequest(resource *corev1beta1.SecurityList, current coresdk.SecurityList) (coresdk.UpdateSecurityListRequest, bool, error) {
	if current.Id == nil || strings.TrimSpace(*current.Id) == "" {
		return coresdk.UpdateSecurityListRequest{}, false, fmt.Errorf("current SecurityList does not expose an OCI identifier")
	}

	if err := validateSecurityListCreateOnlyDrift(resource.Spec, current); err != nil {
		return coresdk.UpdateSecurityListRequest{}, false, err
	}

	updateDetails := coresdk.UpdateSecurityListDetails{}
	updateNeeded := false

	if !stringPtrEqual(current.DisplayName, resource.Spec.DisplayName) {
		updateDetails.DisplayName = common.String(resource.Spec.DisplayName)
		updateNeeded = true
	}

	desiredFreeformTags := desiredFreeformTagsForUpdate(resource.Spec.FreeformTags, current.FreeformTags)
	if !reflect.DeepEqual(current.FreeformTags, desiredFreeformTags) {
		updateDetails.FreeformTags = desiredFreeformTags
		updateNeeded = true
	}

	desiredDefinedTags := desiredDefinedTagsForUpdate(resource.Spec.DefinedTags, current.DefinedTags)
	if !reflect.DeepEqual(current.DefinedTags, desiredDefinedTags) {
		updateDetails.DefinedTags = desiredDefinedTags
		updateNeeded = true
	}

	desiredEgressRules := convertSpecEgressRulesToOCI(resource.Spec.EgressSecurityRules)
	if !normalizedEgressSecurityRulesEqual(current.EgressSecurityRules, desiredEgressRules) {
		updateDetails.EgressSecurityRules = desiredEgressRules
		updateNeeded = true
	}

	desiredIngressRules := convertSpecIngressRulesToOCI(resource.Spec.IngressSecurityRules)
	if !normalizedIngressSecurityRulesEqual(current.IngressSecurityRules, desiredIngressRules) {
		updateDetails.IngressSecurityRules = desiredIngressRules
		updateNeeded = true
	}

	if !updateNeeded {
		return coresdk.UpdateSecurityListRequest{}, false, nil
	}

	return coresdk.UpdateSecurityListRequest{
		SecurityListId:            current.Id,
		UpdateSecurityListDetails: updateDetails,
	}, true, nil
}

func buildCreateSecurityListDetails(spec corev1beta1.SecurityListSpec) coresdk.CreateSecurityListDetails {
	createDetails := coresdk.CreateSecurityListDetails{
		CompartmentId:       common.String(spec.CompartmentId),
		EgressSecurityRules: convertSpecEgressRulesToOCI(spec.EgressSecurityRules),
		IngressSecurityRules: convertSpecIngressRulesToOCI(
			spec.IngressSecurityRules,
		),
		VcnId: common.String(spec.VcnId),
	}

	if spec.DefinedTags != nil {
		createDetails.DefinedTags = *util.ConvertToOciDefinedTags(&spec.DefinedTags)
	}
	if spec.DisplayName != "" {
		createDetails.DisplayName = common.String(spec.DisplayName)
	}
	if spec.FreeformTags != nil {
		createDetails.FreeformTags = spec.FreeformTags
	}

	return createDetails
}

func desiredFreeformTagsForUpdate(spec map[string]string, current map[string]string) map[string]string {
	if spec != nil {
		return cloneStringMap(spec)
	}
	if current != nil {
		return map[string]string{}
	}
	return nil
}

func desiredDefinedTagsForUpdate(spec map[string]shared.MapValue, current map[string]map[string]interface{}) map[string]map[string]interface{} {
	if spec != nil {
		return *util.ConvertToOciDefinedTags(&spec)
	}
	if current != nil {
		return map[string]map[string]interface{}{}
	}
	return nil
}

func validateSecurityListCreateOnlyDrift(spec corev1beta1.SecurityListSpec, current coresdk.SecurityList) error {
	var unsupported []string

	if !stringCreateOnlyMatches(current.CompartmentId, spec.CompartmentId) {
		unsupported = append(unsupported, "compartmentId")
	}
	if !stringCreateOnlyMatches(current.VcnId, spec.VcnId) {
		unsupported = append(unsupported, "vcnId")
	}

	if len(unsupported) == 0 {
		return nil
	}
	return fmt.Errorf("SecurityList create-only field drift is not supported: %s", strings.Join(unsupported, ", "))
}

func convertSpecEgressRulesToOCI(rules []corev1beta1.SecurityListEgressSecurityRule) []coresdk.EgressSecurityRule {
	converted := make([]coresdk.EgressSecurityRule, 0, len(rules))
	for _, rule := range rules {
		convertedRule := coresdk.EgressSecurityRule{
			Destination: common.String(rule.Destination),
			Protocol:    common.String(rule.Protocol),
		}
		if rule.DestinationType != "" {
			convertedRule.DestinationType = coresdk.EgressSecurityRuleDestinationTypeEnum(strings.TrimSpace(rule.DestinationType))
		}
		if icmpOptions := convertSpecIcmpOptions(rule.IcmpOptions.Type, rule.IcmpOptions.Code); icmpOptions != nil {
			convertedRule.IcmpOptions = icmpOptions
		}
		if rule.IsStateless {
			convertedRule.IsStateless = common.Bool(rule.IsStateless)
		}
		if tcpOptions := convertSpecTCPOptions(
			rule.TcpOptions.DestinationPortRange.Min,
			rule.TcpOptions.DestinationPortRange.Max,
			rule.TcpOptions.SourcePortRange.Min,
			rule.TcpOptions.SourcePortRange.Max,
		); tcpOptions != nil {
			convertedRule.TcpOptions = tcpOptions
		}
		if udpOptions := convertSpecUDPOptions(
			rule.UdpOptions.DestinationPortRange.Min,
			rule.UdpOptions.DestinationPortRange.Max,
			rule.UdpOptions.SourcePortRange.Min,
			rule.UdpOptions.SourcePortRange.Max,
		); udpOptions != nil {
			convertedRule.UdpOptions = udpOptions
		}
		if rule.Description != "" {
			convertedRule.Description = common.String(rule.Description)
		}
		converted = append(converted, convertedRule)
	}
	return converted
}

func convertSpecIngressRulesToOCI(rules []corev1beta1.SecurityListIngressSecurityRule) []coresdk.IngressSecurityRule {
	converted := make([]coresdk.IngressSecurityRule, 0, len(rules))
	for _, rule := range rules {
		convertedRule := coresdk.IngressSecurityRule{
			Protocol: common.String(rule.Protocol),
			Source:   common.String(rule.Source),
		}
		if icmpOptions := convertSpecIcmpOptions(rule.IcmpOptions.Type, rule.IcmpOptions.Code); icmpOptions != nil {
			convertedRule.IcmpOptions = icmpOptions
		}
		if rule.IsStateless {
			convertedRule.IsStateless = common.Bool(rule.IsStateless)
		}
		if rule.SourceType != "" {
			convertedRule.SourceType = coresdk.IngressSecurityRuleSourceTypeEnum(strings.TrimSpace(rule.SourceType))
		}
		if tcpOptions := convertSpecTCPOptions(
			rule.TcpOptions.DestinationPortRange.Min,
			rule.TcpOptions.DestinationPortRange.Max,
			rule.TcpOptions.SourcePortRange.Min,
			rule.TcpOptions.SourcePortRange.Max,
		); tcpOptions != nil {
			convertedRule.TcpOptions = tcpOptions
		}
		if udpOptions := convertSpecUDPOptions(
			rule.UdpOptions.DestinationPortRange.Min,
			rule.UdpOptions.DestinationPortRange.Max,
			rule.UdpOptions.SourcePortRange.Min,
			rule.UdpOptions.SourcePortRange.Max,
		); udpOptions != nil {
			convertedRule.UdpOptions = udpOptions
		}
		if rule.Description != "" {
			convertedRule.Description = common.String(rule.Description)
		}
		converted = append(converted, convertedRule)
	}
	return converted
}

func convertSpecIcmpOptions(typ, code int) *coresdk.IcmpOptions {
	if typ == 0 && code == 0 {
		return nil
	}

	options := &coresdk.IcmpOptions{
		Type: common.Int(typ),
	}
	if code != 0 {
		options.Code = common.Int(code)
	}
	return options
}

func convertSpecTCPOptions(dstMin, dstMax, srcMin, srcMax int) *coresdk.TcpOptions {
	options := &coresdk.TcpOptions{}
	if portRange := convertSpecPortRange(dstMin, dstMax); portRange != nil {
		options.DestinationPortRange = portRange
	}
	if portRange := convertSpecPortRange(srcMin, srcMax); portRange != nil {
		options.SourcePortRange = portRange
	}
	if options.DestinationPortRange == nil && options.SourcePortRange == nil {
		return nil
	}
	return options
}

func convertSpecUDPOptions(dstMin, dstMax, srcMin, srcMax int) *coresdk.UdpOptions {
	options := &coresdk.UdpOptions{}
	if portRange := convertSpecPortRange(dstMin, dstMax); portRange != nil {
		options.DestinationPortRange = portRange
	}
	if portRange := convertSpecPortRange(srcMin, srcMax); portRange != nil {
		options.SourcePortRange = portRange
	}
	if options.DestinationPortRange == nil && options.SourcePortRange == nil {
		return nil
	}
	return options
}

func convertSpecPortRange(min, max int) *coresdk.PortRange {
	if min == 0 && max == 0 {
		return nil
	}
	return &coresdk.PortRange{
		Max: common.Int(max),
		Min: common.Int(min),
	}
}

func convertOCIEgressRulesToStatus(rules []coresdk.EgressSecurityRule) []corev1beta1.SecurityListEgressSecurityRuleObservedState {
	converted := make([]corev1beta1.SecurityListEgressSecurityRuleObservedState, 0, len(rules))
	for _, rule := range rules {
		converted = append(converted, corev1beta1.SecurityListEgressSecurityRuleObservedState{
			Destination:     stringValue(rule.Destination),
			Protocol:        stringValue(rule.Protocol),
			DestinationType: string(rule.DestinationType),
			IcmpOptions: corev1beta1.SecurityListEgressSecurityRuleIcmpOptions{
				Type: intValue(rule.IcmpOptions, func(v *coresdk.IcmpOptions) *int { return v.Type }),
				Code: intValue(rule.IcmpOptions, func(v *coresdk.IcmpOptions) *int { return v.Code }),
			},
			IsStateless: boolValue(rule.IsStateless),
			TcpOptions: corev1beta1.SecurityListEgressSecurityRuleTcpOptionsObservedState{
				DestinationPortRange: convertOCIPortRangeToEgressTCPDestinationStatus(rule.TcpOptions, func(v *coresdk.TcpOptions) *coresdk.PortRange {
					return v.DestinationPortRange
				}),
				SourcePortRange: convertOCIPortRangeToEgressTCPSourceStatus(rule.TcpOptions, func(v *coresdk.TcpOptions) *coresdk.PortRange {
					return v.SourcePortRange
				}),
			},
			UdpOptions: corev1beta1.SecurityListEgressSecurityRuleUdpOptionsObservedState{
				DestinationPortRange: convertOCIPortRangeToEgressUDPDestinationStatus(rule.UdpOptions, func(v *coresdk.UdpOptions) *coresdk.PortRange {
					return v.DestinationPortRange
				}),
				SourcePortRange: convertOCIPortRangeToEgressUDPSourceStatus(rule.UdpOptions, func(v *coresdk.UdpOptions) *coresdk.PortRange {
					return v.SourcePortRange
				}),
			},
			Description: stringValue(rule.Description),
		})
	}
	return converted
}

func convertOCIIngressRulesToStatus(rules []coresdk.IngressSecurityRule) []corev1beta1.SecurityListIngressSecurityRuleObservedState {
	converted := make([]corev1beta1.SecurityListIngressSecurityRuleObservedState, 0, len(rules))
	for _, rule := range rules {
		converted = append(converted, corev1beta1.SecurityListIngressSecurityRuleObservedState{
			Protocol: stringValue(rule.Protocol),
			Source:   stringValue(rule.Source),
			IcmpOptions: corev1beta1.SecurityListIngressSecurityRuleIcmpOptions{
				Type: intValue(rule.IcmpOptions, func(v *coresdk.IcmpOptions) *int { return v.Type }),
				Code: intValue(rule.IcmpOptions, func(v *coresdk.IcmpOptions) *int { return v.Code }),
			},
			IsStateless: boolValue(rule.IsStateless),
			SourceType:  string(rule.SourceType),
			TcpOptions: corev1beta1.SecurityListIngressSecurityRuleTcpOptionsObservedState{
				DestinationPortRange: convertOCIPortRangeToIngressTCPDestinationStatus(rule.TcpOptions, func(v *coresdk.TcpOptions) *coresdk.PortRange {
					return v.DestinationPortRange
				}),
				SourcePortRange: convertOCIPortRangeToIngressTCPSourceStatus(rule.TcpOptions, func(v *coresdk.TcpOptions) *coresdk.PortRange {
					return v.SourcePortRange
				}),
			},
			UdpOptions: corev1beta1.SecurityListIngressSecurityRuleUdpOptionsObservedState{
				DestinationPortRange: convertOCIPortRangeToIngressUDPDestinationStatus(rule.UdpOptions, func(v *coresdk.UdpOptions) *coresdk.PortRange {
					return v.DestinationPortRange
				}),
				SourcePortRange: convertOCIPortRangeToIngressUDPSourceStatus(rule.UdpOptions, func(v *coresdk.UdpOptions) *coresdk.PortRange {
					return v.SourcePortRange
				}),
			},
			Description: stringValue(rule.Description),
		})
	}
	return converted
}

func normalizedEgressSecurityRulesEqual(current []coresdk.EgressSecurityRule, desired []coresdk.EgressSecurityRule) bool {
	return reflect.DeepEqual(normalizeEgressSecurityRules(current), normalizeEgressSecurityRules(desired))
}

func normalizedIngressSecurityRulesEqual(current []coresdk.IngressSecurityRule, desired []coresdk.IngressSecurityRule) bool {
	return reflect.DeepEqual(normalizeIngressSecurityRules(current), normalizeIngressSecurityRules(desired))
}

func normalizeEgressSecurityRules(rules []coresdk.EgressSecurityRule) []normalizedSecurityRule {
	normalized := make([]normalizedSecurityRule, 0, len(rules))
	for _, rule := range rules {
		normalized = append(normalized, normalizedSecurityRule{
			endpointType:  strings.ToUpper(strings.TrimSpace(string(rule.DestinationType))),
			endpoint:      strings.TrimSpace(stringValue(rule.Destination)),
			protocol:      strings.TrimSpace(stringValue(rule.Protocol)),
			typeQualifier: "egress",
			stateless:     boolValue(rule.IsStateless),
			description:   strings.TrimSpace(stringValue(rule.Description)),
			icmpType:      intValue(rule.IcmpOptions, func(v *coresdk.IcmpOptions) *int { return v.Type }),
			hasIcmpType:   nestedIntPresent(rule.IcmpOptions, func(v *coresdk.IcmpOptions) *int { return v.Type }),
			icmpCode:      intValue(rule.IcmpOptions, func(v *coresdk.IcmpOptions) *int { return v.Code }),
			hasIcmpCode:   nestedIntPresent(rule.IcmpOptions, func(v *coresdk.IcmpOptions) *int { return v.Code }),
			tcpDstMin:     portRangeMin(rule.TcpOptions, func(v *coresdk.TcpOptions) *coresdk.PortRange { return v.DestinationPortRange }),
			tcpDstMax:     portRangeMax(rule.TcpOptions, func(v *coresdk.TcpOptions) *coresdk.PortRange { return v.DestinationPortRange }),
			hasTCPDst:     portRangePresent(rule.TcpOptions, func(v *coresdk.TcpOptions) *coresdk.PortRange { return v.DestinationPortRange }),
			tcpSrcMin:     portRangeMin(rule.TcpOptions, func(v *coresdk.TcpOptions) *coresdk.PortRange { return v.SourcePortRange }),
			tcpSrcMax:     portRangeMax(rule.TcpOptions, func(v *coresdk.TcpOptions) *coresdk.PortRange { return v.SourcePortRange }),
			hasTCPSrc:     portRangePresent(rule.TcpOptions, func(v *coresdk.TcpOptions) *coresdk.PortRange { return v.SourcePortRange }),
			udpDstMin:     portRangeMin(rule.UdpOptions, func(v *coresdk.UdpOptions) *coresdk.PortRange { return v.DestinationPortRange }),
			udpDstMax:     portRangeMax(rule.UdpOptions, func(v *coresdk.UdpOptions) *coresdk.PortRange { return v.DestinationPortRange }),
			hasUDPDst:     portRangePresent(rule.UdpOptions, func(v *coresdk.UdpOptions) *coresdk.PortRange { return v.DestinationPortRange }),
			udpSrcMin:     portRangeMin(rule.UdpOptions, func(v *coresdk.UdpOptions) *coresdk.PortRange { return v.SourcePortRange }),
			udpSrcMax:     portRangeMax(rule.UdpOptions, func(v *coresdk.UdpOptions) *coresdk.PortRange { return v.SourcePortRange }),
			hasUDPSrc:     portRangePresent(rule.UdpOptions, func(v *coresdk.UdpOptions) *coresdk.PortRange { return v.SourcePortRange }),
		})
	}
	sort.Slice(normalized, func(i, j int) bool {
		return normalizedSecurityRuleKey(normalized[i]) < normalizedSecurityRuleKey(normalized[j])
	})
	return normalized
}

func normalizeIngressSecurityRules(rules []coresdk.IngressSecurityRule) []normalizedSecurityRule {
	normalized := make([]normalizedSecurityRule, 0, len(rules))
	for _, rule := range rules {
		normalized = append(normalized, normalizedSecurityRule{
			endpointType:  strings.ToUpper(strings.TrimSpace(string(rule.SourceType))),
			endpoint:      strings.TrimSpace(stringValue(rule.Source)),
			protocol:      strings.TrimSpace(stringValue(rule.Protocol)),
			typeQualifier: "ingress",
			stateless:     boolValue(rule.IsStateless),
			description:   strings.TrimSpace(stringValue(rule.Description)),
			icmpType:      intValue(rule.IcmpOptions, func(v *coresdk.IcmpOptions) *int { return v.Type }),
			hasIcmpType:   nestedIntPresent(rule.IcmpOptions, func(v *coresdk.IcmpOptions) *int { return v.Type }),
			icmpCode:      intValue(rule.IcmpOptions, func(v *coresdk.IcmpOptions) *int { return v.Code }),
			hasIcmpCode:   nestedIntPresent(rule.IcmpOptions, func(v *coresdk.IcmpOptions) *int { return v.Code }),
			tcpDstMin:     portRangeMin(rule.TcpOptions, func(v *coresdk.TcpOptions) *coresdk.PortRange { return v.DestinationPortRange }),
			tcpDstMax:     portRangeMax(rule.TcpOptions, func(v *coresdk.TcpOptions) *coresdk.PortRange { return v.DestinationPortRange }),
			hasTCPDst:     portRangePresent(rule.TcpOptions, func(v *coresdk.TcpOptions) *coresdk.PortRange { return v.DestinationPortRange }),
			tcpSrcMin:     portRangeMin(rule.TcpOptions, func(v *coresdk.TcpOptions) *coresdk.PortRange { return v.SourcePortRange }),
			tcpSrcMax:     portRangeMax(rule.TcpOptions, func(v *coresdk.TcpOptions) *coresdk.PortRange { return v.SourcePortRange }),
			hasTCPSrc:     portRangePresent(rule.TcpOptions, func(v *coresdk.TcpOptions) *coresdk.PortRange { return v.SourcePortRange }),
			udpDstMin:     portRangeMin(rule.UdpOptions, func(v *coresdk.UdpOptions) *coresdk.PortRange { return v.DestinationPortRange }),
			udpDstMax:     portRangeMax(rule.UdpOptions, func(v *coresdk.UdpOptions) *coresdk.PortRange { return v.DestinationPortRange }),
			hasUDPDst:     portRangePresent(rule.UdpOptions, func(v *coresdk.UdpOptions) *coresdk.PortRange { return v.DestinationPortRange }),
			udpSrcMin:     portRangeMin(rule.UdpOptions, func(v *coresdk.UdpOptions) *coresdk.PortRange { return v.SourcePortRange }),
			udpSrcMax:     portRangeMax(rule.UdpOptions, func(v *coresdk.UdpOptions) *coresdk.PortRange { return v.SourcePortRange }),
			hasUDPSrc:     portRangePresent(rule.UdpOptions, func(v *coresdk.UdpOptions) *coresdk.PortRange { return v.SourcePortRange }),
		})
	}
	sort.Slice(normalized, func(i, j int) bool {
		return normalizedSecurityRuleKey(normalized[i]) < normalizedSecurityRuleKey(normalized[j])
	})
	return normalized
}

func normalizedSecurityRuleKey(rule normalizedSecurityRule) string {
	return strings.Join([]string{
		rule.typeQualifier,
		rule.endpointType,
		rule.endpoint,
		rule.protocol,
		strconvBool(rule.stateless),
		rule.description,
		strconvBool(rule.hasIcmpType),
		strconvInt(rule.icmpType),
		strconvBool(rule.hasIcmpCode),
		strconvInt(rule.icmpCode),
		strconvBool(rule.hasTCPDst),
		strconvInt(rule.tcpDstMin),
		strconvInt(rule.tcpDstMax),
		strconvBool(rule.hasTCPSrc),
		strconvInt(rule.tcpSrcMin),
		strconvInt(rule.tcpSrcMax),
		strconvBool(rule.hasUDPDst),
		strconvInt(rule.udpDstMin),
		strconvInt(rule.udpDstMax),
		strconvBool(rule.hasUDPSrc),
		strconvInt(rule.udpSrcMin),
		strconvInt(rule.udpSrcMax),
	}, "\x00")
}

func (c *securityListRuntimeClient) applyLifecycle(resource *corev1beta1.SecurityList, current coresdk.SecurityList) (servicemanager.OSOKResponse, error) {
	status := &resource.Status.OsokStatus
	now := time.Now()
	if status.CreatedAt == nil && current.Id != nil && strings.TrimSpace(*current.Id) != "" {
		createdAt := metav1Time(now)
		status.CreatedAt = &createdAt
	}
	updatedAt := metav1Time(now)
	status.UpdatedAt = &updatedAt
	if current.Id != nil {
		status.Ocid = shared.OCID(*current.Id)
	}

	message := securityListLifecycleMessage(current)
	status.Message = message

	switch current.LifecycleState {
	case coresdk.SecurityListLifecycleStateAvailable:
		status.Reason = string(shared.Active)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Active, v1.ConditionTrue, "", message, c.manager.Log)
		return servicemanager.OSOKResponse{IsSuccessful: true}, nil
	case coresdk.SecurityListLifecycleStateProvisioning:
		status.Reason = string(shared.Provisioning)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Provisioning, v1.ConditionTrue, "", message, c.manager.Log)
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: securityListRequeueDuration}, nil
	case securityListLifecycleStateUpdate:
		status.Reason = string(shared.Updating)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Updating, v1.ConditionTrue, "", message, c.manager.Log)
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: securityListRequeueDuration}, nil
	case coresdk.SecurityListLifecycleStateTerminating, coresdk.SecurityListLifecycleStateTerminated:
		status.Reason = string(shared.Terminating)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", message, c.manager.Log)
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: securityListRequeueDuration}, nil
	default:
		return c.fail(resource, fmt.Errorf("SecurityList lifecycle state %q is not modeled for create or update", current.LifecycleState))
	}
}

func (c *securityListRuntimeClient) fail(resource *corev1beta1.SecurityList, err error) (servicemanager.OSOKResponse, error) {
	status := &resource.Status.OsokStatus
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	updatedAt := metav1Time(time.Now())
	status.UpdatedAt = &updatedAt
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Failed, v1.ConditionFalse, "", err.Error(), c.manager.Log)
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func (c *securityListRuntimeClient) markDeleted(resource *corev1beta1.SecurityList, message string) {
	status := &resource.Status.OsokStatus
	now := metav1Time(time.Now())
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", message, c.manager.Log)
}

func (c *securityListRuntimeClient) clearTrackedIdentity(resource *corev1beta1.SecurityList) {
	resource.Status = corev1beta1.SecurityListStatus{}
}

func (c *securityListRuntimeClient) markTerminating(resource *corev1beta1.SecurityList, current coresdk.SecurityList) {
	status := &resource.Status.OsokStatus
	now := metav1Time(time.Now())
	status.UpdatedAt = &now
	status.Message = securityListLifecycleMessage(current)
	status.Reason = string(shared.Terminating)
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", status.Message, c.manager.Log)
}

func (c *securityListRuntimeClient) projectStatus(resource *corev1beta1.SecurityList, current coresdk.SecurityList) error {
	resource.Status = corev1beta1.SecurityListStatus{
		OsokStatus:           resource.Status.OsokStatus,
		CompartmentId:        stringValue(current.CompartmentId),
		DisplayName:          stringValue(current.DisplayName),
		EgressSecurityRules:  convertOCIEgressRulesToStatus(current.EgressSecurityRules),
		Id:                   stringValue(current.Id),
		IngressSecurityRules: convertOCIIngressRulesToStatus(current.IngressSecurityRules),
		LifecycleState:       string(current.LifecycleState),
		TimeCreated:          sdkTimeString(current.TimeCreated),
		VcnId:                stringValue(current.VcnId),
		DefinedTags:          convertOCIToStatusDefinedTags(current.DefinedTags),
		FreeformTags:         cloneStringMap(current.FreeformTags),
	}
	return nil
}

func securityListLifecycleMessage(current coresdk.SecurityList) string {
	name := ""
	if current.DisplayName != nil {
		name = *current.DisplayName
	}
	if name == "" && current.Id != nil {
		name = *current.Id
	}
	if name == "" {
		name = "SecurityList"
	}
	return fmt.Sprintf("SecurityList %s is %s", name, current.LifecycleState)
}

func validateSecurityListSDKContract() error {
	securityListSDKContractOnce.Do(func() {
		updateFields := reflect.TypeOf(coresdk.UpdateSecurityListDetails{})
		expectedMutableFields := []string{
			"DefinedTags",
			"DisplayName",
			"EgressSecurityRules",
			"FreeformTags",
			"IngressSecurityRules",
		}
		for _, fieldName := range expectedMutableFields {
			if _, ok := updateFields.FieldByName(fieldName); !ok {
				securityListSDKContractErr = fmt.Errorf("formal/imports/core/securitylist.json assumes SecurityList update field %q exists in vendored SDK", fieldName)
				return
			}
		}
		if _, ok := updateFields.FieldByName("CompartmentId"); ok {
			securityListSDKContractErr = fmt.Errorf("formal/imports/core/securitylist.json marks compartmentId mutable, but vendored UpdateSecurityListDetails unexpectedly exposes CompartmentId")
			return
		}
		if _, ok := updateFields.FieldByName("VcnId"); ok {
			securityListSDKContractErr = fmt.Errorf("formal/imports/core/securitylist.json marks vcnId mutable, but vendored UpdateSecurityListDetails unexpectedly exposes VcnId")
			return
		}

		createFields := reflect.TypeOf(coresdk.CreateSecurityListDetails{})
		for _, fieldName := range []string{"CompartmentId", "EgressSecurityRules", "IngressSecurityRules", "VcnId"} {
			if _, ok := createFields.FieldByName(fieldName); !ok {
				securityListSDKContractErr = fmt.Errorf("formal/imports/core/securitylist.json assumes SecurityList create field %q exists in vendored SDK", fieldName)
				return
			}
		}

		lifecycleValues := make(map[string]struct{}, len(coresdk.GetSecurityListLifecycleStateEnumStringValues()))
		for _, value := range coresdk.GetSecurityListLifecycleStateEnumStringValues() {
			lifecycleValues[value] = struct{}{}
		}
		for _, value := range []string{
			string(coresdk.SecurityListLifecycleStateAvailable),
			string(coresdk.SecurityListLifecycleStateProvisioning),
			string(coresdk.SecurityListLifecycleStateTerminating),
			string(coresdk.SecurityListLifecycleStateTerminated),
		} {
			if _, ok := lifecycleValues[value]; !ok {
				securityListSDKContractErr = fmt.Errorf("vendored SDK no longer exposes SecurityList lifecycle %q", value)
				return
			}
		}
		if _, ok := lifecycleValues["ACTIVE"]; ok {
			securityListSDKContractErr = fmt.Errorf("formal/imports/core/securitylist.json still assumes ACTIVE, but vendored SDK now needs reevaluation because ACTIVE unexpectedly exists")
			return
		}
	})
	return securityListSDKContractErr
}

func normalizeSecurityListOCIError(err error) error {
	var serviceErr common.ServiceError
	if !errors.As(err, &serviceErr) {
		return err
	}
	if _, normalized := errorutil.OciErrorTypeResponse(err); normalized != nil {
		return normalized
	}
	return err
}

func isSecurityListNotFoundOCI(err error) bool {
	return errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound()
}

func stringPtrEqual(actual *string, expected string) bool {
	if actual == nil {
		return strings.TrimSpace(expected) == ""
	}
	return *actual == expected
}

func stringCreateOnlyMatches(actual *string, expected string) bool {
	return strings.TrimSpace(stringValue(actual)) == strings.TrimSpace(expected)
}

func metav1Time(t time.Time) metav1.Time {
	return metav1.NewTime(t)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func boolValue(value *bool) bool {
	if value == nil {
		return false
	}
	return *value
}

func sdkTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Time.Format(time.RFC3339Nano)
}

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func convertOCIToStatusDefinedTags(input map[string]map[string]interface{}) map[string]shared.MapValue {
	if input == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(input))
	for namespace, values := range input {
		convertedValues := make(shared.MapValue, len(values))
		for key, value := range values {
			convertedValues[key] = fmt.Sprint(value)
		}
		converted[namespace] = convertedValues
	}
	return converted
}

func intValue[T any](input *T, getter func(*T) *int) int {
	if input == nil {
		return 0
	}
	value := getter(input)
	if value == nil {
		return 0
	}
	return *value
}

func nestedIntPresent[T any](input *T, getter func(*T) *int) bool {
	if input == nil {
		return false
	}
	return getter(input) != nil
}

func portRangePresent[T any](input *T, getter func(*T) *coresdk.PortRange) bool {
	if input == nil {
		return false
	}
	return getter(input) != nil
}

func portRangeMin[T any](input *T, getter func(*T) *coresdk.PortRange) int {
	if input == nil {
		return 0
	}
	portRange := getter(input)
	if portRange == nil || portRange.Min == nil {
		return 0
	}
	return *portRange.Min
}

func portRangeMax[T any](input *T, getter func(*T) *coresdk.PortRange) int {
	if input == nil {
		return 0
	}
	portRange := getter(input)
	if portRange == nil || portRange.Max == nil {
		return 0
	}
	return *portRange.Max
}

func convertOCIPortRangeToEgressTCPDestinationStatus[T any](input *T, getter func(*T) *coresdk.PortRange) corev1beta1.SecurityListEgressSecurityRuleTcpOptionsDestinationPortRange {
	return corev1beta1.SecurityListEgressSecurityRuleTcpOptionsDestinationPortRange{
		Min: portRangeMin(input, getter),
		Max: portRangeMax(input, getter),
	}
}

func convertOCIPortRangeToEgressTCPSourceStatus[T any](input *T, getter func(*T) *coresdk.PortRange) corev1beta1.SecurityListEgressSecurityRuleTcpOptionsSourcePortRange {
	return corev1beta1.SecurityListEgressSecurityRuleTcpOptionsSourcePortRange{
		Min: portRangeMin(input, getter),
		Max: portRangeMax(input, getter),
	}
}

func convertOCIPortRangeToEgressUDPDestinationStatus[T any](input *T, getter func(*T) *coresdk.PortRange) corev1beta1.SecurityListEgressSecurityRuleUdpOptionsDestinationPortRange {
	return corev1beta1.SecurityListEgressSecurityRuleUdpOptionsDestinationPortRange{
		Min: portRangeMin(input, getter),
		Max: portRangeMax(input, getter),
	}
}

func convertOCIPortRangeToEgressUDPSourceStatus[T any](input *T, getter func(*T) *coresdk.PortRange) corev1beta1.SecurityListEgressSecurityRuleUdpOptionsSourcePortRange {
	return corev1beta1.SecurityListEgressSecurityRuleUdpOptionsSourcePortRange{
		Min: portRangeMin(input, getter),
		Max: portRangeMax(input, getter),
	}
}

func convertOCIPortRangeToIngressTCPDestinationStatus[T any](input *T, getter func(*T) *coresdk.PortRange) corev1beta1.SecurityListIngressSecurityRuleTcpOptionsDestinationPortRange {
	return corev1beta1.SecurityListIngressSecurityRuleTcpOptionsDestinationPortRange{
		Min: portRangeMin(input, getter),
		Max: portRangeMax(input, getter),
	}
}

func convertOCIPortRangeToIngressTCPSourceStatus[T any](input *T, getter func(*T) *coresdk.PortRange) corev1beta1.SecurityListIngressSecurityRuleTcpOptionsSourcePortRange {
	return corev1beta1.SecurityListIngressSecurityRuleTcpOptionsSourcePortRange{
		Min: portRangeMin(input, getter),
		Max: portRangeMax(input, getter),
	}
}

func convertOCIPortRangeToIngressUDPDestinationStatus[T any](input *T, getter func(*T) *coresdk.PortRange) corev1beta1.SecurityListIngressSecurityRuleUdpOptionsDestinationPortRange {
	return corev1beta1.SecurityListIngressSecurityRuleUdpOptionsDestinationPortRange{
		Min: portRangeMin(input, getter),
		Max: portRangeMax(input, getter),
	}
}

func convertOCIPortRangeToIngressUDPSourceStatus[T any](input *T, getter func(*T) *coresdk.PortRange) corev1beta1.SecurityListIngressSecurityRuleUdpOptionsSourcePortRange {
	return corev1beta1.SecurityListIngressSecurityRuleUdpOptionsSourcePortRange{
		Min: portRangeMin(input, getter),
		Max: portRangeMax(input, getter),
	}
}

func strconvBool(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

func strconvInt(value int) string {
	return fmt.Sprintf("%d", value)
}

/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package sddc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocvpsdk "github.com/oracle/oci-go-sdk/v65/ocvp"
	ocvpv1beta1 "github.com/oracle/oci-service-operator/api/ocvp/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type sddcExistingResolver interface {
	ResolveExisting(context.Context, *ocvpv1beta1.Sddc) (string, error)
}

type sddcGeneratedParityClient struct {
	manager  *SddcServiceManager
	delegate SddcServiceClient
}

type generatedSddcServiceClient struct {
	generatedruntime.ServiceClient[*ocvpv1beta1.Sddc]

	sdkClient *ocvpsdk.SddcClient
}

type sanitizedSddcResponse struct {
	Body map[string]any `presentIn:"body"`
}

func init() {
	newSddcServiceClient = func(manager *SddcServiceManager) SddcServiceClient {
		return &sddcGeneratedParityClient{
			manager:  manager,
			delegate: newGeneratedSddcServiceClient(manager),
		}
	}
}

func newGeneratedSddcServiceClient(manager *SddcServiceManager) SddcServiceClient {
	var sdkClient *ocvpsdk.SddcClient
	client, err := ocvpsdk.NewSddcClientWithConfigurationProvider(manager.Provider)
	if err == nil {
		sdkClient = &client
	}

	config := generatedruntime.Config[*ocvpv1beta1.Sddc]{
		Kind:      "Sddc",
		SDKName:   "Sddc",
		Log:       manager.Log,
		Semantics: newSddcRuntimeSemantics(),
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.CreateSddcRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.CreateSddc(ctx, *request.(*ocvpsdk.CreateSddcRequest))
			},
			Fields: []generatedruntime.RequestField{{FieldName: "CreateSddcDetails", RequestName: "CreateSddcDetails", Contribution: "body", PreferResourceID: false}},
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.GetSddcRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				response, err := client.GetSddc(ctx, *request.(*ocvpsdk.GetSddcRequest))
				if err != nil {
					return nil, err
				}
				return sanitizeSddcResponse(response.Sddc)
			},
			Fields: []generatedruntime.RequestField{{FieldName: "SddcId", RequestName: "sddcId", Contribution: "path", PreferResourceID: true}},
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.ListSddcsRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.ListSddcs(ctx, *request.(*ocvpsdk.ListSddcsRequest))
			},
			Fields: []generatedruntime.RequestField{{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", PreferResourceID: false}, {FieldName: "ComputeAvailabilityDomain", RequestName: "computeAvailabilityDomain", Contribution: "query", PreferResourceID: false}, {FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", PreferResourceID: false}, {FieldName: "Limit", RequestName: "limit", Contribution: "query", PreferResourceID: false}, {FieldName: "Page", RequestName: "page", Contribution: "query", PreferResourceID: false}, {FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query", PreferResourceID: false}, {FieldName: "SortBy", RequestName: "sortBy", Contribution: "query", PreferResourceID: false}, {FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query", PreferResourceID: false}},
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.UpdateSddcRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.UpdateSddc(ctx, *request.(*ocvpsdk.UpdateSddcRequest))
			},
			Fields: []generatedruntime.RequestField{{FieldName: "SddcId", RequestName: "sddcId", Contribution: "path", PreferResourceID: true}, {FieldName: "UpdateSddcDetails", RequestName: "UpdateSddcDetails", Contribution: "body", PreferResourceID: false}},
		},
		Delete: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.DeleteSddcRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.DeleteSddc(ctx, *request.(*ocvpsdk.DeleteSddcRequest))
			},
			Fields: []generatedruntime.RequestField{{FieldName: "SddcId", RequestName: "sddcId", Contribution: "path", PreferResourceID: true}},
		},
	}
	if err != nil {
		config.InitError = fmt.Errorf("initialize Sddc OCI client: %w", err)
	}

	return generatedSddcServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*ocvpv1beta1.Sddc](config),
		sdkClient:     sdkClient,
	}
}

var _ sddcExistingResolver = generatedSddcServiceClient{}

func (c generatedSddcServiceClient) ResolveExisting(ctx context.Context, resource *ocvpv1beta1.Sddc) (string, error) {
	if resource == nil || c.sdkClient == nil {
		return "", nil
	}

	displayName := strings.TrimSpace(resource.Spec.DisplayName)
	compartmentID := strings.TrimSpace(resource.Spec.CompartmentId)
	if displayName == "" || compartmentID == "" {
		return "", nil
	}

	response, err := c.sdkClient.ListSddcs(ctx, ocvpsdk.ListSddcsRequest{
		CompartmentId: common.String(compartmentID),
		DisplayName:   common.String(displayName),
	})
	if err != nil {
		return "", fmt.Errorf("resolve existing Sddc via ListSddcs: %w", err)
	}

	matches := make([]string, 0, 1)
	for _, item := range response.Items {
		if !isReusableSddcSummary(item, compartmentID, displayName) {
			continue
		}
		if id := stringPointerValue(item.Id); id != "" {
			matches = append(matches, id)
		}
	}

	switch len(matches) {
	case 0:
		return "", nil
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("Sddc bind lookup found multiple reusable OCI resources for compartmentId=%q displayName=%q", compartmentID, displayName)
	}
}

func (c *sddcGeneratedParityClient) CreateOrUpdate(
	ctx context.Context,
	resource *ocvpv1beta1.Sddc,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return c.fail(resource, fmt.Errorf("sddc generatedruntime delegate is not configured"))
	}
	if sddcIdentityResolutionRequiresDisplayName(resource) {
		return c.fail(resource, fmt.Errorf("Sddc spec.displayName is required when no OCI identifier is recorded"))
	}
	if currentSddcID(resource) == "" {
		if resolver, ok := c.delegate.(sddcExistingResolver); ok {
			resolvedID, err := resolver.ResolveExisting(ctx, resource)
			if err != nil {
				return c.fail(resource, err)
			}
			recordResolvedSddcID(resource, resolvedID)
		}
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c *sddcGeneratedParityClient) Delete(ctx context.Context, resource *ocvpv1beta1.Sddc) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("sddc generatedruntime delegate is not configured")
	}
	return c.delegate.Delete(ctx, resource)
}

func (c *sddcGeneratedParityClient) fail(resource *ocvpv1beta1.Sddc, err error) (servicemanager.OSOKResponse, error) {
	if resource != nil {
		status := &resource.Status.OsokStatus
		status.Message = err.Error()
		status.Reason = string(shared.Failed)
		updatedAt := metav1.NewTime(time.Now())
		status.UpdatedAt = &updatedAt

		log := loggerutil.OSOKLogger{}
		if c.manager != nil {
			log = c.manager.Log
		}
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(
			resource.Status.OsokStatus,
			shared.Failed,
			v1.ConditionFalse,
			"",
			err.Error(),
			log,
		)
	}
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func sddcIdentityResolutionRequiresDisplayName(resource *ocvpv1beta1.Sddc) bool {
	if resource == nil {
		return false
	}
	if strings.TrimSpace(resource.Spec.DisplayName) != "" {
		return false
	}
	return currentSddcID(resource) == ""
}

func currentSddcID(resource *ocvpv1beta1.Sddc) string {
	if resource == nil {
		return ""
	}
	if ocid := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); ocid != "" {
		return ocid
	}
	return strings.TrimSpace(resource.Status.Id)
}

func recordResolvedSddcID(resource *ocvpv1beta1.Sddc, id string) {
	if resource == nil {
		return
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return
	}
	resource.Status.Id = id
	resource.Status.OsokStatus.Ocid = shared.OCID(id)
}

func sanitizeSddcResponse(body ocvpsdk.Sddc) (sanitizedSddcResponse, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return sanitizedSddcResponse{}, fmt.Errorf("marshal Sddc response body: %w", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return sanitizedSddcResponse{}, fmt.Errorf("decode Sddc response body: %w", err)
	}

	sanitized, ok := sanitizeJSONValue(decoded)
	if !ok {
		return sanitizedSddcResponse{}, nil
	}
	sanitizedMap, _ := sanitized.(map[string]any)
	return sanitizedSddcResponse{Body: sanitizedMap}, nil
}

func sanitizeJSONValue(value any) (any, bool) {
	switch concrete := value.(type) {
	case nil:
		return nil, false
	case map[string]any:
		cleaned := make(map[string]any, len(concrete))
		for key, child := range concrete {
			sanitizedChild, ok := sanitizeJSONValue(child)
			if !ok {
				continue
			}
			cleaned[key] = sanitizedChild
		}
		if len(cleaned) == 0 {
			return nil, false
		}
		return cleaned, true
	case []any:
		cleaned := make([]any, 0, len(concrete))
		for _, child := range concrete {
			sanitizedChild, ok := sanitizeJSONValue(child)
			if !ok {
				continue
			}
			cleaned = append(cleaned, sanitizedChild)
		}
		if len(cleaned) == 0 {
			return nil, false
		}
		return cleaned, true
	default:
		return concrete, true
	}
}

func isReusableSddcSummary(item ocvpsdk.SddcSummary, compartmentID string, displayName string) bool {
	if stringPointerValue(item.CompartmentId) != compartmentID {
		return false
	}
	if stringPointerValue(item.DisplayName) != displayName {
		return false
	}

	switch strings.TrimSpace(string(item.LifecycleState)) {
	case "ACTIVE", "CREATING", "UPDATING":
		return true
	default:
		return false
	}
}

func stringPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

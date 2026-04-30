package config

import (
	"context"
	"strings"
	"testing"

	apmconfigsdk "github.com/oracle/oci-go-sdk/v65/apmconfig"
	"github.com/oracle/oci-go-sdk/v65/common"
	apmconfigv1beta1 "github.com/oracle/oci-service-operator/api/apmconfig/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestBuildConfigCreateDetailsDecodesApdexRules(t *testing.T) {
	t.Parallel()

	resource := &apmconfigv1beta1.Config{
		Spec: apmconfigv1beta1.ConfigSpec{
			ApmDomainId: "ocid1.apmdomain.oc1..example",
			ConfigType:  "APDEX",
			DisplayName: "apdex-rules",
			Rules: []apmconfigv1beta1.ConfigRule{
				{
					FilterText: `service.name = "checkout"`,
					Priority:   1,
					IsEnabled:  true,
				},
			},
		},
	}

	details, err := buildConfigCreateDetails(context.Background(), resource, "default")
	if err != nil {
		t.Fatalf("buildConfigCreateDetails() error = %v", err)
	}

	apdexDetails, ok := details.(apmconfigsdk.CreateApdexRulesDetails)
	if !ok {
		t.Fatalf("buildConfigCreateDetails() type = %T, want apmconfig.CreateApdexRulesDetails", details)
	}
	if apdexDetails.DisplayName == nil || *apdexDetails.DisplayName != "apdex-rules" {
		t.Fatalf("DisplayName = %#v, want apdex-rules", apdexDetails.DisplayName)
	}
	if len(apdexDetails.Rules) != 1 {
		t.Fatalf("Rules length = %d, want 1", len(apdexDetails.Rules))
	}
	if apdexDetails.Rules[0].IsEnabled == nil || !*apdexDetails.Rules[0].IsEnabled {
		t.Fatalf("Rules[0].IsEnabled = %#v, want explicit true", apdexDetails.Rules[0].IsEnabled)
	}
}

func TestBuildConfigCreateDetailsRejectsFieldsFromOtherSubtype(t *testing.T) {
	t.Parallel()

	resource := &apmconfigv1beta1.Config{
		Spec: apmconfigv1beta1.ConfigSpec{
			ApmDomainId: "ocid1.apmdomain.oc1..example",
			ConfigType:  "SPAN_FILTER",
			DisplayName: "span-filter",
			FilterText:  `service.name = "checkout"`,
			Group:       "invalid-options-group",
		},
	}

	_, err := buildConfigCreateDetails(context.Background(), resource, "default")
	if err == nil {
		t.Fatal("buildConfigCreateDetails() error = nil, want subtype mismatch rejection")
	}
	if !strings.Contains(err.Error(), "spec.group") {
		t.Fatalf("buildConfigCreateDetails() error = %v, want spec.group rejection", err)
	}
}

func TestBuildConfigUpdateDetailsOptionsNoop(t *testing.T) {
	t.Parallel()

	resource := &apmconfigv1beta1.Config{
		Spec: apmconfigv1beta1.ConfigSpec{
			ApmDomainId: "ocid1.apmdomain.oc1..example",
			ConfigType:  "OPTIONS",
			DisplayName: "feature-flags",
			Group:       "checkout",
			Description: "feature toggles",
			Options:     rawJSONValue(`{"enabled":true}`),
		},
	}
	currentOptions := interface{}(map[string]any{"enabled": true})
	current := apmconfigsdk.GetConfigResponse{
		Config: apmconfigsdk.Options{
			DisplayName: common.String("feature-flags"),
			Group:       common.String("checkout"),
			Description: common.String("feature toggles"),
			Options:     &currentOptions,
		},
	}

	details, updateNeeded, err := buildConfigUpdateDetails(context.Background(), resource, "default", current)
	if err != nil {
		t.Fatalf("buildConfigUpdateDetails() error = %v", err)
	}
	if updateNeeded {
		t.Fatal("buildConfigUpdateDetails() updateNeeded = true, want false for matching OPTIONS payload")
	}
	if _, ok := details.(apmconfigsdk.UpdateOptionsDetails); !ok {
		t.Fatalf("buildConfigUpdateDetails() type = %T, want apmconfig.UpdateOptionsDetails", details)
	}
}

func TestBuildConfigUpdateDetailsSpanFilterDetectsDrift(t *testing.T) {
	t.Parallel()

	resource := &apmconfigv1beta1.Config{
		Spec: apmconfigv1beta1.ConfigSpec{
			ApmDomainId: "ocid1.apmdomain.oc1..example",
			ConfigType:  "SPAN_FILTER",
			DisplayName: "span-filter",
			FilterText:  `service.name = "checkout"`,
			Description: "updated filter",
		},
	}
	current := apmconfigsdk.GetConfigResponse{
		Config: apmconfigsdk.SpanFilter{
			DisplayName: common.String("span-filter"),
			FilterText:  common.String(`service.name = "legacy"`),
			Description: common.String("legacy filter"),
		},
	}

	details, updateNeeded, err := buildConfigUpdateDetails(context.Background(), resource, "default", current)
	if err != nil {
		t.Fatalf("buildConfigUpdateDetails() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildConfigUpdateDetails() updateNeeded = false, want true when span filter text drifts")
	}

	spanFilterDetails, ok := details.(apmconfigsdk.UpdateSpanFilterDetails)
	if !ok {
		t.Fatalf("buildConfigUpdateDetails() type = %T, want apmconfig.UpdateSpanFilterDetails", details)
	}
	if spanFilterDetails.FilterText == nil || *spanFilterDetails.FilterText != `service.name = "checkout"` {
		t.Fatalf("FilterText = %#v, want updated span filter text", spanFilterDetails.FilterText)
	}
	if spanFilterDetails.Description == nil || *spanFilterDetails.Description != "updated filter" {
		t.Fatalf("Description = %#v, want updated description", spanFilterDetails.Description)
	}
}

func TestGuardConfigExistingBeforeCreate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		resource *apmconfigv1beta1.Config
		want     generatedruntime.ExistingBeforeCreateDecision
		wantErr  string
	}{
		{
			name: "options skips without display name or group",
			resource: &apmconfigv1beta1.Config{
				Spec: apmconfigv1beta1.ConfigSpec{
					ApmDomainId: "ocid1.apmdomain.oc1..example",
					ConfigType:  "OPTIONS",
				},
			},
			want: generatedruntime.ExistingBeforeCreateDecisionSkip,
		},
		{
			name: "span filter allows with display name",
			resource: &apmconfigv1beta1.Config{
				Spec: apmconfigv1beta1.ConfigSpec{
					ApmDomainId: "ocid1.apmdomain.oc1..example",
					ConfigType:  "SPAN_FILTER",
					DisplayName: "span-filter",
				},
			},
			want: generatedruntime.ExistingBeforeCreateDecisionAllow,
		},
		{
			name: "missing domain fails",
			resource: &apmconfigv1beta1.Config{
				Spec: apmconfigv1beta1.ConfigSpec{
					ConfigType: "SPAN_FILTER",
				},
			},
			want:    generatedruntime.ExistingBeforeCreateDecisionFail,
			wantErr: "spec.apmDomainId is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := guardConfigExistingBeforeCreate(context.Background(), tt.resource)
			if got != tt.want {
				t.Fatalf("guardConfigExistingBeforeCreate() = %q, want %q", got, tt.want)
			}
			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatalf("guardConfigExistingBeforeCreate() error = %v, want nil", err)
			case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
				t.Fatalf("guardConfigExistingBeforeCreate() error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestConfigStatusMirrorClientMirrorsApmDomainID(t *testing.T) {
	t.Parallel()

	resource := &apmconfigv1beta1.Config{
		Spec: apmconfigv1beta1.ConfigSpec{
			ApmDomainId: "ocid1.apmdomain.oc1..example",
			ConfigType:  "SPAN_FILTER",
		},
	}
	client := wrapConfigStatusMirrorClient(stubConfigServiceClient{
		createOrUpdate: func(context.Context, *apmconfigv1beta1.Config, ctrl.Request) (servicemanager.OSOKResponse, error) {
			return servicemanager.OSOKResponse{IsSuccessful: true}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if resource.Status.ApmDomainId != "ocid1.apmdomain.oc1..example" {
		t.Fatalf("Status.ApmDomainId = %q, want mirrored domain", resource.Status.ApmDomainId)
	}
	if resource.Status.ConfigType != "SPAN_FILTER" {
		t.Fatalf("Status.ConfigType = %q, want mirrored config type", resource.Status.ConfigType)
	}
}

type stubConfigServiceClient struct {
	createOrUpdate func(context.Context, *apmconfigv1beta1.Config, ctrl.Request) (servicemanager.OSOKResponse, error)
	delete         func(context.Context, *apmconfigv1beta1.Config) (bool, error)
}

func (s stubConfigServiceClient) CreateOrUpdate(
	ctx context.Context,
	resource *apmconfigv1beta1.Config,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if s.createOrUpdate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: true}, nil
	}
	return s.createOrUpdate(ctx, resource, req)
}

func (s stubConfigServiceClient) Delete(ctx context.Context, resource *apmconfigv1beta1.Config) (bool, error) {
	if s.delete == nil {
		return true, nil
	}
	return s.delete(ctx, resource)
}

func rawJSONValue(raw string) shared.JSONValue {
	return shared.JSONValue{Raw: []byte(raw)}
}

func TestNewConfigServiceClientWithOCIClientAppliesStatusMirror(t *testing.T) {
	t.Parallel()

	resource := &apmconfigv1beta1.Config{
		Spec: apmconfigv1beta1.ConfigSpec{
			ApmDomainId: "ocid1.apmdomain.oc1..example",
			ConfigType:  "SPAN_FILTER",
			DisplayName: "span-filter",
			FilterText:  `service.name = "checkout"`,
		},
	}
	client := newConfigServiceClientWithOCIClient(
		loggerutil.OSOKLogger{},
		stubOCIConfigClient{
			create: func(context.Context, apmconfigsdk.CreateConfigRequest) (apmconfigsdk.CreateConfigResponse, error) {
				return apmconfigsdk.CreateConfigResponse{
					Config: apmconfigsdk.SpanFilter{
						Id:          common.String("ocid1.config.oc1..example"),
						DisplayName: common.String("span-filter"),
						FilterText:  common.String(`service.name = "checkout"`),
					},
				}, nil
			},
			get: func(context.Context, apmconfigsdk.GetConfigRequest) (apmconfigsdk.GetConfigResponse, error) {
				return apmconfigsdk.GetConfigResponse{
					Config: apmconfigsdk.SpanFilter{
						Id:          common.String("ocid1.config.oc1..example"),
						DisplayName: common.String("span-filter"),
						FilterText:  common.String(`service.name = "checkout"`),
					},
				}, nil
			},
		},
	)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if resource.Status.ApmDomainId != "ocid1.apmdomain.oc1..example" {
		t.Fatalf("Status.ApmDomainId = %q, want mirrored domain", resource.Status.ApmDomainId)
	}
}

type stubOCIConfigClient struct {
	create func(context.Context, apmconfigsdk.CreateConfigRequest) (apmconfigsdk.CreateConfigResponse, error)
	get    func(context.Context, apmconfigsdk.GetConfigRequest) (apmconfigsdk.GetConfigResponse, error)
	list   func(context.Context, apmconfigsdk.ListConfigsRequest) (apmconfigsdk.ListConfigsResponse, error)
	update func(context.Context, apmconfigsdk.UpdateConfigRequest) (apmconfigsdk.UpdateConfigResponse, error)
	delete func(context.Context, apmconfigsdk.DeleteConfigRequest) (apmconfigsdk.DeleteConfigResponse, error)
}

func (s stubOCIConfigClient) CreateConfig(ctx context.Context, req apmconfigsdk.CreateConfigRequest) (apmconfigsdk.CreateConfigResponse, error) {
	return s.create(ctx, req)
}

func (s stubOCIConfigClient) GetConfig(ctx context.Context, req apmconfigsdk.GetConfigRequest) (apmconfigsdk.GetConfigResponse, error) {
	return s.get(ctx, req)
}

func (s stubOCIConfigClient) ListConfigs(ctx context.Context, req apmconfigsdk.ListConfigsRequest) (apmconfigsdk.ListConfigsResponse, error) {
	if s.list == nil {
		return apmconfigsdk.ListConfigsResponse{}, nil
	}
	return s.list(ctx, req)
}

func (s stubOCIConfigClient) UpdateConfig(ctx context.Context, req apmconfigsdk.UpdateConfigRequest) (apmconfigsdk.UpdateConfigResponse, error) {
	if s.update == nil {
		return apmconfigsdk.UpdateConfigResponse{}, nil
	}
	return s.update(ctx, req)
}

func (s stubOCIConfigClient) DeleteConfig(ctx context.Context, req apmconfigsdk.DeleteConfigRequest) (apmconfigsdk.DeleteConfigResponse, error) {
	if s.delete == nil {
		return apmconfigsdk.DeleteConfigResponse{}, nil
	}
	return s.delete(ctx, req)
}

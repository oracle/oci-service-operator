/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package kafkaclusterconfig

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	managedkafkasdk "github.com/oracle/oci-go-sdk/v65/managedkafka"
	managedkafkav1beta1 "github.com/oracle/oci-service-operator/api/managedkafka/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testKafkaClusterConfigID            = "ocid1.kafkaclusterconfig.oc1..config"
	testKafkaClusterConfigOtherID       = "ocid1.kafkaclusterconfig.oc1..other"
	testKafkaClusterConfigCompartmentID = "ocid1.compartment.oc1..kafka"
	testKafkaClusterConfigDisplayName   = "kafka-config-sample"
)

type fakeKafkaClusterConfigOCIClient struct {
	createFn func(context.Context, managedkafkasdk.CreateKafkaClusterConfigRequest) (managedkafkasdk.CreateKafkaClusterConfigResponse, error)
	getFn    func(context.Context, managedkafkasdk.GetKafkaClusterConfigRequest) (managedkafkasdk.GetKafkaClusterConfigResponse, error)
	listFn   func(context.Context, managedkafkasdk.ListKafkaClusterConfigsRequest) (managedkafkasdk.ListKafkaClusterConfigsResponse, error)
	updateFn func(context.Context, managedkafkasdk.UpdateKafkaClusterConfigRequest) (managedkafkasdk.UpdateKafkaClusterConfigResponse, error)
	deleteFn func(context.Context, managedkafkasdk.DeleteKafkaClusterConfigRequest) (managedkafkasdk.DeleteKafkaClusterConfigResponse, error)
}

func (f *fakeKafkaClusterConfigOCIClient) CreateKafkaClusterConfig(
	ctx context.Context,
	request managedkafkasdk.CreateKafkaClusterConfigRequest,
) (managedkafkasdk.CreateKafkaClusterConfigResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return managedkafkasdk.CreateKafkaClusterConfigResponse{}, nil
}

func (f *fakeKafkaClusterConfigOCIClient) GetKafkaClusterConfig(
	ctx context.Context,
	request managedkafkasdk.GetKafkaClusterConfigRequest,
) (managedkafkasdk.GetKafkaClusterConfigResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return managedkafkasdk.GetKafkaClusterConfigResponse{}, nil
}

func (f *fakeKafkaClusterConfigOCIClient) ListKafkaClusterConfigs(
	ctx context.Context,
	request managedkafkasdk.ListKafkaClusterConfigsRequest,
) (managedkafkasdk.ListKafkaClusterConfigsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return managedkafkasdk.ListKafkaClusterConfigsResponse{}, nil
}

func (f *fakeKafkaClusterConfigOCIClient) UpdateKafkaClusterConfig(
	ctx context.Context,
	request managedkafkasdk.UpdateKafkaClusterConfigRequest,
) (managedkafkasdk.UpdateKafkaClusterConfigResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return managedkafkasdk.UpdateKafkaClusterConfigResponse{}, nil
}

func (f *fakeKafkaClusterConfigOCIClient) DeleteKafkaClusterConfig(
	ctx context.Context,
	request managedkafkasdk.DeleteKafkaClusterConfigRequest,
) (managedkafkasdk.DeleteKafkaClusterConfigResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return managedkafkasdk.DeleteKafkaClusterConfigResponse{}, nil
}

func testKafkaClusterConfigClient(fake *fakeKafkaClusterConfigOCIClient) KafkaClusterConfigServiceClient {
	return newKafkaClusterConfigServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)
}

type createKafkaClusterConfigObservation struct {
	listCalls     int
	createCalls   int
	getCalls      int
	listRequest   managedkafkasdk.ListKafkaClusterConfigsRequest
	createRequest managedkafkasdk.CreateKafkaClusterConfigRequest
	getRequest    managedkafkasdk.GetKafkaClusterConfigRequest
}

type mutableKafkaClusterConfigUpdateObservation struct {
	getCalls      int
	updateCalls   int
	updateRequest managedkafkasdk.UpdateKafkaClusterConfigRequest
}

func makeKafkaClusterConfigResource() *managedkafkav1beta1.KafkaClusterConfig {
	return &managedkafkav1beta1.KafkaClusterConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kafka-config",
			Namespace: "default",
			UID:       types.UID("kafka-config-uid"),
		},
		Spec: managedkafkav1beta1.KafkaClusterConfigSpec{
			CompartmentId: testKafkaClusterConfigCompartmentID,
			DisplayName:   testKafkaClusterConfigDisplayName,
			LatestConfig: managedkafkav1beta1.KafkaClusterConfigLatestConfig{
				Properties: map[string]string{"auto.create.topics.enable": "false"},
			},
			FreeformTags: map[string]string{"env": "dev"},
			DefinedTags:  map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func makeSDKKafkaClusterConfig(
	id string,
	compartmentID string,
	displayName string,
	properties map[string]string,
	state managedkafkasdk.KafkaClusterConfigLifecycleStateEnum,
) managedkafkasdk.KafkaClusterConfig {
	created := common.SDKTime{Time: time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)}
	updated := common.SDKTime{Time: time.Date(2026, 4, 28, 13, 0, 0, 0, time.UTC)}
	version := 3
	return managedkafkasdk.KafkaClusterConfig{
		Id:             common.String(id),
		CompartmentId:  common.String(compartmentID),
		DisplayName:    common.String(displayName),
		LifecycleState: state,
		TimeCreated:    &created,
		TimeUpdated:    &updated,
		LatestConfig: &managedkafkasdk.KafkaClusterConfigVersion{
			Properties:    mapsClone(properties),
			ConfigId:      common.String(id),
			VersionNumber: common.Int(version),
			TimeCreated:   &created,
		},
		FreeformTags: map[string]string{"env": "dev"},
		DefinedTags:  map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		SystemTags:   map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "true"}},
	}
}

func makeSDKKafkaClusterConfigSummary(
	id string,
	compartmentID string,
	displayName string,
	state managedkafkasdk.KafkaClusterConfigLifecycleStateEnum,
) managedkafkasdk.KafkaClusterConfigSummary {
	created := common.SDKTime{Time: time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)}
	return managedkafkasdk.KafkaClusterConfigSummary{
		Id:             common.String(id),
		CompartmentId:  common.String(compartmentID),
		DisplayName:    common.String(displayName),
		LifecycleState: state,
		TimeCreated:    &created,
		FreeformTags:   map[string]string{"env": "dev"},
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	}
}

func testKafkaClusterConfigCreateProjectionClient(
	t *testing.T,
	resource *managedkafkav1beta1.KafkaClusterConfig,
	observed *createKafkaClusterConfigObservation,
) KafkaClusterConfigServiceClient {
	t.Helper()
	return testKafkaClusterConfigClient(&fakeKafkaClusterConfigOCIClient{
		listFn: func(_ context.Context, request managedkafkasdk.ListKafkaClusterConfigsRequest) (managedkafkasdk.ListKafkaClusterConfigsResponse, error) {
			observed.listCalls++
			observed.listRequest = request
			return managedkafkasdk.ListKafkaClusterConfigsResponse{
				KafkaClusterConfigCollection: managedkafkasdk.KafkaClusterConfigCollection{},
				OpcRequestId:                 common.String("opc-list"),
			}, nil
		},
		createFn: func(_ context.Context, request managedkafkasdk.CreateKafkaClusterConfigRequest) (managedkafkasdk.CreateKafkaClusterConfigResponse, error) {
			observed.createCalls++
			observed.createRequest = request
			return managedkafkasdk.CreateKafkaClusterConfigResponse{
				KafkaClusterConfig: makeSDKKafkaClusterConfig(
					testKafkaClusterConfigID,
					testKafkaClusterConfigCompartmentID,
					testKafkaClusterConfigDisplayName,
					resource.Spec.LatestConfig.Properties,
					managedkafkasdk.KafkaClusterConfigLifecycleStateActive,
				),
				OpcRequestId: common.String("opc-create"),
			}, nil
		},
		getFn: func(_ context.Context, request managedkafkasdk.GetKafkaClusterConfigRequest) (managedkafkasdk.GetKafkaClusterConfigResponse, error) {
			observed.getCalls++
			observed.getRequest = request
			return managedkafkasdk.GetKafkaClusterConfigResponse{
				KafkaClusterConfig: makeSDKKafkaClusterConfig(
					testKafkaClusterConfigID,
					testKafkaClusterConfigCompartmentID,
					testKafkaClusterConfigDisplayName,
					resource.Spec.LatestConfig.Properties,
					managedkafkasdk.KafkaClusterConfigLifecycleStateActive,
				),
			}, nil
		},
	})
}

func configureMutableKafkaClusterConfigUpdate(resource *managedkafkav1beta1.KafkaClusterConfig) {
	resource.Status.OsokStatus.Ocid = shared.OCID(testKafkaClusterConfigID)
	resource.Spec.DisplayName = "updated-config"
	resource.Spec.LatestConfig.Properties = map[string]string{"auto.create.topics.enable": "true"}
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "84"}}
}

func testKafkaClusterConfigMutableUpdateClient(
	t *testing.T,
	resource *managedkafkav1beta1.KafkaClusterConfig,
	observed *mutableKafkaClusterConfigUpdateObservation,
) KafkaClusterConfigServiceClient {
	t.Helper()
	return testKafkaClusterConfigClient(&fakeKafkaClusterConfigOCIClient{
		getFn: func(_ context.Context, request managedkafkasdk.GetKafkaClusterConfigRequest) (managedkafkasdk.GetKafkaClusterConfigResponse, error) {
			observed.getCalls++
			requireStringPtr(t, "get kafkaClusterConfigId", request.KafkaClusterConfigId, testKafkaClusterConfigID)
			current := currentKafkaClusterConfigForMutableUpdate(observed.getCalls, resource)
			return managedkafkasdk.GetKafkaClusterConfigResponse{KafkaClusterConfig: current}, nil
		},
		updateFn: func(_ context.Context, request managedkafkasdk.UpdateKafkaClusterConfigRequest) (managedkafkasdk.UpdateKafkaClusterConfigResponse, error) {
			observed.updateCalls++
			observed.updateRequest = request
			return managedkafkasdk.UpdateKafkaClusterConfigResponse{
				KafkaClusterConfig: makeSDKKafkaClusterConfig(
					testKafkaClusterConfigID,
					testKafkaClusterConfigCompartmentID,
					resource.Spec.DisplayName,
					resource.Spec.LatestConfig.Properties,
					managedkafkasdk.KafkaClusterConfigLifecycleStateActive,
				),
				OpcRequestId: common.String("opc-update"),
			}, nil
		},
	})
}

func currentKafkaClusterConfigForMutableUpdate(
	getCalls int,
	resource *managedkafkav1beta1.KafkaClusterConfig,
) managedkafkasdk.KafkaClusterConfig {
	current := makeSDKKafkaClusterConfig(
		testKafkaClusterConfigID,
		testKafkaClusterConfigCompartmentID,
		testKafkaClusterConfigDisplayName,
		map[string]string{"auto.create.topics.enable": "false"},
		managedkafkasdk.KafkaClusterConfigLifecycleStateActive,
	)
	if getCalls <= 1 {
		return current
	}
	current.DisplayName = common.String(resource.Spec.DisplayName)
	current.LatestConfig = &managedkafkasdk.KafkaClusterConfigVersion{
		Properties: map[string]string{"auto.create.topics.enable": "true"},
	}
	current.FreeformTags = map[string]string{"env": "prod"}
	current.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "84"}}
	return current
}

func TestKafkaClusterConfigRuntimeSemantics(t *testing.T) {
	t.Parallel()

	got := newKafkaClusterConfigRuntimeSemantics()
	if got == nil {
		t.Fatal("newKafkaClusterConfigRuntimeSemantics() = nil")
	}
	if got.Async == nil || got.Async.Strategy != "lifecycle" || got.Async.Runtime != "generatedruntime" {
		t.Fatalf("async semantics = %#v, want generatedruntime lifecycle", got.Async)
	}
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" || got.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("delete semantics = %#v followUp=%#v, want required confirm-delete", got.Delete, got.DeleteFollowUp)
	}
	requireStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"compartmentId", "displayName", "id"})
	requireStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"displayName", "latestConfig.properties", "freeformTags", "definedTags"})
	requireStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"compartmentId"})
}

func TestKafkaClusterConfigServiceClientCreatesAndProjectsStatus(t *testing.T) {
	t.Parallel()

	resource := makeKafkaClusterConfigResource()
	observed := &createKafkaClusterConfigObservation{}
	client := testKafkaClusterConfigCreateProjectionClient(t, resource, observed)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireSuccessfulCreateOrUpdate(t, response, err)
	requireKafkaClusterConfigCreateProjection(t, observed, resource)
	requireCreatedKafkaClusterConfigStatus(t, resource)
}

func TestKafkaClusterConfigServiceClientBindsFromPaginatedList(t *testing.T) {
	t.Parallel()

	resource := makeKafkaClusterConfigResource()
	listCalls := 0
	getCalls := 0
	var pages []string

	client := testKafkaClusterConfigClient(&fakeKafkaClusterConfigOCIClient{
		listFn: func(_ context.Context, request managedkafkasdk.ListKafkaClusterConfigsRequest) (managedkafkasdk.ListKafkaClusterConfigsResponse, error) {
			listCalls++
			pages = append(pages, stringValue(request.Page))
			if listCalls == 1 {
				return managedkafkasdk.ListKafkaClusterConfigsResponse{
					KafkaClusterConfigCollection: managedkafkasdk.KafkaClusterConfigCollection{
						Items: []managedkafkasdk.KafkaClusterConfigSummary{
							makeSDKKafkaClusterConfigSummary(
								testKafkaClusterConfigOtherID,
								testKafkaClusterConfigCompartmentID,
								"other-config",
								managedkafkasdk.KafkaClusterConfigLifecycleStateActive,
							),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			return managedkafkasdk.ListKafkaClusterConfigsResponse{
				KafkaClusterConfigCollection: managedkafkasdk.KafkaClusterConfigCollection{
					Items: []managedkafkasdk.KafkaClusterConfigSummary{
						makeSDKKafkaClusterConfigSummary(
							testKafkaClusterConfigID,
							testKafkaClusterConfigCompartmentID,
							testKafkaClusterConfigDisplayName,
							managedkafkasdk.KafkaClusterConfigLifecycleStateActive,
						),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, request managedkafkasdk.GetKafkaClusterConfigRequest) (managedkafkasdk.GetKafkaClusterConfigResponse, error) {
			getCalls++
			requireStringPtr(t, "get kafkaClusterConfigId", request.KafkaClusterConfigId, testKafkaClusterConfigID)
			return managedkafkasdk.GetKafkaClusterConfigResponse{
				KafkaClusterConfig: makeSDKKafkaClusterConfig(
					testKafkaClusterConfigID,
					testKafkaClusterConfigCompartmentID,
					testKafkaClusterConfigDisplayName,
					resource.Spec.LatestConfig.Properties,
					managedkafkasdk.KafkaClusterConfigLifecycleStateActive,
				),
			}, nil
		},
		createFn: func(context.Context, managedkafkasdk.CreateKafkaClusterConfigRequest) (managedkafkasdk.CreateKafkaClusterConfigResponse, error) {
			t.Fatal("CreateKafkaClusterConfig() called for existing config")
			return managedkafkasdk.CreateKafkaClusterConfigResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if listCalls != 2 || getCalls != 1 {
		t.Fatalf("call counts list/get = %d/%d, want 2/1", listCalls, getCalls)
	}
	if want := []string{"", "page-2"}; !reflect.DeepEqual(pages, want) {
		t.Fatalf("list pages = %#v, want %#v", pages, want)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testKafkaClusterConfigID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testKafkaClusterConfigID)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestKafkaClusterConfigServiceClientRejectsDuplicateMatchesAcrossPages(t *testing.T) {
	t.Parallel()

	resource := makeKafkaClusterConfigResource()
	listCalls := 0
	client := testKafkaClusterConfigClient(&fakeKafkaClusterConfigOCIClient{
		listFn: func(_ context.Context, request managedkafkasdk.ListKafkaClusterConfigsRequest) (managedkafkasdk.ListKafkaClusterConfigsResponse, error) {
			listCalls++
			if listCalls == 1 {
				return managedkafkasdk.ListKafkaClusterConfigsResponse{
					KafkaClusterConfigCollection: managedkafkasdk.KafkaClusterConfigCollection{
						Items: []managedkafkasdk.KafkaClusterConfigSummary{
							makeSDKKafkaClusterConfigSummary(
								testKafkaClusterConfigID,
								testKafkaClusterConfigCompartmentID,
								testKafkaClusterConfigDisplayName,
								managedkafkasdk.KafkaClusterConfigLifecycleStateActive,
							),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			requireStringPtr(t, "second list page", request.Page, "page-2")
			return managedkafkasdk.ListKafkaClusterConfigsResponse{
				KafkaClusterConfigCollection: managedkafkasdk.KafkaClusterConfigCollection{
					Items: []managedkafkasdk.KafkaClusterConfigSummary{
						makeSDKKafkaClusterConfigSummary(
							testKafkaClusterConfigOtherID,
							testKafkaClusterConfigCompartmentID,
							testKafkaClusterConfigDisplayName,
							managedkafkasdk.KafkaClusterConfigLifecycleStateActive,
						),
					},
				},
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "multiple matching resources") {
		t.Fatalf("CreateOrUpdate() error = %v, want duplicate match error", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if listCalls != 2 {
		t.Fatalf("ListKafkaClusterConfigs calls = %d, want 2", listCalls)
	}
}

func TestKafkaClusterConfigServiceClientNoopsWhenObservedStateMatches(t *testing.T) {
	t.Parallel()

	resource := makeKafkaClusterConfigResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testKafkaClusterConfigID)
	getCalls := 0
	updateCalls := 0

	client := testKafkaClusterConfigClient(&fakeKafkaClusterConfigOCIClient{
		getFn: func(_ context.Context, request managedkafkasdk.GetKafkaClusterConfigRequest) (managedkafkasdk.GetKafkaClusterConfigResponse, error) {
			getCalls++
			requireStringPtr(t, "get kafkaClusterConfigId", request.KafkaClusterConfigId, testKafkaClusterConfigID)
			return managedkafkasdk.GetKafkaClusterConfigResponse{
				KafkaClusterConfig: makeSDKKafkaClusterConfig(
					testKafkaClusterConfigID,
					testKafkaClusterConfigCompartmentID,
					testKafkaClusterConfigDisplayName,
					resource.Spec.LatestConfig.Properties,
					managedkafkasdk.KafkaClusterConfigLifecycleStateActive,
				),
			}, nil
		},
		updateFn: func(context.Context, managedkafkasdk.UpdateKafkaClusterConfigRequest) (managedkafkasdk.UpdateKafkaClusterConfigResponse, error) {
			updateCalls++
			return managedkafkasdk.UpdateKafkaClusterConfigResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if getCalls != 1 || updateCalls != 0 {
		t.Fatalf("call counts get/update = %d/%d, want 1/0", getCalls, updateCalls)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestKafkaClusterConfigServiceClientOmitsBlankDisplayNameOnUpdate(t *testing.T) {
	t.Parallel()

	resource := makeKafkaClusterConfigResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testKafkaClusterConfigID)
	resource.Spec.DisplayName = ""
	getCalls := 0
	updateCalls := 0

	client := testKafkaClusterConfigClient(&fakeKafkaClusterConfigOCIClient{
		getFn: func(_ context.Context, request managedkafkasdk.GetKafkaClusterConfigRequest) (managedkafkasdk.GetKafkaClusterConfigResponse, error) {
			getCalls++
			requireStringPtr(t, "get kafkaClusterConfigId", request.KafkaClusterConfigId, testKafkaClusterConfigID)
			return managedkafkasdk.GetKafkaClusterConfigResponse{
				KafkaClusterConfig: makeSDKKafkaClusterConfig(
					testKafkaClusterConfigID,
					testKafkaClusterConfigCompartmentID,
					testKafkaClusterConfigDisplayName,
					resource.Spec.LatestConfig.Properties,
					managedkafkasdk.KafkaClusterConfigLifecycleStateActive,
				),
			}, nil
		},
		updateFn: func(context.Context, managedkafkasdk.UpdateKafkaClusterConfigRequest) (managedkafkasdk.UpdateKafkaClusterConfigResponse, error) {
			updateCalls++
			return managedkafkasdk.UpdateKafkaClusterConfigResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if getCalls != 1 || updateCalls != 0 {
		t.Fatalf("call counts get/update = %d/%d, want 1/0", getCalls, updateCalls)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestKafkaClusterConfigServiceClientUpdatesMutableFields(t *testing.T) {
	t.Parallel()

	resource := makeKafkaClusterConfigResource()
	configureMutableKafkaClusterConfigUpdate(resource)
	observed := &mutableKafkaClusterConfigUpdateObservation{}
	client := testKafkaClusterConfigMutableUpdateClient(t, resource, observed)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireSuccessfulCreateOrUpdate(t, response, err)
	requireKafkaClusterConfigMutableUpdate(t, observed, resource)
	if resource.Status.OsokStatus.OpcRequestID != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", resource.Status.OsokStatus.OpcRequestID)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestKafkaClusterConfigServiceClientRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := makeKafkaClusterConfigResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testKafkaClusterConfigID)
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..moved"
	updateCalls := 0
	client := testKafkaClusterConfigClient(&fakeKafkaClusterConfigOCIClient{
		getFn: func(_ context.Context, request managedkafkasdk.GetKafkaClusterConfigRequest) (managedkafkasdk.GetKafkaClusterConfigResponse, error) {
			requireStringPtr(t, "get kafkaClusterConfigId", request.KafkaClusterConfigId, testKafkaClusterConfigID)
			return managedkafkasdk.GetKafkaClusterConfigResponse{
				KafkaClusterConfig: makeSDKKafkaClusterConfig(
					testKafkaClusterConfigID,
					testKafkaClusterConfigCompartmentID,
					testKafkaClusterConfigDisplayName,
					resource.Spec.LatestConfig.Properties,
					managedkafkasdk.KafkaClusterConfigLifecycleStateActive,
				),
			}, nil
		},
		updateFn: func(context.Context, managedkafkasdk.UpdateKafkaClusterConfigRequest) (managedkafkasdk.UpdateKafkaClusterConfigResponse, error) {
			updateCalls++
			return managedkafkasdk.UpdateKafkaClusterConfigResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "compartmentId") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartmentId drift rejection", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateKafkaClusterConfig calls = %d, want 0", updateCalls)
	}
	requireLastCondition(t, resource, shared.Failed)
}

func TestKafkaClusterConfigServiceClientRejectsServiceAssignedLatestConfigSpecFields(t *testing.T) {
	t.Parallel()

	resource := makeKafkaClusterConfigResource()
	resource.Spec.LatestConfig.ConfigId = "ocid1.kafkaclusterconfig.oc1..readback"
	client := testKafkaClusterConfigClient(&fakeKafkaClusterConfigOCIClient{
		createFn: func(context.Context, managedkafkasdk.CreateKafkaClusterConfigRequest) (managedkafkasdk.CreateKafkaClusterConfigResponse, error) {
			t.Fatal("CreateKafkaClusterConfig() called with service-assigned latestConfig fields")
			return managedkafkasdk.CreateKafkaClusterConfigResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "latestConfig.configId") {
		t.Fatalf("CreateOrUpdate() error = %v, want latestConfig.configId validation error", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	requireLastCondition(t, resource, shared.Failed)
}

func TestKafkaClusterConfigServiceClientDeleteWaitsForConfirmedDeletion(t *testing.T) {
	t.Parallel()

	resource := makeKafkaClusterConfigResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testKafkaClusterConfigID)
	getCalls := 0
	deleteCalls := 0
	var deleteRequest managedkafkasdk.DeleteKafkaClusterConfigRequest
	client := testKafkaClusterConfigClient(&fakeKafkaClusterConfigOCIClient{
		getFn: func(_ context.Context, request managedkafkasdk.GetKafkaClusterConfigRequest) (managedkafkasdk.GetKafkaClusterConfigResponse, error) {
			getCalls++
			requireStringPtr(t, "get kafkaClusterConfigId", request.KafkaClusterConfigId, testKafkaClusterConfigID)
			state := managedkafkasdk.KafkaClusterConfigLifecycleStateActive
			if getCalls > 1 {
				state = managedkafkasdk.KafkaClusterConfigLifecycleStateDeleted
			}
			return managedkafkasdk.GetKafkaClusterConfigResponse{
				KafkaClusterConfig: makeSDKKafkaClusterConfig(
					testKafkaClusterConfigID,
					testKafkaClusterConfigCompartmentID,
					testKafkaClusterConfigDisplayName,
					resource.Spec.LatestConfig.Properties,
					state,
				),
			}, nil
		},
		deleteFn: func(_ context.Context, request managedkafkasdk.DeleteKafkaClusterConfigRequest) (managedkafkasdk.DeleteKafkaClusterConfigResponse, error) {
			deleteCalls++
			deleteRequest = request
			return managedkafkasdk.DeleteKafkaClusterConfigResponse{
				OpcRequestId: common.String("opc-delete"),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after DELETED readback")
	}
	if getCalls != 2 || deleteCalls != 1 {
		t.Fatalf("call counts get/delete = %d/%d, want 2/1", getCalls, deleteCalls)
	}
	requireStringPtr(t, "delete kafkaClusterConfigId", deleteRequest.KafkaClusterConfigId, testKafkaClusterConfigID)
	if resource.Status.OsokStatus.OpcRequestID != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", resource.Status.OsokStatus.OpcRequestID)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
	requireLastCondition(t, resource, shared.Terminating)
}

func TestKafkaClusterConfigServiceClientKeepsFinalizerOnAuthShapedDelete404(t *testing.T) {
	t.Parallel()

	resource := makeKafkaClusterConfigResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testKafkaClusterConfigID)
	client := testKafkaClusterConfigClient(&fakeKafkaClusterConfigOCIClient{
		getFn: func(_ context.Context, request managedkafkasdk.GetKafkaClusterConfigRequest) (managedkafkasdk.GetKafkaClusterConfigResponse, error) {
			requireStringPtr(t, "get kafkaClusterConfigId", request.KafkaClusterConfigId, testKafkaClusterConfigID)
			return managedkafkasdk.GetKafkaClusterConfigResponse{
				KafkaClusterConfig: makeSDKKafkaClusterConfig(
					testKafkaClusterConfigID,
					testKafkaClusterConfigCompartmentID,
					testKafkaClusterConfigDisplayName,
					resource.Spec.LatestConfig.Properties,
					managedkafkasdk.KafkaClusterConfigLifecycleStateActive,
				),
			}, nil
		},
		deleteFn: func(context.Context, managedkafkasdk.DeleteKafkaClusterConfigRequest) (managedkafkasdk.DeleteKafkaClusterConfigResponse, error) {
			return managedkafkasdk.DeleteKafkaClusterConfigResponse{}, errortest.NewServiceError(
				404,
				errorutil.NotAuthorizedOrNotFound,
				"not authorized or not found",
			)
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404") {
		t.Fatalf("Delete() error = %v, want ambiguous 404 error", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %#v, want nil", resource.Status.OsokStatus.DeletedAt)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want propagated request ID", resource.Status.OsokStatus.OpcRequestID)
	}
}

func requireSuccessfulCreateOrUpdate(
	t *testing.T,
	response servicemanager.OSOKResponse,
	err error,
) {
	t.Helper()
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
}

func requireKafkaClusterConfigCreateProjection(
	t *testing.T,
	observed *createKafkaClusterConfigObservation,
	resource *managedkafkav1beta1.KafkaClusterConfig,
) {
	t.Helper()
	if observed.listCalls != 1 || observed.createCalls != 1 || observed.getCalls != 1 {
		t.Fatalf("call counts list/create/get = %d/%d/%d, want 1/1/1", observed.listCalls, observed.createCalls, observed.getCalls)
	}
	requireStringPtr(t, "list compartmentId", observed.listRequest.CompartmentId, testKafkaClusterConfigCompartmentID)
	requireStringPtr(t, "list displayName", observed.listRequest.DisplayName, testKafkaClusterConfigDisplayName)
	requireStringPtr(t, "create retry token", observed.createRequest.OpcRetryToken, string(resource.UID))
	requireStringPtr(t, "create compartmentId", observed.createRequest.CompartmentId, testKafkaClusterConfigCompartmentID)
	requireStringPtr(t, "create displayName", observed.createRequest.DisplayName, testKafkaClusterConfigDisplayName)
	requireKafkaClusterConfigVersionRequest(t, "create", observed.createRequest.LatestConfig, resource.Spec.LatestConfig.Properties)
	requireStringPtr(t, "get kafkaClusterConfigId", observed.getRequest.KafkaClusterConfigId, testKafkaClusterConfigID)
}

func requireKafkaClusterConfigMutableUpdate(
	t *testing.T,
	observed *mutableKafkaClusterConfigUpdateObservation,
	resource *managedkafkav1beta1.KafkaClusterConfig,
) {
	t.Helper()
	if observed.getCalls != 2 || observed.updateCalls != 1 {
		t.Fatalf("call counts get/update = %d/%d, want 2/1", observed.getCalls, observed.updateCalls)
	}
	requireStringPtr(t, "update kafkaClusterConfigId", observed.updateRequest.KafkaClusterConfigId, testKafkaClusterConfigID)
	requireStringPtr(t, "update displayName", observed.updateRequest.DisplayName, resource.Spec.DisplayName)
	requireKafkaClusterConfigVersionRequest(t, "update", observed.updateRequest.LatestConfig, resource.Spec.LatestConfig.Properties)
	if !reflect.DeepEqual(observed.updateRequest.FreeformTags, map[string]string{"env": "prod"}) {
		t.Fatalf("update freeformTags = %#v, want env=prod", observed.updateRequest.FreeformTags)
	}
}

func requireKafkaClusterConfigVersionRequest(
	t *testing.T,
	operation string,
	got *managedkafkasdk.KafkaClusterConfigVersion,
	wantProperties map[string]string,
) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s latestConfig = nil, want properties %#v", operation, wantProperties)
	}
	if !reflect.DeepEqual(got.Properties, wantProperties) {
		t.Fatalf("%s latestConfig.properties = %#v, want %#v", operation, got, wantProperties)
	}
	if got.ConfigId != nil || got.VersionNumber != nil || got.TimeCreated != nil {
		t.Fatalf("%s latestConfig includes service-assigned fields: %#v", operation, got)
	}
}

func requireCreatedKafkaClusterConfigStatus(t *testing.T, resource *managedkafkav1beta1.KafkaClusterConfig) {
	t.Helper()
	if got := string(resource.Status.OsokStatus.Ocid); got != testKafkaClusterConfigID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testKafkaClusterConfigID)
	}
	if resource.Status.Id != testKafkaClusterConfigID {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, testKafkaClusterConfigID)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", resource.Status.OsokStatus.OpcRequestID)
	}
	if resource.Status.LifecycleState != string(managedkafkasdk.KafkaClusterConfigLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", resource.Status.LifecycleState)
	}
	if resource.Status.LatestConfig.Properties["auto.create.topics.enable"] != "false" {
		t.Fatalf("status.latestConfig.properties = %#v, want projected properties", resource.Status.LatestConfig.Properties)
	}
	if resource.Status.LatestConfig.ConfigId != testKafkaClusterConfigID {
		t.Fatalf("status.latestConfig.configId = %q, want %q", resource.Status.LatestConfig.ConfigId, testKafkaClusterConfigID)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.status.async.current = %#v, want nil for ACTIVE lifecycle", resource.Status.OsokStatus.Async.Current)
	}
	requireLastCondition(t, resource, shared.Active)
}

func requireLastCondition(
	t *testing.T,
	resource *managedkafkav1beta1.KafkaClusterConfig,
	want shared.OSOKConditionType,
) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.status.conditions = nil, want trailing %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last condition = %q, want %q", got, want)
	}
}

func requireStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", name, *got, want)
	}
}

func requireStringSliceEqual(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func mapsClone(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

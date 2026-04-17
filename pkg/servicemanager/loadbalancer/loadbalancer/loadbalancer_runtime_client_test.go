/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package loadbalancer

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	loadbalancersdk "github.com/oracle/oci-go-sdk/v65/loadbalancer"
	loadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/loadbalancer/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	loadBalancerCompartmentID = "ocid1.compartment.oc1..exampleuniqueID"
	loadBalancerCreatedID     = "ocid1.loadbalancer.oc1..createduniqueID"
	loadBalancerExistingID    = "ocid1.loadbalancer.oc1..existinguniqueID"
	loadBalancerDisplayName   = "example_load_balancer"
	loadBalancerSubnetID      = "ocid1.subnet.oc1..exampleuniqueID"
)

type fakeGeneratedLoadBalancerOCIClient struct {
	createRequests []loadbalancersdk.CreateLoadBalancerRequest
	getRequests    []loadbalancersdk.GetLoadBalancerRequest
	listRequests   []loadbalancersdk.ListLoadBalancersRequest
	updateRequests []loadbalancersdk.UpdateLoadBalancerRequest
	deleteRequests []loadbalancersdk.DeleteLoadBalancerRequest

	createErr error
	getErr    error
	listErr   error
	updateErr error
	deleteErr error

	loadBalancers map[string]loadbalancersdk.LoadBalancer
}

func (f *fakeGeneratedLoadBalancerOCIClient) CreateLoadBalancer(_ context.Context, request loadbalancersdk.CreateLoadBalancerRequest) (loadbalancersdk.CreateLoadBalancerResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createErr != nil {
		return loadbalancersdk.CreateLoadBalancerResponse{}, f.createErr
	}
	if f.loadBalancers == nil {
		f.loadBalancers = map[string]loadbalancersdk.LoadBalancer{}
	}
	f.loadBalancers[loadBalancerCreatedID] = loadBalancerFromCreateDetails(loadBalancerCreatedID, request.CreateLoadBalancerDetails)
	return loadbalancersdk.CreateLoadBalancerResponse{}, nil
}

func (f *fakeGeneratedLoadBalancerOCIClient) GetLoadBalancer(_ context.Context, request loadbalancersdk.GetLoadBalancerRequest) (loadbalancersdk.GetLoadBalancerResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getErr != nil {
		return loadbalancersdk.GetLoadBalancerResponse{}, f.getErr
	}

	resource, ok := f.loadBalancers[stringValue(request.LoadBalancerId)]
	if !ok {
		return loadbalancersdk.GetLoadBalancerResponse{}, errortest.NewServiceError(404, "NotFound", "missing load balancer")
	}
	return loadbalancersdk.GetLoadBalancerResponse{LoadBalancer: resource}, nil
}

func (f *fakeGeneratedLoadBalancerOCIClient) ListLoadBalancers(_ context.Context, request loadbalancersdk.ListLoadBalancersRequest) (loadbalancersdk.ListLoadBalancersResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listErr != nil {
		return loadbalancersdk.ListLoadBalancersResponse{}, f.listErr
	}

	var items []loadbalancersdk.LoadBalancer
	for _, resource := range f.loadBalancers {
		if request.CompartmentId != nil && stringValue(resource.CompartmentId) != stringValue(request.CompartmentId) {
			continue
		}
		if request.DisplayName != nil && stringValue(resource.DisplayName) != stringValue(request.DisplayName) {
			continue
		}
		items = append(items, resource)
	}
	return loadbalancersdk.ListLoadBalancersResponse{Items: items}, nil
}

func (f *fakeGeneratedLoadBalancerOCIClient) UpdateLoadBalancer(_ context.Context, request loadbalancersdk.UpdateLoadBalancerRequest) (loadbalancersdk.UpdateLoadBalancerResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateErr != nil {
		return loadbalancersdk.UpdateLoadBalancerResponse{}, f.updateErr
	}

	resource, ok := f.loadBalancers[stringValue(request.LoadBalancerId)]
	if !ok {
		return loadbalancersdk.UpdateLoadBalancerResponse{}, errortest.NewServiceError(404, "NotFound", "missing load balancer")
	}
	if request.UpdateLoadBalancerDetails.DisplayName != nil {
		resource.DisplayName = request.UpdateLoadBalancerDetails.DisplayName
	}
	if request.UpdateLoadBalancerDetails.FreeformTags != nil {
		resource.FreeformTags = request.UpdateLoadBalancerDetails.FreeformTags
	}
	if request.UpdateLoadBalancerDetails.DefinedTags != nil {
		resource.DefinedTags = request.UpdateLoadBalancerDetails.DefinedTags
	}
	f.loadBalancers[stringValue(request.LoadBalancerId)] = resource
	return loadbalancersdk.UpdateLoadBalancerResponse{}, nil
}

func (f *fakeGeneratedLoadBalancerOCIClient) DeleteLoadBalancer(_ context.Context, request loadbalancersdk.DeleteLoadBalancerRequest) (loadbalancersdk.DeleteLoadBalancerResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteErr != nil {
		return loadbalancersdk.DeleteLoadBalancerResponse{}, f.deleteErr
	}

	delete(f.loadBalancers, stringValue(request.LoadBalancerId))
	return loadbalancersdk.DeleteLoadBalancerResponse{}, nil
}

func TestLoadBalancerRuntimeSemanticsEncodesBaselineLifecycle(t *testing.T) {
	t.Parallel()

	got := newReviewedLoadBalancerRuntimeSemantics()
	if got == nil {
		t.Fatal("newReviewedLoadBalancerRuntimeSemantics() = nil")
	}
	if got.FormalService != "loadbalancer" {
		t.Fatalf("FormalService = %q, want loadbalancer", got.FormalService)
	}
	if got.FormalSlug != "loadbalancer" {
		t.Fatalf("FormalSlug = %q, want loadbalancer", got.FormalSlug)
	}
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.CreateFollowUp.Strategy != "read-after-write" {
		t.Fatalf("CreateFollowUp.Strategy = %q, want read-after-write", got.CreateFollowUp.Strategy)
	}
	if got.UpdateFollowUp.Strategy != "read-after-write" {
		t.Fatalf("UpdateFollowUp.Strategy = %q, want read-after-write", got.UpdateFollowUp.Strategy)
	}
	if got.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("DeleteFollowUp.Strategy = %q, want confirm-delete", got.DeleteFollowUp.Strategy)
	}

	assertLoadBalancerStringSliceEqual(t, "Lifecycle.ProvisioningStates", got.Lifecycle.ProvisioningStates, []string{"CREATING"})
	assertLoadBalancerStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE"})
	assertLoadBalancerStringSliceEqual(t, "Delete.PendingStates", got.Delete.PendingStates, []string{"DELETING"})
	assertLoadBalancerStringSliceEqual(t, "Delete.TerminalStates", got.Delete.TerminalStates, []string{"DELETED"})
	assertLoadBalancerStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"displayName", "compartmentId"})
	assertLoadBalancerStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{
		"definedTags",
		"displayName",
		"freeformTags",
	})
	assertLoadBalancerStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{
		"backendSets",
		"certificates",
		"compartmentId",
		"hostnames",
		"ipMode",
		"isPrivate",
		"listeners",
		"networkSecurityGroupIds",
		"pathRouteSets",
		"reservedIps",
		"ruleSets",
		"shapeDetails",
		"shapeName",
		"sslCipherSuites",
		"subnetIds",
	})
}

func TestLoadBalancerRequestFieldsKeepTrackedOperationsScopedToRecordedIdentity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  []generatedruntimeRequestField
		want []generatedruntimeRequestField
	}{
		{
			name: "create",
			got:  requestFieldSnapshot(loadBalancerCreateFields()),
			want: []generatedruntimeRequestField{
				{FieldName: "CreateLoadBalancerDetails", RequestName: "CreateLoadBalancerDetails", Contribution: "body"},
			},
		},
		{
			name: "get",
			got:  requestFieldSnapshot(loadBalancerGetFields()),
			want: []generatedruntimeRequestField{
				{FieldName: "LoadBalancerId", RequestName: "loadBalancerId", Contribution: "path", PreferResourceID: true, LookupPaths: "status.id,status.status.ocid"},
			},
		},
		{
			name: "list",
			got:  requestFieldSnapshot(loadBalancerListFields()),
			want: []generatedruntimeRequestField{
				{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: "status.compartmentId,spec.compartmentId"},
				{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: "status.displayName,spec.displayName"},
			},
		},
		{
			name: "update",
			got:  requestFieldSnapshot(loadBalancerUpdateFields()),
			want: []generatedruntimeRequestField{
				{FieldName: "UpdateLoadBalancerDetails", RequestName: "UpdateLoadBalancerDetails", Contribution: "body"},
				{FieldName: "LoadBalancerId", RequestName: "loadBalancerId", Contribution: "path", PreferResourceID: true, LookupPaths: "status.id,status.status.ocid"},
			},
		},
		{
			name: "delete",
			got:  requestFieldSnapshot(loadBalancerDeleteFields()),
			want: []generatedruntimeRequestField{
				{FieldName: "LoadBalancerId", RequestName: "loadBalancerId", Contribution: "path", PreferResourceID: true, LookupPaths: "status.id,status.status.ocid"},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !reflect.DeepEqual(tc.got, tc.want) {
				t.Fatalf("%s fields = %#v, want %#v", tc.name, tc.got, tc.want)
			}
		})
	}
}

func TestCreateOrUpdateBindsExistingLoadBalancer(t *testing.T) {
	t.Parallel()

	client := &fakeGeneratedLoadBalancerOCIClient{
		loadBalancers: map[string]loadbalancersdk.LoadBalancer{
			loadBalancerExistingID: loadBalancerFromResource(loadBalancerExistingID, baseLoadBalancerResource()),
		},
	}
	serviceClient := newGeneratedLoadBalancerServiceClient(client, loggerutil.OSOKLogger{}, nil, nil)

	resource := baseLoadBalancerResource()
	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful response", response)
	}
	if len(client.listRequests) == 0 {
		t.Fatal("list requests = 0, want bind lookup before create")
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("create requests = %d, want 0 for bind path", len(client.createRequests))
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0 for no-drift bind path", len(client.updateRequests))
	}
	if got := resource.Status.Id; got != loadBalancerExistingID {
		t.Fatalf("status.id = %q, want %q", got, loadBalancerExistingID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != loadBalancerExistingID {
		t.Fatalf("status.status.ocid = %q, want %q", got, loadBalancerExistingID)
	}
}

func TestCreateOrUpdateCreatesMissingLoadBalancer(t *testing.T) {
	t.Parallel()

	client := &fakeGeneratedLoadBalancerOCIClient{
		loadBalancers: map[string]loadbalancersdk.LoadBalancer{},
	}
	serviceClient := newGeneratedLoadBalancerServiceClient(client, loggerutil.OSOKLogger{}, nil, nil)

	resource := baseLoadBalancerResource()
	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful response", response)
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("create requests = %d, want 1", len(client.createRequests))
	}
	if got := stringValue(client.createRequests[0].CreateLoadBalancerDetails.CompartmentId); got != loadBalancerCompartmentID {
		t.Fatalf("create request compartmentId = %q, want %q", got, loadBalancerCompartmentID)
	}
	if got := stringValue(client.createRequests[0].CreateLoadBalancerDetails.DisplayName); got != loadBalancerDisplayName {
		t.Fatalf("create request displayName = %q, want %q", got, loadBalancerDisplayName)
	}
	if got := resource.Status.Id; got != loadBalancerCreatedID {
		t.Fatalf("status.id = %q, want %q", got, loadBalancerCreatedID)
	}
}

func TestCreateOrUpdateUpdatesMutableLoadBalancerFields(t *testing.T) {
	t.Parallel()

	existing := baseLoadBalancerResource()
	existing.Spec.DisplayName = "existing_load_balancer"

	client := &fakeGeneratedLoadBalancerOCIClient{
		loadBalancers: map[string]loadbalancersdk.LoadBalancer{
			loadBalancerExistingID: loadBalancerFromResource(loadBalancerExistingID, existing),
		},
	}
	serviceClient := newGeneratedLoadBalancerServiceClient(client, loggerutil.OSOKLogger{}, nil, nil)

	resource := baseLoadBalancerResource()
	resource.Status.Id = loadBalancerExistingID
	resource.Status.CompartmentId = loadBalancerCompartmentID
	resource.Status.DisplayName = "existing_load_balancer"
	resource.Status.OsokStatus.Ocid = loadBalancerExistingID

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful response", response)
	}
	if len(client.updateRequests) != 1 {
		t.Fatalf("update requests = %d, want 1", len(client.updateRequests))
	}
	if got := stringValue(client.updateRequests[0].UpdateLoadBalancerDetails.DisplayName); got != loadBalancerDisplayName {
		t.Fatalf("update request displayName = %q, want %q", got, loadBalancerDisplayName)
	}
	if got := stringValue(client.updateRequests[0].LoadBalancerId); got != loadBalancerExistingID {
		t.Fatalf("update request loadBalancerId = %q, want %q", got, loadBalancerExistingID)
	}
	if got := resource.Status.DisplayName; got != loadBalancerDisplayName {
		t.Fatalf("status.displayName = %q, want %q", got, loadBalancerDisplayName)
	}
}

func TestDeleteConfirmsLoadBalancerRemoval(t *testing.T) {
	t.Parallel()

	client := &fakeGeneratedLoadBalancerOCIClient{
		loadBalancers: map[string]loadbalancersdk.LoadBalancer{
			loadBalancerExistingID: loadBalancerFromResource(loadBalancerExistingID, baseLoadBalancerResource()),
		},
	}
	serviceClient := newGeneratedLoadBalancerServiceClient(client, loggerutil.OSOKLogger{}, nil, nil)

	resource := baseLoadBalancerResource()
	resource.Status.Id = loadBalancerExistingID
	resource.Status.CompartmentId = loadBalancerCompartmentID
	resource.Status.DisplayName = loadBalancerDisplayName
	resource.Status.OsokStatus.Ocid = loadBalancerExistingID

	deleted, err := serviceClient.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("delete requests = %d, want 1", len(client.deleteRequests))
	}
	if got := stringValue(client.deleteRequests[0].LoadBalancerId); got != loadBalancerExistingID {
		t.Fatalf("delete request loadBalancerId = %q, want %q", got, loadBalancerExistingID)
	}
}

type generatedruntimeRequestField struct {
	FieldName        string
	RequestName      string
	Contribution     string
	PreferResourceID bool
	LookupPaths      string
}

func requestFieldSnapshot(fields []generatedruntime.RequestField) []generatedruntimeRequestField {
	snapshot := make([]generatedruntimeRequestField, 0, len(fields))
	for _, field := range fields {
		snapshot = append(snapshot, generatedruntimeRequestField{
			FieldName:        field.FieldName,
			RequestName:      field.RequestName,
			Contribution:     field.Contribution,
			PreferResourceID: field.PreferResourceID,
			LookupPaths:      strings.Join(field.LookupPaths, ","),
		})
	}
	return snapshot
}

func assertLoadBalancerStringSliceEqual(t *testing.T, field string, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", field, got, want)
	}
}

func baseLoadBalancerResource() *loadbalancerv1beta1.LoadBalancer {
	return &loadbalancerv1beta1.LoadBalancer{
		Spec: loadbalancerv1beta1.LoadBalancerSpec{
			CompartmentId: loadBalancerCompartmentID,
			DisplayName:   loadBalancerDisplayName,
			ShapeName:     "Flexible",
			SubnetIds:     []string{loadBalancerSubnetID},
			ShapeDetails: loadbalancerv1beta1.LoadBalancerShapeDetails{
				MinimumBandwidthInMbps: 10,
				MaximumBandwidthInMbps: 10,
			},
		},
	}
}

func loadBalancerFromResource(id string, resource *loadbalancerv1beta1.LoadBalancer) loadbalancersdk.LoadBalancer {
	return loadbalancersdk.LoadBalancer{
		Id:             common.String(id),
		CompartmentId:  common.String(resource.Spec.CompartmentId),
		DisplayName:    common.String(resource.Spec.DisplayName),
		LifecycleState: loadbalancersdk.LoadBalancerLifecycleStateActive,
		ShapeName:      common.String(resource.Spec.ShapeName),
		SubnetIds:      append([]string(nil), resource.Spec.SubnetIds...),
		ShapeDetails: &loadbalancersdk.ShapeDetails{
			MinimumBandwidthInMbps: common.Int(resource.Spec.ShapeDetails.MinimumBandwidthInMbps),
			MaximumBandwidthInMbps: common.Int(resource.Spec.ShapeDetails.MaximumBandwidthInMbps),
		},
		IsPrivate:               common.Bool(resource.Spec.IsPrivate),
		NetworkSecurityGroupIds: append([]string(nil), resource.Spec.NetworkSecurityGroupIds...),
		FreeformTags:            copyStringMap(resource.Spec.FreeformTags),
		DefinedTags:             copyDefinedTags(resource.Spec.DefinedTags),
	}
}

func loadBalancerFromCreateDetails(id string, details loadbalancersdk.CreateLoadBalancerDetails) loadbalancersdk.LoadBalancer {
	return loadbalancersdk.LoadBalancer{
		Id:                      common.String(id),
		CompartmentId:           details.CompartmentId,
		DisplayName:             details.DisplayName,
		LifecycleState:          loadbalancersdk.LoadBalancerLifecycleStateActive,
		ShapeName:               details.ShapeName,
		SubnetIds:               append([]string(nil), details.SubnetIds...),
		ShapeDetails:            details.ShapeDetails,
		IsPrivate:               details.IsPrivate,
		NetworkSecurityGroupIds: append([]string(nil), details.NetworkSecurityGroupIds...),
		FreeformTags:            copyStringMap(details.FreeformTags),
		DefinedTags:             copyNestedAnyMap(details.DefinedTags),
	}
}

func copyStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func copyDefinedTags(input map[string]shared.MapValue) map[string]map[string]interface{} {
	if input == nil {
		return nil
	}
	output := make(map[string]map[string]interface{}, len(input))
	for namespace, value := range input {
		converted := make(map[string]interface{}, len(value))
		for key, item := range value {
			converted[key] = item
		}
		output[namespace] = converted
	}
	return output
}

func copyNestedAnyMap(input map[string]map[string]interface{}) map[string]map[string]interface{} {
	if input == nil {
		return nil
	}
	output := make(map[string]map[string]interface{}, len(input))
	for namespace, value := range input {
		output[namespace] = copyAnyMap(value)
	}
	return output
}

func copyAnyMap(input map[string]interface{}) map[string]interface{} {
	if input == nil {
		return nil
	}
	output := make(map[string]interface{}, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

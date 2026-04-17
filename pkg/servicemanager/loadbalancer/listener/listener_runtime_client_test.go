/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package listener

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	loadbalancersdk "github.com/oracle/oci-go-sdk/v65/loadbalancer"
	loadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/loadbalancer/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	listenerLoadBalancerID = "ocid1.loadbalancer.oc1..exampleuniqueID"
	listenerNameValue      = "example_listener"
)

type fakeGeneratedListenerOCIClient struct {
	createRequests []loadbalancersdk.CreateListenerRequest
	updateRequests []loadbalancersdk.UpdateListenerRequest
	deleteRequests []loadbalancersdk.DeleteListenerRequest

	getErr    error
	createErr error
	updateErr error
	deleteErr error

	lifecycleState loadbalancersdk.LoadBalancerLifecycleStateEnum
	listeners      map[string]loadbalancersdk.Listener
}

func (f *fakeGeneratedListenerOCIClient) CreateListener(_ context.Context, request loadbalancersdk.CreateListenerRequest) (loadbalancersdk.CreateListenerResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createErr != nil {
		return loadbalancersdk.CreateListenerResponse{}, f.createErr
	}
	if f.listeners == nil {
		f.listeners = map[string]loadbalancersdk.Listener{}
	}
	name := stringValue(request.CreateListenerDetails.Name)
	f.listeners[name] = listenerFromCreateDetails(request.CreateListenerDetails)
	if f.lifecycleState == "" {
		f.lifecycleState = loadbalancersdk.LoadBalancerLifecycleStateActive
	}
	return loadbalancersdk.CreateListenerResponse{}, nil
}

func (f *fakeGeneratedListenerOCIClient) GetLoadBalancer(_ context.Context, request loadbalancersdk.GetLoadBalancerRequest) (loadbalancersdk.GetLoadBalancerResponse, error) {
	if f.getErr != nil {
		return loadbalancersdk.GetLoadBalancerResponse{}, f.getErr
	}

	listeners := make(map[string]loadbalancersdk.Listener, len(f.listeners))
	for name, listener := range f.listeners {
		listeners[name] = listener
	}

	return loadbalancersdk.GetLoadBalancerResponse{
		LoadBalancer: loadbalancersdk.LoadBalancer{
			Id:             request.LoadBalancerId,
			LifecycleState: listenerLifecycleOrDefault(f.lifecycleState),
			Listeners:      listeners,
		},
	}, nil
}

func (f *fakeGeneratedListenerOCIClient) UpdateListener(_ context.Context, request loadbalancersdk.UpdateListenerRequest) (loadbalancersdk.UpdateListenerResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateErr != nil {
		return loadbalancersdk.UpdateListenerResponse{}, f.updateErr
	}
	if f.listeners == nil {
		f.listeners = map[string]loadbalancersdk.Listener{}
	}
	name := stringValue(request.ListenerName)
	existing := f.listeners[name]
	f.listeners[name] = listenerFromUpdateDetails(name, request.UpdateListenerDetails, existing)
	if f.lifecycleState == "" {
		f.lifecycleState = loadbalancersdk.LoadBalancerLifecycleStateActive
	}
	return loadbalancersdk.UpdateListenerResponse{}, nil
}

func (f *fakeGeneratedListenerOCIClient) DeleteListener(_ context.Context, request loadbalancersdk.DeleteListenerRequest) (loadbalancersdk.DeleteListenerResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteErr != nil {
		return loadbalancersdk.DeleteListenerResponse{}, f.deleteErr
	}
	delete(f.listeners, stringValue(request.ListenerName))
	if f.lifecycleState == "" {
		f.lifecycleState = loadbalancersdk.LoadBalancerLifecycleStateDeleting
	}
	return loadbalancersdk.DeleteListenerResponse{}, nil
}

func TestListenerRequestFieldsKeepTrackedOperationsScopedToRecordedPath(t *testing.T) {
	t.Parallel()

	check := func(name string, got, want []generatedruntimeRequestField) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("%s fields len = %d, want %d", name, len(got), len(want))
		}
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("%s field[%d] = %#v, want %#v", name, i, got[i], want[i])
			}
		}
	}

	check("create", requestFieldSnapshot(listenerCreateFields()), []generatedruntimeRequestField{
		{FieldName: "LoadBalancerId", RequestName: "loadBalancerId", Contribution: "path", LookupPaths: strings.Join([]string{"status.loadBalancerId", "spec.loadBalancerId"}, ",")},
		{FieldName: "CreateListenerDetails", RequestName: "CreateListenerDetails", Contribution: "body"},
	})
	check("get", requestFieldSnapshot(listenerGetFields()), []generatedruntimeRequestField{
		{FieldName: "LoadBalancerId", RequestName: "loadBalancerId", Contribution: "path", LookupPaths: strings.Join([]string{"status.loadBalancerId", "spec.loadBalancerId"}, ",")},
		{FieldName: "ListenerName", RequestName: "listenerName", Contribution: "path", LookupPaths: strings.Join([]string{"status.name", "spec.name", "name"}, ",")},
	})
	check("list", requestFieldSnapshot(listenerListFields()), []generatedruntimeRequestField{
		{FieldName: "LoadBalancerId", RequestName: "loadBalancerId", Contribution: "path", LookupPaths: strings.Join([]string{"status.loadBalancerId", "spec.loadBalancerId"}, ",")},
	})
	check("update", requestFieldSnapshot(listenerUpdateFields()), []generatedruntimeRequestField{
		{FieldName: "LoadBalancerId", RequestName: "loadBalancerId", Contribution: "path", LookupPaths: strings.Join([]string{"status.loadBalancerId", "spec.loadBalancerId"}, ",")},
		{FieldName: "ListenerName", RequestName: "listenerName", Contribution: "path", LookupPaths: strings.Join([]string{"status.name", "spec.name", "name"}, ",")},
		{FieldName: "UpdateListenerDetails", RequestName: "UpdateListenerDetails", Contribution: "body"},
	})
	check("delete", requestFieldSnapshot(listenerDeleteFields()), []generatedruntimeRequestField{
		{FieldName: "LoadBalancerId", RequestName: "loadBalancerId", Contribution: "path", LookupPaths: strings.Join([]string{"status.loadBalancerId", "spec.loadBalancerId"}, ",")},
		{FieldName: "ListenerName", RequestName: "listenerName", Contribution: "path", LookupPaths: strings.Join([]string{"status.name", "spec.name", "name"}, ",")},
	})
}

func TestCreateOrUpdateBindsExistingListener(t *testing.T) {
	t.Parallel()

	client := &fakeGeneratedListenerOCIClient{
		lifecycleState: loadbalancersdk.LoadBalancerLifecycleStateActive,
		listeners: map[string]loadbalancersdk.Listener{
			listenerNameValue: {
				Name:                  common.String(listenerNameValue),
				DefaultBackendSetName: common.String("example_backend_set"),
				Port:                  common.Int(80),
				Protocol:              common.String("HTTP"),
			},
		},
	}
	serviceClient := newGeneratedListenerServiceClient(client, loggerutil.OSOKLogger{}, nil, nil)

	resource := &loadbalancerv1beta1.Listener{
		Spec: loadbalancerv1beta1.ListenerSpec{
			LoadBalancerId:        listenerLoadBalancerID,
			Name:                  listenerNameValue,
			DefaultBackendSetName: "example_backend_set",
			Port:                  80,
			Protocol:              "HTTP",
		},
	}

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful response", response)
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("create requests = %d, want 0 for bind path", len(client.createRequests))
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0 for no-drift bind path", len(client.updateRequests))
	}
	if got := resource.Status.LoadBalancerId; got != listenerLoadBalancerID {
		t.Fatalf("status.loadBalancerId = %q, want %q", got, listenerLoadBalancerID)
	}
	if got := resource.Status.Name; got != listenerNameValue {
		t.Fatalf("status.name = %q, want %q", got, listenerNameValue)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != listenerSyntheticOCID(listenerLoadBalancerID, listenerNameValue) {
		t.Fatalf("status.status.ocid = %q, want %q", got, listenerSyntheticOCID(listenerLoadBalancerID, listenerNameValue))
	}
}

func TestCreateOrUpdateCreatesMissingListener(t *testing.T) {
	t.Parallel()

	client := &fakeGeneratedListenerOCIClient{
		lifecycleState: loadbalancersdk.LoadBalancerLifecycleStateActive,
		listeners:      map[string]loadbalancersdk.Listener{},
	}
	serviceClient := newGeneratedListenerServiceClient(client, loggerutil.OSOKLogger{}, nil, nil)

	resource := &loadbalancerv1beta1.Listener{
		Spec: loadbalancerv1beta1.ListenerSpec{
			LoadBalancerId:        listenerLoadBalancerID,
			Name:                  listenerNameValue,
			DefaultBackendSetName: "example_backend_set",
			Port:                  80,
			Protocol:              "HTTP",
		},
	}

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
	if got := stringValue(client.createRequests[0].LoadBalancerId); got != listenerLoadBalancerID {
		t.Fatalf("create request loadBalancerId = %q, want %q", got, listenerLoadBalancerID)
	}
	if got := resource.Status.Name; got != listenerNameValue {
		t.Fatalf("status.name = %q, want %q", got, listenerNameValue)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != listenerSyntheticOCID(listenerLoadBalancerID, listenerNameValue) {
		t.Fatalf("status.status.ocid = %q, want %q", got, listenerSyntheticOCID(listenerLoadBalancerID, listenerNameValue))
	}
}

func TestCreateOrUpdateUpdatesExistingMutableListenerFields(t *testing.T) {
	t.Parallel()

	client := &fakeGeneratedListenerOCIClient{
		lifecycleState: loadbalancersdk.LoadBalancerLifecycleStateActive,
		listeners: map[string]loadbalancersdk.Listener{
			listenerNameValue: {
				Name:                  common.String(listenerNameValue),
				DefaultBackendSetName: common.String("example_backend_set"),
				Port:                  common.Int(80),
				Protocol:              common.String("HTTP"),
			},
		},
	}
	serviceClient := newGeneratedListenerServiceClient(client, loggerutil.OSOKLogger{}, nil, nil)

	resource := &loadbalancerv1beta1.Listener{
		Spec: loadbalancerv1beta1.ListenerSpec{
			LoadBalancerId:        listenerLoadBalancerID,
			Name:                  listenerNameValue,
			DefaultBackendSetName: "example_backend_set",
			Port:                  8080,
			Protocol:              "HTTP",
		},
	}

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
	if got := stringValue(client.updateRequests[0].LoadBalancerId); got != listenerLoadBalancerID {
		t.Fatalf("update request loadBalancerId = %q, want %q", got, listenerLoadBalancerID)
	}
	if got := stringValue(client.updateRequests[0].ListenerName); got != listenerNameValue {
		t.Fatalf("update request listenerName = %q, want %q", got, listenerNameValue)
	}
	if got := resource.Status.Port; got != 8080 {
		t.Fatalf("status.port = %d, want 8080", got)
	}
}

func TestCreateOrUpdatePreservesExplicitFalseVerifyPeerCertificate(t *testing.T) {
	t.Parallel()

	client := &fakeGeneratedListenerOCIClient{
		lifecycleState: loadbalancersdk.LoadBalancerLifecycleStateActive,
		listeners: map[string]loadbalancersdk.Listener{
			listenerNameValue: {
				Name:                  common.String(listenerNameValue),
				DefaultBackendSetName: common.String("example_backend_set"),
				Port:                  common.Int(80),
				Protocol:              common.String("HTTP"),
				SslConfiguration: &loadbalancersdk.SslConfiguration{
					VerifyPeerCertificate: common.Bool(true),
				},
			},
		},
	}
	serviceClient := newGeneratedListenerServiceClient(client, loggerutil.OSOKLogger{}, nil, nil)

	resource := &loadbalancerv1beta1.Listener{
		Spec: loadbalancerv1beta1.ListenerSpec{
			LoadBalancerId:        listenerLoadBalancerID,
			Name:                  listenerNameValue,
			DefaultBackendSetName: "example_backend_set",
			Port:                  80,
			Protocol:              "HTTP",
			SslConfiguration: loadbalancerv1beta1.ListenerSslConfiguration{
				VerifyPeerCertificate: false,
			},
		},
	}

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful response", response)
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("create requests = %d, want 0 for update path", len(client.createRequests))
	}
	if len(client.updateRequests) != 1 {
		t.Fatalf("update requests = %d, want 1", len(client.updateRequests))
	}

	sslConfiguration := client.updateRequests[0].UpdateListenerDetails.SslConfiguration
	if sslConfiguration == nil {
		t.Fatal("UpdateListenerDetails.SslConfiguration = nil, want explicit false bool preserved")
	}
	if got := sslConfiguration.VerifyPeerCertificate; got == nil || *got {
		t.Fatalf("UpdateListenerDetails.SslConfiguration.VerifyPeerCertificate = %#v, want false", got)
	}
	if resource.Status.SslConfiguration.VerifyPeerCertificate {
		t.Fatal("status.sslConfiguration.verifyPeerCertificate = true, want false after update projection")
	}
}

func TestCreateOrUpdateClearsExistingMutableScalarListenerField(t *testing.T) {
	t.Parallel()

	client := &fakeGeneratedListenerOCIClient{
		lifecycleState: loadbalancersdk.LoadBalancerLifecycleStateActive,
		listeners: map[string]loadbalancersdk.Listener{
			listenerNameValue: {
				Name:                  common.String(listenerNameValue),
				DefaultBackendSetName: common.String("example_backend_set"),
				Port:                  common.Int(80),
				Protocol:              common.String("HTTP"),
				PathRouteSetName:      common.String("example_path_route_set"),
			},
		},
	}
	serviceClient := newGeneratedListenerServiceClient(client, loggerutil.OSOKLogger{}, nil, nil)

	resource := &loadbalancerv1beta1.Listener{
		Spec: loadbalancerv1beta1.ListenerSpec{
			LoadBalancerId:        listenerLoadBalancerID,
			Name:                  listenerNameValue,
			DefaultBackendSetName: "example_backend_set",
			Port:                  80,
			Protocol:              "HTTP",
		},
	}

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
	if got := client.updateRequests[0].UpdateListenerDetails.PathRouteSetName; got == nil || *got != "" {
		t.Fatalf("UpdateListenerDetails.PathRouteSetName = %#v, want explicit empty string clear", got)
	}
	if got := resource.Status.PathRouteSetName; got != "" {
		t.Fatalf("status.pathRouteSetName = %q, want cleared value", got)
	}
}

func TestCreateOrUpdateClearsExistingMutableListListenerField(t *testing.T) {
	t.Parallel()

	client := &fakeGeneratedListenerOCIClient{
		lifecycleState: loadbalancersdk.LoadBalancerLifecycleStateActive,
		listeners: map[string]loadbalancersdk.Listener{
			listenerNameValue: {
				Name:                  common.String(listenerNameValue),
				DefaultBackendSetName: common.String("example_backend_set"),
				Port:                  common.Int(80),
				Protocol:              common.String("HTTP"),
				HostnameNames:         []string{"example-hostname"},
			},
		},
	}
	serviceClient := newGeneratedListenerServiceClient(client, loggerutil.OSOKLogger{}, nil, nil)

	resource := &loadbalancerv1beta1.Listener{
		Spec: loadbalancerv1beta1.ListenerSpec{
			LoadBalancerId:        listenerLoadBalancerID,
			Name:                  listenerNameValue,
			DefaultBackendSetName: "example_backend_set",
			Port:                  80,
			Protocol:              "HTTP",
		},
	}

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
	if got := client.updateRequests[0].UpdateListenerDetails.HostnameNames; got == nil || len(got) != 0 {
		t.Fatalf("UpdateListenerDetails.HostnameNames = %#v, want explicit empty slice clear", got)
	}
	if got := resource.Status.HostnameNames; len(got) != 0 {
		t.Fatalf("status.hostnameNames = %#v, want cleared value", got)
	}
}

func TestCreateOrUpdateRejectsUnsupportedNestedListenerClear(t *testing.T) {
	t.Parallel()

	client := &fakeGeneratedListenerOCIClient{
		lifecycleState: loadbalancersdk.LoadBalancerLifecycleStateActive,
		listeners: map[string]loadbalancersdk.Listener{
			listenerNameValue: {
				Name:                  common.String(listenerNameValue),
				DefaultBackendSetName: common.String("example_backend_set"),
				Port:                  common.Int(80),
				Protocol:              common.String("HTTP"),
				SslConfiguration: &loadbalancersdk.SslConfiguration{
					CertificateName: common.String("example-certificate"),
				},
			},
		},
	}
	serviceClient := newGeneratedListenerServiceClient(client, loggerutil.OSOKLogger{}, nil, nil)

	resource := &loadbalancerv1beta1.Listener{
		Spec: loadbalancerv1beta1.ListenerSpec{
			LoadBalancerId:        listenerLoadBalancerID,
			Name:                  listenerNameValue,
			DefaultBackendSetName: "example_backend_set",
			Port:                  80,
			Protocol:              "HTTP",
		},
	}

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want unsupported nested clear error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want unsuccessful response", response)
	}
	if !strings.Contains(err.Error(), "sslConfiguration.certificateName") {
		t.Fatalf("CreateOrUpdate() error = %v, want sslConfiguration.certificateName clear rejection", err)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0 when nested clear is rejected", len(client.updateRequests))
	}
}

func TestCreateOrUpdateRejectsForceNewListenerDrift(t *testing.T) {
	t.Parallel()

	client := &fakeGeneratedListenerOCIClient{
		lifecycleState: loadbalancersdk.LoadBalancerLifecycleStateActive,
		listeners: map[string]loadbalancersdk.Listener{
			listenerNameValue: {
				Name:                  common.String(listenerNameValue),
				DefaultBackendSetName: common.String("example_backend_set"),
				Port:                  common.Int(80),
				Protocol:              common.String("HTTP"),
			},
		},
	}
	serviceClient := newGeneratedListenerServiceClient(client, loggerutil.OSOKLogger{}, nil, nil)

	resource := &loadbalancerv1beta1.Listener{
		Spec: loadbalancerv1beta1.ListenerSpec{
			LoadBalancerId:        listenerLoadBalancerID,
			Name:                  "replacement_listener",
			DefaultBackendSetName: "example_backend_set",
			Port:                  80,
			Protocol:              "HTTP",
		},
		Status: loadbalancerv1beta1.ListenerStatus{
			LoadBalancerId: listenerLoadBalancerID,
			Name:           listenerNameValue,
		},
	}
	resource.Status.OsokStatus.Ocid = "listener/" + listenerLoadBalancerID + "/" + listenerNameValue

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want force-new drift error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want unsuccessful response", response)
	}
	if !strings.Contains(err.Error(), "require replacement when name changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want name replacement message", err)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0 on force-new drift", len(client.updateRequests))
	}
}

func TestDeleteConfirmsListenerRemoval(t *testing.T) {
	t.Parallel()

	client := &fakeGeneratedListenerOCIClient{
		lifecycleState: loadbalancersdk.LoadBalancerLifecycleStateActive,
		listeners: map[string]loadbalancersdk.Listener{
			listenerNameValue: {
				Name:                  common.String(listenerNameValue),
				DefaultBackendSetName: common.String("example_backend_set"),
				Port:                  common.Int(80),
				Protocol:              common.String("HTTP"),
			},
		},
	}
	serviceClient := newGeneratedListenerServiceClient(client, loggerutil.OSOKLogger{}, nil, nil)

	resource := &loadbalancerv1beta1.Listener{
		Spec: loadbalancerv1beta1.ListenerSpec{
			LoadBalancerId:        "ocid1.loadbalancer.oc1..new",
			Name:                  "replacement_listener",
			DefaultBackendSetName: "example_backend_set",
			Port:                  80,
			Protocol:              "HTTP",
		},
		Status: loadbalancerv1beta1.ListenerStatus{
			LoadBalancerId: listenerLoadBalancerID,
			Name:           listenerNameValue,
		},
	}
	resource.Status.OsokStatus.Ocid = "listener/" + listenerLoadBalancerID + "/" + listenerNameValue

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
	if got := stringValue(client.deleteRequests[0].LoadBalancerId); got != listenerLoadBalancerID {
		t.Fatalf("delete request loadBalancerId = %q, want %q", got, listenerLoadBalancerID)
	}
	if got := stringValue(client.deleteRequests[0].ListenerName); got != listenerNameValue {
		t.Fatalf("delete request listenerName = %q, want %q", got, listenerNameValue)
	}
}

type generatedruntimeRequestField struct {
	FieldName    string
	RequestName  string
	Contribution string
	LookupPaths  string
}

func requestFieldSnapshot(fields []generatedruntime.RequestField) []generatedruntimeRequestField {
	snapshot := make([]generatedruntimeRequestField, 0, len(fields))
	for _, field := range fields {
		snapshot = append(snapshot, generatedruntimeRequestField{
			FieldName:    field.FieldName,
			RequestName:  field.RequestName,
			Contribution: field.Contribution,
			LookupPaths:  strings.Join(field.LookupPaths, ","),
		})
	}
	return snapshot
}

func listenerFromCreateDetails(details loadbalancersdk.CreateListenerDetails) loadbalancersdk.Listener {
	return loadbalancersdk.Listener{
		Name:                    details.Name,
		DefaultBackendSetName:   details.DefaultBackendSetName,
		Port:                    details.Port,
		Protocol:                details.Protocol,
		HostnameNames:           details.HostnameNames,
		PathRouteSetName:        details.PathRouteSetName,
		SslConfiguration:        sslConfigFromDetails(details.SslConfiguration),
		ConnectionConfiguration: details.ConnectionConfiguration,
		RuleSetNames:            details.RuleSetNames,
		RoutingPolicyName:       details.RoutingPolicyName,
	}
}

func listenerFromUpdateDetails(name string, details loadbalancersdk.UpdateListenerDetails, existing loadbalancersdk.Listener) loadbalancersdk.Listener {
	listener := existing
	listener.Name = common.String(name)
	listener.DefaultBackendSetName = details.DefaultBackendSetName
	listener.Port = details.Port
	listener.Protocol = details.Protocol
	listener.HostnameNames = details.HostnameNames
	listener.PathRouteSetName = details.PathRouteSetName
	listener.SslConfiguration = sslConfigFromDetails(details.SslConfiguration)
	listener.ConnectionConfiguration = details.ConnectionConfiguration
	listener.RuleSetNames = details.RuleSetNames
	listener.RoutingPolicyName = details.RoutingPolicyName
	return listener
}

func sslConfigFromDetails(details *loadbalancersdk.SslConfigurationDetails) *loadbalancersdk.SslConfiguration {
	if details == nil {
		return nil
	}
	return &loadbalancersdk.SslConfiguration{
		VerifyDepth:                    details.VerifyDepth,
		VerifyPeerCertificate:          details.VerifyPeerCertificate,
		TrustedCertificateAuthorityIds: details.TrustedCertificateAuthorityIds,
		CertificateIds:                 details.CertificateIds,
		CertificateName:                details.CertificateName,
		Protocols:                      details.Protocols,
		CipherSuiteName:                details.CipherSuiteName,
		ServerOrderPreference:          loadbalancersdk.SslConfigurationServerOrderPreferenceEnum(details.ServerOrderPreference),
	}
}

func listenerLifecycleOrDefault(value loadbalancersdk.LoadBalancerLifecycleStateEnum) loadbalancersdk.LoadBalancerLifecycleStateEnum {
	if value == "" {
		return loadbalancersdk.LoadBalancerLifecycleStateActive
	}
	return value
}

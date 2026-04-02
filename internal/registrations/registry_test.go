/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package registrations

import (
	"context"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	containerenginev1beta1 "github.com/oracle/oci-service-operator/api/containerengine/v1beta1"
	databasev1beta1 "github.com/oracle/oci-service-operator/api/database/v1beta1"
	mysqlv1beta1 "github.com/oracle/oci-service-operator/api/mysql/v1beta1"
	nosqlv1beta1 "github.com/oracle/oci-service-operator/api/nosql/v1beta1"
	psqlv1beta1 "github.com/oracle/oci-service-operator/api/psql/v1beta1"
	streamingv1beta1 "github.com/oracle/oci-service-operator/api/streaming/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/metrics"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestAllAddToSchemeRegistersDefaultActiveGroupKinds(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	for _, registration := range All() {
		if err := registration.AddToScheme(scheme); err != nil {
			t.Fatalf("AddToScheme(%q) error = %v", registration.Group, err)
		}
	}

	for _, obj := range []runtime.Object{
		&containerenginev1beta1.Cluster{},
		&databasev1beta1.AutonomousDatabase{},
		&mysqlv1beta1.DbSystem{},
		&nosqlv1beta1.Table{},
		&psqlv1beta1.DbSystem{},
		&streamingv1beta1.Stream{},
	} {
		gvks, _, err := scheme.ObjectKinds(obj)
		if err != nil {
			t.Fatalf("ObjectKinds(%T) error = %v", obj, err)
		}
		if len(gvks) == 0 {
			t.Fatalf("ObjectKinds(%T) returned no GVKs", obj)
		}
	}
}

func TestNewBaseReconcilerUsesSharedRuntimeDeps(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	client := ctrlclientfake.NewClientBuilder().WithScheme(scheme).Build()
	metricsClient := &metrics.Metrics{Name: "oci", ServiceName: "osok"}
	credClient := &fakeCredentialClient{}
	recorderNames := make([]string, 0, 1)

	ctx := Context{
		Client: client,
		Scheme: scheme,
		EventRecorderFor: func(name string) record.EventRecorder {
			recorderNames = append(recorderNames, name)
			return record.NewFakeRecorder(1)
		},
		ServiceManagerDeps: servicemanager.RuntimeDeps{
			Provider:         common.NewRawConfigurationProvider("", "", "", "", "", nil),
			CredentialClient: credClient,
			Metrics:          metricsClient,
		},
	}

	var gotDeps servicemanager.RuntimeDeps
	reconciler := NewBaseReconciler(ctx, "Streams", func(deps servicemanager.RuntimeDeps) servicemanager.OSOKServiceManager {
		gotDeps = deps
		return fakeServiceManager{}
	})

	if gotDeps.CredentialClient != credClient {
		t.Fatal("factory credential client was not forwarded")
	}
	if gotDeps.Scheme != scheme {
		t.Fatal("factory scheme was not forwarded")
	}
	if gotDeps.Metrics != metricsClient {
		t.Fatal("factory metrics were not forwarded")
	}
	if reconciler.Metrics != metricsClient {
		t.Fatal("base reconciler metrics were not forwarded")
	}
	if reconciler.Scheme != scheme {
		t.Fatal("base reconciler scheme was not forwarded")
	}
	if len(recorderNames) != 1 || recorderNames[0] != "Streams" {
		t.Fatalf("recorder names = %v, want [Streams]", recorderNames)
	}
}

type fakeCredentialClient struct{}

func (f *fakeCredentialClient) CreateSecret(context.Context, string, string, map[string]string, map[string][]byte) (bool, error) {
	return true, nil
}

func (f *fakeCredentialClient) DeleteSecret(context.Context, string, string) (bool, error) {
	return true, nil
}

func (f *fakeCredentialClient) GetSecret(context.Context, string, string) (map[string][]byte, error) {
	return map[string][]byte{}, nil
}

func (f *fakeCredentialClient) UpdateSecret(context.Context, string, string, map[string]string, map[string][]byte) (bool, error) {
	return true, nil
}

var _ credhelper.CredentialClient = (*fakeCredentialClient)(nil)

type fakeServiceManager struct{}

func (fakeServiceManager) CreateOrUpdate(context.Context, runtime.Object, ctrl.Request) (servicemanager.OSOKResponse, error) {
	return servicemanager.OSOKResponse{}, nil
}

func (fakeServiceManager) Delete(context.Context, runtime.Object) (bool, error) {
	return true, nil
}

func (fakeServiceManager) GetCrdStatus(runtime.Object) (*shared.OSOKStatus, error) {
	return &shared.OSOKStatus{}, nil
}

var _ servicemanager.OSOKServiceManager = fakeServiceManager{}

func TestAllIncludesGeneratedGroupsAfterManualGroups(t *testing.T) {
	restoreGeneratedGroupRegistrations(t)
	registerGeneratedGroup(GroupRegistration{Group: "events"})

	registrations := All()
	if len(registrations) != len(manualGroupRegistrations)+1 {
		t.Fatalf("len(All()) = %d, want %d", len(registrations), len(manualGroupRegistrations)+1)
	}
	if registrations[len(registrations)-1].Group != "events" {
		t.Fatalf("All()[last].Group = %q, want %q", registrations[len(registrations)-1].Group, "events")
	}
}

func TestAllReturnsUniqueGroupsWithManualPrefix(t *testing.T) {
	t.Parallel()

	registrations := All()
	if len(registrations) < len(manualGroupRegistrations) {
		t.Fatalf("len(All()) = %d, want at least %d manual groups", len(registrations), len(manualGroupRegistrations))
	}

	seen := make(map[string]struct{}, len(registrations))
	for index, registration := range registrations {
		if _, ok := seen[registration.Group]; ok {
			t.Fatalf("All() returned duplicate group %q", registration.Group)
		}
		seen[registration.Group] = struct{}{}
		if index < len(manualGroupRegistrations) && registration.Group != manualGroupRegistrations[index].Group {
			t.Fatalf("All()[%d].Group = %q, want manual prefix %q", index, registration.Group, manualGroupRegistrations[index].Group)
		}
	}
}

func TestAllSkipsGeneratedGroupsThatDuplicateManualEntries(t *testing.T) {
	restoreGeneratedGroupRegistrations(t)
	registerGeneratedGroup(GroupRegistration{Group: "mysql"})

	registrations := All()
	count := 0
	for _, registration := range registrations {
		if registration.Group == "mysql" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("mysql registration count = %d, want 1", count)
	}
}

func restoreGeneratedGroupRegistrations(t *testing.T) {
	t.Helper()

	snapshot := append([]GroupRegistration(nil), generatedGroupRegistrations...)
	generatedGroupRegistrations = nil
	t.Cleanup(func() {
		generatedGroupRegistrations = snapshot
	})
}

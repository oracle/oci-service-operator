/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package privateapplication

import (
	"context"
	"encoding/base64"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	servicecatalogsdk "github.com/oracle/oci-go-sdk/v65/servicecatalog"
	servicecatalogv1beta1 "github.com/oracle/oci-service-operator/api/servicecatalog/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakePrivateApplicationOCIClient struct {
	createPrivateApplicationFn func(context.Context, servicecatalogsdk.CreatePrivateApplicationRequest) (servicecatalogsdk.CreatePrivateApplicationResponse, error)
	getPrivateApplicationFn    func(context.Context, servicecatalogsdk.GetPrivateApplicationRequest) (servicecatalogsdk.GetPrivateApplicationResponse, error)
	listPrivateApplicationsFn  func(context.Context, servicecatalogsdk.ListPrivateApplicationsRequest) (servicecatalogsdk.ListPrivateApplicationsResponse, error)
	updatePrivateApplicationFn func(context.Context, servicecatalogsdk.UpdatePrivateApplicationRequest) (servicecatalogsdk.UpdatePrivateApplicationResponse, error)
	deletePrivateApplicationFn func(context.Context, servicecatalogsdk.DeletePrivateApplicationRequest) (servicecatalogsdk.DeletePrivateApplicationResponse, error)
	listPackagesFn             func(context.Context, servicecatalogsdk.ListPrivateApplicationPackagesRequest) (servicecatalogsdk.ListPrivateApplicationPackagesResponse, error)
	downloadPackageConfigFn    func(context.Context, servicecatalogsdk.GetPrivateApplicationPackageActionDownloadConfigRequest) (servicecatalogsdk.GetPrivateApplicationPackageActionDownloadConfigResponse, error)
	downloadLogoFn             func(context.Context, servicecatalogsdk.GetPrivateApplicationActionDownloadLogoRequest) (servicecatalogsdk.GetPrivateApplicationActionDownloadLogoResponse, error)
}

func (f *fakePrivateApplicationOCIClient) CreatePrivateApplication(
	ctx context.Context,
	req servicecatalogsdk.CreatePrivateApplicationRequest,
) (servicecatalogsdk.CreatePrivateApplicationResponse, error) {
	if f.createPrivateApplicationFn != nil {
		return f.createPrivateApplicationFn(ctx, req)
	}
	return servicecatalogsdk.CreatePrivateApplicationResponse{}, nil
}

func (f *fakePrivateApplicationOCIClient) GetPrivateApplication(
	ctx context.Context,
	req servicecatalogsdk.GetPrivateApplicationRequest,
) (servicecatalogsdk.GetPrivateApplicationResponse, error) {
	if f.getPrivateApplicationFn != nil {
		return f.getPrivateApplicationFn(ctx, req)
	}
	return servicecatalogsdk.GetPrivateApplicationResponse{}, nil
}

func (f *fakePrivateApplicationOCIClient) ListPrivateApplications(
	ctx context.Context,
	req servicecatalogsdk.ListPrivateApplicationsRequest,
) (servicecatalogsdk.ListPrivateApplicationsResponse, error) {
	if f.listPrivateApplicationsFn != nil {
		return f.listPrivateApplicationsFn(ctx, req)
	}
	return servicecatalogsdk.ListPrivateApplicationsResponse{}, nil
}

func (f *fakePrivateApplicationOCIClient) UpdatePrivateApplication(
	ctx context.Context,
	req servicecatalogsdk.UpdatePrivateApplicationRequest,
) (servicecatalogsdk.UpdatePrivateApplicationResponse, error) {
	if f.updatePrivateApplicationFn != nil {
		return f.updatePrivateApplicationFn(ctx, req)
	}
	return servicecatalogsdk.UpdatePrivateApplicationResponse{}, nil
}

func (f *fakePrivateApplicationOCIClient) DeletePrivateApplication(
	ctx context.Context,
	req servicecatalogsdk.DeletePrivateApplicationRequest,
) (servicecatalogsdk.DeletePrivateApplicationResponse, error) {
	if f.deletePrivateApplicationFn != nil {
		return f.deletePrivateApplicationFn(ctx, req)
	}
	return servicecatalogsdk.DeletePrivateApplicationResponse{}, nil
}

func (f *fakePrivateApplicationOCIClient) ListPrivateApplicationPackages(
	ctx context.Context,
	req servicecatalogsdk.ListPrivateApplicationPackagesRequest,
) (servicecatalogsdk.ListPrivateApplicationPackagesResponse, error) {
	if f.listPackagesFn != nil {
		return f.listPackagesFn(ctx, req)
	}
	return servicecatalogsdk.ListPrivateApplicationPackagesResponse{
		PrivateApplicationPackageCollection: servicecatalogsdk.PrivateApplicationPackageCollection{
			Items: []servicecatalogsdk.PrivateApplicationPackageSummary{
				{
					Id:                   common.String("ocid1.privateapplicationpackage.oc1..current"),
					PrivateApplicationId: req.PrivateApplicationId,
					Version:              common.String("1.0.0"),
					PackageType:          servicecatalogsdk.PackageTypeEnumStack,
				},
			},
		},
	}, nil
}

func (f *fakePrivateApplicationOCIClient) GetPrivateApplicationPackageActionDownloadConfig(
	ctx context.Context,
	req servicecatalogsdk.GetPrivateApplicationPackageActionDownloadConfigRequest,
) (servicecatalogsdk.GetPrivateApplicationPackageActionDownloadConfigResponse, error) {
	if f.downloadPackageConfigFn != nil {
		return f.downloadPackageConfigFn(ctx, req)
	}
	return servicecatalogsdk.GetPrivateApplicationPackageActionDownloadConfigResponse{
		Content: io.NopCloser(strings.NewReader("zip-payload")),
	}, nil
}

func (f *fakePrivateApplicationOCIClient) GetPrivateApplicationActionDownloadLogo(
	ctx context.Context,
	req servicecatalogsdk.GetPrivateApplicationActionDownloadLogoRequest,
) (servicecatalogsdk.GetPrivateApplicationActionDownloadLogoResponse, error) {
	if f.downloadLogoFn != nil {
		return f.downloadLogoFn(ctx, req)
	}
	return servicecatalogsdk.GetPrivateApplicationActionDownloadLogoResponse{
		Content: io.NopCloser(strings.NewReader("logo-payload")),
	}, nil
}

func testPrivateApplicationClient(fake *fakePrivateApplicationOCIClient) PrivateApplicationServiceClient {
	return newPrivateApplicationServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)
}

func makePrivateApplicationResource() *servicecatalogv1beta1.PrivateApplication {
	return &servicecatalogv1beta1.PrivateApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "private-app-sample",
			Namespace: "default",
			UID:       types.UID("private-app-uid"),
		},
		Spec: servicecatalogv1beta1.PrivateApplicationSpec{
			CompartmentId:    "ocid1.compartment.oc1..example",
			DisplayName:      "private-app-alpha",
			ShortDescription: "short description",
			LongDescription:  "long description",
			PackageDetails: servicecatalogv1beta1.PrivateApplicationPackageDetails{
				Version:              "1.0.0",
				PackageType:          "STACK",
				ZipFileBase64Encoded: "zip-payload",
			},
			FreeformTags: map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func makeSDKPrivateApplication(
	id string,
	compartmentID string,
	displayName string,
	state servicecatalogsdk.PrivateApplicationLifecycleStateEnum,
) servicecatalogsdk.PrivateApplication {
	return servicecatalogsdk.PrivateApplication{
		Id:               common.String(id),
		CompartmentId:    common.String(compartmentID),
		DisplayName:      common.String(displayName),
		ShortDescription: common.String("short description"),
		LongDescription:  common.String("long description"),
		PackageType:      servicecatalogsdk.PackageTypeEnumStack,
		LifecycleState:   state,
		FreeformTags:     map[string]string{"env": "dev"},
		DefinedTags:      map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	}
}

func makeSDKPrivateApplicationSummary(
	id string,
	compartmentID string,
	displayName string,
	state servicecatalogsdk.PrivateApplicationLifecycleStateEnum,
) servicecatalogsdk.PrivateApplicationSummary {
	return servicecatalogsdk.PrivateApplicationSummary{
		Id:               common.String(id),
		CompartmentId:    common.String(compartmentID),
		DisplayName:      common.String(displayName),
		ShortDescription: common.String("short description"),
		PackageType:      servicecatalogsdk.PackageTypeEnumStack,
		LifecycleState:   state,
		FreeformTags:     map[string]string{"env": "dev"},
		DefinedTags:      map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	}
}

func assertCreateRequestUsesDesiredSpec(
	t *testing.T,
	req servicecatalogsdk.CreatePrivateApplicationRequest,
	resource *servicecatalogv1beta1.PrivateApplication,
) {
	t.Helper()

	if req.CompartmentId == nil || *req.CompartmentId != resource.Spec.CompartmentId {
		t.Fatalf("create compartmentId = %v, want %q", req.CompartmentId, resource.Spec.CompartmentId)
	}
	if req.OpcRetryToken == nil || *req.OpcRetryToken != string(resource.UID) {
		t.Fatalf("create retry token = %v, want resource UID", req.OpcRetryToken)
	}
	assertCreateRequestStackPackage(t, req, "1.0.0", "zip-payload")
}

func assertCreateRequestStackPackage(
	t *testing.T,
	req servicecatalogsdk.CreatePrivateApplicationRequest,
	wantVersion string,
	wantZip string,
) {
	t.Helper()

	packageDetails, ok := req.PackageDetails.(servicecatalogsdk.CreatePrivateApplicationStackPackage)
	if !ok {
		t.Fatalf("create packageDetails type = %T, want CreatePrivateApplicationStackPackage", req.PackageDetails)
	}
	if packageDetails.Version == nil || *packageDetails.Version != wantVersion {
		t.Fatalf("create package version = %v, want %s", packageDetails.Version, wantVersion)
	}
	if packageDetails.ZipFileBase64Encoded == nil || *packageDetails.ZipFileBase64Encoded != wantZip {
		t.Fatalf("create package zip = %v, want %s", packageDetails.ZipFileBase64Encoded, wantZip)
	}
}

func assertPrivateApplicationCreateStatus(t *testing.T, resource *servicecatalogv1beta1.PrivateApplication) {
	t.Helper()

	if resource.Status.Id != "ocid1.privateapplication.oc1..created" {
		t.Fatalf("status.id = %q, want created ID", resource.Status.Id)
	}
	if string(resource.Status.OsokStatus.Ocid) != "ocid1.privateapplication.oc1..created" {
		t.Fatalf("status.ocid = %q, want created ID", resource.Status.OsokStatus.Ocid)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-create-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-create-1", resource.Status.OsokStatus.OpcRequestID)
	}
	if resource.Status.LifecycleState != "ACTIVE" {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", resource.Status.LifecycleState)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil for active create", resource.Status.OsokStatus.Async.Current)
	}
}

func assertSuccessfulNoRequeue(t *testing.T, operation string, response servicemanager.OSOKResponse) {
	t.Helper()

	if !response.IsSuccessful {
		t.Fatalf("%s should report success", operation)
	}
	if response.ShouldRequeue {
		t.Fatalf("%s should not requeue", operation)
	}
}

func pagedPrivateApplicationList(
	t *testing.T,
	listCalls *int,
	listPages *[]string,
) func(context.Context, servicecatalogsdk.ListPrivateApplicationsRequest) (servicecatalogsdk.ListPrivateApplicationsResponse, error) {
	t.Helper()

	return func(_ context.Context, req servicecatalogsdk.ListPrivateApplicationsRequest) (servicecatalogsdk.ListPrivateApplicationsResponse, error) {
		*listCalls = *listCalls + 1
		assertPrivateApplicationListRequest(t, req)
		*listPages = append(*listPages, stringValue(req.Page))
		if req.Page == nil {
			return servicecatalogsdk.ListPrivateApplicationsResponse{
				PrivateApplicationCollection: servicecatalogsdk.PrivateApplicationCollection{
					Items: []servicecatalogsdk.PrivateApplicationSummary{
						makeSDKPrivateApplicationSummary(
							"ocid1.privateapplication.oc1..other",
							"ocid1.compartment.oc1..example",
							"private-app-other",
							servicecatalogsdk.PrivateApplicationLifecycleStateActive,
						),
					},
				},
				OpcNextPage: common.String("next-page"),
			}, nil
		}
		return servicecatalogsdk.ListPrivateApplicationsResponse{
			PrivateApplicationCollection: servicecatalogsdk.PrivateApplicationCollection{
				Items: []servicecatalogsdk.PrivateApplicationSummary{
					makeSDKPrivateApplicationSummary(
						"ocid1.privateapplication.oc1..existing",
						"ocid1.compartment.oc1..example",
						"private-app-alpha",
						servicecatalogsdk.PrivateApplicationLifecycleStateActive,
					),
				},
			},
		}, nil
	}
}

func assertPrivateApplicationListRequest(t *testing.T, req servicecatalogsdk.ListPrivateApplicationsRequest) {
	t.Helper()

	if req.CompartmentId == nil || *req.CompartmentId != "ocid1.compartment.oc1..example" {
		t.Fatalf("list compartmentId = %v, want spec compartment", req.CompartmentId)
	}
	if req.DisplayName == nil || *req.DisplayName != "private-app-alpha" {
		t.Fatalf("list displayName = %v, want spec displayName", req.DisplayName)
	}
}

func mutableDriftPrivateApplicationGet(
	t *testing.T,
	getCalls *int,
) func(context.Context, servicecatalogsdk.GetPrivateApplicationRequest) (servicecatalogsdk.GetPrivateApplicationResponse, error) {
	t.Helper()

	return func(_ context.Context, req servicecatalogsdk.GetPrivateApplicationRequest) (servicecatalogsdk.GetPrivateApplicationResponse, error) {
		*getCalls = *getCalls + 1
		assertTrackedPrivateApplicationGet(t, req)
		switch *getCalls {
		case 1:
			return servicecatalogsdk.GetPrivateApplicationResponse{
				PrivateApplication: staleMutablePrivateApplication(),
			}, nil
		case 2:
			return servicecatalogsdk.GetPrivateApplicationResponse{
				PrivateApplication: makeSDKPrivateApplication(
					"ocid1.privateapplication.oc1..existing",
					"ocid1.compartment.oc1..example",
					"private-app-alpha",
					servicecatalogsdk.PrivateApplicationLifecycleStateActive,
				),
			}, nil
		default:
			t.Fatalf("unexpected GetPrivateApplication() call %d", *getCalls)
			return servicecatalogsdk.GetPrivateApplicationResponse{}, nil
		}
	}
}

func assertTrackedPrivateApplicationGet(t *testing.T, req servicecatalogsdk.GetPrivateApplicationRequest) {
	t.Helper()

	if req.PrivateApplicationId == nil || *req.PrivateApplicationId != "ocid1.privateapplication.oc1..existing" {
		t.Fatalf("get privateApplicationId = %v, want tracked ID", req.PrivateApplicationId)
	}
}

func staleMutablePrivateApplication() servicecatalogsdk.PrivateApplication {
	stale := makeSDKPrivateApplication(
		"ocid1.privateapplication.oc1..existing",
		"ocid1.compartment.oc1..example",
		"private-app-old",
		servicecatalogsdk.PrivateApplicationLifecycleStateActive,
	)
	stale.ShortDescription = common.String("old short")
	stale.LongDescription = common.String("old long")
	stale.FreeformTags = map[string]string{"env": "old"}
	return stale
}

func privateApplicationWithLogo(state servicecatalogsdk.PrivateApplicationLifecycleStateEnum) servicecatalogsdk.PrivateApplication {
	current := makeSDKPrivateApplication(
		"ocid1.privateapplication.oc1..existing",
		"ocid1.compartment.oc1..example",
		"private-app-alpha",
		state,
	)
	current.Logo = &servicecatalogsdk.UploadData{
		DisplayName: common.String("logo.png"),
		ContentUrl:  common.String("https://example.com/logo.png"),
		MimeType:    common.String("image/png"),
	}
	return current
}

func assertPrivateApplicationUpdateRequest(
	t *testing.T,
	req servicecatalogsdk.UpdatePrivateApplicationRequest,
) {
	t.Helper()

	if req.PrivateApplicationId == nil || *req.PrivateApplicationId != "ocid1.privateapplication.oc1..existing" {
		t.Fatalf("update privateApplicationId = %v, want tracked ID", req.PrivateApplicationId)
	}
	if req.DisplayName == nil || *req.DisplayName != "private-app-alpha" {
		t.Fatalf("update displayName = %v, want private-app-alpha", req.DisplayName)
	}
	if req.ShortDescription == nil || *req.ShortDescription != "short description" {
		t.Fatalf("update shortDescription = %v, want short description", req.ShortDescription)
	}
	if got := req.FreeformTags; !reflect.DeepEqual(got, map[string]string{"env": "dev"}) {
		t.Fatalf("update freeformTags = %#v, want desired tags", got)
	}
}

func lifecycleDeletePrivateApplicationGet(
	t *testing.T,
	getCalls *int,
) func(context.Context, servicecatalogsdk.GetPrivateApplicationRequest) (servicecatalogsdk.GetPrivateApplicationResponse, error) {
	t.Helper()

	return func(_ context.Context, req servicecatalogsdk.GetPrivateApplicationRequest) (servicecatalogsdk.GetPrivateApplicationResponse, error) {
		*getCalls = *getCalls + 1
		assertTrackedPrivateApplicationGet(t, req)
		state, ok := deleteReadbackStateForCall(*getCalls)
		if !ok {
			t.Fatalf("unexpected GetPrivateApplication() call %d", *getCalls)
			return servicecatalogsdk.GetPrivateApplicationResponse{}, nil
		}
		return servicecatalogsdk.GetPrivateApplicationResponse{
			PrivateApplication: makeSDKPrivateApplication(
				"ocid1.privateapplication.oc1..existing",
				"ocid1.compartment.oc1..example",
				"private-app-alpha",
				state,
			),
		}, nil
	}
}

func deleteReadbackStateForCall(call int) (servicecatalogsdk.PrivateApplicationLifecycleStateEnum, bool) {
	switch call {
	case 1, 2:
		return servicecatalogsdk.PrivateApplicationLifecycleStateActive, true
	case 3, 4:
		return servicecatalogsdk.PrivateApplicationLifecycleStateDeleting, true
	case 5:
		return servicecatalogsdk.PrivateApplicationLifecycleStateDeleted, true
	default:
		return "", false
	}
}

func assertPrivateApplicationDeleteStarted(t *testing.T, resource *servicecatalogv1beta1.PrivateApplication) {
	t.Helper()

	if resource.Status.LifecycleState != "DELETING" {
		t.Fatalf("status.lifecycleState after first delete = %q, want DELETING", resource.Status.LifecycleState)
	}
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.async.current = nil, want delete tracker")
	}
	if resource.Status.OsokStatus.Async.Current.WorkRequestID != "wr-delete-1" {
		t.Fatalf("status.async.current.workRequestId = %q, want wr-delete-1", resource.Status.OsokStatus.Async.Current.WorkRequestID)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-delete-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-delete-1", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestPrivateApplicationServiceClientCreateOrUpdateCreatesStackPackageAndRefreshesStatus(t *testing.T) {
	t.Parallel()

	var createRequest servicecatalogsdk.CreatePrivateApplicationRequest
	var getRequest servicecatalogsdk.GetPrivateApplicationRequest

	client := testPrivateApplicationClient(&fakePrivateApplicationOCIClient{
		createPrivateApplicationFn: func(_ context.Context, req servicecatalogsdk.CreatePrivateApplicationRequest) (servicecatalogsdk.CreatePrivateApplicationResponse, error) {
			createRequest = req
			return servicecatalogsdk.CreatePrivateApplicationResponse{
				OpcRequestId:     common.String("opc-create-1"),
				OpcWorkRequestId: common.String("wr-create-1"),
				PrivateApplication: makeSDKPrivateApplication(
					"ocid1.privateapplication.oc1..created",
					"ocid1.compartment.oc1..example",
					"private-app-alpha",
					servicecatalogsdk.PrivateApplicationLifecycleStateActive,
				),
			}, nil
		},
		getPrivateApplicationFn: func(_ context.Context, req servicecatalogsdk.GetPrivateApplicationRequest) (servicecatalogsdk.GetPrivateApplicationResponse, error) {
			getRequest = req
			return servicecatalogsdk.GetPrivateApplicationResponse{
				PrivateApplication: makeSDKPrivateApplication(
					"ocid1.privateapplication.oc1..created",
					"ocid1.compartment.oc1..example",
					"private-app-alpha",
					servicecatalogsdk.PrivateApplicationLifecycleStateActive,
				),
			}, nil
		},
	})

	resource := makePrivateApplicationResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	assertSuccessfulNoRequeue(t, "CreateOrUpdate()", response)
	assertCreateRequestUsesDesiredSpec(t, createRequest, resource)
	if getRequest.PrivateApplicationId == nil || *getRequest.PrivateApplicationId != "ocid1.privateapplication.oc1..created" {
		t.Fatalf("get privateApplicationId = %v, want created ID", getRequest.PrivateApplicationId)
	}
	assertPrivateApplicationCreateStatus(t, resource)
}

func TestPrivateApplicationServiceClientCreateOrUpdateBindsExistingFromLaterListPage(t *testing.T) {
	t.Parallel()

	createCalls := 0
	getCalls := 0
	listCalls := 0
	var listPages []string

	client := testPrivateApplicationClient(&fakePrivateApplicationOCIClient{
		createPrivateApplicationFn: func(_ context.Context, _ servicecatalogsdk.CreatePrivateApplicationRequest) (servicecatalogsdk.CreatePrivateApplicationResponse, error) {
			createCalls++
			return servicecatalogsdk.CreatePrivateApplicationResponse{}, nil
		},
		listPrivateApplicationsFn: pagedPrivateApplicationList(t, &listCalls, &listPages),
		getPrivateApplicationFn: func(_ context.Context, req servicecatalogsdk.GetPrivateApplicationRequest) (servicecatalogsdk.GetPrivateApplicationResponse, error) {
			getCalls++
			if req.PrivateApplicationId == nil || *req.PrivateApplicationId != "ocid1.privateapplication.oc1..existing" {
				t.Fatalf("get privateApplicationId = %v, want existing ID", req.PrivateApplicationId)
			}
			return servicecatalogsdk.GetPrivateApplicationResponse{
				PrivateApplication: makeSDKPrivateApplication(
					"ocid1.privateapplication.oc1..existing",
					"ocid1.compartment.oc1..example",
					"private-app-alpha",
					servicecatalogsdk.PrivateApplicationLifecycleStateActive,
				),
			}, nil
		},
	})

	resource := makePrivateApplicationResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if createCalls != 0 {
		t.Fatalf("CreatePrivateApplication() calls = %d, want 0", createCalls)
	}
	if listCalls != 4 {
		t.Fatalf("ListPrivateApplications() calls = %d, want 4", listCalls)
	}
	if got, want := listPages, []string{"", "next-page", "", "next-page"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("list pages = %#v, want %#v", got, want)
	}
	if getCalls != 2 {
		t.Fatalf("GetPrivateApplication() calls = %d, want 2", getCalls)
	}
	if resource.Status.Id != "ocid1.privateapplication.oc1..existing" {
		t.Fatalf("status.id = %q, want existing ID", resource.Status.Id)
	}
}

func TestPrivateApplicationServiceClientCreateOrUpdateRejectsUntrackedBindPackageZipDriftBeforeMutableUpdate(t *testing.T) {
	t.Parallel()

	getCalls := 0
	listCalls := 0
	updateCalls := 0
	var listPages []string

	client := testPrivateApplicationClient(&fakePrivateApplicationOCIClient{
		listPrivateApplicationsFn: pagedPrivateApplicationList(t, &listCalls, &listPages),
		getPrivateApplicationFn: func(_ context.Context, req servicecatalogsdk.GetPrivateApplicationRequest) (servicecatalogsdk.GetPrivateApplicationResponse, error) {
			getCalls++
			assertTrackedPrivateApplicationGet(t, req)
			return servicecatalogsdk.GetPrivateApplicationResponse{
				PrivateApplication: staleMutablePrivateApplication(),
			}, nil
		},
		updatePrivateApplicationFn: func(_ context.Context, _ servicecatalogsdk.UpdatePrivateApplicationRequest) (servicecatalogsdk.UpdatePrivateApplicationResponse, error) {
			updateCalls++
			return servicecatalogsdk.UpdatePrivateApplicationResponse{}, nil
		},
	})

	resource := makePrivateApplicationResource()
	resource.Spec.PackageDetails.ZipFileBase64Encoded = "changed-zip-payload"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want package zip drift rejection")
	}
	if !strings.Contains(err.Error(), "packageDetails.zipFileBase64Encoded") {
		t.Fatalf("CreateOrUpdate() error = %v, want packageDetails.zipFileBase64Encoded drift", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should not report success for untracked bind package drift")
	}
	if updateCalls != 0 {
		t.Fatalf("UpdatePrivateApplication() calls = %d, want 0", updateCalls)
	}
	if listCalls != 2 {
		t.Fatalf("ListPrivateApplications() calls = %d, want 2", listCalls)
	}
	if got, want := listPages, []string{"", "next-page"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("list pages = %#v, want %#v", got, want)
	}
	if getCalls != 1 {
		t.Fatalf("GetPrivateApplication() calls = %d, want 1", getCalls)
	}
}

func TestPrivateApplicationServiceClientCreateOrUpdateSkipsUpdateWhenMutableStateMatches(t *testing.T) {
	t.Parallel()

	updateCalls := 0
	client := testPrivateApplicationClient(&fakePrivateApplicationOCIClient{
		getPrivateApplicationFn: func(_ context.Context, _ servicecatalogsdk.GetPrivateApplicationRequest) (servicecatalogsdk.GetPrivateApplicationResponse, error) {
			return servicecatalogsdk.GetPrivateApplicationResponse{
				PrivateApplication: makeSDKPrivateApplication(
					"ocid1.privateapplication.oc1..existing",
					"ocid1.compartment.oc1..example",
					"private-app-alpha",
					servicecatalogsdk.PrivateApplicationLifecycleStateActive,
				),
			}, nil
		},
		updatePrivateApplicationFn: func(_ context.Context, _ servicecatalogsdk.UpdatePrivateApplicationRequest) (servicecatalogsdk.UpdatePrivateApplicationResponse, error) {
			updateCalls++
			return servicecatalogsdk.UpdatePrivateApplicationResponse{}, nil
		},
	})

	resource := makePrivateApplicationResource()
	resource.Status.Id = "ocid1.privateapplication.oc1..existing"
	resource.Status.OsokStatus.Ocid = "ocid1.privateapplication.oc1..existing"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if updateCalls != 0 {
		t.Fatalf("UpdatePrivateApplication() calls = %d, want 0", updateCalls)
	}
	if resource.Status.DisplayName != "private-app-alpha" {
		t.Fatalf("status.displayName = %q, want private-app-alpha", resource.Status.DisplayName)
	}
}

func TestPrivateApplicationServiceClientCreateOrUpdateUpdatesMutableDrift(t *testing.T) {
	t.Parallel()

	getCalls := 0
	var updateRequest servicecatalogsdk.UpdatePrivateApplicationRequest

	client := testPrivateApplicationClient(&fakePrivateApplicationOCIClient{
		getPrivateApplicationFn: mutableDriftPrivateApplicationGet(t, &getCalls),
		updatePrivateApplicationFn: func(_ context.Context, req servicecatalogsdk.UpdatePrivateApplicationRequest) (servicecatalogsdk.UpdatePrivateApplicationResponse, error) {
			updateRequest = req
			return servicecatalogsdk.UpdatePrivateApplicationResponse{
				OpcRequestId: common.String("opc-update-1"),
				PrivateApplication: makeSDKPrivateApplication(
					"ocid1.privateapplication.oc1..existing",
					"ocid1.compartment.oc1..example",
					"private-app-alpha",
					servicecatalogsdk.PrivateApplicationLifecycleStateUpdating,
				),
			}, nil
		},
	})

	resource := makePrivateApplicationResource()
	resource.Status.Id = "ocid1.privateapplication.oc1..existing"
	resource.Status.OsokStatus.Ocid = "ocid1.privateapplication.oc1..existing"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue once follow-up read sees ACTIVE")
	}
	assertPrivateApplicationUpdateRequest(t, updateRequest)
	if resource.Status.OsokStatus.OpcRequestID != "opc-update-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-update-1", resource.Status.OsokStatus.OpcRequestID)
	}
	if resource.Status.LifecycleState != "ACTIVE" {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", resource.Status.LifecycleState)
	}
}

func TestPrivateApplicationServiceClientCreateOrUpdateUpdatesChangedLogo(t *testing.T) {
	t.Parallel()

	desiredLogo := base64.StdEncoding.EncodeToString([]byte("new-logo-payload"))
	downloadLogoCalls := 0
	var updateRequest servicecatalogsdk.UpdatePrivateApplicationRequest

	client := testPrivateApplicationClient(&fakePrivateApplicationOCIClient{
		getPrivateApplicationFn: func(_ context.Context, req servicecatalogsdk.GetPrivateApplicationRequest) (servicecatalogsdk.GetPrivateApplicationResponse, error) {
			assertTrackedPrivateApplicationGet(t, req)
			return servicecatalogsdk.GetPrivateApplicationResponse{
				PrivateApplication: privateApplicationWithLogo(servicecatalogsdk.PrivateApplicationLifecycleStateActive),
			}, nil
		},
		updatePrivateApplicationFn: func(_ context.Context, req servicecatalogsdk.UpdatePrivateApplicationRequest) (servicecatalogsdk.UpdatePrivateApplicationResponse, error) {
			updateRequest = req
			updated := privateApplicationWithLogo(servicecatalogsdk.PrivateApplicationLifecycleStateActive)
			return servicecatalogsdk.UpdatePrivateApplicationResponse{
				OpcRequestId:       common.String("opc-update-logo-1"),
				PrivateApplication: updated,
			}, nil
		},
		downloadLogoFn: func(_ context.Context, req servicecatalogsdk.GetPrivateApplicationActionDownloadLogoRequest) (servicecatalogsdk.GetPrivateApplicationActionDownloadLogoResponse, error) {
			downloadLogoCalls++
			if req.PrivateApplicationId == nil || *req.PrivateApplicationId != "ocid1.privateapplication.oc1..existing" {
				t.Fatalf("download logo privateApplicationId = %v, want tracked ID", req.PrivateApplicationId)
			}
			return servicecatalogsdk.GetPrivateApplicationActionDownloadLogoResponse{
				Content: io.NopCloser(strings.NewReader("old-logo-payload")),
			}, nil
		},
	})

	resource := makePrivateApplicationResource()
	resource.Spec.LogoFileBase64Encoded = desiredLogo
	resource.Status.Id = "ocid1.privateapplication.oc1..existing"
	resource.Status.OsokStatus.Ocid = "ocid1.privateapplication.oc1..existing"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	assertSuccessfulNoRequeue(t, "CreateOrUpdate()", response)
	if downloadLogoCalls != 2 {
		t.Fatalf("GetPrivateApplicationActionDownloadLogo() calls = %d, want 2", downloadLogoCalls)
	}
	if updateRequest.LogoFileBase64Encoded == nil || *updateRequest.LogoFileBase64Encoded != desiredLogo {
		t.Fatalf("update logoFileBase64Encoded = %v, want desired logo payload", updateRequest.LogoFileBase64Encoded)
	}
}

func TestPrivateApplicationServiceClientCreateOrUpdateSkipsLogoUpdateWhenDownloadedLogoMatches(t *testing.T) {
	t.Parallel()

	updateCalls := 0
	desiredLogo := base64.StdEncoding.EncodeToString([]byte("logo-payload"))
	client := testPrivateApplicationClient(&fakePrivateApplicationOCIClient{
		getPrivateApplicationFn: func(_ context.Context, req servicecatalogsdk.GetPrivateApplicationRequest) (servicecatalogsdk.GetPrivateApplicationResponse, error) {
			assertTrackedPrivateApplicationGet(t, req)
			return servicecatalogsdk.GetPrivateApplicationResponse{
				PrivateApplication: privateApplicationWithLogo(servicecatalogsdk.PrivateApplicationLifecycleStateActive),
			}, nil
		},
		updatePrivateApplicationFn: func(_ context.Context, _ servicecatalogsdk.UpdatePrivateApplicationRequest) (servicecatalogsdk.UpdatePrivateApplicationResponse, error) {
			updateCalls++
			return servicecatalogsdk.UpdatePrivateApplicationResponse{}, nil
		},
		downloadLogoFn: func(_ context.Context, _ servicecatalogsdk.GetPrivateApplicationActionDownloadLogoRequest) (servicecatalogsdk.GetPrivateApplicationActionDownloadLogoResponse, error) {
			return servicecatalogsdk.GetPrivateApplicationActionDownloadLogoResponse{
				Content: io.NopCloser(strings.NewReader("logo-payload")),
			}, nil
		},
	})

	resource := makePrivateApplicationResource()
	resource.Spec.LogoFileBase64Encoded = desiredLogo
	resource.Status.Id = "ocid1.privateapplication.oc1..existing"
	resource.Status.OsokStatus.Ocid = "ocid1.privateapplication.oc1..existing"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	assertSuccessfulNoRequeue(t, "CreateOrUpdate()", response)
	if updateCalls != 0 {
		t.Fatalf("UpdatePrivateApplication() calls = %d, want 0 when downloaded logo matches spec", updateCalls)
	}
}

func TestPrivateApplicationServiceClientCreateOrUpdateRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	updateCalls := 0
	client := testPrivateApplicationClient(&fakePrivateApplicationOCIClient{
		getPrivateApplicationFn: func(_ context.Context, _ servicecatalogsdk.GetPrivateApplicationRequest) (servicecatalogsdk.GetPrivateApplicationResponse, error) {
			return servicecatalogsdk.GetPrivateApplicationResponse{
				PrivateApplication: makeSDKPrivateApplication(
					"ocid1.privateapplication.oc1..existing",
					"ocid1.compartment.oc1..observed",
					"private-app-alpha",
					servicecatalogsdk.PrivateApplicationLifecycleStateActive,
				),
			}, nil
		},
		updatePrivateApplicationFn: func(_ context.Context, _ servicecatalogsdk.UpdatePrivateApplicationRequest) (servicecatalogsdk.UpdatePrivateApplicationResponse, error) {
			updateCalls++
			return servicecatalogsdk.UpdatePrivateApplicationResponse{}, nil
		},
	})

	resource := makePrivateApplicationResource()
	resource.Status.Id = "ocid1.privateapplication.oc1..existing"
	resource.Status.OsokStatus.Ocid = "ocid1.privateapplication.oc1..existing"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if !strings.Contains(err.Error(), "compartmentId") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartmentId drift", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should not report success for create-only drift")
	}
	if updateCalls != 0 {
		t.Fatalf("UpdatePrivateApplication() calls = %d, want 0", updateCalls)
	}
}

func TestPrivateApplicationServiceClientCreateOrUpdateRejectsPackageVersionDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	updateCalls := 0
	client := testPrivateApplicationClient(&fakePrivateApplicationOCIClient{
		listPackagesFn: func(_ context.Context, _ servicecatalogsdk.ListPrivateApplicationPackagesRequest) (servicecatalogsdk.ListPrivateApplicationPackagesResponse, error) {
			return servicecatalogsdk.ListPrivateApplicationPackagesResponse{
				PrivateApplicationPackageCollection: servicecatalogsdk.PrivateApplicationPackageCollection{
					Items: []servicecatalogsdk.PrivateApplicationPackageSummary{
						{
							Id:                   common.String("ocid1.privateapplicationpackage.oc1..current"),
							PrivateApplicationId: common.String("ocid1.privateapplication.oc1..existing"),
							Version:              common.String("1.0.0"),
							PackageType:          servicecatalogsdk.PackageTypeEnumStack,
						},
					},
				},
			}, nil
		},
		updatePrivateApplicationFn: func(_ context.Context, _ servicecatalogsdk.UpdatePrivateApplicationRequest) (servicecatalogsdk.UpdatePrivateApplicationResponse, error) {
			updateCalls++
			return servicecatalogsdk.UpdatePrivateApplicationResponse{}, nil
		},
	})

	resource := makePrivateApplicationResource()
	resource.Spec.PackageDetails.Version = "2.0.0"
	resource.Status.Id = "ocid1.privateapplication.oc1..existing"
	resource.Status.OsokStatus.Ocid = "ocid1.privateapplication.oc1..existing"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want package version drift rejection")
	}
	if !strings.Contains(err.Error(), "packageDetails.version") {
		t.Fatalf("CreateOrUpdate() error = %v, want packageDetails.version drift", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should not report success for package version drift")
	}
	if updateCalls != 0 {
		t.Fatalf("UpdatePrivateApplication() calls = %d, want 0", updateCalls)
	}
}

func TestPrivateApplicationServiceClientCreateOrUpdateRejectsPackageZipDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	updateCalls := 0
	client := testPrivateApplicationClient(&fakePrivateApplicationOCIClient{
		downloadPackageConfigFn: func(_ context.Context, _ servicecatalogsdk.GetPrivateApplicationPackageActionDownloadConfigRequest) (servicecatalogsdk.GetPrivateApplicationPackageActionDownloadConfigResponse, error) {
			return servicecatalogsdk.GetPrivateApplicationPackageActionDownloadConfigResponse{
				Content: io.NopCloser(strings.NewReader("different-zip-payload")),
			}, nil
		},
		updatePrivateApplicationFn: func(_ context.Context, _ servicecatalogsdk.UpdatePrivateApplicationRequest) (servicecatalogsdk.UpdatePrivateApplicationResponse, error) {
			updateCalls++
			return servicecatalogsdk.UpdatePrivateApplicationResponse{}, nil
		},
	})

	resource := makePrivateApplicationResource()
	resource.Status.Id = "ocid1.privateapplication.oc1..existing"
	resource.Status.OsokStatus.Ocid = "ocid1.privateapplication.oc1..existing"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want package zip drift rejection")
	}
	if !strings.Contains(err.Error(), "packageDetails.zipFileBase64Encoded") {
		t.Fatalf("CreateOrUpdate() error = %v, want packageDetails.zipFileBase64Encoded drift", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should not report success for package zip drift")
	}
	if updateCalls != 0 {
		t.Fatalf("UpdatePrivateApplication() calls = %d, want 0", updateCalls)
	}
}

func TestPrivateApplicationServiceClientCreateOrUpdateObservesPendingLifecycleBeforePackageDriftValidation(t *testing.T) {
	t.Parallel()

	for _, state := range []servicecatalogsdk.PrivateApplicationLifecycleStateEnum{
		servicecatalogsdk.PrivateApplicationLifecycleStateCreating,
		servicecatalogsdk.PrivateApplicationLifecycleStateUpdating,
	} {
		t.Run(string(state), func(t *testing.T) {
			t.Parallel()
			runPrivateApplicationPendingLifecyclePackageDriftTest(t, state)
		})
	}
}

func runPrivateApplicationPendingLifecyclePackageDriftTest(
	t *testing.T,
	state servicecatalogsdk.PrivateApplicationLifecycleStateEnum,
) {
	t.Helper()

	getCalls := 0
	updateCalls := 0
	client := testPrivateApplicationClient(&fakePrivateApplicationOCIClient{
		getPrivateApplicationFn: pendingLifecyclePrivateApplicationGet(t, state, &getCalls),
		listPackagesFn: func(context.Context, servicecatalogsdk.ListPrivateApplicationPackagesRequest) (servicecatalogsdk.ListPrivateApplicationPackagesResponse, error) {
			t.Fatal("ListPrivateApplicationPackages() should not run while status lifecycle is pending")
			return servicecatalogsdk.ListPrivateApplicationPackagesResponse{}, nil
		},
		downloadPackageConfigFn: func(context.Context, servicecatalogsdk.GetPrivateApplicationPackageActionDownloadConfigRequest) (servicecatalogsdk.GetPrivateApplicationPackageActionDownloadConfigResponse, error) {
			t.Fatal("GetPrivateApplicationPackageActionDownloadConfig() should not run while status lifecycle is pending")
			return servicecatalogsdk.GetPrivateApplicationPackageActionDownloadConfigResponse{}, nil
		},
		updatePrivateApplicationFn: func(context.Context, servicecatalogsdk.UpdatePrivateApplicationRequest) (servicecatalogsdk.UpdatePrivateApplicationResponse, error) {
			updateCalls++
			return servicecatalogsdk.UpdatePrivateApplicationResponse{}, nil
		},
	})

	resource := pendingPackageDriftPrivateApplicationResource(state)
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success while observing pending lifecycle")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should requeue while lifecycle is pending")
	}
	if getCalls != 1 {
		t.Fatalf("GetPrivateApplication() calls = %d, want 1", getCalls)
	}
	if updateCalls != 0 {
		t.Fatalf("UpdatePrivateApplication() calls = %d, want 0 while lifecycle is pending", updateCalls)
	}
	assertPrivateApplicationAsyncPhaseForLifecycle(t, resource, state)
}

func pendingLifecyclePrivateApplicationGet(
	t *testing.T,
	state servicecatalogsdk.PrivateApplicationLifecycleStateEnum,
	getCalls *int,
) func(context.Context, servicecatalogsdk.GetPrivateApplicationRequest) (servicecatalogsdk.GetPrivateApplicationResponse, error) {
	t.Helper()

	return func(_ context.Context, req servicecatalogsdk.GetPrivateApplicationRequest) (servicecatalogsdk.GetPrivateApplicationResponse, error) {
		*getCalls = *getCalls + 1
		assertTrackedPrivateApplicationGet(t, req)
		return servicecatalogsdk.GetPrivateApplicationResponse{
			PrivateApplication: makeSDKPrivateApplication(
				"ocid1.privateapplication.oc1..existing",
				"ocid1.compartment.oc1..example",
				"private-app-alpha",
				state,
			),
		}, nil
	}
}

func pendingPackageDriftPrivateApplicationResource(
	state servicecatalogsdk.PrivateApplicationLifecycleStateEnum,
) *servicecatalogv1beta1.PrivateApplication {
	resource := makePrivateApplicationResource()
	resource.Spec.PackageDetails.Version = "2.0.0"
	resource.Spec.PackageDetails.ZipFileBase64Encoded = "changed-zip-payload"
	resource.Status.Id = "ocid1.privateapplication.oc1..existing"
	resource.Status.OsokStatus.Ocid = "ocid1.privateapplication.oc1..existing"
	resource.Status.LifecycleState = string(state)
	return resource
}

func TestPrivateApplicationServiceClientCreateOrUpdateDefersPackageDriftErrorWhenLiveLifecycleIsPending(t *testing.T) {
	t.Parallel()

	getCalls := 0
	updateCalls := 0
	client := testPrivateApplicationClient(&fakePrivateApplicationOCIClient{
		getPrivateApplicationFn: func(_ context.Context, req servicecatalogsdk.GetPrivateApplicationRequest) (servicecatalogsdk.GetPrivateApplicationResponse, error) {
			getCalls++
			assertTrackedPrivateApplicationGet(t, req)
			return servicecatalogsdk.GetPrivateApplicationResponse{
				PrivateApplication: makeSDKPrivateApplication(
					"ocid1.privateapplication.oc1..existing",
					"ocid1.compartment.oc1..example",
					"private-app-alpha",
					servicecatalogsdk.PrivateApplicationLifecycleStateUpdating,
				),
			}, nil
		},
		listPackagesFn: func(context.Context, servicecatalogsdk.ListPrivateApplicationPackagesRequest) (servicecatalogsdk.ListPrivateApplicationPackagesResponse, error) {
			return servicecatalogsdk.ListPrivateApplicationPackagesResponse{}, nil
		},
		updatePrivateApplicationFn: func(context.Context, servicecatalogsdk.UpdatePrivateApplicationRequest) (servicecatalogsdk.UpdatePrivateApplicationResponse, error) {
			updateCalls++
			return servicecatalogsdk.UpdatePrivateApplicationResponse{}, nil
		},
	})

	resource := makePrivateApplicationResource()
	resource.Spec.PackageDetails.Version = "2.0.0"
	resource.Status.Id = "ocid1.privateapplication.oc1..existing"
	resource.Status.OsokStatus.Ocid = "ocid1.privateapplication.oc1..existing"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success while observing live pending lifecycle")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should requeue while live lifecycle is pending")
	}
	if getCalls != 2 {
		t.Fatalf("GetPrivateApplication() calls = %d, want 2", getCalls)
	}
	if updateCalls != 0 {
		t.Fatalf("UpdatePrivateApplication() calls = %d, want 0 while live lifecycle is pending", updateCalls)
	}
	assertPrivateApplicationAsyncPhaseForLifecycle(t, resource, servicecatalogsdk.PrivateApplicationLifecycleStateUpdating)
}

func TestPrivateApplicationServiceClientDeleteRetainsFinalizerUntilLifecycleDeleteConfirmed(t *testing.T) {
	t.Parallel()

	getCalls := 0
	deleteCalls := 0

	client := testPrivateApplicationClient(&fakePrivateApplicationOCIClient{
		getPrivateApplicationFn: lifecycleDeletePrivateApplicationGet(t, &getCalls),
		deletePrivateApplicationFn: func(_ context.Context, req servicecatalogsdk.DeletePrivateApplicationRequest) (servicecatalogsdk.DeletePrivateApplicationResponse, error) {
			deleteCalls++
			if req.PrivateApplicationId == nil || *req.PrivateApplicationId != "ocid1.privateapplication.oc1..existing" {
				t.Fatalf("delete privateApplicationId = %v, want tracked ID", req.PrivateApplicationId)
			}
			return servicecatalogsdk.DeletePrivateApplicationResponse{
				OpcRequestId:     common.String("opc-delete-1"),
				OpcWorkRequestId: common.String("wr-delete-1"),
			}, nil
		},
	})

	resource := makePrivateApplicationResource()
	resource.Status.Id = "ocid1.privateapplication.oc1..existing"
	resource.Status.OsokStatus.Ocid = "ocid1.privateapplication.oc1..existing"

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() first call error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() first call should keep the finalizer while OCI reports DELETING")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeletePrivateApplication() calls after first delete = %d, want 1", deleteCalls)
	}
	assertPrivateApplicationDeleteStarted(t, resource)

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() second call error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() second call should release the finalizer after OCI reports DELETED")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeletePrivateApplication() calls after second delete = %d, want no reissue", deleteCalls)
	}
	if resource.Status.LifecycleState != "DELETED" {
		t.Fatalf("status.lifecycleState after second delete = %q, want DELETED", resource.Status.LifecycleState)
	}
}

func TestPrivateApplicationServiceClientDeleteWaitsForStatusCreateOrUpdateOperation(t *testing.T) {
	t.Parallel()

	for _, phase := range []shared.OSOKAsyncPhase{shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncPhaseUpdate} {
		t.Run(string(phase), func(t *testing.T) {
			t.Parallel()

			deleteCalls := 0
			getCalls := 0
			client := testPrivateApplicationClient(&fakePrivateApplicationOCIClient{
				getPrivateApplicationFn: func(_ context.Context, _ servicecatalogsdk.GetPrivateApplicationRequest) (servicecatalogsdk.GetPrivateApplicationResponse, error) {
					getCalls++
					return servicecatalogsdk.GetPrivateApplicationResponse{}, nil
				},
				deletePrivateApplicationFn: func(_ context.Context, _ servicecatalogsdk.DeletePrivateApplicationRequest) (servicecatalogsdk.DeletePrivateApplicationResponse, error) {
					deleteCalls++
					return servicecatalogsdk.DeletePrivateApplicationResponse{}, nil
				},
			})

			resource := makePrivateApplicationResource()
			resource.Status.Id = "ocid1.privateapplication.oc1..existing"
			resource.Status.OsokStatus.Ocid = "ocid1.privateapplication.oc1..existing"
			resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
				Source:          shared.OSOKAsyncSourceLifecycle,
				Phase:           phase,
				RawStatus:       strings.ToUpper(string(phase)),
				NormalizedClass: shared.OSOKAsyncClassPending,
			}

			deleted, err := client.Delete(context.Background(), resource)
			if err != nil {
				t.Fatalf("Delete() error = %v", err)
			}
			if deleted {
				t.Fatal("Delete() deleted = true, want finalizer retained")
			}
			if getCalls != 0 {
				t.Fatalf("GetPrivateApplication() calls = %d, want 0 while status async is active", getCalls)
			}
			if deleteCalls != 0 {
				t.Fatalf("DeletePrivateApplication() calls = %d, want 0 while status async is active", deleteCalls)
			}
		})
	}
}

func TestPrivateApplicationServiceClientDeleteWaitsForLiveCreateOrUpdateLifecycle(t *testing.T) {
	t.Parallel()

	for _, state := range []servicecatalogsdk.PrivateApplicationLifecycleStateEnum{
		servicecatalogsdk.PrivateApplicationLifecycleStateCreating,
		servicecatalogsdk.PrivateApplicationLifecycleStateUpdating,
	} {
		t.Run(string(state), func(t *testing.T) {
			t.Parallel()

			deleteCalls := 0
			client := testPrivateApplicationClient(&fakePrivateApplicationOCIClient{
				getPrivateApplicationFn: func(_ context.Context, _ servicecatalogsdk.GetPrivateApplicationRequest) (servicecatalogsdk.GetPrivateApplicationResponse, error) {
					return servicecatalogsdk.GetPrivateApplicationResponse{
						PrivateApplication: makeSDKPrivateApplication(
							"ocid1.privateapplication.oc1..existing",
							"ocid1.compartment.oc1..example",
							"private-app-alpha",
							state,
						),
					}, nil
				},
				deletePrivateApplicationFn: func(_ context.Context, _ servicecatalogsdk.DeletePrivateApplicationRequest) (servicecatalogsdk.DeletePrivateApplicationResponse, error) {
					deleteCalls++
					return servicecatalogsdk.DeletePrivateApplicationResponse{}, nil
				},
			})

			resource := makePrivateApplicationResource()
			resource.Status.Id = "ocid1.privateapplication.oc1..existing"
			resource.Status.OsokStatus.Ocid = "ocid1.privateapplication.oc1..existing"

			deleted, err := client.Delete(context.Background(), resource)
			if err != nil {
				t.Fatalf("Delete() error = %v", err)
			}
			if deleted {
				t.Fatal("Delete() deleted = true, want finalizer retained")
			}
			if deleteCalls != 0 {
				t.Fatalf("DeletePrivateApplication() calls = %d, want 0 while live lifecycle is %s", deleteCalls, state)
			}
			if resource.Status.LifecycleState != string(state) {
				t.Fatalf("status.lifecycleState = %q, want %s", resource.Status.LifecycleState, state)
			}
			assertPrivateApplicationAsyncPhaseForLifecycle(t, resource, state)
		})
	}
}

func assertPrivateApplicationAsyncPhaseForLifecycle(
	t *testing.T,
	resource *servicecatalogv1beta1.PrivateApplication,
	state servicecatalogsdk.PrivateApplicationLifecycleStateEnum,
) {
	t.Helper()

	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.async.current = nil, want live lifecycle tracker")
	}
	wantPhase := shared.OSOKAsyncPhaseCreate
	if state == servicecatalogsdk.PrivateApplicationLifecycleStateUpdating {
		wantPhase = shared.OSOKAsyncPhaseUpdate
	}
	if resource.Status.OsokStatus.Async.Current.Phase != wantPhase {
		t.Fatalf("status.async.current.phase = %q, want %q", resource.Status.OsokStatus.Async.Current.Phase, wantPhase)
	}
}

func TestPrivateApplicationServiceClientDeleteRejectsAuthShapedNotFound(t *testing.T) {
	t.Parallel()

	serviceErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	serviceErr.OpcRequestID = "opc-delete-error-1"

	client := testPrivateApplicationClient(&fakePrivateApplicationOCIClient{
		getPrivateApplicationFn: func(_ context.Context, _ servicecatalogsdk.GetPrivateApplicationRequest) (servicecatalogsdk.GetPrivateApplicationResponse, error) {
			return servicecatalogsdk.GetPrivateApplicationResponse{
				PrivateApplication: makeSDKPrivateApplication(
					"ocid1.privateapplication.oc1..existing",
					"ocid1.compartment.oc1..example",
					"private-app-alpha",
					servicecatalogsdk.PrivateApplicationLifecycleStateActive,
				),
			}, nil
		},
		deletePrivateApplicationFn: func(_ context.Context, _ servicecatalogsdk.DeletePrivateApplicationRequest) (servicecatalogsdk.DeletePrivateApplicationResponse, error) {
			return servicecatalogsdk.DeletePrivateApplicationResponse{}, serviceErr
		},
	})

	resource := makePrivateApplicationResource()
	resource.Status.Id = "ocid1.privateapplication.oc1..existing"
	resource.Status.OsokStatus.Ocid = "ocid1.privateapplication.oc1..existing"

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped 404 to stay fatal")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous auth-shaped not found", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped 404")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt = %v, want nil for auth-shaped 404", resource.Status.OsokStatus.DeletedAt)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-delete-error-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-delete-error-1", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestPrivateApplicationServiceClientDeleteRejectsAuthShapedPreDeleteConfirmRead(t *testing.T) {
	t.Parallel()

	deleteCalls := 0
	serviceErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	serviceErr.OpcRequestID = "opc-confirm-pre-error-1"

	client := testPrivateApplicationClient(&fakePrivateApplicationOCIClient{
		getPrivateApplicationFn: func(_ context.Context, _ servicecatalogsdk.GetPrivateApplicationRequest) (servicecatalogsdk.GetPrivateApplicationResponse, error) {
			return servicecatalogsdk.GetPrivateApplicationResponse{}, serviceErr
		},
		deletePrivateApplicationFn: func(_ context.Context, _ servicecatalogsdk.DeletePrivateApplicationRequest) (servicecatalogsdk.DeletePrivateApplicationResponse, error) {
			deleteCalls++
			return servicecatalogsdk.DeletePrivateApplicationResponse{}, nil
		},
	})

	resource := makePrivateApplicationResource()
	resource.Status.Id = "ocid1.privateapplication.oc1..existing"
	resource.Status.OsokStatus.Ocid = "ocid1.privateapplication.oc1..existing"
	resource.Status.LifecycleState = "DELETING"

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want pre-delete auth-shaped GetPrivateApplication 404 to stay fatal")
	}
	if !strings.Contains(err.Error(), "delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous confirm-read 404", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped confirm read")
	}
	if deleteCalls != 0 {
		t.Fatalf("DeletePrivateApplication() calls = %d, want 0 after auth-shaped confirm read", deleteCalls)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt = %v, want nil for auth-shaped confirm read", resource.Status.OsokStatus.DeletedAt)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-confirm-pre-error-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-confirm-pre-error-1", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestPrivateApplicationServiceClientCreateOrUpdateRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	serviceErr := errortest.NewServiceError(500, "InternalError", "create failed")
	serviceErr.OpcRequestID = "opc-error-1"

	client := testPrivateApplicationClient(&fakePrivateApplicationOCIClient{
		createPrivateApplicationFn: func(_ context.Context, _ servicecatalogsdk.CreatePrivateApplicationRequest) (servicecatalogsdk.CreatePrivateApplicationResponse, error) {
			return servicecatalogsdk.CreatePrivateApplicationResponse{}, serviceErr
		},
	})

	resource := makePrivateApplicationResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should not report success for OCI error")
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-error-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-error-1", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestPrivateApplicationBuildCreateBodySupportsPackageDetailsJSONData(t *testing.T) {
	t.Parallel()

	resource := makePrivateApplicationResource()
	resource.Spec.PackageDetails = servicecatalogv1beta1.PrivateApplicationPackageDetails{
		JsonData: `{"packageType":"stack","version":"2.0.0","zipFileBase64Encoded":"json-zip"}`,
	}

	body, err := buildPrivateApplicationCreateBody(context.Background(), resource, "")
	if err != nil {
		t.Fatalf("buildPrivateApplicationCreateBody() error = %v", err)
	}
	details, ok := body.(servicecatalogsdk.CreatePrivateApplicationDetails)
	if !ok {
		t.Fatalf("create body type = %T, want CreatePrivateApplicationDetails", body)
	}
	packageDetails, ok := details.PackageDetails.(servicecatalogsdk.CreatePrivateApplicationStackPackage)
	if !ok {
		t.Fatalf("packageDetails type = %T, want CreatePrivateApplicationStackPackage", details.PackageDetails)
	}
	if packageDetails.Version == nil || *packageDetails.Version != "2.0.0" {
		t.Fatalf("package version = %v, want 2.0.0", packageDetails.Version)
	}
	if packageDetails.ZipFileBase64Encoded == nil || *packageDetails.ZipFileBase64Encoded != "json-zip" {
		t.Fatalf("package zip = %v, want json-zip", packageDetails.ZipFileBase64Encoded)
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

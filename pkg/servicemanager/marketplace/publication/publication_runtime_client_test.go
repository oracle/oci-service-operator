/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package publication

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	marketplacesdk "github.com/oracle/oci-go-sdk/v65/marketplace"
	marketplacev1beta1 "github.com/oracle/oci-service-operator/api/marketplace/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testPublicationID      = "ocid1.publication.oc1..exampleuniqueID"
	testPublicationName    = "publication-sample"
	testCompartmentID      = "ocid1.compartment.oc1..exampleuniqueID"
	testPackageVersion     = "1.0.0"
	testPublicationImageID = "ocid1.image.oc1..exampleuniqueID"
	testOperatingSystem    = "Oracle Linux"
)

type fakePublicationOCIClient struct {
	create func(context.Context, marketplacesdk.CreatePublicationRequest) (marketplacesdk.CreatePublicationResponse, error)
	get    func(context.Context, marketplacesdk.GetPublicationRequest) (marketplacesdk.GetPublicationResponse, error)
	list   func(context.Context, marketplacesdk.ListPublicationsRequest) (marketplacesdk.ListPublicationsResponse, error)
	update func(context.Context, marketplacesdk.UpdatePublicationRequest) (marketplacesdk.UpdatePublicationResponse, error)
	delete func(context.Context, marketplacesdk.DeletePublicationRequest) (marketplacesdk.DeletePublicationResponse, error)

	createRequests []marketplacesdk.CreatePublicationRequest
	getRequests    []marketplacesdk.GetPublicationRequest
	listRequests   []marketplacesdk.ListPublicationsRequest
	updateRequests []marketplacesdk.UpdatePublicationRequest
	deleteRequests []marketplacesdk.DeletePublicationRequest
}

func (f *fakePublicationOCIClient) CreatePublication(ctx context.Context, request marketplacesdk.CreatePublicationRequest) (marketplacesdk.CreatePublicationResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.create == nil {
		return marketplacesdk.CreatePublicationResponse{}, nil
	}
	return f.create(ctx, request)
}

func (f *fakePublicationOCIClient) GetPublication(ctx context.Context, request marketplacesdk.GetPublicationRequest) (marketplacesdk.GetPublicationResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.get == nil {
		return marketplacesdk.GetPublicationResponse{}, nil
	}
	return f.get(ctx, request)
}

func (f *fakePublicationOCIClient) ListPublications(ctx context.Context, request marketplacesdk.ListPublicationsRequest) (marketplacesdk.ListPublicationsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.list == nil {
		return marketplacesdk.ListPublicationsResponse{}, nil
	}
	return f.list(ctx, request)
}

func (f *fakePublicationOCIClient) UpdatePublication(ctx context.Context, request marketplacesdk.UpdatePublicationRequest) (marketplacesdk.UpdatePublicationResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.update == nil {
		return marketplacesdk.UpdatePublicationResponse{}, nil
	}
	return f.update(ctx, request)
}

func (f *fakePublicationOCIClient) DeletePublication(ctx context.Context, request marketplacesdk.DeletePublicationRequest) (marketplacesdk.DeletePublicationResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.delete == nil {
		return marketplacesdk.DeletePublicationResponse{}, nil
	}
	return f.delete(ctx, request)
}

func TestPublicationRuntimeClientCreatesWithPolymorphicPackageDetails(t *testing.T) {
	resource := testPublicationResource()
	client := &fakePublicationOCIClient{
		list: func(context.Context, marketplacesdk.ListPublicationsRequest) (marketplacesdk.ListPublicationsResponse, error) {
			return marketplacesdk.ListPublicationsResponse{}, nil
		},
		create: func(_ context.Context, request marketplacesdk.CreatePublicationRequest) (marketplacesdk.CreatePublicationResponse, error) {
			if request.CompartmentId == nil || *request.CompartmentId != testCompartmentID {
				t.Fatalf("CreatePublicationRequest.CompartmentId = %v, want %q", request.CompartmentId, testCompartmentID)
			}
			if request.ListingType != marketplacesdk.ListingTypePartner {
				t.Fatalf("CreatePublicationRequest.ListingType = %q, want PARTNER", request.ListingType)
			}
			if request.Name == nil || *request.Name != testPublicationName {
				t.Fatalf("CreatePublicationRequest.Name = %v, want %q", request.Name, testPublicationName)
			}
			if !boolPointerValue(request.IsAgreementAcknowledged) {
				t.Fatal("CreatePublicationRequest.IsAgreementAcknowledged = false, want true")
			}
			pkg, ok := request.PackageDetails.(marketplacesdk.CreateImagePublicationPackage)
			if !ok {
				t.Fatalf("CreatePublicationRequest.PackageDetails = %T, want CreateImagePublicationPackage", request.PackageDetails)
			}
			if pkg.PackageVersion == nil || *pkg.PackageVersion != testPackageVersion {
				t.Fatalf("PackageVersion = %v, want %q", pkg.PackageVersion, testPackageVersion)
			}
			if pkg.ImageId == nil || *pkg.ImageId != testPublicationImageID {
				t.Fatalf("ImageId = %v, want %q", pkg.ImageId, testPublicationImageID)
			}
			if pkg.OperatingSystem == nil || pkg.OperatingSystem.Name == nil || *pkg.OperatingSystem.Name != testOperatingSystem {
				t.Fatalf("OperatingSystem = %#v, want %q", pkg.OperatingSystem, testOperatingSystem)
			}
			if len(pkg.Eula) != 1 {
				t.Fatalf("Eula count = %d, want 1", len(pkg.Eula))
			}
			return marketplacesdk.CreatePublicationResponse{
				Publication: sdkPublication(testPublicationID, testPublicationName, marketplacesdk.PublicationLifecycleStateActive),
			}, nil
		},
		get: func(_ context.Context, request marketplacesdk.GetPublicationRequest) (marketplacesdk.GetPublicationResponse, error) {
			if request.PublicationId == nil || *request.PublicationId != testPublicationID {
				t.Fatalf("GetPublicationRequest.PublicationId = %v, want %q", request.PublicationId, testPublicationID)
			}
			return marketplacesdk.GetPublicationResponse{
				Publication: sdkPublication(testPublicationID, testPublicationName, marketplacesdk.PublicationLifecycleStateActive),
			}, nil
		},
	}

	runtimeClient := newPublicationServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	response, err := runtimeClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = true, want false")
	}
	if len(client.listRequests) != 1 {
		t.Fatalf("ListPublications() calls = %d, want 1", len(client.listRequests))
	}
	if client.listRequests[0].ListingType != marketplacesdk.ListPublicationsListingTypePartner {
		t.Fatalf("ListPublicationsRequest.ListingType = %q, want PARTNER", client.listRequests[0].ListingType)
	}
	if len(client.listRequests[0].Name) != 0 {
		t.Fatalf("ListPublicationsRequest.Name = %#v, want empty; generatedruntime filters by response match", client.listRequests[0].Name)
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("CreatePublication() calls = %d, want 1", len(client.createRequests))
	}
	if got := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); got != testPublicationID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testPublicationID)
	}
	if got := resource.Status.LifecycleState; got != string(marketplacesdk.PublicationLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
}

func TestBuildPublicationCreateDetailsNormalizesPackageTypeDiscriminator(t *testing.T) {
	tests := []struct {
		name        string
		packageType string
		jsonData    string
	}{
		{
			name:        "lowercase spec packageType",
			packageType: "image",
			jsonData:    `{"eula":[{"eulaType":"TEXT","licenseText":"sample terms"}]}`,
		},
		{
			name:        "mixed case spec packageType",
			packageType: "ImAgE",
			jsonData:    `{"eula":[{"eulaType":"TEXT","licenseText":"sample terms"}]}`,
		},
		{
			name:        "lowercase jsonData packageType",
			packageType: "",
			jsonData:    `{"packageType":"image","eula":[{"eulaType":"TEXT","licenseText":"sample terms"}]}`,
		},
		{
			name:        "mixed case jsonData packageType",
			packageType: "",
			jsonData:    `{"packageType":"iMaGe","eula":[{"eulaType":"TEXT","licenseText":"sample terms"}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := testPublicationResource()
			resource.Spec.PackageDetails.PackageType = tt.packageType
			resource.Spec.PackageDetails.JsonData = tt.jsonData

			details, err := buildPublicationCreateDetails(context.Background(), resource, nil, resource.Namespace)
			if err != nil {
				t.Fatalf("buildPublicationCreateDetails() error = %v", err)
			}
			pkg, ok := details.PackageDetails.(marketplacesdk.CreateImagePublicationPackage)
			if !ok {
				t.Fatalf("PackageDetails = %T, want CreateImagePublicationPackage", details.PackageDetails)
			}
			if pkg.PackageVersion == nil || *pkg.PackageVersion != testPackageVersion {
				t.Fatalf("PackageVersion = %v, want %q", pkg.PackageVersion, testPackageVersion)
			}
			if pkg.ImageId == nil || *pkg.ImageId != testPublicationImageID {
				t.Fatalf("ImageId = %v, want %q", pkg.ImageId, testPublicationImageID)
			}
		})
	}
}

func TestPublicationRuntimeClientBindsExistingPublicationFromList(t *testing.T) {
	resource := testPublicationResource()
	client := &fakePublicationOCIClient{
		list: func(context.Context, marketplacesdk.ListPublicationsRequest) (marketplacesdk.ListPublicationsResponse, error) {
			return marketplacesdk.ListPublicationsResponse{
				Items: []marketplacesdk.PublicationSummary{
					sdkPublicationSummary(testPublicationID, testPublicationName, marketplacesdk.PublicationLifecycleStateActive),
				},
			}, nil
		},
		get: func(_ context.Context, request marketplacesdk.GetPublicationRequest) (marketplacesdk.GetPublicationResponse, error) {
			if request.PublicationId == nil || *request.PublicationId != testPublicationID {
				t.Fatalf("GetPublicationRequest.PublicationId = %v, want %q", request.PublicationId, testPublicationID)
			}
			return marketplacesdk.GetPublicationResponse{
				Publication: sdkPublication(testPublicationID, testPublicationName, marketplacesdk.PublicationLifecycleStateActive),
			}, nil
		},
	}

	runtimeClient := newPublicationServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	response, err := runtimeClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = true, want false")
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("CreatePublication() calls = %d, want 0 on bind path", len(client.createRequests))
	}
	if got := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); got != testPublicationID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testPublicationID)
	}
}

func TestPublicationRuntimeClientUpdatesMutableFieldsOnly(t *testing.T) {
	resource := testPublicationResource()
	resource.Spec.Name = "publication-renamed"
	resource.Status.OsokStatus.Ocid = shared.OCID(testPublicationID)

	getCalls := 0
	client := &fakePublicationOCIClient{
		get: func(_ context.Context, request marketplacesdk.GetPublicationRequest) (marketplacesdk.GetPublicationResponse, error) {
			getCalls++
			if request.PublicationId == nil || *request.PublicationId != testPublicationID {
				t.Fatalf("GetPublicationRequest.PublicationId = %v, want %q", request.PublicationId, testPublicationID)
			}
			name := testPublicationName
			if getCalls > 1 {
				name = resource.Spec.Name
			}
			return marketplacesdk.GetPublicationResponse{
				Publication: sdkPublication(testPublicationID, name, marketplacesdk.PublicationLifecycleStateActive),
			}, nil
		},
		update: func(_ context.Context, request marketplacesdk.UpdatePublicationRequest) (marketplacesdk.UpdatePublicationResponse, error) {
			if request.PublicationId == nil || *request.PublicationId != testPublicationID {
				t.Fatalf("UpdatePublicationRequest.PublicationId = %v, want %q", request.PublicationId, testPublicationID)
			}
			if request.Name == nil || *request.Name != resource.Spec.Name {
				t.Fatalf("UpdatePublicationRequest.Name = %v, want %q", request.Name, resource.Spec.Name)
			}
			if request.ShortDescription != nil {
				t.Fatalf("UpdatePublicationRequest.ShortDescription = %v, want nil when unchanged", request.ShortDescription)
			}
			return marketplacesdk.UpdatePublicationResponse{
				Publication: sdkPublication(testPublicationID, resource.Spec.Name, marketplacesdk.PublicationLifecycleStateActive),
			}, nil
		},
	}

	runtimeClient := newPublicationServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	response, err := runtimeClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if len(client.updateRequests) != 1 {
		t.Fatalf("UpdatePublication() calls = %d, want 1", len(client.updateRequests))
	}
}

func TestPublicationRuntimeClientRejectsForceNewDriftBeforeMutation(t *testing.T) {
	resource := testPublicationResource()
	resource.Spec.ListingType = string(marketplacesdk.ListingTypePrivate)
	resource.Status.OsokStatus.Ocid = shared.OCID(testPublicationID)

	client := &fakePublicationOCIClient{
		get: func(context.Context, marketplacesdk.GetPublicationRequest) (marketplacesdk.GetPublicationResponse, error) {
			return marketplacesdk.GetPublicationResponse{
				Publication: sdkPublication(testPublicationID, testPublicationName, marketplacesdk.PublicationLifecycleStateActive),
			}, nil
		},
	}

	runtimeClient := newPublicationServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	response, err := runtimeClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want force-new drift failure")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = true, want false")
	}
	if !strings.Contains(err.Error(), "listingType changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want listingType force-new failure", err)
	}
	if len(client.updateRequests) != 0 || len(client.createRequests) != 0 {
		t.Fatalf("OCI mutations after drift = create:%d update:%d, want 0", len(client.createRequests), len(client.updateRequests))
	}
}

func TestPublicationRuntimeClientRequeuesWhileCreating(t *testing.T) {
	resource := testPublicationResource()
	client := &fakePublicationOCIClient{
		list: func(context.Context, marketplacesdk.ListPublicationsRequest) (marketplacesdk.ListPublicationsResponse, error) {
			return marketplacesdk.ListPublicationsResponse{}, nil
		},
		create: func(context.Context, marketplacesdk.CreatePublicationRequest) (marketplacesdk.CreatePublicationResponse, error) {
			return marketplacesdk.CreatePublicationResponse{
				Publication: sdkPublication(testPublicationID, testPublicationName, marketplacesdk.PublicationLifecycleStateCreating),
			}, nil
		},
		get: func(context.Context, marketplacesdk.GetPublicationRequest) (marketplacesdk.GetPublicationResponse, error) {
			return marketplacesdk.GetPublicationResponse{
				Publication: sdkPublication(testPublicationID, testPublicationName, marketplacesdk.PublicationLifecycleStateCreating),
			}, nil
		},
	}

	runtimeClient := newPublicationServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	response, err := runtimeClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true while provisioning")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = false, want true while provisioning")
	}
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.async.current = nil, want lifecycle tracker")
	}
	if got := resource.Status.OsokStatus.Async.Current.RawStatus; got != string(marketplacesdk.PublicationLifecycleStateCreating) {
		t.Fatalf("status.async.current.rawStatus = %q, want CREATING", got)
	}
}

func TestPublicationRuntimeClientFailedLifecycleMarksFailure(t *testing.T) {
	resource := testPublicationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testPublicationID)

	client := &fakePublicationOCIClient{
		get: func(context.Context, marketplacesdk.GetPublicationRequest) (marketplacesdk.GetPublicationResponse, error) {
			return marketplacesdk.GetPublicationResponse{
				Publication: sdkPublication(testPublicationID, testPublicationName, marketplacesdk.PublicationLifecycleStateFailed),
			}, nil
		},
	}

	runtimeClient := newPublicationServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	response, err := runtimeClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = true, want false for FAILED lifecycle")
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Failed) {
		t.Fatalf("status.reason = %q, want %q", got, shared.Failed)
	}
}

func TestPublicationRuntimeClientDeleteConfirmsOnReadbackNotFound(t *testing.T) {
	resource := testPublicationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testPublicationID)

	getCalls := 0
	client := &fakePublicationOCIClient{
		get: func(context.Context, marketplacesdk.GetPublicationRequest) (marketplacesdk.GetPublicationResponse, error) {
			getCalls++
			if getCalls == 1 {
				return marketplacesdk.GetPublicationResponse{
					Publication: sdkPublication(testPublicationID, testPublicationName, marketplacesdk.PublicationLifecycleStateActive),
				}, nil
			}
			return marketplacesdk.GetPublicationResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "publication gone")
		},
		delete: func(_ context.Context, request marketplacesdk.DeletePublicationRequest) (marketplacesdk.DeletePublicationResponse, error) {
			if request.PublicationId == nil || *request.PublicationId != testPublicationID {
				t.Fatalf("DeletePublicationRequest.PublicationId = %v, want %q", request.PublicationId, testPublicationID)
			}
			return marketplacesdk.DeletePublicationResponse{}, nil
		},
	}

	runtimeClient := newPublicationServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	deleted, err := runtimeClient.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("DeletePublication() calls = %d, want 1", len(client.deleteRequests))
	}
}

func boolPointerValue(value *bool) bool {
	return value != nil && *value
}

func testPublicationResource() *marketplacev1beta1.Publication {
	return &marketplacev1beta1.Publication{
		Spec: marketplacev1beta1.PublicationSpec{
			ListingType:             string(marketplacesdk.ListingTypePartner),
			Name:                    testPublicationName,
			ShortDescription:        "short description",
			LongDescription:         "long description",
			CompartmentId:           testCompartmentID,
			IsAgreementAcknowledged: true,
			SupportContacts:         []marketplacev1beta1.PublicationSupportContact{{Name: "Support", Email: "support@example.com", Subject: "Help"}},
			DefinedTags:             nil,
			FreeformTags:            map[string]string{"owner": "osok"},
			PackageDetails:          testPublicationPackageDetails(),
		},
	}
}

func testPublicationPackageDetails() marketplacev1beta1.PublicationPackageDetails {
	return marketplacev1beta1.PublicationPackageDetails{
		JsonData:       `{"packageType":"IMAGE","eula":[{"eulaType":"TEXT","licenseText":"sample terms"}]}`,
		PackageVersion: testPackageVersion,
		OperatingSystem: marketplacev1beta1.PublicationPackageDetailsOperatingSystem{
			Name: testOperatingSystem,
		},
		PackageType: string(marketplacesdk.PackageTypeEnumImage),
		ImageId:     testPublicationImageID,
	}
}

func sdkPublication(id string, name string, lifecycleState marketplacesdk.PublicationLifecycleStateEnum) marketplacesdk.Publication {
	return marketplacesdk.Publication{
		Id:                        common.String(id),
		Name:                      common.String(name),
		CompartmentId:             common.String(testCompartmentID),
		ListingType:               marketplacesdk.ListingTypePartner,
		LifecycleState:            lifecycleState,
		ShortDescription:          common.String("short description"),
		LongDescription:           common.String("long description"),
		SupportContacts:           []marketplacesdk.SupportContact{{Name: common.String("Support"), Email: common.String("support@example.com"), Subject: common.String("Help")}},
		PackageType:               marketplacesdk.PackageTypeEnumImage,
		SupportedOperatingSystems: []marketplacesdk.OperatingSystem{{Name: common.String(testOperatingSystem)}},
		FreeformTags:              map[string]string{"owner": "osok"},
	}
}

func sdkPublicationSummary(id string, name string, lifecycleState marketplacesdk.PublicationLifecycleStateEnum) marketplacesdk.PublicationSummary {
	return marketplacesdk.PublicationSummary{
		Id:                        common.String(id),
		Name:                      common.String(name),
		CompartmentId:             common.String(testCompartmentID),
		ListingType:               marketplacesdk.ListingTypePartner,
		LifecycleState:            lifecycleState,
		ShortDescription:          common.String("short description"),
		PackageType:               marketplacesdk.PackageTypeEnumImage,
		SupportedOperatingSystems: []marketplacesdk.OperatingSystem{{Name: common.String(testOperatingSystem)}},
		FreeformTags:              map[string]string{"owner": "osok"},
	}
}

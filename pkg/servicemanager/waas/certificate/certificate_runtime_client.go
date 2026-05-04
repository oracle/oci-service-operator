/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package certificate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	waassdk "github.com/oracle/oci-go-sdk/v65/waas"
	waasv1beta1 "github.com/oracle/oci-service-operator/api/waas/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	certificateKind                     = "Certificate"
	certificateCreateOnlyFingerprintKey = "waasCertificateCreateOnlySHA256="
)

type certificateOCIClient interface {
	ChangeCertificateCompartment(context.Context, waassdk.ChangeCertificateCompartmentRequest) (waassdk.ChangeCertificateCompartmentResponse, error)
	CreateCertificate(context.Context, waassdk.CreateCertificateRequest) (waassdk.CreateCertificateResponse, error)
	GetCertificate(context.Context, waassdk.GetCertificateRequest) (waassdk.GetCertificateResponse, error)
	ListCertificates(context.Context, waassdk.ListCertificatesRequest) (waassdk.ListCertificatesResponse, error)
	UpdateCertificate(context.Context, waassdk.UpdateCertificateRequest) (waassdk.UpdateCertificateResponse, error)
	DeleteCertificate(context.Context, waassdk.DeleteCertificateRequest) (waassdk.DeleteCertificateResponse, error)
}

type certificateCompartmentMoveClient interface {
	ChangeCertificateCompartment(context.Context, waassdk.ChangeCertificateCompartmentRequest) (waassdk.ChangeCertificateCompartmentResponse, error)
}

type certificateIdentity struct {
	compartmentID string
	displayName   string
}

type certificateReadListRequest struct {
	CompartmentId *string
	Id            *string
	DisplayName   *string
}

type certificateCreateOnlyTrackingClient struct {
	delegate CertificateServiceClient
}

type certificateDeleteConfirmationClient struct {
	delegate    CertificateServiceClient
	confirmRead func(context.Context, *waasv1beta1.Certificate, string) (any, error)
}

type certificateAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e certificateAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e certificateAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerCertificateRuntimeHooksMutator(func(manager *CertificateServiceManager, hooks *CertificateRuntimeHooks) {
		moveClient, initErr := newCertificateCompartmentMoveClient(manager)
		applyCertificateRuntimeHooks(hooks, moveClient, initErr)
	})
}

func newCertificateCompartmentMoveClient(manager *CertificateServiceManager) (certificateCompartmentMoveClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("certificate manager is nil")
	}
	client, err := waassdk.NewWaasClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyCertificateRuntimeHooks(
	hooks *CertificateRuntimeHooks,
	moveClient certificateCompartmentMoveClient,
	moveInitErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = certificateRuntimeSemantics()
	hooks.BuildCreateBody = buildCertificateCreateBody
	hooks.BuildUpdateBody = buildCertificateUpdateBody
	hooks.Identity.Resolve = resolveCertificateIdentity
	hooks.Identity.GuardExistingBeforeCreate = guardCertificateExistingBeforeCreate
	hooks.Identity.LookupExisting = func(ctx context.Context, resource *waasv1beta1.Certificate, identity any) (any, error) {
		return lookupExistingCertificate(ctx, hooks, resource, identity)
	}
	hooks.Read.List = certificateReadListOperation(hooks)
	hooks.List.Call = listCertificatesAllPages(hooks.List.Call)
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateCertificateCreateOnlyDrift
	hooks.ParityHooks.RequiresParityHandling = certificateRequiresCompartmentMove
	hooks.ParityHooks.ApplyParityUpdate = func(
		ctx context.Context,
		resource *waasv1beta1.Certificate,
		currentResponse any,
	) (servicemanager.OSOKResponse, error) {
		return applyCertificateCompartmentMove(ctx, resource, currentResponse, moveClient, moveInitErr)
	}
	hooks.DeleteHooks.ConfirmRead = func(ctx context.Context, resource *waasv1beta1.Certificate, currentID string) (any, error) {
		return confirmCertificateDeleteRead(ctx, hooks, resource, currentID)
	}
	hooks.DeleteHooks.HandleError = handleCertificateDeleteError
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate CertificateServiceClient) CertificateServiceClient {
		return certificateCreateOnlyTrackingClient{
			delegate: certificateDeleteConfirmationClient{
				delegate:    delegate,
				confirmRead: hooks.DeleteHooks.ConfirmRead,
			},
		}
	})
}

func newCertificateServiceClientWithOCIClient(client certificateOCIClient) CertificateServiceClient {
	manager := &CertificateServiceManager{}
	hooks := newCertificateRuntimeHooksWithOCIClient(client)
	applyCertificateRuntimeHooks(&hooks, client, nil)
	delegate := defaultCertificateServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*waasv1beta1.Certificate](
			buildCertificateGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapCertificateGeneratedClient(hooks, delegate)
}

func newCertificateRuntimeHooksWithOCIClient(client certificateOCIClient) CertificateRuntimeHooks {
	hooks := newCertificateDefaultRuntimeHooks(waassdk.WaasClient{})
	hooks.Create.Call = func(ctx context.Context, request waassdk.CreateCertificateRequest) (waassdk.CreateCertificateResponse, error) {
		if client == nil {
			return waassdk.CreateCertificateResponse{}, fmt.Errorf("certificate OCI client is not configured")
		}
		return client.CreateCertificate(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request waassdk.GetCertificateRequest) (waassdk.GetCertificateResponse, error) {
		if client == nil {
			return waassdk.GetCertificateResponse{}, fmt.Errorf("certificate OCI client is not configured")
		}
		return client.GetCertificate(ctx, request)
	}
	hooks.List.Call = func(ctx context.Context, request waassdk.ListCertificatesRequest) (waassdk.ListCertificatesResponse, error) {
		if client == nil {
			return waassdk.ListCertificatesResponse{}, fmt.Errorf("certificate OCI client is not configured")
		}
		return client.ListCertificates(ctx, request)
	}
	hooks.Update.Call = func(ctx context.Context, request waassdk.UpdateCertificateRequest) (waassdk.UpdateCertificateResponse, error) {
		if client == nil {
			return waassdk.UpdateCertificateResponse{}, fmt.Errorf("certificate OCI client is not configured")
		}
		return client.UpdateCertificate(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request waassdk.DeleteCertificateRequest) (waassdk.DeleteCertificateResponse, error) {
		if client == nil {
			return waassdk.DeleteCertificateResponse{}, fmt.Errorf("certificate OCI client is not configured")
		}
		return client.DeleteCertificate(ctx, request)
	}
	return hooks
}

func certificateRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "waas",
		FormalSlug:        "certificate",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(waassdk.LifecycleStatesCreating)},
			UpdatingStates:     []string{string(waassdk.LifecycleStatesUpdating)},
			ActiveStates:       []string{string(waassdk.LifecycleStatesActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy: "required",
			PendingStates: []string{
				string(waassdk.LifecycleStatesCreating),
				string(waassdk.LifecycleStatesUpdating),
				string(waassdk.LifecycleStatesDeleting),
			},
			TerminalStates: []string{string(waassdk.LifecycleStatesDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{"compartmentId", "displayName", "freeformTags", "definedTags"},
			Mutable:         []string{"compartmentId", "displayName", "freeformTags", "definedTags"},
			ForceNew: []string{
				"certificateData",
				"privateKeyData",
				"isTrustVerificationDisabled",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "Certificate", Action: "CreateCertificate"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "Certificate", Action: "UpdateCertificate"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "Certificate", Action: "DeleteCertificate"}},
		},
		CreateFollowUp:      generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp:      generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp:      generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func buildCertificateCreateBody(_ context.Context, resource *waasv1beta1.Certificate, _ string) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("%s resource is nil", certificateKind)
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" {
		return nil, fmt.Errorf("%s spec.compartmentId is required", certificateKind)
	}
	if strings.TrimSpace(resource.Spec.CertificateData) == "" {
		return nil, fmt.Errorf("%s spec.certificateData is required", certificateKind)
	}
	if strings.TrimSpace(resource.Spec.PrivateKeyData) == "" {
		return nil, fmt.Errorf("%s spec.privateKeyData is required", certificateKind)
	}

	body := waassdk.CreateCertificateDetails{
		CompartmentId:   common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		CertificateData: common.String(resource.Spec.CertificateData),
		PrivateKeyData:  common.String(resource.Spec.PrivateKeyData),
	}
	if value := strings.TrimSpace(resource.Spec.DisplayName); value != "" {
		body.DisplayName = common.String(value)
	}
	if resource.Spec.IsTrustVerificationDisabled {
		body.IsTrustVerificationDisabled = common.Bool(resource.Spec.IsTrustVerificationDisabled)
	}
	if resource.Spec.FreeformTags != nil {
		body.FreeformTags = cloneCertificateStringMap(resource.Spec.FreeformTags)
	}
	if resource.Spec.DefinedTags != nil {
		body.DefinedTags = certificateDefinedTags(resource.Spec.DefinedTags)
	}
	return body, nil
}

func buildCertificateUpdateBody(
	_ context.Context,
	resource *waasv1beta1.Certificate,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("%s resource is nil", certificateKind)
	}
	current, ok := certificateBodyFromResponse(currentResponse)
	if !ok {
		return nil, false, fmt.Errorf("current %s response does not expose a Certificate body", certificateKind)
	}

	body := waassdk.UpdateCertificateDetails{}
	updateNeeded := certificateCompartmentNeedsMove(resource.Spec, current)
	if desired, ok := desiredCertificateStringUpdate(resource.Spec.DisplayName, current.DisplayName); ok {
		body.DisplayName = desired
		updateNeeded = true
	}
	if resource.Spec.FreeformTags != nil {
		desired := cloneCertificateStringMap(resource.Spec.FreeformTags)
		body.FreeformTags = desired
		if !reflect.DeepEqual(current.FreeformTags, desired) {
			updateNeeded = true
		}
	}
	if resource.Spec.DefinedTags != nil {
		desired := certificateDefinedTags(resource.Spec.DefinedTags)
		body.DefinedTags = desired
		if !reflect.DeepEqual(current.DefinedTags, desired) {
			updateNeeded = true
		}
	}
	return body, updateNeeded, nil
}

func desiredCertificateStringUpdate(desired string, current *string) (*string, bool) {
	value := strings.TrimSpace(desired)
	if value == "" {
		return nil, false
	}
	if current != nil && strings.TrimSpace(*current) == value {
		return nil, false
	}
	return common.String(value), true
}

func resolveCertificateIdentity(resource *waasv1beta1.Certificate) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("resolve %s identity: resource is nil", certificateKind)
	}
	identity := certificateIdentity{
		compartmentID: strings.TrimSpace(resource.Spec.CompartmentId),
		displayName:   strings.TrimSpace(resource.Spec.DisplayName),
	}
	if identity.compartmentID == "" {
		return nil, fmt.Errorf("resolve %s identity: compartmentId is required", certificateKind)
	}
	return identity, nil
}

func guardCertificateExistingBeforeCreate(
	_ context.Context,
	resource *waasv1beta1.Certificate,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if trackedCertificateID(resource) != "" {
		return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
	}
	if resource == nil || strings.TrimSpace(resource.Spec.CompartmentId) == "" || strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func lookupExistingCertificate(
	ctx context.Context,
	hooks *CertificateRuntimeHooks,
	_ *waasv1beta1.Certificate,
	identity any,
) (any, error) {
	typed, ok := identity.(certificateIdentity)
	if !ok {
		return nil, fmt.Errorf("resolve %s identity: expected certificateIdentity, got %T", certificateKind, identity)
	}
	if hooks == nil || hooks.List.Call == nil || typed.displayName == "" {
		return nil, nil
	}
	matches, err := listMatchingCertificates(ctx, hooks.List.Call, typed, certificateUsableForBind)
	if err != nil {
		return nil, err
	}
	return singleCertificateMatch(matches, typed)
}

func certificateReadListOperation(hooks *CertificateRuntimeHooks) *generatedruntime.Operation {
	return &generatedruntime.Operation{
		NewRequest: func() any { return &certificateReadListRequest{} },
		Fields: []generatedruntime.RequestField{
			{
				FieldName:    "CompartmentId",
				RequestName:  "compartmentId",
				Contribution: "query",
				LookupPaths:  []string{"status.compartmentId", "spec.compartmentId", "compartmentId"},
			},
			{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
			{
				FieldName:    "DisplayName",
				RequestName:  "displayName",
				Contribution: "query",
				LookupPaths:  []string{"status.displayName", "spec.displayName", "displayName"},
			},
		},
		Call: func(ctx context.Context, request any) (any, error) {
			typed, ok := request.(*certificateReadListRequest)
			if !ok {
				return nil, fmt.Errorf("expected *certificateReadListRequest, got %T", request)
			}
			if hooks == nil || hooks.List.Call == nil {
				return nil, certificateNotFoundError("certificate list client is not configured")
			}
			if typed.CompartmentId == nil || strings.TrimSpace(*typed.CompartmentId) == "" {
				return nil, certificateNotFoundError("certificate list requires compartmentId")
			}
			listRequest := waassdk.ListCertificatesRequest{
				CompartmentId: common.String(strings.TrimSpace(*typed.CompartmentId)),
			}
			if typed.Id != nil && strings.TrimSpace(*typed.Id) != "" {
				listRequest.Id = []string{strings.TrimSpace(*typed.Id)}
			}
			if typed.DisplayName != nil && strings.TrimSpace(*typed.DisplayName) != "" {
				listRequest.DisplayName = []string{strings.TrimSpace(*typed.DisplayName)}
			}
			return hooks.List.Call(ctx, listRequest)
		},
	}
}

func confirmCertificateDeleteRead(
	ctx context.Context,
	hooks *CertificateRuntimeHooks,
	resource *waasv1beta1.Certificate,
	currentID string,
) (any, error) {
	currentID = firstCertificateNonEmpty(currentID, trackedCertificateID(resource))
	if currentID != "" {
		if hooks == nil || hooks.Get.Call == nil {
			return nil, certificateNotFoundError("certificate get client is not configured")
		}
		return hooks.Get.Call(ctx, waassdk.GetCertificateRequest{CertificateId: common.String(currentID)})
	}

	identity, err := resolveCertificateDeleteIdentity(resource)
	if err != nil {
		return nil, err
	}
	if identity.displayName == "" {
		return nil, certificateNotFoundError("certificate delete confirmation has no recorded id or displayName")
	}
	if hooks == nil || hooks.List.Call == nil {
		return nil, certificateNotFoundError("certificate list client is not configured")
	}
	matches, err := listMatchingCertificates(ctx, hooks.List.Call, identity, func(waassdk.CertificateSummary) bool { return true })
	if err != nil {
		return nil, err
	}
	return singleCertificateMatch(matches, identity)
}

func resolveCertificateDeleteIdentity(resource *waasv1beta1.Certificate) (certificateIdentity, error) {
	if resource == nil {
		return certificateIdentity{}, fmt.Errorf("resolve %s delete identity: resource is nil", certificateKind)
	}
	compartmentID := firstCertificateNonEmpty(resource.Status.CompartmentId, resource.Spec.CompartmentId)
	if compartmentID == "" {
		return certificateIdentity{}, certificateNotFoundError("certificate delete confirmation has no compartmentId")
	}
	return certificateIdentity{
		compartmentID: strings.TrimSpace(compartmentID),
		displayName:   firstCertificateNonEmpty(resource.Status.DisplayName, resource.Spec.DisplayName),
	}, nil
}

func listMatchingCertificates(
	ctx context.Context,
	list func(context.Context, waassdk.ListCertificatesRequest) (waassdk.ListCertificatesResponse, error),
	identity certificateIdentity,
	usable func(waassdk.CertificateSummary) bool,
) ([]waassdk.CertificateSummary, error) {
	var matches []waassdk.CertificateSummary
	var page *string
	for {
		response, err := list(ctx, waassdk.ListCertificatesRequest{
			CompartmentId: common.String(identity.compartmentID),
			DisplayName:   []string{identity.displayName},
			Page:          page,
		})
		if err != nil {
			return nil, err
		}
		for _, item := range response.Items {
			if certificateSummaryMatchesIdentity(item, identity) && usable(item) {
				matches = append(matches, item)
			}
		}
		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			return matches, nil
		}
		page = response.OpcNextPage
	}
}

func listCertificatesAllPages(
	call func(context.Context, waassdk.ListCertificatesRequest) (waassdk.ListCertificatesResponse, error),
) func(context.Context, waassdk.ListCertificatesRequest) (waassdk.ListCertificatesResponse, error) {
	return func(ctx context.Context, request waassdk.ListCertificatesRequest) (waassdk.ListCertificatesResponse, error) {
		if call == nil {
			return waassdk.ListCertificatesResponse{}, fmt.Errorf("certificate list client is not configured")
		}
		var combined waassdk.ListCertificatesResponse
		for {
			response, err := call(ctx, request)
			if err != nil {
				return waassdk.ListCertificatesResponse{}, err
			}
			combined.RawResponse = response.RawResponse
			combined.OpcRequestId = response.OpcRequestId
			combined.Items = append(combined.Items, response.Items...)
			if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
				combined.OpcNextPage = nil
				return combined, nil
			}
			request.Page = response.OpcNextPage
			combined.OpcNextPage = response.OpcNextPage
		}
	}
}

func certificateSummaryMatchesIdentity(summary waassdk.CertificateSummary, identity certificateIdentity) bool {
	return strings.TrimSpace(certificateStringValue(summary.CompartmentId)) == identity.compartmentID &&
		strings.TrimSpace(certificateStringValue(summary.DisplayName)) == identity.displayName
}

func certificateUsableForBind(summary waassdk.CertificateSummary) bool {
	switch summary.LifecycleState {
	case waassdk.LifecycleStatesDeleting, waassdk.LifecycleStatesDeleted, waassdk.LifecycleStatesFailed:
		return false
	default:
		return true
	}
}

func singleCertificateMatch(matches []waassdk.CertificateSummary, identity certificateIdentity) (any, error) {
	switch len(matches) {
	case 0:
		return nil, certificateNotFoundError("Certificate list returned no matching resource")
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf(
			"certificate list returned multiple matching resources for compartmentId %q and displayName %q",
			identity.compartmentID,
			identity.displayName,
		)
	}
}

func (c certificateCreateOnlyTrackingClient) CreateOrUpdate(
	ctx context.Context,
	resource *waasv1beta1.Certificate,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	recorded, hasRecorded := certificateRecordedCreateOnlyFingerprint(resource)
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	switch {
	case hasRecorded && certificateHasTrackedIdentity(resource):
		setCertificateCreateOnlyFingerprint(resource, recorded)
	case err == nil && response.IsSuccessful && certificateHasTrackedIdentity(resource):
		recordCertificateCreateOnlyFingerprint(resource)
	}
	return response, err
}

func (c certificateCreateOnlyTrackingClient) Delete(ctx context.Context, resource *waasv1beta1.Certificate) (bool, error) {
	return c.delegate.Delete(ctx, resource)
}

func (c certificateDeleteConfirmationClient) CreateOrUpdate(
	ctx context.Context,
	resource *waasv1beta1.Certificate,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c certificateDeleteConfirmationClient) Delete(ctx context.Context, resource *waasv1beta1.Certificate) (bool, error) {
	if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c certificateDeleteConfirmationClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *waasv1beta1.Certificate,
) error {
	if c.confirmRead == nil || resource == nil || !certificateHasTrackedIdentity(resource) {
		return nil
	}
	_, err := c.confirmRead(ctx, resource, trackedCertificateID(resource))
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("certificate delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to call delete: %w", err)
}

func handleCertificateDeleteError(resource *waasv1beta1.Certificate, err error) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	requestID := errorutil.OpcRequestID(err)
	if resource != nil {
		servicemanager.SetOpcRequestID(&resource.Status.OsokStatus, requestID)
	}
	return certificateAmbiguousNotFoundError{
		message:      "Certificate delete returned ambiguous 404 NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed",
		opcRequestID: requestID,
	}
}

func certificateRequiresCompartmentMove(resource *waasv1beta1.Certificate, currentResponse any) bool {
	if resource == nil {
		return false
	}
	current, ok := certificateBodyFromResponse(currentResponse)
	if !ok {
		return false
	}
	return certificateCompartmentNeedsMove(resource.Spec, current)
}

func certificateCompartmentNeedsMove(spec waasv1beta1.CertificateSpec, current waassdk.Certificate) bool {
	desired := strings.TrimSpace(spec.CompartmentId)
	observed := strings.TrimSpace(certificateStringValue(current.CompartmentId))
	return desired != "" && observed != "" && desired != observed
}

func applyCertificateCompartmentMove(
	ctx context.Context,
	resource *waasv1beta1.Certificate,
	currentResponse any,
	client certificateCompartmentMoveClient,
	initErr error,
) (servicemanager.OSOKResponse, error) {
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("%s resource is nil", certificateKind)
	}
	if initErr != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("initialize certificate OCI client: %w", initErr)
	}
	if client == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("certificate OCI client is not configured")
	}

	current, ok := certificateBodyFromResponse(currentResponse)
	if !ok {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("current %s response does not expose a Certificate body", certificateKind)
	}
	resourceID := firstCertificateNonEmpty(certificateStringValue(current.Id), trackedCertificateID(resource))
	if resourceID == "" {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("certificate compartment move requires a tracked certificate id")
	}
	compartmentID := strings.TrimSpace(resource.Spec.CompartmentId)
	if compartmentID == "" {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("certificate compartment move requires spec.compartmentId")
	}

	response, err := client.ChangeCertificateCompartment(ctx, waassdk.ChangeCertificateCompartmentRequest{
		CertificateId: common.String(resourceID),
		ChangeCertificateCompartmentDetails: waassdk.ChangeCertificateCompartmentDetails{
			CompartmentId: common.String(compartmentID),
		},
		OpcRetryToken: certificateMoveRetryToken(resource, compartmentID),
	})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	resource.Status.Id = resourceID
	resource.Status.OsokStatus.Ocid = shared.OCID(resourceID)
	return markCertificateCompartmentMovePending(resource), nil
}

func certificateMoveRetryToken(resource *waasv1beta1.Certificate, compartmentID string) *string {
	if resource == nil {
		return nil
	}
	seed := strings.TrimSpace(string(resource.UID))
	if seed == "" {
		seed = strings.TrimSpace(resource.Namespace) + "/" + strings.TrimSpace(resource.Name)
	}
	if strings.Trim(seed, "/") == "" {
		return nil
	}
	sum := sha256.Sum256([]byte(seed + "/move/" + strings.TrimSpace(compartmentID)))
	return common.String(hex.EncodeToString(sum[:16]))
}

func markCertificateCompartmentMovePending(resource *waasv1beta1.Certificate) servicemanager.OSOKResponse {
	now := metav1.Now()
	projection := servicemanager.ApplyAsyncOperation(
		&resource.Status.OsokStatus,
		&shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           shared.OSOKAsyncPhaseUpdate,
			RawStatus:       string(waassdk.LifecycleStatesUpdating),
			NormalizedClass: shared.OSOKAsyncClassPending,
			Message:         "OCI Certificate compartment move is in progress",
			UpdatedAt:       &now,
		},
		loggerutil.OSOKLogger{},
	)
	return servicemanager.OSOKResponse{
		IsSuccessful:    projection.Condition != shared.Failed,
		ShouldRequeue:   projection.ShouldRequeue,
		RequeueDuration: time.Minute,
	}
}

func validateCertificateCreateOnlyDrift(resource *waasv1beta1.Certificate, _ any) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", certificateKind)
	}
	recorded, ok := certificateRecordedCreateOnlyFingerprint(resource)
	if !ok {
		if certificateHasEstablishedTrackedIdentity(resource) {
			return fmt.Errorf("%s create-only fingerprint is missing for tracked resource; recreate the resource before changing create-only fields", certificateKind)
		}
		return nil
	}
	desired, err := certificateCreateOnlyFingerprint(resource.Spec)
	if err != nil {
		return err
	}
	if desired != recorded {
		return fmt.Errorf("%s formal semantics require replacement when create-only fields change", certificateKind)
	}
	return nil
}

func recordCertificateCreateOnlyFingerprint(resource *waasv1beta1.Certificate) {
	if resource == nil {
		return
	}
	fingerprint, err := certificateCreateOnlyFingerprint(resource.Spec)
	if err != nil {
		return
	}
	setCertificateCreateOnlyFingerprint(resource, fingerprint)
}

func setCertificateCreateOnlyFingerprint(resource *waasv1beta1.Certificate, fingerprint string) {
	if resource == nil {
		return
	}
	base := stripCertificateCreateOnlyFingerprint(resource.Status.OsokStatus.Message)
	marker := certificateCreateOnlyFingerprintKey + fingerprint
	if base == "" {
		resource.Status.OsokStatus.Message = marker
		return
	}
	resource.Status.OsokStatus.Message = base + "; " + marker
}

func certificateCreateOnlyFingerprint(spec waasv1beta1.CertificateSpec) (string, error) {
	payload := struct {
		CertificateData             string `json:"certificateData"`
		PrivateKeyData              string `json:"privateKeyData"`
		IsTrustVerificationDisabled bool   `json:"isTrustVerificationDisabled"`
	}{
		CertificateData:             spec.CertificateData,
		PrivateKeyData:              spec.PrivateKeyData,
		IsTrustVerificationDisabled: spec.IsTrustVerificationDisabled,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal %s create-only fingerprint: %w", certificateKind, err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func certificateRecordedCreateOnlyFingerprint(resource *waasv1beta1.Certificate) (string, bool) {
	if resource == nil {
		return "", false
	}
	raw := resource.Status.OsokStatus.Message
	index := strings.LastIndex(raw, certificateCreateOnlyFingerprintKey)
	if index < 0 {
		return "", false
	}
	start := index + len(certificateCreateOnlyFingerprintKey)
	end := start
	for end < len(raw) && certificateIsHexDigit(raw[end]) {
		end++
	}
	fingerprint := raw[start:end]
	if len(fingerprint) != sha256.Size*2 {
		return "", false
	}
	if _, err := hex.DecodeString(fingerprint); err != nil {
		return "", false
	}
	return fingerprint, true
}

func stripCertificateCreateOnlyFingerprint(raw string) string {
	raw = strings.TrimSpace(raw)
	index := strings.LastIndex(raw, certificateCreateOnlyFingerprintKey)
	if index < 0 {
		return raw
	}
	prefix := strings.TrimSpace(strings.TrimRight(raw[:index], "; "))
	start := index + len(certificateCreateOnlyFingerprintKey)
	end := start
	for end < len(raw) && certificateIsHexDigit(raw[end]) {
		end++
	}
	suffix := strings.TrimSpace(strings.TrimLeft(raw[end:], "; "))
	switch {
	case prefix == "":
		return suffix
	case suffix == "":
		return prefix
	default:
		return prefix + "; " + suffix
	}
}

func certificateIsHexDigit(value byte) bool {
	return (value >= '0' && value <= '9') ||
		(value >= 'a' && value <= 'f') ||
		(value >= 'A' && value <= 'F')
}

func certificateHasTrackedIdentity(resource *waasv1beta1.Certificate) bool {
	return trackedCertificateID(resource) != ""
}

func certificateHasEstablishedTrackedIdentity(resource *waasv1beta1.Certificate) bool {
	return certificateHasTrackedIdentity(resource) &&
		resource != nil &&
		resource.Status.OsokStatus.CreatedAt != nil
}

func trackedCertificateID(resource *waasv1beta1.Certificate) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func certificateBodyFromResponse(response any) (waassdk.Certificate, bool) {
	if body, ok := directCertificateBodyFromResponse(response); ok {
		return body, true
	}
	return wrappedCertificateBodyFromResponse(response)
}

func directCertificateBodyFromResponse(response any) (waassdk.Certificate, bool) {
	switch current := response.(type) {
	case waassdk.Certificate:
		return current, true
	case *waassdk.Certificate:
		if current == nil {
			return waassdk.Certificate{}, false
		}
		return *current, true
	case waassdk.CertificateSummary:
		return certificateFromSummary(current), true
	case *waassdk.CertificateSummary:
		if current == nil {
			return waassdk.Certificate{}, false
		}
		return certificateFromSummary(*current), true
	default:
		return waassdk.Certificate{}, false
	}
}

func wrappedCertificateBodyFromResponse(response any) (waassdk.Certificate, bool) {
	switch current := response.(type) {
	case waassdk.CreateCertificateResponse:
		return current.Certificate, true
	case *waassdk.CreateCertificateResponse:
		if current == nil {
			return waassdk.Certificate{}, false
		}
		return current.Certificate, true
	case waassdk.GetCertificateResponse:
		return current.Certificate, true
	case *waassdk.GetCertificateResponse:
		if current == nil {
			return waassdk.Certificate{}, false
		}
		return current.Certificate, true
	case waassdk.UpdateCertificateResponse:
		return current.Certificate, true
	case *waassdk.UpdateCertificateResponse:
		if current == nil {
			return waassdk.Certificate{}, false
		}
		return current.Certificate, true
	default:
		return waassdk.Certificate{}, false
	}
}

func certificateFromSummary(summary waassdk.CertificateSummary) waassdk.Certificate {
	return waassdk.Certificate{
		Id:                summary.Id,
		CompartmentId:     summary.CompartmentId,
		DisplayName:       summary.DisplayName,
		TimeNotValidAfter: summary.TimeNotValidAfter,
		FreeformTags:      cloneCertificateStringMap(summary.FreeformTags),
		DefinedTags:       cloneCertificateDefinedTagMap(summary.DefinedTags),
		LifecycleState:    summary.LifecycleState,
		TimeCreated:       summary.TimeCreated,
	}
}

func certificateNotFoundError(message string) error {
	_, err := errorutil.NewServiceFailureFromResponse(errorutil.NotFound, 404, "", message)
	if err != nil {
		return err
	}
	return fmt.Errorf("%s", message)
}

func cloneCertificateStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func certificateDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(tags))
	for namespace, values := range tags {
		converted[namespace] = make(map[string]interface{}, len(values))
		for key, value := range values {
			converted[namespace][key] = value
		}
	}
	return converted
}

func cloneCertificateDefinedTagMap(source map[string]map[string]interface{}) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	clone := make(map[string]map[string]interface{}, len(source))
	for namespace, values := range source {
		clone[namespace] = make(map[string]interface{}, len(values))
		for key, value := range values {
			clone[namespace][key] = value
		}
	}
	return clone
}

func certificateStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func firstCertificateNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

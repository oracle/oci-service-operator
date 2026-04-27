/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package odaprivateendpointattachment

import (
	"context"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	odasdk "github.com/oracle/oci-go-sdk/v65/oda"
	odav1beta1 "github.com/oracle/oci-service-operator/api/oda/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

type odaPrivateEndpointAttachmentOCIClient interface {
	CreateOdaPrivateEndpointAttachment(context.Context, odasdk.CreateOdaPrivateEndpointAttachmentRequest) (odasdk.CreateOdaPrivateEndpointAttachmentResponse, error)
	GetOdaPrivateEndpointAttachment(context.Context, odasdk.GetOdaPrivateEndpointAttachmentRequest) (odasdk.GetOdaPrivateEndpointAttachmentResponse, error)
	ListOdaPrivateEndpointAttachments(context.Context, odasdk.ListOdaPrivateEndpointAttachmentsRequest) (odasdk.ListOdaPrivateEndpointAttachmentsResponse, error)
	DeleteOdaPrivateEndpointAttachment(context.Context, odasdk.DeleteOdaPrivateEndpointAttachmentRequest) (odasdk.DeleteOdaPrivateEndpointAttachmentResponse, error)
	GetOdaPrivateEndpoint(context.Context, odasdk.GetOdaPrivateEndpointRequest) (odasdk.GetOdaPrivateEndpointResponse, error)
}

type odaPrivateEndpointAttachmentParentReader interface {
	GetOdaPrivateEndpoint(context.Context, odasdk.GetOdaPrivateEndpointRequest) (odasdk.GetOdaPrivateEndpointResponse, error)
}

type odaPrivateEndpointAttachmentIdentity struct {
	odaInstanceID        string
	odaPrivateEndpointID string
	compartmentID        string
}

func init() {
	registerOdaPrivateEndpointAttachmentRuntimeHooksMutator(func(manager *OdaPrivateEndpointAttachmentServiceManager, hooks *OdaPrivateEndpointAttachmentRuntimeHooks) {
		applyOdaPrivateEndpointAttachmentRuntimeHooks(manager, hooks)
	})
}

func applyOdaPrivateEndpointAttachmentRuntimeHooks(manager *OdaPrivateEndpointAttachmentServiceManager, hooks *OdaPrivateEndpointAttachmentRuntimeHooks) {
	parentReader, parentReaderErr := newOdaPrivateEndpointAttachmentParentReader(manager)
	applyOdaPrivateEndpointAttachmentRuntimeHookConfig(hooks, parentReader, parentReaderErr)
}

func applyOdaPrivateEndpointAttachmentRuntimeHookConfig(
	hooks *OdaPrivateEndpointAttachmentRuntimeHooks,
	parentReader odaPrivateEndpointAttachmentParentReader,
	parentReaderErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newOdaPrivateEndpointAttachmentRuntimeSemantics()
	hooks.BuildCreateBody = buildOdaPrivateEndpointAttachmentCreateBody
	hooks.Identity = generatedruntime.IdentityHooks[*odav1beta1.OdaPrivateEndpointAttachment]{
		Resolve:        resolveOdaPrivateEndpointAttachmentIdentity,
		RecordPath:     recordOdaPrivateEndpointAttachmentPathIdentity,
		LookupExisting: lookupOdaPrivateEndpointAttachmentExisting(parentReader, parentReaderErr),
	}
	hooks.TrackedRecreate = generatedruntime.TrackedRecreateHooks[*odav1beta1.OdaPrivateEndpointAttachment]{
		ClearTrackedIdentity: clearOdaPrivateEndpointAttachmentTrackedIdentity,
	}
}

func newOdaPrivateEndpointAttachmentParentReader(manager *OdaPrivateEndpointAttachmentServiceManager) (odaPrivateEndpointAttachmentParentReader, error) {
	if manager == nil || manager.Provider == nil {
		return nil, nil
	}

	client, err := odasdk.NewManagementClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, fmt.Errorf("initialize OdaPrivateEndpointAttachment parent OCI client: %w", err)
	}
	return client, nil
}

func newOdaPrivateEndpointAttachmentRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "oda",
		FormalSlug:    "odaprivateendpointattachment",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(odasdk.OdaPrivateEndpointAttachmentLifecycleStateCreating)},
			UpdatingStates:     []string{string(odasdk.OdaPrivateEndpointAttachmentLifecycleStateUpdating)},
			ActiveStates:       []string{string(odasdk.OdaPrivateEndpointAttachmentLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(odasdk.OdaPrivateEndpointAttachmentLifecycleStateDeleting)},
			TerminalStates: []string{string(odasdk.OdaPrivateEndpointAttachmentLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"odaPrivateEndpointId", "odaInstanceId"},
		},
		Mutation: generatedruntime.MutationSemantics{
			ForceNew:      []string{"odaInstanceId", "odaPrivateEndpointId"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func buildOdaPrivateEndpointAttachmentCreateBody(_ context.Context, resource *odav1beta1.OdaPrivateEndpointAttachment, _ string) (any, error) {
	identity, err := resolveOdaPrivateEndpointAttachmentIdentity(resource)
	if err != nil {
		return nil, err
	}
	attachmentIdentity := identity.(odaPrivateEndpointAttachmentIdentity)
	return odasdk.CreateOdaPrivateEndpointAttachmentDetails{
		OdaInstanceId:        common.String(attachmentIdentity.odaInstanceID),
		OdaPrivateEndpointId: common.String(attachmentIdentity.odaPrivateEndpointID),
	}, nil
}

func resolveOdaPrivateEndpointAttachmentIdentity(resource *odav1beta1.OdaPrivateEndpointAttachment) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("OdaPrivateEndpointAttachment resource is nil")
	}

	identity := odaPrivateEndpointAttachmentIdentity{
		odaInstanceID:        strings.TrimSpace(resource.Spec.OdaInstanceId),
		odaPrivateEndpointID: strings.TrimSpace(resource.Spec.OdaPrivateEndpointId),
		compartmentID:        strings.TrimSpace(resource.Status.CompartmentId),
	}
	if identity.odaInstanceID == "" {
		return nil, fmt.Errorf("OdaPrivateEndpointAttachment requires spec.odaInstanceId")
	}
	if identity.odaPrivateEndpointID == "" {
		return nil, fmt.Errorf("OdaPrivateEndpointAttachment requires spec.odaPrivateEndpointId")
	}
	if err := validateOdaPrivateEndpointAttachmentCreateOnlyStatus("odaInstanceId", resource.Status.OdaInstanceId, identity.odaInstanceID); err != nil {
		return nil, err
	}
	if err := validateOdaPrivateEndpointAttachmentCreateOnlyStatus("odaPrivateEndpointId", resource.Status.OdaPrivateEndpointId, identity.odaPrivateEndpointID); err != nil {
		return nil, err
	}
	return identity, nil
}

func validateOdaPrivateEndpointAttachmentCreateOnlyStatus(field string, statusValue string, specValue string) error {
	statusValue = strings.TrimSpace(statusValue)
	if statusValue == "" || statusValue == specValue {
		return nil
	}
	return fmt.Errorf("OdaPrivateEndpointAttachment cannot change %s from %q to %q; create a replacement resource instead", field, statusValue, specValue)
}

func recordOdaPrivateEndpointAttachmentPathIdentity(resource *odav1beta1.OdaPrivateEndpointAttachment, identity any) {
	attachmentIdentity, ok := identity.(odaPrivateEndpointAttachmentIdentity)
	if !ok || resource == nil {
		return
	}
	resource.Status.OdaInstanceId = attachmentIdentity.odaInstanceID
	resource.Status.OdaPrivateEndpointId = attachmentIdentity.odaPrivateEndpointID
	if attachmentIdentity.compartmentID != "" {
		resource.Status.CompartmentId = attachmentIdentity.compartmentID
	}
}

func lookupOdaPrivateEndpointAttachmentExisting(
	parentReader odaPrivateEndpointAttachmentParentReader,
	parentReaderErr error,
) func(context.Context, *odav1beta1.OdaPrivateEndpointAttachment, any) (any, error) {
	return func(ctx context.Context, resource *odav1beta1.OdaPrivateEndpointAttachment, identity any) (any, error) {
		attachmentIdentity, ok := identity.(odaPrivateEndpointAttachmentIdentity)
		if !ok {
			return nil, fmt.Errorf("OdaPrivateEndpointAttachment identity has unexpected type %T", identity)
		}
		if attachmentIdentity.compartmentID != "" {
			return nil, nil
		}
		if parentReaderErr != nil {
			return nil, parentReaderErr
		}
		if parentReader == nil {
			return nil, fmt.Errorf("OdaPrivateEndpointAttachment requires a parent OdaPrivateEndpoint lookup client to resolve compartmentId")
		}

		response, err := parentReader.GetOdaPrivateEndpoint(ctx, odasdk.GetOdaPrivateEndpointRequest{
			OdaPrivateEndpointId: common.String(attachmentIdentity.odaPrivateEndpointID),
		})
		if err != nil {
			return nil, err
		}
		compartmentID := strings.TrimSpace(odaPrivateEndpointAttachmentStringValue(response.OdaPrivateEndpoint.CompartmentId))
		if compartmentID == "" {
			return nil, fmt.Errorf("OdaPrivateEndpointAttachment parent OdaPrivateEndpoint %q did not include compartmentId", attachmentIdentity.odaPrivateEndpointID)
		}
		resource.Status.CompartmentId = compartmentID
		return nil, nil
	}
}

func odaPrivateEndpointAttachmentStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func clearOdaPrivateEndpointAttachmentTrackedIdentity(resource *odav1beta1.OdaPrivateEndpointAttachment) {
	if resource == nil {
		return
	}
	resource.Status.Id = ""
	resource.Status.OsokStatus.Ocid = shared.OCID("")
}

func newOdaPrivateEndpointAttachmentServiceClientWithOCIClient(client odaPrivateEndpointAttachmentOCIClient) OdaPrivateEndpointAttachmentServiceClient {
	hooks := newOdaPrivateEndpointAttachmentRuntimeHooksWithOCIClient(client)
	applyOdaPrivateEndpointAttachmentRuntimeHookConfig(&hooks, client, nil)
	delegate := defaultOdaPrivateEndpointAttachmentServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*odav1beta1.OdaPrivateEndpointAttachment](
			buildOdaPrivateEndpointAttachmentGeneratedRuntimeConfig(&OdaPrivateEndpointAttachmentServiceManager{}, hooks),
		),
	}
	return wrapOdaPrivateEndpointAttachmentGeneratedClient(hooks, delegate)
}

func newOdaPrivateEndpointAttachmentRuntimeHooksWithOCIClient(client odaPrivateEndpointAttachmentOCIClient) OdaPrivateEndpointAttachmentRuntimeHooks {
	hooks := newOdaPrivateEndpointAttachmentDefaultRuntimeHooks(odasdk.ManagementClient{})
	if client == nil {
		return hooks
	}

	hooks.Create.Call = func(ctx context.Context, request odasdk.CreateOdaPrivateEndpointAttachmentRequest) (odasdk.CreateOdaPrivateEndpointAttachmentResponse, error) {
		return client.CreateOdaPrivateEndpointAttachment(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request odasdk.GetOdaPrivateEndpointAttachmentRequest) (odasdk.GetOdaPrivateEndpointAttachmentResponse, error) {
		return client.GetOdaPrivateEndpointAttachment(ctx, request)
	}
	hooks.List.Call = func(ctx context.Context, request odasdk.ListOdaPrivateEndpointAttachmentsRequest) (odasdk.ListOdaPrivateEndpointAttachmentsResponse, error) {
		return client.ListOdaPrivateEndpointAttachments(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request odasdk.DeleteOdaPrivateEndpointAttachmentRequest) (odasdk.DeleteOdaPrivateEndpointAttachmentResponse, error) {
		return client.DeleteOdaPrivateEndpointAttachment(ctx, request)
	}
	return hooks
}

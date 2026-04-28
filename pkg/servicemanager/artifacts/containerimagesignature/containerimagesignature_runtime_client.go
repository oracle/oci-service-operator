/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package containerimagesignature

import (
	"context"
	"fmt"
	"reflect"

	artifactssdk "github.com/oracle/oci-go-sdk/v65/artifacts"
	"github.com/oracle/oci-go-sdk/v65/common"
	artifactsv1beta1 "github.com/oracle/oci-service-operator/api/artifacts/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
)

type containerImageSignatureIdentity struct {
	compartmentID     string
	imageID           string
	kmsKeyID          string
	kmsKeyVersionID   string
	message           string
	signature         string
	signingAlgorithm  string
	listAlgorithmEnum artifactssdk.ListContainerImageSignaturesSigningAlgorithmEnum
}

func init() {
	registerContainerImageSignatureRuntimeHooksMutator(func(_ *ContainerImageSignatureServiceManager, hooks *ContainerImageSignatureRuntimeHooks) {
		applyContainerImageSignatureRuntimeHooks(hooks)
	})
}

func applyContainerImageSignatureRuntimeHooks(hooks *ContainerImageSignatureRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newContainerImageSignatureRuntimeSemantics()
	hooks.BuildCreateBody = func(_ context.Context, resource *artifactsv1beta1.ContainerImageSignature, _ string) (any, error) {
		return buildContainerImageSignatureCreateBody(resource)
	}
	hooks.BuildUpdateBody = func(_ context.Context, resource *artifactsv1beta1.ContainerImageSignature, _ string, currentResponse any) (any, bool, error) {
		return buildContainerImageSignatureUpdateBody(resource, currentResponse)
	}
	hooks.Identity.Resolve = resolveContainerImageSignatureIdentity
	hooks.List.Fields = containerImageSignatureListFields()
	hooks.Read.List = &generatedruntime.Operation{
		NewRequest: func() any { return &artifactssdk.ListContainerImageSignaturesRequest{} },
		Fields:     containerImageSignatureListFields(),
		Call: func(ctx context.Context, request any) (any, error) {
			return listAllContainerImageSignatures(ctx, *request.(*artifactssdk.ListContainerImageSignaturesRequest), hooks.List.Call)
		},
	}
	hooks.DeleteHooks.HandleError = handleContainerImageSignatureDeleteError
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, wrapContainerImageSignatureSigningAlgorithmClient)
}

func newContainerImageSignatureRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "artifacts",
		FormalSlug:    "containerimagesignature",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ActiveStates: []string{string(artifactssdk.ContainerImageSignatureLifecycleStateAvailable)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy: "required",
			PendingStates: []string{
				string(artifactssdk.ContainerImageSignatureLifecycleStateDeleting),
			},
			TerminalStates: []string{
				string(artifactssdk.ContainerImageSignatureLifecycleStateDeleted),
			},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"compartmentId",
				"imageId",
				"kmsKeyId",
				"kmsKeyVersionId",
				"message",
				"signature",
				"signingAlgorithm",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"definedTags",
				"freeformTags",
			},
			ForceNew: []string{
				"compartmentId",
				"imageId",
				"kmsKeyId",
				"kmsKeyVersionId",
				"message",
				"signature",
				"signingAlgorithm",
			},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "ContainerImageSignature", Action: "CreateContainerImageSignature"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "ContainerImageSignature", Action: "UpdateContainerImageSignature"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "ContainerImageSignature", Action: "DeleteContainerImageSignature"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "ContainerImageSignature", Action: "GetContainerImageSignature"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "ContainerImageSignature", Action: "GetContainerImageSignature"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "ContainerImageSignature", Action: "GetContainerImageSignature"}},
		},
	}
}

func buildContainerImageSignatureCreateBody(resource *artifactsv1beta1.ContainerImageSignature) (artifactssdk.CreateContainerImageSignatureDetails, error) {
	if resource == nil {
		return artifactssdk.CreateContainerImageSignatureDetails{}, fmt.Errorf("ContainerImageSignature resource is nil")
	}
	algorithm, ok := artifactssdk.GetMappingCreateContainerImageSignatureDetailsSigningAlgorithmEnum(resource.Spec.SigningAlgorithm)
	if !ok {
		return artifactssdk.CreateContainerImageSignatureDetails{}, fmt.Errorf("unsupported ContainerImageSignature signingAlgorithm %q", resource.Spec.SigningAlgorithm)
	}

	details := artifactssdk.CreateContainerImageSignatureDetails{
		CompartmentId:    common.String(resource.Spec.CompartmentId),
		ImageId:          common.String(resource.Spec.ImageId),
		KmsKeyId:         common.String(resource.Spec.KmsKeyId),
		KmsKeyVersionId:  common.String(resource.Spec.KmsKeyVersionId),
		Message:          common.String(resource.Spec.Message),
		Signature:        common.String(resource.Spec.Signature),
		SigningAlgorithm: algorithm,
	}
	if resource.Spec.FreeformTags != nil {
		details.FreeformTags = cloneStringMap(resource.Spec.FreeformTags)
	}
	if resource.Spec.DefinedTags != nil {
		details.DefinedTags = definedTagsForOCI(resource.Spec.DefinedTags)
	}
	return details, nil
}

func buildContainerImageSignatureUpdateBody(
	resource *artifactsv1beta1.ContainerImageSignature,
	currentResponse any,
) (artifactssdk.UpdateContainerImageSignatureDetails, bool, error) {
	if resource == nil {
		return artifactssdk.UpdateContainerImageSignatureDetails{}, false, fmt.Errorf("ContainerImageSignature resource is nil")
	}
	current, ok := containerImageSignatureFromResponse(currentResponse)
	if !ok {
		return artifactssdk.UpdateContainerImageSignatureDetails{}, false, fmt.Errorf("current ContainerImageSignature response does not expose a ContainerImageSignature body")
	}

	details := artifactssdk.UpdateContainerImageSignatureDetails{}
	updateNeeded := false

	if resource.Spec.FreeformTags != nil && !reflect.DeepEqual(current.FreeformTags, resource.Spec.FreeformTags) {
		details.FreeformTags = cloneStringMap(resource.Spec.FreeformTags)
		updateNeeded = true
	}
	if resource.Spec.DefinedTags != nil {
		desiredDefinedTags := definedTagsForOCI(resource.Spec.DefinedTags)
		if !reflect.DeepEqual(current.DefinedTags, desiredDefinedTags) {
			details.DefinedTags = desiredDefinedTags
			updateNeeded = true
		}
	}
	return details, updateNeeded, nil
}

func resolveContainerImageSignatureIdentity(resource *artifactsv1beta1.ContainerImageSignature) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("ContainerImageSignature resource is nil")
	}
	listAlgorithm, ok := artifactssdk.GetMappingListContainerImageSignaturesSigningAlgorithmEnum(resource.Spec.SigningAlgorithm)
	if !ok {
		return nil, fmt.Errorf("unsupported ContainerImageSignature signingAlgorithm %q", resource.Spec.SigningAlgorithm)
	}
	canonicalAlgorithm, ok := canonicalContainerImageSignatureSigningAlgorithm(resource.Spec.SigningAlgorithm)
	if !ok {
		return nil, fmt.Errorf("unsupported ContainerImageSignature signingAlgorithm %q", resource.Spec.SigningAlgorithm)
	}
	return containerImageSignatureIdentity{
		compartmentID:     resource.Spec.CompartmentId,
		imageID:           resource.Spec.ImageId,
		kmsKeyID:          resource.Spec.KmsKeyId,
		kmsKeyVersionID:   resource.Spec.KmsKeyVersionId,
		message:           resource.Spec.Message,
		signature:         resource.Spec.Signature,
		signingAlgorithm:  canonicalAlgorithm,
		listAlgorithmEnum: listAlgorithm,
	}, nil
}

type containerImageSignatureSigningAlgorithmClient struct {
	delegate ContainerImageSignatureServiceClient
}

func wrapContainerImageSignatureSigningAlgorithmClient(delegate ContainerImageSignatureServiceClient) ContainerImageSignatureServiceClient {
	return containerImageSignatureSigningAlgorithmClient{delegate: delegate}
}

func (c containerImageSignatureSigningAlgorithmClient) CreateOrUpdate(
	ctx context.Context,
	resource *artifactsv1beta1.ContainerImageSignature,
	request ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if resource == nil {
		return c.delegate.CreateOrUpdate(ctx, resource, request)
	}

	algorithm, ok := canonicalContainerImageSignatureSigningAlgorithm(resource.Spec.SigningAlgorithm)
	if !ok || algorithm == resource.Spec.SigningAlgorithm {
		return c.delegate.CreateOrUpdate(ctx, resource, request)
	}

	original := resource.Spec.SigningAlgorithm
	resource.Spec.SigningAlgorithm = algorithm
	defer func() {
		resource.Spec.SigningAlgorithm = original
	}()
	return c.delegate.CreateOrUpdate(ctx, resource, request)
}

func (c containerImageSignatureSigningAlgorithmClient) Delete(
	ctx context.Context,
	resource *artifactsv1beta1.ContainerImageSignature,
) (bool, error) {
	return c.delegate.Delete(ctx, resource)
}

func canonicalContainerImageSignatureSigningAlgorithm(value string) (string, bool) {
	algorithm, ok := artifactssdk.GetMappingContainerImageSignatureSigningAlgorithmEnum(value)
	if !ok {
		return "", false
	}
	return string(algorithm), true
}

func listAllContainerImageSignatures(
	ctx context.Context,
	request artifactssdk.ListContainerImageSignaturesRequest,
	list func(context.Context, artifactssdk.ListContainerImageSignaturesRequest) (artifactssdk.ListContainerImageSignaturesResponse, error),
) (artifactssdk.ListContainerImageSignaturesResponse, error) {
	if list == nil {
		return artifactssdk.ListContainerImageSignaturesResponse{}, fmt.Errorf("ContainerImageSignature list operation is not configured")
	}

	var aggregate artifactssdk.ListContainerImageSignaturesResponse
	for {
		response, err := list(ctx, request)
		if err != nil {
			return artifactssdk.ListContainerImageSignaturesResponse{}, err
		}
		if aggregate.OpcRequestId == nil {
			aggregate.OpcRequestId = response.OpcRequestId
		}
		aggregate.Items = append(aggregate.Items, response.Items...)
		if response.OpcNextPage == nil || *response.OpcNextPage == "" {
			return aggregate, nil
		}
		request.Page = response.OpcNextPage
	}
}

func containerImageSignatureListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
		{FieldName: "ImageId", RequestName: "imageId", Contribution: "query"},
		{FieldName: "KmsKeyId", RequestName: "kmsKeyId", Contribution: "query"},
		{FieldName: "KmsKeyVersionId", RequestName: "kmsKeyVersionId", Contribution: "query"},
		{FieldName: "SigningAlgorithm", RequestName: "signingAlgorithm", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
	}
}

//nolint:gocyclo // Response normalization must handle each OCI wrapper shape generated for this resource.
func containerImageSignatureFromResponse(response any) (artifactssdk.ContainerImageSignature, bool) {
	switch typed := response.(type) {
	case artifactssdk.ContainerImageSignature:
		return typed, true
	case *artifactssdk.ContainerImageSignature:
		if typed == nil {
			return artifactssdk.ContainerImageSignature{}, false
		}
		return *typed, true
	case artifactssdk.ContainerImageSignatureSummary:
		return containerImageSignatureFromSummary(typed), true
	case *artifactssdk.ContainerImageSignatureSummary:
		if typed == nil {
			return artifactssdk.ContainerImageSignature{}, false
		}
		return containerImageSignatureFromSummary(*typed), true
	case artifactssdk.CreateContainerImageSignatureResponse:
		return typed.ContainerImageSignature, true
	case *artifactssdk.CreateContainerImageSignatureResponse:
		if typed == nil {
			return artifactssdk.ContainerImageSignature{}, false
		}
		return typed.ContainerImageSignature, true
	case artifactssdk.GetContainerImageSignatureResponse:
		return typed.ContainerImageSignature, true
	case *artifactssdk.GetContainerImageSignatureResponse:
		if typed == nil {
			return artifactssdk.ContainerImageSignature{}, false
		}
		return typed.ContainerImageSignature, true
	case artifactssdk.UpdateContainerImageSignatureResponse:
		return typed.ContainerImageSignature, true
	case *artifactssdk.UpdateContainerImageSignatureResponse:
		if typed == nil {
			return artifactssdk.ContainerImageSignature{}, false
		}
		return typed.ContainerImageSignature, true
	default:
		return artifactssdk.ContainerImageSignature{}, false
	}
}

func containerImageSignatureFromSummary(summary artifactssdk.ContainerImageSignatureSummary) artifactssdk.ContainerImageSignature {
	return artifactssdk.ContainerImageSignature{
		CompartmentId:    summary.CompartmentId,
		DisplayName:      summary.DisplayName,
		Id:               summary.Id,
		ImageId:          summary.ImageId,
		KmsKeyId:         summary.KmsKeyId,
		KmsKeyVersionId:  summary.KmsKeyVersionId,
		Message:          summary.Message,
		Signature:        summary.Signature,
		SigningAlgorithm: artifactssdk.ContainerImageSignatureSigningAlgorithmEnum(summary.SigningAlgorithm),
		TimeCreated:      summary.TimeCreated,
		LifecycleState:   summary.LifecycleState,
		FreeformTags:     summary.FreeformTags,
		DefinedTags:      summary.DefinedTags,
		SystemTags:       summary.SystemTags,
	}
}

func handleContainerImageSignatureDeleteError(resource *artifactsv1beta1.ContainerImageSignature, err error) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return fmt.Errorf("ContainerImageSignature delete confirmation returned ambiguous not-found response: %s", err.Error())
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	clone := make(map[string]string, len(values))
	for key, value := range values {
		clone[key] = value
	}
	return clone
}

func definedTagsForOCI(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	return *util.ConvertToOciDefinedTags(&tags)
}

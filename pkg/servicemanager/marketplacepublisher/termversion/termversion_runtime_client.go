/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package termversion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	marketplacepublishersdk "github.com/oracle/oci-go-sdk/v65/marketplacepublisher"
	marketplacepublisherv1beta1 "github.com/oracle/oci-service-operator/api/marketplacepublisher/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	termVersionKind = "TermVersion"

	termVersionTermIDAnnotation            = "marketplacepublisher.oracle.com/term-id"
	termVersionContentSecretAnnotation     = "marketplacepublisher.oracle.com/term-version-content-secret"
	termVersionContentSecretKeyAnnotation  = "marketplacepublisher.oracle.com/term-version-content-secret-key"
	termVersionDefaultContentSecretKey     = "content"
	termVersionContentFingerprintKey       = "osokTermVersionContentSHA256="
	termVersionContentSourceFingerprintKey = "osokTermVersionContentSourceSHA256="

	termVersionDeletePendingMessage = "OCI TermVersion delete is in progress"
)

type termVersionOCIClient interface {
	CreateTermVersion(context.Context, marketplacepublishersdk.CreateTermVersionRequest) (marketplacepublishersdk.CreateTermVersionResponse, error)
	GetTermVersion(context.Context, marketplacepublishersdk.GetTermVersionRequest) (marketplacepublishersdk.GetTermVersionResponse, error)
	ListTermVersions(context.Context, marketplacepublishersdk.ListTermVersionsRequest) (marketplacepublishersdk.ListTermVersionsResponse, error)
	UpdateTermVersion(context.Context, marketplacepublishersdk.UpdateTermVersionRequest) (marketplacepublishersdk.UpdateTermVersionResponse, error)
	UpdateTermVersionContent(context.Context, marketplacepublishersdk.UpdateTermVersionContentRequest) (marketplacepublishersdk.UpdateTermVersionContentResponse, error)
	DeleteTermVersion(context.Context, marketplacepublishersdk.DeleteTermVersionRequest) (marketplacepublishersdk.DeleteTermVersionResponse, error)
}

type termVersionRuntimeClient struct {
	client           termVersionOCIClient
	credentialClient credhelper.CredentialClient
	initErr          error
	log              loggerutil.OSOKLogger
}

type termVersionIdentity struct {
	termID            string
	displayName       string
	contentSecret     string
	contentKey        string
	contentHash       string
	contentSourceHash string
}

func init() {
	registerTermVersionRuntimeHooksMutator(func(manager *TermVersionServiceManager, hooks *TermVersionRuntimeHooks) {
		client, initErr := newTermVersionSDKClient(manager)
		applyTermVersionRuntimeHooks(manager, hooks, client, initErr)
	})
}

func newTermVersionSDKClient(manager *TermVersionServiceManager) (termVersionOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("%s service manager is nil", termVersionKind)
	}
	client, err := marketplacepublishersdk.NewMarketplacePublisherClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyTermVersionRuntimeHooks(
	manager *TermVersionServiceManager,
	hooks *TermVersionRuntimeHooks,
	client termVersionOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newTermVersionRuntimeSemantics()
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(_ TermVersionServiceClient) TermVersionServiceClient {
		return newTermVersionRuntimeClient(manager, client, initErr)
	})
}

func newTermVersionRuntimeClient(
	manager *TermVersionServiceManager,
	client termVersionOCIClient,
	initErr error,
) TermVersionServiceClient {
	runtimeClient := &termVersionRuntimeClient{
		client:  client,
		initErr: initErr,
	}
	if manager != nil {
		runtimeClient.credentialClient = manager.CredentialClient
		runtimeClient.log = manager.Log
	}
	return runtimeClient
}

func newTermVersionServiceClientWithOCIClient(
	client termVersionOCIClient,
	credentialClient credhelper.CredentialClient,
	log loggerutil.OSOKLogger,
) TermVersionServiceClient {
	return &termVersionRuntimeClient{
		client:           client,
		credentialClient: credentialClient,
		log:              log,
	}
}

func newTermVersionRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "marketplacepublisher",
		FormalSlug:    "termversion",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "handwritten",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "read-create-content-secret",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ActiveStates: []string{
				string(marketplacepublishersdk.TermVersionLifecycleStateActive),
				string(marketplacepublishersdk.TermVersionLifecycleStateInactive),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy: "required",
			TerminalStates: []string{
				string(marketplacepublishersdk.TermVersionStatusDeleted),
			},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"termId", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"displayName",
				"definedTags",
				"freeformTags",
				"contentSecret",
				"contentSecretKey",
			},
			ForceNew: []string{
				"termId",
			},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "resource-local TermVersion runtime", EntityType: termVersionKind, Action: "CreateTermVersion"}},
			Update: []generatedruntime.Hook{{Helper: "resource-local TermVersion runtime", EntityType: termVersionKind, Action: "UpdateTermVersion"}},
			Delete: []generatedruntime.Hook{{Helper: "resource-local TermVersion runtime", EntityType: termVersionKind, Action: "DeleteTermVersion"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
		Unsupported: []generatedruntime.UnsupportedSemantic{{
			Category:      "crd-shape",
			StopCondition: fmt.Sprintf("%s and %s annotations are required until termId and upload content are promoted into the CR spec", termVersionTermIDAnnotation, termVersionContentSecretAnnotation),
		}},
	}
}

func (c *termVersionRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.TermVersion,
	request ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if err := c.validateConfigured(); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
	}

	identity, err := resolveTermVersionIdentity(resource)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
	}

	current, err := c.lookupCurrentTermVersion(ctx, resource, identity)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
	}
	if current == nil {
		return c.createTermVersion(ctx, resource, request, identity)
	}

	identity, contentUpdateNeeded, err := c.resolveTermVersionUpdateIdentity(ctx, resource, request, identity, *current)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
	}
	if termVersionDeleted(*current) {
		err := fmt.Errorf("%s %s is already deleted; create a replacement Kubernetes resource", termVersionKind, termVersionString(current.Id))
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
	}
	if contentUpdateNeeded {
		return c.updateTermVersionContent(ctx, resource, request, identity, *current)
	}
	if details, updateNeeded := buildTermVersionUpdateDetails(resource, *current); updateNeeded {
		return c.updateTermVersion(ctx, resource, identity, *current, details)
	}
	return c.markActive(resource, identity, *current), nil
}

func (c *termVersionRuntimeClient) Delete(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.TermVersion,
) (bool, error) {
	if err := c.validateConfigured(); err != nil {
		return false, c.fail(resource, err)
	}
	if resource == nil {
		return false, fmt.Errorf("%s resource is nil", termVersionKind)
	}

	currentID, resolvedDeleted, err := c.resolveTermVersionDeleteID(ctx, resource)
	if err != nil {
		return false, c.fail(resource, err)
	}
	if resolvedDeleted {
		return true, nil
	}

	if termVersionDeleteAlreadyPending(resource) {
		return c.confirmTermVersionDeleted(ctx, resource, currentID)
	}

	response, err := c.client.DeleteTermVersion(ctx, marketplacepublishersdk.DeleteTermVersionRequest{
		TermVersionId: common.String(currentID),
	})
	if err != nil {
		if isTermVersionUnambiguousNotFound(err) {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			c.markDeleted(resource, "OCI resource no longer exists")
			return true, nil
		}
		return false, c.fail(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	return c.confirmTermVersionDeleted(ctx, resource, currentID)
}

func (c *termVersionRuntimeClient) resolveTermVersionDeleteID(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.TermVersion,
) (string, bool, error) {
	currentID := termVersionCurrentID(resource)
	if currentID != "" {
		return currentID, false, nil
	}

	identity, err := resolveTermVersionDeleteIdentity(resource)
	if err != nil {
		c.markDeleted(resource, "OCI resource identifier is not recorded")
		return "", true, nil
	}
	current, err := c.lookupTermVersionByList(ctx, identity, "")
	if err != nil {
		return "", false, err
	}
	if current == nil || termVersionString(current.Id) == "" {
		c.markDeleted(resource, "OCI resource no longer exists")
		return "", true, nil
	}
	if err := projectTermVersionStatus(resource, *current); err != nil {
		return "", false, err
	}
	return termVersionString(current.Id), false, nil
}

func (c *termVersionRuntimeClient) validateConfigured() error {
	if c.initErr != nil {
		return fmt.Errorf("initialize %s OCI client: %w", termVersionKind, c.initErr)
	}
	if c.client == nil {
		return fmt.Errorf("%s OCI client is not configured", termVersionKind)
	}
	return nil
}

func resolveTermVersionIdentity(resource *marketplacepublisherv1beta1.TermVersion) (termVersionIdentity, error) {
	if resource == nil {
		return termVersionIdentity{}, fmt.Errorf("%s resource is nil", termVersionKind)
	}

	annotationTermID := termVersionAnnotation(resource, termVersionTermIDAnnotation)
	trackedTermID := strings.TrimSpace(resource.Status.TermId)
	if trackedTermID != "" && annotationTermID != "" && trackedTermID != annotationTermID {
		return termVersionIdentity{}, fmt.Errorf("%s create-only parent term annotation %q changed; create a replacement resource instead", termVersionKind, termVersionTermIDAnnotation)
	}

	displayName := termVersionDesiredDisplayName(resource)
	if displayName == "" {
		return termVersionIdentity{}, fmt.Errorf("%s spec.displayName or metadata.name is required", termVersionKind)
	}

	return termVersionIdentity{
		termID:        firstNonEmptyTermVersion(trackedTermID, annotationTermID),
		displayName:   displayName,
		contentSecret: termVersionAnnotation(resource, termVersionContentSecretAnnotation),
		contentKey:    firstNonEmptyTermVersion(termVersionAnnotation(resource, termVersionContentSecretKeyAnnotation), termVersionDefaultContentSecretKey),
	}, nil
}

func resolveTermVersionDeleteIdentity(resource *marketplacepublisherv1beta1.TermVersion) (termVersionIdentity, error) {
	if resource == nil {
		return termVersionIdentity{}, fmt.Errorf("%s resource is nil", termVersionKind)
	}

	termID := firstNonEmptyTermVersion(strings.TrimSpace(resource.Status.TermId), termVersionAnnotation(resource, termVersionTermIDAnnotation))
	displayName := firstNonEmptyTermVersion(strings.TrimSpace(resource.Status.DisplayName), strings.TrimSpace(resource.Spec.DisplayName), strings.TrimSpace(resource.Name))
	if termID == "" || displayName == "" {
		return termVersionIdentity{}, fmt.Errorf("%s delete requires recorded status.id or %q plus displayName", termVersionKind, termVersionTermIDAnnotation)
	}
	return termVersionIdentity{termID: termID, displayName: displayName}, nil
}

func (c *termVersionRuntimeClient) lookupCurrentTermVersion(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.TermVersion,
	identity termVersionIdentity,
) (*marketplacepublishersdk.TermVersion, error) {
	currentID := termVersionCurrentID(resource)
	if currentID != "" {
		current, err := c.getTermVersion(ctx, currentID)
		switch {
		case err == nil:
			return current, nil
		case !isTermVersionUnambiguousNotFound(err):
			return nil, err
		case identity.termID == "":
			return nil, nil
		}
	}
	if identity.termID == "" {
		return nil, nil
	}
	return c.lookupTermVersionByList(ctx, identity, currentID)
}

func (c *termVersionRuntimeClient) getTermVersion(ctx context.Context, termVersionID string) (*marketplacepublishersdk.TermVersion, error) {
	response, err := c.client.GetTermVersion(ctx, marketplacepublishersdk.GetTermVersionRequest{
		TermVersionId: common.String(termVersionID),
	})
	if err != nil {
		return nil, err
	}
	current := response.TermVersion
	return &current, nil
}

func (c *termVersionRuntimeClient) lookupTermVersionByList(
	ctx context.Context,
	identity termVersionIdentity,
	preferredID string,
) (*marketplacepublishersdk.TermVersion, error) {
	if strings.TrimSpace(identity.termID) == "" || strings.TrimSpace(identity.displayName) == "" {
		return nil, nil
	}

	matches, err := c.listMatchingTermVersions(ctx, identity, preferredID)
	if err != nil {
		return nil, err
	}
	return c.termVersionFromListMatches(ctx, matches, identity.termID)
}

func (c *termVersionRuntimeClient) listMatchingTermVersions(
	ctx context.Context,
	identity termVersionIdentity,
	preferredID string,
) ([]marketplacepublishersdk.TermVersionSummary, error) {
	request := marketplacepublishersdk.ListTermVersionsRequest{
		TermId:      common.String(identity.termID),
		DisplayName: common.String(identity.displayName),
	}
	var matches []marketplacepublishersdk.TermVersionSummary
	for {
		response, err := c.client.ListTermVersions(ctx, request)
		if err != nil {
			return nil, err
		}
		for _, item := range response.Items {
			if termVersionSummaryMatches(item, identity, preferredID) {
				matches = append(matches, item)
			}
		}
		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			break
		}
		request.Page = response.OpcNextPage
	}
	return matches, nil
}

func (c *termVersionRuntimeClient) termVersionFromListMatches(
	ctx context.Context,
	matches []marketplacepublishersdk.TermVersionSummary,
	termID string,
) (*marketplacepublishersdk.TermVersion, error) {
	switch len(matches) {
	case 0:
		return nil, nil
	case 1:
		id := termVersionString(matches[0].Id)
		if id == "" {
			current := termVersionFromSummary(matches[0], termID)
			return &current, nil
		}
		return c.getTermVersion(ctx, id)
	default:
		return nil, fmt.Errorf("%s list response returned multiple matching resources for termId/displayName", termVersionKind)
	}
}

func termVersionSummaryMatches(summary marketplacepublishersdk.TermVersionSummary, identity termVersionIdentity, preferredID string) bool {
	id := termVersionString(summary.Id)
	if preferredID != "" && id == preferredID {
		return true
	}
	return strings.TrimSpace(identity.displayName) != "" && termVersionString(summary.DisplayName) == identity.displayName
}

func termVersionFromSummary(summary marketplacepublishersdk.TermVersionSummary, termID string) marketplacepublishersdk.TermVersion {
	return marketplacepublishersdk.TermVersion{
		Id:             summary.Id,
		TermId:         common.String(termID),
		CompartmentId:  summary.CompartmentId,
		DisplayName:    summary.DisplayName,
		Status:         summary.Status,
		LifecycleState: summary.LifecycleState,
		TimeCreated:    summary.TimeCreated,
		TimeUpdated:    summary.TimeUpdated,
		FreeformTags:   cloneTermVersionStringMap(summary.FreeformTags),
		DefinedTags:    cloneTermVersionDefinedTags(summary.DefinedTags),
		SystemTags:     cloneTermVersionDefinedTags(summary.SystemTags),
	}
}

func (c *termVersionRuntimeClient) createTermVersion(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.TermVersion,
	request ctrl.Request,
	identity termVersionIdentity,
) (servicemanager.OSOKResponse, error) {
	if strings.TrimSpace(identity.termID) == "" {
		err := fmt.Errorf("%s metadata annotation %q is required because the CRD has no spec termId field", termVersionKind, termVersionTermIDAnnotation)
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
	}

	content, contentHash, contentSourceHash, err := c.loadCreateContent(ctx, resource, request, identity)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
	}
	defer func() {
		_ = content.Close()
	}()
	identity.contentHash = contentHash
	identity.contentSourceHash = contentSourceHash

	createResponse, err := c.client.CreateTermVersion(ctx, marketplacepublishersdk.CreateTermVersionRequest{
		TermId:                   common.String(identity.termID),
		DisplayName:              common.String(identity.displayName),
		CreateTermVersionContent: content,
		OpcRetryToken:            termVersionRetryToken(resource),
	})
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, createResponse)

	current := createResponse.TermVersion
	if id := termVersionString(current.Id); id != "" {
		if refreshed, err := c.getTermVersion(ctx, id); err == nil {
			current = *refreshed
		} else if !isTermVersionUnambiguousNotFound(err) {
			return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
		}
	}
	if termVersionString(current.TermId) == "" {
		current.TermId = common.String(identity.termID)
	}
	if termVersionString(current.DisplayName) == "" {
		current.DisplayName = common.String(identity.displayName)
	}
	return c.finishTermVersionWrite(ctx, resource, identity, current)
}

func (c *termVersionRuntimeClient) loadCreateContent(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.TermVersion,
	request ctrl.Request,
	identity termVersionIdentity,
) (io.ReadCloser, string, string, error) {
	if strings.TrimSpace(identity.contentSecret) == "" {
		return nil, "", "", fmt.Errorf("%s metadata annotation %q is required because CreateTermVersion requires uploaded content", termVersionKind, termVersionContentSecretAnnotation)
	}
	content, contentHash, contentSourceHash, err := c.loadContentBytes(ctx, resource, request, identity)
	if err != nil {
		return nil, "", "", err
	}
	return io.NopCloser(bytes.NewReader(content)), contentHash, contentSourceHash, nil
}

func (c *termVersionRuntimeClient) updateTermVersion(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.TermVersion,
	identity termVersionIdentity,
	current marketplacepublishersdk.TermVersion,
	details marketplacepublishersdk.UpdateTermVersionDetails,
) (servicemanager.OSOKResponse, error) {
	id := termVersionString(current.Id)
	if id == "" {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, fmt.Errorf("%s update requires current OCI id", termVersionKind))
	}

	response, err := c.client.UpdateTermVersion(ctx, marketplacepublishersdk.UpdateTermVersionRequest{
		TermVersionId:            common.String(id),
		UpdateTermVersionDetails: details,
	})
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)

	refreshed := response.TermVersion
	if termVersionString(refreshed.Id) == "" {
		refreshed = current
	}
	if id := termVersionString(refreshed.Id); id != "" {
		if readback, err := c.getTermVersion(ctx, id); err == nil {
			refreshed = *readback
		} else if !isTermVersionUnambiguousNotFound(err) {
			return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
		}
	}
	return c.markActive(resource, identity, refreshed), nil
}

func (c *termVersionRuntimeClient) updateTermVersionContent(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.TermVersion,
	request ctrl.Request,
	identity termVersionIdentity,
	current marketplacepublishersdk.TermVersion,
) (servicemanager.OSOKResponse, error) {
	id := termVersionString(current.Id)
	if id == "" {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, fmt.Errorf("%s content update requires current OCI id", termVersionKind))
	}

	content, contentHash, contentSourceHash, err := c.loadContentBytes(ctx, resource, request, identity)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
	}
	identity.contentHash = contentHash
	identity.contentSourceHash = contentSourceHash

	contentReader := io.NopCloser(bytes.NewReader(content))
	defer func() {
		_ = contentReader.Close()
	}()

	response, err := c.client.UpdateTermVersionContent(ctx, marketplacepublishersdk.UpdateTermVersionContentRequest{
		TermVersionId:            common.String(id),
		UpdateTermVersionContent: contentReader,
		DisplayName:              termVersionContentUpdateDisplayName(resource, current),
	})
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)

	refreshed := response.TermVersion
	if termVersionString(refreshed.Id) == "" {
		refreshed = current
	}
	if id := termVersionString(refreshed.Id); id != "" {
		if readback, err := c.getTermVersion(ctx, id); err == nil {
			refreshed = *readback
		} else if !isTermVersionUnambiguousNotFound(err) {
			return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
		}
	}
	if termVersionString(refreshed.TermId) == "" {
		refreshed.TermId = common.String(identity.termID)
	}
	if termVersionString(refreshed.DisplayName) == "" {
		refreshed.DisplayName = common.String(identity.displayName)
	}
	return c.finishTermVersionWrite(ctx, resource, identity, refreshed)
}

func (c *termVersionRuntimeClient) finishTermVersionWrite(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.TermVersion,
	identity termVersionIdentity,
	current marketplacepublishersdk.TermVersion,
) (servicemanager.OSOKResponse, error) {
	if details, updateNeeded := buildTermVersionUpdateDetails(resource, current); updateNeeded {
		return c.updateTermVersion(ctx, resource, identity, current, details)
	}
	return c.markActive(resource, identity, current), nil
}

func termVersionContentUpdateDisplayName(
	resource *marketplacepublisherv1beta1.TermVersion,
	current marketplacepublishersdk.TermVersion,
) *string {
	if resource == nil {
		return nil
	}
	desired := termVersionDesiredDisplayName(resource)
	if desired == "" || desired == termVersionString(current.DisplayName) {
		return nil
	}
	return common.String(desired)
}

func buildTermVersionUpdateDetails(
	resource *marketplacepublisherv1beta1.TermVersion,
	current marketplacepublishersdk.TermVersion,
) (marketplacepublishersdk.UpdateTermVersionDetails, bool) {
	if resource == nil {
		return marketplacepublishersdk.UpdateTermVersionDetails{}, false
	}

	details := marketplacepublishersdk.UpdateTermVersionDetails{}
	updateNeeded := false
	if desired := termVersionDesiredDisplayName(resource); desired != "" && desired != termVersionString(current.DisplayName) {
		details.DisplayName = common.String(desired)
		updateNeeded = true
	}
	if resource.Spec.FreeformTags != nil && !reflect.DeepEqual(resource.Spec.FreeformTags, current.FreeformTags) {
		details.FreeformTags = cloneTermVersionStringMap(resource.Spec.FreeformTags)
		updateNeeded = true
	}
	if resource.Spec.DefinedTags != nil {
		desired := termVersionDefinedTagsForOCI(resource.Spec.DefinedTags)
		if !reflect.DeepEqual(desired, current.DefinedTags) {
			details.DefinedTags = desired
			updateNeeded = true
		}
	}
	return details, updateNeeded
}

func (c *termVersionRuntimeClient) resolveTermVersionUpdateIdentity(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.TermVersion,
	request ctrl.Request,
	identity termVersionIdentity,
	current marketplacepublishersdk.TermVersion,
) (termVersionIdentity, bool, error) {
	if resource == nil {
		return identity, false, fmt.Errorf("%s resource is nil", termVersionKind)
	}
	currentTermID := termVersionString(current.TermId)
	if identity.termID != "" && currentTermID != "" && currentTermID != identity.termID {
		return identity, false, fmt.Errorf("%s formal semantics require replacement when termId changes", termVersionKind)
	}
	return c.resolveTermVersionContentUpdate(ctx, resource, request, identity)
}

func (c *termVersionRuntimeClient) resolveTermVersionContentUpdate(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.TermVersion,
	request ctrl.Request,
	identity termVersionIdentity,
) (termVersionIdentity, bool, error) {
	recorded, ok := termVersionRecordedContentFingerprint(resource)
	if ok {
		contentHash, contentSourceHash, err := c.currentTermVersionContentHashes(ctx, resource, request, identity)
		if err != nil {
			return identity, false, err
		}
		identity.contentHash = contentHash
		identity.contentSourceHash = contentSourceHash
		return identity, contentHash != recorded, nil
	}
	if strings.TrimSpace(identity.contentSecret) == "" {
		return identity, false, nil
	}
	if termVersionHasTrackedIdentity(resource) && resource.Status.OsokStatus.CreatedAt == nil {
		return identity, false, nil
	}
	contentHash, contentSourceHash, err := c.currentTermVersionContentHashes(ctx, resource, request, identity)
	if err != nil {
		return identity, false, err
	}
	identity.contentHash = contentHash
	identity.contentSourceHash = contentSourceHash
	return identity, termVersionHasTrackedIdentity(resource), nil
}

func (c *termVersionRuntimeClient) currentTermVersionContentHashes(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.TermVersion,
	request ctrl.Request,
	identity termVersionIdentity,
) (string, string, error) {
	_, contentHash, contentSourceHash, err := c.loadContentBytes(ctx, resource, request, identity)
	if err != nil {
		return "", "", err
	}
	return contentHash, contentSourceHash, nil
}

func (c *termVersionRuntimeClient) loadContentBytes(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.TermVersion,
	request ctrl.Request,
	identity termVersionIdentity,
) ([]byte, string, string, error) {
	if strings.TrimSpace(identity.contentSecret) == "" {
		return nil, "", "", fmt.Errorf("%s metadata annotation %q is required to validate original uploaded content", termVersionKind, termVersionContentSecretAnnotation)
	}
	if c.credentialClient == nil {
		return nil, "", "", fmt.Errorf("%s content secret reader is not configured", termVersionKind)
	}

	namespace := termVersionContentNamespace(resource, request)
	if namespace == "" {
		return nil, "", "", fmt.Errorf("%s namespace is required to read content secret %q", termVersionKind, identity.contentSecret)
	}

	secretData, err := c.credentialClient.GetSecret(ctx, identity.contentSecret, namespace)
	if err != nil {
		return nil, "", "", fmt.Errorf("read %s content secret %q: %w", termVersionKind, identity.contentSecret, err)
	}
	content, ok := secretData[identity.contentKey]
	if !ok {
		return nil, "", "", fmt.Errorf("%s content secret %q does not contain key %q", termVersionKind, identity.contentSecret, identity.contentKey)
	}
	if len(content) == 0 {
		return nil, "", "", fmt.Errorf("%s content secret %q key %q is empty", termVersionKind, identity.contentSecret, identity.contentKey)
	}
	return content, termVersionContentFingerprint(content), termVersionContentSourceFingerprint(identity), nil
}

func (c *termVersionRuntimeClient) confirmTermVersionDeleted(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.TermVersion,
	currentID string,
) (bool, error) {
	current, err := c.getTermVersion(ctx, currentID)
	if err != nil {
		if isTermVersionUnambiguousNotFound(err) {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			c.markDeleted(resource, "OCI resource deleted")
			return true, nil
		}
		return false, c.fail(resource, err)
	}
	if err := projectTermVersionStatus(resource, *current); err != nil {
		return false, c.fail(resource, err)
	}
	if termVersionDeleted(*current) {
		c.markDeleted(resource, "OCI resource deleted")
		return true, nil
	}
	c.markTerminating(resource, *current)
	return false, nil
}

func projectTermVersionStatus(resource *marketplacepublisherv1beta1.TermVersion, current marketplacepublishersdk.TermVersion) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", termVersionKind)
	}

	resource.Status.TermId = termVersionString(current.TermId)
	resource.Status.TermAuthor = string(current.TermAuthor)
	resource.Status.DisplayName = termVersionString(current.DisplayName)
	if current.Attachment != nil {
		resource.Status.Attachment = marketplacepublisherv1beta1.TermVersionAttachment{
			ContentUrl: termVersionString(current.Attachment.ContentUrl),
			MimeType:   termVersionString(current.Attachment.MimeType),
		}
	} else {
		resource.Status.Attachment = marketplacepublisherv1beta1.TermVersionAttachment{}
	}
	resource.Status.Status = string(current.Status)
	resource.Status.LifecycleState = string(current.LifecycleState)
	resource.Status.TimeCreated = termVersionTimeString(current.TimeCreated)
	resource.Status.TimeUpdated = termVersionTimeString(current.TimeUpdated)
	resource.Status.Id = termVersionString(current.Id)
	resource.Status.CompartmentId = termVersionString(current.CompartmentId)
	resource.Status.Author = string(current.Author)
	resource.Status.FreeformTags = cloneTermVersionStringMap(current.FreeformTags)
	resource.Status.DefinedTags = termVersionStatusDefinedTags(current.DefinedTags)
	resource.Status.SystemTags = termVersionStatusDefinedTags(current.SystemTags)
	if resource.Status.Id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)
	}
	return nil
}

func (c *termVersionRuntimeClient) markActive(
	resource *marketplacepublisherv1beta1.TermVersion,
	identity termVersionIdentity,
	current marketplacepublishersdk.TermVersion,
) servicemanager.OSOKResponse {
	_ = projectTermVersionStatus(resource, current)

	status := &resource.Status.OsokStatus
	servicemanager.ClearAsyncOperation(status)
	now := metav1.Now()
	if status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	status.UpdatedAt = &now
	status.Message = termVersionObservedMessage(current)
	status.Reason = string(shared.Active)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Active, v1.ConditionTrue, "", status.Message, c.log)
	recordTermVersionContentFingerprint(resource, identity)
	return servicemanager.OSOKResponse{IsSuccessful: true}
}

func (c *termVersionRuntimeClient) markTerminating(
	resource *marketplacepublisherv1beta1.TermVersion,
	current marketplacepublishersdk.TermVersion,
) {
	if resource == nil {
		return
	}
	status := &resource.Status.OsokStatus
	rawStatus := firstNonEmptyTermVersion(string(current.Status), string(current.LifecycleState))
	_ = servicemanager.ApplyAsyncOperation(status, &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		RawStatus:       rawStatus,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         termVersionDeletePendingMessage,
	}, c.log)
}

func (c *termVersionRuntimeClient) markDeleted(resource *marketplacepublisherv1beta1.TermVersion, message string) {
	if resource == nil {
		return
	}
	status := &resource.Status.OsokStatus
	statusMessage := termVersionMessageWithContentFingerprint(resource, message)
	now := metav1.Now()
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = statusMessage
	status.Reason = string(shared.Terminating)
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", statusMessage, c.log)
}

func (c *termVersionRuntimeClient) fail(resource *marketplacepublisherv1beta1.TermVersion, err error) error {
	if err == nil || resource == nil {
		return err
	}
	status := &resource.Status.OsokStatus
	servicemanager.ClearAsyncOperation(status)
	servicemanager.RecordErrorOpcRequestID(status, err)
	statusMessage := termVersionMessageWithContentFingerprint(resource, err.Error())
	status.Message = statusMessage
	status.Reason = string(shared.Failed)
	now := metav1.Now()
	status.UpdatedAt = &now
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, v1.ConditionFalse, "", statusMessage, c.log)
	return err
}

func termVersionObservedMessage(current marketplacepublishersdk.TermVersion) string {
	state := firstNonEmptyTermVersion(string(current.LifecycleState), string(current.Status))
	if state == "" {
		return "OCI TermVersion is available"
	}
	return fmt.Sprintf("OCI TermVersion is %s", state)
}

func termVersionDeleteAlreadyPending(resource *marketplacepublisherv1beta1.TermVersion) bool {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	return current.Phase == shared.OSOKAsyncPhaseDelete && current.NormalizedClass == shared.OSOKAsyncClassPending
}

func termVersionDeleted(current marketplacepublishersdk.TermVersion) bool {
	return current.Status == marketplacepublishersdk.TermVersionStatusDeleted
}

func isTermVersionUnambiguousNotFound(err error) bool {
	return errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound()
}

func termVersionCurrentID(resource *marketplacepublisherv1beta1.TermVersion) string {
	if resource == nil {
		return ""
	}
	return firstNonEmptyTermVersion(
		strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)),
		strings.TrimSpace(resource.Status.Id),
	)
}

func termVersionDesiredDisplayName(resource *marketplacepublisherv1beta1.TermVersion) string {
	if resource == nil {
		return ""
	}
	return firstNonEmptyTermVersion(resource.Spec.DisplayName, resource.Name)
}

func termVersionAnnotation(resource *marketplacepublisherv1beta1.TermVersion, key string) string {
	if resource == nil || len(resource.Annotations) == 0 {
		return ""
	}
	return strings.TrimSpace(resource.Annotations[key])
}

func termVersionRetryToken(resource *marketplacepublisherv1beta1.TermVersion) *string {
	if resource == nil {
		return nil
	}
	if uid := strings.TrimSpace(string(resource.UID)); uid != "" {
		return common.String(uid)
	}
	namespace := strings.TrimSpace(resource.Namespace)
	name := strings.TrimSpace(resource.Name)
	if namespace == "" && name == "" {
		return nil
	}
	sum := sha256.Sum256([]byte(namespace + "/" + name))
	return common.String(fmt.Sprintf("%x", sum[:16]))
}

func termVersionDefinedTagsForOCI(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	return *util.ConvertToOciDefinedTags(&tags)
}

func termVersionStatusDefinedTags(input map[string]map[string]interface{}) map[string]shared.MapValue {
	if input == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(input))
	for key, values := range input {
		if values == nil {
			converted[key] = nil
			continue
		}
		tagValues := make(shared.MapValue, len(values))
		for innerKey, innerValue := range values {
			tagValues[innerKey] = fmt.Sprint(innerValue)
		}
		converted[key] = tagValues
	}
	return converted
}

func cloneTermVersionStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func cloneTermVersionDefinedTags(input map[string]map[string]interface{}) map[string]map[string]interface{} {
	if input == nil {
		return nil
	}
	cloned := make(map[string]map[string]interface{}, len(input))
	for key, values := range input {
		if values == nil {
			cloned[key] = nil
			continue
		}
		child := make(map[string]interface{}, len(values))
		for childKey, childValue := range values {
			child[childKey] = childValue
		}
		cloned[key] = child
	}
	return cloned
}

func termVersionString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func termVersionTimeString(value *common.SDKTime) string {
	if value == nil || value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}

func firstNonEmptyTermVersion(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func recordTermVersionContentFingerprint(resource *marketplacepublisherv1beta1.TermVersion, identity termVersionIdentity) {
	if resource == nil || strings.TrimSpace(identity.contentHash) == "" {
		return
	}
	setTermVersionContentFingerprints(resource, identity.contentHash, identity.contentSourceHash)
}

func setTermVersionContentFingerprints(resource *marketplacepublisherv1beta1.TermVersion, contentHash string, contentSourceHash string) {
	if resource == nil {
		return
	}
	base := stripTermVersionContentFingerprints(resource.Status.OsokStatus.Message)
	markers := termVersionContentFingerprintMarkers(contentHash, contentSourceHash)
	parts := make([]string, 0, len(markers)+1)
	if base != "" {
		parts = append(parts, base)
	}
	parts = append(parts, markers...)
	resource.Status.OsokStatus.Message = strings.Join(parts, "; ")
}

func termVersionMessageWithContentFingerprint(resource *marketplacepublisherv1beta1.TermVersion, message string) string {
	contentHash, contentOK := termVersionRecordedContentFingerprint(resource)
	contentSourceHash, sourceOK := termVersionRecordedContentSourceFingerprint(resource)
	if !contentOK && !sourceOK {
		return message
	}
	base := stripTermVersionContentFingerprints(message)
	markers := termVersionContentFingerprintMarkers(contentHash, contentSourceHash)
	parts := make([]string, 0, len(markers)+1)
	if base != "" {
		parts = append(parts, base)
	}
	parts = append(parts, markers...)
	return strings.Join(parts, "; ")
}

func termVersionContentFingerprintMarkers(contentHash string, contentSourceHash string) []string {
	var markers []string
	if strings.TrimSpace(contentHash) != "" {
		markers = append(markers, termVersionContentFingerprintKey+contentHash)
	}
	if strings.TrimSpace(contentSourceHash) != "" {
		markers = append(markers, termVersionContentSourceFingerprintKey+contentSourceHash)
	}
	return markers
}

func termVersionContentFingerprint(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func termVersionContentSourceFingerprint(identity termVersionIdentity) string {
	hasher := sha256.New()
	_, _ = io.WriteString(hasher, strings.TrimSpace(identity.contentSecret))
	_, _ = hasher.Write([]byte{0})
	_, _ = io.WriteString(hasher, strings.TrimSpace(identity.contentKey))
	return hex.EncodeToString(hasher.Sum(nil))
}

func termVersionContentNamespace(resource *marketplacepublisherv1beta1.TermVersion, request ctrl.Request) string {
	if resource != nil {
		if namespace := strings.TrimSpace(resource.Namespace); namespace != "" {
			return namespace
		}
	}
	return strings.TrimSpace(request.Namespace)
}

func termVersionRecordedContentFingerprint(resource *marketplacepublisherv1beta1.TermVersion) (string, bool) {
	return termVersionRecordedFingerprint(resource, termVersionContentFingerprintKey)
}

func termVersionRecordedContentSourceFingerprint(resource *marketplacepublisherv1beta1.TermVersion) (string, bool) {
	return termVersionRecordedFingerprint(resource, termVersionContentSourceFingerprintKey)
}

func termVersionRecordedFingerprint(resource *marketplacepublisherv1beta1.TermVersion, markerKey string) (string, bool) {
	if resource == nil {
		return "", false
	}
	raw := resource.Status.OsokStatus.Message
	index := strings.LastIndex(raw, markerKey)
	if index < 0 {
		return "", false
	}
	start := index + len(markerKey)
	end := start
	for end < len(raw) && isTermVersionHexDigit(raw[end]) {
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

func stripTermVersionContentFingerprints(raw string) string {
	raw = stripTermVersionContentFingerprint(raw, termVersionContentFingerprintKey)
	raw = stripTermVersionContentFingerprint(raw, termVersionContentSourceFingerprintKey)
	return raw
}

func stripTermVersionContentFingerprint(raw string, markerKey string) string {
	raw = strings.TrimSpace(raw)
	index := strings.LastIndex(raw, markerKey)
	if index < 0 {
		return raw
	}
	prefix := strings.TrimSpace(strings.TrimRight(raw[:index], "; "))
	start := index + len(markerKey)
	end := start
	for end < len(raw) && isTermVersionHexDigit(raw[end]) {
		end++
	}
	suffix := strings.TrimSpace(strings.TrimLeft(raw[end:], "; "))
	return strings.TrimSpace(strings.Trim(prefix+"; "+suffix, "; "))
}

func termVersionHasTrackedIdentity(resource *marketplacepublisherv1beta1.TermVersion) bool {
	if resource == nil {
		return false
	}
	return termVersionCurrentID(resource) != ""
}

func isTermVersionHexDigit(value byte) bool {
	return (value >= '0' && value <= '9') ||
		(value >= 'a' && value <= 'f') ||
		(value >= 'A' && value <= 'F')
}

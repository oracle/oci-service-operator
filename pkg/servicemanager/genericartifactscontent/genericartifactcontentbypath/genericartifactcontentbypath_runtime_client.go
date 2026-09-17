/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package genericartifactcontentbypath

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	artifactssdk "github.com/oracle/oci-go-sdk/v65/artifacts"
	"github.com/oracle/oci-go-sdk/v65/common"
	genericartifactscontentsdk "github.com/oracle/oci-go-sdk/v65/genericartifactscontent"
	genericartifactscontentv1beta1 "github.com/oracle/oci-service-operator/api/genericartifactscontent/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	genericArtifactContentByPathKind              = "GenericArtifactContentByPath"
	genericArtifactContentByPathDefaultContentKey = "content"
	genericArtifactContentByPathDeleteRequeue     = 5 * time.Minute
)

type genericArtifactContentByPathContentClient interface {
	GetGenericArtifactContentByPath(context.Context, genericartifactscontentsdk.GetGenericArtifactContentByPathRequest) (genericartifactscontentsdk.GetGenericArtifactContentByPathResponse, error)
	PutGenericArtifactContentByPath(context.Context, genericartifactscontentsdk.PutGenericArtifactContentByPathRequest) (genericartifactscontentsdk.PutGenericArtifactContentByPathResponse, error)
}

type genericArtifactContentByPathArtifactClient interface {
	DeleteGenericArtifactByPath(context.Context, artifactssdk.DeleteGenericArtifactByPathRequest) (artifactssdk.DeleteGenericArtifactByPathResponse, error)
	ListGenericArtifacts(context.Context, artifactssdk.ListGenericArtifactsRequest) (artifactssdk.ListGenericArtifactsResponse, error)
}

type genericArtifactContentByPathRuntimeClient struct {
	contentClient    genericArtifactContentByPathContentClient
	artifactClient   genericArtifactContentByPathArtifactClient
	credentialClient credhelper.CredentialClient
	log              loggerutil.OSOKLogger
	initErr          error
}

type genericArtifactContentByPathIdentity struct {
	compartmentID string
	repositoryID  string
	artifactPath  string
	version       string
}

type genericArtifactContentByPathReadback struct {
	content []byte
	etag    string
	found   bool
}

type genericArtifactContentByPathArtifact struct {
	id             string
	displayName    string
	compartmentID  string
	repositoryID   string
	artifactPath   string
	version        string
	sha256         string
	sizeInBytes    int64
	lifecycleState string
}

func init() {
	newGenericArtifactContentByPathServiceClient = func(manager *GenericArtifactContentByPathServiceManager) GenericArtifactContentByPathServiceClient {
		contentClient, contentErr := genericartifactscontentsdk.NewGenericArtifactsContentClientWithConfigurationProvider(manager.Provider)
		artifactClient, artifactErr := artifactssdk.NewArtifactsClientWithConfigurationProvider(manager.Provider)
		return &genericArtifactContentByPathRuntimeClient{
			contentClient:    contentClient,
			artifactClient:   artifactClient,
			credentialClient: manager.CredentialClient,
			log:              manager.Log,
			initErr:          errors.Join(contentErr, artifactErr),
		}
	}
}

func (c *genericArtifactContentByPathRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *genericartifactscontentv1beta1.GenericArtifactContentByPath,
	request ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if err := c.validateClients(); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
	}

	identity, err := desiredGenericArtifactContentByPathIdentity(resource)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
	}
	if err := validateGenericArtifactContentByPathIdentityDrift(resource, identity); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
	}

	desiredContent, hasDesiredContent, err := c.loadDesiredContent(ctx, resource, request)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
	}

	current, err := c.readContent(ctx, identity)
	if err != nil {
		if !isGenericArtifactContentByPathUnambiguousNotFound(err) {
			return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
		}
		current = genericArtifactContentByPathReadback{found: false}
	}

	if current.found {
		return c.reconcileExistingContent(ctx, resource, identity, current, desiredContent, hasDesiredContent)
	}

	if !hasDesiredContent {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, fmt.Errorf("spec.content.secretName is required to upload %s content", genericArtifactContentByPathKind))
	}
	return c.putContent(ctx, resource, identity, desiredContent, "")
}

func (c *genericArtifactContentByPathRuntimeClient) Delete(
	ctx context.Context,
	resource *genericartifactscontentv1beta1.GenericArtifactContentByPath,
) (bool, error) {
	if err := c.validateClients(); err != nil {
		return false, c.fail(resource, err)
	}

	identity, ok := genericArtifactContentByPathIdentityForDelete(resource)
	if !ok {
		c.markDeleted(resource, "GenericArtifactContentByPath has no recorded OCI path identity")
		return true, nil
	}

	if isGenericArtifactContentByPathDeletingLifecycle(resource.Status.LifecycleState) {
		return c.confirmDeletedWithMessage(ctx, resource, identity, fmt.Sprintf("%s delete is already in progress; retaining finalizer until deletion is confirmed", genericArtifactContentByPathKind))
	}

	deleteRequest := artifactssdk.DeleteGenericArtifactByPathRequest{
		RepositoryId: common.String(identity.repositoryID),
		ArtifactPath: common.String(identity.artifactPath),
		Version:      common.String(identity.version),
	}
	if strings.TrimSpace(resource.Status.Etag) != "" {
		deleteRequest.IfMatch = common.String(strings.TrimSpace(resource.Status.Etag))
	}
	deleteResponse, err := c.artifactClient.DeleteGenericArtifactByPath(ctx, deleteRequest)
	if err != nil {
		return c.handleDeleteCallError(ctx, resource, identity, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, deleteResponse)

	return c.confirmDeleted(ctx, resource, identity)
}

func (c *genericArtifactContentByPathRuntimeClient) DeleteWithResult(
	ctx context.Context,
	resource *genericartifactscontentv1beta1.GenericArtifactContentByPath,
) (servicemanager.OSOKDeleteResult, error) {
	deleted, err := c.Delete(ctx, resource)
	if deleted || err != nil {
		return servicemanager.OSOKDeleteResult{Deleted: deleted}, err
	}
	return servicemanager.OSOKDeleteResult{Deleted: false, RequeueDuration: genericArtifactContentByPathDeleteRequeue}, nil
}

func (c *genericArtifactContentByPathRuntimeClient) validateClients() error {
	if c.initErr != nil {
		return c.initErr
	}
	if c.contentClient == nil {
		return fmt.Errorf("%s content OCI client is nil", genericArtifactContentByPathKind)
	}
	if c.artifactClient == nil {
		return fmt.Errorf("%s artifact OCI client is nil", genericArtifactContentByPathKind)
	}
	return nil
}

func (c *genericArtifactContentByPathRuntimeClient) reconcileExistingContent(
	ctx context.Context,
	resource *genericartifactscontentv1beta1.GenericArtifactContentByPath,
	identity genericArtifactContentByPathIdentity,
	current genericArtifactContentByPathReadback,
	desiredContent []byte,
	hasDesiredContent bool,
) (servicemanager.OSOKResponse, error) {
	artifact, artifactFound, err := c.findArtifact(ctx, identity)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
	}
	if !artifactFound {
		projectGenericArtifactContentByPathReadback(resource, identity, current.etag)
		return c.markCondition(resource, shared.Updating, true, "GenericArtifactContentByPath content is readable but artifact metadata lookup has not found the exact artifact yet"), nil
	}
	projectGenericArtifactContentByPathArtifact(resource, identity, artifact, current.etag)
	lifecycle := c.markLifecycle(resource, artifact.lifecycleState)
	if lifecycle.ShouldRequeue || !lifecycle.IsSuccessful {
		return lifecycle, nil
	}
	if !hasDesiredContent || bytes.Equal(current.content, desiredContent) {
		return lifecycle, nil
	}
	return c.putContent(ctx, resource, identity, desiredContent, current.etag)
}

func (c *genericArtifactContentByPathRuntimeClient) handleDeleteCallError(
	ctx context.Context,
	resource *genericartifactscontentv1beta1.GenericArtifactContentByPath,
	identity genericArtifactContentByPathIdentity,
	err error,
) (bool, error) {
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	switch {
	case isGenericArtifactContentByPathUnambiguousNotFound(err):
		c.markDeleted(resource, "GenericArtifactContentByPath artifact is no longer present")
		return true, nil
	case isGenericArtifactContentByPathRetryableDeleteConflict(err):
		return c.confirmDeletedWithMessage(ctx, resource, identity, fmt.Sprintf("%s delete returned retryable conflict; retaining finalizer until deletion is confirmed", genericArtifactContentByPathKind))
	case isGenericArtifactContentByPathAuthNotFound(err):
		c.markTerminating(resource, fmt.Sprintf("%s delete returned NotAuthorizedOrNotFound; retaining finalizer until deletion is confirmed", genericArtifactContentByPathKind))
		return false, nil
	default:
		return false, c.fail(resource, err)
	}
}

func (c *genericArtifactContentByPathRuntimeClient) confirmDeleted(
	ctx context.Context,
	resource *genericartifactscontentv1beta1.GenericArtifactContentByPath,
	identity genericArtifactContentByPathIdentity,
) (bool, error) {
	return c.confirmDeletedWithMessage(ctx, resource, identity, fmt.Sprintf("%s delete accepted; retaining finalizer while content remains readable", genericArtifactContentByPathKind))
}

func (c *genericArtifactContentByPathRuntimeClient) confirmDeletedWithMessage(
	ctx context.Context,
	resource *genericartifactscontentv1beta1.GenericArtifactContentByPath,
	identity genericArtifactContentByPathIdentity,
	pendingMessage string,
) (bool, error) {
	current, err := c.readContent(ctx, identity)
	if err != nil {
		return c.handleDeleteReadError(resource, err)
	}
	if !current.found {
		c.markDeleted(resource, "GenericArtifactContentByPath content is no longer readable")
		return true, nil
	}

	artifact, artifactFound, err := c.findArtifact(ctx, identity)
	if err != nil {
		return false, c.fail(resource, err)
	}
	if artifactFound {
		projectGenericArtifactContentByPathArtifact(resource, identity, artifact, current.etag)
		if strings.EqualFold(artifact.lifecycleState, string(artifactssdk.GenericArtifactLifecycleStateDeleted)) {
			c.markDeleted(resource, "GenericArtifactContentByPath artifact lifecycle is DELETED")
			return true, nil
		}
	} else {
		projectGenericArtifactContentByPathReadback(resource, identity, current.etag)
	}
	c.markTerminating(resource, pendingMessage)
	return false, nil
}

func (c *genericArtifactContentByPathRuntimeClient) handleDeleteReadError(
	resource *genericartifactscontentv1beta1.GenericArtifactContentByPath,
	err error,
) (bool, error) {
	switch {
	case isGenericArtifactContentByPathUnambiguousNotFound(err):
		c.markDeleted(resource, "GenericArtifactContentByPath content is no longer readable")
		return true, nil
	case isGenericArtifactContentByPathAuthNotFound(err):
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		c.markTerminating(resource, fmt.Sprintf("%s delete readback returned NotAuthorizedOrNotFound; retaining finalizer until deletion is confirmed", genericArtifactContentByPathKind))
		return false, nil
	default:
		return false, c.fail(resource, err)
	}
}

func (c *genericArtifactContentByPathRuntimeClient) putContent(
	ctx context.Context,
	resource *genericartifactscontentv1beta1.GenericArtifactContentByPath,
	identity genericArtifactContentByPathIdentity,
	content []byte,
	ifMatch string,
) (servicemanager.OSOKResponse, error) {
	body := io.NopCloser(bytes.NewReader(content))
	defer func() {
		_ = body.Close()
	}()

	request := genericartifactscontentsdk.PutGenericArtifactContentByPathRequest{
		RepositoryId:               common.String(identity.repositoryID),
		ArtifactPath:               common.String(identity.artifactPath),
		Version:                    common.String(identity.version),
		GenericArtifactContentBody: body,
	}
	if strings.TrimSpace(ifMatch) != "" {
		request.IfMatch = common.String(ifMatch)
	}

	response, err := c.contentClient.PutGenericArtifactContentByPath(ctx, request)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)

	readback, err := c.readContent(ctx, identity)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, fmt.Errorf("read back %s content after upload: %w", genericArtifactContentByPathKind, err))
	}
	if !readback.found {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, fmt.Errorf("%s content disappeared immediately after upload", genericArtifactContentByPathKind))
	}
	if !bytes.Equal(readback.content, content) {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, fmt.Errorf("%s readback content differs from uploaded content", genericArtifactContentByPathKind))
	}

	etag := stringValue(response.Etag)
	if readback.etag != "" {
		etag = readback.etag
	}
	artifact := genericArtifactContentByPathArtifactFromContent(response.GenericArtifact)
	if artifact.id == "" {
		var found bool
		artifact, found, err = c.findArtifact(ctx, identity)
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
		}
		if !found {
			return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, fmt.Errorf("%s artifact metadata was not found after upload", genericArtifactContentByPathKind))
		}
	}
	projectGenericArtifactContentByPathArtifact(resource, identity, artifact, etag)
	return c.markLifecycle(resource, artifact.lifecycleState), nil
}

func (c *genericArtifactContentByPathRuntimeClient) readContent(
	ctx context.Context,
	identity genericArtifactContentByPathIdentity,
) (genericArtifactContentByPathReadback, error) {
	response, err := c.contentClient.GetGenericArtifactContentByPath(ctx, genericartifactscontentsdk.GetGenericArtifactContentByPathRequest{
		RepositoryId: common.String(identity.repositoryID),
		ArtifactPath: common.String(identity.artifactPath),
		Version:      common.String(identity.version),
	})
	if err != nil {
		return genericArtifactContentByPathReadback{}, err
	}

	var content []byte
	if response.Content != nil {
		defer func() {
			_ = response.Content.Close()
		}()
		var readErr error
		content, readErr = io.ReadAll(response.Content)
		if readErr != nil {
			return genericArtifactContentByPathReadback{}, fmt.Errorf("read %s content body: %w", genericArtifactContentByPathKind, readErr)
		}
	}
	return genericArtifactContentByPathReadback{
		content: content,
		etag:    stringValue(response.Etag),
		found:   true,
	}, nil
}

func (c *genericArtifactContentByPathRuntimeClient) findArtifact(
	ctx context.Context,
	identity genericArtifactContentByPathIdentity,
) (genericArtifactContentByPathArtifact, bool, error) {
	request := artifactssdk.ListGenericArtifactsRequest{
		CompartmentId: common.String(identity.compartmentID),
		RepositoryId:  common.String(identity.repositoryID),
		ArtifactPath:  common.String(identity.artifactPath),
		Version:       common.String(identity.version),
	}
	seenPages := map[string]struct{}{}
	for {
		response, err := c.artifactClient.ListGenericArtifacts(ctx, request)
		if err != nil {
			return genericArtifactContentByPathArtifact{}, false, err
		}
		for _, item := range response.Items {
			artifact := genericArtifactContentByPathArtifactFromSummary(item)
			if genericArtifactContentByPathArtifactMatchesIdentity(artifact, identity) {
				return artifact, true, nil
			}
		}
		if stringValue(response.OpcNextPage) == "" {
			return genericArtifactContentByPathArtifact{}, false, nil
		}
		nextPage := stringValue(response.OpcNextPage)
		if _, seen := seenPages[nextPage]; seen {
			return genericArtifactContentByPathArtifact{}, false, fmt.Errorf("%s list pagination repeated page token %q", genericArtifactContentByPathKind, nextPage)
		}
		seenPages[nextPage] = struct{}{}
		request.Page = common.String(nextPage)
	}
}

func (c *genericArtifactContentByPathRuntimeClient) loadDesiredContent(
	ctx context.Context,
	resource *genericartifactscontentv1beta1.GenericArtifactContentByPath,
	request ctrl.Request,
) ([]byte, bool, error) {
	secretName := strings.TrimSpace(resource.Spec.Content.SecretName)
	if secretName == "" {
		return nil, false, nil
	}
	if c.credentialClient == nil {
		return nil, false, fmt.Errorf("resolve GenericArtifactContentByPath content secret %q: credential client is nil", secretName)
	}
	namespace := strings.TrimSpace(resource.Namespace)
	if namespace == "" {
		namespace = strings.TrimSpace(request.Namespace)
	}
	if namespace == "" {
		return nil, false, fmt.Errorf("resolve GenericArtifactContentByPath content secret %q: namespace is empty", secretName)
	}
	key := strings.TrimSpace(resource.Spec.ContentKey)
	if key == "" {
		key = genericArtifactContentByPathDefaultContentKey
	}

	secretData, err := c.credentialClient.GetSecret(ctx, secretName, namespace)
	if err != nil {
		return nil, false, fmt.Errorf("get GenericArtifactContentByPath content secret %q: %w", secretName, err)
	}
	content, ok := secretData[key]
	if !ok {
		return nil, false, fmt.Errorf("content key %q in secret %q is not found", key, secretName)
	}
	return append([]byte(nil), content...), true, nil
}

func (c *genericArtifactContentByPathRuntimeClient) markLifecycle(
	resource *genericartifactscontentv1beta1.GenericArtifactContentByPath,
	lifecycle string,
) servicemanager.OSOKResponse {
	switch strings.ToUpper(strings.TrimSpace(lifecycle)) {
	case "", string(artifactssdk.GenericArtifactLifecycleStateAvailable):
		return c.markCondition(resource, shared.Active, false, "GenericArtifactContentByPath content is available")
	case string(artifactssdk.GenericArtifactLifecycleStateDeleting):
		return c.markCondition(resource, shared.Terminating, true, "GenericArtifactContentByPath content is deleting")
	case string(artifactssdk.GenericArtifactLifecycleStateDeleted):
		return c.markCondition(resource, shared.Failed, false, "GenericArtifactContentByPath content is already deleted")
	default:
		return c.markCondition(resource, shared.Updating, true, fmt.Sprintf("GenericArtifactContentByPath content lifecycle state is %s", lifecycle))
	}
}

func (c *genericArtifactContentByPathRuntimeClient) markCondition(
	resource *genericartifactscontentv1beta1.GenericArtifactContentByPath,
	condition shared.OSOKConditionType,
	shouldRequeue bool,
	message string,
) servicemanager.OSOKResponse {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	if status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(condition)
	if condition == shared.Active || condition == shared.Failed {
		status.Async.Current = nil
	}

	conditionStatus := v1.ConditionTrue
	if condition == shared.Failed {
		conditionStatus = v1.ConditionFalse
	}
	*status = util.UpdateOSOKStatusCondition(*status, condition, conditionStatus, "", message, c.log)
	return servicemanager.OSOKResponse{
		IsSuccessful:  condition != shared.Failed,
		ShouldRequeue: shouldRequeue,
	}
}

func (c *genericArtifactContentByPathRuntimeClient) markTerminating(
	resource *genericartifactscontentv1beta1.GenericArtifactContentByPath,
	message string,
) {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, c.log)
}

func (c *genericArtifactContentByPathRuntimeClient) markDeleted(
	resource *genericartifactscontentv1beta1.GenericArtifactContentByPath,
	message string,
) {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	status.Async.Current = nil
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, c.log)
}

func (c *genericArtifactContentByPathRuntimeClient) fail(
	resource *genericartifactscontentv1beta1.GenericArtifactContentByPath,
	err error,
) error {
	if err == nil || resource == nil {
		return err
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	_ = c.markCondition(resource, shared.Failed, false, err.Error())
	return err
}

func desiredGenericArtifactContentByPathIdentity(
	resource *genericartifactscontentv1beta1.GenericArtifactContentByPath,
) (genericArtifactContentByPathIdentity, error) {
	identity := genericArtifactContentByPathIdentity{
		compartmentID: strings.TrimSpace(resource.Spec.CompartmentId),
		repositoryID:  strings.TrimSpace(resource.Spec.RepositoryId),
		artifactPath:  strings.TrimSpace(resource.Spec.ArtifactPath),
		version:       strings.TrimSpace(resource.Spec.Version),
	}
	if identity.compartmentID == "" {
		return identity, fmt.Errorf("spec.compartmentId is required")
	}
	if identity.repositoryID == "" {
		return identity, fmt.Errorf("spec.repositoryId is required")
	}
	if identity.artifactPath == "" {
		return identity, fmt.Errorf("spec.artifactPath is required")
	}
	if identity.version == "" {
		return identity, fmt.Errorf("spec.version is required")
	}
	return identity, nil
}

func genericArtifactContentByPathIdentityForDelete(
	resource *genericartifactscontentv1beta1.GenericArtifactContentByPath,
) (genericArtifactContentByPathIdentity, bool) {
	identity := genericArtifactContentByPathIdentity{
		compartmentID: firstNonEmpty(resource.Status.CompartmentId, resource.Spec.CompartmentId),
		repositoryID:  firstNonEmpty(resource.Status.RepositoryId, resource.Spec.RepositoryId),
		artifactPath:  firstNonEmpty(resource.Status.ArtifactPath, resource.Spec.ArtifactPath),
		version:       firstNonEmpty(resource.Status.Version, resource.Spec.Version),
	}
	return identity, identity.compartmentID != "" && identity.repositoryID != "" && identity.artifactPath != "" && identity.version != ""
}

func validateGenericArtifactContentByPathIdentityDrift(
	resource *genericartifactscontentv1beta1.GenericArtifactContentByPath,
	desired genericArtifactContentByPathIdentity,
) error {
	if err := rejectChangedGenericArtifactContentByPathField("compartmentId", resource.Status.CompartmentId, desired.compartmentID); err != nil {
		return err
	}
	if err := rejectChangedGenericArtifactContentByPathField("repositoryId", resource.Status.RepositoryId, desired.repositoryID); err != nil {
		return err
	}
	if err := rejectChangedGenericArtifactContentByPathField("artifactPath", resource.Status.ArtifactPath, desired.artifactPath); err != nil {
		return err
	}
	return rejectChangedGenericArtifactContentByPathField("version", resource.Status.Version, desired.version)
}

func rejectChangedGenericArtifactContentByPathField(field string, observed string, desired string) error {
	observed = strings.TrimSpace(observed)
	if observed == "" || observed == strings.TrimSpace(desired) {
		return nil
	}
	return fmt.Errorf("spec.%s is immutable for %s after OCI identity is recorded", field, genericArtifactContentByPathKind)
}

func projectGenericArtifactContentByPathArtifact(
	resource *genericartifactscontentv1beta1.GenericArtifactContentByPath,
	identity genericArtifactContentByPathIdentity,
	artifact genericArtifactContentByPathArtifact,
	etag string,
) {
	projectGenericArtifactContentByPathReadback(resource, identity, etag)
	if artifact.id != "" {
		resource.Status.Id = artifact.id
		resource.Status.OsokStatus.Ocid = shared.OCID(artifact.id)
	}
	if artifact.compartmentID != "" {
		resource.Status.CompartmentId = artifact.compartmentID
	}
	if artifact.repositoryID != "" {
		resource.Status.RepositoryId = artifact.repositoryID
	}
	if artifact.artifactPath != "" {
		resource.Status.ArtifactPath = artifact.artifactPath
	}
	if artifact.version != "" {
		resource.Status.Version = artifact.version
	}
	resource.Status.DisplayName = artifact.displayName
	resource.Status.Sha256 = artifact.sha256
	resource.Status.SizeInBytes = artifact.sizeInBytes
	if artifact.lifecycleState != "" {
		resource.Status.LifecycleState = artifact.lifecycleState
	}
}

func projectGenericArtifactContentByPathReadback(
	resource *genericartifactscontentv1beta1.GenericArtifactContentByPath,
	identity genericArtifactContentByPathIdentity,
	etag string,
) {
	resource.Status.CompartmentId = identity.compartmentID
	resource.Status.RepositoryId = identity.repositoryID
	resource.Status.ArtifactPath = identity.artifactPath
	resource.Status.Version = identity.version
	resource.Status.LifecycleState = string(artifactssdk.GenericArtifactLifecycleStateAvailable)
	if strings.TrimSpace(etag) != "" {
		resource.Status.Etag = strings.TrimSpace(etag)
	}
}

func isGenericArtifactContentByPathUnambiguousNotFound(err error) bool {
	var serviceErr common.ServiceError
	if !errors.As(err, &serviceErr) {
		return false
	}
	if serviceErr.GetHTTPStatusCode() != 404 {
		return false
	}
	code := strings.TrimSpace(serviceErr.GetCode())
	return code == "" || code == "NotFound" || code == "GENERIC_ARTIFACT_METADATA_NOT_FOUND" || code == "ARTIFACT_NOT_AVAILABLE"
}

func isGenericArtifactContentByPathAuthNotFound(err error) bool {
	var serviceErr common.ServiceError
	if !errors.As(err, &serviceErr) {
		return false
	}
	return serviceErr.GetHTTPStatusCode() == 404 && strings.TrimSpace(serviceErr.GetCode()) == "NotAuthorizedOrNotFound"
}

func isGenericArtifactContentByPathRetryableDeleteConflict(err error) bool {
	var serviceErr common.ServiceError
	if !errors.As(err, &serviceErr) {
		return false
	}
	if serviceErr.GetHTTPStatusCode() != 409 {
		return false
	}
	switch strings.TrimSpace(serviceErr.GetCode()) {
	case "Conflict", "IncorrectState", "LockConflict":
		return true
	default:
		return false
	}
}

func isGenericArtifactContentByPathDeletingLifecycle(lifecycle string) bool {
	return strings.EqualFold(strings.TrimSpace(lifecycle), string(artifactssdk.GenericArtifactLifecycleStateDeleting))
}

func genericArtifactContentByPathArtifactFromContent(artifact genericartifactscontentsdk.GenericArtifact) genericArtifactContentByPathArtifact {
	result := genericArtifactContentByPathArtifact{
		id:             stringValue(artifact.Id),
		displayName:    stringValue(artifact.DisplayName),
		compartmentID:  stringValue(artifact.CompartmentId),
		repositoryID:   stringValue(artifact.RepositoryId),
		artifactPath:   stringValue(artifact.ArtifactPath),
		version:        stringValue(artifact.Version),
		sha256:         stringValue(artifact.Sha256),
		lifecycleState: strings.TrimSpace(string(artifact.LifecycleState)),
	}
	if artifact.SizeInBytes != nil {
		result.sizeInBytes = *artifact.SizeInBytes
	}
	return result
}

func genericArtifactContentByPathArtifactFromSummary(artifact artifactssdk.GenericArtifactSummary) genericArtifactContentByPathArtifact {
	result := genericArtifactContentByPathArtifact{
		id:             stringValue(artifact.Id),
		displayName:    stringValue(artifact.DisplayName),
		compartmentID:  stringValue(artifact.CompartmentId),
		repositoryID:   stringValue(artifact.RepositoryId),
		artifactPath:   stringValue(artifact.ArtifactPath),
		version:        stringValue(artifact.Version),
		sha256:         stringValue(artifact.Sha256),
		lifecycleState: strings.TrimSpace(string(artifact.LifecycleState)),
	}
	if artifact.SizeInBytes != nil {
		result.sizeInBytes = *artifact.SizeInBytes
	}
	return result
}

func genericArtifactContentByPathArtifactMatchesIdentity(
	artifact genericArtifactContentByPathArtifact,
	identity genericArtifactContentByPathIdentity,
) bool {
	return artifact.compartmentID == identity.compartmentID &&
		artifact.repositoryID == identity.repositoryID &&
		artifact.artifactPath == identity.artifactPath &&
		artifact.version == identity.version
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

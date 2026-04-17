/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package project

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	ailanguagesdk "github.com/oracle/oci-go-sdk/v65/ailanguage"
	"github.com/oracle/oci-go-sdk/v65/common"
	ailanguagev1beta1 "github.com/oracle/oci-service-operator/api/ailanguage/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const projectRequeueDuration = time.Minute

var projectWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(ailanguagesdk.OperationStatusAccepted),
		string(ailanguagesdk.OperationStatusInProgress),
		string(ailanguagesdk.OperationStatusWaiting),
		string(ailanguagesdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(ailanguagesdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(ailanguagesdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(ailanguagesdk.OperationStatusCanceled)},
	AttentionStatusTokens: []string{string(ailanguagesdk.OperationStatusNeedsAttention)},
	CreateActionTokens:    []string{string(ailanguagesdk.ActionTypeCreated)},
	UpdateActionTokens:    []string{string(ailanguagesdk.ActionTypeUpdated)},
	DeleteActionTokens:    []string{string(ailanguagesdk.ActionTypeDeleted)},
}

type projectOCIClient interface {
	CreateProject(context.Context, ailanguagesdk.CreateProjectRequest) (ailanguagesdk.CreateProjectResponse, error)
	GetProject(context.Context, ailanguagesdk.GetProjectRequest) (ailanguagesdk.GetProjectResponse, error)
	ListProjects(context.Context, ailanguagesdk.ListProjectsRequest) (ailanguagesdk.ListProjectsResponse, error)
	UpdateProject(context.Context, ailanguagesdk.UpdateProjectRequest) (ailanguagesdk.UpdateProjectResponse, error)
	DeleteProject(context.Context, ailanguagesdk.DeleteProjectRequest) (ailanguagesdk.DeleteProjectResponse, error)
	GetWorkRequest(context.Context, ailanguagesdk.GetWorkRequestRequest) (ailanguagesdk.GetWorkRequestResponse, error)
}

type projectRuntimeClient struct {
	log     loggerutil.OSOKLogger
	client  projectOCIClient
	initErr error
}

var _ ProjectServiceClient = (*projectRuntimeClient)(nil)

func init() {
	newProjectServiceClient = func(manager *ProjectServiceManager) ProjectServiceClient {
		sdkClient, err := ailanguagesdk.NewAIServiceLanguageClientWithConfigurationProvider(manager.Provider)
		runtimeClient := &projectRuntimeClient{
			log:    manager.Log,
			client: sdkClient,
		}
		if err != nil {
			runtimeClient.initErr = fmt.Errorf("initialize Project OCI client: %w", err)
		}
		return runtimeClient
	}
}

func newProjectServiceClientWithOCIClient(log loggerutil.OSOKLogger, client projectOCIClient) ProjectServiceClient {
	return &projectRuntimeClient{
		log:    log,
		client: client,
	}
}

func (c *projectRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *ailanguagev1beta1.Project,
	_ ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.initErr != nil {
		return c.fail(resource, c.initErr)
	}

	if workRequestID, phase := currentProjectWorkRequest(resource); workRequestID != "" {
		switch phase {
		case shared.OSOKAsyncPhaseCreate:
			return c.resumeCreate(ctx, resource, workRequestID)
		case shared.OSOKAsyncPhaseUpdate:
			return c.resumeUpdate(ctx, resource, workRequestID)
		}
	}

	trackedID := currentProjectID(resource)
	if trackedID == "" {
		return c.resolveOrCreate(ctx, resource)
	}

	current, err := c.getProject(ctx, trackedID)
	if err != nil {
		if isProjectReadNotFoundOCI(err) {
			c.clearTrackedIdentity(resource)
			return c.resolveOrCreate(ctx, resource)
		}
		return c.fail(resource, normalizeProjectOCIError(err))
	}

	c.projectStatus(resource, current)

	switch current.LifecycleState {
	case ailanguagesdk.ProjectLifecycleStateCreating,
		ailanguagesdk.ProjectLifecycleStateUpdating,
		ailanguagesdk.ProjectLifecycleStateDeleting,
		ailanguagesdk.ProjectLifecycleStateFailed:
		return c.finishWithLifecycle(resource, current, ""), nil
	}

	updateRequest, updateNeeded, err := c.buildUpdateRequest(resource, current)
	if err != nil {
		return c.fail(resource, err)
	}
	if !updateNeeded {
		return c.finishWithLifecycle(resource, current, ""), nil
	}

	response, err := c.client.UpdateProject(ctx, updateRequest)
	if err != nil {
		if isRetryableProjectUpdateConflict(err) {
			return c.markRetryableUpdateConflict(resource, err), nil
		}
		return c.fail(resource, normalizeProjectOCIError(err))
	}
	c.recordResponseRequestID(resource, response)

	workRequestID := strings.TrimSpace(stringValue(response.OpcWorkRequestId))
	if workRequestID == "" {
		return c.fail(resource, fmt.Errorf("Project update did not return an opc-work-request-id"))
	}

	c.trackAsyncWorkRequest(
		resource,
		shared.OSOKAsyncPhaseUpdate,
		workRequestID,
		fmt.Sprintf("Project update requested; polling work request %s", workRequestID),
	)
	return c.resumeUpdate(ctx, resource, workRequestID)
}

func (c *projectRuntimeClient) Delete(ctx context.Context, resource *ailanguagev1beta1.Project) (bool, error) {
	if c.initErr != nil {
		return false, c.initErr
	}

	trackedID := currentProjectID(resource)
	if workRequestID, phase := currentProjectWorkRequest(resource); workRequestID != "" && phase == shared.OSOKAsyncPhaseDelete {
		return c.resumeDelete(ctx, resource, trackedID, workRequestID)
	}

	if trackedID == "" {
		c.markDeleted(resource, "OCI Project identifier is not recorded")
		return true, nil
	}

	current, found, err := c.resolveProjectForDelete(ctx, resource, trackedID)
	if err != nil {
		return false, err
	}
	if !found {
		c.markDeleted(resource, "OCI Project no longer exists")
		return true, nil
	}

	c.projectStatus(resource, current)

	switch current.LifecycleState {
	case ailanguagesdk.ProjectLifecycleStateDeleted:
		c.markDeleted(resource, "OCI Project deleted")
		return true, nil
	case ailanguagesdk.ProjectLifecycleStateDeleting:
		c.markDeleteProgress(resource, projectLifecycleMessage(current))
		return false, nil
	}

	response, err := c.client.DeleteProject(ctx, ailanguagesdk.DeleteProjectRequest{
		ProjectId: common.String(trackedID),
	})
	if err != nil {
		if isProjectDeleteNotFoundOCI(err) {
			c.recordErrorRequestID(resource, err)
			c.markDeleted(resource, "OCI Project no longer exists")
			return true, nil
		}
		err = normalizeProjectOCIError(err)
		c.recordErrorRequestID(resource, err)
		return false, err
	}
	c.recordResponseRequestID(resource, response)

	workRequestID := strings.TrimSpace(stringValue(response.OpcWorkRequestId))
	if workRequestID == "" {
		return false, fmt.Errorf("Project delete did not return an opc-work-request-id")
	}

	c.trackAsyncWorkRequest(
		resource,
		shared.OSOKAsyncPhaseDelete,
		workRequestID,
		fmt.Sprintf("Project delete requested; polling work request %s", workRequestID),
	)
	return c.resumeDelete(ctx, resource, trackedID, workRequestID)
}

func (c *projectRuntimeClient) resolveOrCreate(
	ctx context.Context,
	resource *ailanguagev1beta1.Project,
) (servicemanager.OSOKResponse, error) {
	current, err := c.resolveExistingProject(ctx, resource)
	if err != nil {
		return c.fail(resource, err)
	}
	if current != nil {
		c.projectStatus(resource, *current)
		return c.finishWithLifecycle(resource, *current, ""), nil
	}

	response, err := c.client.CreateProject(ctx, ailanguagesdk.CreateProjectRequest{
		CreateProjectDetails: buildCreateProjectDetails(resource.Spec),
	})
	if err != nil {
		return c.fail(resource, normalizeProjectOCIError(err))
	}
	c.recordResponseRequestID(resource, response)

	if response.Project.Id != nil {
		c.projectStatus(resource, response.Project)
	}

	workRequestID := strings.TrimSpace(stringValue(response.OpcWorkRequestId))
	if workRequestID == "" {
		return c.fail(resource, fmt.Errorf("Project create did not return an opc-work-request-id"))
	}

	c.trackAsyncWorkRequest(
		resource,
		shared.OSOKAsyncPhaseCreate,
		workRequestID,
		fmt.Sprintf("Project create requested; polling work request %s", workRequestID),
	)
	return c.resumeCreate(ctx, resource, workRequestID)
}

func (c *projectRuntimeClient) resumeCreate(
	ctx context.Context,
	resource *ailanguagev1beta1.Project,
	workRequestID string,
) (servicemanager.OSOKResponse, error) {
	workRequest, err := c.getWorkRequest(ctx, workRequestID)
	if err != nil {
		return c.fail(resource, normalizeProjectOCIError(err))
	}

	currentAsync, err := projectWorkRequestAsyncOperation(resource, workRequest, shared.OSOKAsyncPhaseCreate)
	if err != nil {
		return c.fail(resource, err)
	}

	switch currentAsync.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		return c.setAsyncOperation(resource, currentAsync, shared.OSOKAsyncClassPending, projectWorkRequestMessage(currentAsync.Phase, workRequest)), nil
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		return c.failAsyncOperation(
			resource,
			currentAsync,
			fmt.Errorf("Project %s work request %s finished with status %s", currentAsync.Phase, workRequestID, workRequest.Status),
		)
	case shared.OSOKAsyncClassSucceeded:
		projectID := currentProjectID(resource)
		if projectID == "" {
			projectID, err = resolveProjectIDFromWorkRequest(workRequest, ailanguagesdk.ActionTypeCreated)
			if err != nil {
				return c.failAsyncOperation(resource, currentAsync, err)
			}
		}

		current, err := c.getProject(ctx, projectID)
		if err != nil {
			if isProjectReadNotFoundOCI(err) {
				return c.setAsyncOperation(
					resource,
					currentAsync,
					shared.OSOKAsyncClassPending,
					fmt.Sprintf("Project create work request %s succeeded; waiting for Project %s to become readable", workRequestID, projectID),
				), nil
			}
			return c.failAsyncOperation(resource, currentAsync, normalizeProjectOCIError(err))
		}

		c.projectStatus(resource, current)
		return c.finishWithLifecycle(resource, current, shared.OSOKAsyncPhaseCreate), nil
	default:
		return c.fail(resource, fmt.Errorf("Project create work request %s projected unsupported async class %s", workRequestID, currentAsync.NormalizedClass))
	}
}

func (c *projectRuntimeClient) resumeUpdate(
	ctx context.Context,
	resource *ailanguagev1beta1.Project,
	workRequestID string,
) (servicemanager.OSOKResponse, error) {
	workRequest, err := c.getWorkRequest(ctx, workRequestID)
	if err != nil {
		return c.fail(resource, normalizeProjectOCIError(err))
	}

	currentAsync, err := projectWorkRequestAsyncOperation(resource, workRequest, shared.OSOKAsyncPhaseUpdate)
	if err != nil {
		return c.fail(resource, err)
	}

	switch currentAsync.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		return c.setAsyncOperation(resource, currentAsync, shared.OSOKAsyncClassPending, projectWorkRequestMessage(currentAsync.Phase, workRequest)), nil
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		return c.failAsyncOperation(
			resource,
			currentAsync,
			fmt.Errorf("Project %s work request %s finished with status %s", currentAsync.Phase, workRequestID, workRequest.Status),
		)
	case shared.OSOKAsyncClassSucceeded:
		projectID := currentProjectID(resource)
		if projectID == "" {
			projectID, err = resolveProjectIDFromWorkRequest(workRequest, ailanguagesdk.ActionTypeUpdated)
			if err != nil {
				return c.failAsyncOperation(resource, currentAsync, err)
			}
		}

		current, err := c.getProject(ctx, projectID)
		if err != nil {
			if isProjectReadNotFoundOCI(err) {
				return c.failAsyncOperation(
					resource,
					currentAsync,
					fmt.Errorf("Project update work request %s succeeded but Project %s is no longer readable", workRequestID, projectID),
				)
			}
			return c.failAsyncOperation(resource, currentAsync, normalizeProjectOCIError(err))
		}

		c.projectStatus(resource, current)
		return c.finishWithLifecycle(resource, current, shared.OSOKAsyncPhaseUpdate), nil
	default:
		return c.fail(resource, fmt.Errorf("Project update work request %s projected unsupported async class %s", workRequestID, currentAsync.NormalizedClass))
	}
}

func (c *projectRuntimeClient) resumeDelete(
	ctx context.Context,
	resource *ailanguagev1beta1.Project,
	trackedID string,
	workRequestID string,
) (bool, error) {
	workRequest, err := c.getWorkRequest(ctx, workRequestID)
	if err != nil {
		if trackedID != "" && isProjectDeleteNotFoundOCI(err) {
			current, found, resolveErr := c.resolveProjectForDelete(ctx, resource, trackedID)
			if resolveErr != nil {
				return false, resolveErr
			}
			if !found || current.LifecycleState == ailanguagesdk.ProjectLifecycleStateDeleted {
				c.markDeleted(resource, "OCI Project deleted")
				return true, nil
			}
			c.markDeleteProgress(resource, fmt.Sprintf("Project delete work request %s is no longer readable; waiting for Project %s to disappear", workRequestID, trackedID))
			return false, nil
		}

		err = normalizeProjectOCIError(err)
		c.recordErrorRequestID(resource, err)
		return false, err
	}

	currentAsync, err := projectWorkRequestAsyncOperation(resource, workRequest, shared.OSOKAsyncPhaseDelete)
	if err != nil {
		return false, err
	}

	switch currentAsync.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		_ = c.setAsyncOperation(resource, currentAsync, shared.OSOKAsyncClassPending, projectWorkRequestMessage(currentAsync.Phase, workRequest))
		return false, nil
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		_, err := c.failAsyncOperation(
			resource,
			currentAsync,
			fmt.Errorf("Project %s work request %s finished with status %s", currentAsync.Phase, workRequestID, workRequest.Status),
		)
		return false, err
	case shared.OSOKAsyncClassSucceeded:
		if trackedID == "" {
			var resolveErr error
			trackedID, resolveErr = resolveProjectIDFromWorkRequest(workRequest, ailanguagesdk.ActionTypeDeleted)
			if resolveErr != nil {
				c.markDeleted(resource, "OCI Project delete work request completed")
				return true, nil
			}
		}

		current, found, err := c.resolveProjectForDelete(ctx, resource, trackedID)
		if err != nil {
			_, deleteErr := c.failAsyncOperation(resource, currentAsync, err)
			return false, deleteErr
		}
		if !found || current.LifecycleState == ailanguagesdk.ProjectLifecycleStateDeleted {
			c.markDeleted(resource, "OCI Project deleted")
			return true, nil
		}

		c.projectStatus(resource, current)
		_ = c.setAsyncOperation(
			resource,
			currentAsync,
			shared.OSOKAsyncClassPending,
			fmt.Sprintf("Project delete work request %s succeeded; waiting for Project %s to disappear", workRequestID, trackedID),
		)
		return false, nil
	default:
		return false, fmt.Errorf("Project delete work request %s projected unsupported async class %s", workRequestID, currentAsync.NormalizedClass)
	}
}

func (c *projectRuntimeClient) resolveExistingProject(
	ctx context.Context,
	resource *ailanguagev1beta1.Project,
) (*ailanguagesdk.Project, error) {
	compartmentID := projectLookupCompartmentID(resource)
	displayName := projectLookupDisplayName(resource)
	trackedID := currentProjectID(resource)

	request := ailanguagesdk.ListProjectsRequest{
		CompartmentId: common.String(compartmentID),
	}
	if displayName != "" {
		request.DisplayName = common.String(displayName)
	}
	if trackedID != "" {
		request.ProjectId = common.String(trackedID)
	}

	matches, err := c.listMatchingProjects(ctx, request, compartmentID, displayName, trackedID)
	if err != nil {
		return nil, err
	}

	switch len(matches) {
	case 0:
		return nil, nil
	case 1:
		projectID := strings.TrimSpace(stringValue(matches[0].Id))
		if projectID == "" {
			return nil, fmt.Errorf("resolve Project by displayName %q: OCI match does not expose an identifier", displayName)
		}

		current, err := c.getProject(ctx, projectID)
		if err != nil {
			if isProjectReadNotFoundOCI(err) {
				return nil, fmt.Errorf("confirm Project list match %s: OCI resource is no longer readable", projectID)
			}
			return nil, normalizeProjectOCIError(err)
		}
		return &current, nil
	default:
		return nil, fmt.Errorf("multiple Projects match compartmentId %q and displayName %q", compartmentID, displayName)
	}
}

func (c *projectRuntimeClient) resolveProjectForDelete(
	ctx context.Context,
	resource *ailanguagev1beta1.Project,
	trackedID string,
) (ailanguagesdk.Project, bool, error) {
	current, err := c.getProject(ctx, trackedID)
	if err == nil {
		return current, true, nil
	}
	if !isProjectDeleteNotFoundOCI(err) {
		return ailanguagesdk.Project{}, false, normalizeProjectOCIError(err)
	}

	summary, found, err := c.lookupProjectSummaryByID(ctx, resource, trackedID)
	if err != nil {
		return ailanguagesdk.Project{}, false, err
	}
	if !found {
		return ailanguagesdk.Project{}, false, nil
	}
	return projectFromSummary(summary), true, nil
}

func (c *projectRuntimeClient) lookupProjectSummaryByID(
	ctx context.Context,
	resource *ailanguagev1beta1.Project,
	trackedID string,
) (ailanguagesdk.ProjectSummary, bool, error) {
	compartmentID := projectLookupCompartmentID(resource)
	displayName := projectLookupDisplayName(resource)

	request := ailanguagesdk.ListProjectsRequest{
		CompartmentId: common.String(compartmentID),
		ProjectId:     common.String(trackedID),
	}
	if displayName != "" {
		request.DisplayName = common.String(displayName)
	}

	matches, err := c.listMatchingProjects(ctx, request, compartmentID, displayName, trackedID)
	if err != nil {
		return ailanguagesdk.ProjectSummary{}, false, err
	}

	switch len(matches) {
	case 0:
		return ailanguagesdk.ProjectSummary{}, false, nil
	case 1:
		return matches[0], true, nil
	default:
		return ailanguagesdk.ProjectSummary{}, false, fmt.Errorf("multiple Projects match compartmentId %q, displayName %q, and projectId %q", compartmentID, displayName, trackedID)
	}
}

func (c *projectRuntimeClient) listMatchingProjects(
	ctx context.Context,
	request ailanguagesdk.ListProjectsRequest,
	compartmentID string,
	displayName string,
	projectID string,
) ([]ailanguagesdk.ProjectSummary, error) {
	matches := make([]ailanguagesdk.ProjectSummary, 0, 1)

	for {
		response, err := c.client.ListProjects(ctx, request)
		if err != nil {
			return nil, normalizeProjectOCIError(err)
		}

		for _, item := range response.Items {
			if !projectSummaryMatches(item, compartmentID, displayName, projectID) {
				continue
			}
			if item.LifecycleState == ailanguagesdk.ProjectLifecycleStateDeleted {
				continue
			}
			matches = append(matches, item)
		}

		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			break
		}
		request.Page = response.OpcNextPage
	}

	return matches, nil
}

func (c *projectRuntimeClient) getProject(ctx context.Context, projectID string) (ailanguagesdk.Project, error) {
	response, err := c.client.GetProject(ctx, ailanguagesdk.GetProjectRequest{
		ProjectId: common.String(projectID),
	})
	if err != nil {
		return ailanguagesdk.Project{}, err
	}
	return response.Project, nil
}

func (c *projectRuntimeClient) getWorkRequest(ctx context.Context, workRequestID string) (ailanguagesdk.WorkRequest, error) {
	response, err := c.client.GetWorkRequest(ctx, ailanguagesdk.GetWorkRequestRequest{
		WorkRequestId: common.String(workRequestID),
	})
	if err != nil {
		return ailanguagesdk.WorkRequest{}, err
	}
	return response.WorkRequest, nil
}

func (c *projectRuntimeClient) buildUpdateRequest(
	resource *ailanguagev1beta1.Project,
	current ailanguagesdk.Project,
) (ailanguagesdk.UpdateProjectRequest, bool, error) {
	if current.Id == nil || strings.TrimSpace(*current.Id) == "" {
		return ailanguagesdk.UpdateProjectRequest{}, false, fmt.Errorf("current Project does not expose an OCI identifier")
	}
	if err := validateProjectCreateOnlyDrift(resource.Spec, current); err != nil {
		return ailanguagesdk.UpdateProjectRequest{}, false, err
	}

	updateDetails := ailanguagesdk.UpdateProjectDetails{}
	updateNeeded := false

	if !stringPtrEqual(current.DisplayName, resource.Spec.DisplayName) {
		updateDetails.DisplayName = common.String(resource.Spec.DisplayName)
		updateNeeded = true
	}
	if !stringPtrEqual(current.Description, resource.Spec.Description) {
		updateDetails.Description = common.String(resource.Spec.Description)
		updateNeeded = true
	}

	desiredFreeformTags := desiredProjectFreeformTagsForUpdate(resource.Spec.FreeformTags, current.FreeformTags)
	if !reflect.DeepEqual(current.FreeformTags, desiredFreeformTags) {
		updateDetails.FreeformTags = desiredFreeformTags
		updateNeeded = true
	}

	desiredDefinedTags := desiredProjectDefinedTagsForUpdate(resource.Spec.DefinedTags, current.DefinedTags)
	if !reflect.DeepEqual(current.DefinedTags, desiredDefinedTags) {
		updateDetails.DefinedTags = desiredDefinedTags
		updateNeeded = true
	}

	if !updateNeeded {
		return ailanguagesdk.UpdateProjectRequest{}, false, nil
	}

	return ailanguagesdk.UpdateProjectRequest{
		ProjectId:            current.Id,
		UpdateProjectDetails: updateDetails,
	}, true, nil
}

func buildCreateProjectDetails(spec ailanguagev1beta1.ProjectSpec) ailanguagesdk.CreateProjectDetails {
	createDetails := ailanguagesdk.CreateProjectDetails{
		CompartmentId: common.String(spec.CompartmentId),
	}

	if spec.DisplayName != "" {
		createDetails.DisplayName = common.String(spec.DisplayName)
	}
	if spec.Description != "" {
		createDetails.Description = common.String(spec.Description)
	}
	if spec.FreeformTags != nil {
		createDetails.FreeformTags = cloneStringMap(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		createDetails.DefinedTags = *util.ConvertToOciDefinedTags(&spec.DefinedTags)
	}

	return createDetails
}

func (c *projectRuntimeClient) markRetryableUpdateConflict(
	resource *ailanguagev1beta1.Project,
	err error,
) servicemanager.OSOKResponse {
	normalized := normalizeProjectOCIError(err)
	c.recordErrorRequestID(resource, normalized)

	message := strings.TrimSpace(serviceErrorMessage(err))
	if message == "" {
		message = normalized.Error()
	}

	return c.markAsyncOperation(resource, &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseUpdate,
		RawStatus:       string(ailanguagesdk.ProjectLifecycleStateUpdating),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         message,
	})
}

func (c *projectRuntimeClient) finishWithLifecycle(
	resource *ailanguagev1beta1.Project,
	current ailanguagesdk.Project,
	explicitPhase shared.OSOKAsyncPhase,
) servicemanager.OSOKResponse {
	condition, shouldRequeue := classifyProjectLifecycle(current.LifecycleState)
	message := projectLifecycleMessage(current)
	if asyncCurrent := projectLifecycleAsyncOperation(resource, current, message, explicitPhase); asyncCurrent != nil {
		return c.markAsyncOperation(resource, asyncCurrent)
	}
	return c.markCondition(resource, condition, message, shouldRequeue)
}

func (c *projectRuntimeClient) markCondition(
	resource *ailanguagev1beta1.Project,
	condition shared.OSOKConditionType,
	message string,
	shouldRequeue bool,
) servicemanager.OSOKResponse {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	if resource.Status.Id != "" {
		status.Ocid = shared.OCID(resource.Status.Id)
	}
	if status.Ocid != "" && status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(condition)
	if condition == shared.Active {
		servicemanager.ClearAsyncOperation(status)
	}
	conditionStatus := v1.ConditionTrue
	if condition == shared.Failed {
		conditionStatus = v1.ConditionFalse
	}
	*status = util.UpdateOSOKStatusCondition(*status, condition, conditionStatus, "", message, c.log)

	return servicemanager.OSOKResponse{
		IsSuccessful:    condition != shared.Failed,
		ShouldRequeue:   shouldRequeue,
		RequeueDuration: projectRequeueDuration,
	}
}

func (c *projectRuntimeClient) markAsyncOperation(
	resource *ailanguagev1beta1.Project,
	current *shared.OSOKAsyncOperation,
) servicemanager.OSOKResponse {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	if resource.Status.Id != "" {
		status.Ocid = shared.OCID(resource.Status.Id)
	}
	if status.Ocid != "" && status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	if current.UpdatedAt == nil {
		current.UpdatedAt = &now
	}
	projection := servicemanager.ApplyAsyncOperation(status, current, c.log)
	return servicemanager.OSOKResponse{
		IsSuccessful:    projection.Condition != shared.Failed,
		ShouldRequeue:   projection.ShouldRequeue,
		RequeueDuration: projectRequeueDuration,
	}
}

func (c *projectRuntimeClient) setAsyncOperation(
	resource *ailanguagev1beta1.Project,
	current *shared.OSOKAsyncOperation,
	class shared.OSOKAsyncNormalizedClass,
	message string,
) servicemanager.OSOKResponse {
	next := *current
	next.NormalizedClass = class
	next.Message = message
	next.UpdatedAt = nil
	return c.markAsyncOperation(resource, &next)
}

func (c *projectRuntimeClient) failAsyncOperation(
	resource *ailanguagev1beta1.Project,
	current *shared.OSOKAsyncOperation,
	err error,
) (servicemanager.OSOKResponse, error) {
	if current == nil {
		return c.fail(resource, err)
	}
	c.recordErrorRequestID(resource, err)

	class := current.NormalizedClass
	switch class {
	case shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
	default:
		class = shared.OSOKAsyncClassFailed
	}

	return c.setAsyncOperation(resource, current, class, err.Error()), err
}

func (c *projectRuntimeClient) trackAsyncWorkRequest(
	resource *ailanguagev1beta1.Project,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	message string,
) {
	_ = c.markAsyncOperation(resource, &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           currentProjectAsyncPhase(resource, phase),
		WorkRequestID:   strings.TrimSpace(workRequestID),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         strings.TrimSpace(message),
	})
}

func (c *projectRuntimeClient) markDeleteProgress(resource *ailanguagev1beta1.Project, message string) {
	workRequestID, _ := currentProjectWorkRequest(resource)
	_ = c.markAsyncOperation(resource, &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           currentProjectAsyncPhase(resource, shared.OSOKAsyncPhaseDelete),
		WorkRequestID:   strings.TrimSpace(workRequestID),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         strings.TrimSpace(message),
	})
}

func (c *projectRuntimeClient) fail(
	resource *ailanguagev1beta1.Project,
	err error,
) (servicemanager.OSOKResponse, error) {
	status := &resource.Status.OsokStatus
	servicemanager.RecordErrorOpcRequestID(status, err)
	now := metav1.Now()
	status.UpdatedAt = &now
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	if status.Async.Current != nil {
		current := *status.Async.Current
		current.NormalizedClass = shared.OSOKAsyncClassFailed
		current.Message = err.Error()
		current.UpdatedAt = &now
		_ = servicemanager.ApplyAsyncOperation(status, &current, c.log)
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, v1.ConditionFalse, "", err.Error(), c.log)
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func (c *projectRuntimeClient) recordResponseRequestID(resource *ailanguagev1beta1.Project, response any) {
	if resource == nil {
		return
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
}

func (c *projectRuntimeClient) recordErrorRequestID(resource *ailanguagev1beta1.Project, err error) {
	if resource == nil {
		return
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
}

func (c *projectRuntimeClient) markDeleted(resource *ailanguagev1beta1.Project, message string) {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, c.log)
}

func (c *projectRuntimeClient) clearTrackedIdentity(resource *ailanguagev1beta1.Project) {
	resource.Status = ailanguagev1beta1.ProjectStatus{}
}

func (c *projectRuntimeClient) projectStatus(resource *ailanguagev1beta1.Project, current ailanguagesdk.Project) {
	resource.Status = ailanguagev1beta1.ProjectStatus{
		OsokStatus:       resource.Status.OsokStatus,
		Id:               stringValue(current.Id),
		DisplayName:      stringValue(current.DisplayName),
		CompartmentId:    stringValue(current.CompartmentId),
		TimeCreated:      sdkTimeString(current.TimeCreated),
		LifecycleState:   string(current.LifecycleState),
		Description:      stringValue(current.Description),
		TimeUpdated:      sdkTimeString(current.TimeUpdated),
		LifecycleDetails: stringValue(current.LifecycleDetails),
		FreeformTags:     cloneStringMap(current.FreeformTags),
		DefinedTags:      convertOCIToStatusDefinedTags(current.DefinedTags),
		SystemTags:       convertOCIToStatusDefinedTags(current.SystemTags),
	}
}

func validateProjectCreateOnlyDrift(spec ailanguagev1beta1.ProjectSpec, current ailanguagesdk.Project) error {
	if stringPtrEqual(current.CompartmentId, spec.CompartmentId) {
		return nil
	}
	return fmt.Errorf("Project create-only field drift is not supported: compartmentId")
}

func desiredProjectFreeformTagsForUpdate(spec map[string]string, current map[string]string) map[string]string {
	if spec != nil {
		return cloneStringMap(spec)
	}
	if current != nil {
		return map[string]string{}
	}
	return nil
}

func desiredProjectDefinedTagsForUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) map[string]map[string]interface{} {
	if spec != nil {
		return *util.ConvertToOciDefinedTags(&spec)
	}
	if current != nil {
		return map[string]map[string]interface{}{}
	}
	return nil
}

func resolveProjectIDFromWorkRequest(
	workRequest ailanguagesdk.WorkRequest,
	action ailanguagesdk.ActionTypeEnum,
) (string, error) {
	if id, ok := resolveProjectIDFromResources(workRequest.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resolveProjectIDFromResources(workRequest.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("Project work request %s does not expose a Project identifier", stringValue(workRequest.Id))
}

func resolveProjectIDFromResources(
	resources []ailanguagesdk.WorkRequestResource,
	action ailanguagesdk.ActionTypeEnum,
	preferProjectOnly bool,
) (string, bool) {
	var candidate string

	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferProjectOnly && !isProjectWorkRequestResource(resource) {
			continue
		}

		id := strings.TrimSpace(stringValue(resource.Identifier))
		if id == "" {
			continue
		}
		if candidate == "" {
			candidate = id
			continue
		}
		if candidate != id {
			return "", false
		}
	}

	return candidate, candidate != ""
}

func projectWorkRequestAsyncOperation(
	resource *ailanguagev1beta1.Project,
	workRequest ailanguagesdk.WorkRequest,
	explicitPhase shared.OSOKAsyncPhase,
) (*shared.OSOKAsyncOperation, error) {
	var status *shared.OSOKStatus
	if resource != nil {
		status = &resource.Status.OsokStatus
	}

	fallbackPhase := currentProjectAsyncPhase(resource, explicitPhase)
	if derivedPhase, ok := projectWorkRequestPhaseFromOperationType(workRequest.OperationType); ok {
		if fallbackPhase != "" && fallbackPhase != derivedPhase {
			return nil, fmt.Errorf(
				"Project work request %s exposes operation type %q for phase %q while reconcile expected phase %q",
				stringValue(workRequest.Id),
				workRequest.OperationType,
				derivedPhase,
				fallbackPhase,
			)
		}
		fallbackPhase = derivedPhase
	}

	current, err := servicemanager.BuildWorkRequestAsyncOperation(status, projectWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(workRequest.Status),
		RawOperationType: string(workRequest.OperationType),
		WorkRequestID:    stringValue(workRequest.Id),
		PercentComplete:  workRequest.PercentComplete,
		FallbackPhase:    fallbackPhase,
	})
	if err != nil {
		return nil, err
	}

	current.Message = projectWorkRequestMessage(current.Phase, workRequest)
	return current, nil
}

func projectWorkRequestPhaseFromOperationType(
	operationType ailanguagesdk.OperationTypeEnum,
) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case ailanguagesdk.OperationTypeCreateProject:
		return shared.OSOKAsyncPhaseCreate, true
	case ailanguagesdk.OperationTypeUpdateProject:
		return shared.OSOKAsyncPhaseUpdate, true
	case ailanguagesdk.OperationTypeDeleteProject:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func projectWorkRequestMessage(
	phase shared.OSOKAsyncPhase,
	workRequest ailanguagesdk.WorkRequest,
) string {
	return fmt.Sprintf("Project %s work request %s is %s", phase, stringValue(workRequest.Id), workRequest.Status)
}

func isProjectWorkRequestResource(resource ailanguagesdk.WorkRequestResource) bool {
	entityType := strings.ToLower(strings.TrimSpace(stringValue(resource.EntityType)))
	switch entityType {
	case "project", "projects", "ai_language_project", "ailanguageproject":
		return true
	}
	if strings.Contains(entityType, "project") {
		return true
	}
	entityURI := strings.ToLower(strings.TrimSpace(stringValue(resource.EntityUri)))
	return strings.Contains(entityURI, "/projects/")
}

func projectLifecycleMessage(current ailanguagesdk.Project) string {
	name := strings.TrimSpace(stringValue(current.DisplayName))
	if name == "" {
		name = strings.TrimSpace(stringValue(current.Id))
	}
	if name == "" {
		name = "Project"
	}
	return fmt.Sprintf("Project %s is %s", name, current.LifecycleState)
}

func classifyProjectLifecycle(
	state ailanguagesdk.ProjectLifecycleStateEnum,
) (shared.OSOKConditionType, bool) {
	switch state {
	case ailanguagesdk.ProjectLifecycleStateCreating:
		return shared.Provisioning, true
	case ailanguagesdk.ProjectLifecycleStateUpdating:
		return shared.Updating, true
	case ailanguagesdk.ProjectLifecycleStateDeleting:
		return shared.Terminating, true
	case ailanguagesdk.ProjectLifecycleStateFailed:
		return shared.Failed, false
	default:
		return shared.Active, false
	}
}

func projectLifecycleAsyncOperation(
	resource *ailanguagev1beta1.Project,
	current ailanguagesdk.Project,
	message string,
	explicitPhase shared.OSOKAsyncPhase,
) *shared.OSOKAsyncOperation {
	switch current.LifecycleState {
	case ailanguagesdk.ProjectLifecycleStateCreating:
		return &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           shared.OSOKAsyncPhaseCreate,
			RawStatus:       string(current.LifecycleState),
			NormalizedClass: shared.OSOKAsyncClassPending,
			Message:         message,
		}
	case ailanguagesdk.ProjectLifecycleStateUpdating:
		return &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           shared.OSOKAsyncPhaseUpdate,
			RawStatus:       string(current.LifecycleState),
			NormalizedClass: shared.OSOKAsyncClassPending,
			Message:         message,
		}
	case ailanguagesdk.ProjectLifecycleStateDeleting:
		return &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           shared.OSOKAsyncPhaseDelete,
			RawStatus:       string(current.LifecycleState),
			NormalizedClass: shared.OSOKAsyncClassPending,
			Message:         message,
		}
	case ailanguagesdk.ProjectLifecycleStateFailed:
		phase := currentProjectAsyncPhase(resource, explicitPhase)
		if phase == "" {
			return nil
		}
		return &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           phase,
			RawStatus:       string(current.LifecycleState),
			NormalizedClass: shared.OSOKAsyncClassFailed,
			Message:         message,
		}
	default:
		return nil
	}
}

func currentProjectAsyncPhase(
	resource *ailanguagev1beta1.Project,
	fallback shared.OSOKAsyncPhase,
) shared.OSOKAsyncPhase {
	if fallback != "" {
		return fallback
	}
	if resource == nil {
		return ""
	}
	return servicemanager.ResolveAsyncPhase(&resource.Status.OsokStatus, "")
}

func currentProjectWorkRequest(
	resource *ailanguagev1beta1.Project,
) (string, shared.OSOKAsyncPhase) {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return "", ""
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		return "", ""
	}
	workRequestID := strings.TrimSpace(current.WorkRequestID)
	if workRequestID == "" {
		return "", ""
	}
	return workRequestID, current.Phase
}

func normalizeProjectOCIError(err error) error {
	var serviceErr common.ServiceError
	if !errors.As(err, &serviceErr) {
		return err
	}
	if _, normalized := errorutil.OciErrorTypeResponse(err); normalized != nil {
		return normalized
	}
	return err
}

func isProjectReadNotFoundOCI(err error) bool {
	return errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound()
}

func isProjectDeleteNotFoundOCI(err error) bool {
	classification := errorutil.ClassifyDeleteError(err)
	return classification.IsUnambiguousNotFound() || classification.IsAuthShapedNotFound()
}

func isRetryableProjectUpdateConflict(err error) bool {
	classification := errorutil.ClassifyDeleteError(err)
	if classification.HTTPStatusCode != 409 {
		return false
	}

	message := strings.ToLower(strings.TrimSpace(serviceErrorMessage(err)))
	if classification.IsConflict() {
		return message == "" || strings.Contains(message, "currently being modified")
	}

	return strings.EqualFold(classification.ErrorCode, "Conflict") &&
		strings.Contains(message, "currently being modified")
}

func serviceErrorMessage(err error) string {
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		return serviceErr.GetMessage()
	}
	if err == nil {
		return ""
	}
	return err.Error()
}

func currentProjectID(resource *ailanguagev1beta1.Project) string {
	if resource == nil {
		return ""
	}
	if resource.Status.OsokStatus.Ocid != "" {
		return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	}
	return strings.TrimSpace(resource.Status.Id)
}

func projectLookupCompartmentID(resource *ailanguagev1beta1.Project) string {
	if resource == nil {
		return ""
	}
	if compartmentID := strings.TrimSpace(resource.Status.CompartmentId); compartmentID != "" {
		return compartmentID
	}
	return strings.TrimSpace(resource.Spec.CompartmentId)
}

func projectLookupDisplayName(resource *ailanguagev1beta1.Project) string {
	if resource == nil {
		return ""
	}
	if displayName := strings.TrimSpace(resource.Status.DisplayName); displayName != "" {
		return displayName
	}
	return strings.TrimSpace(resource.Spec.DisplayName)
}

func projectSummaryMatches(
	summary ailanguagesdk.ProjectSummary,
	compartmentID string,
	displayName string,
	projectID string,
) bool {
	if compartmentID != "" && strings.TrimSpace(stringValue(summary.CompartmentId)) != compartmentID {
		return false
	}
	if displayName != "" && strings.TrimSpace(stringValue(summary.DisplayName)) != displayName {
		return false
	}
	if projectID != "" && strings.TrimSpace(stringValue(summary.Id)) != projectID {
		return false
	}
	return true
}

func projectFromSummary(summary ailanguagesdk.ProjectSummary) ailanguagesdk.Project {
	return ailanguagesdk.Project{
		Id:               summary.Id,
		DisplayName:      summary.DisplayName,
		CompartmentId:    summary.CompartmentId,
		TimeCreated:      summary.TimeCreated,
		LifecycleState:   summary.LifecycleState,
		Description:      summary.Description,
		TimeUpdated:      summary.TimeUpdated,
		LifecycleDetails: summary.LifecycleDetails,
		FreeformTags:     cloneStringMap(summary.FreeformTags),
		DefinedTags:      cloneDefinedTags(summary.DefinedTags),
		SystemTags:       cloneDefinedTags(summary.SystemTags),
	}
}

func sdkTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func cloneDefinedTags(input map[string]map[string]interface{}) map[string]map[string]interface{} {
	if input == nil {
		return nil
	}
	cloned := make(map[string]map[string]interface{}, len(input))
	for key, value := range input {
		if value == nil {
			cloned[key] = nil
			continue
		}
		inner := make(map[string]interface{}, len(value))
		for innerKey, innerValue := range value {
			inner[innerKey] = innerValue
		}
		cloned[key] = inner
	}
	return cloned
}

func convertOCIToStatusDefinedTags(input map[string]map[string]interface{}) map[string]shared.MapValue {
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

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func stringPtrEqual(current *string, desired string) bool {
	return strings.TrimSpace(stringValue(current)) == strings.TrimSpace(desired)
}

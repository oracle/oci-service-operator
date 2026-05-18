/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package auditarchiveretrieval

import (
	"context"
	"fmt"
	"strings"

	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func init() {
	registerAuditArchiveRetrievalRuntimeHooksMutator(func(_ *AuditArchiveRetrievalServiceManager, hooks *AuditArchiveRetrievalRuntimeHooks) {
		applyAuditArchiveRetrievalRuntimeHooks(hooks)
	})
}

func applyAuditArchiveRetrievalRuntimeHooks(hooks *AuditArchiveRetrievalRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newAuditArchiveRetrievalRuntimeSemantics()
	wrapAuditArchiveRetrievalListPages(hooks)
	hooks.DeleteHooks.HandleError = handleAuditArchiveRetrievalDeleteError
	wrapAuditArchiveRetrievalDeletePreRead(hooks)
}

func newAuditArchiveRetrievalRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "datasafe",
		FormalSlug:        "auditarchiveretrieval",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(datasafesdk.AuditArchiveRetrievalLifecycleStateCreating)},
			UpdatingStates:     []string{string(datasafesdk.AuditArchiveRetrievalLifecycleStateUpdating)},
			ActiveStates:       []string{string(datasafesdk.AuditArchiveRetrievalLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(datasafesdk.AuditArchiveRetrievalLifecycleStateDeleting)},
			TerminalStates: []string{string(datasafesdk.AuditArchiveRetrievalLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"compartmentId",
				"targetId",
				"startDate",
				"endDate",
				"displayName",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"displayName",
				"description",
				"freeformTags",
				"definedTags",
			},
			ForceNew: []string{
				"compartmentId",
				"targetId",
				"startDate",
				"endDate",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "AuditArchiveRetrieval", Action: "CreateAuditArchiveRetrieval"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "AuditArchiveRetrieval", Action: "UpdateAuditArchiveRetrieval"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "AuditArchiveRetrieval", Action: "DeleteAuditArchiveRetrieval"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "AuditArchiveRetrieval", Action: "GetAuditArchiveRetrieval"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "AuditArchiveRetrieval", Action: "GetAuditArchiveRetrieval"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "AuditArchiveRetrieval", Action: "GetAuditArchiveRetrieval"}},
		},
	}
}

func wrapAuditArchiveRetrievalListPages(hooks *AuditArchiveRetrievalRuntimeHooks) {
	if hooks.List.Call == nil {
		return
	}
	call := hooks.List.Call
	hooks.List.Call = func(ctx context.Context, request datasafesdk.ListAuditArchiveRetrievalsRequest) (datasafesdk.ListAuditArchiveRetrievalsResponse, error) {
		return listAuditArchiveRetrievalPages(ctx, call, request)
	}
}

func listAuditArchiveRetrievalPages(
	ctx context.Context,
	call func(context.Context, datasafesdk.ListAuditArchiveRetrievalsRequest) (datasafesdk.ListAuditArchiveRetrievalsResponse, error),
	request datasafesdk.ListAuditArchiveRetrievalsRequest,
) (datasafesdk.ListAuditArchiveRetrievalsResponse, error) {
	var combined datasafesdk.ListAuditArchiveRetrievalsResponse
	for {
		response, err := call(ctx, request)
		if err != nil {
			return response, err
		}
		if combined.OpcRequestId == nil {
			combined.OpcRequestId = response.OpcRequestId
		}
		combined.RawResponse = response.RawResponse
		combined.Items = append(combined.Items, response.Items...)
		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			combined.OpcNextPage = nil
			return combined, nil
		}
		request.Page = response.OpcNextPage
	}
}

func handleAuditArchiveRetrievalDeleteError(resource *datasafev1beta1.AuditArchiveRetrieval, err error) error {
	if err == nil {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	return fmt.Errorf("AuditArchiveRetrieval delete returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", err)
}

func wrapAuditArchiveRetrievalDeletePreRead(hooks *AuditArchiveRetrievalRuntimeHooks) {
	if hooks.Get.Call == nil {
		return
	}
	getAuditArchiveRetrieval := hooks.Get.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate AuditArchiveRetrievalServiceClient) AuditArchiveRetrievalServiceClient {
		if delegate == nil {
			return nil
		}
		return auditArchiveRetrievalDeletePreReadClient{
			delegate:                 delegate,
			getAuditArchiveRetrieval: getAuditArchiveRetrieval,
		}
	})
}

type auditArchiveRetrievalDeletePreReadClient struct {
	delegate                 AuditArchiveRetrievalServiceClient
	getAuditArchiveRetrieval func(context.Context, datasafesdk.GetAuditArchiveRetrievalRequest) (datasafesdk.GetAuditArchiveRetrievalResponse, error)
}

func (c auditArchiveRetrievalDeletePreReadClient) CreateOrUpdate(
	ctx context.Context,
	resource *datasafev1beta1.AuditArchiveRetrieval,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c auditArchiveRetrievalDeletePreReadClient) Delete(
	ctx context.Context,
	resource *datasafev1beta1.AuditArchiveRetrieval,
) (bool, error) {
	currentID := auditArchiveRetrievalCurrentID(resource)
	if currentID == "" {
		return c.delegate.Delete(ctx, resource)
	}

	_, err := c.getAuditArchiveRetrieval(ctx, datasafesdk.GetAuditArchiveRetrievalRequest{
		AuditArchiveRetrievalId: &currentID,
	})
	switch classification := errorutil.ClassifyDeleteError(err); {
	case err == nil:
		return c.delegate.Delete(ctx, resource)
	case classification.IsUnambiguousNotFound():
		return c.delegate.Delete(ctx, resource)
	case classification.IsAuthShapedNotFound():
		if resource != nil {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		}
		return false, fmt.Errorf("AuditArchiveRetrieval delete pre-read returned ambiguous 404 NotAuthorizedOrNotFound; refusing to call delete: %v", err)
	default:
		if resource != nil {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		}
		return false, fmt.Errorf("AuditArchiveRetrieval delete pre-read failed: %w", err)
	}
}

func auditArchiveRetrievalCurrentID(resource *datasafev1beta1.AuditArchiveRetrieval) string {
	if resource == nil {
		return ""
	}
	if ocid := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); ocid != "" {
		return ocid
	}
	return strings.TrimSpace(resource.Status.Id)
}

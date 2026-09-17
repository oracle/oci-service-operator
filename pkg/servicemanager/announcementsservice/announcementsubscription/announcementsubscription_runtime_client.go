/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package announcementsubscription

import (
	"context"
	"fmt"
	"strings"

	announcementsservicesdk "github.com/oracle/oci-go-sdk/v65/announcementsservice"
	"github.com/oracle/oci-go-sdk/v65/common"
	announcementsservicev1beta1 "github.com/oracle/oci-service-operator/api/announcementsservice/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

func init() {
	registerAnnouncementSubscriptionRuntimeHooksMutator(func(_ *AnnouncementSubscriptionServiceManager, hooks *AnnouncementSubscriptionRuntimeHooks) {
		if hooks == nil {
			return
		}
		hooks.Semantics = announcementSubscriptionRuntimeSemantics()
		hooks.DeleteHooks.ConfirmRead = announcementSubscriptionDeleteConfirmRead(hooks.Get.Call, hooks.List.Call)
	})
}

func announcementSubscriptionRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "announcementsservice",
		FormalSlug:        "announcementsubscription",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy: "lifecycle", Runtime: "generatedruntime", FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ActiveStates: []string{string(announcementsservicesdk.AnnouncementSubscriptionLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			TerminalStates: []string{string(announcementsservicesdk.AnnouncementSubscriptionLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"displayName", "description", "filterGroups", "preferredLanguage", "preferredTimeZone", "freeformTags", "definedTags",
			},
			ForceNew:      []string{"compartmentId", "onsTopicId"},
			ConflictsWith: map[string][]string{},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func announcementSubscriptionDeleteConfirmRead(
	getCall func(context.Context, announcementsservicesdk.GetAnnouncementSubscriptionRequest) (announcementsservicesdk.GetAnnouncementSubscriptionResponse, error),
	listCall func(context.Context, announcementsservicesdk.ListAnnouncementSubscriptionsRequest) (announcementsservicesdk.ListAnnouncementSubscriptionsResponse, error),
) func(context.Context, *announcementsservicev1beta1.AnnouncementSubscription, string) (any, error) {
	return func(ctx context.Context, resource *announcementsservicev1beta1.AnnouncementSubscription, currentID string) (any, error) {
		if getCall == nil {
			return nil, fmt.Errorf("AnnouncementSubscription delete confirmation requires a get operation")
		}
		currentID = strings.TrimSpace(currentID)
		response, err := getCall(ctx, announcementsservicesdk.GetAnnouncementSubscriptionRequest{
			AnnouncementSubscriptionId: common.String(currentID),
		})
		if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
			return response, err
		}
		if listCall == nil || resource == nil {
			return nil, err
		}

		listResponse, listErr := listCall(ctx, announcementsservicesdk.ListAnnouncementSubscriptionsRequest{
			CompartmentId: common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
			Id:            common.String(currentID),
		})
		if listErr != nil {
			return nil, listErr
		}
		for _, item := range listResponse.Items {
			if item.Id != nil && strings.TrimSpace(*item.Id) == currentID {
				return nil, err
			}
		}
		return nil, errorutil.NotFoundOciError(errorutil.OciErrors{
			HTTPStatusCode: 404,
			ErrorCode:      errorutil.NotFound,
			OpcRequestID:   servicemanager.ErrorOpcRequestID(err),
			Description:    "announcement subscription not found after scoped delete confirmation",
		})
	}
}

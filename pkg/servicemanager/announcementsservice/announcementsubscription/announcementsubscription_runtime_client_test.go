/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package announcementsubscription

import (
	"context"
	"testing"

	announcementsservicesdk "github.com/oracle/oci-go-sdk/v65/announcementsservice"
	"github.com/oracle/oci-go-sdk/v65/common"
	announcementsservicev1beta1 "github.com/oracle/oci-service-operator/api/announcementsservice/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
)

func TestAnnouncementSubscriptionDeleteConfirmReadUsesScopedListForAuthShapedNotFound(t *testing.T) {
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization or existence is ambiguous")
	resource := &announcementsservicev1beta1.AnnouncementSubscription{
		Spec: announcementsservicev1beta1.AnnouncementSubscriptionSpec{CompartmentId: "ocid1.compartment.oc1..example"},
	}
	hook := announcementSubscriptionDeleteConfirmRead(
		func(context.Context, announcementsservicesdk.GetAnnouncementSubscriptionRequest) (announcementsservicesdk.GetAnnouncementSubscriptionResponse, error) {
			return announcementsservicesdk.GetAnnouncementSubscriptionResponse{}, authErr
		},
		func(_ context.Context, request announcementsservicesdk.ListAnnouncementSubscriptionsRequest) (announcementsservicesdk.ListAnnouncementSubscriptionsResponse, error) {
			if request.CompartmentId == nil || *request.CompartmentId != resource.Spec.CompartmentId {
				t.Fatalf("list compartment = %#v, want %q", request.CompartmentId, resource.Spec.CompartmentId)
			}
			if request.Id == nil || *request.Id != "ocid1.announcementsubscription.oc1..example" {
				t.Fatalf("list id = %#v, want tracked ID", request.Id)
			}
			return announcementsservicesdk.ListAnnouncementSubscriptionsResponse{}, nil
		},
	)

	_, err := hook(context.Background(), resource, "ocid1.announcementsubscription.oc1..example")
	if err == nil || !errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
		t.Fatalf("confirm read error = %v, want unambiguous NotFound", err)
	}
}

func TestAnnouncementSubscriptionRuntimeSemanticsModelsTerminalDeletion(t *testing.T) {
	semantics := announcementSubscriptionRuntimeSemantics()
	if semantics == nil || len(semantics.Lifecycle.ActiveStates) != 1 || semantics.Lifecycle.ActiveStates[0] != "ACTIVE" {
		t.Fatalf("lifecycle semantics = %#v, want ACTIVE", semantics)
	}
	if len(semantics.Delete.TerminalStates) != 1 || semantics.Delete.TerminalStates[0] != "DELETED" {
		t.Fatalf("delete semantics = %#v, want DELETED terminal state", semantics.Delete)
	}
	if semantics.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("delete follow-up = %q, want confirm-delete", semantics.DeleteFollowUp.Strategy)
	}
}

func TestAnnouncementSubscriptionDeleteConfirmReadRemainsConservativeWhenListFindsID(t *testing.T) {
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization or existence is ambiguous")
	resource := &announcementsservicev1beta1.AnnouncementSubscription{
		Spec: announcementsservicev1beta1.AnnouncementSubscriptionSpec{CompartmentId: "ocid1.compartment.oc1..example"},
	}
	hook := announcementSubscriptionDeleteConfirmRead(
		func(context.Context, announcementsservicesdk.GetAnnouncementSubscriptionRequest) (announcementsservicesdk.GetAnnouncementSubscriptionResponse, error) {
			return announcementsservicesdk.GetAnnouncementSubscriptionResponse{}, authErr
		},
		func(context.Context, announcementsservicesdk.ListAnnouncementSubscriptionsRequest) (announcementsservicesdk.ListAnnouncementSubscriptionsResponse, error) {
			return announcementsservicesdk.ListAnnouncementSubscriptionsResponse{
				AnnouncementSubscriptionCollection: announcementsservicesdk.AnnouncementSubscriptionCollection{
					Items: []announcementsservicesdk.AnnouncementSubscriptionSummary{{Id: common.String("ocid1.announcementsubscription.oc1..example")}},
				},
			}, nil
		},
	)

	_, err := hook(context.Background(), resource, "ocid1.announcementsubscription.oc1..example")
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		t.Fatalf("confirm read error = %v, want original ambiguous 404", err)
	}
}

/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package adhocquery

import (
	"encoding/json"
	"fmt"

	cloudguardsdk "github.com/oracle/oci-go-sdk/v65/cloudguard"
	cloudguardv1beta1 "github.com/oracle/oci-service-operator/api/cloudguard/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

func init() {
	registerAdhocQueryRuntimeHooksMutator(func(_ *AdhocQueryServiceManager, hooks *AdhocQueryRuntimeHooks) {
		hooks.Semantics = newAdhocQueryRuntimeSemantics()
		hooks.StatusHooks.ProjectStatus = projectAdhocQueryStatus
	})
}

func newAdhocQueryRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "cloudguard",
		FormalSlug:    "adhocquery",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{"CREATING"},
			ActiveStates:       []string{"ACTIVE"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{"DELETING"},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId"},
		},
		Mutation: generatedruntime.MutationSemantics{
			ForceNew:      []string{"compartmentId", "adhocQueryDetails"},
			ConflictsWith: map[string][]string{},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func projectAdhocQueryStatus(resource *cloudguardv1beta1.AdhocQuery, response any) error {
	if resource == nil {
		return fmt.Errorf("AdhocQuery resource is nil")
	}
	current, ok := adhocQueryFromResponse(response)
	if !ok {
		return nil
	}
	payload, err := json.Marshal(current)
	if err != nil {
		return fmt.Errorf("marshal AdhocQuery status: %w", err)
	}
	values := map[string]any{}
	if err := json.Unmarshal(payload, &values); err != nil {
		return fmt.Errorf("decode AdhocQuery status: %w", err)
	}
	if sdkStatus, exists := values["status"]; exists {
		values["sdkStatus"] = sdkStatus
		delete(values, "status")
	}
	payload, err = json.Marshal(values)
	if err != nil {
		return fmt.Errorf("marshal projected AdhocQuery status: %w", err)
	}
	projected := cloudguardv1beta1.AdhocQueryStatus{OsokStatus: resource.Status.OsokStatus}
	if err := json.Unmarshal(payload, &projected); err != nil {
		return fmt.Errorf("decode projected AdhocQuery status: %w", err)
	}
	resource.Status = projected
	if resource.Status.Id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)
	}
	return nil
}

func adhocQueryFromResponse(response any) (cloudguardsdk.AdhocQuery, bool) {
	switch current := response.(type) {
	case cloudguardsdk.CreateAdhocQueryResponse:
		return current.AdhocQuery, current.AdhocQuery.Id != nil
	case *cloudguardsdk.CreateAdhocQueryResponse:
		if current != nil {
			return current.AdhocQuery, current.AdhocQuery.Id != nil
		}
	case cloudguardsdk.GetAdhocQueryResponse:
		return current.AdhocQuery, current.AdhocQuery.Id != nil
	case *cloudguardsdk.GetAdhocQueryResponse:
		if current != nil {
			return current.AdhocQuery, current.AdhocQuery.Id != nil
		}
	case cloudguardsdk.AdhocQuery:
		return current, current.Id != nil
	case *cloudguardsdk.AdhocQuery:
		if current != nil {
			return *current, current.Id != nil
		}
	}
	return cloudguardsdk.AdhocQuery{}, false
}

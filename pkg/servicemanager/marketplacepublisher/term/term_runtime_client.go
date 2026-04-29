/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package term

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	marketplacepublishersdk "github.com/oracle/oci-go-sdk/v65/marketplacepublisher"
	marketplacepublisherv1beta1 "github.com/oracle/oci-service-operator/api/marketplacepublisher/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
)

type termOCIClient interface {
	CreateTerm(context.Context, marketplacepublishersdk.CreateTermRequest) (marketplacepublishersdk.CreateTermResponse, error)
	GetTerm(context.Context, marketplacepublishersdk.GetTermRequest) (marketplacepublishersdk.GetTermResponse, error)
	ListTerms(context.Context, marketplacepublishersdk.ListTermsRequest) (marketplacepublishersdk.ListTermsResponse, error)
	UpdateTerm(context.Context, marketplacepublishersdk.UpdateTermRequest) (marketplacepublishersdk.UpdateTermResponse, error)
	DeleteTerm(context.Context, marketplacepublishersdk.DeleteTermRequest) (marketplacepublishersdk.DeleteTermResponse, error)
}

type termDeleteConfirmationClient struct {
	delegate  TermServiceClient
	initErr   error
	getTerm   func(context.Context, marketplacepublishersdk.GetTermRequest) (marketplacepublishersdk.GetTermResponse, error)
	listTerms func(context.Context, marketplacepublishersdk.ListTermsRequest) (marketplacepublishersdk.ListTermsResponse, error)
}

type termAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e termAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e termAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerTermRuntimeHooksMutator(func(_ *TermServiceManager, hooks *TermRuntimeHooks) {
		applyTermRuntimeHooks(hooks)
	})
}

func applyTermRuntimeHooks(hooks *TermRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newTermRuntimeSemantics()
	hooks.BuildCreateBody = buildTermCreateBody
	hooks.BuildUpdateBody = buildTermUpdateBody
	hooks.List.Fields = termListFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listTermsAllPages(hooks.List.Call)
	}
	hooks.DeleteHooks.HandleError = handleTermDeleteError
	wrapTermDeleteConfirmation(hooks)
}

func newTermRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "marketplacepublisher",
		FormalSlug:    "term",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ActiveStates: []string{
				string(marketplacepublishersdk.TermLifecycleStateActive),
				string(marketplacepublishersdk.TermLifecycleStateInactive),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "best-effort",
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "name", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"freeformTags", "definedTags"},
			ForceNew:      []string{"compartmentId", "name"},
			ConflictsWith: map[string][]string{},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func termListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "Name", RequestName: "name", Contribution: "query", LookupPaths: []string{"status.name", "spec.name", "metadataName", "name"}},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func buildTermCreateBody(_ context.Context, resource *marketplacepublisherv1beta1.Term, _ string) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("term resource is nil")
	}
	if err := validateTermSpec(resource.Spec); err != nil {
		return nil, err
	}

	body := marketplacepublishersdk.CreateTermDetails{
		CompartmentId: common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		Name:          common.String(strings.TrimSpace(resource.Spec.Name)),
	}
	if resource.Spec.FreeformTags != nil {
		body.FreeformTags = cloneTermStringMap(resource.Spec.FreeformTags)
	}
	if resource.Spec.DefinedTags != nil {
		body.DefinedTags = termDefinedTags(resource.Spec.DefinedTags)
	}
	return body, nil
}

func buildTermUpdateBody(_ context.Context, resource *marketplacepublisherv1beta1.Term, _ string, currentResponse any) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("term resource is nil")
	}
	if err := validateTermSpec(resource.Spec); err != nil {
		return nil, false, err
	}
	current, ok := termBodyFromResponse(currentResponse)
	if !ok {
		return nil, false, fmt.Errorf("current Term response does not expose a Term body")
	}

	body := marketplacepublishersdk.UpdateTermDetails{}
	updateNeeded := false
	if resource.Spec.FreeformTags != nil {
		desired := cloneTermStringMap(resource.Spec.FreeformTags)
		body.FreeformTags = desired
		if !reflect.DeepEqual(current.FreeformTags, desired) {
			updateNeeded = true
		}
	}
	if resource.Spec.DefinedTags != nil {
		desired := termDefinedTags(resource.Spec.DefinedTags)
		body.DefinedTags = desired
		if !reflect.DeepEqual(current.DefinedTags, desired) {
			updateNeeded = true
		}
	}
	return body, updateNeeded, nil
}

func validateTermSpec(spec marketplacepublisherv1beta1.TermSpec) error {
	var missing []string
	if strings.TrimSpace(spec.CompartmentId) == "" {
		missing = append(missing, "compartmentId")
	}
	if strings.TrimSpace(spec.Name) == "" {
		missing = append(missing, "name")
	}
	if len(missing) > 0 {
		return fmt.Errorf("term spec is missing required field(s): %s", strings.Join(missing, ", "))
	}
	return nil
}

func listTermsAllPages(
	call func(context.Context, marketplacepublishersdk.ListTermsRequest) (marketplacepublishersdk.ListTermsResponse, error),
) func(context.Context, marketplacepublishersdk.ListTermsRequest) (marketplacepublishersdk.ListTermsResponse, error) {
	return func(ctx context.Context, request marketplacepublishersdk.ListTermsRequest) (marketplacepublishersdk.ListTermsResponse, error) {
		var combined marketplacepublishersdk.ListTermsResponse
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
				return combined, nil
			}
			request.Page = response.OpcNextPage
		}
	}
}

func wrapTermDeleteConfirmation(hooks *TermRuntimeHooks) {
	if hooks == nil || hooks.Get.Call == nil {
		return
	}
	getTerm := hooks.Get.Call
	listTerms := hooks.List.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate TermServiceClient) TermServiceClient {
		return termDeleteConfirmationClient{
			delegate:  delegate,
			initErr:   termGeneratedDelegateInitError(delegate),
			getTerm:   getTerm,
			listTerms: listTerms,
		}
	})
}

func (c termDeleteConfirmationClient) CreateOrUpdate(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.Term,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("term runtime client is not configured")
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c termDeleteConfirmationClient) Delete(ctx context.Context, resource *marketplacepublisherv1beta1.Term) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("term runtime client is not configured")
	}
	if c.initErr != nil {
		return false, c.initErr
	}
	if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c termDeleteConfirmationClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.Term,
) error {
	if c.getTerm == nil || resource == nil {
		return nil
	}
	if currentID := currentTermID(resource); currentID != "" {
		return c.rejectAuthShapedGet(ctx, resource, currentID)
	}
	return c.rejectAuthShapedListMatch(ctx, resource)
}

func (c termDeleteConfirmationClient) rejectAuthShapedListMatch(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.Term,
) error {
	if !termSpecHasListIdentity(resource.Spec) || c.listTerms == nil {
		return nil
	}

	response, err := c.listTerms(ctx, marketplacepublishersdk.ListTermsRequest{
		CompartmentId: common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		Name:          common.String(strings.TrimSpace(resource.Spec.Name)),
	})
	if err != nil {
		return rejectTermAmbiguousNotFound(resource, err, "delete lookup")
	}

	return c.rejectAuthShapedMatchedTerms(ctx, resource, matchingTermSummaries(response.Items, resource.Spec))
}

func (c termDeleteConfirmationClient) rejectAuthShapedMatchedTerms(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.Term,
	matches []marketplacepublishersdk.TermSummary,
) error {
	switch len(matches) {
	case 0:
		return nil
	case 1:
		return c.rejectAuthShapedGet(ctx, resource, stringValue(matches[0].Id))
	default:
		return fmt.Errorf("multiple OCI Terms matched compartmentId %q and name %q", resource.Spec.CompartmentId, resource.Spec.Name)
	}
}

func matchingTermSummaries(
	items []marketplacepublishersdk.TermSummary,
	spec marketplacepublisherv1beta1.TermSpec,
) []marketplacepublishersdk.TermSummary {
	var matches []marketplacepublishersdk.TermSummary
	for _, item := range items {
		if termSummaryMatchesSpec(item, spec) {
			matches = append(matches, item)
		}
	}
	return matches
}

func termSpecHasListIdentity(spec marketplacepublisherv1beta1.TermSpec) bool {
	return strings.TrimSpace(spec.CompartmentId) != "" && strings.TrimSpace(spec.Name) != ""
}

func (c termDeleteConfirmationClient) rejectAuthShapedGet(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.Term,
	termID string,
) error {
	if strings.TrimSpace(termID) == "" {
		return nil
	}
	_, err := c.getTerm(ctx, marketplacepublishersdk.GetTermRequest{TermId: common.String(strings.TrimSpace(termID))})
	if err == nil {
		return nil
	}
	return rejectTermAmbiguousNotFound(resource, err, "delete confirmation")
}

func handleTermDeleteError(resource *marketplacepublisherv1beta1.Term, err error) error {
	return rejectTermAmbiguousNotFound(resource, err, "delete")
}

func rejectTermAmbiguousNotFound(resource *marketplacepublisherv1beta1.Term, err error, operation string) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	classification := errorutil.ClassifyDeleteError(err)
	if classification.IsUnambiguousNotFound() {
		return nil
	}
	if !classification.IsAuthShapedNotFound() {
		return err
	}
	return termAmbiguousNotFoundError{
		message:      fmt.Sprintf("term %s returned ambiguous 404 NotAuthorizedOrNotFound; retaining finalizer until deletion is unambiguously confirmed: %v", operation, err),
		opcRequestID: servicemanager.ErrorOpcRequestID(err),
	}
}

func currentTermID(resource *marketplacepublisherv1beta1.Term) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); id != "" {
		return id
	}
	return strings.TrimSpace(resource.Status.Id)
}

func termGeneratedDelegateInitError(delegate TermServiceClient) error {
	if delegate == nil {
		return nil
	}

	var resource *marketplacepublisherv1beta1.Term
	_, err := delegate.Delete(context.Background(), resource)
	if err == nil || isTermNilResourceProbeError(err) {
		return nil
	}
	return err
}

func isTermNilResourceProbeError(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "resource is nil") || strings.Contains(message, "expected pointer resource")
}

type termBody struct {
	Id             *string                                        `json:"id,omitempty"`
	CompartmentId  *string                                        `json:"compartmentId,omitempty"`
	Name           *string                                        `json:"name,omitempty"`
	Author         marketplacepublishersdk.TermAuthorEnum         `json:"author,omitempty"`
	LifecycleState marketplacepublishersdk.TermLifecycleStateEnum `json:"lifecycleState,omitempty"`
	FreeformTags   map[string]string                              `json:"freeformTags,omitempty"`
	DefinedTags    map[string]map[string]interface{}              `json:"definedTags,omitempty"`
}

func termBodyFromResponse(response any) (termBody, bool) {
	switch current := response.(type) {
	case marketplacepublishersdk.CreateTermResponse:
		return termBodyFromSDKTerm(current.Term), true
	case marketplacepublishersdk.GetTermResponse:
		return termBodyFromSDKTerm(current.Term), true
	case marketplacepublishersdk.UpdateTermResponse:
		return termBodyFromSDKTerm(current.Term), true
	case marketplacepublishersdk.Term:
		return termBodyFromSDKTerm(current), true
	case marketplacepublishersdk.TermSummary:
		return termBodyFromSDKSummary(current), true
	case *marketplacepublishersdk.Term:
		if current == nil {
			return termBody{}, false
		}
		return termBodyFromSDKTerm(*current), true
	case *marketplacepublishersdk.TermSummary:
		if current == nil {
			return termBody{}, false
		}
		return termBodyFromSDKSummary(*current), true
	default:
		return termBody{}, false
	}
}

func termBodyFromSDKTerm(term marketplacepublishersdk.Term) termBody {
	return termBody{
		Id:             term.Id,
		CompartmentId:  term.CompartmentId,
		Name:           term.Name,
		Author:         term.Author,
		LifecycleState: term.LifecycleState,
		FreeformTags:   term.FreeformTags,
		DefinedTags:    term.DefinedTags,
	}
}

func termBodyFromSDKSummary(summary marketplacepublishersdk.TermSummary) termBody {
	return termBody{
		Id:             summary.Id,
		CompartmentId:  summary.CompartmentId,
		Name:           summary.Name,
		Author:         summary.Author,
		LifecycleState: summary.LifecycleState,
		FreeformTags:   summary.FreeformTags,
		DefinedTags:    summary.DefinedTags,
	}
}

func termSummaryMatchesSpec(summary marketplacepublishersdk.TermSummary, spec marketplacepublisherv1beta1.TermSpec) bool {
	return stringValue(summary.CompartmentId) == strings.TrimSpace(spec.CompartmentId) &&
		stringValue(summary.Name) == strings.TrimSpace(spec.Name)
}

func cloneTermStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func termDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	return *util.ConvertToOciDefinedTags(&tags)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func newTermServiceClientWithOCIClient(log loggerutil.OSOKLogger, client termOCIClient) TermServiceClient {
	hooks := newTermRuntimeHooksWithOCIClient(client)
	applyTermRuntimeHooks(&hooks)
	delegate := defaultTermServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*marketplacepublisherv1beta1.Term](
			buildTermGeneratedRuntimeConfig(&TermServiceManager{Log: log}, hooks),
		),
	}
	return wrapTermGeneratedClient(hooks, delegate)
}

func newTermRuntimeHooksWithOCIClient(client termOCIClient) TermRuntimeHooks {
	return TermRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*marketplacepublisherv1beta1.Term]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*marketplacepublisherv1beta1.Term]{},
		StatusHooks:     generatedruntime.StatusHooks[*marketplacepublisherv1beta1.Term]{},
		ParityHooks:     generatedruntime.ParityHooks[*marketplacepublisherv1beta1.Term]{},
		Async:           generatedruntime.AsyncHooks[*marketplacepublisherv1beta1.Term]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*marketplacepublisherv1beta1.Term]{},
		Create: runtimeOperationHooks[marketplacepublishersdk.CreateTermRequest, marketplacepublishersdk.CreateTermResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateTermDetails", RequestName: "CreateTermDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request marketplacepublishersdk.CreateTermRequest) (marketplacepublishersdk.CreateTermResponse, error) {
				if client == nil {
					return marketplacepublishersdk.CreateTermResponse{}, fmt.Errorf("term OCI client is nil")
				}
				return client.CreateTerm(ctx, request)
			},
		},
		Get: runtimeOperationHooks[marketplacepublishersdk.GetTermRequest, marketplacepublishersdk.GetTermResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "TermId", RequestName: "termId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request marketplacepublishersdk.GetTermRequest) (marketplacepublishersdk.GetTermResponse, error) {
				if client == nil {
					return marketplacepublishersdk.GetTermResponse{}, fmt.Errorf("term OCI client is nil")
				}
				return client.GetTerm(ctx, request)
			},
		},
		List: runtimeOperationHooks[marketplacepublishersdk.ListTermsRequest, marketplacepublishersdk.ListTermsResponse]{
			Fields: termListFields(),
			Call: func(ctx context.Context, request marketplacepublishersdk.ListTermsRequest) (marketplacepublishersdk.ListTermsResponse, error) {
				if client == nil {
					return marketplacepublishersdk.ListTermsResponse{}, fmt.Errorf("term OCI client is nil")
				}
				return client.ListTerms(ctx, request)
			},
		},
		Update: runtimeOperationHooks[marketplacepublishersdk.UpdateTermRequest, marketplacepublishersdk.UpdateTermResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "TermId", RequestName: "termId", Contribution: "path", PreferResourceID: true}, {FieldName: "UpdateTermDetails", RequestName: "UpdateTermDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request marketplacepublishersdk.UpdateTermRequest) (marketplacepublishersdk.UpdateTermResponse, error) {
				if client == nil {
					return marketplacepublishersdk.UpdateTermResponse{}, fmt.Errorf("term OCI client is nil")
				}
				return client.UpdateTerm(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[marketplacepublishersdk.DeleteTermRequest, marketplacepublishersdk.DeleteTermResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "TermId", RequestName: "termId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request marketplacepublishersdk.DeleteTermRequest) (marketplacepublishersdk.DeleteTermResponse, error) {
				if client == nil {
					return marketplacepublishersdk.DeleteTermResponse{}, fmt.Errorf("term OCI client is nil")
				}
				return client.DeleteTerm(ctx, request)
			},
		},
		WrapGeneratedClient: []func(TermServiceClient) TermServiceClient{},
	}
}

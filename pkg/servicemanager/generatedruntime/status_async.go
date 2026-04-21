/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c ServiceClient[T]) applySuccess(resource T, response any, fallback shared.OSOKConditionType) (servicemanager.OSOKResponse, error) {
	if err := mergeResponseIntoStatus(resource, response); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	stampSecretSourceStatus(resource)

	status, err := osokStatus(resource)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	resourceID := responseID(response)
	if resourceID == "" {
		resourceID = c.currentID(resource)
	}
	if resourceID != "" {
		status.Ocid = shared.OCID(resourceID)
	}

	now := metav1.Now()
	if resourceID != "" && status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	evaluation := c.classifyLifecycleAsync(response, status, fallback)
	if evaluation.current != nil {
		evaluation.current.UpdatedAt = &now
		projection := servicemanager.ApplyAsyncOperation(status, evaluation.current, c.config.Log)
		return servicemanager.OSOKResponse{
			IsSuccessful:    projection.Condition != shared.Failed,
			ShouldRequeue:   projection.ShouldRequeue,
			RequeueDuration: defaultRequeueDuration,
		}, nil
	}

	servicemanager.ClearAsyncOperation(status)
	status.Message = evaluation.message
	status.Reason = string(evaluation.condition)
	status.UpdatedAt = &now
	*status = util.UpdateOSOKStatusCondition(*status, evaluation.condition, conditionStatusForCondition(evaluation.condition), "", evaluation.message, c.config.Log)

	return servicemanager.OSOKResponse{
		IsSuccessful:    evaluation.condition != shared.Failed,
		ShouldRequeue:   evaluation.shouldRequeue,
		RequeueDuration: defaultRequeueDuration,
	}, nil
}

func (c ServiceClient[T]) markFailure(resource T, err error) error {
	status, statusErr := osokStatus(resource)
	if statusErr != nil {
		return err
	}
	servicemanager.RecordErrorOpcRequestID(status, err)
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	now := metav1.Now()
	status.UpdatedAt = &now
	if status.Async.Current != nil {
		current := *status.Async.Current
		current.NormalizedClass = shared.OSOKAsyncClassFailed
		current.Message = err.Error()
		current.UpdatedAt = &now
		_ = servicemanager.ApplyAsyncOperation(status, &current, c.config.Log)
		return err
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, v1.ConditionFalse, "", err.Error(), c.config.Log)
	return err
}

func (c ServiceClient[T]) markDeleted(resource T, message string) {
	status, err := osokStatus(resource)
	if err != nil {
		return
	}
	now := metav1.Now()
	status.DeletedAt = &now
	status.UpdatedAt = &now
	if message != "" {
		status.Message = message
	}
	status.Reason = string(shared.Terminating)
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", status.Message, c.config.Log)
}

func (c ServiceClient[T]) markCondition(resource T, condition shared.OSOKConditionType, message string) {
	status, err := osokStatus(resource)
	if err != nil {
		return
	}
	now := metav1.Now()
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(condition)
	if condition == shared.Terminating {
		current := &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           shared.OSOKAsyncPhaseDelete,
			NormalizedClass: shared.OSOKAsyncClassPending,
			Message:         message,
			UpdatedAt:       &now,
		}
		_ = servicemanager.ApplyAsyncOperation(status, current, c.config.Log)
		return
	}
	*status = util.UpdateOSOKStatusCondition(*status, condition, v1.ConditionTrue, "", message, c.config.Log)
}

func responseID(response any) string {
	body, ok := responseBody(response)
	if !ok || body == nil {
		return ""
	}
	values := jsonMap(body)
	return firstNonEmpty(values, "id", "ocid")
}

func responseWorkRequestID(response any) string {
	if response == nil {
		return ""
	}

	value, ok := indirectValue(reflect.ValueOf(response))
	if !ok || value.Kind() != reflect.Struct {
		return ""
	}

	typ := value.Type()
	for i := 0; i < value.NumField(); i++ {
		fieldType := typ.Field(i)
		if !fieldType.IsExported() || !isWorkRequestHeaderField(fieldType) {
			continue
		}
		if workRequestID := stringFieldValue(value.Field(i)); workRequestID != "" {
			return workRequestID
		}
	}
	return ""
}

func responseRequestID(response any) string {
	return servicemanager.ResponseOpcRequestID(response)
}

func isWorkRequestHeaderField(fieldType reflect.StructField) bool {
	return fieldType.Name == "OpcWorkRequestId" ||
		(fieldType.Tag.Get("presentIn") == "header" && fieldType.Tag.Get("name") == "opc-work-request-id")
}

func stringFieldValue(value reflect.Value) string {
	value, ok := indirectValue(value)
	if !ok || value.Kind() != reflect.String {
		return ""
	}
	return strings.TrimSpace(value.String())
}

func (c ServiceClient[T]) seedOpeningWorkRequestID(resource T, response any, phase shared.OSOKAsyncPhase) {
	workRequestID := responseWorkRequestID(response)
	if workRequestID == "" || phase == "" {
		return
	}

	status, err := osokStatus(resource)
	if err != nil {
		return
	}

	now := metav1.Now()
	_ = servicemanager.ApplyAsyncOperation(status, &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           phase,
		WorkRequestID:   workRequestID,
		NormalizedClass: shared.OSOKAsyncClassPending,
		UpdatedAt:       &now,
	}, c.config.Log)
}

func (c ServiceClient[T]) seedOpeningRequestID(resource T, response any) {
	status, err := osokStatus(resource)
	if err != nil {
		return
	}

	servicemanager.RecordResponseOpcRequestID(status, response)
}

func (c ServiceClient[T]) recordErrorRequestID(resource T, err error) {
	status, statusErr := osokStatus(resource)
	if statusErr != nil {
		return
	}

	servicemanager.RecordErrorOpcRequestID(status, err)
}

type lifecycleAsyncEvaluation struct {
	current       *shared.OSOKAsyncOperation
	condition     shared.OSOKConditionType
	shouldRequeue bool
	message       string
}

func (c ServiceClient[T]) classifyLifecycleAsync(response any, status *shared.OSOKStatus, fallback shared.OSOKConditionType) lifecycleAsyncEvaluation {
	if c.config.Semantics == nil {
		return classifyLifecycleAsyncHeuristics(response, status, fallback)
	}
	return classifyLifecycleAsyncSemantics(response, status, fallback, c.config.Semantics)
}

func classifyLifecycleSemantics(response any, fallback shared.OSOKConditionType, semantics *Semantics) (shared.OSOKConditionType, bool, string) {
	evaluation := classifyLifecycleAsyncSemantics(response, nil, fallback, semantics)
	return evaluation.condition, evaluation.shouldRequeue, evaluation.message
}

func classifyLifecycleAsyncSemantics(response any, status *shared.OSOKStatus, fallback shared.OSOKConditionType, semantics *Semantics) lifecycleAsyncEvaluation {
	body, ok := responseBody(response)
	if !ok || body == nil {
		return lifecycleAsyncEvaluation{
			condition:     fallback,
			shouldRequeue: shouldRequeueForCondition(fallback),
			message:       defaultConditionMessage(fallback),
		}
	}

	values := jsonMap(body)
	lifecycleState := strings.ToUpper(firstNonEmpty(values, "lifecycleState", "status"))
	message := firstNonEmpty(values, "lifecycleDetails", "message", "displayName", "name")
	if message == "" {
		message = defaultConditionMessage(fallback)
	}

	switch {
	case lifecycleState == "":
		return lifecycleAsyncEvaluation{
			condition:     fallback,
			shouldRequeue: shouldRequeueForCondition(fallback),
			message:       message,
		}
	case containsString(semantics.Lifecycle.ProvisioningStates, lifecycleState):
		return newLifecycleAsyncEvaluation(status, message, lifecycleState, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassPending)
	case containsString(semantics.Lifecycle.UpdatingStates, lifecycleState):
		return newLifecycleAsyncEvaluation(status, message, lifecycleState, shared.OSOKAsyncPhaseUpdate, shared.OSOKAsyncClassPending)
	case containsString(semantics.Delete.PendingStates, lifecycleState):
		return newLifecycleAsyncEvaluation(status, message, lifecycleState, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending)
	case containsString(semantics.Delete.TerminalStates, lifecycleState):
		return newLifecycleAsyncEvaluation(status, message, lifecycleState, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassSucceeded)
	case containsString(semantics.Lifecycle.ActiveStates, lifecycleState):
		return lifecycleAsyncEvaluation{condition: shared.Active, shouldRequeue: false, message: message}
	default:
		failureMessage := fmt.Sprintf("formal lifecycle state %q is not modeled: %s", lifecycleState, message)
		return lifecycleFailureEvaluation(status, fallback, lifecycleState, failureMessage, shared.OSOKAsyncClassUnknown)
	}
}

func classifyLifecycleHeuristics(response any, fallback shared.OSOKConditionType) (shared.OSOKConditionType, bool, string) {
	evaluation := classifyLifecycleAsyncHeuristics(response, nil, fallback)
	return evaluation.condition, evaluation.shouldRequeue, evaluation.message
}

func classifyLifecycleAsyncHeuristics(response any, status *shared.OSOKStatus, fallback shared.OSOKConditionType) lifecycleAsyncEvaluation {
	body, ok := responseBody(response)
	if !ok || body == nil {
		return lifecycleAsyncEvaluation{
			condition:     fallback,
			shouldRequeue: shouldRequeueForCondition(fallback),
			message:       defaultConditionMessage(fallback),
		}
	}

	values := jsonMap(body)
	lifecycleState := strings.ToUpper(firstNonEmpty(values, "lifecycleState", "status"))
	message := firstNonEmpty(values, "lifecycleDetails", "message", "displayName", "name")
	if message == "" {
		message = defaultConditionMessage(fallback)
	}

	switch {
	case lifecycleState == "":
		return lifecycleAsyncEvaluation{
			condition:     fallback,
			shouldRequeue: shouldRequeueForCondition(fallback),
			message:       message,
		}
	case strings.Contains(lifecycleState, "FAIL"),
		strings.Contains(lifecycleState, "ERROR"),
		strings.Contains(lifecycleState, "INOPERABLE"):
		return lifecycleFailureEvaluation(status, fallback, lifecycleState, message, shared.OSOKAsyncClassFailed)
	case strings.Contains(lifecycleState, "NEEDS_ATTENTION"):
		return lifecycleFailureEvaluation(status, fallback, lifecycleState, message, shared.OSOKAsyncClassAttention)
	case strings.Contains(lifecycleState, "DELETED"),
		strings.Contains(lifecycleState, "TERMINATED"):
		return newLifecycleAsyncEvaluation(status, message, lifecycleState, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassSucceeded)
	case strings.Contains(lifecycleState, "DELETE"),
		strings.Contains(lifecycleState, "TERMINAT"):
		return newLifecycleAsyncEvaluation(status, message, lifecycleState, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending)
	case strings.Contains(lifecycleState, "UPDAT"),
		strings.Contains(lifecycleState, "MODIFY"),
		strings.Contains(lifecycleState, "PATCH"):
		return newLifecycleAsyncEvaluation(status, message, lifecycleState, shared.OSOKAsyncPhaseUpdate, shared.OSOKAsyncClassPending)
	case strings.Contains(lifecycleState, "CREATE"),
		strings.Contains(lifecycleState, "PROVISION"),
		strings.Contains(lifecycleState, "PENDING"),
		strings.Contains(lifecycleState, "IN_PROGRESS"),
		strings.Contains(lifecycleState, "ACCEPT"),
		strings.Contains(lifecycleState, "START"):
		return newLifecycleAsyncEvaluation(status, message, lifecycleState, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassPending)
	default:
		return lifecycleAsyncEvaluation{condition: shared.Active, shouldRequeue: false, message: message}
	}
}

func newLifecycleAsyncEvaluation(status *shared.OSOKStatus, message string, lifecycleState string, phase shared.OSOKAsyncPhase, class shared.OSOKAsyncNormalizedClass) lifecycleAsyncEvaluation {
	if phase == "" {
		return lifecycleAsyncEvaluation{condition: shared.Failed, shouldRequeue: false, message: message}
	}

	current := &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           servicemanager.ResolveAsyncPhase(status, phase),
		RawStatus:       lifecycleState,
		NormalizedClass: class,
		Message:         message,
	}
	projection := servicemanager.ProjectAsyncCondition(class, current.Phase)
	if strings.TrimSpace(message) == "" {
		message = projection.DefaultMessage
		current.Message = message
	}

	return lifecycleAsyncEvaluation{
		current:       current,
		condition:     projection.Condition,
		shouldRequeue: projection.ShouldRequeue,
		message:       message,
	}
}

func lifecycleFailureEvaluation(status *shared.OSOKStatus, fallback shared.OSOKConditionType, lifecycleState string, message string, class shared.OSOKAsyncNormalizedClass) lifecycleAsyncEvaluation {
	phase := servicemanager.ResolveAsyncPhase(status, fallbackAsyncPhase(fallback))
	if phase == "" {
		return lifecycleAsyncEvaluation{condition: shared.Failed, shouldRequeue: false, message: message}
	}
	return newLifecycleAsyncEvaluation(status, message, lifecycleState, phase, class)
}

func fallbackAsyncPhase(condition shared.OSOKConditionType) shared.OSOKAsyncPhase {
	switch condition {
	case shared.Provisioning:
		return shared.OSOKAsyncPhaseCreate
	case shared.Updating:
		return shared.OSOKAsyncPhaseUpdate
	case shared.Terminating:
		return shared.OSOKAsyncPhaseDelete
	default:
		return ""
	}
}

func shouldRequeueForCondition(condition shared.OSOKConditionType) bool {
	return condition == shared.Provisioning || condition == shared.Updating || condition == shared.Terminating
}

func conditionStatusForCondition(condition shared.OSOKConditionType) v1.ConditionStatus {
	if condition == shared.Failed {
		return v1.ConditionFalse
	}
	return v1.ConditionTrue
}

func defaultConditionMessage(condition shared.OSOKConditionType) string {
	switch condition {
	case shared.Provisioning:
		return "OCI resource provisioning is in progress"
	case shared.Updating:
		return "OCI resource update is in progress"
	case shared.Terminating:
		return "OCI resource delete is in progress"
	case shared.Failed:
		return "OCI resource reconcile failed"
	default:
		return "OCI resource is active"
	}
}

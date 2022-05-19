/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package manager

import (
	"context"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"

	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/errors"
)

type CustomResourceValidator interface {
	ValidateOnCreate(context context.Context, object client.Object) (bool, string)
	ValidateOnUpdate(context context.Context, object client.Object, oldObject client.Object) (bool, string)
	GetStatus(object client.Object) (*servicemeshapi.ServiceMeshStatus, error)
	ResolveRef(object client.Object) (bool, string)
	ValidateObject(object client.Object) error
	GetEntityType() client.Object
}

type ServiceMeshValidationManager struct {
	log               loggerutil.OSOKLogger
	validationManager CustomResourceValidator
}

func (v *ServiceMeshValidationManager) ValidateCreateRequest(ctx context.Context, object client.Object) admission.Response {

	err := v.validationManager.ValidateObject(object)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	allowed, reason := v.validationManager.ResolveRef(object)
	if !allowed {
		return admission.ValidationResponse(false, errors.GetValidationErrorMessage(object, reason))
	}

	// Additional validating condition can be added here
	allowed, reason = v.validationManager.ValidateOnCreate(ctx, object)
	if !allowed {
		return admission.ValidationResponse(false, errors.GetValidationErrorMessage(object, reason))
	}

	v.log.InfoLogWithFixedMessage(ctx, "Create Request Passes Validation", "Type", object.GetObjectKind().GroupVersionKind().Kind, "Name", object.GetName(), "Namespace", object.GetNamespace())
	return admission.ValidationResponse(true, "")
}

func (v *ServiceMeshValidationManager) ValidateUpdateRequest(ctx context.Context, object client.Object, oldObject client.Object) admission.Response {
	err := v.validationManager.ValidateObject(object)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	status, err := v.validationManager.GetStatus(oldObject)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if object.GetGeneration() == oldObject.GetGeneration() {
		return admission.ValidationResponse(true, "")
	}

	if v.validateServiceMeshCondition(servicemeshapi.ServiceMeshDependenciesActive, status, metav1.ConditionUnknown) {
		return admission.ValidationResponse(false, errors.GetValidationErrorMessage(object, string(commons.DependenciesIsUnknownOnUpdate)))
	}

	if v.validateServiceMeshCondition(servicemeshapi.ServiceMeshConfigured, status, metav1.ConditionUnknown) {
		return admission.ValidationResponse(false, errors.GetValidationErrorMessage(object, string(commons.UnknownStateOnUpdate)))
	}

	isResourceConfigured := v.validateServiceMeshCondition(servicemeshapi.ServiceMeshConfigured, status, metav1.ConditionTrue)
	if isResourceConfigured && !v.validateServiceMeshCondition(servicemeshapi.ServiceMeshActive, status, metav1.ConditionTrue) {
		return admission.ValidationResponse(false, errors.GetValidationErrorMessage(object, string(commons.NotActiveOnUpdate)))
	}

	allowed, reason := v.validationManager.ValidateOnUpdate(ctx, object, oldObject)
	if !allowed {
		return admission.ValidationResponse(false, errors.GetValidationErrorMessage(object, reason))
	}

	// Additional validating condition can be added here
	v.log.InfoLogWithFixedMessage(ctx, "Resource Update Request passes validation", "Type", object.GetObjectKind().GroupVersionKind().Kind, "Name", object.GetName(), "Namespace", object.GetNamespace())
	return admission.ValidationResponse(true, "")
}

func (v *ServiceMeshValidationManager) ValidateDeleteRequest(ctx context.Context, object client.Object) admission.Response {
	status, err := v.validationManager.GetStatus(object)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if v.validateServiceMeshCondition(servicemeshapi.ServiceMeshDependenciesActive, status, metav1.ConditionTrue) &&
		v.validateServiceMeshCondition(servicemeshapi.ServiceMeshConfigured, status, metav1.ConditionTrue) &&
		v.validateServiceMeshCondition(servicemeshapi.ServiceMeshActive, status, metav1.ConditionUnknown) {
		return admission.ValidationResponse(false, errors.GetValidationErrorMessage(object, string(commons.UnknownStatusOnDelete)))
	}

	// Additional validating condition can be added here
	v.log.InfoLogWithFixedMessage(ctx, "Resource Delete Request passes validation", "Type", object.GetObjectKind().GroupVersionKind().Kind, "Name", object.GetName(), "Namespace", object.GetNamespace())
	return admission.ValidationResponse(true, "")
}

func (v *ServiceMeshValidationManager) GetObject() client.Object {
	return v.validationManager.GetEntityType()
}

func (v *ServiceMeshValidationManager) validateServiceMeshCondition(conditionType servicemeshapi.ServiceMeshConditionType, oldStatus *servicemeshapi.ServiceMeshStatus, status metav1.ConditionStatus) bool {
	for _, condition := range oldStatus.Conditions {
		if condition.Type == conditionType && condition.Status == status {
			return true
		}
	}
	return false
}

func NewServiceMeshValidationManager(validationManager CustomResourceValidator, log loggerutil.OSOKLogger) *ServiceMeshValidationManager {
	return &ServiceMeshValidationManager{validationManager: validationManager, log: log}
}

/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package commons

import (
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
)

// GetServiceMeshCondition will get pointer to service mesh's existing condition.
func GetServiceMeshCondition(serviceMeshStatus *servicemeshapi.ServiceMeshStatus, conditionType servicemeshapi.ServiceMeshConditionType) *servicemeshapi.ServiceMeshCondition {
	for i := range serviceMeshStatus.Conditions {
		if serviceMeshStatus.Conditions[i].Type == conditionType {
			return &serviceMeshStatus.Conditions[i]
		}
	}
	return nil
}

// UpdateServiceMeshCondition will update service mesh's condition and return whether it needs to be updated.
func UpdateServiceMeshCondition(serviceMeshStatus *servicemeshapi.ServiceMeshStatus, conditionType servicemeshapi.ServiceMeshConditionType, status metav1.ConditionStatus, reason string, message string, generation int64) bool {
	now := metav1.Now()
	existingCondition := GetServiceMeshCondition(serviceMeshStatus, conditionType)
	if existingCondition == nil {
		newCondition := servicemeshapi.ServiceMeshCondition{
			Type: conditionType,
			ResourceCondition: servicemeshapi.ResourceCondition{
				Status:             status,
				LastTransitionTime: &now,
				Reason:             reason,
				Message:            message,
				ObservedGeneration: generation,
			},
		}
		serviceMeshStatus.Conditions = append(serviceMeshStatus.Conditions, newCondition)
		return true
	}

	hasChanged := false
	if existingCondition.Status != status {
		existingCondition.Status = status
		existingCondition.LastTransitionTime = &now
		hasChanged = true
	}
	if existingCondition.ObservedGeneration != generation {
		existingCondition.ObservedGeneration = generation
		hasChanged = true
	}
	if existingCondition.Reason != reason {
		existingCondition.Reason = reason
		hasChanged = true
	}
	if existingCondition.Message != message {
		existingCondition.Message = message
		hasChanged = true
	}
	return hasChanged
}

// GetConditionStatus returns the state of the condition based on its lifecycle state
func GetConditionStatus(state string) metav1.ConditionStatus {
	switch state {
	case Active:
		return metav1.ConditionTrue
	case Failed:
		return metav1.ConditionFalse
	case Deleted:
		return metav1.ConditionFalse
	default:
		return metav1.ConditionUnknown
	}
}

// GetReason returns a reason based on the state of the condition
func GetReason(status metav1.ConditionStatus) ResourceConditionReason {
	switch status {
	case metav1.ConditionTrue:
		return Successful
	default:
		return LifecycleStateChanged
	}
}

// GetMessage returns message based on the state
func GetMessage(state string) ResourceConditionMessage {
	switch state {
	case Active:
		return ResourceActive
	case Creating:
		return ResourceCreating
	case Updating:
		return ResourceUpdating
	case Deleting:
		return ResourceDeleting
	case Deleted:
		return ResourceDeleted
	default:
		return ResourceFailed
	}
}

// GetVirtualDeploymentBindingConditionReason returns a reason for VirtualDeploymentBinding based on the state of the condition
func GetVirtualDeploymentBindingConditionReason(status metav1.ConditionStatus) ResourceConditionReason {
	switch status {
	case metav1.ConditionTrue:
		return Successful
	default:
		return DependenciesNotResolved
	}
}

// GetVirtualDeploymentBindingConditionMessage returns message for VirtualDeploymentBinding based on the state of the condition
func GetVirtualDeploymentBindingConditionMessage(state string) ResourceConditionMessageVDB {
	switch state {
	case Active:
		return ResourceActiveVDB
	case Creating:
		return ResourceCreatingVDB
	case Updating:
		return ResourceUpdatingVDB
	case Deleting:
		return ResourceDeletingVDB
	case Deleted:
		return ResourceDeletedVDB
	default:
		return ResourceFailedVDB
	}
}

// GetConditionStatusFromK8sError returns the state of the condition based on the error returned from K8s
func GetConditionStatusFromK8sError(err error) metav1.ConditionStatus {
	status := metav1.ConditionUnknown
	if kerrors.IsNotFound(err) || kerrors.IsResourceExpired(err) {
		status = metav1.ConditionFalse
	}
	return status
}

/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package servicemanager

import (
	"strings"

	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AsyncProjection struct {
	Condition       shared.OSOKConditionType
	ConditionStatus v1.ConditionStatus
	ShouldRequeue   bool
	DefaultMessage  string
}

func ProjectAsyncCondition(class shared.OSOKAsyncNormalizedClass, phase shared.OSOKAsyncPhase) AsyncProjection {
	action := asyncActionLabel(phase)
	condition := conditionForAsyncPhase(phase)

	switch class {
	case shared.OSOKAsyncClassPending:
		return AsyncProjection{
			Condition:       condition,
			ConditionStatus: v1.ConditionTrue,
			ShouldRequeue:   true,
			DefaultMessage:  "OCI " + action + " is in progress",
		}
	case shared.OSOKAsyncClassSucceeded:
		if phase == shared.OSOKAsyncPhaseDelete {
			return AsyncProjection{
				Condition:       shared.Terminating,
				ConditionStatus: v1.ConditionTrue,
				ShouldRequeue:   true,
				DefaultMessage:  "OCI delete completed; waiting for final confirmation",
			}
		}
		return AsyncProjection{
			Condition:       shared.Active,
			ConditionStatus: v1.ConditionTrue,
			ShouldRequeue:   false,
			DefaultMessage:  "OCI " + action + " completed",
		}
	case shared.OSOKAsyncClassCanceled:
		return AsyncProjection{
			Condition:       shared.Failed,
			ConditionStatus: v1.ConditionFalse,
			ShouldRequeue:   false,
			DefaultMessage:  "OCI " + action + " was canceled",
		}
	case shared.OSOKAsyncClassAttention:
		return AsyncProjection{
			Condition:       shared.Failed,
			ConditionStatus: v1.ConditionFalse,
			ShouldRequeue:   false,
			DefaultMessage:  "OCI " + action + " requires attention",
		}
	case shared.OSOKAsyncClassUnknown:
		return AsyncProjection{
			Condition:       shared.Failed,
			ConditionStatus: v1.ConditionFalse,
			ShouldRequeue:   false,
			DefaultMessage:  "OCI " + action + " status is unknown",
		}
	default:
		return AsyncProjection{
			Condition:       shared.Failed,
			ConditionStatus: v1.ConditionFalse,
			ShouldRequeue:   false,
			DefaultMessage:  "OCI " + action + " failed",
		}
	}
}

func ApplyAsyncOperation(status *shared.OSOKStatus, current *shared.OSOKAsyncOperation, log loggerutil.OSOKLogger) AsyncProjection {
	if status == nil || current == nil {
		return AsyncProjection{}
	}

	projection := ProjectAsyncCondition(current.NormalizedClass, current.Phase)
	message := strings.TrimSpace(current.Message)
	if message == "" {
		message = projection.DefaultMessage
	}

	updatedAt := current.UpdatedAt
	if updatedAt == nil {
		now := metav1.Now()
		updatedAt = &now
	}

	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:           current.Source,
		Phase:            current.Phase,
		WorkRequestID:    strings.TrimSpace(current.WorkRequestID),
		RawStatus:        strings.TrimSpace(current.RawStatus),
		RawOperationType: strings.TrimSpace(current.RawOperationType),
		NormalizedClass:  current.NormalizedClass,
		PercentComplete:  cloneFloat32Ptr(current.PercentComplete),
		Message:          message,
		UpdatedAt:        updatedAt.DeepCopy(),
	}
	status.UpdatedAt = updatedAt.DeepCopy()
	status.Message = message
	status.Reason = string(projection.Condition)
	*status = util.UpdateOSOKStatusCondition(*status, projection.Condition, projection.ConditionStatus, "", message, log)

	return projection
}

func ClearAsyncOperation(status *shared.OSOKStatus) {
	if status == nil {
		return
	}
	status.Async.Current = nil
}

func ResolveAsyncPhase(status *shared.OSOKStatus, explicit shared.OSOKAsyncPhase) shared.OSOKAsyncPhase {
	if explicit != "" {
		return explicit
	}
	if status != nil && status.Async.Current != nil && status.Async.Current.Phase != "" {
		return status.Async.Current.Phase
	}
	return ""
}

func conditionForAsyncPhase(phase shared.OSOKAsyncPhase) shared.OSOKConditionType {
	switch phase {
	case shared.OSOKAsyncPhaseUpdate:
		return shared.Updating
	case shared.OSOKAsyncPhaseDelete:
		return shared.Terminating
	default:
		return shared.Provisioning
	}
}

func asyncActionLabel(phase shared.OSOKAsyncPhase) string {
	switch phase {
	case shared.OSOKAsyncPhaseUpdate:
		return "update"
	case shared.OSOKAsyncPhaseDelete:
		return "delete"
	case shared.OSOKAsyncPhaseCreate:
		return "create"
	default:
		return "reconcile"
	}
}

func cloneFloat32Ptr(in *float32) *float32 {
	if in == nil {
		return nil
	}
	out := new(float32)
	*out = *in
	return out
}

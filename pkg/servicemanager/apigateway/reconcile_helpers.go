/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package apigateway

import (
	"errors"
	"fmt"

	apigatewaysdk "github.com/oracle/oci-go-sdk/v65/apigateway"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
)

func apiGatewayBadRequestCode(err error) (string, bool) {
	var badRequest errorutil.BadRequestOciError
	if !errors.As(err, &badRequest) {
		return "", false
	}
	return badRequest.ErrorCode, true
}

func safeGatewayString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func applyGatewayCreateFailure(status *shared.OSOKStatus, err error, log loggerutil.OSOKLogger, kind string) {
	servicemanager.RecordErrorOpcRequestID(status, err)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, v1.ConditionFalse, "", err.Error(), log)
	if code, ok := apiGatewayBadRequestCode(err); ok {
		status.Message = code
		log.ErrorLog(err, fmt.Sprintf("Create %s bad request", kind))
		return
	}
	log.ErrorLog(err, fmt.Sprintf("Create %s failed", kind))
}

func setGatewayProvisioning(status *shared.OSOKStatus, kind, displayName string, ocid shared.OCID,
	log loggerutil.OSOKLogger) {
	status.Ocid = ocid
	*status = util.UpdateOSOKStatusCondition(*status, shared.Provisioning, v1.ConditionTrue, "",
		fmt.Sprintf("%s %s Provisioning", kind, displayName), log)
}

func reconcileGatewayLifecycle(status *shared.OSOKStatus, instance *apigatewaysdk.Gateway,
	log loggerutil.OSOKLogger) servicemanager.OSOKResponse {
	displayName := safeGatewayString(instance.DisplayName)
	state := string(instance.LifecycleState)

	switch instance.LifecycleState {
	case apigatewaysdk.GatewayLifecycleStateFailed, apigatewaysdk.GatewayLifecycleStateDeleted:
		*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, v1.ConditionFalse, "",
			fmt.Sprintf("ApiGateway %s is %s", displayName, state), log)
		log.InfoLog(fmt.Sprintf("ApiGateway %s is %s", displayName, state))
		return servicemanager.OSOKResponse{IsSuccessful: false}
	case apigatewaysdk.GatewayLifecycleStateActive:
		*status = util.UpdateOSOKStatusCondition(*status, shared.Active, v1.ConditionTrue, "",
			fmt.Sprintf("ApiGateway %s is %s", displayName, state), log)
		log.InfoLog(fmt.Sprintf("ApiGateway %s is Active", displayName))
		return servicemanager.OSOKResponse{IsSuccessful: true}
	default:
		*status = util.UpdateOSOKStatusCondition(*status, shared.Provisioning, v1.ConditionTrue, "",
			fmt.Sprintf("ApiGateway %s is %s", displayName, state), log)
		log.InfoLog(fmt.Sprintf("ApiGateway %s is %s, requeueing", displayName, state))
		return servicemanager.OSOKResponse{IsSuccessful: false, ShouldRequeue: true}
	}
}

func reconcileDeploymentLifecycle(status *shared.OSOKStatus, instance *apigatewaysdk.Deployment,
	log loggerutil.OSOKLogger) servicemanager.OSOKResponse {
	displayName := safeGatewayString(instance.DisplayName)
	state := string(instance.LifecycleState)

	switch instance.LifecycleState {
	case apigatewaysdk.DeploymentLifecycleStateFailed, apigatewaysdk.DeploymentLifecycleStateDeleted:
		*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, v1.ConditionFalse, "",
			fmt.Sprintf("ApiGatewayDeployment %s is %s", displayName, state), log)
		log.InfoLog(fmt.Sprintf("ApiGatewayDeployment %s is %s", displayName, state))
		return servicemanager.OSOKResponse{IsSuccessful: false}
	case apigatewaysdk.DeploymentLifecycleStateActive:
		*status = util.UpdateOSOKStatusCondition(*status, shared.Active, v1.ConditionTrue, "",
			fmt.Sprintf("ApiGatewayDeployment %s is %s", displayName, state), log)
		log.InfoLog(fmt.Sprintf("ApiGatewayDeployment %s is Active", displayName))
		return servicemanager.OSOKResponse{IsSuccessful: true}
	default:
		*status = util.UpdateOSOKStatusCondition(*status, shared.Provisioning, v1.ConditionTrue, "",
			fmt.Sprintf("ApiGatewayDeployment %s is %s", displayName, state), log)
		log.InfoLog(fmt.Sprintf("ApiGatewayDeployment %s is %s, requeueing", displayName, state))
		return servicemanager.OSOKResponse{IsSuccessful: false, ShouldRequeue: true}
	}
}

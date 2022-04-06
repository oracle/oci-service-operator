/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dbsystem

import (
	"context"
	"errors"
	"fmt"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/mysql"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"strings"
	"time"
)

type DbSystemServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
}

func NewDbSystemServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger) *DbSystemServiceManager {
	return &DbSystemServiceManager{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
	}
}

func (c *DbSystemServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {

	mysqlDbSystem, err := c.convert(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	var mySqlDbSystemInstance *mysql.DbSystem
	if strings.TrimSpace(string(mysqlDbSystem.Spec.MySqlDbSystemId)) == "" {

		c.Log.DebugLog("MySqlDbSystem Id is empty. Check if mysql DB exists.")

		mySqlDbSystemOcid, err := c.GetMySqlDbSystemOcid(ctx, *mysqlDbSystem)
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if mySqlDbSystemOcid == nil {
			// Geting the Admin Secret from the secret before creating MySqlDbSystem instance
			c.Log.DebugLog("Getting Admin Username from Secret")
			unameMap, err := c.CredentialClient.GetSecret(ctx, mysqlDbSystem.Spec.AdminUsername.Secret.SecretName, req.Namespace)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting the admin secret")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}

			uname, ok := unameMap["username"]
			if !ok {
				c.Log.ErrorLog(err, "username key in admin secret is not found")
				return servicemanager.OSOKResponse{IsSuccessful: false}, errors.New("username key in admin secret is not found")
			}

			c.Log.DebugLog("Getting Admin password from Secret")
			pwdMap, err := c.CredentialClient.GetSecret(ctx, mysqlDbSystem.Spec.AdminPassword.Secret.SecretName, req.Namespace)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting the admin secret")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}

			pwd, ok := pwdMap["password"]
			if !ok {
				c.Log.ErrorLog(err, "password key in admin secret is not found")
				return servicemanager.OSOKResponse{IsSuccessful: false}, errors.New("password key in admin secret is not found")
			}

			resp, err := c.CreateDbSystem(ctx, *mysqlDbSystem, string(uname), string(pwd))
			if err != nil {
				mysqlDbSystem.Status.OsokStatus = util.UpdateOSOKStatusCondition(mysqlDbSystem.Status.OsokStatus,
					ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
				if _, ok := err.(errorutil.BadRequestOciError); !ok {
					c.Log.ErrorLog(err, "Assertion Error for BadRequestOciError")
					return servicemanager.OSOKResponse{IsSuccessful: false}, err
				} else {
					mysqlDbSystem.Status.OsokStatus.Message = err.(common.ServiceError).GetCode()
					c.Log.ErrorLog(err, "Create MySqlDbSystem failed")
					return servicemanager.OSOKResponse{IsSuccessful: false}, err
				}
			}
			c.Log.InfoLog(fmt.Sprintf("MySqlDbSystem %s is Provisioning", mysqlDbSystem.Spec.DisplayName))
			mysqlDbSystem.Status.OsokStatus = util.UpdateOSOKStatusCondition(mysqlDbSystem.Status.OsokStatus,
				ociv1beta1.Provisioning, v1.ConditionTrue, "", "MySqlDbSystem Provisioning", c.Log)
			mysqlDbSystem.Status.OsokStatus.Ocid = ociv1beta1.OCID(*resp.Id)
			retryPolicy := c.getDbSystemRetryPolicy(30)

			mySqlDbSystemInstance, err = c.GetMySqlDbSystem(ctx, ociv1beta1.OCID(*resp.Id), &retryPolicy)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting MySqlDbSystem")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
		} else {
			c.Log.InfoLog(fmt.Sprintf("Getting MySqlDbSystem %s", *mySqlDbSystemOcid))
			mySqlDbSystemInstance, err = c.GetMySqlDbSystem(ctx, *mySqlDbSystemOcid, nil)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting MySqlDbSystem database")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err

			}
		}
		mysqlDbSystem.Status.OsokStatus = util.UpdateOSOKStatusCondition(mysqlDbSystem.Status.OsokStatus,
			ociv1beta1.Active, v1.ConditionTrue, "",
			fmt.Sprintf("MySqlDbSystem %s is %s", *mySqlDbSystemInstance.DisplayName, mySqlDbSystemInstance.LifecycleState), c.Log)
		c.Log.InfoLog(fmt.Sprintf("MySqlDbSystem %s is %s", *mySqlDbSystemInstance.DisplayName, mySqlDbSystemInstance.LifecycleState))

	} else {
		// Bind CRD with an existing MySql instance
		mySqlDbSystemInstance, err = c.GetMySqlDbSystem(ctx, mysqlDbSystem.Spec.MySqlDbSystemId, nil)
		if err != nil {
			c.Log.ErrorLog(err, "Error while getting the MySqlDbSystem")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if isValidUpdate(*mysqlDbSystem, *mySqlDbSystemInstance) {
			if err = c.UpdateMySqlDbSystem(ctx, mysqlDbSystem); err != nil {
				c.Log.ErrorLog(err, "Error while updating MysqlDbSystem")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
			mysqlDbSystem.Status.OsokStatus = util.UpdateOSOKStatusCondition(mysqlDbSystem.Status.OsokStatus,
				ociv1beta1.Active, v1.ConditionTrue, "", "MysqlDbSystem update success", c.Log)
			c.Log.InfoLog(fmt.Sprintf("MySqlDbSystem %s is updated successfully", *mySqlDbSystemInstance.DisplayName))
		} else {
			mysqlDbSystem.Status.OsokStatus = util.UpdateOSOKStatusCondition(mysqlDbSystem.Status.OsokStatus,
				ociv1beta1.Active, v1.ConditionTrue, "", "MysqlDbSystem Bound success", c.Log)
			c.Log.InfoLog(fmt.Sprintf("MysqlDbSystem %s is bound successfully", *mySqlDbSystemInstance.DisplayName))

		}

	}

	mysqlDbSystem.Status.OsokStatus.Ocid = ociv1beta1.OCID(*mySqlDbSystemInstance.Id)
	if mysqlDbSystem.Status.OsokStatus.CreatedAt != nil {
		now := metav1.NewTime(time.Now())
		mysqlDbSystem.Status.OsokStatus.CreatedAt = &now
	}

	if mySqlDbSystemInstance.LifecycleState == "FAILED" {
		mysqlDbSystem.Status.OsokStatus = util.UpdateOSOKStatusCondition(mysqlDbSystem.Status.OsokStatus,
			ociv1beta1.Failed, v1.ConditionFalse, "",
			fmt.Sprintf("MySqlDbSystem %s creation Failed", *mySqlDbSystemInstance.DisplayName), c.Log)
		c.Log.InfoLog(fmt.Sprintf("MySqlDbSystem %s creation Failed", *mySqlDbSystemInstance.DisplayName))
	} else {
		_, err := c.addToSecret(ctx, mysqlDbSystem.Namespace, mysqlDbSystem.Name, *mySqlDbSystemInstance)
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				return servicemanager.OSOKResponse{IsSuccessful: true}, nil
			}
			c.Log.InfoLog(fmt.Sprintf("Secret creation failed"))
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
	}

	return servicemanager.OSOKResponse{IsSuccessful: true}, nil
}

func isValidUpdate(dbSystem ociv1beta1.MySqlDbSystem, mySqlDbInstance mysql.DbSystem) bool {

	definedTagUpdated := false
	if dbSystem.Spec.DefinedTags != nil {
		if defTag := *util.ConvertToOciDefinedTags(&dbSystem.Spec.DefinedTags); !reflect.DeepEqual(mySqlDbInstance.DefinedTags, defTag) {
			definedTagUpdated = true
		}
	}

	return dbSystem.Spec.DisplayName != "" && dbSystem.Spec.DisplayName != *mySqlDbInstance.DisplayName ||
		dbSystem.Spec.Description != "" && dbSystem.Spec.Description != *mySqlDbInstance.Description ||
		dbSystem.Spec.ConfigurationId.Id != "" && string(dbSystem.Spec.ConfigurationId.Id) != *mySqlDbInstance.ConfigurationId ||
		dbSystem.Spec.FreeFormTags != nil && !reflect.DeepEqual(dbSystem.Spec.FreeFormTags, mySqlDbInstance.FreeformTags) ||
		definedTagUpdated
}

func (c *DbSystemServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	return true, nil
}

func (c *DbSystemServiceManager) GetCrdStatus(obj runtime.Object) (*ociv1beta1.OSOKStatus, error) {
	resource, err := c.convert(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (c *DbSystemServiceManager) convert(obj runtime.Object) (*ociv1beta1.MySqlDbSystem, error) {
	copy, err := obj.(*ociv1beta1.MySqlDbSystem)
	if !err {
		return nil, fmt.Errorf("failed to convert the type assertion for MySqlDbSystem")
	}
	return copy, nil
}

func (c *DbSystemServiceManager) getDbSystemRetryPolicy(attempts uint) common.RetryPolicy {
	shouldRetry := func(response common.OCIOperationResponse) bool {
		if resp, ok := response.Response.(mysql.GetDbSystemResponse); ok {
			return resp.LifecycleState == "CREATING"
		}
		return true
	}
	nextDuration := func(response common.OCIOperationResponse) time.Duration {
		return time.Duration(1) * time.Minute
	}
	return common.NewRetryPolicy(attempts, shouldRetry, nextDuration)
}

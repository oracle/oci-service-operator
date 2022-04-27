/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package adb

import (
	"context"
	"errors"
	"fmt"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/util"
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/database"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

type AdbServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
}

func NewAdbServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger) *AdbServiceManager {
	return &AdbServiceManager{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
	}
}

func (c *AdbServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {

	autonomousDatabases, err := c.convert(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	var adbInstance *database.AutonomousDatabase
	if strings.TrimSpace(string(autonomousDatabases.Spec.AdbId)) == "" {

		c.Log.DebugLog("AutonomousDatabase Id is empty. Check if adb is already existing.")

		adbOcid, err := c.GetAdbOcid(ctx, *autonomousDatabases)
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if adbOcid == nil {
			// Geting the Admin password from the secret before creating ADB instance
			c.Log.DebugLog("Getting Admin password from Secret")
			pwdMap, err := c.CredentialClient.GetSecret(ctx, autonomousDatabases.Spec.AdminPassword.Secret.SecretName, req.Namespace)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting the admin password secret")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}

			pwd, ok := pwdMap["password"]
			if !ok {
				c.Log.ErrorLog(err, "password key in admin password secret is not found")
				return servicemanager.OSOKResponse{IsSuccessful: false}, errors.New("password key in admin password secret is not found")
			}

			resp, err := c.CreateAdb(ctx, *autonomousDatabases, string(pwd))
			if err != nil {
				autonomousDatabases.Status.OsokStatus = util.UpdateOSOKStatusCondition(autonomousDatabases.Status.OsokStatus,
					ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
				if err.(common.ServiceError).GetHTTPStatusCode() == 400 && err.(common.ServiceError).GetCode() == "InvalidParameter" {
					autonomousDatabases.Status.OsokStatus.Message = err.(common.ServiceError).GetCode()
					c.Log.ErrorLog(err, "Create AutonomousDatabase failed")
					return servicemanager.OSOKResponse{IsSuccessful: false}, nil
				}
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}

			c.Log.InfoLog(fmt.Sprintf("AutonomousDatabase %s is Provisioning", autonomousDatabases.Spec.DisplayName))
			autonomousDatabases.Status.OsokStatus = util.UpdateOSOKStatusCondition(autonomousDatabases.Status.OsokStatus,
				ociv1beta1.Provisioning, v1.ConditionTrue, "", "AutonomousDatabase Provisioning", c.Log)

			retryPolicy := c.getAdbRetryPolicy(9)
			adbInstance, err = c.GetAdb(ctx, ociv1beta1.OCID(*resp.Id), &retryPolicy)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting Autonomous database")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
		} else {
			c.Log.InfoLog(fmt.Sprintf("Getting Autonomous Database %s", *adbOcid))
			adbInstance, err = c.GetAdb(ctx, *adbOcid, nil)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting Autonomous database")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
		}
		autonomousDatabases.Status.OsokStatus = util.UpdateOSOKStatusCondition(autonomousDatabases.Status.OsokStatus,
			ociv1beta1.Active, v1.ConditionTrue, "",
			fmt.Sprintf("AutonomousDatabase %s is Active", *adbInstance.DisplayName), c.Log)
		c.Log.InfoLog(fmt.Sprintf("AutonomousDatabase %s is Active", *adbInstance.DisplayName))

		autonomousDatabases.Status.OsokStatus.Ocid = ociv1beta1.OCID(*adbInstance.Id)
		if autonomousDatabases.Status.OsokStatus.CreatedAt != nil {
			now := metav1.NewTime(time.Now())
			autonomousDatabases.Status.OsokStatus.CreatedAt = &now
		}

	} else {
		// Bind CRD with an existing ADB.
		adbInstance, err = c.GetAdb(ctx, autonomousDatabases.Spec.AdbId, nil)
		if err != nil {
			c.Log.ErrorLog(err, "Error while getting Autonomous database")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if isValidUpdate(*autonomousDatabases, *adbInstance) {
			if err = c.UpdateAdb(ctx, autonomousDatabases); err != nil {
				c.Log.ErrorLog(err, "Error while updating Autonomous database")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
			autonomousDatabases.Status.OsokStatus = util.UpdateOSOKStatusCondition(autonomousDatabases.Status.OsokStatus,
				ociv1beta1.Active, v1.ConditionTrue, "", "AutonomousDatabase Update success", c.Log)
			c.Log.InfoLog(fmt.Sprintf("AutonomousDatabase %s is updated successfully", *adbInstance.DisplayName))
		} else {
			autonomousDatabases.Status.OsokStatus = util.UpdateOSOKStatusCondition(autonomousDatabases.Status.OsokStatus,
				ociv1beta1.Active, v1.ConditionTrue, "", "AutonomousDatabase Bound success", c.Log)
			autonomousDatabases.Status.OsokStatus.Ocid = ociv1beta1.OCID(*adbInstance.Id)
			now := metav1.NewTime(time.Now())
			autonomousDatabases.Status.OsokStatus.CreatedAt = &now

			c.Log.InfoLog(fmt.Sprintf("AutonomousDatabase %s is bounded successfully", *adbInstance.DisplayName))
		}
		autonomousDatabases.Status.OsokStatus.Ocid = ociv1beta1.OCID(*adbInstance.Id)
		if autonomousDatabases.Status.OsokStatus.CreatedAt != nil {
			now := metav1.NewTime(time.Now())
			autonomousDatabases.Status.OsokStatus.CreatedAt = &now
		}
	}

	if autonomousDatabases.Spec.Wallet.WalletPassword.Secret.SecretName != "" {
		c.Log.InfoLog(fmt.Sprintf("Wallet Password Secret Name provided for %s Autonomous Database", autonomousDatabases.Spec.DisplayName))
		response, err := c.GenerateWallet(ctx, *adbInstance.Id, *adbInstance.DisplayName, autonomousDatabases.Spec.Wallet.WalletPassword.Secret.SecretName,
			autonomousDatabases.Namespace, autonomousDatabases.Spec.Wallet.WalletName, autonomousDatabases.Name)
		return servicemanager.OSOKResponse{IsSuccessful: response}, err
	} else {
		c.Log.InfoLog(fmt.Sprintf("Wallet Password Secret Name is empty. Not creating wallet for %s Autonomous Database",
			autonomousDatabases.Spec.DisplayName))
	}

	return servicemanager.OSOKResponse{IsSuccessful: true}, nil
}

func isValidUpdate(autonomousDatabases ociv1beta1.AutonomousDatabases, adbInstance database.AutonomousDatabase) bool {

	definedTagUpdated := false
	if autonomousDatabases.Spec.DefinedTags != nil {
		if defTag := *util.ConvertToOciDefinedTags(&autonomousDatabases.Spec.DefinedTags); !reflect.DeepEqual(adbInstance.DefinedTags, defTag) {
			definedTagUpdated = true
		}
	}

	return autonomousDatabases.Spec.DisplayName != "" && autonomousDatabases.Spec.DisplayName != *adbInstance.DisplayName ||
		autonomousDatabases.Spec.DbName != "" && autonomousDatabases.Spec.DbName != *adbInstance.DbName ||
		autonomousDatabases.Spec.CpuCoreCount != 0 && autonomousDatabases.Spec.CpuCoreCount != *adbInstance.CpuCoreCount ||
		autonomousDatabases.Spec.DataStorageSizeInTBs != 0 && autonomousDatabases.Spec.DataStorageSizeInTBs != *adbInstance.DataStorageSizeInTBs ||
		autonomousDatabases.Spec.DbWorkload != "" && autonomousDatabases.Spec.DbWorkload != string(adbInstance.DbWorkload) ||
		autonomousDatabases.Spec.DbVersion != "" && autonomousDatabases.Spec.DbVersion != *adbInstance.DbVersion ||
		autonomousDatabases.Spec.IsAutoScalingEnabled != false && autonomousDatabases.Spec.IsAutoScalingEnabled != *adbInstance.IsAutoScalingEnabled ||
		autonomousDatabases.Spec.IsFreeTier != false && autonomousDatabases.Spec.IsFreeTier != *adbInstance.IsFreeTier ||
		autonomousDatabases.Spec.LicenseModel != "" && autonomousDatabases.Spec.LicenseModel != string(adbInstance.LicenseModel) ||
		autonomousDatabases.Spec.FreeFormTags != nil && !reflect.DeepEqual(autonomousDatabases.Spec.FreeFormTags, adbInstance.FreeformTags) ||
		definedTagUpdated
}

func (c *AdbServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	return true, nil
}

func (c *AdbServiceManager) GetCrdStatus(obj runtime.Object) (*ociv1beta1.OSOKStatus, error) {

	resource, err := c.convert(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (c *AdbServiceManager) convert(obj runtime.Object) (*ociv1beta1.AutonomousDatabases, error) {
	copy, err := obj.(*ociv1beta1.AutonomousDatabases)
	if !err {
		return nil, fmt.Errorf("failed to convert the type assertion for Autonomous Databases")
	}
	return copy, nil
}

func (c *AdbServiceManager) getAdbRetryPolicy(attempts uint) common.RetryPolicy {
	shouldRetry := func(response common.OCIOperationResponse) bool {
		if resp, ok := response.Response.(database.GetAutonomousDatabaseResponse); ok {
			return resp.LifecycleState == "PROVISIONING"
		}
		return true
	}
	nextDuration := func(response common.OCIOperationResponse) time.Duration {
		return time.Duration(math.Pow(float64(2), float64(response.AttemptNumber-1))) * time.Second
	}
	return common.NewRetryPolicy(attempts, shouldRetry, nextDuration)
}

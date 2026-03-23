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
	databasev1beta1 "github.com/oracle/oci-service-operator/api/database/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
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
	ociClient        DatabaseClientInterface
}

func NewAdbServiceManagerWithDeps(deps servicemanager.RuntimeDeps) *AdbServiceManager {
	return &AdbServiceManager{
		Provider:         deps.Provider,
		CredentialClient: deps.CredentialClient,
		Scheme:           deps.Scheme,
		Log:              deps.Log,
	}
}

func NewAdbServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger) *AdbServiceManager {
	return NewAdbServiceManagerWithDeps(servicemanager.RuntimeDeps{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
	})
}

func (c *AdbServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {

	autonomousDatabases, err := c.convert(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	var adbInstance *database.AutonomousDatabase
	secretReferenceApplied := false
	if strings.TrimSpace(string(autonomousDatabases.Spec.AdbId)) == "" {

		c.Log.DebugLog("AutonomousDatabase Id is empty. Check if adb is already existing.")

		adbOcid, err := c.GetAdbOcid(ctx, *autonomousDatabases)
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if adbOcid == nil {
			adminPwd := ""
			if strings.TrimSpace(autonomousDatabases.Spec.SecretId) == "" {
				// Get the admin password from the secret before creating the ADB instance when OCI Vault is not used.
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
				adminPwd = string(pwd)
			}

			resp, err := c.CreateAdb(ctx, *autonomousDatabases, adminPwd)
			if err != nil {
				autonomousDatabases.Status.OsokStatus = util.UpdateOSOKStatusCondition(autonomousDatabases.Status.OsokStatus,
					shared.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
				if err.(common.ServiceError).GetHTTPStatusCode() == 400 && err.(common.ServiceError).GetCode() == "InvalidParameter" {
					autonomousDatabases.Status.OsokStatus.Message = err.(common.ServiceError).GetCode()
					c.Log.ErrorLog(err, "Create AutonomousDatabase failed")
					return servicemanager.OSOKResponse{IsSuccessful: false}, nil
				}
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}

			c.Log.InfoLog(fmt.Sprintf("AutonomousDatabase %s is Provisioning", autonomousDatabases.Spec.DisplayName))
			autonomousDatabases.Status.OsokStatus = util.UpdateOSOKStatusCondition(autonomousDatabases.Status.OsokStatus,
				shared.Provisioning, v1.ConditionTrue, "", "AutonomousDatabase Provisioning", c.Log)

			retryPolicy := c.getAdbRetryPolicy(9)
			adbInstance, err = c.GetAdb(ctx, shared.OCID(*resp.Id), &retryPolicy)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting Autonomous database")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
			secretReferenceApplied = true
		} else {
			c.Log.InfoLog(fmt.Sprintf("Getting Autonomous Database %s", *adbOcid))
			adbInstance, err = c.GetAdb(ctx, *adbOcid, nil)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting Autonomous database")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
		}
		autonomousDatabases.Status.OsokStatus = util.UpdateOSOKStatusCondition(autonomousDatabases.Status.OsokStatus,
			shared.Active, v1.ConditionTrue, "",
			fmt.Sprintf("AutonomousDatabase %s is Active", *adbInstance.DisplayName), c.Log)
		c.Log.InfoLog(fmt.Sprintf("AutonomousDatabase %s is Active", *adbInstance.DisplayName))

		autonomousDatabases.Status.OsokStatus.Ocid = shared.OCID(*adbInstance.Id)
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

		updateNeeded, err := needsAutonomousDatabaseUpdate(*autonomousDatabases, *adbInstance)
		if err != nil {
			c.Log.ErrorLog(err, "Error while determining Autonomous database update intent")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if updateNeeded {
			if err = c.UpdateAdb(ctx, autonomousDatabases); err != nil {
				c.Log.ErrorLog(err, "Error while updating Autonomous database")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
			autonomousDatabases.Status.OsokStatus = util.UpdateOSOKStatusCondition(autonomousDatabases.Status.OsokStatus,
				shared.Active, v1.ConditionTrue, "", "AutonomousDatabase Update success", c.Log)
			c.Log.InfoLog(fmt.Sprintf("AutonomousDatabase %s is updated successfully", *adbInstance.DisplayName))
		} else {
			autonomousDatabases.Status.OsokStatus = util.UpdateOSOKStatusCondition(autonomousDatabases.Status.OsokStatus,
				shared.Active, v1.ConditionTrue, "", "AutonomousDatabase Bound success", c.Log)
			autonomousDatabases.Status.OsokStatus.Ocid = shared.OCID(*adbInstance.Id)
			now := metav1.NewTime(time.Now())
			autonomousDatabases.Status.OsokStatus.CreatedAt = &now

			c.Log.InfoLog(fmt.Sprintf("AutonomousDatabase %s is bounded successfully", *adbInstance.DisplayName))
		}
		autonomousDatabases.Status.OsokStatus.Ocid = shared.OCID(*adbInstance.Id)
		if autonomousDatabases.Status.OsokStatus.CreatedAt != nil {
			now := metav1.NewTime(time.Now())
			autonomousDatabases.Status.OsokStatus.CreatedAt = &now
		}
		secretReferenceApplied = true
	}

	if secretReferenceApplied {
		recordLastAppliedSecretReferenceStatus(autonomousDatabases)
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

func needsAutonomousDatabaseUpdate(autonomousDatabases databasev1beta1.AutonomousDatabases, adbInstance database.AutonomousDatabase) (bool, error) {

	definedTagUpdated := false
	if autonomousDatabases.Spec.DefinedTags != nil {
		if defTag := *util.ConvertToOciDefinedTags(&autonomousDatabases.Spec.DefinedTags); !reflect.DeepEqual(adbInstance.DefinedTags, defTag) {
			definedTagUpdated = true
		}
	}

	if autonomousDatabases.Spec.DisplayName != "" && autonomousDatabases.Spec.DisplayName != *adbInstance.DisplayName ||
		autonomousDatabases.Spec.DbName != "" && autonomousDatabases.Spec.DbName != *adbInstance.DbName ||
		autonomousDatabases.Spec.CpuCoreCount != 0 && autonomousDatabases.Spec.CpuCoreCount != *adbInstance.CpuCoreCount ||
		autonomousDatabases.Spec.DataStorageSizeInTBs != 0 && autonomousDatabases.Spec.DataStorageSizeInTBs != *adbInstance.DataStorageSizeInTBs ||
		autonomousDatabases.Spec.DbWorkload != "" && autonomousDatabases.Spec.DbWorkload != string(adbInstance.DbWorkload) ||
		autonomousDatabases.Spec.DbVersion != "" && autonomousDatabases.Spec.DbVersion != *adbInstance.DbVersion ||
		autonomousDatabases.Spec.IsAutoScalingEnabled != false && autonomousDatabases.Spec.IsAutoScalingEnabled != *adbInstance.IsAutoScalingEnabled ||
		autonomousDatabases.Spec.IsFreeTier != false && autonomousDatabases.Spec.IsFreeTier != *adbInstance.IsFreeTier ||
		autonomousDatabases.Spec.LicenseModel != "" && autonomousDatabases.Spec.LicenseModel != string(adbInstance.LicenseModel) ||
		autonomousDatabases.Spec.FreeFormTags != nil && !reflect.DeepEqual(autonomousDatabases.Spec.FreeFormTags, adbInstance.FreeformTags) ||
		definedTagUpdated {
		return true, nil
	}

	if secretReferenceUpdateNeeded(autonomousDatabases.Spec, autonomousDatabases.Status) {
		return true, nil
	}

	return additionalAutonomousDatabaseUpdateNeededForSpec(autonomousDatabases.Spec, adbInstance)
}

// OCI GetAutonomousDatabase does not return vault secret references, so status tracks the last applied value.
func desiredAutonomousDatabaseSecretReference(spec databasev1beta1.AutonomousDatabasesSpec) (string, int, bool) {
	secretID := strings.TrimSpace(spec.SecretId)
	if secretID == "" {
		return "", 0, false
	}

	return secretID, spec.SecretVersionNumber, true
}

func secretReferenceUpdateNeeded(spec databasev1beta1.AutonomousDatabasesSpec, status databasev1beta1.AutonomousDatabasesStatus) bool {
	secretID, secretVersionNumber, ok := desiredAutonomousDatabaseSecretReference(spec)
	if !ok {
		return false
	}

	return secretID != status.LastAppliedSecretId || secretVersionNumber != status.LastAppliedSecretVersionNumber
}

func recordLastAppliedSecretReferenceStatus(autonomousDatabases *databasev1beta1.AutonomousDatabases) {
	secretID, secretVersionNumber, ok := desiredAutonomousDatabaseSecretReference(autonomousDatabases.Spec)
	if !ok {
		return
	}

	autonomousDatabases.Status.LastAppliedSecretId = secretID
	autonomousDatabases.Status.LastAppliedSecretVersionNumber = secretVersionNumber
}

func (c *AdbServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	return true, nil
}

func (c *AdbServiceManager) GetCrdStatus(obj runtime.Object) (*shared.OSOKStatus, error) {

	resource, err := c.convert(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (c *AdbServiceManager) convert(obj runtime.Object) (*databasev1beta1.AutonomousDatabases, error) {
	copy, err := obj.(*databasev1beta1.AutonomousDatabases)
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

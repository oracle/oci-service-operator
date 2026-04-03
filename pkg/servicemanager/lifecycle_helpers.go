/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package servicemanager

import (
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ResolveResourceID(statusID, specID shared.OCID) (shared.OCID, error) {
	if statusID != "" {
		return statusID, nil
	}
	if specID != "" {
		return specID, nil
	}
	return "", fmt.Errorf("resource ocid is empty")
}

func SetCreatedAtIfUnset(status *shared.OSOKStatus) {
	if status == nil || status.CreatedAt != nil {
		return
	}
	now := metav1.NewTime(metav1.Now().Time)
	status.CreatedAt = &now
}

func IsNotFoundServiceError(err error) bool {
	serviceErr, ok := err.(common.ServiceError)
	return ok && serviceErr.GetHTTPStatusCode() == 404
}

func IsNotFoundErrorString(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "404") || strings.Contains(msg, "notfound") || strings.Contains(msg, "not found")
}

func IsSecretNotFoundError(err error) bool {
	return k8serrors.IsNotFound(err) || IsNotFoundErrorString(err)
}

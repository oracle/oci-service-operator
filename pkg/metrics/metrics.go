/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package metrics

import (
	"fmt"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"strconv"
	"time"
)

const defaultMetricsNamespace = "oci"

const (
	ReconcileSuccess = "oci_service_operator_reconcile_success"
	ReconcileFault   = "oci_service_operator_reconcile_fault"
	CRDeleteSuccess  = "oci_service_operator_cr_delete_success"
	CRDeleteFault    = "oci_service_operator_cr_delete_fault"
	CRSuccess        = "oci_service_operator_cr_success"
	CRFault          = "oci_service_operator_cr_fault"
	CRCount          = "oci_service_operator_cr_count"
	SecretCount      = "oci_service_operator_secret_count"
	CRLatency        = "oci_service_operator_cr_latency"
)

var (
	reconcileSuccess = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: ReconcileSuccess,
		Help: "Total Number of Reconcile operation successful",
	}, []string{"component", "resourcename", "namespace", "state", "message"})

	reconcileFault = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: ReconcileFault,
		Help: "Total Number of Reconcile operation failed",
	}, []string{"component", "resourcename", "namespace", "state", "message"})

	crDeleteSuccessCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: CRDeleteSuccess,
		Help: "Total Number of CR Delete with Success Status",
	}, []string{"component", "resourcename", "namespace", "state", "message"})

	crDeleteFaultCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: CRDeleteFault,
		Help: "Total Number of CR Delete with Fault Status",
	}, []string{"component", "resourcename", "namespace", "state", "message"})

	crSuccessCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: CRSuccess,
		Help: "Total Number of CR with Success Status",
	}, []string{"component", "resourcename", "namespace", "state", "message"})

	crFaultCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: CRFault,
		Help: "Total Number of CR with Fault Status",
	}, []string{"component", "resourcename", "namespace", "state", "message"})

	crCountCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: CRCount,
		Help: "Total Number of CR managed by the operators",
	}, []string{"component", "resourcename", "namespace", "state", "message"})

	secretCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: SecretCount,
		Help: "Total Number of secret managed by the operators",
	}, []string{"component", "resourcename", "namespace", "state", "message"})
)

type Metrics struct {
	Name        string
	ServiceName string
	Logger      loggerutil.OSOKLogger
}

func Init(serviceName string, log loggerutil.OSOKLogger) *Metrics {
	metrics.Registry.MustRegister(
		reconcileSuccess,
		reconcileFault,
		crCountCounter,
		crSuccessCounter,
		crFaultCounter,
		crDeleteFaultCounter,
		crDeleteSuccessCounter,
		secretCounter,
	)
	return &Metrics{
		Name:        defaultMetricsNamespace,
		ServiceName: serviceName,
		Logger:      log,
	}
}

func (m *Metrics) AddReconcileSuccessMetrics(component string, msg string, resourceName string, namespace string) {
	m.Logger.FixedLogs["name"] = resourceName
	m.Logger.FixedLogs["namespace"] = namespace
	m.Logger.InfoLog(fmt.Sprintf("Recording the reconcile success metrics for %s", resourceName))
	reconcileSuccess.WithLabelValues(component, resourceName, namespace, "Success", msg).Inc()
}

func (m *Metrics) AddReconcileFaultMetrics(component string, msg string, resourceName string, namespace string) {
	m.Logger.FixedLogs["name"] = resourceName
	m.Logger.FixedLogs["namespace"] = namespace
	m.Logger.InfoLog(fmt.Sprintf("Recording the reconcile fault metrics for %s", resourceName))
	reconcileFault.WithLabelValues(component, resourceName, namespace, "Fault", msg).Inc()
}

func (m *Metrics) AddCRSuccessMetrics(component string, msg string, resourceName string, namespace string) {
	m.Logger.FixedLogs["name"] = resourceName
	m.Logger.FixedLogs["namespace"] = namespace
	m.Logger.InfoLog(fmt.Sprintf("Recording the cr success metrics for %s", resourceName))
	crSuccessCounter.WithLabelValues(component, resourceName, namespace, "Success", msg).Inc()
}

func (m *Metrics) AddCRFaultMetrics(component string, msg string, resourceName string, namespace string) {
	m.Logger.FixedLogs["name"] = resourceName
	m.Logger.FixedLogs["namespace"] = namespace
	m.Logger.InfoLog(fmt.Sprintf("Recording the cr fault metrics for %s", resourceName))
	crFaultCounter.WithLabelValues(component, resourceName, namespace, "Fault", msg).Inc()
}

func (m *Metrics) AddCRDeleteFaultMetrics(component string, msg string, resourceName string, namespace string) {
	m.Logger.FixedLogs["name"] = resourceName
	m.Logger.FixedLogs["namespace"] = namespace
	m.Logger.InfoLog(fmt.Sprintf("Recording the cr delete fault metrics for %s", resourceName))
	crDeleteFaultCounter.WithLabelValues(component, resourceName, namespace, "Fault", msg).Inc()
}

func (m *Metrics) AddCRDeleteSuccessMetrics(component string, msg string, resourceName string, namespace string) {
	m.Logger.FixedLogs["name"] = resourceName
	m.Logger.FixedLogs["namespace"] = namespace
	m.Logger.InfoLog(fmt.Sprintf("Recording the cr delete success metrics for %s", resourceName))
	crDeleteSuccessCounter.WithLabelValues(component, resourceName, namespace, "Success", msg).Inc()
}

func (m *Metrics) AddCRCountMetrics(component string, msg string, resourceName string, namespace string) {
	m.Logger.FixedLogs["name"] = resourceName
	m.Logger.FixedLogs["namespace"] = namespace
	m.Logger.InfoLog(fmt.Sprintf("Recording the cr count metrics for %s", resourceName))
	crCountCounter.WithLabelValues(component, resourceName, namespace, "Success", msg).Inc()
}

func (m *Metrics) AddSecretCountMetrics(component string, msg string, resourceName string, namespace string) {
	m.Logger.FixedLogs["name"] = resourceName
	m.Logger.FixedLogs["namespace"] = namespace
	m.Logger.InfoLog(fmt.Sprintf("Recording the secret count metrics for %s", resourceName))
	secretCounter.WithLabelValues(component, resourceName, namespace, "Success", msg).Inc()
}

func sendPresentEpcoh() string {
	return strconv.FormatInt(time.Now().UnixNano()/1000000, 10)
}

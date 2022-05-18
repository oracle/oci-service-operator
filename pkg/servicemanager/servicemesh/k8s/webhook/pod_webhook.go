/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"strings"

	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/k8s/inject"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	customCache "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/k8s/cache"
	podUtil "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/pod"
	virtualDeploymentBindingUtil "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/virtualdeploymentbinding"
)

type PodMutatorHandler struct {
	directReader client.Reader // direct reader reads data from api server
	decoder      *admission.Decoder
	Caches       customCache.CacheMapClient
	configMap    string
}

func NewDefaultPodMutatorHandler(
	k8sDirectReader client.Reader,
	caches customCache.CacheMapClient,
	configMap string) *PodMutatorHandler {
	return &PodMutatorHandler{
		directReader: k8sDirectReader,
		Caches:       caches,
		configMap:    configMap}
}

/**
1. Match the namespace label and check if the label is set
2. Match the labels of pod if the injection is enabled
3. Match VDB match criteria
4. If 1,2,3 are satisfied mutate pod by injecting envoy and init container
	a. Optimization - Using cache to reduce calls to api server
5. Add annotation of Virtual Deployment Binding and Virtual Deployment
*/

//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps/status,verbs=get;update;patch

func (p *PodMutatorHandler) Handle(ctx context.Context, req admission.Request) admission.Response {
	var logger = loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("podMutator").WithName(req.Namespace)}
	fixedLogMap := make(map[string]string)
	fixedLogMap["name"] = req.Name
	fixedLogMap["namespace"] = req.Namespace
	ctx = context.WithValue(ctx, loggerutil.FixedLogMapCtxKey, fixedLogMap)
	pod := &corev1.Pod{}
	err := p.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	if pod.Namespace == "" {
		pod.Namespace = req.Namespace
	}

	// Fetch Namespace from Cache
	namespace, keyerr := p.Caches.GetNamespaceByKey(req.Namespace)
	if keyerr != nil {
		logger.ErrorLogWithFixedMessage(ctx, err, "Failed to fetch namespace")
		return admission.Errored(http.StatusInternalServerError, keyerr)
	}

	namespaceInjectionLabel, namespaceInjectionLabelExists := labels.Set(namespace.Labels)[commons.ProxyInjectionLabel]
	podInjectionLabel, podInjectionLabelExists := labels.Set(pod.Labels)[commons.ProxyInjectionLabel]
	if !podInjectionLabelExists {
		podInjectionLabel = ""
	}

	if !namespaceInjectionLabelExists || !podUtil.IsInjectionLabelEnabled(namespaceInjectionLabel, podInjectionLabel) {
		return admission.Allowed("Pod did not match criteria for side car injection")
	}

	virtualDeploymentBindingList, err := virtualDeploymentBindingUtil.ListVDB(ctx, p.directReader)
	if err != nil {
		logger.ErrorLogWithFixedMessage(ctx, err, "Error in listing virtualDeploymentBinding")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// Match VDB target ref to pod label
	matchingVDB := podUtil.GetVDBForPod(ctx, p.directReader, pod, virtualDeploymentBindingList)

	if matchingVDB == nil {
		logger.InfoLogWithFixedMessage(ctx, "Pod did not match Virtual Deployment Binding for side car injection")
		return admission.Allowed("Pod did not match Virtual Deployment Binding for side car injection")
	}
	fixedLogMap["virtualDeploymentBindingName"] = matchingVDB.Name
	ctx = context.WithValue(ctx, loggerutil.FixedLogMapCtxKey, fixedLogMap)

	if !virtualDeploymentBindingUtil.IsVirtualDeploymentBindingActive(matchingVDB) {
		logger.InfoLogWithFixedMessage(ctx, "Virtual Deployment Binding with name = %s namespace = %s is not active", matchingVDB.Name, matchingVDB.Namespace)
		return admission.Allowed("Virtual Deployment Binding is not active yet")
	}

	fixedLogMap["virtualDeploymentId"] = string(matchingVDB.Status.VirtualDeploymentId)
	ctx = context.WithValue(ctx, loggerutil.FixedLogMapCtxKey, fixedLogMap)
	// Fetch latest global Config to inject init and side car containers
	configMap, keyerr := p.Caches.GetConfigMapByKey(p.configMap)
	if keyerr != nil {
		logger.ErrorLogWithFixedMessage(ctx, keyerr, "Failed to fetch configmap")
		return admission.Errored(http.StatusInternalServerError, keyerr)
	}

	mutateAllHttpProbes(pod, logger)

	mutateHandler := inject.NewMutateHandler(configMap, *matchingVDB)
	err = mutateHandler.Inject(pod)
	if err != nil {
		logger.ErrorLogWithFixedMessage(ctx, err, "Failed to mutate pod")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// Add VDB annotation for audit
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}
	pod.Annotations[commons.VirtualDeploymentBindingAnnotation] = client.ObjectKeyFromObject(matchingVDB).String()
	pod.Annotations[commons.VirtualDeploymentAnnotation] = string(matchingVDB.Status.VirtualDeploymentId)

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		logger.ErrorLogWithFixedMessage(ctx, err, "Failed to serialize pod operations")
		return admission.Errored(http.StatusInternalServerError, err)

	}
	logger.InfoLogWithFixedMessage(ctx, "Side car Injected successfully")
	// Create Patch from request and mutated pod
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

func (p *PodMutatorHandler) InjectDecoder(d *admission.Decoder) error {
	p.decoder = d
	return nil
}

// mutateAllHttpProbes rotates on all containers and mutates all HttpProbes
// s.a: Startup, Readiness and Liveness by using method 'mutateProbe'
func mutateAllHttpProbes(pod *corev1.Pod, logger loggerutil.OSOKLogger) {
	logger.InfoLog("Starting to mutate all HTTP Probes...")
	for _, container := range pod.Spec.Containers {
		if container.Name != commons.ProxyContainerName {
			mutatedStartupProbe := mutateProbe(container.StartupProbe, "Startup", container.Ports, logger)
			mutatedReadinessProbe := mutateProbe(container.ReadinessProbe, "Readiness", container.Ports, logger)
			mutatedLivenessProbe := mutateProbe(container.LivenessProbe, "Liveness", container.Ports, logger)
			if mutatedStartupProbe || mutatedReadinessProbe || mutatedLivenessProbe {
				logger.InfoLog("Successfully mutated all Http Probes for '" + container.Name + "' container.")
			} else {
				logger.InfoLog("No Http Probes were found for '" + container.Name + "' container.")
			}
		}
	}
}

// mutateProbe preserves the http components (scheme, host, port, path) and appends
// them to the http headers, and points the probe to the health proxy endpoint
func mutateProbe(probe *corev1.Probe, probeName string, containerPorts []corev1.ContainerPort, logger loggerutil.OSOKLogger) bool {
	logger.InfoLog("Mutating " + probeName + " probe...")
	if probe == nil || probe.HTTPGet == nil {
		logger.InfoLog("No HTTPGet " + probeName + " probe was found.")
		return false
	}

	host := probe.HTTPGet.Host
	if host == "" {
		host = commons.LocalHost
	}

	probe.HTTPGet.HTTPHeaders = append(probe.HTTPGet.HTTPHeaders,
		corev1.HTTPHeader{
			Name:  string(commons.MeshUserScheme),
			Value: strings.ToLower(string(probe.HTTPGet.Scheme)),
		},
		corev1.HTTPHeader{
			Name:  string(commons.MeshUserHost),
			Value: host,
		},
		corev1.HTTPHeader{
			Name:  string(commons.MeshUserPort),
			Value: toPortString(probe.HTTPGet.Port, containerPorts),
		},
		corev1.HTTPHeader{
			Name:  string(commons.MeshUserPath),
			Value: probe.HTTPGet.Path,
		})

	probe.HTTPGet.Scheme = corev1.URISchemeHTTP
	probe.HTTPGet.Port = intstr.FromInt(int(commons.LivenessProbeEndpointPort))
	probe.HTTPGet.Path = commons.HealthProxyEndpointPath
	return true
}

func toPortString(port intstr.IntOrString, containerPorts []corev1.ContainerPort) string {
	if port.Type == intstr.String {
		for _, containerPort := range containerPorts {
			if containerPort.Name == port.String() {
				return fmt.Sprint(containerPort.ContainerPort)
			}
		}
	}
	return port.String()
}

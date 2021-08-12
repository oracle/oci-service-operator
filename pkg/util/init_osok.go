/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package util

import (
	"context"
	"encoding/json"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

func InitOSOK(config *rest.Config, log loggerutil.OSOKLogger) {
	files, err := ioutil.ReadDir("/")
	if err != nil {
		log.ErrorLog(err, "failed to get files in root directory")
		os.Exit(1)
	}

	log.InfoLog("Starting OSOK initialization. Will install CRDs and Webhooks needed for OSOK operator to run")
	// Prepare a RESTMapper to find GVR
	dc, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		log.ErrorLog(err, "failed to create discovery client")
		os.Exit(1)
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))

	// Prepare the dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.ErrorLog(err, "failed to dynamic client")
		os.Exit(1)
	}

	// Loop through all files in root directory
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		// Process only resources present in yaml files
		if filepath.Ext(file.Name()) != ".yaml" {
			continue
		}

		data, err := ioutil.ReadFile("/" + file.Name())
		if err != nil {
			log.ErrorLog(err, "failed reading data from file")
			os.Exit(1)
		}
		log.FixedLogs["fileName"] = file.Name()
		log.InfoLog("Installing resource present in file")
		err = installResource(context.TODO(), data, mapper, dynamicClient)
		if err != nil {
			log.ErrorLog(err, "error in installing resource")
		}

	}
}

// installResource loads the content of a yaml file into a unstructured.Unstructured object
// which parses all the TypeMeta fields. The resource thus loaded is installed into the cluster
// using a Dynamic Client
func installResource(ctx context.Context, data []byte, mapper *restmapper.DeferredDiscoveryRESTMapper, dyn dynamic.Interface) error {

	// Decode YAML manifest into unstructured.Unstructured
	decUnstructured := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	obj := &unstructured.Unstructured{}
	_, gvk, err := decUnstructured.Decode(data, nil, obj)
	if err != nil {
		return errors.Wrap(err, "failed to decode YAML manifest")
	}

	// Find GVR
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return errors.Wrap(err, "failed to get resource mapping from GVR")
	}

	// Obtain REST interface for the GVR
	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		// namespaced resources should specify the namespace
		dr = dyn.Resource(mapping.Resource).Namespace(obj.GetNamespace())
	} else {
		// for cluster-wide resources
		dr = dyn.Resource(mapping.Resource)
	}

	// Marshal object into JSON
	data, err = json.Marshal(obj)
	if err != nil {
		return errors.Wrap(err, "failed to marshal resource into json")
	}

	// Create or Update the object
	// FieldManager specifies the field owner ID.
	_, err = dr.Get(ctx, obj.GetName(), metav1.GetOptions{})
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get resource by name")
		}
		_, err = dr.Create(ctx, obj, metav1.CreateOptions{
			FieldManager: "osok",
		})
		// note: if err is nil errors.Wrap will return nil
		return errors.Wrap(err, "failed to create resource")
	}
	_, err = dr.Patch(ctx, obj.GetName(), types.StrategicMergePatchType, data, metav1.PatchOptions{
		FieldManager: "osok",
	})

	// note: if err is nil errors.Wrap will return nil
	return errors.Wrap(err, "failed to patch existing resource")
}

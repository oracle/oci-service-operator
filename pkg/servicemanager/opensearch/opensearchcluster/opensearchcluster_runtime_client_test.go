package opensearchcluster

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	opensearchsdk "github.com/oracle/oci-go-sdk/v65/opensearch"
	opensearchv1beta1 "github.com/oracle/oci-service-operator/api/opensearch/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateOpensearchClusterRequestUsesSDKGBKeys(t *testing.T) {
	t.Parallel()

	resource := &opensearchv1beta1.OpensearchCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "opensearchcluster-sample",
			Namespace: "default",
		},
		Spec: opensearchv1beta1.OpensearchClusterSpec{
			CompartmentId:                  "ocid1.compartment.oc1..example",
			DisplayName:                    "opensearchcluster-sample",
			SoftwareVersion:                "2.11.0",
			MasterNodeCount:                1,
			MasterNodeHostType:             "FLEX",
			MasterNodeHostOcpuCount:        1,
			MasterNodeHostMemoryGB:         16,
			DataNodeCount:                  1,
			DataNodeHostType:               "FLEX",
			DataNodeHostOcpuCount:          1,
			DataNodeHostMemoryGB:           16,
			DataNodeStorageGB:              50,
			OpendashboardNodeCount:         1,
			OpendashboardNodeHostOcpuCount: 1,
			OpendashboardNodeHostMemoryGB:  8,
			VcnId:                          "ocid1.vcn.oc1..example",
			SubnetId:                       "ocid1.subnet.oc1..example",
			VcnCompartmentId:               "ocid1.compartment.oc1..example",
			SubnetCompartmentId:            "ocid1.compartment.oc1..example",
			SecurityMode:                   "DISABLED",
		},
	}

	hooks := newOpensearchClusterDefaultRuntimeHooks(opensearchsdk.OpensearchClusterClient{})
	applyOpensearchClusterRuntimeHooks(&OpensearchClusterServiceManager{}, &hooks)
	if hooks.BuildCreateBody == nil {
		t.Fatal("hooks.BuildCreateBody = nil, want opensearch create builder")
	}

	createBody, err := hooks.BuildCreateBody(context.Background(), resource, resource.Namespace)
	if err != nil {
		t.Fatalf("hooks.BuildCreateBody() error = %v", err)
	}
	details, ok := createBody.(opensearchsdk.CreateOpensearchClusterDetails)
	if !ok {
		t.Fatalf("hooks.BuildCreateBody() body type = %T, want opensearch.CreateOpensearchClusterDetails", createBody)
	}

	request := opensearchsdk.CreateOpensearchClusterRequest{
		CreateOpensearchClusterDetails: details,
	}
	httpRequest, err := request.HTTPRequest(http.MethodPost, "/opensearchClusters", nil, nil)
	if err != nil {
		t.Fatalf("HTTPRequest() error = %v", err)
	}

	body, err := io.ReadAll(httpRequest.Body)
	if err != nil {
		t.Fatalf("ReadAll(request.Body) error = %v", err)
	}
	got := string(body)

	wantKeys := []string{
		`"masterNodeHostMemoryGB":16`,
		`"dataNodeHostMemoryGB":16`,
		`"dataNodeStorageGB":50`,
		`"opendashboardNodeHostMemoryGB":8`,
	}
	for _, want := range wantKeys {
		if !strings.Contains(got, want) {
			t.Fatalf("request body %s does not contain %s", got, want)
		}
	}

	unexpectedKeys := []string{
		`masterNodeHostMemoryGb`,
		`dataNodeHostMemoryGb`,
		`dataNodeStorageGb`,
		`opendashboardNodeHostMemoryGb`,
	}
	for _, unexpected := range unexpectedKeys {
		if strings.Contains(got, unexpected) {
			t.Fatalf("request body unexpectedly contains %s: %s", unexpected, got)
		}
	}
}

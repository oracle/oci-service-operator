package main

import (
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	keymanagementv1beta1 "github.com/oracle/oci-service-operator/api/keymanagement/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/manager"
	managerservices "github.com/oracle/oci-service-operator/pkg/manager/services"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(keymanagementv1beta1.AddToScheme(scheme))
}

func main() {
	if err := manager.Run(manager.Options{
		Scheme:             scheme,
		MetricsServiceName: "keymanagement",
		LeaderElectionID:   "40558063.oci.keymanagement",
		SkipFIPS:           true,
	}, managerservices.ForGroup("keymanagement")); err != nil {
		os.Exit(1)
	}
}

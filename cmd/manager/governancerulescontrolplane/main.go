package main

import (
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	governancerulescontrolplanev1beta1 "github.com/oracle/oci-service-operator/api/governancerulescontrolplane/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/manager"
	managerservices "github.com/oracle/oci-service-operator/pkg/manager/services"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(governancerulescontrolplanev1beta1.AddToScheme(scheme))
}

func main() {
	if err := manager.Run(manager.Options{
		Scheme:             scheme,
		MetricsServiceName: "governancerulescontrolplane",
		LeaderElectionID:   "40558063.oci.governancerulescontrolplane",
		SkipFIPS:           true,
	}, managerservices.ForGroup("governancerulescontrolplane")); err != nil {
		os.Exit(1)
	}
}

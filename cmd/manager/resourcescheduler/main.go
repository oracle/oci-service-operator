package main

import (
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	resourceschedulerv1beta1 "github.com/oracle/oci-service-operator/api/resourcescheduler/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/manager"
	managerservices "github.com/oracle/oci-service-operator/pkg/manager/services"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(resourceschedulerv1beta1.AddToScheme(scheme))
}

func main() {
	if err := manager.Run(manager.Options{
		Scheme:             scheme,
		MetricsServiceName: "resourcescheduler",
		LeaderElectionID:   "40558063.oci.resourcescheduler",
		SkipFIPS:           true,
	}, managerservices.ForGroup("resourcescheduler")); err != nil {
		os.Exit(1)
	}
}

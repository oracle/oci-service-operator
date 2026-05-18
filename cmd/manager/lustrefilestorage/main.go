package main

import (
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	lustrefilestoragev1beta1 "github.com/oracle/oci-service-operator/api/lustrefilestorage/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/manager"
	managerservices "github.com/oracle/oci-service-operator/pkg/manager/services"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(lustrefilestoragev1beta1.AddToScheme(scheme))
}

func main() {
	if err := manager.Run(manager.Options{
		Scheme:             scheme,
		MetricsServiceName: "lustrefilestorage",
		LeaderElectionID:   "40558063.oci.lustrefilestorage",
		SkipFIPS:           true,
	}, managerservices.ForGroup("lustrefilestorage")); err != nil {
		os.Exit(1)
	}
}

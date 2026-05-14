package main

import (
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	jmsjavadownloadsv1beta1 "github.com/oracle/oci-service-operator/api/jmsjavadownloads/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/manager"
	managerservices "github.com/oracle/oci-service-operator/pkg/manager/services"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(jmsjavadownloadsv1beta1.AddToScheme(scheme))
}

func main() {
	if err := manager.Run(manager.Options{
		Scheme:             scheme,
		MetricsServiceName: "jmsjavadownloads",
		LeaderElectionID:   "40558063.oci.jmsjavadownloads",
		SkipFIPS:           true,
	}, managerservices.ForGroup("jmsjavadownloads")); err != nil {
		os.Exit(1)
	}
}

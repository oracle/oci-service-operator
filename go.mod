module github.com/oracle/oci-service-operator

go 1.15

replace (
	k8s.io/api => k8s.io/api v0.19.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.2
	k8s.io/client-go => k8s.io/client-go v0.19.2
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.7.0
)

require (
	github.com/go-logr/logr v0.4.0
	github.com/golang/mock v1.6.0
	github.com/google/go-cmp v0.5.7
	github.com/iancoleman/strcase v0.2.0
	github.com/onsi/ginkgo v1.16.3
	github.com/onsi/gomega v1.13.0
	github.com/oracle/oci-go-sdk/v65 v65.28.1
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.10.0
	github.com/stretchr/testify v1.6.1
	golang.org/x/time v0.0.0-20220224211638-0e9765cccd65
	gomodules.xyz/jsonpatch/v2 v2.2.0
	k8s.io/api v0.19.2
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v0.19.2
	sigs.k8s.io/controller-runtime v0.7.0
)

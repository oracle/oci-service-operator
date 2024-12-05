module github.com/oracle/oci-service-operator

go 1.15

replace (
	k8s.io/api => k8s.io/api v0.20.6
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.6
	k8s.io/client-go => k8s.io/client-go v0.20.6
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.9.0
)

require (
	github.com/go-logr/logr v1.4.1
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/go-cmp v0.6.0
	github.com/iancoleman/strcase v0.2.0
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.34.0
	github.com/oracle/oci-go-sdk/v65 v65.61.1
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.1
	github.com/stretchr/testify v1.8.4
	golang.org/x/time v0.0.0-20220224211638-0e9765cccd65
	gomodules.xyz/jsonpatch/v2 v2.2.0
	k8s.io/api v0.21.1
	k8s.io/apimachinery v0.21.1
	k8s.io/client-go v0.21.1
	sigs.k8s.io/controller-runtime v0.9.0
)

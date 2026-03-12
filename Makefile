#
# Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
# Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
#


# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=1.0.0)
# - use environment variables to overwrite this value (e.g export VERSION=1.0.0)
VERSION ?= 1.0.0

# CHANNELS define the bundle channels used in the bundle.
# Add a new line here if you would like to change its default config. (E.g CHANNELS = "preview,fast,stable")
# To re-generate a bundle for other specific channels without changing the standard setup, you can:
# - use the CHANNELS as arg of the bundle target (e.g make bundle CHANNELS=preview,fast,stable)
# - use environment variables to overwrite this value (e.g export CHANNELS="preview,fast,stable")
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

# DEFAULT_CHANNEL defines the default channel used in the bundle.
# Add a new line here if you would like to change its default config. (E.g DEFAULT_CHANNEL = "stable")
# To re-generate a bundle for any other default channel without changing the default setup, you can:
# - use the DEFAULT_CHANNEL as arg of the bundle target (e.g make bundle DEFAULT_CHANNEL=stable)
# - use environment variables to overwrite this value (e.g export DEFAULT_CHANNEL="stable")
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# IMAGE_TAG_BASE defines the docker.io namespace and part of the image name for remote images.
# This variable is used to construct full image tags for bundle and catalog images.
#
# For example, running 'make bundle-build bundle-push catalog-build catalog-push' will build and push both
# oraclecloud/upgradeoperatorsdk-bundle:$VERSION and oraclecloud/upgradeoperatorsdk-catalog:$VERSION.
IMAGE_TAG_BASE ?= iad.ocir.io/oracle/oci-service-operator

# BUNDLE_IMG defines the image:tag used for the bundle.
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)
BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-bundle:$(VERSION)
BUNDLE_PACKAGE ?= oci-service-operator
OPERATOR_SDK_VERSION ?= v1.37.0

# Image URL to use all building/pushing image targets
IMG ?= $(IMAGE_TAG_BASE):$(VERSION)
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:generateEmbeddedObjectMeta=true,allowDangerousTypes=true"
GENERATED_CRD_ARTIFACTS ?= config/crd/bases/*.yaml
GENERATED_CSV_ARTIFACTS ?= config/manifests/bases/*.clusterserviceversion.yaml
GENERATED_BUNDLE_ARTIFACTS ?= bundle/manifests/*.yaml bundle/metadata/annotations.yaml bundle.Dockerfile
GROUP ?= database
PACKAGES_DIR ?= packages
PACKAGE_DIR ?= $(PACKAGES_DIR)/$(GROUP)
PACKAGE_OUTPUT_DIR ?= dist/packages/$(GROUP)
PACKAGE_SCRIPT ?= hack/package.sh
CONTROLLER_IMG ?=
API_GENERATOR_CONFIG ?= internal/generator/config/services.yaml
API_GENERATOR_OUTPUT_ROOT ?= .
API_SERVICE ?=
API_ALL ?=
API_OVERWRITE ?=

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	rm -f $(GENERATED_CRD_ARTIFACTS)
	$(CONTROLLER_GEN) $(CRD_OPTIONS) paths="./api/..." output:crd:artifacts:config=config/crd/bases
	$(CONTROLLER_GEN) rbac:roleName=manager-role paths="./controllers/..." output:rbac:artifacts:config=config/rbac
	$(CONTROLLER_GEN) webhook paths="./api/..." output:webhook:artifacts:config=config/webhook

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

api-generate: ## Run the OSOK API generator. Use API_SERVICE=<service> or API_ALL=true. Set API_OVERWRITE=true to replace existing generator-owned outputs.
	@set -e; \
	is_true() { case "$$1" in 1|true|TRUE|yes|YES) return 0 ;; *) return 1 ;; esac; }; \
	if [ -n "$(API_SERVICE)" ] && is_true "$(API_ALL)"; then \
		echo "Use either API_SERVICE=<service> or API_ALL=true, not both."; \
		exit 1; \
	fi; \
	if [ -z "$(API_SERVICE)" ] && ! is_true "$(API_ALL)"; then \
		echo "Set API_SERVICE=<service> or API_ALL=true."; \
		exit 1; \
	fi; \
	args="--config $(API_GENERATOR_CONFIG) --output-root $(API_GENERATOR_OUTPUT_ROOT)"; \
	if [ -n "$(API_SERVICE)" ]; then \
		args="$$args --service $(API_SERVICE)"; \
	else \
		args="$$args --all"; \
	fi; \
	if is_true "$(API_OVERWRITE)"; then \
		args="$$args --overwrite"; \
	fi; \
	go run ./cmd/osok-api-generator $$args

api-refresh: ## Run the OSOK API generator, then refresh deepcopy and manifest artifacts.
	@$(MAKE) api-generate API_SERVICE="$(API_SERVICE)" API_ALL="$(API_ALL)" API_OVERWRITE="$(API_OVERWRITE)" API_GENERATOR_CONFIG="$(API_GENERATOR_CONFIG)" API_GENERATOR_OUTPUT_ROOT="$(API_GENERATOR_OUTPUT_ROOT)"
	@$(MAKE) generate
	@$(MAKE) manifests

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

BASH ?= /bin/bash
ENVTEST_ASSETS_DIR ?= $(shell pwd)/testbin/$(shell uname)
ENVTEST_HOME ?= $(shell pwd)/.envtest-home
ENVTEST_CACHE_DIR ?= $(ENVTEST_HOME)/.cache
ENVTEST_CONFIG_DIR ?= $(ENVTEST_HOME)/.config
ENVTEST_K8S_VERSION ?= 1.28.0
ENVTEST_ENV ?= HOME=$(ENVTEST_HOME) XDG_CACHE_HOME=$(ENVTEST_CACHE_DIR) XDG_CONFIG_HOME=$(ENVTEST_CONFIG_DIR)
# setup-envtest is published from a separate tool module; pin the release-0.17-compatible revision.
SETUP_ENVTEST_VERSION ?= v0.0.0-20240812162837-9557f1031fe4
SETUP_ENVTEST_GOFLAGS ?= $(strip $(filter-out -mod=%,$(GOFLAGS)) -mod=mod)
SETUP_ENVTEST ?= $(ENVTEST_ENV) GOFLAGS="$(SETUP_ENVTEST_GOFLAGS)" go run sigs.k8s.io/controller-runtime/tools/setup-envtest@$(SETUP_ENVTEST_VERSION) use $(ENVTEST_K8S_VERSION) -p path --bin-dir $(ENVTEST_ASSETS_DIR) --use-deprecated-gcs=false

test: manifests generate fmt vet ## Run tests.
	mkdir -p $(ENVTEST_ASSETS_DIR) $(ENVTEST_CACHE_DIR) $(ENVTEST_CONFIG_DIR)
	$(BASH) -o pipefail -ec '\
		envtest_assets="$$( $(SETUP_ENVTEST) )"; \
		$(ENVTEST_ENV) KUBEBUILDER_ASSETS="$$envtest_assets" go test ./... -coverprofile cover.out | tee unittests.cover'
	go tool cover -func cover.out | grep total | awk '{print substr($$3, 1, length($$3)-1)}' > unittests.percent

functionaltest: ## Run functionaltest (placeholder — no functional tests yet).
	@echo "No functional tests available."

##@ Build Service

test-sample: fmt vet ## Run tests.
	mkdir -p $(ENVTEST_ASSETS_DIR) $(ENVTEST_CACHE_DIR) $(ENVTEST_CONFIG_DIR)
	$(BASH) -o pipefail -ec '\
		envtest_assets="$$( $(SETUP_ENVTEST) )"; \
		$(ENVTEST_ENV) KUBEBUILDER_ASSETS="$$envtest_assets" go test -v ./... -coverprofile cover.out -args -ginkgo.v'

docker-build-sample: ## Build docker image with the manager.
	docker build -t ${IMG} .

##@ Build

build: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

docker-build: test bundle ## Build docker image with the manager and CRDs
	docker build -t ${IMG} .

docker-push: ## Push docker image with the manager.
	docker push ${IMG}

##@ Packages

packages: ## List configured package groups under packages/.
	@find $(PACKAGES_DIR) -mindepth 1 -maxdepth 1 -type d -exec basename {} \; | sort

package-generate: controller-gen ## Generate CRDs and optional controller RBAC for GROUP into packages/<group>/install/generated.
	@test -f "$(PACKAGE_DIR)/metadata.env" || { echo "Unknown GROUP '$(GROUP)'. See 'make packages'."; exit 1; }
	@CONTROLLER_GEN="$(CONTROLLER_GEN)" "$(PACKAGE_SCRIPT)" generate "$(GROUP)"

package-install: controller-gen kustomize ## Render a single install YAML for GROUP into dist/packages/<group>/install.yaml.
	@test -f "$(PACKAGE_DIR)/metadata.env" || { echo "Unknown GROUP '$(GROUP)'. See 'make packages'."; exit 1; }
	@CONTROLLER_GEN="$(CONTROLLER_GEN)" KUSTOMIZE="$(KUSTOMIZE)" CONTROLLER_IMG="$(CONTROLLER_IMG)" OUT="$(PACKAGE_OUTPUT_DIR)/install.yaml" \
		"$(PACKAGE_SCRIPT)" render "$(GROUP)"

##@ Deployment

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete -f -


CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.17.0)

KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v5@v5.4.2)

OPERATOR_SDK = $(shell pwd)/bin/operator-sdk
operator-sdk: ## Download a compatible operator-sdk locally if necessary.
ifeq (,$(wildcard $(OPERATOR_SDK)))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPERATOR_SDK)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPERATOR_SDK) https://github.com/operator-framework/operator-sdk/releases/download/$(OPERATOR_SDK_VERSION)/operator-sdk_$${OS}_$${ARCH} ;\
	chmod +x $(OPERATOR_SDK) ;\
	}
endif

# go-get-tool will 'go install' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
}
endef

.PHONY: bundle
bundle: manifests kustomize operator-sdk ## Generate bundle manifests and metadata, then validate generated files.
	rm -f $(GENERATED_BUNDLE_ARTIFACTS)
	$(OPERATOR_SDK) generate kustomize manifests -q --interactive=false --package $(BUNDLE_PACKAGE)
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | $(OPERATOR_SDK) generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	$(OPERATOR_SDK) bundle validate ./bundle

.PHONY: bundle-build
bundle-build: ## Build the bundle image.
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	$(MAKE) docker-push IMG=$(BUNDLE_IMG)

.PHONY: update-bundle-image-version
update-bundle-image-version: ## Updates versioning info in bundle/manifests/oci-service-operator.clusterserviceversion.yaml.
	sed -i "s/name: oci-service-operator.v1.0.0/name: oci-service-operator.v${VERSION}/g" bundle/manifests/oci-service-operator.clusterserviceversion.yaml
	sed -i "s#iad.ocir.io/oracle/oci-service-operator:1.0.0#${IMAGE_TAG_BASE}:${VERSION}#g" bundle/manifests/oci-service-operator.clusterserviceversion.yaml
	sed -i "s/version: 1.0.0/version: ${VERSION}/g" bundle/manifests/oci-service-operator.clusterserviceversion.yaml

.PHONY: opm
OPM = ./bin/opm
opm: ## Download opm locally if necessary.
ifeq (,$(wildcard $(OPM)))
ifeq (,$(shell which opm 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPM)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/v1.15.1/$${OS}-$${ARCH}-opm ;\
	chmod +x $(OPM) ;\
	}
else
OPM = $(shell which opm)
endif
endif

# A comma-separated list of bundle images (e.g. make catalog-build BUNDLE_IMGS=example.com/operator-bundle:v0.1.0,example.com/operator-bundle:v0.2.0).
# These images MUST exist in a registry and be pull-able.
BUNDLE_IMGS ?= $(BUNDLE_IMG)

# The image tag given to the resulting catalog image (e.g. make catalog-build CATALOG_IMG=example.com/operator-catalog:v0.2.0).
CATALOG_IMG ?= $(IMAGE_TAG_BASE)-catalog:v$(VERSION)

# Set CATALOG_BASE_IMG to an existing catalog image tag to add $BUNDLE_IMGS to that image.
ifneq ($(origin CATALOG_BASE_IMG), undefined)
FROM_INDEX_OPT := --from-index $(CATALOG_BASE_IMG)
endif

# Build a catalog image by adding bundle images to an empty catalog using the operator package manager tool, 'opm'.
# This recipe invokes 'opm' in 'semver' bundle add mode. For more information on add modes, see:
# https://github.com/operator-framework/community-operators/blob/7f1438c/docs/packaging-operator.md#updating-your-existing-operator
.PHONY: catalog-build
catalog-build: opm ## Build a catalog image.
	$(OPM) index add --container-tool docker --mode semver --tag $(CATALOG_IMG) --bundles $(BUNDLE_IMGS) $(FROM_INDEX_OPT)

# Push the catalog image.
.PHONY: catalog-push
catalog-push: ## Push a catalog image.
	$(MAKE) docker-push IMG=$(CATALOG_IMG)

delete-crds:
	kubectl delete crd autonomousdatabases.oci.oracle.com &
	kubectl delete crd mysqldbsystems.oci.oracle.com &
	kubectl delete crd streams.oci.oracle.com &

delete-operator:
	kubectl delete ns $(OPERATOR_NAMESPACE)

.PHONY: delete-crds-force
delete-crds-force:
	kubectl patch crd/autonomousdatabases.oci.oracle.com -p '{"metadata":{"finalizers":[]}}' --type=merge &
	kubectl patch crd/mysqldbsystems.oci.oracle.com -p '{"metadata":{"finalizers":[]}}' --type=merge &
	kubectl patch crd/streams.oci.oracle.com -p '{"metadata":{"finalizers":[]}}' --type=merge

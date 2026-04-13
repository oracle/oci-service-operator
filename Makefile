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
SERVICE ?=
PUBLISH_VERSION ?=
PUBLISH_REGISTRY ?=
PUBLISH_PLATFORMS ?=
PUBLISH_PLATFORM ?= linux_amd64
PUBLISH_CGO_ENABLED ?= 0
PUBLISH_GOEXPERIMENT ?=
PUBLISH_USE_DOCKER_PLATFORM ?= false
MONOLITH_OLM_BUNDLE_IMG ?=
TARGETOS ?= linux
TARGETARCH ?= amd64
CONTROLLER_MAIN ?= .
SERVICE_IMG ?= $(IMAGE_TAG_BASE)-$(SERVICE):$(VERSION)
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
PACKAGE_OLM_SCRIPT ?= hack/package-olm.sh
MONOLITH_SCRIPT ?= hack/monolith.sh
CONTROLLER_IMG ?=
GENERATOR_ENTRYPOINT ?= ./cmd/generator
GENERATOR_CONFIG ?=
GENERATOR_OUTPUT_ROOT ?=
GENERATOR_SERVICE ?=
GENERATOR_ALL ?=
GENERATOR_OVERWRITE ?=
EFFECTIVE_GENERATOR_CONFIG = $(or $(strip $(GENERATOR_CONFIG)),internal/generator/config/services.yaml)
EFFECTIVE_GENERATOR_OUTPUT_ROOT = $(or $(strip $(GENERATOR_OUTPUT_ROOT)),.)
EFFECTIVE_GENERATOR_SERVICE = $(strip $(GENERATOR_SERVICE))
EFFECTIVE_GENERATOR_ALL = $(strip $(GENERATOR_ALL))
EFFECTIVE_GENERATOR_OVERWRITE = $(strip $(GENERATOR_OVERWRITE))

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
	"$(CONTROLLER_GEN_RUNNER)" "$(CONTROLLER_GEN)" $(CRD_OPTIONS) paths="./api/..." output:crd:artifacts:config=config/crd/bases
	@$(MAKE) crd-kustomization-sync
	"$(CONTROLLER_GEN_RUNNER)" "$(CONTROLLER_GEN)" rbac:roleName=manager-role paths="./controllers/..." output:rbac:artifacts:config=config/rbac
	"$(CONTROLLER_GEN_RUNNER)" "$(CONTROLLER_GEN)" webhook paths="./api/..." output:webhook:artifacts:config=config/webhook

crd-kustomization-sync: ## Refresh shared CRD aggregation from config/crd/bases.
	go run ./cmd/osok-crd-sync --kustomization config/crd/kustomization.yaml --bases-dir config/crd/bases

DEEPCOPY_GEN_PATHS ?= "./api/...;./pkg/shared"

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	"$(CONTROLLER_GEN_RUNNER)" "$(CONTROLLER_GEN)" object:headerFile="hack/boilerplate.go.txt" paths=$(DEEPCOPY_GEN_PATHS)

generator-generate: ## Run the OSOK generator. Use GENERATOR_SERVICE=<service> or GENERATOR_ALL=true.
	@set -e; \
	is_true() { case "$$1" in 1|true|TRUE|yes|YES) return 0 ;; *) return 1 ;; esac; }; \
	service="$(EFFECTIVE_GENERATOR_SERVICE)"; \
	all="$(EFFECTIVE_GENERATOR_ALL)"; \
	config="$(EFFECTIVE_GENERATOR_CONFIG)"; \
	output_root="$(EFFECTIVE_GENERATOR_OUTPUT_ROOT)"; \
	overwrite="$(EFFECTIVE_GENERATOR_OVERWRITE)"; \
	if [ -n "$$service" ] && is_true "$$all"; then \
		echo "Use either GENERATOR_SERVICE=<service> or GENERATOR_ALL=true, not both."; \
		exit 1; \
	fi; \
	if [ -z "$$service" ] && ! is_true "$$all"; then \
		echo "Set GENERATOR_SERVICE=<service> or GENERATOR_ALL=true."; \
		exit 1; \
	fi; \
	args="--config $$config --output-root $$output_root"; \
	if [ -n "$$service" ]; then \
		args="$$args --service $$service"; \
	else \
		args="$$args --all"; \
	fi; \
	if is_true "$$overwrite"; then \
		args="$$args --overwrite"; \
	fi; \
	go run $(GENERATOR_ENTRYPOINT) $$args

generator-refresh: ## Run the OSOK generator, then refresh deepcopy and manifest artifacts.
	@$(MAKE) generator-generate
	@$(MAKE) generate
	@$(MAKE) manifests

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

lint: ## Run golangci-lint against handwritten repo code.
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "golangci-lint is required but not installed. Install it locally and retry 'make lint'."; \
		exit 1; \
	}
	golangci-lint run ./...

DOCS_PYTHON ?= python3
DOCS_SITE_DIR ?= site
DOCS_OUTPUT_ROOT ?= .
DOCS_VERIFY_STRICT_PUBLIC_DESCRIPTIONS ?=
DOCS_VERIFY_STRICT_PUBLIC_DESCRIPTIONS_ARG = $(if $(filter 1 true TRUE yes YES,$(DOCS_VERIFY_STRICT_PUBLIC_DESCRIPTIONS)),--strict-public-descriptions,)

docs-generate: ## Regenerate checked-in docs/reference outputs from repo metadata and CRD schemas.
	go run ./cmd/sitegen reference --repo-root . --output-root $(DOCS_OUTPUT_ROOT)

docs-build: ## Build the MkDocs site locally with strict validation.
	$(DOCS_PYTHON) -m mkdocs build --strict --site-dir $(DOCS_SITE_DIR)

docs-serve: ## Serve the MkDocs site locally for preview.
	$(DOCS_PYTHON) -m mkdocs serve

docs-verify: manifests docs-build ## Verify docs drift, built-site links/anchors, and description coverage.
	go run ./cmd/sitegen verify --repo-root . --site-dir $(DOCS_SITE_DIR) $(DOCS_VERIFY_STRICT_PUBLIC_DESCRIPTIONS_ARG)

SCHEMA_VALIDATOR_PROVIDER_PATH ?= .
SCHEMA_VALIDATOR_FORMAT ?= json
SCHEMA_VALIDATOR_REPORT ?= validator-report.json
SCHEMA_VALIDATOR_SERVICE ?=
SCHEMA_VALIDATOR_SERVICE_ARG = $(if $(strip $(SCHEMA_VALIDATOR_SERVICE)),--service $(SCHEMA_VALIDATOR_SERVICE),)
FORMAL_ROOT ?= formal
FORMAL_IMPORT_PROVIDER_PATH ?=
FORMAL_IMPORT_PROVIDER_REVISION ?=
FORMAL_IMPORT_SOURCE_NAME ?= terraform-provider-oci
FORMAL_PROVIDER_PATH ?= $(FORMAL_IMPORT_PROVIDER_PATH)
FORMAL_PROVIDER_PATH_ARG = $(if $(strip $(FORMAL_PROVIDER_PATH)),--provider-path $(FORMAL_PROVIDER_PATH),)
FORMAL_IMPORT_PROVIDER_REVISION_ARG = $(if $(strip $(FORMAL_IMPORT_PROVIDER_REVISION)),--provider-revision $(FORMAL_IMPORT_PROVIDER_REVISION),)

schema-validator: ## Run OSOK schema validator and write report to SCHEMA_VALIDATOR_REPORT.
	go run ./cmd/osok-schema-validator --provider-path $(SCHEMA_VALIDATOR_PROVIDER_PATH) $(SCHEMA_VALIDATOR_SERVICE_ARG) --format $(SCHEMA_VALIDATOR_FORMAT) > $(SCHEMA_VALIDATOR_REPORT)
	@echo "Wrote schema validator report to $(SCHEMA_VALIDATOR_REPORT)"

formal-diagrams: ## Regenerate shared and controller-local formal diagram artifacts from formal metadata.
	go run ./cmd/formal-diagrams --root $(FORMAL_ROOT)

formal-import: ## Refresh provider-fact JSON and pin sources.lock from FORMAL_IMPORT_PROVIDER_PATH.
	@test -n "$(FORMAL_IMPORT_PROVIDER_PATH)" || (echo "Set FORMAL_IMPORT_PROVIDER_PATH=/path/to/terraform-provider-oci" && exit 1)
	go run ./cmd/formal-import --root $(FORMAL_ROOT) --provider-path $(FORMAL_IMPORT_PROVIDER_PATH) --source-name $(FORMAL_IMPORT_SOURCE_NAME) $(FORMAL_IMPORT_PROVIDER_REVISION_ARG)

formal-verify: ## Validate the repo-local formal schema scaffold, bindings, and gap categories.
	go run ./cmd/formal-verify --root $(FORMAL_ROOT)

formal-scaffold: ## Expand scaffold-only formal entries from the published default-active API surface and optional matching terraform-provider-oci facts.
	go run ./cmd/formal-scaffold --root $(FORMAL_ROOT) --config $(EFFECTIVE_GENERATOR_CONFIG) $(FORMAL_PROVIDER_PATH_ARG)

formal-scaffold-verify: ## Verify formal scaffold coverage for the published default-active API surface against matching terraform-provider-oci facts.
	@test -n "$(FORMAL_PROVIDER_PATH)" || (echo "Set FORMAL_PROVIDER_PATH=/path/to/terraform-provider-oci" && exit 1)
	go run ./cmd/formal-scaffold-verify --root $(FORMAL_ROOT) --config $(EFFECTIVE_GENERATOR_CONFIG) --provider-path $(FORMAL_PROVIDER_PATH)

GENERATED_COVERAGE_REPORT ?= generated-coverage-report.json
GENERATED_COVERAGE_SERVICE ?=
GENERATED_COVERAGE_TOP ?= 10
GENERATED_COVERAGE_SNAPSHOT_DIR ?=
GENERATED_COVERAGE_KEEP_SNAPSHOT ?=
GENERATED_COVERAGE_VALIDATOR_JSON ?=
GENERATED_COVERAGE_BASELINE ?= internal/generator/config/generated_coverage_baseline.json
GENERATED_COVERAGE_SERVICE_ARG = $(if $(strip $(GENERATED_COVERAGE_SERVICE)),--service $(GENERATED_COVERAGE_SERVICE),--all)
GENERATED_COVERAGE_SNAPSHOT_ARG = $(if $(strip $(GENERATED_COVERAGE_SNAPSHOT_DIR)),--snapshot-dir $(GENERATED_COVERAGE_SNAPSHOT_DIR),)
GENERATED_COVERAGE_KEEP_ARG = $(if $(filter 1 true TRUE yes YES,$(GENERATED_COVERAGE_KEEP_SNAPSHOT)),--keep-snapshot,)
GENERATED_COVERAGE_VALIDATOR_JSON_ARG = $(if $(strip $(GENERATED_COVERAGE_VALIDATOR_JSON)),--validator-json-out $(GENERATED_COVERAGE_VALIDATOR_JSON),)
GENERATED_COVERAGE_BASELINE_ARG = $(if $(strip $(GENERATED_COVERAGE_BASELINE)),--baseline $(GENERATED_COVERAGE_BASELINE),)
GENERATED_RUNTIME_CONFIG ?= $(EFFECTIVE_GENERATOR_CONFIG)
GENERATED_RUNTIME_REPORT ?= generated-runtime-report.json
GENERATED_RUNTIME_SERVICE ?=
GENERATED_RUNTIME_SNAPSHOT_DIR ?=
GENERATED_RUNTIME_KEEP_SNAPSHOT ?=
GENERATED_RUNTIME_SERVICE_ARG = $(if $(strip $(GENERATED_RUNTIME_SERVICE)),--service $(GENERATED_RUNTIME_SERVICE),--all)
GENERATED_RUNTIME_SNAPSHOT_ARG = $(if $(strip $(GENERATED_RUNTIME_SNAPSHOT_DIR)),--snapshot-dir $(GENERATED_RUNTIME_SNAPSHOT_DIR),)
GENERATED_RUNTIME_KEEP_ARG = $(if $(filter 1 true TRUE yes YES,$(GENERATED_RUNTIME_KEEP_SNAPSHOT)),--keep-snapshot,)

generated-coverage-report: controller-gen ## Generate APIs in a snapshot tree, run validator coverage, and write a JSON summary.
	"$(CONTROLLER_GEN_RUNNER)" go run ./cmd/osok-generated-coverage --config $(EFFECTIVE_GENERATOR_CONFIG) $(GENERATED_COVERAGE_SERVICE_ARG) --top $(GENERATED_COVERAGE_TOP) --controller-gen $(CONTROLLER_GEN) --report-out $(GENERATED_COVERAGE_REPORT) $(GENERATED_COVERAGE_SNAPSHOT_ARG) $(GENERATED_COVERAGE_KEEP_ARG) $(GENERATED_COVERAGE_VALIDATOR_JSON_ARG)
	@echo "Wrote generated coverage report to $(GENERATED_COVERAGE_REPORT)"

generated-coverage-baseline: controller-gen ## Refresh the checked-in generated coverage baseline intentionally.
	"$(CONTROLLER_GEN_RUNNER)" go run ./cmd/osok-generated-coverage --config $(EFFECTIVE_GENERATOR_CONFIG) --all --top $(GENERATED_COVERAGE_TOP) --controller-gen $(CONTROLLER_GEN) --report-out $(GENERATED_COVERAGE_REPORT) --write-baseline $(GENERATED_COVERAGE_BASELINE) $(GENERATED_COVERAGE_SNAPSHOT_ARG) $(GENERATED_COVERAGE_KEEP_ARG) $(GENERATED_COVERAGE_VALIDATOR_JSON_ARG)
	@echo "Wrote generated coverage report to $(GENERATED_COVERAGE_REPORT)"
	@echo "Updated generated coverage baseline at $(GENERATED_COVERAGE_BASELINE)"

generated-coverage-gate: controller-gen ## Fail when generated API coverage regresses compared to the checked-in baseline.
	"$(CONTROLLER_GEN_RUNNER)" go run ./cmd/osok-generated-coverage --config $(EFFECTIVE_GENERATOR_CONFIG) --all --top $(GENERATED_COVERAGE_TOP) --controller-gen $(CONTROLLER_GEN) --report-out $(GENERATED_COVERAGE_REPORT) $(GENERATED_COVERAGE_BASELINE_ARG) --fail-on-regression $(GENERATED_COVERAGE_SNAPSHOT_ARG) $(GENERATED_COVERAGE_KEEP_ARG) $(GENERATED_COVERAGE_VALIDATOR_JSON_ARG)
	@echo "Generated coverage gate passed; report at $(GENERATED_COVERAGE_REPORT)"

generated-runtime-report: controller-gen ## Generate a runtime snapshot and compile generated controller/service-manager outputs.
	"$(CONTROLLER_GEN_RUNNER)" go run ./cmd/osok-generated-runtime-check --config $(GENERATED_RUNTIME_CONFIG) $(GENERATED_RUNTIME_SERVICE_ARG) --controller-gen $(CONTROLLER_GEN) --report-out $(GENERATED_RUNTIME_REPORT) $(GENERATED_RUNTIME_SNAPSHOT_ARG) $(GENERATED_RUNTIME_KEEP_ARG)
	@echo "Wrote generated runtime report to $(GENERATED_RUNTIME_REPORT)"

generated-runtime-gate: controller-gen ## Fail when generated controller/service-manager snapshot outputs do not compile.
	"$(CONTROLLER_GEN_RUNNER)" go run ./cmd/osok-generated-runtime-check --config $(GENERATED_RUNTIME_CONFIG) --all --controller-gen $(CONTROLLER_GEN) --report-out $(GENERATED_RUNTIME_REPORT) $(GENERATED_RUNTIME_SNAPSHOT_ARG) $(GENERATED_RUNTIME_KEEP_ARG)
	@echo "Generated runtime gate passed; report at $(GENERATED_RUNTIME_REPORT)"

generator-validation: generated-coverage-gate generated-runtime-gate ## Run generator regression gates for API coverage and generated runtime outputs.
	@echo "Generator validation passed."

BASH ?= /bin/bash
# Keep envtest state outside the module tree so controller-gen never walks its module cache.
ENVTEST_TMPDIR ?= $(patsubst %/,%,$(or $(TMPDIR),/tmp))
ENVTEST_ROOT ?= $(ENVTEST_TMPDIR)/oci-service-operator-envtest
ENVTEST_ASSETS_DIR ?= $(ENVTEST_ROOT)/testbin/$(shell go env GOOS)-$(shell go env GOARCH)
ENVTEST_HOME ?= $(ENVTEST_ROOT)/home
ENVTEST_CACHE_DIR ?= $(ENVTEST_HOME)/.cache
ENVTEST_CONFIG_DIR ?= $(ENVTEST_HOME)/.config
ENVTEST_K8S_VERSION ?= 1.28.0
ENVTEST_ENV ?= HOME=$(ENVTEST_HOME) XDG_CACHE_HOME=$(ENVTEST_CACHE_DIR) XDG_CONFIG_HOME=$(ENVTEST_CONFIG_DIR)
ENVTEST_INSTALLED_ONLY ?=
ENVTEST_USE_ENV ?=
ENVTEST_LEGACY_GOMODCACHE ?= $(CURDIR)/.envtest-home/.gomodcache
SETUP_ENVTEST_GOPATH ?= $(ENVTEST_ROOT)/gopath
SETUP_ENVTEST_ENV ?= env -u GOMODCACHE $(ENVTEST_ENV) GOPATH=$(SETUP_ENVTEST_GOPATH)
# setup-envtest is published from a separate tool module; pin the release-0.17-compatible revision.
SETUP_ENVTEST_VERSION ?= v0.0.0-20240812162837-9557f1031fe4
SETUP_ENVTEST_GOFLAGS ?= $(strip $(filter-out -mod=%,$(GOFLAGS)) -mod=mod)
SETUP_ENVTEST_RUN ?= $(SETUP_ENVTEST_ENV) GOFLAGS="$(SETUP_ENVTEST_GOFLAGS)" go run sigs.k8s.io/controller-runtime/tools/setup-envtest@$(SETUP_ENVTEST_VERSION)
SETUP_ENVTEST ?= $(SETUP_ENVTEST_RUN) use $(ENVTEST_K8S_VERSION) -p path --bin-dir $(ENVTEST_ASSETS_DIR) --use-deprecated-gcs=false
ENVTEST_PREPARE_DIRS = rm -rf $(ENVTEST_LEGACY_GOMODCACHE); mkdir -p $(ENVTEST_ASSETS_DIR) $(ENVTEST_CACHE_DIR) $(ENVTEST_CONFIG_DIR) $(SETUP_ENVTEST_GOPATH)

define ENVTEST_RESOLVE_ASSETS
is_true() { case "$$1" in 1|true|TRUE|yes|YES) return 0 ;; *) return 1 ;; esac; }; \
has_envtest_assets() { \
	test -x "$(ENVTEST_ASSETS_DIR)/kube-apiserver" && \
	test -x "$(ENVTEST_ASSETS_DIR)/etcd"; \
}; \
if is_true "$(ENVTEST_USE_ENV)"; then \
	if [ -z "$${KUBEBUILDER_ASSETS:-}" ]; then \
		echo "ENVTEST_USE_ENV=true requires KUBEBUILDER_ASSETS to point at an envtest asset directory." >&2; \
		exit 1; \
	fi; \
	envtest_assets="$$KUBEBUILDER_ASSETS"; \
elif has_envtest_assets; then \
	envtest_assets="$(ENVTEST_ASSETS_DIR)"; \
elif is_true "$(ENVTEST_INSTALLED_ONLY)"; then \
	echo "No installed envtest assets were found in $(ENVTEST_ASSETS_DIR)." >&2; \
	echo "Run make envtest while network access is available, or set KUBEBUILDER_ASSETS and ENVTEST_USE_ENV=true." >&2; \
	exit 1; \
else \
	if ! envtest_assets="$$( $(SETUP_ENVTEST) )"; then \
		echo "envtest bootstrap failed before package tests started." >&2; \
		echo "Run make envtest while network access is available, then rerun ENVTEST_INSTALLED_ONLY=true make test." >&2; \
		echo "If you already have an asset bundle, set KUBEBUILDER_ASSETS=/path and ENVTEST_USE_ENV=true." >&2; \
		exit 1; \
	fi; \
fi
endef

envtest: ## Download and cache the pinned envtest assets for later installed-only test runs.
	$(ENVTEST_PREPARE_DIRS)
	@envtest_assets="$$( $(SETUP_ENVTEST) )"; \
		echo "Envtest assets available at $$envtest_assets"

test: manifests generate fmt vet ## Run tests.
	$(ENVTEST_PREPARE_DIRS)
	$(BASH) -o pipefail -ec '\
		$(ENVTEST_RESOLVE_ASSETS); \
		$(ENVTEST_ENV) KUBEBUILDER_ASSETS="$$envtest_assets" go test ./... -coverprofile cover.out | tee unittests.cover'
	go tool cover -func cover.out | grep total | awk '{print substr($$3, 1, length($$3)-1)}' > unittests.percent

functionaltest: ## Run functionaltest (placeholder — no functional tests yet).
	@echo "No functional tests available."

##@ Build Service

test-sample: fmt vet ## Run tests.
	$(ENVTEST_PREPARE_DIRS)
	$(BASH) -o pipefail -ec '\
		$(ENVTEST_RESOLVE_ASSETS); \
		$(ENVTEST_ENV) KUBEBUILDER_ASSETS="$$envtest_assets" go test -v ./... -coverprofile cover.out -args -ginkgo.v'

docker-build-sample: ## Build docker image with the manager.
	docker build -t ${IMG} .

##@ Build

build: generate fmt vet ## Build manager binary.
	go build -buildvcs=false -o bin/manager .

run: manifests generate fmt vet ## Run a controller from your host.
	go run -buildvcs=false .

docker-build: test bundle ## Build docker image with the manager and CRDs
	docker build -t ${IMG} .

docker-build-raw: ## Build docker image without running tests/bundle dependencies.
	docker build --build-arg CONTROLLER_MAIN=$(CONTROLLER_MAIN) --build-arg TARGETOS=$(TARGETOS) --build-arg TARGETARCH=$(TARGETARCH) --build-arg CGO_ENABLED=0 --build-arg GOEXPERIMENT= -t ${IMG} .

build-service: generate fmt vet ## Build a service-scoped manager binary.
	@[ -n "$(SERVICE)" ] || { echo "SERVICE must be set"; exit 1; }
	@[ -f "cmd/manager/$(SERVICE)/main.go" ] || { echo "SERVICE '$(SERVICE)' does not have a dedicated controller entrypoint under cmd/manager."; exit 1; }
	go build -buildvcs=false -o bin/manager-$(SERVICE) ./cmd/manager/$(SERVICE)

docker-build-service: ## Build docker image for SERVICE using service manager entrypoint.
	@[ -n "$(SERVICE)" ] || { echo "SERVICE must be set"; exit 1; }
	@[ -f "cmd/manager/$(SERVICE)/main.go" ] || { echo "SERVICE '$(SERVICE)' does not have a dedicated controller entrypoint under cmd/manager."; exit 1; }
	docker build --build-arg CONTROLLER_MAIN=./cmd/manager/$(SERVICE) --build-arg TARGETOS=$(TARGETOS) --build-arg TARGETARCH=$(TARGETARCH) --build-arg CGO_ENABLED=0 --build-arg GOEXPERIMENT= -t $(SERVICE_IMG) .

docker-build-service-raw: ## Build service image without running tests/bundle dependencies.
	@[ -n "$(SERVICE)" ] || { echo "SERVICE must be set"; exit 1; }
	@[ -f "cmd/manager/$(SERVICE)/main.go" ] || { echo "SERVICE '$(SERVICE)' does not have a dedicated controller entrypoint under cmd/manager."; exit 1; }
	docker build --build-arg CONTROLLER_MAIN=./cmd/manager/$(SERVICE) --build-arg TARGETOS=$(TARGETOS) --build-arg TARGETARCH=$(TARGETARCH) --build-arg CGO_ENABLED=0 --build-arg GOEXPERIMENT= -t $(SERVICE_IMG) .

docker-push: ## Push docker image with the manager.
	docker push ${IMG}

##@ Packages

packages: ## List configured package groups under packages/.
	@find $(PACKAGES_DIR) -mindepth 1 -maxdepth 1 -type d -exec basename {} \; | sort

package-generate: controller-gen ## Generate CRDs and optional controller RBAC for GROUP into packages/<group>/install/generated.
	@test -f "$(PACKAGE_DIR)/metadata.env" || { echo "Unknown GROUP '$(GROUP)'. See 'make packages'."; exit 1; }
	@CONTROLLER_GEN_RUNNER="$(CONTROLLER_GEN_RUNNER)" CONTROLLER_GEN="$(CONTROLLER_GEN)" "$(PACKAGE_SCRIPT)" generate "$(GROUP)"

package-install: controller-gen kustomize ## Render a single install YAML for GROUP into dist/packages/<group>/install.yaml.
	@test -f "$(PACKAGE_DIR)/metadata.env" || { echo "Unknown GROUP '$(GROUP)'. See 'make packages'."; exit 1; }
	@CONTROLLER_GEN_RUNNER="$(CONTROLLER_GEN_RUNNER)" CONTROLLER_GEN="$(CONTROLLER_GEN)" KUSTOMIZE="$(KUSTOMIZE)" CONTROLLER_IMG="$(CONTROLLER_IMG)" OUT="$(PACKAGE_OUTPUT_DIR)/install.yaml" \
		"$(PACKAGE_SCRIPT)" render "$(GROUP)"

monolith-install: kustomize ## Render the monolithic install YAML into dist/monolith/install.yaml.
	@KUSTOMIZE="$(KUSTOMIZE)" CONTROLLER_IMG="$(CONTROLLER_IMG)" OUT="dist/monolith/install.yaml" \
		"$(BASH)" "$(PWD)/$(MONOLITH_SCRIPT)" render

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

CONTROLLER_GEN_GODEBUG ?= gotypesalias=0
CONTROLLER_GEN_RUNNER ?= $(shell pwd)/hack/with-controller-gen-godebug.sh
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

package-bundle: package-bundle-olm ## Compatibility alias for package-bundle-olm.

package-bundle-olm: controller-gen kustomize operator-sdk ## Generate an OLM bundle for GROUP from packages/<group>/install into bundle/.
	@test -f "$(PACKAGE_DIR)/metadata.env" || { echo "Unknown GROUP '$(GROUP)'. See 'make packages'."; exit 1; }
	@[ -n "$(VERSION)" ] || { echo "VERSION must be set"; exit 1; }
	@CONTROLLER_GEN_RUNNER="$(CONTROLLER_GEN_RUNNER)" CONTROLLER_GEN="$(CONTROLLER_GEN)" KUSTOMIZE="$(KUSTOMIZE)" OPERATOR_SDK="$(OPERATOR_SDK)" CONTROLLER_IMG="$(CONTROLLER_IMG)" VERSION="$(VERSION)" \
		"$(BASH)" "$(PWD)/$(PACKAGE_OLM_SCRIPT)" bundle "$(GROUP)"

publish-service-olm: controller-gen kustomize operator-sdk ## Build/push a per-service controller image, generate the matching bundle, and build/push the bundle image. Use GROUP=<service>.
	@test -f "$(PACKAGE_DIR)/metadata.env" || { echo "Unknown GROUP '$(GROUP)'. See 'make packages'."; exit 1; }
	@test -f "cmd/manager/$(GROUP)/main.go" || { echo "GROUP '$(GROUP)' does not have a dedicated controller entrypoint under cmd/manager."; exit 1; }
	@[ -n "$(PUBLISH_VERSION)" ] || { echo "PUBLISH_VERSION must be set (image tag)"; exit 1; }
	@[ -n "$(PUBLISH_REGISTRY)" ] || { echo "PUBLISH_REGISTRY must be set (e.g. iad.ocir.io/org)"; exit 1; }
	@set -eu; \
		image="$(PUBLISH_REGISTRY)/oci-service-operator-$(GROUP):$(PUBLISH_VERSION)"; \
		bundle_image="$(PUBLISH_REGISTRY)/oci-service-operator-$(GROUP)-bundle:$(PUBLISH_VERSION)"; \
		if [ -n "$(PUBLISH_PLATFORMS)" ]; then \
			echo ">>> Building and pushing $$image for $(PUBLISH_PLATFORMS)"; \
			IMAGE="$$image" CONTROLLER_MAIN="./cmd/manager/$(GROUP)" PLATFORMS="$(PUBLISH_PLATFORMS)" USE_DOCKER_PLATFORM="$(PUBLISH_USE_DOCKER_PLATFORM)" CGO_ENABLED="$(PUBLISH_CGO_ENABLED)" GOEXPERIMENT="$(PUBLISH_GOEXPERIMENT)" "$(BASH)" "$(PWD)/hack/publish-platform-image.sh"; \
		else \
			platform="$(PUBLISH_PLATFORM)"; \
			platform="$$(printf '%s' "$$platform" | tr '_' '/')"; \
			os="$${platform%%/*}"; \
			arch="$${platform##*/}"; \
			echo ">>> Building $$image for $$platform"; \
			docker build --build-arg CONTROLLER_MAIN=./cmd/manager/$(GROUP) --build-arg TARGETOS="$$os" --build-arg TARGETARCH="$$arch" --build-arg CGO_ENABLED=$(PUBLISH_CGO_ENABLED) --build-arg GOEXPERIMENT=$(PUBLISH_GOEXPERIMENT) -t "$$image" .; \
			echo ">>> Pushing $$image"; \
			docker push "$$image"; \
		fi; \
		echo ">>> Generating $(GROUP) bundle for $$bundle_image"; \
		$(MAKE) --no-print-directory package-bundle-olm GROUP="$(GROUP)" CONTROLLER_IMG="$$image" VERSION="$(PUBLISH_VERSION)"; \
		echo ">>> Building $$bundle_image"; \
	$(MAKE) --no-print-directory bundle-build BUNDLE_IMG="$$bundle_image"; \
	echo ">>> Pushing $$bundle_image"; \
	$(MAKE) --no-print-directory bundle-push BUNDLE_IMG="$$bundle_image"; \
	echo ">>> Bundle image ready: $$bundle_image"

publish-monolith-olm: operator-sdk ## Build/push the monolithic controller image, generate the matching bundle, and build/push the bundle image.
	@[ -n "$(PUBLISH_VERSION)" ] || { echo "PUBLISH_VERSION must be set (image tag)"; exit 1; }
	@[ -n "$(PUBLISH_REGISTRY)" ] || { echo "PUBLISH_REGISTRY must be set (e.g. iad.ocir.io/org)"; exit 1; }
	@set -eu; \
	image="$(PUBLISH_REGISTRY)/oci-service-operator:$(PUBLISH_VERSION)"; \
	bundle_image="$(MONOLITH_OLM_BUNDLE_IMG)"; \
	bundle_version="$(PUBLISH_VERSION)"; \
	bundle_version="$${bundle_version#v}"; \
	if [ -z "$$bundle_image" ]; then \
		bundle_image="$(PUBLISH_REGISTRY)/oci-service-operator-bundle:$(PUBLISH_VERSION)"; \
	fi; \
	echo ">>> Building $$image"; \
	$(MAKE) --no-print-directory docker-build-raw IMG="$$image" CONTROLLER_MAIN="."; \
	echo ">>> Pushing $$image"; \
	$(MAKE) --no-print-directory docker-push IMG="$$image"; \
	echo ">>> Generating monolith bundle version $$bundle_version for $$bundle_image"; \
	$(MAKE) --no-print-directory bundle IMG="$$image" VERSION="$$bundle_version" BUNDLE_IMG="$$bundle_image"; \
	echo ">>> Building $$bundle_image"; \
	$(MAKE) --no-print-directory bundle-build BUNDLE_IMG="$$bundle_image"; \
	echo ">>> Pushing $$bundle_image"; \
	$(MAKE) --no-print-directory bundle-push BUNDLE_IMG="$$bundle_image"; \
	echo ">>> Bundle image ready: $$bundle_image"

publish-monolith: kustomize ## Build, push, and render the monolithic controller manifest.
	@[ -n "$(PUBLISH_VERSION)" ] || { echo "PUBLISH_VERSION must be set (image tag)"; exit 1; }
	@[ -n "$(PUBLISH_REGISTRY)" ] || { echo "PUBLISH_REGISTRY must be set (e.g. iad.ocir.io/org)"; exit 1; }
	@set -eu; \
	image="$(PUBLISH_REGISTRY)/oci-service-operator:$(PUBLISH_VERSION)"; \
	echo ">>> Building $$image"; \
	$(MAKE) --no-print-directory docker-build-raw IMG="$$image" CONTROLLER_MAIN="."; \
	echo ">>> Pushing $$image"; \
	$(MAKE) --no-print-directory docker-push IMG="$$image"; \
	output="dist/monolith/install-$(PUBLISH_VERSION).yaml"; \
	KUSTOMIZE="$(KUSTOMIZE)" CONTROLLER_IMG="$$image" OUT="$$output" "$(BASH)" "$(PWD)/$(MONOLITH_SCRIPT)" render; \
	echo ">>> Wrote $$output"

install-service-olm: operator-sdk ## Install a per-service bundle into a cluster with OLM. Use GROUP=<service>.
	@test -f "$(PACKAGE_DIR)/metadata.env" || { echo "Unknown GROUP '$(GROUP)'. See 'make packages'."; exit 1; }
	@[ -n "$(PUBLISH_VERSION)" ] || { echo "PUBLISH_VERSION must be set (bundle tag)"; exit 1; }
	@[ -n "$(PUBLISH_REGISTRY)" ] || { echo "PUBLISH_REGISTRY must be set (e.g. iad.ocir.io/org)"; exit 1; }
	$(OPERATOR_SDK) run bundle "$(PUBLISH_REGISTRY)/oci-service-operator-$(GROUP)-bundle:$(PUBLISH_VERSION)"

install-monolith-olm: operator-sdk ## Install the monolithic controller bundle into a cluster with OLM.
	@set -eu; \
	bundle_image="$(MONOLITH_OLM_BUNDLE_IMG)"; \
	if [ -z "$$bundle_image" ]; then \
		[ -n "$(PUBLISH_VERSION)" ] || { echo "Set MONOLITH_OLM_BUNDLE_IMG or PUBLISH_VERSION"; exit 1; }; \
		[ -n "$(PUBLISH_REGISTRY)" ] || { echo "Set MONOLITH_OLM_BUNDLE_IMG or PUBLISH_REGISTRY"; exit 1; }; \
		bundle_image="$(PUBLISH_REGISTRY)/oci-service-operator-bundle:$(PUBLISH_VERSION)"; \
	fi; \
	$(OPERATOR_SDK) run bundle "$$bundle_image"

upgrade-service-olm: operator-sdk ## Upgrade a per-service bundle in a cluster with OLM. Use GROUP=<service>.
	@test -f "$(PACKAGE_DIR)/metadata.env" || { echo "Unknown GROUP '$(GROUP)'. See 'make packages'."; exit 1; }
	@[ -n "$(PUBLISH_VERSION)" ] || { echo "PUBLISH_VERSION must be set (bundle tag)"; exit 1; }
	@[ -n "$(PUBLISH_REGISTRY)" ] || { echo "PUBLISH_REGISTRY must be set (e.g. iad.ocir.io/org)"; exit 1; }
	$(OPERATOR_SDK) run bundle-upgrade "$(PUBLISH_REGISTRY)/oci-service-operator-$(GROUP)-bundle:$(PUBLISH_VERSION)"

upgrade-monolith-olm: operator-sdk ## Upgrade the monolithic controller bundle in a cluster with OLM.
	@set -eu; \
	bundle_image="$(MONOLITH_OLM_BUNDLE_IMG)"; \
	if [ -z "$$bundle_image" ]; then \
		[ -n "$(PUBLISH_VERSION)" ] || { echo "Set MONOLITH_OLM_BUNDLE_IMG or PUBLISH_VERSION"; exit 1; }; \
		[ -n "$(PUBLISH_REGISTRY)" ] || { echo "Set MONOLITH_OLM_BUNDLE_IMG or PUBLISH_REGISTRY"; exit 1; }; \
		bundle_image="$(PUBLISH_REGISTRY)/oci-service-operator-bundle:$(PUBLISH_VERSION)"; \
	fi; \
	$(OPERATOR_SDK) run bundle-upgrade "$$bundle_image"

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
	kubectl delete crd dbsystems.mysql.oracle.com &
	kubectl delete crd streams.oci.oracle.com &

delete-operator:
	kubectl delete ns $(OPERATOR_NAMESPACE)

.PHONY: delete-crds-force
delete-crds-force:
	kubectl patch crd/autonomousdatabases.oci.oracle.com -p '{"metadata":{"finalizers":[]}}' --type=merge &
	kubectl patch crd/dbsystems.mysql.oracle.com -p '{"metadata":{"finalizers":[]}}' --type=merge &
	kubectl patch crd/streams.oci.oracle.com -p '{"metadata":{"finalizers":[]}}' --type=merge

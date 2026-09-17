# OSOK Integration And Live E2E

`e2e/e2e-lite-local` creates a lightweight local Kind cluster for OSOK development.

OSOK has two complementary integration paths:

- package-local typed mock tests exercise OCI SDK serialization and
  service-manager reconciliation without cloud credentials;
- live standalone and composite scenarios install the real controller and
  perform create, update, and delete operations against OCI.

Run the deterministic integration suite with:

```bash
make integrationtest
```

Run only the service-manager mock tests, or print their generated-resource
coverage inventory, with:

```bash
make mockintegrationtest
make mock-integration-inventory
```

It is intended to:

- use the local `docker` CLI/runtime from Rancher Desktop
- install `kind` and `operator-sdk` into the repo-local `bin/` directory when missing
- create a Docker-backed Kind cluster
- pull the Kind node image from `mirror.gcr.io` by default
- rewrite Kind node DNS to public resolvers before OLM install
- probe the OLM registry endpoint before invoking `operator-sdk olm install`
- install OLM
- install cert-manager when a selected package manifest requires it
- mount the current OSOK checkout into the Kind node
- mount `~/.oci` into the Kind node

By default it stops at cluster prerequisites only. When `--service <group>` is
set, the same helper will also:

- build a local controller image
- load that image into the Kind cluster
- refresh and render the package install manifest for the selected group
- create or reuse the `ocicredentials` secret in the package namespace
- patch the local-Kind deployment defaults (`imagePullPolicy=IfNotPresent`,
  `useinstanceprincipal=false`)
- wait for the controller rollout
- apply all matching sample manifests for that group
- log per-resource `PASS` or `FAIL`

## Prerequisites

- `docker` from Rancher Desktop
- `kubectl`
- `curl`
- a host OCI directory under `~/.oci` or a custom `OCI_DIR`

The script expects Docker to already be available and uses the active Docker context.

On macOS, the repo checkout and `~/.oci` must be under `$HOME` so the active Docker runtime can expose them to the Kind node mount path used by the script.

## OCI Directory Requirement

The script expects `OCI_DIR` to exist because it mounts that host directory into the Kind node for later follow-on OSOK steps.

## Basic Usage

Bring up a local cluster and install prerequisites:

```bash
./e2e/e2e-lite-local
```

This is the same as:

```bash
./e2e/e2e-lite-local up
```

Check status:

```bash
./e2e/e2e-lite-local status
```

Render the generated Kind config without creating a cluster:

```bash
./e2e/e2e-lite-local render
```

Tear everything down:

```bash
./e2e/e2e-lite-local down
```

Run the full local package flow for one operator group:

```bash
SKIP_OLM=true ./e2e/e2e-lite-local up --service streaming
```

This is also available as an explicit test action:

```bash
SKIP_OLM=true ./e2e/e2e-lite-local test --service streaming
```

Run an explicit create/update/delete lifecycle instead of the package samples:

```bash
export OCI_COMPARTMENT_ID=ocid1.compartment.oc1..example
SKIP_OLM=true ./e2e/e2e-lite-local test \
  --service objectstorage \
  --scenario e2e/scenarios/objectstorage/basic/scenario.yaml
```

The equivalent Make target is `make e2e-live`. Override `E2E_SERVICE` and
`E2E_SCENARIO` for another checked-in scenario.

Run a dependency graph with Chainsaw:

```bash
export OCI_COMPARTMENT_ID=ocid1.compartment.oc1..example
SKIP_OLM=true ./e2e/e2e-lite-local test \
  --service core-network \
  --composite e2e/composite/core-network/basic
```

The equivalent target is:

```bash
make e2e-composite \
  E2E_SERVICE=core-network \
  E2E_COMPOSITE=e2e/composite/core-network/basic
```

`e2e/chainsaw` installs the pinned Chainsaw release into the ignored `bin/`
directory and verifies the release checksum. It is also used by
`make e2e-composite-lint`, which validates every checked-in composite test
without contacting Kubernetes or OCI.

The direct package-install path does not require OLM, so `SKIP_OLM=true` is the
recommended local setting for `--service` runs.

## Common Overrides

Use a different cluster name:

```bash
CLUSTER_NAME=osok-dev ./e2e/e2e-lite-local
```

Force cluster recreation:

```bash
RECREATE_CLUSTER=true ./e2e/e2e-lite-local
```

Skip OLM installation:

```bash
SKIP_OLM=true ./e2e/e2e-lite-local
```

Pin a specific OLM version:

```bash
OLM_VERSION=v0.28.0 ./e2e/e2e-lite-local
```

The script now defaults `OLM_VERSION` to `v0.28.0`, which matches the newer
`operator-sdk` default instead of relying on older repo-local `operator-sdk`
releases that still resolve `latest`.

Use a different Kind node image:

```bash
KIND_NODE_IMAGE=mirror.gcr.io/kindest/node:v1.34.0 ./e2e/e2e-lite-local
```

Use a Kind node image hosted in OCIR:

```bash
KIND_NODE_IMAGE=iad.ocir.io/mytenancy/osok/kindest-node:v1.35.0-osok ./e2e/e2e-lite-local
```

Use different DNS servers inside the Kind node:

```bash
KIND_NODE_NAMESERVERS="10.0.0.2 1.1.1.1" ./e2e/e2e-lite-local
```

Use a different OLM registry probe endpoint:

```bash
OLM_REGISTRY_PROBE_URL=https://quay.example.internal/v2/ ./e2e/e2e-lite-local
```

This override changes the preflight endpoint only. If `operator-sdk olm install`
still references `quay.io`, you also need mirrored OLM images or mirrored OLM
install manifests for the actual image pulls to succeed.

Use a different OCI profile for the generated `ocicredentials` secret:

```bash
SKIP_OLM=true ./e2e/e2e-lite-local test --service streaming --oci-profile WORKLOAD
```

Reuse an already-built image and skip loading it into Kind:

```bash
SKIP_OLM=true ./e2e/e2e-lite-local test --service streaming \
  --controller-img osok-streaming:dev \
  --skip-build \
  --skip-load
```

Point the service suite at a custom sample directory:

```bash
SKIP_OLM=true ./e2e/e2e-lite-local test --service streaming \
  --sample-dir /path/to/manifests
```

The helper also accepts trailing `NAME=value` arguments, so the following is
equivalent to exporting the variable first:

```bash
./e2e/e2e-lite-local up SKIP_OLM=true
```

## Important Paths

- script: `e2e/e2e-lite-local`
- repo-local tools: `bin/kind`, `bin/operator-sdk`
- generated artifacts: `e2e/.e2e-lite-local-<cluster-name>/`
- kubeconfig: `e2e/.e2e-lite-local-<cluster-name>/kubeconfig`
- kind config: `e2e/.e2e-lite-local-<cluster-name>/kind-config.yaml`
- service results: `e2e/.e2e-lite-local-<cluster-name>/results/<group>.tsv`
- lifecycle evidence: `e2e/.e2e-lite-local-<cluster-name>/scenarios/<name>/result.json`
- composite evidence: `e2e/.e2e-lite-local-<cluster-name>/composite/<name>/result.json`
- controller logs: `e2e/.e2e-lite-local-<cluster-name>/logs/<group>-controller.log`

## What The Script Does

The script:

- creates the Kind cluster
- pre-pulls the Kind node image and passes it explicitly to `kind create cluster`
- rewrites `/etc/resolv.conf` inside each Kind node using `KIND_NODE_NAMESERVERS`
- checks the OLM registry endpoint from inside the Kind node before attempting OLM install
- installs OLM unless `SKIP_OLM=true`
- mounts the current checkout into the Kind node
- mounts the host OCI directory into the Kind node
- exports a dedicated kubeconfig for the cluster
- when `--service` is set, builds and loads a local controller image
- when `--service` is set, renders and applies `dist/packages/<group>/install.yaml`
- when `--service` is set, ensures the `ocicredentials` secret exists in the
  package namespace
- when `--service` is set, patches the prefixed package secret
  `oci-service-operator-<group>-osokconfig` to disable instance principals for
  local runs
- when `--service` is set, applies the local image and credentials in one
  controller Deployment revision and waits for it
- when `--service` is set, applies matching sample manifests and records
  per-resource `PASS`/`FAIL`
- when `--scenario` is set, renders its variables, performs create/update/delete,
  waits for status convergence, confirms deletion, and writes JSON evidence
- when `--composite` is set, runs an explicit Chainsaw resource graph, passes
  observed OCI identifiers to downstream CRs, and writes a JSON report

## What The Script Does Not Do

The script does not:

- rewrite placeholder OCI values inside sample manifests
- infer dependencies from arbitrary manifests; composite dependencies are
  declared explicitly in their Chainsaw test
- turn CRD-only resources into controller-backed resources
- make admission-only resources appear reconciled

## Service Suite Notes

- Sample discovery defaults to `config/samples/<group>_*.yaml`.
- The service run creates or refreshes `ocicredentials` from `~/.oci/config`
  and the selected `--oci-profile`. If a secret already exists, it is reused
  unless `--refresh-oci-secret` is set.
- Resources with a matching controller under `controllers/<group>/` are treated
  as controller-backed and must become active to log `PASS`.
- Resources with no matching controller are logged as `PASS` only when the
  Kubernetes object was admitted successfully. The result line is marked
  `admitted-only` to make that distinction explicit.
- The `Stream` resource gets one extra check: the result includes whether the
  generated endpoint secret named after the resource is present.

## Composite Scenario Contract

Composite suites live under `e2e/composite/<service>/<suite>/` and use the
explicit Chainsaw `chainsaw-test.yaml` format. Each suite keeps all graph
operations in one step so an upstream `kubectl get` output binding can be used
by later resource templates. Chainsaw owns reverse-order cleanup; OSOK
finalizers continue to own OCI deletion confirmation.

The first suites are built from standalone resources already proven against
live OCI:

| Suite | Graph | Additional operator inputs |
| --- | --- | --- |
| `core-network/basic` | VCN -> Subnet, Internet Gateway, Network Security Group | `OCI_COMPARTMENT_ID` |
| `containerengine/cluster-node-pool` | OKE Cluster -> IMDSv2 NodePool | `OCI_COMPARTMENT_ID`, `OCI_VCN_ID`, `OCI_CLUSTER_SUBNET_ID`, `OCI_SERVICE_LB_SUBNET_ID`, `OCI_NODE_SUBNET_ID`, `OCI_POD_SUBNET_ID`, `OCI_KUBERNETES_VERSION`, `OCI_COMPUTE_SHAPE`, `OCI_IMAGE_ID`, and `OCI_AVAILABILITY_DOMAIN` |
| `apigateway/gateway-deployment` | API Gateway -> Deployment, including endpoint Secret verification | `OCI_COMPARTMENT_ID`, private `OCI_SUBNET_ID` |
| `filestorage/filesystem-export` | File System + Mount Target -> Export | `OCI_COMPARTMENT_ID`, `OCI_AVAILABILITY_DOMAIN`, private `OCI_SUBNET_ID` |
| `logging/log-group-log` | Log Group -> Custom Log | `OCI_COMPARTMENT_ID` |
| `ons/topic-subscription` | Topic -> Subscription | `OCI_COMPARTMENT_ID`, `OCI_NOTIFICATION_PROTOCOL`, and a confirmation-free `OCI_NOTIFICATION_ENDPOINT` |

The local helper generates `OSOK_E2E_SUFFIX` when it is absent. Composite
files read operator inputs from environment variables and contain no live
OCIDs or credentials. A suite currently targets one package group so its
controller image, CRDs, credentials, logs, and cleanup remain isolated.

## Lifecycle Scenario Contract

Scenarios live under `e2e/scenarios/<service>/<scenario>/` and contain:

```text
scenario.yaml
create.yaml
update.yaml       # optional
dependencies.yaml # optional
```

Manifest placeholders such as `${OCI_COMPARTMENT_ID}` are resolved from the
environment. `OCI_TENANCY_ID` and `OCI_REGION` default from the selected OCI
profile. When OCI CLI access is available, `OCI_AVAILABILITY_DOMAIN` defaults
to the first AD visible from the target compartment. `OSOK_E2E_SUFFIX` is generated when it is not provided, and
`OSOK_E2E_NAMESPACE` resolves to the scenario namespace. Any other unset
variable is a hard authoring error.

`OSOK_E2E_ID` is derived automatically from `OSOK_E2E_SUFFIX` by retaining
only letters and numbers. Use it when an OCI resource name or DDL identifier
cannot contain the punctuation valid in a Kubernetes resource name.

Readiness can require condition types, lifecycle states, an OCI identifier, and
field equality such as `spec.displayName == status.displayName`. Field equality
prevents an update from passing against status left over from the create phase.
`relatedObjects` can also require companion Kubernetes objects and selected
Secret data keys after create/update, plus their deletion with the primary CR.
The runner cleans up the primary CR after success or failure and deletes
scenario-owned dependencies in reverse order.

The checked-in live reference scenarios cover several distinct controller
behaviors:

| Scenario | Additional operator inputs | Behavior exercised |
| --- | --- | --- |
| Object Storage Bucket | none | basic lifecycle and metadata update |
| Streaming Stream | none | lifecycle plus endpoint Secret |
| ADM KnowledgeBase | none | service work requests |
| Cluster Placement Group | none; availability domain is discovered | independent work requests |
| Core VCN | none | foundational network lifecycle |
| Core DRG | none | dynamic-routing gateway lifecycle and metadata update |
| Core Subnet | `OCI_VCN_ID` | resource with an existing-network prerequisite |
| Core Network Security Group | `OCI_VCN_ID` | network security lifecycle and metadata update |
| Core Internet Gateway | `OCI_VCN_ID` for a VCN that can accept an Internet Gateway | gateway enablement and network lifecycle |
| Core NAT Gateway | `OCI_VCN_ID` | NAT gateway lifecycle, traffic blocking, and metadata update |
| Core Route Table | `OCI_VCN_ID` | empty route-table lifecycle and metadata update |
| Core Security List | `OCI_VCN_ID` | network-rule lifecycle and metadata update |
| Core Service Gateway | `OCI_VCN_ID` | empty-service gateway lifecycle, traffic blocking, and metadata update |
| OKE Cluster | `OCI_VCN_ID`, `OCI_SUBNET_ID`, and a region-supported `OCI_KUBERNETES_VERSION` | private Basic control-plane lifecycle, work requests, and metadata update |
| OKE NodePool | `OCI_CLUSTER_ID`, `OCI_SUBNET_ID`, `OCI_POD_SUBNET_ID`, `OCI_IMAGE_ID`, `OCI_COMPUTE_SHAPE`, a cluster-compatible `OCI_KUBERNETES_VERSION`, and `OCI_AVAILABILITY_DOMAIN` for delegated cross-tenancy sessions | one-node VCN-native IMDSv2 pool lifecycle, work requests, and metadata update |
| MySQL DBSystem | private `OCI_SUBNET_ID`, `OCI_MYSQL_SHAPE`, `OCI_MYSQL_ADMIN_PASSWORD`, and `OCI_AVAILABILITY_DOMAIN` for delegated cross-tenancy sessions | database lifecycle, secret-backed admin credentials, metadata update, and endpoint Secret |
| Network Load Balancer | private `OCI_SUBNET_ID` | work-request lifecycle, private address allocation, and metadata update |
| Logging LogGroup | none | work-request lifecycle, description update, and metadata convergence |
| Logging Custom Log | `OCI_LOG_GROUP_ID` | parent-scoped lifecycle, retention update, and metadata convergence |
| Bastion | private `OCI_SUBNET_ID` | work-request lifecycle, CIDR/TTL update, and private endpoint allocation |
| Redis Cluster | private `OCI_SUBNET_ID` and `OCI_REDIS_SOFTWARE_VERSION` | two-node cache lifecycle, work requests, and metadata update |
| PostgreSQL DBSystem | private `OCI_SUBNET_ID`, `OCI_PSQL_SHAPE`, `OCI_PSQL_DB_VERSION`, and `OCI_PSQL_ADMIN_PASSWORD` | database lifecycle, Secret-backed admin credentials, and metadata update |
| API Gateway | private `OCI_SUBNET_ID` | private gateway lifecycle, metadata update, and endpoint Secret |
| API Gateway Deployment | `OCI_APIGATEWAY_ID` | parent-scoped stock-response deployment lifecycle and route update |
| Container Repository | none | empty public repository lifecycle, README update, and metadata convergence |
| Events Rule | `OCI_NOTIFICATION_TOPIC_ID` | disabled ONS action lifecycle, filter update, and metadata convergence |
| Certificate Management CA Bundle | none | public PEM bundle lifecycle, description update, and metadata convergence |
| Autoscaling Configuration | `OCI_INSTANCE_POOL_ID` | disabled scheduled autoscaling lifecycle, cooldown update, and metadata convergence |
| Queue | none | work requests plus endpoint Secret |
| NoSQL Table | none | eventual-consistency lifecycle and sequenced updates |
| Core Instance | `OCI_SUBNET_ID`, `OCI_IMAGE_ID`, and `OCI_COMPUTE_SHAPE` | compute lifecycle and in-place update |
| File Storage File System | none; availability domain is discovered | foundational storage lifecycle |
| File Storage Mount Target | private `OCI_SUBNET_ID`; availability domain is discovered | network-attached file-storage endpoint lifecycle and metadata update |
| File Storage Export | `OCI_FILE_STORAGE_EXPORT_SET_ID` and `OCI_FILE_SYSTEM_ID` | parent-scoped NFS export lifecycle and access-policy update |
| Load Balancer | `OCI_SUBNET_ID` | private flexible Load Balancer lifecycle |
| Notifications Topic | none | notification lifecycle and metadata update |
| Notifications Subscription | `OCI_NOTIFICATION_TOPIC_ID`, `OCI_NOTIFICATION_PROTOCOL`, and `OCI_NOTIFICATION_ENDPOINT`; use a confirmation-free endpoint | endpoint-scoped subscription lifecycle and metadata update |
| Email Domain | none | email-domain lifecycle, description update, and metadata convergence |
| Email Sender | `OCI_EMAIL_SENDER_ADDRESS` | approved-sender lifecycle and metadata update |
| Monitoring Alarm | `OCI_NOTIFICATION_TOPIC_ID` | disabled alarm lifecycle and query update |

The Instance and OKE NodePool scenarios disable legacy IMDS endpoints, which
is required in tenancies that enforce IMDSv2. The OKE scenarios use long
timeouts because create and delete are asynchronous, potentially expensive
operations. Do not commit live OCIDs into these manifests; the per-operator
inputs remain environment values. The MySQL scenario creates a temporary
Kubernetes admin Secret from `OCI_MYSQL_ADMIN_PASSWORD`; both that Secret and
the generated endpoint Secret are removed during scenario cleanup.

See [Service-manager mock integration](../docs/contributor/service-manager-mock-integration.md)
for typed fixture authoring and test-selection guidance.

## Notes

- The Kind provider is forced to `docker` through `KIND_EXPERIMENTAL_PROVIDER=docker`.
- The default Kind node image is `mirror.gcr.io/kindest/node:v1.35.0`. Override that with `KIND_NODE_IMAGE` if you need a different mirror or version.
- The default node DNS override is `1.1.1.1 8.8.8.8`. Override that with `KIND_NODE_NAMESERVERS` if your environment requires different upstream resolvers.
- The default OLM version passed to `operator-sdk olm install` is `v0.28.0`.
- The default OLM registry probe URL is `https://quay.io/v2/`.
- The preflight probes the configured endpoint from inside the Kind node, then retries with `curl -4` before failing. This avoids false negatives from broken dual-stack registry paths while still failing fast on real egress issues.
- The host OCI directory is mounted into the Kind node at `/var/oci-host` by default. Override that with `MOUNTED_OCI_PATH` if needed.
- The mounted checkout is there so follow-on steps can pull or build OSOK assets against the same source tree after the cluster bootstrap is done.

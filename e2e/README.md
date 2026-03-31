# E2E Local Bootstrap

`e2e/e2e-lite-local` creates a lightweight local Kind cluster for OSOK development.

This README documents `e2e/e2e-lite-local` only.

It is intended to:

- use the local `docker` CLI/runtime from Rancher Desktop
- install `kind` and `operator-sdk` into the repo-local `bin/` directory when missing
- create a Docker-backed Kind cluster
- pull the Kind node image from `mirror.gcr.io` by default
- rewrite Kind node DNS to public resolvers before OLM install
- probe the OLM registry endpoint before invoking `operator-sdk olm install`
- install OLM
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
- when `--service` is set, restarts and waits for the controller deployment
- when `--service` is set, applies matching sample manifests and records
  per-resource `PASS`/`FAIL`

## What The Script Does Not Do

The script does not:

- rewrite placeholder OCI values inside sample manifests
- infer resource dependency order beyond the discovered manifest list
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

## Notes

- The Kind provider is forced to `docker` through `KIND_EXPERIMENTAL_PROVIDER=docker`.
- The default Kind node image is `mirror.gcr.io/kindest/node:v1.35.0`. Override that with `KIND_NODE_IMAGE` if you need a different mirror or version.
- The default node DNS override is `1.1.1.1 8.8.8.8`. Override that with `KIND_NODE_NAMESERVERS` if your environment requires different upstream resolvers.
- The default OLM version passed to `operator-sdk olm install` is `v0.28.0`.
- The default OLM registry probe URL is `https://quay.io/v2/`.
- The preflight probes the configured endpoint from inside the Kind node, then retries with `curl -4` before failing. This avoids false negatives from broken dual-stack registry paths while still failing fast on real egress issues.
- The host OCI directory is mounted into the Kind node at `/var/oci-host` by default. Override that with `MOUNTED_OCI_PATH` if needed.
- The mounted checkout is there so follow-on steps can pull or build OSOK assets against the same source tree after the cluster bootstrap is done.

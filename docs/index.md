# OCI Service Operator for Kubernetes

OCI Service Operator for Kubernetes (OSOK) makes it easier to create, manage,
and connect to Oracle Cloud Infrastructure (OCI) resources from Kubernetes by
using Kubernetes APIs and package-scoped controllers instead of direct OCI CLI
or service-API workflows.

OSOK is built on the Operator Framework and `controller-runtime`, and the docs
here are organized around the current published package bundles, generated API
surface, and resource-specific guidance.

> **Important:** Start in a test or non-production OCI and Kubernetes
> environment first.
>
> Validate authentication, IAM policy scope, create and delete behavior,
> finalizers, and service-specific limits before promoting any package bundle
> to production.

## Start Here

- Use [Installation](installation.md) to install a published package bundle and
  configure authentication.
- Use [Quick start with KRO](user-guide.md) for the end-to-end MySQL example
  that assumes the installation prerequisites are already complete and adds kro
  for the example workflow.
- Use [Supported Resources](reference/index.md) to browse the generated catalog
  of shipped packages, resource kinds, samples, and API entry points.
- Use [API Reference](reference/api/index.md) for generated group/version pages
  and CRD-derived field documentation.
- Use [Contributor Docs](contributor/index.md) for generator, validation, and
  docs-pipeline details.

## What You Will Find Here

- Published package install and authentication paths.
- Generated resource and API reference surfaces derived from checked-in CRDs,
  samples, and release metadata.
- Resource-specific guides for the currently documented services.
- Contributor-oriented references for regeneration, validation, and docs
  maintenance.

## Documentation Paths

### Get Started

Start with the core user flow:

- [Installation](installation.md)
- [Quick start with KRO](user-guide.md)
- [Supported Resources](reference/index.md)
- [API Reference](reference/api/index.md)

### Resource Guides

Move into resource-specific walkthroughs and operating guidance:

- [Autonomous Database](adb.md)
- [MySQL DB Systems](mysql.md)
- [Queue](queue.md)
- [Streaming](oss.md)
- [Troubleshooting](TROUBLESHOOT.md)

### Contributor Docs

Maintainer-oriented generator and validation references live under
[Contributor Docs](contributor/index.md).

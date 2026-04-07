# Services

This page no longer carries the full hand-maintained package and version
inventory. The canonical customer-facing catalog is generated from checked-in
package metadata, release manifests, sample inventory, and CRD schemas.

Use these generated surfaces instead:

- [Supported Resources](reference/index.md) for released packages, exposed
  kinds, sample links, and API entry points.
- [API Reference](reference/api/index.md) for group/version pages and field
  tables generated from checked-in CRD schemas.
- [User Guide](user-guide.md) for the primary quickstart before you browse the
  broader catalog.

If you are maintaining the docs pipeline rather than consuming the catalog,
start in [Contributor Docs](contributor/index.md), treat
`internal/generator/config/services.yaml` as the rollout source of truth, and
use `make docs-generate` plus `make docs-verify` when the checked-in docs
inputs change.

# Contributor Docs

This section is for maintainers and contributors working on OSOK generation,
validation, and repository-owned documentation flows.

## Maintainer References

- [Access Governance Control Plane Onboarding Audit](accessgovernancecp-onboarding-audit.md)
- [ADM Onboarding Audit](adm-onboarding-audit.md)
- [AI Language Onboarding Audit](ailanguage-onboarding-audit.md)
- [AI Vision Onboarding Audit](aivision-onboarding-audit.md)
- [Analytics Onboarding Audit](analytics-onboarding-audit.md)
- [APM Control Plane Onboarding Audit](apmcontrolplane-onboarding-audit.md)
- [API Error Coverage Registry](api-error-coverage-registry.md)
- [API Platform Onboarding Audit](apiplatform-onboarding-audit.md)
- [Budget Onboarding Audit](budget-onboarding-audit.md)
- [Cluster Placement Groups Onboarding Audit](clusterplacementgroups-onboarding-audit.md)
- [Data Labeling Service Onboarding Audit](datalabelingservice-onboarding-audit.md)
- [Database Migration Onboarding Audit](databasemigration-onboarding-audit.md)
- [Generator Contract](../api-generator-contract.md)
- [Enabled Resource Async Strategy Audit](async-strategy-audit.md)
- [Runtime Hook Adoption Audit](runtime-hook-adoption-audit.md)
- [Shared Async Status Contract](shared-async-contract.md)
- [Validator Guide](../validator-guide.md)
- [GitHub Pages Handoff](github-pages-handoff.md)

## Local Docs Commands

- `make docs-generate`
- `make docs-build`
- `make docs-serve`
- `make docs-verify`

Phase 1 keeps missing public description coverage as warnings. When the public
spec-field backlog is resolved, set
`DOCS_VERIFY_STRICT_PUBLIC_DESCRIPTIONS=true` in CI to promote those warnings to
hard failures.

The main customer quickstart remains [Quick start with KRO](../user-guide.md),
which now sits after [Installation](../installation.md) in the customer entry
path.

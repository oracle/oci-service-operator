---
schemaVersion: 1
surface: repo-authored-semantics
service: datasafe
slug: alertpolicyrule
gaps:
- category: seed-corpus
  status: open
  stopCondition: Close when the placeholder provider import is replaced by reviewed
    provider facts or the resource is explicitly promoted as an SDK-owned formal contract.
---

# Logic Gaps

This scaffold row tracks the published AlertPolicyRule API shape and reviewed
runtime lifecycle. Provider facts remain placeholders until a scoped formal
import is available; complete that import before promotion.

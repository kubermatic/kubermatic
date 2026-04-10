---
name: Testing Issue
about: Track release testing for a KKP PR that must be validated during a minor release cycle
title: "<name> - Test Release <minor>"
---

<!--
Use this template for PR-specific release testing in KKP.

Document the change in a way that allows release testing to be executed without reopening the PR for basic context.

This ticket should capture:
- what behavior changed
- what environment or state is required
- how the change is validated
- what outcome determines pass, fail, or blocked

-->

### Summary

<!-- Summarize what changed and why it needs release testing. Focus on the observable behavior and testing relevance, not only implementation details. -->



### Release tracking

<!-- Required for release tracking. -->

- Release minor: <!-- e.g. 2.31 -->
- GitHub milestone: <!-- the milestone this release is tracked under -->
- Parent release testing epic: <!-- link to the umbrella issue that tracks all testing for this release -->
- Related pull request(s):
- Related issue(s):
- Related documentation:

### Scope

<!--
List the concrete KKP surface affected by this change.
Use exact architecture components, not broad categories. Add a short qualifier when needed to clarify what part of the surface changed.
-->

- Affected KKP surface(s) (controllers, APIs, CRDs, manifests, or charts):
<!--
Examples:
- `Cluster` resource reconciliation in `kkp-kubernetes-controller` within `seed-controller-manager`
- `Seed` admission validation in `kubermatic-webhook`
- `ApplicationInstallation` reconciliation in `kkp-app-installation-controller` within `user-cluster-controller-manager`
-->

### Prerequisites

<!--
List only what must already exist or be enabled before testing starts.
Include configuration, feature gates, required objects, existing cluster state, or external dependencies only when they affect reproducibility.
-->

- Required preconditions, configuration, or existing state:
- Required objects, manifests, secrets, or external dependencies:

### Test environment

<!-- Provide the environment/configuration needed for someone else to repeat the test. -->

- Environment: <!-- e.g. QA, local -->
- Edition (CE or EE):
- Topology (shared or separate master/seed):
- Starting state: <!-- e.g. fresh install, existing installation, upgrade from <version>, migration from <state> -->
- Provider / datacenter:
- Kubernetes version(s):
- Operating system(s):

### Relevant manifests or API objects (if applicable)

<!--
Include only the snippets that are useful for reproducing the test.
-->

<details>

```yaml
# paste manifest snippet here
```

</details>


### Validation steps

<!--
List the validation steps in execution order. For each step, state the action, expected result, and what to inspect when it is not obvious.
Include only the checks that matter for this PR. Group the steps under `Positive path` and `Negative / edge cases`.
Include permission, upgrade, rollback, cleanup, compatibility, or failure-handling checks when they are relevant.

Example:
Positive path:
1. Action: Update the relevant `Cluster` or `Seed` object with the new configuration.
   Expected: the change is accepted, reconciliation completes, and the affected resources reach the expected state without repeated errors.
   Inspect: object conditions, related controller logs, and the affected rendered resources.

Negative / edge cases:
1. Action: Apply an invalid or unsupported configuration.
   Expected: validation rejects the change or reconciliation reports the expected failure without affecting existing healthy resources.
   Inspect: webhook response, events, and controller logs.
-->

Positive path:
<!-- Required: add at least one numbered step. -->
1. Action:
   Expected:
   Inspect:
<!-- Add more numbered steps as needed. -->

Negative / edge cases (if applicable):
1. Action:
   Expected:
   Inspect:
<!-- Add more numbered steps as needed. -->

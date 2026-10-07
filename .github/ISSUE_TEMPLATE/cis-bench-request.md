---
name: CIS Benchmark Request
about: Request a CIS Kubernetes Benchmark conformance run against a KKP release
title: "CIS Benchmark: KKP <kkp-version> / k8s <k8s-version>"
labels: cis-bench-request
---

<!--
This template requests a CIS Kubernetes Benchmark run via the cis-bench.yml
workflow. Once the issue is created with the `cis-bench-request` label, the
workflow (defined in .github/workflows/cis-bench.yml) will:

  1. Post an acknowledgement comment here with the parsed parameters.
  2. Run the CIS conformance suite in kubermatic/conformance-ee against a
     KKP seed for each requested Kubernetes version.
  3. Comment per-k8s status (✅ pass / ❌ N failure(s)) back on this issue.
  4. Open a follow-up failure-tracking issue if any CIS control returned Fail.
  5. Open a PR in kubermatic/docs.

Keep the two fields below on their own lines with the exact key names —
they are parsed by the workflow. Values are semver strings; comma-separate
multiple Kubernetes versions.
-->

## CIS Benchmark Request

kkp-version: 2.30.0
k8s-versions: 1.34.5

### Context

<!--
Optional. Note anything the reviewer should know: reason for the run
(release gate, ad-hoc audit, follow-up on a previous failure), the seed
where it should execute, or the expected outcome (baseline pass, targeted
control fix, etc.).
-->

### Related

<!--
Optional. Link tracking issues, PRs, or previous CIS bench runs.
-->

# e2e pipeline test candidates: PRs merged between v2.30.0 and main

Date: 2026-07-21
Range: `v2.30.0..main` (main tip `f466b475e`)
Total merged PRs in range: 186

Purpose: identify which merged PRs are good candidates for a new feature test in
the shared e2e pipeline (`pkg/test/e2e/pipeline`), using PR #16131
(UserProjectBinding pending-delete, already implemented) as the reference bar.

## What makes a good pipeline-test candidate

The pipeline slice attaches to a running KKP seed and asserts reconciled state via
a controller-runtime client. A strong candidate is:

1. Behavioral and observable through the K8s API (CRs, RBAC, deployments,
   configmaps, netpol, webhook admit/reject), not a refactor/log/dep change.
2. Assertable by creating inputs and observing outputs against a live seed.
3. Cost, not feasibility, is the axis: seed-only is cheapest; a `Cluster` object
   (control-plane only) is moderate; a full user cluster (jig + nodes/cloud) is doable
   but expensive. None of these are impossible — see "cluster cost" below.
4. A clear regression it would catch, ideally with a linked bug.
5. Not already covered by an existing e2e job, and complementary to any unit tests
   the PR already added.

## Method

- Enumerated all 186 squash-merge commits in range (each maps 1:1 to a PR via
  the `(#NNNNN)` suffix).
- Classified each by changed-file subsystem signals (master/seed/user-cluster
  controllers, webhook, resources, api, installer, provider, sdk-types, has-unit-test)
  and excluded dep-bumps, docs-only, and chart-only changes.
- Scored the remainder for pipeline-test suitability; pruned title-level noise
  (typo/cleanup/rename, and PRs that ARE e2e tests rather than features to test).
- Deep-assessed the 14 strongest candidates (excluding #16131 already done, and the
  Gateway API PRs #15862/#15628/#15896 already covered by the gateway-api e2e job).

## Excluded en masse (not candidates)

- Dependency bumps (Go modules, images, Helm charts with no code behavior): 48
- Docs-only: 9
- Chart-only (no pkg/ or cmd/ code): 7

These change no runtime behavior an e2e could assert, or are already validated by
build/install jobs.

## Deep candidate assessments

14 PRs deep-assessed against the bar. Consolidated verdict, best first.

### On "cluster cost" — none of these are infeasible

User-cluster e2e testing IS doable in KKP. The `pkg/test/e2e/jig` package provisions a
real Project + Cluster (+ optionally Machines) and hands back a client to the user cluster
(`TestJig.Setup` in `jig/presets.go`, `ClusterJig.Create`, `ClusterJig.ClusterClient`,
`WaitForHealthyControlPlane`). 12 existing e2e packages provision a user cluster via
`jig.NewXCluster` (canal, cilium, cluster-backup, defaultappcatalog, encryption-at-rest,
etcd-launcher, expose-strategy, ipam, konnectivity, mla, opa, usersshkeysagent). So "needs
a user cluster" is a **cost/scope** signal, NOT a feasibility blocker. (Note: nodeport-proxy
and gateway-api reference the jig package but do NOT provision a user cluster — nodeport-proxy's
`jig` mention is an unrelated upstream comment and `npp.Setup` is its own setup; gateway-api
tests the seed-scoped KKP gateway.)

The current `pkg/test/e2e/pipeline` slice is deliberately seed-only (its `TestMain` gates on
`master-controller-manager` and never creates a jig cluster; the design doc scoped user-cluster
features out of the first slice). Any PR below tagged user-cluster is buildable by either
extending the pipeline harness with a jig-backed user-cluster tier, or writing the test in an
existing user-cluster e2e package. Cost tiers, cheapest first:

- **none** — master/seed CRs only, like the UPB reference. No cluster at all.
- **cluster-object** — needs a `Cluster` CR so a controller reconciles, but the observed
  object lives in the `cluster-xxx` namespace *on the seed*. Use `jig.ClusterJig.Create`
  **without a MachineJig**: control plane comes up, no worker nodes (though
  `AllHealthy()` still requires the cloud-provider-infrastructure controller to report Up,
  so there is some cloud-provider activity, just no Node provisioning). Moderate.
- **user-cluster (control-plane)** — needs the user-cluster control plane healthy on the seed
  (e.g. an apiserver flag, a control-plane Deployment). Same jig create, no nodes. Moderate.
- **user-cluster (nodes/in-cluster)** — needs `MachineJig` + a real cloud provider and worker
  nodes, or objects reconciled *inside* the user cluster. Expensive and higher flake surface,
  but still doable via the jig. This is the only genuinely heavy tier.

### Recommended to build (clear the bar)

| PR | Title | Verdict | Cluster cost | Regression | Effort |
|----|-------|---------|--------------|-----------|--------|
| #15691 | propagate defaultValuesBlock on upgrade | STRONG | none | admin value silently overwritten or upstream change never propagated on upgrade (4-branch hash/annotation switch) | low-med |
| #15823 | reduce revisionHistoryLimit to 2 | STRONG (seed CP) | cluster-object | control-plane workloads accumulate unbounded ReplicaSet history; a modifier-list regression drops the limit (#15788) | low-med |
| #15996 | Cilium NodeLocalDNS exclude-local-address | STRONG | user-cluster (in-cluster) | NodeLocalDNS IP treated as host identity, `0.0.0.0/0` egress netpol drops DNS (#15236) | med |
| #15740 | AuthenticationConfiguration for UCs | MODERATE | none (webhook) / cluster-object (full) | `--authentication-config` silently unwired or issuer egress netpol dropped, breaking JWT auth (#15359) | low (webhook slice) / med (full) |

### Detail

**#15691 — defaultValuesBlock propagation (STRONG, best overall).**
`pkg/applicationdefinitions/application_catalog.go`. Reconciler uses a SHA1
`apps.kubermatic.k8c.io/file-default-values-hash` annotation plus an opt-in
`apps.kubermatic.k8c.io/allow-default-values-overwrite` to decide whether to preserve
an admin's `defaultValuesBlock` or overwrite with the embedded file value. The
observed object is a cluster-scoped system `ApplicationDefinition` on the seed/master —
**no cluster object needed**, exactly the UPB reference shape. Four assertable branches:
empty value seeded from file + hash set; admin edit preserved on hash drift; missing
annotation preserves admin value; opt-in annotation lets the file win. No existing e2e
(`pkg/test/e2e/appdefinitions` only counts appdefs). Unit tests added (282 lines) cover
the logic; the e2e verifies the live reconcile wiring the unit test cannot.

**#15996 — Cilium NodeLocalDNS (STRONG, user-cluster test).**
`pkg/controller/seed-controller-manager/cni-application-installation-controller/controller.go`
injects `169.254.20.10/32` into the CNI ApplicationInstallation's Cilium Helm
`extraConfig["exclude-local-address"]` when CNI=Cilium and NodeLocalDNS is enabled
(append/dedup). The controller reconciles the CNI `ApplicationInstallation` into the
**user cluster's `kube-system`** via `userClusterConnectionProvider.GetClient`
(`controller.go:316,324`), not onto the seed — there is no seed-side copy. So asserting
the value needs a user-cluster client (jig `ClusterClient`): this is a user-cluster
(tier-C) test, not a cluster-object one. Requires a `Cluster` CR with a reconciled control
plane plus a user-cluster client. Assert positive (key contains the CIDR), negative
(non-Cilium or NodeLocalDNS disabled → key absent), and merge-with-existing. Real
regression (#15236, broke Web Terminal DNS). Unit tests added (80 lines).

**#15823 — revisionHistoryLimit=2 (STRONG for the seed control-plane path).**
`pkg/resources/reconciling/modifier/revisionhistorylimit.go` wired into the
seed-controller-manager `kubernetes` reconciler (`resources.go`), so control-plane
Deployments/StatefulSets in the `cluster-xxx` namespace get `revisionHistoryLimit: 2`.
Assert `client.Get` + field on apiserver/controller-manager/etcd. Needs a `Cluster` with
a reconciled control plane (cluster-object cost). The user-cluster DaemonSet/coredns path
is user-cluster and out of slice. Static-field check, but it catches a wiring regression
the modifier's unit test cannot. Related #15788.

**#15740 — AuthenticationConfiguration (MODERATE, best of the moderates).**
Adds `Cluster.Spec.AuthenticationConfiguration` (+ Seed/Datacenter, precedence
Cluster>DC>Seed), generates the apiserver AuthenticationConfiguration Secret, adds
`--authentication-config`, and feeds issuer IPs into the `oidc-issuer-allow` NetworkPolicy;
plus seed webhook validation. Two clean angles: **(1) webhook rejection** — set a
Seed/Datacenter `authenticationConfiguration` with an empty `secretName`/`secretKey`
**field** and assert the webhook rejects admission (no cluster object, low effort, high
signal). Note the webhook only validates the reference fields; it does NOT read the Secret
object, so a referenced Secret whose data lacks the key is not caught at admission — that
fails later at reconcile time (`resources.go:1720`, `apiserver/authenticationconfiguration.go:54`),
which is not a clean webhook-rejection assertion. **(2) full** — create a Cluster
referencing a valid Secret and assert the generated apiserver Secret, the
`--authentication-config` flag on the apiserver Deployment, and the netpol egress rule
(cluster-object). Unit tests added. Linked #15359.

### Feasible but costlier or lower-value (deprioritized, not blocked)

All of these are buildable — several need a jig-provisioned user cluster, which KKP e2e
already supports. "Cost" and "marginal value over existing unit tests" are why they rank
below the recommended set, not feasibility.

| PR | Title | Verdict | Cost tier | Note |
|----|-------|---------|-----------|------|
| #16034 | Kyverno policy cleanup during teardown | MODERATE | user-cluster (control-plane, EE) | outer finalizer dance is cluster-object; the motivating generated-resource removal needs Kyverno running in the user cluster. High value (#15523, #15997). |
| #16014 | apiserver NetworkPolicy OIDC fix (KubeLB) | MODERATE | cluster-object | clean netpol assertion; staging a seed LB Service whose VIP matches the OIDC issuer to trigger the path is fiddly; unit tests already cover the peer logic (#15939). |
| #15900 | auto node-exporter + kube-state-metrics on user-mla | MODERATE | user-cluster (in-cluster) | writes ApplicationInstallations *into* the user cluster; assert via `jig.ClusterClient` after `TestJig.Setup` (#12476). |
| #15736 | configurable Kyverno controller resources | MODERATE | user-cluster (control-plane, EE) | set `Spec.Kyverno.*.Resources`, assert the rendered control-plane Deployment; unit tests already pin the rendering. |
| #15601 | anti-affinity for seed nodeport-proxy-envoy | WEAK | none | operator-managed seed Deployment, cheap to read — but only a static label-selector check already covered by unit tests; real co-scheduling bug needs multi-node scheduling. |
| #16024 | seed Prometheus envoy-agent scrape (tunneling) | WEAK | user-cluster (control-plane) | rendered configmap assertion is cluster-object/brittle and duplicates the PR's golden unit test; proving the scrape works needs a tunneling user cluster (#16023). |
| #15848 | remove k8c.io/kubelb dependency | WEAK | external mgmt cluster | Tenant create/delete hits the external KubeLB management cluster (not just a KKP user cluster); only the `defaultTenantSpec` schemaless pass-through is cheaply seed-assertable. Narrow value. |
| #15892 | Helm release recovery for stuck ApplicationInstallations | WEAK | user-cluster (in-cluster) + hard-to-stage | needs a real stuck Helm release in the user cluster, which is hard to force deterministically; thoroughly unit-tested (868 lines) (#14880). |
| #15722 | seed controller cache-sync timeout | NOT-A-CANDIDATE | n/a | genuinely not object-assertable: a startup-timing/scale property, needs thousands of objects to reproduce; no CR/object diff to observe. This is the one true non-candidate. |
| #15958 | KubeVirt ee quota validation for namespaced instancetypes | WEAK | user-cluster (KubeVirt provider) | needs a real KubeVirt datacenter + infra namespace with instancetypes; belongs in the KubeVirt provider e2e (dev KubeVirt is TCG-emulation flaky); unit tests cover the resolution. |

Only #15722 is a genuine non-candidate (no observable object). Everything else is doable;
the ranking reflects effort and marginal value over existing unit tests.

## Recommendation

Two questions decide what to build: (1) is it feasible (almost all are), and (2) is the
marginal value over the PR's existing unit tests worth the cost. Build in this order:

**Tier A — no cluster, clone the current pipeline harness directly:**
1. **#15691** — master/seed CR-only, four crisp branches, uncovered by e2e. Same shape as
   the UPB test. Start here.
2. **#15740 (webhook slice)** — admission rejection on a bad Secret ref; no cluster object.

**Tier B — cluster-object (jig `ClusterJig.Create` without MachineJig, control plane only):**
3. **#15823** — control-plane Deployment `revisionHistoryLimit` field guard.
4. **#15740 (full slice)** — apiserver Secret + `--authentication-config` + netpol.
5. **#16014** — OIDC-issuer NetworkPolicy egress (fiddly input staging).

**Tier C — needs a full user cluster (jig `TestJig.Setup` with MachineJig / cloud); do these
once we add a user-cluster tier to the pipeline, or place them in an existing user-cluster
e2e package:**
6. **#15996** — Cilium NodeLocalDNS CIDR in the user-cluster CNI ApplicationInstallation
   (observed via `ClusterClient`, not seed-side).
7. **#15900** (in-cluster ApplicationInstallations), **#15736** (EE Kyverno control-plane),
   **#16034** (Kyverno teardown), **#16024** (tunneling scrape), **#15958** (KubeVirt),
   **#15892** (stuck Helm state — also hard to stage deterministically).

**Skip:** #15722 (no observable object), #15601 / #15848 (low marginal value over unit tests).

Note on the pipeline harness: Tier B/C require adding jig-based cluster provisioning to
`pkg/test/e2e/pipeline` (the current `main_test.go` is seed-only by design) OR writing the
test in one of the nine existing user-cluster e2e packages. Both are established patterns.

## Shortlist (pre-assessment ranking)

Ranked by the file-signal heuristic; the deep assessment above refines this.

| Score | PR | Title | Signals |
|------:|----|-------|---------|
| 9 | #15958 | fix KubeVirt ee resource quota validation for user-deployed namespaced instancetypes | api, webhook, resources, provider, has-gotest |
| 8 | #15740 | feat: support AuthenticationConfiguration for UCs | webhook, resources, seed-ctrl, installer, sdk-types, has-gotest |
| 7 | #15823 | feat: reduce revisionHistoryLimit to 2 | resources, seed-ctrl, usercluster-ctrl, api, has-gotest |
| 6 | #16014 | Fix apiserver NetworkPolicy OIDC bug for LB backed issuers (KubeLB) | resources, seed-ctrl, has-gotest |
| 5 | #16034 | Improve Kyverno policy handling during template and cluster teardown | api, resources, sdk-types, has-gotest |
| 5 | #15900 | [user-mla] auto-add node-exporter and kube-state-metrics | api, seed-ctrl, has-gotest |
| 5 | #15996 | fix Cilium NodeLocalDNS access with network policies | seed-ctrl, has-gotest |
| 5 | #15848 | Remove direct dependency on k8c.io/kubelb | api, sdk-types |
| 4 | #16024 | Fix seed Prometheus envoy agent scraping for tunneling clusters | resources, has-gotest |
| 4 | #15892 | Fix Helm release recovery for stuck ApplicationInstallations | usercluster-ctrl, has-gotest |
| 4 | #15736 | Add configurable resource overrides for Kyverno controllers | api, sdk-types, has-gotest |
| 4 | #15691 | propagate upstream defaultValuesBlock changes on upgrade | sdk-types, has-gotest |
| 4 | #15722 | fix seed controller manager cache sync timeout issue | seed-ctrl |
| 4 | #15601 | Fix ineffective anti-affinity for seed nodeport-proxy-envoy | resources, has-gotest |

Already covered / excluded from deep assessment:
- #16131 UserProjectBinding pending-delete — implemented (the reference test).
- #15862, #15628, #15896 Gateway API — covered by `pre-kubermatic-gateway-api-e2e`.

## Complete PR inventory (all 186)

Every PR in `v2.30.0..main`, sorted by testability. The Testability column now separates feasibility from cost: `BUILD tier-A` (no cluster), `tier-B` (needs a `Cluster` object, control-plane only), `tier-C` (needs a full user cluster via the jig — feasible, just costlier), plus `SKIP`/`NON-CANDIDATE`/`COVERED`/`IMPLEMENTED`/`excluded`. Only #15722 is a true non-candidate. See the sections above for reasoning.

| PR | Title | Subsystems | Files | Testability |
|----|-------|-----------|------:|-------------|
| [#16014](https://github.com/kubermatic/kubermatic/pull/16014) | Fix apiserver NetworkPolicy OIDC bug for LB backed issuers (like KubeLB) | has-gotest, resources, seed-ctrl | 6 | BUILD tier-B: MODERATE, cluster-object (fiddly) |
| [#15996](https://github.com/kubermatic/kubermatic/pull/15996) | fix Cilium NodeLocalDNS access with network policies | has-gotest, seed-ctrl | 2 | BUILD tier-C: STRONG, user-cluster (CNI AppInstallation lives in user-cluster kube-system, not seed) |
| [#15740](https://github.com/kubermatic/kubermatic/pull/15740) | feat: support AuthenticationConfiguration for UCs | has-gotest, installer, resources, sdk-types, seed-ctrl, webhook | 90 | BUILD tier-A/B: MODERATE (webhook=no cluster) |
| [#15823](https://github.com/kubermatic/kubermatic/pull/15823) | feat: reduce revisionHistoryLimit to 2 | api, has-gotest, resources, seed-ctrl, usercluster-ctrl | 12 | BUILD tier-B: STRONG, cluster-object |
| [#15691](https://github.com/kubermatic/kubermatic/pull/15691) | propagate upstream defaultValuesBlock changes on upgrade | has-gotest, sdk-types | 3 | BUILD tier-A: STRONG, no cluster |
| [#15946](https://github.com/kubermatic/kubermatic/pull/15946) | Enforce Gateway API as default and remove nginx-ingress  path | has-gotest, installer | 46 | COVERED (gateway-api e2e) |
| [#16131](https://github.com/kubermatic/kubermatic/pull/16131) | Prevent deleting pending UserProjectBindings before their User exists | has-gotest, master-ctrl | 2 | IMPLEMENTED (reference test) |
| [#16034](https://github.com/kubermatic/kubermatic/pull/16034) | Improve Kyverno policy handling during template and cluster teardown in controllers | api, has-gotest, resources, sdk-types | 17 | tier-C: MODERATE, needs user cluster (EE Kyverno) |
| [#15900](https://github.com/kubermatic/kubermatic/pull/15900) | [user-mla] Add node-exporter and kube-state-metrics automatically on enabling user-mla monitoring | api, has-gotest, seed-ctrl | 6 | tier-C: MODERATE, needs user cluster |
| [#15958](https://github.com/kubermatic/kubermatic/pull/15958) | fix KubeVirt ee resource quota validation for user-deployed namespaced instancetypes | api, has-gotest, provider, resources, webhook | 27 | tier-C: WEAK, needs KubeVirt user cluster |
| [#16024](https://github.com/kubermatic/kubermatic/pull/16024) | Fix seed Prometheus envoy agent scraping for tunneling clusters | has-gotest, resources | 2 | tier-C: WEAK, needs tunneling user cluster |
| [#15892](https://github.com/kubermatic/kubermatic/pull/15892) | Fix Helm release recovery for stuck ApplicationInstallations | has-gotest, usercluster-ctrl | 10 | tier-C: WEAK, user cluster + hard-to-stage |
| [#15896](https://github.com/kubermatic/kubermatic/pull/15896) | Fix Gateway API migration readiness | has-gotest, installer | 12 | COVERED (gateway-api e2e) |
| [#15862](https://github.com/kubermatic/kubermatic/pull/15862) | Add Bring Your Own Gateway (BYO Gateway) support for KKP | has-gotest, installer, master-ctrl, sdk-types | 40 | COVERED (gateway-api e2e) |
| [#15848](https://github.com/kubermatic/kubermatic/pull/15848) | Remove direct dependency on k8c.io/kubelb | api, sdk-types | 10 | WEAK: external KubeLB mgmt cluster |
| [#15736](https://github.com/kubermatic/kubermatic/pull/15736) | Add configurable resource overrides for Kyverno controllers | api, has-gotest, sdk-types | 13 | tier-C: MODERATE, needs user cluster (EE) |
| [#15722](https://github.com/kubermatic/kubermatic/pull/15722) | fix seed controller manager cache sync timeout issue | seed-ctrl | 3 | NON-CANDIDATE: no observable object (scale/timing) |
| [#15628](https://github.com/kubermatic/kubermatic/pull/15628) | preserve dynamic Gateway listeners in operator reconciliation | has-gotest, master-ctrl | 5 | COVERED (gateway-api e2e) |
| [#15601](https://github.com/kubermatic/kubermatic/pull/15601) | Fix ineffective anti-affinity for seed nodeport-proxy-envoy | has-gotest, resources | 4 | SKIP: static field, unit-covered |
| [#16029](https://github.com/kubermatic/kubermatic/pull/16029) | update terminal web image to 0.13.0 | dep-bump, resources | 1 | unlikely |
| [#16148](https://github.com/kubermatic/kubermatic/pull/16148) | installer: keep --force-conflicts but stop forcing --server-side=true for Helm 4 | has-gotest, installer | 2 | unlikely |
| [#16145](https://github.com/kubermatic/kubermatic/pull/16145) | Bump machine controller to include hetzner datacenter fix | dep-bump, resources | 92 | unlikely |
| [#16138](https://github.com/kubermatic/kubermatic/pull/16138) | Enable server-side apply with conflict forcing when the installer runs with Helm 4 | has-gotest, installer | 2 | unlikely |
| [#16132](https://github.com/kubermatic/kubermatic/pull/16132) | bump KubeLB CCM to v1.4.3 | api, dep-bump | 2 | unlikely |
| [#16065](https://github.com/kubermatic/kubermatic/pull/16065) | Bump github.com/sigstore/fulcio from 1.8.5 to 1.8.6 | dep-bump | 2 | excluded: dep-bump |
| [#16129](https://github.com/kubermatic/kubermatic/pull/16129) | Bump github.com/sigstore/timestamp-authority/v2 from 2.0.6 to 2.1.0 | dep-bump | 2 | excluded: dep-bump |
| [#16049](https://github.com/kubermatic/kubermatic/pull/16049) | Bump github.com/sigstore/rekor from 1.5.1 to 1.5.2 | dep-bump | 2 | excluded: dep-bump |
| [#16118](https://github.com/kubermatic/kubermatic/pull/16118) | Bump Go dependencies | dep-bump, provider | 6 | excluded: dep-bump |
| [#15861](https://github.com/kubermatic/kubermatic/pull/15861) | user-cluster namespace prometheus configmap optimized | resources | 50 | unlikely |
| [#16125](https://github.com/kubermatic/kubermatic/pull/16125) | Bump MLA minio Helm chart to 5.4.0 | chart-only, dep-bump | 4 | excluded: dep-bump |
| [#16124](https://github.com/kubermatic/kubermatic/pull/16124) | Bump cert-manager Helm chart to v1.20.3 | chart-only, dep-bump | 5 | excluded: dep-bump |
| [#16123](https://github.com/kubermatic/kubermatic/pull/16123) | Bump dex chart to 0.24.1 (#16104) | chart-only, dep-bump | 6 | excluded: dep-bump |
| [#16007](https://github.com/kubermatic/kubermatic/pull/16007) | remove sigs and introduce dev-kkp team | api, installer, master-ctrl, provider, resources, sdk-types, seed-ctrl, usercluster-ctrl | 42 | unlikely |
| [#16092](https://github.com/kubermatic/kubermatic/pull/16092) | Add standard Prometheus path annotation to nodeport proxy | has-gotest | 2 | unlikely |
| [#14405](https://github.com/kubermatic/kubermatic/pull/14405) | add option to install kube ovn cni for local kind command | installer | 4 | unlikely |
| [#16084](https://github.com/kubermatic/kubermatic/pull/16084) | Bump MLA Gateway nginx image and add data-plane e2e coverage | dep-bump, has-gotest, seed-ctrl | 4 | unlikely |
| [#16085](https://github.com/kubermatic/kubermatic/pull/16085) | refactor: change pull_request to pull_request_target and add workflow_dispatch for conformance issue tracking | - | 1 | unlikely |
| [#16070](https://github.com/kubermatic/kubermatic/pull/16070) | Bump golang.org/x/net from 0.49.0 to 0.55.0 in /sdk | dep-bump, sdk-types | 2 | excluded: dep-bump |
| [#16082](https://github.com/kubermatic/kubermatic/pull/16082) | Add enableImageDiscovery to OpenStack settings | sdk-types | 2 | unlikely |
| [#16080](https://github.com/kubermatic/kubermatic/pull/16080) | remove opened from issue type | - | 1 | unlikely |
| [#15976](https://github.com/kubermatic/kubermatic/pull/15976) | Introduce Cilium 1.19 support and make 1.19.4  the default | has-gotest | 7 | unlikely |
| [#16077](https://github.com/kubermatic/kubermatic/pull/16077) | skip deployment for github action | - | 2 | unlikely |
| [#16074](https://github.com/kubermatic/kubermatic/pull/16074) | apply kubevirt netpol first when apiserver ip is available | api, has-gotest | 3 | unlikely |
| [#16072](https://github.com/kubermatic/kubermatic/pull/16072) | add issue link in the comment in case of failures | - | 1 | unlikely |
| [#16067](https://github.com/kubermatic/kubermatic/pull/16067) | Add issue template for CIS benchmark request | - | 2 | unlikely |
| [#16068](https://github.com/kubermatic/kubermatic/pull/16068) | rename node-exporter mirror key to match upstream chart name | - | 1 | unlikely |
| [#15668](https://github.com/kubermatic/kubermatic/pull/15668) | Update AIKit and MetalLB documentation URLs in application catalog | api, dep-bump | 7 | unlikely |
| [#16064](https://github.com/kubermatic/kubermatic/pull/16064) | Refactor image deletion step in CIS benchmark workflow to use skopeo | - | 1 | unlikely |
| [#16061](https://github.com/kubermatic/kubermatic/pull/16061) | run cis-bench workflow in kkp | - | 2 | unlikely |
| [#16062](https://github.com/kubermatic/kubermatic/pull/16062) | Add AgentGateway charts to mirror list | - | 1 | unlikely |
| [#16060](https://github.com/kubermatic/kubermatic/pull/16060) | Add environment specification for create-issues job in conformance-trigger workflow | - | 1 | unlikely |
| [#16054](https://github.com/kubermatic/kubermatic/pull/16054) | Fix workflow to trigger cis bench run | - | 1 | unlikely |
| [#16058](https://github.com/kubermatic/kubermatic/pull/16058) | fix small issue with MachineJig network config | - | 1 | unlikely |
| [#16017](https://github.com/kubermatic/kubermatic/pull/16017) | Bump github.com/containerd/containerd/v2 from 2.0.3 to 2.0.10 | dep-bump | 2 | excluded: dep-bump |
| [#16059](https://github.com/kubermatic/kubermatic/pull/16059) | Mirror latest Cilium patch charts | - | 1 | unlikely |
| [#16042](https://github.com/kubermatic/kubermatic/pull/16042) | Add in-cluster LB/PV cleanup finalizers in e2e ClusterJig to stop cloud LB leaks | - | 1 | unlikely |
| [#15978](https://github.com/kubermatic/kubermatic/pull/15978) | Add Kubernetes 1.33 KubeVirt e2e job | - | 1 | unlikely |
| [#16051](https://github.com/kubermatic/kubermatic/pull/16051) | upgrade cilium go.mod dependency which is being used in tests | - | 2 | unlikely |
| [#16027](https://github.com/kubermatic/kubermatic/pull/16027) | Update Alloy with appversion v1.17.0 | chart-only, dep-bump | 5 | excluded: dep-bump |
| [#16048](https://github.com/kubermatic/kubermatic/pull/16048) | fix: update condition for issue creation in conformance workflow | - | 1 | unlikely |
| [#16043](https://github.com/kubermatic/kubermatic/pull/16043) | add workflow for cis benchmark trigger | - | 1 | unlikely |
| [#16047](https://github.com/kubermatic/kubermatic/pull/16047) | Sync changelog for KKP patch releases v2.30.5/v2.29.9/v2.28.12 | docs | 3 | excluded: docs |
| [#16032](https://github.com/kubermatic/kubermatic/pull/16032) | Bump KubeVirt CSI driver operator to v0.5.3 | dep-bump | 1 | excluded: dep-bump |
| [#16033](https://github.com/kubermatic/kubermatic/pull/16033) | update storageclass of kubevirt conformance | dep-bump | 3 | excluded: dep-bump |
| [#15804](https://github.com/kubermatic/kubermatic/pull/15804) | add DisabledAuditWebhookBackendDCs to kubermaticsettings CRD | sdk-types | 3 | unlikely |
| [#16018](https://github.com/kubermatic/kubermatic/pull/16018) | Bump github.com/containerd/containerd from 1.7.32 to 1.7.33 | dep-bump | 2 | excluded: dep-bump |
| [#16022](https://github.com/kubermatic/kubermatic/pull/16022) | Update machine-controller to v1.65.3 | dep-bump, resources | 91 | unlikely |
| [#16015](https://github.com/kubermatic/kubermatic/pull/16015) | Bump go.mongodb.org/mongo-driver from 1.16.1 to 1.17.7 | dep-bump | 2 | excluded: dep-bump |
| [#16020](https://github.com/kubermatic/kubermatic/pull/16020) | Update OSM to v1.10.7 | dep-bump, resources | 99 | unlikely |
| [#16006](https://github.com/kubermatic/kubermatic/pull/16006) | add traceability annotations to conformance LB test Service | - | 1 | unlikely |
| [#15992](https://github.com/kubermatic/kubermatic/pull/15992) | Add support for k8s patch releases v1.35.6/v1.34.9/v1.33.13 | docs | 3 | excluded: docs |
| [#15983](https://github.com/kubermatic/kubermatic/pull/15983) | wait for Gatekeeper health before asserting OPA enforcement | has-gotest | 3 | unlikely |
| [#15990](https://github.com/kubermatic/kubermatic/pull/15990) | Add e2e validation for seed Cilium apiserver policy | has-gotest | 1 | unlikely |
| [#15984](https://github.com/kubermatic/kubermatic/pull/15984) | Reconcile Cilium CCNP as unstructured object | has-gotest | 6 | unlikely |
| [#15980](https://github.com/kubermatic/kubermatic/pull/15980) | update mla controller to handle 404 errors gracefully | dep-bump, has-gotest, seed-ctrl | 9 | unlikely |
| [#15979](https://github.com/kubermatic/kubermatic/pull/15979) | add existingSecret support across KKP-authored Helm charts | chart-only | 23 | excluded: chart-only |
| [#15985](https://github.com/kubermatic/kubermatic/pull/15985) | Validate Kubernetes version before downloading test binaries | - | 1 | unlikely |
| [#15981](https://github.com/kubermatic/kubermatic/pull/15981) | Bump Go version to 1.26.4 | dep-bump, has-gotest, sdk-types, seed-ctrl | 34 | unlikely |
| [#15973](https://github.com/kubermatic/kubermatic/pull/15973) | Clean up LB conformance test namespace to avoid exhausting LB quota | - | 1 | unlikely |
| [#15963](https://github.com/kubermatic/kubermatic/pull/15963) | Add kind labels handling to conformance issue creation workflow | - | 1 | unlikely |
| [#15972](https://github.com/kubermatic/kubermatic/pull/15972) | Remove Cilium skip for multi-protocol service conformance test | - | 1 | unlikely |
| [#15971](https://github.com/kubermatic/kubermatic/pull/15971) | Deprecate superseded Cilium CNI versions from supported versions | has-gotest | 2 | unlikely |
| [#15970](https://github.com/kubermatic/kubermatic/pull/15970) | Mirror Cilium 1.19.4 chart to Kubermatic mirror | - | 1 | unlikely |
| [#15967](https://github.com/kubermatic/kubermatic/pull/15967) | Ignore dockerTagSuffix in mirror-images when --ignore-repository-overrides is set | has-gotest, installer | 2 | unlikely |
| [#15937](https://github.com/kubermatic/kubermatic/pull/15937) | Record ApplicationInstallation Prometheus metrics | usercluster-ctrl | 2 | unlikely |
| [#15956](https://github.com/kubermatic/kubermatic/pull/15956) | Add conformance issue trigger workflow for PRs | - | 1 | unlikely |
| [#15960](https://github.com/kubermatic/kubermatic/pull/15960) | add timeout to nvidia gpu app | api | 1 | unlikely |
| [#15954](https://github.com/kubermatic/kubermatic/pull/15954) | fix: return KubeLB enabled/enforced settings when set to false | sdk-types | 5 | unlikely |
| [#15893](https://github.com/kubermatic/kubermatic/pull/15893) | Bump github.com/go-git/go-git/v5 from 5.18.0 to 5.19.1 | dep-bump | 2 | excluded: dep-bump |
| [#15879](https://github.com/kubermatic/kubermatic/pull/15879) | Upgrade seed mla loki chart to v7.0.0 | chart-only | 6 | excluded: chart-only |
| [#15689](https://github.com/kubermatic/kubermatic/pull/15689) | Bump github.com/go-jose/go-jose/v3 from 3.0.4 to 3.0.5 | dep-bump | 2 | excluded: dep-bump |
| [#15961](https://github.com/kubermatic/kubermatic/pull/15961) | Update default Cilium version to 1.18.10 | dep-bump | 3 | excluded: dep-bump |
| [#15906](https://github.com/kubermatic/kubermatic/pull/15906) | [user-mla] grafana upgrade and dashboard cleanup | has-gotest | 8 | unlikely |
| [#15933](https://github.com/kubermatic/kubermatic/pull/15933) | Synchronize OWNERS_ALIASES file with Github teams | - | 1 | unlikely |
| [#15860](https://github.com/kubermatic/kubermatic/pull/15860) | Change default OIDC clientID to kubermaticIssuer | - | 9 | unlikely |
| [#15950](https://github.com/kubermatic/kubermatic/pull/15950) | Mirror Cilium 1.17.16 and 1.18.10 charts to Kubermatic mirror | - | 3 | unlikely |
| [#15947](https://github.com/kubermatic/kubermatic/pull/15947) | Use mirror Quay credentials for chart mirroring | - | 1 | unlikely |
| [#15944](https://github.com/kubermatic/kubermatic/pull/15944) | Mirror Cilium 1.17.16 chart to Kubermatic mirror | - | 1 | unlikely |
| [#15938](https://github.com/kubermatic/kubermatic/pull/15938) | Fail installer on inconsistent gateway api config | has-gotest, installer | 2 | unlikely |
| [#15888](https://github.com/kubermatic/kubermatic/pull/15888) | Add e2e test for BYO gateway migration | has-gotest | 3 | unlikely |
| [#15926](https://github.com/kubermatic/kubermatic/pull/15926) | Update Machine Controller to v1.65.2 | dep-bump, resources | 93 | unlikely |
| [#15930](https://github.com/kubermatic/kubermatic/pull/15930) | Add changelog for 2.30.4/2.29.8/2.28.11 | docs | 3 | excluded: docs |
| [#15919](https://github.com/kubermatic/kubermatic/pull/15919) | Update OSM to v1.10.6 | dep-bump, resources | 99 | unlikely |
| [#15917](https://github.com/kubermatic/kubermatic/pull/15917) | upgrade golang.org/x/net to v0.55.0 | - | 2 | unlikely |
| [#15912](https://github.com/kubermatic/kubermatic/pull/15912) | Support `ReclaimPolicy` and `AllowVolumeExpansion` for the KubeVirt CSI Driver | sdk-types | 7 | unlikely |
| [#15903](https://github.com/kubermatic/kubermatic/pull/15903) | Bump github.com/containerd/containerd from 1.7.30 to 1.7.32 | dep-bump | 2 | excluded: dep-bump |
| [#15902](https://github.com/kubermatic/kubermatic/pull/15902) | Make the installer's deploy command support Helm 4 | has-gotest, installer | 6 | unlikely |
| [#15897](https://github.com/kubermatic/kubermatic/pull/15897) | Clarify mirror-images cmd supports helm 4 | installer | 1 | unlikely |
| [#15895](https://github.com/kubermatic/kubermatic/pull/15895) | chore: Grant write access to chart mirror script | - | 2 | unlikely |
| [#15880](https://github.com/kubermatic/kubermatic/pull/15880) | Fix app catalog e2e | - | 4 | unlikely |
| [#15878](https://github.com/kubermatic/kubermatic/pull/15878) | Bump github.com/go-git/go-billy/v5 from 5.8.0 to 5.9.0 | dep-bump | 2 | excluded: dep-bump |
| [#15863](https://github.com/kubermatic/kubermatic/pull/15863) | Preserve external SSH keys in user-ssh-key-agent | has-gotest | 6 | unlikely |
| [#15828](https://github.com/kubermatic/kubermatic/pull/15828) | Expose pod scheduling fields for KKP core components | has-gotest, sdk-types | 17 | unlikely |
| [#15869](https://github.com/kubermatic/kubermatic/pull/15869) | Add support of k8s patch releases v1.35.5/vv1.34.8/v1.33.12 | docs | 3 | excluded: docs |
| [#15859](https://github.com/kubermatic/kubermatic/pull/15859) | chore(ccm/hetzner): Bumped hcloud-cloud-controller-manager to v1.30.1 | resources | 1 | unlikely |
| [#15832](https://github.com/kubermatic/kubermatic/pull/15832) | Drop support for k8s v1.32 | has-gotest, installer, resources | 744 | unlikely |
| [#15847](https://github.com/kubermatic/kubermatic/pull/15847) | Update OSM to v1.10.5 and remove Flatcar AMI workaround | dep-bump, has-gotest, resources | 132 | unlikely |
| [#15851](https://github.com/kubermatic/kubermatic/pull/15851) | Specify Vault OIDC path within app chart mirror sh | - | 1 | unlikely |
| [#15849](https://github.com/kubermatic/kubermatic/pull/15849) | Upgrade KubeLB to v1.4.1 | api | 1 | unlikely |
| [#15846](https://github.com/kubermatic/kubermatic/pull/15846) | Fix app helm chart sync CI job | - | 2 | unlikely |
| [#15810](https://github.com/kubermatic/kubermatic/pull/15810) | Make host/zone anti-affinity configurable for more components | installer, resources, sdk-types, usercluster-ctrl | 212 | unlikely |
| [#15839](https://github.com/kubermatic/kubermatic/pull/15839) | Auto-mirror new charts and version bumps when mirror-application-charts.sh changes | - | 5 | unlikely |
| [#15815](https://github.com/kubermatic/kubermatic/pull/15815) | Bump go.opentelemetry.io/otel from 1.40.0 to 1.41.0 | dep-bump | 2 | excluded: dep-bump |
| [#15837](https://github.com/kubermatic/kubermatic/pull/15837) | Sync changelogs for KKP patch release v2.30.3 | docs | 1 | excluded: docs |
| [#15830](https://github.com/kubermatic/kubermatic/pull/15830) | Remove the clusterrole and clusterrolebinding from KubeVirt provider reconciling | provider | 1 | unlikely |
| [#15824](https://github.com/kubermatic/kubermatic/pull/15824) | pin Flatcar AMI in AWS dualstack e2e tests to work around Flatcar 4593.2.0 regression | has-gotest | 1 | unlikely |
| [#15701](https://github.com/kubermatic/kubermatic/pull/15701) | add dependabot configuration for Go modules | - | 1 | unlikely |
| [#15806](https://github.com/kubermatic/kubermatic/pull/15806) | Upgrade KubeLB CCM version to 1.3.10 | api | 1 | unlikely |
| [#15766](https://github.com/kubermatic/kubermatic/pull/15766) | Update vSphere CSI driver to v3.6.0 | dep-bump | 2 | excluded: dep-bump |
| [#15779](https://github.com/kubermatic/kubermatic/pull/15779) | Use kkp-e2e-ubuntu-24.04 source VM for vSphere CI | - | 1 | unlikely |
| [#15773](https://github.com/kubermatic/kubermatic/pull/15773) | Bump github.com/go-git/go-git/v5 from 5.17.1 to 5.18.0 | dep-bump | 2 | excluded: dep-bump |
| [#15775](https://github.com/kubermatic/kubermatic/pull/15775) | Sync changelogs for KKP patch releases v2.30.2/v2.29.7/v2.28.10 | docs | 3 | excluded: docs |
| [#15769](https://github.com/kubermatic/kubermatic/pull/15769) | Remove stale vSphere 1.34 scenario presubmits | - | 1 | unlikely |
| [#15728](https://github.com/kubermatic/kubermatic/pull/15728) | Synchronize OWNERS_ALIASES file with Github teams | - | 1 | unlikely |
| [#15767](https://github.com/kubermatic/kubermatic/pull/15767) | Bump OSM version to v1.10.4 | dep-bump, resources | 129 | unlikely |
| [#15762](https://github.com/kubermatic/kubermatic/pull/15762) | Upgrade KubeLB version to 1.3.9 | api | 1 | unlikely |
| [#15747](https://github.com/kubermatic/kubermatic/pull/15747) | upgrade containerd to v2.2.3 from v2.2.2 | - | 2 | unlikely |
| [#15739](https://github.com/kubermatic/kubermatic/pull/15739) | update app catalog to include the new nvidia gpu operator | api, dep-bump | 7 | unlikely |
| [#15748](https://github.com/kubermatic/kubermatic/pull/15748) | add support for k8s patch releases v1.35.4, v1.34.7 and v1.33.11 | docs | 3 | excluded: docs |
| [#15753](https://github.com/kubermatic/kubermatic/pull/15753) | Bump github.com/moby/spdystream from 0.5.0 to 0.5.1 | dep-bump | 2 | excluded: dep-bump |
| [#15720](https://github.com/kubermatic/kubermatic/pull/15720) | Add Cilium 1.18.8 and 1.17.14 | - | 4 | unlikely |
| [#15712](https://github.com/kubermatic/kubermatic/pull/15712) | reconcile Gateway API resources before Deployments | has-gotest | 4 | unlikely |
| [#15621](https://github.com/kubermatic/kubermatic/pull/15621) | chore: fix run-kubermatic-kind script | - | 2 | unlikely |
| [#15744](https://github.com/kubermatic/kubermatic/pull/15744) | Sync changelogs for KKP patch releases v2.30.1/v2.29.6/v2.28.9 | docs | 3 | excluded: docs |
| [#15704](https://github.com/kubermatic/kubermatic/pull/15704) | update kubeone image tag | dep-bump, resources | 2 | unlikely |
| [#15732](https://github.com/kubermatic/kubermatic/pull/15732) | DRAFT/Proposal - Support manual TLS secrets for Gateway listener sync | has-gotest, master-ctrl, sdk-types | 14 | unlikely |
| [#15721](https://github.com/kubermatic/kubermatic/pull/15721) | Set Canal default version to v3.31 | - | 1 | unlikely |
| [#15602](https://github.com/kubermatic/kubermatic/pull/15602) | add clusterrole and crb for cluster scope resources | provider | 1 | unlikely |
| [#15725](https://github.com/kubermatic/kubermatic/pull/15725) | Add support for Gateway infrastructure annotations to KubermaticConfiguration | has-gotest, sdk-types | 9 | unlikely |
| [#15717](https://github.com/kubermatic/kubermatic/pull/15717) | Bump helm.sh/helm/v3 from 3.19.0 to 3.20.2 | dep-bump | 2 | excluded: dep-bump |
| [#15716](https://github.com/kubermatic/kubermatic/pull/15716) | Add release testing issue template for KKP | - | 1 | unlikely |
| [#15673](https://github.com/kubermatic/kubermatic/pull/15673) | Bump github.com/go-git/go-git/v5 from 5.16.2 to 5.17.1 | dep-bump | 2 | excluded: dep-bump |
| [#15565](https://github.com/kubermatic/kubermatic/pull/15565) | Bump github.com/docker/cli | dep-bump | 2 | excluded: dep-bump |
| [#15690](https://github.com/kubermatic/kubermatic/pull/15690) | Bump github.com/go-jose/go-jose/v4 from 4.1.3 to 4.1.4 | dep-bump | 2 | excluded: dep-bump |
| [#15703](https://github.com/kubermatic/kubermatic/pull/15703) | Bump github.com/aws/aws-sdk-go-v2/service/s3 from 1.78.2 to 1.97.3 | dep-bump | 2 | excluded: dep-bump |
| [#15702](https://github.com/kubermatic/kubermatic/pull/15702) | Bump github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream | dep-bump | 2 | excluded: dep-bump |
| [#15678](https://github.com/kubermatic/kubermatic/pull/15678) | Bump OSM version to v1.10.3 | dep-bump, resources | 131 | unlikely |
| [#15685](https://github.com/kubermatic/kubermatic/pull/15685) | Add test issue to the PR template | - | 1 | unlikely |
| [#15642](https://github.com/kubermatic/kubermatic/pull/15642) | Add ownership to kubermatic-operator's Gateway resources | has-gotest | 3 | unlikely |
| [#15679](https://github.com/kubermatic/kubermatic/pull/15679) | Add support for k8s patch releases v1.35.3/v1.34.6/v1.33.10 | docs | 3 | excluded: docs |
| [#15643](https://github.com/kubermatic/kubermatic/pull/15643) | update tag to 1.33.4 for kubectl being reference in velero | chart-only, dep-bump | 4 | excluded: dep-bump |
| [#14702](https://github.com/kubermatic/kubermatic/pull/14702) | fix various typos | api, has-gotest, master-ctrl, provider, resources, seed-ctrl | 19 | unlikely |
| [#14857](https://github.com/kubermatic/kubermatic/pull/14857) | Introduced the $datasource variable in the Kubermatic Dashboards to fix #14856. | chart-only | 2 | excluded: chart-only |
| [#15664](https://github.com/kubermatic/kubermatic/pull/15664) | Upgrade KubeLB to v1.3.7 | api | 1 | unlikely |
| [#15615](https://github.com/kubermatic/kubermatic/pull/15615) | Fix flaky metallb and other app catalog e2e tests | api, has-gotest | 4 | unlikely |
| [#15659](https://github.com/kubermatic/kubermatic/pull/15659) | Add missing condition to skip MLA Secrets deployment | installer | 1 | unlikely |
| [#15656](https://github.com/kubermatic/kubermatic/pull/15656) | Update OSM to 1.10.2 after hostname removal fix for Openstack | dep-bump, resources | 129 | unlikely |
| [#15651](https://github.com/kubermatic/kubermatic/pull/15651) | Mirror the missing cluster-autoscaler images | installer | 2 | unlikely |
| [#15630](https://github.com/kubermatic/kubermatic/pull/15630) | [user-mla] added alerts for cortex monitoring at seed | chart-only | 7 | excluded: chart-only |
| [#15637](https://github.com/kubermatic/kubermatic/pull/15637) | chore: fix ci file selectors, bump test image | - | 1 | unlikely |
| [#15591](https://github.com/kubermatic/kubermatic/pull/15591) | trigger appcatalog e2e tests on application definition changes | - | 1 | unlikely |
| [#15627](https://github.com/kubermatic/kubermatic/pull/15627) | Make Dex HTTPRoute path and pathType configurable | chart-only | 2 | excluded: chart-only |
| [#15606](https://github.com/kubermatic/kubermatic/pull/15606) | change label key to cluster-id for kubevirt infra netpols | api, has-gotest | 3 | unlikely |
| [#15620](https://github.com/kubermatic/kubermatic/pull/15620) | Fix dual-stack CI failures by using quay.io for Canal Calico images | - | 2 | unlikely |
| [#15614](https://github.com/kubermatic/kubermatic/pull/15614) | Forward kyverno-enabled to the local user-cluster controller-manager helper | - | 1 | unlikely |
| [#15597](https://github.com/kubermatic/kubermatic/pull/15597) | Update OSM to v1.10.1 | dep-bump, resources | 129 | unlikely |
| [#15603](https://github.com/kubermatic/kubermatic/pull/15603) | [user-mla] added monitoring dashboards to monitor cortex itself | chart-only | 12 | excluded: chart-only |
| [#15595](https://github.com/kubermatic/kubermatic/pull/15595) | envoy-gateway: allow separate image repository and tag overrides | chart-only | 7 | excluded: chart-only |
| [#15593](https://github.com/kubermatic/kubermatic/pull/15593) | bump app catalog manager | dep-bump | 6 | excluded: dep-bump |
| [#15578](https://github.com/kubermatic/kubermatic/pull/15578) | kubermatic-installer: deploy ingress-controller or envoy gateway controller in separate seed | installer, resources | 7 | unlikely |
| [#15582](https://github.com/kubermatic/kubermatic/pull/15582) | cleanup: delete OSM migrator controller | seed-ctrl | 3 | unlikely |
| [#15580](https://github.com/kubermatic/kubermatic/pull/15580) | update cert-manager to v1.19.4 | chart-only, dep-bump | 5 | excluded: dep-bump |
| [#15588](https://github.com/kubermatic/kubermatic/pull/15588) | Upgrade to KubeLB v1.3.5 | api, sdk-types | 5 | unlikely |
| [#15583](https://github.com/kubermatic/kubermatic/pull/15583) | cleanup: remove registry template function | has-gotest, resources, seed-ctrl | 3 | unlikely |
| [#15581](https://github.com/kubermatic/kubermatic/pull/15581) | chore: update e2e build images | - | 2 | unlikely |
| [#15576](https://github.com/kubermatic/kubermatic/pull/15576) | update repo for velero and kubectl | chart-only, dep-bump | 4 | excluded: dep-bump |

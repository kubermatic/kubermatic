# AGENTS.md — writing e2e tests in the shared pipeline package

Scope: this file governs `pkg/test/e2e/pipeline/`. It tells a coding agent how to add a
feature test here correctly. Read it fully before writing any test in this directory.

## What this package is

A single e2e test package driven by `sigs.k8s.io/e2e-framework`. It attaches to an
already-running KKP and asserts reconciled state through a controller-runtime client. It
does NOT provision KKP or kind — the run script `hack/ci/run-pipeline-e2e-tests.sh`
stands up kind + KKP once, then runs this package. By default the package is seed-only (no
user cluster). When run with `-with-user-cluster`, `TestMain` additionally provisions one
shared BYO base user cluster for Tier B/C1 features and tears it down at suite end. One
package holds many feature tests; they share the same live KKP (and, if enabled, the same
base cluster).

Files:
- `main_test.go` — `TestMain`, flag registration, and the suite-level health gate. Shared
  by every feature. Do not add feature logic here.
- `cluster_fixture_test.go` — the shared BYO base cluster for Tier B/C1 features (eager,
  `-with-user-cluster`-gated). Read it once if you write a user-cluster test; leave it
  alone otherwise.
- `<feature>_test.go` — one file per feature (e.g. `userprojectbinding_test.go`,
  `cilium_nodelocaldns_test.go`). This is what you add.

## Scope rules (read before choosing what to test)

This slice is **seed-only**. The suite gates on `master-controller-manager` readiness and
talks to the seed/master apiserver. Pick features whose behavior is observable there:

- Good: master/seed controller reconciles, webhook admit/reject, RBAC objects, operator-
  managed Deployments/Secrets/ConfigMaps/NetworkPolicies, cluster-scoped CRs
  (`ApplicationDefinition`, `Project`, `User`, `UserProjectBinding`, ...).
- Needs a `Cluster` object: some controllers only reconcile once a `Cluster` CR with a
  reconciled cluster namespace exists (e.g. control-plane Deployments). This is more
  expensive but allowed; create the `Cluster` in Setup and wait.
- **Tier C1 — user-cluster apiserver objects (allowed here, opt-in):** a feature MAY assert
  on objects the seed-side control plane reconciles *into the user-cluster apiserver* (the
  CNI `ApplicationInstallation`, Secrets/ConfigMaps the seed pushes down, RBAC) using the
  shared base cluster's user-cluster client. No worker Nodes are required: the BYO base
  cluster brings up the apiserver + etcd, and `ClusterClient` derives from the exposed
  apiserver address, not from Nodes. Opt in by passing `-with-user-cluster`; the shared base
  cluster is then provisioned eagerly by `TestMain` and torn down at suite end. Grab it with
  `requireUserCluster(t)` and do all mutate/assert/revert work inside one Assess closure
  (see `cilium_nodelocaldns_test.go`). Not Tier C1: anything that needs a Pod *scheduled on
  a user-cluster Node* (MLA agents, Kyverno, in-cluster Applications) — the BYO cluster has
  no Nodes, so those Pods stay Pending. That is Tier C2 and does NOT belong here; put it in
  a cloud-backed e2e package (konnectivity, mla, ...) that uses `NewAWSCluster` + Machines.

If unsure what a feature needs: find the controller and check which client it writes with.
`mgr.GetClient()` / a seed client → seed-observable (fits this slice, possibly with a
`Cluster` object). A user-cluster client (`userClusterConnection`, `provider.GetClient`) →
the object lives in the user cluster; needs the jig tier.

## The recipe (add one feature)

1. Create `pkg/test/e2e/pipeline/<feature>_test.go`.
2. First line `//go:build e2e`, then the standard Apache header (copy it from
   `userprojectbinding_test.go`).
3. `package pipeline`.
4. One top-level `func Test<Feature>(t *testing.T)` that builds a single
   `features.New("<human name>")` with Setup / Assess / Teardown steps and ends with
   `testEnv.Test(t, feature)`.
5. Get the client with the shared `getClient(t)` helper (wraps `utils.GetClientsOrDie()`).
6. Generate KKP objects with `k8c.io/kubermatic/v2/pkg/test/generator` where one exists
   (`GenProject`, `GenUser`, `GenBinding`, ...). See the naming pitfall below.
7. Assert with gomega `Eventually` / `Consistently` (never bare `if err != nil { t.Fatalf }`
   inside a poll — see rules).
8. Add an assert-clean baseline in Setup and best-effort cleanup in Teardown (shared KKP is
   reused across features and runs).

Skeleton:

```go
//go:build e2e

// <Apache header>

package pipeline

import (
	"context"
	"testing"
	"time"

	"github.com/onsi/gomega"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/generator"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestMyFeature(t *testing.T) {
	client := getClient(t)

	feature := features.New("MyFeature does X").
		Setup(func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
			g := gomega.NewWithT(t)
			// assert-clean baseline, then create inputs
			return ctx
		}).
		Assess("reconciles to expected state", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
			g := gomega.NewWithT(t)
			g.Eventually(func(g gomega.Gomega) {
				obj := &kubermaticv1.SomeKind{}
				g.Expect(client.Get(ctx, types.NamespacedName{Name: "..."}, obj)).To(gomega.Succeed())
				g.Expect(obj.Spec.Field).To(gomega.Equal("expected"))
			}).WithContext(ctx).WithTimeout(time.Minute).WithPolling(time.Second).Should(gomega.Succeed())
			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
			bestEffortDelete(ctx, client, &kubermaticv1.SomeKind{ /* ObjectMeta.Name */ })
			return ctx
		}).
		Feature()

	testEnv.Test(t, feature)
}
```

### Tier C1 recipe (user-cluster apiserver objects)

A Tier C1 feature asserts on an object the seed control plane reconciles *into the
user-cluster apiserver* (no worker Nodes needed). It does NOT use Setup/Teardown for its
work — it grabs the shared base cluster and does everything in one Assess closure:

```go
func TestMyC1Feature(t *testing.T) {
	uc := requireUserCluster(t) // fails fast if -with-user-cluster was not set

	feature := features.New("MyC1Feature does X").
		Assess("reconciles into the user cluster", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
			g := gomega.NewWithT(t)
			g.Eventually(func(g gomega.Gomega) {
				obj := &appskubermaticv1.SomeKind{}
				g.Expect(uc.user.Get(ctx, types.NamespacedName{Name: "...", Namespace: "..."}, obj)).To(gomega.Succeed())
				// assert on the reconciled object
			}).WithContext(ctx).WithTimeout(5 * time.Minute).WithPolling(2 * time.Second).Should(gomega.Succeed())
			return ctx
		}).
		Feature()

	testEnv.Test(t, feature)
}
```

Use `uc.seed` (seed client), `uc.user` (user-cluster client), and `uc.cluster` (the base
`Cluster` CR). Model your test on `cilium_nodelocaldns_test.go`.

**Immutable-field caveat (learned the hard way):** many `Cluster.Spec` fields are immutable
after creation (the cluster-validation webhook rejects changes with "field is immutable" —
e.g. `clusterNetwork.nodeLocalDNSCacheEnabled`). You cannot toggle them on the shared warm
cluster. Either assert against the cluster's created/default state, or restrict mutations to
fields that are actually mutable. If your test needs an immutable field in a specific state,
that state must be set at cluster creation in `cluster_fixture_test.go` (affects every C1
test) or you need a dedicated cluster — not the shared one.

**Read-path caveat:** KKP reconcilers that write Helm values often store them in
`Spec.ValuesBlock` (YAML) and reset `Spec.Values` to `{}`. Read via `app.Spec.GetParsedValues()`,
not `Spec.Values`, or you get an always-empty map. See `cilium_nodelocaldns_test.go`.

## Hard rules

1. **Build tag.** Every file starts with `//go:build e2e` above the header. The package is
   test-only; it only compiles under `-tags e2e`.
2. **Assertions poll, they do not fatal mid-loop.** Use gomega `Eventually`/`Consistently`
   with an inner `func(g gomega.Gomega)` and `g.Expect(...)`. A failed inner expectation is
   retried until the timeout; it is NOT an immediate `t.Fatalf`. Do not write
   `if err != nil { t.Fatalf(...) }` chains inside a poll — they turn a transient error
   (object not yet reconciled, momentary API blip) into a hard failure. Bare
   `g.Expect(client.Create(...)).To(gomega.Succeed())` for one-shot setup calls is fine.
3. **Prefer matchers over hand-rolled helpers.** `ContainElement(HaveField("Kind", ...))`,
   `HaveKeyWithValue(...)`, `And(...)` — not custom `hasOwnerRef`/`hasFinalizer` loops.
4. **RFC 1123 names.** Real apiserver validation rejects names that are not lowercase RFC
   1123 subdomains. The `generator.Gen*` helpers were built for unit tests with a fake
   client and produce names that a live apiserver rejects: `GenProject` appends an
   uppercase `-ID`; `GenBinding` embeds the email (`@`, `.`) in the object name. When you
   use a generator, override `.Name` to a valid lowercase value and thread that one value
   through everything derived from it (RBAC names, `Spec.ProjectID`, etc.). This is the
   single most common way a test passes locally (fake client) but fails in CI.
5. **Seed-only by default.** See scope rules. A feature reaches for the user-cluster
   client only under the Tier C1 exception (objects the seed control plane reconciles into
   the user-cluster apiserver, no worker Nodes): opt the run in with `-with-user-cluster` and
   grab the shared base cluster via `requireUserCluster(t)`. Anything needing a Node-scheduled
   Pod is Tier C2 and belongs in a cloud-backed e2e package, not here.
6. **No `t.Parallel()`.** Features share one KKP; parallel features race on shared state.
7. **Isolate via baseline + teardown.** Setup ensures no leftover objects with your test's
   names exist; `assertCleanBaseline` (in `userprojectbinding_test.go`) does this by
   converging — it re-issues deletes until the objects are gone, because KKP cleanup
   finalizers can outlive a previous run's teardown. Copy that helper rather than failing
   fast on a leftover. Teardown best-effort deletes everything you created. Do not assume a
   clean cluster — the KKP is reused across features and reruns.
8. **Constants that mirror unexported source literals** (e.g. an annotation key defined
   unexported in a controller) must be hardcoded here with a comment pointing at the source
   file, and end the comment with a period (the `godot` linter is enforced in CI).
9. **`main_test.go` is shared infra.** Do not change the existing flag setup or the health
   gate to suit one feature. The `-with-user-cluster` / `-byo-kkp-datacenter` flags and the
   gated `provisionBaseCluster` / `tearDownBaseCluster` steps are already there for every
   Tier C1 feature — reuse them, do not add per-feature provisioning. Do not switch to
   `envconf.NewFromFlags()` — it registers a `--namespace` flag that collides with jig's and
   panics at startup. Note `TestMain` calls `flag.Parse()` explicitly; if you register a new
   flag in `init()`, that call is what makes it take effect (the `testing` package does not
   parse flags automatically when `TestMain` is defined).

## Verify before you push

Run these from the repo root. The first three need no cluster:

```
go test -tags e2e -c -o /dev/null ./pkg/test/e2e/pipeline   # compiles (test-only pkg)
go vet -tags e2e ./pkg/test/e2e/pipeline
gofmt -l pkg/test/e2e/pipeline/                              # empty = clean
golangci-lint run --build-tags e2e ./pkg/test/e2e/pipeline/...
```

Then, against a live KKP (locally or CI). Seed-only (Tier A):

```
go test -tags e2e -p 1 -count=1 -timeout 30m ./pkg/test/e2e/pipeline
```

With the user-cluster tier (Tier B/C1), point `KUBECONFIG` at a seed whose Seed CR has a
BYO datacenter and pass the custom flags. Custom flags MUST go AFTER the package path —
`go test`'s own flag parser runs before the test binary and chokes on an unknown flag placed
before it (you get "no Go files in ."):

```
KUBECONFIG=<seed-kubeconfig> \
go test -tags e2e -p 1 -count=1 -timeout 30m ./pkg/test/e2e/pipeline \
  -with-user-cluster=true \
  -byo-kkp-datacenter <byo-datacenter-name>
```

`go build` reports "no non-test Go files" for this package — that is expected, use the
`go test -c` compile check instead.

## The most important gate: prove the test catches the regression

If the test guards a specific fix, it MUST fail when that fix is reverted. Temporarily
revert the fix commit, run the test, confirm the relevant assessment FAILS, then restore.
A green test that stays green under revert is worthless. Design the assessment so the
pre-fix behavior violates it (e.g. the old code deleted the object → assert it keeps
existing).

## How it runs in CI

- `hack/ci/run-pipeline-e2e-tests.sh` sets up kind + KKP once (built from the current source
  tree, so the running KKP includes the fixes under test), captures logs to
  `$ARTIFACTS/logs`, then runs this package with `-with-user-cluster` (junit →
  `junit.pipeline.xml`).
- `.prow/features.yaml` job `pre-kubermatic-pipeline-e2e` runs the script; its
  `run_if_changed` covers `pkg/test/e2e/pipeline/`, the script, `.prow/`, and the controllers
  currently under test (`master-controller-manager/user-project-binding`,
  `master-controller-manager/rbac`, `seed-controller-manager/cni-application-installation-controller`).
  If your new feature exercises a different controller, extend that `run_if_changed` so the
  job triggers on changes to it.
- **Version assumption:** CI builds KKP from the PR branch, so every test guards a fix that
  is present in the running cluster. If you run this suite against a released KKP (e.g.
  v2.30.x), any test guarding a `main`-only fix that was not backported will fail — that is
  the test doing its job, not a broken test. Cross-check the fix's release-branch status
  before assuming a failure is a bug.

## Reference

- `userprojectbinding_test.go` — the canonical seed-only (Tier A) example (PR #16131
  regression). Copy its structure: Setup (converging baseline + create + wait-active),
  Assess (pending / converge / orphan), Teardown (best-effort delete).
- `cilium_nodelocaldns_test.go` — the canonical Tier C1 example (PR #15996 regression).
  Copy its structure: `requireUserCluster(t)`, single Assess closure, `GetParsedValues()`
  read path, positive-only assertion against the shared cluster's default state.
- `cluster_fixture_test.go` + `main_test.go` — the harness; read each once to understand
  the health gate, the `-with-user-cluster` flag, and the shared base cluster, then leave
  them alone.

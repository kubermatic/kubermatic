# AGENTS.md — writing e2e tests in the shared pipeline package

Scope: this file governs `pkg/test/e2e/pipeline/`. It tells a coding agent how to add a
feature test here correctly. Read it fully before writing any test in this directory.

## What this package is

A single e2e test package driven by `sigs.k8s.io/e2e-framework`. It attaches to an
already-running KKP and asserts reconciled state through a controller-runtime client. It
does NOT provision KKP, kind, or a cluster — the run script `hack/ci/run-pipeline-e2e-tests.sh`
stands up kind + KKP once, then runs this package. One package holds many feature tests;
they share the same live KKP.

Files:
- `main_test.go` — `TestMain`, flag registration, and the suite-level health gate. Shared
  by every feature. Do not add feature logic here.
- `<feature>_test.go` — one file per feature (e.g. `userprojectbinding_test.go`). This is
  what you add.

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
7. **Isolate via baseline + teardown.** Setup asserts no leftover objects with your test's
   names exist (fail loudly if they do); Teardown best-effort deletes everything you created.
   Do not assume a clean cluster — the KKP is reused across features and reruns.
8. **Constants that mirror unexported source literals** (e.g. an annotation key defined
   unexported in a controller) must be hardcoded here with a comment pointing at the source
   file, and end the comment with a period (the `godot` linter is enforced in CI).
9. **`main_test.go` is shared infra.** Do not change the flag setup or the health gate to
   suit one feature. Do not switch to `envconf.NewFromFlags()` — it registers a `--namespace`
   flag that collides with jig's and panics at startup.

## Verify before you push

Run these from the repo root. The first three need no cluster:

```
go test -tags e2e -c -o /dev/null ./pkg/test/e2e/pipeline   # compiles (test-only pkg)
go vet -tags e2e ./pkg/test/e2e/pipeline
gofmt -l pkg/test/e2e/pipeline/                              # empty = clean
golangci-lint run --build-tags e2e ./pkg/test/e2e/pipeline/...
```

Then, against a live KKP (locally or CI):

```
go test -tags e2e -p 1 -count=1 -timeout 30m ./pkg/test/e2e/pipeline
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

- `hack/ci/run-pipeline-e2e-tests.sh` sets up kind + KKP once, captures logs to
  `$ARTIFACTS/logs`, then runs this package (junit → `junit.pipeline.xml`).
- `.prow/features.yaml` job `pre-kubermatic-pipeline-e2e` runs the script; its
  `run_if_changed` covers `pkg/test/e2e/pipeline/`, the script, `.prow/`, and the
  controllers currently under test. If your new feature exercises a different controller,
  extend that `run_if_changed` so the job triggers on changes to it.

## Reference

`userprojectbinding_test.go` is the canonical example (PR #16131 regression). Copy its
structure: Setup (baseline + create + wait-active), Assess (pending / converge / orphan),
Teardown (best-effort delete). `main_test.go` is the harness; read it once to understand
the health gate and flag handling, then leave it alone.

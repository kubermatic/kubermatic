# MLA e2e suites

Three Ginkgo suites test KKP monitoring, logging and alerting from the
customer's point of view. They share one set of checks:

| Path | What |
|---|---|
| `pkg/test/e2e/testkit/mla` | The checks, as plain functions (`func(ctx, Input) error`). Importable by other suites, for example conformance-ee. |
| `pkg/test/e2e/features/mla` | The feature registry (ID, scope, mode, labels) and the Ginkgo wrappers. |
| `pkg/test/e2e/mla` | User cluster MLA suite, build tag `mla`. |
| `pkg/test/e2e/seedmla` | Seed MLA suite, build tag `seedmla`. |
| `pkg/test/e2e/mlaupgrade` | Upgrade suite, build tag `mlaupgrade`. |
| `pkg/test/e2e/testkit/mla/metric-contract.yaml` | Metric names the seed rules and the MLA dashboards use. `hack/verify-mla-metric-contract.sh` keeps it in sync with the charts. |

## Requirements on the KKP installation

- User cluster suite: the `usercluster-mla` stack in namespace `mla`, and a KKP
  admin `User` whose email is passed with `-grafana-admin-email` (default
  `roxy-admin@kubermatic.com`). With `-existing-cluster`, a user cluster on that
  seed; checks labelled `needs-nodes` need worker nodes in it.
- Seed suite: the `seed-mla` stack in namespaces `monitoring` and `logging`.
  Checks labelled `needs-user-cluster` need an MLA user cluster on the seed.
- The user cluster suite turns user cluster MLA on in the Seed and in the
  cluster before the first check. The `disruptive` check, which verifies that
  disabling MLA removes the agents, the gateway and the Grafana org, turns it
  off again.

## Running

From the repository root, with bash (`hack/lib.sh` does not work in zsh).
Ginkgo flags go before the package, suite flags after `--`.

```sh
# user cluster MLA
bash -c 'source hack/lib.sh && ginkgo_test mla --tags mla --procs 2 -v ./pkg/test/e2e/mla -- "$@"' _ \
  -kubeconfig <seed kubeconfig> -existing-cluster <cluster name>

# seed MLA
bash -c 'source hack/lib.sh && ginkgo_test seedmla --tags seedmla --procs 2 -v ./pkg/test/e2e/seedmla -- "$@"' _ \
  -kubeconfig <seed kubeconfig>

# upgrade: once before and once after the KKP upgrade, with the same state file
bash -c 'source hack/lib.sh && ginkgo_test up --tags mlaupgrade --label-filter pre-upgrade -v ./pkg/test/e2e/mlaupgrade -- "$@"' _ \
  -kubeconfig <seed kubeconfig> -existing-cluster <cluster name> -upgrade-state-file state.json
# ... upgrade KKP and the MLA stacks ...
bash -c 'source hack/lib.sh && ginkgo_test up --tags mlaupgrade --label-filter post-upgrade -v ./pkg/test/e2e/mlaupgrade -- "$@"' _ \
  -kubeconfig <seed kubeconfig> -upgrade-state-file state.json
```

Without `-existing-cluster` the user cluster and upgrade suites create an AWS
user cluster and need `-aws-kkp-datacenter`, the AWS credentials from the
environment and `-ssh-pub-key`, as in `hack/ci/run-mla-e2e-tests.sh`.

| Goal | Ginkgo flag |
|---|---|
| Cluster without worker nodes | `--label-filter='!needs-nodes'` |
| Seed without MLA user clusters | `--label-filter='!needs-user-cluster'` |
| Keep MLA on after the run | `--label-filter='!(scope: containsAny disruptive)'` |
| Seed shared with other users | `--label-filter='!(scope: containsAny {seed-wide, disruptive})'` |
| One check | `--label-filter='id: containsAny U10'` |

Combine filters with `&&`.

## Adding a check

1. Write the check as `func(ctx context.Context, in Input) error` (or
   `SeedInput`) in `pkg/test/e2e/testkit/mla`. It must clean up what it
   creates and must not create clusters.
2. Add an entry to `UserClusterFeatures` or `SeedFeatures` in
   `pkg/test/e2e/features/mla/registry.go` with a scope and a mode. Use
   `ModeSerial` when the check changes config shared by the cluster (the
   Alertmanager config, `spec.mla`, rate limits) or restarts shared
   components.
3. `go test ./pkg/test/e2e/features/...` validates the registry.

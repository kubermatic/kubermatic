# Metering e2e suite

Tests KKP metering (Enterprise Edition) from the customer's point of view:
the operator deploys the metering Prometheus and the report CronJobs, the
reports land in S3 with the documented keys, columns and fields, and the
values match the cluster's usage.

| Path | What |
|---|---|
| `pkg/test/e2e/testkit/metering` | The checks, as plain functions (`func(ctx, Input) error`). |
| `pkg/test/e2e/features/metering` | The feature registry (ID, scope, mode, labels) and the Ginkgo wrappers. |
| `pkg/test/e2e/metering` | The suite, build tag `metering`. |
| `hack/ci/run-metering-e2e-tests.sh` | CI entry point, KKP EE in kind with an AWS user cluster. |

## Requirements on the KKP installation

- KKP EE, master and seed in one cluster.
- A metering bucket. When the seed has no `metering-s3` Secret, the suite
  creates the bucket `-s3-bucket` in the seed MinIO of the kubermatic-seed
  stack (namespace `minio`) and a Secret for it, and deletes the Secret
  afterwards. An existing Secret is used as it is.
- When metering is off, the suite turns it on with `-storage-class` and
  `-storage-size`, and turns it off again at the end. Storage settings of a
  Seed that already has metering on are never changed: changing them needs
  manual steps.
- Checks labelled `needs-user-cluster` need a user cluster with worker nodes
  and a default StorageClass. The suite deploys a probe pod with a PVC into
  namespace `metering-e2e` of that cluster.

The suite adds two report configurations named `e2e-<random>-csv` and
`e2e-<random>-json` to the Seed and removes them at the end, together with
the reports under that prefix. Two things stay behind:

- When the suite turned metering on itself, turning it off keeps the
  Prometheus PVC `storage-metering-prometheus-0`, as KKP does for every Seed.
- The lifecycle rule of the CSV report stays in the bucket when no other
  report has a retention, because the operator skips the lifecycle update in
  that case (a known issue).

## Known issues

Checks labelled `known-issue` fail because of confirmed product bugs and are
skipped in CI. The `KnownIssue` field of each entry in the registry says why.
The biggest one: the Seed validating webhook selects namespaces labelled
`app.kubernetes.io/name=<KKP namespace>`, and nothing sets that label, so the
webhook never runs. The checks for rejecting invalid report settings, for
deploying every report configuration the Seed accepts and for the Seed example
in the metering docs depend on it. Labelling the namespace by hand makes the
webhook run.

## How the report checks work

Scheduled reports cover whole days that end at midnight UTC, so a cluster
created by the test never shows up in them on the same day. The report checks
therefore run the metering tool once with the CronJob's own pod template and
an explicit `--start`/`--end` window that ends a few minutes before now, and
compare every column with the documented queries evaluated at the same time.
The check for the scheduled uploads runs the CronJobs as they are and only
checks the keys, content types and headers.

The column lists, JSON fields and queries are those of metering v1.4.1. A
metering bump that renames, moves or removes a column or field, or changes
how a value is computed, fails the report format and report value checks.
Appending columns is allowed.

## Running

From the repository root, with bash (`hack/lib.sh` does not work in zsh).

```sh
bash -c 'source hack/lib.sh && ginkgo_test metering --tags metering --procs 2 -v --label-filter "!known-issue" ./pkg/test/e2e/metering -- "$@"' _ \
  -kubeconfig <seed kubeconfig> -existing-cluster <cluster name>
```

| Goal | Flag |
|---|---|
| No user cluster | `-no-user-cluster` and `--label-filter='!needs-user-cluster'` |
| Keep metering on after the run | `--label-filter='!(scope: containsAny disruptive)'` |
| Seed shared with other users | `--label-filter='!(scope: containsAny {seed-wide, disruptive})'` |
| Only the known product bugs | `--label-filter='known-issue'` |
| One check | `--label-filter='id: containsAny M32'` |

Without `-existing-cluster` the suite creates an AWS user cluster and needs
`-aws-kkp-datacenter`, the AWS credentials from the environment and
`-ssh-pub-key`.

## Adding a check

1. Write the check as `func(ctx context.Context, in Input) error` in
   `pkg/test/e2e/testkit/metering`. It must clean up what it creates and
   must not create clusters.
2. Add an entry to `Features` in `pkg/test/e2e/features/metering/registry.go`.
   Use `ModeSerial` when the check changes the Seed, the bucket lifecycle or
   the cluster under test.
3. `go test ./pkg/test/e2e/features/metering/ ./pkg/test/e2e/testkit/metering/`
   validates the registry and the offline helpers.

# Scripts list and their description

This file is generated

## hack/changelog-gen.sh

Create a changelog since last release, commit and create a new release tag

    Usage:
    changelog-gen.sh -r v2.x.x - create changelog, commit and tag new release, using closed PRs release-note

## hack/ci/canary-github-release.sh

TBD

## hack/ci/deploy-ci-kubermatic-io.sh

TBD

## hack/ci/deploy-dev-asia.sh

TBD

## hack/ci/deploy-dev.sh

TBD

## hack/ci/deploy-offline.sh

TBD

## hack/ci/deploy-run-2-lab-alpha.sh

TBD

## hack/ci/deploy-run-lab-stable.sh

TBD

## hack/ci/deploy-run.sh

TBD

## hack/ci/deploy.sh

TBD

## hack/ci/download-gocache.sh

TBD

## hack/ci/github-release.sh

TBD

## hack/ci/push-images.sh

TBD

## hack/ci/run-api-e2e.sh

TBD

## hack/ci/run-conformance-tester.sh

TBD

## hack/ci/run-e2e-tests.sh

TBD

## hack/ci/run-lint.sh

TBD

## hack/ci/run-offline-test.sh

TBD

## hack/ci/setup-kubermatic-in-kind.sh

TBD

## hack/ci/setup-legacy-kubermatic-in-kind.sh

TBD

## hack/ci/sync-apiclient.sh

TBD

## hack/ci/sync-charts.sh

TBD

## hack/ci/test-github-release.sh

TBD

## hack/ci/update-docs.sh

TBD

## hack/ci/upload-gocache.sh

TBD

## hack/ci/verify-chart-versions.sh

TBD

## hack/ci/verify-user-cluster-prometheus-configs.sh

TBD

## hack/coverage.sh

Generate test coverage statistics for Go packages.

Works around the fact that `go test -coverprofile` currently does not work  
with multiple packages, see https://code.google.com/p/go/issues/detail?id=6909

    Usage: cover.sh [--html]

    --html      Additionally create HTML report and open it in browser

## hack/gen-api-client.sh

TBD

## hack/lib.sh

TBD

## hack/meta.sh

This README.md generator

That generates README.md with all scripts from this directory described.

## hack/publish-s3-exporter.sh

TBD

## hack/release-docker-images.sh

TBD

## hack/retag-images.sh

TBD

## hack/run-api.sh

TBD

## hack/run-dashboard-and-api.sh

TBD

## hack/run-machine-controller.sh

TBD

## hack/run-master-controller-manager.sh

TBD

## hack/run-operator.sh

TBD

## hack/run-seed-controller-manager.sh

TBD

## hack/run-user-cluster-controller-manager.sh

TBD

## hack/update-cert-manager-crds.sh

TBD

## hack/update-codegen.sh

TBD

## hack/update-docs.sh

TBD

## hack/update-fixtures.sh

TBD

## hack/update-grafana-dashboards.sh

TBD

## hack/update-kubermatic-chart.sh

TBD

## hack/update-openshift-version-codegen.sh

This script can be used to update the generated image names for Openshift.  
The desired versions msut be configured first in  
codegen/openshift_versions/main.go and a const for each version must be  
added to pkg/controller/openshift/resources/const.go

Also, executing this script requires access to the ocp quay repo.

## hack/update-prometheus-rules.sh

TBD

## hack/update-swagger.sh

TBD

## hack/update-velero-crds.sh

TBD

## hack/verify-api-client.sh

TBD

## hack/verify-boilerplate.sh

TBD

## hack/verify-codegen.sh

TBD

## hack/verify-docs.sh

TBD

## hack/verify-forbidden-functions.sh

TBD

## hack/verify-grafana-dashboards.sh

TBD

## hack/verify-kubermatic-chart.sh

TBD

## hack/verify-licenses.sh

TBD

## hack/verify-prometheus-rules.sh

TBD

## hack/verify-swagger.sh

TBD


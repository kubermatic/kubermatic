# Cluster Backup

This directory contains two Helm charts that are used to create automated
backups of resources inside the seed cluster. They wrap Ark and restic and
need to be deployed sequentially.

## Architecture

See the [official Ark documentation](https://heptio.github.io/ark/v0.9.0/)
for more information on what Ark does and does not. In Kubermatic Ark is
can be used either in a cloud-native way where all volume snapshots are
done using the cloud provider's APIs or in a custom mode where volume
backups are created file-by-file using restic. Mixed setups are possible
as well.

## Configuration

The two Helm charts should share the same `values.yaml`, because they both
use some of the same keys to work together. The Helm chart was split
because `ark-config` creates a manifest that depends on a CRD that the
`ark` chart deploys.

See the `values.yaml` in both charts for more details on the configuration.

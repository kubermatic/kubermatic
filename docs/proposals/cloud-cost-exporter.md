# Cloud Cost Exporter

**Author**: Matthias Loibl (@metalmatze)

**Status**: Draft proposal; prototype in progress.

## Motivation and Background

Billing in Cloud Environments is hard and sometimes almost obscure.
We want to improve this situation for our customers.
For this reason, we want to create a Cloud Cost Exporter for Prometheus, which
allows Prometheus to scrape the cost of the Kubermatic managed infrastructure.

In the end we want to show the current cost of the cluster in the Kubermatic Dashboard.

## Implementation

Create a new Prometheus Exporter in Go.
This Exporter uses SDKs of various Cloud Providers:

* https://github.com/aws/aws-sdk-go
* https://github.com/digitalocean/godo
* https://github.com/GoogleCloudPlatform/google-cloud-go
* https://github.com/hetznercloud/hcloud-go

We want the exporter to be configurable via a config file / configmap.

The exporter can then scrape the Cloud Providers for the number of Nodes, Disks and similar infrastructure under control by Kubermatic.
With this data ingested into Prometheus we can query various different costs with PromQL queries.

## Task & effort:
* Implement the basic exporter and one first collector - 1d
* Adding a new Provider - 0.5d

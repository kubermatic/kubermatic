# Global Monitoring

**Author**: Christoph (@xrstf)
**Status**: Draft proposal

This is a proposal to create a centralized view for all monitoring data across all seed
and user clusters. Karma (the centralized Alertmanager dashboard) is a great step in
that direction, but being able to run queries and see dashboards in a single Grafana is
still missing.

## Motivation

* Currently, user-cluster monitoring is severely lacking; while some data is federated,
  we only have a single etcd dashboard that actually uses it. Prometheus is mainly used
  for alerting right now.
* Due to performance issues when ingesting *all* metrics, we decided to pre-compute the
  metrics necessary for the etcd dashboard and only federate those. This has the massive
  downside that none of the other dashboards (e.g. for kube-state-metrics) can work for
  user clusters.
* Likewise, the interesting user-cluster metrics are gone after a few hours or after the
  pod is restarted, whichever happens first. This makes debugging a problem inside user
  clusters harder.
* It's not possible to run ad-hoc queries across user clusters (or even in a specific
  user cluster) without manually port-forwarding everyhwere.

## Proposal

Let's use one of the available products for H/A Prometheus setups. This can not only help
with aggregating metrics from various Prometheus instances, but also solve the backup
question by utilizing an object store.

### Options

We evaluated the following products:

#### VictoriaMetrics 1.28.3

**Pros**:

* Great performance
* Simple configuration (single binary)

**Cons**:

* Does not use object store and would require backup jobs (via snapshots like Prometheus)
* Cannot de-duplicate data (recommends extra promxy instance)
* Returns wrong query results
* 1-man project re-implementing everything

#### Cortex

**Pros**:

* Maintained Grafana/Prometheus community
* (never actually tested due to cons)

**Cons**:

* Requires [Cassandra/DynamoDB/BigTable](https://github.com/cortexproject/cortex/blob/master/docs/architecture.md#chunk-store)
  for index storage
* Requires its own backup for the index

#### Thanos 0.8

**Pros**:

* Maintained Grafana/Prometheus community
* Purely based on object store
* Flexible to configure

**Cons**:

* Needs lots of resources (especially for compacting)
* No remote-write receiver

#### Conclusion

VictoriaMetrics is out because of the wrong query results and the need for a backup job.
Thanos beats Cortex because of its architecture around an object store, even though the
missing remote-write receiver ([proposal](https://thanos.io/proposals/201812_thanos-remote-receive.md/))
is making user-cluster aggregation harder. Also, Cortex's architecture is much more
complicated than Thanos's.

## Goals

* **Provide near-realtime access to user-cluster metrics.**

  Currently we override the block duration to be 15min, but this is highly discouraged
  and we should stop doing it. It's an ugly hack to work around too many missing metrics
  because the user-cluster Prometheus pods have no persistent storage.

  Near realtime is what makes everything else meaningful. When sh*t hits the fan and
  you want to check the dashboards, data being delayed by up to 2 hours (again, when we
  stop doing nasty things) is a major inconvenience.

* **Keep User-Cluster Prometheus stateless.**

  We could give every Prometheus a 10Gi disk, but this would quickly balloon. And
  balloon unnecessarily because the data is streamed somewhere else anyway in order to
  eventually make it durable.

* **Be able to scale to accommodate many user clusters.**

  If for example the Seed-Prometheus would scrape all User-Cluster Prometheus instances,
  it would quickly eat more memory than we're willing to give it. If it then also has to
  run queries, things quickly die.

* **Provide master-Grafana** with dashboards providing an overview over all seeds and user
  clusters.

* **Keep seed-Grafanas** so that if the master goes down, monitoring in seeds is still
  possible.

## Masterplan

### User Clusters

These Prometheus instances are stateless and mainly used so spread the scrape/alerting load
across many instances (instead of having the Seed Prometheus scrape everything). They are
configured with a bunch of recording rules, primarily centered around etcd (because
historically the etcd alerts we inherited from Kubernetes jsonnet mixins are veeeery
sensitive to fluctuations in The Force). Only a few prerecorded metrics are labelled with
`federate=true` in order to reduce the load on the Seed Prometheus.

As this prerecordning obscures metrics and is only a hack, I propose to stop doing it entirely.
Instead, we should drop a couple of "worst offender" metrics, namely:

* apiserver request latencies histogram
* apiserver request duration histogram
* apiserver response size histogram

These account for 33% of all timeseries in a user-cluster. If we remove them and then start
to "export" (see below) all metrics, the existing Grafana dashboards can be used for
user-clusters.

To get the data out, we have basically three options:

* **Federation** (Seed-Prometheus scrapes User-Cluster Prometheus)

  This is what we do right now. It makes data available pretty much immediately inside the
  seed and thereby makes it durable as well.

  Having the Seed Prometheus scrape *all* timeseries (even if we ignore the apiserver ones
  mentioned above) will likely overload it. We could setup a dedicated "Federation Prometheus"
  that only scrapes and produces blocks to be uploaded via Thanos Sidecar.

* **Thanos Sidecar** (Sidecar container uploads newly created blocks)

  This has the downside of delaying metrics by the block duration. Also, metrics are lost
  when the pod restarts. This makes it unsuitable for our setup.

* **Remote Write** (Prometheus streams data somewhere)

  This solves the durability and the immediate availability, but is not something supported
  by Thanos directly.

Given the choice above, we only have the choice to stick with federation until Thanos
supports Remote Write.

As mentioned above, letting the existing Seed Prometheus scrape all metrics again would
likely cause issues, so we should setup a new HA pair of Prometheus that is responsible
only for federation. While this seems overkill at first, it has the nice side effect of
decoupling seed monitoring from user-cluster monitoring. Also, managing multiple smaller
Prometheus instances is easier than one big one.

This leaves us with this plan:

* Remove the block duration hack and go back to regular 2h block duration.
* Remove the prerecording rules inside user-cluster Prometheus.
* Drop the three metrics named above.
* Setup new HA Prometheus pair to federate from user-clusters and produce blocks.
* Have a Thanos Sidecar upload these blocks to an object store.
* Configure additional Thanos Compactor to compact if the Sidecar uses a different
  bucket than the seed (which I would recommend (if different bucket or entirely
  different Minio is not important)).

### Seed Clusters

Inside seeds we already make use of Thanos to upload and compact blocks. A compactor
does downsampling and store and query components are setup to provide access to the
data in the seed-owned object store.

Since a new Prometheus pair now does federation, the Seed Prometheus can stop doing
so. This will break the etcd dashboard in Grafana, but we want to re-purpose the
"etcd (Seed)" dashboard anyway.

So the action items for the seeds are:

* Remove federation configuration.
* Remove outdated etcd dashboard.
* Update other etcd dashboard to work transparently on seed or user-clusters.

### Master Cluster

TBD

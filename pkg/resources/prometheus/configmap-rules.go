/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package prometheus

// prometheusRules contains the prerecoding and alerting rules for
// Prometheus. Be careful when changing alert names, as some of them
// are used for alert inhibitions and configured inside the
// seed-cluster's Alertmanager.
// Also make sure that groups list ends with the kubernetes-absent alerting group,
// so that prometheusRuleDNSResolverDownAlert works as expected.
const prometheusRules = `
groups:
- name: kubermatic.goprocess
  rules:
  - record: job:process_resident_memory_bytes:clone
    expr: process_resident_memory_bytes
    labels:
      kubermatic: federate

  - record: job:process_cpu_seconds_total:rate5m
    expr: rate(process_cpu_seconds_total[5m])
    labels:
      kubermatic: federate

  - record: job:process_open_fds:clone
    expr: process_open_fds
    labels:
      kubermatic: federate

- name: kubermatic.etcd
  rules:
  - alert: EtcdInsufficientMembers
    annotations:
      message: 'Etcd cluster "{{ $labels.job }}": insufficient members ({{ $value }}).'
    expr: |
      sum(up{job="etcd"} == bool 1) by (job) < ((count(up{job="etcd"}) by (job) + 1) / 2)
    for: 15m
    labels:
      severity: critical

  - alert: EtcdNoLeader
    annotations:
      message: 'Etcd cluster "{{ $labels.job }}": member {{ $labels.instance }} has no leader.'
    expr: |
      etcd_server_has_leader{job="etcd"} == 0
    for: 15m
    labels:
      severity: critical

  - alert: EtcdHighNumberOfLeaderChanges
    annotations:
      message: 'Etcd cluster "{{ $labels.job }}": instance {{ $labels.instance }} has seen {{ $value }} leader changes within the last hour.'
    expr: |
      rate(etcd_server_leader_changes_seen_total{job="etcd"}[15m]) > 3
    for: 15m
    labels:
      severity: warning

  - alert: EtcdGRPCRequestsSlow
    annotations:
      message: 'Etcd cluster "{{ $labels.job }}": gRPC requests to {{ $labels.grpc_method }} are taking {{ $value }}s on etcd instance {{ $labels.instance }}.'
    expr: |
      histogram_quantile(0.99, sum(rate(grpc_server_handling_seconds_bucket{job="etcd", grpc_type="unary"}[5m])) by (job, instance, grpc_service, grpc_method, le))
      > 0.15
    for: 10m
    labels:
      severity: critical

  - alert: EtcdMemberCommunicationSlow
    annotations:
      message: 'Etcd cluster "{{ $labels.job }}": member communication with {{ $labels.To }} is taking {{ $value }}s on etcd instance {{ $labels.instance }}.'
    expr: |
      histogram_quantile(0.99, rate(etcd_network_peer_round_trip_time_seconds_bucket{job="etcd"}[5m]))
      > 0.15
    for: 10m
    labels:
      severity: warning

  - alert: EtcdHighNumberOfFailedProposals
    annotations:
      message: 'Etcd cluster "{{ $labels.job }}": {{ $value }} proposal failures within the last hour on etcd instance {{ $labels.instance }}.'
    expr: |
      rate(etcd_server_proposals_failed_total{job="etcd"}[15m]) > 5
    for: 15m
    labels:
      severity: warning

  - alert: EtcdHighFsyncDurations
    annotations:
      message: 'Etcd cluster "{{ $labels.job }}": 99th percentile fync durations are {{ $value }}s on etcd instance {{ $labels.instance }}.'
    expr: |
      histogram_quantile(0.99, rate(etcd_disk_wal_fsync_duration_seconds_bucket{job="etcd"}[5m]))
      > 0.5
    for: 10m
    labels:
      severity: warning

  - alert: EtcdHighCommitDurations
    annotations:
      message: 'Etcd cluster "{{ $labels.job }}": 99th percentile commit durations {{ $value }}s on etcd instance {{ $labels.instance }}.'
    expr: |
      histogram_quantile(0.99, rate(etcd_disk_backend_commit_duration_seconds_bucket{job="etcd"}[5m]))
      > 0.25
    for: 10m
    labels:
      severity: warning

  - alert: EtcdDatabaseQuotaLowSpace
    annotations:
      message: 'Etcd cluster "{{ $labels.job }}": database size exceeds the defined quota on etcd instance {{ $labels.instance }}, please defrag or increase the quota as the writes to etcd will be disabled when it is full.'
    expr: |
      (last_over_time(etcd_mvcc_db_total_size_in_bytes[5m]) / last_over_time(etcd_server_quota_backend_bytes[5m]))*100 > 95
    for: 10m
    labels:
      severity: critical

  - alert: EtcdExcessiveDatabaseGrowth
    annotations:
      message: 'Etcd cluster "{{ $labels.job }}": Predicting running out of disk space in the next four hours, based on write observations within the past four hours on etcd instance {{ $labels.instance }}, please check as it might be disruptive.'
    expr: |
      predict_linear(etcd_mvcc_db_total_size_in_bytes[4h], 4*60*60) > etcd_server_quota_backend_bytes
    for: 10m
    labels:
      severity: warning

  - alert: EtcdDatabaseHighFragmentationRatio
    annotations:
      message: 'Etcd cluster "{{ $labels.job }}": database size in use on instance {{ $labels.instance }} is {{ $value | humanizePercentage }} of the actual allocated disk space, please run defragmentation (e.g. etcdctl defrag) to retrieve the unused fragmented disk space.'
    expr: |
      (last_over_time(etcd_mvcc_db_total_size_in_use_in_bytes[5m]) / last_over_time(etcd_mvcc_db_total_size_in_bytes[5m])) < 0.5 and etcd_mvcc_db_total_size_in_use_in_bytes > 104857600
    for: 10m
    labels:
      severity: warning

  - record: job:etcd_server_has_leader:sum
    expr: sum(etcd_server_has_leader)
    labels:
      kubermatic: federate

  - record: job:etcd_disk_wal_fsync_duration_seconds_bucket:99percentile
    expr: histogram_quantile(0.99, sum(rate(etcd_disk_wal_fsync_duration_seconds_bucket[5m])) by (instance, le))
    labels:
      kubermatic: federate

  - record: job:etcd_disk_backend_commit_duration_seconds_bucket:99percentile
    expr: histogram_quantile(0.99, sum(rate(etcd_disk_backend_commit_duration_seconds_bucket[5m])) by (instance, le))
    labels:
      kubermatic: federate

  - record: job:etcd_mvcc_db_total_size_in_bytes:clone
    expr: etcd_mvcc_db_total_size_in_bytes
    labels:
      kubermatic: federate

  - record: job:etcd_network_client_grpc_received_bytes_total:rate5m
    expr: rate(etcd_network_client_grpc_received_bytes_total[5m])
    labels:
      kubermatic: federate

  - record: job:etcd_network_client_grpc_sent_bytes_total:rate5m
    expr: rate(etcd_network_client_grpc_sent_bytes_total[5m])
    labels:
      kubermatic: federate

  - record: job:etcd_network_peer_received_bytes_total:rate5msum
    expr: sum(rate(etcd_network_peer_received_bytes_total[5m])) by (instance)
    labels:
      kubermatic: federate

  - record: job:etcd_network_peer_sent_bytes_total:rate5msum
    expr: sum(rate(etcd_network_peer_sent_bytes_total[5m])) by (instance)
    labels:
      kubermatic: federate

  - record: job:etcd_server_proposals_failed_total:rate5msum
    expr: sum(rate(etcd_server_proposals_failed_total[5m]))
    labels:
      kubermatic: federate

  - record: job:etcd_server_proposals_pending:sum
    expr: sum(etcd_server_proposals_pending)
    labels:
      kubermatic: federate

  - record: job:etcd_server_proposals_committed_total:rate5msum
    expr: sum(rate(etcd_server_proposals_committed_total[5m]))
    labels:
      kubermatic: federate

  - record: job:etcd_server_proposals_applied_total:rate5msum
    expr: sum(rate(etcd_server_proposals_applied_total[5m]))
    labels:
      kubermatic: federate

  - record: job:etcd_server_leader_changes_seen_total:changes1d
    expr: changes(etcd_server_leader_changes_seen_total[1d])
    labels:
      kubermatic: federate

  - record: job:etcd_mvcc_delete_total:rate5m
    expr: rate(etcd_mvcc_delete_total[5m])
    labels:
      kubermatic: federate

  - record: job:etcd_mvcc_put_total:rate5m
    expr: rate(etcd_mvcc_put_total[5m])
    labels:
      kubermatic: federate

  - record: job:etcd_mvcc_range_total:rate5m
    expr: rate(etcd_mvcc_range_total[5m])
    labels:
      kubermatic: federate

  - record: job:etcd_debugging_mvcc_watcher_total:rate5m
    expr: rate(etcd_debugging_mvcc_watcher_total[5m])
    labels:
      kubermatic: federate

  - record: job:etcd_mvcc_txn_total:rate5m
    expr: rate(etcd_mvcc_txn_total[5m])
    labels:
      kubermatic: federate

  - record: job:etcd_debugging_mvcc_keys_total:clone
    expr: etcd_debugging_mvcc_keys_total
    labels:
      kubermatic: federate

  - record: job:etcd_debugging_store_reads_total:rate5m
    expr: rate(etcd_debugging_store_reads_total[5m])
    labels:
      kubermatic: federate

  - record: job:etcd_debugging_store_writes_total:rate5m
    expr: rate(etcd_debugging_store_writes_total[5m])
    labels:
      kubermatic: federate

  - record: job:etcd_debugging_store_expires_total:rate5m
    expr: rate(etcd_debugging_store_expires_total[5m])
    labels:
      kubermatic: federate

  - record: job:apiserver_storage_size_bytes:sum
    expr: sum(apiserver_storage_size_bytes)
    labels:
      kubermatic: federate

- name: machine-controller
  rules:
  - alert: MachineControllerTooManyErrors
    annotations:
      message: Machine Controller in {{ $labels.namespace }} has too many errors in its loop.
    expr: |
      sum(rate(machine_controller_errors_total[5m])) by (namespace) > 0.01
    for: 20m
    labels:
      severity: warning

  - alert: MachineControllerMachineDeletionTakesTooLong
    annotations:
      message: Machine {{ $labels.machine }} of cluster {{ $labels.cluster }} is stuck in deletion for more than 30min.
    expr: (time() - machine_controller_machine_deleted) > 30*60
    for: 0m
    labels:
      severity: warning

  - alert: AWSInstanceCountTooHigh
    annotations:
      message: '{{ $labels.machine }} has more than one instance at AWS'
    expr: machine_controller_aws_instances_for_machine > 1
    for: 30m
    labels:
      severity: warning

  - alert: KubernetesAdmissionWebhookHighRejectionRate
    annotations:
      message: '{{ $labels.operation }} requests for Machine objects are failing (Admission) with a high rate. Consider checking the affected objects'
    expr: rate(apiserver_admission_webhook_admission_latencies_seconds_count{name="machine-controller.kubermatic.io-machines",rejected="true"}[5m]) > 0.01
    for: 5m
    labels:
      severity: warning

  - record: job:machine_controller_errors_total:rate5m
    expr: rate(machine_controller_errors_total[5m])
    labels:
      kubermatic: federate

  - record: job:machine_controller_workers:sum
    expr: sum(machine_controller_workers)
    labels:
      kubermatic: federate

  - record: job:machine_controller_machines_total:sum
    expr: sum(machine_controller_machines_total) by (kubelet_version, os, provider, size)
    labels:
      kubermatic: federate

  - record: job:machine_deployment_available_replicas:user:sum
    expr: sum(machine_deployment_available_replicas) by (name, namespace)
    labels:
      kubermatic: federate

  - record: job:machine_deployment_ready_replicas:user:sum
    expr: sum(machine_deployment_ready_replicas) by (name, namespace)
    labels:
      kubermatic: federate

  - record: job:machine_deployment_replicas:user:sum
    expr: sum(machine_deployment_replicas) by (name, namespace)
    labels:
      kubermatic: federate

  - record: job:machine_deployment_updated_replicas:user:sum
    expr: sum(machine_deployment_updated_replicas) by (name, namespace)
    labels:
      kubermatic: federate

- name: process.filedescriptors
  rules:
  - expr: process_open_fds / process_max_fds
    record: instance:fd_utilization

  - alert: FdExhaustionClose
    annotations:
      message: '{{ $labels.job }} instance {{ $labels.instance }} will exhaust its file descriptors soon'
    expr: |
      predict_linear(instance:fd_utilization[1h], 3600 * 4) > 1
    for: 10m
    labels:
      severity: warning

  - alert: FdExhaustionClose
    annotations:
      message: '{{ $labels.job }} instance {{ $labels.instance }} will exhaust its file descriptors soon'
    expr: |
      predict_linear(instance:fd_utilization[10m], 3600) > 1
    for: 10m
    labels:
      severity: critical

- name: kubernetes-nodes
  rules:
  - alert: KubernetesNodeNotReady
    annotations:
      message: '{{ $labels.node }} has been unready for more than an hour.'
    expr: kube_node_status_condition{condition="Ready",status="true"} == 0
    for: 30m
    labels:
      severity: warning

  - record: job:kube_node_info:count
    expr: count(kube_node_info)
    labels:
      kubermatic: federate

- name: kubernetes-absent
  rules:
  - alert: KubernetesApiserverDown
    annotations:
      message: Kubernetes apiserver has disappeared from Prometheus target discovery.
    expr: absent(up{job="apiserver"} == 1)
    for: 15m
    labels:
      severity: critical

  - alert: MachineControllerDown
    annotations:
      message: Machine controller has disappeared from Prometheus target discovery.
    expr: absent(up{job="machine-controller"} == 1)
    for: 15m
    labels:
      severity: critical

  - alert: UserClusterControllerDown
    annotations:
      message: User Cluster Controller has disappeared from Prometheus target discovery.
    expr: absent(up{job="usercluster-controller"} == 1)
    for: 15m
    labels:
      severity: critical

  - alert: KubeStateMetricsDown
    annotations:
      message: Kube-state-metrics has disappeared from Prometheus target discovery.
    expr: absent(up{job="kube-state-metrics"} == 1)
    for: 15m
    labels:
      severity: warning

  - alert: EtcdDown
    annotations:
      message: Etcd has disappeared from Prometheus target discovery.
    expr: absent(up{job="etcd"} == 1)
    for: 15m
    labels:
      severity: critical

  # This is triggered if the cluster does have nodes, but the cadvisor could
  # not successfully be scraped for whatever reason. An absent() on cadvisor
  # metrics is not a good alert because clusters could simply have no nodes
  # and hence no cadvisors.
  - alert: CAdvisorDown
    annotations:
      message: cAdvisor on {{ $labels.kubernetes_io_hostname }} could not be scraped.
    expr: up{job="cadvisor"} == 0
    for: 15m
    labels:
      severity: warning

  # This functions similarly to the cadvisor alert above.
  - alert: KubernetesNodeDown
    annotations:
      message: The kubelet on {{ $labels.kubernetes_io_hostname }} could not be scraped.
    expr: up{job="kubernetes-nodes"} == 0
    for: 15m
    labels:
      severity: warning
`

// prometheusRuleDNSResolverDownAlert contains the DNSResolverDown alerting rule for Prometheus.
const prometheusRuleDNSResolverDownAlert = `
  - alert: DNSResolverDown
    annotations:
      message: DNS resolver has disappeared from Prometheus target discovery.
    expr: absent(up{job="dns-resolver"} == 1)
    for: 15m
    labels:
      severity: warning
`

const prometheusRuleEnvoyAgentFederation = `
- name: kubermatic.envoy
  rules:
  - record: job:envoy_cluster_upstream_cx_active:sum
    expr: sum(envoy_cluster_upstream_cx_active)
    labels:
      kubermatic: federate

  - record: job:envoy_cluster_upstream_rq_total:irate1msum
    expr: sum(irate(envoy_cluster_upstream_rq_total[1m])) by (envoy_cluster_name)
    labels:
      kubermatic: federate

  - record: job:envoy_cluster_upstream_cx_rx_bytes_total:irate1msum
    expr: sum(irate(envoy_cluster_upstream_cx_rx_bytes_total[1m])) by (envoy_cluster_name)
    labels:
      kubermatic: federate

  - record: job:envoy_cluster_upstream_cx_tx_bytes_total:irate1msum
    expr: sum(irate(envoy_cluster_upstream_cx_tx_bytes_total[1m])) by (envoy_cluster_name)
    labels:
      kubermatic: federate

  - record: job:envoy_cluster_upstream_rq_time_bucket:rate1msum
    expr: sum(rate(envoy_cluster_upstream_rq_time_bucket[1m])) by (le, envoy_cluster_name)
    labels:
      kubermatic: federate
`

// prometheusRuleKonnectivity contains recording and alerting rules for Konnectivity server metrics.
// Recording rules are federated to the seed Prometheus for cluster-level monitoring.
const prometheusRuleKonnectivity = `
- name: kubermatic.konnectivity
  rules:
  - record: job:konnectivity_ready_backend_connections:count
    expr: konnectivity_network_proxy_server_ready_backend_connections
    labels:
      kubermatic: federate

  - record: job:konnectivity_dial_failure_rate:by_reason
    expr: sum(rate(konnectivity_network_proxy_server_dial_failure_count[5m])) by (reason)
    labels:
      kubermatic: federate

  - record: job:konnectivity_network_proxy_server_full_receive_channels:sum
    expr: sum(konnectivity_network_proxy_server_full_receive_channels)
    labels:
      kubermatic: federate

  - alert: KonnectivityNoAgents
    annotations:
      message: 'Cluster {{ $labels.cluster }} has no Konnectivity agents connected. New tunnels cannot be established.'
    expr: |
      job:konnectivity_ready_backend_connections:count == 0
    for: 2m
    labels:
      severity: critical

  - alert: KonnectivityAgentsUnhealthy
    annotations:
      message: 'Cluster {{ $labels.cluster }} has only {{ $value }} Konnectivity agent(s) connected (expected: 2).'
    expr: |
      job:konnectivity_ready_backend_connections:count < 2
    for: 3m
    labels:
      severity: warning

  - alert: KonnectivityHighDialFailureRate
    annotations:
      message: 'Cluster {{ $labels.cluster }} has {{ $value | humanize }} dial failures/sec (reason: {{ $labels.reason }}). Check agent and node health.'
    expr: |
      job:konnectivity_dial_failure_rate:by_reason > 0.1
    for: 10m
    labels:
      severity: warning
`

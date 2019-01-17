package prometheus

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

- name: kubermatic.machine_controller
  rules:
  - record: job:machine_controller_errors_total:rate5m
    expr: rate(machine_controller_errors_total[5m])
    labels:
      kubermatic: federate

- name: kubermatic.etcd
  rules:
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

  - record: job:etcd_debugging_mvcc_db_total_size_in_bytes:clone
    expr: etcd_debugging_mvcc_db_total_size_in_bytes
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

  - record: job:etcd_debugging_mvcc_delete_total:rate5m
    expr: rate(etcd_debugging_mvcc_delete_total[5m])
    labels:
      kubermatic: federate

  - record: job:etcd_debugging_mvcc_put_total:rate5m
    expr: rate(etcd_debugging_mvcc_put_total[5m])
    labels:
      kubermatic: federate

  - record: job:etcd_debugging_mvcc_range_total:rate5m
    expr: rate(etcd_debugging_mvcc_range_total[5m])
    labels:
      kubermatic: federate

  - record: job:etcd_debugging_mvcc_watcher_total:rate5m
    expr: rate(etcd_debugging_mvcc_watcher_total[5m])
    labels:
      kubermatic: federate

  - record: job:etcd_debugging_mvcc_txn_total:rate5m
    expr: rate(etcd_debugging_mvcc_txn_total[5m])
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

  - record: job:machine_controller_errors_total:rate5m
    expr: rate(machine_controller_errors_total[5m])
    labels:
      kubermatic: federate

  - record: job:machine_controller_workers:sum
    expr: sum(machine_controller_workers)
    labels:
      kubermatic: federate

  - record: job:machine_controller_machines:sum
    expr: sum(machine_controller_machines)
    labels:
      kubermatic: federate

- name: etcd
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
      description: '{{ $labels.job }} instance {{ $labels.instance }} will exhaust its file descriptors soon'
    expr: |
      predict_linear(instance:fd_utilization[10m], 3600) > 1
    for: 10m
    labels:
      severity: critical

- name: kubernetes-absent
  rules:
  - alert: KubernetesApiserverDown
    annotations:
      message: Kubernetes apiserver has disappeared from Prometheus target discovery.
    expr: absent(up{job="apiserver"} == 1)
    for: 15m
    labels:
      severity: critical

  - alert: KubernetesControllerManagerDown
    annotations:
      message: Kubernetes controller-manager has disappeared from Prometheus target discovery.
    expr: absent(up{job="controller-manager"} == 1)
    for: 15m
    labels:
      severity: critical

  - alert: KubernetesSchedulerDown
    annotations:
      message: Kubernetes scheduler has disappeared from Prometheus target discovery.
    expr: absent(up{job="scheduler"} == 1)
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

  - alert: EtcdDown
    annotations:
      message: Etcd has disappeared from Prometheus target discovery.
    expr: absent(up{job="etcd"} == 1)
    for: 15m
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
`

local k = import 'ksonnet/ksonnet.beta.3/k.libsonnet';

{
  _config+:: {
    namespace: 'monitoring-statefulset',

    prometheus+:: {
      name: 'testing',
      replicas: 3,
      external: 'https://prometheus.example.kubermatic.io',
      portName: 'web',
      storageSize: '10Gi',

      config+: std.manifestYamlDoc(
        (import 'config.jsonnet') {
          _config+:: {
            namespace: $._config.namespace,
          },
        },
      ),
    },
  },

  prometheus+: {
    local prometheusName = 'prometheus-' + $._config.prometheus.name,
    local prometheusLabels = { app: prometheusName },
    local serviceAccountName = prometheusName,

    serviceaccount+:
      local serviceAccount = k.core.v1.serviceAccount;
      serviceAccount.new(serviceAccountName) +
      serviceAccount.mixin.metadata.withNamespace($._config.namespace),

    service+:
      local service = k.core.v1.service;
      local servicePort = k.core.v1.service.mixin.spec.portsType;

      local prometheusPort = servicePort.newNamed($._config.prometheus.portName, 9090, $._config.prometheus.portName);
      service.new(prometheusName, prometheusLabels, prometheusPort) +
      service.mixin.metadata.withNamespace($._config.namespace) +
      service.mixin.metadata.withLabels(prometheusLabels),

    clusterrole+:
      local clusterRole = k.rbac.v1.clusterRole;
      local policyRule = clusterRole.rulesType;

      local rules = [
        policyRule.new() +
        policyRule.withApiGroups(['']) +
        policyRule.withResources(['nodes', 'services', 'endpoints', 'pods']) +
        policyRule.withVerbs(['get', 'list', 'watch']),

        policyRule.new() +
        policyRule.withApiGroups(['']) +
        policyRule.withResources(['configmaps']) +
        policyRule.withVerbs(['get']),

        policyRule.new() +
        policyRule.withNonResourceUrls('/metrics') +
        policyRule.withVerbs(['get']),
      ];

      clusterRole.new() +
      clusterRole.mixin.metadata.withName(prometheusName) +
      clusterRole.withRules(rules),

    clusterrolebinding+:
      local clusterRoleBinding = k.rbac.v1.clusterRoleBinding;

      clusterRoleBinding.new() +
      clusterRoleBinding.mixin.metadata.withName(prometheusName) +
      clusterRoleBinding.mixin.metadata.withNamespace($._config.namespace) +
      clusterRoleBinding.mixin.roleRef.withApiGroup('rbac.authorization.k8s.io') +
      clusterRoleBinding.mixin.roleRef.withName(prometheusName) +
      clusterRoleBinding.mixin.roleRef.mixinInstance({ kind: 'ClusterRole' }) +
      clusterRoleBinding.withSubjects([{
        kind: 'ServiceAccount',
        name: serviceAccountName,
        namespace: $._config.namespace,
      }]),

    config+:
      local cm = k.core.v1.configMap;
      cm.new(prometheusName + '-config', { 'prometheus.yml': $._config.prometheus.config }),

    statefulset+:
      local sts = k.apps.v1beta2.statefulSet;
      local container = sts.mixin.spec.template.spec.containersType;
      local containerPort = container.portsType;
      local containerVolumeMount = container.volumeMountsType;
      local volume = sts.mixin.spec.template.spec.volumesType;
      local volumeClaim = k.core.v1.persistentVolumeClaim;

      local prometheusArgs = [
        '--config.file=/etc/prometheus/config/prometheus.yml',
        '--storage.tsdb.no-lockfile',
        '--storage.tsdb.path=/prometheus',
        '--storage.tsdb.retention=360h',
        '--web.enable-lifecycle',
        '--web.external-url=' + $._config.prometheus.external,
        '--web.route-prefix=/',
      ];

      local probes = {
        initialDelaySeconds: 15,
        path: '/-/healthy',
        periodSeconds: 5,
        successThreshold: 1,
        timeouts: 5,
      };

      local volumeMounts = [
        containerVolumeMount.new(prometheusName + '-config', '/etc/prometheus/config/'),
      ];

      local c =
        container.new('prometheus', 'quay.io/prometheus/prometheus:v2.2.1')
        .withArgs(prometheusArgs)
        .withPorts(containerPort.newNamed($._config.prometheus.portName, 9090)) +
        container.mixin.resources.withRequests({ cpu: '100m', memory: '512Mi' }) +
        container.mixin.resources.withLimits({ cpu: 1, memory: '2Gi' }) +
        container.mixin.livenessProbe.httpGet.withPath(probes.path) +
        container.mixin.livenessProbe.httpGet.withPort($._config.prometheus.portName) +
        container.mixin.livenessProbe.withFailureThreshold(6) +
        container.mixin.livenessProbe.withPeriodSeconds(probes.periodSeconds) +
        container.mixin.livenessProbe.withSuccessThreshold(probes.successThreshold) +
        container.mixin.livenessProbe.withInitialDelaySeconds(probes.initialDelaySeconds) +
        container.mixin.livenessProbe.withTimeoutSeconds(probes.timeouts) +
        container.mixin.readinessProbe.httpGet.withPath(probes.path) +
        container.mixin.readinessProbe.httpGet.withPort($._config.prometheus.portName) +
        container.mixin.readinessProbe.withFailureThreshold(120) +
        container.mixin.readinessProbe.withInitialDelaySeconds(probes.initialDelaySeconds) +
        container.mixin.readinessProbe.withPeriodSeconds(probes.periodSeconds) +
        container.mixin.readinessProbe.withSuccessThreshold(probes.successThreshold) +
        container.mixin.readinessProbe.withTimeoutSeconds(probes.timeouts) +
        container.withVolumeMounts(
          volumeMounts + [
            containerVolumeMount.new(prometheusName + '-db', '/prometheus') +
            containerVolumeMount.withSubPath('prometheus-db'),
          ]
        );

      local cReloader =
        container.new('reloader', 'jimmidyson/configmap-reload')
        .withArgs([
          '--volume-dir=/etc/prometheus/config',
          // '--volume-dir=/etc/prometheus/rules',
          '--webhook-url=http://localhost:9090/-/reload',
        ]) +
        container.mixin.resources.withRequests({ cpu: '25m', memory: '16Mi' }) +
        container.mixin.resources.withLimits({ cpu: '100m', memory: '64Mi' }) +
        container.withVolumeMounts(volumeMounts);

      local v =
        volumeClaim.new() +
        volumeClaim.mixin.metadata.withName(prometheusName + '-db') +
        volumeClaim.mixin.spec.withAccessModes('ReadWriteOnce') +
        volumeClaim.mixin.spec.resources.withRequests({ storage: $._config.prometheus.storageSize }) +
        volumeClaim.mixin.spec.withStorageClassName('kubermatic-fast');

      sts.new('prometheus-' + $._config.prometheus.name, $._config.prometheus.replicas, [c, cReloader], [v]) +
      sts.mixin.metadata.withNamespace($._config.namespace) +
      sts.mixin.spec.selector.withMatchLabels(prometheusLabels) +
      sts.mixin.spec.withServiceName(prometheusName) +
      sts.mixin.spec.withRevisionHistoryLimit(10) +
      sts.mixin.spec.template.spec.securityContext.withRunAsNonRoot(true) +
      sts.mixin.spec.template.spec.securityContext.withRunAsUser(1000) +
      sts.mixin.spec.template.spec.securityContext.withFsGroup(2000) +
      sts.mixin.spec.template.spec.withVolumes([
        volume.fromConfigMap(prometheusName + '-config', prometheusName + '-config', []),
      ]),
  },
}.prometheus

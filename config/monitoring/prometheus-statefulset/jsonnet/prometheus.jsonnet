local k = import 'ksonnet/ksonnet.beta.3/k.libsonnet';

{
  _config+:: {
    namespace: 'monitoring-statefulset',

    prometheus+:: {
      name: 'kubermatic',
      replicas: 3,
      external: 'https://prometheus.example.kubermatic.io',
      portName: 'web',
      storageSize: '10Gi',
    },
  },

  prometheus+: {
    local prometheusName = 'prometheus-' + $._config.prometheus.name,
    local prometheusLabels = { app: prometheusName },

    serviceaccount+:
      local serviceAccount = k.core.v1.serviceAccount;
      serviceAccount.new(prometheusName) +
      serviceAccount.mixin.metadata.withNamespace($._config.namespace),

    service+:
      local service = k.core.v1.service;
      local servicePort = k.core.v1.service.mixin.spec.portsType;

      local prometheusPort = servicePort.newNamed($._config.prometheus.portName, 9090, $._config.prometheus.portName);
      service.new(prometheusName, prometheusLabels, prometheusPort) +
      service.mixin.metadata.withNamespace($._config.namespace) +
      service.mixin.metadata.withLabels(prometheusLabels),

    rolebindingdefault+:
      local roleBinding = k.rbac.v1.roleBinding;

      roleBinding.new() +
      roleBinding.mixin.metadata.withName(prometheusName) +
      roleBinding.mixin.metadata.withNamespace('default') +
      roleBinding.mixin.roleRef.withApiGroup('rbac.authorization.k8s.io') +
      roleBinding.mixin.roleRef.withName(prometheusName) +
      roleBinding.mixin.roleRef.mixinInstance({ kind: 'Role' }) +
      roleBinding.withSubjects([{ kind: 'ServiceAccount', name: prometheusName, namespace: $._config.namespace }]),
    clusterrole+:
      local clusterRole = k.rbac.v1.clusterRole;
      local policyRule = clusterRole.rulesType;

      local rules = [
        policyRule.new() +
        policyRule.withApiGroups(['']) +
        policyRule.withResources(['nodes/metrics']) +
        policyRule.withVerbs(['get']),

        policyRule.new() +
        policyRule.withNonResourceUrls('/metrics') +
        policyRule.withVerbs(['get']),
      ];

      clusterRole.new() +
      clusterRole.mixin.metadata.withName(prometheusName) +
      clusterRole.withRules(rules),

    role:
      local role = k.rbac.v1.role;
      local policyRule = role.rulesType;

      local configmapRule =
        policyRule.new() +
        policyRule.withApiGroups(['']) +
        policyRule.withResources([
          'configmaps',
        ]) +
        policyRule.withVerbs(['get']);

      role.new() +
      role.mixin.metadata.withName(prometheusName + '-config') +
      role.mixin.metadata.withNamespace($._config.namespace) +
      role.withRules(configmapRule),
    statefulset+:
      local sts = k.apps.v1beta2.statefulSet;
      local container = sts.mixin.spec.template.spec.containersType;
      local containerPort = container.portsType;
      local containerVolumeMount = container.volumeMountsType;
      local volumeClaim = k.core.v1.persistentVolumeClaim;

      local prometheusArgs = [
        '--config.file=/etc/prometheus/prometheus.yml',
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
        container.withVolumeMounts([
          containerVolumeMount.new(prometheusName + '-db', '/prometheus') +
          containerVolumeMount.withSubPath('prometheus-db'),
        ]);

      local v =
        volumeClaim.new() +
        volumeClaim.mixin.metadata.withName(prometheusName + '-db') +
        volumeClaim.mixin.spec.withAccessModes('ReadWriteOnce') +
        volumeClaim.mixin.spec.resources.withRequests({ storage: $._config.prometheus.storageSize }) +
        volumeClaim.mixin.spec.withStorageClassName('kubermatic-fast');

      sts.new('prometheus-' + $._config.prometheus.name, $._config.prometheus.replicas, [c], [v]) +
      sts.mixin.metadata.withNamespace($._config.namespace) +
      sts.mixin.spec.selector.withMatchLabels(prometheusLabels) +
      sts.mixin.spec.withServiceName(prometheusName) +
      sts.mixin.spec.withRevisionHistoryLimit(10) +
      sts.mixin.spec.template.spec.securityContext.withRunAsNonRoot(true) +
      sts.mixin.spec.template.spec.securityContext.withRunAsUser(1000) +
      sts.mixin.spec.template.spec.securityContext.withFsGroup(2000),
  },
}.prometheus

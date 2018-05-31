local drone = import 'drone/drone.libsonnet';

// Please take a look at the README.md for instructions on how to build this.

{
  workspace: drone.workspace.new('/go', 'src/github.com/kubermatic/kubermatic'),
  pipeline: {
    local goImage = 'golang:1.9',

    '0-dep': drone.step.new('metalmatze/dep:0.4.1', commands=['cd api', 'dep status -v']),
    '1-gofmt': drone.step.new(goImage, commands=['cd api', 'make gofmt']),
    '2-verify-codegen': drone.step.new(goImage, commands=['cd api', './hack/verify-codegen.sh']),

    // Linting
    '3-license-validation': drone.step.new('metalmatze/wwhrd:1.9', group='lint', commands=['cd api', 'wwhrd check -f ../allowed_licensed.yaml']),
    '3-lint': drone.step.new('metalmatze/gometalinter:1.9', group='lint', commands=['cd api', 'make lint']),
    '3-verify-swagger-spec': drone.step.new(goImage, group='lint', commands=['cd api', './hack/verify-swagger.sh']),

    // Building
    '4-test': drone.step.new(goImage, group='build', commands=['cd api', 'make test']),
    '4-build': drone.step.new(goImage, group='build', commands=['cd api', 'CGO_ENABLED=0 make build']),
    '4-write-version': drone.step.new('ubuntu', group='build', commands= [
      'cd config',
      'sed -i "s/{API_IMAGE_TAG}/${DRONE_TAG=$DRONE_COMMIT}/g" versions-values.yaml',
      'sed -i "s/{CONTROLLER_IMAGE_TAG}/${DRONE_TAG=$DRONE_COMMIT}/g" versions-values.yaml',
      'sed -i "s/{BARE_METAL_PROVIDER_IMAGE_TAG}/${DRONE_TAG=$DRONE_COMMIT}/g" versions-values.yaml',
      'cat versions-values.yaml',
    ]),

    local dockerSecrets = ['docker_username', 'docker_password'],

    // Push Always

    '5-kubermatic-docker-always': drone.step.docker.new(
        'kubermatic/api',
        group='push-always',
        secrets=dockerSecrets,
        dockerfile='api/Dockerfile',
        tags=['${DRONE_COMMIT}'],
        context='api',
      ),
    '5-kubermatic-installer-docker-always': drone.step.docker.new(
      'kubermatic/installer',
      group='push-always',
      secrets=dockerSecrets,
      dockerfile='config/Dockerfile',
      tags=['${DRONE_COMMIT}'],
      context='config',
      ),

    // Push Master

    local whenBranchMaster = {when: {branch: 'master'}},
    '6-kubermatic-docker-master': drone.step.docker.new(
        'kubermatic/api',
        group='push-master',
        secrets=dockerSecrets,
        dockerfile='api/Dockerfile',
        tags=['master'],
        context='api',
      ) + whenBranchMaster,
    '6-kubermatic-installer-docker-master': drone.step.docker.new(
      'kubermatic/installer',
      group='push-master',
      secrets=dockerSecrets,
      dockerfile='config/Dockerfile',
      tags=['master'],
      context='config',
      ) + whenBranchMaster,

    // Push Release

    local whenEventTag = {when: {event: ['tag']}},
    '7-kubermatic-docker-release': drone.step.docker.new(
        'kubermatic/api',
        group='push-master',
        secrets=dockerSecrets,
        dockerfile='api/Dockerfile',
        tags=['${DRONE_TAG}', 'latest'],
        context='api',
      ) + whenEventTag,
    '7-kubermatic-installer-docker-release': drone.step.docker.new(
        'kubermatic/installer',
        group='push-release',
        secrets=dockerSecrets,
        dockerfile='config/Dockerfile',
        tags=['${DRONE_TAG}', 'latest'],
        context='config',
      ) + whenEventTag,

    '8-sync-charts': drone.step.new('alpine:3.7', commands=[
      'apk add --no-cache -U git',
      'git config --global user.email "dev@loodse.com"',
      'git config --global user.name "drone"',
      'export INSTALLER_DIR="/go/src/github.com/kubermatic/kubermatic-installer"',
      'git clone https://github.com/kubermatic/kubermatic-installer.git $INSTALLER_DIR && mkdir -p $INSTALLER_DIR/charts',
      'cp -r config/cert-manager config/certs config/kubermatic config/monitoring config/nginx-ingress-controller config/nodeport-proxy config/oauth $INSTALLER_DIR/charts',
      'cd $INSTALLER_DIR',
      'git add . && git commit -m "Synchronising helm charts from commit ${DRONE_COMMIT}"',
      'git tag ${DRONE_TAG}',
      'git push origin master --tags',
    ]) + {
      when: {
        event: [ 'tag' ],
        branch: {
          include: [ 'master', 'release/*' ],
          exclude: [ 'release/v1.*' ],
        },
      },
    },

    // Deployments

    local charts = [
      {namespace: 'ingress-nginx', name: 'nginx', path: 'config/nginx-ingress-controller/'},
      {namespace: 'oauth', name: 'oauth', path: 'config/oauth/'},
      {namespace: 'kubermatic', name: 'kubermatic', path: 'config/kubermatic/'},
      {namespace: 'cert-manager', name: 'cert-manager', path: 'config/cert-manager/'},
      {namespace: 'default', name: 'certs', path: 'config/certs/'},
      {namespace: 'nodeport-proxy', name: 'nodeport-proxy', path: 'config/nodeport-proxy/'},
    ],

    local chartsMonitoring = [
      {namespace: 'monitoring', name: 'prometheus-operator', path: 'config/monitoring/prometheus-operator/'},
      {namespace: 'monitoring', name: 'node-exporter', path: 'config/monitoring/node-exporter/'},
      {namespace: 'monitoring', name: 'kube-state-metrics', path: 'config/monitoring/kube-state-metrics/'},
      {namespace: 'monitoring', name: 'grafana', path: 'config/monitoring/grafana/'},
      {namespace: 'monitoring', name: 'alertmanager', path: 'config/monitoring/alertmanager/'},
      {namespace: 'monitoring', name: 'prometheus', path: 'config/monitoring/prometheus/'},
    ],

    local versionsValues = ' --values config/versions-values.yaml',
    local tillerNamespace = ' --tiller-namespace=kubermatic-installer',

    // dev

    '9-deploy-dev': drone.step.new('kubeciio/helm') + {
        helm: 'upgrade --install' + tillerNamespace + versionsValues,
        secrets: [
          {source: 'kubeconfig_dev', target: 'kubeconfig'},
          {source: 'values_dev', target: 'values'},
        ],
        charts: charts,
        values: [ 'values' ],
      } + {when: {branch: 'master'}},

    // cloud

    '10-deploy-cloud-europe': drone.step.new('kubeciio/helm', group='deploy-cloud') + {
        helm: 'upgrade --install --kube-context=europe-west3-c-1' + tillerNamespace + versionsValues,
        secrets: [
          {source: 'kubeconfig_cloud', target: 'kubeconfig'},
          {source: 'values_cloud_eu', target: 'values'},
        ],
        charts: charts + chartsMonitoring,
        values: [ 'values' ],
      } + {when: {branch: 'master'}},

    '10-deploy-cloud-us': drone.step.new('kubeciio/helm', group='deploy-cloud') + {
        helm: 'upgrade --install --kube-context=us-central1-c-1' + tillerNamespace + versionsValues,
        secrets: [
          {source: 'kubeconfig_cloud', target: 'kubeconfig'},
          {source: 'values_cloud_us', target: 'values'},
        ],
        values: [ 'values' ],
        charts: charts,
      } + {when: {branch: 'master'}},

    '10-deploy-cloud-asia': drone.step.new('kubeciio/helm', group='deploy-cloud') + {
        helm: 'upgrade --install --kube-context=asia-east1-a-1' + tillerNamespace + versionsValues,
        secrets: [
          {source: 'kubeconfig_cloud', target: 'kubeconfig'},
          {source: 'values_cloud_as', target: 'values'},
        ],
        values: [ 'values' ],
        charts: charts,
      } + {when: {branch: 'master'}},

    // Slack
    '11-slack': drone.step.new('kubermaticbot/drone-slack', group='slack') + {
      webhook: 'https://hooks.slack.com/services/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',
      username: 'drone',
      icon_url: 'https://avatars2.githubusercontent.com/u/2181346?v=4&s=200',
      channel: 'dev',
      template: '${DRONE_COMMIT_AUTHOR} deployed a new API & Controller to dev & cloud. :heart:',
      when: {status:['success'], branch: 'master'},
    },
    '11-slack-failure': drone.step.new('kubermaticbot/drone-slack', group='slack') + {
      webhook: 'https://hooks.slack.com/services/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',
      username: 'drone',
      icon_url: 'https://avatars2.githubusercontent.com/u/2181346?v=4&s=200',
      recipient: '${DRONE_COMMIT_AUTHOR}',
      image_url: 'https://media.giphy.com/media/m6tmCnGCNvTby/giphy-downsized.gif',
      template: 'Your build failed! Shame. Shame. Shame.
      ${DRONE_BUILD_LINK}',
      author_recipient_mapping: [
        'alvaroaleman=alvaro',
        'guusvw=guus',
        'j3ank=eugenia',
        'kgroschoff=kristin',
        'metalmatze=matthias',
        'mrIncompetent=henrik',
        'p0lyn0mial=lukasz',
        'scheeles=sebastian',
        'kdomanski=kamil',
        'kron4eg=artiom.diomin',
        'thz=thz',
        'cbrgm=Chris',
        'toschneck=Tobi',
      ],
      when: {status: [ 'failure']},
    },
  }
}

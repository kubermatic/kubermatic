local drone = import 'drone/drone.libsonnet';

// Please take a look at the README.md for instructions on how to build this.

{
  workspace: drone.workspace.new('/go', 'src/github.com/kubermatic/kubermatic'),
  pipeline: {

    local goImage = 'golang:1.11.0',
    local dockerSecrets = ['docker_username', 'docker_password'],
    local whenBranchMaster = { when: { branch: 'master' } },
    local whenBranchRelease = { when: { branch: 'release/v2.7.*' } },
    local whenEventTag = { when: { event: ['tag'] } },
    local charts = [
      { namespace: 'kubermatic', name: 'kubermatic', path: 'config/kubermatic/' },
      { namespace: 'nodeport-proxy', name: 'nodeport-proxy', path: 'config/nodeport-proxy/' },
      { namespace: 'minio', name: 'minio', path: 'config/minio/' },
    ],
    local chartsMaster = [
      { namespace: 'kubermatic', name: 'kubermatic-master', path: 'config/kubermatic/master' },
      { namespace: 'ingress-nginx', name: 'nginx', path: 'config/nginx-ingress-controller/' },
      { namespace: 'oauth', name: 'oauth', path: 'config/oauth/' },
      { namespace: 'cert-manager', name: 'cert-manager', path: 'config/cert-manager/' },
      { namespace: 'default', name: 'certs', path: 'config/certs/' },
      { namespace: 'iap', name: 'iap', path: 'config/iap/' },
      { namespace: 'monitoring', name: 'node-exporter', path: 'config/monitoring/node-exporter/' },
      { namespace: 'monitoring', name: 'kube-state-metrics', path: 'config/monitoring/kube-state-metrics/' },
      { namespace: 'monitoring', name: 'grafana', path: 'config/monitoring/grafana/' },
      { namespace: 'monitoring', name: 'alertmanager', path: 'config/monitoring/alertmanager/' },
      { namespace: 'monitoring', name: 'prometheus', path: 'config/monitoring/prometheus/' },
    ],
    local versionsValues = ' --values config/versions-values.yaml',
    local tillerNamespace = ' --tiller-namespace=kubermatic-installer',
    local e2eStep = {
      secrets: [
        { source: 'kubeconfig_dev', target: 'kubeconfig' },
        { source: 'aws_1.10.5_cluster_yaml', target: 'cluster_yaml' },
        { source: 'aws_1.10.5_node_yaml', target: 'node_yaml' },
      ],
      commands: [
        'echo "$KUBECONFIG" | base64 -d > /tmp/kubeconfig',
        'echo "$CLUSTER_YAML" > /tmp/cluster.yaml',
        'echo "$NODE_YAML" > /tmp/node.yaml',
        '/kubermatic-e2e -kubeconfig=/tmp/kubeconfig -kubermatic-cluster=/tmp/cluster.yaml -kubermatic-node=/tmp/node.yaml',
      ],
    },


    '0-dep': drone.step.new('metalmatze/dep:0.5.0') + {
      commands: [
        'cd api',
        'dep ensure -v',
        '[[ -z "$(git diff)" ]]'
      ],
    },

    '1-gofmt': drone.step.new(goImage) + {
      commands: [
        'cd api',
        'make gofmt',
      ],
    },

    '2-verify-codegen': drone.step.new(goImage) + {
      commands: [
        'cd api',
        './hack/verify-codegen.sh',
      ],
    },

    // Linting
    '3-license-validation': drone.step.new('metalmatze/wwhrd:1.9', group='lint') + {
      commands: [
        'cd api',
        'wwhrd check -f ../allowed_licensed.yaml',
      ],
    },

    '3-lint': drone.step.new('quay.io/kubermatic/gometalinter:latest', group='lint') + {
      commands: [
        'cd api',
        'make lint',
      ],
    },

    '3-verify-swagger-spec': drone.step.new(goImage, group='lint') + {
      commands: [
        'cd api',
        './hack/verify-swagger.sh',
      ],
    },

    '3-verify-addons-up-to-date': drone.step.new('docker:dind', group='lint') + {
      commands: ['./api/hack/verify-addon-version.sh'],
      volumes: ['/var/run/docker.sock:/var/run/docker.sock'],
    },

    // Building
    '4-test': drone.step.new(goImage, group='build') + {
      commands: [
        'cd api',
        'make test',
      ],
    },

    '4-build': drone.step.new(goImage, group='build') + {
      commands: [
        'cd api',
        'make build',
      ],
    },

    '4-write-version': drone.step.new('ubuntu', group='build') + {
      commands: [
        'cd config',
        'sed -i "s/{API_IMAGE_TAG}/${DRONE_TAG=$DRONE_COMMIT}/g" versions-values.yaml',
        'sed -i "s/{CONTROLLER_IMAGE_TAG}/${DRONE_TAG=$DRONE_COMMIT}/g" versions-values.yaml',
        'sed -i "s/{BARE_METAL_PROVIDER_IMAGE_TAG}/${DRONE_TAG=$DRONE_COMMIT}/g" versions-values.yaml',
        'cat versions-values.yaml',
      ],
    },

    // Push Master
    '5-kubermatic-docker-master': drone.step.docker.new('kubermatic/api', group='push-master') + {
      secrets: dockerSecrets,
      dockerfile: 'api/Dockerfile',
      tags: ['master', '${DRONE_COMMIT}'],
      context: 'api',
    } + whenBranchMaster,


    // Push Release
    '6-kubermatic-docker-release': drone.step.docker.new('kubermatic/api', group='push-master') + {
      secrets: dockerSecrets,
      dockerfile: 'api/Dockerfile',
      tags: ['${DRONE_TAG}', 'latest'],
      context: 'api',
    } + whenEventTag,

    // e2e
    '6-kubermatic-e2e-docker-push-on-master': drone.step.docker.new('quay.io/kubermatic/e2e') + {
      secrets: [
        { source: 'docker_quay_username', target: 'docker_username' },
        { source: 'docker_quay_password', target: 'docker_password' },
      ],
      dockerfile: 'api/Dockerfile.e2e',
      tags: ['latest'],
      context: 'api',
      registry: 'quay.io',
    } + whenBranchMaster,

    '6-kubermatic-e2e-docker-push-on-tag': drone.step.docker.new('quay.io/kubermatic/e2e') + {
      secrets: [
        { source: 'docker_quay_username', target: 'docker_username' },
        { source: 'docker_quay_password', target: 'docker_password' },
      ],
      dockerfile: 'api/Dockerfile.e2e',
      tags: ['${DRONE_TAG}'],
      context: 'api',
      registry: 'quay.io',
    } + whenEventTag,

    '7-sync-charts': drone.step.new('alpine:3.7') + {
      commands: [
        'apk add --no-cache -U git bash openssh',
        'git config --global user.email "dev@loodse.com"',
        'git config --global user.name "drone"',
        'cd api && ./hack/sync-charts.sh ${DRONE_BRANCH} ../config',
      ],
      when: {
        branch: {
          include: ['release/*'],
          exclude: ['release/v1.*', 'release/*cherry*'],
        },
      },
    },

    // deploy dev
    '8-deploy-dev': drone.step.new('kubeciio/helm') + {
      helm: 'upgrade --install --wait --timeout 300' + tillerNamespace + versionsValues,
      secrets: [
        { source: 'kubeconfig_dev', target: 'kubeconfig' },
        { source: 'values_dev', target: 'values' },
      ],
      charts: charts + chartsMaster,
      values: ['values'],
    } + whenBranchMaster,

    // deploy run
    '9-deploy-run': drone.step.new('kubeciio/helm') + {
      helm: 'upgrade --install --wait --timeout 300' + tillerNamespace + versionsValues,
      secrets: [
        { source: 'kubeconfig_run', target: 'kubeconfig' },
        { source: 'values_run', target: 'values' },
      ],
      charts: charts + chartsMaster,
      values: ['values'],
    } + whenBranchRelease,

    // deploy cloud
    '9-deploy-cloud-europe': drone.step.new('kubeciio/helm', group='deploy-cloud') + {
      helm: 'upgrade --install --wait --timeout 300 --kube-context=europe-west3-c-1' + tillerNamespace + versionsValues,
      secrets: [
        { source: 'kubeconfig_cloud', target: 'kubeconfig' },
        { source: 'values_cloud_eu', target: 'values' },
      ],
      charts: charts + chartsMaster,
      values: ['values'],
    } + whenBranchMaster,

    '9-deploy-cloud-us': drone.step.new('kubeciio/helm', group='deploy-cloud') + {
      helm: 'upgrade --install --wait --timeout 300 --kube-context=us-central1-c-1' + tillerNamespace + versionsValues,
      secrets: [
        { source: 'kubeconfig_cloud', target: 'kubeconfig' },
        { source: 'values_cloud_us', target: 'values' },
      ],
      values: ['values'],
      charts: charts,
    } + whenBranchMaster,

    '9-deploy-cloud-asia': drone.step.new('kubeciio/helm', group='deploy-cloud') + {
      helm: 'upgrade --install --wait --timeout 300 --kube-context=asia-east1-a-1' + tillerNamespace + versionsValues,
      secrets: [
        { source: 'kubeconfig_cloud', target: 'kubeconfig' },
        { source: 'values_cloud_as', target: 'values' },
      ],
      values: ['values'],
      charts: charts,
    } + whenBranchMaster,

    // run e2e tests
    '10-e2e-on-master': drone.step.new('quay.io/kubermatic/e2e:latest') + e2eStep + whenBranchMaster,
    // the default value 'no_such_tag' will prevent YAML parsing error on non-tag builds
    '10-e2e-on-tag': drone.step.new('quay.io/kubermatic/e2e:${DRONE_TAG=no_such_tag}') + e2eStep + whenEventTag,

    // Slack
    '11-slack': drone.step.new('kubermaticbot/drone-slack', group='slack') + {
      webhook: 'https://hooks.slack.com/services/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',
      username: 'drone',
      icon_url: 'https://avatars2.githubusercontent.com/u/2181346?v=4&s=200',
      channel: 'dev',
      template: '${DRONE_COMMIT_AUTHOR} deployed a new API & Controller to dev & cloud. :heart:',
      when: { status: ['success'], branch: 'master' },
    },

    '11-slack-failure': drone.step.new('kubermaticbot/drone-slack', group='slack') + {
      webhook: 'https://hooks.slack.com/services/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',
      username: 'drone',
      icon_url: 'https://avatars2.githubusercontent.com/u/2181346?v=4&s=200',
      recipient: '${DRONE_COMMIT_AUTHOR}',
      image_url: 'https://media.giphy.com/media/m6tmCnGCNvTby/giphy-downsized.gif',
      template: 'Your build failed! Shame. Shame. Shame.\n      ${DRONE_BUILD_LINK}',
      author_recipient_mapping: [
        'alvaroaleman=alvaro',
        'cbrgm=christian',
        'glower=igor.komlew',
        'guusvw=guus',
        'j3ank=eugenia',
        'kdomanski=kamil',
        'kgroschoff=kristin.groschoff',
        'kron4eg=artiom',
        'mrIncompetent=henrik',
        'p0lyn0mial=lukasz',
        'pkavajin=patrick',
        'scheeles=sebastian',
        'thz=tobias.hintze',
        'toschneck=tobias.schneck',
        'xrstf=christoph',
        'xmudrii=marko',
        'maciaszczykm=marcin',
        'floreks=sebastian.florek',
      ],
      when: { status: ['failure'] },
    },
  },
}

#/usr/bin/env bash

set -euo pipefail

## CI run conformance tester
source ./api/hack/lib.sh

### Defaults
export VERSIONS=${VERSIONS_TO_TEST:-"v1.12.4"}
export EXCLUDE_DISTRIBUTIONS=${EXCLUDE_DISTRIBUTIONS:-ubuntu,centos}
export ONLY_TEST_CREATION=${ONLY_TEST_CREATION:-false}
provider=${PROVIDER:-"aws"}
export WORKER_NAME=${BUILD_ID}
if [[ "${KUBERMATIC_NO_WORKER_NAME:-}" = "true" ]]; then
  WORKER_NAME=""
fi

mkdir -p $HOME/.ssh
if ! [[ -e HOME/.ssh/id_rsa.pub ]]; then
  echo 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCo3amVmCkIZo4cgj2kjU2arZKlzzOhaOuveH9aJbL4mlVHVsEcVk+RSty4AMK1GQL3+Ii7iGicKWwge4yefc75aOtUncfF01rnBsNvi3lOqJR/6POHy4OnPXJElvEn7jii/pAUeyr8halBezQTUkvRiUtlJo6oEb2dRN5ujyFm5TuIxgM0UFVGBRoD0agGr87GaQsUahf+PE1zHEid+qQPz7EdMo8/eRNtgikhBG1/ae6xRstAi0QU8EgjKvK1ROXOYTlpTBFElApOXZacH91WvG0xgPnyxIXoKtiCCNGeu/0EqDAgiXfmD2HK/WAXwJNwcmRvBaedQUS4H0lNmvj5' \
    > $HOME/.ssh/id_rsa.pub
fi
chmod 0700 $HOME/.ssh

if [[ -n ${OPENSHIFT:-} ]]; then
  OPENSHIFT_ARG="-openshift=true"
  export VERSIONS=${OPENSHIFT_VERSION}
  OPENSHIFT_HELM_ARGS="--set-string=kubermatic.controller.featureGates=OpenIDAuthPlugin=true
 --set-string=kubermatic.auth.caBundle=$(cat /etc/oidc-data/oidc-ca-file|base64 -w0)
 --set-string=kubermatic.auth.tokenIssuer=$OIDC_ISSUER_URL
 --set-string=kubermatic.auth.issuerClientID=$OIDC_ISSUER_CLIENT_ID
 --set-string=kubermatic.auth.issuerClientSecret=$OIDC_ISSUER_CLIENT_SECRET"
fi

if [[ $provider == "aws" ]]; then
  EXTRA_ARGS="-aws-access-key-id=${AWS_E2E_TESTS_KEY_ID}
     -aws-secret-access-key=${AWS_E2E_TESTS_SECRET}"
elif [[ $provider == "packet" ]]; then
  EXTRA_ARGS="-packet-api-key=${PACKET_API_KEY}
     -packet-project-id=${PACKET_PROJECT_ID}"
elif [[ $provider == "gcp" ]]; then
  EXTRA_ARGS="-gcp-service-account=${GOOGLE_SERVICE_ACCOUNT}"
elif [[ $provider == "azure" ]]; then
  EXTRA_ARGS="-azure-client-id=${AZURE_E2E_TESTS_CLIENT_ID}
    -azure-client-secret=${AZURE_E2E_TESTS_CLIENT_SECRET}
    -azure-tenant-id=${AZURE_E2E_TESTS_TENANT_ID}
    -azure-subscription-id=${AZURE_E2E_TESTS_SUBSCRIPTION_ID}"
elif [[ $provider == "digitalocean" ]]; then
  EXTRA_ARGS="-digitalocean-token=${DO_E2E_TESTS_TOKEN}"
elif [[ $provider == "hetzner" ]]; then
  EXTRA_ARGS="-hetzner-token=${HZ_E2E_TOKEN}"
elif [[ $provider == "openstack" ]]; then
  EXTRA_ARGS="-openstack-domain=${OS_DOMAIN}
    -openstack-tenant=${OS_TENANT_NAME}
    -openstack-username=${OS_USERNAME}
    -openstack-password=${OS_PASSWORD}"
elif [[ $provider == "vsphere" ]]; then
  EXTRA_ARGS="-vsphere-username=${VSPHERE_E2E_USERNAME}
    -vsphere-password=${VSPHERE_E2E_PASSWORD}"
elif [[ $provider == "kubevirt" ]]; then
  EXTRA_ARGS="-kubevirt-kubeconfig=${KUBEVIRT_E2E_TESTS_KUBECONFIG}"
fi

# Needed when running in kind
which kind && EXTRA_ARGS="$EXTRA_ARGS -create-oidc-token=true"

kubermatic_delete_cluster="true"
if [ -n "${UPGRADE_TEST_BASE_HASH:-}" ]; then
  kubermatic_delete_cluster="false"
fi

timeout -s 9 90m ./api/_build/conformance-tests ${EXTRA_ARGS:-} \
  -debug \
  -worker-name=${WORKER_NAME} \
  -kubeconfig=$KUBECONFIG \
  -kubermatic-nodes=3 \
  -kubermatic-parallel-clusters=1 \
  -name-prefix=prow-e2e \
  -reports-root=/reports \
  -cleanup-on-start=false \
  -run-kubermatic-controller-manager=false \
  -versions="$VERSIONS" \
  -providers=$provider \
  -only-test-creation="${ONLY_TEST_CREATION}" \
  -exclude-distributions="${EXCLUDE_DISTRIBUTIONS}" \
  ${OPENSHIFT_ARG:-} \
  -kubermatic-delete-cluster=${kubermatic_delete_cluster} \
  -print-ginkgo-logs=true \

# No upgradetest, just exit
if [[ -z ${UPGRADE_TEST_BASE_HASH:-} ]]; then
  echodate "Success!"
  exit 0
fi

which kind && echodate "Upgrade tests are not supported yet with kind" && exit 1

echodate "Checking out current version of Kubermatic"
git checkout ${GIT_HEAD_HASH}
build_tag_if_not_exists

echodate "Installing current version of Kubermatic"
retry 3 helm upgrade --install --force --atomic --timeout 300 \
  --tiller-namespace=$NAMESPACE \
  --set=kubermatic.isMaster=true \
  --set-string=kubermatic.controller.addons.kubernetes.image.tag=${GIT_HEAD_HASH} \
  --set-string=kubermatic.controller.addons.kubernetes.image.repository=127.0.0.1:5000/kubermatic/addons \
  --set-string=kubermatic.controller.addons.openshift.image.tag=${GIT_HEAD_HASH} \
  --set-string=kubermatic.controller.addons.openshift.image.repository=127.0.0.1:5000/kubermatic/openshift_addons \
  --set-string=kubermatic.controller.image.tag=${GIT_HEAD_HASH} \
  --set-string=kubermatic.controller.image.repository=127.0.0.1:5000/kubermatic/api \
  --set-string=kubermatic.api.image.repository=127.0.0.1:5000/kubermatic/api \
  --set-string=kubermatic.api.image.tag=${GIT_HEAD_HASH} \
  --set-string=kubermatic.masterController.image.tag=${GIT_HEAD_HASH} \
  --set-string=kubermatic.masterController.image.repository=127.0.0.1:5000/kubermatic/api \
  --set-string=kubermatic.ui.image.tag=${LATEST_DASHBOARD} \
  --set-string=kubermatic.kubermaticImage=127.0.0.1:5000/kubermatic/api \
  --set-string=kubermatic.dnatcontrollerImage=127.0.0.1:5000/kubermatic/kubeletdnat-controller \
  --set-string=kubermatic.worker_name=$BUILD_ID \
  --set=kubermatic.ingressClass=non-existent \
  --set=kubermatic.checks.crd.disable=true \
  ${OPENSHIFT_HELM_ARGS:-} \
  --values ${VALUES_FILE} \
  --namespace $NAMESPACE \
  kubermatic-$BUILD_ID ./config/kubermatic/
echodate "Successfully installed current version of Kubermatic"

# We have to rebuild it so it is based on the newer Kubermatic
echodate "Building conformance-tests cli"
# Force rebuild and let go decide if thats needed rather than make
rm -f api/_build/conformance-tests
time make -C api conformance-tests

echodate "Running conformance tester with existing cluster"

# We increase the number of nodes to make sure creation
# of nodes still work
timeout -s 9 60m ./api/_build/conformance-tests $EXTRA_ARGS \
  -debug \
  -existing-cluster-label=worker-name=$BUILD_ID \
  -worker-name=$BUILD_ID \
  -kubeconfig=$KUBECONFIG \
  -datacenters=$DATACENTERS_FILE \
  -kubermatic-nodes=5 \
  -kubermatic-parallel-clusters=1 \
  -kubermatic-delete-cluster=true \
  -name-prefix=prow-e2e \
  -reports-root=/reports \
  -cleanup-on-start=false \
  -versions="$VERSIONS" \
  -providers=$provider \
  -only-test-creation="${ONLY_TEST_CREATION}" \
  -exclude-distributions="${EXCLUDE_DISTRIBUTIONS}" \
  ${OPENSHIFT_ARG:-} \
  -print-ginkgo-logs=true \

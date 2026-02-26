#!/usr/bin/env bash

###
### This script can be used to run a whole range of conformance tests against
### a single KKP seed cluster. Use this to verify somewhat quickly that all
### provider/OS/release combinations work as expected.
###
### This script is not part of the KKP repo as it is very much tweaked towards
### our QA process and does not rely on Vault, but rather on a Preset. This is
### important as during QA we usually do not use the regular e2e credentials,
### for example for AWS we have a dedicated account.
###
### To use this script, COPY it into your local kubermatic repository (in the
### root directory):
###
### $ cp conformance-tester.sh ~/go/k8c.io/kubermatic
### $ cd ~/go/k8c.io/kubermatic
###
### When using RHEL, make sure you're logged into Vault:
###
### $ export VAULT_ADDR=https://vault.kubermatic.com/
### $ vault login --method=oidc --path=loodse
###
### Before running the script, take a look at the default variable values in here.
### For example, RELEASES often needs to be adjusted for new KKP releases, as the
### range of supported Kubernetes releases changes.
###
### Now you can run tests. Make sure to not overdo it, be nice to the seed cluster
### and your coworkers. Often it's preferrable to run one provider at a time.
###
### $ PROVIDERS=aws,azure,openstack RELEASES=1.25,1.26 ./conformance-tester.sh
###

set -euo pipefail

source hack/lib.sh

###########################################################
# configuration
###########################################################

PROVIDERS="${PROVIDERS:-}"
DISTRIBUTIONS="${DISTRIBUTIONS:-ubuntu,flatcar,rhel,rockylinux}"
RELEASES="${RELEASES:-1.28,1.29,1.30,1.31}"
RUNTIMES="${RUNTIMES:-containerd}"
UPDATE=${UPDATE:-false}
PARALLEL=${PARALLEL:-2}
NAME_PREFIX="${NAME_PREFIX:-$(hostname | tr -cd '[:alnum:]' | tr '[:upper:]' '[:lower:]')}"
SEED="${SEED:-kkp-qa-env}"
PRESET="${PRESET:-kubermatic}"
PROJECT="${PROJECT:-}"
EXCLUDE_TESTS="${EXCLUDE_TESTS:-conformance,telemetry}"

if [ -z "$PROVIDERS" ]; then
  echo "Error: Must set \$PROVIDERS variable to a comma-separated list of cloud providers."
  exit 1
fi

if [ -z "$PROJECT" ]; then
  echo "Error: Must set \$PROJECT variable to the ID of the project to use."
  exit 1
fi

# ensure consistent naming
PROVIDERS="${PROVIDERS/vcd/vmwareclouddirector}"
PROVIDERS="${PROVIDERS/vmware-cloud-director/vmwareclouddirector}"
PROVIDERS="${PROVIDERS/equinixmetal/packet}"
PROVIDERS="${PROVIDERS/google/gcp}"
PROVIDERS="${PROVIDERS/gce/gcp}"

###########################################################
# assemble flags for all selected providers
###########################################################

EXTRA_FLAGS=()

if [[ "$PROVIDERS" =~ aws ]]; then
  echodate "Fetching AWS credentials…"

  accessKey="$(kubectl get preset $PRESET -o json | jq -r '.spec.aws.accessKeyID')"
  secretAccessKey="$(kubectl get preset $PRESET -o json | jq -r '.spec.aws.secretAccessKey')"

  EXTRA_FLAGS+=(
    -aws-access-key-id "$accessKey"
    -aws-secret-access-key "$secretAccessKey"
    -aws-kkp-datacenter "aws-eu-central-1a"
  )
fi

if [[ "$PROVIDERS" =~ azure ]]; then
  echodate "Fetching Azure credentials…"

  clientID="$(kubectl get preset $PRESET -o json | jq -r '.spec.azure.clientID')"
  clientSecret="$(kubectl get preset $PRESET -o json | jq -r '.spec.azure.clientSecret')"
  subscriptionID="$(kubectl get preset $PRESET -o json | jq -r '.spec.azure.subscriptionID')"
  tenantID="$(kubectl get preset $PRESET -o json | jq -r '.spec.azure.tenantID')"

  EXTRA_FLAGS+=(
    -azure-client-id "$clientID"
    -azure-client-secret "$clientSecret"
    -azure-subscription-id "$subscriptionID"
    -azure-tenant-id "$tenantID"
    -azure-kkp-datacenter "azure-westeurope"
  )
fi

if [[ "$PROVIDERS" =~ digitalocean ]]; then
  echodate "Fetching Digitalocean credentials…"

  token="$(kubectl get preset $PRESET -o json | jq -r '.spec.digitalocean.token')"

  EXTRA_FLAGS+=(
    -digitalocean-token "$token"
    -digitalocean-kkp-datacenter "do-ams3"
  )
fi

if [[ "$PROVIDERS" =~ packet ]]; then
  echodate "Fetching Equinix Metal / Packet credentials…"

  apiKey="$(kubectl get preset $PRESET -o json | jq -r '.spec.packet.apiKey')"
  projectID="$(kubectl get preset $PRESET -o json | jq -r '.spec.packet.projectID')"

  EXTRA_FLAGS+=(
    -packet-api-key "$apiKey"
    -packet-project-id "$projectID"
    -packet-kkp-datacenter "packet-am"
  )
fi

if [[ "$PROVIDERS" =~ gcp ]]; then
  echodate "Fetching GCP credentials…"

  serviceAccount="$(kubectl get preset $PRESET -o json | jq -r '.spec.gcp.serviceAccount')"

  EXTRA_FLAGS+=(
    -gcp-service-account "$serviceAccount"
    -gcp-kkp-datacenter "gcp-westeurope"
  )
fi

if [[ "$PROVIDERS" =~ hetzner ]]; then
  echodate "Fetching Hetzner credentials…"

  token="$(kubectl get preset $PRESET -o json | jq -r '.spec.hetzner.token')"

  EXTRA_FLAGS+=(
    -hetzner-token "$token"
    -hetzner-kkp-datacenter "hetzner-nbg1"
  )
fi

if [[ "$PROVIDERS" =~ nutanix ]]; then
  echodate "Fetching Nutanix credentials…"

  username="$(kubectl get preset $PRESET -o json | jq -r '.spec.nutanix.username')"
  password="$(kubectl get preset $PRESET -o json | jq -r '.spec.nutanix.password')"
  csiUsername="$(kubectl get preset $PRESET -o json | jq -r '.spec.nutanix.csiUsername')"
  csiPassword="$(kubectl get preset $PRESET -o json | jq -r '.spec.nutanix.csiPassword')"
  csiEndpoint="$(kubectl get preset $PRESET -o json | jq -r '.spec.nutanix.csiEndpoint')"
  proxyURL="$(kubectl get preset $PRESET -o json | jq -r '.spec.nutanix.proxyURL')"
  clusterName="$(kubectl get preset $PRESET -o json | jq -r '.spec.nutanix.clusterName')"
  projectName="$(kubectl get preset $PRESET -o json | jq -r '.spec.nutanix.projectName')"

  EXTRA_FLAGS+=(
    -nutanix-username "$username"
    -nutanix-password "$password"
    -nutanix-csi-username "$csiUsername"
    -nutanix-csi-password "$csiPassword"
    -nutanix-csi-endpoint "$csiEndpoint"
    -nutanix-proxy-url "$proxyURL"
    -nutanix-cluster-name "$clusterName"
    -nutanix-project-name "$projectName"
    -nutanix-kkp-datacenter "nutanix-hamburg"
  )
fi

if [[ "$PROVIDERS" =~ openstack ]]; then
  echodate "Fetching Openstack credentials…"

  domain="$(kubectl get preset $PRESET -o json | jq -r '.spec.openstack.domain')"
  project="$(kubectl get preset $PRESET -o json | jq -r '.spec.openstack.project')"
  projectID="$(kubectl get preset $PRESET -o json | jq -r '.spec.openstack.projectID')"
  username="$(kubectl get preset $PRESET -o json | jq -r '.spec.openstack.username')"
  password="$(kubectl get preset $PRESET -o json | jq -r '.spec.openstack.password')"

  EXTRA_FLAGS+=(
    -openstack-domain "$domain"
    -openstack-project "$project"
    -openstack-project-id "$projectID"
    -openstack-username "$username"
    -openstack-password "$password"
    -openstack-kkp-datacenter "syseleven-dbl1"
  )
fi

if [[ "$PROVIDERS" =~ vmwareclouddirector ]]; then
  echodate "Fetching VMware Cloud Director credentials…"

  username="$(kubectl get preset $PRESET -o json | jq -r '.spec.vmwareclouddirector.username')"
  password="$(kubectl get preset $PRESET -o json | jq -r '.spec.vmwareclouddirector.password')"
  organization="$(kubectl get preset $PRESET -o json | jq -r '.spec.vmwareclouddirector.organization')"
  vdc="$(kubectl get preset $PRESET -o json | jq -r '.spec.vmwareclouddirector.vdc')"
  ovdcNetwork="$(kubectl get preset $PRESET -o json | jq -r '.spec.vmwareclouddirector.ovdcNetwork')"

  EXTRA_FLAGS+=(
    -vmware-cloud-director-username "$username"
    -vmware-cloud-director-password "$password"
    -vmware-cloud-director-organization "$organization"
    -vmware-cloud-director-vdc "$vdc"
    -vmware-cloud-director-ovdc-networks "$ovdcNetwork"
    -vmware-cloud-director-kkp-datacenter "vmware-cloud-director-ger"
  )
fi

if [[ "$PROVIDERS" =~ vsphere ]]; then
  echodate "Fetching VSphere credentials…"

  username="$(kubectl get preset $PRESET -o json | jq -r '.spec.vsphere.username')"
  password="$(kubectl get preset $PRESET -o json | jq -r '.spec.vsphere.password')"

  EXTRA_FLAGS+=(
    -vsphere-username "$username"
    -vsphere-password "$password"
    -vsphere-kkp-datacenter "vsphere-hamburg"
  )
fi

if [[ "$PROVIDERS" =~ kubevirt ]]; then
  echodate "Setting KubeVirt args"

  if [[ -z "${KUBEVIRT_KUBECONFIG+x}" ]]; then
      echo "Error: Must set \$KUBEVIRT_KUBECONFIG variable with path to the KubeVirt infra cluster kubeconfig"
      exit 1
  fi

  EXTRA_FLAGS+=(
    -kubevirt-kkp-datacenter kv-hamburg
    -kubevirt-kubeconfig "$KUBEVIRT_KUBECONFIG"
  )
fi

###########################################################
# assemble extra flags for the chosen distributions
###########################################################

if [[ "$DISTRIBUTIONS" =~ rhel ]]; then
  echodate "Fetching RHEL subscription…"

  rhelData="$(vault kv get --format=json dev/redhat-subscription)"
  rhelSubscriptionUser="$(echo "$rhelData" | jq -r '.data.data.user')"
  rhelSubscriptionPassword="$(echo "$rhelData" | jq -r '.data.data.password')"
  rhelOfflineToken="$(echo "$rhelData" | jq -r '.data.data.offlineToken')"

  EXTRA_FLAGS+=(
    -rhel-subscription-user "$rhelSubscriptionUser"
    -rhel-subscription-password "$rhelSubscriptionPassword"
    -rhel-offline-token "$rhelOfflineToken"
  )
fi

###########################################################
# make sure conformance-tester is up-to-date
###########################################################

echodate "Compiling fresh conformance-tester…"
KUBERMATIC_EDITION=ee make clean conformance-tester

###########################################################
# run the tester
###########################################################

mkdir -p reports

echo
echo "====== TEST PARAMETERS ====================================="
echo "Cloud Providers...: $PROVIDERS"
echo "Distributions.....: $DISTRIBUTIONS"
echo "Releases..........: $RELEASES"
echo "Container Runtimes: $RUNTIMES"
echo "Update Clusters...: $UPDATE"
echo "Name Prefix.......: $NAME_PREFIX"
echo "============================================================"
echo

echodate "Running conformance-tester…"
_build/conformance-tester \
  -providers "$PROVIDERS" \
  -distributions "$DISTRIBUTIONS" \
  -releases "$RELEASES" \
  -update-cluster=$UPDATE \
  -kubermatic-seed-cluster "$SEED" \
  -kubermatic-project "$PROJECT" \
  -kubermatic-parallel-clusters $PARALLEL \
  -name-prefix "$NAME_PREFIX" \
  -client "kube" \
  -log-format "Console" \
  -exclude-tests "${EXCLUDE_TESTS}" \
  -wait-for-cluster-deletion=false \
  -reports-root "$(realpath reports)" \
  "${EXTRA_FLAGS[@]}" $@

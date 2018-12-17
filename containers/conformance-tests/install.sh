#!/usr/bin/env bash

ROOT_DIR="/opt/kube-test"

set -euox pipefail

apt update && apt install -y git-crypt curl

TMP_ROOT="./.install-tmp"

for VERSION in 1.10 1.11 1.12; do
    DIRECTORY="${ROOT_DIR}/${VERSION}-kubernetes"
    if [ ! -d "${DIRECTORY}" ]; then

        FULL_VERSION=$(curl -Ss https://storage.googleapis.com/kubernetes-release/release/stable-${VERSION}.txt)
        echo "kube-test for ${VERSION} not found. Downloading to ${DIRECTORY} ..."

        TMP_DIR="${TMP_ROOT}/${VERSION}"
        mkdir -p ${TMP_DIR}
        mkdir -p ${DIRECTORY}

        curl -L http://gcsweb.k8s.io/gcs/kubernetes-release/release/${FULL_VERSION}/kubernetes.tar.gz -o ${TMP_DIR}/kubernetes.tar.gz
        tar -zxvf ${TMP_DIR}/kubernetes.tar.gz -C ${TMP_DIR}
        mv ${TMP_DIR}/kubernetes/* ${DIRECTORY}/

        cd ${DIRECTORY}/ && KUBE_VERSION="${FULL_VERSION}" KUBERNETES_DOWNLOAD_TESTS=true KUBERNETES_SKIP_CONFIRM=true ./cluster/get-kube-binaries.sh
        cd -
    fi
done

rm -rf ${TMP_ROOT}

curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl
chmod +x kubectl
mv kubectl /usr/local/bin

mkdir $HOME/.ssh
echo 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCo3amVmCkIZo4cgj2kjU2arZKlzzOhaOuveH9aJbL4mlVHVsEcVk+RSty4AMK1GQL3+Ii7iGicKWwge4yefc75aOtUncfF01rnBsNvi3lOqJR/6POHy4OnPXJElvEn7jii/pAUeyr8halBezQTUkvRiUtlJo6oEb2dRN5ujyFm5TuIxgM0UFVGBRoD0agGr87GaQsUahf+PE1zHEid+qQPz7EdMo8/eRNtgikhBG1/ae6xRstAi0QU8EgjKvK1ROXOYTlpTBFElApOXZacH91WvG0xgPnyxIXoKtiCCNGeu/0EqDAgiXfmD2HK/WAXwJNwcmRvBaedQUS4H0lNmvj5' > $HOME/.ssh/id_rsa.pub
chmod 0700 $HOME/.ssh

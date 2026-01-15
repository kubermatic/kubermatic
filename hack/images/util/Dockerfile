# Copyright 2020 The Kubermatic Kubernetes Platform contributors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM docker.io/alpine:3.23
LABEL org.opencontainers.image.source="https://github.com/kubermatic/kubermatic/blob/main/hack/images/util/Dockerfile"
LABEL org.opencontainers.image.vendor="Kubermatic"
LABEL org.opencontainers.image.authors="support@kubermatic.com"

    # https://dl.minio.io/client/mc/release/linux-amd64/
ENV MC_VERSION=RELEASE.2025-08-13T08-35-41Z \
    KUBECTL_VERSION=v1.34.3 \
    # https://github.com/helm/helm/releases
    HELM_VERSION=v3.18.5 \
    # https://github.com/hashicorp/vault/releases
    VAULT_VERSION=1.21.2 \
    # https://github.com/jqlang/jq/releases
    JQ_VERSION=1.8.1 \
    # https://github.com/mikefarah/yq/releases
    YQ_VERSION=4.50.1

# ensure that we install a curl version that fixes CVE-2023-38545 and CVE-2023-38546.
RUN apk add --no-cache -U \
    bash \
    ca-certificates \
    curl \
    git \
    iproute2 \
    iptables \
    ipvsadm \
    make \
    openssh-client \
    rsync \
    socat \
    unzip \
    tar

RUN curl --fail -LO https://github.com/jqlang/jq/releases/download/jq-${JQ_VERSION}/jq-linux64 && \
    chmod +x jq-linux64 && \
    mv jq-linux64 /usr/bin/jq && \
    jq -V

RUN curl -Lo /usr/bin/yq https://github.com/mikefarah/yq/releases/download/v${YQ_VERSION}/yq_linux_amd64 && \
    chmod +x /usr/bin/yq && \
    yq --version

RUN curl -Lo /usr/bin/mc https://dl.minio.io/client/mc/release/linux-amd64/archive/mc.${MC_VERSION} && \
    chmod +x /usr/bin/mc && \
    mc --version

RUN curl -Lo /usr/bin/kubectl https://dl.k8s.io/${KUBECTL_VERSION}/bin/linux/amd64/kubectl && \
    chmod +x /usr/bin/kubectl && \
    kubectl version --client

RUN curl --fail -L https://get.helm.sh/helm-${HELM_VERSION}-linux-amd64.tar.gz | tar -xzO linux-amd64/helm > /usr/local/bin/helm && \
    chmod +x /usr/local/bin/helm && \
    helm version --short

RUN curl -Lo vault.zip https://releases.hashicorp.com/vault/${VAULT_VERSION}/vault_${VAULT_VERSION}_linux_amd64.zip && \
    unzip vault.zip && \
    rm vault.zip && \
    mv vault /usr/bin/vault && \
    vault version

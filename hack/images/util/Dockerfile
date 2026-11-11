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

FROM docker.io/alpine:3.24
LABEL org.opencontainers.image.source="https://github.com/kubermatic/kubermatic/blob/main/hack/images/util/Dockerfile"
LABEL org.opencontainers.image.vendor="Kubermatic"
LABEL org.opencontainers.image.authors="support@kubermatic.com"

    # https://github.com/minio/mc/releases
    # update MC_SHA256 together with MC_VERSION, it is the digest of mc.linux-amd64.${MC_VERSION}
ENV MC_VERSION=RELEASE.2025-08-13T08-35-41Z \
    MC_SHA256=01f866e9c5f9b87c2b09116fa5d7c06695b106242d829a8bb32990c00312e891 \
    KUBECTL_VERSION=v1.37.2 \
    # https://github.com/helm/helm/releases
    HELM_VERSION=v3.22.1 \
    # https://github.com/hashicorp/vault/releases
    VAULT_VERSION=2.2.1 \
    # https://github.com/jqlang/jq/releases
    JQ_VERSION=1.8.2 \
    # https://github.com/mikefarah/yq/releases
    YQ_VERSION=4.54.1

# apk add only pulls patched versions for the packages named below; without this
  # the rest of the rootfs (openssl for example) stays on whatever the base shipped.
RUN apk upgrade --no-cache -U

RUN apk add --no-cache \
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

RUN curl --fail -Lo /usr/bin/yq https://github.com/mikefarah/yq/releases/download/v${YQ_VERSION}/yq_linux_amd64 && \
    chmod +x /usr/bin/yq && \
    yq --version

RUN curl --fail -Lo /usr/bin/mc https://github.com/minio/mc/releases/download/${MC_VERSION}/mc.linux-amd64.${MC_VERSION} && \
    echo "${MC_SHA256}  /usr/bin/mc" | sha256sum -c - && \
    chmod +x /usr/bin/mc && \
    mc --version

RUN curl --fail -Lo /usr/bin/kubectl https://dl.k8s.io/${KUBECTL_VERSION}/bin/linux/amd64/kubectl && \
    chmod +x /usr/bin/kubectl && \
    kubectl version --client

RUN curl --fail -L https://get.helm.sh/helm-${HELM_VERSION}-linux-amd64.tar.gz | tar -xzO linux-amd64/helm > /usr/local/bin/helm && \
    chmod +x /usr/local/bin/helm && \
    helm version --short

RUN curl --fail -Lo vault.zip https://releases.hashicorp.com/vault/${VAULT_VERSION}/vault_${VAULT_VERSION}_linux_amd64.zip && \
    unzip vault.zip && \
    rm vault.zip && \
    mv vault /usr/bin/vault && \
    vault version

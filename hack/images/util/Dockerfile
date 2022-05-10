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

FROM alpine:3.13
LABEL maintainer="support@kubermatic.com"

ENV MC_VERSION=RELEASE.2022-05-09T04-08-26Z \
    KUBECTL_VERSION=v1.22.9 \
    HELM_VERSION=v3.8.1 \
    VAULT_VERSION=1.10.2 \
    YQ3_VERSION=3.4.1 \
    YQ4_VERSION=4.25.1

RUN apk add --no-cache -U \
    bash \
    ca-certificates \
    curl \
    git \
    iproute2 \
    iptables \
    ipvsadm \
    jq \
    make \
    openssh-client \
    rsync \
    socat \
    unzip \
    tar

RUN curl -Lo /usr/bin/yq3 https://github.com/mikefarah/yq/releases/download/${YQ3_VERSION}/yq_linux_amd64 && \
    chmod +x /usr/bin/yq3 && \
    yq3 --version && \
    ln -s /usr/bin/yq3 /usr/bin/yq

RUN curl -Lo /usr/bin/yq4 https://github.com/mikefarah/yq/releases/download/v${YQ4_VERSION}/yq_linux_amd64 && \
    chmod +x /usr/bin/yq4 && \
    yq4 --version

RUN curl -Lo /usr/bin/mc https://dl.minio.io/client/mc/release/linux-amd64/archive/mc.${MC_VERSION} && \
    chmod +x /usr/bin/mc && \
    mc --version

RUN curl -Lo /usr/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl && \
    chmod +x /usr/bin/kubectl && \
    kubectl version --short --client

RUN curl --fail -L https://get.helm.sh/helm-${HELM_VERSION}-linux-amd64.tar.gz | tar -xzO linux-amd64/helm > /usr/local/bin/helm && \
    chmod +x /usr/local/bin/helm && \
    helm version --short

RUN curl -Lo vault.zip https://releases.hashicorp.com/vault/${VAULT_VERSION}/vault_${VAULT_VERSION}_linux_amd64.zip && \
    unzip vault.zip && \
    rm vault.zip && \
    mv vault /usr/bin/vault && \
    vault version

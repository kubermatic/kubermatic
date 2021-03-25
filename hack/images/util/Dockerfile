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

ENV MC_VERSION=RELEASE.2021-03-12T03-36-59Z \
    KUBECTL_VERSION=v1.20.5 \
    HELM_VERSION=v2.17.0 \
    VAULT_VERSION=1.6.3 \
    YQ_VERSION=3.3.4 \
    WWHRD_VERSION=0.4.0

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

RUN curl -Lo /usr/bin/yq https://github.com/mikefarah/yq/releases/download/${YQ_VERSION}/yq_linux_amd64 && \
    chmod +x /usr/bin/yq && \
    yq --version

RUN curl -Lo /usr/bin/mc https://dl.minio.io/client/mc/release/linux-amd64/archive/mc.${MC_VERSION} && \
    chmod +x /usr/bin/mc && \
    mc --version

RUN curl -Lo /usr/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl && \
    chmod +x /usr/bin/kubectl && \
    kubectl version --short --client

RUN curl -L https://storage.googleapis.com/kubernetes-helm/helm-${HELM_VERSION}-linux-amd64.tar.gz | tar -xvz && \
    mv linux-amd64/helm /usr/bin/helm && \
    rm -rf linux-amd64 && \
    helm version --client --short

RUN curl -Lo vault.zip https://releases.hashicorp.com/vault/${VAULT_VERSION}/vault_${VAULT_VERSION}_linux_amd64.zip && \
    unzip vault.zip && \
    rm vault.zip && \
    mv vault /usr/bin/vault && \
    vault version

RUN curl -L https://github.com/frapposelli/wwhrd/releases/download/v{$WWHRD_VERSION}/wwhrd_{$WWHRD_VERSION}_linux_amd64.tar.gz | tar -xvz wwhrd && \
    mv wwhrd /usr/bin/wwhrd && \
    wwhrd -v

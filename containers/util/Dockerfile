FROM alpine:3.9

ENV MC_VERSION=RELEASE.2019-04-03T17-59-57Z \
    KUBECTL_VERSION=v1.14.1 \
    HELM_VERSION=v2.13.1 \
    VAULT_VERSION=0.11.6

RUN apk add --no-cache -U \
    bash \
    ca-certificates \
    curl \
    jq \
    rsync \
    unzip

RUN curl -Lo /usr/bin/mc https://dl.minio.io/client/mc/release/linux-amd64/mc.${MC_VERSION} && \
    chmod +x /usr/bin/mc

RUN curl -Lo /usr/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl && \
    chmod +x /usr/bin/kubectl

RUN curl -L https://storage.googleapis.com/kubernetes-helm/helm-${HELM_VERSION}-linux-amd64.tar.gz | tar -xvz && \
    mv linux-amd64/helm /usr/bin/helm && \
    rm -rf linux-amd64

RUN curl -Lo vault.zip https://releases.hashicorp.com/vault/${VAULT_VERSION}/vault_${VAULT_VERSION}_linux_amd64.zip && \
    unzip vault.zip && \
    rm vault.zip && \
    mv vault /usr/bin/vault

FROM alpine

RUN apk add --no-cache -U ca-certificates unzip jq bash curl
RUN curl -Lo /usr/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/v1.12.2/bin/linux/amd64/kubectl && \
    chmod +x /usr/bin/kubectl

RUN curl -L https://storage.googleapis.com/kubernetes-helm/helm-v2.11.0-linux-amd64.tar.gz | tar -xvz && \
    mv linux-amd64/helm /usr/bin/helm

RUN curl -Lo  vault.zip https://releases.hashicorp.com/vault/0.11.5/vault_0.11.5_linux_amd64.zip && \
    unzip vault.zip && \
    rm vault.zip && \
    mv vault /bin/

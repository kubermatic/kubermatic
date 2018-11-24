FROM alpine

RUN apk add --no-cache ca-certificates curl
RUN curl -Lo /usr/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/v1.12.2/bin/linux/amd64/kubectl && \
    chmod +x /usr/bin/kubectl

RUN curl -Lo /tmp/helm.tar.gz https://storage.googleapis.com/kubernetes-helm/helm-v2.11.0-linux-amd64.tar.gz && \
    cd /tmp && tar -zxvf /tmp/helm.tar.gz && cp linux-amd64/helm /usr/bin/helm

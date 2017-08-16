FROM alpine:3.6

# Install Curl & bash
RUN apk add --no-cache curl bash

ENV HELM_VERSION v2.5.1

# Download & install helm
RUN curl -LO https://storage.googleapis.com/kubernetes-helm/helm-$HELM_VERSION-linux-amd64.tar.gz && \
    tar -zxvf helm-$HELM_VERSION-linux-amd64.tar.gz && \
    mv linux-amd64/helm /usr/local/bin/helm

# Copy all required folder
COPY bare-metal-provider /kubermatic/bare-metal-provider
COPY coreos-ipxe-server /kubermatic/coreos-ipxe-server
COPY efk-logging /kubermatic/efk-logging
COPY k8sniff /kubermatic/k8sniff
COPY kube-lego /kubermatic/kube-lego
COPY kubermatic /kubermatic/kubermatic
COPY nginx-ingress-controller /kubermatic/nginx-ingress-controller
COPY prometheus /kubermatic/prometheus
COPY storage /kubermatic/storage
COPY thirdpartyresources /kubermatic/thirdpartyresources

COPY deploy.sh /kubermatic/
COPY kubermatic-values.yaml /kubermatic

CMD ["/kubermatic/deploy.sh"]

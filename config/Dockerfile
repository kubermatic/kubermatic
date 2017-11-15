FROM alpine:3.6

ARG HELM_VER
ARG KUBECTL_VER
ENV HELM_VER ${HELM_VER:-v2.7.0}
ENV KUBECTL_VER ${KUBECTL_VER:-v1.8.3}

RUN apk add --no-cache bash
ADD https://storage.googleapis.com/kubernetes-helm/helm-$HELM_VER-linux-amd64.tar.gz /tmp/helm.tar.gz
ADD https://storage.googleapis.com/kubernetes-release/release/$KUBECTL_VER/bin/linux/amd64/kubectl /tmp
RUN tar -zxvf /tmp/helm.tar.gz -C /tmp && \
    mv /tmp/linux-amd64/helm /usr/local/bin/helm && chmod +x /usr/local/bin/helm && \
    rm -rf /tmp/linux-amd64 /tmp/helm-$HELM_VERSION-linux-amd64.tar.gz
RUN mv /tmp/kubectl /usr/local/bin/kubectl && chmod +x /usr/local/bin/kubectl

COPY bare-metal-provider /kubermatic/bare-metal-provider
COPY coreos-ipxe-server /kubermatic/coreos-ipxe-server
COPY efk-logging /kubermatic/efk-logging
COPY k8sniff /kubermatic/k8sniff
COPY kubermatic /kubermatic/kubermatic
COPY nginx-ingress-controller /kubermatic/nginx-ingress-controller
COPY oauth /kubermatic/oauth
COPY storage /kubermatic/storage
COPY cert-manager /kubermatic/cert-manager
COPY certs /kubermatic/certs
COPY monitoring /kubermatic/monitoring

COPY installer-assets /kubermatic/installer-assets
COPY deploy_helm_charts.sh /kubermatic/

CMD ["/kubermatic/deploy_helm_charts.sh"]

FROM alpine:3.6

ARG HELM_VERSION
ENV HELM_VERSION ${HELM_VERSION:-v2.7.0}

RUN apk add --no-cache bash
ADD https://storage.googleapis.com/kubernetes-helm/helm-$HELM_VERSION-linux-amd64.tar.gz /tmp
RUN tar -zxvf /tmp/helm-$HELM_VERSION-linux-amd64.tar.gz -C /tmp && \
    mv /tmp/linux-amd64/helm /usr/local/bin/helm && \
    rm -rf /tmp/linux-amd64 /tmp/helm-$HELM_VERSION-linux-amd64.tar.gz

COPY bare-metal-provider /kubermatic/bare-metal-provider
COPY coreos-ipxe-server /kubermatic/coreos-ipxe-server
COPY efk-logging /kubermatic/efk-logging
COPY k8sniff /kubermatic/k8sniff
COPY kubermatic /kubermatic/kubermatic
COPY nginx-ingress-controller /kubermatic/nginx-ingress-controller
COPY prometheus /kubermatic/prometheus
COPY storage /kubermatic/storage

COPY deploy_helm_charts.sh /kubermatic/
COPY kubermatic-values.yaml /kubermatic/

CMD ["/kubermatic/deploy_helm_charts.sh"]

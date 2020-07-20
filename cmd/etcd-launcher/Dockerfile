ARG ETCD_VERSION=v3.4.3
FROM gcr.io/etcd-development/etcd:$ETCD_VERSION
LABEL maintainer="support@kubermatic.com"

COPY ./_build/etcd-launcher /usr/local/bin/

ARG ECTD_VERSION=v3.4.3
FROM gcr.io/etcd-development/etcd:$ECTD_VERSION
LABEL maintainer="support@loodse.com"
RUN apt update && apt install bash curl -y
COPY etcd-launcher /usr/local/bin/

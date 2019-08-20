FROM debian:buster

LABEL maintainer="sebastian@loodse.com"

RUN apt update && apt install -y curl iptables

ADD https://storage.googleapis.com/kubernetes-release/release/v1.13.5/bin/linux/amd64/kubectl /usr/local/bin/kubectl
RUN chmod +x /usr/local/bin/kubectl && \
  curl -L https://github.com/openshift/origin/releases/download/v3.11.0/openshift-origin-client-tools-v3.11.0-0cbc58b-linux-64bit.tar.gz \
  |tar -xvz && \
  mv openshift-origin-client-tools-v3.11.0-0cbc58b-linux-64bit/oc /usr/local/bin

COPY ./_build/* /usr/local/bin/
COPY ./cmd/kubermatic-api/swagger.json /opt/swagger.json

USER nobody

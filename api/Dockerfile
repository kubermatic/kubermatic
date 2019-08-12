FROM alpine:3.10

LABEL maintainer="sebastian@loodse.com"

ADD https://storage.googleapis.com/kubernetes-release/release/v1.14.5/bin/linux/amd64/kubectl /usr/local/bin/kubectl
RUN chmod +x /usr/local/bin/kubectl

COPY ./_build/* /usr/local/bin/
COPY ./cmd/kubermatic-api/swagger.json /opt/swagger.json

USER nobody

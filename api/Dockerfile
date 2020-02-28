FROM alpine:3.10
LABEL maintainer="support@loodse.com"

ADD https://storage.googleapis.com/kubernetes-release/release/v1.17.3/bin/linux/amd64/kubectl /usr/local/bin/kubectl
# We need the ca-certs so they api doesn't crash because it can't verify the certificate of Dex
RUN chmod +x /usr/local/bin/kubectl && apk add ca-certificates

COPY ./_build/* /usr/local/bin/
COPY ./cmd/kubermatic-api/swagger.json /opt/swagger.json

USER nobody

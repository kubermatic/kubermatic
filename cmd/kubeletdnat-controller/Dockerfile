FROM alpine:3.10
LABEL maintainer="support@loodse.com"

RUN apk add -u iptables
COPY ./_build/kubeletdnat-controller /usr/local/bin/kubeletdnat-controller

FROM alpine:3.5

LABEL maintainer "sebastian@loodse.com"

RUN apk add -U ca-certificates && rm -rf /var/cache/apk/*

ADD ./_build/kubermatic-api /
ADD ./_build/kubermatic-cluster-controller /

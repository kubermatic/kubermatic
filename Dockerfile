FROM alpine:3.5

LABEL maintainer "sebastian@loodse.com"

RUN apk add -U ca-certificates && rm -rf /var/cache/apk/*

ADD ./kubermatic-api /
ADD ./kubermatic-cluster-controller /

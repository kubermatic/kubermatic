FROM alpine:3.5
MAINTAINER Dr. Stefan Schimanski <stefan.schimanski@gmail.com>

RUN apk add -U ca-certificates && rm -rf /var/cache/apk/*

ADD ./kubermatic-api /
ADD ./kubermatic-cluster-controller /

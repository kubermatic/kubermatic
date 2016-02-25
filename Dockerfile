FROM alpine:3.1
MAINTAINER Dr. Stefan Schimanski <stefan.schimanski@gmail.com>

RUN apk add -U ca-certificates && rm -rf /var/cache/apk/*

COPY kubermatic-api /usr/bin/kubermatic-api
RUN mkdir -p /opt
WORKDIR /opt
ADD datacenters.yaml /opt/datacenters.yaml

CMD ["kubermatic-api"]

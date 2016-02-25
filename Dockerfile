FROM alpine:3.1
MAINTAINER Dr. Stefan Schimanski <stefan.schimanski@gmail.com>

RUN apk add -U ca-certificates && rm -rf /var/cache/apk/*

COPY kubermatic-api /usr/bin/
RUN mkdir -p /opt
WORKDIR /opt
ADD datacenters.yaml /opt/

CMD ["kubermatic-api"]

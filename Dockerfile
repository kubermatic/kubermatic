FROM progrium/busybox
MAINTAINER Stefan Schimanski <stefan.schimanski@gmail.com>

COPY api /usr/bin/kubermatic-api
RUN mkdir -p /opt
WORKDIR /opt

CMD ["kubermatic-api"]

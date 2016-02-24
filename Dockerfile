FROM progrium/busybox
MAINTAINER Stefan Schimanski <stefan.schimanski@gmail.com>

COPY kubermatic-api /usr/bin/kubermatic-api
RUN mkdir -p /opt
WORKDIR /opt
ADD datacenters.yaml /opt/datacenters.yaml

CMD ["kubermatic-api"]

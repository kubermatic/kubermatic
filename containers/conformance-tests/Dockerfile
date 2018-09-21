FROM debian:stretch

LABEL maintainer "henrik@loodse.com"

RUN apt-get update && apt-get install -y curl

COPY ./conformance-tests /usr/local/bin/
COPY ./install.sh /opt/install.sh

RUN chmod +x /opt/install.sh
RUN /opt/install.sh

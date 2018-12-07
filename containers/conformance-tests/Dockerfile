FROM golang:1.11.2-stretch

LABEL maintainer "henrik@loodse.com"

COPY ./install.sh /opt/install.sh

RUN chmod +x /opt/install.sh && /opt/install.sh

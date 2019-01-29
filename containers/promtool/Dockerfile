FROM alpine

RUN apk add --no-cache -U curl make

RUN curl -L https://github.com/prometheus/prometheus/releases/download/v2.7.0/prometheus-2.7.0.linux-amd64.tar.gz | tar -xvz && \
    mv prometheus-2.7.0.linux-amd64/promtool /usr/bin/promtool

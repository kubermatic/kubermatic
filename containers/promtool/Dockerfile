FROM alpine

RUN apk add --no-cache -U curl make jq

RUN curl -L https://github.com/prometheus/prometheus/releases/download/v2.7.0/prometheus-2.7.0.linux-amd64.tar.gz | tar -xvz && \
    mv prometheus-2.7.0.linux-amd64/promtool /usr/bin/promtool

RUN curl -L -o /usr/bin/yaml2json https://github.com/bronze1man/yaml2json/releases/download/v1.3/yaml2json_linux_amd64 && chmod +x /usr/bin/yaml2json

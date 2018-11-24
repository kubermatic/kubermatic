FROM docker:18.09.0-dind

RUN apk add --no-cache -U ca-certificates wget unzip jq bash
RUN wget https://releases.hashicorp.com/vault/0.11.5/vault_0.11.5_linux_amd64.zip && \
    unzip vault_0.11.5_linux_amd64.zip && \
    rm vault_0.11.5_linux_amd64.zip && \
    mv vault /bin/

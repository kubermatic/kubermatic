FROM alpine:3.7

LABEL maintainer "sebastian@loodse.com"

RUN echo https://mirror1.hs-esslingen.de/pub/Mirrors/alpine/v3.7/main > /etc/apk/repositories; \
    echo https://mirror1.hs-esslingen.de/pub/Mirrors/alpine/v3.7/community >> /etc/apk/repositories

RUN apk add -U ca-certificates && rm -rf /var/cache/apk/*

RUN mkdir -p /_artifacts/

ADD https://storage.googleapis.com/kubernetes-release/release/v1.10.3/bin/linux/amd64/kubectl /usr/local/bin/kubectl
RUN chmod +x /usr/local/bin/kubectl

ADD ./_build/kubermatic-api /
ADD ./_build/kubermatic-controller-manager /
ADD ./_build/client /

WORKDIR /opt/

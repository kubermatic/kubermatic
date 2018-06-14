FROM alpine:3.7

LABEL maintainer "sebastian@loodse.com"

RUN echo https://mirror1.hs-esslingen.de/pub/Mirrors/alpine/v3.7/main > /etc/apk/repositories; \
    echo https://mirror1.hs-esslingen.de/pub/Mirrors/alpine/v3.7/community >> /etc/apk/repositories

RUN apk add --no-cache ca-certificates

ADD https://storage.googleapis.com/kubernetes-release/release/v1.10.3/bin/linux/amd64/kubectl /usr/local/bin/kubectl
RUN chmod +x /usr/local/bin/kubectl

COPY ./_build/* /usr/local/bin/

USER nobody


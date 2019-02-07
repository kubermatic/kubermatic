FROM alpine:3.9

LABEL maintainer "sebastian@loodse.com"

RUN apk add --no-cache ca-certificates

ADD https://storage.googleapis.com/kubernetes-release/release/v1.13.3/bin/linux/amd64/kubectl /usr/local/bin/kubectl
RUN chmod +x /usr/local/bin/kubectl

COPY ./_build/* /usr/local/bin/

USER nobody

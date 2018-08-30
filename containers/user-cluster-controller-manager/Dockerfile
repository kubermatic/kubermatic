FROM alpine:3.7
LABEL maintainer "tobias.hintze@loodse.com"

RUN apk add -U \
	bash \
	ca-certificates \
	curl \
	--no-cache

COPY ./user-cluster-controller-manager /usr/local/bin/

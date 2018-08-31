FROM alpine:3.7
LABEL maintainer "tobias.hintze@loodse.com"

RUN apk add -U \
	bash \
	ca-certificates \
	curl \
	iptables \
	--no-cache

COPY ./kubeletdnat-controller /usr/local/bin/

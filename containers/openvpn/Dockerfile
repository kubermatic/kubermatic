FROM alpine:3.11
ARG OPENVPN_VERSION
LABEL maintainer="support@loodse.com"

RUN apk add -U \
	bash \
	ca-certificates \
	curl \
	iptables \
	jq \
	openssl \
	openvpn=$OPENVPN_VERSION \
	--no-cache

# add kubectl
RUN curl \
	-o /usr/bin/kubectl \
	-L "https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl"
RUN chmod 0755 /usr/bin/kubectl

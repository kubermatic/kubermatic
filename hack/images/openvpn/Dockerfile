# Copyright 2020 The Kubermatic Kubernetes Platform contributors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM alpine:3.11
ARG OPENVPN_VERSION
LABEL maintainer="support@kubermatic.com"

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

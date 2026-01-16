# Copyright 2024 The Kubermatic Kubernetes Platform contributors.
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

FROM docker.io/alpine:3.23
LABEL org.opencontainers.image.source="https://github.com/kubermatic/kubermatic/blob/main/cmd/conformance-tester/Dockerfile"
LABEL org.opencontainers.image.vendor="Kubermatic"
LABEL org.opencontainers.image.authors="support@kubermatic.com"

ENV KUBECTL_VERSION=v1.34.3

RUN apk add --no-cache -U bash ca-certificates curl jq

RUN curl -Lo /usr/bin/kubectl https://dl.k8s.io/${KUBECTL_VERSION}/bin/linux/amd64/kubectl && \
    chmod +x /usr/bin/kubectl && \
    kubectl version --client

COPY ./_build/conformance-tester /usr/local/bin/conformance-tester

WORKDIR /conformance-tester
USER nobody
ENTRYPOINT [ "conformance-tester" ]

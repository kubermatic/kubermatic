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

ARG BASE_IMAGE=docker.io/golang:1.23.2
FROM $BASE_IMAGE AS builder

ARG GOPROXY=
ARG KUBERMATIC_EDITION=ce

ENV GOPROXY=$GOPROXY
ENV KUBERMATIC_EDITION=$KUBERMATIC_EDITION

WORKDIR /go/src/k8c.io/kubermatic
COPY . .
RUN make conformance-tester

FROM docker.io/alpine:3.19
LABEL org.opencontainers.image.source="https://github.com/kubermatic/kubermatic/blob/main/cmd/conformance-tester/Dockerfile"
LABEL org.opencontainers.image.vendor="Kubermatic"
LABEL org.opencontainers.image.authors="support@kubermatic.com"

ENV KUBECTL_VERSION=v1.28.8

RUN apk add --no-cache -U bash ca-certificates curl jq

RUN curl -Lo /usr/bin/kubectl https://dl.k8s.io/${KUBECTL_VERSION}/bin/linux/amd64/kubectl && \
    chmod +x /usr/bin/kubectl && \
    kubectl version --client

COPY --from=builder /go/src/k8c.io/kubermatic/_build /usr/local/bin/

WORKDIR /conformance-tester
USER nobody
ENTRYPOINT ["conformance-tester"]

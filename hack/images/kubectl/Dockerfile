# Copyright 2026 The Kubermatic Kubernetes Platform contributors.
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

# kubectl image for the Helm hook Jobs of the velero chart (7.x and 11.x).
# The upgrade-crds hook copies `sh` and `kubectl` from this image into an
# emptyDir and runs them inside the velero image, which has no shell and an
# older glibc. The static busybox from the musl variant runs anywhere, so
# keep this image on busybox-musl and do not switch to a glibc or alpine base.

FROM --platform=$BUILDPLATFORM docker.io/alpine:3.24 AS downloader

ARG TARGETARCH

# https://dl.k8s.io/release/stable-1.34.txt
ARG KUBECTL_VERSION=v1.34.13
# https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/${TARGETARCH}/kubectl.sha256
ARG KUBECTL_SHA256_AMD64=b4bb494e9466a7a94bb851414548bc0d3e2f8da90757e6eccc3371e648648591
ARG KUBECTL_SHA256_ARM64=66e8283d52d9bc1f9c0e9daade7c971cd20db5a379306862897dfb25395f3e03

RUN apk add --no-cache curl && \
    case "$TARGETARCH" in \
      amd64) sum="$KUBECTL_SHA256_AMD64" ;; \
      arm64) sum="$KUBECTL_SHA256_ARM64" ;; \
      *) echo "unsupported architecture: $TARGETARCH" >&2; exit 1 ;; \
    esac && \
    curl --fail -sSLo /kubectl "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/${TARGETARCH}/kubectl" && \
    echo "$sum  /kubectl" | sha256sum -c - && \
    chmod 0755 /kubectl

FROM docker.io/busybox:1.38.0-musl
LABEL org.opencontainers.image.source="https://github.com/kubermatic/kubermatic/blob/main/hack/images/kubectl/Dockerfile"
LABEL org.opencontainers.image.vendor="Kubermatic"
LABEL org.opencontainers.image.authors="support@kubermatic.com"

COPY --from=downloader /kubectl /usr/local/bin/kubectl

USER 65532:65532
ENTRYPOINT ["/usr/local/bin/kubectl"]

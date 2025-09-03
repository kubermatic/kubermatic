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

FROM quay.io/kubermatic/build:go-1.23-node-20-kind-0.29-13
LABEL org.opencontainers.image.source="https://github.com/kubermatic/kubermatic/blob/main/hack/images/integration-tests/Dockerfile"
LABEL org.opencontainers.image.vendor="Kubermatic"
LABEL org.opencontainers.image.authors="support@kubermatic.com"

ENV KUBE_VERSION=1.31.0

RUN os=$(go env GOOS) && \
    arch=$(go env GOARCH) && \
    mkdir -p /usr/local/kubebuilder/bin/ && \
    curl --fail -sL "https://github.com/kubernetes-sigs/controller-tools/releases/download/envtest-v${KUBE_VERSION}/envtest-v${KUBE_VERSION}-$os-$arch.tar.gz" | tar -xz --strip-components=2 -C /usr/local/kubebuilder/bin/ && \
    echo 'export PATH=$PATH:/usr/local/kubebuilder/bin' >> ~/.bashrc

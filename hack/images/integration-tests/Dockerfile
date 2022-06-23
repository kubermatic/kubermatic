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

FROM quay.io/kubermatic/build:go-1.18-node-16-kind-0.14-9
LABEL maintainer="support@kubermatic.com"

# envtest binaries are not available for all k8s patch releases, so beware when updating
ENV KUBE_VERSION=1.23.5

RUN os=$(go env GOOS) && \
    arch=$(go env GOARCH) && \
    mkdir -p /usr/local/kubebuilder/ && \
    curl --fail -sL "https://go.kubebuilder.io/test-tools/${KUBE_VERSION}/$os/$arch" | tar -xz --strip-components=1 -C /usr/local/kubebuilder/ && \
    curl --fail https://storage.googleapis.com/kubernetes-release/release/v$KUBE_VERSION/bin/$os/${arch}/kube-apiserver -L -o /tmp/kube-apiserver && \
    chmod +x /tmp/kube-apiserver && \
    mv /tmp/kube-apiserver /usr/local/kubebuilder/bin/kube-apiserver && \
    echo 'export PATH=$PATH:/usr/local/kubebuilder/bin' >> ~/.bashrc

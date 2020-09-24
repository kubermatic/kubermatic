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

FROM golang:1.15.1
LABEL maintainer="support@kubermatic.com"

RUN os=$(go env GOOS) && \
    arch=$(go env GOARCH) && \
    curl --fail -sL https://go.kubebuilder.io/dl/2.2.0/${os}/${arch} | tar -xz -C /tmp/ && \
    mv /tmp/kubebuilder_2.2.0_${os}_${arch} /usr/local/kubebuilder && \
    curl --fail https://storage.googleapis.com/kubernetes-release/release/v1.18.6/bin/${os}/${arch}/kube-apiserver -L -o /tmp/kube-apiserver && \
    chmod +x /tmp/kube-apiserver && \
    mv /tmp/kube-apiserver /usr/local/kubebuilder/bin/kube-apiserver && \
    echo 'export PATH=$PATH:/usr/local/kubebuilder/bin' >> ~/.bashrc

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

FROM docker.io/alpine:3.23
LABEL org.opencontainers.image.source="https://github.com/kubermatic/kubermatic/blob/main/Dockerfile"
LABEL org.opencontainers.image.vendor="Kubermatic"
LABEL org.opencontainers.image.authors="support@kubermatic.com"

ENV KUBERMATIC_CHARTS_DIRECTORY=/opt/charts/

# To support a wider range of Kubernetes userclusters, we ship multiple
# kubectl binaries and deduce which one to use based on the version skew
# policy.
ADD https://dl.k8s.io/v1.35.0/bin/linux/amd64/kubectl /usr/local/bin/kubectl-1.35
ADD https://dl.k8s.io/v1.33.7/bin/linux/amd64/kubectl /usr/local/bin/kubectl-1.33

RUN wget -O- https://get.helm.sh/helm-v3.18.5-linux-amd64.tar.gz | tar xzOf - linux-amd64/helm > /usr/local/bin/helm

# We need the ca-certs so the KKP API can verify the certificates of the OIDC server (usually Dex)
RUN chmod +x /usr/local/bin/kubectl-* /usr/local/bin/helm && apk add ca-certificates

# Do not needless copy all files from _build/ into the image.
COPY ./_build/kubermatic-operator \
     ./_build/kubermatic-installer \
     ./_build/kubermatic-webhook \
     ./_build/master-controller-manager \
     ./_build/seed-controller-manager \
     ./_build/user-cluster-controller-manager \
     ./_build/user-cluster-webhook \
     /usr/local/bin/

COPY ./charts /opt/charts

USER nobody

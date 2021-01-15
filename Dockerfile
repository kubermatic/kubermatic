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

FROM alpine:3.12
LABEL maintainer="support@kubermatic.com"

ENV KUBERMATIC_CHARTS_DIRECTORY=/opt/charts/

ADD https://storage.googleapis.com/kubernetes-release/release/v1.19.7/bin/linux/amd64/kubectl /usr/local/bin/kubectl
RUN wget -O- https://get.helm.sh/helm-v3.5.0-linux-amd64.tar.gz | tar xzOf - linux-amd64/helm > /usr/local/bin/helm

# We need the ca-certs so they api doesn't crash because it can't verify the certificate of Dex
RUN chmod +x /usr/local/bin/kubectl /usr/local/bin/helm && apk add ca-certificates

# Do not needless copy all binaries into the image.
COPY ./_build/image-loader \
     ./_build/kubermatic-api \
     ./_build/kubermatic-operator \
     ./_build/kubermatic-installer \
     ./_build/master-controller-manager \
     ./_build/owner-remover \
     ./_build/seed-controller-manager \
     ./_build/user-cluster-controller-manager \
     ./_build/userdata-openshift \
     /usr/local/bin/

COPY ./cmd/kubermatic-api/swagger.json /opt/swagger.json
COPY ./charts /opt/charts

USER nobody

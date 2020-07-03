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

ADD https://storage.googleapis.com/kubernetes-release/release/v1.18.5/bin/linux/amd64/kubectl /usr/local/bin/kubectl

# We need the ca-certs so they api doesn't crash because it can't verify the certificate of Dex
RUN chmod +x /usr/local/bin/kubectl && apk add ca-certificates

# Do not needless copy all binaries into the image.
COPY ./_build/image-loader \
     ./_build/kubermatic-api \
     ./_build/kubermatic-operator \
     ./_build/kubermatic-operator-util \
     ./_build/master-controller-manager \
     ./_build/owner-remover \
     ./_build/seed-controller-manager \
     ./_build/user-cluster-controller-manager \
     ./_build/userdata-openshift \
     /usr/local/bin/

COPY ./cmd/kubermatic-api/swagger.json /opt/swagger.json

USER nobody

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

FROM docker:18.09.0-dind

RUN apk add --no-cache -U ca-certificates wget unzip jq bash
RUN wget https://releases.hashicorp.com/vault/1.1.3/vault_1.1.3_linux_amd64.zip && \
    unzip vault_1.1.3_linux_amd64.zip && \
    rm vault_1.1.3_linux_amd64.zip && \
    mv vault /bin/

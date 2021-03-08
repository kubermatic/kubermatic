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

FROM alpine:3.12 AS builder

RUN apk update
RUN apk add curl
RUN cd tmp && curl -L --fail https://github.com/frapposelli/wwhrd/releases/download/v0.4.0/wwhrd_0.4.0_linux_amd64.tar.gz | tar -xvz
RUN /tmp/wwhrd -v

FROM golang:1.15.1

COPY --from=builder /tmp/wwhrd /usr/local/bin/
ENTRYPOINT ["wwhrd"]

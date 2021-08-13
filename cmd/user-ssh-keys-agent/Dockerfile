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

FROM docker.io/golang:1.16.4 as builder

ARG HOST_GOCACHE
WORKDIR /go/src/github.com/kubermatic/kubermatic

COPY ./cmd/user-ssh-keys-agent ./cmd/user-ssh-keys-agent
COPY ./cmd/http-prober/api ./cmd/http-prober/api
COPY ./pkg ./pkg
COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum

ENV CGO_ENABLED=0
ENV GOCACHE=/go/src/github.com/kubermatic/kubermatic/cmd/user-ssh-keys-agent/$HOST_GOCACHE

# test
RUN echo $GOCACHE
RUN ls $GOCACHE
# end test

RUN go build -v -o ./_build/user-ssh-keys-agent ./cmd/user-ssh-keys-agent

FROM docker.io/alpine:3.13
LABEL maintainer="support@kubermatic.com"

COPY --from=builder /go/src/github.com/kubermatic/kubermatic/_build/user-ssh-keys-agent /usr/local/bin/user-ssh-keys-agent
ENTRYPOINT ["/usr/local/bin/user-ssh-keys-agent"]

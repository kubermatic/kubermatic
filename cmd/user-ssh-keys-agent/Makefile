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

DOCKER_REPO ?= "quay.io/kubermatic"
GOOS ?= $(shell go env GOOS)
GOPATH ?= $(shell go env GOPATH)
GOCACHE ?= $(shell go env GOCACHE)

.PHONY: build
build:
	docker run \
		-e GOCACHE=/go/.cache/go-build \
 		-e CGO_ENABLED=0 \
		-v $(GOPATH)/src/github.com/kubermatic/kubermatic/:/go/src/github.com/kubermatic/kubermatic \
		-v $(GOCACHE):/go/.cache/go-build \
		-w /go/src/github.com/kubermatic/kubermatic/cmd/user-ssh-keys-agent \
		docker.io/golang:1.16.4 \
		go build -v -o ./_build/

.PHONY: docker
docker:
	cd ../.. && cp -r $(GOCACHE) cmd/user-ssh-keys-agent && \
	docker build \
		-t $(DOCKER_REPO)/user-ssh-keys-agent:$(TAG) \
		--build-arg HOST_GOCACHE=$(shell basename $$GOCACHE) \
		-f cmd/user-ssh-keys-agent/Dockerfile .

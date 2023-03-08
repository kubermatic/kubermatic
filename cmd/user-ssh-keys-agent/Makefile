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

KUBERMATIC_EDITION ?= ce
DOCKER_REPO ?= "quay.io/kubermatic"
LDFLAGS ?= -w -extldflags '-static'
GOTOOLFLAGS ?= -ldflags '$(LDFLAGS)' -v

export CGO_ENABLED=0

.PHONY: build
build:
	go build $(GOTOOLFLAGS) -tags "$(KUBERMATIC_EDITION)" -o ./_build/user-ssh-keys-agent

.PHONY: docker
docker: build
	docker build -t $(DOCKER_REPO)/user-ssh-keys-agent:$(TAG) .

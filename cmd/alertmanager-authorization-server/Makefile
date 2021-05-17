# Copyright 2021 The Kubermatic Kubernetes Platform contributors.
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
TAG ?= 0.1.0

.PHONY: alertmanager-authz-server
alertmanager-authz-server:
	GOOS=$(GOOS) CGO_ENABLED=0 go build -o ./_build/alertmanager-authorization-server k8c.io/kubermatic/v2/cmd/alertmanager-authorization-server

.PHONY: build
build: alertmanager-authz-server

.PHONY: clean
clean:
	rm -rf ./_build

.PHONY: docker-build
docker-build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./_build/alertmanager-authorization-server k8c.io/kubermatic/v2/cmd/alertmanager-authorization-server
	docker build -t $(DOCKER_REPO)/alertmanager-authorization-server:$(TAG) .

.PHONY: docker-push
docker-push:
	docker push $(DOCKER_REPO)/alertmanager-authorization-server:$(TAG)

test:
	go test ./...

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

default: test build

lb-updater:
	CGO_ENABLED=0 go build -o ./_build/lb-updater github.com/kubermatic/kubermatic/api/cmd/nodeport-proxy/lb-updater

envoy-manager:
	CGO_ENABLED=0 go build -o ./_build/envoy-manager github.com/kubermatic/kubermatic/api/cmd/nodeport-proxy/envoy-manager

build: envoy-manager lb-updater

clean:
	rm -f envoy-manager lb-updater

docker: build
	docker build -t quay.io/kubermatic/nodeport-proxy:$(TAG) .

test:
	go test ./...

.PHONY: test envoy-manager lb-updater

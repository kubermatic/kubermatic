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

export CGO_ENABLED ?= 0
export GOFLAGS ?= -mod=readonly -trimpath
export GO111MODULE = on
export KUBERMATIC_EDITION ?= ce
DOCKER_REPO ?= quay.io/kubermatic
REPO = $(DOCKER_REPO)/kubermatic$(shell [ "$(KUBERMATIC_EDITION)" != "ce" ] && echo "-$(KUBERMATIC_EDITION)" )
CMD = $(filter-out OWNERS nodeport-proxy kubeletdnat-controller, $(notdir $(wildcard ./cmd/*)))
GOBUILDFLAGS ?= -v
GOOS ?= $(shell go env GOOS)
TAGS ?= $(shell git describe --tags --always)
DOCKERTAGS = $(TAGS) latestbuild
DOCKER_BUILD_FLAG += $(foreach tag, $(DOCKERTAGS), -t $(REPO):$(tag))
KUBERMATICCOMMIT ?= $(shell git log -1 --format=%H)
KUBERMATICDOCKERTAG ?= $(KUBERMATICCOMMIT)
UIDOCKERTAG ?= NA
LDFLAGS += -extldflags '-static' \
  -X k8c.io/kubermatic/v2/pkg/version/kubermatic.gitHash=$(KUBERMATICCOMMIT) \
  -X k8c.io/kubermatic/v2/pkg/version/kubermatic.kubermaticDockerTag=$(KUBERMATICDOCKERTAG) \
  -X k8c.io/kubermatic/v2/pkg/version/kubermatic.uiDockerTag=$(UIDOCKERTAG)
LDFLAGS_EXTRA=-w
BUILD_DEST ?= _build
GOTOOLFLAGS ?= $(GOBUILDFLAGS) -ldflags '$(LDFLAGS_EXTRA) $(LDFLAGS)' $(GOTOOLFLAGS_EXTRA)
GOBUILDIMAGE ?= golang:1.16.1
DOCKER_BIN := $(shell which docker)

.PHONY: all
all: build test

.PHONY: build
build: $(CMD)

.PHONY: $(CMD)
$(CMD): %: $(BUILD_DEST)/%

$(BUILD_DEST)/%: cmd/% download-gocache
	GOOS=$(GOOS) go build -tags "$(KUBERMATIC_EDITION)" $(GOTOOLFLAGS) -o $@ ./cmd/$*

.PHONY: install
install:
	go install $(GOTOOLFLAGS) ./cmd/...

.PHONY: showenv
showenv:
	@go env

download-gocache:
	@./hack/ci/download-gocache.sh
	@# Prevent this from getting executed multiple times
	@touch download-gocache

.PHONY: test
test: download-gocache run-tests build-tests

.PHONY:  run-tests
run-tests:
	CGO_ENABLED=1 go test -tags "unit,$(KUBERMATIC_EDITION)" -race ./pkg/... ./cmd/... ./codegen/...

.PHONY: build-tests
build-tests:
	@# Make sure all e2e tests compile with their individual build tag
	@# without actually running them by using `-run` with a non-existing test.
	@# **Imortant:** Do not replace this with one `go test` with multiple tags,
	@# as that doesn't properly reflect if each individual tag still builds
	go test -tags "cloud,$(KUBERMATIC_EDITION)" -run nope ./pkg/test/e2e/api/...
	go test -tags "create,$(KUBERMATIC_EDITION)" -run nope ./pkg/test/e2e/api/...
	go test -tags "e2e,$(KUBERMATIC_EDITION)" -run nope ./pkg/test/e2e/api/...
	go test -tags "e2e,$(KUBERMATIC_EDITION)" -run nope ./pkg/test/e2e/nodeport-proxy/...
	go test -tags "integration,$(KUBERMATIC_EDITION)" -run nope ./pkg/... ./cmd/... ./codegen/...

.PHONY: test-integration
test-integration: CGO_ENABLED=1
test-integration: download-gocache
	@# Run integration tests and only integration tests by:
	@# * Finding all files that contain the build tag via grep
	@# * Extracting the dirname as the `go test` command doesn't play well with individual files as args
	@# * Prefixing them with `./` as that's needed by `go test` as well
	@grep --files-with-matches --recursive --extended-regexp '\+build.+integration' cmd/ pkg/ \
		|xargs dirname \
		|xargs --max-args=1 -I ^ go test -tags "integration $(KUBERMATIC_EDITION)"  -race ./^

.PHONY: test-update
test-update:
	-go test ./pkg/resources/test -update
	-go test ./pkg/provider/cloud/aws -update

.PHONY: clean
clean:
	rm -rf $(BUILD_DEST)
	@echo "Cleaned $(BUILD_DEST)"

.PHONY: docker-build
docker-build: build
ifndef DOCKER_BIN
	$(error "Docker not available in your environment, please install it and retry.")
endif
	$(DOCKER_BIN) build $(DOCKER_BUILD_FLAG) .

.PHONY: docker-push
docker-push:
ifndef DOCKER_BIN
	$(error "Docker not available in your environment, please install it and retry.")
endif
	@for tag in $(DOCKERTAGS) ; do \
		echo "docker push $(REPO):$$tag"; \
		$(DOCKER_BIN) push $(REPO):$$tag; \
	done

.PHONY: lint
lint:
	golangci-lint run \
		--verbose \
		--build-tags "$(KUBERMATIC_EDITION)" \
		--print-resources-usage \
		./pkg/... ./cmd/... ./codegen/...

.PHONY: shellcheck
shellcheck:
ifndef DOCKER_BIN
	shellcheck $$(find . -name '*.sh')
endif

.PHONY: spellcheck
spellcheck:
	./hack/verify-spelling.sh

.PHONY: cover
cover:
	./hack/cover.sh --html

.PHONY: run-controller-manager
run-controller-manager:
	./hack/run-controller.sh

.PHONY: run-api-server
run-api-server:
	./hack/run-api.sh

.PHONY: run-operator
run-operator:
	./hack/run-operator.sh

.PHONY: run-master-controller-manager
run-master-controller-manager:
	./hack/run-master-controller-manager.sh

.PHONY: verify
verify:
	./hack/verify-codegen.sh
	./hack/verify-swagger.sh
	./hack/verify-api-client.sh

.PHONY: check-dependencies
check-dependencies:
	go mod tidy
	go mod verify
	git diff --exit-code

.PHONY: gen-api-client
gen-api-client:
	./hack/gen-api-client.sh

.PHONY: shfmt
shfmt:
	shfmt -w -sr -i 2 hack

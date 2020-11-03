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
GITTAG = $(shell git describe --tags --always)
TAGS ?= $(GITTAG)
DOCKERTAGS = $(TAGS) latestbuild
DOCKER_BUILD_FLAG += $(foreach tag, $(DOCKERTAGS), -t $(REPO):$(tag))
KUBERMATICCOMMIT ?= $(shell git log -1 --format=%H)
KUBERMATICDOCKERTAG ?= $(KUBERMATICCOMMIT)
UIDOCKERTAG ?= NA
LDFLAGS += -extldflags '-static' \
  -X k8c.io/kubermatic/v2/pkg/resources.KUBERMATICCOMMIT=$(KUBERMATICCOMMIT) \
  -X k8c.io/kubermatic/v2/pkg/resources.KUBERMATICGITTAG=$(GITTAG) \
  -X k8c.io/kubermatic/v2/pkg/controller/operator/common.KUBERMATICDOCKERTAG=$(KUBERMATICDOCKERTAG) \
  -X k8c.io/kubermatic/v2/pkg/controller/operator/common.UIDOCKERTAG=$(UIDOCKERTAG)
LDFLAGS_EXTRA=-w
BUILD_DEST ?= _build
GOTOOLFLAGS ?= $(GOBUILDFLAGS) -ldflags '$(LDFLAGS_EXTRA) $(LDFLAGS)' $(GOTOOLFLAGS_EXTRA)
GOBUILDIMAGE ?= golang:1.15.1
CODESPELL_IMAGE ?= quay.io/kubermatic/codespell:1.17.1
CODESPELL_BIN := $(shell which codespell)
DOCKER_BIN := $(shell which docker)

default: all

all: build test

.PHONY: $(CMD)
build: $(CMD)

$(CMD): download-gocache
	GOOS=$(GOOS) go build -tags "$(KUBERMATIC_EDITION)" $(GOTOOLFLAGS) -o $(BUILD_DEST)/$@ ./cmd/$@

install:
	go install $(GOTOOLFLAGS) ./cmd/...

showenv:
	@go env

download-gocache:
	@./hack/ci/download-gocache.sh
	@# Prevent this from getting executed multiple times
	@touch download-gocache

test: download-gocache run-tests build-tests

run-tests:
	CGO_ENABLED=1 go test -tags "unit,$(KUBERMATIC_EDITION)" -race ./pkg/... ./cmd/... ./codegen/...

build-tests:
	@# Make sure all e2e tests compile with their individual build tag
	@# without actually running them by using `-run` with a non-existing test.
	@# **Imortant:** Do not replace this with one `go test` with multiple tags,
	@# as that doesn't properly reflect if each individual tag still builds
	go test -tags "cloud,$(KUBERMATIC_EDITION)" -run nope ./pkg/test/e2e/api/...
	go test -tags "create,$(KUBERMATIC_EDITION)" -run nope ./pkg/test/e2e/api/...
	go test -tags "e2e,$(KUBERMATIC_EDITION)" -run nope ./pkg/test/e2e/api/...
	go test -tags "integration,$(KUBERMATIC_EDITION)" -run nope ./pkg/... ./cmd/... ./codegen/...

test-integration : CGO_ENABLED = 1
test-integration: download-gocache
	@# Run integration tests and only integration tests by:
	@# * Finding all files that contain the build tag via grep
	@# * Extracting the dirname as the `go test` command doesn't play well with individual files as args
	@# * Prefixing them with `./` as that's needed by `go test` as well
	@grep --files-with-matches --recursive --extended-regexp '\+build.+integration' cmd/ pkg/ \
		|xargs dirname \
		|xargs --max-args=1 -I ^ go test -tags "integration $(KUBERMATIC_EDITION)"  -race ./^

test-update:
	-go test ./pkg/userdata/openshift -update
	-go test ./pkg/resources/test -update
	-go test ./pkg/provider/cloud/aws -update
	-go test ./pkg/controller/seed-controller-manager/openshift -update
	-go test ./pkg/controller/seed-controller-manager/openshift/resources -update
	-go test ./codegen/openshift_versions -update

clean:
	rm -f $(TARGET)
	@echo "Cleaned $(BUILD_DEST)"

docker-build: build
ifndef DOCKER_BIN
	$(error "Docker not available in your environment, please install it and retry.")
endif
	$(DOCKER_BIN) build $(DOCKER_BUILD_FLAG) .

docker-push:
ifndef DOCKER_BIN
	$(error "Docker not available in your environment, please install it and retry.")
endif
	@for tag in $(DOCKERTAGS) ; do \
		echo "docker push $(REPO):$$tag"; \
		$(DOCKER_BIN) push $(REPO):$$tag; \
	done

gittag:
	@echo $(GITTAG)

lint:
	./hack/ci/run-lint.sh

shellcheck:
ifndef DOCKER_BIN
	shellcheck $$(find . -name '*.sh')
endif

spellcheck:
ifndef CODESPELL_BIN
	$(error "codespell not available in your environment, use spellcheck-in-docker if you have Docker installed.")
endif
	$(CODESPELL_BIN) -S *.png,*.po,.git,*.jpg,*.mod,*.sum,*.woff,*.woff2,swagger.json,*/_build/*,*/_dist/*,./vendor/* -I .codespell.exclude -f

spellcheck-in-docker:
ifndef DOCKER_BIN
	$(error "Docker not available in your environment, please install it and retry.")
endif
	$(DOCKER_BIN) run -it -v ${PWD}:/kubermatic -w /kubermatic $(CODESPELL_IMAGE) make spellcheck

cover:
	./hack/cover.sh --html

run-controller-manager:
	./hack/run-controller.sh

run-api-server:
	./hack/run-api.sh

run-operator:
	./hack/run-operator.sh

run-master-controller-manager:
	./hack/run-master-controller-manager.sh

verify:
	./hack/verify-codegen.sh
	./hack/verify-swagger.sh
	./hack/verify-api-client.sh

check-dependencies:
	go mod tidy
	go mod verify
	git diff --exit-code

gen-api-client:
	./hack/gen-api-client.sh

.PHONY: build install test cover docker-build docker-push run-controller-manager run-api-server run-rbac-generator test-update-fixture run-tests build-tests $(TARGET)

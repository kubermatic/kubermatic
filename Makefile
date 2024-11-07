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
CMD ?= $(filter-out OWNERS nodeport-proxy kubeletdnat-controller network-interface-manager, $(notdir $(wildcard ./cmd/*)))
GOBUILDFLAGS ?= -v
GOOS ?= $(shell go env GOOS)
GIT_VERSION = $(shell git describe --tags --always --match='v*')
TAGS ?= $(GIT_VERSION)
KUBERMATICCOMMIT ?= $(shell git log -1 --format=%H)
KUBERMATICDOCKERTAG ?= $(KUBERMATICCOMMIT)
UIDOCKERTAG ?= NA
LDFLAGS += -extldflags '-static' \
  -X k8c.io/kubermatic/v2/pkg/version/kubermatic.gitVersion=$(GIT_VERSION) \
  -X k8c.io/kubermatic/v2/pkg/version/kubermatic.kubermaticContainerTag=$(KUBERMATICDOCKERTAG) \
  -X k8c.io/kubermatic/v2/pkg/version/kubermatic.uiContainerTag=$(UIDOCKERTAG)
LDFLAGS_EXTRA=-w
BUILD_DEST ?= _build
GOTOOLFLAGS ?= $(GOBUILDFLAGS) -ldflags '$(LDFLAGS_EXTRA) $(LDFLAGS)' $(GOTOOLFLAGS_EXTRA)

DOCKER_REPO ?= quay.io/kubermatic
KKP_REPO = $(DOCKER_REPO)/kubermatic$(shell [ "$(KUBERMATIC_EDITION)" != "ce" ] && echo "-$(KUBERMATIC_EDITION)" )
DOCKER_TAGS = $(TAGS) latestbuild
DOCKER_VERSION_LABEL = org.opencontainers.image.version=$(KUBERMATICDOCKERTAG)
DOCKER_BUILD_FLAG = --label "$(VERSION_LABEL)"
DOCKER_BUILD_FLAG += $(foreach tag, $(DOCKER_TAGS), -t $(KKP_REPO):$(tag))


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

download-gocache:
	@./hack/ci/download-gocache.sh
	@# Prevent this from getting executed multiple times
	@touch download-gocache

.PHONY: test
test: download-gocache run-tests build-tests

.PHONY:  run-tests
run-tests:
	./hack/run-tests.sh

.PHONY: build-tests
build-tests:
	@# Make sure all e2e tests compile with their individual build tag
	@# without actually running them by using `-run` with a non-existing test.
	@# **Important:** Do not replace this with one `go test` with multiple tags,
	@# as that doesn't properly reflect if each individual tag still builds
	go test -tags "e2e,$(KUBERMATIC_EDITION)" -run nope ./pkg/test/e2e/nodeport-proxy/...
	go test -tags "integration,$(KUBERMATIC_EDITION)" -run nope ./pkg/... ./cmd/... ./codegen/...

.PHONY: test-integration
test-integration:
	./hack/run-integration-tests.sh

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
	docker build $(DOCKER_BUILD_FLAG) --label "org.opencontainers.image.version=$(KUBERMATICDOCKERTAG)" .


docker buildx build \
      --load \
      --platform "linux/$arch" \
      --build-arg "GOPROXY=${GOPROXY:-}" \
      --build-arg "GOCACHE=/go/src/k8c.io/kubermatic/.gocache" \
      --build-arg "KUBERMATIC_EDITION=$KUBERMATIC_EDITION" \
      --provenance false \
      --file "$file" \
      --tag "$repository:$tag-$arch" \
      --label "$VERSION_LABEL" \
      $context

.PHONY: docker-push
docker-push:
	@for tag in $(DOCKER_TAGS) ; do \
		echo "docker push $(REPO):$$tag"; \
		docker push $(REPO):$$tag; \
	done

.PHONY: lint
lint: lint-sdk
	golangci-lint run \
		--verbose \
		--print-resources-usage \
		./pkg/... ./cmd/... ./codegen/...

.PHONY: lint-sdk
lint-sdk:
	make -C sdk lint

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
	./hack/coverage.sh --html

.PHONY: run-controller-manager
run-controller-manager:
	./hack/run-controller.sh

.PHONY: run-operator
run-operator:
	./hack/run-operator.sh

.PHONY: run-master-controller-manager
run-master-controller-manager:
	./hack/run-master-controller-manager.sh

.PHONY: verify
verify:
	./hack/verify-codegen.sh
	./hack/verify-import-order.sh

.PHONY: check-dependencies
check-dependencies:
	go mod tidy
	go mod verify
	git diff --exit-code

.PHONY: shfmt
shfmt:
	shfmt -w -sr -i 2 hack

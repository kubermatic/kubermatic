# Copyright YEAR The Kubermatic Kubernetes Platform contributors.
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

export CGO_ENABLED?=0
KUBERMATIC_EDITION?=ce
REPO=quay.io/kubermatic/kubermatic$(shell [ "$(KUBERMATIC_EDITION)" != "ce" ] && echo "-$(KUBERMATIC_EDITION)" )
CMD=$(filter-out OWNERS nodeport-proxy kubeletdnat-controller, $(notdir $(wildcard ./cmd/*)))
GOBUILDFLAGS?=-v
GITTAG=$(shell git describe --tags --always)
TAGS?=$(GITTAG)
DOCKERTAGS=$(TAGS) latestbuild
DOCKER_BUILD_FLAG += $(foreach tag, $(DOCKERTAGS), -t $(REPO):$(tag))
KUBERMATICCOMMIT?=$(shell git log -1 --format=%H)
KUBERMATICDOCKERTAG?=$(KUBERMATICCOMMIT)
UIDOCKERTAG?=NA
LDFLAGS += -extldflags '-static' \
  -X github.com/kubermatic/kubermatic/api/pkg/resources.KUBERMATICCOMMIT=$(KUBERMATICCOMMIT) \
  -X github.com/kubermatic/kubermatic/api/pkg/resources.KUBERMATICGITTAG=$(GITTAG) \
  -X github.com/kubermatic/kubermatic/api/pkg/controller/operator/common.KUBERMATICDOCKERTAG=$(KUBERMATICDOCKERTAG) \
  -X github.com/kubermatic/kubermatic/api/pkg/controller/operator/common.UIDOCKERTAG=$(UIDOCKERTAG)
HAS_DEP:= $(shell command -v dep 2> /dev/null)
HAS_GIT:= $(shell command -v git 2> /dev/null)
BUILD_DEST?=_build
GOTOOLFLAGS?=$(GOBUILDFLAGS) -ldflags '-w $(LDFLAGS)'

default: all

all: check vendor build test

.PHONY: $(CMD)
build: $(CMD)

$(CMD): download-gocache
	go build -tags "$(KUBERMATIC_EDITION)" $(GOTOOLFLAGS) -o $(BUILD_DEST)/$@ ./cmd/$@

install:
	go install $(GOTOOLFLAGS) ./cmd/...

showenv:
	@go env

check: gofmt lint

download-gocache:
	@./hack/ci/ci-download-gocache.sh
	@# Prevent this from getting executed multiple times
	@touch download-gocache

test: download-gocache
	CGO_ENABLED=1 go test -tags "unit,$(KUBERMATIC_EDITION)" -race ./...
	@# Make sure all e2e tests compile with their individual build tag
	@# without actually running them by using `-run` with a non-existing test.
	@# **Imortant:** Do not replace this with one `go test` with multiple tags,
	@# as that doesn't properly reflect if each individual tag still builds
	go test -tags "cloud,$(KUBERMATIC_EDITION)" -run nope ./pkg/test/e2e/api/...
	go test -tags "create,$(KUBERMATIC_EDITION)" -run nope ./pkg/test/e2e/api/...
	go test -tags "e2e,$(KUBERMATIC_EDITION)" -run nope ./pkg/test/e2e/api/...
	go test -tags "integration,$(KUBERMATIC_EDITION)" -run nope ./...

test-integration : CGO_ENABLED = 1
test-integration: download-gocache
	@# Run integration tests and only integration tests by:
	@# * Finding all files that contain the build tag via grep
	@# * Extracting the dirname as the `go test` command doesn't play well with individual files as args
	@# * Prefixing them with `./` as thats needed by `go test` as well
	@grep --files-with-matches --recursive --extended-regexp '\+build.+integration' cmd/ pkg/ \
		|xargs dirname \
		|xargs --max-args=1 -I ^ go test -tags "integration $(KUBERMATIC_EDITION)"  -race ./^

test-update:
	-go test ./... -update

clean:
	rm -f $(TARGET)
	@echo "Cleaned $(BUILD_DEST)"

docker-build: build
	docker build $(DOCKER_BUILD_FLAG) .

docker-push:
	@for tag in $(DOCKERTAGS) ; do \
		echo "docker push $(REPO):$$tag"; \
		docker push $(REPO):$$tag; \
	done

vendor:
	dep ensure -v

gittag:
	@echo $(GITTAG)

GFMT=find . -not \( \( -wholename "./vendor" \) -prune \) -name "*.go" | xargs gofmt -l
gofmt:
	@UNFMT=$$($(GFMT)); if [ -n "$$UNFMT" ]; then echo "gofmt needed on" $$UNFMT && exit 1; fi

fix:
	@UNFMT=$$($(GFMT)); if [ -n "$$UNFMT" ]; then echo "goimports -w" $$UNFMT; goimports -w $$UNFMT; fi

lint:
	./hack/ci/ci-run-lint.sh

shellcheck:
	shellcheck $$(find . -name '*.sh')

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

verify: gofmt
	./hack/verify-codegen.sh
	./hack/verify-swagger.sh
	./hack/verify-api-client.sh

check-dependencies:
	# We need mercurial for bitbucket.org/ww/goautoneg, otherwise dep hangs forever
	which hg >/dev/null 2>&1 || apt update && apt install -y mercurial
	dep version || go get -u github.com/golang/dep/cmd/dep
	dep check
	git diff --exit-code

license-validation:
	wwhrd check -f ../allowed_licensed.yaml

gen-api-client:
	./hack/gen-api-client.sh

.PHONY: vendor build install test check cover docker-build docker-push run-controller-manager run-api-server run-rbac-generator test-update-fixture $(TARGET)

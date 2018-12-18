export CGO_ENABLED?=0
REPO=kubermatic/api
GO?=go
CMD=$(filter-out OWNERS, $(notdir $(wildcard ./cmd/*)))
GOBUILDFLAGS?=-v
GITTAG=$(shell git describe --tags --always)
TAGS?=$(GITTAG)
DOCKERTAGS=$(TAGS) latestbuild
DOCKER_BUILD_FLAG += $(foreach tag, $(DOCKERTAGS), -t $(REPO):$(tag))
KUBERMATICCOMMIT=$(shell git log -1 --format=%H)
LDFLAGS += -extldflags '-static' -X github.com/kubermatic/kubermatic/api/pkg/resources.KUBERMATICCOMMIT=$(KUBERMATICCOMMIT)
HAS_DEP:= $(shell command -v dep 2> /dev/null)
HAS_GIT:= $(shell command -v git 2> /dev/null)
BUILD_DEST?=_build
GOTOOLFLAGS?=$(GOBUILDFLAGS) -ldflags '-w $(LDFLAGS)'

default: all

all: check vendor build test

build: $(CMD)

$(CMD):
	$(GO) build $(GOTOOLFLAGS) -o $(BUILD_DEST)/$@ ./cmd/$@

install:
	$(GO) install $(GOTOOLFLAGS) ./cmd/...

showenv:
	@$(GO) env

check: gofmt lint

test:
	$(GO) test -v ./...

test-update:
	-$(GO) test -v ./... -update

clean:
	rm -f $(TARGET)
	@echo "Cleaned $(BUILD_DEST)"

docker-build:
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
	golangci-lint run

cover:
	./hack/cover.sh --html

run-controller-manager:
	./hack/run-controller.sh

run-api-server:
	./hack/run-api.sh

run-rbac-generator:
	./hack/run-rbac-generator.sh

verify: gofmt
		./hack/verify-codegen.sh
		./hack/verify-swagger.sh

check-dependencies:
	# We need mercurial for bitbucket.org/ww/goautoneg, otherwise dep hangs forever
	which hg >/dev/null 2>&1 || apt update && apt install -y mercurial
	dep version || go get -u github.com/golang/dep/cmd/dep
	dep status
	git diff --exit-code

license-validation:
	wwhrd check -f ../allowed_licensed.yaml

.PHONY: vendor build install test check cover docker-build docker-push run-controller-manager run-api-server run-rbac-generator test-update-fixture $(TARGET)

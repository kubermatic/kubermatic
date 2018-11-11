REPO=kubermatic/api
GO?=go
CMD=$(shell ls ./cmd/)
GOBUILDFLAGS += -v
GITTAG=$(shell git describe --tags --always)
TAGS?=$(GITTAG)
DOCKERTAGS=$(TAGS) latestbuild
DOCKER_BUILD_FLAG += $(foreach tag, $(DOCKERTAGS), -t $(REPO):$(tag))
KUBERMATICCOMMIT=$(shell git log -1 --format=%H)
LDFLAGS += -extldflags '-static' -X github.com/kubermatic/kubermatic/api/pkg/resources.KUBERMATICCOMMIT=$(KUBERMATICCOMMIT)
HAS_GOMETALINTER:= $(shell command -v gometalinter 2> /dev/null)
HAS_DEP:= $(shell command -v dep 2> /dev/null)
HAS_GIT:= $(shell command -v git 2> /dev/null)

default: all

all: check test build

build: $(CMD)

$(CMD): vendor
	CGO_ENABLED=0 $(GO) build $(GOBUILDFLAGS) -ldflags '-w $(LDFLAGS)' -o _build/$@ ./cmd/$@

check: gofmt lint
test: vendor
	@$(GO) generate ./pkg/controller/ipam/
	@$(GO) test -v ./...

test-update:
	-@$(GO) test -v ./... -update

clean:
	@cd _build
	@rm -f $(CMD)
	@echo "Cleaned _build"

install: vendor
	@for PRODUCT in $(CMD); do \
		$(GO) install -v -ldflags '-w $(LDFLAGS)' github.com/kubermatic/kubermatic/api/cmd/$$PRODUCT ; \
	done

docker-build:
	docker build $(DOCKER_BUILD_FLAG) .

docker-push:
	@for tag in $(DOCKERTAGS) ; do \
		echo "docker push $(REPO):$$tag"; \
		docker push $(REPO):$$tag; \
	done

vendor: Gopkg.lock Gopkg.toml
ifndef HAS_GIT
	$(error You must install git)
endif
ifndef HAS_DEP
	go get -u github.com/golang/dep/cmd/dep
endif
	dep ensure -v

gittag:
	@echo $(GITTAG)

GFMT=find . -not \( \( -wholename "./vendor" \) -prune \) -name "*.go" | xargs gofmt -l
gofmt:
	@UNFMT=$$($(GFMT)); if [ -n "$$UNFMT" ]; then echo "gofmt needed on" $$UNFMT && exit 1; fi
fix:
	@UNFMT=$$($(GFMT)); if [ -n "$$UNFMT" ]; then echo "goimports -w" $$UNFMT; goimports -w $$UNFMT; fi

lint:
ifndef HAS_GOMETALINTER
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install
endif
	{ \
	set -e ;\
	echo "gometalinter" ;\
	gometalinter --config gometalinter.json ./... ;\
	echo "forbidden functions" ;\
	./hack/verify-forbidden-functions.sh ;\
	GOFILES=$$(find . -type f -name '*.go' -not -path "./vendor/*") ;\
	echo "misspell -error -locale US" ;\
	misspell -error -locale US $$GOFILES ;\
	}

cover:
	./hack/cover.sh --html

run-controller-manager:
	./hack/run-controller.sh

run-api-server:
	./hack/run-api.sh

run-rbac-generator:
	./hack/run-rbac-generator.sh

.PHONY: build test check cover docker-build docker-push run-controller-manager run-api-server run-rbac-generator test-update-fixture $(CMD)

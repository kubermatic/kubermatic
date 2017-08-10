REPO=kubermatic/api
SHELL:=/bin/bash
GO?=go
GOBUILD=$(GO) build
GOTEST=$(GO) test
GOINSTALL=$(GO) install
CMD=kubermatic-api kubermatic-cluster-controller client
GOBUILDFLAGS= -i
GITTAG=$(shell git describe --tags --always)
GOFLAGS=
TAGS?=$(GITTAG)
DOCKERTAGS=$(TAGS) latestbuild
DOCKER_BUILD_FLAG += $(foreach tag, $(DOCKERTAGS), -t $(REPO):$(tag))
HAS_GOMETALINTER:= $(shell command -v gometalinter;)
HAS_DEP:= $(shell command -v dep;)
HAS_GIT:= $(shell command -v git;)

default: all

all: check test build

build: $(CMD)

$(CMD): vendor
	$(GOFLAGS) $(GOBUILD) $(GOBUILDFLAGS) -o _build/$@ github.com/kubermatic/api/cmd/$@

check: gofmt lint
test:
	@$(GOTEST) -v $$($(GO) list ./... | grep -v /vendor/)

clean:
	@cd _build
	@rm -f $(CMD)
	@echo "Cleaned _build"

install: vendor
	@for PRODUCT in $(CMD); do \
		$(GOINSTALL) github.com/kubermatic/api/cmd/$$PRODUCT ; \
	done

docker-build:
	docker build $(DOCKER_BUILD_FLAG) .

docker-push:
	@for tag in $(DOCKERTAGS) ; do \
		echo "docker push $(REPO):$$tag"; \
		docker push $(REPO):$$tag; \
	done

e2e: client-up
	docker run -it -v  $(CURDIR)/_artifacts/kubeconfig:/workspace/kubermatickubeconfig kubermatic/e2e-conformance:1.6

vendor:
ifndef HAS_GIT
	$(error You must install git)
endif
	dep ensure

bootstrap: vendor
ifndef HAS_GOMETALINTER
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install
endif
ifndef HAS_DEP
	go get -u github.com/golang/dep/cmd/dep
endif

client-up:
	mkdir -p $(CURDIR)/_artifacts
	docker run -v $(CURDIR)/_artifacts/:/_artifacts -it $(REPO):latestbuild /client up

client-down:
	docker run -it $(REPO):latestbuild /client purge

gittag:
	@echo $(GITTAG)

GFMT=find . -not \( \( -wholename "./vendor" \) -prune \) -name "*.go" | xargs gofmt -l
gofmt:
	@UNFMT=$$($(GFMT)); if [ -n "$$UNFMT" ]; then echo "gofmt needed on" $$UNFMT && exit 1; fi
fix:
	@UNFMT=$$($(GFMT)); if [ -n "$$UNFMT" ]; then echo "goimports -w" $$UNFMT; goimports -w $$UNFMT; fi

lint:
	{ \
	set -e ;\
	PACKAGES=$$(go list ./... | grep -v /vendor/) ;\
	go vet $$PACKAGES ;\
	golint $$PACKAGES ;\
	errcheck -blank $$PACKAGES ;\
	varcheck $$PACKAGES ;\
	structcheck $$PACKAGES ;\
	gosimple $$PACKAGES ;\
	unused $$PACKAGES ;\
	GOFILES=$$(find . -type f -name '*.go' -not -path "./vendor/*") ;\
	misspell -error -locale US $$GOFILES ;\
	}

.PHONY: build test check e2e-build client-build client-down client-up e2e docker-build docker-push bootstrap $(CMD)


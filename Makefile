SHELL:=/bin/bash
GO?=go
GOBUILD=$(GO) build
GOTEST=$(GO) test
GOINSTALL=$(GO) install
CMD=kubermatic-api kubermatic-cluster-controller
SHELL=/bin/bash
CMD=kubermatic-api kubermatic-cluster-controller client
GOBUILD=go build
GOBUILDFLAGS= -i
REPO=kubermatic/api
GITTAG=$(shell git describe --tags --always)
GOFLAGS=
DOCKERTAGS=$(TAGS) latestbuild
DOCKER_BUILD_FLAG += $(foreach tag, $(DOCKERTAGS), -t $(REPO):$(tag))

default: all

all: check test build

.PHONY: $(CMD)
$(CMD):
	$(GOFLAGS) $(GOBUILD) $(GOBUILDFLAGS) -o _build/$@ github.com/kubermatic/api/cmd/$@

build: $(CMD)


test:
	$(GOTEST) -v $$($(GO) list ./... | grep -v /vendor/)

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

check: gofmt lint

clean:
	@cd _build
	@rm -f $(CMD)
	@echo "Cleaned _build"

install:
	for PRODUCT in $(CMD); do \
		$(GOINSTALL) github.com/kubermatic/api/cmd/$$PRODUCT ; \
	done

docker-build: GOFLAGS := $(GOFLAGS) GOOS=linux CGO_ENABLED=0
docker-build: GOBUILDFLAGS := $(GOBUILDFLAGS) -ldflags "-s" -a -installsuffix cgo
docker-build: build
	docker build $(DOCKER_BUILD_FLAG) .

docker-push:
	@for tag in $(DOCKERTAGS) ; do \
		echo "docker push $(REPO):$$tag"; \
		docker push $(REPO):$$tag; \
	done

e2e:
	docker run -it -v  $(CURDIR)/_artifacts/kubeconfig:/workspace/kubermatickubeconfig kubermatic/e2e-conformance:1.6
HAS_GOMETALINTER:= $(shell command -v gometalinter;)
HAS_GLIDE:= $(shell command -v glide;)
HAS_GIT:= $(shell command -v git;)

vendor:
	glide install --strip-vendor

.PHONY: bootstrap
bootstrap:
ifndef HAS_GOMETALINTER
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install
endif
ifndef HAS_GLIDE
	go get -u github.com/Masterminds/glide
endif
ifndef HAS_GIT
	$(error You must install git)
endif
	glide install --strip-vendor

client-up: docker-build
	mkdir -p $(CURDIR)/_artifacts
	docker run -v $(CURDIR)/_artifacts/:/_artifacts -it $(REPO):$(GITTAG) ./client up

client-down:
	docker run -it $(REPO):$(GITTAG) ./client purge

gittag:
	@echo $(GITTAG)

.PHONY: build test check e2e-build client-build client-down client-up e2e docker-build docker-push

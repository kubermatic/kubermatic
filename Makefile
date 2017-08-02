SHELL:=/bin/bash
GO?=go
GOBUILD=$(GO) build
GOTEST=$(GO) test
GOINSTALL=$(GO) install
CMD=kubermatic-api kubermatic-cluster-controller
REPO=kubermatic/api
TAGS=dev
BUILD_FLAG+= $(foreach tag, $(TAGS), -t $(REPO):$(tag))

default: all

all: check test build

.PHONY: $(CMD)
$(CMD):
	$(GOBUILD) github.com/kubermatic/api/cmd/$@

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
	rm -f $(CMD)

install: 
	for PRODUCT in $(CMD); do \
		$(GOINSTALL) github.com/kubermatic/api/cmd/$$PRODUCT ; \
	done

docker-build:
	docker build $(BUILD_FLAG) .

docker-push:
	for TAG in $(TAGS) ; do \
		docker push $(REPO):$$TAG ; \
	done

HAS_GOMETALINTER:= $(shell command -v gometalinter;)
HAS_GLIDE:= $(shell command -v glide;)
HAS_GIT:= $(shell command -v git;)

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

.PHONY: build test check

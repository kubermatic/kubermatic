SHELL=/bin/bash
CMD=kubermatic-api kubermatic-cluster-controller client
GOBUILD=go build
GOBUILDFLAGS= -i
REPO=kubermatic/api
GITTAG=$(shell date +v%Y%m%d)-$(shell git describe --tags --always --dirty)
GOFLAGS=

default: all

all: check test build

.PHONY: $(CMD)
$(CMD):
	$(GOFLAGS) $(GOBUILD) $(GOBUILDFLAGS) -o _build/$@ github.com/kubermatic/api/cmd/$@

build: $(CMD)

test:
	go test -v $$(go list ./... | grep -v /vendor/)

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
	glide install --strip-vendor

docker-build: GOFLAGS := $(GOFLAGS) GOOS=linux CGO_ENABLED=0
docker-build: GOBUILDFLAGS := $(GOBUILDFLAGS) -ldflags "-s" -a -installsuffix cgo
docker-build: build
	docker build -t $(REPO):$(GITTAG) .

docker-push:
	docker push $(REPO)

e2e-build:
	$(MAKE) -C hack/e2e build

e2e: e2e-build
	docker run -it -v $(KUBECONFIG):/workspace/kubermatickubeconfig kubermatic.io/api/e2e-conformance:1.6

client-up: docker-build
	docker run -v kubeconfgi:kubeconfig -it $(REPO):$(GITTAG) ./client up

client-down:
	docker run -it $(REPO):$(GITTAG) ./client purge

.PHONY: build test check e2e-build client-build client-down client-up e2e docker-build docker-push

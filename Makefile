SHELL=/bin/bash
CMD=kubermatic-api kubermatic-cluster-controller
GOBUILD=go build
REPO=kubermatic/api

default: all

all: check test build

.PHONY: $(CMD)
$(CMD):
	$(GOBUILD) github.com/kubermatic/api/cmd/$@

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
	gosimple $$PACKAGES ;\
	unused $$PACKAGES ;\
	GOFILES=$$(find . -type f -name '*.go' -not -path "./vendor/*") ;\
	misspell -error -locale US $$GOFILES ;\
	}

check: gofmt lint

clean:
	rm -f $(CMD)

install:
	glide install --strip-vendor

docker:
	docker build -t $(REPO) .

push: docker
	docker push $(REPO)

.PHONY: build test check

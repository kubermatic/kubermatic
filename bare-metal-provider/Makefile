SHELL=/bin/bash
CMD=bare-metal-provider
GOBUILD=go build
REPO=kubermatic/bare-metal-provider

default: all

all: check test build

.PHONY: $(CMD)
$(CMD):
	$(GOBUILD) github.com/kubermatic/kubermatic/bare-metal-provider/cmd/$@

build: $(CMD)

test:
	go test -v $$(go list ./... | grep -v /vendor/)

GFMT=find . -not \( \( -wholename "./vendor" \) -prune \) -name "*.go" | xargs gofmt -l
gofmt:
	@UNFMT=$$($(GFMT)); if [ -n "$$UNFMT" ]; then echo "gofmt needed on" $$UNFMT && exit 1; fi
fix:
	@UNFMT=$$($(GFMT)); if [ -n "$$UNFMT" ]; then echo "goimports -w" $$UNFMT; goimports -w $$UNFMT; fi

gometalinter:
	gometalinter --enable-all --disable=gas --disable=vet --disable=lll --disable=gotype --disable=dupl --vendor --deadline=999s --cyclo-over=13 ./...

check: gofmt

.PHONY: build test check

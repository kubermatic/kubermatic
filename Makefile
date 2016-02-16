SHELL=/bin/bash
CMD=kubermatic-api kubermatic-cluster-controller
GOBUILD=go build

default: all

all: check test build

$(CMD):
	$(GOBUILD) github.com/kubermatic/api/cmd/$@

build: $(CMD)

test:
	go test -v $$(go list ./... | grep -v /vendor/)

gofmt:
	UNFMT=$$(find . -not \( \( -wholename "./vendor" \) -prune \) -name "*.go" | xargs gofmt -l); if [ -n "$$UNFMT" ]; then echo "gofmt needed on" $$UNFMT && exit 1; fi

gometalinter:
	gometalinter \
		--vendor \
		--cyclo-over=13 \
		--tests \
		--deadline=120s \
		--dupl-threshold=53 \
		--disable=gotype --disable=aligncheck --disable=structcheck --disable=interfacer --disable=deadcode --disable=gocyclo --disable=dupl \
		./...

check: gofmt gometalinter

clean:
	rm -f $(CMD)

.PHONY: build test check

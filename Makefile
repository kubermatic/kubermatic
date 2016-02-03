SHELL=/bin/bash

default: all

all: check test build

build:
	go build github.com/kubermatic/api/cmd/kubermatic-api

test:
	go test $$(go list ./... | grep -v /vendor/)

gofmt:
	UNFMT=$$(find . -not \( \( -wholename "./vendor" \) -prune \) -name "*.go" | xargs gofmt -l); if [ -n "$$UNFMT" ]; then echo "gofmt needed on" $$UNFMT && exit 1; fi

gometalinter:
	gometalinter \
		--vendor \
		--cyclo-over=12 \
		--tests \
		--deadline=120s \
		--dupl-threshold=53 \
		--disable=gotype --disable=aligncheck --disable=structcheck --disable=interfacer --disable=deadcode --disable=dupl \
		./...

check: gofmt gometalinter

clean:
	rm -f kubermatic-api

.PHONY: build test check

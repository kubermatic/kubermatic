default: all

all: check test build

build:
	go build github.com/kubermatic/api/cmd/kubermatic-api

test:
	go test $$(go list ./... | grep -v /vendor/)

check:
	gometalinter --vendor --cyclo-over=12 --tests --deadline=120s --disable=gotype --disable=aligncheck --disable=structcheck --disable=interfacer ./...

clean:
	rm -f kubermatic-api

.PHONY: build test check

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

docker: $(CMD)
	@if [ "$$GOOS" != linux ]; then echo "Run make with GOOS=linux"; exit 1; fi
	docker build -t $(REPO) .

push: docker
	docker push $(REPO)

.PHONY: build test check

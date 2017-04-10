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
		--deadline=600s \
		--dupl-threshold=53 \
		--concurrency=2 \
		--exclude="vendor" \
		--disable=gotype --disable=aligncheck --disable=unconvert --disable=structcheck --disable=interfacer --disable=deadcode --disable=gocyclo --disable=dupl --disable=gosimple --disable=gas --disable=vet --disable=vetshadow\
		./...

check: gofmt gometalinter

clean:
	rm -f $(CMD)

install:
	glide install --strip-vendor

docker: $(CMD)
	@if [ "$$GOOS" != linux ]; then echo "Run make with GOOS=linux"; exit 1; fi
	docker build -t $(REPO) .

push: docker
	docker push $(REPO)

.PHONY: alpine-3.1.tar.bz2
alpine-3.1.tar.bz2:
	docker run -i alpine:3.1 /bin/sh -c "apk add -U ca-certificates && rm -rf /var/cache/apk/*"
	docker export $$(docker ps -l -q) | bzip2 > $@

.PHONY: build test check

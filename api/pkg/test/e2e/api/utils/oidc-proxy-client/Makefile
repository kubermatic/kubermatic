export CGO_ENABLED?=0
GO?=go
GOBUILDFLAGS?=-v
LDFLAGS += -extldflags '-static'
BUILD_DEST?=_build
GOTOOLFLAGS?=$(GOBUILDFLAGS) -ldflags '-w $(LDFLAGS)'
BINARYNAME=oidc-proxy-client

default: all

all: check build

build:
	$(GO) build $(GOTOOLFLAGS) -o $(BUILD_DEST)/$(BINARYNAME)

check: fix gofmt

GFMT=find . -not \( \( -wholename "./vendor" \) -prune \) -name "*.go" | xargs gofmt -l
gofmt:
	@UNFMT=$$($(GFMT)); if [ -n "$$UNFMT" ]; then echo "gofmt needed on" $$UNFMT && exit 1; fi

fix:
	@UNFMT=$$($(GFMT)); if [ -n "$$UNFMT" ]; then echo "goimports -w" $$UNFMT; goimports -w $$UNFMT; fi

run: build
	./$(BUILD_DEST)/$(BINARYNAME) \
		--client-id=$(KUBERMATIC_OIDC_CLIENT_ID) \
		--client-secret=$(KUBERMATIC_OIDC_CLIENT_SECRET) \
		--issuer=$(KUBERMATIC_OIDC_ISSUER) \
		--redirect-uri=$(KUBERMATIC_OIDC_REDIRECT_URI) \
		--debug

.PHONY: check build run

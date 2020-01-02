export CGO_ENABLED?=0
GO?=go
GOBUILDFLAGS?=-v
LDFLAGS += -extldflags '-static'
BUILD_DEST?=_build
GOTOOLFLAGS?=$(GOBUILDFLAGS) -ldflags '-w $(LDFLAGS)'
BINARYNAME=oidc-proxy-client
KUBERMATIC_OIDC_CLIENT_ID ?= kubermatic
KUBERMATIC_OIDC_CLIENT_SECRET ?= ZXhhbXBsZS1hcHAtc2VjcmV0
KUBERMATIC_OIDC_ISSUER ?= http://dex.oauth:5556/dex
KUBERMATIC_OIDC_REDIRECT_URI ?= http://localhost:8000

default: build

build:
	$(GO) build $(GOTOOLFLAGS) -o $(BUILD_DEST)/$(BINARYNAME)

run: build
	./$(BUILD_DEST)/$(BINARYNAME) \
		--client-id=$(KUBERMATIC_OIDC_CLIENT_ID) \
		--client-secret=$(KUBERMATIC_OIDC_CLIENT_SECRET) \
		--issuer=$(KUBERMATIC_OIDC_ISSUER) \
		--redirect-uri=$(KUBERMATIC_OIDC_REDIRECT_URI) \
		--debug

.PHONY: build run

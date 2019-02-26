default: lint test build

lb-updater:
	CGO_ENABLED=0 go build -o ./_build/lb-updater -a -installsuffix cgo github.com/kubermatic/nodeport-proxy/cmd/lb-updater

envoy-manager:
	CGO_ENABLED=0 go build -o ./_build/envoy-manager -a -installsuffix cgo github.com/kubermatic/nodeport-proxy/cmd/envoy-manager

build: envoy-manager lb-updater

clean:
	rm -f envoy-manager lb-updater

docker: build
	docker build -t quay.io/kubermatic/nodeport-proxy:$(TAG) .
	docker push quay.io/kubermatic/nodeport-proxy:$(TAG)

test:
	go test ./...

lint:
	golangci-lint run -E gosec -E interfacer -E goconst -E gofmt -E goimports -E misspell -E unparam -E prealloc ./...

.PHONY: test lint envoy-manager lb-updater

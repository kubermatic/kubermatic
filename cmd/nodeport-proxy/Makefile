default: test build

lb-updater:
	CGO_ENABLED=0 go build -o ./_build/lb-updater github.com/kubermatic/kubermatic/api/cmd/nodeport-proxy/lb-updater

envoy-manager:
	CGO_ENABLED=0 go build -o ./_build/envoy-manager github.com/kubermatic/kubermatic/api/cmd/nodeport-proxy/envoy-manager

build: envoy-manager lb-updater

clean:
	rm -f envoy-manager lb-updater

docker: build
	docker build -t quay.io/kubermatic/nodeport-proxy:$(TAG) .

test:
	go test ./...

.PHONY: test envoy-manager lb-updater

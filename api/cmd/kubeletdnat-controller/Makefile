build:
	CGO_ENABLED=0 go build -o ./_build/kubeletdnat-controller .

docker: build
	docker build -t quay.io/kubermatic/kubeletdnat-controller:$(TAG) .

.PHONY: build docker

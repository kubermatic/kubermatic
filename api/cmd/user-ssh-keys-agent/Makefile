build:
	CGO_ENABLED=0 go build -o user-ssh-keys-agent

docker: build
	docker build -t quay.io/kubermatic/user-ssh-keys-agent:$(TAG) .

.PHONY: build docker

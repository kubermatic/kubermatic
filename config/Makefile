SHELL=/bin/bash
REPO=kubermatic/installer
TAGS=dev
BUILD_FLAG += $(foreach tag, $(TAGS), -t $(REPO):$(tag))

default: all

all: docker-build

clean:
	rm -f $(CMD)

docker-build:
	docker build $(BUILD_FLAG) .

docker-push:
	for TAG in $(TAGS) ; do \
		docker push $(REPO):$$TAG ; \
	done

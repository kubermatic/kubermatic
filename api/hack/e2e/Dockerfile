ARG GO_VERSION=1.9.2
FROM golang:${GO_VERSION}

# install e2e dependencies
RUN apt-get update && apt-get install -y rsync && rm -rf /var/lib/apt/lists/*

# Git Repository Configuration
ARG E2E_REPO=https://github.com/kubernetes/kubernetes
ARG E2E_REF=v1.8.4

# clone Kubernetes repository
RUN mkdir -p ${GOPATH}/src/k8s.io && \
      git clone --branch ${E2E_REF} --depth 1 --single-branch ${E2E_REPO} ${GOPATH}/src/k8s.io/kubernetes
WORKDIR ${GOPATH}/src/k8s.io/kubernetes

# install build dependencies
RUN go get -u github.com/jteeuwen/go-bindata/go-bindata

# build all test dependencies
RUN GOLDFLAGS="--s -w" make all WHAT="cmd/kubectl vendor/github.com/onsi/ginkgo/ginkgo test/e2e/e2e.test"

# testing defaults
ENV KUBECONFIG=/workspace/kubermatickubeconfig
ENV KUBERNETES_CONFORMANCE_TEST=Y HOME=/go/src/k8s.io/kubernetes SKEW=false FOCUS=Conformance
CMD go run hack/e2e.go -- -v --test --check-version-skew=${SKEW} --test_args="--ginkgo.focus=\[${FOCUS}\] ${TEST_ARGS}"

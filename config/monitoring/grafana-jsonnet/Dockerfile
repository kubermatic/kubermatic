FROM golang:1.10-alpine

RUN apk add --update git

RUN go get -u github.com/brancz/gojsontoyaml && \
    go get -u github.com/fatih/color && \
    go get github.com/google/go-jsonnet && \
    cd $GOPATH/src/github.com/google/go-jsonnet/jsonnet && \
    go build -o $GOPATH/bin/jsonnet

RUN mkdir -p $GOPATH/src/github.com/grafana && \
    git clone https://github.com/grafana/grafonnet-lib.git $GOPATH/src/github.com/grafana/grafonnet-lib

RUN mkdir -p $GOPATH/src/github.com/ksonnet && \
    git clone https://github.com/ksonnet/ksonnet-lib.git $GOPATH/src/github.com/ksonnet/ksonnet-lib

RUN mkdir -p $GOPATH/src/github.com/brancz && \
    git clone https://github.com/brancz/kubernetes-grafana.git $GOPATH/src/github.com/brancz/kubernetes-grafana

RUN mkdir -p $GOPATH/src/github.com/kubermatic/kubermatic/config/monitoring/grafana-jsonnet

WORKDIR /go/src/github.com/kubermatic/kubermatic/config/monitoring/grafana-jsonnet

FROM golang:1.10 as builddeps

RUN go get -u gopkg.in/alecthomas/gometalinter.v2
RUN mv /go/bin/gometalinter.v2 /go/bin/gometalinter
RUN gometalinter --install
RUN curl -o /go/bin/dep -L https://github.com/golang/dep/releases/download/v0.5.0/dep-linux-amd64
RUN chmod +x /go/bin/dep
RUN go get -u github.com/frapposelli/wwhrd

FROM golang:1.10
COPY --from=builddeps /go/bin/* /usr/local/bin/

RUN curl -o /usr/bin/kubectl -L "https://storage.googleapis.com/kubernetes-release/release/v1.10.6/bin/linux/amd64/kubectl"
RUN chmod 0755 /usr/bin/kubectl

RUN apt update && apt -y install git

RUN mkdir -p /go/src/github.com/kubermatic/kubermatic/api

WORKDIR /go/src/github.com/kubermatic/kubermatic/api
ENTRYPOINT ["/bin/bash", "-exc"]

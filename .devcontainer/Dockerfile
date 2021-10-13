# https://hub.docker.com/r/qmcgaw/godevcontainer
FROM qmcgaw/godevcontainer

RUN apk add make curl bash

RUN export GO111MODULE=on
RUN export CGO_ENABLED=1

RUN go get -u github.com/go-delve/delve/cmd/dlv@master
RUN go get -u honnef.co/go/tools/cmd/staticcheck
RUN go get -u golang.org/x/lint/golint
RUN go get -u github.com/fzipp/gocyclo/cmd/gocyclo
RUN go get -u github.com/amitbet/gorename


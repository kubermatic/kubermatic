FROM golang:1.11.1

RUN apt-get update && apt-get install -y mercurial

RUN wget --quiet -O /usr/local/bin/dep https://github.com/golang/dep/releases/download/v0.5.0/dep-linux-amd64
RUN chmod +x /usr/local/bin/dep

FROM golang:1.11.1

RUN mkdir tmp
RUN cd tmp && curl -L --fail https://github.com/frapposelli/wwhrd/releases/download/v0.2.1/wwhrd_0.2.1_linux_amd64.tar.gz | tar -xvz
RUN mv tmp/wwhrd /usr/local/bin/wwhrd
RUN rm -rf tmp

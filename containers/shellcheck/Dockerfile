# Build-only image
FROM ubuntu:18.04 AS build
USER root

ARG version=v0.6.0

# Install OS deps
RUN apt-get update && apt-get install -y ghc cabal-install git

RUN git clone https://github.com/koalaman/shellcheck.git /opt/shellCheck
WORKDIR /opt/shellCheck
RUN git checkout $version

# Install Haskell deps
# (This is a separate copy/run so that source changes don't require rebuilding)
RUN cabal update && cabal install --dependencies-only --ghc-options="-optlo-Os -split-sections"

RUN cabal build Paths_ShellCheck && \
  ghc -optl-static -optl-pthread -isrc -idist/build/autogen --make shellcheck -split-sections -optc-Wl,--gc-sections -optlo-Os && \
  strip --strip-all shellcheck

RUN mkdir -p /out/bin && \
  cp shellcheck  /out/bin/

# Resulting Alpine image
FROM alpine:3.10
LABEL maintainer="Kamil Domański <kamil@loodse.com>"
COPY --from=build /out /

ENTRYPOINT ["/bin/shellcheck"]
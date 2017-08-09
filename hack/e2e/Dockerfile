# TODO(realfake): Strip this nonsense
FROM gcr.io/google-containers/kubekins-test:latest
MAINTAINER Luk Burchard <luk@loodse.com>

ADD "https://dl.google.com/dl/cloudsdk/channels/rapid/google-cloud-sdk.tar.gz" \
 "https://raw.githubusercontent.com/kubernetes/kubernetes/master/hack/e2e.go" \
 "/workspace/"

ENV KUBERNETES_CONFORMANCE_TEST=y \
    KUBERNETES_PROVIDER=skeleton \
    KUBECONFIG=/workspace/kubermatickubeconfig
ENV PATH=/google-cloud-sdk/bin:${PATH} \
    CLOUDSDK_CORE_DISABLE_PROMPTS=1
RUN tar xzf /workspace/google-cloud-sdk.tar.gz -C / && \
    /google-cloud-sdk/install.sh \
        --disable-installation-options \
        --bash-completion=false \
        --path-update=false \
        --usage-reporting=false && \
    gcloud components install beta && \
    gcloud components install alpha && \
    gcloud components update kubectl && \
    gcloud info | tee /workspace/gcloud-info.txt

RUN go run e2e.go -- -v --extract=release/stable-1.6
WORKDIR /workspace/kubernetes
CMD go run hack/e2e.go -- -v --test --test_args='--ginkgo.focus=\[Conformance\]'


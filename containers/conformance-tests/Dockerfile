FROM fedora:30

ENV GO_VERSION="1.12.12"
ENV VAULT_VERSION="1.1.3"
ENV HELM_VERSION="2.13.1"
ENV KUBECTL_VERSION="1.14.4"
ENV TERRAFORM_VERSION="0.12.13"
ENV KUBETEST_COMMIT="eb1a87c9083a31f9a9dda7822cb674f9f8d99439"

RUN dnf install -y curl bash unzip jq buildah which findutils make git gcc

ADD registries.conf /etc/containers/registries.conf
# metacopy is only available on Kernels 4.18+. Ubuntu 18.04 ships with 4.15
RUN sed -i "s/mountopt = \"nodev,metacopy=on\"/mountopt = \"nodev\"/g" /etc/containers/storage.conf

# Install vault, Helm, Terraform and kubectl
RUN cd /tmp && curl -fLO https://releases.hashicorp.com/vault/${VAULT_VERSION}/vault_${VAULT_VERSION}_linux_amd64.zip && \
    unzip vault_*.zip && \
    mv vault /usr/local/bin && \
    rm /tmp/vault_*.zip && \
    curl -fL https://storage.googleapis.com/kubernetes-helm/helm-v${HELM_VERSION}-linux-amd64.tar.gz | tar -xvz && \
    mv linux-amd64/helm /usr/local/bin && \
    curl -fLO https://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl && \
    chmod +x kubectl && \
    mv kubectl /usr/local/bin && \
    cd /tmp && curl -fLO https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip && \
    unzip terraform_*.zip && \
    mv terraform /usr/local/bin && \
    rm /tmp/terraform_*.zip

# Install Golang
ENV PATH=$PATH:/usr/local/go/bin
ENV GOPATH /go
RUN curl -fL -o /tmp/golang.tag.gz https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf /tmp/golang.tag.gz && \
    rm /tmp/golang.tag.gz && \
    git clone https://github.com/kubernetes/test-infra.git $GOPATH/src/k8s.io/test-infra && \
    cd $GOPATH/src/k8s.io/test-infra && \
    git checkout 9601c165917cab9b850f24453365f1177aa934f0 && \
    CGO_ENABLED=0 GO111MODULE=on GOPROXY=https://goproxy.io go build -v -o /usr/local/bin/boskosctl ./boskos/cmd/cli

# Install the kubetest binary
RUN GO111MODULE=on GOPROXY="https://proxy.golang.org" go get -v k8s.io/test-infra/kubetest@${KUBETEST_COMMIT} && \
    mv $(go env GOPATH)/bin/kubetest /usr/local/bin

# Install the kubernetes conformance test binaries
COPY ./install-kube-tests.sh /opt/install-kube-tests.sh
RUN /opt/install-kube-tests.sh
# This is linux only, thus we can remove the windows and darwin files
RUN rm -rf /opt/kube-test/*/platforms/darwin && rm -rf /opt/kube-test/*/platforms/windows

# Install the machine-controller pubkey
RUN mkdir $HOME/.ssh && \
    echo 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCo3amVmCkIZo4cgj2kjU2arZKlzzOhaOuveH9aJbL4mlVHVsEcVk+RSty4AMK1GQL3+Ii7iGicKWwge4yefc75aOtUncfF01rnBsNvi3lOqJR/6POHy4OnPXJElvEn7jii/pAUeyr8halBezQTUkvRiUtlJo6oEb2dRN5ujyFm5TuIxgM0UFVGBRoD0agGr87GaQsUahf+PE1zHEid+qQPz7EdMo8/eRNtgikhBG1/ae6xRstAi0QU8EgjKvK1ROXOYTlpTBFElApOXZacH91WvG0xgPnyxIXoKtiCCNGeu/0EqDAgiXfmD2HK/WAXwJNwcmRvBaedQUS4H0lNmvj5' > $HOME/.ssh/id_rsa.pub && \
    chmod 0700 $HOME/.ssh

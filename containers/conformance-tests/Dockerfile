FROM fedora:30

ENV GO_VERSION="1.12.6"
ENV VAULT_VERSION="1.1.3"
ENV HELM_VERSION="2.13.1"
ENV KUBECTL_VERSION="1.14.4"

RUN dnf install -y curl bash unzip jq buildah which findutils make git

ADD registries.conf /etc/containers/registries.conf

# Install vault
RUN cd /tmp && curl -LO https://releases.hashicorp.com/vault/${VAULT_VERSION}/vault_${VAULT_VERSION}_linux_amd64.zip && \
    unzip vault_*.zip && \
    mv vault /usr/local/bin && \
    rm /tmp/vault_*.zip

# Install Helm
RUN curl -L https://storage.googleapis.com/kubernetes-helm/helm-v${HELM_VERSION}-linux-amd64.tar.gz | tar -xvz && \
    mv linux-amd64/helm /usr/local/bin

# Install kubectl
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl && \
    chmod +x kubectl && \
    mv kubectl /usr/local/bin

# Install Golang
ENV PATH=$PATH:/usr/local/go/bin
ENV GOPATH /go
RUN curl -L -o /tmp/golang.tag.gz https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf /tmp/golang.tag.gz && \
    rm /tmp/golang.tag.gz

# Install the kubernetes conformance test binaries
COPY ./install-kube-tests.sh /opt/install-kube-tests.sh
RUN /opt/install-kube-tests.sh
# This is linux only, thus we can remove the windows and darwin files
RUN rm -rf /opt/kube-test/*/platforms/darwin && rm -rf /opt/kube-test/*/platforms/windows

# Install the machine-controller pubkey
RUN mkdir $HOME/.ssh && \
    echo 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCo3amVmCkIZo4cgj2kjU2arZKlzzOhaOuveH9aJbL4mlVHVsEcVk+RSty4AMK1GQL3+Ii7iGicKWwge4yefc75aOtUncfF01rnBsNvi3lOqJR/6POHy4OnPXJElvEn7jii/pAUeyr8halBezQTUkvRiUtlJo6oEb2dRN5ujyFm5TuIxgM0UFVGBRoD0agGr87GaQsUahf+PE1zHEid+qQPz7EdMo8/eRNtgikhBG1/ae6xRstAi0QU8EgjKvK1ROXOYTlpTBFElApOXZacH91WvG0xgPnyxIXoKtiCCNGeu/0EqDAgiXfmD2HK/WAXwJNwcmRvBaedQUS4H0lNmvj5' > $HOME/.ssh/id_rsa.pub && \
    chmod 0700 $HOME/.ssh

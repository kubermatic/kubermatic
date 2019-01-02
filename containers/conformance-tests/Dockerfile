FROM alpine:latest
RUN apk --no-cache add ca-certificates curl bash
COPY ./install-kube-tests.sh /opt/install-kube-tests.sh
RUN /opt/install-kube-tests.sh

# This is linux only, thus we can remove the windows and darwin files
RUN rm -rf /opt/kube-test/*/platforms/darwin
RUN rm -rf /opt/kube-test/*/platforms/windows

FROM golang:1.11.4-stretch
COPY --from=0 /opt/kube-test /opt/kube-test
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN apt update && apt install -y git-crypt curl apt-transport-https software-properties-common unzip jq && \
  curl -fsSL https://download.docker.com/linux/debian/gpg | apt-key add - && \
  curl -fsSL https://download.docker.com/linux/$(. /etc/os-release; echo "$ID")/gpg \
    | apt-key add - && \
  add-apt-repository \
   "deb [arch=amd64] https://download.docker.com/linux/$(. /etc/os-release; echo "$ID") \
   $(lsb_release -cs) stable" && \
  apt-get update && \
  apt-get install -y --no-install-recommends docker-ce=18.06.0* && \
  sed -i 's/cgroupfs_mount$/#cgroupfs_mount\n/' /etc/init.d/docker && \
  echo 'DOCKER_OPTS="${DOCKER_OPTS} --data-root=/docker-graph"' | \
    tee --append /etc/default/docker && \
  cd /tmp && curl -LO https://releases.hashicorp.com/vault/1.0.1/vault_1.0.1_linux_amd64.zip && unzip vault_*.zip && mv vault /usr/local/bin && \
  curl -L https://storage.googleapis.com/kubernetes-helm/helm-v2.12.1-linux-amd64.tar.gz|tar -xvz && mv linux-amd64/helm /usr/local/bin && \
  curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl && \
    chmod +x kubectl && \
    mv kubectl /usr/local/bin

RUN mkdir $HOME/.ssh && \
    echo 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCo3amVmCkIZo4cgj2kjU2arZKlzzOhaOuveH9aJbL4mlVHVsEcVk+RSty4AMK1GQL3+Ii7iGicKWwge4yefc75aOtUncfF01rnBsNvi3lOqJR/6POHy4OnPXJElvEn7jii/pAUeyr8halBezQTUkvRiUtlJo6oEb2dRN5ujyFm5TuIxgM0UFVGBRoD0agGr87GaQsUahf+PE1zHEid+qQPz7EdMo8/eRNtgikhBG1/ae6xRstAi0QU8EgjKvK1ROXOYTlpTBFElApOXZacH91WvG0xgPnyxIXoKtiCCNGeu/0EqDAgiXfmD2HK/WAXwJNwcmRvBaedQUS4H0lNmvj5' > $HOME/.ssh/id_rsa.pub && \
    chmod 0700 $HOME/.ssh

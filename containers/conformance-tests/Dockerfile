FROM alpine:latest
RUN apk --no-cache add ca-certificates curl bash
COPY ./install-kube-tests.sh /opt/install-kube-tests.sh
RUN /opt/install-kube-tests.sh

# This is linux only, thus we can remove the windows and darwin files
RUN rm -rf /opt/kube-test/*/platforms/darwin
RUN rm -rf /opt/kube-test/*/platforms/windows

FROM golang:1.11.2-stretch
COPY --from=0 /opt/kube-test /opt/kube-test
RUN apt update && apt install -y git-crypt curl

RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl && \
    chmod +x kubectl && \
    mv kubectl /usr/local/bin

RUN mkdir $HOME/.ssh && \
    echo 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCo3amVmCkIZo4cgj2kjU2arZKlzzOhaOuveH9aJbL4mlVHVsEcVk+RSty4AMK1GQL3+Ii7iGicKWwge4yefc75aOtUncfF01rnBsNvi3lOqJR/6POHy4OnPXJElvEn7jii/pAUeyr8halBezQTUkvRiUtlJo6oEb2dRN5ujyFm5TuIxgM0UFVGBRoD0agGr87GaQsUahf+PE1zHEid+qQPz7EdMo8/eRNtgikhBG1/ae6xRstAi0QU8EgjKvK1ROXOYTlpTBFElApOXZacH91WvG0xgPnyxIXoKtiCCNGeu/0EqDAgiXfmD2HK/WAXwJNwcmRvBaedQUS4H0lNmvj5' > $HOME/.ssh/id_rsa.pub && \
    chmod 0700 $HOME/.ssh

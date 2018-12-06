FROM golang:1.11.2-stretch

LABEL maintainer "henrik@loodse.com"

RUN apt-get update && apt-get install -y curl

COPY ./install.sh /opt/install.sh

RUN chmod +x /opt/install.sh
RUN /opt/install.sh
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl && \
  chmod +x kubectl && \
  mv kubectl /usr/local/bin
RUN mkdir $HOME/.ssh && \
  echo 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCo3amVmCkIZo4cgj2kjU2arZKlzzOhaOuveH9aJbL4mlVHVsEcVk+RSty4AMK1GQL3+Ii7iGicKWwge4yefc75aOtUncfF01rnBsNvi3lOqJR/6POHy4OnPXJElvEn7jii/pAUeyr8halBezQTUkvRiUtlJo6oEb2dRN5ujyFm5TuIxgM0UFVGBRoD0agGr87GaQsUahf+PE1zHEid+qQPz7EdMo8/eRNtgikhBG1/ae6xRstAi0QU8EgjKvK1ROXOYTlpTBFElApOXZacH91WvG0xgPnyxIXoKtiCCNGeu/0EqDAgiXfmD2HK/WAXwJNwcmRvBaedQUS4H0lNmvj5' > $HOME/.ssh/id_rsa.pub && \
  chmod 0700 $HOME/.ssh
RUN apt update && apt install -y git-crypt

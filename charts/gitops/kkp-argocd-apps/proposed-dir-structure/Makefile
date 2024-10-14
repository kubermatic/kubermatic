KKP_VERSION=v2.24.1
# To upgrade KKP, update the version of kkp here.
#KKP_VERSION=v2.25.6
INSTALL_DIR=./kubermatic/releases/${KKP_VERSION}

#use e.g. for MAC OS: BIN_ARCH=darwin-amd64 make download-kkp-release
BIN_ARCH ?= linux-amd64
download-kkp-release:
	mkdir -p ${INSTALL_DIR}
	wget https://github.com/kubermatic/kubermatic/releases/download/${KKP_VERSION}/kubermatic-ee-${KKP_VERSION}-${BIN_ARCH}.tar.gz -O- | tar -xz --directory ./kubermatic/releases/${KKP_VERSION}/

install-kkp-dev:
	${INSTALL_DIR}/kubermatic-installer deploy \
	  --charts-directory ${INSTALL_DIR}/charts --config ./dev/demo-master/k8cConfig.yaml --helm-values ./dev/demo-master/values.yaml --storageclass aws \
	  --skip-charts='cert-manager,nginx-ingress-controller,dex'

install-kkp-dev-user-mla:
	${INSTALL_DIR}/kubermatic-installer deploy usercluster-mla \
	  --charts-directory ${INSTALL_DIR}/charts --config ./dev/demo-master/k8cConfig.yaml --helm-values ./dev/demo-master/values-usermla.yaml

create-long-lived-master-seed-kubeconfig:
	${INSTALL_DIR}/kubermatic-installer convert-kubeconfig ./kubeone-install/dev-master/argodemo-dev-master-kubeconfig | base64 -w0 > ./seed-ready-kube-config

# Setup Seed1 - DEV Master
deploy-argo-dev-master:
	helm upgrade --install argocd --version 5.36.10 --namespace argocd --create-namespace argo/argo-cd -f values-argocd.yaml --set 'server.ingress.hosts[0]=argocd.argodemo.lab.kubermatic.io' --set 'server.ingress.tls[0].hosts[0]=argocd.argodemo.lab.kubermatic.io'

deploy-argo-apps-dev-master:
	helm template argo-apps --set kkpVersion=${KKP_VERSION} -f ./dev/demo-master/argoapps-values.yaml ${INSTALL_DIR}/charts/gitops/kkp-argocd-apps | kubectl apply -f -

push-git-tag-dev:
	git tag -f dev-kkp-${KKP_VERSION}
	git push origin -f dev-kkp-${KKP_VERSION}

# Setup Seed2 - DEV India Seed
deploy-argo-dev-seed:
	helm upgrade --install argocd --version 5.36.10 --namespace argocd --create-namespace argo/argo-cd -f values-argocd.yaml --set 'server.ingress.hosts[0]=argocd.india.argodemo.lab.kubermatic.io' --set 'server.ingress.tls[0].hosts[0]=argocd.india.argodemo.lab.kubermatic.io'

deploy-argo-apps-dev-seed:
	helm template argo-apps --set kkpVersion=${KKP_VERSION} -f ./dev/india-seed/argoapps-values.yaml ${INSTALL_DIR}/charts/gitops/kkp-argocd-apps | kubectl apply -f -

create-long-lived-seed-kubeconfig:
	${INSTALL_DIR}/kubermatic-installer convert-kubeconfig ./kubeone-install/dev-seed/argodemo-dev-seed-kubeconfig | base64 -w0 > ./seed-ready-kube-config

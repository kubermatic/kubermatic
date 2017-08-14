#!/bin/bash

echo "This will deploy Kubermatic"


if [ -e kubermatic-config.yaml ]
then
  echo "Kubermatic Config exist"
else
  read -p  "Please provide the URL fo the seed cluster, followed by [ENTER]:" URL
  KUBECONFIG=$(cat kubeconfig | base64)
  DATACENTERS=$(cat datacenters.yaml | base64)
  cp kubermatic-config.yaml.templ kubermatic-config.yaml

  sed -i.bak "s/url/$URL/g" kubermatic-config.yaml
  sed -i.bak "s/kubeconfig/$KUBECONFIG/g" kubermatic-config.yaml
  sed -i.bak "s/datacenters/$DATACENTERS/g" kubermatic-config.yaml

  read -p "Is this a bare-metal setup (y/n) ? "
  if [[ $REPLY =~ ^[Yy]$ ]]
  then
    sed -i.bak "s/#IsBareMetal/IsBareMetal/g" kubermatic-config.yaml
    sed -i.bak "s/#BareMetalProviderURL/BareMetalProviderURL/g" kubermatic-config.yaml

    read -p  "Please provide the URL Bare Metal Provider, followed by [ENTER]:" URL
    sed -i.bak "s/baremetalurl/$URL/g" kubermatic-config.yaml

    read -p  "Please provide the Bare Metal StorageProvider, followed by [ENTER]:" STORAGE
    sed -i.bak "s/storageprovider/bare-metal/g" kubermatic-config.yaml

    read -p  "Please provide the Bare Metal StorageURL, followed by [ENTER]:" STORAGEURL
    sed -i.bak "s/#StorageURL/StorageURL/g" kubermatic-config.yaml
    sed -i.bak "s/baremetalstorageurl/$STORAGEURL/g" kubermatic-config.yaml

  else
    read -p "Deploy on AWS(1) or GKE (2) ? "
    if [[ $REPLY =~ ^[1]$ ]]
    then
      sed -i.bak "s/storageprovider/aws/g" kubermatic-config.yaml
    elif [[ $REPLY =~ ^[2]$ ]]
    then
      sed -i.bak "s/storageprovider/gke/g" kubermatic-config.yaml
    fi
    read -p  "Please provide the Storage zone, followed by [ENTER]:" STORAGEZONE
    sed -i.bak "s/#StorageZone/StorageZone/g" kubermatic-config.yaml
    sed -i.bak "s/storageprovider/aws/g" kubermatic-config.yaml
    sed -i.bak "s/storagezone/$STORAGEZONE/g" kubermatic-config.yaml
  fi

  read -p "Deploy monitoring (y/n) ? "
  if [[ $REPLY =~ ^[Yy]$ ]]
  then
    sed -i.bak "s/#Prometheus/Prometheus/g" kubermatic-config.yaml
    echo "Create a password for Prometheus:"
    htpasswd -c auth admin
    PASSWORD=$(cat auth | base64)
    sed -i.bak "s/#PrometheusAuth/PrometheusAuth/g" kubermatic-config.yaml
    sed -i.bak "s/prometheusauth/$PASSWORD/g" kubermatic-config.yaml
  fi
  read -p "Deploy logging (y/n) ? "
  if [[ $REPLY =~ ^[Yy]$ ]]
  then
    sed -i.bak "s/#Logging/Logging/g" kubermatic-config.yaml
  fi

fi

read -p "Which version of Kubermatic do you want to install (e.g. 1.4) ? " VERSION
cp kubermatic-job.yaml.templ kubermatic-job.yaml
sed -i.bak "s/{tag}/$VERSION/g" kubermatic-job.yaml
echo "deploy Kubermatic"
kubectl --kubeconfig=kubeconfig apply -f kubermatic-namespace.yaml
kubectl --kubeconfig=kubeconfig apply -f dockercfg-secret.yaml
kubectl --kubeconfig=kubeconfig apply -f kubermatic-config.yaml
kubectl --kubeconfig=kubeconfig apply -f kubermatic-job.yaml

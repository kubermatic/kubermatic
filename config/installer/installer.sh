#!/usr/bin/env bash
die() {
  {
    echo >&2 "$@"
    rm -f kubermatic-config.yaml
    exit 1
  }
}

test_input() {
  test -z "$1" && die "$2"
}

if [[ ! -f kubermatic-config.yaml ]]; then
  read -p "Please provide future Kubermatic URL ( example: kubermatic.example.com ): " URL
  test_input "$URL" "Please provide future Kubermatic URL"
  KUBECONFIG="$(base64 -w 0 kubeconfig)"
  DATACENTERS="$(base64 -w 0 datacenters.yaml)"
  if ! cp kubermatic-config.yaml.templ kubermatic-config.yaml; then
    die "kubermatic-config.yaml.templ is missing"
  fi

  if [[ ! -z $KUBECONFIG ]] && [[ ! -z $DATACENTERS ]]; then
    sed -i.bak "s~<url>~$URL~g" kubermatic-config.yaml
    sed -i.bak "s/<kubeconfig>/$KUBECONFIG/" kubermatic-config.yaml
    sed -i.bak "s/<datacenters>/$DATACENTERS/" kubermatic-config.yaml
  else
    test -f kubeconfig || die "kubeconfig file is missing"
    test -f datacenters.yaml || die "datacenters.yaml file is missing"
  fi
  unset URL

  read -p "Is this a bare-metal setup (y/n) ? " yn
  case $yn in
    [Yy])
      sed -i.bak "s/#IsBareMetal/IsBareMetal/" kubermatic-config.yaml
      sed -i.bak "s/#BareMetalProviderURL/BareMetalProviderURL/" kubermatic-config.yaml

      read -p  "Please provide Bare Metal Provider URL: " URL
      test_input "$URL" "Please provide Bare Metal Provider URL"
      sed -i.bak "s~<baremetalurl>~$URL~g" kubermatic-config.yaml

      read -p  "Please provide Bare Metal StorageProvider: " STORAGE
      test_input "$STORAGE" "Please provide Bare Metal StorageProvider"
      sed -i.bak "s/<storageprovider>/bare-metal/" kubermatic-config.yaml

      read -p  "Please provide Bare Metal Storage URL: " STORAGEURL
      test_input "$STORAGEURL" "Please provide Bare Metal StorageURL"
      sed -i.bak "s/#StorageURL/StorageURL/" kubermatic-config.yaml
      sed -i.bak "s~<baremetalstorageurl>~$STORAGEURL~" kubermatic-config.yaml
      ;;
    [Nn])
      read -p "Deploy on AWS (aws), GKE (gke) or OpenStack (openstack) ? " provider
      case $provider in
        aws)
          sed -i.bak "s/<storageprovider>/aws/" kubermatic-config.yaml
          ;;
        gke)
          sed -i.bak "s/<storageprovider>/gke/" kubermatic-config.yaml
          ;;
        openstack)
          sed -i.bak "s/<storageprovider>/openstack-cinder/" kubermatic-config.yaml
          ;;
        *)
          die "Please select aws, gke or openstack"
          ;;
      esac

      read -p  "Please provide Storage zone ( example: bare-metal-provider.kubermatic.example.com ): " STORAGEZONE
      test_input "$STORAGEZONE" "Please provide Storage zone"
      sed -i.bak "s/#StorageZone/StorageZone/" kubermatic-config.yaml
      sed -i.bak "s/<storagezone>/$STORAGEZONE/" kubermatic-config.yaml
      ;;
    [Nn])
      ;;
    *)
      die "Please answer y or n"
  esac

  read -p "Deploy monitoring (y/n) ? " yn
  case $yn in
    [Yy])
      sed -i.bak "s/#Prometheus/Prometheus/g" kubermatic-config.yaml
      command -v htpasswd >/dev/null 2>&1 || die "Please install htpasswd"
      echo "Create password for Prometheus:"
      htpasswd -c auth admin
      PASSWORD="$(base64 -w 0 auth)"
      sed -i.bak "s/#PrometheusAuth/PrometheusAuth/" kubermatic-config.yaml
      sed -i.bak "s~<prometheusauth~$PASSWORD~" kubermatic-config.yaml
      ;;
    [Nn])
      ;;
    *)
      die "Please answer y or n"
  esac
  read -p "Deploy logging (y/n) ? " yn
  case $yn in
    [Yy])
      sed -i.bak "s/#Logging/Logging/g" kubermatic-config.yaml
      ;;
    [Nn])
      ;;
    *)
      die "Please answer y or n"
  esac
else
  echo "Kubermatic Config exist"
fi

read -p "Which version of Kubermatic do you want to install (e.g. 1.4) ? " VERSION
test_input "$VERSION" "Please select Kubermatic version"
if ! cp kubermatic-job.yaml.templ kubermatic-job.yaml; then
  die "kubermatic-job.yaml.templ is missing"
fi
sed -i.bak "s/{tag}/$VERSION/g" kubermatic-job.yaml
echo "deploy Kubermatic"
kubectl --kubeconfig=kubeconfig apply -f kubermatic-namespace.yaml
kubectl --kubeconfig=kubeconfig apply -f dockercfg-secret.yaml
kubectl --kubeconfig=kubeconfig apply -f kubermatic-config.yaml
kubectl --kubeconfig=kubeconfig apply -f kubermatic-job.yaml

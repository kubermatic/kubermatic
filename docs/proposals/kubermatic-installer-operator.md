# Kubermatic Installer Operator: Enhanced Kubermatic Installer 

**Author**: Mohamed Rafraf (@mohamed-rafraf)
**Status**: Draft proposal

This proposal introduces a Kubernetes controller and a Custom Resource Definition (CRD) designed to simplify the installation,
upgrade, and management of Kubermatic systems, replacing the traditional CLI-based approach.


## Motivation

The existing CLI-based installation and upgrade process for Kubermatic, while effective, 
requires manual intervention and extensive Kubernetes and Helm knowledge. By transitioning to a controller-based model, 
we aim to automate these processes and make them more accessible to users with varying 
levels of expertise. Key motivations include:

* **Automation of Upgrades and Installation:** Automate the entire lifecycle management, including initial installation, 
upgrades, changes, and potentially scaling.
* **Reduction of Errors and Early Validation:** Minimize the chance of errors from manual steps, enhancing system reliability 
by integrating validation directly into the CRD to catch configuration errors before applying changes.
* **Ready for Future Enhancements:** A controller-based installer makes it easier to introduce new features and integrations. The installer can be quickly updated or expanded to levarga new features without breaking underlying mechanism.

## Proposal

A new Kubermatic Installer Operator that utilizes a CRD called `KubermaticInstallation` 
(Since `KubermaticConfiguration` exist).The operator will handle the deployment and management 
of the Kubermatic stack by watching for changes to instances of `KubermaticInstallation` CRD.

* **KubermaticInstalltion CRD:** Define all necessary configurations needed for the Kubermatic installation 
in a single Kubernetes resource
* **Kubermatic Installer Operator:**  A controller that reacts to changes to KubermaticInstallation resources 
by deploying or updating the Kubermatic stack according to the specified configuration.
* **Dependencies Management:** The operator will manage dependencies like cert-manager, nginx-ingress-controller, 
and Dex, ensuring they are installed or upgraded as necessary before deploying Kubermatic components.


### User Story

An administrator wishes to install or upgrade their Kubermatic system::

1. The admin prepares a `KubermaticInstallation` manifest. The goal here is to avoid splitting the configuration and customization to be split into different places and manifests like Helm `values.yaml` file, `KubermaticConfiguration` etc
1. The admin create or apply the manifest and the operator will detect the new/changed configuration 
and performs the necessary actions to bring the system to the desired state, handling dependencies and ordering automatically.
1. admin can follow the installation details by watching the logs of the operator.
1. The administrator monitors the status of the installation or upgrade through Kubernetes standard tools like kubectl describe.

Example of `KubermaticInstallation`. Each stack could be defined in different manifests and different objects.

```yaml
apiVersion: kubermatic.k8c.io/v1
kind: KubermaticInstallation
metadata:
  name: kubermatic
  namespace: kubermatic

spec:
  master:
    domain: demo.kubermatic.io
    dex:
      clients:
        - id: kubermatic
          name: kubermatic
          #######################
          ### SKIP DEX CONFIG ###
          #######################
    controller:
      replicas: 2
    seed:
      replicas: 2
    cert-manager:
      clusterIssuers:
        letsencrypt-prod:
          email: mohamed.rafraf@kubermatic.com
      enabled: true
    nginx: 
      enabled: true
      controller:
        replicaCount: 2 

  seed:
    - name: seed-1 
      etcdBackupRestore:
        defaultDestination: minio-ext-endpoint
          #######################
          ### SKIP SEED CONFIG ###
          #######################

  monitoring:
    grafana:
      enabled: true
    prometheus:
      enabled: true 
    nodeExprter:
      enabled: true

  mla:
    loki-distributed:
      enabled: true
      ingester:
        replicas: 3

    cortex:
      enabled: true
      server:
        replicas: 2

  backup:
    velero:
      enabled: true
        ##########################
        ### SKIP VELERO CONFIG ###
        ##########################
    minio: 
      enabled: true
      storeSize: 100Gi
```

### Goals

* Develop the `KubermaticInstallation` CRD to capture all required installer settings.
* Implement the Kubermatic Installer Operator to manage lifecycle events based on CRD changes.
* Integrate thorough validation within the operator to ensure configuration validity before application.
* Package and distribute the operator for easy deployment.
* Provide clear, helpful log output and error messages.

### Non-Goals

* Handling downgrades which can be significantly complex and risky.


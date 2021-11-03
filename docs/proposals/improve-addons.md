# Improve Addons

**Author**: @vgramer @SimonTheLeg

**Status**: Proposal

## Table of Contents

- [Improve Addons](#improve-addons)
  - [Table of Contents](#table-of-contents)
  - [Introduction](#introduction)
  - [Motivation and Background](#motivation-and-background)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Implementation](#implementation)
  - [Architecture](#architecture)
  - [Components](#components)
    - [AddonCatalogueDistributor](#addoncataloguedistributor)
    - [AddonCatalogue CR](#addoncatalogue-cr)
    - [AddonController](#addoncontroller)
    - [AddonInstallation CR](#addoninstallation-cr)
    - [CoreAddonController](#coreaddoncontroller)
    - [AddonInstallationController](#addoninstallationcontroller)
  - [Common User Flows](#common-user-flows)
  - [Alternatives considered](#alternatives-considered)
- [Glossary](#glossary)
  - [CoreAddon (old name: Default **Addon**)](#coreaddon-old-name-default-addon)
  - [OptionalAddon (old name: Accessible Addon)](#optionaladdon-old-name-accessible-addon)
  - [CustomerAddon (old name: Custom Addon)](#customeraddon-old-name-custom-addon)
- [Task & effort](#task--effort)

## Introduction

This proposal introduces improvements to the current [Addon system](https://docs.kubermatic.com/kubermatic/v2.18/guides/addons/). It aims to make current addons more flexible and user-friendly. Additionally, it simplifies the usage of addons we ship with (almost) every cluster, so-called "Default Addons".

## Motivation and Background

Currently, KKP only supports one mechanism for addons: For this, all addons are stored in a single docker image that is configured in KKP. Afterwards, an initContainer is being run that starts the image and copies all addons onto the seed-controller-managers local file-system. From there, addon manifests are being rendered and rolled out to user clusters.

The current implementation of Addons has some flaws that make the usage and maintenance of Addons cumbersome for cluster administrators:

- To add “Custom Addons” the cluster-admin needs to bundle manifests and templates for these Addons in a Docker image that is based on the official addon-manifest docker image by Kubermatic.
- Some addons are directly shipped with Kubermatic Code, while others are hosted separately (see #5499)
- It’s currently only possible to deploy multiple instances of the same Addon when using a workaround
- Addons are strictly tied to KKP versions due to the composition logic of the manifest image tag; for any provided manifest image the tag gets overwritten with `:<KKP VERSION>\[-CUSTOM SUFFIX\]`
- Mandatory core components of user clusters are named “Default Addons” which implies that these can be deleted by Cluster Admins
- Complex Addons can leave a trail of reconciliation errors until successfully deployed because all manifest files are applied at the same time
- Creating a cluster is a two-step process; firstly you have to create a cluster and only  afterward you can select custom Addons

*note: PS team report that some clients are using weekly KKP update in dev environments, consequently they need to rebuild the addon image every week which is painful and error-prone.*

## Goals

- Decouple custom addons from the KKP lifecycle
- Handle dependency **inside** an Addon (e.g. CRD need to be created before CRs, namespace needs to be created before other objects)
- Addons are correctly reconciled on changes. [kubermatic/kubermatic#6035](https://github.com/kubermatic/kubermatic/issues/6035)
- Be able to choose which addons should be installed during the cluster creation or in the cluster template
- Be compatible with private registries in preparation for future air-gapped environment stories. The current Policy regarding the air-gapped environment is to assume a private trusted registry is available. Addon Source (manifest of the workload to deployed) and workload images of the addon should be  configurable to point on a private registry

## Non-Goals

- The intent is not to invent a whole new package manager. Therefore dependency management outside an Addon is currently out of scope. For core components and Addons shipped with KKP, the platform manages the dependencies. For manifest-based Custom Addons, the user needs to take care of dependencies and for Helm-based Custom Addons Helm is taking care of it
- Handle installation of  "Addons" cross user-cluster (e.g. service-mesh or queuing system). These are considered as "Extensions" and should be handled by [KubeCarrier](https://docs.kubermatic.com/kubecarrier)
- Provide a private registry to store Addons
- Notifying users if a new version of an addon is available or showing in the UI that a new version of an installed addon is available. While we think this is quite valuable for end-users, we have decided to leave this out for know in order to keep proposal scope manageable. The good news is that the core principles of this proposal will make it possible to implement this feature later on. It is definitely a feature that should be on roadmap for addons after the implementation of this proposal is done
- Adding kustomize as a rendering method. We did have a longer discussion to decide which rendering methods should be included in the first version. In order to keep the size of this proposal manageable, we have decided to postpone adding kustomize as a rendering method. It is definitely a feature that should be on roadmap for addons after the implementation of this proposal is done and should fit right in the current architecture
- Generic Pull from object storage. It is definitely a feature that should be on roadmap for addons after the implementation of this proposal is done and should fit right in the current architecture. We have made this decision after consulting with PS. It was decided to go implement Git as an external source first, as this is going to reach the majority of customers
- Reworking of the [dashboard into an addon](https://github.com/kubermatic/dashboard/issues/3666). In order to keep the size of this proposal manageable, we have decided to postpone this. It is definitely a feature that should be on roadmap for addons after the implementation of this proposal is done and should fit right in the current architecture
- Handling Logos. The previous version for addons allows to base64 encode logos into the CR for display in the UI. Currently there are some size concerns raised by @kubermatic/sig-api and @kubermatic/sig-ui. In order to move forward, we have decided to exclude logos for now, until a KKP wide solution has been found. However it should be no problem with the current architecture to add them later on. Either as base64 encoded field or a reference to an external source

# Implementation

## Architecture

See below for a high level architecture view. Each component will be explained in detail in the subsequent chapters of this proposal.

![Architecture](./images/improveaddons-architecture.png)

([source](https://app.diagrams.net/#G1drryGHt2MbCDY6wRz99u7lMSBYXgI2lH))

## Components

### AddonCatalogueDistributor

The AddonCatalogueDistributor resides in the master cluster. Its main job is to merge different AddonCatalogues and push an aggregated AddonCatalogue into different seed clusters. We have decided to create a separate component for this, so there is a single source of truth on available addons, which can be synced across multiple seed clusters.

In order to work properly, the AddonCatalogueDistributor must (dynamically) be able to discover all available SeedClusters, so it knows where to push the catalogue to.

In the case of the MasterCluster and SeedCluster being the same, we propose to still run the AddonCatalogueDistributor exactly the same. Concretely this means it would have one target seed cluster (itself) and create a merged catalogue for it. This would have the advantage, that in case customers decide to add new seed clusters, the new seed clusters would work right out-of-the-box. Additionally, this helps keeping the code generic, which is in line with KKPs development pattern.

In the final implementation, we propose to have at least two catalogues per KKP installation. We think this makes sense to protect core addons, which are essential for all clusters, to be protected from third-party influence:

- CoreAddonCatalogue → will be maintained by Kubermatic.  We think it makes sense to add this as a layer of protection against misconfiguration. This catalogue is rarely changed. Moreover, this catalogue will not be shown in the UI. This catalogue contains addons for the supported Kubernetes versions (eg KubeProxy will have 3 versions of the addons, one per K8s version. OpenVPN will have one version of the addons compatible with the 3 K8s versions)
- CustomAddonCatalogue(s) →one or multiple catalogues that contain custom addons. This catalogue can be changed more often. Additionally, it is possible to have multiple CustomAddonCatalogues

Furthermore, the AddonCatalogueDistributor ensures that there are no duplicate addons inside the AddonCatalogue. A duplicate is defined by two addons having the same `name` and `version`. We propose to protect against accidental edits using a ValidationWebhook.
Lastly, we envisioned two possible sources for a catalogue:

- watching for changes of an AddonCatalogue Custom Resource directly inside KKP master
- watching a git repository → this idea was developed from PS feedback. It would enable customers to manage their addon catalogues in a GitOps way

  *tbd:  Alternatively to achieve GitOps, it would also be possible to have a CI/CD pipeline directly update the AddonCatalogue CR in the master cluster. We are currently unsure if this is better or worse*

In order to track available addons, the AddonCatalogueDistributor makes use of AddonCatalogue CRs

### AddonCatalogue CR

- contains a list of all addons
- each addon comes in multiple versions
- each version has
  - template
    - a rendering method. All methods will automatically inject [template data](https://docs.kubermatic.com/kubermatic/v2.18/guides/addons/#manifest-templating) into the values. Possible methods are (additional methods can easily be added in the future):
      - go-template → same as what we currently offer in addons
      - helm → rendering of helm charts
    - formSpec → Similar to the current formSpec. It is important to know that this value overwrites any other value store (e.g. helm's values.yaml):
      - displayName →Name in the UI
      - internalName →Templating reference. Allows for nested values using the dot-syntax. For example, if you want to overwrite a values.yaml file, you could do `internalName: spec.replicas`
      - type →Type of the field. We propose that references for now can only be primary types (number, string, ...), but do not allow for nested types (e.g. objects). This makes rendering them a lot more convenient
      - (optional) helpText →helpText to be displayed in the UI
  - a source from where to pull the data from. Possible sources are:
    - docker-image →same as we currently have. A Docker image contains manifests for one or multiple addons and is being pulled by the controller
    - git → a git repository from which to pull the manifests. This was considered to be especially useful by Kubermatic PS who work directly with customers
    - helm → a helm repository to pull charts from. Only compatible with rendering method helm
  - constraints. These describe conditions that must apply for an add-on to be compatible with a user cluster. Possible constraints are:
    - kkp-version →semVer range that describes compatible KKP versions
    - k8s-version →semVer range that describes compatible k8s versions
  - values → These are values that can be used for overwriting defaults. We think this field will be required for CoreAddons if a customer is using a private registry. So addons can be configured

Note: The current implementation of go-templating allows to declare a `RequiredResourceTypes` field, to handle dependencies. We propose to keep this as is for go-templating, but not have it for helm. The reason is that helm offers its own dependency management and we think that building an additional dependency mechanism on top will drastically increase complexity. As a result, this field will be moved into under go-template method and not on the general addon level.

An example CR could like this:

```yaml
apiVersion: kubermatic.k8c.io/v1
kind: AddonCatalogue
metadata:
 name: custom-catalogue
spec:
  addons:
    - name: "prometheus"
      description: "The Prometheus Node Exporter exposes a wide variety of hardware- and kernel-related metrics."
      versions:
        - version: "v1"
          template:
            method: go-template # plain manifest templated with go-template
            requiredResourceTypes:
              - group: "monitoring.coreos.com"
                kind: "prometheuses"
                version: "v1"
            formSpec:
              - displayName: Replicas
                internalName: spec.replicas
                required: true
                type: number
                helpText: "Number of replicas."
              - displayName: Description
                internalName: desc
                required: false
                type: text
            source:
              type:
                docker:
                  # docker related params...
            constraints:
              kkpVersion:
              k8sVersion:
        - version: "v2"
          template:
              method: helm
            source:
              type: git
                # git related params...
            constraints: # omitted for clarity
```

Note on CRDs in general: The AddonCatalogue strongly benefits from [OpenAPIv3's `oneOf` functionality](https://swagger.io/docs/specification/data-models/oneof-anyof-allof-not/), which is supported by Kubernetes. For example in the yaml above, you could say a source must be `oneOf` `docker` or `git`. The cool thing about `oneOf` is that `docker` and `git` can have totally have different fields from each other, but still would be statically validated. However, syntax like this is [not yet supported in kubebuilder](https://github.com/kubernetes-sigs/controller-tools/issues/461), which we use to generate the CRDs. Nonetheless, we would still propose to use kubebuilder for now as the advantages of an automated generation outweigh the downside of not being able to do oneOf (yet).

### AddonController

The AddonController resides in the seed cluster. Each seed cluster has its own instance of an AddonController. It has two main jobs:

- selecting compatible addons from the AddonCatalogue for each user cluster
- managing AddonInstallations for each user cluster. It is important to note that it only tracks which addon should be installed in which cluster, but does not do the installation of the addon itself (this task will be done by the AddonInstallation controller).
- providing a query-endpoint/function that takes in a clusterConstraints and returns a list of all available addons. We need this functionality to later on make it possible for users to select addons in the cluster wizard (see #6000). We think it makes sense to include this already in the proposal, so we build an architecture that will work for future requirements

The AddonController works by watching the AddonCatalogue CR from the Catalogue Distributor and managing an AddonCatalogue per UserCluster. It stores a version containing only compatible addons in the SeedCluster-namespace that corresponds to a UserCluster. In order to evaluate if an addon is compatible, the AddonController reads the `constraints` field of an AddonCatalogue CR.

In order to track which addon should be installed in which cluster, the AddonController makes use of AddonInstallation CRs.

### AddonInstallation CR

- contains a mapping of a specific addon and cluster where to install
  - the addon is specified by an addonRef. Currently, this consists of a name and version
  - the cluster where to install is specified by the namespace the AddonInstallation was created in (`metadata.namespace`) →The AddonController makes sure the AddonInstallation CR is being placed in the correct namespace
- tells if an add-on is a CoreAddon or not → We propose to have this per installation, as different user clusters can have different CoreAddons. Furthermore, it should be noted that the AddonController will always set the `coreAddon` field to false, while the CoreAddonController will always set it to true
- contains the merged values that are being passed for the installation. This is needed as customers can set custom values for each installation.

```yaml
apiVersion: kubermatic.k8c.io/v1
kind: AddonInstallation
metadata:
  name: prometheus
    namespace: cluster-47s2ddlfgj
spec:
  coreAddon: false #This can be omitted, false will be the default
  targetNamespace: monitoring
  createNamespace: true # flag to create the targetNamespace
  addonRef:
    name: prometheus
    version: v1
  values: # Both structure and values depend on what is configured in the addon
    abc: xyz
```

### CoreAddonController

The CoreAddonController resides in the seed cluster. Its main purpose is to manage AddonInstallation CRs for CoreAddons. This job entails standard create/update/delete operations as well as picking the correct CoreAddons per cluster. The picking is required as some CoreAddons are specific to the type of cluster (e.g. chosen cloud provider, chosen CNI,...). Additionally the CoreAddonController protects against accidental deletion of AddonInstallation CRs of type Core. We think this makes sense as losing a CoreAddon (e.g. OpenVPN) could be detrimental to User Cluster health.

In order to know which UserCluster requires which CoreAddons, the CoreAddonController  works by maintaining a sub-list of CoreAddons for each UserCluster computed from the following sources:

- cluster.kubermatic.io/v1 CR → Will be used for CoreAddons that depend on a clusters configuration (e.g. cilium vs canal). The logic for this will be coded directly into CoreAddonController as it can become quite complicated
- CustomCoreAddon configmap →A configmap that contains a list of addons that should be added to every cluster. This map can be edited by the Cluster Admin. This allows Cluster Admins to specify their own Addons that should be shipped with every cluster, which is in line with the current [defaultManifests](https://docs.kubermatic.com/kubermatic/v2.18/guides/addons/) functionality

In order to protect against accidental deletion of an AddonInstallation of type Core, we propose to make use of k8s [finalizers](https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers/). In this case the CoreAddonController would automatically add itself as a finalizer to any AddonInstallation it creates. Then whenever an AddonInstallation is being deleted, kubernetes will wait with the deletion until the finalizer is removed. Afterwards CoreAddonController is going to run a couple of checks internally:

1. Ask KKP api if the cluster was marked for deletion → if yes, remove finalizer and internal addon list; We need this check to ensure that CoreAddonController is not blocking the deletion of a cluster
2. Check if the CoreAddon is still in internal list
   - yes →keep CR as is to block deletion
   - no →remove finalizer; CR will be deleted by k8s. This allows ClusterAdmins to remove CoreAddons if they really want to by removing them from CoreAddonControllers internal list, but still protects against accidental deletion

An AddonInstallation CR by the CoreAddonController could look like this. Note the changed `spec.coreAddon` and `metadata.finalizer` fields:

```yaml
apiVersion: kubermatic.k8c.io/v1
kind: AddonInstallation
metadata:
  name: OpenVPN
    namespace: cluster-47s2ddlfgj
    finalizers:
     - kubermatic.CoreAddonController
spec:
  coreAddon: true
  targetNamespace: openvpn
  createNamespace: true
 addonRef:
  name: openvpn
  version: v1
```

### AddonInstallationController

The AddonInstallationController resides in the seed cluster. Its main purpose is to watch AddonInstallation CRs and ensure that the requested addons are installed/modified/deleted in the corresponding user clusters. By extension of this, the AddonInstallationController needs to be able to fetch the manifests for addons.

The functionality of the AddonInstallationController depends on the selected `source` and `method` in the AddonInstallation CR.

**Sources**

*git*

- external git repository to pull from
- for this source, the AddonInstallationController runs a git checkout from a remote repository
- spec:
  - remote →URL of the git repository
  - ref →git ref (e.g. branch, commit, sha, ...)
  - path →local file path. This is useful as it allows users to use one git repository for multiple addons
  - auth →can be configured to be pulled from a k8s secret (e.g. ssh-key, user-pw)
- Note: we have considered offloading the spec into a separate CR (e.g. AddonRegistry). However, we noticed that it might not be desirable to have ref and path in a separate CR, as you most likely want these to be different for different addons/versions. For the remaining remote and auth, we think it is not worth the extra effort to offload them.

*docker*

- this is basically the same as the current addon-system we have
- an external docker registry to pull an image from which contains one or multiple addon-manifests at a predefined path
- for this source we propose AddonInstallationController to make use of [go-containerregistry](https://github.com/google/go-containerregistry) to pull the image directly onto its filesystem. Afterward, AddonInstallationController would untar all the different layers and construct the filesystem, from which it would the requested addon-manifests
- spec:
  - image → full URL to docker image
  - path →local file path. This is useful as it allows users to use one docker image for multiple addons
  - auth →docker-auth. can be configured to be pulled from a k8s secret

*helm*

- external helm repository to pull from
- only compatible with method helm
- spec:
  - url → url of the helm repository
  - repository name → will be automatically created by the AddonInstallationController to ensure the name is unique per url
  - chart → name of the helm chart
  - chart version → version of the helm chart
  - auth →authentication. Currently, only user-pw is [supported by helm](https://helm.sh/docs/topics/registries/#auth).

**Methods**

*go-template*

- this will be largely the same as the current addon approach
- we propose to add a small functionality to handle dependencies. The AddonController should parse all manifests first -> check for namespace and CRD manifests -> apply namespaces and CRDs -> wait for apply to finish -> apply remaining manifests

*helm*

- helm release of a helm chart
- the AddonController pulls the corresponding chart and runs a helm installation/modification/deletion
- spec:
  - name → name of the helm release
  - values →override of values. Uses helms standard `values.yaml` override
- we propose to use helm's build in dependency mechanism and not build any custom logic around it

*note: the \[TemplateData\](*[*Addons - Kubermatic Documentation*](https://docs.kubermatic.com/kubermatic/v2.18/guides/addons/)*) is passed as an additional value to helm*

## Common User Flows

![User Flows](images/improveaddons-user-flows.png)
\
([source](https://app.diagrams.net/#G1drryGHt2MbCDY6wRz99u7lMSBYXgI2lH))

## Alternatives considered

- building our own package manager. The idea behind this was that we could also solve  the issue of having complex packages with dependencies. We have decided against this as it would require significant effort to implement, which is not worth it for this single feature. Additionally using for example helm also offers a basic variant of [handling dependencies](https://helm.sh/docs/chart_best_practices/dependencies/)
- there has been [previous work](https://github.com/kubermatic/kubermatic/blob/master/docs/proposals/platform-extensions.md#competitive-landscape) done on considering KubeApps, Flux, OLM, and Kyma for this task

# Glossary

## CoreAddon (old name: Default **Addon**)

A *CoreAddon* is installed on all user clusters based on configuration. The user can not delete *CoreAddons*. *tbd: Should CoreAddons be hidden in the UI, or should show them in a different way (e.g. are shown in the UI in a special frame and with a special symbol to indicate that they are core addons)*

- Shipped by Kubermatic,  tied to KKP version
- Tested by Kubermatic (Kubernetes versions)
- Supported by Kubermatic

## OptionalAddon (old name: Accessible Addon)

An *OptionalAddon* is installed on all user clusters on demand during runtime. The user can also uninstall and reinstall them. They are visible in the KKP UI.

- Shipped by Kubermatic, tied to KKP version
- Tested by Kubermatic (Kubernetes versions and customizations)
- Supported by Kubermatic

## CustomerAddon (old name: Custom Addon)

A *CustomerAddon* is an additional addon installed in all user clusters on-demand, but not shipped with KKP. It is provided by the customer. The visibility in the KKP UI depends (tbd).

- Shipped by the customer
- Not tested nor supported by Kubermatic
- Could break clusters
- The customer is fully responsible

*Note: From a technical perspective, OptionalAddons and CustomerAddons will be handled the same way. However from a contractual perspective, it is important to differentiate: OptionalAddons are supported by Kubermatic, while CustomerAddons are not.*

# Task & effort

*Specify the tasks and the effort in days (samples unit 0.5days) e.g.*

- This section will be added as soon as we have worked on creating user-stories

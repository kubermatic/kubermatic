# Platform Extension

**Author**: Marcin Franczyk, Sascha Haase, Moritz Bracht, Sankalp Rangare

**Status**: Draft proposal.

## Goals

This proposal is to introduce a new platform-extension mechanism which will
provide a more flexible and more transparent way to manage the life-cycle of
extensions in KKP than the current Addon mechanism.

## Non-Goals

Replacing the current Addon mechanism is not in the scope of this proposal, but
could be a long-term result.

## Motivation and Background

When new core concepts are being introduced to KKP we currently face two
questions:

1. How is it going to be implemented?
2. How is it going to be integrated?

To be able to focus on the implementation part, the integration should be
streamlined. The current approach of extending the functionality of user
clusters (Addons) has some limitations which makes it not suitable to use as a
general approach for extensions.

### Current Addon Mechanism

There are *Default Addons*, *Accessible Addons* and *Custom Addons*. Default
Addons are installed on every user cluster, Accessible Addons are available for
installation or provide a configuration interface for Default Addons and Custom
Addons are like Accessible Addons, but provided by the users themselves.  To
add Custom Addons to the catalog of installable Addons the user needs to
provide manifest templates in a docker image that is based on Kubermatic's
provided `kubermatic/addons` docker image.

To select, which Accessible and Custom addons are deployed to all user
clusters, they are being added in the KKP configuration. The KKP operator
reconciles the installation to match the configuration.

#### Limitations

This mechanism is suitable for immutable core components like OpenVPN, Canal or
kube-proxy, but has its limitations with more complex components:

* Extensibility
  * Default Addons and Accessible Addons are shipped in the same medium (docker
    image)
  * Adding Custom Addons is rather complex
* The life-cycle of Addons is tied to the life-cycle of KKP
  * User provided manifest images need to be rebuilt on every KKP upgrade
* No control over dependencies
  * When Addons depend on other Addons the order of installation can not be
    defined. All Addons will be installed at once and it takes several
    reconciliation loops to get everything ready.
* Installing more complex K8s addons can lead to complications
  * If the shipped manifests of an Addon contains multiple CRDs and CRs it
    takes multiple reconciliation loops until all manifests are successfully
    applied. This results in several warnings in the cluster event log.

## Requirements for Implementations

*The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT", "SHOULD",
"SHOULD NOT", "RECOMMENDED",  "MAY", and "OPTIONAL" in this document are to be
interpreted as described in RFC 2119.*

1. Extension source
	* (1.1) The source for extensions MUST be independent from the KKP
	  installation, so rather a catalog than a shipped docker image containing
	  manifests.
	* (1.2) The source for extensions MUST be extensible by external sources.
	  For example by providing a list of additional catalogs.
2. Dependency management
	* (2.1) It MUST be possible to define or resolve dependencies between
	  extensions, so the order of installation can be considered.
3. Complex extensions
	* (3.1) It MUST be possible to install extension-operators that handle the
	  installation of complex extensions.
4. Life-cycle management
	* (4.1) The life-cycle of native extensions MUST be tied to the life-cycle
	  of KKP.
	* (4.2) The life-cycle of external extensions SHOULD be tied to the
	  life-cycle of Kubernetes.
5. Maintainability at runtime
	* (5.1) External extensions MUST be up- and downgradable at runtime
	  dependent on Kubernetes version.
	* (5.2) External extensions MUST be uninstallable at runtime.
6. Access to cluster data
	* (6.1) Extension MAY need access to data from the Master Cluster. For
	  example a mapping of cluster users to SSH keys - to be used for
	  authentication within an extension.

## Competitive Landscape

There are already some solutions out there that solve the problem of extending
Kubernetes clusters. In the following we want to take a look at those and see
if we can apply some of their concepts to solve our requirements.

### KubeApps

https://kubeapps.com/docs/

KubeApps offers 2 ways to install applications to a cluster:

1. Helm charts
2. Operator Lifecycle manager (Operator hub)

#### Helm

##### Sources

Helm charts can be installed from public and private Helm repositories. The
Bitnami repository is enabled by default, but other public repositories can be
added easily. Private repositories like ChartMuseum, Harbor or Artifactory
(pro) can also be added.

There is a controller that watches an AppRepository CR and creates a cronjob
for each that repeatedly scans the Helm charts available and stores the chart's
metadata in KubeApps internal database.

##### Installation

Installation of Helm charts happens imperatively through the KubeApps API.
Their roadmap states that they want to create plugin-based system and support
both this imperative approach and also a declarative approach by adding flux as
a plugin.

#### Operators

##### Sources

If OLM is installed the KubeApps Dashboard also allows to install operators
from the operator hub (and other sources added to OLM).

##### Installation

Once an operator is installed, applications managed by this operator get listed
along with existing helm charts and are ready to be installed.

### Flux

https://fluxcd.io/docs/use-cases/helm/
https://fluxcd.io/docs/guides/helmreleases/
https://github.com/fluxcd/helm-controller/blob/main/docs/spec/README.md

Flux is a "GitOps toolkit" that conists of different tools to manage
applications based on helm charts in a fully declarative approach.

#### Helm

##### Sources

An operator watches HelmRelease CRs and creates artifacts from charts found in
Git repositories, Helm repositories and S3 buckets.

##### Installation

Another operator watches HelmChart CRs which use HelmRelease CRs as kind of
template. It checks for the availability of the referenced chart artifact and
all required dependencies and then fetches the artifact and takes all required
Helm actions like install or upgrade to reach the desired state of the
application.  If defiend Helm test actions are also executed. Retries, rollback
or uninstall are executed as configured if any Helm action fails.

#### Operators

There is no concept for applications that are too complex to be installed
directly via Helm charts.

### Operator Lifecycle Manager

https://olm.operatorframework.io/docs/

OLM provides a declarative way to handle the lifecycle and dependencies of
Kubernetes operators.

#### Helm

##### Sources

With the operator SDK simple operators can be generated directly from Helm charts.

##### Installation

To install applications based on Helm charts the generated operators need to be
installed and then a CR of the corresponding kind has to be created so the
operator takes care of bringing the application up.

#### Operators

##### Sources

Operators from operatorhub can be installed natively. The catalog of operators
can be extended by 3rd-party catalogs or by bundeling own operators into
catalogs.

##### Installation

After operators are being installed, creating the corresponding CRs trigger the
installation of applications.

### Kyma Service Calatlog: Helm Broker

https://kyma-project.io/docs/components/service-catalog/
https://kyma-project.io/docs/components/helm-broker/

Kyma's Helm Broker implements the [Open Service Broker
API](https://github.com/openservicebrokerapi/servicebroker/blob/v2.14/profile.md#service-metadata)
as an abstraction layer on top of Helm. It makes services deployed via Helm
available the same way as cloud services.

#### Helm

##### Sources

Helm charts are being exposed as Service Classes in a Service Catalog. To
convert a Helm chart into a Service Class it is enriched with all necessary
information and metadata into so-called Addons. To make these addons available
they are bundled in repositories.

##### Installation

To provision such a Service in the cluster, the user creates a set of custom
resources (ServiceInstance, ServiceBinding, ServiceBindingUsage). The service
broker then creates an instance of that service and injects a set of user
credetials to make it ready to use.

#### Operators

There is no concept for applications that are too complex to be installed
directly via Helm charts.

## Implementation Proposals

(unordered list)

* [Helm magager](./platform-extension-implementation-helm-manager.md)
* tbd.

## Open questions

* Is there really a difference between “native extensions” (like Eventing) and
  “external extensions” (like KubeVirt)? If so, does this difference justify
  different life-cycles?

## Glossary

* Addon - KKP extension in the "old sense"
* Catalog - Generalization for repository/registry
* Extension - KKP extension in the "new sense"
* External extension - Extension developed by third parties
* KKP - Kubermatic Kubernetes Platform
* Native extension - Extension developed and supported directly by Kubermatic

## References

* [Platform Extension mechanism #6992](https://app.zenhub.com/workspaces/development-input--estimation-5fa947bf2732730014ef98c1/issues/kubermatic/kubermatic/6992) 
* [AddOn Management improvements #6180](https://app.zenhub.com/workspaces/development-input--estimation-5fa947bf2732730014ef98c1/issues/kubermatic/kubermatic/6180)

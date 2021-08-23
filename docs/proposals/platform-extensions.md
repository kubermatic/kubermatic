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

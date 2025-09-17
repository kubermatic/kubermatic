# Application Catalog Revamp

**Author**: Burak Sekili (@buraksekili)

**Status**: Draft proposal; prototype in progress.

## Table of Contents

* [Application Catalog Revamp](#application-catalog-revamp)
  * [Table of Contents](#table-of-contents)
  * [Goals](#goals)
  * [Non-Goals](#non-goals)
  * [Motivation and Background](#motivation-and-background)
    * [Solution: An External Application Catalog](#solution-an-external-application-catalog)
  * [Implementation](#implementation)
    * [Git workflows](#git-workflows)
      * [Repository structure](#repository-structure)
      * [OCI registry](#oci-registry)
      * [OCI Artifact Structure for Application Definitions](#oci-artifact-structure-for-application-definitions)
    * [New Controller Manager](#new-controller-manager)
    * [Filtering Mechanisms](#filtering-mechanisms)
    * [Reconcile ApplicationDefinitions](#reconcile-applicationdefinitions)
      * [Reconciliation Process](#reconciliation-process)
  * [Enabling the new manager](#enabling-the-new-manager)

## Goals

Redesign the KKP Application Catalog to make it more maintainable, scalable, and independent of core KKP releases, while also enabling flexible support tiering and broader usage.

This technical document only considers the Application catalog revamp from a technical point of view.

It looks for answers for how to create an Application Catalog Git repository and manage Application catalogs based on this new Git repository at the heart of KKP.

## Non-Goals

This proposal does not aim to redesign the core application installation process, methods, or flow. Its goal is not to define or scope the tiering support levels.

## Motivation and Background

The ApplicationDefinition changes are tightly coupled with KKP releases, affecting the flexibility and increasing the maintenance overhead.

Currently, updating an `ApplicationDefinition` is tied directly to the KKP release cycle. This is because ApplicationDefinition manifests are embedded within the KKP codebase, requiring a new KKP release to make updates available to users. This dependency makes delivering application updates tied to the KKP release.

### Solution: An External Application Catalog

We propose decoupling the ApplicationDefinition lifecycle from KKP releases. The solution outlined in this document involves a dedicated Git repository that will host ApplicationDefinition manifests. This repository will be managed by a new Kubernetes controller that is not attached to the KKP product itself (out-tree implementation).

Whenever an ApplicationDefinition is updated and merged into the main branch of this Git repository, an automated job will be triggered. This job will package the updated manifests and push them to an OCI registry, which makes the ApplicationDefinition updates available through an OCI artifact. This registry will serve as the single source of truth for the latest ApplicationDefinition manifests.

## Implementation

This section focuses on technical details, including Git workflows and Kubernetes controllers.

### Git workflows

#### Repository structure

We will create a new GitHub repository to host all supported ApplicationDefinition resources on KKP. These definitions are going to include the tiering metadata, which represents the level of support provided for each ApplicationDefinition.

```
/applications/
	<application_name>/
		application.yaml
		metadata.yaml
```

Each application will include metadata files that define the tier level of the corresponding application for now. It can also include any additional information that we consider necessary, like contact information.

```
# application/<application_name>/metadata.yaml
tier: "gold"
```

As part of this revamp, we currently plan to maintain ***a single*** ApplicationDefinition catalog. However, to support potential future use cases where catalogs may need to be customizable or extensible, we may introduce a dedicated “catalogs/” directory within the Git repository. Each catalog can then reference specific ApplicationDefinitions stored in the same repository.

```
catalogs/
	default/ # this will correspond to our default Application Catalog
		metadata.yaml
```

The catalog metadata lists the names of the ApplicationDefinition custom resources (i.e., the values of metadata.name) to indicate which applications are included in a given catalog.

#### OCI registry

To keep the cluster’s application catalog in sync with the repository, we propose an automated update flow that is triggered on changes to the main branch. Whenever the main branch is updated, typically once a month or week, KKP Admins should be able to retrieve all  (or selected ones) ApplicationDefinitions and update the ApplicationInstallations running in their cluster without waiting for the next KKP release. This will help us to update and maintain ApplicationDefinitions on KKP without depending on the release cycle of KKP.

To achieve this, we are going to define a workflow that publishes the updated manifests to a public Kubermatic OCI registry, which will serve as the manifest source. A dedicated controller manager can then pull these manifests and reconcile them into the cluster state. For example, if the Kubermatic team updates the “nginx” Application, merging the change into the main branch triggers the Git workflow to update the OCI registry. If a customer needs this fix, they simply update the KubermaticConfiguration CR to reference the tag containing the nginx update. Once the KubermaticConfiguration CR is updated, the new controller-manager reconciles the ApplicationDefinitions and applies the updated definition. From that point onward, the rest of the process, such as updating the ApplicationInstallation, is handled by the existing controllers in KKP. 

Whenever an ApplicationDefinition is updated and merged into the main branch of this Git repository, an automated job will be triggered. This job will package the updated manifests and push them to an OCI registry, which makes the ApplicationDefinition updates available through an OCI artifact. This registry will serve as the single source of truth for the latest ApplicationDefinition manifests.

This process should be highly configurable, allowing users to define a custom registry, particularly important for air-gapped or restricted environments. Fortunately, Kubermatic already implements registry resolution logic within its core controllers, so we can follow the existing pattern to ensure seamless integration.

#### OCI Artifact Structure for Application Definitions

The OCI registry follows the Git repository's `applications/` directory structure. This means you can easily pull the contents for a specific Git commit, ensuring you have the exact set of manifests from that point in time. For example:

```shell
oras pull -o ./apps quay.io/kubermatic/applications:<git_commit>
```

This command will result in a directory structure like this:

```shell
tree ./apps
.
├── applications
│   └── nginx
│       ├── application.yaml
│       └── metadata.yaml
│   └── nvidia-gpu-operator
│       ├── application.yaml
│       └── metadata.yaml
│   └── <another_application>
│       ├── application.yaml
│       └── metadata.yaml
...
```

The controller will then pull the specified OCI artifact (as defined in the above `oras` command), extract the `applications/` directory, and process the manifests to update the application catalog in the cluster. 

The new Kubernetes controller requires the OCI artifact to follow the specific format, as follows:

```json
$ cat manifest.json
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.manifest.v1+json",
  "artifactType": "application/vnd.kubermatic.application-catalog.v1",
  "config": {
    "mediaType": "application/vnd.oci.empty.v1+json",
    "digest": "sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a",
    "size": 2,
    "data": "e30="
  },
  "layers": [
    {
      "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip",
      "digest": "sha256:89122c18ad470e7ac0686bc18836a011a1cf4daf9e450d5764ead0525aba9b4b",
      "size": 20063,
      "annotations": {
        "io.deis.oras.content.digest": "sha256:3ef48f58f03bfad268dcdee5990a5e9a4ef056e449cd9a9f14e3215cf02e705c",
        "io.deis.oras.content.unpack": "true",
        "org.opencontainers.image.title": "applications"
      }
    }
  ],
  "annotations": {
    "org.opencontainers.image.created": "2025-08-08T15:07:10Z"
  }
}
```

The key requirements for this artifact are:

- The manifest must have its `artifactType` set to: `"application/vnd.kubermatic.application-catalog.v1"`  
- While the artifact can contain multiple layers, the controller is designed to process only the specific layer that meets **both** of the following criteria:  
  - `"mediaType": "application/vnd.oci.image.layer.v1.tar+gzip"`  
  - `annotations["org.opencontainers.image.title"] == "applications"`  
- If multiple layers matching these conditions are found within the same artifact, the controller will throw an error. This action is necessary because the ambiguity makes it impossible to determine which layer contains the intended application content.

To support future changes to the OCI registry's directory structure, we can include the optional `artifactType` field in the OCI Artifact manifest when pushing manifests. By specifying the artifact's type in this field, we can enable a controller to dynamically identify the expected directory structure, ensuring adaptability based on the `artifactType` field; for instance, if `artifactType == v1, follow X directory structure, else follow Y directory structure`.

### New Controller Manager

Instead of modifying the existing master-controller-manager, a cleaner and more modular solution would be to introduce a dedicated controller manager that runs on the KKP master cluster. This new manager would be responsible for:

- Pulling ApplicationDefinition manifests from the specified OCI registry and reconcile them in the cluster.  
- Delegating the actual reconciliation of the pulled ApplicationDefinitions to the existing application-definition controller running in the master-controller-manager.

Configuration for this new manager can still be passed through the KubermaticConfiguration CR, and its lifecycle can be managed by the Kubermatic Operator.

This separation provides the following benefits:

- Decouples the application definition delivery flow from the KKP core release cycle.  
- Enables faster iteration and independent delivery of fixes or updates to application definitions.  
- Allows safer experimentation without risking the stability of the core master-controller-manager.

The following demonstrates the expected directory structure in the OCI artifact registry:

```
/applications/
    <application_name>/
        application.yaml    # actual ApplicationDefinition CR
        metadata.yaml       # metadata information about ApplicationDefinition, mainly `tiering`
```

This registry is going to all applications that KKP aims to support out of the box. Thus, we should allow users to filter out which ApplicationDefinitions to pull from this OCI registry.

### Filtering Mechanisms

We propose two distinct levels of filtering to manage `ApplicationDefinition` resources effectively:

1. **Registry-Level Filtering**: This method applies directly to ApplicationDefinition manifests stored within the OCI registry. It dictates which ApplicationDefinition resources are *reconciled* based on the registry.  
     
2. **Cluster-Level Filtering**: This method applies to ApplicationDefinition resources that already exist within the master cluster at the time of reconciliation. It determines which existing ApplicationDefinition resources are *reconciled* against the state of the OCI registry.

Consider an OCI registry containing ApplicationDefinition manifests for applications "X", "Y", and "Z".

As a user, I might have the following requirements:

- Selective Reconciliation from Registry: I may not want to pull or reconcile all ApplicationDefinition resources available in the OCI registry. For instance, I might only be interested in "gold" tier applications, or specifically ApplicationDefinition resources for applications "X" and "Y".  
- Skipping Reconciliation of Existing Resources: I might want to prevent the reconciliation of existing ApplicationDefinition resources (e.g., "X", "Y", or "Z") because they are already present in my cluster, and I prefer to manage their updates independently, rather than constantly aligning them with the latest state of the OCI registry.

Therefore, our filtering approach will rely on two primary mechanisms: one applied to ApplicationDefinition resources residing in the OCI registry, and another applied to ApplicationDefinition resources already deployed within the cluster.

1. Bypass method

Sometimes, you might have an ApplicationDefinition (e.g., for application "X") already deployed in your cluster, and you don't want it automatically updated from the OCI registry. The bypass mechanism lets you explicitly skip reconciliation for such ApplicationDefinitions.

To use this, simply add a specific bypass label to the ApplicationDefinition resource in your cluster. The new controller-manager will recognize this label and ensure that the particular ApplicationDefinition is ignored during subsequent reconciliation cycles.

2. Selector

We'll introduce a new configuration option within KubermaticConfiguration that allows users to specify which ApplicationDefinitions to reconcile based on the OCI artifact registry.

```
limit:
    metadataSelector:
        tierRefs:
        - "gold"
        - "silver"
    nameSelector:
    - "X"
    - "Y"
```

- **`metadataSelector`**: Allows filtering `ApplicationDefinitions` based on metadata defined in their `applications/<application_name>/metadata.yaml` file.  
  - Currently, only the `tierRefs` field is supported. This means users can filter `ApplicationDefinitions` from the OCI registry based on their support tier level.  
  - The `tierRefs` field is an array of strings and uses **additive logic**. For instance, the example above translates to "include ApplicationDefinitions with a 'gold' **or** 'silver' tier."  
- **`nameSelector`**: Enables users to select `ApplicationDefinitions` based on their `.metadata.name` field.  
  - Similar to `tierRefs`, this field also uses **additive logic**. The example above means "include `ApplicationDefinitions` named 'X' **or** 'Y'."

If both selectors are used at the same time, the new controller-manager will “AND” to conditions. For example, the above `limit` will be interpreted as “I am looking for ApplicationDefinition resources where (tier is gold OR silver) AND (name is X or Y)”

If no `limit` field is defined in your KubermaticConfiguration, the controller will default to reconciling all ApplicationDefinitions from the OCI registry without any filtering.

### Reconcile ApplicationDefinitions

The new controller watches for KubermaticConfiguration CR, and reconciles ApplicationDefinitions based on the configuration parameters defined in the KubermaticConfiguration CR.

The following example demonstrates the configuration options for the new ApplicationDefinition manager through KubermaticConfiguration CR.

```yaml
apiVersion: kubermatic.k8c.io/v1
kind: KubermaticConfiguration
metadata:
  name: kubermatic
  namespace: kubermatic
spec:
  applications:
    manager:
      logLevel: "debug"
      registrySettings:
        registryURL: quay.io/buraksekili/k8c-apps
        tag: 7a4ad7a14ba4bb67ec89fdc1e61993d6297b1005
        credentials: # same as current ApplicationDefinition approach
          password:
            key: pass
            name: <secret-name>
          username:
            key: user
            name: <secret-name>
  featureGates:
    ExternalApplicationCatalogManager: true
```

`applications.manager`: Contains the configurations for the new external ApplicationDefinition manager.

- `applications.manager.logLevel`: Configures the log level for the new manager  
- `applications.manager.registrySettings`: Configures the OCI registry that contains the ApplicationDefinition artifacts.  
- `featureGates.ExternalApplicationCatalogManager` A feature gate that, when enabled, instructs the Kubermatic Operator to deploy the external ApplicationDefinition manager.

When this feature gate is enabled, the Kubermatic Operator deploys the manager to the `kubermatic` namespace on the master cluster.

#### Reconciliation Process

**Note**: For clarity in the following section, `ApplicationDefinitions` read from the OCI registry are referred to as `artifactAd`, while those existing on the cluster are referred to as `clusterAd`.

The controller performs the following steps to reconcile ApplicationDefinitions:

- Reads OCI registry URL and tag from the KubermaticConfiguration CR  
  - If these are not defined, the controller uses the default values.  
    - Registry URL can be set to the public Kubermatic OCI registry  
    - The tag can be "latest" by default  
- Pulls the OCI artifact content and saves it into memory, as `[]byte`  
  - This `[]byte` array corresponds to the tar.gzip file.  
- Reads the ApplicationDefinition files from the `applications/` directory in the zipped file and instantiates ApplicationDefinition Go structs from each `applications/<application_name>/application.yaml` file.  
  - Also, process `metadata.yaml` file of each ApplicationDefinition  
  - This process is similar to how existing ApplicationDefinitions are loaded from embedded static files.  
- Applies `selectors` on the parsed ApplicationDefinition if the selector is defined.  
- Reconcile ApplicationDefinitions after applying selectors on the file content.  
- Lists all ApplicationDefinition CRs on the cluster (`clusterAd`s).  
  - List with label selector, e.g, `!apps.k8c.io/application-bypass`  
- For each `clusterAd`, the controller checks if it belongs to the filtered OCI registry content and updates the metadata accordingly.  
  - If `clusterAd` belongs to filtered OCI registry content, the controller adds a managed-by label and the registry digest as an annotation. Any existing unmanaged labels are removed.  
  - Else, the controller adds a label to indicate the resource is unmanaged and removes any managed-by metadata if it exists.  
  - This unmanaged label helps the UI identify custom or orphaned ApplicationDefinitions that may not have been created intentionally.  
- Add/Update KubermaticConfiguration to include up-to-date OCI registry digest in the annotations.

## Enabling the new manager

To prevent unintended updates to applications on user clusters, the new manager is activated only when the `ExternalApplicationCatalogManager` feature gate is enabled. If this gate is disabled, the Kubermatic Operator will not deploy the manager.

Once enabled, the manager begins reconciling all existing ApplicationDefinitions on the cluster. To exclude critical applications from this process, users can apply a specific bypass label to the corresponding ApplicationDefinition. The controller ignores any ApplicationDefinition that includes this label, ensuring critical workloads remain untouched.


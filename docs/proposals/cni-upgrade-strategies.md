# CNI Upgrade Strategies

**Authors**: Rastislav Szabo (@rastislavs)

**Status**: Draft proposal.

## Goals

Allow flexible CNI upgrades in KKP user clusters:

- Allow upgrading CNI version in existing KKP user clusters.
- Make sure that CNI upgrades can be made with minimal impact on the workload running in the user clusters.
- If possible, also allow disruptive upgrades (like migration between different CNIs) for experienced users, but do not put any guarantees on it.

## Non-Goals

This proposal focuses on CNI upgrades per single user cluster, not for batch upgrades across multiple user clusters at once.

## Motivation and Background

At the moment of writing, the CNI version in KKP is fixed and the CNI cannot be upgraded to a newer version without re-creating the user cluster (CNI version is an immutable field in the Cluster API). This means that the long-lived user clusters may eventually be stuck on a very old version of the CNI.

Historically, the CNI upgrades were done unconditionally together with the KKP upgrade. That approach however, brought some disadvantages:

- If the CNI version difference between two KKP versions is too big, the upgrade may not be safe or possible in an easily automated way.
- No possibility to stay on an existing version in case that it is needed (e.g. in case of a regression bug in the CNI).
- No possibility to rollback to the version before the upgrade if something goes wrong.
- The CNI upgrade is triggered for all user clusters at once.

Because of these reasons, since the CNI is a critical component of Kubernetes clusters, the CNI version in KKP has been fixed in the past, which is not ideal.

Another motivation for advanced CNI upgrade strategies is introduction of additional CNIs into KKP. With this proposal, we would also like to address the migration between different CNIs for experienced users as well.

## Implementation

The main idea of the CNI upgrade strategies is to not upgrade CNI unconditionally, but allow upgrading whenever the KKP user triggers it (from KKP UI or API). That way the users can be in full control of the CNI in their clusters, and can plan the upgrade for a maintenance window if necessary. For CNIs that provide upgrade pre-flight checks and post-upgrade verification checks (like CIlium), the users can run those manually to make sure that the upgrade process is smooth.

There will be 3 types of CNI upgrades allowed:

- “Safe” CNI upgrade
- “Unsafe” CNI upgrade
- Migration between different CNIs (also unsafe)

### “Safe” CNI upgrade

“Safe” CNI upgrade will be allowed between two minor versions of the same CNI (e.g. from Canal v3.19 to Canal v3.20). Skipping any minor version in between (e.g. from Canal v3.18 to Canal v3.20) will not be allowed. Triggering this type of upgrade will be possible from KKP UI and Cluster API at any time. This type of upgrade will be supported and tested by Kubermatic.

### “Unsafe” CNI upgrade

“Unsafe” CNI upgrade can be allowed between any versions of the same CNI, but only when the cluster is labeled with the `unsafe-cni-upgrade` label. This type of upgrade is not supported by Kubermatic and the users are fully responsible for the consequences it may have. However, Kubermatic will provide some guidelines and documentation for specific version upgrades, like the upgrade of Canal from v3.8 (default for KKP v2.12 - v2.17) to v3.19 (default for KKP v2.18).

### Migration between different CNIs

Migration between different CNIs can be allowed when the cluster is labeled with the `unsafe-cni-migration` label. This type of upgrade is not supported by Kubermatic and the users are fully responsible for the consequences it may have. The process can be quite complex and the users should be aware of the upgrade path prerequisites and steps. However, Kubermatic will provide some guidelines and documentation for migration between selected CNIs, eg. from Canal to CIlium.

### Handling of New Clusters

For new clusters, KKP will always default to the latest supported version of the selected CNI.

### Deprecating CNI Versions

CNI versions considered as too old (e.g. release date lower than the release date of the last supported Kubernetes version) will be deprecated; they will not receive any Kubermatic support. This will be noted in the KKP CNI documentation and the respective KKP upgrade documentation.

## Alternatives Considered

Apart from the manually triggered upgrade strategy, we also considered two other approaches:

- Unconditional CNI upgrade together with the KKP upgrade.
- Unconditional CNI upgrade together with the Kubernetes version upgrade.

The disadvantages of the first one were mentioned in the motivation part of this proposal. Unconditional CNI upgrade together with the Kubernetes version upgrade shares some disadvantages with the first one, and also adds the complexity of maintaining a complicated support matrix.

After considering those, we decided to go with the manually triggered upgrade strategy, which gives KKP users the most flexibility and control over the user clusters.

## Tasks & Efforts

See the Epic [CNI upgrade support](https://github.com/kubermatic/kubermatic/issues/7916).

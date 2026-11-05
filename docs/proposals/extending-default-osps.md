# Extending Default OperatingSystemProfiles

**Author**: Burak Sekili (@buraksekili)

**Status**: Draft proposal

**Issues**:

* https://github.com/kubermatic/kubermatic/issues/12310
* https://github.com/kubermatic/operating-system-manager/issues/231

## Table of Contents

- [Goals](#goals)
- [Non-Goals](#non-goals)
- [Motivation and Background](#motivation-and-background)
- [Prior Art](#prior-art)
- [Implementation](#implementation)
- [Compatibility Guarantees](#compatibility-guarantees)
- [Example Walkthrough](#example-walkthrough)
- [Alternatives Considered](#alternatives-considered)

## Goals

Platform admins add typed bootstrap content to the shipped default OperatingSystemProfiles and keep receiving updates to those defaults.

Concretely:

* Add files (for example a company root CA), systemd units, and command lists that run before or after the default commands.
* Deliver additions to selected datacenters, either enforced or as a default.
* Receive OSM updates to the default profiles without reapplying, reviewing or re-merging the customization.

## Non-Goals

* Changing the `OperatingSystemProfile` API, the default profiles, or the immutability webhook in the OSM repository.
* A general templating engine or a patch DSL.
* A dashboard screen in the first cut. Admins apply the extension object with kubectl.
* Covering the `provisioningConfig` phase. Additions target `bootstrapConfig` only. A selector can follow if a use case appears.

## Motivation and Background

An OperatingSystemProfile (OSP) describes how a worker node bootstraps: files, systemd units and templates, split into a `bootstrapConfig` phase for first boot and a `provisioningConfig` phase for later re-provisioning. The Operating System Manager (OSM) ships six default profiles (for example `osp-ubuntu`) embedded in its image and applies them into `kube-system` of each user cluster.

Admins need four kinds of additions on worker nodes: a company root CA, a custom NTP configuration, additional drivers, and extra packages. All four are additive. None of them needs to change what the default profile already does.

Today there are two ways to get there, and both fail:

* Copy a default OSP, edit the copy, and maintain it as a `CustomOperatingSystemProfile`. The copy stops tracking the default. Every upstream fix has to be re-applied by hand. This is the maintenance cost the issue names: "in order to reduce the maintenance effort as I will receive improvements/bugfixes of any OSP default change as well as I can customize OSPs with additional logic".
* Run a privileged DaemonSet that mutates the host after the node joined. The issue links a community NTP addon as the example and reports it as a security risk in a BSI certification.

The copy path for custom profiles exists and works. The seed-controller-manager's `operating-system-profile-synchronizer` watches `CustomOperatingSystemProfile` objects in the seed's `kubermatic` namespace and copies them into `kube-system` of every healthy user cluster, gated on a version change. An OSP spec is immutable without a version bump. Propagation is solved. Composition is not. A `CustomOperatingSystemProfile` is a full profile authored by hand. It never tracks the shipped defaults.

In the issue discussion, embik prefers composition: ship the default profile's building blocks as snippets, pull them in when generating the custom profile, allow custom commands before and after, and resolve everything before the profile reaches the seed or the user cluster. The reporter endorsed that timing. The OSM-side twin issue closed in 2024 without shipping composition, and no extension work is in flight in OSM. This proposal implements the composition step on the KKP side.

## Prior Art

* Cluster API's bootstrap provider (CABPK) types additive hooks on `KubeadmConfig`: `files`, `bootCommands`, `preKubeadmCommands`, `postKubeadmCommands`. This is the only surveyed precedent with a real before-hook. It is the model for `beforeCommands` and `afterCommands`.
* Gardener's `OperatingSystemConfig` types units and files as named lists with `name` and `path` merge keys. This is the model for the list shape and the collision keys.
* ClusterClass patches may only append or prepend to arrays. This is the model for the merge rule: append and prepend only, collision is an error, never a replace.
* kops' `additionalUserData` merges additional parts into the generated bootstrap and never rewrites the core. Same user story, but append-only, without a before-hook.

Patch dialects over rendered output (Talos config patches, Karpenter's MIME merge) do not fit here. OSP templates carry Go-template data that only resolves at render time, so the rendered result cannot be patched on the seed.

## Implementation

### The extension object

A new KKP-owned custom resource `OperatingSystemProfileExtension`:

```yaml
apiVersion: kubermatic.k8c.io/v1
kind: OperatingSystemProfileExtension
metadata:
  name: corp-trust-and-ntp
spec:
  ospTarget:
    profileName: osp-ubuntu   # or os: ubuntu, to select every profile of an OS
  enforced: true              # or default: true; a selector picks datacenters
  files:
    - path: /usr/local/share/ca-certificates/corp-root.crt
      permissions: 644
      content:
        inline:
          data: |
            -----BEGIN CERTIFICATE-----
  afterCommands:
    - update-ca-certificates
```

`files` and `units` are named lists keyed by `path` and `name`. `beforeCommands` and `afterCommands` hold command lists. The types are local minimal structs mirroring OSM's field names. The SDK keeps no dependency on OSM.

Scoping follows the ApplicationDefinition model: `enforced` versus `default`, plus a `selector` over datacenters. A new master-controller-manager synchronizer replicates extension objects to seeds, on the pattern of the existing `application-definition-synchronizer`.

### Composition

A composer in the `operating-system-profile-synchronizer` package merges default profiles with matching extensions:

1. Load the default profiles from the pinned OSM module. The defaults are embedded via `go:embed` (`deploy/osps/default`), and kubermatic already imports the module. The image tag and the module pin must name the same release (see Compatibility Guarantees), so the composer reads the same bytes OSM's own defaults controller applies.
2. Select extensions by scope and by OSP target.
3. Append additions into `bootstrapConfig` only. `beforeCommands` run before the default's bootstrap units, `afterCommands` after them. Command lists render as oneshot units.
4. Reject collisions. A unit name or file path already present in the default is a hard error.
5. Write the result as a `CustomOperatingSystemProfile` with a stable name, for example `osp-ubuntu-ext`, and a version `<default-version>+ext.<hash>`, the hash computed over the whole composed spec.

The name never changes, so the datacenter's `DefaultOperatingSystemProfiles` map and the MachineDeployment profile annotations never move. The version changes whenever the composed content changes, on either side. An OSM update to the default changes the hash even though the extension object is untouched. The existing synchronizer then propagates on the version change, as it does today.

### Rollout

An OSP version bump re-renders the bootstrap data but does not rotate machines. OSM documents rotation as the user's responsibility. A roll trigger closes this gap. When the composed version changes, it stamps the new version into `spec.template.metadata.annotations` of MachineDeployments whose profile annotation references the composed name. The template change produces a new MachineDeployment revision, and machine-controller rotates the machines.

### Admission

A validating webhook rejects unsafe extensions at write time. Unit names and file paths must be unique within the extension and must not collide with the selected default's own entries.

### Files touched

* `sdk/apis/kubermatic/v1/operatingsystemprofileextension.go`: the type
* `sdk/apis/kubermatic/v1/zz_generated.deepcopy.go`: generated
* `pkg/crd/k8c.io/kubermatic.k8c.io_operatingsystemprofileextensions.yaml`: generated, annotated `location: "master,seed"`
* `pkg/controller/seed-controller-manager/operating-system-profile-synchronizer/compose.go` and `roll.go`, with tests
* `pkg/controller/master-controller-manager/operating-system-profile-extension-synchronizer/controller.go`
* `pkg/webhook/operatingsystemprofileextension/validation/validation.go`
* `cmd/kubermatic-webhook/main.go`, `cmd/master-controller-manager/controllers.go`, `cmd/seed-controller-manager/controllers.go`: registration
* kkp-docs, OSM usage page: a section on extending default OSPs

## Compatibility Guarantees

A user who does not use extensions must observe nothing after the upgrade. Each rule below is an acceptance criterion with a test:

1. Zero matching extensions means the composer writes nothing. No object is created, updated or deleted.
2. The composer labels the objects it writes (`kubermatic.k8c.io/composed-by: osp-extension-composer`) and refuses to adopt or overwrite an existing object of the same name that does not carry the label. A name collision surfaces as an error.
3. The roll trigger selects MachineDeployments through the ownership label on the referenced object, never by annotation string alone.
4. The go.mod module pin and the OSM image tag name the same OSM release, and both move forward together. Today they diverge: the module pins `v1.11.4` while the image deploys a master SHA whose embedded defaults differ in all six default profiles. Aligning the pins at the newer content is a prerequisite of this feature. The image is never reverted to an older release. The composer refuses to compose while the pins disagree, so a composed profile never claims defaults the deployed OSM does not carry.
5. Composed names are reserved. An extension cannot derive onto a name that already exists as a hand-authored `CustomOperatingSystemProfile`.

With no extension object present, every added component is inert. No composed profile exists, nothing propagates, no MachineDeployment is stamped. The default flow is unchanged: OSM applies the embedded defaults, and the datacenter map or OSM's mutation webhook selects `osp-<os>`.

## Example Walkthrough

A worked example, from the user's point of view. The user runs a seed with ubuntu worker nodes and wants the company root CA on every bootstrapped node.

Day one:

1. Save the extension from the Implementation section as `corp-trust-and-ntp.yaml` and apply it:

```text
kubectl apply -f corp-trust-and-ntp.yaml
operatingsystemprofileextension.kubermatic.k8c.io/corp-trust-and-ntp created
```

2. The composer runs on the seed. The user reads the composed object:

```text
kubectl -n kubermatic get customoperatingsystemprofile osp-ubuntu-ext -o jsonpath='{.spec.version}'
v1.2.3+ext.9f2e1a
```

The version reads as the default profile's version followed by a hash of the composed spec.

3. Wire the datacenter once. Set `DefaultOperatingSystemProfiles` for `ubuntu` to `osp-ubuntu-ext`. Every new MachineDeployment then carries the profile annotation `osp-ubuntu-ext`.

4. After a node bootstraps, the user verifies the addition on the node. The file `/usr/local/share/ca-certificates/corp-root.crt` exists, and `update-ca-certificates` ran as the after-command.

Upgrade day. KKP ships a release pinning a newer OSM, and the default `osp-ubuntu` gains a kubelet fix at version `v1.2.4`. The user does nothing. On its next run the composer reads the new defaults, and the composed version moves to `v1.2.4+ext.4c7d2b`. The default side changed, so the hash changed, although the extension object is untouched. The synchronizer propagates, the roll trigger stamps the MachineDeployments that reference `osp-ubuntu-ext`, and machines rotate. A node bootstrapped after the roll carries the corp root CA and the kubelet fix.

What the user never does on that day: re-merge the default, re-copy the profile, re-verify the extension against the new defaults, or touch the composed object.

Internally, per composition run: load defaults from the module, select by scope and target, append with collision checks, derive the name and version, write the `CustomOperatingSystemProfile`, kind-swap and propagate on version change, OSM admission accepts the update, the osc controller re-renders the bootstrap secrets, the roll trigger stamps MachineDeployments, machines rotate.

## Alternatives Considered

* Extending the `CustomOperatingSystemProfile` schema with `extends` and additive fields. The CRD is authored in the OSM repository and vendored here. KKP-only fields would be wiped by every re-vendor, or would have to go upstream and recouple this feature to OSM's release cadence. The required hand-maintained `version` conflicts with a controller-derived version.
* A templating engine in OSM that assembles profiles from snippets at generation time. This is the sketch in the issue discussion. It moves all work into the OSM repository and its release train. The KKP-side composer delivers the same composition timing with no OSM change.
* Privileged DaemonSets that mutate the host. Rejected as a security risk, per the BSI certification finding in the issue.
* Patch dialects (strategic merge, JSON 6902) over the profile. Rejected for the rendered form: it contains runtime-only template data. Patching the structured spec would bypass typed admission.

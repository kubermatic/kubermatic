# Kubermatic Installer

**Author**: Christoph (@xrstf)
**Status**: Draft proposal

This is a proposal to simplify both the installation and upgrades of a Kubermatic system.

## Motivation

* Getting the Helm values and KubermaticConfiguration right is sometimes hard,
  an early validation before an installation is attempted would be nice to have.
* Upgrading Helm releases is a pain when migration steps are required. Currently
  we try to limit ourselves and keep the stack stable, sometimes at the cost of
  cleaning up legacy stuff, just so the upgrade is as smooth as possible. Having
  a way to automate *and test* the upgrades would help dramatically in having
  confidence in updates.
* In https://github.com/kubermatic/kubermatic/pull/5552 we began making CRD
  updates explicit (because Helm does never update or delete CRDs). This introduced
  another manual step in the installation procedure that would be nice to
  automate away. Proper CRD handling is important for upgrades anyway.

## Proposal

A Go program will be created that assists the user in performing the steps outlined
above. This `kubermatic-installer` (not to be confused with the *Kubermatic Operator*)
will work similarly to KubeOne's `apply` command, i.e. it will provide an idempotent
command that brings the target environment up to the installer's version, installing
and upgrading components as needed.

The "chain of command" is then as follows:

* The *Kubermatic Installer* installs cert-manager, nginx-ingress-controller, Dex
  and the *Kubermatic Operator* into the cluster.
* The *Kubermatic Operator* then manages *Kubermatic* itself.

As our documentation already separates Kubermatic from the monitoring and logging stack,
the installer will do the same. The first version will only include support for
intalling the Kubermatic stack (cert-manager, nginx, Dex, Kubermatic Operator), later
versions will then gain support for the other stacks.

### User Story

Installing Kubermatic then works like this:

1. The admin prepares a `KubermaticConfiguration` and a Helm `values.yaml`. The goal here
   is to require as minimal customization as needed and to smartly make the right choices:
   For example, things that need to be configured identically in both files can be
   defaulted from one into the other.
1. The admin uses the installer like so:

    ```
    export KUBECONFIG=...
    # or use -kubecofig=...
    ./installer deploy --config kubermaticconfig.yaml --values values.yaml
    ```

   And off they go! The installer will validate the configuration and perform all steps
   to install the Kubermatic stack.
1. Done.

### Goals

* Provide a command-line application for the installer and bundle it together with the
  relevant Helm charts into a Docker image. Maybe even embed the charts into the installer
  binary.
* Use Helm 3 for managing releases.
* Do not require the admin to manually use Helm to manage their installation.
* Allow the admin to manually use Helm if they so choose. Do not hide things from an admin,
  but make it clear that they are leaving the supported paths.
* Provide clear, helpful log output and error messages.
* Never perform dangerous (as in: can possibly lose data or change LoadBalancer IPs)
  migration steps without warning the user first (e.g. "run again with --yes-migrate-x
  to perform the migration").
* Pin the installer to a specific Kubermatic version, i.e. to choose a different version
  a different installer release must be downloaded.
* Use the installer during E2E tests.

### Non-Goals

* Downgrades.
* Interactive CLI/ncurses interface.
* Migrate manual installations to be installer-managed.
* Migrate Helm-based installations to the Kubermatic Operator.

### Implementation

The installation generally done sequentially. First experiments with doing it in parallel
tasks yielded no good results:

* It was difficult to print an understandable log for the user. Keeping the user informed
  about what is happening to their cluster is important.
* Error handling was difficult, especially in longer running parallel Helm invocations
  and dependent things.
* It makes reasoning when writing migration steps much harder if you have to very
  carefully plan your migration code in all the goroutines that run during an upgrade.

Having sequential steps made the code much easier to read and maintain. It sacrifices some
speed for ease of use.

During `./installer deploy`, the chosen "stack" is installed. By default (and currently
because it's the only one) this is the "kubermatic" stack. The stack then does whatever
it needs to run. For Kubermatic this means

* setting up a "kubermatic-fast" StorageClass; this is a leftover from very early installer
  versions and requires us to know the cloud provider. This requires yet another setting
  to be made from the user, so maybe we should instead just inform the user that they need
  to create a StorageClass?
* installing cert-manager, nginx-ingress-controller, Dex and the Kubermatic Operator
  Helm charts. "Installing" can also mean an upgrade. If a failed release is encountered,
  the release is purged first, otherwise another install with Helm is impossible. If the
  release in the cluster is identical to the Helm chart that is shipped with the installer,
  nothing happens unless the user runs the installer with `--force`.
* `kubectl apply` the given `KubermaticConfiguration`.

If anything goes wrong during this, the installation is aborted. The user then can fix
the problem and re-run the installer.

It would be possible to also manage Seeds while we're at it, but the win between having to
explain to the user that the installer can also install the seeds, vs. just making them
`kubectl apply` them is questionable.

Helm is not easy to embed, so I chose to just shell out to calling a Helm binary. As the
installer is delivered in a Docker image (so it can bundle the Helm charts), this seemed
a simple choice. The interaction with Helm goes through a super simple interface, so it
would be possible to swap out the implementation later.

The installer uses Logrus because it was easier to make it print nice-looking, well-indented
logs. Doing the same with Zap is very painful.

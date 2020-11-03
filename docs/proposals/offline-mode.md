# Offline modus

**Author:** Alvaro Aleman (@alvaroaleman)

**Status**: Draft

## Motivation and background

In order to allow customers do use Kubermatic in an environment that has no access to the Internet we must
find all places where Kubermatic downloads stuff from the Internet, add configuration options, test and document
those.

## Implementation

machine-controller:

* Ubuntu: Download url for Kubernetes binaries must be configurable
* Container Linux: Download url for `hyperkube` image must be configurable
* CentOS/RHEL: Repo url must be configurable
* All: An optional config option for `--pod-infra-container` must be exposed

Kubermatic:

* Making the base repository for all images in all charts configurable (Currently 15 components)
* Writing a script that downloads all required images, retags them and uploads them to a private registry
* Add flag to the `cluster-controller` to set just the repository for all images
* Provide an easy/convenient way to have an image pull secret available on all nodes
* Allow dex to be configured to either use a customer-provided IDP or static user definitions
* Write an e2e test for this:
    * Create a kubeadm seed + master cluster, during this phase Internet may be reachable
    * Execute the script that downloads and retags the images, during this phase Internet may be reachable
    * Deploys Kubermatic master cluster, during this time Internet must not be available
    * Deploys a custmer cluster, during this time Internet must not be available

## Tasks and efforts

* machine-controller: 0.5d
* collecting all images and writing a script for downloading, retagging and uploading: 0.5d
* Proving a convenient way to set just the repo for all images: 0.25d
* Make certificates for dex configurable: 2h
* Write an e2e testing script, verify it, fixup reamining issues: 4d

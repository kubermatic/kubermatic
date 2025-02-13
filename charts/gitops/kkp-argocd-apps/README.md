# KKP Components management via GitOps

> This is a Alpha version of Kubermatic management via GitOps which can become the default way to manage KKP in future. But this feature can also change significantly and in backward incompatible ways. **Please use this setup in production at your own risk.**

For KKP to function effectively, we need to install a bunch of components like seed-mla (monitoring logging and alerting stack), minio, nginx-ingress-controller, user-cluster-mla stack, etc.

Using a GitOps tool to manage these seed components can be very useful. This folder offers a slightly opinionated tooling to achieve the same. The workflow to get this setup would be like below:

1. Install KKP using kkp-installer. (optionally, skip-charts for dex, nginx and cert-manager. See note below.)
1. (Optionally) Setup individual seed clusters. Seed could be master-seed or standalone seed. 
1. Install ArgoCD as helm-chart in each seed that you want to managed via GitOps.
1. Install content of this folder as helm-chart in each seed to deploy various compoentns in each seed. Take a look at [values.yaml](./values.yaml) for various customizations possible for customizing what gets installed in each seed.

> Note: If you are deploying this on master-seed and choosing to deploy Cert-manager, Nginx Ingress controller and Dex to be managed by ArgoCD, then remember to remove installation of them via `kubermatic-installer` via `--skip-charts='cert-manager,nginx-ingress-controller,dex'`

This helm-chart is an opinionated view on how the customization files are stored. If your customization files are stored in different naming convention, please look at the [_helper templates](./templates/_helpers.tpl) where the path are generated and adjust them as necessary.

Currently, this helm chart sets up ArgoCD `Applications` for various KKP components via Git Repo based helm-charts. So we will have 2 sources for helm charts..
1. The Kubermatic Git repo charts
1. Your installation specific local repo store `values.yaml` for your installation specific customizations. 

See below for the folder structure of your local repo structure. As mentioned above, if your directory structure is different, you would need to adjust the functions defined in `_helpers.tpl` for generating ArgoCD Applications properly.

You can use `helm template` command to check if the generated ArgoCD Application definitions look fine or not.

## Folder and File structure:

The current helm-chart templates assume below file structure for your customization repository:

```shell
├── Makefile
├── <environment> # All files for the given environment e.g. dev
│   ├── values.yaml # environment level common values
│   ├── clusterIssuer.yaml
│   ├── settings
│   │   ├── 00_kubermaticsettings.yaml
│   │   └── seed-cluster-<seed1>.yaml
│   │   └── seed-cluster-<seed2>.yaml
│   ├── common   # any files common across seeds in the given environment
│   │   ├── custom-ca-bundle.yaml 
│   │   └── sc-kubermatic-fast.yaml
│   ├── <seed1>  # seed specific files
│   │   ├── argoapps-values.yaml  # This is where we control what ArgoCD apps to get installed in given seed.
│   │   ├── values-usermla.yaml  # customize user-cluster mla stack in this file.
│   │   └── values.yaml # customize seed stack's values e.g. minio, seed prometheus, etc.
│   └── <master-seed>  # master specific files
│       ├── k8cConfig.yaml # master seed needs kubermatic-configuration yaml as well.
│       ├── seed-kubeconfig-secret-<seed1>.yaml  # secret to hold kubeconfig for each seed.
│       ├── seed-kubeconfig-secret-<seed2>.yaml  # secret to hold kubeconfig for each seed.
│   │   ├── argoapps-values.yaml  # This is where we control what ArgoCD apps to get installed in given seed.
│   │   ├── values-usermla.yaml  # customize user-cluster mla stack in this file.
│   │   └── values.yaml # customize seed stack's values e.g. minio, seed 
├── <environment2>     # Similar directory structure as above but for other environment e.g. PROD
└── values-argocd.yaml  # argoCD helm values
```

## Deploy
Check the [Makefile](./proposed-dir-structure/Makefile) on getting some ideas on how to deploy these.

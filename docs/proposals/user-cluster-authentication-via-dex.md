**User Cluster Authentication via Dex**

**Author**: Henrik Schmidt

**Status**: Proposal

*short description of the topic e.g.*
Currently user's of kubermatic authenticate via Dex against the kubermatic dashboard, to manage their clusters but they need to download a special created kubeconfig for each cluster to use it.
This proposal aims to describe all needed tasks that need to be done, so users can use Dex to authenticate against their clusters.

## Motivation and Background

To enable some degree of multi tenancy for user clusters, it must be possible to login via different identities against the same cluster.
This is not possible at the moment as each user cluster only offers a single Kubeconfig which contains a token with a predefined admin identity.   
As kubernetes supports openid-connect providers for authentication, we will configure it to use the Dex instance we have in each master cluster.

## User flow

A user(`Henrik`) creates a new project called `testing`. Within the project `testing` a new cluster(`test-A`) will be created.
The user(`Henrik`) wants to share a specific cluster with his coworkers. 
The coworkers though should not have admin access to the cluster - thus sharing the admin kubeconfig is not an option.
 
Instead each coworker should receive a URL which they can use to retrieve a kubeconfig for the cluster.
When accessing the URL, the coworkers need to login via Dex and after successful authentication, the kubeconfig will be downloaded.
With this kubeconfig the user can access the cluster `test-A`.

When the coworkers access the cluster, they will have no permission to read anything in the cluster. 
`Henrik` must manually manage RBAC objects which will grant the coworkers permission.  

## Implementation

This feature will be made available via a feature-flag. Only when this flag is enabled, the feature will be used.

### API server deployment

The kubernetes API server of each user cluster needs to be configured to accept our Dex instance for authentication.
For this the following flags must be configured:
```
# The URL where Dex is available
--oidc-issuer-url=https://auth.example.com/dex
# The client ID we configured in dex. Kubernetes will compare this to the `aud` field
# in any bearer token from Dex before accepting it.
--oidc-client-id=kubermatic
# Since Dex is configured with TLS, add the CA cert to initiate trust
--oidc-ca-file=/etc/kubernetes/ssl/dex-ca.pem
# The claim field to identify users. For us this means users are granted the username
--oidc-username-claim=email
```

### Dashboard

The dashboard should show a button, next to the download-kubeconfig button, which will open up a modal,
explaining how to share the cluster using OpenID-Connect.
As we implement this via a feature flag, a new config option must be introduced in the dashboard config.

### Client side
As `kubectl` has no way of doing the initial OpenID-Connect flow(Refreshing works), the retrieval of the initial token must be done by a dedicated tool.  

In our case we'll implement the initial retrieval on the kubermatic API.
A dedicated endpoint which takes the cluster name, redirect to Dex, accept the callback and builds the kubeconfig.

The kubeconfig should look like:
```yaml
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURHakNDQWdLZ0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRc0ZBREErTVR3d09nWURWUVFERXpOeWIyOTAKTFdOaExqSnFiWGRxYzJ0bmVtNHVaWFZ5YjNCbExYZGxjM1F6TFdNdVpHVjJMbXQxWW1WeWJXRjBhV011YVc4dwpIaGNOTVRneE1ERTJNRGcwTnpJMVdoY05Namd4TURFek1EZzBOekkxV2pBK01Ud3dPZ1lEVlFRREV6TnliMjkwCkxXTmhMakpxYlhkcWMydG5lbTR1WlhWeWIzQmxMWGRsYzNRekxXTXVaR1YyTG10MVltVnliV0YwYVdNdWFXOHcKZ2dFaU1BMEdDU3FHU0liM0RRRUJBUVVBQTRJQkR3QXdnZ0VLQW9JQkFRQzlYYW9LNjhueHlOc2tSOHR6NExXVApHaldQK0x4bGtibEFMVkdSNkR0OTh4YVBsVEpVcWJFTlR1S2ZvQ0k0S3JDVWxMMEl0c2I3eG9zSHRvb2tGM0hkCmpPbDVtUHROY0NCN1hKS1BSWkRvVlNValVwcEo4bThWTjU2VlVrYjVocEF0dnFzbDQ2MEE4dGp1c2RGU3Q5V28KUUxTZjhTRUE1VGVWNThpZ3RlMHppVTByS2h6WmkxOHBRTHRuU3NhbjVkcmkvaXllb1puaUFxMEh4Z0grZzcxTAp5QXBuYi93NTFCY05Tbk1uMlh3NW4yQnNXTTdZcm1KZmlUb3lBS004cVZnMFlwWTFKNkNzZy8xeEpkR0VhY1ByCjFReFN3WUVocmtCUVU0UG5GM09DSXY5QzZTWHQrYnNlem1XSm1pL0FkcWNzL3ljSGFib0FmYktQRUF1UFc1dlAKQWdNQkFBR2pJekFoTUE0R0ExVWREd0VCL3dRRUF3SUNwREFQQmdOVkhSTUJBZjhFQlRBREFRSC9NQTBHQ1NxRwpTSWIzRFFFQkN3VUFBNElCQVFBdTdqMTIxZTVPR05JbEM1dWhzNFVjTUFvblViZ0FyUU1Db21hUFVwSkwwWllwCllvTkdJZ2l6QWJVeUFuR0RTTzhaQlJPSTV4KytxSlhxdFkxZjRWaUhaaEtzODU5ZExoZGE5M1Vic0ozYlVDWWMKK2FrNXFqWjB1Q1JaRjBCeTMwaGRNeDZHK0dNN2lneERTK0xkUmwyZmU0MGx4NDEyemk2K1hUWmhsMmR3QWpXbQo4MXRYUlZnUHIyMlVzNEVORFd2OEVTSDBOYzB0VHNUMHozSWQrMUtMK0c3OWtTS3diS2NETTZ0Ym9FVE43WW1RCkhuYlZ4bTRLZ2Rqb1NwN2dxQ1MwRjhaSHU0bVJRMnVVdUxSamxaZTh3N1krZmRtRVNpMllmMEI0UEN4NHdNOUYKSC80Q2lnaHg4V1JhZlh4WGNBQ2U0Q2laVkNMd2NLc0NFdk9ScHpBSQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
    server: https://2jmwjskgzn.europe-west3-c.dev.kubermatic.io:31533
  name: 2jmwjskgzn
contexts:
- context:
    cluster: 2jmwjskgzn
    user: default
  name: default
current-context: default
kind: Config
preferences: {}
users:
- name: henrik@loodse.com
  user:
    auth-provider:
      config:
        client-id: kubermatic
        client-secret: <SOME_SUPER_SECRET_STRING>
        id-token: <TOKEN_RETRIEVED_FROM_THE_EXAMPLE_APP>
        idp-issuer-url: https://auth.example.com/dex
        refresh-token: <REFRESH_TOKEN_RETRIEVED_FROM_THE_EXAMPLE_APP>
      name: oidc
```

#### Example code via a CLI tool:  
Dex provides a CLI tool to execute the initial flow: https://github.com/coreos/dex/tree/master/cmd/example-app
It must be called via:
```bash
./example-app -client-id=kubermatic -client-secret=<SOME_SUPER_SECRET_STRING> -issuer=https://auth.example.com/dex -issuer-root-ca=ca.pem
```

## Task & effort:
* Add initial feature-flag handling - 1d
* Configure the kubernetes API server of each cluster to use OpenID-Connect - 0.5d
* Implement an endpoint which will generate the Kubeconfig - 1d
* Write relevant documentation describing on how to create basic RBAC manifests to grant others permission - 0.5d
* Implement modal in Dashboard + feature flag(config variable) - 1d

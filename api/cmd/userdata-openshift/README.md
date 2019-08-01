# OpenShift userdata

This userdata plugin generates a cloud-init script which can be used to provision a CentOS node to be a OpenShift worker node.

## Requirements 

OpenShift worker nodes require a "bootstrap token" to start.
That token must be a ServiceAccount token which has the permissions to create CSRs.
In a default OpenShift installation, that token is located at: `openshift-infra/node-bootstrapper`
Therefore the machine-controller must be started with `-bootstrap-token-service-account-name="openshift-infra/node-bootstrapper"`

OpenShift has a controller to automatically approve CSRs but that only works with machines from the OpenShift machine-controller.
Thus we require a custom controller.  

Sidenote:
A new node creates 2 CSRs:
- Kubelet client certificate
- Kubelet serving certificate

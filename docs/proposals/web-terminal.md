**User Cluster WEB terminal**

**Author**: Lukasz Zajaczkowski  (zreigz@gmail.com)

**Status**: Draft proposal. Prototype in progress

## Goals

The KKP user have a WEB terminal interface where can access own clusters via CLI directly inside the browser. The Visual
Web Terminal, located in the cluster details page. It allows running a commands that return Kubernetes resources. The 
terminal has also other tools by default like helm, git, curl, wget.

## Implementation
 - the user opens the WEB terminal and is redirected to the OIDC login page to get tokens for the kubeconfig
 - the OIDC kubeconfig is stored in secret
 - the new WEB terminal deployment is created for the user and the kubeconfig secret is mounted to the Pod.
 - when Pod is ready we open the WEB socket session and exec to the Pod
 - we set an expiration time for the deployment at 30min. Five minutes before when session is still open user has to extend
   the time for another 30min (there will be some pop-up and the user has to agree to extend the expiration time explicitly).

 
## Requirements
 - Every terminal session is unique for the user cluster RBAC user assigned to the cluster role
 - For security reason the Pod with command line tools is deployed on the user cluster
 - The kubeconfig needs to run as the login dex user (we don't want to expose admin kubeconfig).


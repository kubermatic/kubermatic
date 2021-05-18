# Alertmanager Authorization Server

A gRPC server which is used in user MLA (Monitoring, Logging, and Alerting) stack for authorizing Cortex 
Alertmanager UI.

## Releasing
A manual release needs to be done if any changes are made in `cmd/alertmanager-authorization-server`:

```
# Go to the Kubermatic project directory
cd <kubermatic_project_directory>

# Increase the TAG variable in the script
vim hack/release-alertmanager-authorization-server-image.sh

# Build and push docker images with new version tag
./hack/release-alertmanager-authorization-server-image.sh
```

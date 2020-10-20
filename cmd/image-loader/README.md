# image-loader

A little utility that downloads all required Docker images for KKP, retags then
and pushes them to a local registry.

If you're using the KKP Operator and a KubermaticConfiguration, run the utility
with `-configuration-file YOUR_FILE.yaml`, otherwise specify the path to the
`versions.yaml` from the legacy Helm chart via `-versions-file VERSIONS.yaml`.

Synopsis: `image-loader -registry registry.corp.com -configuration-file YOUR_FILE.yaml`

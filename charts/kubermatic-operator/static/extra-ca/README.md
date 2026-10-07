# Additional CA certificates

Every `*.pem` file in this directory is appended to the CA bundle that the
`kubermatic-operator` chart renders into the `ca-bundle` ConfigMap. KKP trusts
that bundle everywhere: the operator itself, every seed and every user cluster.

Use this directory to add your own CA certificates without touching
`../ca-bundle.pem`, which ships with the chart and is replaced wholesale
whenever the bundle is refreshed from <https://curl.se/ca/cacert.pem>.

## Usage

After extracting the KKP release archive, copy your PEM-encoded certificates
here, then install as usual:

```bash
cp /path/to/corporate-ca.pem charts/kubermatic-operator/static/extra-ca/
kubermatic-installer deploy kubermatic-master --config kubermatic.yaml --helm-values values.yaml
```

Files placed here are **not** carried over when you extract the next release
archive; copy them again after an upgrade.

## Limitations

This directory is only resolvable when the chart is installed from an unpacked
charts directory, which is what `kubermatic-installer` does. If you pull the
chart from a Helm repository or render it with a GitOps tool such as Argo CD,
the files are not available and the bundle is rendered without them. Configure
`caBundle.certificates` or `caBundle.additionalCertificates` in your Helm values
instead; passing `--ca-bundle` or `--additional-ca-bundle` to
`kubermatic-installer deploy` writes such a values file for you.

Note that none of this governs how the installer itself reaches a registry.
`kubermatic-installer mirror-images` copies images using the trust store of the
machine it runs on, so a registry signed by a private CA needs that CA installed
on the host (or `--insecure` to skip verification).

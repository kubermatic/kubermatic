# Helm test data

This directory contains test data for the integration tests.

## PKI

The three PEM files (crt.pem, key.pem, rootca.crt) are copied from Helm:

https://github.com/helm/helm/tree/v3.15.2/testdata

See https://github.com/kubermatic/kubermatic/pull/13406 for more information
about why these files have been copied here.

Once they expire in 2032, replace them with freshly generated certs.

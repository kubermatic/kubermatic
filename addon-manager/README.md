# Building
```bash
docker build -t kubermatic/addon-manager .
docker tag kubermatic/addon-manager kubermatic/addon-manager:<Kubernetes Version>
docker push kubermatic/addon-manager:<Kubernetes Version>
```

## Versions
The Version should be the kubernetes version.

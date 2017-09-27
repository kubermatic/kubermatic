# Building
```bash
docker build -t kubermatic/addon-manager .
docker tag kubermatic/addon-manager kubermatic/addon-manager:v1.7.0
docker push kubermatic/addon-manager:v1.7.0
```

## Versions
The Version should be the kubernetes version.

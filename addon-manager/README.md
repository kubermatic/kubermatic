# Building
Drone will automatically build the addon-container 

```bash
docker build -t kubermatic/addon-manager .
docker tag kubermatic/addon-manager kubermatic/addon-manager:<Version>
docker push kubermatic/addon-manager:<Version>
```

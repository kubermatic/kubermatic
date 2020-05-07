module github.com/kubermatic/kubermatic/api

go 1.14

require (
	code.cloudfoundry.org/go-pubsub v0.0.0-20180503211407-becd51dc37cb
	github.com/Azure/azure-sdk-for-go v38.0.0+incompatible
	github.com/Azure/go-autorest/autorest v0.9.5
	github.com/Azure/go-autorest/autorest/azure/auth v0.4.2
	github.com/Azure/go-autorest/autorest/to v0.3.0
	github.com/Masterminds/semver v1.4.2
	github.com/Masterminds/sprig v2.17.1+incompatible
	github.com/aliyun/alibaba-cloud-sdk-go v0.0.0-20190828035149-111b102694f9
	github.com/apoydence/onpar v0.0.0-20200406201722-06f95a1c68e8 // indirect
	github.com/aws/aws-sdk-go v1.27.4
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/coreos/locksmith v0.6.2
	github.com/cristim/ec2-instances-info v0.0.0-20190708120723-b53a9860c46d
	github.com/digitalocean/godo v1.7.3
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/envoyproxy/go-control-plane v0.9.4
	github.com/evanphx/json-patch v4.5.0+incompatible
	github.com/fsnotify/fsnotify v1.4.9
	github.com/ghodss/yaml v1.0.0
	github.com/go-ini/ini v1.55.0 // indirect
	github.com/go-kit/kit v0.9.0
	github.com/go-logr/zapr v0.1.1
	github.com/go-openapi/errors v0.19.4
	github.com/go-openapi/runtime v0.19.12
	github.com/go-openapi/strfmt v0.19.5
	github.com/go-openapi/swag v0.19.8
	github.com/go-openapi/validate v0.19.7
	github.com/go-swagger/go-swagger v0.23.0
	github.com/go-test/deep v1.0.4
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.3.5
	github.com/gophercloud/gophercloud v0.2.1-0.20190626201551-2949719e8258
	github.com/gorilla/handlers v1.4.2
	github.com/gorilla/mux v1.7.3
	github.com/gorilla/securecookie v1.1.1
	github.com/gorilla/websocket v1.4.1
	github.com/heptiolabs/healthcheck v0.0.0-20180807145615-6ff867650f40
	github.com/hetznercloud/hcloud-go v1.15.1
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/iancoleman/strcase v0.0.0-20190422225806-e506e3ef7365
	github.com/jetstack/cert-manager v0.11.0
	github.com/kubermatic/machine-controller v1.13.2
	github.com/minio/minio-go v6.0.14+incompatible
	github.com/nelsam/hel v2.3.1+incompatible // indirect
	github.com/oklog/run v1.0.0
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.1 // indirect
	github.com/packethost/packngo v0.1.1-0.20190410075950-a02c426e4888
	github.com/pkg/errors v0.9.1
	github.com/pmezard/go-difflib v1.0.0
	github.com/poy/onpar v0.0.0-20200406201722-06f95a1c68e8 // indirect
	github.com/prometheus/client_golang v1.4.1
	github.com/robfig/cron v1.2.0
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.4.0
	github.com/urfave/cli v1.20.0
	github.com/vmware/govmomi v0.22.2
	go.uber.org/zap v1.13.0
	golang.org/x/crypto v0.0.0-20200311171314-f7b00557c8c4
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/tools v0.0.0-20200313205530-4303120df7d8
	google.golang.org/api v0.10.0
	google.golang.org/grpc v1.27.1
	gopkg.in/square/go-jose.v2 v2.4.1
	gopkg.in/yaml.v2 v2.2.8
	gopkg.in/yaml.v3 v3.0.0-20190905181640-827449938966
	k8s.io/api v0.17.2
	k8s.io/apiextensions-apiserver v0.16.4
	k8s.io/apimachinery v0.17.2
	k8s.io/autoscaler v0.0.0-20190218140445-7f77136aeea4 // git digest for VPA v0.4.0
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.17.1
	k8s.io/klog v1.0.0
	k8s.io/kube-aggregator v0.16.4
	k8s.io/metrics v0.16.4
	k8s.io/test-infra v0.0.0-20200220102703-18fae0a00a2c
	k8s.io/utils v0.0.0-20200124190032-861946025e34
	sigs.k8s.io/controller-runtime v0.4.0
	sigs.k8s.io/yaml v1.2.0
)

replace (
	k8s.io/api => k8s.io/api v0.16.4
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.16.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.16.4
	k8s.io/client-go => k8s.io/client-go v0.16.4
	k8s.io/code-generator => k8s.io/code-generator v0.16.4
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.16.4
	k8s.io/kubelet => k8s.io/kubelet v0.16.4
	k8s.io/metrics => k8s.io/metrics v0.16.4
)

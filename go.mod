module k8c.io/kubermatic/v2

go 1.14

require (
	code.cloudfoundry.org/go-pubsub v0.0.0-20180503211407-becd51dc37cb
	github.com/Azure/azure-sdk-for-go v38.0.0+incompatible
	github.com/Azure/go-autorest/autorest v0.9.6
	github.com/Azure/go-autorest/autorest/azure/auth v0.4.2
	github.com/Azure/go-autorest/autorest/to v0.3.0
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/Masterminds/sprig v2.17.1+incompatible
	github.com/aliyun/alibaba-cloud-sdk-go v1.61.733
	github.com/apoydence/onpar v0.0.0-20200406201722-06f95a1c68e8 // indirect
	github.com/aws/aws-sdk-go v1.27.4
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/coreos/locksmith v0.6.2
	github.com/cristim/ec2-instances-info v0.0.0-20201110114654-2dfcc09f67d4
	github.com/digitalocean/godo v1.54.0
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/envoyproxy/go-control-plane v0.9.7
	github.com/evanphx/json-patch v4.9.0+incompatible
	github.com/ghodss/yaml v1.0.0
	github.com/go-ini/ini v1.55.0 // indirect
	github.com/go-kit/kit v0.10.0
	github.com/go-logr/logr v0.2.1
	github.com/go-logr/zapr v0.2.0
	github.com/go-openapi/errors v0.19.6
	github.com/go-openapi/runtime v0.19.20
	github.com/go-openapi/strfmt v0.19.5
	github.com/go-openapi/swag v0.19.9
	github.com/go-openapi/validate v0.19.10
	github.com/go-swagger/go-swagger v0.25.0
	github.com/go-test/deep v1.0.4
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.4.2
	github.com/google/go-cmp v0.5.1 // indirect
	github.com/gophercloud/gophercloud v0.2.1-0.20190626201551-2949719e8258
	github.com/gorilla/handlers v1.4.2
	github.com/gorilla/mux v1.7.3
	github.com/gorilla/securecookie v1.1.1
	github.com/gorilla/websocket v1.4.2
	github.com/hashicorp/go-multierror v1.0.0
	github.com/heptiolabs/healthcheck v0.0.0-20180807145615-6ff867650f40
	github.com/hetznercloud/hcloud-go v1.23.1
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/iancoleman/strcase v0.0.0-20190422225806-e506e3ef7365
	github.com/imdario/mergo v0.3.10 // indirect
	github.com/jetstack/cert-manager v0.11.0
	github.com/kubermatic/machine-controller v1.20.2
	github.com/minio/minio-go v6.0.14+incompatible
	github.com/nelsam/hel v0.0.0-20200611165952-2d829bae0c66 // indirect
	github.com/oklog/run v1.1.0
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.1
	github.com/open-policy-agent/frameworks/constraint v0.0.0-20200803193800-bcb6432d79b7
	github.com/packethost/packngo v0.1.1-0.20190410075950-a02c426e4888
	github.com/pkg/errors v0.9.1
	github.com/pmezard/go-difflib v1.0.0
	github.com/poy/onpar v0.0.0-20200406201722-06f95a1c68e8 // indirect
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/common v0.11.1 // indirect
	github.com/robfig/cron v1.2.0
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	github.com/urfave/cli v1.22.4
	github.com/vmware/govmomi v0.22.2
	go.etcd.io/etcd/v3 v3.3.0-rc.0.0.20200728214110-6c81b20ec8de
	go.uber.org/zap v1.16.0
	golang.org/x/crypto v0.0.0-20201002170205-7f63de1d35b0
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sys v0.0.0-20200819171115-d785dc25833f // indirect
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	golang.org/x/tools v0.0.0-20200717024301-6ddee64345a6
	gomodules.xyz/jsonpatch/v2 v2.1.0 // indirect
	google.golang.org/api v0.15.0
	google.golang.org/grpc v1.27.1
	gopkg.in/fsnotify.v1 v1.4.7
	gopkg.in/square/go-jose.v2 v2.5.1
	gopkg.in/yaml.v2 v2.3.0
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
	k8s.io/api v0.19.4
	k8s.io/apiextensions-apiserver v0.19.3
	k8s.io/apimachinery v0.19.4
	k8s.io/autoscaler v0.0.0-20190218140445-7f77136aeea4 // git digest for VPA v0.4.0
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.19.3
	k8s.io/klog v1.0.0
	k8s.io/klog/v2 v2.3.0 // indirect
	k8s.io/kube-aggregator v0.16.4
	k8s.io/kubectl v0.19.4
	k8s.io/metrics v0.19.4
	k8s.io/test-infra v0.0.0-20200220102703-18fae0a00a2c
	k8s.io/utils v0.0.0-20200731180307-f00132d28269
	sigs.k8s.io/controller-runtime v0.6.4
	sigs.k8s.io/yaml v1.2.0
)

replace (
	// prevent 2.0.0 dependency because it's a broken release
	gomodules.xyz/jsonpatch/v2 => gomodules.xyz/jsonpatch/v2 v2.1.0
	k8s.io/api => k8s.io/api v0.19.3
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.3
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.3
	k8s.io/client-go => k8s.io/client-go v0.19.3
	k8s.io/code-generator => k8s.io/code-generator v0.19.3
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.19.3
	k8s.io/kubelet => k8s.io/kubelet v0.19.3
	k8s.io/metrics => k8s.io/metrics v0.19.3
)

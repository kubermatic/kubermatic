module k8c.io/kubermatic/v2

go 1.16

require (
	code.cloudfoundry.org/go-pubsub v0.0.0-20180503211407-becd51dc37cb
	github.com/Azure/azure-sdk-for-go v57.0.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.20
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.8
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/Masterminds/sprig/v3 v3.1.0
	github.com/aliyun/alibaba-cloud-sdk-go v1.61.751
	github.com/anexia-it/go-anxcloud v0.3.8
	github.com/apoydence/onpar v0.0.0-20200406201722-06f95a1c68e8 // indirect
	github.com/aws/aws-sdk-go v1.37.22
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/coreos/locksmith v0.6.2
	github.com/cristim/ec2-instances-info v0.0.0-20201110114654-2dfcc09f67d4
	github.com/digitalocean/godo v1.65.0
	github.com/docker/distribution v2.7.1+incompatible
	github.com/envoyproxy/go-control-plane v0.9.7
	github.com/evanphx/json-patch v4.11.0+incompatible
	github.com/ghodss/yaml v1.0.0
	github.com/go-ini/ini v1.62.0 // indirect
	github.com/go-kit/kit v0.10.0
	github.com/go-logr/logr v0.4.0
	github.com/go-logr/zapr v0.4.0
	github.com/go-macaron/session v1.0.2 // indirect
	github.com/go-openapi/errors v0.20.0
	github.com/go-openapi/runtime v0.19.27
	github.com/go-openapi/strfmt v0.20.1
	github.com/go-openapi/swag v0.19.15
	github.com/go-openapi/validate v0.20.2
	github.com/go-swagger/go-swagger v0.27.0
	github.com/go-test/deep v1.0.7
	github.com/google/go-cmp v0.5.6
	github.com/gophercloud/gophercloud v0.20.0
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/securecookie v1.1.1
	github.com/gorilla/websocket v1.4.2
	github.com/grafana/grafana v6.1.6+incompatible
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hetznercloud/hcloud-go v1.32.0
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/inconshreveable/log15 v0.0.0-20201112154412-8562bdadbbac // indirect
	github.com/jetstack/cert-manager v1.1.0
	github.com/kubermatic/grafanasdk v0.9.10
	github.com/kubermatic/machine-controller v1.35.1
	github.com/minio/minio-go v6.0.14+incompatible
	github.com/minio/minio-go/v7 v7.0.13
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.14.0
	github.com/open-policy-agent/frameworks/constraint v0.0.0-20210802220920-c000ec35322e
	github.com/open-policy-agent/gatekeeper v0.0.0-20201111000257-4450f08fa95e
	github.com/packethost/packngo v0.19.0
	github.com/pkg/errors v0.9.1
	github.com/pmezard/go-difflib v1.0.0
	github.com/poy/onpar v1.0.1 // indirect
	github.com/prometheus/client_golang v1.11.0
	github.com/robfig/cron v1.2.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	github.com/teris-io/shortid v0.0.0-20201117134242-e59966efd125 // indirect
	github.com/urfave/cli v1.22.5
	github.com/vmware/govmomi v0.23.1
	go.etcd.io/etcd/v3 v3.3.0-rc.0.0.20200728214110-6c81b20ec8de
	go.uber.org/zap v1.18.1
	golang.org/x/crypto v0.0.0-20210513164829-c07d793c2f9a
	golang.org/x/oauth2 v0.0.0-20210514164344-f6687ab2804c
	golang.org/x/tools v0.1.2
	gomodules.xyz/jsonpatch/v2 v2.2.0
	google.golang.org/api v0.36.0
	google.golang.org/genproto v0.0.0-20210602131652-f16073e35f0c
	google.golang.org/grpc v1.38.0
	google.golang.org/protobuf v1.27.1
	gopkg.in/fsnotify.v1 v1.4.7
	gopkg.in/gcfg.v1 v1.2.3
	gopkg.in/square/go-jose.v2 v2.5.1
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	k8s.io/api v0.21.3
	k8s.io/apiextensions-apiserver v0.21.3
	k8s.io/apimachinery v0.21.3
	k8s.io/autoscaler v0.0.0-20190218140445-7f77136aeea4 // git digest for VPA v0.4.0
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.21.3
	k8s.io/klog v1.0.0
	k8s.io/kube-aggregator v0.21.3
	k8s.io/kubectl v0.21.3
	k8s.io/metrics v0.21.3
	k8s.io/test-infra v0.0.0-20210826180422-39483c498f0f
	k8s.io/utils v0.0.0-20210722164352-7f3ee0f31471
	sigs.k8s.io/controller-runtime v0.9.6
	sigs.k8s.io/yaml v1.2.0
)

replace (
	// etcd.v3 needs an old version for the google.golang.org/grpc/naming package, which got removed in grpc 1.30+
	google.golang.org/grpc => google.golang.org/grpc v1.29.1
	k8s.io/api => k8s.io/api v0.21.3
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.21.3
	k8s.io/apimachinery => k8s.io/apimachinery v0.21.3
	k8s.io/client-go => k8s.io/client-go v0.21.3
	k8s.io/code-generator => k8s.io/code-generator v0.21.3
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.21.3
	k8s.io/kubelet => k8s.io/kubelet v0.21.3
	k8s.io/metrics => k8s.io/metrics v0.21.3
)

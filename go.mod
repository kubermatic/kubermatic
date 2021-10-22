module k8c.io/kubermatic/v2

go 1.17

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
	github.com/envoyproxy/go-control-plane v0.9.9-0.20210217033140-668b12f5399d
	github.com/evanphx/json-patch v4.11.0+incompatible
	github.com/ghodss/yaml v1.0.0
	github.com/go-ini/ini v1.62.0 // indirect
	github.com/go-kit/kit v0.10.0
	github.com/go-logr/logr v0.4.0
	github.com/go-logr/zapr v0.4.0
	github.com/go-openapi/errors v0.20.0
	github.com/go-openapi/runtime v0.19.27
	github.com/go-openapi/strfmt v0.20.1
	github.com/go-openapi/swag v0.19.15
	github.com/go-openapi/validate v0.20.2
	github.com/go-swagger/go-swagger v0.27.0
	github.com/go-test/deep v1.0.7
	github.com/google/go-cmp v0.5.6
	github.com/gophercloud/gophercloud v0.20.0
	github.com/gopherjs/gopherjs v0.0.0-20190430165422-3e4dfb77656c // indirect
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/securecookie v1.1.1
	github.com/gorilla/websocket v1.4.2
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hetznercloud/hcloud-go v1.32.0
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/jetstack/cert-manager v1.1.0
	github.com/kubermatic/grafanasdk v0.9.11
	github.com/kubermatic/machine-controller v1.36.0
	github.com/minio/minio-go v6.0.14+incompatible
	github.com/minio/minio-go/v7 v7.0.13
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.15.0
	github.com/open-policy-agent/frameworks/constraint v0.0.0-20210802220920-c000ec35322e
	github.com/open-policy-agent/gatekeeper v0.0.0-20201111000257-4450f08fa95e
	github.com/packethost/packngo v0.19.0
	github.com/pkg/errors v0.9.1
	github.com/pmezard/go-difflib v1.0.0
	github.com/poy/onpar v1.0.1 // indirect
	github.com/prometheus/client_golang v1.11.0
	github.com/robfig/cron v1.2.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	github.com/urfave/cli v1.22.5
	github.com/vmware/govmomi v0.23.1
	go.etcd.io/etcd/api/v3 v3.5.0
	go.etcd.io/etcd/client/pkg/v3 v3.5.0
	go.etcd.io/etcd/client/v3 v3.5.0
	go.etcd.io/etcd/etcdutl/v3 v3.5.0
	go.uber.org/zap v1.19.0
	golang.org/x/crypto v0.0.0-20210513164829-c07d793c2f9a
	golang.org/x/oauth2 v0.0.0-20210514164344-f6687ab2804c
	golang.org/x/tools v0.1.5
	gomodules.xyz/jsonpatch/v2 v2.2.0
	google.golang.org/api v0.44.0
	google.golang.org/genproto v0.0.0-20210602131652-f16073e35f0c
	google.golang.org/grpc v1.38.0
	google.golang.org/protobuf v1.27.1
	gopkg.in/fsnotify.v1 v1.4.7
	gopkg.in/gcfg.v1 v1.2.3
	gopkg.in/square/go-jose.v2 v2.5.1
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	k8s.io/api v0.22.2
	k8s.io/apiextensions-apiserver v0.22.2
	k8s.io/apimachinery v0.22.2
	k8s.io/apiserver v0.22.2
	k8s.io/autoscaler v0.0.0-20190218140445-7f77136aeea4 // git digest for VPA v0.4.0
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.22.2
	k8s.io/klog v1.0.0
	k8s.io/kube-aggregator v0.21.3
	k8s.io/kubectl v0.21.3
	k8s.io/metrics v0.22.2
	k8s.io/test-infra v0.0.0-20210826180422-39483c498f0f
	k8s.io/utils v0.0.0-20210819203725-bdf08cb9a70a
	sigs.k8s.io/controller-runtime v0.10.1
	sigs.k8s.io/controller-tools v0.7.0
	sigs.k8s.io/yaml v1.2.0
)

require (
	cloud.google.com/go v0.81.0 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.13 // indirect
	github.com/Azure/go-autorest/autorest/azure/cli v0.4.2 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.3.0 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/BurntSushi/toml v0.3.1 // indirect
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/ajeddeloh/go-json v0.0.0-20170920214419-6a2fe990e083 // indirect
	github.com/ajeddeloh/yaml v0.0.0-20170912190910-6b94386aeefd // indirect
	github.com/alecthomas/units v0.0.0-20190924025748-f65c72e2690d // indirect
	github.com/asaskevich/govalidator v0.0.0-20210307081110-f21760c49a8d // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/census-instrumentation/opencensus-proto v0.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.1 // indirect
	github.com/clarketm/json v1.13.4 // indirect
	github.com/cncf/udpa/go v0.0.0-20201120205902-5459f2c99403 // indirect
	github.com/coreos/container-linux-config-transpiler v0.9.0 // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf // indirect
	github.com/coreos/go-systemd/v22 v22.3.2 // indirect
	github.com/coreos/ignition v0.35.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dimchansky/utfbom v1.1.1 // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/emicklei/go-restful v2.11.2+incompatible // indirect
	github.com/envoyproxy/protoc-gen-validate v0.1.0 // indirect
	github.com/fatih/color v1.12.0 // indirect
	github.com/felixge/httpsnoop v1.0.1 // indirect
	github.com/form3tech-oss/jwt-go v3.2.3+incompatible // indirect
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/go-logfmt/logfmt v0.5.0 // indirect
	github.com/go-openapi/analysis v0.20.0 // indirect
	github.com/go-openapi/inflect v0.19.0 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.19.5 // indirect
	github.com/go-openapi/loads v0.20.2 // indirect
	github.com/go-openapi/spec v0.20.3 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/gobuffalo/flect v0.2.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/gofuzz v1.2.1-0.20210504230335-f78f29fc09ea // indirect
	github.com/google/uuid v1.2.0 // indirect
	github.com/googleapis/gax-go/v2 v2.0.5 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/gosimple/slug v1.1.1 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jessevdk/go-flags v1.5.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/jonboulle/clockwork v0.2.2 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.11 // indirect
	github.com/klauspost/cpuid v1.3.1 // indirect
	github.com/kr/pretty v0.2.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/magiconair/properties v1.8.5 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/mattn/go-isatty v0.0.12 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/minio/md5-simd v1.1.0 // indirect
	github.com/minio/sha256-simd v0.1.1 // indirect
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.4.1 // indirect
	github.com/mitchellh/reflectwalk v1.0.1 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/pelletier/go-toml v1.9.3 // indirect
	github.com/pquerna/cachecontrol v0.0.0-20201205024021-ac21108117ac // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.26.0 // indirect
	github.com/prometheus/procfs v0.6.0 // indirect
	github.com/rainycape/unidecode v0.0.0-20150907023854-cb7f23ec59be // indirect
	github.com/rs/xid v1.2.1 // indirect
	github.com/russross/blackfriday/v2 v2.0.1 // indirect
	github.com/shurcooL/sanitized_anchor_name v1.0.0 // indirect
	github.com/spf13/afero v1.6.0 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/viper v1.8.1 // indirect
	github.com/subosito/gotenv v1.2.0 // indirect
	github.com/toqueteos/webbrowser v1.2.0 // indirect
	github.com/vincent-petithory/dataurl v0.0.0-20160330182126-9a301d65acbb // indirect
	github.com/xiang90/probing v0.0.0-20190116061207-43a291ad63a2 // indirect
	go.etcd.io/bbolt v1.3.6 // indirect
	go.etcd.io/etcd/client/v2 v2.305.0 // indirect
	go.etcd.io/etcd/pkg/v3 v3.5.0 // indirect
	go.etcd.io/etcd/raft/v3 v3.5.0 // indirect
	go.etcd.io/etcd/server/v3 v3.5.0 // indirect
	go.mongodb.org/mongo-driver v1.5.1 // indirect
	go.opencensus.io v0.23.0 // indirect
	go.opentelemetry.io/contrib v0.20.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.20.0 // indirect
	go.opentelemetry.io/otel v0.20.0 // indirect
	go.opentelemetry.io/otel/metric v0.20.0 // indirect
	go.opentelemetry.io/otel/trace v0.20.0 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	go4.org v0.0.0-20201209231011-d4a079459e60 // indirect
	golang.org/x/mod v0.4.2 // indirect
	golang.org/x/net v0.0.0-20210525063256-abc453219eb5 // indirect
	golang.org/x/sys v0.0.0-20210817190340-bfb29a6856f2 // indirect
	golang.org/x/term v0.0.0-20210503060354-a79de5458b56 // indirect
	golang.org/x/text v0.3.6 // indirect
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.62.0 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	k8s.io/component-base v0.22.2 // indirect
	k8s.io/gengo v0.0.0-20201214224949-b6c5ce23f027 // indirect
	k8s.io/klog/v2 v2.9.0 // indirect
	k8s.io/kube-openapi v0.0.0-20210527164424-3c818078ee3d // indirect
	k8s.io/kubelet v0.19.4 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.1.2 // indirect
)

replace (
	k8s.io/api => k8s.io/api v0.22.2
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.22.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.22.2
	k8s.io/client-go => k8s.io/client-go v0.22.2
	k8s.io/code-generator => k8s.io/code-generator v0.22.2
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.22.2
	k8s.io/kubectl => k8s.io/kubectl v0.22.2
	k8s.io/kubelet => k8s.io/kubelet v0.22.2
	k8s.io/metrics => k8s.io/metrics v0.22.2
)

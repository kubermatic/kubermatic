package kubernetes

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
	"k8s.io/client-go/1.5/pkg/api/v1"
)

const (
	// RoleLabelKey is the label key set to the value kubermatic-cluster.
	RoleLabelKey = "role"
	// ClusterRoleLabel is the value of the role label of a cluster namespace.
	ClusterRoleLabel = "kubermatic-cluster"
	// DevLabelKey identifies clusters that are only processed by dev cluster controllers.
	DevLabelKey = "dev"
	// DevLabelValue is the value for DevLabelKey.
	DevLabelValue = "true"

	// annotationPrefix is the prefix string of every cluster namespace annotation.
	annotationPrefix = "kubermatic.io/"

	// namePrefix is the prefix string of every cluster namespace name.
	namePrefix = "cluster"

	urlAnnotation               = annotationPrefix + "url"             // kubermatic.io/url
	tokenAnnotation             = annotationPrefix + "token"           // kubermatic.io/token
	customAnnotationPrefix      = annotationPrefix + "annotation-"     // kubermatic.io/annotation-
	cloudAnnotationPrefix       = annotationPrefix + "cloud-provider-" // kubermatic.io/cloud-provider-
	providerAnnotation          = annotationPrefix + "cloud-provider"  // kubermatic.io/cloud-provider
	cloudDCAnnotation           = annotationPrefix + "cloud-dc"        // kubermatic.io/cloud-dc
	phaseTimestampAnnotation    = annotationPrefix + "phase-ts"        // kubermatic.io/phase-ts
	healthAnnotation            = annotationPrefix + "health"          // kubermatic.io/health
	userAnnotation              = annotationPrefix + "user"            // kubermatic.io/user
	humanReadableNameAnnotation = annotationPrefix + "name"            // kubermatic.io/name
	etcdURLAnnotation           = annotationPrefix + "etcd-url"        // kubermatic.io/etcd-url
	rootCAKeyAnnotation         = annotationPrefix + "root-ca-key"     // kubermatic.io/root-ca-key
	rootCACertAnnotation        = annotationPrefix + "root-ca-cert"    // kubermatic.io/root-cert
	apiserverPubSSHAnnotation   = annotationPrefix + "ssh-pub"         // kubermatic.io/ssh-pub

	userLabelKey  = "user"
	nameLabelKey  = "name"
	phaseLabelKey = "phase"

	flannelCIDRADefault = "172.17.0.0/16"
)

// NamespaceName create a namespace name for a given user and cluster.
func NamespaceName(user, cluster string) string {
	return fmt.Sprintf("%s-%s", namePrefix, cluster)
}

// LabelUser encodes an arbitrary user string into a Kubernetes label value
// compatible value. This is never decoded again. It shall be without
// collisions, i.e. no hash.
func LabelUser(user string) string {
	if user == "" {
		return user
	}
	user64 := base64.URLEncoding.EncodeToString([]byte(user))
	return strings.TrimRight(user64, "=")
}

func cloudProviderAnnotationPrefix(name string) string {
	return cloudAnnotationPrefix + name + "-"
}

// UnmarshalCluster decodes a Kubernetes namespace into a Kubermatic cluster.
func UnmarshalCluster(cps map[string]provider.CloudProvider, ns *v1.Namespace) (*api.Cluster, error) {
	phaseTS, err := time.Parse(time.RFC3339, ns.Annotations[phaseTimestampAnnotation])
	if err != nil {
		glog.Warningf(
			"Invalid Cluster.Status.LastTransitionTime string %q in namespace %q",
			ns.Annotations[phaseTimestampAnnotation],
			ns.Name,
		)
		phaseTS = time.Now() // gracefully use "now"
	}

	apiserverSSH, _ := base64.StdEncoding.DecodeString(ns.Annotations[apiserverPubSSHAnnotation])
	c := api.Cluster{
		Metadata: api.Metadata{
			Name:        ns.Labels[nameLabelKey],
			User:        ns.Annotations[userAnnotation],
			Revision:    ns.ResourceVersion,
			UID:         string(ns.UID),
			Annotations: map[string]string{},
		},
		Spec: api.ClusterSpec{
			HumanReadableName: ns.Annotations[humanReadableNameAnnotation],
			Dev:               ns.Labels[DevLabelKey] == DevLabelValue,
		},
		Status: api.ClusterStatus{
			LastTransitionTime: phaseTS,
			Phase:              ClusterPhase(ns),
			RootCA: api.SecretKeyCert{
				Key:  api.NewBytes(ns.Annotations[rootCAKeyAnnotation]),
				Cert: api.NewBytes(ns.Annotations[rootCACertAnnotation]),
			},
			ApiserverSSH: string(apiserverSSH),
		},
	}

	// unprefix and copy kubermatic annotations
	for k, v := range ns.Annotations {
		if !strings.HasPrefix(k, customAnnotationPrefix) {
			continue
		}
		k = k[len(customAnnotationPrefix):]
		c.Metadata.Annotations[k] = v
	}

	// get address
	if url, found := ns.Annotations[urlAnnotation]; found {
		token, _ := ns.Annotations[tokenAnnotation]
		etcdPort, _ := ns.Annotations[etcdURLAnnotation]
		c.Address = &api.ClusterAddress{
			URL:     url,
			Token:   token,
			EtcdURL: etcdPort,
		}
	}

	// decode the cloud spec from annotations
	cpName, found := ns.Annotations[providerAnnotation]
	if found {
		cp, found := cps[cpName]
		if !found {
			return nil, fmt.Errorf("unsupported cloud provider %q", cpName)
		}

		var err error
		c.Spec.Cloud, err = unmarshalClusterCloud(cpName, cp, ns.Annotations)
		if err != nil {
			return nil, err
		}
	}

	// decode health
	if healthJSON, found := ns.Annotations[healthAnnotation]; found {
		health := api.ClusterHealth{}
		err := json.Unmarshal([]byte(healthJSON), &health)
		if err != nil {
			glog.Errorf("Error unmarshaling the cluster health of %q: %s", c.Metadata.Name, err.Error())
		} else {
			c.Status.Health = &health
		}
	}

	return &c, nil
}

// MarshalCluster updates a Kubernetes namespace from a Kubermatic cluster.
func MarshalCluster(cps map[string]provider.CloudProvider, c *api.Cluster, ns *v1.Namespace) (*v1.Namespace, error) {
	// filter out old annotations in our domain
	as := map[string]string{}
	for k, v := range ns.Annotations {
		if strings.HasPrefix(k, annotationPrefix) {
			continue
		}
		as[k] = v
	}
	ns.Annotations = as

	// check name
	if ns.Name != "" && ns.Name != NamespaceName(c.Metadata.User, c.Metadata.Name) {
		return nil, fmt.Errorf("cannot rename cluster %s", ns.Name)
	}

	// copy custom annotations
	for k, v := range c.Metadata.Annotations {
		ns.Annotations[customAnnotationPrefix+k] = v
	}

	// set address
	if c.Address != nil {
		if c.Address.URL != "" {
			ns.Annotations[urlAnnotation] = c.Address.URL
		}

		if c.Address.Token != "" {
			ns.Annotations[tokenAnnotation] = c.Address.Token
		}

		if c.Address.EtcdURL != "" {
			ns.Annotations[etcdURLAnnotation] = c.Address.EtcdURL
		}
	}

	// encode cloud spec
	cpName, cp, err := provider.ClusterCloudProvider(cps, c)
	if err != nil {
		return nil, err
	}
	if cp != nil {
		cloudAs, err := marshalClusterCloud(cpName, cp, c)
		if err != nil {
			return nil, err
		}
		for k, v := range cloudAs {
			ns.Annotations[k] = v
		}
	}

	// encode health as json
	if c.Status.Health != nil {
		health, err := json.Marshal(c.Status.Health)
		if err != nil {
			glog.Errorf("Error marshaling the cluster health of %q: %s", c.Metadata.Name, err.Error())
		}
		if health != nil {
			ns.Annotations[healthAnnotation] = string(health)
		}
	}

	ns.Annotations[phaseTimestampAnnotation] = c.Status.LastTransitionTime.Format(time.RFC3339)
	ns.Annotations[userAnnotation] = c.Metadata.User
	ns.Annotations[humanReadableNameAnnotation] = c.Spec.HumanReadableName
	if c.Status.RootCA.Key != nil {
		ns.Annotations[rootCAKeyAnnotation] = c.Status.RootCA.Key.Base64()
	}
	if c.Status.RootCA.Cert != nil {
		ns.Annotations[rootCACertAnnotation] = c.Status.RootCA.Cert.Base64()
	}
	ns.Annotations[apiserverPubSSHAnnotation] = base64.StdEncoding.EncodeToString([]byte(c.Status.ApiserverSSH))

	ns.Labels[RoleLabelKey] = ClusterRoleLabel
	ns.Labels[nameLabelKey] = c.Metadata.Name
	ns.Labels[userLabelKey] = LabelUser(c.Metadata.User)
	if c.Spec.Dev {
		ns.Labels[DevLabelKey] = DevLabelValue
	}

	if c.Status.Phase != api.UnknownClusterStatusPhase {
		ns.Labels[phaseLabelKey] = strings.ToLower(string(c.Status.Phase))
	}

	return ns, nil
}

// marshalClusterCloud returns annotations to persist Spec.Cloud
func marshalClusterCloud(cpName string, cp provider.CloudProvider, c *api.Cluster) (map[string]string, error) {
	cloudAs, err := cp.CreateAnnotations(c.Spec.Cloud)
	if err != nil {
		return nil, err
	}

	prefix := cloudProviderAnnotationPrefix(cpName)
	as := make(map[string]string, len(cloudAs))
	for k, v := range cloudAs {
		as[prefix+k] = v
	}

	as[providerAnnotation] = cpName
	as[cloudDCAnnotation] = c.Spec.Cloud.DC

	return as, nil
}

// unmarshalClusterCloud sets the Spec.Cloud field according to the annotations.
func unmarshalClusterCloud(cpName string, cp provider.CloudProvider, as map[string]string) (*api.CloudSpec, error) {
	prefix := cloudProviderAnnotationPrefix(cpName)
	cloudAs := map[string]string{}
	for k, v := range as {
		if strings.HasPrefix(k, prefix) {
			cloudAs[k[len(prefix):]] = v
		}
	}

	var err error
	spec, err := cp.Cloud(cloudAs)
	if err != nil {
		return nil, err
	}
	spec.DC = as[cloudDCAnnotation]
	spec.Network.Flannel.CIDR = flannelCIDRADefault

	return spec, nil
}

// ClusterPhase derives the cluster phase from the Kubernetes namespace.
func ClusterPhase(ns *v1.Namespace) api.ClusterPhase {
	if ns.Status.Phase == v1.NamespaceTerminating {
		return api.DeletingClusterStatusPhase
	}

	switch api.ClusterPhase(toCapital(ns.Labels[phaseLabelKey])) {
	case api.PendingClusterStatusPhase:
		return api.PendingClusterStatusPhase
	case api.LaunchingClusterStatusPhase:
		return api.LaunchingClusterStatusPhase
	case api.FailedClusterStatusPhase:
		return api.FailedClusterStatusPhase
	case api.RunningClusterStatusPhase:
		return api.RunningClusterStatusPhase
	case api.PausedClusterStatusPhase:
		return api.PausedClusterStatusPhase
	default:
		return api.UnknownClusterStatusPhase
	}
}

// toCapital upper-cases the first character.
func toCapital(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[0:1]) + s[1:]
}

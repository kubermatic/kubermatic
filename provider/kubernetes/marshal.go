package kubernetes

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
	"github.com/lytics/base62"
	kapi "k8s.io/kubernetes/pkg/api"
)

const (
	// RoleLabelKey is the label key set to the value kubermatic-cluster.
	RoleLabelKey = "role"
	// ClusterRoleLabel is the value of the role label of a cluster namespace.
	ClusterRoleLabel = "kubermatic-cluster"

	// annotationPrefix is the prefix string of every cluster namespace annotation.
	annotationPrefix = "kubermatic.io/"

	// namePrefix is the prefix string of every cluster namespace name.
	namePrefix = "cluster"

	urlAnnotation               = annotationPrefix + "url"           // kubermatic.io/url
	tokenAnnoation              = annotationPrefix + "token"         // kubermatic.io/token
	customAnnotationPrefix      = annotationPrefix + "annotation-"   // kubermatic.io/annotation-
	cloudAnnotationPrefix       = annotationPrefix + "cloud-"        // kubermatic.io/cloud-
	providerAnnotation          = cloudAnnotationPrefix + "provider" // kubermatic.io/cloud-provider
	phaseTimestampAnnotation    = annotationPrefix + "phase-ts"      // kubermatic.io/phase-ts
	healthAnnotation            = annotationPrefix + "health"        // kubermatic.io/health
	userAnnotation              = annotationPrefix + "user"          // kubermatic.io/user
	humanReadableNameAnnotation = annotationPrefix + "name"          // kubermatic.io/name

	userLabelKey  = "user"
	nameLabelKey  = "name"
	phaseLabelKey = "phase"
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
	user62 := base62.StdEncoding.EncodeToString([]byte(user))
	return strings.TrimRight(user62, "+")
}

func cloudProviderAnnotationPrefix(cp provider.CloudProvider) string {
	return cloudAnnotationPrefix + cp.Name() + "-"
}

// UnmarshalCluster decodes a Kubernetes namespace into a Kubermatic cluster.
func UnmarshalCluster(cps map[string]provider.CloudProvider, ns *kapi.Namespace) (*api.Cluster, error) {
	phaseTS, err := time.Parse(time.RFC3339, ns.Annotations[phaseTimestampAnnotation])
	if err != nil {
		glog.Warningf(
			"Invalid Cluster.Status.LastTransitionTime string %q in namespace %q",
			ns.Annotations[phaseTimestampAnnotation],
			ns.Name,
		)
		phaseTS = time.Now() // gracefully use "now"
	}

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
		},
		Status: api.ClusterStatus{
			LastTransitionTime: phaseTS,
			Phase:              ClusterPhase(ns),
		},
	}

	// unprefix and copy kubermatic  annotations
	for k, v := range ns.Annotations {
		if !strings.HasPrefix(k, customAnnotationPrefix) {
			continue
		}
		k = k[len(customAnnotationPrefix):]
		c.Metadata.Annotations[k] = v
	}

	// get address
	if url, found := ns.Annotations[urlAnnotation]; found {
		token, _ := ns.Annotations[tokenAnnoation]
		c.Address = &api.ClusterAddress{
			URL:   url,
			Token: token,
		}
	}

	// decode the cloud spec from annotations
	provider, found := ns.Annotations[providerAnnotation]
	if found {
		cp, found := cps[provider]
		if !found {
			return nil, fmt.Errorf("unsupported cloud provider %q", provider)
		}

		var err error
		c.Spec.Cloud, err = unmarshalClusterCloud(cp, ns.Annotations)
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
func MarshalCluster(cps map[string]provider.CloudProvider, c *api.Cluster, ns *kapi.Namespace) (*kapi.Namespace, error) {
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
			ns.Annotations[tokenAnnoation] = c.Address.Token
		}
	}

	// encode cloud spec
	cp, err := provider.ClusterCloudProvider(cps, c)
	if err != nil {
		return nil, err
	}
	if cp != nil {
		cloudAs, err := marshalClusterCloud(cp, c)
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
	ns.Labels[RoleLabelKey] = ClusterRoleLabel
	ns.Labels[nameLabelKey] = c.Metadata.Name
	ns.Labels[userLabelKey] = LabelUser(c.Metadata.User)
	if c.Status.Phase != api.UnknownClusterStatusPhase {
		ns.Labels[phaseLabelKey] = strings.ToLower(string(c.Status.Phase))
	}

	return ns, nil
}

// marshalClusterCloud returns annotations to persist Spec.Cloud
func marshalClusterCloud(cp provider.CloudProvider, c *api.Cluster) (map[string]string, error) {
	cloudAs, err := cp.CreateAnnotations(c.Spec.Cloud)
	if err != nil {
		return nil, err
	}

	prefix := cloudProviderAnnotationPrefix(cp)
	as := make(map[string]string, len(cloudAs))
	for k, v := range cloudAs {
		as[prefix+k] = v
	}

	as[providerAnnotation] = cp.Name()

	return as, nil
}

// unmarshalClusterCloud sets the Spec.Cloud field according to the annotations.
func unmarshalClusterCloud(cp provider.CloudProvider, as map[string]string) (*api.CloudSpec, error) {
	prefix := cloudProviderAnnotationPrefix(cp)
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

	return spec, nil
}

// ClusterPhase derives the cluster phase from the Kubernetes namespace.
func ClusterPhase(ns *kapi.Namespace) api.ClusterPhase {
	if ns.Status.Phase == kapi.NamespaceTerminating {
		return api.DeletingClusterStatusPhase
	}

	switch api.ClusterPhase(toCapital(ns.Labels[phaseLabelKey])) {
	case api.PendingClusterStatusPhase:
		return api.PendingClusterStatusPhase
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

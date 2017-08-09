package kubernetes

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"crypto/sha1"

	"github.com/golang/glog"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
	"k8s.io/client-go/pkg/api/v1"
)

const (
	// RoleLabelKey is the label key set to the value kubermatic-cluster.
	RoleLabelKey = "role"
	// ClusterRoleLabel is the value of the role label of a cluster namespace.
	ClusterRoleLabel = "kubermatic-cluster"
	// WorkerNameLabelKey identifies clusters that are only processed by workerName cluster controllers.
	WorkerNameLabelKey = "worker-name"

	// annotationPrefix is the prefix string of every cluster namespace annotation.
	annotationPrefix = "kubermatic.io/"

	// namePrefix is the prefix string of every cluster namespace name.
	namePrefix = "cluster"

	urlAnnotation                   = annotationPrefix + "url"                    // kubermatic.io/url
	adminTokenAnnotation            = annotationPrefix + "token"                  // kubermatic.io/token
	kubeletTokenAnnotation          = annotationPrefix + "kubelet-token"          // kubermatic.io/kubelet-token
	customAnnotationPrefix          = annotationPrefix + "annotation-"            // kubermatic.io/annotation-
	cloudAnnotationPrefix           = annotationPrefix + "cloud-provider-"        // kubermatic.io/cloud-provider-
	providerAnnotation              = annotationPrefix + "cloud-provider"         // kubermatic.io/cloud-provider
	cloudDCAnnotation               = annotationPrefix + "cloud-dc"               // kubermatic.io/cloud-dc
	phaseTimestampAnnotation        = annotationPrefix + "phase-ts"               // kubermatic.io/phase-ts
	healthAnnotation                = annotationPrefix + "health"                 // kubermatic.io/health
	userAnnotation                  = annotationPrefix + "user"                   // kubermatic.io/user
	humanReadableNameAnnotation     = annotationPrefix + "name"                   // kubermatic.io/name
	rootCAKeyAnnotation             = annotationPrefix + "root-ca-key"            // kubermatic.io/root-ca-key
	rootCACertAnnotation            = annotationPrefix + "root-ca-cert"           // kubermatic.io/root-cert
	apiserverCertAnnotation         = annotationPrefix + "apiserver-cert"         // kubermatic.io/apiserver-cert
	apiserverCertKeyAnnotation      = annotationPrefix + "apiserver-cert-key"     // kubermatic.io/apiserver-cert-key
	kubeletCertAnnotation           = annotationPrefix + "kubelet-cert"           // kubermatic.io/kubelet-cert
	kubeletCertKeyAnnotation        = annotationPrefix + "kubelet-cert-key"       // kubermatic.io/kubelet-cert-key
	apiserverSSHPrivKeyAnnotation   = annotationPrefix + "apiserver-ssh-priv-key" // kubermatic.io/apiserver-ssh-priv-key
	apiserverSSHPubKeyAnnotation    = annotationPrefix + "apiserver-ssh-pub-key"  // kubermatic.io/apiserver-ssh-pub-key
	apiserverPubSSHAnnotation       = annotationPrefix + "ssh-pub"                // kubermatic.io/ssh-pub
	serviceAccountKeyAnnotation     = annotationPrefix + "service-account-key"    // kubermatic.io/service-account-key
	seedProviderUsedAnnotation      = annotationPrefix + "seed-provider-used"     // kubermatic.io/seed-provider-used
	apiserverExternalNameAnnotation = annotationPrefix + "external-name"          // external-name
	apiserverExternalPortAnnotation = annotationPrefix + "external-port"          // external-port

	// LastDeployedMasterVersionAnnotation represents the annotation key for the LastDeployedMasterVersion
	LastDeployedMasterVersionAnnotation = annotationPrefix + "last-deployed-master-verion" // kubermatic.io/last-deployed-master-version

	// MasterUpdatePhaseAnnotation represents the annotation key for the MasterUpdatePhase
	MasterUpdatePhaseAnnotation = annotationPrefix + "master-update-phase" // kubermatic.io/master-update-phase

	// MasterVersionAnnotation represents the annotation key for the MasterVersion
	MasterVersionAnnotation = annotationPrefix + "master-version" // kubermatic.io/master-verion

	userLabelKey  = "user"
	nameLabelKey  = "name"
	phaseLabelKey = "phase"

	flannelCIDRADefault = "172.25.0.0/16"
)

// NamespaceName create a namespace name for a given cluster.
func NamespaceName(cluster string) string {
	return fmt.Sprintf("%s-%s", namePrefix, cluster)
}

// LabelUser encodes an arbitrary user string into a Kubernetes label value
// compatible value. This is never decoded again. It shall be without
// collisions, i.e. no hash. This is a one-way-function!
// When the user is to long it will be hashed.
// This is done for backwards compatibility!
func LabelUser(user string) string {
	if user == "" {
		return user
	}
        // This part has to stay for backwards capability.
        // It we need this for old clusters which use an auth provider with useres, which will encode
        // in less then 63 chars.
        b := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte(name))
        if len(b) <= 63 {
                return b
        }

        // This is the new way.
        // We can use a weak hash because we trust the authority, which generates the name.
        // This will always yield a string which makes the user identifiable and is less than 63 chars
        // due to the usage of a hash function.
        // Potentially we could have collisions, but this is not avoidable, due to the fact that the
        // set of our domain is smaller than our codomain.
        // It's trivial to see that we can't reverse this due to the fact that our function is not injective. q.e.d
        sh := sha1.New()
        fmt.Fprint(sh, name)
        return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(sh.Sum(nil))
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

	apiserverSSH, err := base64.StdEncoding.DecodeString(ns.Annotations[apiserverPubSSHAnnotation])
	if err != nil {
		return nil, fmt.Errorf("failed to decode apiserver pub ssh key: %v", err)
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
			MasterVersion:     ns.Annotations[MasterVersionAnnotation],
			WorkerName:        ns.Labels[WorkerNameLabelKey],
		},
		Status: api.ClusterStatus{
			LastTransitionTime: phaseTS,
			Phase:              ClusterPhase(ns),
			LastDeployedMasterVersion: ns.Annotations[LastDeployedMasterVersionAnnotation],
			MasterUpdatePhase:         api.MasterUpdatePhase(ns.Annotations[MasterUpdatePhaseAnnotation]),
			ServiceAccountKey:         api.NewBytes(ns.Annotations[serviceAccountKeyAnnotation]),
			RootCA: api.SecretKeyCert{
				Key:  api.NewBytes(ns.Annotations[rootCAKeyAnnotation]),
				Cert: api.NewBytes(ns.Annotations[rootCACertAnnotation]),
			},
			ApiserverCert: api.KeyCert{
				Key:  api.NewBytes(ns.Annotations[apiserverCertKeyAnnotation]),
				Cert: api.NewBytes(ns.Annotations[apiserverCertAnnotation]),
			},
			KubeletCert: api.KeyCert{
				Key:  api.NewBytes(ns.Annotations[kubeletCertKeyAnnotation]),
				Cert: api.NewBytes(ns.Annotations[kubeletCertAnnotation]),
			},
			ApiserverSSHKey: api.SecretRSAKeys{
				PublicKey:  api.NewBytes(ns.Annotations[apiserverSSHPubKeyAnnotation]),
				PrivateKey: api.NewBytes(ns.Annotations[apiserverSSHPrivKeyAnnotation]),
			},
			ApiserverSSH: string(apiserverSSH),
		},
		Seed: ns.Annotations[seedProviderUsedAnnotation],
	}

	// unprefix and copy kubermatic annotations
	for k, v := range ns.Annotations {
		if !strings.HasPrefix(k, customAnnotationPrefix) {
			continue
		}
		k = k[len(customAnnotationPrefix):]
		c.Metadata.Annotations[k] = v
	}

	c.Address = &api.ClusterAddress{}
	c.Address.URL = ns.Annotations[urlAnnotation]
	c.Address.AdminToken = ns.Annotations[adminTokenAnnotation]
	c.Address.KubeletToken = ns.Annotations[kubeletTokenAnnotation]
	c.Address.ExternalName = ns.Annotations[apiserverExternalNameAnnotation]
	if externalPort, found := ns.Annotations[apiserverExternalPortAnnotation]; found {
		iExternalPort, err := strconv.Atoi(externalPort)
		if err != nil {
			return nil, fmt.Errorf("failed to parse external apiserver port: %v", err)
		}
		c.Address.ExternalPort = int(iExternalPort)
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
	if ns.Name != "" && ns.Name != NamespaceName(c.Metadata.Name) {
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

		if c.Address.AdminToken != "" {
			ns.Annotations[adminTokenAnnotation] = c.Address.AdminToken
		}

		if c.Address.KubeletToken != "" {
			ns.Annotations[kubeletTokenAnnotation] = c.Address.KubeletToken
		}

		if c.Address.ExternalName != "" {
			ns.Annotations[apiserverExternalNameAnnotation] = c.Address.ExternalName
		}

		if c.Address.ExternalPort != 0 {
			ns.Annotations[apiserverExternalPortAnnotation] = strconv.Itoa(c.Address.ExternalPort)
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
	ns.Annotations[MasterVersionAnnotation] = c.Spec.MasterVersion
	ns.Annotations[MasterUpdatePhaseAnnotation] = string(c.Status.MasterUpdatePhase)
	ns.Annotations[LastDeployedMasterVersionAnnotation] = c.Status.LastDeployedMasterVersion
	ns.Annotations[seedProviderUsedAnnotation] = c.Seed
	if c.Status.RootCA.Key != nil {
		ns.Annotations[rootCAKeyAnnotation] = c.Status.RootCA.Key.Base64()
	}
	if c.Status.RootCA.Cert != nil {
		ns.Annotations[rootCACertAnnotation] = c.Status.RootCA.Cert.Base64()
	}
	if c.Status.ApiserverSSHKey.PrivateKey != nil {
		ns.Annotations[apiserverSSHPrivKeyAnnotation] = c.Status.ApiserverSSHKey.PrivateKey.Base64()
	}
	if c.Status.ApiserverSSHKey.PublicKey != nil {
		ns.Annotations[apiserverSSHPubKeyAnnotation] = c.Status.ApiserverSSHKey.PublicKey.Base64()
	}
	if c.Status.ApiserverCert.Cert != nil {
		ns.Annotations[apiserverCertAnnotation] = c.Status.ApiserverCert.Cert.Base64()
	}
	if c.Status.ApiserverCert.Key != nil {
		ns.Annotations[apiserverCertKeyAnnotation] = c.Status.ApiserverCert.Key.Base64()
	}
	if c.Status.KubeletCert.Cert != nil {
		ns.Annotations[kubeletCertAnnotation] = c.Status.KubeletCert.Cert.Base64()
	}
	if c.Status.KubeletCert.Key != nil {
		ns.Annotations[kubeletCertKeyAnnotation] = c.Status.KubeletCert.Key.Base64()
	}
	if c.Status.ServiceAccountKey != nil {
		ns.Annotations[serviceAccountKeyAnnotation] = c.Status.ServiceAccountKey.Base64()
	}

	ns.Annotations[apiserverPubSSHAnnotation] = base64.StdEncoding.EncodeToString([]byte(c.Status.ApiserverSSH))

	ns.Labels[RoleLabelKey] = ClusterRoleLabel
	ns.Labels[nameLabelKey] = c.Metadata.Name
	ns.Labels[userLabelKey] = LabelUser(c.Metadata.User)
	ns.Labels[WorkerNameLabelKey] = c.Spec.WorkerName

	if c.Status.Phase != api.UnknownClusterStatusPhase {
		ns.Labels[phaseLabelKey] = strings.ToLower(string(c.Status.Phase))
	}

	return ns, nil
}

// marshalClusterCloud returns annotations to persist Spec.Cloud
func marshalClusterCloud(cpName string, cp provider.CloudProvider, c *api.Cluster) (annotations map[string]string, err error) {
	cloudAs, err := cp.MarshalCloudSpec(c.Spec.Cloud)
	if err != nil {
		return nil, err
	}

	prefix := cloudProviderAnnotationPrefix(cpName)
	annotations = make(map[string]string, len(cloudAs))
	for k, v := range cloudAs {
		annotations[prefix+k] = v
	}

	annotations[providerAnnotation] = cpName
	annotations[cloudDCAnnotation] = c.Spec.Cloud.DatacenterName

	return annotations, nil
}

// unmarshalClusterCloud sets the Spec.Cloud field according to the annotations.
func unmarshalClusterCloud(cpName string, cp provider.CloudProvider, annotations map[string]string) (*api.CloudSpec, error) {
	prefix := cloudProviderAnnotationPrefix(cpName)
	cloudAs := map[string]string{}
	for k, v := range annotations {
		if strings.HasPrefix(k, prefix) {
			cloudAs[k[len(prefix):]] = v
		}
	}

	var err error
	spec, err := cp.UnmarshalCloudSpec(cloudAs)
	if err != nil {
		return nil, err
	}
	spec.DatacenterName = annotations[cloudDCAnnotation]
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
	case api.UpdatingMasterClusterStatusPhase:
		return api.UpdatingMasterClusterStatusPhase
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

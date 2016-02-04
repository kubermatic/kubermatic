package kubernetes

import (
	"fmt"
	"strings"

	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
	kapi "k8s.io/kubernetes/pkg/api"
)

const (
	annotationPrefix = "kubermatic.io/"

	namePrefix = "kubermatic-cluster-"

	addressURLAnnoation    = annotationPrefix + "address-url"
	addressTokenAnnoation  = annotationPrefix + "address-token"
	customAnnotationPrefix = annotationPrefix + "annotations-"
	cloudAnnotationPrefix  = annotationPrefix + "cloud-"

	roleLabelKey     = "role"
	nameLabelKey     = "name"
	clusterRoleLabel = "kubermatic-cluster"
)

func cloudProviderAnnotationPrefix(cp provider.CloudProvider) string {
	return cloudAnnotationPrefix + cp.Name() + "-"
}

// unmarshalCluster decodes a Kubernetes namespace into a Kubermatic cluster.
func unmarshalCluster(cps map[string]provider.CloudProvider, ns *kapi.Namespace) (*api.Cluster, error) {
	c := api.Cluster{
		Metadata: api.Metadata{
			Name:        ns.Labels["name"],
			Revision:    ns.ResourceVersion,
			UID:         string(ns.UID),
			Annotations: map[string]string{},
		},
		Spec: api.ClusterSpec{},
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
	if url, found := ns.Annotations[addressURLAnnoation]; found {
		token, _ := ns.Annotations[addressTokenAnnoation]
		c.Address = &api.ClusterAddress{
			URL:   url,
			Token: token,
		}
	}

	// decode the cloud spec from annoations
	name, found := ns.Annotations[cloudAnnotationPrefix+"provider"]
	if found {
		cp, found := cps[name]
		if !found {
			return nil, fmt.Errorf("unsupported cloud provider %q", name)
		}

		var err error
		c.Spec.Cloud, err = unmarshalClusterCloud(cp, ns.Annotations)
		if err != nil {
			return nil, err
		}
	}

	return &c, nil
}

// marshalCluster updates a Kubernetes namespace from a Kubermatic cluster.
func marshalCluster(cps map[string]provider.CloudProvider, c *api.Cluster, ns *kapi.Namespace) (*kapi.Namespace, error) {
	// filter out old annotations in our domain
	as := map[string]string{}
	for k, v := range ns.Annotations {
		if !strings.HasPrefix(k, annotationPrefix) {
			continue
		}
		k = k[len(annotationPrefix):]
		as[k] = v
	}

	// set name
	if ns.Name != "" && ns.Name != namePrefix+c.Metadata.Name {
		return nil, fmt.Errorf("cannot rename cluster %s", ns.Name)
	}
	ns.Name = namePrefix + c.Metadata.Name
	as[nameLabelKey] = c.Metadata.Name

	// copy custom annotations
	for k, v := range c.Metadata.Annotations {
		as[customAnnotationPrefix+k] = v
	}

	// set address
	if c.Address != nil {
		if c.Address.URL != "" {
			as[addressURLAnnoation] = c.Address.URL
		}
		if c.Address.Token != "" {
			as[addressTokenAnnoation] = c.Address.Token
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
			as[k] = v
		}
	}

	ns.Annotations = as
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

	as[cloudAnnotationPrefix+"provider"] = cp.Name()

	return as, nil
}

// UnmarshalClusterCloud sets the Spec.Cloud field according to the annotations
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

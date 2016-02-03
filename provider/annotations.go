package provider

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kubermatic/api"
)

const cloudAnnotationPrefix = "cloud-"

func cloudProviderAnnotationPrefix(cp CloudProvider) string {
	return cloudAnnotationPrefix + cp.Name() + "-"
}

// MarshalClusterCloud sets the annotations to persist Spec.Cloud
func MarshalClusterCloud(cps map[string]CloudProvider, c *api.Cluster) error {
	cp, err := ClusterCloudProvider(cps, c)
	if err != nil {
		return err
	}

	cloudAs, err := cp.CreateAnnotations(c.Spec.Cloud)
	if err != nil {
		return err
	}

	prefix := cloudProviderAnnotationPrefix(cp)
	newAs := make(map[string]string, len(cloudAs)+len(c.Metadata.Annotations))
	for k, v := range c.Metadata.Annotations {
		if !strings.HasPrefix(k, cloudAnnotationPrefix) {
			newAs[k] = v
		}
	}
	for k, v := range cloudAs {
		newAs[prefix+k] = v
	}

	newAs[cloudAnnotationPrefix+"provider"] = cp.Name()

	return nil
}

// UnmarshalClusterCloud sets the Spec.Cloud field according to the annotations
func UnmarshalClusterCloud(cps map[string]CloudProvider, c *api.Cluster) error {
	name, found := c.Metadata.Annotations[cloudAnnotationPrefix+"provider"]
	if !found {
		return errors.New("no cloud provider annotation found")
	}

	cp, found := cps[name]
	if !found {
		return fmt.Errorf("unsupported cloud provider %q", name)
	}

	prefix := cloudProviderAnnotationPrefix(cp)
	cloudAs := map[string]string{}
	for k, v := range c.Metadata.Annotations {
		if strings.HasPrefix(k, prefix) {
			cloudAs[k[len(prefix):]] = v
		}
	}

	var err error
	c.Spec.Cloud, err = cp.Cloud(cloudAs)
	if err != nil {
		return err
	}

	return nil
}

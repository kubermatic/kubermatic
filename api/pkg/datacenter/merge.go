package datacenter

import (
	"errors"
	"path"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

// DefaultFromDatacenter adds defaults coming from the datacenter to the cluster object
func DefaultFromDatacenter(spec *kubermaticv1.CloudSpec, dc provider.DatacenterMeta) (*kubermaticv1.CloudSpec, error) {
	if spec.Digitalocean != nil {
		return spec, nil
	} else if spec.Openstack != nil {
		return defaultFromOpenstackDatacenter(spec, dc)
	} else if spec.VSphere != nil {
		return defaultFromVSphereDatacenter(spec, dc)
	} else if spec.Azure != nil {
		return spec, nil
	} else if spec.AWS != nil {
		return defaultFromAWSDatacenter(spec, dc)
	} else if spec.Hetzner != nil {
		return spec, nil
	} else if spec.Fake != nil {
		return spec, nil
	}

	return nil, errors.New("no cloud provider specified")
}

func defaultFromOpenstackDatacenter(spec *kubermaticv1.CloudSpec, dc provider.DatacenterMeta) (*kubermaticv1.CloudSpec, error) {
	if dc.Spec.Openstack == nil {
		return nil, errors.New("node datacenter has no openstack specification")
	}

	if spec.Openstack.AuthURL == "" {
		spec.Openstack.AuthURL = dc.Spec.Openstack.AuthURL
	}
	if spec.Openstack.Region == "" {
		spec.Openstack.Region = dc.Spec.Openstack.Region
	}
	if len(spec.Openstack.DNSServers) == 0 {
		spec.Openstack.DNSServers = dc.Spec.Openstack.DNSServers
	}

	spec.Openstack.IgnoreVolumeAZ = dc.Spec.Openstack.IgnoreVolumeAZ
	return spec, nil
}

func defaultFromVSphereDatacenter(spec *kubermaticv1.CloudSpec, dc provider.DatacenterMeta) (*kubermaticv1.CloudSpec, error) {
	if dc.Spec.VSphere == nil {
		return nil, errors.New("node datacenter has no vSphere specification")
	}

	if spec.VSphere.Endpoint == "" {
		spec.VSphere.Endpoint = dc.Spec.VSphere.Endpoint
		spec.VSphere.AllowInsecure = dc.Spec.VSphere.AllowInsecure
	}

	if spec.VSphere.Cluster == "" {
		spec.VSphere.Cluster = dc.Spec.VSphere.Cluster
	}
	if spec.VSphere.VMFolder == "" {
		if dc.Spec.VSphere.RootPath == "" {
			return nil, errors.New("no vSphere root path specified. Unable to define vm folder for cluster")
		}
		spec.VSphere.VMFolder = path.Join(dc.Spec.VSphere.RootPath, dc.Spec.VSphere.Cluster)
	}

	if spec.VSphere.Datacenter == "" {
		spec.VSphere.Datacenter = dc.Spec.VSphere.Datacenter
	}

	if spec.VSphere.Datastore == "" {
		spec.VSphere.Datastore = dc.Spec.VSphere.Datastore
	}

	return spec, nil
}

func defaultFromAWSDatacenter(spec *kubermaticv1.CloudSpec, dc provider.DatacenterMeta) (*kubermaticv1.CloudSpec, error) {
	if dc.Spec.AWS == nil {
		return nil, errors.New("node datacenter has no AWS specification")
	}

	if spec.AWS.Region == "" {
		spec.AWS.Region = dc.Spec.AWS.Region
	}
	return spec, nil
}

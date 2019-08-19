package vsphere

import (
	"context"
	"fmt"
	"path"

	"github.com/vmware/govmomi/object"
	"k8s.io/apimachinery/pkg/util/runtime"
)

type NetworkInfo struct {
	Name         string
	AbsolutePath string
	Type         string
}

func getPossibleVMNetworks(ctx context.Context, session *Session) ([]NetworkInfo, error) {
	var infos []NetworkInfo

	networks, err := session.Finder.NetworkList(ctx, "*")
	if err != nil {
		return nil, err
	}
	for _, network := range networks {
		if _, err := network.EthernetCardBackingInfo(ctx); err != nil {
			// Some network devices cannot be used by VM's.
			if err == object.ErrNotSupported {
				continue
			}

			// Just log the error. We do not want a single weird network to break the entire listing.
			runtime.HandleError(fmt.Errorf("Failed to get network backing info for %q: %v", network.Reference().String(), err))
			continue
		}

		// We need to pull the elements info from the API because there's no sane way of retrieving the path for a NetworkReference via the SDK
		// unless we wan't to maintain a long switch statement with all kind of types
		element, err := session.Finder.Element(ctx, network.Reference())
		if err != nil {
			return nil, fmt.Errorf("failed to get details for %q: %v", network.Reference().String(), err)
		}

		info := NetworkInfo{
			AbsolutePath: element.Path,
			Type:         network.Reference().Type,
			Name:         path.Base(element.Path),
		}
		infos = append(infos, info)
	}

	return infos, nil
}

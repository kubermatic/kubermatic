/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package vsphere

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/vmware/govmomi/object"
	"k8s.io/apimachinery/pkg/util/runtime"
)

type NetworkInfo struct {
	Name         string
	RelativePath string
	AbsolutePath string
	Type         string
}

func getPossibleVMNetworks(ctx context.Context, session *Session) ([]NetworkInfo, error) {
	var infos []NetworkInfo

	datacenterFolders, err := session.Datacenter.Folders(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load the datacenter folders: %v", err)
	}

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

			// Just log the error. If we cannot create a backing info, that network device is not suitable for VM's.
			// Normally we should cover unsupported network devices with the ErrNotSupported above.
			// This is just a fallback to prevent that a single network device breaks the list operation
			runtime.HandleError(fmt.Errorf("failed to get network backing info for %q: %v", network.Reference().String(), err))
			continue
		}

		// We need to pull the elements info from the API because there's no sane way of retrieving the path for a NetworkReference via the SDK
		// unless we want to maintain a long switch statement with all kind of types
		element, err := session.Finder.Element(ctx, network.Reference())
		if err != nil {
			return nil, fmt.Errorf("failed to get details for %q: %v", network.Reference().String(), err)
		}

		info := NetworkInfo{
			AbsolutePath: element.Path,
			RelativePath: strings.TrimPrefix(element.Path, datacenterFolders.NetworkFolder.InventoryPath+"/"),
			Type:         network.Reference().Type,
			Name:         path.Base(element.Path),
		}
		infos = append(infos, info)
	}

	return infos, nil
}

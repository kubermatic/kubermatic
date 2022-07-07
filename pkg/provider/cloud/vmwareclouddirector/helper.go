/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package vmwareclouddirector

import (
	"fmt"

	"github.com/vmware/go-vcloud-director/v2/govcd"
)

func deleteVApp(vdc *govcd.Vdc, vapp *govcd.VApp) error {
	// Undeploy failed, it's still safe to delete vApp directly since it will take care of all the cleanup.
	// Most common reason for failure is that the vApp is not in "running" state.
	task, err := vapp.Undeploy()
	if err == nil {
		if err = task.WaitTaskCompletion(); err != nil {
			return fmt.Errorf("error waiting for vApp undeploy to complete: %w", err)
		}
	}

	task, err = vapp.Delete()
	if err != nil {
		return fmt.Errorf("failed to delete vApp: %w", err)
	}
	err = task.WaitTaskCompletion()
	if err != nil {
		return fmt.Errorf("error waiting for vApp deletion to complete: %w", err)
	}
	return nil
}

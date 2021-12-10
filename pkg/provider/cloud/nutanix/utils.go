/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package nutanix

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

func waitForCompletion(client *ClientSet, taskID string) error {
	if err := wait.Poll(10*time.Second, 5*time.Minute, func() (bool, error) {
		task, err := client.Prism.V3.GetTask(taskID)
		if err != nil {
			return false, err
		}

		if task.Status == nil {
			return false, nil
		}

		switch *task.Status {
		case "INVALID_UUID", "FAILED":
			return false, fmt.Errorf("bad status: %s", *task.Status)
		case "QUEUED", "RUNNING":
			return false, nil
		case "SUCCEEDED":
			return true, nil
		default:
			return false, fmt.Errorf("unknown status: %s", *task.Status)
		}

	}); err != nil {
		return err
	}

	return nil
}

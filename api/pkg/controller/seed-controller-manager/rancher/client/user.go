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

package client

import (
	"encoding/json"
	"fmt"
)

func (c *Client) SetUserPassword(user *User, newPassword *SetPasswordInput) error {
	endpoint, err := appendFilters(fmt.Sprintf("%s/v3/users/%s", c.options.Endpoint, user.ID), map[string]string{"action": "setpassword"})
	if err != nil {
		return err
	}
	data, err := json.Marshal(newPassword)
	if err != nil {
		return fmt.Errorf("failed marshal setPasswordInput object: %v", err)
	}
	return c.do(endpoint, string(data), nil)

}

func (c *Client) ListUsers(filters Filters) (*UserList, error) {
	endpoint, err := appendFilters(fmt.Sprintf("%s/v3/users/", c.options.Endpoint), filters)
	if err != nil {
		return nil, err
	}
	list := &UserList{}
	err = c.do(endpoint, "", list)
	return list, err
}

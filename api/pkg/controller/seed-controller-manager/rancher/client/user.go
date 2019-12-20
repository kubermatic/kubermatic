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

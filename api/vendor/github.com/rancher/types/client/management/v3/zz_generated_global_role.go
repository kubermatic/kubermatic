package client

import (
	"github.com/rancher/norman/types"
)

const (
	GlobalRoleType                 = "globalRole"
	GlobalRoleFieldAnnotations     = "annotations"
	GlobalRoleFieldBuiltin         = "builtin"
	GlobalRoleFieldCreated         = "created"
	GlobalRoleFieldCreatorID       = "creatorId"
	GlobalRoleFieldDescription     = "description"
	GlobalRoleFieldLabels          = "labels"
	GlobalRoleFieldName            = "name"
	GlobalRoleFieldNewUserDefault  = "newUserDefault"
	GlobalRoleFieldOwnerReferences = "ownerReferences"
	GlobalRoleFieldRemoved         = "removed"
	GlobalRoleFieldRules           = "rules"
	GlobalRoleFieldUUID            = "uuid"
)

type GlobalRole struct {
	types.Resource
	Annotations     map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	Builtin         bool              `json:"builtin,omitempty" yaml:"builtin,omitempty"`
	Created         string            `json:"created,omitempty" yaml:"created,omitempty"`
	CreatorID       string            `json:"creatorId,omitempty" yaml:"creatorId,omitempty"`
	Description     string            `json:"description,omitempty" yaml:"description,omitempty"`
	Labels          map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Name            string            `json:"name,omitempty" yaml:"name,omitempty"`
	NewUserDefault  bool              `json:"newUserDefault,omitempty" yaml:"newUserDefault,omitempty"`
	OwnerReferences []OwnerReference  `json:"ownerReferences,omitempty" yaml:"ownerReferences,omitempty"`
	Removed         string            `json:"removed,omitempty" yaml:"removed,omitempty"`
	Rules           []PolicyRule      `json:"rules,omitempty" yaml:"rules,omitempty"`
	UUID            string            `json:"uuid,omitempty" yaml:"uuid,omitempty"`
}

type GlobalRoleCollection struct {
	types.Collection
	Data   []GlobalRole `json:"data,omitempty"`
	client *GlobalRoleClient
}

type GlobalRoleClient struct {
	apiClient *Client
}

type GlobalRoleOperations interface {
	List(opts *types.ListOpts) (*GlobalRoleCollection, error)
	Create(opts *GlobalRole) (*GlobalRole, error)
	Update(existing *GlobalRole, updates interface{}) (*GlobalRole, error)
	Replace(existing *GlobalRole) (*GlobalRole, error)
	ByID(id string) (*GlobalRole, error)
	Delete(container *GlobalRole) error
}

func newGlobalRoleClient(apiClient *Client) *GlobalRoleClient {
	return &GlobalRoleClient{
		apiClient: apiClient,
	}
}

func (c *GlobalRoleClient) Create(container *GlobalRole) (*GlobalRole, error) {
	resp := &GlobalRole{}
	err := c.apiClient.Ops.DoCreate(GlobalRoleType, container, resp)
	return resp, err
}

func (c *GlobalRoleClient) Update(existing *GlobalRole, updates interface{}) (*GlobalRole, error) {
	resp := &GlobalRole{}
	err := c.apiClient.Ops.DoUpdate(GlobalRoleType, &existing.Resource, updates, resp)
	return resp, err
}

func (c *GlobalRoleClient) Replace(obj *GlobalRole) (*GlobalRole, error) {
	resp := &GlobalRole{}
	err := c.apiClient.Ops.DoReplace(GlobalRoleType, &obj.Resource, obj, resp)
	return resp, err
}

func (c *GlobalRoleClient) List(opts *types.ListOpts) (*GlobalRoleCollection, error) {
	resp := &GlobalRoleCollection{}
	err := c.apiClient.Ops.DoList(GlobalRoleType, opts, resp)
	resp.client = c
	return resp, err
}

func (cc *GlobalRoleCollection) Next() (*GlobalRoleCollection, error) {
	if cc != nil && cc.Pagination != nil && cc.Pagination.Next != "" {
		resp := &GlobalRoleCollection{}
		err := cc.client.apiClient.Ops.DoNext(cc.Pagination.Next, resp)
		resp.client = cc.client
		return resp, err
	}
	return nil, nil
}

func (c *GlobalRoleClient) ByID(id string) (*GlobalRole, error) {
	resp := &GlobalRole{}
	err := c.apiClient.Ops.DoByID(GlobalRoleType, id, resp)
	return resp, err
}

func (c *GlobalRoleClient) Delete(container *GlobalRole) error {
	return c.apiClient.Ops.DoResourceDelete(GlobalRoleType, &container.Resource)
}

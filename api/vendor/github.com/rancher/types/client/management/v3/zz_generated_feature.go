package client

import (
	"github.com/rancher/norman/types"
)

const (
	FeatureType                 = "feature"
	FeatureFieldAnnotations     = "annotations"
	FeatureFieldCreated         = "created"
	FeatureFieldCreatorID       = "creatorId"
	FeatureFieldDefault         = "default"
	FeatureFieldLabels          = "labels"
	FeatureFieldName            = "name"
	FeatureFieldOwnerReferences = "ownerReferences"
	FeatureFieldRemoved         = "removed"
	FeatureFieldUUID            = "uuid"
	FeatureFieldValue           = "value"
)

type Feature struct {
	types.Resource
	Annotations     map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	Created         string            `json:"created,omitempty" yaml:"created,omitempty"`
	CreatorID       string            `json:"creatorId,omitempty" yaml:"creatorId,omitempty"`
	Default         bool              `json:"default,omitempty" yaml:"default,omitempty"`
	Labels          map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Name            string            `json:"name,omitempty" yaml:"name,omitempty"`
	OwnerReferences []OwnerReference  `json:"ownerReferences,omitempty" yaml:"ownerReferences,omitempty"`
	Removed         string            `json:"removed,omitempty" yaml:"removed,omitempty"`
	UUID            string            `json:"uuid,omitempty" yaml:"uuid,omitempty"`
	Value           *bool             `json:"value,omitempty" yaml:"value,omitempty"`
}

type FeatureCollection struct {
	types.Collection
	Data   []Feature `json:"data,omitempty"`
	client *FeatureClient
}

type FeatureClient struct {
	apiClient *Client
}

type FeatureOperations interface {
	List(opts *types.ListOpts) (*FeatureCollection, error)
	Create(opts *Feature) (*Feature, error)
	Update(existing *Feature, updates interface{}) (*Feature, error)
	Replace(existing *Feature) (*Feature, error)
	ByID(id string) (*Feature, error)
	Delete(container *Feature) error
}

func newFeatureClient(apiClient *Client) *FeatureClient {
	return &FeatureClient{
		apiClient: apiClient,
	}
}

func (c *FeatureClient) Create(container *Feature) (*Feature, error) {
	resp := &Feature{}
	err := c.apiClient.Ops.DoCreate(FeatureType, container, resp)
	return resp, err
}

func (c *FeatureClient) Update(existing *Feature, updates interface{}) (*Feature, error) {
	resp := &Feature{}
	err := c.apiClient.Ops.DoUpdate(FeatureType, &existing.Resource, updates, resp)
	return resp, err
}

func (c *FeatureClient) Replace(obj *Feature) (*Feature, error) {
	resp := &Feature{}
	err := c.apiClient.Ops.DoReplace(FeatureType, &obj.Resource, obj, resp)
	return resp, err
}

func (c *FeatureClient) List(opts *types.ListOpts) (*FeatureCollection, error) {
	resp := &FeatureCollection{}
	err := c.apiClient.Ops.DoList(FeatureType, opts, resp)
	resp.client = c
	return resp, err
}

func (cc *FeatureCollection) Next() (*FeatureCollection, error) {
	if cc != nil && cc.Pagination != nil && cc.Pagination.Next != "" {
		resp := &FeatureCollection{}
		err := cc.client.apiClient.Ops.DoNext(cc.Pagination.Next, resp)
		resp.client = cc.client
		return resp, err
	}
	return nil, nil
}

func (c *FeatureClient) ByID(id string) (*Feature, error) {
	resp := &Feature{}
	err := c.apiClient.Ops.DoByID(FeatureType, id, resp)
	return resp, err
}

func (c *FeatureClient) Delete(container *Feature) error {
	return c.apiClient.Ops.DoResourceDelete(FeatureType, &container.Resource)
}

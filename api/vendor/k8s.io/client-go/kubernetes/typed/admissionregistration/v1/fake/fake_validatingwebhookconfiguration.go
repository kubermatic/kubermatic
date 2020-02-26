/*
Copyright The Kubernetes Authors.

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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeValidatingWebhookConfigurations implements ValidatingWebhookConfigurationInterface
type FakeValidatingWebhookConfigurations struct {
	Fake *FakeAdmissionregistrationV1
}

var validatingwebhookconfigurationsResource = schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "validatingwebhookconfigurations"}

var validatingwebhookconfigurationsKind = schema.GroupVersionKind{Group: "admissionregistration.k8s.io", Version: "v1", Kind: "ValidatingWebhookConfiguration"}

// Get takes name of the validatingWebhookConfiguration, and returns the corresponding validatingWebhookConfiguration object, and an error if there is any.
func (c *FakeValidatingWebhookConfigurations) Get(name string, options v1.GetOptions) (result *admissionregistrationv1.ValidatingWebhookConfiguration, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(validatingwebhookconfigurationsResource, name), &admissionregistrationv1.ValidatingWebhookConfiguration{})
	if obj == nil {
		return nil, err
	}
	return obj.(*admissionregistrationv1.ValidatingWebhookConfiguration), err
}

// List takes label and field selectors, and returns the list of ValidatingWebhookConfigurations that match those selectors.
func (c *FakeValidatingWebhookConfigurations) List(opts v1.ListOptions) (result *admissionregistrationv1.ValidatingWebhookConfigurationList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(validatingwebhookconfigurationsResource, validatingwebhookconfigurationsKind, opts), &admissionregistrationv1.ValidatingWebhookConfigurationList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &admissionregistrationv1.ValidatingWebhookConfigurationList{ListMeta: obj.(*admissionregistrationv1.ValidatingWebhookConfigurationList).ListMeta}
	for _, item := range obj.(*admissionregistrationv1.ValidatingWebhookConfigurationList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested validatingWebhookConfigurations.
func (c *FakeValidatingWebhookConfigurations) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(validatingwebhookconfigurationsResource, opts))
}

// Create takes the representation of a validatingWebhookConfiguration and creates it.  Returns the server's representation of the validatingWebhookConfiguration, and an error, if there is any.
func (c *FakeValidatingWebhookConfigurations) Create(validatingWebhookConfiguration *admissionregistrationv1.ValidatingWebhookConfiguration) (result *admissionregistrationv1.ValidatingWebhookConfiguration, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(validatingwebhookconfigurationsResource, validatingWebhookConfiguration), &admissionregistrationv1.ValidatingWebhookConfiguration{})
	if obj == nil {
		return nil, err
	}
	return obj.(*admissionregistrationv1.ValidatingWebhookConfiguration), err
}

// Update takes the representation of a validatingWebhookConfiguration and updates it. Returns the server's representation of the validatingWebhookConfiguration, and an error, if there is any.
func (c *FakeValidatingWebhookConfigurations) Update(validatingWebhookConfiguration *admissionregistrationv1.ValidatingWebhookConfiguration) (result *admissionregistrationv1.ValidatingWebhookConfiguration, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(validatingwebhookconfigurationsResource, validatingWebhookConfiguration), &admissionregistrationv1.ValidatingWebhookConfiguration{})
	if obj == nil {
		return nil, err
	}
	return obj.(*admissionregistrationv1.ValidatingWebhookConfiguration), err
}

// Delete takes name of the validatingWebhookConfiguration and deletes it. Returns an error if one occurs.
func (c *FakeValidatingWebhookConfigurations) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(validatingwebhookconfigurationsResource, name), &admissionregistrationv1.ValidatingWebhookConfiguration{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeValidatingWebhookConfigurations) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(validatingwebhookconfigurationsResource, listOptions)

	_, err := c.Fake.Invokes(action, &admissionregistrationv1.ValidatingWebhookConfigurationList{})
	return err
}

// Patch applies the patch and returns the patched validatingWebhookConfiguration.
func (c *FakeValidatingWebhookConfigurations) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *admissionregistrationv1.ValidatingWebhookConfiguration, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(validatingwebhookconfigurationsResource, name, pt, data, subresources...), &admissionregistrationv1.ValidatingWebhookConfiguration{})
	if obj == nil {
		return nil, err
	}
	return obj.(*admissionregistrationv1.ValidatingWebhookConfiguration), err
}

package fake

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
	v1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

// FakeMachineClasses implements MachineClassInterface
type FakeMachineClasses struct {
	Fake *FakeClusterV1alpha1
	ns   string
}

var machineclassesResource = schema.GroupVersionResource{Group: "cluster.k8s.io", Version: "v1alpha1", Resource: "machineclasses"}

var machineclassesKind = schema.GroupVersionKind{Group: "cluster.k8s.io", Version: "v1alpha1", Kind: "MachineClass"}

// Get takes name of the machineClass, and returns the corresponding machineClass object, and an error if there is any.
func (c *FakeMachineClasses) Get(name string, options v1.GetOptions) (result *v1alpha1.MachineClass, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(machineclassesResource, c.ns, name), &v1alpha1.MachineClass{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MachineClass), err
}

// List takes label and field selectors, and returns the list of MachineClasses that match those selectors.
func (c *FakeMachineClasses) List(opts v1.ListOptions) (result *v1alpha1.MachineClassList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(machineclassesResource, machineclassesKind, c.ns, opts), &v1alpha1.MachineClassList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.MachineClassList{}
	for _, item := range obj.(*v1alpha1.MachineClassList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested machineClasses.
func (c *FakeMachineClasses) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(machineclassesResource, c.ns, opts))

}

// Create takes the representation of a machineClass and creates it.  Returns the server's representation of the machineClass, and an error, if there is any.
func (c *FakeMachineClasses) Create(machineClass *v1alpha1.MachineClass) (result *v1alpha1.MachineClass, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(machineclassesResource, c.ns, machineClass), &v1alpha1.MachineClass{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MachineClass), err
}

// Update takes the representation of a machineClass and updates it. Returns the server's representation of the machineClass, and an error, if there is any.
func (c *FakeMachineClasses) Update(machineClass *v1alpha1.MachineClass) (result *v1alpha1.MachineClass, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(machineclassesResource, c.ns, machineClass), &v1alpha1.MachineClass{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MachineClass), err
}

// Delete takes name of the machineClass and deletes it. Returns an error if one occurs.
func (c *FakeMachineClasses) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(machineclassesResource, c.ns, name), &v1alpha1.MachineClass{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeMachineClasses) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(machineclassesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.MachineClassList{})
	return err
}

// Patch applies the patch and returns the patched machineClass.
func (c *FakeMachineClasses) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.MachineClass, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(machineclassesResource, c.ns, name, data, subresources...), &v1alpha1.MachineClass{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MachineClass), err
}

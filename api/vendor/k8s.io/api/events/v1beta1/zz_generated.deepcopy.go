// +build !ignore_autogenerated

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

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1beta1

import (
	v1 "k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Event) DeepCopyInto(out *Event) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.EventTime.DeepCopyInto(&out.EventTime)
	if in.Series != nil {
		in, out := &in.Series, &out.Series
		if *in == nil {
			*out = nil
		} else {
			*out = new(EventSeries)
			(*in).DeepCopyInto(*out)
		}
	}
	out.Regarding = in.Regarding
	if in.Related != nil {
		in, out := &in.Related, &out.Related
		if *in == nil {
			*out = nil
		} else {
			*out = new(v1.ObjectReference)
			**out = **in
		}
	}
	out.DeprecatedSource = in.DeprecatedSource
	in.DeprecatedFirstTimestamp.DeepCopyInto(&out.DeprecatedFirstTimestamp)
	in.DeprecatedLastTimestamp.DeepCopyInto(&out.DeprecatedLastTimestamp)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Event.
func (in *Event) DeepCopy() *Event {
	if in == nil {
		return nil
	}
	out := new(Event)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Event) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EventList) DeepCopyInto(out *EventList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Event, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EventList.
func (in *EventList) DeepCopy() *EventList {
	if in == nil {
		return nil
	}
	out := new(EventList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EventList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EventSeries) DeepCopyInto(out *EventSeries) {
	*out = *in
	in.LastObservedTime.DeepCopyInto(&out.LastObservedTime)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EventSeries.
func (in *EventSeries) DeepCopy() *EventSeries {
	if in == nil {
		return nil
	}
	out := new(EventSeries)
	in.DeepCopyInto(out)
	return out
}

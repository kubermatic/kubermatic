package kubernetes

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/golang/glog"
	"vbom.ml/util/sortorder"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/integer"
	"k8s.io/client-go/util/jsonpath"
	"k8s.io/kubernetes/pkg/printers"
)

func sortObjects(decoder runtime.Decoder, objs []runtime.Object, fieldInput string) (*runtimeSort, error) {
	for ix := range objs {
		item := objs[ix]
		switch u := item.(type) {
		case *runtime.Unknown:
			var err error
			// decode runtime.Unknown to runtime.Unstructured for sorting.
			// we don't actually want the internal versions of known types.
			if objs[ix], _, err = decoder.Decode(u.Raw, nil, &unstructured.Unstructured{}); err != nil {
				return nil, err
			}
		}
	}

	field, err := printers.RelaxedJSONPathExpression(fieldInput)
	if err != nil {
		return nil, err
	}

	parser := jsonpath.New("sorting").AllowMissingKeys(true)
	if err := parser.Parse(field); err != nil {
		return nil, err
	}

	// We don't do any model validation here, so we traverse all objects to be sorted
	// and, if the field is valid to at least one of them, we consider it to be a
	// valid field; otherwise error out.
	// Note that this requires empty fields to be considered later, when sorting.
	var fieldFoundOnce bool
	for _, obj := range objs {
		var values [][]reflect.Value
		if unstructured, ok := obj.(*unstructured.Unstructured); ok {
			values, err = parser.FindResults(unstructured.Object)
		} else {
			values, err = parser.FindResults(reflect.ValueOf(obj).Elem().Interface())
		}
		if err != nil {
			return nil, err
		}
		if len(values) > 0 && len(values[0]) > 0 {
			fieldFoundOnce = true
			break
		}
	}
	if !fieldFoundOnce {
		return nil, fmt.Errorf("couldn't find any field with path %q in the list of objects", field)
	}

	sorter := newRuntimeSort(field, objs)
	sort.Sort(sorter)
	return sorter, nil
}

// runtimeSort is an implementation of the golang sort interface that knows how to sort
// lists of runtime.Object
type runtimeSort struct {
	field        string
	objs         []runtime.Object
	origPosition []int
}

func newRuntimeSort(field string, objs []runtime.Object) *runtimeSort {
	sorter := &runtimeSort{field: field, objs: objs, origPosition: make([]int, len(objs))}
	for ix := range objs {
		sorter.origPosition[ix] = ix
	}
	return sorter
}

func (r *runtimeSort) Len() int {
	return len(r.objs)
}

func (r *runtimeSort) Swap(i, j int) {
	r.objs[i], r.objs[j] = r.objs[j], r.objs[i]
	r.origPosition[i], r.origPosition[j] = r.origPosition[j], r.origPosition[i]
}

func isLess(i, j reflect.Value) (bool, error) {
	switch i.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return i.Int() < j.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return i.Uint() < j.Uint(), nil
	case reflect.Float32, reflect.Float64:
		return i.Float() < j.Float(), nil
	case reflect.String:
		return sortorder.NaturalLess(i.String(), j.String()), nil
	case reflect.Ptr:
		return isLess(i.Elem(), j.Elem())
	case reflect.Struct:
		// sort metav1.Time
		in := i.Interface()
		if t, ok := in.(metav1.Time); ok {
			time := j.Interface().(metav1.Time)
			return t.Before(&time), nil
		}
		// fallback to the fields comparison
		for idx := 0; idx < i.NumField(); idx++ {
			less, err := isLess(i.Field(idx), j.Field(idx))
			if err != nil || !less {
				return less, err
			}
		}
		return true, nil
	case reflect.Array, reflect.Slice:
		// note: the length of i and j may be different
		for idx := 0; idx < integer.IntMin(i.Len(), j.Len()); idx++ {
			less, err := isLess(i.Index(idx), j.Index(idx))
			if err != nil || !less {
				return less, err
			}
		}
		return true, nil

	case reflect.Interface:
		switch itype := i.Interface().(type) {
		case uint8:
			if jtype, ok := j.Interface().(uint8); ok {
				return itype < jtype, nil
			}
		case uint16:
			if jtype, ok := j.Interface().(uint16); ok {
				return itype < jtype, nil
			}
		case uint32:
			if jtype, ok := j.Interface().(uint32); ok {
				return itype < jtype, nil
			}
		case uint64:
			if jtype, ok := j.Interface().(uint64); ok {
				return itype < jtype, nil
			}
		case int8:
			if jtype, ok := j.Interface().(int8); ok {
				return itype < jtype, nil
			}
		case int16:
			if jtype, ok := j.Interface().(int16); ok {
				return itype < jtype, nil
			}
		case int32:
			if jtype, ok := j.Interface().(int32); ok {
				return itype < jtype, nil
			}
		case int64:
			if jtype, ok := j.Interface().(int64); ok {
				return itype < jtype, nil
			}
		case uint:
			if jtype, ok := j.Interface().(uint); ok {
				return itype < jtype, nil
			}
		case int:
			if jtype, ok := j.Interface().(int); ok {
				return itype < jtype, nil
			}
		case float32:
			if jtype, ok := j.Interface().(float32); ok {
				return itype < jtype, nil
			}
		case float64:
			if jtype, ok := j.Interface().(float64); ok {
				return itype < jtype, nil
			}
		case string:
			if jtype, ok := j.Interface().(string); ok {
				return sortorder.NaturalLess(itype, jtype), nil
			}
		default:
			return false, fmt.Errorf("unsortable type: %T", itype)
		}
		return false, fmt.Errorf("unsortable interface: %v", i.Kind())

	default:
		return false, fmt.Errorf("unsortable type: %v", i.Kind())
	}
}

func (r *runtimeSort) Less(i, j int) bool {
	var err error

	iObj := r.objs[i]
	jObj := r.objs[j]

	parser := jsonpath.New("sorting").AllowMissingKeys(true)
	err = parser.Parse(r.field)
	if err != nil {
		glog.Fatalf("Unable to parse the filed %s", r.field)
	}

	var iValues [][]reflect.Value
	var jValues [][]reflect.Value

	if unstructured, ok := iObj.(*unstructured.Unstructured); ok {
		iValues, err = parser.FindResults(unstructured.Object)
	} else {
		iValues, err = parser.FindResults(reflect.ValueOf(iObj).Elem().Interface())
	}
	if err != nil {
		glog.Fatalf("Failed to get i values for %#v using %s (%#v)", iObj, r.field, err)
	}

	if unstructured, ok := jObj.(*unstructured.Unstructured); ok {
		jValues, err = parser.FindResults(unstructured.Object)
	} else {
		jValues, err = parser.FindResults(reflect.ValueOf(jObj).Elem().Interface())
	}
	if err != nil {
		glog.Fatalf("Failed to get j values for %#v using %s (%v)", jObj, r.field, err)
	}

	if len(iValues) == 0 || len(iValues[0]) == 0 {
		return true
	}
	if len(jValues) == 0 || len(jValues[0]) == 0 {
		return false
	}
	iField := iValues[0][0]
	jField := jValues[0][0]

	less, err := isLess(iField, jField)
	if err != nil {
		glog.Fatalf("Field %s in %T is an unsortable type: %s, err: %v", r.field, iObj, iField.Kind().String(), err)
	}
	return less
}

// Returns the starting (original) position of a particular index.  e.g. If originalPosition(0) returns 5 than the
// the item currently at position 0 was at position 5 in the original unsorted array.
func (r *runtimeSort) originalPosition(ix int) int {
	if ix < 0 || ix > len(r.origPosition) {
		return -1
	}
	return r.origPosition[ix]
}

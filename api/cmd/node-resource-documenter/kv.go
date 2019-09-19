package main

import (
	"fmt"
	"strconv"
)

// kv simulates the nested structs for template execution.
type kv map[string]interface{}

// postprocess transforms the generic types into via at()
// addressables.
func (nkv kv) postprocess() {
	for k, v := range nkv {
		switch tv := v.(type) {
		case map[string]interface{}:
			// Convert map into key/value recursively.
			kvv := kv(tv)
			kvv.postprocess()
			nkv[k] = kvv
		case []interface{}:
			// Convert slice into key/values with stringified indices.
			kvv := kv{}
			for sli, slv := range tv {
				ssli := strconv.Itoa(sli)
				switch tslv := slv.(type) {
				case map[string]interface{}:
					kvv[ssli] = kv(tslv)
				default:
					kvv[ssli] = tslv
				}
			}
			kvv.postprocess()
			nkv[k] = kvv
		default:
			// Don't do anything.
		}
	}
}

// at recursively retrieves a value from a nested key/value set.
func (nkv kv) at(ids ...string) interface{} {
	lids := len(ids)
	switch lids {
	case 0:
		return nil
	case 1:
		return nkv[ids[0]]
	default:
		v := nkv[ids[0]]
		switch tv := v.(type) {
		case kv:
			// Value is a nested key/value, dive deeper.
			return tv.at(ids[1:]...)
		default:
			// Return value as is.
			return v
		}
	}
}

// stringAt recursively retrieves a value from a nested key/value set
// and interprets it as a string.
func (nkv kv) stringAt(ids ...string) string {
	v := nkv.at(ids...)
	if v == nil {
		return "none"
	}
	return fmt.Sprintf("%v", v)
}

// kvAt recursively retrieves a value from a nested key/value set
// and interprets it as key/values.
func (nkv kv) kvAt(ids ...string) kv {
	v := nkv.at(ids...)
	if v == nil {
		return nil
	}
	at := v.(kv)
	at.postprocess()
	return at
}

// do iterates over the keys and values of the current key/value set.
func (nkv kv) do(f func(string, interface{})) {
	for k, v := range nkv {
		f(k, v)
	}
}

// exists returns true if the addressed value exists.
func (nkv kv) exists(ids ...string) bool {
	return nkv.at(ids...) != nil
}

// any returns true if an addressed sub-value exists.
func (nkv kv) any(ids ...string) bool {
	var ok bool
	nkv.do(func(k string, v interface{}) {
		skv := v.(kv)
		skv.postprocess()
		if skv.exists(ids...) {
			ok = true
		}
	})
	return ok
}

// fillKV creates the key/values needed for template execution.
func fillKV() kv {
	return kv{
		"Cluster": kv{
			"Spec": kv{
				"ClusterNetwork": kv{
					"Pods": kv{
						"CIDRBlocks": []string{"first"},
					},
				},
			},
		},
	}
}

package flag

// Flags is used to work with flag structs.
type Flags map[string]interface{}

// String will get a string flag value.
func (f Flags) String(key string) string {
	var res string
	if rawVal, ok := f[key]; ok {
		res = rawVal.(string)
	}
	return res
}

// StringSlice will get a []string flag value.
func (f Flags) StringSlice(key string) []string {
	var res []string
	if rawVal, ok := f[key]; ok {
		res = rawVal.([]string)
	}
	return res
}

// Int will get a int flag value.
func (f Flags) Int(key string) int {
	var res int
	if rawVal, ok := f[key]; ok {
		res = rawVal.(int)
	}
	return res
}

// Bool will get a bool flag value.
func (f Flags) Bool(key string) bool {
	var res bool
	rawVal, okRead := f[key]
	res, okCast := rawVal.(bool)
	return okRead && okCast && res
}

// Patch will combine two Flag sets.
// All (k, v) in the preset will add/overwrite the (k, v) in the base.
func Patch(base Flags, preset Flags) {
	for k, v := range preset {
		base[k] = v
	}
}

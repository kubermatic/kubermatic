package flagopts

import (
	"strings"
)

// StringArray is an implementation flag.Value for parsing comma separated flags
type StringArray []string

// String is flag.Value implementation method
func (s StringArray) String() string {
	return strings.Join(s, ",")
}

// Set is flag.Value implementation method
func (s *StringArray) Set(val string) error {
	tmp := strings.Split(val, ",")

	var result []string
	for _, item := range tmp {
		if item != "" {
			result = append(result, item)
		}
	}
	*s = result
	return nil
}

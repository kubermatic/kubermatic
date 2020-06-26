/*
Copyright 2019 The Machine Controller Authors.

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

package ini

import (
	"time"
)

// Duration is the encoding.TextUnmarshaler interface for time.Duration
type Duration struct {
	time.Duration
}

// UnmarshalText is used to convert from text to Duration
func (d *Duration) UnmarshalText(text []byte) error {
	res, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	d.Duration = res
	return nil
}

// MarshalText is used to convert from Duration to text
func (d *Duration) MarshalText() []byte {
	return []byte(d.Duration.String())
}

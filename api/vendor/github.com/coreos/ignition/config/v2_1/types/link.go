// Copyright 2017 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package types

import (
	"fmt"

	"github.com/coreos/ignition/config/validate/report"
)

func (s Link) Validate() report.Report {
	r := report.Report{}
	if !s.Hard {
		err := validatePath(s.Target)
		if err != nil {
			r.Add(report.Entry{
				Message: fmt.Sprintf("problem with target path %q: %v", s.Target, err),
				Kind:    report.EntryError,
			})
		}
	}
	return r
}

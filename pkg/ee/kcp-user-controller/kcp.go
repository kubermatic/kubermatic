//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2022 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package kcpusercontroller

import (
	"crypto/sha1"
	"encoding/binary"
	"regexp"
	"strings"

	"github.com/kcp-dev/logicalcluster/v2"
)

const (
	// bucketLevels is kcp's default value.
	bucketLevels = 2

	// bucketSize is kcp's default value.
	bucketSize = 2
)

/*
Copyright 2022 The KCP Authors.

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

// reHomeWorkspaceNameDisallowedChars is the regexp that defines what characters
// are disallowed in a home workspace name.
// Home workspace name is derived from the user name, with disallowed characters
// replaced.
// Adapted from kcp/pkg/server/home_workspaces.go getHomeLogicalClusterName()
var reHomeWorkspaceNameDisallowedChars = regexp.MustCompile("[^a-z0-9-]")

// Adapted from kcp/pkg/server/home_workspaces.go getHomeLogicalClusterName()
func getHomeLogicalClusterName(homePrefix logicalcluster.Name, userName string) logicalcluster.Name {
	bytes := sha1.Sum([]byte(userName))

	result := homePrefix
	for level := 0; level < bucketLevels; level++ {
		var bucketBytes = make([]byte, bucketSize)
		bucketBytesStart := level
		bucketCharInteger := binary.BigEndian.Uint32(bytes[bucketBytesStart : bucketBytesStart+4])
		for bucketCharIndex := 0; bucketCharIndex < bucketSize; bucketCharIndex++ {
			bucketChar := byte('a') + byte(bucketCharInteger%26)
			bucketBytes[bucketCharIndex] = bucketChar
			bucketCharInteger /= 26
		}
		result = result.Join(string(bucketBytes))
	}

	userName = reHomeWorkspaceNameDisallowedChars.ReplaceAllLiteralString(userName, "-")
	userName = strings.TrimLeftFunc(userName, func(r rune) bool {
		return r <= '9'
	})
	userName = strings.TrimRightFunc(userName, func(r rune) bool {
		return r == '-'
	})

	return result.Join(userName)
}

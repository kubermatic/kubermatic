/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package util

import (
	"io"
	"os"
)

func PrintFileUnbuffered(filename string) error {
	fd, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer fd.Close()
	return PrintUnbuffered(fd)
}

// printUnbuffered uses io.Copy to print data to stdout.
// It should be used for all bigger logs, to avoid buffering
// them in memory and getting oom killed because of that.
func PrintUnbuffered(src io.Reader) error {
	_, err := io.Copy(os.Stdout, src)
	return err
}

/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

/*
Package controller contains all our controllers. They are sorted by binary they run in, which means that
for all folders here a corresponding folder in the `cmd/` directory has to exist.

The only exception here are the `util` package which does not contain any controllers but some helpers
and the `shared` package which contains controllers that run within more than one binary.
*/
package controller

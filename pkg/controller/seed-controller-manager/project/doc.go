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

/*
Package project contains a controller responsible for removing all
clusters in a project whenever a project is deleted, and (importantly)
waiting until all clusters are gone before releasing the project.

Note that the project-synchronizer controller in the master-ctrl-mgr
takes care of synchronizing the project deletion to all seeds (i.e.
you delete a project on the master, the project-synchronizer controller
then deletes the projects on all seeds, and then this controller cleans
them up by deleting the clusters).
*/
package project

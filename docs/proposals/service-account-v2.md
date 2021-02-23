# Service Accounts V2
**Author**: Lukasz Zajaczkowski(@zreigz)

**Status**: Draft proposal.

## Introduction

A service account is a special type of account intended to represent a non-human user that needs to authenticate and be
authorized to access resources in Kubermatic APIs.

Service accounts are used in scenarios such as:

 - Running workloads on user clusters.
 - Running workloads which are not tied to the lifecycle of a human user.

## Core concept
A service accounts will be considered as a main resource. Only the human user can create a service account.
There is no need to create a new groups for SA, we want to assign a service account to one of the already defined groups:
`owners`, `editors` or `viewers`.

The current service account belongs to the project and can not create it. The new implementation will change it. To keep 
consistent API we have to add new endpoints for service account management. We will use V2 router.
The new service account path will be `/api/v2/serviceaccounts` instead of: `/api/v1/projects/{project_id}/serviceaccounts`.

During service account creation by the human user, the service account will be bound with desired privileges to the all
owned by user projects.

Service account can create a project. The new service account consists label with a human owner to be able to bind a new
created project for the human user as owner. The controller will create also the binding to the service account. Only human
user will be displayed in the UI.


```
apiVersion: kubermatic.k8s.io/v1
kind: User
metadata:
  creationTimestamp: "2021-01-12T14:02:27Z"
  generation: 1
  name: serviceaccount-wv8gptgdl6
  labels:
    owner: user@kubermatic.com
    role: owner
spec:
  admin: false
  email: serviceaccount-wv8gptgdl6@dev.kubermatic.io
  id: wv8gptgdl6
  name: test
```


### Migration

The existing service account is bound only to exactly one project. The controller will create additional user project
bindings for the owned projects only when a label with the human user exists. Because we have V1 and V2 routers the current
solution can stay as it is now and doesn't require any migration. End-user will use V1 endpoints for existing service accounts.
We can keep service accounts for the project and as a main resource.


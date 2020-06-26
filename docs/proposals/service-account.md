# Service Accounts
**Author**: Lukasz Zajaczkowski(@zreigz)

**Status**: Draft proposal.

## Introduction
A service account is a special type of user account that belongs to the Kubermatic project, instead of to an individual
end user. Your project resources assume the identity of the service account to call Kubermatic APIs, so that the users
are not directly involved. A service account has JWT token which is used to authenticate to Kubermatic API.

## Core concept
A Service accounts are considered as project's resource. Only the owner of a project  can create a service account.
There is no need to create a new groups for SA, we want to assign a service account to one of the already defined groups:
`owners`, `editors` or `viewers`.

## Implementation

The Kubermatic User object is used as a service account. To avoid confusions about the purpose of the user the name convention
was introduced. Service account name starts with prefix `serviceaccount-`. The Regular user starts with name: `user-`.
For example:

```bash
$ kubectl get users
NAME                                                               AGE
serviceaccount-z97l228h4z                                          7d
serviceaccount-zjl54fmlks                                          26d
user-26xq2                                                         311d
```

A service account is linked to the project automatically by service account binding controller. The controller creates
`UserProjectBinding` which specifies a binding between a service account and a project. A `UserProjectBinding` uses a
`OwnerRef` to create connection with the project. A service account will be automatically deleted after project removal.

The `yaml` example of service account object:

```yaml
apiVersion: kubermatic.k8s.io/v1
kind: User
metadata:
  creationTimestamp: "2019-03-27T07:57:55Z"
  generation: 1
  name: serviceaccount-xxxxxxxxxx
  ownerReferences:
  - apiVersion: kubermatic.k8s.io/v1
    kind: Project
    name: yyyyyyyyyy
    uid: c7694392-43e4-11e9-b04b-42010a9c0119
spec:
  email: serviceaccount-xxxxxxxxxx@localhost
  id: 3fa771ea25b4a2065ace5f3d508b2335d450402f0d73d5e59fa84b41_KUBE
  name: test
```

Service accounts are tied to a set of credentials stored as Secrets. Because a `Secret` is namespaced resource the
system needs predefined namespace for it: `kubermatic`.

Secret label `project-id` is used to create link between secret and project. The `OwnerRef` links the secret with the
service account. A secret will be automatically deleted after service account removal.

```yaml
 apiVersion: v1
 data:
   token: abcdefgh=
 kind: Secret
 metadata:
   labels:
     name: test
     project-id: yyyyyyyyyy
   name: sa-token-zzzzzzzzzz
   namespace: kubermatic
   ownerReferences:
   - apiVersion: kubermatic.k8s.io/v1
     kind: User
     name: serviceaccount-xxxxxxxxxx
     uid: 26127a31-507a-11e9-9ea9-42010a9c0125
 type: Opaque

```

A service account is an automatically enabled authenticator that uses signed bearer tokens to verify requests. The Kubermatic API takes a flag:

   - service-account-signing-key A signing key authenticates the service account's token value using HMAC. It is recommended to use a key with 32 bytes or longer.


 ### API endpoints
 #### Add SA
```
POST /api/v1/projects/{project_id}/serviceaccounts
 Consumes:
  - application/json

 Produces:
  - application/json
```

Body json example:
```json
{"name":"test","group":"editors"}
```
A service account can belongs to the one the following group: `viewers` or `editors`


#### List SA
```
GET /api/v1/projects/{project_id}/serviceaccounts
 Produces:
  - application/json

```

#### Update SA
```
PUT /api/v1/projects/{project_id}/serviceaccounts/{serviceaccount_id}
 Consumes:
  - application/json

 Produces:
  - application/json
```
Body json example:
```json
  {"name":"test","group":"editors","id":"serviceaccount_id"}
```
A service account can belongs to the one the following group: `viewers` or `editors`

#### Delete SA

```
Delete /api/v1/projects/{project_id}/serviceaccounts/{serviceaccount_id}
 Produces:
  - application/json
```

#### Create token

```
POST /api/v1/projects/{project_id}/serviceaccounts/{serviceaccount_id}/tokens
 Consumes:
  - application/json

 Produces:
  - application/json
```

This is the one of the place when user can display authentication token.

Body json example:
```json
  {"name":"test"}
```

#### Update token
If user lost token or forget then must call this endpoint to revoke the old one and generate/display the new one.
```
PUT /api/v1/projects/{project_id}/serviceaccounts/{serviceaccount_id}/tokens/{token_id}
 Consumes:
  - application/json

 Produces:
  - application/json
```

User can regenerate token and change internal token name:
```json
  {"name":"new name","id":"token_id"}
```

Empty `body` request only regenerates the token.

#### Patch token

This endpoint is generally used to change the token name.

```
PATCH /api/v1/projects/{project_id}/serviceaccounts/{serviceaccount_id}/tokens/{token_id}
 Consumes:
  - application/json

 Produces:
  - application/json
```

Body json example:

```json
  {"name":"new name"}
```

#### Delete token

```
DELETE /api/v1/projects/{project_id}/serviceaccounts/{serviceaccount_id}/tokens/{token_id}
 Produces:
  - application/json
```

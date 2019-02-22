package resources
const policyJSON = `
{
    "kind": "List",
    "apiVersion": "v1",
    "metadata": {},
    "items": [
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "sudoer",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "impersonate"
                    ],
                    "apiGroups": [
                        "",
                        "user.openshift.io"
                    ],
                    "resources": [
                        "systemusers",
                        "users"
                    ],
                    "resourceNames": [
                        "system:admin"
                    ]
                },
                {
                    "verbs": [
                        "impersonate"
                    ],
                    "apiGroups": [
                        "",
                        "user.openshift.io"
                    ],
                    "resources": [
                        "groups",
                        "systemgroups"
                    ],
                    "resourceNames": [
                        "system:masters"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:scope-impersonation",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "impersonate"
                    ],
                    "apiGroups": [
                        "authentication.k8s.io"
                    ],
                    "resources": [
                        "userextras/scopes.authorization.openshift.io"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "cluster-reader",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": null,
            "aggregationRule": {
                "clusterRoleSelectors": [
                    {
                        "matchLabels": {
                            "rbac.authorization.k8s.io/aggregate-to-cluster-reader": "true"
                        }
                    },
                    {
                        "matchLabels": {
                            "rbac.authorization.k8s.io/aggregate-to-view": "true"
                        }
                    }
                ]
            }
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:aggregate-to-cluster-reader",
                "creationTimestamp": null,
                "labels": {
                    "rbac.authorization.k8s.io/aggregate-to-cluster-reader": "true"
                },
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "componentstatuses",
                        "nodes",
                        "nodes/status",
                        "persistentvolumeclaims/status",
                        "persistentvolumes",
                        "persistentvolumes/status",
                        "pods/binding",
                        "pods/eviction",
                        "podtemplates",
                        "securitycontextconstraints",
                        "services/status"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "admissionregistration.k8s.io"
                    ],
                    "resources": [
                        "mutatingwebhookconfigurations",
                        "validatingwebhookconfigurations"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "apps"
                    ],
                    "resources": [
                        "controllerrevisions",
                        "daemonsets/status",
                        "deployments/status",
                        "replicasets/status",
                        "statefulsets/status"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "apiextensions.k8s.io"
                    ],
                    "resources": [
                        "customresourcedefinitions",
                        "customresourcedefinitions/status"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "apiregistration.k8s.io"
                    ],
                    "resources": [
                        "apiservices",
                        "apiservices/status"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "autoscaling"
                    ],
                    "resources": [
                        "horizontalpodautoscalers/status"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "batch"
                    ],
                    "resources": [
                        "cronjobs/status",
                        "jobs/status"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "extensions"
                    ],
                    "resources": [
                        "daemonsets/status",
                        "deployments/status",
                        "horizontalpodautoscalers",
                        "horizontalpodautoscalers/status",
                        "ingresses/status",
                        "jobs",
                        "jobs/status",
                        "podsecuritypolicies",
                        "replicasets/status",
                        "replicationcontrollers",
                        "storageclasses",
                        "thirdpartyresources"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "events.k8s.io"
                    ],
                    "resources": [
                        "events"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "policy"
                    ],
                    "resources": [
                        "poddisruptionbudgets/status",
                        "podsecuritypolicies"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "rbac.authorization.k8s.io"
                    ],
                    "resources": [
                        "clusterrolebindings",
                        "clusterroles",
                        "rolebindings",
                        "roles"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "settings.k8s.io"
                    ],
                    "resources": [
                        "podpresets"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "storage.k8s.io"
                    ],
                    "resources": [
                        "storageclasses",
                        "volumeattachments"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "scheduling.k8s.io"
                    ],
                    "resources": [
                        "priorityclasses"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "certificates.k8s.io"
                    ],
                    "resources": [
                        "certificatesigningrequests",
                        "certificatesigningrequests/approval",
                        "certificatesigningrequests/status"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "authorization.openshift.io"
                    ],
                    "resources": [
                        "clusterrolebindings",
                        "clusterroles",
                        "rolebindingrestrictions",
                        "rolebindings",
                        "roles"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "build.openshift.io"
                    ],
                    "resources": [
                        "builds/details"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "images",
                        "imagesignatures"
                    ]
                },
                {
                    "verbs": [
                        "get"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "imagestreams/layers"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "oauth.openshift.io"
                    ],
                    "resources": [
                        "oauthclientauthorizations"
                    ]
                },
                {
                    "verbs": [
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "project.openshift.io"
                    ],
                    "resources": [
                        "projects"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "project.openshift.io"
                    ],
                    "resources": [
                        "projectrequests"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "quota.openshift.io"
                    ],
                    "resources": [
                        "clusterresourcequotas",
                        "clusterresourcequotas/status"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "network.openshift.io"
                    ],
                    "resources": [
                        "clusternetworks",
                        "egressnetworkpolicies",
                        "hostsubnets",
                        "netnamespaces"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "security.openshift.io"
                    ],
                    "resources": [
                        "securitycontextconstraints"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "security.openshift.io"
                    ],
                    "resources": [
                        "rangeallocations"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "template.openshift.io"
                    ],
                    "resources": [
                        "brokertemplateinstances",
                        "templateinstances/status"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "user.openshift.io"
                    ],
                    "resources": [
                        "groups",
                        "identities",
                        "useridentitymappings",
                        "users"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "",
                        "authorization.openshift.io"
                    ],
                    "resources": [
                        "localresourceaccessreviews",
                        "localsubjectaccessreviews",
                        "resourceaccessreviews",
                        "selfsubjectrulesreviews",
                        "subjectaccessreviews",
                        "subjectrulesreviews"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "authorization.k8s.io"
                    ],
                    "resources": [
                        "localsubjectaccessreviews",
                        "selfsubjectaccessreviews",
                        "selfsubjectrulesreviews",
                        "subjectaccessreviews"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "authentication.k8s.io"
                    ],
                    "resources": [
                        "tokenreviews"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "",
                        "security.openshift.io"
                    ],
                    "resources": [
                        "podsecuritypolicyreviews",
                        "podsecuritypolicyselfsubjectreviews",
                        "podsecuritypolicysubjectreviews"
                    ]
                },
                {
                    "verbs": [
                        "get"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "nodes/metrics",
                        "nodes/spec"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "get"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "nodes/stats"
                    ]
                },
                {
                    "verbs": [
                        "get"
                    ],
                    "nonResourceURLs": [
                        "*"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "cluster-debugger",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get"
                    ],
                    "nonResourceURLs": [
                        "/debug/pprof",
                        "/debug/pprof/*",
                        "/metrics"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:build-strategy-docker",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "",
                        "build.openshift.io"
                    ],
                    "resources": [
                        "builds/docker",
                        "builds/optimizeddocker"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:build-strategy-custom",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "",
                        "build.openshift.io"
                    ],
                    "resources": [
                        "builds/custom"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:build-strategy-source",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "",
                        "build.openshift.io"
                    ],
                    "resources": [
                        "builds/source"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:build-strategy-jenkinspipeline",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "",
                        "build.openshift.io"
                    ],
                    "resources": [
                        "builds/jenkinspipeline"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "storage-admin",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "persistentvolumes"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "storage.k8s.io"
                    ],
                    "resources": [
                        "storageclasses"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events",
                        "persistentvolumeclaims"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:aggregate-to-admin",
                "creationTimestamp": null,
                "labels": {
                    "rbac.authorization.k8s.io/aggregate-to-admin": "true"
                },
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "authorization.openshift.io"
                    ],
                    "resources": [
                        "rolebindings",
                        "roles"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "",
                        "authorization.openshift.io"
                    ],
                    "resources": [
                        "localresourceaccessreviews",
                        "localsubjectaccessreviews",
                        "subjectrulesreviews"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "",
                        "security.openshift.io"
                    ],
                    "resources": [
                        "podsecuritypolicyreviews",
                        "podsecuritypolicyselfsubjectreviews",
                        "podsecuritypolicysubjectreviews"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "authorization.openshift.io"
                    ],
                    "resources": [
                        "rolebindingrestrictions"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "build.openshift.io"
                    ],
                    "resources": [
                        "buildconfigs",
                        "buildconfigs/webhooks",
                        "builds"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "build.openshift.io"
                    ],
                    "resources": [
                        "builds/log"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "",
                        "build.openshift.io"
                    ],
                    "resources": [
                        "buildconfigs/instantiate",
                        "buildconfigs/instantiatebinary",
                        "builds/clone"
                    ]
                },
                {
                    "verbs": [
                        "update"
                    ],
                    "apiGroups": [
                        "",
                        "build.openshift.io"
                    ],
                    "resources": [
                        "builds/details"
                    ]
                },
                {
                    "verbs": [
                        "admin",
                        "edit",
                        "view"
                    ],
                    "apiGroups": [
                        "build.openshift.io"
                    ],
                    "resources": [
                        "jenkins"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "apps.openshift.io"
                    ],
                    "resources": [
                        "deploymentconfigs",
                        "deploymentconfigs/scale"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "",
                        "apps.openshift.io"
                    ],
                    "resources": [
                        "deploymentconfigrollbacks",
                        "deploymentconfigs/instantiate",
                        "deploymentconfigs/rollback"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "apps.openshift.io"
                    ],
                    "resources": [
                        "deploymentconfigs/log",
                        "deploymentconfigs/status"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "imagestreamimages",
                        "imagestreammappings",
                        "imagestreams",
                        "imagestreams/secrets",
                        "imagestreamtags"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "imagestreams/status"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "update"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "imagestreams/layers"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "imagestreamimports"
                    ]
                },
                {
                    "verbs": [
                        "delete",
                        "get",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        "",
                        "project.openshift.io"
                    ],
                    "resources": [
                        "projects"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "quota.openshift.io"
                    ],
                    "resources": [
                        "appliedclusterresourcequotas"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "route.openshift.io"
                    ],
                    "resources": [
                        "routes"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "",
                        "route.openshift.io"
                    ],
                    "resources": [
                        "routes/custom-host"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "route.openshift.io"
                    ],
                    "resources": [
                        "routes/status"
                    ]
                },
                {
                    "verbs": [
                        "update"
                    ],
                    "apiGroups": [
                        "",
                        "route.openshift.io"
                    ],
                    "resources": [
                        "routes/status"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "template.openshift.io"
                    ],
                    "resources": [
                        "processedtemplates",
                        "templateconfigs",
                        "templateinstances",
                        "templates"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "extensions",
                        "networking.k8s.io"
                    ],
                    "resources": [
                        "networkpolicies"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "build.openshift.io"
                    ],
                    "resources": [
                        "buildlogs"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "resourcequotausages"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "",
                        "authorization.openshift.io"
                    ],
                    "resources": [
                        "resourceaccessreviews",
                        "subjectaccessreviews"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:aggregate-to-edit",
                "creationTimestamp": null,
                "labels": {
                    "rbac.authorization.k8s.io/aggregate-to-edit": "true"
                },
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "build.openshift.io"
                    ],
                    "resources": [
                        "buildconfigs",
                        "buildconfigs/webhooks",
                        "builds"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "build.openshift.io"
                    ],
                    "resources": [
                        "builds/log"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "",
                        "build.openshift.io"
                    ],
                    "resources": [
                        "buildconfigs/instantiate",
                        "buildconfigs/instantiatebinary",
                        "builds/clone"
                    ]
                },
                {
                    "verbs": [
                        "update"
                    ],
                    "apiGroups": [
                        "",
                        "build.openshift.io"
                    ],
                    "resources": [
                        "builds/details"
                    ]
                },
                {
                    "verbs": [
                        "edit",
                        "view"
                    ],
                    "apiGroups": [
                        "build.openshift.io"
                    ],
                    "resources": [
                        "jenkins"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "apps.openshift.io"
                    ],
                    "resources": [
                        "deploymentconfigs",
                        "deploymentconfigs/scale"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "",
                        "apps.openshift.io"
                    ],
                    "resources": [
                        "deploymentconfigrollbacks",
                        "deploymentconfigs/instantiate",
                        "deploymentconfigs/rollback"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "apps.openshift.io"
                    ],
                    "resources": [
                        "deploymentconfigs/log",
                        "deploymentconfigs/status"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "imagestreamimages",
                        "imagestreammappings",
                        "imagestreams",
                        "imagestreams/secrets",
                        "imagestreamtags"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "imagestreams/status"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "update"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "imagestreams/layers"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "imagestreamimports"
                    ]
                },
                {
                    "verbs": [
                        "get"
                    ],
                    "apiGroups": [
                        "",
                        "project.openshift.io"
                    ],
                    "resources": [
                        "projects"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "quota.openshift.io"
                    ],
                    "resources": [
                        "appliedclusterresourcequotas"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "route.openshift.io"
                    ],
                    "resources": [
                        "routes"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "",
                        "route.openshift.io"
                    ],
                    "resources": [
                        "routes/custom-host"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "route.openshift.io"
                    ],
                    "resources": [
                        "routes/status"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "template.openshift.io"
                    ],
                    "resources": [
                        "processedtemplates",
                        "templateconfigs",
                        "templateinstances",
                        "templates"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "extensions",
                        "networking.k8s.io"
                    ],
                    "resources": [
                        "networkpolicies"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "build.openshift.io"
                    ],
                    "resources": [
                        "buildlogs"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "resourcequotausages"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:aggregate-to-view",
                "creationTimestamp": null,
                "labels": {
                    "rbac.authorization.k8s.io/aggregate-to-view": "true"
                },
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "build.openshift.io"
                    ],
                    "resources": [
                        "buildconfigs",
                        "buildconfigs/webhooks",
                        "builds"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "build.openshift.io"
                    ],
                    "resources": [
                        "builds/log"
                    ]
                },
                {
                    "verbs": [
                        "view"
                    ],
                    "apiGroups": [
                        "build.openshift.io"
                    ],
                    "resources": [
                        "jenkins"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "apps.openshift.io"
                    ],
                    "resources": [
                        "deploymentconfigs",
                        "deploymentconfigs/scale"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "apps.openshift.io"
                    ],
                    "resources": [
                        "deploymentconfigs/log",
                        "deploymentconfigs/status"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "imagestreamimages",
                        "imagestreammappings",
                        "imagestreams",
                        "imagestreamtags"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "imagestreams/status"
                    ]
                },
                {
                    "verbs": [
                        "get"
                    ],
                    "apiGroups": [
                        "",
                        "project.openshift.io"
                    ],
                    "resources": [
                        "projects"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "quota.openshift.io"
                    ],
                    "resources": [
                        "appliedclusterresourcequotas"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "route.openshift.io"
                    ],
                    "resources": [
                        "routes"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "route.openshift.io"
                    ],
                    "resources": [
                        "routes/status"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "template.openshift.io"
                    ],
                    "resources": [
                        "processedtemplates",
                        "templateconfigs",
                        "templateinstances",
                        "templates"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "build.openshift.io"
                    ],
                    "resources": [
                        "buildlogs"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "resourcequotausages"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "basic-user",
                "creationTimestamp": null,
                "annotations": {
                    "openshift.io/description": "A user that can get basic information about projects.",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get"
                    ],
                    "apiGroups": [
                        "",
                        "user.openshift.io"
                    ],
                    "resources": [
                        "users"
                    ],
                    "resourceNames": [
                        "~"
                    ]
                },
                {
                    "verbs": [
                        "list"
                    ],
                    "apiGroups": [
                        "",
                        "project.openshift.io"
                    ],
                    "resources": [
                        "projectrequests"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list"
                    ],
                    "apiGroups": [
                        "",
                        "authorization.openshift.io"
                    ],
                    "resources": [
                        "clusterroles"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "rbac.authorization.k8s.io"
                    ],
                    "resources": [
                        "clusterroles"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list"
                    ],
                    "apiGroups": [
                        "storage.k8s.io"
                    ],
                    "resources": [
                        "storageclasses"
                    ]
                },
                {
                    "verbs": [
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "project.openshift.io"
                    ],
                    "resources": [
                        "projects"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "",
                        "authorization.openshift.io"
                    ],
                    "resources": [
                        "selfsubjectrulesreviews"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "authorization.k8s.io"
                    ],
                    "resources": [
                        "selfsubjectaccessreviews"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "self-access-reviewer",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "",
                        "authorization.openshift.io"
                    ],
                    "resources": [
                        "selfsubjectrulesreviews"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "authorization.k8s.io"
                    ],
                    "resources": [
                        "selfsubjectaccessreviews"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "self-provisioner",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "openshift.io/description": "A user that can request projects.",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "",
                        "project.openshift.io"
                    ],
                    "resources": [
                        "projectrequests"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "cluster-status",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "openshift.io/description": "A user that can get basic cluster status information.",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get"
                    ],
                    "nonResourceURLs": [
                        "/healthz",
                        "/healthz/*"
                    ]
                },
                {
                    "verbs": [
                        "get"
                    ],
                    "nonResourceURLs": [
                        "/version",
                        "/version/*",
                        "/api",
                        "/api/*",
                        "/apis",
                        "/apis/*",
                        "/oapi",
                        "/oapi/*",
                        "/openapi/v2",
                        "/swaggerapi",
                        "/swaggerapi/*",
                        "/swagger.json",
                        "/swagger-2.0.0.pb-v1",
                        "/osapi",
                        "/osapi/",
                        "/.well-known",
                        "/.well-known/*",
                        "/"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:image-auditor",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "images"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:image-puller",
                "creationTimestamp": null,
                "annotations": {
                    "openshift.io/description": "Grants the right to pull images from within a project.",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "imagestreams/layers"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:image-pusher",
                "creationTimestamp": null,
                "annotations": {
                    "openshift.io/description": "Grants the right to push and pull images from within a project.",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get",
                        "update"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "imagestreams/layers"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:image-builder",
                "creationTimestamp": null,
                "annotations": {
                    "openshift.io/description": "Grants the right to build, push and pull images from within a project.  Used primarily with service accounts for builds.",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get",
                        "update"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "imagestreams/layers"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "imagestreams"
                    ]
                },
                {
                    "verbs": [
                        "update"
                    ],
                    "apiGroups": [
                        "",
                        "build.openshift.io"
                    ],
                    "resources": [
                        "builds/details"
                    ]
                },
                {
                    "verbs": [
                        "get"
                    ],
                    "apiGroups": [
                        "",
                        "build.openshift.io"
                    ],
                    "resources": [
                        "builds"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:image-pruner",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get",
                        "list"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "pods",
                        "replicationcontrollers"
                    ]
                },
                {
                    "verbs": [
                        "list"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "limitranges"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list"
                    ],
                    "apiGroups": [
                        "",
                        "build.openshift.io"
                    ],
                    "resources": [
                        "buildconfigs",
                        "builds"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list"
                    ],
                    "apiGroups": [
                        "",
                        "apps.openshift.io"
                    ],
                    "resources": [
                        "deploymentconfigs"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list"
                    ],
                    "apiGroups": [
                        "apps",
                        "extensions"
                    ],
                    "resources": [
                        "daemonsets"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list"
                    ],
                    "apiGroups": [
                        "apps",
                        "extensions"
                    ],
                    "resources": [
                        "deployments"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list"
                    ],
                    "apiGroups": [
                        "apps",
                        "extensions"
                    ],
                    "resources": [
                        "replicasets"
                    ]
                },
                {
                    "verbs": [
                        "delete"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "images"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "images",
                        "imagestreams"
                    ]
                },
                {
                    "verbs": [
                        "update"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "imagestreams/status"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:image-signer",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "images",
                        "imagestreams/layers"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "imagesignatures"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:deployer",
                "creationTimestamp": null,
                "annotations": {
                    "openshift.io/description": "Grants the right to deploy within a project.  Used primarily with service accounts for automated deployments.",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "delete"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "replicationcontrollers"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "replicationcontrollers"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "replicationcontrollers/scale"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "pods"
                    ]
                },
                {
                    "verbs": [
                        "get"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "pods/log"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "list"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                },
                {
                    "verbs": [
                        "update"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "imagestreamtags"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:master",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "*"
                    ],
                    "apiGroups": [
                        "*"
                    ],
                    "resources": [
                        "*"
                    ]
                },
                {
                    "verbs": [
                        "*"
                    ],
                    "nonResourceURLs": [
                        "*"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:oauth-token-deleter",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "delete"
                    ],
                    "apiGroups": [
                        "",
                        "oauth.openshift.io"
                    ],
                    "resources": [
                        "oauthaccesstokens",
                        "oauthauthorizetokens"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:router",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "endpoints"
                    ]
                },
                {
                    "verbs": [
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "services"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "authentication.k8s.io"
                    ],
                    "resources": [
                        "tokenreviews"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "authorization.k8s.io"
                    ],
                    "resources": [
                        "subjectaccessreviews"
                    ]
                },
                {
                    "verbs": [
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "route.openshift.io"
                    ],
                    "resources": [
                        "routes"
                    ]
                },
                {
                    "verbs": [
                        "update"
                    ],
                    "apiGroups": [
                        "",
                        "route.openshift.io"
                    ],
                    "resources": [
                        "routes/status"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:registry",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "list"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "imagestreams",
                        "limitranges",
                        "resourcequotas"
                    ]
                },
                {
                    "verbs": [
                        "delete",
                        "get"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "images",
                        "imagestreamtags"
                    ]
                },
                {
                    "verbs": [
                        "get"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "imagestreamimages",
                        "imagestreams/secrets"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "update"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "images",
                        "imagestreams"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "imagestreammappings"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:node-admin",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "nodes"
                    ]
                },
                {
                    "verbs": [
                        "proxy"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "nodes"
                    ]
                },
                {
                    "verbs": [
                        "*"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "nodes/log",
                        "nodes/metrics",
                        "nodes/proxy",
                        "nodes/spec",
                        "nodes/stats"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:node-reader",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "nodes"
                    ]
                },
                {
                    "verbs": [
                        "get"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "nodes/metrics",
                        "nodes/spec"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "get"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "nodes/stats"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:sdn-reader",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "network.openshift.io"
                    ],
                    "resources": [
                        "egressnetworkpolicies",
                        "hostsubnets",
                        "netnamespaces"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "namespaces",
                        "nodes"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "extensions"
                    ],
                    "resources": [
                        "networkpolicies"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "networking.k8s.io"
                    ],
                    "resources": [
                        "networkpolicies"
                    ]
                },
                {
                    "verbs": [
                        "get"
                    ],
                    "apiGroups": [
                        "",
                        "network.openshift.io"
                    ],
                    "resources": [
                        "clusternetworks"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:sdn-manager",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create",
                        "delete",
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "network.openshift.io"
                    ],
                    "resources": [
                        "hostsubnets",
                        "netnamespaces"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "get"
                    ],
                    "apiGroups": [
                        "",
                        "network.openshift.io"
                    ],
                    "resources": [
                        "clusternetworks"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "nodes"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:webhook",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create",
                        "get"
                    ],
                    "apiGroups": [
                        "",
                        "build.openshift.io"
                    ],
                    "resources": [
                        "buildconfigs/webhooks"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:discovery",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get"
                    ],
                    "nonResourceURLs": [
                        "/version",
                        "/version/*",
                        "/api",
                        "/api/*",
                        "/apis",
                        "/apis/*",
                        "/oapi",
                        "/oapi/*",
                        "/openapi/v2",
                        "/swaggerapi",
                        "/swaggerapi/*",
                        "/swagger.json",
                        "/swagger-2.0.0.pb-v1",
                        "/osapi",
                        "/osapi/",
                        "/.well-known",
                        "/.well-known/*",
                        "/"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "registry-admin",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "secrets",
                        "serviceaccounts"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "imagestreamimages",
                        "imagestreammappings",
                        "imagestreams",
                        "imagestreams/secrets",
                        "imagestreamtags"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "imagestreamimports"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "update"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "imagestreams/layers"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "authorization.openshift.io"
                    ],
                    "resources": [
                        "rolebindings",
                        "roles"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "rbac.authorization.k8s.io"
                    ],
                    "resources": [
                        "rolebindings",
                        "roles"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "",
                        "authorization.openshift.io"
                    ],
                    "resources": [
                        "localresourceaccessreviews",
                        "localsubjectaccessreviews",
                        "subjectrulesreviews"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "authorization.k8s.io"
                    ],
                    "resources": [
                        "localsubjectaccessreviews"
                    ]
                },
                {
                    "verbs": [
                        "get"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "namespaces"
                    ]
                },
                {
                    "verbs": [
                        "delete",
                        "get"
                    ],
                    "apiGroups": [
                        "",
                        "project.openshift.io"
                    ],
                    "resources": [
                        "projects"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "",
                        "authorization.openshift.io"
                    ],
                    "resources": [
                        "resourceaccessreviews",
                        "subjectaccessreviews"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "registry-editor",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "secrets",
                        "serviceaccounts"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "imagestreamimages",
                        "imagestreammappings",
                        "imagestreams",
                        "imagestreams/secrets",
                        "imagestreamtags"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "imagestreamimports"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "update"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "imagestreams/layers"
                    ]
                },
                {
                    "verbs": [
                        "get"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "namespaces"
                    ]
                },
                {
                    "verbs": [
                        "get"
                    ],
                    "apiGroups": [
                        "",
                        "project.openshift.io"
                    ],
                    "resources": [
                        "projects"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "registry-viewer",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "imagestreamimages",
                        "imagestreammappings",
                        "imagestreams",
                        "imagestreamtags"
                    ]
                },
                {
                    "verbs": [
                        "get"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "imagestreams/layers"
                    ]
                },
                {
                    "verbs": [
                        "get"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "namespaces"
                    ]
                },
                {
                    "verbs": [
                        "get"
                    ],
                    "apiGroups": [
                        "",
                        "project.openshift.io"
                    ],
                    "resources": [
                        "projects"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:templateservicebroker-client",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "delete",
                        "get",
                        "put",
                        "update"
                    ],
                    "nonResourceURLs": [
                        "/brokers/template.openshift.io/*"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:replication-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": null
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:endpoint-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": null
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:replicaset-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": null
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:garbage-collector-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": null
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:job-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": null
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:hpa-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": null
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:daemonset-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": null
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:disruption-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": null
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:namespace-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": null
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:gc-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": null
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:certificate-signing-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": null
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:statefulset-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": null
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:build-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": null
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:deploymentconfig-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": null
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:deployment-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": null
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:build-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "delete",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "build.openshift.io"
                    ],
                    "resources": [
                        "builds"
                    ]
                },
                {
                    "verbs": [
                        "update"
                    ],
                    "apiGroups": [
                        "",
                        "build.openshift.io"
                    ],
                    "resources": [
                        "builds/finalizers"
                    ]
                },
                {
                    "verbs": [
                        "get"
                    ],
                    "apiGroups": [
                        "",
                        "build.openshift.io"
                    ],
                    "resources": [
                        "buildconfigs"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "",
                        "build.openshift.io"
                    ],
                    "resources": [
                        "builds/custom",
                        "builds/docker",
                        "builds/jenkinspipeline",
                        "builds/optimizeddocker",
                        "builds/source"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "imagestreams"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "secrets"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "configmaps"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "get",
                        "list"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "pods"
                    ]
                },
                {
                    "verbs": [
                        "get"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "namespaces"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "serviceaccounts"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "",
                        "security.openshift.io"
                    ],
                    "resources": [
                        "podsecuritypolicysubjectreviews"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:build-config-change-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "build.openshift.io"
                    ],
                    "resources": [
                        "buildconfigs"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "",
                        "build.openshift.io"
                    ],
                    "resources": [
                        "buildconfigs/instantiate"
                    ]
                },
                {
                    "verbs": [
                        "delete"
                    ],
                    "apiGroups": [
                        "",
                        "build.openshift.io"
                    ],
                    "resources": [
                        "builds"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:deployer-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create",
                        "delete",
                        "get",
                        "list",
                        "patch",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "pods"
                    ]
                },
                {
                    "verbs": [
                        "delete"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "replicationcontrollers"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "replicationcontrollers"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "replicationcontrollers/scale"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:deploymentconfig-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create",
                        "delete",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "replicationcontrollers"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "replicationcontrollers/scale"
                    ]
                },
                {
                    "verbs": [
                        "update"
                    ],
                    "apiGroups": [
                        "",
                        "apps.openshift.io"
                    ],
                    "resources": [
                        "deploymentconfigs/status"
                    ]
                },
                {
                    "verbs": [
                        "update"
                    ],
                    "apiGroups": [
                        "",
                        "apps.openshift.io"
                    ],
                    "resources": [
                        "deploymentconfigs/finalizers"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "apps.openshift.io"
                    ],
                    "resources": [
                        "deploymentconfigs"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:template-instance-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "authorization.k8s.io"
                    ],
                    "resources": [
                        "subjectaccessreviews"
                    ]
                },
                {
                    "verbs": [
                        "update"
                    ],
                    "apiGroups": [
                        "template.openshift.io"
                    ],
                    "resources": [
                        "templateinstances/status"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:template-instance-finalizer-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "update"
                    ],
                    "apiGroups": [
                        "template.openshift.io"
                    ],
                    "resources": [
                        "templateinstances/status"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:origin-namespace-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "namespaces"
                    ]
                },
                {
                    "verbs": [
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "namespaces/finalize",
                        "namespaces/status"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:serviceaccount-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create",
                        "delete",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "serviceaccounts"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:serviceaccount-pull-secrets-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create",
                        "get",
                        "list",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "serviceaccounts"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "secrets"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "services"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:image-trigger-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "imagestreams"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "update"
                    ],
                    "apiGroups": [
                        "extensions"
                    ],
                    "resources": [
                        "daemonsets"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "update"
                    ],
                    "apiGroups": [
                        "apps",
                        "extensions"
                    ],
                    "resources": [
                        "deployments"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "update"
                    ],
                    "apiGroups": [
                        "apps"
                    ],
                    "resources": [
                        "statefulsets"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "update"
                    ],
                    "apiGroups": [
                        "batch"
                    ],
                    "resources": [
                        "cronjobs"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "update"
                    ],
                    "apiGroups": [
                        "",
                        "apps.openshift.io"
                    ],
                    "resources": [
                        "deploymentconfigs"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "",
                        "build.openshift.io"
                    ],
                    "resources": [
                        "buildconfigs/instantiate"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "",
                        "build.openshift.io"
                    ],
                    "resources": [
                        "builds/custom",
                        "builds/docker",
                        "builds/jenkinspipeline",
                        "builds/optimizeddocker",
                        "builds/source"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:service-serving-cert-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "list",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "services"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "get",
                        "list",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "secrets"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:image-import-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create",
                        "get",
                        "list",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "imagestreams"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "images"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "imagestreamimports"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:sdn-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create",
                        "get",
                        "update"
                    ],
                    "apiGroups": [
                        "",
                        "network.openshift.io"
                    ],
                    "resources": [
                        "clusternetworks"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "get",
                        "list",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "network.openshift.io"
                    ],
                    "resources": [
                        "hostsubnets"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "get",
                        "list",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "network.openshift.io"
                    ],
                    "resources": [
                        "netnamespaces"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "pods"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "services"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "namespaces"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "nodes"
                    ]
                },
                {
                    "verbs": [
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "nodes/status"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:cluster-quota-reconciliation-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get",
                        "list"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "configmaps"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "secrets"
                    ]
                },
                {
                    "verbs": [
                        "update"
                    ],
                    "apiGroups": [
                        "",
                        "quota.openshift.io"
                    ],
                    "resources": [
                        "clusterresourcequotas/status"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:unidling-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "endpoints",
                        "replicationcontrollers/scale"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "replicationcontrollers"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        "",
                        "apps.openshift.io"
                    ],
                    "resources": [
                        "deploymentconfigs"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "update"
                    ],
                    "apiGroups": [
                        "apps",
                        "extensions"
                    ],
                    "resources": [
                        "deployments/scale",
                        "replicasets/scale"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "update"
                    ],
                    "apiGroups": [
                        "",
                        "apps.openshift.io"
                    ],
                    "resources": [
                        "deploymentconfigs/scale"
                    ]
                },
                {
                    "verbs": [
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:service-ingress-ip-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "list",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "services"
                    ]
                },
                {
                    "verbs": [
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "services/status"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:ingress-to-route-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "secrets",
                        "services"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "extensions"
                    ],
                    "resources": [
                        "ingress"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "route.openshift.io"
                    ],
                    "resources": [
                        "routes"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "update"
                    ],
                    "apiGroups": [
                        "route.openshift.io"
                    ],
                    "resources": [
                        "routes/custom-host"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:pv-recycler-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create",
                        "delete",
                        "get",
                        "list",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "persistentvolumes"
                    ]
                },
                {
                    "verbs": [
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "persistentvolumes/status"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "persistentvolumeclaims"
                    ]
                },
                {
                    "verbs": [
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "persistentvolumeclaims/status"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "pods"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:resourcequota-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "resourcequotas/status"
                    ]
                },
                {
                    "verbs": [
                        "list"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "resourcequotas"
                    ]
                },
                {
                    "verbs": [
                        "list"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "services"
                    ]
                },
                {
                    "verbs": [
                        "list"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "configmaps"
                    ]
                },
                {
                    "verbs": [
                        "list"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "secrets"
                    ]
                },
                {
                    "verbs": [
                        "list"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "replicationcontrollers"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:horizontal-pod-autoscaler",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get",
                        "update"
                    ],
                    "apiGroups": [
                        "",
                        "apps.openshift.io"
                    ],
                    "resources": [
                        "deploymentconfigs/scale"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:template-service-broker",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "authorization.k8s.io"
                    ],
                    "resources": [
                        "subjectaccessreviews"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "authorization.openshift.io"
                    ],
                    "resources": [
                        "subjectaccessreviews"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "get",
                        "update"
                    ],
                    "apiGroups": [
                        "template.openshift.io"
                    ],
                    "resources": [
                        "brokertemplateinstances"
                    ]
                },
                {
                    "verbs": [
                        "update"
                    ],
                    "apiGroups": [
                        "template.openshift.io"
                    ],
                    "resources": [
                        "brokertemplateinstances/finalizers"
                    ]
                },
                {
                    "verbs": [
                        "assign",
                        "create",
                        "delete",
                        "get"
                    ],
                    "apiGroups": [
                        "template.openshift.io"
                    ],
                    "resources": [
                        "templateinstances"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "template.openshift.io"
                    ],
                    "resources": [
                        "templates"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "get"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "secrets"
                    ]
                },
                {
                    "verbs": [
                        "get"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "configmaps",
                        "services"
                    ]
                },
                {
                    "verbs": [
                        "get"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "routes"
                    ]
                },
                {
                    "verbs": [
                        "get"
                    ],
                    "apiGroups": [
                        "route.openshift.io"
                    ],
                    "resources": [
                        "routes"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:default-rolebindings-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "rbac.authorization.k8s.io"
                    ],
                    "resources": [
                        "rolebindings"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "namespaces"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "rbac.authorization.k8s.io"
                    ],
                    "resources": [
                        "rolebindings"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:namespace-security-allocation-controller",
                "creationTimestamp": null,
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create",
                        "get",
                        "update"
                    ],
                    "apiGroups": [
                        "security.openshift.io"
                    ],
                    "resources": [
                        "rangeallocations"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "namespaces"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "cluster-admin",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "*"
                    ],
                    "apiGroups": [
                        "*"
                    ],
                    "resources": [
                        "*"
                    ]
                },
                {
                    "verbs": [
                        "*"
                    ],
                    "nonResourceURLs": [
                        "*"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:discovery",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get"
                    ],
                    "nonResourceURLs": [
                        "/api",
                        "/api/*",
                        "/apis",
                        "/apis/*",
                        "/healthz",
                        "/openapi",
                        "/openapi/*",
                        "/swagger-2.0.0.pb-v1",
                        "/swagger.json",
                        "/swaggerapi",
                        "/swaggerapi/*",
                        "/version",
                        "/version/"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:basic-user",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "authorization.k8s.io"
                    ],
                    "resources": [
                        "selfsubjectaccessreviews",
                        "selfsubjectrulesreviews"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "admin",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "openshift.io/description": "A user that has edit rights within the project and can change the project's membership.",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": null,
            "aggregationRule": {
                "clusterRoleSelectors": [
                    {
                        "matchLabels": {
                            "rbac.authorization.k8s.io/aggregate-to-admin": "true"
                        }
                    }
                ]
            }
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "edit",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "openshift.io/description": "A user that can create and edit most objects in a project, but can not update the project's membership.",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": null,
            "aggregationRule": {
                "clusterRoleSelectors": [
                    {
                        "matchLabels": {
                            "rbac.authorization.k8s.io/aggregate-to-edit": "true"
                        }
                    }
                ]
            }
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "view",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "openshift.io/description": "A user who can view but not edit any resources within the project. They can not view secrets or membership.",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": null,
            "aggregationRule": {
                "clusterRoleSelectors": [
                    {
                        "matchLabels": {
                            "rbac.authorization.k8s.io/aggregate-to-view": "true"
                        }
                    }
                ]
            }
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:aggregate-to-admin",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults",
                    "rbac.authorization.k8s.io/aggregate-to-admin": "true"
                },
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "pods",
                        "pods/attach",
                        "pods/exec",
                        "pods/portforward",
                        "pods/proxy"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "configmaps",
                        "endpoints",
                        "persistentvolumeclaims",
                        "replicationcontrollers",
                        "replicationcontrollers/scale",
                        "secrets",
                        "serviceaccounts",
                        "services",
                        "services/proxy"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "bindings",
                        "events",
                        "limitranges",
                        "namespaces/status",
                        "pods/log",
                        "pods/status",
                        "replicationcontrollers/status",
                        "resourcequotas",
                        "resourcequotas/status"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "namespaces"
                    ]
                },
                {
                    "verbs": [
                        "impersonate"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "serviceaccounts"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "apps"
                    ],
                    "resources": [
                        "daemonsets",
                        "deployments",
                        "deployments/rollback",
                        "deployments/scale",
                        "replicasets",
                        "replicasets/scale",
                        "statefulsets",
                        "statefulsets/scale"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "autoscaling"
                    ],
                    "resources": [
                        "horizontalpodautoscalers"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "batch"
                    ],
                    "resources": [
                        "cronjobs",
                        "jobs"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "extensions"
                    ],
                    "resources": [
                        "daemonsets",
                        "deployments",
                        "deployments/rollback",
                        "deployments/scale",
                        "ingresses",
                        "networkpolicies",
                        "replicasets",
                        "replicasets/scale",
                        "replicationcontrollers/scale"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "policy"
                    ],
                    "resources": [
                        "poddisruptionbudgets"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "networking.k8s.io"
                    ],
                    "resources": [
                        "networkpolicies"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "authorization.k8s.io"
                    ],
                    "resources": [
                        "localsubjectaccessreviews"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "rbac.authorization.k8s.io"
                    ],
                    "resources": [
                        "rolebindings",
                        "roles"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:aggregate-to-edit",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults",
                    "rbac.authorization.k8s.io/aggregate-to-edit": "true"
                },
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "pods",
                        "pods/attach",
                        "pods/exec",
                        "pods/portforward",
                        "pods/proxy"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "configmaps",
                        "endpoints",
                        "persistentvolumeclaims",
                        "replicationcontrollers",
                        "replicationcontrollers/scale",
                        "secrets",
                        "serviceaccounts",
                        "services",
                        "services/proxy"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "bindings",
                        "events",
                        "limitranges",
                        "namespaces/status",
                        "pods/log",
                        "pods/status",
                        "replicationcontrollers/status",
                        "resourcequotas",
                        "resourcequotas/status"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "namespaces"
                    ]
                },
                {
                    "verbs": [
                        "impersonate"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "serviceaccounts"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "apps"
                    ],
                    "resources": [
                        "daemonsets",
                        "deployments",
                        "deployments/rollback",
                        "deployments/scale",
                        "replicasets",
                        "replicasets/scale",
                        "statefulsets",
                        "statefulsets/scale"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "autoscaling"
                    ],
                    "resources": [
                        "horizontalpodautoscalers"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "batch"
                    ],
                    "resources": [
                        "cronjobs",
                        "jobs"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "extensions"
                    ],
                    "resources": [
                        "daemonsets",
                        "deployments",
                        "deployments/rollback",
                        "deployments/scale",
                        "ingresses",
                        "networkpolicies",
                        "replicasets",
                        "replicasets/scale",
                        "replicationcontrollers/scale"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "policy"
                    ],
                    "resources": [
                        "poddisruptionbudgets"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete",
                        "deletecollection",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "networking.k8s.io"
                    ],
                    "resources": [
                        "networkpolicies"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:aggregate-to-view",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults",
                    "rbac.authorization.k8s.io/aggregate-to-view": "true"
                },
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "configmaps",
                        "endpoints",
                        "persistentvolumeclaims",
                        "pods",
                        "replicationcontrollers",
                        "replicationcontrollers/scale",
                        "serviceaccounts",
                        "services"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "bindings",
                        "events",
                        "limitranges",
                        "namespaces/status",
                        "pods/log",
                        "pods/status",
                        "replicationcontrollers/status",
                        "resourcequotas",
                        "resourcequotas/status"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "namespaces"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "apps"
                    ],
                    "resources": [
                        "daemonsets",
                        "deployments",
                        "deployments/scale",
                        "replicasets",
                        "replicasets/scale",
                        "statefulsets",
                        "statefulsets/scale"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "autoscaling"
                    ],
                    "resources": [
                        "horizontalpodautoscalers"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "batch"
                    ],
                    "resources": [
                        "cronjobs",
                        "jobs"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "extensions"
                    ],
                    "resources": [
                        "daemonsets",
                        "deployments",
                        "deployments/scale",
                        "ingresses",
                        "networkpolicies",
                        "replicasets",
                        "replicasets/scale",
                        "replicationcontrollers/scale"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "policy"
                    ],
                    "resources": [
                        "poddisruptionbudgets"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "networking.k8s.io"
                    ],
                    "resources": [
                        "networkpolicies"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:heapster",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events",
                        "namespaces",
                        "nodes",
                        "pods"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "extensions"
                    ],
                    "resources": [
                        "deployments"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:node",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "authentication.k8s.io"
                    ],
                    "resources": [
                        "tokenreviews"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "authorization.k8s.io"
                    ],
                    "resources": [
                        "localsubjectaccessreviews",
                        "subjectaccessreviews"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "services"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "nodes"
                    ]
                },
                {
                    "verbs": [
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "nodes/status"
                    ]
                },
                {
                    "verbs": [
                        "delete",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "nodes"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "pods"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "delete"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "pods"
                    ]
                },
                {
                    "verbs": [
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "pods/status"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "pods/eviction"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "configmaps",
                        "secrets"
                    ]
                },
                {
                    "verbs": [
                        "get"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "persistentvolumeclaims",
                        "persistentvolumes"
                    ]
                },
                {
                    "verbs": [
                        "get"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "endpoints"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "certificates.k8s.io"
                    ],
                    "resources": [
                        "certificatesigningrequests"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "persistentvolumeclaims/status"
                    ]
                },
                {
                    "verbs": [
                        "get"
                    ],
                    "apiGroups": [
                        "storage.k8s.io"
                    ],
                    "resources": [
                        "volumeattachments"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:node-problem-detector",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "nodes"
                    ]
                },
                {
                    "verbs": [
                        "patch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "nodes/status"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:node-proxier",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "endpoints",
                        "services"
                    ]
                },
                {
                    "verbs": [
                        "get"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "nodes"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:kubelet-api-admin",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "nodes"
                    ]
                },
                {
                    "verbs": [
                        "proxy"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "nodes"
                    ]
                },
                {
                    "verbs": [
                        "*"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "nodes/log",
                        "nodes/metrics",
                        "nodes/proxy",
                        "nodes/spec",
                        "nodes/stats"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:node-bootstrapper",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create",
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "certificates.k8s.io"
                    ],
                    "resources": [
                        "certificatesigningrequests"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:auth-delegator",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "authentication.k8s.io"
                    ],
                    "resources": [
                        "tokenreviews"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "authorization.k8s.io"
                    ],
                    "resources": [
                        "subjectaccessreviews"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:kube-aggregator",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "endpoints",
                        "services"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:kube-controller-manager",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "endpoints",
                        "secrets",
                        "serviceaccounts"
                    ]
                },
                {
                    "verbs": [
                        "delete"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "secrets"
                    ]
                },
                {
                    "verbs": [
                        "get"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "endpoints",
                        "namespaces",
                        "secrets",
                        "serviceaccounts"
                    ]
                },
                {
                    "verbs": [
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "endpoints",
                        "secrets",
                        "serviceaccounts"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "authentication.k8s.io"
                    ],
                    "resources": [
                        "tokenreviews"
                    ]
                },
                {
                    "verbs": [
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "*"
                    ],
                    "resources": [
                        "*"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:kube-scheduler",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "endpoints"
                    ]
                },
                {
                    "verbs": [
                        "delete",
                        "get",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "endpoints"
                    ],
                    "resourceNames": [
                        "kube-scheduler"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "nodes"
                    ]
                },
                {
                    "verbs": [
                        "delete",
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "pods"
                    ]
                },
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "bindings",
                        "pods/binding"
                    ]
                },
                {
                    "verbs": [
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "pods/status"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "replicationcontrollers",
                        "services"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "apps",
                        "extensions"
                    ],
                    "resources": [
                        "replicasets"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "apps"
                    ],
                    "resources": [
                        "statefulsets"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "policy"
                    ],
                    "resources": [
                        "poddisruptionbudgets"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "persistentvolumeclaims",
                        "persistentvolumes"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:kube-dns",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "endpoints",
                        "services"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:persistent-volume-provisioner",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create",
                        "delete",
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "persistentvolumes"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "persistentvolumeclaims"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "storage.k8s.io"
                    ],
                    "resources": [
                        "storageclasses"
                    ]
                },
                {
                    "verbs": [
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:csi-external-provisioner",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create",
                        "delete",
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "persistentvolumes"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "persistentvolumeclaims"
                    ]
                },
                {
                    "verbs": [
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "storage.k8s.io"
                    ],
                    "resources": [
                        "storageclasses"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:csi-external-attacher",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "persistentvolumes"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "nodes"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        "storage.k8s.io"
                    ],
                    "resources": [
                        "volumeattachments"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:aws-cloud-provider",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get",
                        "patch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "nodes"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:certificates.k8s.io:certificatesigningrequests:nodeclient",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "certificates.k8s.io"
                    ],
                    "resources": [
                        "certificatesigningrequests/nodeclient"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:certificates.k8s.io:certificatesigningrequests:selfnodeclient",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create"
                    ],
                    "apiGroups": [
                        "certificates.k8s.io"
                    ],
                    "resources": [
                        "certificatesigningrequests/selfnodeclient"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRole",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:volume-scheduler",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "authorization.openshift.io/system-only": "true",
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get",
                        "list",
                        "patch",
                        "update",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "persistentvolumes"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "storage.k8s.io"
                    ],
                    "resources": [
                        "storageclasses"
                    ]
                }
            ]
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:masters",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "Group",
                    "apiGroup": "rbac.authorization.k8s.io",
                    "name": "system:masters"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:master"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:node-admins",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "User",
                    "apiGroup": "rbac.authorization.k8s.io",
                    "name": "system:master"
                },
                {
                    "kind": "Group",
                    "apiGroup": "rbac.authorization.k8s.io",
                    "name": "system:node-admins"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:node-admin"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "cluster-admins",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "Group",
                    "apiGroup": "rbac.authorization.k8s.io",
                    "name": "system:cluster-admins"
                },
                {
                    "kind": "User",
                    "apiGroup": "rbac.authorization.k8s.io",
                    "name": "system:admin"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "cluster-admin"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "cluster-readers",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "Group",
                    "apiGroup": "rbac.authorization.k8s.io",
                    "name": "system:cluster-readers"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "cluster-reader"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "basic-users",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "Group",
                    "apiGroup": "rbac.authorization.k8s.io",
                    "name": "system:authenticated"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "basic-user"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "self-access-reviewers",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "Group",
                    "apiGroup": "rbac.authorization.k8s.io",
                    "name": "system:authenticated"
                },
                {
                    "kind": "Group",
                    "apiGroup": "rbac.authorization.k8s.io",
                    "name": "system:unauthenticated"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "self-access-reviewer"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "self-provisioners",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "Group",
                    "apiGroup": "rbac.authorization.k8s.io",
                    "name": "system:authenticated:oauth"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "self-provisioner"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:oauth-token-deleters",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "Group",
                    "apiGroup": "rbac.authorization.k8s.io",
                    "name": "system:authenticated"
                },
                {
                    "kind": "Group",
                    "apiGroup": "rbac.authorization.k8s.io",
                    "name": "system:unauthenticated"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:oauth-token-deleter"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "cluster-status-binding",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "Group",
                    "apiGroup": "rbac.authorization.k8s.io",
                    "name": "system:authenticated"
                },
                {
                    "kind": "Group",
                    "apiGroup": "rbac.authorization.k8s.io",
                    "name": "system:unauthenticated"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "cluster-status"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:node-proxiers",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "Group",
                    "apiGroup": "rbac.authorization.k8s.io",
                    "name": "system:nodes"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:node-proxier"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:sdn-readers",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "Group",
                    "apiGroup": "rbac.authorization.k8s.io",
                    "name": "system:nodes"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:sdn-reader"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:webhooks",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "Group",
                    "apiGroup": "rbac.authorization.k8s.io",
                    "name": "system:authenticated"
                },
                {
                    "kind": "Group",
                    "apiGroup": "rbac.authorization.k8s.io",
                    "name": "system:unauthenticated"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:webhook"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:discovery",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "Group",
                    "apiGroup": "rbac.authorization.k8s.io",
                    "name": "system:authenticated"
                },
                {
                    "kind": "Group",
                    "apiGroup": "rbac.authorization.k8s.io",
                    "name": "system:unauthenticated"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:openshift:discovery"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:build-strategy-docker-binding",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "Group",
                    "apiGroup": "rbac.authorization.k8s.io",
                    "name": "system:authenticated"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:build-strategy-docker"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:build-strategy-source-binding",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "Group",
                    "apiGroup": "rbac.authorization.k8s.io",
                    "name": "system:authenticated"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:build-strategy-source"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:build-strategy-jenkinspipeline-binding",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "Group",
                    "apiGroup": "rbac.authorization.k8s.io",
                    "name": "system:authenticated"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:build-strategy-jenkinspipeline"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:node-bootstrapper",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "node-bootstrapper",
                    "namespace": "openshift-infra"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:node-bootstrapper"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:scope-impersonation",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "Group",
                    "apiGroup": "rbac.authorization.k8s.io",
                    "name": "system:authenticated"
                },
                {
                    "kind": "Group",
                    "apiGroup": "rbac.authorization.k8s.io",
                    "name": "system:unauthenticated"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:scope-impersonation"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:nodes",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:node"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:discovery-binding",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:discovery"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:build-controller",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "build-controller",
                    "namespace": "openshift-infra"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:openshift:controller:build-controller"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:build-config-change-controller",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "build-config-change-controller",
                    "namespace": "openshift-infra"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:openshift:controller:build-config-change-controller"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:deployer-controller",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "deployer-controller",
                    "namespace": "openshift-infra"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:openshift:controller:deployer-controller"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:deploymentconfig-controller",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "deploymentconfig-controller",
                    "namespace": "openshift-infra"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:openshift:controller:deploymentconfig-controller"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:template-instance-controller",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "template-instance-controller",
                    "namespace": "openshift-infra"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:openshift:controller:template-instance-controller"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:template-instance-controller:admin",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "template-instance-controller",
                    "namespace": "openshift-infra"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "admin"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:template-instance-finalizer-controller",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "template-instance-finalizer-controller",
                    "namespace": "openshift-infra"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:openshift:controller:template-instance-finalizer-controller"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:template-instance-finalizer-controller:admin",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "template-instance-finalizer-controller",
                    "namespace": "openshift-infra"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "admin"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:origin-namespace-controller",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "origin-namespace-controller",
                    "namespace": "openshift-infra"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:openshift:controller:origin-namespace-controller"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:serviceaccount-controller",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "serviceaccount-controller",
                    "namespace": "openshift-infra"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:openshift:controller:serviceaccount-controller"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:serviceaccount-pull-secrets-controller",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "serviceaccount-pull-secrets-controller",
                    "namespace": "openshift-infra"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:openshift:controller:serviceaccount-pull-secrets-controller"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:image-trigger-controller",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "image-trigger-controller",
                    "namespace": "openshift-infra"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:openshift:controller:image-trigger-controller"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:service-serving-cert-controller",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "service-serving-cert-controller",
                    "namespace": "openshift-infra"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:openshift:controller:service-serving-cert-controller"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:image-import-controller",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "image-import-controller",
                    "namespace": "openshift-infra"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:openshift:controller:image-import-controller"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:sdn-controller",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "sdn-controller",
                    "namespace": "openshift-infra"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:openshift:controller:sdn-controller"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:cluster-quota-reconciliation-controller",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "cluster-quota-reconciliation-controller",
                    "namespace": "openshift-infra"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:openshift:controller:cluster-quota-reconciliation-controller"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:unidling-controller",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "unidling-controller",
                    "namespace": "openshift-infra"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:openshift:controller:unidling-controller"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:service-ingress-ip-controller",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "service-ingress-ip-controller",
                    "namespace": "openshift-infra"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:openshift:controller:service-ingress-ip-controller"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:ingress-to-route-controller",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "ingress-to-route-controller",
                    "namespace": "openshift-infra"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:openshift:controller:ingress-to-route-controller"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:pv-recycler-controller",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "pv-recycler-controller",
                    "namespace": "openshift-infra"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:openshift:controller:pv-recycler-controller"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:resourcequota-controller",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "resourcequota-controller",
                    "namespace": "openshift-infra"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:openshift:controller:resourcequota-controller"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:horizontal-pod-autoscaler",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "horizontal-pod-autoscaler",
                    "namespace": "openshift-infra"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:openshift:controller:horizontal-pod-autoscaler"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:controller:horizontal-pod-autoscaler",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "horizontal-pod-autoscaler",
                    "namespace": "openshift-infra"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:controller:horizontal-pod-autoscaler"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:template-service-broker",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "template-service-broker",
                    "namespace": "openshift-infra"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:openshift:controller:template-service-broker"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:image-puller",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "default-rolebindings-controller",
                    "namespace": "openshift-infra"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:image-puller"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:image-builder",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "default-rolebindings-controller",
                    "namespace": "openshift-infra"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:image-builder"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:deployer",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "default-rolebindings-controller",
                    "namespace": "openshift-infra"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:deployer"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:default-rolebindings-controller",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "default-rolebindings-controller",
                    "namespace": "openshift-infra"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:openshift:controller:default-rolebindings-controller"
            }
        },
        {
            "kind": "ClusterRoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:openshift:controller:namespace-security-allocation-controller",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "namespace-security-allocation-controller",
                    "namespace": "openshift-infra"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "ClusterRole",
                "name": "system:openshift:controller:namespace-security-allocation-controller"
            }
        },
        {
            "kind": "Role",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:controller:bootstrap-signer",
                "namespace": "kube-public",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "configmaps"
                    ]
                },
                {
                    "verbs": [
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "configmaps"
                    ],
                    "resourceNames": [
                        "cluster-info"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                }
            ]
        },
        {
            "kind": "Role",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "extension-apiserver-authentication-reader",
                "namespace": "kube-system",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "configmaps"
                    ],
                    "resourceNames": [
                        "extension-apiserver-authentication"
                    ]
                }
            ]
        },
        {
            "kind": "Role",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:controller:bootstrap-signer",
                "namespace": "kube-system",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "secrets"
                    ]
                }
            ]
        },
        {
            "kind": "Role",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:controller:cloud-provider",
                "namespace": "kube-system",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "create",
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "configmaps"
                    ]
                }
            ]
        },
        {
            "kind": "Role",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:controller:token-cleaner",
                "namespace": "kube-system",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "delete",
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "secrets"
                    ]
                },
                {
                    "verbs": [
                        "create",
                        "patch",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "events"
                    ]
                }
            ]
        },
        {
            "kind": "Role",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system::leader-locking-kube-controller-manager",
                "namespace": "kube-system",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "configmaps"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "configmaps"
                    ],
                    "resourceNames": [
                        "kube-controller-manager"
                    ]
                }
            ]
        },
        {
            "kind": "Role",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system::leader-locking-kube-scheduler",
                "namespace": "kube-system",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "watch"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "configmaps"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "update"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "configmaps"
                    ],
                    "resourceNames": [
                        "kube-scheduler"
                    ]
                }
            ]
        },
        {
            "kind": "Role",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "shared-resource-viewer",
                "namespace": "openshift",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "template.openshift.io"
                    ],
                    "resources": [
                        "templates"
                    ]
                },
                {
                    "verbs": [
                        "get",
                        "list",
                        "watch"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "imagestreamimages",
                        "imagestreams",
                        "imagestreamtags"
                    ]
                },
                {
                    "verbs": [
                        "get"
                    ],
                    "apiGroups": [
                        "",
                        "image.openshift.io"
                    ],
                    "resources": [
                        "imagestreams/layers"
                    ]
                }
            ]
        },
        {
            "kind": "Role",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:node-config-reader",
                "namespace": "openshift-node",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "rules": [
                {
                    "verbs": [
                        "get"
                    ],
                    "apiGroups": [
                        ""
                    ],
                    "resources": [
                        "configmaps"
                    ]
                }
            ]
        },
        {
            "kind": "RoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:controller:bootstrap-signer",
                "namespace": "kube-public",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "bootstrap-signer",
                    "namespace": "kube-system"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "Role",
                "name": "system:controller:bootstrap-signer"
            }
        },
        {
            "kind": "RoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system::leader-locking-kube-controller-manager",
                "namespace": "kube-system",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "kube-controller-manager",
                    "namespace": "kube-system"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "Role",
                "name": "system::leader-locking-kube-controller-manager"
            }
        },
        {
            "kind": "RoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system::leader-locking-kube-scheduler",
                "namespace": "kube-system",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "kube-scheduler",
                    "namespace": "kube-system"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "Role",
                "name": "system::leader-locking-kube-scheduler"
            }
        },
        {
            "kind": "RoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:controller:bootstrap-signer",
                "namespace": "kube-system",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "bootstrap-signer",
                    "namespace": "kube-system"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "Role",
                "name": "system:controller:bootstrap-signer"
            }
        },
        {
            "kind": "RoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:controller:cloud-provider",
                "namespace": "kube-system",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "cloud-provider",
                    "namespace": "kube-system"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "Role",
                "name": "system:controller:cloud-provider"
            }
        },
        {
            "kind": "RoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:controller:token-cleaner",
                "namespace": "kube-system",
                "creationTimestamp": null,
                "labels": {
                    "kubernetes.io/bootstrapping": "rbac-defaults"
                },
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "ServiceAccount",
                    "name": "token-cleaner",
                    "namespace": "kube-system"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "Role",
                "name": "system:controller:token-cleaner"
            }
        },
        {
            "kind": "RoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "shared-resource-viewers",
                "namespace": "openshift",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "Group",
                    "apiGroup": "rbac.authorization.k8s.io",
                    "name": "system:authenticated"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "Role",
                "name": "shared-resource-viewer"
            }
        },
        {
            "kind": "RoleBinding",
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "metadata": {
                "name": "system:node-config-reader",
                "namespace": "openshift-node",
                "creationTimestamp": null,
                "annotations": {
                    "rbac.authorization.kubernetes.io/autoupdate": "true"
                }
            },
            "subjects": [
                {
                    "kind": "Group",
                    "apiGroup": "rbac.authorization.k8s.io",
                    "name": "system:nodes"
                }
            ],
            "roleRef": {
                "apiGroup": "rbac.authorization.k8s.io",
                "kind": "Role",
                "name": "system:node-config-reader"
            }
        }
    ]
}
`

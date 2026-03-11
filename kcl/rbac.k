"""
RBAC for homerun2-scout — grants ServiceAccount permission to read ScoutProfile CRs
"""

import labels

config = labels.config

role = {
    apiVersion: "rbac.authorization.k8s.io/v1"
    kind: "Role"
    metadata: {
        name: "{}-scoutprofile-reader".format(config.name)
        namespace: config.namespace
        labels: labels.commonLabels
    }
    rules: [
        {
            apiGroups: ["homerun2.stuttgart-things.com"]
            resources: ["scoutprofiles"]
            verbs: ["get"]
        }
    ]
}

roleBinding = {
    apiVersion: "rbac.authorization.k8s.io/v1"
    kind: "RoleBinding"
    metadata: {
        name: "{}-scoutprofile-reader".format(config.name)
        namespace: config.namespace
        labels: labels.commonLabels
    }
    roleRef: {
        apiGroup: "rbac.authorization.k8s.io"
        kind: "Role"
        name: "{}-scoutprofile-reader".format(config.name)
    }
    subjects: [
        {
            kind: "ServiceAccount"
            name: config.name
            namespace: config.namespace
        }
    ]
}

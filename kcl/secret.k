"""
Secrets for homerun2-scout
"""

import base64
import labels

config = labels.config

# Secret - Auth Token
secretToken = {
    apiVersion: "v1"
    kind: "Secret"
    metadata: {
        name: "{}-token".format(config.name)
        namespace: config.namespace
        labels: labels.commonLabels
    }
    type: "Opaque"
    data: {
        "auth-token": base64.encode(config.authToken)
    }
} if config.authToken else None

# Secret - Redis
secretRedis = {
    apiVersion: "v1"
    kind: "Secret"
    metadata: {
        name: "{}-redis".format(config.name)
        namespace: config.namespace
        labels: labels.commonLabels
    }
    type: "Opaque"
    data: {
        password: base64.encode(config.redisPassword)
    }
} if config.redisPassword else None

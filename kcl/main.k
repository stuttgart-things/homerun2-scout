"""
Main entry point for homerun2-scout Kubernetes deployment
Imports all resources and exports manifests
"""

import namespace
import serviceaccount
import configmap
import secret
import deploy
import service
import ingress
import httproute

# Export all manifests (filter out None values)
manifests = [
    m for m in [
        namespace.namespace
        serviceaccount.serviceAccount
        configmap.configMap
        secret.secretToken
        secret.secretRedis
        deploy.deployment
        service.service
        ingress.ingress
        httproute.httproute
    ] if m
]

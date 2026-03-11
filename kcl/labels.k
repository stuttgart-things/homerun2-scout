"""
Common labels for homerun2-scout resources
"""

import schema

# Get command-line options
_image = option("config.image")
_namespace = option("config.namespace")
_ingressEnabled = option("config.ingressEnabled")
_ingressHost = option("config.ingressHost")
_ingressTlsEnabled = option("config.ingressTlsEnabled")
_ingressTlsSecretName = option("config.ingressTlsSecretName")
_ingressAnnotations = option("config.ingressAnnotations") or {}
_httpRouteEnabled = option("config.httpRouteEnabled")
_httpRouteParentRefName = option("config.httpRouteParentRefName")
_httpRouteParentRefNamespace = option("config.httpRouteParentRefNamespace")
_httpRouteHostname = option("config.httpRouteHostname")
_httpRouteAnnotations = option("config.httpRouteAnnotations") or {}
_redisAddr = option("config.redisAddr")
_redisPort = option("config.redisPort")
_redisearchIndex = option("config.redisearchIndex")
_scoutInterval = option("config.scoutInterval")
_authToken = option("config.authToken")
_redisPassword = option("config.redisPassword")
_extraEnvVars = option("config.extraEnvVars") or {}

# Config instance with command-line overrides
config: schema.Scout = schema.Scout {
    if _image:
        image = _image
    if _namespace:
        namespace = _namespace
    if _ingressEnabled:
        ingressEnabled = _ingressEnabled in [True, "True", "true"]
    if _ingressHost:
        ingressHost = _ingressHost
    if _ingressTlsEnabled:
        ingressTlsEnabled = _ingressTlsEnabled in [True, "True", "true"]
    if _ingressTlsSecretName:
        ingressTlsSecretName = _ingressTlsSecretName
    if _ingressAnnotations:
        ingressAnnotations = _ingressAnnotations
    if _httpRouteEnabled:
        httpRouteEnabled = _httpRouteEnabled in [True, "True", "true"]
    if _httpRouteParentRefName:
        httpRouteParentRefName = _httpRouteParentRefName
    if _httpRouteParentRefNamespace:
        httpRouteParentRefNamespace = _httpRouteParentRefNamespace
    if _httpRouteHostname:
        httpRouteHostname = _httpRouteHostname
    if _httpRouteAnnotations:
        httpRouteAnnotations = _httpRouteAnnotations
    if _redisAddr:
        redisAddr = _redisAddr
    if _redisPort:
        redisPort = _redisPort
    if _redisearchIndex:
        redisearchIndex = _redisearchIndex
    if _scoutInterval:
        scoutInterval = _scoutInterval
    if _authToken:
        authToken = _authToken
    if _redisPassword:
        redisPassword = _redisPassword
    if _extraEnvVars:
        extraEnvVars = _extraEnvVars
}

# Common labels applied to all resources
commonLabels = {
    "app.kubernetes.io/name": config.name
    "app.kubernetes.io/instance": config.name
    "app.kubernetes.io/component": "analytics"
    "app.kubernetes.io/part-of": "homerun2"
    **config.labels
}

# Selector labels for pods
selectorLabels = {
    "app.kubernetes.io/name": config.name
    "app.kubernetes.io/instance": config.name
}

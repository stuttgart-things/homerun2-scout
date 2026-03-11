"""
Ingress for homerun2-scout
"""

import labels

config = labels.config

ingress = {
    apiVersion: "networking.k8s.io/v1"
    kind: "Ingress"
    metadata: {
        name: config.name
        namespace: config.namespace
        labels: labels.commonLabels
        annotations: {
            "nginx.ingress.kubernetes.io/ssl-redirect": "true" if config.ingressTlsEnabled else "false"
            "nginx.ingress.kubernetes.io/proxy-body-size": "10m"
            "nginx.ingress.kubernetes.io/limit-rps": "100"
            **config.annotations
            **config.ingressAnnotations
        }
    }
    spec: {
        ingressClassName: config.ingressClassName
        if config.ingressTlsEnabled:
            tls: [
                {
                    hosts: [config.ingressHost]
                    secretName: config.ingressTlsSecretName
                }
            ]
        rules: [
            {
                host: config.ingressHost
                http: {
                    paths: [
                        {
                            path: "/"
                            pathType: "Prefix"
                            backend: {
                                service: {
                                    name: config.name
                                    port: {
                                        number: config.servicePort
                                    }
                                }
                            }
                        }
                    ]
                }
            }
        ]
    }
} if config.ingressEnabled else None

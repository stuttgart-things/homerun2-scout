"""
Deployment for homerun2-scout
"""

import labels

config = labels.config

deployment = {
    apiVersion: "apps/v1"
    kind: "Deployment"
    metadata: {
        name: config.name
        namespace: config.namespace
        labels: labels.commonLabels
    }
    spec: {
        replicas: config.replicas
        selector: {
            matchLabels: labels.selectorLabels
        }
        strategy: {
            type: "RollingUpdate"
            rollingUpdate: {
                maxSurge: 1
                maxUnavailable: 0
            }
        }
        template: {
            metadata: {
                labels: labels.commonLabels
            }
            spec: {
                serviceAccountName: config.name
                containers: [
                    {
                        name: config.name
                        image: config.image
                        imagePullPolicy: config.imagePullPolicy
                        ports: [
                            {
                                name: "http"
                                containerPort: config.containerPort
                                protocol: "TCP"
                            }
                        ]
                        env: [
                            {
                                name: "PORT"
                                value: str(config.containerPort)
                            }
                            {
                                name: "AUTH_TOKEN"
                                valueFrom: {
                                    secretKeyRef: {
                                        name: "{}-token".format(config.name)
                                        key: "auth-token"
                                    }
                                }
                            }
                            {
                                name: "REDIS_ADDR"
                                value: config.redisAddr
                            }
                            {
                                name: "REDIS_PORT"
                                value: str(config.redisPort)
                            }
                            {
                                name: "REDIS_PASSWORD"
                                valueFrom: {
                                    secretKeyRef: {
                                        name: "{}-redis".format(config.name)
                                        key: "password"
                                    }
                                }
                            }
                            {
                                name: "REDISEARCH_INDEX"
                                value: config.redisearchIndex
                            }
                            {
                                name: "SCOUT_PROFILE_NAME"
                                value: config.scoutProfileName
                            }
                            {
                                name: "POD_NAMESPACE"
                                valueFrom: {
                                    fieldRef: {
                                        fieldPath: "metadata.namespace"
                                    }
                                }
                            }
                            {
                                name: "SCOUT_INTERVAL"
                                value: config.scoutInterval
                            }
                        ]
                        envFrom: [
                            {
                                configMapRef: {
                                    name: "{}-config".format(config.name)
                                }
                            }
                        ]
                        resources: {
                            requests: {
                                cpu: config.cpuRequest
                                memory: config.memoryRequest
                            }
                            limits: {
                                cpu: config.cpuLimit
                                memory: config.memoryLimit
                            }
                        }
                        livenessProbe: {
                            httpGet: {
                                path: "/health"
                                port: "http"
                            }
                            initialDelaySeconds: 10
                            periodSeconds: 10
                            timeoutSeconds: 5
                            failureThreshold: 3
                        }
                        readinessProbe: {
                            httpGet: {
                                path: "/health"
                                port: "http"
                            }
                            initialDelaySeconds: 5
                            periodSeconds: 5
                            timeoutSeconds: 3
                            failureThreshold: 3
                        }
                        securityContext: {
                            readOnlyRootFilesystem: True
                            runAsNonRoot: True
                            allowPrivilegeEscalation: False
                            capabilities: {
                                drop: ["ALL"]
                            }
                        }
                        volumeMounts: [
                            {
                                name: "tmp"
                                mountPath: "/tmp"
                            }
                        ]
                    }
                ]
                securityContext: {
                    runAsNonRoot: True
                    runAsUser: 65532
                    runAsGroup: 65532
                    fsGroup: 65532
                    seccompProfile: {
                        type: "RuntimeDefault"
                    }
                }
                volumes: [
                    {
                        name: "tmp"
                        emptyDir: {}
                    }
                ]
                affinity: {
                    podAntiAffinity: {
                        preferredDuringSchedulingIgnoredDuringExecution: [
                            {
                                weight: 100
                                podAffinityTerm: {
                                    labelSelector: {
                                        matchLabels: labels.selectorLabels
                                    }
                                    topologyKey: "kubernetes.io/hostname"
                                }
                            }
                        ]
                    }
                }
            }
        }
    }
}

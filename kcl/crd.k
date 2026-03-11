"""
CustomResourceDefinition for ScoutProfile
"""

scoutProfileCRD = {
    apiVersion: "apiextensions.k8s.io/v1"
    kind: "CustomResourceDefinition"
    metadata: {
        name: "scoutprofiles.homerun2.stuttgart-things.com"
    }
    spec: {
        group: "homerun2.stuttgart-things.com"
        names: {
            kind: "ScoutProfile"
            plural: "scoutprofiles"
            singular: "scoutprofile"
            shortNames: ["sp"]
        }
        scope: "Namespaced"
        versions: [
            {
                name: "v1alpha1"
                served: True
                storage: True
                schema: {
                    openAPIV3Schema: {
                        type: "object"
                        properties: {
                            spec: {
                                type: "object"
                                properties: {
                                    scoutInterval: {
                                        type: "string"
                                        description: "Aggregation interval (Go duration, e.g. 60s)"
                                    }
                                    retention: {
                                        type: "object"
                                        properties: {
                                            enabled: {
                                                type: "boolean"
                                                description: "Enable periodic RediSearch index cleanup"
                                            }
                                            ttl: {
                                                type: "string"
                                                description: "Retention TTL (Go duration, e.g. 168h)"
                                            }
                                        }
                                    }
                                    alerting: {
                                        type: "object"
                                        properties: {
                                            pitcherURL: {
                                                type: "string"
                                                description: "omni-pitcher /pitch endpoint URL"
                                            }
                                            pitcherToken: {
                                                type: "string"
                                                description: "Bearer token for omni-pitcher (prefer env var ALERT_PITCHER_TOKEN)"
                                            }
                                            errorThreshold: {
                                                type: "integer"
                                                description: "Error count threshold to trigger an alert"
                                            }
                                            criticalThreshold: {
                                                type: "integer"
                                                description: "Critical count threshold to trigger an alert"
                                            }
                                            cooldown: {
                                                type: "string"
                                                description: "Minimum time between alerts (Go duration, e.g. 5m)"
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        ]
    }
}

package profile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// scoutProfileCR is the wrapper used to deserialise the full CR from the Kubernetes API.
type scoutProfileCR struct {
	Spec ScoutProfile `json:"spec"`
}

var scoutProfileGVR = schema.GroupVersionResource{
	Group:    "homerun2.stuttgart-things.com",
	Version:  "v1alpha1",
	Resource: "scoutprofiles",
}

// ProfileLoader loads a ScoutProfile by namespace and name.
type ProfileLoader interface {
	Load(ctx context.Context, namespace, name string) (*ScoutProfile, error)
}

// KubernetesLoader reads a ScoutProfile CR from the Kubernetes API.
type KubernetesLoader struct {
	client dynamic.Interface
}

// NewKubernetesLoader creates a loader using in-cluster config,
// falling back to KUBECONFIG for local development.
func NewKubernetesLoader() (*KubernetesLoader, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		kubeconfig := os.Getenv("KUBECONFIG")
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("build k8s config: %w", err)
		}
	}
	client, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("build dynamic client: %w", err)
	}
	return &KubernetesLoader{client: client}, nil
}

// Load fetches the named ScoutProfile CR from the given namespace.
// Returns nil, nil if the resource is not found.
func (l *KubernetesLoader) Load(ctx context.Context, namespace, name string) (*ScoutProfile, error) {
	obj, err := l.client.Resource(scoutProfileGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("get ScoutProfile %s/%s: %w", namespace, name, err)
	}
	raw, err := obj.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshal ScoutProfile: %w", err)
	}
	var cr scoutProfileCR
	if err := json.Unmarshal(raw, &cr); err != nil {
		return nil, fmt.Errorf("unmarshal ScoutProfile: %w", err)
	}
	return &cr.Spec, nil
}

// NopLoader always returns nil — used in tests and when profile loading is disabled.
type NopLoader struct{}

func (NopLoader) Load(_ context.Context, _, _ string) (*ScoutProfile, error) { return nil, nil }

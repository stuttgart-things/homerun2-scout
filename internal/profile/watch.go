package profile

import (
	"context"
	"log/slog"
	"reflect"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// Watch reads the ScoutProfile namespace/name every interval and calls onChange
// once, with the reason, when it differs from loaded - the profile scout
// started with, nil when it was missing. Then it returns.
//
// scout applies its profile only at startup, so a profile written after scout
// started would otherwise wait for the next restart. That is not hypothetical:
// argocd's install chart rolls scout for a changed profile and applies the
// ScoutProfile in a later sync-wave, which Argo does not wait for, so scout
// came up with the old profile (stuttgart-things/argocd#394). onChange is where
// the caller restarts scout.
//
// A read that fails for any other reason than "not found" changes nothing: an
// API server that is briefly unreachable must not restart scout.
func Watch(ctx context.Context, loader ProfileLoader, namespace, name string, loaded *ScoutProfile, interval time.Duration, onChange func(reason string)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		if reason := changed(ctx, loader, namespace, name, loaded); reason != "" {
			onChange(reason)
			return
		}
	}
}

// changed returns why the current profile differs from loaded, or "" when it
// does not or cannot be read.
func changed(ctx context.Context, loader ProfileLoader, namespace, name string, loaded *ScoutProfile) string {
	current, err := loader.Load(ctx, namespace, name)
	switch {
	case apierrors.IsNotFound(err):
		current = nil
	case err != nil:
		slog.Debug("ScoutProfile watch: read failed, keeping the running profile", "name", name, "error", err)
		return ""
	}

	switch {
	case loaded == nil && current == nil:
		return ""
	case loaded == nil:
		return "ScoutProfile created"
	case current == nil:
		return "ScoutProfile deleted"
	case !reflect.DeepEqual(*loaded, *current):
		return "ScoutProfile changed"
	default:
		return ""
	}
}

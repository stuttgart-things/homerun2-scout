package profile

import (
	"context"
	"testing"
)

func TestNopLoader(t *testing.T) {
	p, err := NopLoader{}.Load(context.Background(), "homerun2", "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p != nil {
		t.Error("NopLoader should return nil profile")
	}
}

// Compile-time check that NopLoader satisfies the interface.
var _ ProfileLoader = NopLoader{}

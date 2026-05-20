package helm

import (
	"context"
	"testing"

	"github.com/MarvinJWendt/testza"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// fakeRESTGetter is a no-op RESTClientGetter sufficient for cache exercises.
// action.Configuration.Init lazily creates clients; it does not actually call
// into the cluster on construction.
type fakeRESTGetter struct {
	*genericclioptions.ConfigFlags
}

func newFakeRESTGetter() *fakeRESTGetter {
	ns := "default"
	return &fakeRESTGetter{ConfigFlags: &genericclioptions.ConfigFlags{Namespace: &ns}}
}

func TestClientCache_HitReturnsSamePointer(t *testing.T) {
	c := NewClientCache()
	getter := newFakeRESTGetter()
	a, err := c.Get(context.Background(), "ctx1", "default", getter, nil)
	testza.AssertNoError(t, err)
	b, err := c.Get(context.Background(), "ctx1", "default", getter, nil)
	testza.AssertNoError(t, err)
	if a != b {
		t.Fatalf("expected same pointer on cache hit, got %p vs %p", a, b)
	}
}

func TestClientCache_DifferentKeysAreDistinct(t *testing.T) {
	c := NewClientCache()
	getter := newFakeRESTGetter()
	a, _ := c.Get(context.Background(), "ctx1", "ns-a", getter, nil)
	b, _ := c.Get(context.Background(), "ctx1", "ns-b", getter, nil)
	if a == b {
		t.Fatal("expected distinct pointers across namespaces")
	}
}

func TestClientCache_Evict(t *testing.T) {
	c := NewClientCache()
	getter := newFakeRESTGetter()
	a, _ := c.Get(context.Background(), "ctx1", "default", getter, nil)
	_, _ = c.Get(context.Background(), "ctx2", "default", getter, nil)
	c.Evict("ctx1")
	a2, _ := c.Get(context.Background(), "ctx1", "default", getter, nil)
	if a == a2 {
		t.Fatal("expected fresh pointer after Evict")
	}
	// ctx2 unaffected.
	c.mu.Lock()
	_, has := c.items[cacheKey{contextName: "ctx2", namespace: "default"}]
	c.mu.Unlock()
	testza.AssertTrue(t, has)
}

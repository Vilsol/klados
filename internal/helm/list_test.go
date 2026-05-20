package helm

import (
	"context"
	"errors"
	"testing"

	"github.com/MarvinJWendt/testza"
	"helm.sh/helm/v4/pkg/release/common"
	corev1 "k8s.io/api/core/v1"
)

func TestBackend_List(t *testing.T) {
	lister := newFakeSecretLister()
	lister.rv = "100"
	lister.put("ctx1", "default", []corev1.Secret{
		makeReleaseSecret(t, "a", "default", 1, common.StatusDeployed, nil, ""),
		makeReleaseSecret(t, "a", "default", 2, common.StatusDeployed, nil, ""),
		makeReleaseSecret(t, "b", "default", 1, common.StatusDeployed, nil, ""),
	})
	b := NewBackend(NewClientCache(), lister)
	out, rv, err := b.List(context.Background(), "ctx1", "default")
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "100", rv)
	testza.AssertEqual(t, 2, len(out))
	// "a" latest is rev 2.
	testza.AssertEqual(t, "a", out[0]["metadata"].(map[string]any)["name"])
	testza.AssertEqual(t, int64(2), out[0]["spec"].(map[string]any)["revision"])
}

func TestBackend_List_EmptyNamespace(t *testing.T) {
	lister := newFakeSecretLister()
	lister.put("ctx1", "", []corev1.Secret{
		makeReleaseSecret(t, "x", "ns1", 1, common.StatusDeployed, nil, ""),
		makeReleaseSecret(t, "y", "ns2", 1, common.StatusFailed, nil, ""),
	})
	b := NewBackend(NewClientCache(), lister)
	out, _, err := b.List(context.Background(), "ctx1", "")
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 2, len(out))
}

func TestBackend_List_ListerError(t *testing.T) {
	lister := newFakeSecretLister()
	lister.err = errors.New("boom")
	b := NewBackend(NewClientCache(), lister)
	_, _, err := b.List(context.Background(), "ctx1", "default")
	testza.AssertNotNil(t, err)
}

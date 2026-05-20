package helm

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/MarvinJWendt/testza"
	"helm.sh/helm/v4/pkg/release/common"
	corev1 "k8s.io/api/core/v1"
)

func TestDiffRevisions_Equal(t *testing.T) {
	lister := newFakeSecretLister()
	lister.put("ctx1", "default", []corev1.Secret{
		makeReleaseSecret(t, "a", "default", 1, common.StatusSuperseded, map[string]any{"replicas": 2}, "kind: Deployment\nmetadata:\n  name: a\n"),
		makeReleaseSecret(t, "a", "default", 2, common.StatusDeployed, map[string]any{"replicas": 2}, "kind: Deployment\nmetadata:\n  name: a\n"),
	})
	b := NewBackend(NewClientCache(), lister)
	d, err := b.DiffRevisions(context.Background(), "ctx1", "default", "a", 1, 2)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "", d.Values)
	testza.AssertEqual(t, "", d.Manifest)
}

func TestDiffRevisions_DifferentValues(t *testing.T) {
	lister := newFakeSecretLister()
	lister.put("ctx1", "default", []corev1.Secret{
		makeReleaseSecret(t, "a", "default", 1, common.StatusSuperseded, map[string]any{"replicas": 2}, "kind: Deployment\nspec:\n  replicas: 2\n"),
		makeReleaseSecret(t, "a", "default", 2, common.StatusDeployed, map[string]any{"replicas": 5}, "kind: Deployment\nspec:\n  replicas: 5\n"),
	})
	b := NewBackend(NewClientCache(), lister)
	d, err := b.DiffRevisions(context.Background(), "ctx1", "default", "a", 1, 2)
	testza.AssertNoError(t, err)
	testza.AssertNotEqual(t, "", d.Values)
	testza.AssertContains(t, d.Values, "- replicas: 2")
	testza.AssertContains(t, d.Values, "+ replicas: 5")
	testza.AssertContains(t, d.Manifest, "- ")
	testza.AssertContains(t, d.Manifest, "+ ")
}

func TestDiffRevisions_MasksSecrets(t *testing.T) {
	// Different secret keys exist across revisions, so the diff will contain
	// at least one masked-key line, while no raw value should leak.
	lister := newFakeSecretLister()
	lister.put("ctx1", "default", []corev1.Secret{
		makeReleaseSecret(t, "a", "default", 1, common.StatusSuperseded, map[string]any{
			"password": "swordfish-1",
			"replicas": 1,
		}, ""),
		makeReleaseSecret(t, "a", "default", 2, common.StatusDeployed, map[string]any{
			"apiToken": "swordfish-2",
			"replicas": 1,
		}, ""),
	})
	b := NewBackend(NewClientCache(), lister)
	d, err := b.DiffRevisions(context.Background(), "ctx1", "default", "a", 1, 2)
	testza.AssertNoError(t, err)
	if strings.Contains(d.Values, "swordfish") {
		t.Fatalf("secret leaked into diff: %s", d.Values)
	}
	testza.AssertContains(t, d.Values, MaskValue)
}

func TestDiff_MasksSecretValues(t *testing.T) {
	// Same secret key with two different values across revisions. A second
	// non-secret key differs too so the unified diff is non-empty and includes
	// the masked password lines on both +/- sides.
	lister := newFakeSecretLister()
	lister.put("ctx1", "default", []corev1.Secret{
		makeReleaseSecret(t, "a", "default", 1, common.StatusSuperseded, map[string]any{
			"password": "hunter2",
			"replicas": 2,
		}, ""),
		makeReleaseSecret(t, "a", "default", 2, common.StatusDeployed, map[string]any{
			"password": "hunter3",
			"replicas": 5,
		}, ""),
	})
	b := NewBackend(NewClientCache(), lister)
	d, err := b.DiffRevisions(context.Background(), "ctx1", "default", "a", 1, 2)
	testza.AssertNoError(t, err)
	if strings.Contains(d.Values, "hunter2") {
		t.Fatalf("hunter2 leaked into diff: %s", d.Values)
	}
	if strings.Contains(d.Values, "hunter3") {
		t.Fatalf("hunter3 leaked into diff: %s", d.Values)
	}
	if strings.Contains(d.ComputedValues, "hunter2") || strings.Contains(d.ComputedValues, "hunter3") {
		t.Fatalf("raw password leaked into computed diff: %s", d.ComputedValues)
	}
	// The masked password line must appear in the diff context (replicas
	// differs, so the password line shows up as an unchanged context line).
	testza.AssertContains(t, d.Values, MaskValue)
}

func TestDiffRevisions_MissingRevision(t *testing.T) {
	lister := newFakeSecretLister()
	lister.put("ctx1", "default", []corev1.Secret{
		makeReleaseSecret(t, "a", "default", 1, common.StatusDeployed, nil, ""),
	})
	b := NewBackend(NewClientCache(), lister)
	_, err := b.DiffRevisions(context.Background(), "ctx1", "default", "a", 1, 99)
	testza.AssertTrue(t, errors.Is(err, ErrRevisionNotFound))
}

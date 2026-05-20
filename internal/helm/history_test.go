package helm

import (
	"context"
	"errors"
	"testing"

	"github.com/MarvinJWendt/testza"
	"helm.sh/helm/v4/pkg/release/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetHistory_DescendingSort(t *testing.T) {
	lister := newFakeSecretLister()
	lister.put("ctx1", "default", []corev1.Secret{
		makeReleaseSecret(t, "a", "default", 1, common.StatusSuperseded, nil, ""),
		makeReleaseSecret(t, "a", "default", 3, common.StatusDeployed, nil, ""),
		makeReleaseSecret(t, "a", "default", 2, common.StatusSuperseded, nil, ""),
	})
	b := NewBackend(NewClientCache(), lister)
	revs, err := b.GetHistory(context.Background(), "ctx1", "default", "a")
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 3, len(revs))
	testza.AssertEqual(t, 3, revs[0].Number)
	testza.AssertEqual(t, 2, revs[1].Number)
	testza.AssertEqual(t, 1, revs[2].Number)
	testza.AssertEqual(t, "deployed", revs[0].Status)
}

func TestGetHistory_SkipsMalformed(t *testing.T) {
	lister := newFakeSecretLister()
	good := makeReleaseSecret(t, "a", "default", 1, common.StatusDeployed, nil, "")
	bad := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "sh.helm.release.v1.a.v2", Namespace: "default"},
		Type:       ReleaseSecretType,
		Data:       map[string][]byte{"release": []byte("!!!not-base64!!!")},
	}
	lister.put("ctx1", "default", []corev1.Secret{good, bad})
	b := NewBackend(NewClientCache(), lister)
	revs, err := b.GetHistory(context.Background(), "ctx1", "default", "a")
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 1, len(revs))
	testza.AssertEqual(t, 1, revs[0].Number)
}

func TestGetHistory_NotFound(t *testing.T) {
	lister := newFakeSecretLister()
	lister.put("ctx1", "default", nil)
	b := NewBackend(NewClientCache(), lister)
	_, err := b.GetHistory(context.Background(), "ctx1", "default", "missing")
	testza.AssertTrue(t, errors.Is(err, ErrReleaseNotFound))
}

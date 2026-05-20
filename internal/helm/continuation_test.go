package helm

import (
	"errors"
	"testing"

	"github.com/MarvinJWendt/testza"
	"helm.sh/helm/v4/pkg/release/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReassembleContinuation_PassThrough(t *testing.T) {
	in := []corev1.Secret{
		makeReleaseSecret(t, "foo", "default", 1, common.StatusDeployed, nil, ""),
	}
	out, err := ReassembleContinuation(in)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 1, len(out))
	testza.AssertEqual(t, "sh.helm.release.v1.foo.v1", out[0].Name)
}

func TestReassembleContinuation_TwoChunks(t *testing.T) {
	full := makeReleaseSecret(t, "foo", "default", 1, common.StatusDeployed, nil, "")
	data := full.Data["release"]
	mid := len(data) / 2
	in := []corev1.Secret{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "sh.helm.release.v1.foo.v1.0", Namespace: "default"},
			Type:       ReleaseSecretType,
			Data:       map[string][]byte{"release": data[:mid]},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "sh.helm.release.v1.foo.v1.1", Namespace: "default"},
			Type:       ReleaseSecretType,
			Data:       map[string][]byte{"release": data[mid:]},
		},
	}
	out, err := ReassembleContinuation(in)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 1, len(out))
	testza.AssertEqual(t, "sh.helm.release.v1.foo.v1", out[0].Name)
	testza.AssertEqual(t, data, out[0].Data["release"])
	// Round-trip decode to confirm payload is intact.
	rel, err := DecodeRelease(&out[0])
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "foo", rel.Name)
}

func TestReassembleContinuation_ThreeChunksOutOfOrder(t *testing.T) {
	full := makeReleaseSecret(t, "bar", "ns2", 5, common.StatusDeployed, nil, "")
	data := full.Data["release"]
	a := len(data) / 3
	b := 2 * len(data) / 3
	in := []corev1.Secret{
		// Insert in scrambled order.
		{
			ObjectMeta: metav1.ObjectMeta{Name: "sh.helm.release.v1.bar.v5.2", Namespace: "ns2"},
			Type:       ReleaseSecretType,
			Data:       map[string][]byte{"release": data[b:]},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "sh.helm.release.v1.bar.v5.0", Namespace: "ns2"},
			Type:       ReleaseSecretType,
			Data:       map[string][]byte{"release": data[:a]},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "sh.helm.release.v1.bar.v5.1", Namespace: "ns2"},
			Type:       ReleaseSecretType,
			Data:       map[string][]byte{"release": data[a:b]},
		},
	}
	out, err := ReassembleContinuation(in)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 1, len(out))
	testza.AssertEqual(t, data, out[0].Data["release"])
}

func TestReassembleContinuation_MissingChunk(t *testing.T) {
	in := []corev1.Secret{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "sh.helm.release.v1.baz.v1.0", Namespace: "default"},
			Type:       ReleaseSecretType,
			Data:       map[string][]byte{"release": []byte("a")},
		},
		// Skip chunk 1.
		{
			ObjectMeta: metav1.ObjectMeta{Name: "sh.helm.release.v1.baz.v1.2", Namespace: "default"},
			Type:       ReleaseSecretType,
			Data:       map[string][]byte{"release": []byte("c")},
		},
	}
	_, err := ReassembleContinuation(in)
	testza.AssertTrue(t, errors.Is(err, ErrMissingChunk))
}

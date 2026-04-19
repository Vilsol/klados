package volumebrowser

import (
	"context"
	"testing"

	"github.com/MarvinJWendt/testza"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func orphanPod(name, namespace, pvc, session string) *unstructured.Unstructured {
	return orphanPodFull(name, namespace, pvc, session, "host-me", "user-me")
}

func orphanPodFull(name, namespace, pvc, session, host, user string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
				"labels": map[string]any{
					LabelManagedBy: ManagedByValue,
					LabelPurpose:   PurposeValue,
					LabelPVC:       pvc,
					LabelSession:   session,
					LabelHost:      host,
					LabelUser:      user,
				},
			},
			"spec":   map[string]any{},
			"status": map[string]any{"phase": "Running"},
		},
	}
}

func TestScanOrphans_FiltersOwnSession(t *testing.T) {
	mine := orphanPod("klados-pvc-a-1234", "default", "pvc-a", "session-me")
	other := orphanPod("klados-pvc-b-5678", "default", "pvc-b", "session-other")
	conn := connWithObjs(mine, other)

	orphans, err := ScanOrphans(context.Background(), conn, "ctx1", "session-me", "host-me", "user-me")
	testza.AssertNoError(t, err)
	testza.AssertLen(t, orphans, 1)
	testza.AssertEqual(t, "pvc-b", orphans[0].PVCName)
	testza.AssertEqual(t, "ctx1", orphans[0].ContextName)
	testza.AssertEqual(t, "session-other", orphans[0].SessionUUID)
}

func TestScanOrphans_LabelSelectorExcludesUnrelatedPods(t *testing.T) {
	// A pod without the purpose label should not be returned.
	unrelated := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata":   map[string]any{"name": "app", "namespace": "default"},
			"spec":       map[string]any{},
			"status":     map[string]any{"phase": "Running"},
		},
	}
	managed := orphanPod("klados-pvc-a", "default", "pvc-a", "session-other")
	conn := connWithObjs(unrelated, managed)

	orphans, err := ScanOrphans(context.Background(), conn, "ctx1", "session-me", "host-me", "user-me")
	testza.AssertNoError(t, err)
	testza.AssertLen(t, orphans, 1)
	testza.AssertEqual(t, "klados-pvc-a", orphans[0].PodName)
}

func TestScanOrphans_Empty(t *testing.T) {
	conn := connWithObjs()
	orphans, err := ScanOrphans(context.Background(), conn, "ctx1", "session-me", "host-me", "user-me")
	testza.AssertNoError(t, err)
	testza.AssertLen(t, orphans, 0)
}

func TestScanOrphans_ExcludesOtherHostOrUser(t *testing.T) {
	// Real orphan on our host+user, different session.
	orphan := orphanPodFull("klados-pvc-mine-orphan", "default", "pvc-mine", "session-other", "host-me", "user-me")
	// Our own live pod — same session, must be excluded.
	mine := orphanPodFull("klados-pvc-mine-live", "default", "pvc-mine", "session-me", "host-me", "user-me")
	// Another user's live pod on a different host — must NOT appear as our orphan.
	otherHost := orphanPodFull("klados-pvc-other-user", "default", "pvc-shared", "session-foreign", "host-them", "user-them")
	conn := connWithObjs(orphan, mine, otherHost)

	orphans, err := ScanOrphans(context.Background(), conn, "ctx1", "session-me", "host-me", "user-me")
	testza.AssertNoError(t, err)
	testza.AssertLen(t, orphans, 1)
	testza.AssertEqual(t, "klados-pvc-mine-orphan", orphans[0].PodName)
}

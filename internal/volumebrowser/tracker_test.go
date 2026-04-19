package volumebrowser

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/MarvinJWendt/testza"
)

func mp(id, ctx, ns, pvc string) *ManagedPod {
	return &ManagedPod{
		ID:          id,
		ContextName: ctx,
		Namespace:   ns,
		PVCName:     pvc,
		PodName:     "klados-pvc-" + pvc,
		CreatedAt:   time.Now(),
	}
}

func TestTracker_AddGetRemove(t *testing.T) {
	tr := NewTracker()
	testza.AssertNoError(t, tr.Add(mp("a", "ctx1", "default", "pvc-a")))

	got, ok := tr.Get("a")
	testza.AssertTrue(t, ok)
	testza.AssertEqual(t, "pvc-a", got.PVCName)

	removed := tr.Remove("a")
	testza.AssertNotNil(t, removed)
	_, ok = tr.Get("a")
	testza.AssertFalse(t, ok)
}

func TestTracker_AddCollision(t *testing.T) {
	tr := NewTracker()
	testza.AssertNoError(t, tr.Add(mp("a", "ctx1", "default", "pvc-a")))
	err := tr.Add(mp("b", "ctx1", "default", "pvc-a"))
	testza.AssertEqual(t, ErrCollision, err)

	// Different context is fine.
	testza.AssertNoError(t, tr.Add(mp("c", "ctx2", "default", "pvc-a")))
	// Different namespace is fine.
	testza.AssertNoError(t, tr.Add(mp("d", "ctx1", "other", "pvc-a")))
}

func TestTracker_ListForContext(t *testing.T) {
	tr := NewTracker()
	testza.AssertNoError(t, tr.Add(mp("a", "ctx1", "default", "pvc-a")))
	testza.AssertNoError(t, tr.Add(mp("b", "ctx1", "default", "pvc-b")))
	testza.AssertNoError(t, tr.Add(mp("c", "ctx2", "default", "pvc-a")))

	ctx1 := tr.ListForContext("ctx1")
	testza.AssertLen(t, ctx1, 2)

	ctx2 := tr.ListForContext("ctx2")
	testza.AssertLen(t, ctx2, 1)
}

func TestTracker_RemoveAll(t *testing.T) {
	tr := NewTracker()
	testza.AssertNoError(t, tr.Add(mp("a", "ctx1", "default", "pvc-a")))
	testza.AssertNoError(t, tr.Add(mp("b", "ctx2", "default", "pvc-b")))

	drained := tr.RemoveAll()
	testza.AssertLen(t, drained, 2)
	testza.AssertLen(t, tr.ListAll(), 0)
}

func TestTracker_SetTerminalTabID(t *testing.T) {
	tr := NewTracker()
	testza.AssertNoError(t, tr.Add(mp("a", "ctx1", "default", "pvc-a")))
	ok := tr.SetTerminalTabID("a", "tab-42")
	testza.AssertTrue(t, ok)

	got, _ := tr.Get("a")
	testza.AssertEqual(t, "tab-42", got.TerminalTabID)

	testza.AssertFalse(t, tr.SetTerminalTabID("missing", "tab-x"))
}

func TestTracker_ConcurrentSafe(t *testing.T) {
	tr := NewTracker()
	var wg sync.WaitGroup
	const N = 200
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprintf("id-%d", i)
			tr.AddUnchecked(mp(id, "ctx1", "default", fmt.Sprintf("pvc-%d", i)))
			_, _ = tr.Get(id)
			tr.ListForContext("ctx1")
			tr.Remove(id)
		}(i)
	}
	wg.Wait()

	testza.AssertLen(t, tr.ListAll(), 0)
}

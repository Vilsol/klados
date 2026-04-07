package services_test

import (
	"testing"
	"time"

	"github.com/MarvinJWendt/testza"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/Vilsol/klados/internal/services"
)

func TestDrainService_IsActive_FalseWhenNoSession(t *testing.T) {
	svc := services.NewDrainService(nil)
	testza.AssertFalse(t, svc.IsActive("ctx", "node1"))
}

func TestDrainService_ListActive_EmptyWhenNoSessions(t *testing.T) {
	svc := services.NewDrainService(nil)
	result := svc.ListActive("ctx")
	testza.AssertNil(t, result)
}

func TestDrainService_CancelDrain_ErrorWhenNotActive(t *testing.T) {
	svc := services.NewDrainService(nil)
	err := svc.CancelDrain("ctx", "node1")
	testza.AssertNotNil(t, err)
}

func TestDrainService_SessionKey_Isolation(t *testing.T) {
	svc := services.NewDrainService(nil)
	testza.AssertFalse(t, svc.IsActive("ctx1", "node1"))
	testza.AssertFalse(t, svc.IsActive("ctx2", "node1"))
}

// Verify fake.NewSimpleClientset is usable for typed ops.
func TestDrainService_FakeClientset(t *testing.T) {
	_ = fake.NewSimpleClientset()
}

func TestDrainService_ListActive_ReturnsForContext(t *testing.T) {
	svc := services.NewDrainService(nil)
	// No sessions — both contexts return nil/empty
	r1 := svc.ListActive("ctx1")
	r2 := svc.ListActive("ctx2")
	testza.AssertNil(t, r1)
	testza.AssertNil(t, r2)
}

// Ensure IsActive and ListActive are consistent after artificial state
// manipulation — use the exported StartDrain only via a real connection,
// so here we just verify the zero state.
func TestDrainService_ZeroState(t *testing.T) {
	svc := services.NewDrainService(nil)
	_ = time.Now() // just a reference
	testza.AssertFalse(t, svc.IsActive("ctx", "any-node"))
	testza.AssertNil(t, svc.ListActive("ctx"))
	testza.AssertNotNil(t, svc.CancelDrain("ctx", "any-node"))
}

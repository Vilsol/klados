package volumebrowser

import (
	"errors"
	"time"

	"github.com/Vilsol/klados/internal/cluster"
	"github.com/Vilsol/klados/internal/config"
)

// ConnectionProvider mirrors the pattern used elsewhere (portforward/logs/exec)
// so the volumebrowser package has no direct dependency on cluster.Manager.
type ConnectionProvider interface {
	GetConnection(contextName string) (*cluster.Connection, error)
}

type SpawnRequest struct {
	ContextName string
	Namespace   string
	PVCName     string
	Overrides   *SpawnOverrides // nil = use resolved config as-is
}

type SpawnOverrides struct {
	Image                 *string
	MountPath             *string
	ReadOnly              *bool
	ActiveDeadlineSeconds *int64 // pointer-to-nil = explicitly unset
	Resources             *config.ResourceReqs
	NodeSelector          map[string]string
	Tolerations           []map[string]any
}

type ManagedPod struct {
	ID            string // UUID (the tracker key)
	ContextName   string
	Namespace     string
	PodName       string
	PVCName       string
	CreatedAt     time.Time
	SessionUUID   string
	TerminalTabID string // set via Manager.AttachTab(id, tabID)
}

type OrphanPod struct {
	ContextName string
	Namespace   string
	PodName     string
	PVCName     string
	CreatedAt   time.Time
	SessionUUID string
}

// Sentinel errors.
var (
	// ErrCollision is returned when a managed pod for the same (context, namespace, PVC)
	// tuple already exists in the tracker.
	ErrCollision = errors.New("volumebrowser: managed pod already exists for PVC")

	// ErrPVCNotBound is returned when the source PVC is not in the Bound phase.
	ErrPVCNotBound = errors.New("volumebrowser: PVC is not Bound")
)

// Label keys (authoritative; also used by orphan scanner and later tasks).
const (
	LabelManagedBy = "app.kubernetes.io/managed-by"
	LabelPurpose   = "klados.io/purpose"
	LabelPVC       = "klados.io/pvc"
	LabelSession   = "klados.io/session"
	ManagedByValue = "klados"
	PurposeValue   = "pvc-browser"
)

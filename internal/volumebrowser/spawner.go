package volumebrowser

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Vilsol/klados/internal/cluster"
	"github.com/Vilsol/klados/internal/config"
)

// Spawner builds pod specs and creates them via the dynamic client.
type Spawner struct {
	sessionUUID string
}

func NewSpawner(sessionUUID string) *Spawner {
	return &Spawner{sessionUUID: sessionUUID}
}

// SpawnParams bundles the inputs required to spawn a browser pod.
// The caller (Manager) is responsible for resolving the effective config
// (merging cluster overrides onto global defaults) and applying any ad-hoc
// SpawnOverrides before passing it here.
type SpawnParams struct {
	Request  SpawnRequest
	Resolved config.VolumeBrowserConfig
}

// Spawn builds a pod spec, resolves the target node, and creates the pod via
// the dynamic client. On success returns a *ManagedPod describing the created pod.
func (s *Spawner) Spawn(ctx context.Context, conn *cluster.Connection, params SpawnParams) (*ManagedPod, error) {
	req := params.Request
	cfg := params.Resolved

	// Fetch and validate the source PVC.
	pvc, err := conn.Dynamic.Resource(pvcGVR).Namespace(req.Namespace).Get(ctx, req.PVCName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting pvc %s/%s: %w", req.Namespace, req.PVCName, err)
	}
	phase, _, _ := unstructured.NestedString(pvc.Object, "status", "phase")
	if phase != "Bound" {
		return nil, fmt.Errorf("%w: %s/%s is in phase %q", ErrPVCNotBound, req.Namespace, req.PVCName, phase)
	}

	// Resolve the node the PVC is attached to (may return "").
	nodeName, err := ResolveNode(ctx, conn, pvc)
	if err != nil {
		return nil, fmt.Errorf("resolving node for pvc %s/%s: %w", req.Namespace, req.PVCName, err)
	}

	podName, err := buildPodName(req.PVCName)
	if err != nil {
		return nil, fmt.Errorf("generating pod name: %w", err)
	}

	podObj := buildPodSpec(podName, req.PVCName, nodeName, s.sessionUUID, cfg)

	created, err := conn.Dynamic.Resource(podGVR).Namespace(req.Namespace).Create(ctx, podObj, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating pod %s/%s: %w", req.Namespace, podName, err)
	}

	id, err := newManagedID()
	if err != nil {
		return nil, fmt.Errorf("generating managed id: %w", err)
	}
	return &ManagedPod{
		ID:          id,
		ContextName: req.ContextName,
		Namespace:   req.Namespace,
		PodName:     created.GetName(),
		PVCName:     req.PVCName,
		CreatedAt:   time.Now(),
		SessionUUID: s.sessionUUID,
	}, nil
}

func newManagedID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// buildPodName returns fmt.Sprintf("klados-pvc-%s-%s", truncate(pvc, 40), randHex8)
// where the suffix is 8 hex chars (4 random bytes).
func buildPodName(pvc string) (string, error) {
	h := make([]byte, 4)
	if _, err := rand.Read(h); err != nil {
		return "", err
	}
	return fmt.Sprintf("klados-pvc-%s-%s", truncate(pvc, 40), hex.EncodeToString(h)), nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func buildPodSpec(podName, pvcName, nodeName, sessionUUID string, cfg config.VolumeBrowserConfig) *unstructured.Unstructured {
	// Resolve effective scalar values with defaults.
	image := cfg.Image
	if image == "" {
		image = "alpine:edge"
	}
	mountPath := cfg.MountPath
	if mountPath == "" {
		mountPath = "/mnt/volume"
	}
	readOnly := false
	if cfg.ReadOnly != nil {
		readOnly = *cfg.ReadOnly
	}

	container := map[string]any{
		"name":    "browser",
		"image":   image,
		"command": []any{"sh", "-c", "sleep infinity"},
		"volumeMounts": []any{
			map[string]any{
				"name":      "volume",
				"mountPath": mountPath,
				"readOnly":  readOnly,
			},
		},
		"securityContext": map[string]any{
			"runAsNonRoot": false,
		},
	}

	if cfg.Resources != nil {
		resources := map[string]any{}
		if len(cfg.Resources.Requests) > 0 {
			reqs := map[string]any{}
			for k, v := range cfg.Resources.Requests {
				reqs[k] = v
			}
			resources["requests"] = reqs
		}
		if len(cfg.Resources.Limits) > 0 {
			lim := map[string]any{}
			for k, v := range cfg.Resources.Limits {
				lim[k] = v
			}
			resources["limits"] = lim
		}
		if len(resources) > 0 {
			container["resources"] = resources
		}
	}

	gracePeriod := int64(1)
	spec := map[string]any{
		"restartPolicy":                 "Never",
		"terminationGracePeriodSeconds": gracePeriod,
		"containers":                    []any{container},
		"volumes": []any{
			map[string]any{
				"name": "volume",
				"persistentVolumeClaim": map[string]any{
					"claimName": pvcName,
					"readOnly":  readOnly,
				},
			},
		},
	}

	if cfg.ActiveDeadlineSeconds != nil {
		spec["activeDeadlineSeconds"] = *cfg.ActiveDeadlineSeconds
	}
	if nodeName != "" {
		spec["nodeName"] = nodeName
	}
	if len(cfg.NodeSelector) > 0 {
		ns := map[string]any{}
		for k, v := range cfg.NodeSelector {
			ns[k] = v
		}
		spec["nodeSelector"] = ns
	}
	if len(cfg.Tolerations) > 0 {
		tols := make([]any, 0, len(cfg.Tolerations))
		for _, t := range cfg.Tolerations {
			m := map[string]any{}
			for k, v := range t {
				m[k] = v
			}
			tols = append(tols, m)
		}
		spec["tolerations"] = tols
	}

	labels := map[string]any{
		LabelManagedBy: ManagedByValue,
		LabelPurpose:   PurposeValue,
		LabelPVC:       pvcName,
		LabelSession:   sessionUUID,
	}

	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]any{
				"name":   podName,
				"labels": labels,
			},
			"spec": spec,
		},
	}
}

// applyOverrides returns a copy of cfg with req.Overrides applied.
// Pointer fields replace when non-nil; map/slice replace when non-nil.
// This is called by Manager.Spawn before handing off to Spawner.
func applyOverrides(cfg config.VolumeBrowserConfig, o *SpawnOverrides) config.VolumeBrowserConfig {
	if o == nil {
		return cfg
	}
	if o.Image != nil {
		cfg.Image = *o.Image
	}
	if o.MountPath != nil {
		cfg.MountPath = *o.MountPath
	}
	if o.ReadOnly != nil {
		v := *o.ReadOnly
		cfg.ReadOnly = &v
	}
	if o.ActiveDeadlineSeconds != nil {
		// Pointer-to-value → set; pointer to nil not representable here (ptr itself is nil-checked above).
		v := *o.ActiveDeadlineSeconds
		cfg.ActiveDeadlineSeconds = &v
	}
	if o.Resources != nil {
		cp := *o.Resources
		cfg.Resources = &cp
	}
	if o.NodeSelector != nil {
		cfg.NodeSelector = o.NodeSelector
	}
	if o.Tolerations != nil {
		cfg.Tolerations = o.Tolerations
	}
	return cfg
}

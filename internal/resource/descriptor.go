package resource

import (
	"fmt"
	"sync"

	"github.com/google/cel-go/cel"
)

type RenderType string

const (
	RenderText     RenderType = "text"
	RenderBadge    RenderType = "badge"
	RenderAge      RenderType = "age"
	RenderProgress RenderType = "progress"
)

type AlignType string

const (
	AlignLeft   AlignType = "left"
	AlignRight  AlignType = "right"
	AlignCenter AlignType = "center"
)

type Column struct {
	Name       string     `json:"name"`
	Expr       string     `json:"expr"`
	RenderType RenderType `json:"renderType"`
	Width      int        `json:"width,omitempty"`
	Align      AlignType  `json:"align,omitempty"`
	Hidden     bool       `json:"hidden,omitempty"`
}

type OverviewField struct {
	Label      string     `json:"label"`
	Expr       string     `json:"expr"`
	RenderType RenderType `json:"renderType"`
}

type Action struct {
	Name           string `json:"name"`
	Label          string `json:"label"`
	DisabledWhen   string `json:"disabledWhen,omitempty"`
	DisabledReason string `json:"disabledReason,omitempty"`
}

type Descriptor struct {
	Group          string          `json:"group"`
	Version        string          `json:"version"`
	Resource       string          `json:"resource"`
	Kind           string          `json:"kind,omitempty"`
	Columns        []Column        `json:"columns"`
	OverviewFields []OverviewField `json:"overviewFields,omitempty"`
	DetailPanels   []string        `json:"detailPanels,omitempty"`
	Actions        []Action        `json:"actions,omitempty"`
	ClusterScoped  bool            `json:"clusterScoped,omitempty"`
	// IsVirtual marks a descriptor as backed by a VirtualBackend rather than
	// the dynamic Kubernetes client. Engine and WatchManager dispatch to a
	// registered virtual source when this is true.
	IsVirtual bool `json:"isVirtual,omitempty"`
	// GroupLabel is a synthetic sidebar group label (e.g. "Helm"). Optional.
	GroupLabel string `json:"groupLabel,omitempty"`
	// Available signals whether the descriptor is currently usable on the
	// active cluster. Defaults to true; toggled by the registry via
	// SetAvailable for capability-gated entries (e.g. Helm releases when no
	// helm secrets are present yet).
	//
	// Read this field via Registry.IsAvailable(gvr) — direct access races with
	// concurrent SetAvailable calls. The JSON tag is preserved so the field
	// serializes correctly when the Registry snapshots descriptors for the
	// frontend under its own lock.
	Available bool `json:"available"`
}

func (d *Descriptor) GVR() string {
	g := d.Group
	if g == "" {
		g = "core"
	}
	return fmt.Sprintf("%s.%s.%s", g, d.Version, d.Resource)
}

type Registry struct {
	mu          sync.RWMutex
	descriptors map[string]*Descriptor
	celEnv      *cel.Env
}

func NewRegistry() (*Registry, error) {
	env, err := cel.NewEnv(
		cel.Variable("metadata", cel.DynType),
		cel.Variable("spec", cel.DynType),
		cel.Variable("status", cel.DynType),
	)
	if err != nil {
		return nil, fmt.Errorf("creating CEL env: %w", err)
	}
	return &Registry{
		descriptors: make(map[string]*Descriptor),
		celEnv:      env,
	}, nil
}

func (r *Registry) Register(d *Descriptor) error {
	for _, col := range d.Columns {
		_, issues := r.celEnv.Parse(col.Expr)
		if issues != nil && issues.Err() != nil {
			return fmt.Errorf("column %q expr %q: %w", col.Name, col.Expr, issues.Err())
		}
	}
	for _, f := range d.OverviewFields {
		_, issues := r.celEnv.Parse(f.Expr)
		if issues != nil && issues.Err() != nil {
			return fmt.Errorf("overview field %q expr %q: %w", f.Label, f.Expr, issues.Err())
		}
	}
	if !d.Available {
		d.Available = true
	}
	r.mu.Lock()
	r.descriptors[d.GVR()] = d
	r.mu.Unlock()
	return nil
}

// SetAvailable toggles the Available flag for a registered descriptor. Safe
// for concurrent use. No-op if the GVR isn't registered.
func (r *Registry) SetAvailable(gvr string, available bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if d, ok := r.descriptors[gvr]; ok {
		d.Available = available
	}
}

// IsAvailable returns the current Available flag for the registered
// descriptor. Returns false if the GVR isn't registered. This is the
// race-free way to read availability — direct access to Descriptor.Available
// races with SetAvailable.
func (r *Registry) IsAvailable(gvr string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.descriptors[gvr]
	return ok && d.Available
}

// Validate checks all CEL expressions in d without storing it.
func (r *Registry) Validate(d *Descriptor) error {
	for _, col := range d.Columns {
		_, issues := r.celEnv.Parse(col.Expr)
		if issues != nil && issues.Err() != nil {
			return fmt.Errorf("column %q expr %q: %w", col.Name, col.Expr, issues.Err())
		}
	}
	for _, f := range d.OverviewFields {
		_, issues := r.celEnv.Parse(f.Expr)
		if issues != nil && issues.Err() != nil {
			return fmt.Errorf("overview field %q expr %q: %w", f.Label, f.Expr, issues.Err())
		}
	}
	return nil
}

func (r *Registry) Get(gvr string) (*Descriptor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.descriptors[gvr]
	return d, ok
}

func (r *Registry) List() []*Descriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Descriptor, 0, len(r.descriptors))
	for _, d := range r.descriptors {
		out = append(out, d)
	}
	return out
}

// Snapshot returns a value-copy of every registered descriptor under the
// registry's read lock. Use this when serializing descriptors to the frontend
// or any other consumer that needs a stable read of dynamic fields like
// Available without racing concurrent SetAvailable calls.
func (r *Registry) Snapshot() []Descriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Descriptor, 0, len(r.descriptors))
	for _, d := range r.descriptors {
		out = append(out, *d)
	}
	return out
}

package resource

import (
	"fmt"

	"github.com/google/cel-go/cel"
)

type RenderType string

const (
	RenderText     RenderType = "text"
	RenderBadge    RenderType = "badge"
	RenderAge      RenderType = "age"
	RenderProgress RenderType = "progress"
)

type Column struct {
	Name       string     `json:"name"`
	Expr       string     `json:"expr"`
	RenderType RenderType `json:"renderType"`
	Width      int        `json:"width,omitempty"`
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
}

func (d *Descriptor) GVR() string {
	g := d.Group
	if g == "" {
		g = "core"
	}
	return fmt.Sprintf("%s.%s.%s", g, d.Version, d.Resource)
}

type Registry struct {
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
	r.descriptors[d.GVR()] = d
	return nil
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
	d, ok := r.descriptors[gvr]
	return d, ok
}

func (r *Registry) List() []*Descriptor {
	out := make([]*Descriptor, 0, len(r.descriptors))
	for _, d := range r.descriptors {
		out = append(out, d)
	}
	return out
}

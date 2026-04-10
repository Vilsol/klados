package resource

import (
	"bufio"
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/sasha-s/go-deadlock"
	"gopkg.in/yaml.v3"
	"k8s.io/client-go/discovery"
)

//go:embed templates/*.yaml
var templateFiles embed.FS

type TemplateRegistry struct {
	mu      deadlock.RWMutex
	builtin map[string][]Template
	plugins map[string][]Template
}

func NewTemplateRegistry() *TemplateRegistry {
	return &TemplateRegistry{
		builtin: make(map[string][]Template),
		plugins: make(map[string][]Template),
	}
}

func (r *TemplateRegistry) Register(t Template) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.builtin[t.GVR] = append(r.builtin[t.GVR], t)
}

func (r *TemplateRegistry) RegisterPlugin(pluginName string, t Template) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.plugins[pluginName] = append(r.plugins[pluginName], t)
}

func (r *TemplateRegistry) UnregisterPlugin(pluginName string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.plugins, pluginName)
}

func (r *TemplateRegistry) GetTemplates(gvr string) []Template {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Template, 0)
	out = append(out, r.builtin[gvr]...)
	for _, templates := range r.plugins {
		for _, t := range templates {
			if t.GVR == gvr {
				out = append(out, t)
			}
		}
	}
	return out
}

// parseTemplateFile extracts name, description, and YAML content.
// Lines starting with "# name: " or "# description: " are metadata; the rest is content.
func parseTemplateFile(data string) (name, description, content string) {
	var contentLines []string
	scanner := bufio.NewScanner(strings.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "# name: ") {
			name = strings.TrimPrefix(line, "# name: ")
		} else if strings.HasPrefix(line, "# description: ") {
			description = strings.TrimPrefix(line, "# description: ")
		} else {
			contentLines = append(contentLines, line)
		}
	}
	content = strings.TrimSpace(strings.Join(contentLines, "\n"))
	return
}

// gvrFromFilename derives the GVR from a template filename.
// "apps.v1.deployments_worker.yaml" → "apps.v1.deployments"
func gvrFromFilename(filename string) string {
	base := strings.TrimSuffix(filename, ".yaml")
	if idx := strings.Index(base, "_"); idx != -1 {
		base = base[:idx]
	}
	return base
}

// LoadBuiltinTemplates reads all embedded template files and registers them on reg.
func LoadBuiltinTemplates(reg *TemplateRegistry) error {
	entries, err := fs.ReadDir(templateFiles, "templates")
	if err != nil {
		return fmt.Errorf("read embedded templates: %w", err)
	}
	for _, e := range entries {
		data, err := fs.ReadFile(templateFiles, "templates/"+e.Name())
		if err != nil {
			return fmt.Errorf("read template %s: %w", e.Name(), err)
		}
		name, desc, content := parseTemplateFile(string(data))
		if name == "" {
			return fmt.Errorf("template %s missing # name: line", e.Name())
		}
		gvr := gvrFromFilename(e.Name())
		reg.Register(Template{
			GVR:         gvr,
			Name:        name,
			Description: desc,
			Content:     content,
			Source:      "builtin",
		})
	}
	return nil
}

func guessKind(resource string) string {
	if resource == "" {
		return ""
	}
	r := resource
	if strings.HasSuffix(r, "s") {
		r = r[:len(r)-1]
	}
	if len(r) == 0 {
		return resource
	}
	return strings.ToUpper(r[:1]) + r[1:]
}

func (r *TemplateRegistry) GenerateFromSchema(_ context.Context, gvr string, disc discovery.DiscoveryInterface) (Template, error) {
	parsed, err := ParseGVR(gvr)
	if err != nil {
		return Template{}, err
	}

	var apiVersion string
	if parsed.Group == "" {
		apiVersion = parsed.Version
	} else {
		apiVersion = parsed.Group + "/" + parsed.Version
	}

	kind := ""
	if groups, resourceLists, err := disc.ServerGroupsAndResources(); err == nil && groups != nil {
		gv := apiVersion
		for _, rl := range resourceLists {
			if rl.GroupVersion != gv {
				continue
			}
			for _, ar := range rl.APIResources {
				if ar.Name == parsed.Resource {
					kind = ar.Kind
					break
				}
			}
			if kind != "" {
				break
			}
		}
	}
	if kind == "" {
		kind = guessKind(parsed.Resource)
	}

	includeSpec := true
	if doc, err := disc.OpenAPISchema(); err == nil && doc != nil {
		defs := doc.GetDefinitions()
		if defs != nil {
			required := []string{}
			lowerKind := strings.ToLower(kind)
			for _, ns := range defs.AdditionalProperties {
				nsName := strings.ToLower(ns.GetName())
				if nsName == lowerKind || strings.HasSuffix(nsName, "."+lowerKind) {
					required = ns.GetValue().GetRequired()
					break
				}
			}
			hasSpec := false
			for _, f := range required {
				if f == "spec" {
					hasSpec = true
					break
				}
			}
			includeSpec = hasSpec || len(required) == 0
		}
	}

	obj := map[string]any{
		"apiVersion": apiVersion,
		"kind":       kind,
		"metadata":   map[string]any{"name": ""},
	}
	if includeSpec {
		obj["spec"] = map[string]any{}
	}

	content := ""
	if data, err := yaml.Marshal(obj); err == nil {
		content = strings.TrimSpace(string(data))
	} else {
		content = "apiVersion: " + apiVersion + "\nkind: " + kind + "\nmetadata:\n  name: \"\""
	}

	return Template{
		GVR:         gvr,
		Name:        "Default",
		Description: "Auto-generated from schema",
		Content:     content,
		Source:      "schema",
	}, nil
}

func (r *TemplateRegistry) GetAllGVRs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	seen := make(map[string]struct{})
	for gvr := range r.builtin {
		seen[gvr] = struct{}{}
	}
	for _, templates := range r.plugins {
		for _, t := range templates {
			seen[t.GVR] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for gvr := range seen {
		out = append(out, gvr)
	}
	sort.Strings(out)
	return out
}

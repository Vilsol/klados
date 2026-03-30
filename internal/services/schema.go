package services

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/Vilsol/klados/internal/resource"
)

type SchemaService struct {
	appSvc *AppService
	ctx    context.Context
}

func NewSchemaService(appSvc *AppService) *SchemaService {
	return &SchemaService{appSvc: appSvc}
}

func (s *SchemaService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	s.ctx = ctx
	return nil
}

// GetSchema returns the JSON Schema for the given GVR, extracted from the cluster's OpenAPI v3 spec.
// The schema is cached to disk keyed by server URL + server version to survive restarts.
// kind must be the PascalCase resource Kind (e.g. "Deployment") used to locate the schema in OpenAPI.
func (s *SchemaService) GetSchema(contextName, gvr, kind string) (map[string]any, error) {
	conn, err := s.appSvc.ClusterManager().GetConnection(contextName)
	if err != nil {
		return nil, err
	}

	sv, err := conn.Clientset.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("getting server version: %w", err)
	}

	// Use server URL as a stable cluster identifier (more stable than context name).
	serverURL := conn.Config.Host
	urlHash := fmt.Sprintf("%x", sha256.Sum256([]byte(serverURL)))[:12]
	safeGVR := strings.ReplaceAll(gvr, "/", "_")
	cacheKey := fmt.Sprintf("%s-%s-%s", urlHash, sv.GitVersion, safeGVR)
	cachePath := filepath.Join(xdg.CacheHome, "klados", "schemas", cacheKey+".json")

	if data, err := os.ReadFile(cachePath); err == nil {
		var schema map[string]any
		if json.Unmarshal(data, &schema) == nil {
			return schema, nil
		}
	}

	parsed, err := resource.ParseGVR(gvr)
	if err != nil {
		return nil, err
	}

	var apiPath string
	if parsed.Group == "" {
		apiPath = fmt.Sprintf("/openapi/v3/api/%s", parsed.Version)
	} else {
		apiPath = fmt.Sprintf("/openapi/v3/apis/%s/%s", parsed.Group, parsed.Version)
	}

	body, err := conn.Clientset.Discovery().RESTClient().Get().
		AbsPath(apiPath).
		DoRaw(s.ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching schema from %s: %w", apiPath, err)
	}

	var fullDoc map[string]any
	if err := json.Unmarshal(body, &fullDoc); err != nil {
		return nil, fmt.Errorf("parsing OpenAPI doc: %w", err)
	}

	schema := extractResourceSchema(fullDoc, parsed.Group, parsed.Version, kind)
	if schema == nil {
		// Fallback: return full doc so caller still has something
		schema = fullDoc
	}

	data, _ := json.Marshal(schema)
	if data != nil {
		if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err == nil {
			_ = os.WriteFile(cachePath, data, 0o644)
		}
	}

	return schema, nil
}

// extractResourceSchema finds the JSON Schema for a specific resource Kind within an OpenAPI v3 document.
// Kubernetes uses the x-kubernetes-group-version-kind extension to tag schemas.
func extractResourceSchema(doc map[string]any, group, version, kind string) map[string]any {
	if kind == "" {
		return nil
	}

	components, ok := doc["components"].(map[string]any)
	if !ok {
		return nil
	}
	schemas, ok := components["schemas"].(map[string]any)
	if !ok {
		return nil
	}

	wantGroup := group
	if wantGroup == "" {
		wantGroup = ""
	}

	for _, v := range schemas {
		schema, ok := v.(map[string]any)
		if !ok {
			continue
		}
		gvkList, ok := schema["x-kubernetes-group-version-kind"].([]any)
		if !ok {
			continue
		}
		for _, gvkRaw := range gvkList {
			gvk, ok := gvkRaw.(map[string]any)
			if !ok {
				continue
			}
			if gvk["group"] == wantGroup && gvk["version"] == version && gvk["kind"] == kind {
				return schema
			}
		}
	}
	return nil
}

package services

import (
	"testing"

	"github.com/MarvinJWendt/testza"
)

func TestBundleSchemaRefs(t *testing.T) {
	fullDoc := map[string]any{
		"components": map[string]any{
			"schemas": map[string]any{
				"io.k8s.api.apps.v1.DeploymentSpec": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"replicas": map[string]any{"type": "integer"},
						"template": map[string]any{
							"$ref": "#/components/schemas/io.k8s.api.core.v1.PodTemplateSpec",
						},
					},
				},
				"io.k8s.api.core.v1.PodTemplateSpec": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"metadata": map[string]any{"type": "object"},
					},
				},
				"io.k8s.unrelated.Type": map[string]any{
					"type": "string",
				},
			},
		},
	}

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"spec": map[string]any{
				"allOf": []any{
					map[string]any{"$ref": "#/components/schemas/io.k8s.api.apps.v1.DeploymentSpec"},
				},
			},
		},
	}

	bundleSchemaRefs(schema, fullDoc)

	// allOf should be unwrapped to direct $ref, rewritten to #/definitions/
	spec := schema["properties"].(map[string]any)["spec"].(map[string]any)
	testza.AssertNil(t, spec["allOf"])
	testza.AssertEqual(t, "#/definitions/io.k8s.api.apps.v1.DeploymentSpec", spec["$ref"])

	// definitions should contain both transitively referenced schemas
	defs := schema["definitions"].(map[string]any)
	testza.AssertNotNil(t, defs["io.k8s.api.apps.v1.DeploymentSpec"])
	testza.AssertNotNil(t, defs["io.k8s.api.core.v1.PodTemplateSpec"])

	// Unrelated type should NOT be included
	testza.AssertNil(t, defs["io.k8s.unrelated.Type"])

	// $ref inside collected def should also be rewritten
	deploySpec := defs["io.k8s.api.apps.v1.DeploymentSpec"].(map[string]any)
	templateRef := deploySpec["properties"].(map[string]any)["template"].(map[string]any)["$ref"]
	testza.AssertEqual(t, "#/definitions/io.k8s.api.core.v1.PodTemplateSpec", templateRef)
}

func TestBundleSchemaRefs_AllOfUnwrap(t *testing.T) {
	fullDoc := map[string]any{
		"components": map[string]any{
			"schemas": map[string]any{
				"io.k8s.Meta": map[string]any{
					"type":       "object",
					"properties": map[string]any{"name": map[string]any{"type": "string"}},
				},
			},
		},
	}

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"metadata": map[string]any{
				"allOf": []any{
					map[string]any{"$ref": "#/components/schemas/io.k8s.Meta"},
				},
				"default":     map[string]any{},
				"description": "Standard metadata",
			},
		},
	}

	bundleSchemaRefs(schema, fullDoc)

	meta := schema["properties"].(map[string]any)["metadata"].(map[string]any)
	// allOf unwrapped, $ref promoted, sibling fields preserved
	testza.AssertNil(t, meta["allOf"])
	testza.AssertEqual(t, "#/definitions/io.k8s.Meta", meta["$ref"])
	testza.AssertEqual(t, "Standard metadata", meta["description"])
}

func TestBundleSchemaRefs_NoComponents(t *testing.T) {
	schema := map[string]any{"type": "object"}
	fullDoc := map[string]any{}

	// Should not panic
	bundleSchemaRefs(schema, fullDoc)
	testza.AssertNil(t, schema["definitions"])
}

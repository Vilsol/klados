package plugin

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/blang/semver/v4"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	k8syaml "sigs.k8s.io/yaml"

	"github.com/Vilsol/klados/internal/plugin/types"
	"github.com/Vilsol/klados/internal/resource"
)

//go:embed schema/manifest.v1.json
var manifestSchemaBytes []byte

type LoadedPlugin struct {
	Dir              string
	Manifest         *types.ManifestV1Json
	Descriptors      []*resource.Descriptor
	Status           PluginStatus
	ErrorMessage     string
	ConflictWarnings []string
}

type Loader struct {
	pluginsDir string
	schema     *jsonschema.Schema
}

func NewLoader(pluginsDir string) (*Loader, error) {
	var schemaDoc any
	if err := json.Unmarshal(manifestSchemaBytes, &schemaDoc); err != nil {
		return nil, fmt.Errorf("parsing manifest schema: %w", err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("manifest.v1.json", schemaDoc); err != nil {
		return nil, fmt.Errorf("adding manifest schema: %w", err)
	}
	schema, err := compiler.Compile("manifest.v1.json")
	if err != nil {
		return nil, fmt.Errorf("compiling manifest schema: %w", err)
	}
	return &Loader{pluginsDir: pluginsDir, schema: schema}, nil
}

// Load scans pluginsDir and loads all valid plugins. Returns partial results
// alongside any per-plugin errors so callers can warn without aborting.
func (l *Loader) Load() ([]*LoadedPlugin, []error) {
	entries, err := os.ReadDir(l.pluginsDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, []error{fmt.Errorf("reading plugins dir: %w", err)}
	}

	var plugins []*LoadedPlugin
	var errs []error

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(l.pluginsDir, entry.Name())
		p, err := l.loadPlugin(dir)
		if err != nil {
			errs = append(errs, fmt.Errorf("plugin %s: %w", entry.Name(), err))
			continue
		}
		plugins = append(plugins, p)
	}
	return plugins, errs
}

// LoadPlugin loads and validates a single plugin from the given directory.
func (l *Loader) LoadPlugin(dir string) (*LoadedPlugin, error) {
	return l.loadPlugin(dir)
}

func (l *Loader) loadPlugin(dir string) (*LoadedPlugin, error) {
	manifestPath := filepath.Join(dir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("reading manifest.json: %w", err)
	}

	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing manifest.json: %w", err)
	}
	if err := l.schema.Validate(raw); err != nil {
		return nil, fmt.Errorf("validating manifest.json: %w", err)
	}

	var manifest types.ManifestV1Json
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("unmarshaling manifest: %w", err)
	}

	hostSemver, err := semver.ParseTolerant(HostVersion)
	if err != nil {
		return nil, fmt.Errorf("parsing host version: %w", err)
	}
	minSemver, err := semver.ParseTolerant(manifest.MinHostVersion)
	if err != nil {
		return nil, fmt.Errorf("parsing minHostVersion %q: %w", manifest.MinHostVersion, err)
	}
	if minSemver.GT(hostSemver) {
		return nil, fmt.Errorf("requires host >= %s (current: %s)", manifest.MinHostVersion, HostVersion)
	}

	var descriptors []*resource.Descriptor
	if manifest.Extensions != nil {
		for _, relPath := range manifest.Extensions.Descriptors {
			d, err := loadDescriptor(filepath.Join(dir, relPath))
			if err != nil {
				return nil, fmt.Errorf("loading descriptor %q: %w", relPath, err)
			}
			descriptors = append(descriptors, d)
		}
	}

	return &LoadedPlugin{
		Dir:         dir,
		Manifest:    &manifest,
		Descriptors: descriptors,
	}, nil
}

func loadDescriptor(path string) (*resource.Descriptor, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	jsonData, err := k8syaml.YAMLToJSON(data)
	if err != nil {
		return nil, fmt.Errorf("converting YAML to JSON: %w", err)
	}
	var d resource.Descriptor
	if err := json.Unmarshal(jsonData, &d); err != nil {
		return nil, fmt.Errorf("unmarshaling descriptor: %w", err)
	}
	return &d, nil
}

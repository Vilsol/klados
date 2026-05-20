package helm

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/Vilsol/slox"
	release "helm.sh/helm/v4/pkg/release/v1"
)

// ErrReleaseNotFound is returned when no Secrets exist for the requested
// release.
var ErrReleaseNotFound = errors.New("helm: release not found")

// Revision is a flattened view of one Helm release revision.
type Revision struct {
	Number       int
	Status       string
	ChartName    string
	ChartVersion string
	AppVersion   string
	Description  string
	DeployedAt   time.Time
}

// GetHistory lists every revision Secret for (ctx, ns, releaseName) and
// returns them sorted by revision number, descending. Malformed Secrets are
// skipped with a warning log.
func (b *Backend) GetHistory(ctx context.Context, contextName, namespace, releaseName string) ([]Revision, error) {
	secrets, _, err := b.secretLister.ListSecrets(ctx, contextName, namespace, "", "name="+releaseName)
	if err != nil {
		return nil, fmt.Errorf("helm history: %w", err)
	}
	if len(secrets) == 0 {
		return nil, fmt.Errorf("%w: %s/%s", ErrReleaseNotFound, namespace, releaseName)
	}
	flat, err := ReassembleContinuation(secrets)
	if err != nil {
		slox.Warn(ctx, "helm: continuation reassembly failed during history; using raw secrets", "err", err)
		flat = secrets
	}
	revs := make([]Revision, 0, len(flat))
	for i := range flat {
		rel, err := DecodeRelease(&flat[i])
		if err != nil {
			slox.Warn(ctx, "helm: skipping malformed release secret in history", "secret", flat[i].Name, "err", err)
			continue
		}
		revs = append(revs, revisionFromRelease(rel))
	}
	sort.Slice(revs, func(i, j int) bool { return revs[i].Number > revs[j].Number })
	return revs, nil
}

func revisionFromRelease(rel *release.Release) Revision {
	r := Revision{Number: rel.Version}
	if rel.Info != nil {
		r.Status = string(rel.Info.Status)
		r.Description = rel.Info.Description
		r.DeployedAt = rel.Info.LastDeployed
	}
	if rel.Chart != nil && rel.Chart.Metadata != nil {
		r.ChartName = rel.Chart.Metadata.Name
		r.ChartVersion = rel.Chart.Metadata.Version
		r.AppVersion = rel.Chart.Metadata.AppVersion
	}
	return r
}

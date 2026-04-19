package config

type ResolvedPrefs struct {
	Theme                  string                     `json:"theme"`
	AccentColor            string                     `json:"accentColor"`
	FontSize               int                        `json:"fontSize"`
	CompactRows            bool                       `json:"compactRows"`
	ReadOnly               bool                       `json:"readOnly"`
	TerminalWebGL          bool                       `json:"terminalWebGL"`
	Metrics                *MetricsConfig             `json:"metrics,omitempty"`
	ColumnPrefs            map[string]*GVRColumnPrefs `json:"columnPrefs,omitempty"`
	FavoriteNS             []string                   `json:"favoriteNamespaces,omitempty"`
	Keybindings            map[string]string          `json:"keybindings,omitempty"`
	SavedFilters           map[string][]SavedFilter   `json:"savedFilters,omitempty"`
	ContextualAutocomplete bool                       `json:"contextualAutocomplete"`
	VolumeBrowser          VolumeBrowserConfig        `json:"volumeBrowser"`
}

func (c *Config) ResolveForCluster(ctxName string) ResolvedPrefs {
	c.mu.Lock()
	defer c.mu.Unlock()

	r := ResolvedPrefs{
		Theme:         c.Theme,
		AccentColor:   c.AccentColor,
		FontSize:      c.FontSize,
		CompactRows:   c.CompactRows,
		ReadOnly:      c.ReadOnly,
		TerminalWebGL: c.TerminalWebGL,
		ColumnPrefs:   copyColumnPrefs(c.ColumnPrefs),
		Keybindings:   copyStringMap(c.Keybindings),
		SavedFilters:  copySavedFilters(c.SavedFilters),
		VolumeBrowser: cloneVolumeBrowser(c.VolumeBrowser),
	}

	if c.ContextualAutocomplete != nil {
		r.ContextualAutocomplete = *c.ContextualAutocomplete
	} else {
		r.ContextualAutocomplete = true
	}

	if c.Metrics != nil {
		if m, ok := c.Metrics[ctxName]; ok {
			cp := *m
			r.Metrics = &cp
		}
	}

	if ctxName == "" {
		return r
	}

	cluster, ok := c.Clusters[ctxName]
	if !ok || cluster == nil {
		return r
	}

	if cluster.ReadOnly != nil {
		r.ReadOnly = *cluster.ReadOnly
	}
	if cluster.CompactRows != nil {
		r.CompactRows = *cluster.CompactRows
	}
	if cluster.AccentColor != nil {
		r.AccentColor = *cluster.AccentColor
	}
	if cluster.Metrics != nil {
		cp := *cluster.Metrics
		r.Metrics = &cp
	}
	if len(cluster.FavoriteNS) > 0 {
		ns := make([]string, len(cluster.FavoriteNS))
		copy(ns, cluster.FavoriteNS)
		r.FavoriteNS = ns
	}
	for gvr, prefs := range cluster.ColumnPrefs {
		if r.ColumnPrefs == nil {
			r.ColumnPrefs = make(map[string]*GVRColumnPrefs)
		}
		cp := *prefs
		r.ColumnPrefs[gvr] = &cp
	}
	if cluster.VolumeBrowser != nil {
		mergeVolumeBrowser(&r.VolumeBrowser, cluster.VolumeBrowser)
	}
	for gvr, filters := range cluster.SavedFilters {
		if r.SavedFilters == nil {
			r.SavedFilters = make(map[string][]SavedFilter)
		}
		r.SavedFilters[gvr] = append(r.SavedFilters[gvr], filters...)
	}

	return r
}

func copyStringMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func copySavedFilters(m map[string][]SavedFilter) map[string][]SavedFilter {
	if m == nil {
		return nil
	}
	out := make(map[string][]SavedFilter, len(m))
	for k, v := range m {
		cp := make([]SavedFilter, len(v))
		copy(cp, v)
		out[k] = cp
	}
	return out
}

func cloneResourceReqs(r *ResourceReqs) *ResourceReqs {
	if r == nil {
		return nil
	}
	cp := *r
	cp.Requests = copyStringMap(r.Requests)
	cp.Limits = copyStringMap(r.Limits)
	return &cp
}

func cloneTolerations(ts []map[string]any) []map[string]any {
	if ts == nil {
		return nil
	}
	out := make([]map[string]any, len(ts))
	for i, t := range ts {
		m := make(map[string]any, len(t))
		for k, vv := range t {
			m[k] = vv
		}
		out[i] = m
	}
	return out
}

func cloneBoolPtr(b *bool) *bool {
	if b == nil {
		return nil
	}
	v := *b
	return &v
}

func cloneVolumeBrowser(v VolumeBrowserConfig) VolumeBrowserConfig {
	out := v
	out.ReadOnly = cloneBoolPtr(v.ReadOnly)
	out.PromptBeforeSpawn = cloneBoolPtr(v.PromptBeforeSpawn)
	if v.ActiveDeadlineSeconds != nil {
		d := *v.ActiveDeadlineSeconds
		out.ActiveDeadlineSeconds = &d
	}
	out.Resources = cloneResourceReqs(v.Resources)
	out.NodeSelector = copyStringMap(v.NodeSelector)
	out.Tolerations = cloneTolerations(v.Tolerations)
	return out
}

// mergeVolumeBrowser applies non-zero fields from override onto dst.
// Pointer fields (ActiveDeadlineSeconds, Resources, ReadOnly, PromptBeforeSpawn) replace when non-nil.
// Scalar zero values fall through to preserve the global default.
func mergeVolumeBrowser(dst *VolumeBrowserConfig, override *VolumeBrowserConfig) {
	if override == nil {
		return
	}
	if override.Image != "" {
		dst.Image = override.Image
	}
	if override.MountPath != "" {
		dst.MountPath = override.MountPath
	}
	if override.ReadOnly != nil {
		dst.ReadOnly = cloneBoolPtr(override.ReadOnly)
	}
	if override.ActiveDeadlineSeconds != nil {
		d := *override.ActiveDeadlineSeconds
		dst.ActiveDeadlineSeconds = &d
	}
	if override.Resources != nil {
		dst.Resources = cloneResourceReqs(override.Resources)
	}
	if override.NodeSelector != nil {
		dst.NodeSelector = copyStringMap(override.NodeSelector)
	}
	if override.Tolerations != nil {
		dst.Tolerations = cloneTolerations(override.Tolerations)
	}
	if override.PromptBeforeSpawn != nil {
		dst.PromptBeforeSpawn = cloneBoolPtr(override.PromptBeforeSpawn)
	}
	if override.OrphanCleanupOnStartup != "" {
		dst.OrphanCleanupOnStartup = override.OrphanCleanupOnStartup
	}
}

func copyColumnPrefs(m map[string]*GVRColumnPrefs) map[string]*GVRColumnPrefs {
	if m == nil {
		return nil
	}
	out := make(map[string]*GVRColumnPrefs, len(m))
	for k, v := range m {
		cp := *v
		out[k] = &cp
	}
	return out
}

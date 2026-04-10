package config

type ResolvedPrefs struct {
	Theme         string                     `json:"theme"`
	AccentColor   string                     `json:"accentColor"`
	FontSize      int                        `json:"fontSize"`
	CompactRows   bool                       `json:"compactRows"`
	ReadOnly      bool                       `json:"readOnly"`
	TerminalWebGL bool                       `json:"terminalWebGL"`
	Metrics       *MetricsConfig             `json:"metrics,omitempty"`
	ColumnPrefs   map[string]*GVRColumnPrefs `json:"columnPrefs,omitempty"`
	FavoriteNS    []string                   `json:"favoriteNamespaces,omitempty"`
	Keybindings   map[string]string          `json:"keybindings,omitempty"`
	SavedFilters  map[string][]SavedFilter   `json:"savedFilters,omitempty"`
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

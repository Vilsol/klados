package resource

// Template is a named YAML scaffold for creating a Kubernetes resource.
type Template struct {
	GVR         string `json:"gvr"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Content     string `json:"content"`
	Source      string `json:"source"`
}

package plugins

type Plugin struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	APIVersion  int      `json:"apiVersion"`
	Description string   `json:"description"`
	Transport   string   `json:"transport"`
	Domains     []string `json:"domains"`
	Path        string   `json:"-"`
}

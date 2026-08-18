package plugins

import (
	"time"

	"github.com/dop251/goja"
)

type Manifest struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	APIVersion  int      `json:"apiVersion"`
	Description string   `json:"description"`
	Transport   string   `json:"transport"`
	Domains     []string `json:"domains"`
}

type Plugin struct {
	Manifest

	Path    string
	Enabled bool
}

type PluginRuntime struct {
	Plugin *Plugin

	vm     *goja.Runtime
	client *PluginAPIClient

	search          goja.Callable
	browse          goja.Callable
	getTitle        goja.Callable
	downloadChapter goja.Callable
}

type TitleSummary struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Cover string `json:"cover"`
}

type TitleDetails struct {
	ID                string    `json:"id"`
	Title             string    `json:"title"`
	Cover             *string   `json:"cover"`
	AlternativeTitles []string  `json:"alternativeTitles"`
	Author            *string   `json:"author"`
	Artist            *string   `json:"artist"`
	Status            *string   `json:"status"`
	Description       *string   `json:"description"`
	Chapters          []Chapter `json:"chapters"`
}

type Chapter struct {
	ID          string     `json:"id"`
	Number      float64    `json:"number"`
	Volume      *int       `json:"volume"`
	Season      *int       `json:"season"`
	Title       *string    `json:"title"`
	GroupName   *string    `json:"groupName"`
	Language    *string    `json:"language"`
	PublishedAt *time.Time `json:"publishedAt"`
}

package services

import "main/plugins"

type PluginTitle struct {
	plugins.TitleDetails

	SavedID *int64 `json:"savedId"`
}

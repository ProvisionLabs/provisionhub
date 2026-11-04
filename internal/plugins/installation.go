package plugins

import "time"

// Installation identifies local managed state, independently of technology IDs.
type Installation struct {
	PluginID           string    `json:"pluginId"`
	Version            string    `json:"version"`
	Source             string    `json:"source"`
	Path               string    `json:"installationPath"`
	Checksum           string    `json:"checksum"`
	ContentChecksum    string    `json:"contentChecksum"`
	MinimumCoreVersion string    `json:"minimumCoreVersion"`
	InstalledAt        time.Time `json:"installedAt"`
	State              string    `json:"state"`
	Status             string    `json:"status,omitempty"`
	Name               string    `json:"name,omitempty"`
}

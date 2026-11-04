// Package registry defines the static, transport-independent plugin registry v1.
package registry

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

const OfficialURL = "https://raw.githubusercontent.com/ProvisionLabs/provisionhub-official-plugins/main/registry.json"
const CoreVersion = "0.1.0"

type Index struct {
	SchemaVersion int      `json:"schemaVersion"`
	Plugins       []Plugin `json:"plugins"`
}
type Plugin struct {
	ID            string    `json:"id"`
	ManifestID    string    `json:"manifestId"`
	Publisher     string    `json:"publisher"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	TechnologyID  string    `json:"technologyId"`
	Category      string    `json:"category"`
	Documentation string    `json:"documentation,omitempty"`
	Source        string    `json:"source,omitempty"`
	Versions      []Release `json:"versions"`
}
type Release struct {
	Version            string `json:"version"`
	MinimumCoreVersion string `json:"minimumCoreVersion"`
	Artifact           string `json:"artifact"`
	SHA256             string `json:"sha256"`
}

var slug = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var stableVersion = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

func ValidID(id string) bool {
	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		return false
	}
	for _, part := range parts {
		if len(part) > 63 || !slug.MatchString(part) {
			return false
		}
		upper := strings.ToUpper(part)
		if upper == "CON" || upper == "PRN" || upper == "AUX" || upper == "NUL" || len(upper) == 4 && (strings.HasPrefix(upper, "COM") || strings.HasPrefix(upper, "LPT")) && upper[3] >= '1' && upper[3] <= '9' {
			return false
		}
	}
	return true
}
func ValidVersion(v string) bool {
	if len(v) > 32 || !stableVersion.MatchString(v) {
		return false
	}
	for _, p := range strings.Split(v, ".") {
		if _, err := strconv.ParseUint(p, 10, 32); err != nil {
			return false
		}
	}
	return true
}
func Compare(a, b string) int {
	aa, bb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < 3; i++ {
		x, _ := strconv.ParseUint(aa[i], 10, 32)
		y, _ := strconv.ParseUint(bb[i], 10, 32)
		if x < y {
			return -1
		}
		if x > y {
			return 1
		}
	}
	return 0
}
func secureURL(value string) bool {
	u, err := url.Parse(value)
	return err == nil && u.Scheme == "https" && u.Host != "" && u.User == nil && u.Fragment == "" && u.RawQuery == ""
}
func Parse(data []byte) (Index, error) {
	var index Index
	if err := rejectDuplicateKeys(data); err != nil {
		return index, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&index); err != nil {
		return index, fmt.Errorf("decode registry: %w", err)
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return index, errors.New("registry must contain one JSON document")
	}
	return index, index.Validate()
}
func (index Index) Validate() error {
	if index.SchemaVersion != 1 {
		return fmt.Errorf("unsupported registry schemaVersion %d; supported: 1", index.SchemaVersion)
	}
	ids := map[string]bool{}
	for _, p := range index.Plugins {
		for _, text := range []string{p.Publisher, p.Name, p.Description} {
			if len(text) > 4096 || strings.TrimSpace(text) != text {
				return fmt.Errorf("invalid display metadata for %q", p.ID)
			}
			for _, r := range text {
				if unicode.IsControl(r) {
					return fmt.Errorf("control character in metadata for %q", p.ID)
				}
			}
		}
		if !ValidID(p.ID) || !slug.MatchString(p.ManifestID) || p.Publisher == "" || p.Name == "" || p.Description == "" || !slug.MatchString(p.TechnologyID) || (p.Category != "backend" && p.Category != "frontend") {
			return fmt.Errorf("invalid metadata for plugin %q", p.ID)
		}
		if ids[p.ID] {
			return fmt.Errorf("duplicate plugin %q", p.ID)
		}
		ids[p.ID] = true
		if len(p.Versions) == 0 {
			return fmt.Errorf("plugin %q has no releases", p.ID)
		}
		versions := map[string]bool{}
		for _, v := range p.Versions {
			if !ValidVersion(v.Version) || !ValidVersion(v.MinimumCoreVersion) {
				return fmt.Errorf("plugin %q has invalid stable semantic version", p.ID)
			}
			if versions[v.Version] {
				return fmt.Errorf("duplicate release %s@%s", p.ID, v.Version)
			}
			versions[v.Version] = true
			digest, err := hex.DecodeString(v.SHA256)
			if err != nil || len(digest) != 32 || strings.ToLower(v.SHA256) != v.SHA256 {
				return fmt.Errorf("invalid SHA-256 for %s@%s", p.ID, v.Version)
			}
			if !secureURL(v.Artifact) || !strings.Contains(v.Artifact, "/releases/download/") || !strings.Contains(v.Artifact, "/"+strings.ReplaceAll(p.ID, "/", "-")+"-v"+v.Version+"/") {
				return fmt.Errorf("release %s@%s requires an immutable versioned HTTPS release artifact", p.ID, v.Version)
			}
		}
		for _, link := range []string{p.Documentation, p.Source} {
			if link != "" && !secureURL(link) {
				return fmt.Errorf("invalid HTTPS link for %q", p.ID)
			}
		}
	}
	return nil
}
func (index Index) Search(query string) []Plugin {
	result := []Plugin{}
	query = strings.ToLower(query)
	for _, p := range index.Plugins {
		if strings.Contains(strings.ToLower(strings.Join([]string{p.ID, p.Name, p.TechnologyID, p.Category}, " ")), query) {
			result = append(result, p)
		}
	}
	slices.SortFunc(result, func(a, b Plugin) int { return strings.Compare(a.ID, b.ID) })
	return result
}
func (index Index) Resolve(id, version, core string) (Plugin, Release, error) {
	if !ValidVersion(core) {
		return Plugin{}, Release{}, errors.New("invalid core compatibility version")
	}
	for _, p := range index.Plugins {
		if p.ID != id {
			continue
		}
		var best Release
		for _, v := range p.Versions {
			if (version == "" || version == v.Version) && Compare(core, v.MinimumCoreVersion) >= 0 && (best.Version == "" || Compare(v.Version, best.Version) > 0) {
				best = v
			}
		}
		if best.Version != "" {
			return p, best, nil
		}
		return p, Release{}, fmt.Errorf("no compatible release for %s@%s (core %s)", id, version, core)
	}
	return Plugin{}, Release{}, fmt.Errorf("plugin %q not found", id)
}
func (index Index) Marshal() ([]byte, error) {
	if err := index.Validate(); err != nil {
		return nil, err
	}
	index.Plugins = slices.Clone(index.Plugins)
	slices.SortFunc(index.Plugins, func(a, b Plugin) int { return strings.Compare(a.ID, b.ID) })
	for i := range index.Plugins {
		index.Plugins[i].Versions = slices.Clone(index.Plugins[i].Versions)
		slices.SortFunc(index.Plugins[i].Versions, func(a, b Release) int { return Compare(a.Version, b.Version) })
	}
	data, err := json.MarshalIndent(index, "", "  ")
	return append(data, '\n'), err
}

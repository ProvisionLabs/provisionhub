// Package plugins loads small, declarative file generators from local directories.
package plugins

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/ProvisionLabs/provisionhub/internal/catalog"
	"github.com/ProvisionLabs/provisionhub/internal/project"
	"go.yaml.in/yaml/v3"
)

const APIVersion = "provisionhub.dev/plugin/v1alpha1"

type File struct {
	Source      string   `yaml:"source"`
	Destination string   `yaml:"destination"`
	Type        string   `yaml:"type"` // template or copy
	Digest      [32]byte `yaml:"-"`
	Content     []byte   `yaml:"-"`
}

type Manifest struct {
	APIVersion   string           `yaml:"apiVersion"`
	ID           string           `yaml:"id"`
	Version      string           `yaml:"version"`
	TechnologyID string           `yaml:"technologyId"`
	Category     catalog.Category `yaml:"category"`
	Name         string           `yaml:"name"`
	Description  string           `yaml:"description"`
	Capability   string           `yaml:"capability"`
	Files        []File           `yaml:"files"`
	Digest       [32]byte         `yaml:"-"`
}

type Set struct{ byTechnology map[string]Manifest }

func (s Set) ForTechnology(id string) (Manifest, bool) {
	m, ok := s.byTechnology[id]
	return m, ok
}

// LoadDirectory discovers immediate child directories containing plugin.yaml.
// An empty path disables discovery. No plugin code is executed.
func LoadDirectory(root string, technologies catalog.Catalog) (Set, error) {
	set := Set{byTechnology: map[string]Manifest{}}
	if root == "" {
		return set, nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return Set{}, fmt.Errorf("read plugin directory %q: %w", root, err)
	}
	ids := map[string]bool{}
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 {
			return Set{}, fmt.Errorf("plugin entry %q is a symlink", entry.Name())
		}
		if !entry.IsDir() {
			continue
		}
		folder := filepath.Join(root, entry.Name())
		manifestPath := filepath.Join(folder, "plugin.yaml")
		info, err := os.Lstat(manifestPath)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return Set{}, fmt.Errorf("inspect %q: %w", manifestPath, err)
		}
		if !info.Mode().IsRegular() {
			return Set{}, fmt.Errorf("plugin manifest %q must be a regular file", manifestPath)
		}
		if info.Size() > 1<<20 {
			return Set{}, fmt.Errorf("plugin manifest %q exceeds 1 MiB", manifestPath)
		}
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			return Set{}, fmt.Errorf("read %q: %w", manifestPath, err)
		}
		m, err := parse(data)
		if err != nil {
			return Set{}, fmt.Errorf("plugin %q: %w", manifestPath, err)
		}
		m.Digest = sha256.Sum256(data)
		if ids[m.ID] {
			return Set{}, fmt.Errorf("duplicate plugin ID %q", m.ID)
		}
		ids[m.ID] = true
		if _, ok := set.byTechnology[m.TechnologyID]; ok {
			return Set{}, fmt.Errorf("duplicate generator for technology %q", m.TechnologyID)
		}
		technology, err := technologies.ValidateSelection(m.Category, m.TechnologyID)
		if err != nil {
			return Set{}, fmt.Errorf("plugin %q: %w", m.ID, err)
		}
		if technology.Name != m.Name {
			return Set{}, fmt.Errorf("plugin %q changes catalog display name for %q", m.ID, m.TechnologyID)
		}
		for i := range m.Files {
			content, err := readLocalFile(folder, m.Files[i].Source)
			if err != nil {
				return Set{}, fmt.Errorf("plugin %q file %q: %w", m.ID, m.Files[i].Source, err)
			}
			if m.Files[i].Type == "template" {
				if err := validateVariables(content); err != nil {
					return Set{}, fmt.Errorf("plugin %q file %q: %w", m.ID, m.Files[i].Source, err)
				}
			}
			m.Files[i].Content = content
			m.Files[i].Digest = sha256.Sum256(content)
		}
		set.byTechnology[m.TechnologyID] = m
	}
	return set, nil
}

func parse(data []byte) (Manifest, error) {
	var m Manifest
	d := yaml.NewDecoder(bytes.NewReader(data))
	d.KnownFields(true)
	if err := d.Decode(&m); err != nil {
		return m, fmt.Errorf("decode manifest: %w", err)
	}
	var extra any
	if err := d.Decode(&extra); !errors.Is(err, io.EOF) {
		return m, errors.New("plugin manifest must contain one YAML document")
	}
	if m.APIVersion != APIVersion {
		return m, fmt.Errorf("unsupported plugin apiVersion %q", m.APIVersion)
	}
	if !validID(m.ID) || !validID(m.TechnologyID) || !validVersion(m.Version) {
		return m, errors.New("invalid plugin ID, technology ID, or version")
	}
	if m.Name == "" || m.Description == "" {
		return m, errors.New("plugin name and description are required")
	}
	if m.Category != catalog.Backend || m.Capability != "generate_backend" {
		return m, errors.New("this plugin contract currently supports only generate_backend")
	}
	if len(m.Files) == 0 {
		return m, errors.New("plugin must declare at least one file")
	}
	paths := map[string]bool{}
	for _, file := range m.Files {
		if err := validatePath(file.Source); err != nil {
			return m, fmt.Errorf("invalid source %q: %w", file.Source, err)
		}
		if err := validatePath(file.Destination); err != nil {
			return m, fmt.Errorf("invalid destination %q: %w", file.Destination, err)
		}
		if file.Type != "template" && file.Type != "copy" {
			return m, fmt.Errorf("unsupported file type %q", file.Type)
		}
		if strings.EqualFold(file.Destination, project.ManifestName) {
			return m, fmt.Errorf("destination %q is reserved for the project manifest", file.Destination)
		}
		key := strings.ToLower(file.Destination)
		for existing := range paths {
			if existing == key || strings.HasPrefix(existing, key+"/") || strings.HasPrefix(key, existing+"/") {
				return m, fmt.Errorf("duplicate or overlapping output path %q", file.Destination)
			}
		}
		if paths[key] {
			return m, fmt.Errorf("duplicate output path %q", file.Destination)
		}
		paths[key] = true
	}
	slices.SortFunc(m.Files, func(a, b File) int { return strings.Compare(a.Destination, b.Destination) })
	return m, nil
}

func validID(s string) bool {
	if s == "" || len(s) > 63 || s[0] == '-' || s[len(s)-1] == '-' {
		return false
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}

var versionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[a-z0-9][a-z0-9.-]*)?$`)

func validVersion(s string) bool { return len(s) <= 64 && versionPattern.MatchString(s) }

// Paths use slash separators on every platform. Reject all ambiguous forms.
func validatePath(p string) error {
	if p == "" || path.IsAbs(p) || strings.ContainsAny(p, "\\:\x00") || strings.HasPrefix(p, "//") || !fs.ValidPath(p) {
		return errors.New("path must be relative and use slash separators")
	}
	if _, err := filepath.Localize(p); err != nil {
		return fmt.Errorf("path is not portable: %w", err)
	}
	if path.Clean(p) != p {
		return errors.New("path is not clean")
	}
	for _, segment := range strings.Split(p, "/") {
		if segment == "." || segment == ".." || segment == "" {
			return errors.New("path contains an unsafe segment")
		}
		if strings.TrimRight(segment, " .") != segment {
			return errors.New("path segment ends in a dot or space")
		}
		base := strings.ToUpper(strings.Split(segment, ".")[0])
		if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" || len(base) == 4 && ((strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '1' && base[3] <= '9') {
			return errors.New("path contains a Windows reserved device name")
		}
		for _, r := range segment {
			if r < 32 {
				return errors.New("path contains a control character")
			}
		}
	}
	return nil
}

func readLocalFile(folder, name string) ([]byte, error) {
	if err := validatePath(name); err != nil {
		return nil, err
	}
	current := folder
	parts := strings.Split(name, "/")
	for i, part := range parts {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("symlinked template paths are not allowed")
		}
		if i == len(parts)-1 {
			if !info.Mode().IsRegular() || info.Size() > 1<<20 {
				return nil, errors.New("template must be a regular file of at most 1 MiB")
			}
		} else if !info.IsDir() {
			return nil, errors.New("template path contains a non-directory")
		}
	}
	root, err := os.OpenRoot(folder)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return root.ReadFile(name)
}

func validateVariables(content []byte) error {
	s := string(content)
	for strings.Contains(s, "{{") {
		before, rest, _ := strings.Cut(s, "{{")
		if strings.Contains(before, "}}") {
			return errors.New("unmatched template delimiter")
		}
		value, after, ok := strings.Cut(rest, "}}")
		if !ok || value != "projectName" {
			return fmt.Errorf("unsupported template variable %q (only {{projectName}} is allowed)", value)
		}
		s = after
	}
	if strings.Contains(s, "}}") {
		return errors.New("unmatched template delimiter")
	}
	return nil
}

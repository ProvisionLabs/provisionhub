// Package projectlock stores portable, exact project plugin selections.
package projectlock

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ProvisionLabs/provisionhub/internal/registry"
	"go.yaml.in/yaml/v3"
)

const RelativePath = ".provisionhub/plugins.lock.yaml"

type Lock struct {
	LockVersion int      `yaml:"lockVersion"`
	Plugins     []Plugin `yaml:"plugins"`
}

type Plugin struct {
	ID            string `yaml:"id"`
	Version       string `yaml:"version"`
	Source        string `yaml:"source"` // official or local
	Artifact      string `yaml:"artifact,omitempty"`
	SHA256        string `yaml:"sha256,omitempty"`
	ContentSHA256 string `yaml:"contentSha256"`
}

func validHash(s string) bool {
	b, err := hex.DecodeString(s)
	return err == nil && len(b) == 32 && strings.ToLower(s) == s
}

func (l Lock) Validate() error {
	if l.LockVersion != 1 {
		return fmt.Errorf("unsupported lockVersion %d; supported: 1", l.LockVersion)
	}
	seen := map[string]bool{}
	for _, p := range l.Plugins {
		if !registry.ValidID(p.ID) || !registry.ValidVersion(p.Version) || !validHash(p.ContentSHA256) {
			return fmt.Errorf("invalid lock identity or content checksum for %q", p.ID)
		}
		if seen[p.ID] {
			return fmt.Errorf("duplicate locked plugin %q", p.ID)
		}
		seen[p.ID] = true
		switch p.Source {
		case "official":
			u, err := url.Parse(p.Artifact)
			if !validHash(p.SHA256) || err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Fragment != "" || u.RawQuery != "" || !strings.Contains(u.Path, "/releases/download/"+strings.ReplaceAll(p.ID, "/", "-")+"-v"+p.Version+"/") {
				return fmt.Errorf("invalid immutable official artifact for %q", p.ID)
			}
		case "local":
			if p.Artifact != "" || p.SHA256 != "" {
				return fmt.Errorf("local plugin %q cannot declare a remote artifact", p.ID)
			}
		default:
			return fmt.Errorf("unsupported source %q for %q", p.Source, p.ID)
		}
	}
	return nil
}

func Marshal(l Lock) ([]byte, error) {
	if err := l.Validate(); err != nil {
		return nil, err
	}
	l.Plugins = slices.Clone(l.Plugins)
	slices.SortFunc(l.Plugins, func(a, b Plugin) int { return strings.Compare(a.ID, b.ID) })
	return yaml.Marshal(l)
}

func Unmarshal(data []byte) (Lock, error) {
	var l Lock
	d := yaml.NewDecoder(bytes.NewReader(data))
	d.KnownFields(true)
	if err := d.Decode(&l); err != nil {
		return l, err
	}
	var extra any
	if err := d.Decode(&extra); !errors.Is(err, io.EOF) {
		return l, errors.New("lockfile must contain exactly one YAML document")
	}
	return l, l.Validate()
}

// Path anchors the lock to the manifest's directory, without following a
// symlinked manifest, lock directory, or lockfile.
func Path(manifestPath string) (string, error) {
	absolute, err := filepath.Abs(manifestPath)
	if err != nil {
		return "", err
	}
	for _, p := range []string{filepath.Dir(absolute), absolute, filepath.Join(filepath.Dir(absolute), ".provisionhub"), filepath.Join(filepath.Dir(absolute), filepath.FromSlash(RelativePath))} {
		info, e := os.Lstat(p)
		if e == nil && info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("lock path contains symlink %q", p)
		}
		if e != nil && !errors.Is(e, os.ErrNotExist) {
			return "", e
		}
	}
	return filepath.Join(filepath.Dir(absolute), filepath.FromSlash(RelativePath)), nil
}

func Load(manifestPath string) (Lock, bool, error) {
	path, err := Path(manifestPath)
	if err != nil {
		return Lock{}, false, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Lock{}, false, nil
	}
	if err != nil {
		return Lock{}, false, err
	}
	if len(data) > 1<<20 {
		return Lock{}, true, errors.New("lockfile exceeds 1 MiB")
	}
	l, err := Unmarshal(data)
	return l, true, err
}

// Save publishes a new lock without overwriting an existing one. Updates use
// a same-directory rename after the caller has explicitly requested an update.
func Save(manifestPath string, l Lock, update bool) error {
	data, err := Marshal(l)
	if err != nil {
		return err
	}
	path, err := Path(manifestPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if _, err := Path(manifestPath); err != nil {
		return err
	}
	if !update {
		if _, err := os.Lstat(path); err == nil {
			return fmt.Errorf("lockfile already exists at %q; use --update", path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".plugins-lock-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err = tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if update {
		return os.Rename(tmp.Name(), path)
	}
	return os.Link(tmp.Name(), path)
}

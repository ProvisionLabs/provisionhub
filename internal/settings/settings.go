// Package settings defines non-sensitive application preferences, separate from projects.
package settings

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

type Values struct {
	Workspace         string `json:"workspace"`
	PreferredProvider string `json:"preferredProvider"`
}

var ErrInvalidStored = errors.New("stored application settings are invalid")

// Empty values mean no preference, preserving the existing local/offline workflow.
func (v Values) Validate() error {
	if v.Workspace != "" {
		if !filepath.IsAbs(v.Workspace) || strings.IndexFunc(v.Workspace, unicode.IsControl) >= 0 {
			return errors.New("workspace must be an absolute local directory path")
		}
		info, err := os.Stat(v.Workspace)
		if err != nil || !info.IsDir() {
			return errors.New("workspace must be an existing accessible directory")
		}
	}
	switch v.PreferredProvider {
	case "", "github", "gitlab", "bitbucket":
	default:
		return errors.New("preferred provider must be github, gitlab, bitbucket, or empty")
	}
	return nil
}

package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ProvisionLabs/provisionhub/internal/localdb"
	"github.com/ProvisionLabs/provisionhub/internal/project"
	"github.com/ProvisionLabs/provisionhub/internal/projectlock"
)

// SavedProjects owns local references. Its database never contains manifest data.
type SavedProjects struct {
	DatabasePath string
	Projects     ProjectService
}

type SavedProjectView struct {
	Reference     localdb.SavedProject
	Manifest      project.Manifest
	ManifestError error
	Lock          projectlock.Lock
	LockFound     bool
	LockError     error
}

func NewSavedProjects(projects ProjectService) (SavedProjects, error) {
	path, err := localdb.DefaultPath()
	return SavedProjects{DatabasePath: path, Projects: projects}, err
}

func (s SavedProjects) open(ctx context.Context) (*localdb.Store, error) {
	store, err := localdb.Open(ctx, s.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("saved projects database: %w", err)
	}
	return store, nil
}

// CanonicalManifestPath resolves aliases and symlinks for an existing manifest.
func CanonicalManifestPath(path string) (string, error) {
	if strings.TrimSpace(path) == "" || strings.ContainsRune(path, 0) {
		return "", errors.New("manifest path is required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve manifest path: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("inspect manifest %q: %w", abs, err)
	}
	if info.IsDir() {
		abs = filepath.Join(abs, project.ManifestName)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("resolve manifest %q: %w", abs, err)
	}
	info, err = os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("inspect manifest %q: %w", resolved, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("manifest %q must be a regular file", resolved)
	}
	return filepath.Clean(resolved), nil
}

func (s SavedProjects) Register(ctx context.Context, path string) (localdb.SavedProject, error) {
	canonical, err := CanonicalManifestPath(path)
	if err != nil {
		return localdb.SavedProject{}, err
	}
	if _, err := s.Projects.LoadProject(canonical); err != nil {
		return localdb.SavedProject{}, err
	}
	store, err := s.open(ctx)
	if err != nil {
		return localdb.SavedProject{}, err
	}
	defer store.Close()
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return localdb.SavedProject{}, fmt.Errorf("create project identity: %w", err)
	}
	ref := localdb.SavedProject{ID: hex.EncodeToString(bytes), ManifestPath: canonical, RegisteredAt: time.Now().UTC()}
	if err := store.InsertSavedProject(ctx, ref); err != nil {
		return localdb.SavedProject{}, err
	}
	return ref, nil
}

func (s SavedProjects) List(ctx context.Context) ([]SavedProjectView, error) {
	store, err := s.open(ctx)
	if err != nil {
		return nil, err
	}
	defer store.Close()
	refs, err := store.ListSavedProjects(ctx)
	if err != nil {
		return nil, err
	}
	views := make([]SavedProjectView, 0, len(refs))
	for _, ref := range refs {
		views = append(views, s.inspect(ref))
	}
	return views, nil
}

func (s SavedProjects) inspect(ref localdb.SavedProject) SavedProjectView {
	view := SavedProjectView{Reference: ref}
	view.Manifest, view.ManifestError = s.Projects.LoadProject(ref.ManifestPath)
	if view.ManifestError == nil {
		view.Lock, view.LockFound, view.LockError = projectlock.Load(ref.ManifestPath)
	}
	return view
}

func (s SavedProjects) Open(ctx context.Context, id string) (SavedProjectView, error) {
	store, err := s.open(ctx)
	if err != nil {
		return SavedProjectView{}, err
	}
	defer store.Close()
	ref, err := store.SavedProjectByID(ctx, id)
	if err != nil {
		return SavedProjectView{}, err
	}
	view := s.inspect(ref)
	if view.ManifestError != nil {
		return view, view.ManifestError
	}
	if err := store.MarkSavedProjectOpened(ctx, id, time.Now().UTC()); err != nil {
		return view, err
	}
	return view, nil
}

func (s SavedProjects) Relocate(ctx context.Context, id, path string) error {
	canonical, err := CanonicalManifestPath(path)
	if err != nil {
		return err
	}
	if _, err := s.Projects.LoadProject(canonical); err != nil {
		return err
	}
	store, err := s.open(ctx)
	if err != nil {
		return err
	}
	defer store.Close()
	return store.RelocateSavedProject(ctx, id, canonical)
}

func (s SavedProjects) Forget(ctx context.Context, id string) error {
	store, err := s.open(ctx)
	if err != nil {
		return err
	}
	defer store.Close()
	return store.ForgetSavedProject(ctx, id)
}

func (s SavedProjects) FindByPath(ctx context.Context, path string) (localdb.SavedProject, error) {
	canonical, err := CanonicalManifestPath(path)
	if err != nil {
		return localdb.SavedProject{}, err
	}
	store, err := s.open(ctx)
	if err != nil {
		return localdb.SavedProject{}, err
	}
	defer store.Close()
	return store.SavedProjectByPath(ctx, canonical)
}

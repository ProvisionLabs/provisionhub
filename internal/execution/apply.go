// Package execution applies a freshly validated, reviewed file-generation plan.
package execution

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/ProvisionLabs/provisionhub/internal/catalog"
	"github.com/ProvisionLabs/provisionhub/internal/planning"
	"github.com/ProvisionLabs/provisionhub/internal/plugins"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

type Result struct {
	Destination string
	Created     []string
}

type output struct {
	path string
	data []byte
}

// Preflight performs read-only destination and freshness checks. Apply repeats
// them because the filesystem can change between review and publication.
func Preflight(reviewed planning.Plan, manifest project.Manifest, loaded plugins.Set, destination string) error {
	fresh := planning.BuildWithPlugins(manifest, catalog.Builtin(), loaded)
	if !fresh.Executable || !reflect.DeepEqual(reviewed, fresh) {
		return errors.New("generation plan changed or is unavailable; review a new plan before applying")
	}
	if destination == "" {
		return errors.New("destination is required")
	}
	resolved, err := checkDestination(destination)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(resolved)
	if err != nil {
		return fmt.Errorf("open project destination: %w", err)
	}
	defer root.Close()
	info, err := root.Lstat(project.ManifestName)
	if err != nil || !info.Mode().IsRegular() {
		return errors.New("destination must contain a regular provisionhub.yaml")
	}
	fileCount := 0
	for _, op := range fresh.Operations {
		if op.Type == planning.GenerateFile {
			fileCount++
			if len(op.TargetPaths) != 1 {
				return fmt.Errorf("invalid file operation %q", op.ID)
			}
			if err := checkTarget(root, op.TargetPaths[0]); err != nil {
				return err
			}
		}
	}
	if fileCount == 0 {
		return errors.New("plan has no concrete file operations")
	}
	return nil
}

// Apply rechecks the plan against the current manifest and plugin snapshot.
// It stages every rendered file, then publishes through no-overwrite hard links.
func Apply(reviewed planning.Plan, manifest project.Manifest, loaded plugins.Set, destination string) (Result, error) {
	if err := Preflight(reviewed, manifest, loaded, destination); err != nil {
		return Result{}, err
	}
	fresh := planning.BuildWithPlugins(manifest, catalog.Builtin(), loaded)
	if !fresh.Executable || !reflect.DeepEqual(reviewed, fresh) {
		return Result{}, errors.New("generation plan changed or is unavailable; review a new plan before applying")
	}
	if destination == "" {
		return Result{}, errors.New("destination is required")
	}
	resolved, err := checkDestination(destination)
	if err != nil {
		return Result{}, err
	}
	root, err := os.OpenRoot(resolved)
	if err != nil {
		return Result{}, fmt.Errorf("open project destination: %w", err)
	}
	defer root.Close()
	info, err := root.Lstat(project.ManifestName)
	if err != nil || !info.Mode().IsRegular() {
		return Result{}, errors.New("destination must contain a regular provisionhub.yaml")
	}

	outputs := make([]output, 0)
	for _, op := range fresh.Operations {
		if op.Type != planning.GenerateFile {
			continue
		}
		component, err := componentForOperation(manifest, op)
		if err != nil {
			return Result{}, err
		}
		plugin, ok := loaded.ForTechnology(op.Technology)
		if !ok || plugin.ID != op.PluginID {
			return Result{}, fmt.Errorf("plugin for %q changed", op.Technology)
		}
		var matched *plugins.File
		for i := range plugin.Files {
			if plugin.Files[i].Source == op.Source && plugins.DestinationFor(component, plugin.Files[i]) == op.TargetPaths[0] {
				matched = &plugin.Files[i]
				break
			}
		}
		if matched == nil {
			return Result{}, fmt.Errorf("source for operation %q changed", op.ID)
		}
		data := matched.Content
		if matched.Type == "template" {
			componentName := component.Name
			if component.Legacy {
				componentName = manifest.Metadata.Name
			}
			data = []byte(strings.NewReplacer("{{projectName}}", manifest.Metadata.Name, "{{componentName}}", componentName).Replace(string(data)))
		}
		outputs = append(outputs, output{path: op.TargetPaths[0], data: data})
	}
	if len(outputs) == 0 {
		return Result{}, errors.New("plan has no concrete file operations")
	}
	for _, item := range outputs {
		if err := checkTarget(root, item.path); err != nil {
			return Result{}, err
		}
	}

	stage, err := createStage(root)
	if err != nil {
		return Result{}, err
	}
	defer root.RemoveAll(stage)
	for i, item := range outputs {
		name := fmt.Sprintf("%s/%d", stage, i)
		file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			return Result{}, fmt.Errorf("stage %q: %w", item.path, err)
		}
		_, err = file.Write(item.data)
		if err == nil {
			err = file.Sync()
		}
		closeErr := file.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			return Result{}, fmt.Errorf("stage %q: %w", item.path, err)
		}
	}
	created := []string{}
	dirs := []string{}
	rollback := func(cause error) (Result, error) {
		var cleanup []error
		for i := len(created) - 1; i >= 0; i-- {
			// A concurrent replacement must never be removed as part of rollback.
			published, a := root.Stat(created[i])
			staged, b := root.Stat(fmt.Sprintf("%s/%d", stage, i))
			if a == nil && b == nil && os.SameFile(published, staged) {
				if err := root.Remove(created[i]); err != nil {
					cleanup = append(cleanup, err)
				}
			}
		}
		for i := len(dirs) - 1; i >= 0; i-- {
			if err := root.Remove(dirs[i]); err != nil && !errors.Is(err, fs.ErrNotExist) { /* nonempty dirs can remain */
			}
		}
		if len(cleanup) > 0 {
			return Result{Destination: destination, Created: created}, fmt.Errorf("%w; cleanup failed: %v", cause, errors.Join(cleanup...))
		}
		return Result{}, cause
	}
	for i, item := range outputs {
		if err := makeParents(root, path.Dir(item.path), &dirs); err != nil {
			return rollback(err)
		}
		if err := checkTarget(root, item.path); err != nil {
			return rollback(err)
		}
		if err := root.Link(fmt.Sprintf("%s/%d", stage, i), item.path); err != nil {
			return rollback(fmt.Errorf("publish %q without overwrite: %w", item.path, err))
		}
		created = append(created, item.path)
	}
	return Result{Destination: destination, Created: created}, nil
}

func componentForOperation(manifest project.Manifest, operation planning.Operation) (project.Component, error) {
	for _, component := range manifest.CanonicalComponents() {
		if component.Technology != operation.Technology {
			continue
		}
		if (component.Legacy && operation.ComponentID == "") || (!component.Legacy && component.ID() == operation.ComponentID) {
			return component, nil
		}
	}
	return project.Component{}, fmt.Errorf("component %q for operation %q changed", operation.ComponentID, operation.ID)
}

func checkTarget(root *os.Root, target string) error {
	parts := strings.Split(target, "/")
	for i := 1; i < len(parts); i++ {
		parent := path.Join(parts[:i]...)
		info, err := root.Lstat(parent)
		if errors.Is(err, fs.ErrNotExist) {
			break
		}
		if err != nil {
			return fmt.Errorf("inspect output parent %q: %w", parent, err)
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("output parent %q is not a regular directory", parent)
		}
	}
	if _, err := root.Lstat(target); err == nil {
		return fmt.Errorf("output %q already exists; refusing overwrite", target)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("inspect output %q: %w", target, err)
	}
	return nil
}

func makeParents(root *os.Root, parent string, created *[]string) error {
	if parent == "." {
		return nil
	}
	parts := strings.Split(parent, "/")
	for i := range parts {
		name := path.Join(parts[:i+1]...)
		info, err := root.Lstat(name)
		if errors.Is(err, fs.ErrNotExist) {
			if err := root.Mkdir(name, 0o755); err != nil {
				return fmt.Errorf("create output directory %q: %w", name, err)
			}
			*created = append(*created, name)
			continue
		}
		if err != nil {
			return fmt.Errorf("inspect output directory %q: %w", name, err)
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("output directory %q is unsafe", name)
		}
	}
	return nil
}

func createStage(root *os.Root) (string, error) {
	// Root-relative mkdir ensures an attacker cannot redirect staging outside the destination.
	for i := 0; i < 10; i++ {
		var random [12]byte
		if _, err := rand.Read(random[:]); err != nil {
			return "", err
		}
		candidate := ".provisionhub-stage-" + hex.EncodeToString(random[:])
		if err := root.Mkdir(candidate, 0o700); err == nil {
			return candidate, nil
		} else if !errors.Is(err, fs.ErrExist) {
			return "", fmt.Errorf("create staging directory: %w", err)
		}
	}
	return "", errors.New("cannot allocate a unique staging directory")
}

func checkDestination(destination string) (string, error) {
	abs, err := filepath.Abs(destination)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return "", fmt.Errorf("inspect destination %q: %w", abs, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("destination %q is a symlink", abs)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("destination %q is not a directory", abs)
	}
	// Standard macOS paths such as /var can be symlinked ancestors. Resolve
	// those before opening the confined root; the chosen directory itself must
	// still be a real directory.
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("resolve destination %q: %w", abs, err)
	}
	return resolved, nil
}

package localdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrSavedProjectExists = errors.New("project is already registered")
var ErrSavedProjectMissing = errors.New("saved project was not found")

// SavedProject contains only a local reference; project data lives on disk.
type SavedProject struct {
	ID           string
	ManifestPath string
	RegisteredAt time.Time
	LastOpenedAt *time.Time
}

func scanSavedProject(row interface{ Scan(...any) error }) (SavedProject, error) {
	var p SavedProject
	var registered string
	var opened sql.NullString
	if err := row.Scan(&p.ID, &p.ManifestPath, &registered, &opened); err != nil {
		return p, err
	}
	var err error
	p.RegisteredAt, err = time.Parse(time.RFC3339Nano, registered)
	if err != nil {
		return p, fmt.Errorf("invalid saved project registration time: %w", err)
	}
	if opened.Valid {
		value, err := time.Parse(time.RFC3339Nano, opened.String)
		if err != nil {
			return p, fmt.Errorf("invalid saved project open time: %w", err)
		}
		p.LastOpenedAt = &value
	}
	return p, nil
}

const savedProjectColumns = "id,manifest_path,registered_at,last_opened_at"

func (s *Store) InsertSavedProject(ctx context.Context, p SavedProject) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO saved_projects(id,manifest_path,registered_at,last_opened_at) VALUES(?,?,?,NULL)", p.ID, p.ManifestPath, p.RegisteredAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		var count int
		if e := s.db.QueryRowContext(ctx, "SELECT count(*) FROM saved_projects WHERE manifest_path=?", p.ManifestPath).Scan(&count); e == nil && count != 0 {
			return fmt.Errorf("%w: %s", ErrSavedProjectExists, p.ManifestPath)
		}
		return fmt.Errorf("register saved project: %w", err)
	}
	return nil
}

func (s *Store) ListSavedProjects(ctx context.Context) ([]SavedProject, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT "+savedProjectColumns+" FROM saved_projects ORDER BY registered_at, id")
	if err != nil {
		return nil, fmt.Errorf("list saved projects: %w", err)
	}
	defer rows.Close()
	projects := []SavedProject{}
	for rows.Next() {
		p, err := scanSavedProject(rows)
		if err != nil {
			return nil, fmt.Errorf("read saved project: %w", err)
		}
		projects = append(projects, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list saved projects: %w", err)
	}
	return projects, nil
}

func (s *Store) SavedProjectByID(ctx context.Context, id string) (SavedProject, error) {
	p, err := scanSavedProject(s.db.QueryRowContext(ctx, "SELECT "+savedProjectColumns+" FROM saved_projects WHERE id=?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return p, ErrSavedProjectMissing
	}
	return p, err
}

func (s *Store) SavedProjectByPath(ctx context.Context, path string) (SavedProject, error) {
	p, err := scanSavedProject(s.db.QueryRowContext(ctx, "SELECT "+savedProjectColumns+" FROM saved_projects WHERE manifest_path=?", path))
	if errors.Is(err, sql.ErrNoRows) {
		return p, ErrSavedProjectMissing
	}
	return p, err
}

func (s *Store) RelocateSavedProject(ctx context.Context, id, path string) error {
	result, err := s.db.ExecContext(ctx, "UPDATE saved_projects SET manifest_path=? WHERE id=?", path, id)
	if err != nil {
		var count int
		if e := s.db.QueryRowContext(ctx, "SELECT count(*) FROM saved_projects WHERE manifest_path=?", path).Scan(&count); e == nil && count != 0 {
			return ErrSavedProjectExists
		}
		return fmt.Errorf("relocate saved project: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrSavedProjectMissing
	}
	return nil
}

func (s *Store) MarkSavedProjectOpened(ctx context.Context, id string, at time.Time) error {
	result, err := s.db.ExecContext(ctx, "UPDATE saved_projects SET last_opened_at=? WHERE id=?", at.UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return fmt.Errorf("mark saved project opened: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrSavedProjectMissing
	}
	return nil
}

func (s *Store) ForgetSavedProject(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM saved_projects WHERE id=?", id)
	if err != nil {
		return fmt.Errorf("forget saved project: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrSavedProjectMissing
	}
	return nil
}

// ForgetSavedProjects removes an explicitly selected set in one transaction.
func (s *Store) ForgetSavedProjects(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return errors.New("select at least one saved project")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin saved project cleanup: %w", err)
	}
	defer tx.Rollback()
	for _, id := range ids {
		result, err := tx.ExecContext(ctx, "DELETE FROM saved_projects WHERE id=?", id)
		if err != nil {
			return fmt.Errorf("forget saved project %s: %w", id, err)
		}
		count, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if count != 1 {
			return fmt.Errorf("%w: %s", ErrSavedProjectMissing, id)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit saved project cleanup: %w", err)
	}
	return nil
}

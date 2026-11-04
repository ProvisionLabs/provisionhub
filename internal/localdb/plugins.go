package localdb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ProvisionLabs/provisionhub/internal/plugins"
	"github.com/ProvisionLabs/provisionhub/internal/registry"
)

func (s *Store) Installations(ctx context.Context) ([]plugins.Installation, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT plugin_id,version,source,installation_path,checksum,installed_at,state,content_checksum FROM installed_plugins ORDER BY plugin_id,version")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []plugins.Installation{}
	for rows.Next() {
		var item plugins.Installation
		var stamp string
		if err := rows.Scan(&item.PluginID, &item.Version, &item.Source, &item.Path, &item.Checksum, &stamp, &item.State, &item.ContentChecksum); err != nil {
			return nil, err
		}
		item.InstalledAt, err = time.Parse(time.RFC3339Nano, stamp)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) SetInstallationContent(ctx context.Context, id, version, digest string) error {
	result, err := s.db.ExecContext(ctx, "UPDATE installed_plugins SET content_checksum=? WHERE plugin_id=? AND version=? AND state='installing'", digest, id, version)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("installation changed before content validation")
	}
	return nil
}
func (s *Store) ReserveInstallation(ctx context.Context, item plugins.Installation) error {
	if !registry.ValidID(item.PluginID) || !registry.ValidVersion(item.Version) || item.State != "installing" || item.Path == "" || len(item.Checksum) != 64 {
		return errors.New("invalid installation reservation")
	}
	_, err := s.db.ExecContext(ctx, "INSERT INTO installed_plugins(plugin_id,version,source,installation_path,checksum,installed_at,state) VALUES(?,?,?,?,?,?,?)", item.PluginID, item.Version, item.Source, item.Path, item.Checksum, item.InstalledAt.UTC().Format(time.RFC3339Nano), item.State)
	return err
}
func (s *Store) TransitionInstallation(ctx context.Context, id, version, from, to string) error {
	result, err := s.db.ExecContext(ctx, "UPDATE installed_plugins SET state=? WHERE plugin_id=? AND version=? AND state=?", to, id, version, from)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("installation %s@%s changed concurrently; reload its status", id, version)
	}
	return nil
}
func (s *Store) ForgetInstallation(ctx context.Context, id, version string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM installed_plugins WHERE plugin_id=? AND version=? AND state<>'ready'", id, version)
	return err
}

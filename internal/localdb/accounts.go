package localdb

import (
	"context"
	"github.com/ProvisionLabs/provisionhub/internal/gitprovider"
)

func (s *Store) Accounts(ctx context.Context) ([]gitprovider.Account, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,provider,instance_url,subject_id,username,credential_ref,disconnected,created_at,updated_at FROM git_accounts ORDER BY provider,instance_url,username,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	accounts := []gitprovider.Account{}
	for rows.Next() {
		var a gitprovider.Account
		if err := rows.Scan(&a.ID, &a.Provider, &a.InstanceURL, &a.SubjectID, &a.Username, &a.CredentialRef, &a.Disconnected, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}
func (s *Store) PutAccount(ctx context.Context, a gitprovider.Account) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO git_accounts(id,provider,instance_url,subject_id,username,credential_ref,disconnected,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET username=excluded.username,credential_ref=excluded.credential_ref,disconnected=excluded.disconnected,updated_at=excluded.updated_at`, a.ID, a.Provider, a.InstanceURL, a.SubjectID, a.Username, a.CredentialRef, a.Disconnected, a.CreatedAt, a.UpdatedAt)
	return err
}
func (s *Store) MarkDisconnected(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE git_accounts SET disconnected=1 WHERE id=?", id)
	return err
}
func (s *Store) DeleteAccount(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM git_accounts WHERE id=?", id)
	return err
}

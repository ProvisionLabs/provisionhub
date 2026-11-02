package application

import (
	"context"
	"github.com/ProvisionLabs/provisionhub/internal/credentials"
	"github.com/ProvisionLabs/provisionhub/internal/gitprovider"
	"github.com/ProvisionLabs/provisionhub/internal/localdb"
	"github.com/ProvisionLabs/provisionhub/internal/settings"
	"os"
)

// DefaultWorkspace is a best-effort preference read; it never creates storage.
func (s *LocalSettings) DefaultWorkspace(ctx context.Context) (string, error) {
	filename := s.Path
	if filename == "" {
		var err error
		filename, err = localdb.DefaultPath()
		if err != nil {
			return "", err
		}
	}
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return "", nil
	} else if err != nil {
		return "", err
	}
	db, err := localdb.Open(ctx, filename)
	if err != nil {
		return "", err
	}
	defer db.Close()
	values, err := db.LoadSettings(ctx)
	return values.Workspace, err
}

// LocalSettings opens persistence only for explicit settings/account operations.
// Every operation owns and closes its database connection, including error paths.
type LocalSettings struct {
	Path, ClientID string
	Vault          credentials.Store
}

func NewLocalSettings() *LocalSettings {
	return &LocalSettings{ClientID: os.Getenv("PROVISIONHUB_GITHUB_CLIENT_ID"), Vault: credentials.Native{}}
}
func (s *LocalSettings) open(ctx context.Context) (*localdb.Store, *AccountService, error) {
	filename := s.Path
	if filename == "" {
		var err error
		filename, err = localdb.DefaultPath()
		if err != nil {
			return nil, nil, err
		}
	}
	db, err := localdb.Open(ctx, filename)
	if err != nil {
		return nil, nil, err
	}
	vault := s.Vault
	if vault == nil {
		vault = credentials.Native{}
	}
	return db, &AccountService{Repository: db, Credentials: vault, Provider: gitprovider.NewGitHub(s.ClientID)}, nil
}
func (s *LocalSettings) Configured() bool { return gitprovider.NewGitHub(s.ClientID).Configured() }
func (s *LocalSettings) Load(ctx context.Context) (settings.Values, []gitprovider.Status, error) {
	db, _, err := s.open(ctx)
	if err != nil {
		return settings.Values{}, nil, err
	}
	defer db.Close()
	values, err := (SettingsService{Repository: db}).Load(ctx)
	if err != nil {
		return values, nil, err
	}
	accounts, err := db.Accounts(ctx)
	if err != nil {
		return values, nil, err
	}
	statuses := make([]gitprovider.Status, 0, len(accounts))
	for _, account := range accounts {
		state := gitprovider.Unverified
		if account.Disconnected {
			state = gitprovider.NotConnected
		}
		statuses = append(statuses, gitprovider.Status{Account: account, State: state})
	}
	return values, statuses, nil
}
func (s *LocalSettings) Save(ctx context.Context, values settings.Values) error {
	if err := values.Validate(); err != nil {
		return err
	}
	db, _, err := s.open(ctx)
	if err != nil {
		return err
	}
	defer db.Close()
	return (SettingsService{Repository: db}).Save(ctx, values)
}
func (s *LocalSettings) Begin(ctx context.Context) (gitprovider.Authorization, error) {
	db, accounts, err := s.open(ctx)
	if err != nil {
		return gitprovider.Authorization{}, err
	}
	defer db.Close()
	return accounts.Begin(ctx)
}
func (s *LocalSettings) Complete(ctx context.Context, auth gitprovider.Authorization) error {
	db, accounts, err := s.open(ctx)
	if err != nil {
		return err
	}
	defer db.Close()
	return accounts.Complete(ctx, auth)
}
func (s *LocalSettings) Status(ctx context.Context, validate bool) ([]gitprovider.Status, error) {
	db, accounts, err := s.open(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	return accounts.Status(ctx, validate)
}
func (s *LocalSettings) Disconnect(ctx context.Context, id string) error {
	db, accounts, err := s.open(ctx)
	if err != nil {
		return err
	}
	defer db.Close()
	return accounts.Disconnect(ctx, id)
}

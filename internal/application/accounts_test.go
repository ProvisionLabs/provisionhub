package application

import (
	"context"
	"errors"
	"github.com/ProvisionLabs/provisionhub/internal/credentials"
	"github.com/ProvisionLabs/provisionhub/internal/gitprovider"
	"github.com/ProvisionLabs/provisionhub/internal/localdb"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type memoryVault struct {
	values                           map[string]credentials.Secret
	failSet, failDelete, unavailable bool
}

func (v *memoryVault) Get(ref string) (credentials.Secret, error) {
	if v.unavailable {
		return credentials.Secret{}, credentials.ErrUnavailable
	}
	s, ok := v.values[ref]
	if !ok {
		return s, credentials.ErrMissing
	}
	return s, nil
}
func (v *memoryVault) Set(ref string, s credentials.Secret) error {
	if v.failSet {
		return credentials.ErrUnavailable
	}
	v.values[ref] = s
	return nil
}
func (v *memoryVault) Delete(ref string) error {
	if v.failDelete {
		return credentials.ErrUnavailable
	}
	delete(v.values, ref)
	return nil
}

type testProvider struct{ invalid, offline bool }

func (p *testProvider) Describe() gitprovider.Descriptor {
	return gitprovider.Descriptor{ID: "github", InstanceURL: "https://github.com"}
}
func (p *testProvider) Begin(context.Context) (gitprovider.Authorization, error) {
	return gitprovider.Authorization{}, gitprovider.ErrNotConfigured
}
func (p *testProvider) Account(context.Context, credentials.Secret) (gitprovider.Identity, error) {
	if p.invalid {
		return gitprovider.Identity{}, gitprovider.ErrInvalid
	}
	if p.offline {
		return gitprovider.Identity{}, gitprovider.ErrUnavailable
	}
	return gitprovider.Identity{ID: "42", Username: "sample"}, nil
}
func (p *testProvider) Refresh(context.Context, credentials.Secret) (credentials.Secret, error) {
	return credentials.Secret{}, gitprovider.ErrInvalid
}
func (p *testProvider) Disconnect(context.Context, credentials.Secret) error { return nil }

type failRepository struct {
	AccountRepository
	put, remove, mark bool
}

func (r failRepository) PutAccount(ctx context.Context, a gitprovider.Account) error {
	if r.put {
		return errors.New("SQL failed")
	}
	return r.AccountRepository.PutAccount(ctx, a)
}
func (r failRepository) DeleteAccount(ctx context.Context, id string) error {
	if r.remove {
		return errors.New("SQL failed")
	}
	return r.AccountRepository.DeleteAccount(ctx, id)
}
func (r failRepository) MarkDisconnected(ctx context.Context, id string) error {
	if r.mark {
		return errors.New("SQL failed")
	}
	return r.AccountRepository.MarkDisconnected(ctx, id)
}

func TestAccountConsistencyAndNoSQLiteSecrets(t *testing.T) {
	ctx := context.Background()
	filename := filepath.Join(t.TempDir(), "app.db")
	db, err := localdb.Open(ctx, filename)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	vault := &memoryVault{values: map[string]credentials.Secret{}}
	provider := &testProvider{}
	service := &AccountService{Repository: db, Credentials: vault, Provider: provider}
	secret := credentials.Secret{AccessToken: "token-never-in-sqlite", RefreshToken: "refresh-never-in-sqlite"}
	if err := service.Connect(ctx, secret); err != nil {
		t.Fatal(err)
	}
	statuses, err := service.Status(ctx, true)
	if err != nil || len(statuses) != 1 || statuses[0].State != gitprovider.Connected {
		t.Fatal(statuses, err)
	}
	account := statuses[0].Account
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), secret.AccessToken) || strings.Contains(string(data), secret.RefreshToken) {
		t.Fatal("SQLite contains secret")
	}
	if err := service.Connect(ctx, secret); err != nil {
		t.Fatal(err)
	}
	if len(vault.values) != 1 {
		t.Fatal("reconnect leaked credential")
	}
	provider.invalid = true
	statuses, _ = service.Status(ctx, true)
	if statuses[0].State != gitprovider.Invalid {
		t.Fatal(statuses)
	}
	provider.invalid = false
	provider.offline = true
	statuses, _ = service.Status(ctx, true)
	if statuses[0].State != gitprovider.Unavailable {
		t.Fatal(statuses)
	}
	provider.offline = false
	vault.unavailable = true
	statuses, _ = service.Status(ctx, true)
	if statuses[0].State != gitprovider.StorageUnavailable {
		t.Fatal(statuses)
	}
	vault.unavailable = false
	service.Repository = failRepository{AccountRepository: db, put: true}
	if err := service.Connect(ctx, secret); err == nil {
		t.Fatal("accepted metadata failure")
	}
	if len(vault.values) != 1 {
		t.Fatal("failed connection leaked new credential")
	}
	service.Repository = failRepository{AccountRepository: db, remove: true}
	if err := service.Disconnect(ctx, account.ID); err == nil {
		t.Fatal("accepted cleanup failure")
	}
	if len(vault.values) != 0 {
		t.Fatal("disconnect retained credential")
	}
	statuses, _ = service.Status(ctx, true)
	if statuses[0].State != gitprovider.NotConnected {
		t.Fatal("metadata falsely connected", statuses)
	}
	service.Repository = db
	if err := service.Disconnect(ctx, account.ID); err != nil {
		t.Fatal(err)
	}
	accounts, _ := db.Accounts(ctx)
	if len(accounts) != 0 {
		t.Fatal("metadata not deleted")
	}
}

func TestCredentialErrorsAndMissingCredential(t *testing.T) {
	ctx := context.Background()
	db, err := localdb.Open(ctx, filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	vault := &memoryVault{values: map[string]credentials.Secret{}, failSet: true}
	service := &AccountService{Repository: db, Credentials: vault, Provider: &testProvider{}}
	if err := service.Connect(ctx, credentials.Secret{AccessToken: "test"}); !errors.Is(err, credentials.ErrUnavailable) {
		t.Fatal(err)
	}
	accounts, _ := db.Accounts(ctx)
	if len(accounts) != 0 {
		t.Fatal("stored failed credential metadata")
	}
	vault.failSet = false
	service.Connect(ctx, credentials.Secret{AccessToken: "test", ExpiresAt: time.Now().Add(-time.Hour)})
	statuses, _ := service.Status(ctx, true)
	if statuses[0].State != gitprovider.Invalid {
		t.Fatal(statuses)
	}
	clear(vault.values)
	statuses, _ = service.Status(ctx, true)
	if statuses[0].State != gitprovider.Invalid {
		t.Fatal(statuses)
	}
}

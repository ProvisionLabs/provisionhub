package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"github.com/ProvisionLabs/provisionhub/internal/credentials"
	"github.com/ProvisionLabs/provisionhub/internal/gitprovider"
	"sync"
	"time"
)

type AccountRepository interface {
	Accounts(context.Context) ([]gitprovider.Account, error)
	PutAccount(context.Context, gitprovider.Account) error
	MarkDisconnected(context.Context, string) error
	DeleteAccount(context.Context, string) error
}
type AccountService struct {
	Repository  AccountRepository
	Credentials credentials.Store
	Provider    gitprovider.Provider
	mu          sync.Mutex
}

func (s *AccountService) Begin(ctx context.Context) (gitprovider.Authorization, error) {
	if configured, ok := s.Provider.(interface{ Configured() bool }); ok && !configured.Configured() {
		return gitprovider.Authorization{}, gitprovider.ErrNotConfigured
	}
	if _, err := s.Credentials.Get("availability-check"); err != nil && !errors.Is(err, credentials.ErrMissing) {
		return gitprovider.Authorization{}, credentials.ErrUnavailable
	}
	return s.Provider.Begin(ctx)
}
func (s *AccountService) Complete(ctx context.Context, auth gitprovider.Authorization) error {
	secret, err := auth.Wait(ctx)
	if err != nil {
		return err
	}
	return s.Connect(ctx, secret)
}

// Connect receives an ephemeral secret; only its random vault reference enters SQL.
func (s *AccountService) Connect(ctx context.Context, secret credentials.Secret) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	identity, err := s.Provider.Account(ctx, secret)
	if err != nil {
		return err
	}
	accounts, err := s.Repository.Accounts(ctx)
	if err != nil {
		return errors.New("load Git account metadata failed")
	}
	descriptor := s.Provider.Describe()
	now := time.Now().UTC()
	account := gitprovider.Account{Provider: descriptor.ID, InstanceURL: descriptor.InstanceURL, SubjectID: identity.ID, Username: identity.Username, CreatedAt: now, UpdatedAt: now}
	var oldRef string
	for _, old := range accounts {
		if old.Provider == account.Provider && old.InstanceURL == account.InstanceURL && old.SubjectID == identity.ID {
			account.ID = old.ID
			account.CreatedAt = old.CreatedAt
			oldRef = old.CredentialRef
		}
	}
	var random [24]byte
	if _, err := rand.Read(random[:]); err != nil {
		return errors.New("create credential reference failed")
	}
	account.CredentialRef = "git-" + hex.EncodeToString(random[:])
	if account.ID == "" {
		account.ID = account.CredentialRef
	}
	if err := s.Credentials.Set(account.CredentialRef, secret); err != nil {
		return credentials.ErrUnavailable
	}
	if err := s.Repository.PutAccount(ctx, account); err != nil {
		if cleanup := s.Credentials.Delete(account.CredentialRef); cleanup != nil {
			return errors.New("account metadata save failed; new vault credential cleanup also failed")
		}
		return errors.New("account metadata save failed; new vault credential removed")
	}
	if oldRef != "" && oldRef != account.CredentialRef {
		if err := s.Credentials.Delete(oldRef); err != nil {
			return errors.New("account connected; previous vault credential cleanup failed")
		}
	}
	return nil
}

// Status reads the vault and optionally validates with the provider, never on redraw.
func (s *AccountService) Status(ctx context.Context, validate bool) ([]gitprovider.Status, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	accounts, err := s.Repository.Accounts(ctx)
	if err != nil {
		return nil, errors.New("load Git account metadata failed")
	}
	statuses := []gitprovider.Status{}
	for _, a := range accounts {
		status := gitprovider.Status{Account: a, State: gitprovider.NotConnected}
		if a.Disconnected {
			statuses = append(statuses, status)
			continue
		}
		secret, err := s.Credentials.Get(a.CredentialRef)
		switch {
		case errors.Is(err, credentials.ErrMissing):
			status.State = gitprovider.Invalid
		case err != nil:
			status.State = gitprovider.StorageUnavailable
		default:
			status.State = gitprovider.Unverified
			if a.Provider != s.Provider.Describe().ID || a.InstanceURL != s.Provider.Describe().InstanceURL {
				statuses = append(statuses, status)
				continue
			}
			if !secret.ExpiresAt.IsZero() && time.Now().After(secret.ExpiresAt) && validate {
				secret, err = s.Provider.Refresh(ctx, secret)
				if err == nil {
					err = s.Credentials.Set(a.CredentialRef, secret)
				}
			}
			if !validate && !secret.ExpiresAt.IsZero() && time.Now().After(secret.ExpiresAt) {
				err = gitprovider.ErrInvalid
			}
			if err == nil && validate {
				var identity gitprovider.Identity
				identity, err = s.Provider.Account(ctx, secret)
				if err == nil && identity.ID != a.SubjectID {
					err = gitprovider.ErrInvalid
				}
			}
			switch {
			case errors.Is(err, credentials.ErrUnavailable):
				status.State = gitprovider.StorageUnavailable
			case errors.Is(err, gitprovider.ErrInvalid):
				status.State = gitprovider.Invalid
			case err != nil:
				status.State = gitprovider.Unavailable
			case validate:
				status.State = gitprovider.Connected
			}
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

func (s *AccountService) Disconnect(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	accounts, err := s.Repository.Accounts(ctx)
	if err != nil {
		return errors.New("load Git account metadata failed")
	}
	for _, a := range accounts {
		if a.ID == id {
			// Persist an explicit disconnected marker before touching the vault. A later
			// SQL deletion failure must never resurrect a connected-looking account.
			if err := s.Repository.MarkDisconnected(ctx, id); err != nil {
				return errors.New("disconnect metadata update failed; credential retained")
			}
			if err := s.Credentials.Delete(a.CredentialRef); err != nil {
				return credentials.ErrUnavailable
			}
			if err := s.Repository.DeleteAccount(ctx, id); err != nil {
				return errors.New("credential removed; disconnected metadata cleanup failed")
			}
			return nil
		}
	}
	return errors.New("Git account not found")
}

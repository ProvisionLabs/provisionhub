// Package credentials keeps secrets exclusively in the native OS credential vault.
package credentials

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/zalando/go-keyring"
)

var ErrMissing = errors.New("credential is missing")
var ErrUnavailable = errors.New("secure credential storage is unavailable; unlock or configure your OS credential vault")

type Secret struct {
	AccessToken      string    `json:"accessToken"`
	RefreshToken     string    `json:"refreshToken,omitempty"`
	ExpiresAt        time.Time `json:"expiresAt,omitempty"`
	RefreshExpiresAt time.Time `json:"refreshExpiresAt,omitempty"`
}

func (Secret) String() string   { return "[redacted credential]" }
func (Secret) GoString() string { return "[redacted credential]" }

type Store interface {
	Get(string) (Secret, error)
	Set(string, Secret) error
	Delete(string) error
}

type vaultBackend interface {
	Get(string, string) (string, error)
	Set(string, string, string) error
	Delete(string, string) error
}
type systemVault struct{}

func (systemVault) Get(service, ref string) (string, error) { return keyring.Get(service, ref) }
func (systemVault) Set(service, ref, value string) error    { return keyring.Set(service, ref, value) }
func (systemVault) Delete(service, ref string) error        { return keyring.Delete(service, ref) }

type Native struct{ backend vaultBackend }

func (n Native) vault() vaultBackend {
	if n.backend != nil {
		return n.backend
	}
	return systemVault{}
}

const service = "ProvisionHub.git-accounts"

func (n Native) Get(ref string) (Secret, error) {
	data, err := n.vault().Get(service, ref)
	if errors.Is(err, keyring.ErrNotFound) {
		return Secret{}, ErrMissing
	}
	if err != nil {
		return Secret{}, ErrUnavailable
	}
	var secret Secret
	if json.Unmarshal([]byte(data), &secret) != nil || secret.AccessToken == "" {
		return Secret{}, ErrMissing
	}
	return secret, nil
}
func (n Native) Set(ref string, secret Secret) error {
	if secret.AccessToken == "" {
		return ErrMissing
	}
	data, err := json.Marshal(secret)
	if err != nil {
		return ErrUnavailable
	}
	if err := n.vault().Set(service, ref, string(data)); err != nil {
		return ErrUnavailable
	}
	return nil
}
func (n Native) Delete(ref string) error {
	if err := n.vault().Delete(service, ref); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return ErrUnavailable
	}
	return nil
}

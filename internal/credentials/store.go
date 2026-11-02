// Package credentials keeps secrets exclusively in the native OS credential vault.
package credentials

import (
	"encoding/json"
	"errors"
	"github.com/zalando/go-keyring"
	"time"
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

type Native struct{}

const service = "ProvisionHub.git-accounts"

func (Native) Get(ref string) (Secret, error) {
	data, err := keyring.Get(service, ref)
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
func (Native) Set(ref string, secret Secret) error {
	data, err := json.Marshal(secret)
	if err != nil {
		return ErrUnavailable
	}
	if err := keyring.Set(service, ref, string(data)); err != nil {
		return ErrUnavailable
	}
	return nil
}
func (Native) Delete(ref string) error {
	if err := keyring.Delete(service, ref); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return ErrUnavailable
	}
	return nil
}

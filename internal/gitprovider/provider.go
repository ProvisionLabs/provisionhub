// Package gitprovider defines identity/authentication contracts without repository APIs.
package gitprovider

import (
	"context"
	"errors"
	"github.com/ProvisionLabs/provisionhub/internal/credentials"
	"time"
)

var ErrNotConfigured = errors.New("GitHub OAuth is not configured; set PROVISIONHUB_GITHUB_CLIENT_ID to your registered device-flow client ID")
var ErrInvalid = errors.New("credential is invalid or expired; connect again")
var ErrUnavailable = errors.New("Git provider is unavailable; check the network and retry")
var ErrExpired = errors.New("device authorization expired; connect again")
var ErrDenied = errors.New("device authorization was denied or cancelled")
var ErrUnsupported = errors.New("authentication method is not supported")

type Identity struct{ ID, Username string }
type Descriptor struct {
	ID, InstanceURL string
	Methods         []string
}
type Authorization struct {
	UserCode        string
	VerificationURL string
	ExpiresAt       time.Time
	complete        func(context.Context) (credentials.Secret, error)
}

func (a Authorization) Wait(ctx context.Context) (credentials.Secret, error) { return a.complete(ctx) }

type Provider interface {
	Describe() Descriptor
	Begin(context.Context) (Authorization, error)
	Account(context.Context, credentials.Secret) (Identity, error)
	Refresh(context.Context, credentials.Secret) (credentials.Secret, error)
	Disconnect(context.Context, credentials.Secret) error
}

// Account is metadata only. Credentials must never enter the database model.
type Account struct {
	ID            string    `json:"id"`
	Provider      string    `json:"provider"`
	InstanceURL   string    `json:"instanceUrl"`
	SubjectID     string    `json:"subjectId"`
	Username      string    `json:"username"`
	CredentialRef string    `json:"-"`
	Disconnected  bool      `json:"disconnected"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}
type State string

const (
	NotConfigured      State = "not configured"
	NotConnected       State = "not connected"
	Unverified         State = "stored credential (not validated)"
	Authenticating     State = "authentication in progress"
	Connected          State = "connected"
	Invalid            State = "credential invalid or expired"
	Unavailable        State = "provider unavailable"
	StorageUnavailable State = "secure storage unavailable"
)

type Status struct {
	Account Account `json:"account"`
	State   State   `json:"state"`
}

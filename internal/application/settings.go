package application

import (
	"context"
	"github.com/ProvisionLabs/provisionhub/internal/settings"
)

type SettingsRepository interface {
	LoadSettings(context.Context) (settings.Values, error)
	SaveSettings(context.Context, settings.Values) error
}

type SettingsService struct{ Repository SettingsRepository }

func (s SettingsService) Load(ctx context.Context) (settings.Values, error) {
	return s.Repository.LoadSettings(ctx)
}
func (s SettingsService) Save(ctx context.Context, values settings.Values) error {
	if err := values.Validate(); err != nil {
		return err
	}
	return s.Repository.SaveSettings(ctx, values)
}

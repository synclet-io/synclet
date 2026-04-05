package app

import (
	"encoding/base64"
	"errors"

	"github.com/go-pnp/go-pnp/config/configutil"
	"github.com/go-pnp/go-pnp/logging"
	"go.uber.org/fx"
	"gorm.io/gorm"

	_ "github.com/synclet-io/synclet/modules/secret/secretdbstate"
	"github.com/synclet-io/synclet/modules/secret/secretservice"
	"github.com/synclet-io/synclet/modules/secret/secretstorage"
)

type EncryptionKey []byte

func (e *EncryptionKey) UnmarshalText(text []byte) error {
	key, err := base64.StdEncoding.DecodeString(string(text))
	if err != nil || len(key) != 32 {
		return errors.New("ENCRYPTION_KEY must be a valid base64-encoded 32-byte key")
	}

	*e = key

	return nil
}

type secretConfig struct {
	EncryptionKey         EncryptionKey `env:"ENCRYPTION_KEY,notEmpty"`
	EncryptionKeyPrevious EncryptionKey `env:"ENCRYPTION_KEY_PREVIOUS"`
}

func secretModule() fx.Option {
	return fx.Module(
		"secret",
		logging.DecorateNamed("secret"),

		secretConfigModule(),
		secretDependenciesModule(),
		secretUseCasesModule(),
	)
}

func secretConfigModule() fx.Option {
	return fx.Provide(
		configutil.NewPrefixedConfigProvider[secretConfig]("SECRET_"),
		configutil.NewPrefixedConfigInfoProvider[secretConfig]("SECRET_"),
		newSecretServiceConfig,
	)
}

func secretDependenciesModule() fx.Option {
	return fx.Provide(
		fx.Annotate(newSecretStorage, fx.As(new(secretservice.Storage))),
	)
}

func secretUseCasesModule() fx.Option {
	return fx.Provide(
		secretservice.NewStoreSecret,
		secretservice.NewRetrieveSecret,
		secretservice.NewDeleteSecret,
	)
}

func newSecretStorage(db *gorm.DB, logger *logging.Logger) *secretstorage.Storages {
	return secretstorage.NewStorages(db, logger, nil)
}

func newSecretServiceConfig(cfg *secretConfig) secretservice.Config {
	var prevKey []byte

	if len(cfg.EncryptionKeyPrevious) > 0 {
		prevKey = cfg.EncryptionKeyPrevious
	}

	return secretservice.Config{
		MasterKey:   cfg.EncryptionKey,
		PreviousKey: prevKey,
		KeyVersion:  1,
	}
}

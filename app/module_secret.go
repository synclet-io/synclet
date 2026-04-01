package app

import (
	"encoding/base64"
	"errors"

	"github.com/go-pnp/go-pnp/config/configutil"
	logging "github.com/go-pnp/go-pnp/logging"
	"go.uber.org/fx"
	"gorm.io/gorm"

	_ "github.com/synclet-io/synclet/modules/secret/secretdbstate"
	"github.com/synclet-io/synclet/modules/secret/secretservice"
	"github.com/synclet-io/synclet/modules/secret/secretstorage"
)

// secretConfig holds secret module configuration loaded from SECRET_ environment variables.
type secretConfig struct {
	EncryptionKey         string `env:"ENCRYPTION_KEY,notEmpty"`
	EncryptionKeyPrevious string `env:"ENCRYPTION_KEY_PREVIOUS"`
}

// encryptionKey is a type alias for the master encryption key bytes.
type encryptionKey []byte

func secretModule() fx.Option {
	return fx.Options(
		fx.Provide(
			configutil.NewPrefixedConfigProvider[secretConfig]("SECRET_"),
			configutil.NewPrefixedConfigInfoProvider[secretConfig]("SECRET_"),
		),
		fx.Provide(
			func(db *gorm.DB, logger *logging.Logger) *secretstorage.Storages {
				return secretstorage.NewStorages(db, logger, nil)
			},
			fx.Annotate(
				func(s *secretstorage.Storages) secretservice.Storage { return s },
				fx.As(new(secretservice.Storage)),
			),
			func(cfg *secretConfig) (encryptionKey, error) {
				key, err := base64.StdEncoding.DecodeString(cfg.EncryptionKey)
				if err != nil || len(key) != 32 {
					return nil, errors.New("ENCRYPTION_KEY must be a valid base64-encoded 32-byte key")
				}

				return key, nil
			},
			func(storage secretservice.Storage, key encryptionKey) *secretservice.StoreSecret {
				return secretservice.NewStoreSecret(storage, key, 1)
			},
			func(storage secretservice.Storage, key encryptionKey, cfg *secretConfig) *secretservice.RetrieveSecret {
				var prevKey []byte

				if cfg.EncryptionKeyPrevious != "" {
					if decoded, err := base64.StdEncoding.DecodeString(cfg.EncryptionKeyPrevious); err == nil && len(decoded) == 32 {
						prevKey = decoded
					}
				}

				return secretservice.NewRetrieveSecret(storage, key, prevKey)
			},
			secretservice.NewDeleteSecret,
		),
	)
}

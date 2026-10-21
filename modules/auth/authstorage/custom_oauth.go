package authstorage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/synclet-io/synclet/modules/auth/authservice"
)

// OAuthStorage runs the OAuth queries the generated storage cannot express.
type OAuthStorage struct {
	db *gorm.DB
}

func NewOAuthStorage(db *gorm.DB) *OAuthStorage {
	return &OAuthStorage{db: db}
}

var _ authservice.OAuthStorage = (*OAuthStorage)(nil)

func (s *OAuthStorage) ConsumeOAuthCode(ctx context.Context, codeHash string) (*authservice.OAuthCode, error) {
	var row dbOAuthCode

	result := s.db.WithContext(ctx).
		Raw(`DELETE FROM auth.oauth_codes WHERE code_hash = ? RETURNING *`, codeHash).
		Scan(&row)
	if result.Error != nil {
		return nil, fmt.Errorf("consuming OAuth code: %w", wrapOAuthCodeQueryError(result.Error))
	}

	if result.RowsAffected == 0 {
		return nil, authservice.ErrOAuthCodeNotFound
	}

	return convertOAuthCodeFromDB(&row)
}

func (s *OAuthStorage) MarkOAuthRefreshTokenUsed(ctx context.Context, tokenHash string, at time.Time) (*authservice.OAuthRefreshToken, error) {
	var row dbOAuthRefreshToken

	result := s.db.WithContext(ctx).
		Raw(`UPDATE auth.oauth_refresh_tokens SET used_at = ? WHERE token_hash = ? AND used_at IS NULL RETURNING *`, at.UTC(), tokenHash).
		Scan(&row)
	if result.Error != nil {
		return nil, fmt.Errorf("marking OAuth refresh token used: %w", wrapOAuthRefreshTokenQueryError(result.Error))
	}

	if result.RowsAffected == 0 {
		return nil, authservice.ErrOAuthRefreshTokenNotFound
	}

	return convertOAuthRefreshTokenFromDB(&row)
}

func (s *OAuthStorage) DeleteExpiredOAuthKeys(ctx context.Context, before time.Time) error {
	if err := s.db.WithContext(ctx).
		Exec(`DELETE FROM auth.api_keys k
			WHERE k.oauth_grant_id IS NOT NULL AND k.expires_at < @before
			  AND NOT (
			    EXISTS (SELECT 1 FROM auth.oauth_refresh_tokens t
			            WHERE t.grant_id = k.oauth_grant_id AND t.used_at IS NULL AND t.expires_at >= @before)
			    AND k.id = (SELECT l.id FROM auth.api_keys l WHERE l.oauth_grant_id = k.oauth_grant_id
			                ORDER BY l.created_at DESC, l.id DESC LIMIT 1))`,
			sql.Named("before", before.UTC())).Error; err != nil {
		return fmt.Errorf("deleting expired OAuth keys: %w", wrapAPIKeyQueryError(err))
	}

	return nil
}

// DeleteLapsedOAuthGrants deletes the expired refresh tokens, then the grants
// left without one; their API keys go with them by the foreign key cascade.
func (s *OAuthStorage) DeleteLapsedOAuthGrants(ctx context.Context, before time.Time) error {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`DELETE FROM auth.oauth_refresh_tokens WHERE expires_at < ?`, before.UTC()).Error; err != nil {
			return fmt.Errorf("deleting expired OAuth refresh tokens: %w", wrapOAuthRefreshTokenQueryError(err))
		}

		if err := tx.Exec(`DELETE FROM auth.oauth_grants g
			WHERE NOT EXISTS (SELECT 1 FROM auth.oauth_refresh_tokens t WHERE t.grant_id = g.id)`).Error; err != nil {
			return fmt.Errorf("deleting lapsed OAuth grants: %w", wrapOAuthGrantQueryError(err))
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("deleting lapsed OAuth grants: %w", err)
	}

	return nil
}

func (s *OAuthStorage) DeleteUnusedOAuthClients(ctx context.Context, neverGrantedBefore, lastGrantedBefore time.Time) error {
	if err := s.db.WithContext(ctx).Exec(`DELETE FROM auth.oauth_clients c
		WHERE NOT EXISTS (SELECT 1 FROM auth.oauth_grants g WHERE g.client_id = c.client_id)
		  AND ((c.last_granted_at IS NULL AND c.created_at < ?) OR c.last_granted_at < ?)`,
		neverGrantedBefore.UTC(), lastGrantedBefore.UTC()).Error; err != nil {
		return fmt.Errorf("deleting unused OAuth clients: %w", wrapOAuthClientQueryError(err))
	}

	return nil
}
